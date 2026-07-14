package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BackupInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"-"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
	Kind      string    `json:"kind"`
}

func (s *Store) CreateBackup(ctx context.Context, label string) (BackupInfo, error) {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	if err := s.flushGatewayUsage(ctx); err != nil {
		return BackupInfo{}, err
	}
	if err := s.flushKeySuccesses(ctx); err != nil {
		return BackupInfo{}, err
	}
	if err := s.flushRequestLogs(ctx); err != nil {
		return BackupInfo{}, err
	}

	dir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return BackupInfo{}, err
	}
	label = sanitizeBackupLabel(label)
	name := fmt.Sprintf("gateway-%s-%s.db", label, time.Now().UTC().Format("20060102T150405.000000000Z"))
	path := filepath.Join(dir, name)
	query := `VACUUM INTO '` + strings.ReplaceAll(path, `'`, `''`) + `'`
	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return BackupInfo{}, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return BackupInfo{}, err
	}
	if err := s.pruneBackupsLocked(); err != nil {
		return BackupInfo{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return BackupInfo{}, err
	}
	return BackupInfo{Name: name, Path: path, Size: info.Size(), CreatedAt: info.ModTime().UTC(), Kind: "database"}, nil
}

func (s *Store) ListBackups() ([]BackupInfo, error) {
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()
	return s.listBackupsLocked()
}

func (s *Store) IntegrityCheck(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(result), "ok") {
			return fmt.Errorf("database integrity check failed: %s", result)
		}
	}
	return rows.Err()
}

func (s *Store) Checkpoint(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func (s *Store) pruneBackupsLocked() error {
	if s.options.BackupRetention <= 0 {
		return nil
	}
	items, err := s.listBackupsLocked()
	if err != nil {
		return err
	}
	if len(items) <= s.options.BackupRetention {
		return nil
	}
	for _, item := range items[s.options.BackupRetention:] {
		if err := os.Remove(item.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *Store) listBackupsLocked() ([]BackupInfo, error) {
	dir := filepath.Join(filepath.Dir(s.path), "backups")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "gateway-") || !(strings.HasSuffix(entry.Name(), ".db") || strings.HasSuffix(entry.Name(), ".zip")) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		kind := "database"
		if strings.HasSuffix(entry.Name(), ".zip") {
			kind = "portable"
		}
		items = append(items, BackupInfo{Name: entry.Name(), Path: filepath.Join(dir, entry.Name()), Size: info.Size(), CreatedAt: info.ModTime().UTC(), Kind: kind})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func sanitizeBackupLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	var out strings.Builder
	for _, r := range label {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			out.WriteRune(r)
		}
	}
	if out.Len() == 0 {
		return "manual"
	}
	return out.String()
}
