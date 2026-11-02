package admin

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
	"local-ai-gateway/internal/config"
)

func TestRoutingConcurrentPersistenceAndSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("GATEWAY_CONFIG", path)
	cfg := config.Default()
	cfg.Routing.TimeoutSeconds, cfg.Routing.StreamIdleTimeoutSeconds = 20, 40
	svc := New(testAdminStore(t), cfg)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for writer := range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := range 10 {
				routing := cfg.Routing
				routing.TimeoutSeconds = 60 + writer*10 + i
				routing.StreamIdleTimeoutSeconds = routing.TimeoutSeconds * 2
				body, _ := json.Marshal(routing)
				res := authorizedAdminRequest(t, svc, http.MethodPatch, "/admin/api/routing", string(body))
				if res.Code != http.StatusOK {
					t.Errorf("PATCH: %d %s", res.Code, res.Body.String())
					return
				}
			}
		}()
	}
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range 30 {
				for _, endpoint := range []string{"routing", "dashboard"} {
					res := authorizedAdminRequest(t, svc, http.MethodGet, "/admin/api/"+endpoint, "")
					if res.Code != http.StatusOK {
						t.Errorf("GET %s: %d", endpoint, res.Code)
						return
					}
					var routing config.RoutingConfig
					if endpoint == "routing" {
						_ = json.Unmarshal(res.Body.Bytes(), &routing)
					} else {
						var dashboard struct {
							Routing config.RoutingConfig `json:"routing"`
						}
						_ = json.Unmarshal(res.Body.Bytes(), &dashboard)
						routing = dashboard.Routing
					}
					if routing.StreamIdleTimeoutSeconds != routing.TimeoutSeconds*2 {
						t.Errorf("torn routing snapshot: %+v", routing)
						return
					}
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted config.Config
	if err := yaml.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Routing != svc.configSnapshot().Routing {
		t.Fatalf("disk routing and admin snapshot diverged: disk=%+v memory=%+v", persisted.Routing, svc.configSnapshot().Routing)
	}
}

func TestRoutingSaveFailureDoesNotPublishSnapshot(t *testing.T) {
	path := t.TempDir() // Cannot atomically replace a directory with YAML.
	t.Setenv("GATEWAY_CONFIG", path)
	svc := New(testAdminStore(t), config.Default())
	before := svc.configSnapshot().Routing
	updated := before
	updated.TimeoutSeconds++
	body, _ := json.Marshal(updated)
	res := authorizedAdminRequest(t, svc, http.MethodPatch, "/admin/api/routing", string(body))
	if res.Code != http.StatusInternalServerError || svc.configSnapshot().Routing != before {
		t.Fatalf("failed save published state: status=%d routing=%+v", res.Code, svc.configSnapshot().Routing)
	}
}
