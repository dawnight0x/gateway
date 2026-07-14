package store

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"local-ai-gateway/internal/model"
)

func TestLoadOrCreateKeyUsesExternalMasterKeyWithoutFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	want := bytes.Repeat([]byte{0x42}, 32)
	t.Setenv("GATEWAY_MASTER_KEY", base64.StdEncoding.EncodeToString(want))
	got, err := loadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("key mismatch")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("secret file should not be created when env key is used: %v", err)
	}
}

func TestLoadOrCreateKeyProtectsNewKeyOnWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	key, err := loadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}
	raw := readSecretFile(t, path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(raw, protectedKeyPrefix) {
		t.Fatalf("expected DPAPI-protected key on Windows, got %q", raw)
	}
	if runtime.GOOS != "windows" && strings.HasPrefix(raw, protectedKeyPrefix) {
		t.Fatalf("did not expect DPAPI-protected key on %s", runtime.GOOS)
	}
	again, err := loadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(key) {
		t.Fatal("loaded key changed")
	}
}

func TestLoadOrCreateKeyMigratesPlainKeyOnWindows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.key")
	plain := []byte("01234567890123456789012345678901")
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(plain)), 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := loadOrCreateKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != string(plain) {
		t.Fatal("loaded key mismatch")
	}
	raw := readSecretFile(t, path)
	if runtime.GOOS == "windows" && !strings.HasPrefix(raw, protectedKeyPrefix) {
		t.Fatalf("expected plaintext key to be migrated to DPAPI on Windows, got %q", raw)
	}
}

func TestMasterKeyRotationReencryptsSecretsAndSurvivesReopen(t *testing.T) {
	t.Setenv("GATEWAY_MASTER_KEY", "")
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
	if _, err := st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "rotation-secret", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	oldKey := append([]byte(nil), st.crypto.key...)
	if err := st.RotateMasterKey(ctx); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(oldKey, st.crypto.key) {
		t.Fatal("master key did not change")
	}
	if raw := readSecretFile(t, secretPath); !strings.HasPrefix(raw, keyBundlePrefix) {
		t.Fatalf("secret file is not a crash-safe key bundle: %q", raw)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	keys, err := reopened.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Secret != "rotation-secret" {
		t.Fatalf("keys after rotation = %#v", keys)
	}
}

func readSecretFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}
