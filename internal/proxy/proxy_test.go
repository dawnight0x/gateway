package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/router"
	"local-ai-gateway/internal/store"
)

func testGateway(t *testing.T, st *store.Store, cfg config.Config) *httptest.Server {
	t.Helper()
	rt := router.New(st, cfg.Routing)
	px := New(st, rt, cfg)
	mux := http.NewServeMux()
	px.Register(mux)
	return httptest.NewServer(mux)
}

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

func TestFailoverToSecondKey(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") == "Bearer bad" {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limit"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "bad", ProviderID: "p", Name: "bad", Secret: "bad", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "good", ProviderID: "p", Name: "good", Secret: "good", Priority: 90, Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Routing.RetryPerRequest = 2
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", bytes.NewReader([]byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestNativeStreamFailureIsForwardedAndReportedAsUpstreamError(t *testing.T) {
	cfg := config.Default()
	svc := &Service{cfg: cfg}
	request := httptest.NewRequest(http.MethodPost, "http://upstream/v1/responses", nil)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(
			"event: response.failed\n" +
				`data: {"type":"response.failed","response":{"status":"failed","error":{"message":"capacity exhausted"}}}` + "\n\n",
		)),
		Request: request,
	}
	recorder := httptest.NewRecorder()
	result := svc.pipeStream(recorder, response, model.Key{}, router.ProtocolOpenAIResponses, router.ProtocolOpenAIResponses)

	if result.ok || !result.committed || result.errorType != "upstream_error" || !strings.Contains(result.message, "capacity exhausted") {
		t.Fatalf("stream result = %#v", result)
	}
	if !strings.Contains(recorder.Body.String(), "response.failed") || !strings.Contains(recorder.Body.String(), "capacity exhausted") {
		t.Fatalf("forwarded stream = %s", recorder.Body.String())
	}
}

func TestModelRouteExhaustsProvidersBeforeFallbackModel(t *testing.T) {
	var callsMu sync.Mutex
	calls := make([]string, 0)
	record := func(provider, upstreamModel string) {
		callsMu.Lock()
		calls = append(calls, provider+":"+upstreamModel)
		callsMu.Unlock()
	}
	readCalls := func() []string {
		callsMu.Lock()
		defer callsMu.Unlock()
		return append([]string(nil), calls...)
	}
	resetCalls := func() {
		callsMu.Lock()
		calls = calls[:0]
		callsMu.Unlock()
	}

	modelError := func(w http.ResponseWriter, upstreamModel string) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprintf(w, `{"error":{"message":"model %s not found"}}`, upstreamModel)
	}
	modelFromRequest := func(t *testing.T, r *http.Request) string {
		t.Helper()
		var payload struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode upstream request: %v", err)
		}
		return payload.Model
	}
	writeSuccess := func(w http.ResponseWriter, content string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":%q},"finish_reason":"stop"}]}`, content)
	}

	var highAuthFailure atomic.Bool
	high := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamModel := modelFromRequest(t, r)
		record("high", upstreamModel)
		if highAuthFailure.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid upstream key"}}`))
			return
		}
		if upstreamModel == "high-primary" {
			modelError(w, upstreamModel)
			return
		}
		writeSuccess(w, "high fallback")
	}))
	defer high.Close()

	var lowPrimaryUnavailable atomic.Bool
	low := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamModel := modelFromRequest(t, r)
		record("low", upstreamModel)
		if upstreamModel == "low-primary" && lowPrimaryUnavailable.Load() {
			modelError(w, upstreamModel)
			return
		}
		writeSuccess(w, strings.ReplaceAll(upstreamModel, "-", " "))
	}))
	defer low.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "high", Name: "high", Type: model.ProviderOpenAICompatible, BaseURL: high.URL, Priority: 100, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "low", Name: "low", Type: model.ProviderOpenAICompatible, BaseURL: low.URL, Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "high-key", ProviderID: "high", Name: "high", Secret: "high", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "low-key", ProviderID: "low", Name: "low", Secret: "low", Enabled: true})
	_, err := st.UpsertModelRoute(ctx, model.ModelRoute{
		ID: "coding-auto", Name: "Coding", Enabled: true,
		Models: []model.ModelRouteModel{
			{Name: "primary", Priority: 100, Enabled: true, Targets: []model.ModelRouteTarget{
				{ProviderID: "high", UpstreamModel: "high-primary", Enabled: true},
				{ProviderID: "low", UpstreamModel: "low-primary", Enabled: true},
			}},
			{Name: "fallback", Priority: 10, Enabled: true, Targets: []model.ModelRouteTarget{
				{ProviderID: "high", UpstreamModel: "high-fallback", Enabled: true},
				{ProviderID: "low", UpstreamModel: "low-fallback", Enabled: true},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	gatewayKey, err := st.CreateGatewayKey(ctx, "model-route")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Routing.RetryPerRequest = 1

	send := func(gw *httptest.Server) string {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"coding-auto","messages":[{"role":"user","content":"hi"}]}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, body = %s", res.StatusCode, body)
		}
		return string(body)
	}

	gw := testGateway(t, st, cfg)
	if body := send(gw); !strings.Contains(body, "low primary") {
		t.Fatalf("first response = %s", body)
	}
	gw.Close()
	if got := strings.Join(readCalls(), ","); got != "high:high-primary,low:low-primary" {
		t.Fatalf("first attempt order = %s", got)
	}

	if err := st.ResetProviderModelStates(ctx, "high", "high-primary"); err != nil {
		t.Fatal(err)
	}
	lowPrimaryUnavailable.Store(true)
	resetCalls()
	gw = testGateway(t, st, cfg)
	if body := send(gw); !strings.Contains(body, "high fallback") {
		t.Fatalf("fallback response = %s", body)
	}
	if got := strings.Join(readCalls(), ","); got != "high:high-primary,low:low-primary,high:high-fallback" {
		t.Fatalf("fallback attempt order = %s", got)
	}
	gw.Close()

	for _, providerModel := range [][2]string{{"high", "high-primary"}, {"low", "low-primary"}} {
		if err := st.ResetProviderModelStates(ctx, providerModel[0], providerModel[1]); err != nil {
			t.Fatal(err)
		}
	}
	highAuthFailure.Store(true)
	resetCalls()
	gw = testGateway(t, st, cfg)
	defer gw.Close()
	if body := send(gw); !strings.Contains(body, "low fallback") {
		t.Fatalf("key failure response = %s", body)
	}
	if got := strings.Join(readCalls(), ","); got != "high:high-primary,low:low-primary,low:low-fallback" {
		t.Fatalf("key failure attempt order = %s", got)
	}
}

func TestCandidatesAfterFailureScopesModelErrors(t *testing.T) {
	failed := model.Key{ID: "provider-key-a", ProviderID: "provider", UpstreamModel: "shared"}
	candidates := []model.Key{
		{ID: "provider-key-b", ProviderID: "provider", UpstreamModel: "shared"},
		{ID: "provider-key-b", ProviderID: "provider", UpstreamModel: "fallback"},
		{ID: "other-key", ProviderID: "other", UpstreamModel: "shared"},
	}

	providerScoped := candidatesAfterFailure(append([]model.Key(nil), candidates...), failed, attemptResult{
		errorType: "model_unavailable", providerModelUnavailable: true,
	})
	if len(providerScoped) != 2 || providerScoped[0].UpstreamModel != "fallback" || providerScoped[1].ProviderID != "other" {
		t.Fatalf("provider-scoped candidates = %#v", providerScoped)
	}

	keyScoped := candidatesAfterFailure(append([]model.Key(nil), candidates...), failed, attemptResult{errorType: "model_unavailable"})
	if len(keyScoped) != len(candidates) {
		t.Fatalf("key-scoped candidates = %#v", keyScoped)
	}
}

func TestModelRouteBusyFallbackRequiresExplicitOptIn(t *testing.T) {
	var highCalls, lowCalls atomic.Int32
	high := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		highCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"high"},"finish_reason":"stop"}]}`))
	}))
	defer high.Close()
	low := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lowCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"low"},"finish_reason":"stop"}]}`))
	}))
	defer low.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "high", Name: "high", Type: model.ProviderOpenAICompatible, BaseURL: high.URL, Priority: 100, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "low", Name: "low", Type: model.ProviderOpenAICompatible, BaseURL: low.URL, Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "high-key", ProviderID: "high", Name: "high", Secret: "high", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "low-key", ProviderID: "low", Name: "low", Secret: "low", Enabled: true})
	_, err := st.UpsertModelRoute(ctx, model.ModelRoute{
		ID: "busy-auto", Name: "Busy", Enabled: true,
		Models: []model.ModelRouteModel{
			{Name: "primary", Priority: 100, Enabled: true, Targets: []model.ModelRouteTarget{{ProviderID: "high", UpstreamModel: "primary", Enabled: true}}},
			{Name: "fallback", Priority: 10, Enabled: true, Targets: []model.ModelRouteTarget{{ProviderID: "low", UpstreamModel: "fallback", Enabled: true}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	gatewayKey, err := st.CreateGatewayKey(ctx, "busy")
	if err != nil {
		t.Fatal(err)
	}

	run := func(fallbackOnBusy bool) (int, string) {
		t.Helper()
		cfg := config.Default()
		cfg.Routing.MaxConcurrentPerKey = 1
		cfg.Routing.QueueTimeoutMilliseconds = 10
		cfg.Routing.FallbackOnBusy = fallbackOnBusy
		rt := router.New(st, cfg.Routing)
		px := New(st, rt, cfg)
		release, ok := px.limits.tryAcquire("high", "high-key")
		if !ok {
			t.Fatal("failed to occupy high-priority key")
		}
		defer release()
		mux := http.NewServeMux()
		px.Register(mux)
		gw := httptest.NewServer(mux)
		defer gw.Close()
		req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"busy-auto","messages":[{"role":"user","content":"test"}]}`))
		req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(body)
	}

	status, body := run(false)
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "gateway_busy") || highCalls.Load() != 0 || lowCalls.Load() != 0 {
		t.Fatalf("strict result status=%d body=%s high=%d low=%d", status, body, highCalls.Load(), lowCalls.Load())
	}
	status, body = run(true)
	if status != http.StatusOK || !strings.Contains(body, "low") || highCalls.Load() != 0 || lowCalls.Load() != 1 {
		t.Fatalf("spillover result status=%d body=%s high=%d low=%d", status, body, highCalls.Load(), lowCalls.Load())
	}
}

func TestEqualPriorityProviderURLsFailOverWhenAmbiguousRetriesEnabled(t *testing.T) {
	var primaryCalls int
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalls++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"temporarily unavailable"}}`))
	}))
	defer primary.Close()

	var fallbackCalls int
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"fallback ok"},"finish_reason":"stop"}]}`))
	}))
	defer fallback.Close()

	ctx := context.Background()
	st := testStore(t)
	const priority = 3
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "a-primary", Name: "primary", Type: model.ProviderOpenAICompatible, BaseURL: primary.URL, Priority: priority, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "b-fallback", Name: "fallback", Type: model.ProviderOpenAICompatible, BaseURL: fallback.URL, Priority: priority, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "a-primary-key", ProviderID: "a-primary", Name: "primary", Secret: "primary", Priority: priority, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "b-fallback-key", ProviderID: "b-fallback", Name: "fallback", Secret: "fallback", Priority: priority, Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "equal-priority")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Routing.RetryPerRequest = 2
	cfg.Routing.RetryAmbiguousErrors = true
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", bytes.NewReader([]byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "fallback ok") {
		t.Fatalf("status = %d, body = %s", res.StatusCode, string(body))
	}
	if primaryCalls != 1 || fallbackCalls != 1 {
		t.Fatalf("primary calls = %d, fallback calls = %d, want 1 each", primaryCalls, fallbackCalls)
	}
}

func TestServerErrorDoesNotRepeatBillableRequestByDefault(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"temporarily unavailable"}}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "first", ProviderID: "p", Name: "first", Secret: "first", Priority: 2, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "second", ProviderID: "p", Name: "second", Secret: "second", Priority: 1, Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "safe-retry")
	cfg := config.Default()
	cfg.Routing.RetryPerRequest = 2
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if calls != 1 {
		t.Fatalf("upstream calls = %d, want 1", calls)
	}
}

func TestOpenAIRequestRoutesToGeminiUpstream(t *testing.T) {
	var captured struct {
		Path   string
		APIKey string
		Body   map[string]any
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.Path
		captured.APIKey = r.Header.Get("x-goog-api-key")
		_ = json.NewDecoder(r.Body).Decode(&captured.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"gemini ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":1,"candidatesTokenCount":2,"totalTokenCount":3}}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "gemini", Name: "gemini", Type: model.ProviderGeminiCompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "gk", ProviderID: "gemini", Name: "gk", Secret: "secret", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", bytes.NewReader([]byte(`{"model":"gemini-2.5-pro","messages":[{"role":"user","content":"hi"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if captured.APIKey != "secret" {
		t.Fatalf("api key = %s", captured.APIKey)
	}
	if captured.Path != "/v1beta/models/gemini-2.5-pro:generateContent" {
		t.Fatalf("path = %s", captured.Path)
	}
}

func TestProxyDoesNotDuplicateV1WhenProviderBaseIncludesV1(t *testing.T) {
	var capturedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if capturedPath != "/v1/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"not found"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: upstream.URL + "/v1", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", bytes.NewReader([]byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d body = %s capturedPath = %s", res.StatusCode, string(body), capturedPath)
	}
	if capturedPath != "/v1/chat/completions" {
		t.Fatalf("path = %s", capturedPath)
	}
}

func TestProxyDoesNotDuplicateV1WhenProviderBaseHasPathPrefix(t *testing.T) {
	if got := upstreamURL("https://example.test/openai/v1", "/v1/chat/completions"); got != "https://example.test/openai/v1/chat/completions" {
		t.Fatalf("url = %s", got)
	}
}

func TestProxyAcceptsCommonPathsWithoutV1Prefix(t *testing.T) {
	var capturedPaths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPaths = append(capturedPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: upstream.URL + "/v1", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	for _, item := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/models"},
		{method: http.MethodPost, path: "/chat/completions", body: `{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`},
	} {
		var body io.Reader
		if item.body != "" {
			body = bytes.NewReader([]byte(item.body))
		}
		req, _ := http.NewRequest(item.method, gw.URL+item.path, body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s status = %d", item.path, res.StatusCode)
		}
	}
	if len(capturedPaths) != 1 || capturedPaths[0] != "/v1/chat/completions" {
		t.Fatalf("captured paths = %#v", capturedPaths)
	}
}

func TestOpenAIResponsesUsesNativeUpstream(t *testing.T) {
	var captured struct {
		Path string
		Body map[string]any
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&captured.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_native","object":"response","status":"completed","model":"auto","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"response ok"}]}],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: upstream.URL + "/v1", Enabled: true, ModelMap: map[string]string{"*": "auto"}})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", bytes.NewReader([]byte(`{"model":"gpt-5","input":"hello","max_output_tokens":64}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.StatusCode, string(body))
	}
	if captured.Path != "/v1/responses" {
		t.Fatalf("path = %s", captured.Path)
	}
	if got := captured.Body["model"]; got != "auto" {
		t.Fatalf("model = %v", got)
	}
	if !strings.Contains(string(body), `"object":"response"`) || !strings.Contains(string(body), `"text":"response ok"`) {
		t.Fatalf("response body = %s", string(body))
	}
}

func TestOpenAIResponsesFallsBackToChatCompletions(t *testing.T) {
	var paths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1/responses" {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"message": "not found"}})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-fallback","model":"gpt-5","choices":[{"message":{"content":"fallback ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "test")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5","input":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"object":"response"`) || !strings.Contains(string(body), `"output_text":"fallback ok"`) {
		t.Fatalf("status = %d body = %s", res.StatusCode, body)
	}
	if strings.Join(paths, ",") != "/v1/responses,/v1/chat/completions" {
		t.Fatalf("paths = %#v", paths)
	}
	logs, err := st.ListLogs(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || len(logs[0].AttemptTrace) != 2 {
		t.Fatalf("request logs = %#v", logs)
	}
	trace := logs[0].AttemptTrace
	if trace[0].UpstreamProtocol != router.ProtocolOpenAIResponses || trace[0].Status != http.StatusNotFound || trace[0].ErrorType == "" {
		t.Fatalf("native Responses attempt = %#v", trace[0])
	}
	if trace[1].UpstreamProtocol != router.ProtocolOpenAI || trace[1].Status != http.StatusOK || trace[1].ErrorType != "" {
		t.Fatalf("Chat fallback attempt = %#v", trace[1])
	}
}

func TestStatefulResponsesDoesNotFallBackToChatCompletions(t *testing.T) {
	var chatCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			chatCalls.Add(1)
		}
		writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"message": "not found"}})
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k", ProviderID: "p", Name: "k", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "test")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5","input":"next","previous_response_id":"resp_existing"}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if chatCalls.Load() != 0 {
		t.Fatalf("stateful request made %d Chat fallback calls", chatCalls.Load())
	}
}

func TestResponsesAffinitySticksToOriginalKey(t *testing.T) {
	var firstCalls, secondCalls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_affinity","object":"response","status":"completed","model":"gpt","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"first"}]}]}`))
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_wrong","object":"response","status":"completed","model":"gpt","output":[]}`))
	}))
	defer second.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "first", Name: "first", Type: model.ProviderOpenAICompatible, BaseURL: first.URL, Priority: 100, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "second", Name: "second", Type: model.ProviderOpenAICompatible, BaseURL: second.URL, Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "first-key", ProviderID: "first", Name: "first", Secret: "secret", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "second-key", ProviderID: "second", Name: "second", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "test")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	doResponses := func(body string) {
		req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			payload, _ := io.ReadAll(res.Body)
			t.Fatalf("status = %d body = %s", res.StatusCode, payload)
		}
	}
	doResponses(`{"model":"gpt","input":"first"}`)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "first", Name: "first", Type: model.ProviderOpenAICompatible, BaseURL: first.URL, Priority: 1, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "second", Name: "second", Type: model.ProviderOpenAICompatible, BaseURL: second.URL, Priority: 200, Enabled: true})
	time.Sleep(300 * time.Millisecond)
	doResponses(`{"model":"gpt","input":"next","previous_response_id":"resp_affinity"}`)
	if firstCalls.Load() != 2 || secondCalls.Load() != 0 {
		t.Fatalf("affinity calls first=%d second=%d", firstCalls.Load(), secondCalls.Load())
	}
}

func TestMalformedSuccessfulUpstreamResponseReturnsProtocolError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer upstream.Close()
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k", ProviderID: "p", Name: "k", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "test")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusBadGateway || !strings.Contains(string(body), `"type":"protocol_error"`) {
		t.Fatalf("status = %d body = %s", res.StatusCode, body)
	}
}

func TestChatCompletionsFallsBackToResponses(t *testing.T) {
	var paths []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1/chat/completions" {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"message": "not found"}})
			return
		}
		var request map[string]any
		_ = json.NewDecoder(r.Body).Decode(&request)
		if _, ok := request["input"]; !ok {
			t.Fatalf("responses input missing: %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_fallback","object":"response","status":"completed","model":"gpt-5","output":[{"id":"fc_1","type":"function_call","call_id":"call_1","name":"weather","arguments":"{}"}],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}`))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "sub2api", Name: "sub2api", Type: model.ProviderSub2API, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "sub2api", Name: "k1", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "test")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-5","messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function","function":{"name":"weather","parameters":{"type":"object"}}}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"finish_reason":"tool_calls"`) || !strings.Contains(string(body), `"tool_calls"`) {
		t.Fatalf("status = %d body = %s", res.StatusCode, body)
	}
	if strings.Join(paths, ",") != "/v1/chat/completions,/v1/responses" {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestStatusAndMetricsRequireGatewayAuth(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	gatewayKey, err := st.CreateGatewayKey(ctx, "status")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	for _, path := range []string{"/status", "/metrics"} {
		res, err := http.Get(gw.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s without auth status = %d, want 401", path, res.StatusCode)
		}

		req, _ := http.NewRequest(http.MethodGet, gw.URL+path, nil)
		req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s with auth status = %d, want 200", path, res.StatusCode)
		}
	}
}

func TestReadyReflectsUsableGatewayConfiguration(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	res, err := http.Get(gw.URL + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), `"reason":"no_providers"`) {
		t.Fatalf("unconfigured readiness status = %d body = %s", res.StatusCode, body)
	}

	if _, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://provider.example", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateGatewayKey(ctx, "client"); err != nil {
		t.Fatal(err)
	}
	res, err = http.Get(gw.URL + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"ready"`) || !strings.Contains(string(body), `"ready":true`) {
		t.Fatalf("configured readiness status = %d body = %s", res.StatusCode, body)
	}
}

func TestProxyRejectsQueryStringGatewayKey(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	gatewayKey, err := st.CreateGatewayKey(ctx, "query")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	res, err := http.Get(gw.URL + "/v1/models?key=" + gatewayKey.Plaintext)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("query key status = %d, want 401", res.StatusCode)
	}
}

func TestSanitizeUpstreamMessageRedactsSecretsAndSensitiveHeaders(t *testing.T) {
	if got := sanitizeUpstreamMessage("upstream rejected sk-secret-value", "sk-secret-value"); got != "upstream rejected ***" {
		t.Fatalf("redacted message = %q", got)
	}
	if got := sanitizeUpstreamMessage("Authorization: Bearer sk-secret-value", "sk-secret-value"); got != "upstream request failed; sensitive details redacted" {
		t.Fatalf("sensitive header message = %q", got)
	}
	long := strings.Repeat("a", 350)
	if got := sanitizeUpstreamMessage(long, ""); len(got) != 300 {
		t.Fatalf("long message length = %d", len(got))
	}
}

func TestUnsupportedOpenAIEndpointDetection(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		want   bool
	}{
		{http.StatusNotFound, `{"error":"model not found"}`, true},
		{http.StatusMethodNotAllowed, ``, true},
		{http.StatusBadRequest, `{"error":{"message":"Unsupported legacy protocol; use /v1/responses"}}`, true},
		{http.StatusBadRequest, `{"error":{"message":"max_tokens is unsupported"}}`, false},
		{http.StatusUnauthorized, `{"error":"invalid key"}`, false},
	} {
		if got := isUnsupportedOpenAIEndpoint(test.status, []byte(test.body)); got != test.want {
			t.Fatalf("status=%d body=%s got=%v want=%v", test.status, test.body, got, test.want)
		}
	}
}

func TestProxyRejectsOversizedRequestBody(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	gatewayKey, err := st.CreateGatewayKey(ctx, "large")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(strings.Repeat("x", maxProxyRequestBody+1)))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status = %d, body = %s", res.StatusCode, string(body))
	}
}

func TestEmptyStreamFailsOverBeforeFirstByte(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/event-stream")
		if r.Header.Get("Authorization") == "Bearer empty" {
			w.Header().Set("OpenAI-Request-ID", "failed-attempt")
			return
		}
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "empty", ProviderID: "p", Name: "empty", Secret: "empty", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "good", ProviderID: "p", Name: "good", Secret: "good", Priority: 90, Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "stream")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Routing.RetryPerRequest = 2
	cfg.Routing.RetryAmbiguousErrors = true
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "ok") {
		t.Fatalf("status = %d, body = %s", res.StatusCode, string(body))
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if got := res.Header.Get("OpenAI-Request-ID"); got != "" {
		t.Fatalf("failed attempt metadata leaked into final response: %q", got)
	}
}

func TestEmptyStreamAfterImmediateCommitRecordsFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "empty", ProviderID: "p", Name: "empty", Secret: "empty", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "stream")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Routing.StreamRetryBeforeFirstByte = false
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	_, _ = io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("committed stream status = %d", res.StatusCode)
	}
	logs, err := st.ListLogs(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || logs[0].ErrorType != "empty_response" || len(logs[0].AttemptTrace) != 1 || logs[0].AttemptTrace[0].ErrorType != "empty_response" {
		t.Fatalf("empty stream log = %#v", logs)
	}
	keys, err := st.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].ConsecutiveFailures != 1 || keys[0].LastError != "empty upstream stream" {
		t.Fatalf("empty stream key health = %#v", keys)
	}
}

func TestCommittedStreamInterruptionDoesNotAppendJSONError(t *testing.T) {
	const event = "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("upstream response writer does not support hijacking")
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fmt.Fprintf(rw, "HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\nTransfer-Encoding: chunked\r\n\r\n%x\r\n%s\r\n", len(event), event)
		_ = rw.Flush()
		_ = conn.Close()
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "p", Name: "p", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k", ProviderID: "p", Name: "k", Secret: "secret", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "stream")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "partial") {
		t.Fatalf("body = %s", string(body))
	}
	if strings.Contains(string(body), `"error"`) || strings.Contains(string(body), "stream_interrupted") {
		t.Fatalf("JSON error appended after stream commit: %s", string(body))
	}
}

func TestStreamingConvertsHighestPriorityCrossProtocolCandidate(t *testing.T) {
	var geminiCalls int
	var openAICalls int
	gemini := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		geminiCalls++
		if r.URL.Path != "/v1beta/models/gpt:streamGenerateContent" || r.URL.Query().Get("alt") != "sse" {
			t.Fatalf("Gemini stream target = %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"converted\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":2,\"candidatesTokenCount\":1,\"totalTokenCount\":3}}\n\n"))
	}))
	defer gemini.Close()
	openAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		openAICalls++
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"fallback\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer openAI.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "gemini", Name: "gemini", Type: model.ProviderGeminiCompatible, BaseURL: gemini.URL, Priority: 100, Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "openai", Name: "openai", Type: model.ProviderOpenAICompatible, BaseURL: openAI.URL, Priority: 1, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "gemini-key", ProviderID: "gemini", Name: "gemini", Secret: "gemini", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "openai-key", ProviderID: "openai", Name: "openai", Secret: "openai", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "stream")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "converted") || !strings.Contains(string(body), "[DONE]") {
		t.Fatalf("status = %d, body = %s", res.StatusCode, string(body))
	}
	if geminiCalls != 1 || openAICalls != 0 {
		t.Fatalf("Gemini calls = %d, OpenAI calls = %d", geminiCalls, openAICalls)
	}
}

func TestOpenAIResponsesStreamingConversion(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/responses" {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": map[string]any{"message": "not found"}})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"model\":\"gpt-5\",\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"hello\"},\"finish_reason\":null}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"chatcmpl-test\",\"model\":\"gpt-5\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1,\"total_tokens\":3}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "openai", Name: "openai", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "openai-key", ProviderID: "openai", Name: "openai", Secret: "openai", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "stream")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5","stream":true,"input":"hi"}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	for _, expected := range []string{"response.created", "response.output_text.delta", "hello", "response.completed", `"total_tokens":3`} {
		if !strings.Contains(string(body), expected) {
			t.Fatalf("response stream missing %q: %s", expected, string(body))
		}
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.StatusCode, string(body))
	}
}

func TestOpenAIResponsesNativeStreamingPassthrough(t *testing.T) {
	const nativeStream = "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_native\",\"model\":\"gpt-5\",\"status\":\"in_progress\"}}\n\nevent: response.output_item.added\ndata: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"id\":\"msg_native\",\"type\":\"message\",\"status\":\"in_progress\",\"role\":\"assistant\",\"content\":[]}}\n\nevent: response.content_part.added\ndata: {\"type\":\"response.content_part.added\",\"item_id\":\"msg_native\",\"output_index\":0,\"content_index\":0,\"part\":{\"type\":\"output_text\",\"text\":\"\"}}\n\nevent: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"item_id\":\"msg_native\",\"output_index\":0,\"content_index\":0,\"delta\":\"native\"}\n\nevent: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_native\",\"model\":\"gpt-5\",\"status\":\"completed\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1,\"total_tokens\":2}}}\n\n"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(nativeStream))
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "newapi", Name: "newapi", Type: model.ProviderNewAPI, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "stream")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/responses", strings.NewReader(`{"model":"gpt-5","stream":true,"input":"hi"}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"delta":"native"`) || !strings.Contains(string(body), "response.content_part.added") {
		t.Fatalf("status = %d body = %s", res.StatusCode, body)
	}
}

func TestActiveStreamCanOutliveRequestTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		frames := []string{
			"data: {\"id\":\"chatcmpl-live\",\"model\":\"gpt-5\",\"choices\":[{\"delta\":{\"content\":\"still\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-live\",\"model\":\"gpt-5\",\"choices\":[{\"delta\":{\"content\":\" active\"},\"finish_reason\":null}]}\n\n",
			"data: [DONE]\n\n",
		}
		for i, frame := range frames {
			_, _ = io.WriteString(w, frame)
			flusher.Flush()
			if i < len(frames)-1 {
				time.Sleep(600 * time.Millisecond)
			}
		}
	}))
	defer upstream.Close()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "stream", Name: "stream", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "stream-key", ProviderID: "stream", Name: "stream", Secret: "secret", Enabled: true})
	gatewayKey, _ := st.CreateGatewayKey(ctx, "stream")
	cfg := config.Default()
	cfg.Routing.TimeoutSeconds = 1
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt-5","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	start := time.Now()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < time.Second {
		t.Fatalf("stream finished too quickly to exercise total timeout: %s", elapsed)
	}
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), "still") || !strings.Contains(string(body), " active") {
		t.Fatalf("status = %d body = %s", res.StatusCode, body)
	}
}

func TestModelsReflectConfiguredMappingsAndProtocol(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "enabled", Name: "enabled", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true, ModelMap: map[string]string{"public-b": "upstream-b", "public-a": "upstream-a", "*": "fallback"}})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "discovered", Name: "discovered", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true})
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "disabled", Name: "disabled", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: false, ModelMap: map[string]string{"hidden": "hidden"}})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "enabled-key", ProviderID: "enabled", Name: "enabled", Secret: "enabled", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "discovered-key", ProviderID: "discovered", Name: "discovered", Secret: "discovered", Enabled: true})
	if err := st.ReplaceProviderModels(ctx, "discovered", []string{"identity-model"}); err != nil {
		t.Fatal(err)
	}
	gatewayKey, _ := st.CreateGatewayKey(ctx, "models")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	request := func(path, header string) map[string]any {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, gw.URL+path, nil)
		value := gatewayKey.Plaintext
		if header == "Authorization" {
			value = "Bearer " + value
		}
		req.Header.Set(header, value)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}

	openAI := request("/v1/models", "Authorization")
	openAIData := openAI["data"].([]any)
	if len(openAIData) != 3 || openAIData[0].(map[string]any)["id"] != "identity-model" || openAIData[1].(map[string]any)["id"] != "public-a" || openAIData[2].(map[string]any)["id"] != "public-b" {
		t.Fatalf("OpenAI models = %#v", openAIData)
	}
	gemini := request("/v1beta/models", "x-goog-api-key")
	geminiData := gemini["models"].([]any)
	if len(geminiData) != 3 || geminiData[0].(map[string]any)["name"] != "models/identity-model" {
		t.Fatalf("Gemini models = %#v", geminiData)
	}
}

func TestModelsHideLogicalRouteWithoutUsableKey(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true})
	_, err := st.UpsertModelRoute(ctx, model.ModelRoute{
		ID: "logical", Name: "logical", Enabled: true,
		Models: []model.ModelRouteModel{{Name: "primary", Priority: 100, Enabled: true, Targets: []model.ModelRouteTarget{{ProviderID: "provider", UpstreamModel: "upstream", Enabled: true}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	gatewayKey, _ := st.CreateGatewayKey(ctx, "models")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	list := func() []any {
		t.Helper()
		req, _ := http.NewRequest(http.MethodGet, gw.URL+"/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var payload map[string]any
		if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		return payload["data"].([]any)
	}

	if got := list(); len(got) != 0 {
		t.Fatalf("models without key = %#v", got)
	}
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: false})
	if got := list(); len(got) != 0 {
		t.Fatalf("models with disabled key = %#v", got)
	}
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: true})
	got := list()
	if len(got) != 1 || got[0].(map[string]any)["id"] != "logical" {
		t.Fatalf("models with enabled key = %#v", got)
	}
}

func TestModelsRespectProviderModelAllowlist(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{
		ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true,
		ModelAllowlistEnabled: true, ModelAllowlist: []string{"allowed"},
	})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: true})
	if err := st.ReplaceProviderModels(ctx, "provider", []string{"allowed", "blocked"}); err != nil {
		t.Fatal(err)
	}
	_, err := st.UpsertModelRoute(ctx, model.ModelRoute{
		ID: "blocked-logical", Name: "blocked", Enabled: true,
		Models: []model.ModelRouteModel{{Name: "blocked", Priority: 100, Enabled: true, Targets: []model.ModelRouteTarget{{ProviderID: "provider", UpstreamModel: "blocked", Enabled: true}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	gatewayKey, _ := st.CreateGatewayKey(ctx, "models")
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	req, _ := http.NewRequest(http.MethodGet, gw.URL+"/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data) != 1 || payload.Data[0].ID != "allowed" {
		t.Fatalf("models = %#v", payload.Data)
	}
}

func TestProxyRejectsInvalidOrMissingModelBeforeRouting(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	gatewayKey, err := st.CreateGatewayKey(ctx, "invalid-input")
	if err != nil {
		t.Fatal(err)
	}
	gw := testGateway(t, st, config.Default())
	defer gw.Close()

	for name, body := range map[string]string{
		"malformed":     `{"model":`,
		"missing model": `{"messages":[]}`,
		"numeric model": `{"model":42,"messages":[]}`,
		"long model":    `{"model":"` + strings.Repeat("m", maxModelNameLength+1) + `","messages":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			payload, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(payload), `"type":"invalid_request"`) {
				t.Fatalf("status = %d body = %s", res.StatusCode, payload)
			}
			if requestID := res.Header.Get("X-Gateway-Request-ID"); requestID == "" {
				t.Fatal("missing X-Gateway-Request-ID")
			}
		})
	}
}

func TestClientCancellationDoesNotPenalizeUpstreamKey(t *testing.T) {
	started := make(chan struct{})
	upstreamDone := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(upstreamDone)
	}))
	defer func() {
		upstream.CloseClientConnections()
		upstream.Close()
	}()

	ctx := context.Background()
	st := testStore(t)
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "cancel-provider", Name: "cancel", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "cancel-key", ProviderID: "cancel-provider", Name: "cancel", Secret: "secret", Enabled: true})
	gatewayKey, err := st.CreateGatewayKey(ctx, "cancel-client")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Routing.RetryPerRequest = 1
	gw := testGateway(t, st, cfg)
	defer gw.Close()

	requestCtx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","messages":[{"role":"user","content":"wait"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+gatewayKey.Plaintext)
	requestDone := make(chan error, 1)
	go func() {
		res, err := http.DefaultClient.Do(req)
		if res != nil {
			_ = res.Body.Close()
		}
		requestDone <- err
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream request did not start")
	}
	cancel()
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("expected canceled client request to fail")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("client request did not stop after cancellation")
	}
	select {
	case <-upstreamDone:
	case <-time.After(3 * time.Second):
		t.Fatal("upstream request context was not canceled")
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		logs, err := st.ListLogs(context.Background(), 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) > 0 {
			got := logs[0]
			if got.Status != statusClientClosed || got.ErrorType != "client_canceled" {
				t.Fatalf("canceled request log = %#v", got)
			}
			if got.ProviderID != "cancel-provider" || got.KeyID != "cancel-key" {
				t.Fatalf("canceled request attribution = provider %q key %q", got.ProviderID, got.KeyID)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("canceled request was not logged")
		}
		time.Sleep(10 * time.Millisecond)
	}
	keys, err := st.ListKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].FailureCount != 0 || keys[0].ConsecutiveFailures != 0 {
		t.Fatalf("canceled request changed key health: %#v", keys)
	}
}

func TestDrainingRejectsNewRequestsAndReadiness(t *testing.T) {
	st := testStore(t)
	cfg := config.Default()
	cfg.Server.ProxyToken = "local-token"
	svc := New(st, router.New(st, cfg.Routing), cfg)
	svc.BeginDrain()

	mux := http.NewServeMux()
	svc.Register(mux)
	gw := httptest.NewServer(mux)
	defer gw.Close()

	ready, err := http.Get(gw.URL + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	readyBody, _ := io.ReadAll(ready.Body)
	_ = ready.Body.Close()
	if ready.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(readyBody), `"status":"draining"`) {
		t.Fatalf("draining readiness = %d %s", ready.StatusCode, readyBody)
	}

	req, _ := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"gpt","messages":[]}`))
	req.Header.Set("Authorization", "Bearer local-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(body), "gateway_draining") || res.Header.Get("Retry-After") == "" {
		t.Fatalf("draining response = %d %s headers=%v", res.StatusCode, body, res.Header)
	}
}
