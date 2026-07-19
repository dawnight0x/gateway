package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/protocol"
	"local-ai-gateway/internal/redact"
	"local-ai-gateway/internal/upstreamhttp"
)

type keyTestResult struct {
	ProviderID       string   `json:"providerId,omitempty"`
	KeyID            string   `json:"keyId"`
	Endpoint         string   `json:"endpoint,omitempty"`
	TokenEndpoint    string   `json:"tokenEndpoint,omitempty"`
	Status           string   `json:"status"`
	ConnectionStatus string   `json:"connectionStatus,omitempty"`
	TokenStatus      string   `json:"tokenStatus,omitempty"`
	StatusCode       int      `json:"statusCode,omitempty"`
	TokenStatusCode  int      `json:"tokenStatusCode,omitempty"`
	LatencyMS        int64    `json:"latencyMs,omitempty"`
	TokenLatencyMS   int64    `json:"tokenLatencyMs,omitempty"`
	Model            string   `json:"model,omitempty"`
	Models           []string `json:"models,omitempty"`
	ModelCount       *int     `json:"modelCount,omitempty"`
	PromptTokens     *int     `json:"promptTokens,omitempty"`
	CompletionTokens *int     `json:"completionTokens,omitempty"`
	TotalTokens      *int     `json:"totalTokens,omitempty"`
	Error            string   `json:"error,omitempty"`
	CheckedAt        string   `json:"checkedAt"`
}

func (s *Service) testUpstreamKey(ctx context.Context, id string) keyTestResult {
	result := keyTestResult{KeyID: id, Status: "not_found", Error: "upstream key not found", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	keys, err := s.store.ListKeys(ctx)
	if err != nil {
		result.Status = "store_error"
		result.Error = err.Error()
		return result
	}
	var key *model.Key
	for i := range keys {
		if keys[i].ID == id {
			key = &keys[i]
			break
		}
	}
	if key == nil {
		return result
	}
	result.ProviderID = key.ProviderID
	if !key.ProviderEnabled || !key.Enabled {
		result.Status = "disabled"
		result.Error = "provider or key is disabled"
		return result
	}
	if strings.TrimSpace(key.ProviderBaseURL) == "" {
		result.Status = "config_error"
		result.Error = "provider base URL is empty"
		return result
	}
	paths := testPathsForKey(*key)
	var last keyTestResult
	for _, path := range paths {
		attempt := s.tryTestKeyPath(ctx, *key, path)
		if attempt.Status == "ok" {
			return attempt
		}
		last = attempt
		if attempt.Status != "not_found" {
			break
		}
	}
	if last.Status == "" {
		last = result
	}
	return last
}

func (s *Service) tryTestKeyPath(ctx context.Context, key model.Key, path string) keyTestResult {
	result := s.discoverModelsAtPath(ctx, key, path)
	if result.Status == "ok" {
		s.tryTokenProbe(ctx, key, &result)
	}
	return result
}

func (s *Service) discoverModelsAtPath(ctx context.Context, key model.Key, path string) (result keyTestResult) {
	result = keyTestResult{
		ProviderID: key.ProviderID,
		KeyID:      key.ID,
		Status:     "unknown",
		CheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	defer func() {
		if result.Status == "ok" {
			return
		}
		if err := s.store.RecordProviderModelDiscoveryFailure(ctx, key.ProviderID, result.Status, result.Error); err != nil {
			slog.Warn("record model discovery failure failed", "provider_id", key.ProviderID, "error", err)
		}
	}()
	endpoint, err := joinURL(key.ProviderBaseURL, path)
	if err != nil {
		result.Status = "config_error"
		result.Error = err.Error()
		return result
	}
	result.Endpoint = endpoint
	timeoutSeconds := s.cfg.ModelDiscovery.TimeoutSeconds
	if timeoutSeconds <= 0 || timeoutSeconds > 120 {
		timeoutSeconds = 30
	}
	client := keyTestHTTPClient(timeoutSeconds)
	var resp *http.Response
	for attempt := 0; attempt < 2; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
		if err != nil {
			cancel()
			result.Status = "config_error"
			result.Error = err.Error()
			return result
		}
		req.Header.Set("Accept", "application/json")
		setUpstreamAuthHeaders(req, key)
		req.Header.Set("User-Agent", "Local-AI-Gateway/1.0")

		start := time.Now()
		resp, err = client.Do(req)
		result.LatencyMS = time.Since(start).Milliseconds()
		if err == nil {
			defer cancel()
			break
		}
		cancel()
		if attempt == 1 || !retryableKeyTestError(err) {
			result.Status = "network_error"
			result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
			return result
		}
		if err := waitForRetry(ctx, 300*time.Millisecond); err != nil {
			result.Status = "network_error"
			result.Error = err.Error()
			return result
		}
	}
	defer resp.Body.Close()
	result.StatusCode = resp.StatusCode
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result.Status = "network_error"
		result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
		return result
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Status = classifyBalanceError(resp.StatusCode, string(body))
		result.Error = redactSecret(sanitizeBalanceError(fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))), key.Secret)
		return result
	}
	models, count, err := parseModels(body)
	if err != nil {
		result.Status = "parse_error"
		result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
		return result
	}
	result.Status = "ok"
	result.ConnectionStatus = "ok"
	result.ModelCount = count
	result.Models = trimModelList(models, 300)
	if len(models) == 0 {
		result.Status = "empty"
		result.Error = "model endpoint returned no models"
		return result
	}
	if err := s.store.ReplaceProviderModels(ctx, key.ProviderID, models); err != nil {
		result.Status = "storage_error"
		result.Error = "connected successfully but failed to cache discovered models"
		return result
	}
	if err := s.store.RecordProviderModelDiscoverySuccess(ctx, key.ProviderID, len(models)); err != nil {
		result.Status = "storage_error"
		result.Error = "connected successfully but failed to save model discovery state"
		return result
	}
	result.Model = selectProbeModel(key, models)
	return result
}

func (s *Service) tryTokenProbe(ctx context.Context, key model.Key, result *keyTestResult) {
	if strings.TrimSpace(result.Model) == "" {
		result.TokenStatus = "skipped"
		return
	}
	endpoint, body, err := tokenProbeRequest(key, result.Model)
	if err != nil {
		result.TokenStatus = "config_error"
		result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
		return
	}
	result.TokenEndpoint = endpoint
	timeoutSeconds := s.cfg.Routing.TimeoutSeconds
	if timeoutSeconds <= 0 || timeoutSeconds > 30 {
		timeoutSeconds = 30
	}
	client := keyTestHTTPClient(timeoutSeconds)
	var resp *http.Response
	for attempt := 0; attempt < 2; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			cancel()
			result.TokenStatus = "config_error"
			result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
			return
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Local-AI-Gateway/1.0")
		setUpstreamAuthHeaders(req, key)

		start := time.Now()
		resp, err = client.Do(req)
		result.TokenLatencyMS = time.Since(start).Milliseconds()
		if err == nil {
			defer cancel()
			break
		}
		cancel()
		if attempt == 1 || !retryableKeyTestError(err) {
			result.TokenStatus = "network_error"
			result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
			return
		}
		if err := waitForRetry(ctx, 300*time.Millisecond); err != nil {
			result.TokenStatus = "network_error"
			result.Error = err.Error()
			return
		}
	}
	defer resp.Body.Close()
	result.TokenStatusCode = resp.StatusCode
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result.TokenStatus = "network_error"
		result.Error = redactSecret(sanitizeBalanceError(err.Error()), key.Secret)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.TokenStatus = classifyBalanceError(resp.StatusCode, string(respBody))
		result.Error = redactSecret(sanitizeBalanceError(fmt.Sprintf("token test status %d: %s", resp.StatusCode, string(respBody))), key.Secret)
		return
	}
	usage := protocol.ExtractUsage(respBody)
	result.PromptTokens = usage.PromptTokens
	result.CompletionTokens = usage.CompletionTokens
	result.TotalTokens = usage.TotalTokens
	result.TokenStatus = "ok"
}

func keyTestHTTPClient(timeoutSeconds int) *http.Client {
	timeout := time.Duration(timeoutSeconds) * time.Second
	return upstreamhttp.New(timeout, timeout, 4)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func retryableKeyTestError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "timeout") ||
		strings.Contains(text, "deadline") ||
		strings.Contains(text, "tls handshake") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "temporary") ||
		strings.Contains(text, "eof")
}

func setUpstreamAuthHeaders(req *http.Request, key model.Key) {
	switch key.ProviderType {
	case model.ProviderGeminiCompatible:
		req.Header.Set("x-goog-api-key", key.Secret)
	case model.ProviderAnthropicCompatible:
		req.Header.Set("x-api-key", key.Secret)
	default:
		req.Header.Set("Authorization", "Bearer "+key.Secret)
		req.Header.Set("x-api-key", key.Secret)
	}
}

func testPathsForKey(key model.Key) []string {
	base := strings.TrimRight(strings.ToLower(key.ProviderBaseURL), "/")
	switch key.ProviderType {
	case model.ProviderOpenAICompatible, model.ProviderNewAPI, model.ProviderSub2API:
		if strings.HasSuffix(base, "/v1") {
			return []string{"/models"}
		}
		return []string{"/models", "/v1/models"}
	case model.ProviderGeminiCompatible:
		return []string{"/v1beta/models"}
	case model.ProviderAnthropicCompatible:
		return []string{"/v1/models", "/models"}
	default:
		return []string{"/models", "/v1/models"}
	}
}

func tokenProbeRequest(key model.Key, modelID string) (string, []byte, error) {
	modelID = strings.TrimSpace(modelID)
	base := strings.TrimRight(strings.ToLower(key.ProviderBaseURL), "/")
	var path string
	var payload map[string]any
	switch key.ProviderType {
	case model.ProviderGeminiCompatible:
		path = "/v1beta/models/" + url.PathEscape(modelID) + ":generateContent"
		payload = map[string]any{
			"contents": []map[string]any{{"role": "user", "parts": []map[string]string{{"text": "ping"}}}},
			"generationConfig": map[string]any{
				"maxOutputTokens": 1,
			},
		}
	case model.ProviderAnthropicCompatible:
		if strings.HasSuffix(base, "/v1") {
			path = "/messages"
		} else {
			path = "/v1/messages"
		}
		payload = map[string]any{
			"model":      modelID,
			"max_tokens": 1,
			"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		}
	default:
		if strings.HasSuffix(base, "/v1") {
			path = "/chat/completions"
		} else {
			path = "/v1/chat/completions"
		}
		payload = map[string]any{
			"model":      modelID,
			"messages":   []map[string]string{{"role": "user", "content": "ping"}},
			"max_tokens": 1,
			"stream":     false,
		}
	}
	endpoint, err := joinURL(key.ProviderBaseURL, path)
	if err != nil {
		return "", nil, err
	}
	body, err := json.Marshal(payload)
	return endpoint, body, err
}

func parseModelCount(body []byte) (*int, error) {
	_, count, err := parseModels(body)
	return count, err
}

func parseModels(body []byte) ([]string, *int, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, nil, nil
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, err
	}
	parsed := collectModels(raw, false)
	return parsed.IDs, parsed.Count, nil
}

type modelParseResult struct {
	IDs   []string
	Count *int
}

func countModels(raw any, collection bool) *int {
	return collectModels(raw, collection).Count
}

func collectModels(raw any, collection bool) modelParseResult {
	switch value := raw.(type) {
	case []any:
		count := len(value)
		ids := make([]string, 0, len(value))
		for _, item := range value {
			if id := modelID(item); id != "" {
				ids = append(ids, id)
			}
		}
		return modelParseResult{IDs: ids, Count: &count}
	case map[string]any:
		if collection {
			count := len(value)
			ids := make([]string, 0, len(value))
			for key, item := range value {
				id := modelID(item)
				if id == "" {
					id = key
				}
				if id != "" {
					ids = append(ids, id)
				}
			}
			return modelParseResult{IDs: ids, Count: &count}
		}
		for _, key := range []string{"data", "models", "items"} {
			if child, ok := value[key]; ok {
				if parsed := collectModels(child, key == "models" || key == "items"); parsed.Count != nil {
					return parsed
				}
			}
		}
		for _, child := range value {
			if nested, ok := child.(map[string]any); ok {
				if parsed := collectModels(nested, false); parsed.Count != nil {
					return parsed
				}
			}
		}
	}
	return modelParseResult{}
}

func modelID(raw any) string {
	switch item := raw.(type) {
	case string:
		return item
	case map[string]any:
		for _, key := range []string{"id", "name", "model"} {
			if id, ok := item[key].(string); ok && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	}
	return ""
}

func selectProbeModel(key model.Key, models []string) string {
	for _, id := range models {
		if strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	for _, mapped := range key.ProviderModelMap {
		if strings.TrimSpace(mapped) != "" {
			return strings.TrimSpace(mapped)
		}
	}
	for public := range key.ProviderModelMap {
		if strings.TrimSpace(public) != "" {
			return strings.TrimSpace(public)
		}
	}
	return ""
}

func trimModelList(models []string, limit int) []string {
	if limit <= 0 || len(models) == 0 {
		return nil
	}
	out := make([]string, 0, len(models))
	seen := map[string]bool{}
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		out = append(out, model)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func redactSecret(message, secret string) string {
	return redact.Secret(message, secret)
}
