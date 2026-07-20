package router

import (
	"context"
	"errors"
	"net/http"
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

	cacheMu           sync.Mutex
	cachedKeys        []model.Key
	cachedRoutes      map[string]model.ModelRoute
	cachedModelStates map[string]model.ProviderModelState
	cacheUntil        time.Time

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
	return &Router{
		store:             st,
		cfg:               cfg,
		probes:            make(map[string]struct{}),
		cachedRoutes:      make(map[string]model.ModelRoute),
		cachedModelStates: make(map[string]model.ProviderModelState),
	}
}

// AcquireRecoveryProbe ensures only one request tests a key whose cooldown has expired.
func (r *Router) AcquireRecoveryProbe(key model.Key) (func(), bool) {
	now := time.Now()
	probeIDs := make([]string, 0, 2)
	if key.CooldownUntil != nil && !key.CooldownUntil.After(now) {
		probeIDs = append(probeIDs, "key:"+key.ID)
	}
	if key.ModelCooldownUntil != nil && !key.ModelCooldownUntil.After(now) {
		probeIDs = append(probeIDs, "model:"+modelStateKey(key.ProviderID, key.ID, key.UpstreamModel))
	}
	if len(probeIDs) == 0 {
		return func() {}, true
	}
	r.probeMu.Lock()
	defer r.probeMu.Unlock()
	for _, probeID := range probeIDs {
		if _, exists := r.probes[probeID]; exists {
			return func() {}, false
		}
	}
	for _, probeID := range probeIDs {
		r.probes[probeID] = struct{}{}
	}
	return func() {
		r.probeMu.Lock()
		for _, probeID := range probeIDs {
			delete(r.probes, probeID)
		}
		r.probeMu.Unlock()
	}, true
}

func (r *Router) Candidates(ctx context.Context, modelName string, inboundProtocol string) ([]model.Key, error) {
	keys, routes, modelStates, err := r.routingData(ctx)
	if err != nil {
		return nil, err
	}
	if route, ok := routes[modelName]; ok && route.Enabled {
		return routeCandidates(keys, route, modelStates, inboundProtocol), nil
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
		if !ProviderAllowsModel(k.ProviderType, k.ProviderModelAllowlistEnabled, k.ProviderModelAllowlist, k.UpstreamModel) {
			continue
		}
		if state, ok := modelStates[modelStateKey(k.ProviderID, k.ID, k.UpstreamModel)]; ok {
			if state.CooldownUntil != nil && state.CooldownUntil.After(now) {
				continue
			}
			k.ModelCooldownUntil = state.CooldownUntil
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

func routeCandidates(keys []model.Key, route model.ModelRoute, modelStates map[string]model.ProviderModelState, inboundProtocol string) []model.Key {
	now := time.Now()
	out := make([]model.Key, 0)
	for _, routeModel := range route.Models {
		if !routeModel.Enabled {
			continue
		}
		for _, target := range routeModel.Targets {
			if !target.Enabled {
				continue
			}
			for _, key := range keys {
				if key.ProviderID != target.ProviderID || !key.Enabled || !key.ProviderEnabled {
					continue
				}
				if key.CooldownUntil != nil && key.CooldownUntil.After(now) {
					continue
				}
				if !ProviderAllowsModel(key.ProviderType, key.ProviderModelAllowlistEnabled, key.ProviderModelAllowlist, target.UpstreamModel) {
					continue
				}
				state, hasState := modelStates[modelStateKey(target.ProviderID, key.ID, target.UpstreamModel)]
				if hasState && state.CooldownUntil != nil && state.CooldownUntil.After(now) {
					continue
				}
				upstreamProtocol := ChooseUpstreamProtocol(inboundProtocol, key)
				if inboundProtocol == ProtocolOpenAIResponses && upstreamProtocol != ProtocolOpenAIResponses {
					continue
				}
				if !supportsProtocol(key.ProviderType, upstreamProtocol) {
					continue
				}
				key.UpstreamModel = target.UpstreamModel
				key.RouteID = route.ID
				key.RouteModel = routeModel.Name
				key.ModelPriority = routeModel.Priority
				if hasState {
					key.ModelCooldownUntil = state.CooldownUntil
				}
				out = append(out, key)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.ModelPriority != b.ModelPriority {
			return a.ModelPriority > b.ModelPriority
		}
		if a.RouteModel != b.RouteModel {
			return a.RouteModel < b.RouteModel
		}
		if a.ProviderPriority != b.ProviderPriority {
			return a.ProviderPriority > b.ProviderPriority
		}
		if a.ManualPreferred != b.ManualPreferred {
			return a.ManualPreferred
		}
		if a.Priority != b.Priority {
			return a.Priority > b.Priority
		}
		if a.ProviderID != b.ProviderID {
			return a.ProviderID < b.ProviderID
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		return a.UpstreamModel < b.UpstreamModel
	})
	return out
}

func ProviderAllowsModel(providerType string, allowlistEnabled bool, allowlist []string, modelID string) bool {
	if !allowlistEnabled {
		return true
	}
	modelID = model.NormalizeModelID(providerType, modelID)
	for _, allowed := range allowlist {
		if model.NormalizeModelID(providerType, allowed) == modelID {
			return true
		}
	}
	return false
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

func (r *Router) RecordCandidateSuccess(ctx context.Context, key model.Key) error {
	keyErr := r.RecordSuccess(ctx, key)
	var modelErr error
	if key.ProviderID != "" && key.UpstreamModel != "" {
		modelErr = r.store.RecordProviderModelSuccess(ctx, key.ProviderID, key.ID, key.UpstreamModel)
	}
	if key.ModelCooldownUntil != nil {
		r.invalidateCache()
	}
	return errors.Join(keyErr, modelErr)
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

func (r *Router) RecordCandidateFailure(ctx context.Context, key model.Key, f Failure) error {
	if !CountsAgainstModelHealth(f.ErrorType) {
		return r.RecordFailure(ctx, key, f)
	}
	status := f.Status
	var statusPtr *int
	if status > 0 {
		statusPtr = &status
	}
	policy := store.FailurePolicy{
		Threshold:     1,
		Cooldown:      time.Duration(r.cfg.CooldownSeconds) * time.Second,
		ForceCooldown: true,
	}
	if f.RetryAfterSeconds > 0 {
		policy.OverrideCooldown = time.Duration(f.RetryAfterSeconds) * time.Second
	}
	if err := r.store.RecordProviderModelFailure(ctx, key.ProviderID, key.ID, key.UpstreamModel, statusPtr, f.Message, policy); err != nil {
		return err
	}
	r.invalidateCache()
	return nil
}

func (r *Router) routingData(ctx context.Context) ([]model.Key, map[string]model.ModelRoute, map[string]model.ProviderModelState, error) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	if time.Now().Before(r.cacheUntil) {
		return append([]model.Key(nil), r.cachedKeys...), r.cachedRoutes, r.cachedModelStates, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	keys, err := r.store.ListKeys(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	routes, err := r.store.ListModelRoutes(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	states, err := r.store.ListProviderModelStates(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	r.cachedKeys = append(r.cachedKeys[:0], keys...)
	r.cachedRoutes = make(map[string]model.ModelRoute, len(routes))
	for _, route := range routes {
		r.cachedRoutes[route.ID] = route
	}
	r.cachedModelStates = make(map[string]model.ProviderModelState, len(states))
	for _, state := range states {
		r.cachedModelStates[modelStateKey(state.ProviderID, state.KeyID, state.ModelID)] = state
	}
	r.cacheUntil = time.Now().Add(routingCacheTTL)
	return append([]model.Key(nil), keys...), r.cachedRoutes, r.cachedModelStates, nil
}

func modelStateKey(providerID, keyID, modelID string) string {
	return providerID + "\x00" + keyID + "\x00" + modelID
}

func (r *Router) invalidateCache() {
	r.cacheMu.Lock()
	r.cacheUntil = time.Time{}
	r.cacheMu.Unlock()
}

func Classify(status int, message string) string {
	text := strings.ToLower(message)
	if isModelUnavailable(status, text) {
		return "model_unavailable"
	}
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

// recoverableUpstreamError reports whether an upstream error type reflects a genuine upstream
// problem (as opposed to a client cancel or protocol mismatch). Both retry eligibility and key
// health accounting currently key off the same set; they are kept as distinct exported functions
// so their policies can diverge later without touching call sites.
func recoverableUpstreamError(errorType string) bool {
	switch errorType {
	case "auth_error", "rate_limit", "server_error", "timeout", "empty_response", "upstream_error":
		return true
	default:
		return false
	}
}

func Retryable(errorType string) bool {
	return recoverableUpstreamError(errorType) || errorType == "model_unavailable"
}

func CountsAgainstKeyHealth(errorType string) bool {
	return recoverableUpstreamError(errorType)
}

func CountsAgainstModelHealth(errorType string) bool {
	return errorType == "model_unavailable"
}

// ProviderModelUnavailable distinguishes a model that the upstream does not implement from a
// key-specific entitlement error. Only definitive absence/unsupported signals are provider-wide.
func ProviderModelUnavailable(status int, message string) bool {
	text := strings.ToLower(message)
	if !isModelUnavailable(status, text) {
		return false
	}
	normalized := strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(text)
	for _, marker := range []string{
		"model not found", "model does not exist", "unknown model", "unsupported model", "invalid model",
		"模型不存在", "未找到模型", "找不到模型", "不支持该模型", "模型不支持",
	} {
		if strings.Contains(normalized, marker) || strings.Contains(text, marker) {
			return true
		}
	}
	return status == http.StatusNotFound && (strings.Contains(normalized, "model") || strings.Contains(text, "模型"))
}

func isModelUnavailable(status int, text string) bool {
	if status != 400 && status != 403 && status != 404 && status != 422 {
		return false
	}
	normalized := strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(text)
	if !strings.Contains(normalized, "model") && !strings.Contains(text, "模型") {
		return false
	}
	for _, marker := range []string{
		"not found", "does not exist", "unavailable", "not available", "unsupported",
		"invalid model", "no access", "permission", "not enabled", "disabled",
		"模型不存在", "未找到模型", "找不到模型", "模型不可用", "无可用模型",
		"不支持该模型", "模型不支持", "模型未启用", "模型已禁用", "模型无权限",
		"没有模型权限", "无权访问模型", "无可用渠道",
	} {
		if strings.Contains(normalized, marker) || strings.Contains(text, marker) {
			return true
		}
	}
	return false
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
