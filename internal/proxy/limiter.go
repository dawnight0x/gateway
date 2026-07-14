package proxy

import (
	"context"
	"sync"
)

type requestLimiter struct {
	global chan struct{}

	mu        sync.Mutex
	changed   chan struct{}
	providers map[string]chan struct{}
	keys      map[string]chan struct{}
	providerN int
	keyN      int
}

func newRequestLimiter(global, provider, key int) *requestLimiter {
	return &requestLimiter{
		global:    makeLimit(global),
		changed:   make(chan struct{}),
		providers: make(map[string]chan struct{}),
		keys:      make(map[string]chan struct{}),
		providerN: provider,
		keyN:      key,
	}
}

func (l *requestLimiter) acquire(ctx context.Context, providerID, keyID string) (func(), bool) {
	for {
		// Capture the change signal before attempting acquisition so a release that
		// races between a failed tryAcquire and the wait below is not lost.
		signal := l.changeSignal()
		if release, ok := l.tryAcquire(providerID, keyID); ok {
			return release, true
		}
		if !l.wait(ctx, signal) {
			return func() {}, false
		}
	}
}

func (l *requestLimiter) tryAcquire(providerID, keyID string) (func(), bool) {
	provider := l.provider(providerID)
	key := l.key(keyID)
	acquired := make([]chan struct{}, 0, 3)
	// Acquire the narrowest limits first and never wait while holding one.
	for _, limit := range []chan struct{}{key, provider, l.global} {
		if limit == nil {
			continue
		}
		select {
		case limit <- struct{}{}:
			acquired = append(acquired, limit)
		default:
			releaseLimits(acquired)
			return func() {}, false
		}
	}
	return func() {
		releaseLimits(acquired)
		l.signalChanged()
	}, true
}

// changeSignal returns the current broadcast channel. Callers should capture it before a
// tryAcquire attempt so a concurrent release cannot be missed between the failed attempt and wait.
func (l *requestLimiter) changeSignal() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.changed
}

func (l *requestLimiter) wait(ctx context.Context, signal <-chan struct{}) bool {
	select {
	case <-signal:
		return true
	case <-ctx.Done():
		return false
	}
}

// signalChanged wakes every waiter so they can all re-attempt acquisition. Closing and
// replacing the channel broadcasts to all current waiters, avoiding the starvation that a
// single-slot signal caused when a woken waiter lost the freed slot to a racing tryAcquire.
func (l *requestLimiter) signalChanged() {
	l.mu.Lock()
	close(l.changed)
	l.changed = make(chan struct{})
	l.mu.Unlock()
}

func (l *requestLimiter) provider(id string) chan struct{} {
	if l.providerN <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if limit := l.providers[id]; limit != nil {
		return limit
	}
	limit := makeLimit(l.providerN)
	l.providers[id] = limit
	return limit
}

func (l *requestLimiter) key(id string) chan struct{} {
	if l.keyN <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if limit := l.keys[id]; limit != nil {
		return limit
	}
	limit := makeLimit(l.keyN)
	l.keys[id] = limit
	return limit
}

func makeLimit(size int) chan struct{} {
	if size <= 0 {
		return nil
	}
	return make(chan struct{}, size)
}

func releaseLimits(acquired []chan struct{}) {
	for index := len(acquired) - 1; index >= 0; index-- {
		<-acquired[index]
	}
}
