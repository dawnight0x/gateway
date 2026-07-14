package router

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
)

const (
	ProtocolOpenAI          = "openai"
	ProtocolOpenAIResponses = "openai-responses"
	ProtocolAnthropic       = "anthropic"
	ProtocolGemini          = "gemini"
)

type Router struct {
	store *store.Store
	cfg   config.RoutingConfig

	cacheMu    sync.Mutex
	cachedKeys []model.Key
	cacheUntil time.Time

	probeMu sync.Mutex
	probes  map[string]struct{}
}

const routingCacheTTL = 250 * time.Millisecond

type Failure struct {
	Status            int
	ErrorType         string
	Message           string
	RetryAfterSeconds int
}

func New(st *store.Store, cfg config.RoutingConfig) *Router {
	return &Router{store: st, cfg: cfg, probes: make(map[string]struct{})}
}

// AcquireRecoveryProbe ensures only one request tests a key whose cooldown has expired.
func (r *Router) AcquireRecoveryProbe(key model.Key) (func(), bool) {
	if key.CooldownUntil == nil || key.CooldownUntil.After(time.Now()) {
		return func() {}, true
	}
	r.probeMu.Lock()
	defer r.probeMu.Unlock()
	if _, exists := r.probes[key.ID]; exists {
		return func() {}, false
	}
	r.probes[key.ID] = struct{}{}
	return func() {
		r.probeMu.Lock()
		delete(r.probes, key.ID)
		r.probeMu.Unlock()
	}, true
}

func (r *Router) Candidates(ctx context.Context, modelName string, inboundProtocol string) ([]model.Key, error) {
	keys, err := r.routingKeys(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var out []model.Key
	for _, k := range keys {
		if !k.Enabled || !k.ProviderEnabled {
			continue
		}
		if k.CooldownUntil != nil && k.CooldownUntil.After(now) {
			continue
		}
		upstreamProtocol := ChooseUpstreamProtocol(inboundProtocol, k)
		if inboundProtocol == ProtocolOpenAIResponses && upstreamProtocol != ProtocolOpenAIResponses {
			continue
		}
		if !supportsProtocol(k.ProviderType, upstreamProtocol) {
			continue
		}
		k.UpstreamModel = mapModel(k.ProviderModelMap, modelName)
		if k.UpstreamModel == "" {
			continue
		}
		out = append(out, k)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.ManualPreferred != b.ManualPreferred {
			return a.ManualPreferred
		}
		if a.ProviderPriority != b.ProviderPriority {
			return a.ProviderPriority > b.ProviderPriority
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		if a.ProviderID != b.ProviderID {
			return a.ProviderID < b.ProviderID
		}
		return a.ID < b.ID
	})
	return out, nil
}

func (r *Router) RecordSuccess(ctx context.Context, key model.Key) error {
	resetHealth := key.ConsecutiveFailures > 0 || key.CooldownUntil != nil || key.LastError != ""
	if err := r.store.RecordSuccess(ctx, key.ID, resetHealth); err != nil {
		return err
	}
	if resetHealth {
		r.invalidateCache()
	}
	return nil
}

func (r *Router) RecordFailure(ctx context.Context, key model.Key, f Failure) error {
	status := f.Status
	var statusPtr *int
	if status > 0 {
		statusPtr = &status
	}
	policy := store.FailurePolicy{
		Threshold:         r.cfg.FailureThreshold,
		Cooldown:          time.Duration(r.cfg.CooldownSeconds) * time.Second,
		ThresholdCooldown: time.Minute,
		ForceCooldown:     f.ErrorType == "auth_error" || f.ErrorType == "rate_limit",
	}
	if f.RetryAfterSeconds > 0 {
		policy.OverrideCooldown = time.Duration(f.RetryAfterSeconds) * time.Second
	} else if f.ErrorType == "auth_error" {
		policy.OverrideCooldown = time.Duration(r.cfg.AuthCooldownSeconds) * time.Second
	}
	if err := r.store.RecordFailure(ctx, key.ID, statusPtr, f.Message, policy); err != nil {
		return err
	}
	r.invalidateCache()
	return nil
}

func (r *Router) routingKeys(ctx context.Context) ([]model.Key, error) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	if time.Now().Before(r.cacheUntil) {
		return append([]model.Key(nil), r.cachedKeys...), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keys, err := r.store.ListKeys(ctx)
	if err != nil {
		return nil, err
	}
	r.cachedKeys = append(r.cachedKeys[:0], keys...)
	r.cacheUntil = time.Now().Add(routingCacheTTL)
	return append([]model.Key(nil), keys...), nil
}

func (r *Router) invalidateCache() {
	r.cacheMu.Lock()
	r.cacheUntil = time.Time{}
	r.cacheMu.Unlock()
}

func Classify(status int, message string) string {
	text := strings.ToLower(message)
	if status == 401 || status == 403 {
		return "auth_error"
	}
	if status == 429 || strings.Contains(text, "rate limit") || strings.Contains(text, "quota") {
		return "rate_limit"
	}
	if status == 500 || status == 502 || status == 503 || status == 504 {
		return "server_error"
	}
	if status >= 400 && status < 500 {
		return "client_error"
	}
	if strings.Contains(text, "context canceled") {
		return "client_canceled"
	}
	if strings.Contains(text, "timeout") || strings.Contains(text, "deadline") {
		return "timeout"
	}
	if strings.Contains(text, "empty") {
		return "empty_response"
	}
	return "upstream_error"
}

func Retryable(errorType string) bool {
	switch errorType {
	case "auth_error", "rate_limit", "server_error", "timeout", "empty_response", "upstream_error":
		return true
	default:
		return false
	}
}

func CountsAgainstKeyHealth(errorType string) bool {
	switch errorType {
	case "auth_error", "rate_limit", "server_error", "timeout", "empty_response", "upstream_error":
		return true
	default:
		return false
	}
}

func ChooseUpstreamProtocol(inboundProtocol string, key model.Key) string {
	switch key.ProviderType {
	case model.ProviderGeminiCompatible:
		return ProtocolGemini
	case model.ProviderAnthropicCompatible:
		return ProtocolAnthropic
	case model.ProviderOpenAICompatible, model.ProviderNewAPI:
		if inboundProtocol == ProtocolOpenAIResponses {
			return ProtocolOpenAIResponses
		}
		return ProtocolOpenAI
	case model.ProviderSub2API:
		if inboundProtocol == ProtocolAnthropic {
			return ProtocolAnthropic
		}
		if inboundProtocol == ProtocolGemini {
			return ProtocolGemini
		}
		if inboundProtocol == ProtocolOpenAIResponses {
			return ProtocolOpenAIResponses
		}
		return ProtocolOpenAI
	default:
		if inboundProtocol == ProtocolAnthropic || inboundProtocol == ProtocolGemini || inboundProtocol == ProtocolOpenAIResponses {
			return inboundProtocol
		}
		return ProtocolOpenAI
	}
}

func supportsProtocol(providerType, upstreamProtocol string) bool {
	switch upstreamProtocol {
	case ProtocolOpenAI:
		return providerType == model.ProviderOpenAICompatible || providerType == model.ProviderNewAPI || providerType == model.ProviderSub2API || providerType == model.ProviderCustom
	case ProtocolOpenAIResponses:
		return providerType == model.ProviderOpenAICompatible || providerType == model.ProviderNewAPI || providerType == model.ProviderSub2API || providerType == model.ProviderCustom
	case ProtocolAnthropic:
		return providerType == model.ProviderAnthropicCompatible || providerType == model.ProviderSub2API || providerType == model.ProviderCustom
	case ProtocolGemini:
		return providerType == model.ProviderGeminiCompatible || providerType == model.ProviderSub2API || providerType == model.ProviderCustom
	default:
		return false
	}
}

func mapModel(modelMap map[string]string, modelName string) string {
	if modelName == "" {
		return ""
	}
	if len(modelMap) == 0 {
		return modelName
	}
	if v, ok := modelMap[modelName]; ok {
		return v
	}
	if v, ok := modelMap["*"]; ok {
		return v
	}
	if v, ok := modelMap["default"]; ok {
		return v
	}
	return modelName
}
