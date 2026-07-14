package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotateGatewayKeyInvalidatesOldKey(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "gateway.db"), filepath.Join(t.TempDir(), "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	created, err := st.CreateGatewayKey(ctx, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	if created.Plaintext == "" {
		t.Fatal("expected created plaintext")
	}
	if !strings.HasPrefix(created.Plaintext, "sk-") {
		t.Fatalf("created plaintext prefix = %q", created.Plaintext)
	}
	ok, err := st.VerifyGatewayKey(ctx, created.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected original key to verify before rotation")
	}

	rotated, err := st.RotateGatewayKey(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.ID != created.ID {
		t.Fatalf("id changed: %s -> %s", created.ID, rotated.ID)
	}
	if rotated.Plaintext == "" || rotated.Plaintext == created.Plaintext {
		t.Fatalf("unexpected rotated plaintext: %q", rotated.Plaintext)
	}
	if !strings.HasPrefix(rotated.Plaintext, "sk-") {
		t.Fatalf("rotated plaintext prefix = %q", rotated.Plaintext)
	}
	if rotated.KeyHint == created.KeyHint {
		t.Fatal("expected key hint to change")
	}
	ok, err = st.VerifyGatewayKey(ctx, created.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected old key to be invalid after rotation")
	}
	ok, err = st.VerifyGatewayKey(ctx, rotated.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected rotated key to verify")
	}
}

func TestGatewayKeyUsageFlushesBeforeListing(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	created, err := st.CreateGatewayKey(ctx, "usage")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		ok, err := st.VerifyGatewayKey(ctx, created.Plaintext)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("expected gateway key to verify")
		}
	}

	items, err := st.ListGatewayKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RequestCount != 3 || items[0].LastUsedAt == nil {
		t.Fatalf("gateway key usage = %#v", items)
	}
}

func TestGatewayKeyUsageFlushesOnClose(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "gateway.db")
	secretPath := filepath.Join(dir, "secret.key")
	st, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	created, err := st.CreateGatewayKey(ctx, "close")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if ok, err := st.VerifyGatewayKey(ctx, created.Plaintext); err != nil || !ok {
			t.Fatalf("verify ok=%v err=%v", ok, err)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dbPath, secretPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	items, err := reopened.ListGatewayKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].RequestCount != 2 {
		t.Fatalf("persisted gateway key usage = %#v", items)
	}
}
