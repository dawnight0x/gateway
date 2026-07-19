package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"local-ai-gateway/internal/model"
)

const maxDiscoveredModelsPerProvider = 5000

func (s *Store) ReplaceProviderModels(ctx context.Context, providerID string, models []string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return sql.ErrNoRows
	}
	if ok, err := s.providerExists(ctx, providerID); err != nil {
		return err
	} else if !ok {
		return sql.ErrNoRows
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_models WHERE provider_id=?`, providerID); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, modelID := range models {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" || len(modelID) > 512 {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}
		if _, err := tx.ExecContext(ctx, `INSERT INTO provider_models (provider_id,model_id,discovered_at) VALUES (?,?,CURRENT_TIMESTAMP)`, providerID, modelID); err != nil {
			return err
		}
		if len(seen) >= maxDiscoveredModelsPerProvider {
			break
		}
	}
	return tx.Commit()
}

func (s *Store) ListProviderModels(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id,model_id FROM provider_models ORDER BY provider_id,model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]string)
	for rows.Next() {
		var providerID, modelID string
		if err := rows.Scan(&providerID, &modelID); err != nil {
			return nil, err
		}
		out[providerID] = append(out[providerID], modelID)
	}
	return out, rows.Err()
}

func (s *Store) RecordProviderModelDiscoverySuccess(ctx context.Context, providerID string, modelCount int) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return sql.ErrNoRows
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO provider_model_discovery (provider_id,status,model_count,last_attempt_at,last_success_at,last_error)
		VALUES (?,'ok',?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,'')
		ON CONFLICT(provider_id) DO UPDATE SET
			status='ok',model_count=excluded.model_count,last_attempt_at=CURRENT_TIMESTAMP,
			last_success_at=CURRENT_TIMESTAMP,last_error=''
	`, providerID, max(modelCount, 0))
	return err
}

func (s *Store) RecordProviderModelDiscoveryFailure(ctx context.Context, providerID, status, message string) error {
	providerID = strings.TrimSpace(providerID)
	status = strings.TrimSpace(status)
	message = strings.TrimSpace(message)
	if providerID == "" {
		return sql.ErrNoRows
	}
	if status == "" || status == "ok" {
		status = "error"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO provider_model_discovery (provider_id,status,model_count,last_attempt_at,last_error)
		VALUES (?,?,0,CURRENT_TIMESTAMP,?)
		ON CONFLICT(provider_id) DO UPDATE SET
			status=excluded.status,last_attempt_at=CURRENT_TIMESTAMP,last_error=excluded.last_error
	`, providerID, status, message)
	return err
}

func (s *Store) ListProviderModelDiscoveries(ctx context.Context) (map[string]model.ProviderModelDiscovery, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider_id,status,model_count,last_attempt_at,last_success_at,last_error
		FROM provider_model_discovery ORDER BY provider_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]model.ProviderModelDiscovery)
	for rows.Next() {
		var item model.ProviderModelDiscovery
		var lastAttempt, lastSuccess sql.NullString
		if err := rows.Scan(&item.ProviderID, &item.Status, &item.ModelCount, &lastAttempt, &lastSuccess, &item.LastError); err != nil {
			return nil, err
		}
		item.LastAttemptAt = nullableModelDiscoveryTime(lastAttempt)
		item.LastSuccessAt = nullableModelDiscoveryTime(lastSuccess)
		out[item.ProviderID] = item
	}
	return out, rows.Err()
}

func nullableModelDiscoveryTime(value sql.NullString) *time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	parsed := parseTime(value.String)
	return &parsed
}

func (s *Store) providerExists(ctx context.Context, id string) (bool, error) {
	var found int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM providers WHERE id=?`, id).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
