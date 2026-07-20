package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"local-ai-gateway/internal/model"
)

func (s *Service) modelRoutes(w http.ResponseWriter, r *http.Request, parts []string) {
	id := ""
	if len(parts) > 1 {
		id = strings.TrimSpace(parts[1])
	}
	switch r.Method {
	case http.MethodGet:
		if id != "" {
			item, err := s.store.GetModelRoute(r.Context(), id)
			if err != nil {
				status := http.StatusBadRequest
				if err == sql.ErrNoRows {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, item)
			return
		}
		items, err := s.store.ListModelRoutes(r.Context())
		if err != nil {
			writeAdminStoreError(w, "load model routes", err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		var route model.ModelRoute
		if !decodeJSONBody(w, r, &route) {
			return
		}
		if id != "" {
			route.ID = id
		}
		if err := s.validateModelRoute(r.Context(), &route); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		item, err := s.store.UpsertModelRoute(r.Context(), route)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "model route id is required"})
			return
		}
		if err := s.store.DeleteModelRoute(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if err == sql.ErrNoRows {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Service) validateModelRoute(ctx context.Context, route *model.ModelRoute) error {
	route.ID = strings.TrimSpace(route.ID)
	route.Name = strings.TrimSpace(route.Name)
	if route.ID == "" || len(route.ID) > 512 || route.ID == "*" || route.ID == "default" {
		return fmt.Errorf("model route id must be 1-512 characters and cannot be * or default")
	}
	if route.Name == "" {
		route.Name = route.ID
	}
	if len(route.Name) > 256 {
		return fmt.Errorf("model route name must not exceed 256 characters")
	}
	providers, err := s.store.ListProviders(ctx)
	if err != nil {
		return err
	}
	providerTypes := make(map[string]string, len(providers))
	for _, provider := range providers {
		providerTypes[provider.ID] = provider.Type
	}
	modelNames := make(map[string]struct{})
	activeTargets := 0
	for modelIndex := range route.Models {
		item := &route.Models[modelIndex]
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || len(item.Name) > 512 {
			return fmt.Errorf("route model name must be 1-512 characters")
		}
		if err := validatePriority("route model", item.Priority); err != nil {
			return err
		}
		if _, duplicate := modelNames[item.Name]; duplicate {
			return fmt.Errorf("duplicate route model %q", item.Name)
		}
		modelNames[item.Name] = struct{}{}
		targets := make(map[string]struct{})
		providersInLayer := make(map[string]struct{})
		for targetIndex := range item.Targets {
			target := &item.Targets[targetIndex]
			target.ProviderID = strings.TrimSpace(target.ProviderID)
			providerType, exists := providerTypes[target.ProviderID]
			if !exists {
				return fmt.Errorf("provider %q does not exist", target.ProviderID)
			}
			target.UpstreamModel = model.NormalizeModelID(providerType, target.UpstreamModel)
			if target.ProviderID == "" || target.UpstreamModel == "" || len(target.UpstreamModel) > 512 {
				return fmt.Errorf("model route targets require a provider and a 1-512 character upstream model")
			}
			if _, duplicate := providersInLayer[target.ProviderID]; duplicate {
				return fmt.Errorf("route model %q can contain only one target for provider %q", item.Name, target.ProviderID)
			}
			providersInLayer[target.ProviderID] = struct{}{}
			key := target.ProviderID + "\x00" + target.UpstreamModel
			if _, duplicate := targets[key]; duplicate {
				return fmt.Errorf("duplicate target for provider %q and model %q", target.ProviderID, target.UpstreamModel)
			}
			targets[key] = struct{}{}
			if item.Enabled && target.Enabled {
				activeTargets++
			}
		}
	}
	if route.Enabled && activeTargets == 0 {
		return fmt.Errorf("an enabled model route requires at least one enabled target")
	}
	return nil
}
