package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"local-ai-gateway/internal/desktop"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
)

const backupTestPassphrase = "correct horse battery staple"

func TestOfflineMaintenanceRefusesGatewayLockWithoutChangingData(t *testing.T) {
	for _, command := range []string{"restore", "rotate-key"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			database, secret := filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key")
			st := createBackupTestStore(t, database, secret)
			defer st.Close()
			lock, exists, err := desktop.AcquireLock(desktop.LockPath(database), false)
			if err != nil || exists {
				t.Fatalf("acquire gateway lock: exists=%v err=%v", exists, err)
			}
			defer lock.Release()
			beforeDB, _ := os.ReadFile(database)
			beforeKey, _ := os.ReadFile(secret)
			beforeLock, _ := os.ReadFile(desktop.LockPath(database))
			t.Setenv("GATEWAY_BACKUP_PASSPHRASE", backupTestPassphrase)
			args := []string{command, "--database", database, "--secret", secret}
			if command == "restore" {
				args = append(args, "--input", filepath.Join(dir, "missing-backup.zip"), "--force")
			}
			if err := run(args); err == nil || !strings.Contains(err.Error(), "stop gateway") {
				t.Fatalf("maintenance must refuse a live lock before touching storage: %v", err)
			}
			for path, before := range map[string][]byte{database: beforeDB, secret: beforeKey, desktop.LockPath(database): beforeLock} {
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("locked file changed: %s (%v)", path, err)
				}
			}
			keys, err := st.ListKeys(context.Background())
			if err != nil || len(keys) != 1 || keys[0].Secret != "sk-backup-test" {
				t.Fatalf("running gateway secrets changed: keys=%v err=%v", keys, err)
			}
		})
	}
}

func TestMaintenanceDoesNotStealUnverifiedStaleLock(t *testing.T) {
	database := filepath.Join(t.TempDir(), "gateway.db")
	path := desktop.LockPath(database)
	// A partial lock may be from a process currently starting, not a crash.
	if err := os.WriteFile(path, []byte("pid="), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireMaintenanceLock(database); err == nil {
		t.Fatal("maintenance stole an unverified lock")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "pid=" {
		t.Fatalf("existing lock changed: %q, %v", data, err)
	}
}

func TestMaintenanceReleasesLockOnErrors(t *testing.T) {
	for _, command := range []string{"restore", "rotate-key"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			database := filepath.Join(dir, "database-directory")
			if err := os.Mkdir(database, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GATEWAY_BACKUP_PASSPHRASE", backupTestPassphrase)
			args := []string{command, "--database", database, "--secret", filepath.Join(dir, "secret.key")}
			if command == "restore" {
				args = append(args, "--input", filepath.Join(dir, "missing.zip"))
			}
			if err := run(args); err == nil {
				t.Fatal("expected maintenance failure")
			}
			if _, err := os.Stat(desktop.LockPath(database)); !os.IsNotExist(err) {
				t.Fatalf("lock retained after failure: %v", err)
			}
		})
	}
}

func TestRunValidatesRestoreArguments(t *testing.T) {
	t.Setenv("GATEWAY_BACKUP_PASSPHRASE", "")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing command", want: "usage:"},
		{name: "unknown command", args: []string{"unknown"}, want: "usage:"},
		{name: "missing input", args: []string{"restore"}, want: "--input is required"},
		{name: "missing passphrase", args: []string{"restore", "--input", "backup.zip"}, want: "GATEWAY_BACKUP_PASSPHRASE is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := run(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("run() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestRunRestoresPortableBackup(t *testing.T) {
	sourceDir := filepath.Join(t.TempDir(), "source")
	sourceDB := filepath.Join(sourceDir, "gateway.db")
	sourceSecret := filepath.Join(sourceDir, "secret.key")
	st := createBackupTestStore(t, sourceDB, sourceSecret)
	backup, err := st.CreatePortableBackup(context.Background(), backupTestPassphrase)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(t.TempDir(), "restored")
	targetDB := filepath.Join(targetDir, "gateway.db")
	targetSecret := filepath.Join(targetDir, "secret.key")
	t.Setenv("GATEWAY_BACKUP_PASSPHRASE", backupTestPassphrase)
	if err := run([]string{"restore", "--input", backup.Path, "--database", targetDB, "--secret", targetSecret}); err != nil {
		t.Fatal(err)
	}
	assertBackupTestKey(t, targetDB, targetSecret)
}

func TestRotateKeyPreservesSecretsAndCreatesSnapshot(t *testing.T) {
	dir := t.TempDir()
	database := filepath.Join(dir, "gateway.db")
	secret := filepath.Join(dir, "secret.key")
	st := createBackupTestStore(t, database, secret)
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}
	if err := rotateKey([]string{"--database", database, "--secret", secret}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) {
		t.Fatal("master key bundle did not change")
	}
	snapshots, err := filepath.Glob(filepath.Join(dir, "backups", "gateway-pre-key-rotation-*.db"))
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("pre-rotation snapshots = %d, want 1", len(snapshots))
	}
	assertBackupTestKey(t, database, secret)
}

func createBackupTestStore(t *testing.T, database, secret string) *store.Store {
	t.Helper()
	t.Setenv("GATEWAY_MASTER_KEY", "")
	st, err := store.Open(database, secret)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{
		ID: "provider", Name: "Provider", Type: model.ProviderOpenAICompatible,
		BaseURL: "https://api.example/v1", Enabled: true,
	}); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	if _, err := st.UpsertKey(ctx, model.Key{
		ID: "provider-key", ProviderID: "provider", Name: "Primary", Secret: "sk-backup-test", Enabled: true,
	}); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	return st
}

func assertBackupTestKey(t *testing.T, database, secret string) {
	t.Helper()
	st, err := store.Open(database, secret)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	keys, err := st.ListKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Secret != "sk-backup-test" {
		t.Fatalf("restored keys = %#v", keys)
	}
}
