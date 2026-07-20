package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
)

func TestAdminAPIRejectsMissingToken(t *testing.T) {
	svc := New(nil, config.Default())
	req := httptest.NewRequest(http.MethodGet, "http://localhost:18787/admin/api/dashboard", nil)
	req.Host = "localhost:18787"
	res := httptest.NewRecorder()

	svc.handle(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestAdminStaticResponsesSetSecurityHeaders(t *testing.T) {
	st := testAdminStore(t)
	svc := New(st, config.Default())
	req := httptest.NewRequest(http.MethodGet, "http://localhost:18787/admin/", nil)
	res := httptest.NewRecorder()
	svc.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	if got := res.Header().Get("Content-Security-Policy"); !strings.Contains(got, "frame-ancestors 'none'") || !strings.Contains(got, "connect-src 'self'") || strings.Contains(got, "unsafe-eval") {
		t.Fatalf("CSP = %q", got)
	}
	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := res.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q", got)
	}
}

func TestAdminAPIRejectsNonLocalHost(t *testing.T) {
	svc := New(nil, config.Default())
	req := httptest.NewRequest(http.MethodGet, "http://evil.example/admin/api/dashboard", nil)
	req.Host = "evil.example"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()

	svc.handle(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestAdminAuthToken(t *testing.T) {
	svc := New(nil, config.Default())
	req := httptest.NewRequest(http.MethodGet, "http://localhost:18787/admin/api/dashboard", nil)
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)

	if !svc.authorizedAdmin(req) {
		t.Fatal("expected valid admin token")
	}

	req.Header.Set("X-Admin-Token", "wrong")
	if svc.authorizedAdmin(req) {
		t.Fatal("expected invalid admin token")
	}

	req.Header.Del("X-Admin-Token")
	req.Header.Set("Authorization", "bearer "+config.Default().Server.AdminToken)
	if !svc.authorizedAdmin(req) {
		t.Fatal("expected bearer scheme to be case-insensitive")
	}
}

func TestAdminOriginGuard(t *testing.T) {
	svc := New(nil, config.Default())

	if !svc.allowedAdminOrigin("http://localhost:18787/admin") {
		t.Fatal("expected localhost origin to be allowed")
	}
	if svc.allowedAdminOrigin("https://evil.example/app") {
		t.Fatal("expected remote origin to be rejected")
	}
}

func TestAdminRejectsInvalidProviderURLs(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st, config.Default())
	for _, body := range []string{
		`{"name":"bad","type":"openai-compatible","baseUrl":"file:///etc/passwd"}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"https://user:pass@example.com/v1"}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"localhost:18787"}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"https://example.com/v1","balancePath":"ftp://example.com/balance"}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"https://example.com/v1","balancePath":"https://collector.example/balance"}`,
		`{"name":"bad","type":"unknown","baseUrl":"https://example.com/v1"}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"https://example.com/v1?token=bad"}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"https://example.com/v1","modelMap":{"public":""}}`,
		`{"name":"bad","type":"openai-compatible","baseUrl":"https://example.com/v1","balancePath":"/balance#secret"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/providers", strings.NewReader(body))
		req.Host = "localhost:18787"
		req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
		res := httptest.NewRecorder()

		svc.handle(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("body %s expected 400, got %d: %s", body, res.Code, res.Body.String())
		}
	}
}

func TestAdminRejectsInsecureRemoteUpstreamByDefault(t *testing.T) {
	st := testAdminStore(t)
	svc := New(st, config.Default())
	req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/providers", strings.NewReader(`{"name":"remote","type":"openai-compatible","baseUrl":"http://api.example.com/v1"}`))
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()

	svc.handle(res, req)

	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "must use https") {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminLogsSupportFilteringPaginationCSVAndClear(t *testing.T) {
	st := testAdminStore(t)
	ctx := context.Background()
	for _, item := range []model.RequestLog{
		{RequestID: "req-ok", InboundProtocol: "openai", ProviderID: "p1", KeyID: "k1", Model: "gpt-4o", Status: 200, LatencyMS: 12},
		{RequestID: "req-fail", InboundProtocol: "anthropic", ProviderID: "p2", KeyID: "k2", Model: "claude", Status: 429, LatencyMS: 20, ErrorType: "rate_limit"},
	} {
		if err := st.LogRequest(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(st, config.Default())

	res := authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/logs?status=429&providerId=p2&limit=1&offset=0&q=rate", "")
	if res.Code != http.StatusOK {
		t.Fatalf("filter status = %d, body = %s", res.Code, res.Body.String())
	}
	var page store.LogPage
	if err := json.NewDecoder(res.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].RequestID != "req-fail" || page.Limit != 1 {
		t.Fatalf("page = %#v", page)
	}

	res = authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/logs?format=csv&model=gpt-4o", "")
	if res.Code != http.StatusOK || !strings.HasPrefix(res.Header().Get("Content-Type"), "text/csv") || !strings.Contains(res.Body.String(), "req-ok") {
		t.Fatalf("csv status = %d, type = %q, body = %s", res.Code, res.Header().Get("Content-Type"), res.Body.String())
	}

	res = authorizedAdminRequest(t, svc, http.MethodDelete, "/admin/api/logs", "")
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"deleted":2`) {
		t.Fatalf("delete status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminMaintenanceIntegrityAndBackup(t *testing.T) {
	st := testAdminStore(t)
	svc := New(st, config.Default())

	res := authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/maintenance/integrity", "")
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"status":"ok"`) {
		t.Fatalf("integrity status = %d, body = %s", res.Code, res.Body.String())
	}

	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/maintenance/backup", "")
	if res.Code != http.StatusOK || !strings.Contains(res.Header().Get("Content-Disposition"), "gateway-manual-") || !strings.HasPrefix(res.Body.String(), "SQLite format 3") {
		t.Fatalf("backup status = %d, disposition = %q, bytes = %d", res.Code, res.Header().Get("Content-Disposition"), res.Body.Len())
	}

	res = authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/maintenance/backups", "")
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "gateway-manual-") {
		t.Fatalf("backups status = %d, body = %s", res.Code, res.Body.String())
	}

	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/maintenance/portable-backup", `{"passphrase":"short"}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "at least 12 bytes") {
		t.Fatalf("portable backup validation status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminRoutingPatchPersistsWithoutReplacingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 19000\nlogging:\n  max_size_mb: 8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATEWAY_CONFIG", path)
	st := testAdminStore(t)
	svc := New(st, config.Default())
	routing, err := json.Marshal(config.Default().Routing)
	if err != nil {
		t.Fatal(err)
	}
	res := authorizedAdminRequest(t, svc, http.MethodPatch, "/admin/api/routing", string(routing))
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"restartRequired":true`) {
		t.Fatalf("routing status = %d, body = %s", res.Code, res.Body.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if text := string(data); !strings.Contains(text, "port: 19000") || !strings.Contains(text, "max_size_mb: 8") || !strings.Contains(text, "max_concurrent_per_key: 4") {
		t.Fatalf("saved config = %s", text)
	}
}

func authorizedAdminRequest(t *testing.T, svc *Service, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "http://localhost:18787"+path, strings.NewReader(body))
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()
	svc.handle(res, req)
	return res
}

func TestAdminRejectsIncompleteUpstreamKey(t *testing.T) {
	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())
	for _, body := range []string{
		`{"providerId":"provider","name":"","secret":"secret"}`,
		`{"providerId":"provider","name":"key","secret":""}`,
		`{"providerId":"","name":"key","secret":"secret"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/keys", strings.NewReader(body))
		req.Host = "localhost:18787"
		req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
		res := httptest.NewRecorder()
		svc.handle(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("body %s expected 400, got %d: %s", body, res.Code, res.Body.String())
		}
	}
}

func TestAdminReadOnlyEndpointsRejectWritesAndDisableCaching(t *testing.T) {
	st := testAdminStore(t)
	svc := New(st, config.Default())
	req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/dashboard", strings.NewReader(`{}`))
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()
	svc.handle(res, req)
	if res.Code != http.StatusMethodNotAllowed || res.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("status = %d Allow = %q", res.Code, res.Header().Get("Allow"))
	}
	if cache := res.Header().Get("Cache-Control"); !strings.Contains(cache, "no-store") {
		t.Fatalf("Cache-Control = %q", cache)
	}
}

func TestAdminRotatesGatewayKey(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	created, err := st.CreateGatewayKey(ctx, "cursor")
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())
	req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/gateway-keys/"+created.ID+"/rotate", nil)
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()

	svc.handle(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var rotated struct {
		ID        string `json:"id"`
		KeyHint   string `json:"keyHint"`
		Plaintext string `json:"plaintext"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.ID != created.ID {
		t.Fatalf("id = %s, want %s", rotated.ID, created.ID)
	}
	if rotated.Plaintext == "" || rotated.Plaintext == created.Plaintext {
		t.Fatalf("unexpected plaintext %q", rotated.Plaintext)
	}
	if rotated.KeyHint == created.KeyHint {
		t.Fatal("expected key hint to change")
	}
}

func TestAdminTestsNewAPIKeyViaModelsEndpoint(t *testing.T) {
	var capturedAuth string
	var capturedGoogleKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedGoogleKey = r.Header.Get("x-goog-api-key")
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/models":
			if r.Method != http.MethodGet {
				t.Fatalf("models method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"gpt-4o-mini"},{"id":"claude-sonnet"}]}`))
		case "/v1/chat/completions":
			if r.Method != http.MethodPost {
				t.Fatalf("chat method = %s", r.Method)
			}
			var body struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Model != "gpt-4o-mini" {
				t.Fatalf("model = %s", body.Model)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	_, err = st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL + "/v1",
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	key, err := st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())
	req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/keys/"+key.ID+"/test", nil)
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()

	svc.handle(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if capturedAuth != "Bearer secret" {
		t.Fatalf("auth = %q", capturedAuth)
	}
	if capturedGoogleKey != "" {
		t.Fatalf("google api key header = %q", capturedGoogleKey)
	}
	var result struct {
		Status           string   `json:"status"`
		ConnectionStatus string   `json:"connectionStatus"`
		TokenStatus      string   `json:"tokenStatus"`
		StatusCode       int      `json:"statusCode"`
		TokenStatusCode  int      `json:"tokenStatusCode"`
		Model            string   `json:"model"`
		Models           []string `json:"models"`
		ModelCount       *int     `json:"modelCount"`
		PromptTokens     *int     `json:"promptTokens"`
		CompletionTokens *int     `json:"completionTokens"`
		TotalTokens      *int     `json:"totalTokens"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || result.StatusCode != 200 {
		t.Fatalf("result = %#v", result)
	}
	if result.ModelCount == nil || *result.ModelCount != 2 {
		t.Fatalf("model count = %#v", result.ModelCount)
	}
	if result.ConnectionStatus != "ok" || result.TokenStatus != "ok" || result.TokenStatusCode != 200 {
		t.Fatalf("token test result = %#v", result)
	}
	if result.Model != "gpt-4o-mini" {
		t.Fatalf("model = %s", result.Model)
	}
	if len(result.Models) != 2 || result.Models[0] != "gpt-4o-mini" || result.Models[1] != "claude-sonnet" {
		t.Fatalf("models = %#v", result.Models)
	}
	if result.PromptTokens == nil || *result.PromptTokens != 3 || result.CompletionTokens == nil || *result.CompletionTokens != 1 || result.TotalTokens == nil || *result.TotalTokens != 4 {
		t.Fatalf("usage = %#v", result)
	}
}

func TestAdminRefreshesProviderModelsWithoutTokenProbeAndRetainsInventoryOnEmptyResult(t *testing.T) {
	var empty atomic.Bool
	var modelCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Errorf("unexpected discovery request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		modelCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if empty.Load() {
			_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"model-b"},{"id":"model-a"}]}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL + "/v1", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	res := authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/providers/provider/models", `{}`)
	if res.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", res.Code, res.Body.String())
	}
	var result keyTestResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || result.TokenStatus != "" || len(result.Models) != 2 || modelCalls.Load() != 1 {
		t.Fatalf("discovery result = %#v, calls = %d", result, modelCalls.Load())
	}

	empty.Store(true)
	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/providers/provider/models", `{}`)
	result = keyTestResult{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "empty" || result.ConnectionStatus != "ok" || result.Error == "" || modelCalls.Load() != 2 {
		t.Fatalf("empty discovery result = %#v, calls = %d", result, modelCalls.Load())
	}
	inventory, err := st.ListProviderModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := inventory["provider"]; len(got) != 2 || got[0] != "model-a" || got[1] != "model-b" {
		t.Fatalf("retained inventory = %#v", got)
	}
}

func TestAdminDiscoversPaginatedGeminiModelsWithVersionedBase(t *testing.T) {
	var requests []string
	var failSecondPage atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		if r.Method != http.MethodGet || r.URL.Path != "/v1beta/models" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.Header.Get("x-goog-api-key") != "gemini-secret" {
			t.Fatalf("google api key = %q", r.Header.Get("x-goog-api-key"))
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("pageToken") == "" {
			_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-b"}],"nextPageToken":"page-2"}`))
			return
		}
		if r.URL.Query().Get("pageToken") != "page-2" {
			t.Fatalf("page token = %q", r.URL.Query().Get("pageToken"))
		}
		if failSecondPage.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"temporary"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-a"}]}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{ID: "gemini", Name: "gemini", Type: model.ProviderGeminiCompatible, BaseURL: upstream.URL + "/v1beta", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	key, err := st.UpsertKey(ctx, model.Key{ID: "gemini-key", ProviderID: "gemini", Name: "gemini", Secret: "gemini-secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	result := New(st, config.Default()).refreshProviderModels(ctx, "gemini")
	if result.Status != "ok" || result.ModelCount == nil || *result.ModelCount != 2 {
		t.Fatalf("discovery result = %#v", result)
	}
	if len(requests) != 2 || requests[0] != "/v1beta/models" || requests[1] != "/v1beta/models?pageToken=page-2" {
		t.Fatalf("requests = %#v", requests)
	}
	inventory, err := st.ListProviderModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := inventory["gemini"]; len(got) != 2 || got[0] != "gemini-a" || got[1] != "gemini-b" {
		t.Fatalf("Gemini inventory = %#v", got)
	}
	failSecondPage.Store(true)
	result = New(st, config.Default()).refreshProviderModels(ctx, "gemini")
	if result.Status != "server_error" {
		t.Fatalf("failed pagination result = %#v", result)
	}
	inventory, err = st.ListProviderModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := inventory["gemini"]; len(got) != 2 || got[0] != "gemini-a" || got[1] != "gemini-b" {
		t.Fatalf("inventory after failed page = %#v", got)
	}
	endpoint, _, err := tokenProbeRequest(key, "models/gemini-a")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.EscapedPath() != "/v1beta/models/gemini-a:generateContent" {
		t.Fatalf("token probe endpoint = %s", endpoint)
	}
}

func TestProviderModelDiscoveryFallsBackToNextEnabledKey(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") == "Bearer bad" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid key"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"model-a"}]}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL + "/v1", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "bad", ProviderID: "provider", Name: "bad", Secret: "bad", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "good", ProviderID: "provider", Name: "good", Secret: "good", Priority: 10, Enabled: true})

	result := New(st, config.Default()).refreshProviderModels(ctx, "provider")
	if result.Status != "partial" || result.KeyID != "good" || !strings.Contains(result.Error, "bad (auth_error)") || calls.Load() != 2 {
		t.Fatalf("result = %#v, calls = %d", result, calls.Load())
	}
}

func TestProviderModelDiscoveryUnionsSuccessfulKeyInventories(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		modelID := "model-a"
		if r.Header.Get("Authorization") == "Bearer second" {
			modelID = "model-b"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"` + modelID + `"}]}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: upstream.URL + "/v1", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "first", ProviderID: "provider", Name: "first", Secret: "first", Priority: 100, Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "second", ProviderID: "provider", Name: "second", Secret: "second", Priority: 10, Enabled: true})

	result := New(st, config.Default()).refreshProviderModels(ctx, "provider")
	if result.Status != "ok" || result.ModelCount == nil || *result.ModelCount != 2 || calls.Load() != 2 {
		t.Fatalf("result = %#v, calls = %d", result, calls.Load())
	}
	inventory, err := st.ListProviderModels(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := inventory["provider"]; len(got) != 2 || got[0] != "model-a" || got[1] != "model-b" {
		t.Fatalf("union inventory = %#v", got)
	}
}

func TestAdminModelRouteCRUDAndValidation(t *testing.T) {
	st := testAdminStore(t)
	ctx := context.Background()
	for _, provider := range []model.Provider{
		{ID: "high", Name: "high", Type: model.ProviderOpenAICompatible, BaseURL: "https://high.example", Priority: 100, Enabled: true},
		{ID: "low", Name: "low", Type: model.ProviderOpenAICompatible, BaseURL: "https://low.example", Priority: 1, Enabled: true},
	} {
		if _, err := st.UpsertProvider(ctx, provider); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(st, config.Default())
	body := `{
		"id":"coding-auto","name":"Coding","enabled":true,
		"models":[
			{"name":"fallback","priority":10,"enabled":true,"targets":[{"providerId":"low","upstreamModel":"fallback","enabled":true}]},
			{"name":"primary","priority":100,"enabled":true,"targets":[
				{"providerId":"low","upstreamModel":"low-primary","enabled":true},
				{"providerId":"high","upstreamModel":"high-primary","enabled":true}
			]}
		]
	}`
	res := authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-routes", body)
	if res.Code != http.StatusOK {
		t.Fatalf("create route status = %d, body = %s", res.Code, res.Body.String())
	}
	var route model.ModelRoute
	if err := json.NewDecoder(res.Body).Decode(&route); err != nil {
		t.Fatal(err)
	}
	if len(route.Models) != 2 || route.Models[0].Name != "primary" || route.Models[0].Targets[0].ProviderID != "high" {
		t.Fatalf("saved route = %#v", route)
	}

	res = authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/model-routes/coding-auto", "")
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"upstreamModel":"high-primary"`) {
		t.Fatalf("get route status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-routes", `{"id":"invalid","enabled":true,"models":[{"name":"primary","priority":1,"enabled":true,"targets":[{"providerId":"missing","upstreamModel":"model","enabled":true}]}]}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "does not exist") {
		t.Fatalf("missing provider validation status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-routes", `{"id":"ambiguous","enabled":true,"models":[{"name":"primary","priority":1,"enabled":true,"targets":[{"providerId":"high","upstreamModel":"model-a","enabled":true},{"providerId":"high","upstreamModel":"model-b","enabled":true}]}]}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "only one target") {
		t.Fatalf("duplicate provider target validation status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodDelete, "/admin/api/model-routes/coding-auto", "")
	if res.Code != http.StatusOK {
		t.Fatalf("delete route status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/model-routes/coding-auto", "")
	if res.Code != http.StatusNotFound {
		t.Fatalf("deleted route status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminResetsProviderModelHealth(t *testing.T) {
	st := testAdminStore(t)
	ctx := context.Background()
	_, _ = st.UpsertProvider(ctx, model.Provider{ID: "provider", Name: "provider", Type: model.ProviderOpenAICompatible, BaseURL: "https://provider.example", Enabled: true})
	_, _ = st.UpsertKey(ctx, model.Key{ID: "key", ProviderID: "provider", Name: "key", Secret: "secret", Enabled: true})
	status := http.StatusNotFound
	if err := st.RecordProviderModelFailure(ctx, "provider", "key", "model-a", &status, "model not found", store.FailurePolicy{Threshold: 1, ForceCooldown: true}); err != nil {
		t.Fatal(err)
	}

	svc := New(st, config.Default())
	res := authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-states/reset", `{"providerId":"provider","modelId":"model-a"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("reset status = %d, body = %s", res.Code, res.Body.String())
	}
	states, err := st.ListProviderModelStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 || states[0].KeyID != "key" || states[0].ConsecutiveFailures != 0 || states[0].CooldownUntil != nil || states[0].LastError != "" {
		t.Fatalf("states after reset = %#v", states)
	}
}

func TestAdminNormalizesGeminiModelRouteTarget(t *testing.T) {
	st := testAdminStore(t)
	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{ID: "gemini", Name: "gemini", Type: model.ProviderGeminiCompatible, BaseURL: "https://generativelanguage.googleapis.com", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())
	res := authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-routes", `{
		"id":"gemini-auto","enabled":true,
		"models":[{"name":"primary","priority":100,"enabled":true,"targets":[
			{"providerId":"gemini","upstreamModel":"models/gemini-2.0-flash","enabled":true}
		]}]
	}`)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var route model.ModelRoute
	if err := json.NewDecoder(res.Body).Decode(&route); err != nil {
		t.Fatal(err)
	}
	if got := route.Models[0].Targets[0].UpstreamModel; got != "gemini-2.0-flash" {
		t.Fatalf("normalized model = %q", got)
	}
}

func TestValidatePriorityRange(t *testing.T) {
	for _, test := range []struct {
		priority int
		valid    bool
	}{
		{priority: -1, valid: false},
		{priority: 0, valid: true},
		{priority: 1000, valid: true},
		{priority: 1001, valid: false},
	} {
		err := validatePriority("test", test.priority)
		if (err == nil) != test.valid {
			t.Errorf("validatePriority(%d) error = %v, valid = %t", test.priority, err, test.valid)
		}
	}
}

func TestAdminRejectsOutOfRangePriorities(t *testing.T) {
	st := testAdminStore(t)
	svc := New(st, config.Default())

	res := authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/providers", `{
		"id":"negative-provider","name":"Negative","type":"openai-compatible",
		"baseUrl":"https://negative.example","priority":-1
	}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "provider priority") {
		t.Fatalf("negative provider priority status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/providers", `{
		"id":"high-provider","name":"High","type":"openai-compatible",
		"baseUrl":"https://high.example","priority":1001
	}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "provider priority") {
		t.Fatalf("high provider priority status = %d, body = %s", res.Code, res.Body.String())
	}

	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{
		ID: "provider", Name: "Provider", Type: model.ProviderOpenAICompatible,
		BaseURL: "https://provider.example", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}

	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/keys", `{
		"id":"negative-key","providerId":"provider","name":"Negative",
		"secret":"secret","priority":-1,"enabled":true
	}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "key priority") {
		t.Fatalf("negative key priority status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/keys", `{
		"id":"high-key","providerId":"provider","name":"High",
		"secret":"secret","priority":1001,"enabled":true
	}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "key priority") {
		t.Fatalf("high key priority status = %d, body = %s", res.Code, res.Body.String())
	}

	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-routes", `{
		"id":"negative-route","enabled":true,"models":[{
			"name":"fallback","priority":-1,"enabled":true,
			"targets":[{"providerId":"provider","upstreamModel":"fallback","enabled":true}]
		}]
	}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "route model priority") {
		t.Fatalf("negative route model priority status = %d, body = %s", res.Code, res.Body.String())
	}
	res = authorizedAdminRequest(t, svc, http.MethodPost, "/admin/api/model-routes", `{
		"id":"high-route","enabled":true,"models":[{
			"name":"fallback","priority":1001,"enabled":true,
			"targets":[{"providerId":"provider","upstreamModel":"fallback","enabled":true}]
		}]
	}`)
	if res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "route model priority") {
		t.Fatalf("high route model priority status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestAdminPatchesProviderAndKeyForEditing(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	ctx := context.Background()
	_, err = st.UpsertProvider(ctx, model.Provider{
		ID:                    "newapi",
		Name:                  "old",
		Type:                  model.ProviderNewAPI,
		BaseURL:               "https://old.example/v1",
		Priority:              3,
		Enabled:               false,
		ModelMap:              map[string]string{"old": "model"},
		ModelAllowlistEnabled: true,
		ModelAllowlist:        []string{"model-a"},
		BalancePath:           "/api/user/self",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{
		ID:         "k1",
		ProviderID: "newapi",
		Name:       "old-key",
		Secret:     "old-secret",
		Priority:   4,
		Enabled:    false,
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := New(st, config.Default())
	req := httptest.NewRequest(http.MethodPatch, "http://localhost:18787/admin/api/providers/newapi", strings.NewReader(`{"name":"new","baseUrl":"https://new.example/v1","balancePath":"","priority":7,"modelMap":{"new":"model"}}`))
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()
	svc.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("provider patch expected 200, got %d: %s", res.Code, res.Body.String())
	}
	provider, err := st.GetProvider(ctx, "newapi")
	if err != nil {
		t.Fatal(err)
	}
	if provider == nil || provider.BaseURL != "https://new.example/v1" || provider.BalancePath != "" || provider.Priority != 7 || provider.Enabled || !provider.ModelAllowlistEnabled || len(provider.ModelAllowlist) != 1 || provider.ModelAllowlist[0] != "model-a" {
		t.Fatalf("provider = %#v", provider)
	}

	res = authorizedAdminRequest(t, svc, http.MethodPatch, "/admin/api/providers/newapi", `{"modelAllowlistEnabled":true,"modelAllowlist":["model-b","model-b"," model-c "]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("provider allowlist patch expected 200, got %d: %s", res.Code, res.Body.String())
	}
	provider, err = st.GetProvider(ctx, "newapi")
	if err != nil {
		t.Fatal(err)
	}
	if len(provider.ModelAllowlist) != 2 || provider.ModelAllowlist[0] != "model-b" || provider.ModelAllowlist[1] != "model-c" {
		t.Fatalf("provider allowlist = %#v", provider.ModelAllowlist)
	}

	req = httptest.NewRequest(http.MethodPatch, "http://localhost:18787/admin/api/keys/k1", strings.NewReader(`{"name":"new-key","secret":"","priority":9}`))
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res = httptest.NewRecorder()
	svc.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("key patch expected 200, got %d: %s", res.Code, res.Body.String())
	}
	keys, err := st.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := findKey(keys, "k1")
	if got == nil || got.Name != "new-key" || got.Secret != "old-secret" || got.Priority != 9 || got.Enabled {
		t.Fatalf("key = %#v", got)
	}
}

func TestAdminReturnsNotFoundForMissingResources(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st, config.Default())
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "get provider", method: http.MethodGet, path: "/admin/api/providers/missing"},
		{name: "patch provider", method: http.MethodPatch, path: "/admin/api/providers/missing", body: `{"name":"missing"}`},
		{name: "delete provider", method: http.MethodDelete, path: "/admin/api/providers/missing"},
		{name: "patch key", method: http.MethodPatch, path: "/admin/api/keys/missing", body: `{"name":"missing"}`},
		{name: "delete key", method: http.MethodDelete, path: "/admin/api/keys/missing"},
		{name: "prefer key", method: http.MethodPost, path: "/admin/api/keys/missing/prefer"},
		{name: "reset key", method: http.MethodPost, path: "/admin/api/keys/missing/reset"},
		{name: "patch gateway key", method: http.MethodPatch, path: "/admin/api/gateway-keys/missing", body: `{"name":"missing"}`},
		{name: "delete gateway key", method: http.MethodDelete, path: "/admin/api/gateway-keys/missing"},
		{name: "rotate gateway key", method: http.MethodPost, path: "/admin/api/gateway-keys/missing/rotate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://localhost:18787"+tt.path, strings.NewReader(tt.body))
			req.Host = "localhost:18787"
			req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
			res := httptest.NewRecorder()

			svc.handle(res, req)

			if res.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestParseModelCountSupportsNestedNewAPIShapes(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "openai data", body: `{"object":"list","data":[{"id":"gpt-4o"},{"id":"gpt-4o-mini"}]}`, want: 2},
		{name: "nested data", body: `{"success":true,"data":{"data":[{"id":"gpt-4o"},{"id":"claude"},{"id":"gemini"}]}}`, want: 3},
		{name: "items", body: `{"data":{"items":[{"id":"a"},{"id":"b"},{"id":"c"},{"id":"d"}]}}`, want: 4},
		{name: "models object", body: `{"data":{"models":{"gpt-4o":{},"claude":{},"gemini":{}}}}`, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := parseModelCount([]byte(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			if count == nil || *count != tt.want {
				t.Fatalf("count = %#v, want %d", count, tt.want)
			}
		})
	}
}

func TestParseModelPageSupportsAnthropicCursor(t *testing.T) {
	models, count, cursor, err := parseModelPage([]byte(`{
		"data":[{"id":"claude-sonnet"}],"has_more":true,"last_id":"claude-sonnet"
	}`), model.ProviderAnthropicCompatible)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0] != "claude-sonnet" || count == nil || *count != 1 {
		t.Fatalf("models = %#v, count = %#v", models, count)
	}
	if cursor == nil || cursor.QueryParam != "after_id" || cursor.Value != "claude-sonnet" {
		t.Fatalf("cursor = %#v", cursor)
	}
}

func TestAdminRejectsInvalidJSONBodies(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st, config.Default())

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "malformed", method: http.MethodPost, path: "/admin/api/gateway-keys", body: `{"name":`},
		{name: "unknown field", method: http.MethodPost, path: "/admin/api/providers", body: `{"name":"p","type":"openai-compatible","baseUrl":"https://example.com","unexpected":true}`},
		{name: "multiple objects", method: http.MethodPost, path: "/admin/api/keys", body: `{"providerId":"p","name":"k","secret":"s"}{"providerId":"p2"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://localhost:18787"+tt.path, strings.NewReader(tt.body))
			req.Host = "localhost:18787"
			req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
			res := httptest.NewRecorder()
			svc.handle(res, req)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
			}
		})
	}
	keys, err := st.ListGatewayKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 0 {
		t.Fatalf("gateway keys created from invalid JSON: %#v", keys)
	}
}

func TestAdminRejectsOversizedJSONBody(t *testing.T) {
	svc := New(nil, config.Default())
	body := `{"name":"` + strings.Repeat("x", maxAdminRequestBody) + `"}`
	req := httptest.NewRequest(http.MethodPost, "http://localhost:18787/admin/api/gateway-keys", strings.NewReader(body))
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()
	svc.handle(res, req)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
}

func TestDashboardReturnsCompleteBootstrapPayload(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	cfg := config.Default()
	cfg.Storage.Timezone = "America/New_York"
	ctx := context.Background()
	if _, err := st.UpsertProvider(ctx, model.Provider{ID: "persisted", Name: "persisted", Type: model.ProviderOpenAICompatible, BaseURL: "https://example.test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceProviderModels(ctx, "persisted", []string{"model-a"}); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordProviderModelDiscoverySuccess(ctx, "persisted", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertModelRoute(ctx, model.ModelRoute{ID: "logical-model", Name: "Logical", Enabled: true, Models: []model.ModelRouteModel{{Name: "model-a", Priority: 100, Enabled: true, Targets: []model.ModelRouteTarget{{ProviderID: "persisted", UpstreamModel: "model-a", Enabled: true}}}}}); err != nil {
		t.Fatal(err)
	}
	svc := New(st, cfg)
	req := httptest.NewRequest(http.MethodGet, "http://localhost:18787/admin/api/dashboard", nil)
	req.Host = "localhost:18787"
	req.Header.Set("X-Admin-Token", config.Default().Server.AdminToken)
	res := httptest.NewRecorder()
	svc.handle(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}

	var payload map[string]json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"service", "stats", "providers", "keys", "gatewayKeys", "balances", "logs", "providerModels", "modelDiscovery", "modelRoutes", "modelStates", "routing", "snippets"} {
		if _, ok := payload[field]; !ok {
			t.Errorf("dashboard payload missing %q", field)
		}
	}
	var service struct {
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal(payload["service"], &service); err != nil {
		t.Fatal(err)
	}
	if service.Timezone != cfg.Storage.Timezone {
		t.Fatalf("service timezone = %q, want %q", service.Timezone, cfg.Storage.Timezone)
	}
	var providerModels map[string][]string
	if err := json.Unmarshal(payload["providerModels"], &providerModels); err != nil {
		t.Fatal(err)
	}
	if got := providerModels["persisted"]; len(got) != 1 || got[0] != "model-a" {
		t.Fatalf("provider models = %#v", providerModels)
	}
	var routes []model.ModelRoute
	if err := json.Unmarshal(payload["modelRoutes"], &routes); err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].ID != "logical-model" {
		t.Fatalf("model routes = %#v", routes)
	}
}
