package readiness

import (
	"net/http"
	"strings"

	"local-ai-gateway/internal/model"
)

// Result describes whether the process can serve useful, authenticated gateway traffic.
// Process liveness is intentionally separate: a running process can still need setup.
type Result struct {
	State  string
	Reason string
	Ready  bool
}

func Evaluate(providers []model.Provider, keys []model.Key, gatewayKeys []model.GatewayKey, stats model.Stats, proxyToken string) Result {
	enabledProviders := 0
	for _, provider := range providers {
		if provider.Enabled {
			enabledProviders++
		}
	}
	if enabledProviders == 0 {
		return Result{State: "unconfigured", Reason: "no_providers"}
	}
	if len(keys) == 0 {
		return Result{State: "unconfigured", Reason: "no_upstream_keys"}
	}
	if stats.ActiveKeys == 0 {
		return Result{State: "unavailable", Reason: "no_active_keys"}
	}
	if !GatewayCredentialConfigured(gatewayKeys, proxyToken) {
		return Result{State: "unconfigured", Reason: "no_gateway_key"}
	}
	if stats.FailedKeys > 0 {
		return Result{State: "degraded", Reason: "upstream_failures", Ready: true}
	}
	return Result{State: "ready", Reason: "ready", Ready: true}
}

func GatewayCredentialConfigured(gatewayKeys []model.GatewayKey, proxyToken string) bool {
	if strings.TrimSpace(proxyToken) != "" {
		return true
	}
	for _, key := range gatewayKeys {
		if key.Enabled {
			return true
		}
	}
	return false
}

func HTTPStatus(result Result) int {
	if result.Ready {
		return http.StatusOK
	}
	return http.StatusServiceUnavailable
}
