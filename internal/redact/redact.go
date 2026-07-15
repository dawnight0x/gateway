// Package redact centralizes removal of upstream secrets and sensitive header
// material from error messages before they are logged or returned to clients.
package redact

import "strings"

// authHeaderMarkers identify lines that carry credentials. If an error message
// echoes any of them we drop the whole message rather than risk leaking a token.
var authHeaderMarkers = []string{
	"authorization:", "bearer ", "x-api-key:", "x-goog-api-key:", "anthropic-auth-token:",
}

// Secret replaces every occurrence of secret in message with a fixed mask.
// It performs no other transformation and is safe to call with an empty secret.
func Secret(message, secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return message
	}
	return strings.ReplaceAll(message, secret, "***")
}

// Message sanitizes an upstream error message: it trims the text, masks the
// provided secret, drops the message entirely if it still contains an auth
// header marker, and caps the length. fallback is returned when message is empty.
func Message(message, secret, fallback string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return fallback
	}
	message = Secret(message, secret)
	lower := strings.ToLower(message)
	for _, marker := range authHeaderMarkers {
		if strings.Contains(lower, marker) {
			return fallback + "; sensitive details redacted"
		}
	}
	if len(message) > 300 {
		return message[:300]
	}
	return message
}
