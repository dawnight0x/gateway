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

func TestCreateGatewayKeyDisambiguatesCollidingIDs(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "gateway.db"), filepath.Join(t.TempDir(), "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	// The same name slugs to the same base ID, so the second create must fall back to a
	// random suffix instead of failing on the primary-key collision.
	first, err := st.CreateGatewayKey(ctx, "duplicate")
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.CreateGatewayKey(ctx, "duplicate")
	if err != nil {
		t.Fatalf("second create with colliding name failed: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("expected distinct ids, both were %q", first.ID)
	}
	if !strings.HasPrefix(second.ID, first.ID+"-") {
		t.Fatalf("second id %q should extend base %q with a suffix", second.ID, first.ID)
	}

	items, err := st.ListGatewayKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 persisted keys, got %d", len(items))
	}
}
