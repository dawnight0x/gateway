package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"local-ai-gateway/internal/model"
)

func openTestStore(t *testing.T, options Options) *Store {
	t.Helper()
	dir := t.TempDir()
	st, err := OpenWithOptions(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"), options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestMaskSecret(t *testing.T) {
	tests := map[string]string{
		"sk-1234567890obsc":  "sk-*****obsc",
		"sk-proj-abcdef1234": "sk-*****1234",
		"abcd1234":           "ab*****34",
		"short":              "sh*****rt",
		"abc":                "***",
		"  sk-trim1234  ":    "sk-*****1234",
	}

	for input, want := range tests {
		if got := MaskSecret(input); got != want {
			t.Fatalf("MaskSecret(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSchemaMigrationsCreateExpectedIndexes(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	var versions int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if versions != len(schemaMigrations) {
		t.Fatalf("schema versions = %d, want %d", versions, len(schemaMigrations))
	}
	var missingChecksums int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE checksum=''`).Scan(&missingChecksums); err != nil {
		t.Fatal(err)
	}
	if missingChecksums != 0 {
		t.Fatalf("migrations without checksums = %d", missingChecksums)
	}

	var indexes int
	if err := st.db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name IN (
			'idx_request_logs_created_at',
			'idx_request_logs_provider_created_at',
			'idx_request_logs_key_created_at'
		)
	`).Scan(&indexes); err != nil {
		t.Fatal(err)
	}
	if indexes != 3 {
		t.Fatalf("request log indexes = %d, want 3", indexes)
	}
}

func TestUpgradeCreatesPreMigrationBackupAndIntegrityCheck(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "gateway.db")
	secretPath := filepath.Join(dir, "secret.key")
	st, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`DROP TABLE provider_models`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`DELETE FROM schema_migrations WHERE version=?`, len(schemaMigrations)); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := OpenWithOptions(dbPath, secretPath, Options{Timezone: "Asia/Singapore", BackupBeforeMigration: true, BackupRetention: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	backups, err := upgraded.ListBackups()
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 || backups[0].Size == 0 {
		t.Fatalf("pre-migration backups = %#v", backups)
	}
	if err := upgraded.IntegrityCheck(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestPortableBackupRestoresDatabaseAndMasterKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "gateway.db")
	secretPath := filepath.Join(dir, "secret.key")
	st, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.com", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "portable-secret", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	backup, err := st.CreatePortableBackup(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(dir, "restored")
	restoredDB := filepath.Join(restoreDir, "gateway.db")
	restoredSecret := filepath.Join(restoreDir, "secret.key")
	if err := RestorePortableBackup(backup.Path, restoredDB, restoredSecret, "wrong password", false); err == nil {
		t.Fatal("expected incorrect passphrase to fail")
	}
	if _, err := os.Stat(restoredDB); !os.IsNotExist(err) {
		t.Fatalf("failed restore created a database: %v", err)
	}
	if err := RestorePortableBackup(backup.Path, restoredDB, restoredSecret, "correct horse battery staple", false); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(restoredDB, restoredSecret)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	keys, err := restored.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Secret != "portable-secret" {
		t.Fatalf("restored keys = %#v", keys)
	}
}

func TestStoreUsesConcurrentSQLiteConnections(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "gateway.db"), filepath.Join(t.TempDir(), "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if got := st.db.Stats().MaxOpenConnections; got != 4 {
		t.Fatalf("max open connections = %d, want 4", got)
	}
}

func TestPruneRequestLogsAppliesMaxEntries(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenWithOptions(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"), Options{LogMaxEntries: 3})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := st.LogRequest(ctx, model.RequestLog{RequestID: string(rune('a' + i)), InboundProtocol: "openai", Status: 200}); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := st.PruneRequestLogs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Fatalf("deleted logs = %d, want 2", deleted)
	}
	logs, err := st.ListLogs(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 3 || logs[0].RequestID != "e" || logs[2].RequestID != "c" {
		t.Fatalf("remaining logs = %#v", logs)
	}
}

func TestPruneRequestLogsAppliesRetention(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenWithOptions(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"), Options{LogRetentionDays: 7})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if err := st.LogRequest(ctx, model.RequestLog{RequestID: "old", InboundProtocol: "openai", Status: 200}); err != nil {
		t.Fatal(err)
	}
	if err := st.flushRequestLogs(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, `UPDATE request_logs SET created_at='2000-01-01 00:00:00'`); err != nil {
		t.Fatal(err)
	}
	deleted, err := st.PruneRequestLogs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted logs = %d, want 1", deleted)
	}
}

func TestDeletingProviderCascadesBalances(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.com", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertBalance(ctx, model.Balance{ProviderID: "provider", Status: "ok"}); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteProvider(ctx, "provider"); err != nil {
		t.Fatal(err)
	}
	balances, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(balances) != 0 {
		t.Fatalf("balances after provider delete = %#v", balances)
	}
}

func TestStatsUsesAvailableKeysAndConfiguredDayBoundary(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenWithOptions(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"), Options{Timezone: "Asia/Singapore"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "enabled", Name: "enabled", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.com", Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "disabled", Name: "disabled", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.com", Enabled: false})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "active", ProviderID: "enabled", Name: "active", Secret: "a", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "cooling", ProviderID: "enabled", Name: "cooling", Secret: "b", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "provider-disabled", ProviderID: "disabled", Name: "provider-disabled", Secret: "c", Enabled: true})
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	if _, err := st.db.ExecContext(ctx, `UPDATE key_state SET cooldown_until=? WHERE key_id='cooling'`, future); err != nil {
		t.Fatal(err)
	}

	location, _ := time.LoadLocation("Asia/Singapore")
	now := time.Now().In(location)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).UTC()
	for _, item := range []struct {
		id      string
		created time.Time
		tokens  int
	}{
		{"before", start.Add(-time.Minute), 100},
		{"inside", start.Add(time.Minute), 7},
	} {
		if _, err := st.db.ExecContext(ctx, `INSERT INTO request_logs (request_id,inbound_protocol,status,latency_ms,total_tokens,created_at) VALUES (?,'openai',200,1,?,?)`, item.id, item.tokens, item.created.Format("2006-01-02 15:04:05")); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := st.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalKeys != 3 || stats.ActiveKeys != 1 || stats.FailedKeys != 1 {
		t.Fatalf("key stats = %#v", stats)
	}
	if stats.TodayRequests != 1 || stats.TodayTokens != 7 {
		t.Fatalf("today stats = %#v", stats)
	}
}

func TestRequestLogsFlushOnClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "gateway.db")
	secretPath := filepath.Join(dir, "secret.key")
	st, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.LogRequest(context.Background(), model.RequestLog{RequestID: "pending", InboundProtocol: "openai", Status: 200}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	logs, err := reopened.ListLogs(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].RequestID != "pending" {
		t.Fatalf("logs = %#v", logs)
	}
}

func TestHealthyKeySuccessesBatchAndRecoveryFlushesImmediately(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "gateway.db"), filepath.Join(t.TempDir(), "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.com", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: true})
	if err := st.RecordSuccess(ctx, "key", false); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordSuccess(ctx, "key", false); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := st.db.QueryRowContext(ctx, `SELECT success_count FROM key_state WHERE key_id='key'`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 0 {
		t.Fatalf("successes were not batched: %d", before)
	}
	if err := st.flushKeySuccesses(ctx); err != nil {
		t.Fatal(err)
	}
	var flushed int
	if err := st.db.QueryRowContext(ctx, `SELECT success_count FROM key_state WHERE key_id='key'`).Scan(&flushed); err != nil {
		t.Fatal(err)
	}
	if flushed != 2 {
		t.Fatalf("flushed successes = %d", flushed)
	}

	policy := FailurePolicy{Threshold: 1, Cooldown: time.Minute}
	if err := st.RecordFailure(ctx, "key", nil, "down", policy); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordSuccess(ctx, "key", true); err != nil {
		t.Fatal(err)
	}
	var failures int
	var cooldown *string
	if err := st.db.QueryRowContext(ctx, `SELECT consecutive_failures,cooldown_until,success_count FROM key_state WHERE key_id='key'`).Scan(&failures, &cooldown, &flushed); err != nil {
		t.Fatal(err)
	}
	if failures != 0 || cooldown != nil || flushed != 3 {
		t.Fatalf("recovery state failures=%d cooldown=%v successes=%d", failures, cooldown, flushed)
	}
}

func TestUpsertKeyKeepsSingleManualPreference(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t, Options{LogRetentionDays: 30, LogMaxEntries: 100000, Timezone: "Asia/Singapore"})
	_, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "first", ProviderID: "provider", Name: "first", Secret: "first-secret", Enabled: true, ManualPreferred: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "second", ProviderID: "provider", Name: "second", Secret: "second-secret", Enabled: true, ManualPreferred: true})
	if err != nil {
		t.Fatal(err)
	}
	keys, err := st.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	preferred := ""
	for _, key := range keys {
		if key.ManualPreferred {
			if preferred != "" {
				t.Fatalf("multiple preferred keys: %q and %q", preferred, key.ID)
			}
			preferred = key.ID
		}
	}
	if preferred != "second" {
		t.Fatalf("preferred key = %q, want second", preferred)
	}
}

func TestUpsertKeyRollsBackWhenStateCreationFails(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t, Options{LogRetentionDays: 30, LogMaxEntries: 100000, Timezone: "Asia/Singapore"})
	_, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, `
		CREATE TRIGGER reject_atomic_key_state
		BEFORE INSERT ON key_state
		WHEN NEW.key_id='atomic-key'
		BEGIN
			SELECT RAISE(ABORT, 'state rejected');
		END;
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertKey(ctx, model.Key{ID: "atomic-key", ProviderID: "provider", Name: "atomic", Secret: "secret", Enabled: true}); err == nil {
		t.Fatal("expected key state insertion failure")
	}
	var count int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM keys WHERE id='atomic-key'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("partial key row remained after rollback: count = %d", count)
	}
}

func TestRequestLogBufferDropsOldestEntriesWhenFull(t *testing.T) {
	st := openTestStore(t, Options{LogRetentionDays: 30, LogMaxEntries: 100000, Timezone: "Asia/Singapore"})
	batch := make([]model.RequestLog, maxPendingRequestLogs+17)
	for i := range batch {
		batch[i].RequestID = fmt.Sprintf("request-%d", i)
	}
	st.restoreRequestLogs(batch)
	st.requestLogMu.Lock()
	pendingCount := len(st.pendingRequestLogs)
	oldest := st.pendingRequestLogs[0].RequestID
	st.pendingRequestLogs = nil
	st.requestLogMu.Unlock()
	if pendingCount != maxPendingRequestLogs {
		t.Fatalf("pending logs = %d, want %d", pendingCount, maxPendingRequestLogs)
	}
	if got := oldest; got != "request-17" {
		t.Fatalf("oldest retained request = %q, want request-17", got)
	}
	if dropped := st.droppedRequestLogs.Load(); dropped != 17 {
		t.Fatalf("dropped logs = %d, want 17", dropped)
	}
}
