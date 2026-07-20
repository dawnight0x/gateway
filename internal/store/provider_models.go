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
	return s.replaceProviderModels(ctx, providerID, "", models)
}

func (s *Store) ReplaceProviderKeyModels(ctx context.Context, providerID, keyID string, models []string) error {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return sql.ErrNoRows
	}
	var found int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM keys WHERE id=? AND provider_id=?`, keyID, strings.TrimSpace(providerID)).Scan(&found); err != nil {
		return err
	}
	return s.replaceProviderModels(ctx, providerID, keyID, models)
}

func (s *Store) replaceProviderModels(ctx context.Context, providerID, keyID string, models []string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return sql.ErrNoRows
	}
	providerType, ok, err := s.providerType(ctx, providerID)
	if err != nil {
		return err
	}
	if !ok {
		return sql.ErrNoRows
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	deleteQuery := `DELETE FROM provider_models WHERE provider_id=? AND key_id=?`
	deleteArgs := []any{providerID, keyID}
	if keyID != "" {
		deleteQuery = `DELETE FROM provider_models WHERE provider_id=? AND (key_id=? OR key_id='')`
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return err
	}
	seen := make(map[string]struct{})
	for _, modelID := range models {
		modelID = model.NormalizeModelID(providerType, modelID)
		if modelID == "" || len(modelID) > 512 {
			continue
		}
		if _, exists := seen[modelID]; exists {
			continue
		}
		seen[modelID] = struct{}{}
		if _, err := tx.ExecContext(ctx, `INSERT INTO provider_models (provider_id,key_id,model_id,discovered_at) VALUES (?,?,?,CURRENT_TIMESTAMP)`, providerID, keyID, modelID); err != nil {
			return err
		}
		if len(seen) >= maxDiscoveredModelsPerProvider {
			break
		}
	}
	return tx.Commit()
}

func (s *Store) ListProviderModels(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT pm.provider_id,p.type,pm.model_id
		FROM provider_models pm JOIN providers p ON p.id=pm.provider_id
		ORDER BY pm.provider_id,pm.model_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]string)
	seen := make(map[string]map[string]struct{})
	for rows.Next() {
		var providerID, providerType, modelID string
		if err := rows.Scan(&providerID, &providerType, &modelID); err != nil {
			return nil, err
		}
		modelID = model.NormalizeModelID(providerType, modelID)
		if modelID == "" {
			continue
		}
		if seen[providerID] == nil {
			seen[providerID] = make(map[string]struct{})
		}
		if _, duplicate := seen[providerID][modelID]; duplicate {
			continue
		}
		seen[providerID][modelID] = struct{}{}
		out[providerID] = append(out[providerID], modelID)
	}
	return out, rows.Err()
}

func (s *Store) CountProviderModels(ctx context.Context, providerID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT model_id) FROM provider_models WHERE provider_id=?`, strings.TrimSpace(providerID)).Scan(&count)
	return count, err
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

func (s *Store) RecordProviderModelDiscoveryPartial(ctx context.Context, providerID string, modelCount int, message string) error {
	providerID = strings.TrimSpace(providerID)
	message = strings.TrimSpace(message)
	if providerID == "" {
		return sql.ErrNoRows
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO provider_model_discovery (provider_id,status,model_count,last_attempt_at,last_success_at,last_error)
		VALUES (?,'partial',?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,?)
		ON CONFLICT(provider_id) DO UPDATE SET
			status='partial',model_count=excluded.model_count,last_attempt_at=CURRENT_TIMESTAMP,
			last_success_at=CURRENT_TIMESTAMP,last_error=excluded.last_error
	`, providerID, max(modelCount, 0), message)
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

func (s *Store) providerType(ctx context.Context, id string) (string, bool, error) {
	var providerType string
	err := s.db.QueryRowContext(ctx, `SELECT type FROM providers WHERE id=?`, id).Scan(&providerType)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return providerType, true, nil
}
