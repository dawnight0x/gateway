package admin

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"
	"time"

	"local-ai-gateway/internal/model"
)

const modelDiscoveryStartupDelay = 2 * time.Second

func (s *Service) StartModelDiscovery(parent context.Context) func() {
	if !s.cfg.ModelDiscovery.Enabled {
		return func() {}
	}
	s.discoveryMu.Lock()
	if s.discoveryCancel != nil {
		s.discoveryMu.Unlock()
		return s.StopModelDiscovery
	}
	ctx, cancel := context.WithCancel(parent)
	s.discoveryCtx = ctx
	s.discoveryCancel = cancel
	s.discoveryWG.Add(1)
	s.discoveryMu.Unlock()
	go func() {
		defer s.discoveryWG.Done()
		timer := time.NewTimer(modelDiscoveryStartupDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.scheduleStaleModelDiscoveries(ctx)
		}
		interval := time.Duration(s.cfg.ModelDiscovery.RefreshIntervalHours) * time.Hour
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.scheduleStaleModelDiscoveries(ctx)
			}
		}
	}()
	return s.StopModelDiscovery
}

func (s *Service) StopModelDiscovery() {
	s.discoveryMu.Lock()
	cancel := s.discoveryCancel
	s.discoveryCtx = nil
	s.discoveryCancel = nil
	s.discoveryMu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.discoveryWG.Wait()
}

func (s *Service) scheduleStaleModelDiscoveries(ctx context.Context) {
	discoveries, err := s.store.ListProviderModelDiscoveries(ctx)
	if err != nil {
		slog.Warn("load model discovery state failed", "error", err)
		return
	}
	keys, err := s.store.ListKeys(ctx)
	if err != nil {
		slog.Warn("load keys for model discovery failed", "error", err)
		return
	}
	staleBefore := time.Now().Add(-time.Duration(s.cfg.ModelDiscovery.RefreshIntervalHours) * time.Hour)
	seen := make(map[string]struct{})
	for _, key := range keys {
		if !key.Enabled || !key.ProviderEnabled {
			continue
		}
		if _, ok := seen[key.ProviderID]; ok {
			continue
		}
		seen[key.ProviderID] = struct{}{}
		state, ok := discoveries[key.ProviderID]
		if ok && state.LastSuccessAt != nil && state.LastSuccessAt.After(staleBefore) {
			continue
		}
		s.scheduleProviderModelDiscovery(key.ProviderID)
	}
}

func (s *Service) scheduleProviderModelDiscovery(providerID string) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" || !s.cfg.ModelDiscovery.Enabled {
		return
	}
	s.discoveryMu.Lock()
	ctx := s.discoveryCtx
	if ctx == nil || ctx.Err() != nil {
		s.discoveryMu.Unlock()
		return
	}
	if _, running := s.discoveryRunning[providerID]; running {
		s.discoveryPending[providerID] = true
		s.discoveryMu.Unlock()
		return
	}
	s.discoveryRunning[providerID] = struct{}{}
	s.discoveryWG.Add(1)
	s.discoveryMu.Unlock()
	go func() {
		defer func() {
			s.discoveryMu.Lock()
			delete(s.discoveryRunning, providerID)
			rerun := s.discoveryPending[providerID]
			delete(s.discoveryPending, providerID)
			s.discoveryMu.Unlock()
			s.discoveryWG.Done()
			if rerun {
				s.scheduleProviderModelDiscovery(providerID)
			}
		}()
		if result := s.refreshProviderModels(ctx, providerID); result.Status != "ok" {
			slog.Warn("automatic model discovery failed", "provider_id", providerID, "status", result.Status, "error", result.Error)
		}
	}()
}

func (s *Service) refreshProviderModels(ctx context.Context, providerID string) keyTestResult {
	result := keyTestResult{ProviderID: providerID, Status: "not_found", Error: "enabled upstream key not found", CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	keys, err := s.store.ListKeys(ctx)
	if err != nil {
		result.Status = "store_error"
		result.Error = err.Error()
		_ = s.store.RecordProviderModelDiscoveryFailure(ctx, providerID, result.Status, result.Error)
		return result
	}
	var selected *model.Key
	for i := range keys {
		if keys[i].ProviderID == providerID && keys[i].ProviderEnabled && keys[i].Enabled {
			selected = &keys[i]
			break
		}
	}
	if selected == nil {
		_ = s.store.RecordProviderModelDiscoveryFailure(ctx, providerID, result.Status, result.Error)
		return result
	}
	for _, path := range testPathsForKey(*selected) {
		attempt := s.discoverModelsAtPath(ctx, *selected, path)
		if attempt.Status == "ok" {
			return attempt
		}
		result = attempt
		if attempt.Status != "not_found" {
			break
		}
	}
	if result.Status == "" {
		result.Status = "not_found"
		result.Error = sql.ErrNoRows.Error()
	}
	return result
}
