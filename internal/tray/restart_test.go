package tray

import (
	"errors"
	"testing"
)

func TestRestartAndQuitOnlyQuitsAfterSuccessfulRestart(t *testing.T) {
	restartErr := errors.New("start replacement")
	quitCalled := false
	if err := restartAndQuit(func() error { return restartErr }, func() { quitCalled = true }); !errors.Is(err, restartErr) {
		t.Fatalf("restartAndQuit() error = %v, want %v", err, restartErr)
	}
	if quitCalled {
		t.Fatal("quit called after failed restart")
	}

	if err := restartAndQuit(func() error { return nil }, func() { quitCalled = true }); err != nil {
		t.Fatal(err)
	}
	if !quitCalled {
		t.Fatal("quit was not called after successful restart")
	}
}

func TestRestartAndQuitRejectsMissingRestart(t *testing.T) {
	if err := restartAndQuit(nil, nil); err == nil {
		t.Fatal("restartAndQuit() accepted a missing restart callback")
	}
}
