package proxy

import (
	"context"
	"sync"
)

type requestLimiter struct {
	global chan struct{}

	mu        sync.Mutex
	changed   chan struct{}
	providers map[string]*limiterEntry
	keys      map[string]*limiterEntry
	providerN int
	keyN      int
}

type limiterEntry struct {
	limit chan struct{}
	refs  int
}

func newRequestLimiter(global, provider, key int) *requestLimiter {
	return &requestLimiter{
		global:    makeLimit(global),
		changed:   make(chan struct{}),
		providers: make(map[string]*limiterEntry),
		keys:      make(map[string]*limiterEntry),
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
	releaseEntries := func() {
		l.releaseEntry(l.keys, keyID, key)
		l.releaseEntry(l.providers, providerID, provider)
	}
	acquired := make([]chan struct{}, 0, 3)
	// Acquire the narrowest limits first and never wait while holding one.
	for _, limit := range []chan struct{}{entryLimit(key), entryLimit(provider), l.global} {
		if limit == nil {
			continue
		}
		select {
		case limit <- struct{}{}:
			acquired = append(acquired, limit)
		default:
			releaseLimits(acquired)
			releaseEntries()
			return func() {}, false
		}
	}
	return func() {
		releaseLimits(acquired)
		releaseEntries()
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

func (l *requestLimiter) provider(id string) *limiterEntry {
	if l.providerN <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry := l.providers[id]; entry != nil {
		entry.refs++
		return entry
	}
	entry := &limiterEntry{limit: makeLimit(l.providerN), refs: 1}
	l.providers[id] = entry
	return entry
}

func (l *requestLimiter) key(id string) *limiterEntry {
	if l.keyN <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry := l.keys[id]; entry != nil {
		entry.refs++
		return entry
	}
	entry := &limiterEntry{limit: makeLimit(l.keyN), refs: 1}
	l.keys[id] = entry
	return entry
}

func (l *requestLimiter) releaseEntry(entries map[string]*limiterEntry, id string, entry *limiterEntry) {
	if entry == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry.refs--
	if entry.refs == 0 && entries[id] == entry {
		delete(entries, id)
	}
}

func entryLimit(entry *limiterEntry) chan struct{} {
	if entry == nil {
		return nil
	}
	return entry.limit
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
