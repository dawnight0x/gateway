package router

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestCandidatesPreferManualThenPriority(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "low", Name: "low", Type: model.ProviderOpenAICompatible, BaseURL: "http://low", Priority: 1, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "high", Name: "high", Type: model.ProviderOpenAICompatible, BaseURL: "http://high", Priority: 10, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "low-key", ProviderID: "low", Name: "low", Secret: "sk-low", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "high-key", ProviderID: "high", Name: "high", Secret: "sk-high", Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "preferred", ProviderID: "low", Name: "preferred", Secret: "sk-pref", Enabled: true, ManualPreferred: true})

	rt := New(st, config.Default().Routing)
	items, err := rt.Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if got := items[0].ID; got != "preferred" {
		t.Fatalf("first key = %s, want preferred", got)
	}
	if got := items[1].ID; got != "high-key" {
		t.Fatalf("second key = %s, want high-key", got)
	}
}

func TestCandidatesKeepStableKeyPriorityWithoutRoundRobin(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: "http://newapi", Priority: 10, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key1", ProviderID: "newapi", Name: "key1", Secret: "sk-1", Priority: 2, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key2", ProviderID: "newapi", Name: "key2", Secret: "sk-2", Priority: 1, Enabled: true})

	rt := New(st, config.Default().Routing)
	for i := 0; i < 5; i++ {
		items, err := rt.Candidates(ctx, "gpt", ProtocolOpenAI)
		if err != nil {
			t.Fatal(err)
		}
		if got := items[0].ID; got != "key1" {
			t.Fatalf("iteration %d first key = %s, want key1", i, got)
		}
		if got := items[1].ID; got != "key2" {
			t.Fatalf("iteration %d second key = %s, want key2", i, got)
		}
	}
}

func TestKeyCoolsAfterFailureThreshold(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: "http://p", Enabled: true})
	key, _ := st.UpsertKey(ctx, model.Key{ID: "k", ProviderID: "p", Name: "k", Secret: "sk", Enabled: true})
	cfg := config.Default().Routing
	rt := New(st, cfg)
	for i := 0; i < cfg.FailureThreshold; i++ {
		rt.RecordFailure(ctx, key, Failure{Status: 500, ErrorType: "server_error", Message: "boom"})
		keys, _ := st.ListKeys(ctx)
		key = keys[0]
	}
	keys, _ := st.ListKeys(ctx)
	if keys[0].CooldownUntil == nil {
		t.Fatal("expected cooldown")
	}
	items, err := rt.Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected cooled key to be skipped, got %d", len(items))
	}
}

func TestPreferredKeyFailsOverThenReturnsForRecoveryProbe(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: "http://newapi", Priority: 10, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "sub2api", Name: "sub2api", Type: model.ProviderSub2API, BaseURL: "http://sub2api", Priority: 1, Enabled: true})
	key1, _ := st.UpsertKey(ctx, model.Key{ID: "newapi-key1", ProviderID: "newapi", Name: "key1", Secret: "sk-1", Priority: 2, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "newapi-key2", ProviderID: "newapi", Name: "key2", Secret: "sk-2", Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "sub2api-key1", ProviderID: "sub2api", Name: "sub-key1", Secret: "sk-sub", Priority: 1, Enabled: true})

	cfg := config.Default().Routing
	rt := New(st, cfg)
	for i := 0; i < cfg.FailureThreshold; i++ {
		rt.RecordFailure(ctx, key1, Failure{Status: 500, ErrorType: "server_error", Message: "boom"})
		keys, _ := st.ListKeys(ctx)
		key1 = *findRouterTestKey(keys, "newapi-key1")
	}

	keys, _ := st.ListKeys(ctx)
	key1Ptr := findRouterTestKey(keys, "newapi-key1")
	if key1Ptr == nil || key1Ptr.CooldownUntil == nil {
		t.Fatal("expected newapi key1 to enter cooldown")
	}
	key1 = *key1Ptr
	firstCooldown := time.Until(*key1.CooldownUntil)
	if firstCooldown < 50*time.Second || firstCooldown > 70*time.Second {
		t.Fatalf("first recovery probe cooldown = %s, want about 60s", firstCooldown)
	}

	items, err := rt.Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if got := items[0].ID; got != "newapi-key2" {
		t.Fatalf("first key after key1 cooldown = %s, want newapi-key2", got)
	}
	if got := items[1].ID; got != "sub2api-key1" {
		t.Fatalf("fallback key after same URL keys = %s, want sub2api-key1", got)
	}

	rt.RecordFailure(ctx, key1, Failure{Status: 500, ErrorType: "server_error", Message: "still down"})
	keys, _ = st.ListKeys(ctx)
	key1Ptr = findRouterTestKey(keys, "newapi-key1")
	if key1Ptr == nil || key1Ptr.CooldownUntil == nil {
		t.Fatal("expected newapi key1 to re-enter cooldown")
	}
	secondCooldown := time.Until(*key1Ptr.CooldownUntil)
	if secondCooldown < 290*time.Second || secondCooldown > 310*time.Second {
		t.Fatalf("second recovery probe cooldown = %s, want about 300s", secondCooldown)
	}

	rt.RecordSuccess(ctx, *key1Ptr)
	items, err = rt.Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if got := items[0].ID; got != "newapi-key1" {
		t.Fatalf("first key after successful recovery probe = %s, want newapi-key1", got)
	}
}

func TestProviderFallsBackOnlyAfterSameProviderKeysUnavailable(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: "http://newapi", Priority: 10, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "sub2api", Name: "sub2api", Type: model.ProviderSub2API, BaseURL: "http://sub2api", Priority: 1, Enabled: true})
	key1, _ := st.UpsertKey(ctx, model.Key{ID: "newapi-key1", ProviderID: "newapi", Name: "key1", Secret: "sk-1", Priority: 2, Enabled: true})
	key2, _ := st.UpsertKey(ctx, model.Key{ID: "newapi-key2", ProviderID: "newapi", Name: "key2", Secret: "sk-2", Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "sub2api-key1", ProviderID: "sub2api", Name: "sub-key1", Secret: "sk-sub", Priority: 1, Enabled: true})

	rt := New(st, config.Default().Routing)
	cooldown := time.Now().Add(5 * time.Minute)
	coolKey(ctx, rt, key1, cooldown)
	items, err := rt.Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if got := items[0].ID; got != "newapi-key2" {
		t.Fatalf("first key with only key1 down = %s, want newapi-key2", got)
	}

	coolKey(ctx, rt, key2, cooldown)
	items, err = rt.Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 || items[0].ID != "sub2api-key1" {
		t.Fatalf("candidates after all newapi keys down = %#v, want sub2api-key1 first", items)
	}
}

func TestModelMapSupportsWildcardDefault(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{
		ID:       "newapi",
		Name:     "newapi",
		Type:     model.ProviderNewAPI,
		BaseURL:  "http://newapi",
		Enabled:  true,
		ModelMap: map[string]string{"*": "auto"},
	})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k", ProviderID: "newapi", Name: "k", Secret: "sk", Enabled: true})

	rt := New(st, config.Default().Routing)
	items, err := rt.Candidates(ctx, "claude-opus-4-7", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("candidates = %d, want 1", len(items))
	}
	if got := items[0].UpstreamModel; got != "auto" {
		t.Fatalf("upstream model = %q, want auto", got)
	}
}

func TestCountsAgainstKeyHealthIgnoresClientModelErrors(t *testing.T) {
	if CountsAgainstKeyHealth("client_error") {
		t.Fatal("client errors should not cool down an upstream key")
	}
	if CountsAgainstKeyHealth("protocol_error") {
		t.Fatal("protocol errors should not cool down an upstream key")
	}
	if CountsAgainstKeyHealth("client_canceled") {
		t.Fatal("client cancellations should not cool down an upstream key")
	}
	for _, errType := range []string{"auth_error", "rate_limit", "server_error", "timeout", "empty_response", "upstream_error"} {
		if !CountsAgainstKeyHealth(errType) {
			t.Fatalf("%s should count against key health", errType)
		}
	}
}

func TestClassifyClientCancellationIsNotRetryable(t *testing.T) {
	if got := Classify(0, "Post upstream: context canceled"); got != "client_canceled" {
		t.Fatalf("classification = %q", got)
	}
	if Retryable("client_canceled") {
		t.Fatal("client cancellation should not be retried")
	}
}

func findRouterTestKey(keys []model.Key, id string) *model.Key {
	for i := range keys {
		if keys[i].ID == id {
			return &keys[i]
		}
	}
	return nil
}

func coolKey(ctx context.Context, rt *Router, key model.Key, until time.Time) {
	for i := 0; i < rt.cfg.FailureThreshold; i++ {
		retryAfter := 0
		if i == rt.cfg.FailureThreshold-1 {
			retryAfter = int(time.Until(until).Seconds())
		}
		_ = rt.RecordFailure(ctx, key, Failure{
			Status:            500,
			ErrorType:         "server_error",
			Message:           "down",
			RetryAfterSeconds: retryAfter,
		})
	}
}

func TestConcurrentFailuresTriggerCooldownAtThreshold(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: "http://p", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k", ProviderID: "p", Name: "k", Secret: "secret", Enabled: true})
	keys, err := st.ListKeys(ctx)
	if err != nil || len(keys) != 1 {
		t.Fatalf("keys = %#v, err = %v", keys, err)
	}
	cfg := config.Default().Routing
	cfg.FailureThreshold = 5
	rt := New(st, cfg)

	start := make(chan struct{})
	errs := make(chan error, cfg.FailureThreshold)
	var wg sync.WaitGroup
	for i := 0; i < cfg.FailureThreshold; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- rt.RecordFailure(ctx, keys[0], Failure{Status: 500, ErrorType: "server_error", Message: "down"})
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	keys, err = st.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if keys[0].ConsecutiveFailures != cfg.FailureThreshold {
		t.Fatalf("failures = %d", keys[0].ConsecutiveFailures)
	}
	if keys[0].CooldownUntil == nil || !keys[0].CooldownUntil.After(time.Now()) {
		t.Fatalf("cooldown = %v", keys[0].CooldownUntil)
	}
}

func TestRecoveryProbeAllowsOnlyOneConcurrentRequest(t *testing.T) {
	router := New(nil, config.Default().Routing)
	expired := time.Now().Add(-time.Second)
	key := model.Key{ID: "recovering", CooldownUntil: &expired}
	release, ok := router.AcquireRecoveryProbe(key)
	if !ok {
		t.Fatal("first recovery probe was not acquired")
	}
	if _, second := router.AcquireRecoveryProbe(key); second {
		t.Fatal("second concurrent recovery probe was acquired")
	}
	release()
	releaseAgain, ok := router.AcquireRecoveryProbe(key)
	if !ok {
		t.Fatal("probe was not available after release")
	}
	releaseAgain()
}

func TestCandidatesSortAcrossProviderProtocols(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "openai", Name: "openai", Type: model.ProviderOpenAICompatible, BaseURL: "http://openai", Priority: 1, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "gemini", Name: "gemini", Type: model.ProviderGeminiCompatible, BaseURL: "http://gemini", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "openai-key", ProviderID: "openai", Name: "openai", Secret: "sk-openai", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "gemini-key", ProviderID: "gemini", Name: "gemini", Secret: "sk-gemini", Priority: 1, Enabled: true})

	items, err := New(st, config.Default().Routing).Candidates(ctx, "gpt", ProtocolOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "gemini-key" || items[1].ID != "openai-key" {
		t.Fatalf("candidates = %#v", items)
	}
}

func TestOpenAIResponsesOnlyUsesOpenAIUpstreams(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "openai", Name: "openai", Type: model.ProviderOpenAICompatible, BaseURL: "http://openai", Priority: 1, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "gemini", Name: "gemini", Type: model.ProviderGeminiCompatible, BaseURL: "http://gemini", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "openai-key", ProviderID: "openai", Name: "openai", Secret: "sk-openai", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "gemini-key", ProviderID: "gemini", Name: "gemini", Secret: "sk-gemini", Enabled: true})

	items, err := New(st, config.Default().Routing).Candidates(ctx, "gpt", ProtocolOpenAIResponses)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "openai-key" {
		t.Fatalf("candidates = %#v", items)
	}
}
