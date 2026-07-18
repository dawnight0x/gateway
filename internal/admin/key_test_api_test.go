package admin

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitForRetryReturnsWhenDelayExpires(t *testing.T) {
	if err := waitForRetry(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForRetryReturnsPromptlyWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := waitForRetry(ctx, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForRetry() error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("canceled retry wait took %s", elapsed)
	}
}
