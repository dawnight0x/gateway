package upstreamhttp

import (
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestClientBoundsSameOriginRedirects(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		hop, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
		http.Redirect(w, r, "/"+strconv.Itoa(hop+1), http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	// No total client timeout: the redirect policy itself must bound the loop.
	_, err := New(time.Second, 0, 4).Get(server.URL + "/0")
	if err == nil || !strings.Contains(err.Error(), "10 upstream redirects") {
		t.Fatalf("redirect error = %v", err)
	}
	if got := requests.Load(); got != 10 {
		t.Fatalf("requests = %d, want 10", got)
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
