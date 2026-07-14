package main

import (
	"strings"
	"testing"

	"local-ai-gateway/internal/config"
)

func TestAdminLoginURLUsesFragmentToken(t *testing.T) {
	cfg := config.Default()
	cfg.Server.AdminToken = "gat-test token"

	got := adminLoginURL(cfg)
	if !strings.HasPrefix(got, "http://localhost:18787/admin#token=") {
		t.Fatalf("admin login url = %q", got)
	}
	if strings.Contains(got, "?") {
		t.Fatalf("admin token should not be placed in query string: %q", got)
	}
	if strings.Contains(got, "gat-test token") {
		t.Fatalf("admin token should be escaped in URL: %q", got)
	}
}
