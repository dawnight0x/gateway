package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingLogWriterKeepsBoundedBackups(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.log")
	writer, err := newRotatingLogWriter(path, 12, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("first-entry\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("second-entry\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if string(current) != "second-entry\n" || string(backup) != "first-entry\n" {
		t.Fatalf("current = %q, backup = %q", current, backup)
	}
}

func TestRotatingLogWriterRecoversAfterRotationFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")
	writer, err := newRotatingLogWriter(path, 8, 2)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	if _, err := writer.Write([]byte("initial\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".1", []byte("backup\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	blockedTarget := path + ".2"
	if err := os.Mkdir(blockedTarget, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blockedTarget, "keep"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := writer.Write([]byte("rotate\n")); err == nil {
		t.Fatal("rotation unexpectedly succeeded with a non-empty backup target directory")
	}
	writer.maxBytes = 0
	if _, err := writer.Write([]byte("recovered\n")); err != nil {
		t.Fatalf("write after failed rotation: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "initial\nrecovered\n" {
		t.Fatalf("current log = %q", content)
	}
}

func TestRotatingLogWriterRejectsWriteAfterClose(t *testing.T) {
	writer, err := newRotatingLogWriter(filepath.Join(t.TempDir(), "gateway.log"), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("closed")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("write error = %v, want os.ErrClosed", err)
	}
}
