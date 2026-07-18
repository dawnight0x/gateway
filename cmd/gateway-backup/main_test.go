package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
)

const backupTestPassphrase = "correct horse battery staple"

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
