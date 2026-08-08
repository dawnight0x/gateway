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
	"sync"
	"time"

	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/protocol"
	"local-ai-gateway/internal/redact"
	"local-ai-gateway/internal/upstreamhttp"
)

type keyTestResult struct {
	ProviderID               string   `json:"providerId,omitempty"`
	KeyID                    string   `json:"keyId"`
	Endpoint                 string   `json:"endpoint,omitempty"`
	TokenEndpoint            string   `json:"tokenEndpoint,omitempty"`
	TokenProtocol            string   `json:"tokenProtocol,omitempty"`
	Capabilities             []string `json:"capabilities,omitempty"`
	Status                   string   `json:"status"`
	ConnectionStatus         string   `json:"connectionStatus,omitempty"`
	TokenStatus              string   `json:"tokenStatus,omitempty"`
	StatusCode               int      `json:"statusCode,omitempty"`
	TokenStatusCode          int      `json:"tokenStatusCode,omitempty"`
	LatencyMS                int64    `json:"latencyMs,omitempty"`
	TokenLatencyMS           int64    `json:"tokenLatencyMs,omitempty"`
	Model                    string   `json:"model,omitempty"`
	Models                   []string `json:"models,omitempty"`
	ModelCount               *int     `json:"modelCount,omitempty"`
	PromptTokens             *int     `json:"promptTokens,omitempty"`
	CompletionTokens         *int     `json:"completionTokens,omitempty"`
	TotalTokens              *int     `json:"totalTokens,omitempty"`
	CacheCreationInputTokens *int     `json:"cacheCreationInputTokens,omitempty"`
	CacheReadInputTokens     *int     `json:"cacheReadInputTokens,omitempty"`
	ReasoningTokens          *int     `json:"reasoningTokens,omitempty"`
	ToolUseTokens            *int     `json:"toolUseTokens,omitempty"`
	CachedContentTokens      *int     `json:"cachedContentTokens,omitempty"`
	ThoughtsTokens           *int     `json:"thoughtsTokens,omitempty"`
	Error                    string   `json:"error,omitempty"`
	CheckedAt                string   `json:"checkedAt"`
}

const (
	maxModelDiscoveryPages  = 100
	maxDiscoveredModelCount = 5000
)

type modelPageCursor struct {
	QueryParam string
	Value      string
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

func (s *Service) testAllUpstreamKeys(ctx context.Context) ([]keyTestResult, error) {
	keys, err := s.store.ListKeys(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]keyTestResult, len(keys))
	if len(keys) == 0 {
		return results, nil
	}
	jobs := make(chan int)
	workers := minInt(len(keys), 4)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				results[index] = s.testUpstreamKey(ctx, keys[index].ID)
			}
		}()
	}
	for index := range keys {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return results, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) tryTestKeyPath(ctx context.Context, key model.Key, path string) keyTestResult {
	result := s.discoverModelsAtPath(ctx, key, path)
	if result.Status == "ok" {
		result.Capabilities = providerCapabilities(key.ProviderType)
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
	timeoutSeconds := s.cfg.ModelDiscovery.TimeoutSeconds
	if timeoutSeconds <= 0 || timeoutSeconds > 120 {
		timeoutSeconds = 30
	}
	client := keyTestHTTPClient(timeoutSeconds)
	currentEndpoint := endpoint
	models := make([]string, 0)
	seenModels := make(map[string]struct{})
	seenCursors := make(map[string]struct{})
	completed := false
	for page := 0; page < maxModelDiscoveryPages; page++ {
		result.Endpoint = currentEndpoint
		status, body, latency, requestErr := fetchModelPage(ctx, client, key, currentEndpoint, timeoutSeconds)
		result.LatencyMS += latency
		if requestErr != nil {
			result.Status = "network_error"
			result.Error = redactSecret(sanitizeBalanceError(requestErr.Error()), key.Secret)
			return result
		}
		result.StatusCode = status
		if status < 200 || status >= 300 {
			result.Status = classifyBalanceError(status, string(body))
			result.Error = redactSecret(sanitizeBalanceError(fmt.Sprintf("status %d: %s", status, string(body))), key.Secret)
			return result
		}
		pageModels, _, cursor, parseErr := parseModelPage(body, key.ProviderType)
		if parseErr != nil {
			result.Status = "parse_error"
			result.Error = redactSecret(sanitizeBalanceError(parseErr.Error()), key.Secret)
			return result
		}
		for _, modelID := range pageModels {
			modelID = model.NormalizeModelID(key.ProviderType, modelID)
			if modelID == "" || len(modelID) > 512 {
				continue
			}
			if _, exists := seenModels[modelID]; exists {
				continue
			}
			seenModels[modelID] = struct{}{}
			models = append(models, modelID)
			if len(models) >= maxDiscoveredModelCount {
				completed = true
				break
			}
		}
		if completed || cursor == nil {
			completed = true
			break
		}
		cursorKey := cursor.QueryParam + "\x00" + cursor.Value
		if _, duplicate := seenCursors[cursorKey]; duplicate {
			result.Status = "parse_error"
			result.Error = "model endpoint repeated a pagination cursor"
			return result
		}
		seenCursors[cursorKey] = struct{}{}
		currentEndpoint, err = withModelPageCursor(endpoint, *cursor)
		if err != nil {
			result.Status = "parse_error"
			result.Error = err.Error()
			return result
		}
	}
	if !completed {
		result.Status = "parse_error"
		result.Error = fmt.Sprintf("model endpoint exceeded %d pagination pages", maxModelDiscoveryPages)
		return result
	}
	count := len(models)
	result.ConnectionStatus = "ok"
	result.ModelCount = &count
	result.Models = trimModelList(models, 300)
	if len(models) == 0 {
		result.Status = "empty"
		result.Error = "model endpoint returned no models"
		return result
	}
	result.Status = "ok"
	if err := s.store.ReplaceProviderKeyModels(ctx, key.ProviderID, key.ID, models); err != nil {
		result.Status = "storage_error"
		result.Error = "connected successfully but failed to cache discovered models"
		return result
	}
	providerModelCount, err := s.store.CountProviderModels(ctx, key.ProviderID)
	if err != nil {
		result.Status = "storage_error"
		result.Error = "connected successfully but failed to count discovered models"
		return result
	}
	if err := s.store.RecordProviderModelDiscoverySuccess(ctx, key.ProviderID, providerModelCount); err != nil {
		result.Status = "storage_error"
		result.Error = "connected successfully but failed to save model discovery state"
		return result
	}
	result.Model = selectProbeModel(key, models)
	return result
}

func fetchModelPage(ctx context.Context, client *http.Client, key model.Key, endpoint string, timeoutSeconds int) (int, []byte, int64, error) {
	var totalLatency int64
	for attempt := 0; attempt < 2; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
		if err != nil {
			cancel()
			return 0, nil, totalLatency, err
		}
		req.Header.Set("Accept", "application/json")
		setUpstreamAuthHeaders(req, key)
		req.Header.Set("User-Agent", "Local-AI-Gateway/1.0")
		start := time.Now()
		resp, err := client.Do(req)
		totalLatency += time.Since(start).Milliseconds()
		if err != nil {
			cancel()
			if attempt == 1 || !retryableKeyTestError(err) {
				return 0, nil, totalLatency, err
			}
			if err := waitForRetry(ctx, 300*time.Millisecond); err != nil {
				return 0, nil, totalLatency, err
			}
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
		_ = resp.Body.Close()
		cancel()
		if readErr != nil {
			return resp.StatusCode, nil, totalLatency, readErr
		}
		if len(body) > 1<<20 {
			return resp.StatusCode, nil, totalLatency, fmt.Errorf("model response exceeds 1 MiB limit")
		}
		return resp.StatusCode, body, totalLatency, nil
	}
	return 0, nil, totalLatency, fmt.Errorf("model request failed")
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
	result.TokenProtocol = tokenProbeProtocol(key.ProviderType)
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
	result.CacheCreationInputTokens = usage.CacheCreationInputTokens
	result.CacheReadInputTokens = usage.CacheReadInputTokens
	result.ReasoningTokens = usage.ReasoningTokens
	result.ToolUseTokens = usage.ToolUseTokens
	result.CachedContentTokens = usage.CachedContentTokens
	result.ThoughtsTokens = usage.ThoughtsTokens
	result.TokenStatus = "ok"
}

func providerCapabilities(providerType string) []string {
	switch providerType {
	case model.ProviderNewAPI, model.ProviderSub2API:
		return []string{"openai-chat", "openai-responses", "anthropic-messages", "gemini-generate-content"}
	case model.ProviderAnthropicCompatible:
		return []string{"anthropic-messages"}
	case model.ProviderGeminiCompatible:
		return []string{"gemini-generate-content"}
	case model.ProviderOpenAICompatible:
		return []string{"openai-chat", "openai-responses"}
	default:
		return []string{"openai-chat"}
	}
}

func tokenProbeProtocol(providerType string) string {
	switch providerType {
	case model.ProviderAnthropicCompatible:
		return "anthropic-messages"
	case model.ProviderGeminiCompatible:
		return "gemini-generate-content"
	default:
		return "openai-chat"
	}
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
		if strings.HasSuffix(base, "/v1beta") || strings.HasSuffix(base, "/v1") {
			return []string{"/models"}
		}
		return []string{"/v1beta/models"}
	case model.ProviderAnthropicCompatible:
		return []string{"/v1/models", "/models"}
	default:
		return []string{"/models", "/v1/models"}
	}
}

func tokenProbeRequest(key model.Key, modelID string) (string, []byte, error) {
	modelID = model.NormalizeModelID(key.ProviderType, modelID)
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

func parseModelPage(body []byte, providerType string) ([]string, *int, *modelPageCursor, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, nil, nil, nil
	}
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, nil, nil, err
	}
	parsed := collectModels(raw, false)
	cursor, err := nextModelPageCursor(raw, providerType)
	if err != nil {
		return nil, nil, nil, err
	}
	return parsed.IDs, parsed.Count, cursor, nil
}

func nextModelPageCursor(raw any, providerType string) (*modelPageCursor, error) {
	root, ok := raw.(map[string]any)
	if !ok {
		return nil, nil
	}
	containers := []map[string]any{root}
	for _, name := range []string{"meta", "pagination"} {
		if nested, ok := root[name].(map[string]any); ok {
			containers = append(containers, nested)
		}
	}
	for _, container := range containers {
		for _, candidate := range []struct {
			field string
			query string
		}{
			{field: "nextPageToken", query: "pageToken"},
			{field: "next_page_token", query: "page_token"},
			{field: "nextCursor", query: "cursor"},
			{field: "next_cursor", query: "cursor"},
		} {
			if value, ok := container[candidate.field].(string); ok && strings.TrimSpace(value) != "" {
				return &modelPageCursor{QueryParam: candidate.query, Value: strings.TrimSpace(value)}, nil
			}
		}
	}
	hasMore, _ := root["has_more"].(bool)
	if !hasMore {
		return nil, nil
	}
	lastID, _ := root["last_id"].(string)
	lastID = strings.TrimSpace(lastID)
	if lastID == "" {
		return nil, fmt.Errorf("model endpoint set has_more without last_id")
	}
	query := "after_id"
	if providerType == model.ProviderGeminiCompatible {
		query = "pageToken"
	}
	return &modelPageCursor{QueryParam: query, Value: lastID}, nil
}

func withModelPageCursor(endpoint string, cursor modelPageCursor) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set(cursor.QueryParam, cursor.Value)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
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
		if id = model.NormalizeModelID(key.ProviderType, id); id != "" {
			return id
		}
	}
	for _, mapped := range key.ProviderModelMap {
		if mapped = model.NormalizeModelID(key.ProviderType, mapped); mapped != "" {
			return mapped
		}
	}
	for public := range key.ProviderModelMap {
		if public = model.NormalizeModelID(key.ProviderType, public); public != "" {
			return public
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
