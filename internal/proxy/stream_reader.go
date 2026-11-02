package proxy

import (
	"errors"
	"io"
	"sync"
	"time"
)

// streamIdleReader measures upstream inactivity between received bytes, not
// complete SSE lines. Closing the body interrupts a blocked HTTP read.
type streamIdleReader struct {
	body     io.ReadCloser
	timeout  time.Duration
	mu       sync.Mutex
	timer    *time.Timer
	deadline time.Time
	expired  bool
	stopped  bool
}

func newStreamIdleReader(body io.ReadCloser, timeout time.Duration) *streamIdleReader {
	r := &streamIdleReader{body: body, timeout: timeout, deadline: time.Now().Add(timeout)}
	r.mu.Lock()
	r.timer = time.AfterFunc(timeout, r.onIdle)
	r.mu.Unlock()
	return r
}

func (r *streamIdleReader) onIdle() {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	// A read may have extended the deadline while this callback was queued.
	if remaining := time.Until(r.deadline); remaining > 0 {
		r.timer.Reset(remaining)
		r.mu.Unlock()
		return
	}
	r.expired, r.stopped = true, true
	r.mu.Unlock()
	_ = r.body.Close()
}

func (r *streamIdleReader) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.expired {
		return n, errors.New("upstream stream idle timeout")
	}
	if n > 0 && !r.stopped {
		r.deadline = time.Now().Add(r.timeout)
		r.timer.Reset(r.timeout)
	}
	return n, err
}

func (r *streamIdleReader) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	r.timer.Stop()
}
