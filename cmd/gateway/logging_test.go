package main

import (
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
