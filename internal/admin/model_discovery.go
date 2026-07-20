package admin

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"
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
	foundKey := false
	var successful *keyTestResult
	failures := make([]string, 0)
	for i := range keys {
		key := keys[i]
		if key.ProviderID != providerID || !key.ProviderEnabled || !key.Enabled {
			continue
		}
		foundKey = true
		keySucceeded := false
		var keyFailure keyTestResult
		for _, path := range testPathsForKey(key) {
			attempt := s.discoverModelsAtPath(ctx, key, path)
			if attempt.Status == "ok" {
				keySucceeded = true
				if successful == nil {
					copy := attempt
					successful = &copy
				}
				break
			}
			result = attempt
			keyFailure = attempt
			if attempt.Status != "not_found" {
				break
			}
		}
		if !keySucceeded {
			failures = append(failures, fmt.Sprintf("%s (%s): %s", key.ID, keyFailure.Status, keyFailure.Error))
		}
	}
	if successful != nil {
		inventories, err := s.store.ListProviderModels(ctx)
		if err != nil {
			result = *successful
			result.Status = "storage_error"
			result.Error = "discovered models but failed to load the provider inventory"
			_ = s.store.RecordProviderModelDiscoveryFailure(ctx, providerID, result.Status, result.Error)
			return result
		}
		models := inventories[providerID]
		count := len(models)
		result = *successful
		result.Status = "ok"
		result.Error = ""
		result.ConnectionStatus = "ok"
		result.Models = trimModelList(models, 300)
		result.ModelCount = &count
		var recordErr error
		if len(failures) > 0 {
			result.Status = "partial"
			result.Error = strings.Join(failures, "; ")
			if len(result.Error) > 2048 {
				result.Error = result.Error[:2048]
			}
			recordErr = s.store.RecordProviderModelDiscoveryPartial(ctx, providerID, count, result.Error)
		} else {
			recordErr = s.store.RecordProviderModelDiscoverySuccess(ctx, providerID, count)
		}
		if recordErr != nil {
			result.Status = "storage_error"
			result.Error = "discovered models but failed to save the provider discovery state"
		}
		return result
	}
	if !foundKey {
		_ = s.store.RecordProviderModelDiscoveryFailure(ctx, providerID, result.Status, result.Error)
		return result
	}
	if result.Status == "" {
		result.Status = "not_found"
		result.Error = sql.ErrNoRows.Error()
	}
	return result
}
