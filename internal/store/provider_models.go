package store

import (
	"context"
	"database/sql"
	"strings"
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

func (s *Store) providerExists(ctx context.Context, id string) (bool, error) {
	var found int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM providers WHERE id=?`, id).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
