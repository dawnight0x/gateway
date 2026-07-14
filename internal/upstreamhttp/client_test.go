package upstreamhttp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientBlocksCrossOriginRedirectWithoutLeakingCustomKey(t *testing.T) {
	var leaked atomic.Bool
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "" || r.Header.Get("x-goog-api-key") != "" {
			leaked.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL+"/capture", http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	req, _ := http.NewRequest(http.MethodPost, source.URL, strings.NewReader(`{"ok":true}`))
	req.Header.Set("x-api-key", "secret")
	_, err := New(time.Second, time.Second, 4).Do(req)
	if err == nil || !strings.Contains(err.Error(), "cross-origin") {
		t.Fatalf("redirect error = %v", err)
	}
	if leaked.Load() {
		t.Fatal("custom API key reached cross-origin redirect target")
	}
}

func TestClientAllowsSameOriginRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", http.StatusTemporaryRedirect)
			return
		}
		if r.Header.Get("x-api-key") != "secret" {
			t.Fatalf("same-origin key = %q", r.Header.Get("x-api-key"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/start", nil)
	req.Header.Set("x-api-key", "secret")
	resp, err := New(time.Second, time.Second, 4).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
