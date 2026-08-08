package readiness

import (
	"testing"

	"local-ai-gateway/internal/model"
)

func TestEvaluate(t *testing.T) {
	provider := model.Provider{Enabled: true}
	key := model.Key{Enabled: true}
	gatewayKey := model.GatewayKey{Enabled: true}
	tests := []struct {
		name        string
		providers   []model.Provider
		keys        []model.Key
		gatewayKeys []model.GatewayKey
		stats       model.Stats
		wantState   string
		wantReason  string
		wantReady   bool
	}{
		{name: "no providers", wantState: "unconfigured", wantReason: "no_providers"},
		{name: "no upstream key", providers: []model.Provider{provider}, wantState: "unconfigured", wantReason: "no_upstream_keys"},
		{name: "no active key", providers: []model.Provider{provider}, keys: []model.Key{key}, wantState: "unavailable", wantReason: "no_active_keys"},
		{name: "no gateway key", providers: []model.Provider{provider}, keys: []model.Key{key}, stats: model.Stats{ActiveKeys: 1}, wantState: "unconfigured", wantReason: "no_gateway_key"},
		{name: "degraded", providers: []model.Provider{provider}, keys: []model.Key{key}, gatewayKeys: []model.GatewayKey{gatewayKey}, stats: model.Stats{ActiveKeys: 1, FailedKeys: 1}, wantState: "degraded", wantReason: "upstream_failures", wantReady: true},
		{name: "ready", providers: []model.Provider{provider}, keys: []model.Key{key}, gatewayKeys: []model.GatewayKey{gatewayKey}, stats: model.Stats{ActiveKeys: 1}, wantState: "ready", wantReason: "ready", wantReady: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.providers, tt.keys, tt.gatewayKeys, tt.stats, "")
			if got.State != tt.wantState || got.Reason != tt.wantReason || got.Ready != tt.wantReady {
				t.Fatalf("result = %#v, want state=%s reason=%s ready=%v", got, tt.wantState, tt.wantReason, tt.wantReady)
			}
		})
	}
}
