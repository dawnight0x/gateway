package model

import "time"

const (
	ProviderOpenAICompatible    = "openai-compatible"
	ProviderAnthropicCompatible = "anthropic-compatible"
	ProviderGeminiCompatible    = "gemini-compatible"
	ProviderNewAPI              = "new-api"
	ProviderSub2API             = "sub2api"
	ProviderCustom              = "custom"
)

type Provider struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	BaseURL     string            `json:"baseUrl"`
	Priority    int               `json:"priority"`
	Enabled     bool              `json:"enabled"`
	ModelMap    map[string]string `json:"modelMap"`
	BalancePath string            `json:"balancePath"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type Key struct {
	ID                  string            `json:"id"`
	ProviderID          string            `json:"providerId"`
	ProviderName        string            `json:"providerName"`
	ProviderType        string            `json:"providerType"`
	ProviderBaseURL     string            `json:"providerBaseUrl"`
	ProviderPriority    int               `json:"providerPriority"`
	ProviderEnabled     bool              `json:"providerEnabled"`
	ProviderModelMap    map[string]string `json:"providerModelMap"`
	ProviderBalancePath string            `json:"providerBalancePath"`
	Name                string            `json:"name"`
	Secret              string            `json:"-"`
	KeyHint             string            `json:"keyHint"`
	Priority            int               `json:"priority"`
	Enabled             bool              `json:"enabled"`
	ManualPreferred     bool              `json:"manualPreferred"`
	ConsecutiveFailures int               `json:"consecutiveFailures"`
	CooldownUntil       *time.Time        `json:"cooldownUntil,omitempty"`
	LastError           string            `json:"lastError"`
	LastStatusCode      *int              `json:"lastStatusCode,omitempty"`
	SuccessCount        int               `json:"successCount"`
	FailureCount        int               `json:"failureCount"`
	LastUsedAt          *time.Time        `json:"lastUsedAt,omitempty"`
	UpstreamModel       string            `json:"upstreamModel,omitempty"`
	RouteID             string            `json:"routeId,omitempty"`
	RouteModel          string            `json:"routeModel,omitempty"`
	ModelPriority       int               `json:"modelPriority,omitempty"`
	ModelCooldownUntil  *time.Time        `json:"modelCooldownUntil,omitempty"`
}

type ProviderModelDiscovery struct {
	ProviderID    string     `json:"providerId"`
	Status        string     `json:"status"`
	ModelCount    int        `json:"modelCount"`
	LastAttemptAt *time.Time `json:"lastAttemptAt,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
	LastError     string     `json:"lastError,omitempty"`
}

type ProviderModelState struct {
	ProviderID          string     `json:"providerId"`
	ModelID             string     `json:"modelId"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	CooldownUntil       *time.Time `json:"cooldownUntil,omitempty"`
	LastError           string     `json:"lastError,omitempty"`
	LastStatusCode      *int       `json:"lastStatusCode,omitempty"`
	SuccessCount        int        `json:"successCount"`
	FailureCount        int        `json:"failureCount"`
	LastUsedAt          *time.Time `json:"lastUsedAt,omitempty"`
}

type ModelRouteTarget struct {
	ProviderID    string `json:"providerId"`
	UpstreamModel string `json:"upstreamModel"`
	Enabled       bool   `json:"enabled"`
}

type ModelRouteModel struct {
	Name     string             `json:"name"`
	Priority int                `json:"priority"`
	Enabled  bool               `json:"enabled"`
	Targets  []ModelRouteTarget `json:"targets"`
}

type ModelRoute struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Enabled   bool              `json:"enabled"`
	Models    []ModelRouteModel `json:"models"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type GatewayKey struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	KeyHint      string     `json:"keyHint"`
	Enabled      bool       `json:"enabled"`
	RequestCount int        `json:"requestCount"`
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	Plaintext    string     `json:"plaintext,omitempty"`
}

type RequestLog struct {
	ID               int64     `json:"id"`
	RequestID        string    `json:"requestId"`
	InboundProtocol  string    `json:"inboundProtocol"`
	ProviderID       string    `json:"providerId,omitempty"`
	KeyID            string    `json:"keyId,omitempty"`
	Model            string    `json:"model,omitempty"`
	RouteID          string    `json:"routeId,omitempty"`
	UpstreamModel    string    `json:"upstreamModel,omitempty"`
	Attempts         int       `json:"attempts,omitempty"`
	Status           int       `json:"status"`
	LatencyMS        int64     `json:"latencyMs"`
	PromptTokens     *int      `json:"promptTokens,omitempty"`
	CompletionTokens *int      `json:"completionTokens,omitempty"`
	TotalTokens      *int      `json:"totalTokens,omitempty"`
	ErrorType        string    `json:"errorType,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Balance struct {
	ID          int64      `json:"id"`
	ProviderID  string     `json:"providerId"`
	KeyID       string     `json:"keyId,omitempty"`
	Balance     *float64   `json:"balance,omitempty"`
	Currency    string     `json:"currency,omitempty"`
	QuotaUsed   *float64   `json:"quotaUsed,omitempty"`
	QuotaLimit  *float64   `json:"quotaLimit,omitempty"`
	Source      string     `json:"source"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	RefreshedAt *time.Time `json:"refreshedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Stats struct {
	TotalKeys          int          `json:"totalKeys"`
	ActiveKeys         int          `json:"activeKeys"`
	FailedKeys         int          `json:"failedKeys"`
	TodayRequests      int          `json:"todayRequests"`
	TodayTokens        int          `json:"todayTokens"`
	DroppedRequestLogs uint64       `json:"droppedRequestLogs"`
	Recent             []RequestLog `json:"recent"`
}
