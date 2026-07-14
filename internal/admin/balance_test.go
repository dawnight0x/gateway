package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
)

func TestRefreshBalancesUsesProviderBalancePath(t *testing.T) {
	var capturedAuth string
	var capturedGoogleKey string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedGoogleKey = r.Header.Get("x-goog-api-key")
		if r.URL.Path != "/api/user/self" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"balance":"12.5","currency":"USD","quota_used":3,"quota_limit":20}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:          "newapi",
		Name:        "newapi",
		Type:        model.ProviderNewAPI,
		BaseURL:     upstream.URL,
		BalancePath: "/api/user/self",
		Enabled:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "ok" {
		t.Fatalf("results = %#v", results)
	}
	if capturedAuth != "Bearer secret" {
		t.Fatalf("auth = %q", capturedAuth)
	}
	if capturedGoogleKey != "" {
		t.Fatalf("google api key header = %q", capturedGoogleKey)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("balances = %d, want 1", len(items))
	}
	if items[0].Balance == nil || *items[0].Balance != 12.5 {
		t.Fatalf("balance = %#v", items[0].Balance)
	}
	if items[0].QuotaLimit == nil || *items[0].QuotaLimit != 20 {
		t.Fatalf("quota limit = %#v", items[0].QuotaLimit)
	}
}

func TestRefreshBalancesUsesDefaultNewAPIPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"quota":1973659.94,"used_quota":"250","request_count":10}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL,
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "ok" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("balances = %d, want 1", len(items))
	}
	if items[0].Balance == nil || *items[0].Balance != 1973659.94 {
		t.Fatalf("remaining balance = %#v", items[0].Balance)
	}
	if items[0].Source != "new-api:/api/user/self" {
		t.Fatalf("source = %s", items[0].Source)
	}
}

func TestRefreshBalancesFallsBackToNewAPIKeyBalance(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/user/self":
			_, _ = w.Write([]byte(`{"message":"Unauthorized, invalid access token"}`))
		case "/api/usage/token/":
			_, _ = w.Write([]byte(`{"data":{"total_granted":100,"total_used":"25","total_available":75}}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL,
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "ok" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "ok" || items[0].Balance == nil || *items[0].Balance != 75 {
		t.Fatalf("balances = %#v", items)
	}
	if items[0].Source != "new-api:/api/usage/token/" {
		t.Fatalf("source = %s", items[0].Source)
	}
}

func TestRefreshBalancesStripsV1BaseForNewAPIManagementPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user/self" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/api/usage/token/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"total_granted":20,"total_used":8,"total_available":12}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL + "/v1",
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "ok" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "ok" || items[0].Balance == nil || *items[0].Balance != 12 {
		t.Fatalf("balances = %#v", items)
	}
	if items[0].Currency != "" {
		t.Fatalf("currency = %s", items[0].Currency)
	}
}

func TestRefreshBalancesMarksNewAPIUnlimitedQuota(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user/self" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/api/usage/token/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"unlimited_quota":true}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL + "/v1",
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "unlimited" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("balances = %d, want 1", len(items))
	}
	if items[0].Status != "unlimited" || items[0].Balance != nil || items[0].QuotaUsed != nil || items[0].QuotaLimit != nil {
		t.Fatalf("balance = %#v", items[0])
	}
}

func TestRefreshBalancesKeepsNewAPINumericBalanceWhenUnlimitedQuotaPresent(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/user/self" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"balance":1973659.94,"currency":"USD","unlimited_quota":true}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL,
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "ok" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("balances = %d, want 1", len(items))
	}
	if items[0].Status != "ok" || items[0].Balance == nil || *items[0].Balance != 1973659.94 {
		t.Fatalf("balance = %#v", items[0])
	}
}

func TestRefreshBalancesMarksNewAPIOverThresholdAsUnlimited(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/user/self" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/api/usage/token/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"total_granted":277415660427,"total_used":277420035617,"total_available":-4375190}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "newapi",
		Name:    "newapi",
		Type:    model.ProviderNewAPI,
		BaseURL: upstream.URL,
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "unlimited" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("balances = %d, want 1", len(items))
	}
	if items[0].Status != "unlimited" || items[0].Balance != nil || items[0].QuotaUsed != nil || items[0].QuotaLimit != nil {
		t.Fatalf("balance = %#v", items[0])
	}
}

func TestRefreshBalancesUsesSub2APIUsageEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"mode":"quota_limited","quota":{"limit":50,"used":10,"remaining":40,"unit":"USD"}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:      "sub2api",
		Name:    "sub2api",
		Type:    model.ProviderSub2API,
		BaseURL: upstream.URL,
		Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "sub2api", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "ok" {
		t.Fatalf("results = %#v", results)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("balances = %d, want 1", len(items))
	}
	if items[0].Balance == nil || *items[0].Balance != 40 {
		t.Fatalf("balance = %#v", items[0].Balance)
	}
	if items[0].QuotaLimit == nil || *items[0].QuotaLimit != 50 {
		t.Fatalf("quota limit = %#v", items[0].QuotaLimit)
	}
	if items[0].QuotaUsed == nil || *items[0].QuotaUsed != 10 {
		t.Fatalf("quota used = %#v", items[0].QuotaUsed)
	}
	if items[0].Currency != "USD" {
		t.Fatalf("currency = %s", items[0].Currency)
	}
}

func TestRefreshBalancesClassifiesAuthError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key sk-secret"}}`))
	}))
	defer upstream.Close()

	st := testAdminStore(t)
	ctx := context.Background()
	_, err := st.UpsertProvider(ctx, model.Provider{
		ID:          "newapi",
		Name:        "newapi",
		Type:        model.ProviderNewAPI,
		BaseURL:     upstream.URL,
		BalancePath: "/api/user/self",
		Enabled:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.UpsertKey(ctx, model.Key{ID: "k1", ProviderID: "newapi", Name: "k1", Secret: "secret", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st, config.Default())

	results := svc.refreshBalances(ctx)
	if len(results) != 1 || results[0].Status != "auth_error" {
		t.Fatalf("results = %#v", results)
	}
	if strings.Contains(results[0].Error, "secret") {
		t.Fatalf("result leaked upstream secret: %s", results[0].Error)
	}
	items, err := st.ListBalances(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Status != "auth_error" {
		t.Fatalf("balances = %#v", items)
	}
	if strings.Contains(items[0].Error, "secret") {
		t.Fatalf("stored balance error leaked upstream secret: %s", items[0].Error)
	}
}

func testAdminStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "gateway.db"), filepath.Join(dir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
