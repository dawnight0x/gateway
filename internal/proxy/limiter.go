package proxy

import (
	"context"
	"sync"
)

type requestLimiter struct {
	global  chan struct{}
	changed chan struct{}

	mu        sync.Mutex
	providers map[string]chan struct{}
	keys      map[string]chan struct{}
	providerN int
	keyN      int
}

func newRequestLimiter(global, provider, key int) *requestLimiter {
	return &requestLimiter{
		global:    makeLimit(global),
		changed:   make(chan struct{}, 1),
		providers: make(map[string]chan struct{}),
		keys:      make(map[string]chan struct{}),
		providerN: provider,
		keyN:      key,
	}
}

func (l *requestLimiter) acquire(ctx context.Context, providerID, keyID string) (func(), bool) {
	for {
		if release, ok := l.tryAcquire(providerID, keyID); ok {
			return release, true
		}
		if !l.wait(ctx) {
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

func (l *requestLimiter) wait(ctx context.Context) bool {
	select {
	case <-l.changed:
		return true
	case <-ctx.Done():
		return false
	}
}

func (l *requestLimiter) signalChanged() {
	select {
	case l.changed <- struct{}{}:
	default:
	}
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
