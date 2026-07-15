package redact

import (
	"strings"
	"testing"
)

func TestSecret(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		message string
		secret  string
		want    string
	}{
		{"empty secret is a no-op", "hello sk-abc", "", "hello sk-abc"},
		{"whitespace secret is a no-op", "hello sk-abc", "   ", "hello sk-abc"},
		{"replaces every occurrence", "sk-abc and sk-abc", "sk-abc", "*** and ***"},
		{"absent secret unchanged", "nothing here", "sk-abc", "nothing here"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Secret(tt.message, tt.secret); got != tt.want {
				t.Fatalf("Secret(%q,%q)=%q, want %q", tt.message, tt.secret, got, tt.want)
			}
		})
	}
}

func TestMessage(t *testing.T) {
	t.Parallel()
	const fallback = "upstream request failed"

	t.Run("empty returns fallback", func(t *testing.T) {
		if got := Message("   ", "", fallback); got != fallback {
			t.Fatalf("got %q, want %q", got, fallback)
		}
	})

	t.Run("masks secret", func(t *testing.T) {
		got := Message("token sk-secret rejected", "sk-secret", fallback)
		if strings.Contains(got, "sk-secret") {
			t.Fatalf("secret leaked: %q", got)
		}
		if !strings.Contains(got, "***") {
			t.Fatalf("expected mask, got %q", got)
		}
	})

	t.Run("drops message on auth header marker", func(t *testing.T) {
		got := Message("failed: Authorization: Bearer xyz", "", fallback)
		if !strings.Contains(got, "sensitive details redacted") {
			t.Fatalf("expected redaction, got %q", got)
		}
		if strings.Contains(got, "xyz") {
			t.Fatalf("token leaked: %q", got)
		}
	})

	t.Run("drops message when secret substitution reveals bearer", func(t *testing.T) {
		// Even after masking the known secret, a lingering bearer marker forces a drop.
		got := Message("bearer leftover-token", "", fallback)
		if !strings.Contains(got, "sensitive details redacted") {
			t.Fatalf("expected redaction, got %q", got)
		}
	})

	t.Run("caps length at 300", func(t *testing.T) {
		long := strings.Repeat("a", 500)
		got := Message(long, "", fallback)
		if len(got) != 300 {
			t.Fatalf("expected length 300, got %d", len(got))
		}
	})

	t.Run("short clean message passes through", func(t *testing.T) {
		if got := Message("model not found", "", fallback); got != "model not found" {
			t.Fatalf("got %q", got)
		}
	})
}
