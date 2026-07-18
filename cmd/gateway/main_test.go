package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"local-ai-gateway/internal/config"
)

func TestAdminLoginURLUsesFragmentToken(t *testing.T) {
	cfg := config.Default()
	cfg.Server.AdminToken = "gat-test token"

	got := adminLoginURL(cfg)
	if !strings.HasPrefix(got, "http://localhost:18787/admin#token=") {
		t.Fatalf("admin login url = %q", got)
	}
	if strings.Contains(got, "?") {
		t.Fatalf("admin token should not be placed in query string: %q", got)
	}
	if strings.Contains(got, "gat-test token") {
		t.Fatalf("admin token should be escaped in URL: %q", got)
	}
}

func TestPerformRestartRunsStepsInOrder(t *testing.T) {
	var calls []string
	step := func(name string) func() error {
		return func() error {
			calls = append(calls, name)
			return nil
		}
	}
	start := func() (func(), error) {
		calls = append(calls, "start")
		return func() { calls = append(calls, "abort") }, nil
	}
	started, err := performRestart(start, step("shutdown"), step("database"), step("lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !started {
		t.Fatal("replacement process was not marked as started")
	}
	want := []string{"start", "shutdown", "database", "lock"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("restart calls = %v, want %v", calls, want)
	}
}

func TestPerformRestartCompletesCleanupAfterFailure(t *testing.T) {
	restartErr := errors.New("database flush failed")
	var calls []string
	started, err := performRestart(
		func() (func(), error) {
			calls = append(calls, "start")
			return func() { calls = append(calls, "abort") }, nil
		},
		func() error { calls = append(calls, "shutdown"); return nil },
		func() error { calls = append(calls, "database"); return restartErr },
		func() error { calls = append(calls, "lock"); return nil },
	)
	if !started {
		t.Fatal("replacement process was not marked as started")
	}
	if !errors.Is(err, restartErr) {
		t.Fatalf("performRestart() error = %v, want %v", err, restartErr)
	}
	want := []string{"start", "shutdown", "database", "lock"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("restart calls = %v, want %v", calls, want)
	}
}

func TestPerformRestartLeavesCurrentProcessRunningWhenReplacementCannotStart(t *testing.T) {
	startErr := errors.New("access denied")
	cleanupCalled := false
	started, err := performRestart(
		func() (func(), error) { return nil, startErr },
		func() error { cleanupCalled = true; return nil },
		func() error { cleanupCalled = true; return nil },
		func() error { cleanupCalled = true; return nil },
	)
	if !errors.Is(err, startErr) {
		t.Fatalf("performRestart() error = %v, want %v", err, startErr)
	}
	if started {
		t.Fatal("replacement process was marked as started after launch failure")
	}
	if cleanupCalled {
		t.Fatal("current process cleanup ran after replacement failed to start")
	}
}
