package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const upstreamAffinityTTL = 30 * 24 * time.Hour

type UpstreamAffinity struct {
	ProviderID    string
	KeyID         string
	UpstreamModel string
}

func (s *Store) PutUpstreamAffinity(ctx context.Context, resourceIDs []string, providerID, keyID, upstreamModel string) error {
	if strings.TrimSpace(providerID) == "" || strings.TrimSpace(keyID) == "" {
		return nil
	}
	expires := time.Now().Add(upstreamAffinityTTL).UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, resourceID := range resourceIDs {
		resourceID = strings.TrimSpace(resourceID)
		if resourceID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO upstream_affinity (resource_id,provider_id,key_id,upstream_model,expires_at,updated_at)
			VALUES (?,?,?,?,?,CURRENT_TIMESTAMP)
			ON CONFLICT(resource_id) DO UPDATE SET
				provider_id=excluded.provider_id,key_id=excluded.key_id,upstream_model=excluded.upstream_model,
				expires_at=excluded.expires_at,updated_at=CURRENT_TIMESTAMP
		`, resourceID, providerID, keyID, strings.TrimSpace(upstreamModel), expires); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM upstream_affinity WHERE expires_at<=?`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetUpstreamAffinity(ctx context.Context, resourceID string) (UpstreamAffinity, bool, error) {
	var affinity UpstreamAffinity
	err := s.db.QueryRowContext(ctx, `
		SELECT provider_id,key_id,upstream_model FROM upstream_affinity
		WHERE resource_id=? AND expires_at>?
	`, strings.TrimSpace(resourceID), time.Now().UTC().Format(time.RFC3339)).Scan(&affinity.ProviderID, &affinity.KeyID, &affinity.UpstreamModel)
	if errors.Is(err, sql.ErrNoRows) {
		return UpstreamAffinity{}, false, nil
	}
	if err != nil {
		return UpstreamAffinity{}, false, err
	}
	return affinity, true, nil
}
