package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"local-ai-gateway/internal/model"
)

type balanceRefreshResult struct {
	ProviderID string `json:"providerId"`
	KeyID      string `json:"keyId,omitempty"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

func (s *Service) balance(w http.ResponseWriter, r *http.Request, parts []string) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListBalances(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		if len(parts) < 2 || parts[1] != "refresh" {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
			return
		}
		results := s.refreshBalances(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "results": results})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Service) refreshBalances(ctx context.Context) []balanceRefreshResult {
	keys, err := s.store.ListKeys(ctx)
	if err != nil {
		return []balanceRefreshResult{{Status: "error", Error: err.Error()}}
	}
	type refreshTask struct {
		key   model.Key
		paths []string
	}
	var tasks []refreshTask
	for _, key := range keys {
		if !key.ProviderEnabled || !key.Enabled || strings.TrimSpace(key.ProviderBaseURL) == "" {
			continue
		}
		paths := balancePathsForKey(key)
		if len(paths) == 0 {
			continue
		}
		tasks = append(tasks, refreshTask{key: key, paths: paths})
	}
	if len(tasks) == 0 {
		return []balanceRefreshResult{{Status: "skipped", Error: "no enabled upstream key has balance adapter configured"}}
	}

	results := make([]balanceRefreshResult, len(tasks))
	workers := min(4, len(tasks))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				task := tasks[index]
				results[index] = s.refreshBalanceForKey(ctx, task.key, task.paths)
			}
		}()
	}
	for index := range tasks {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
	return results
}

func balancePathsForKey(key model.Key) []string {
	if custom := strings.TrimSpace(key.ProviderBalancePath); custom != "" {
		return []string{custom}
	}
	switch key.ProviderType {
	case model.ProviderNewAPI:
		return []string{"/api/user/self", "/api/usage/token/", "/api/user/token", "/dashboard/billing/subscription", "/v1/dashboard/billing/subscription"}
	case model.ProviderSub2API:
		return []string{"/v1/usage", "/api/v1/user/profile", "/api/v1/auth/me"}
	default:
		return nil
	}
}

func (s *Service) refreshBalanceForKey(ctx context.Context, key model.Key, paths []string) balanceRefreshResult {
	var last balanceRefreshResult
	for _, path := range paths {
		result, ok := s.tryRefreshBalancePath(ctx, key, path)
		if ok {
			return result
		}
		last = result
	}
	if last.Status == "" {
		last = balanceRefreshResult{ProviderID: key.ProviderID, KeyID: key.ID, Status: "skipped", Error: "no balance path configured"}
	}
	return last
}

func (s *Service) tryRefreshBalancePath(ctx context.Context, key model.Key, path string) (balanceRefreshResult, bool) {
	source := balanceSource(key.ProviderType, path)
	endpoint, err := joinBalanceURL(key.ProviderBaseURL, path)
	if err != nil {
		msg := sanitizeBalanceErrorForKey(err.Error(), key.Secret)
		return s.persistBalanceFailure(ctx, key, source, "config_error", msg)
	}
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.Routing.TimeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		msg := sanitizeBalanceErrorForKey(err.Error(), key.Secret)
		return s.persistBalanceFailure(ctx, key, source, "config_error", msg)
	}
	req.Header.Set("Accept", "application/json")
	setUpstreamAuthHeaders(req, key)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		msg := sanitizeBalanceErrorForKey(err.Error(), key.Secret)
		return s.persistBalanceFailure(ctx, key, source, "network_error", msg)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return s.persistBalanceFailure(ctx, key, source, "network_error", sanitizeBalanceErrorForKey(err.Error(), key.Secret))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := sanitizeBalanceErrorForKey(fmt.Sprintf("status %d: %s", resp.StatusCode, string(body)), key.Secret)
		status := classifyBalanceError(resp.StatusCode, string(body))
		if status == "not_found" && key.ProviderBalancePath == "" {
			return balanceRefreshResult{ProviderID: key.ProviderID, KeyID: key.ID, Status: status, Error: msg}, false
		}
		return s.persistBalanceFailure(ctx, key, source, status, msg)
	}
	balance, err := parseBalanceResponse(body)
	if err != nil {
		msg := sanitizeBalanceErrorForKey(err.Error(), key.Secret)
		return s.persistBalanceFailure(ctx, key, source, "parse_error", msg)
	}
	balance.ProviderID = key.ProviderID
	balance.KeyID = key.ID
	balance.Source = source
	if balance.Status == "" {
		balance.Status = "ok"
	}
	balance = normalizeBalanceForProvider(balance, key.ProviderType, path)
	if err := s.store.UpsertBalance(ctx, balance); err != nil {
		return balanceRefreshResult{ProviderID: key.ProviderID, KeyID: key.ID, Status: "store_error", Error: err.Error()}, false
	}
	return balanceRefreshResult{ProviderID: key.ProviderID, KeyID: key.ID, Status: balance.Status}, true
}

func (s *Service) persistBalanceFailure(ctx context.Context, key model.Key, source, status, message string) (balanceRefreshResult, bool) {
	result := balanceRefreshResult{ProviderID: key.ProviderID, KeyID: key.ID, Status: status, Error: message}
	if err := s.store.UpsertBalance(ctx, model.Balance{ProviderID: key.ProviderID, KeyID: key.ID, Source: source, Status: status, Error: message}); err != nil {
		result.Status = "store_error"
		result.Error = err.Error()
	}
	return result, false
}

func parseBalanceResponse(body []byte) (model.Balance, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return model.Balance{}, err
	}
	if nested, ok := raw["data"].(map[string]any); ok {
		raw = mergeMaps(raw, nested)
	}
	if user, ok := raw["user"].(map[string]any); ok {
		raw = mergeMaps(raw, user)
	}
	out := model.Balance{
		Balance:    firstNumber(raw, "balance", "remaining", "remain_quota", "quota_remaining", "credit", "credits", "quota", "total_available"),
		Currency:   firstString(raw, "currency", "currency_code", "unit"),
		QuotaUsed:  firstNumber(raw, "used", "quota_used", "used_quota", "total_used"),
		QuotaLimit: firstNumber(raw, "limit", "quota_limit", "total_quota", "hard_limit_usd", "total_granted"),
	}
	unlimited := firstBool(raw, "unlimited_quota", "unlimited", "is_unlimited")
	if quota, ok := raw["quota"].(map[string]any); ok {
		if out.Balance == nil {
			out.Balance = firstNumber(quota, "remaining", "remain_quota", "quota_remaining", "total_available")
		}
		if out.QuotaUsed == nil {
			out.QuotaUsed = firstNumber(quota, "used", "quota_used", "used_quota", "total_used")
		}
		if out.QuotaLimit == nil {
			out.QuotaLimit = firstNumber(quota, "limit", "quota", "quota_limit", "total_quota", "total_granted")
		}
		if out.Currency == "" {
			out.Currency = firstString(quota, "currency", "currency_code", "unit")
		}
	}
	if out.Balance == nil && out.QuotaLimit != nil && out.QuotaUsed != nil {
		remaining := *out.QuotaLimit - *out.QuotaUsed
		out.Balance = &remaining
	}
	if unlimited && out.Balance == nil {
		out.Status = "unlimited"
		out.QuotaUsed = nil
		out.QuotaLimit = nil
	}
	if out.Status == "" && out.Balance == nil && out.QuotaUsed == nil && out.QuotaLimit == nil {
		return model.Balance{}, fmt.Errorf("no balance or quota fields found")
	}
	return out, nil
}

func normalizeBalanceForProvider(balance model.Balance, providerType string, path string) model.Balance {
	if providerType != model.ProviderNewAPI {
		return balance
	}
	if isNewAPIUnlimitedBalance(balance) {
		balance.Status = "unlimited"
		balance.Balance = nil
		balance.QuotaUsed = nil
		balance.QuotaLimit = nil
		balance.Error = ""
	}
	return balance
}

const newAPIUnlimitedBalanceThreshold = 10000000.0

func isNewAPIUnlimitedBalance(balance model.Balance) bool {
	for _, value := range []*float64{balance.Balance, balance.QuotaLimit} {
		if value != nil && *value > newAPIUnlimitedBalanceThreshold {
			return true
		}
	}
	return false
}

func firstNumber(raw map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if n, ok := asFloat(v); ok {
				return &n
			}
		}
	}
	return nil
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := raw[key].(string); ok {
			return v
		}
	}
	return ""
}

func firstBool(raw map[string]any, keys ...string) bool {
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			switch b := v.(type) {
			case bool:
				return b
			case string:
				parsed, err := strconv.ParseBool(strings.TrimSpace(b))
				return err == nil && parsed
			}
		}
	}
	return false
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func mergeMaps(base, overlay map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

func joinURL(base, path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	u, err := url.Parse(strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/"))
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func joinBalanceURL(base, path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if !sameURLOrigin(base, path) {
			return "", fmt.Errorf("absolute balance URL must use the same origin as provider base URL")
		}
		return path, nil
	}
	return joinURL(balanceControlBaseURL(base), path)
}

func balanceControlBaseURL(base string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(base), "/")
	if trimmed == "" {
		return trimmed
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return trimmed
	}
	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return trimmed
	}
	last := strings.ToLower(segments[len(segments)-1])
	if last == "v1" || last == "v1beta" || last == "v1beta1" || last == "compatible-mode" {
		segments = segments[:len(segments)-1]
		u.Path = "/" + strings.Join(segments, "/")
		if len(segments) == 0 {
			u.Path = ""
		}
		return strings.TrimRight(u.String(), "/")
	}
	return trimmed
}

func sanitizeBalanceError(message string) string {
	message = strings.TrimSpace(message)
	if len(message) > 240 {
		message = message[:240]
	}
	return message
}

func sanitizeBalanceErrorForKey(message, secret string) string {
	return redactSecret(sanitizeBalanceError(message), secret)
}

func classifyBalanceError(status int, message string) string {
	text := strings.ToLower(message)
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return "auth_error"
	case status == http.StatusTooManyRequests || strings.Contains(text, "rate limit") || strings.Contains(text, "quota"):
		return "rate_limit"
	case status == http.StatusNotFound:
		return "not_found"
	case status >= 500:
		return "server_error"
	case status >= 400:
		return "client_error"
	default:
		return "upstream_error"
	}
}

func balanceSource(providerType, path string) string {
	if providerType == "" {
		providerType = "custom"
	}
	return providerType + ":" + path
}
