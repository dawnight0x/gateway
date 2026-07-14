package proxy

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequestLimiterEnforcesPerKeyLimitAndReleases(t *testing.T) {
	limiter := newRequestLimiter(2, 2, 1)
	release, ok := limiter.acquire(context.Background(), "provider", "key")
	if !ok {
		t.Fatal("first request was not admitted")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, admitted := limiter.acquire(ctx, "provider", "key"); admitted {
		t.Fatal("second request exceeded per-key limit")
	}
	release()

	releaseAgain, ok := limiter.acquire(context.Background(), "provider", "key")
	if !ok {
		t.Fatal("capacity was not restored after release")
	}
	releaseAgain()
}

func TestRequestLimiterConcurrentLoadNeverExceedsGlobalLimit(t *testing.T) {
	const limit = int64(8)
	const requests = 256
	limiter := newRequestLimiter(int(limit), 0, 0)
	var active atomic.Int64
	var maximum atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			release, ok := limiter.acquire(context.Background(), "provider", "key")
			if !ok {
				t.Error("request was unexpectedly rejected")
				return
			}
			current := active.Add(1)
			for {
				seen := maximum.Load()
				if current <= seen || maximum.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(100 * time.Microsecond)
			active.Add(-1)
			release()
		}()
	}
	close(start)
	wg.Wait()
	if got := maximum.Load(); got > limit || got < 2 {
		t.Fatalf("maximum concurrency = %d, want 2..%d", got, limit)
	}
}

func TestRequestLimiterCanUseIdleAlternateKey(t *testing.T) {
	limiter := newRequestLimiter(2, 2, 1)
	releaseBusy, ok := limiter.tryAcquire("provider", "busy")
	if !ok {
		t.Fatal("failed to occupy first key")
	}
	defer releaseBusy()
	if _, ok := limiter.tryAcquire("provider", "busy"); ok {
		t.Fatal("busy key admitted a second request")
	}
	releaseIdle, ok := limiter.tryAcquire("provider", "idle")
	if !ok {
		t.Fatal("idle alternate key was ignored")
	}
	releaseIdle()
}

func BenchmarkRequestLimiterParallel(b *testing.B) {
	limiter := newRequestLimiter(64, 16, 4)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			release, ok := limiter.acquire(context.Background(), "provider", "key")
			if !ok {
				b.Fatal("request was unexpectedly rejected")
			}
			release()
		}
	})
}
