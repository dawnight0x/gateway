package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"local-ai-gateway/internal/model"
)

const (
	maxProviderModelStatesPerProvider = 2048
	providerModelStateRetention       = 30 * 24 * time.Hour
)

func (s *Store) ListProviderModelStates(ctx context.Context) ([]model.ProviderModelState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT provider_id,key_id,model_id,consecutive_failures,cooldown_until,last_error,last_status_code,
			success_count,failure_count,last_used_at,'key' AS scope
		FROM provider_model_state
		UNION ALL
		SELECT provider_id,'' AS key_id,model_id,consecutive_failures,cooldown_until,last_error,last_status_code,
			success_count,failure_count,last_used_at,'provider' AS scope
		FROM provider_model_global_state
		ORDER BY provider_id,model_id,scope,key_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.ProviderModelState, 0)
	for rows.Next() {
		var item model.ProviderModelState
		var cooldown, lastUsed sql.NullString
		var status sql.NullInt64
		if err := rows.Scan(
			&item.ProviderID, &item.KeyID, &item.ModelID, &item.ConsecutiveFailures, &cooldown, &item.LastError,
			&status, &item.SuccessCount, &item.FailureCount, &lastUsed, &item.Scope,
		); err != nil {
			return nil, err
		}
		item.CooldownUntil = nullableStoreTime(cooldown)
		item.LastUsedAt = nullableStoreTime(lastUsed)
		if status.Valid {
			value := int(status.Int64)
			item.LastStatusCode = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecordProviderModelSuccess(ctx context.Context, providerID, keyID, modelID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE provider_model_state
		SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL,
			success_count=success_count+1,last_used_at=CURRENT_TIMESTAMP
		WHERE provider_id=? AND key_id=? AND model_id=?
	`, strings.TrimSpace(providerID), strings.TrimSpace(keyID), strings.TrimSpace(modelID))
	return err
}

func (s *Store) RecordProviderWideModelSuccess(ctx context.Context, providerID, modelID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE provider_model_global_state
		SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL,
			success_count=success_count+1,last_used_at=CURRENT_TIMESTAMP
		WHERE provider_id=? AND model_id=?
	`, strings.TrimSpace(providerID), strings.TrimSpace(modelID))
	return err
}

func (s *Store) RecordProviderModelFailure(ctx context.Context, providerID, keyID, modelID string, status *int, message string, policy FailurePolicy) error {
	providerID = strings.TrimSpace(providerID)
	keyID = strings.TrimSpace(keyID)
	modelID = strings.TrimSpace(modelID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var failures int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO provider_model_state (
			provider_id,key_id,model_id,consecutive_failures,last_error,last_status_code,failure_count,last_used_at
		) VALUES (?,?,?,1,?,?,1,CURRENT_TIMESTAMP)
		ON CONFLICT(provider_id,key_id,model_id) DO UPDATE SET
			consecutive_failures=consecutive_failures+1,last_error=excluded.last_error,
			last_status_code=excluded.last_status_code,failure_count=failure_count+1,last_used_at=CURRENT_TIMESTAMP
		RETURNING consecutive_failures
	`, providerID, keyID, modelID, message, status).Scan(&failures)
	if err != nil {
		return err
	}
	if policy.ForceCooldown || failures >= policy.Threshold {
		duration := policy.Cooldown
		if failures == policy.Threshold && policy.ThresholdCooldown > 0 {
			duration = policy.ThresholdCooldown
		}
		if policy.OverrideCooldown > 0 {
			duration = policy.OverrideCooldown
		}
		until := time.Now().Add(duration).UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `
			UPDATE provider_model_state SET cooldown_until=? WHERE provider_id=? AND key_id=? AND model_id=?
		`, until, providerID, keyID, modelID); err != nil {
			return err
		}
	}
	if err := pruneProviderModelStates(ctx, tx, providerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) RecordProviderWideModelFailure(ctx context.Context, providerID, modelID string, status *int, message string, policy FailurePolicy) error {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var failures int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO provider_model_global_state (
			provider_id,model_id,consecutive_failures,last_error,last_status_code,failure_count,last_used_at
		) VALUES (?,?,1,?,?,1,CURRENT_TIMESTAMP)
		ON CONFLICT(provider_id,model_id) DO UPDATE SET
			consecutive_failures=consecutive_failures+1,last_error=excluded.last_error,
			last_status_code=excluded.last_status_code,failure_count=failure_count+1,last_used_at=CURRENT_TIMESTAMP
		RETURNING consecutive_failures
	`, providerID, modelID, message, status).Scan(&failures)
	if err != nil {
		return err
	}
	if policy.ForceCooldown || failures >= policy.Threshold {
		duration := policy.Cooldown
		if failures == policy.Threshold && policy.ThresholdCooldown > 0 {
			duration = policy.ThresholdCooldown
		}
		if policy.OverrideCooldown > 0 {
			duration = policy.OverrideCooldown
		}
		until := time.Now().Add(duration).UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `
			UPDATE provider_model_global_state SET cooldown_until=? WHERE provider_id=? AND model_id=?
		`, until, providerID, modelID); err != nil {
			return err
		}
	}
	if err := pruneProviderWideModelStates(ctx, tx, providerID); err != nil {
		return err
	}
	return tx.Commit()
}

func pruneProviderModelStates(ctx context.Context, tx *sql.Tx, providerID string) error {
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM provider_model_state
		WHERE provider_id=? AND last_used_at<?
			AND (cooldown_until IS NULL OR cooldown_until<=?)
	`, providerID, now.Add(-providerModelStateRetention).Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		DELETE FROM provider_model_state
		WHERE provider_id=? AND rowid NOT IN (
			SELECT rowid FROM provider_model_state
			WHERE provider_id=?
			ORDER BY COALESCE(last_used_at,'') DESC,rowid DESC
			LIMIT ?
		)
	`, providerID, providerID, maxProviderModelStatesPerProvider)
	return err
}

func pruneProviderWideModelStates(ctx context.Context, tx *sql.Tx, providerID string) error {
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM provider_model_global_state
		WHERE provider_id=? AND last_used_at<?
			AND (cooldown_until IS NULL OR cooldown_until<=?)
	`, providerID, now.Add(-providerModelStateRetention).Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		DELETE FROM provider_model_global_state
		WHERE provider_id=? AND rowid NOT IN (
			SELECT rowid FROM provider_model_global_state
			WHERE provider_id=?
			ORDER BY COALESCE(last_used_at,'') DESC,rowid DESC
			LIMIT ?
		)
	`, providerID, providerID, maxProviderModelStatesPerProvider)
	return err
}

func (s *Store) ResetProviderModelState(ctx context.Context, providerID, keyID, modelID string) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE provider_model_state
		SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL
		WHERE provider_id=? AND key_id=? AND model_id=?
	`, strings.TrimSpace(providerID), strings.TrimSpace(keyID), strings.TrimSpace(modelID))
	return requireRowsAffected(result, err)
}

func (s *Store) ResetProviderModelStates(ctx context.Context, providerID, modelID string) error {
	providerID = strings.TrimSpace(providerID)
	modelID = strings.TrimSpace(modelID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	keyResult, err := tx.ExecContext(ctx, `
		UPDATE provider_model_state
		SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL
		WHERE provider_id=? AND model_id=?
	`, providerID, modelID)
	if err != nil {
		return err
	}
	providerResult, err := tx.ExecContext(ctx, `
		UPDATE provider_model_global_state
		SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL
		WHERE provider_id=? AND model_id=?
	`, providerID, modelID)
	if err != nil {
		return err
	}
	keyRows, err := keyResult.RowsAffected()
	if err != nil {
		return err
	}
	providerRows, err := providerResult.RowsAffected()
	if err != nil {
		return err
	}
	if keyRows+providerRows == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func nullableStoreTime(value sql.NullString) *time.Time {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	parsed := parseTime(value.String)
	return &parsed
}
