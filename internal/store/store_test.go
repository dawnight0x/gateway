package store

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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

func TestPortableBackupV2AuthenticatesDatabaseAndManifest(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	backup, err := st.CreatePortableBackup(context.Background(), "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name  string
		entry string
		edit  func([]byte) []byte
	}{
		{name: "database", entry: "database.enc", edit: func(data []byte) []byte {
			out := append([]byte(nil), data...)
			out[len(out)-1] ^= 0x01
			return out
		}},
		{name: "manifest", entry: "manifest.json", edit: func(data []byte) []byte {
			out := append([]byte(nil), data...)
			marker := []byte(`"databaseSha256": "`)
			index := bytes.Index(out, marker)
			if index < 0 {
				t.Fatal("manifest checksum field not found")
			}
			index += len(marker)
			if out[index] == '0' {
				out[index] = '1'
			} else {
				out[index] = '0'
			}
			return out
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			tampered := filepath.Join(dir, "tampered-"+test.name+".zip")
			rewriteZipEntry(t, backup.Path, tampered, test.entry, test.edit)
			restoreDir := filepath.Join(dir, "restore-"+test.name)
			err := RestorePortableBackup(tampered, filepath.Join(restoreDir, "gateway.db"), filepath.Join(restoreDir, "secret.key"), "correct horse battery staple", false)
			if err == nil {
				t.Fatal("tampered backup restored successfully")
			}
			if _, statErr := os.Stat(filepath.Join(restoreDir, "gateway.db")); !os.IsNotExist(statErr) {
				t.Fatalf("tampered restore wrote database: %v", statErr)
			}
		})
	}
}

func TestOpenRecoversInterruptedPortableRestore(t *testing.T) {
	dir := t.TempDir()
	databasePath := filepath.Join(dir, "gateway.db")
	secretPath := filepath.Join(dir, "secret.key")
	st, err := Open(databasePath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	originalDatabase, _ := os.ReadFile(databasePath)
	originalSecret, _ := os.ReadFile(secretPath)
	databaseBackup := databasePath + ".restore-old-test"
	secretBackup := secretPath + ".restore-old-test"
	if err := os.WriteFile(databaseBackup, originalDatabase, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secretBackup, originalSecret, 0o600); err != nil {
		t.Fatal(err)
	}
	absoluteDatabase, _ := filepath.Abs(databasePath)
	absoluteSecret, _ := filepath.Abs(secretPath)
	journal, _ := json.Marshal(restoreTransaction{
		Database: restoreTransactionFile{Target: absoluteDatabase, Backup: databaseBackup, Existed: true},
		Secret:   restoreTransactionFile{Target: absoluteSecret, Backup: secretBackup, Existed: true},
	})
	if err := os.WriteFile(restoreJournalPath(absoluteDatabase), journal, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(databasePath, []byte("partial restore"), 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := Open(databasePath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := recovered.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(restoreJournalPath(absoluteDatabase)); !os.IsNotExist(err) {
		t.Fatalf("restore journal was not removed: %v", err)
	}
}

func rewriteZipEntry(t *testing.T, source, target, entryName string, edit func([]byte) []byte) {
	t.Helper()
	input, err := zip.OpenReader(source)
	if err != nil {
		t.Fatal(err)
	}
	defer input.Close()
	outputFile, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	output := zip.NewWriter(outputFile)
	for _, item := range input.File {
		reader, err := item.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if item.Name == entryName {
			data = edit(data)
		}
		writer, err := output.CreateHeader(&zip.FileHeader{Name: item.Name, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
	if err := outputFile.Close(); err != nil {
		t.Fatal(err)
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
		tokens := item.tokens
		if err := st.LogRequest(ctx, model.RequestLog{RequestID: item.id, InboundProtocol: "openai", Status: 200, LatencyMS: 1, TotalTokens: &tokens, CreatedAt: item.created}); err != nil {
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

func TestStatsRetainedWhenDetailedRequestLoggingIsDisabled(t *testing.T) {
	dir := t.TempDir()
	st, err := OpenWithOptions(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"), Options{Timezone: "Asia/Singapore", DisableRequestLogging: true})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	tokens := 9
	if err := st.LogRequest(context.Background(), model.RequestLog{RequestID: "metrics-only", InboundProtocol: "openai", Status: 200, TotalTokens: &tokens}); err != nil {
		t.Fatal(err)
	}
	stats, err := st.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.TodayRequests != 1 || stats.TodayTokens != 9 || len(stats.Recent) != 0 {
		t.Fatalf("metrics-only stats = %#v", stats)
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

func TestProviderModelInventoryPersistsAndDiscoveryFailureRetainsLastSuccess(t *testing.T) {
	st := openTestStore(t, Options{Timezone: "Asia/Singapore"})
	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{ID: "inventory", Name: "inventory", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceProviderModels(ctx, "inventory", []string{"model-b", "model-a", "model-a"}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordProviderModelDiscoverySuccess(ctx, "inventory", 2); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordProviderModelDiscoveryFailure(ctx, "inventory", "network_error", "temporary outage"); err != nil {
		t.Fatal(err)
	}

	inventory, err := st.ListProviderModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := inventory["inventory"]; len(got) != 2 || got[0] != "model-a" || got[1] != "model-b" {
		t.Fatalf("inventory = %#v", got)
	}
	discoveries, err := st.ListProviderModelDiscoveries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	state := discoveries["inventory"]
	if state.Status != "network_error" || state.ModelCount != 2 || state.LastSuccessAt == nil || state.LastError != "temporary outage" {
		t.Fatalf("discovery state = %#v", state)
	}
}

func TestModelRouteRoundTripPreservesModelAndProviderPriorityOrder(t *testing.T) {
	st := openTestStore(t, Options{Timezone: "Asia/Singapore"})
	ctx := context.Background()
	for _, provider := range []model.Provider{
		{ID: "high", Name: "high", Type: model.ProviderOpenAICompatible, BaseURL: "https://high.test", Priority: 10, Enabled: true},
		{ID: "low", Name: "low", Type: model.ProviderOpenAICompatible, BaseURL: "https://low.test", Priority: 1, Enabled: true},
	} {
		if _, err := st.UpsertProvider(ctx, provider); err != nil {
			t.Fatal(err)
		}
	}
	route, err := st.UpsertModelRoute(ctx, model.ModelRoute{
		ID: "coding-auto", Name: "Coding", Enabled: true,
		Models: []model.ModelRouteModel{
			{Name: "fallback", Priority: 20, Enabled: true, Targets: []model.ModelRouteTarget{{ProviderID: "low", UpstreamModel: "fallback", Enabled: true}}},
			{Name: "primary", Priority: 100, Enabled: true, Targets: []model.ModelRouteTarget{
				{ProviderID: "low", UpstreamModel: "primary", Enabled: true},
				{ProviderID: "high", UpstreamModel: "primary", Enabled: true},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(route.Models) != 2 || route.Models[0].Name != "primary" || route.Models[1].Name != "fallback" {
		t.Fatalf("route models = %#v", route.Models)
	}
	if got := route.Models[0].Targets; len(got) != 2 || got[0].ProviderID != "high" || got[1].ProviderID != "low" {
		t.Fatalf("primary targets = %#v", got)
	}
	if err := st.DeleteModelRoute(ctx, route.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetModelRoute(ctx, route.ID); err != sql.ErrNoRows {
		t.Fatalf("deleted route error = %v", err)
	}
}
