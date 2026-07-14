package desktop

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPathsUseDataDirFromDB(t *testing.T) {
	db := filepath.Join("state", "gateway.db")
	if got := DataDir(db); got != "state" {
		t.Fatalf("data dir = %q", got)
	}
	if got := LogPath(db); got != filepath.Join("state", "gateway.log") {
		t.Fatalf("log path = %q", got)
	}
	if got := LockPath(db); got != filepath.Join("state", "gateway.lock") {
		t.Fatalf("lock path = %q", got)
	}
}

func TestAcquireLockBlocksSecondInstanceAndReleases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.lock")
	first, exists, err := AcquireLock(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("first lock reported existing instance")
	}
	t.Cleanup(func() { _ = first.Release() })

	second, exists, err := AcquireLock(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if second != nil || !exists {
		t.Fatalf("second lock = %v exists = %v, want existing instance", second, exists)
	}

	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("idempotent release failed: %v", err)
	}
	third, exists, err := AcquireLock(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if exists || third == nil {
		t.Fatalf("third lock = %v exists = %v, want acquired", third, exists)
	}
	_ = third.Release()
}

func TestAcquireLockDoesNotStealFromLiveProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.lock")
	if err := os.WriteFile(path, []byte("pid="+strconv.Itoa(os.Getpid())+"\nstarted_at=2000-01-01T00:00:00Z\ntoken=live\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, exists, err := AcquireLock(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if lock != nil || !exists {
		t.Fatalf("lock = %v exists = %v, want live owner preserved", lock, exists)
	}
}

func TestExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "marker")
	if Exists(path) {
		t.Fatal("missing path reported as existing")
	}
	if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Exists(path) {
		t.Fatal("existing path reported as missing")
	}
}

func TestStartupCommandQuotesExecutable(t *testing.T) {
	cmd, err := startupCommand(filepath.Join("Program Files", "Gateway", "gateway.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cmd, `"`) || !strings.HasSuffix(cmd, `"`) {
		t.Fatalf("startup command = %q, want quoted executable", cmd)
	}
	if !strings.Contains(cmd, "gateway.exe") {
		t.Fatalf("startup command = %q, want executable path", cmd)
	}
}
