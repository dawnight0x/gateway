package proxy

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/protocol"
	"local-ai-gateway/internal/router"
	"local-ai-gateway/internal/store"
	"local-ai-gateway/internal/upstreamhttp"
)

type Service struct {
	store  *store.Store
	router *router.Router
	cfg    config.Config
	client *http.Client
	limits *requestLimiter

	inFlight         atomic.Int64
	upstreamAttempts atomic.Uint64
	retryAttempts    atomic.Uint64
	rejectedRequests atomic.Uint64
}

const (
	maxProxyRequestBody  = 16 << 20
	maxProxyResponseBody = 64 << 20
	maxStreamAggregate   = 64 << 20
	maxModelNameLength   = 512
	statusClientClosed   = 499
)

type attemptResult struct {
	ok                  bool
	committed           bool
	status              int
	errorType           string
	message             string
	retryable           bool
	retryAfterSeconds   int
	usage               protocol.Usage
	endpointUnsupported bool
	ambiguous           bool
}

func New(st *store.Store, rt *router.Router, cfg config.Config) *Service {
	maxConns := max(cfg.Routing.MaxConcurrentPerProvider, cfg.Routing.MaxConcurrentPerKey)
	return &Service{
		store:  st,
		router: rt,
		cfg:    cfg,
		client: upstreamhttp.New(time.Duration(cfg.Routing.TimeoutSeconds)*time.Second, 0, maxConns),
		limits: newRequestLimiter(cfg.Routing.MaxConcurrentRequests, cfg.Routing.MaxConcurrentPerProvider, cfg.Routing.MaxConcurrentPerKey),
	}
}

func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.health)
	mux.HandleFunc("/status", s.status)
	mux.HandleFunc("/metrics", s.metrics)
	mux.HandleFunc("/v1/models", s.models)
	mux.HandleFunc("/models", s.models)
	mux.HandleFunc("/v1/chat/completions", s.proxy)
	mux.HandleFunc("/chat/completions", s.proxy)
	mux.HandleFunc("/v1/completions", s.proxy)
	mux.HandleFunc("/completions", s.proxy)
	mux.HandleFunc("/v1/responses", s.proxy)
	mux.HandleFunc("/responses", s.proxy)
	mux.HandleFunc("/v1/messages", s.proxy)
	mux.HandleFunc("/messages", s.proxy)
	mux.HandleFunc("/v1beta/models", s.models)
	mux.HandleFunc("/v1beta/models/", s.proxy)
	mux.HandleFunc("/v1/models/", s.proxy)
}

func (s *Service) proxy(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := requestID()
	w.Header().Set("X-Gateway-Request-ID", requestID)
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "invalid local gateway key", "type": "auth_error"}})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxProxyRequestBody)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if r.Context().Err() != nil {
			s.log(r.Context(), requestID, protocol.DetectInbound(r.URL.Path), "", "", "", statusClientClosed, start, protocol.Usage{}, "client_canceled")
			return
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			s.log(r.Context(), requestID, protocol.DetectInbound(r.URL.Path), "", "", "", http.StatusRequestEntityTooLarge, start, protocol.Usage{}, "request_too_large")
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": map[string]any{"message": "request body exceeds 16 MiB limit", "type": "request_too_large"}})
			return
		}
		s.log(r.Context(), requestID, protocol.DetectInbound(r.URL.Path), "", "", "", http.StatusBadRequest, start, protocol.Usage{}, "invalid_request")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": err.Error(), "type": "invalid_request"}})
		return
	}

	inbound := protocol.DetectInbound(r.URL.Path)
	pathModel := protocol.ExtractPathModel(r.URL.Path)
	modelName, err := modelFromBody(body)
	if err != nil {
		s.log(r.Context(), requestID, inbound, "", "", "", http.StatusBadRequest, start, protocol.Usage{}, "invalid_request")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "invalid JSON request: " + err.Error(), "type": "invalid_request"}})
		return
	}
	if modelName == "" {
		modelName = pathModel
	}
	if modelName == "" {
		s.log(r.Context(), requestID, inbound, "", "", "", http.StatusBadRequest, start, protocol.Usage{}, "invalid_request")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "model is required", "type": "invalid_request"}})
		return
	}
	if len(modelName) > maxModelNameLength {
		s.log(r.Context(), requestID, inbound, "", "", "", http.StatusBadRequest, start, protocol.Usage{}, "invalid_request")
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": "model exceeds 512 byte limit", "type": "invalid_request"}})
		return
	}
	candidates, err := s.router.Candidates(r.Context(), modelName, inbound)
	if err != nil {
		if r.Context().Err() != nil {
			s.log(r.Context(), requestID, inbound, "", "", modelName, statusClientClosed, start, protocol.Usage{}, "client_canceled")
			return
		}
		s.log(r.Context(), requestID, inbound, "", "", modelName, http.StatusInternalServerError, start, protocol.Usage{}, "storage_error")
		writeProxyStoreError(w, "load routing candidates", err)
		return
	}
	if len(candidates) == 0 {
		s.log(r.Context(), requestID, inbound, "", "", modelName, http.StatusServiceUnavailable, start, protocol.Usage{}, "no_available_key")
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]any{"message": "no available upstream key", "type": "no_available_key"}})
		return
	}

	maxAttempts := s.cfg.Routing.RetryPerRequest
	if maxAttempts > len(candidates) {
		maxAttempts = len(candidates)
	}
	requestedStream := isStreamRequest(r.URL.Path, body)
	queueCtx := r.Context()
	queueCancel := func() {}
	if s.cfg.Routing.QueueTimeoutMilliseconds > 0 {
		queueCtx, queueCancel = context.WithTimeout(r.Context(), time.Duration(s.cfg.Routing.QueueTimeoutMilliseconds)*time.Millisecond)
	}
	defer queueCancel()
	attempted := 0
	busy := false
	var last attemptResult
	var lastProviderID, lastKeyID string
	for _, key := range candidates {
		if attempted >= maxAttempts {
			break
		}
		releaseProbe, probeOK := s.router.AcquireRecoveryProbe(key)
		if !probeOK {
			busy = true
			continue
		}
		releaseCapacity, capacityOK := s.limits.acquire(queueCtx, key.ProviderID, key.ID)
		if !capacityOK {
			releaseProbe()
			busy = true
			break
		}
		s.inFlight.Add(1)
		attempted++
		s.upstreamAttempts.Add(1)
		if attempted > 1 {
			s.retryAttempts.Add(1)
		}
		lastProviderID, lastKeyID = key.ProviderID, key.ID
		native := router.ChooseUpstreamProtocol(inbound, key)
		protocols := []string{native}
		if fallback := openAIFallbackProtocol(r.URL.Path, inbound, native); fallback != "" {
			protocols = append(protocols, fallback)
		}
		var result attemptResult
		for protocolIndex, upstream := range protocols {
			if protocolIndex > 0 {
				s.upstreamAttempts.Add(1)
				s.retryAttempts.Add(1)
			}
			converted, convertErr := protocol.ConvertRequest(body, inbound, upstream, key.UpstreamModel, pathModel)
			if convertErr != nil {
				result = attemptResult{status: http.StatusBadRequest, errorType: "protocol_error", message: convertErr.Error()}
				break
			}
			if requestedStream {
				converted, convertErr = protocol.EnableStreaming(converted, upstream)
				if convertErr != nil {
					result = attemptResult{status: http.StatusBadRequest, errorType: "protocol_error", message: convertErr.Error()}
					break
				}
			}
			target := upstreamURL(key.ProviderBaseURL, protocol.UpstreamPath(r.URL.Path, upstream, key.UpstreamModel, converted.Stream))
			if upstream == router.ProtocolGemini && converted.Stream {
				target = withQuery(target, "alt", "sse")
			}
			result = s.forward(w, r, target, key, inbound, upstream, requestID, converted)
			if !result.ok && r.Context().Err() != nil {
				result = clientCanceledResult(result)
			}
			if result.ok {
				if err := s.router.RecordSuccess(r.Context(), key); err != nil {
					slog.Warn("record upstream success failed", "key_id", key.ID, "error", err)
				}
				s.inFlight.Add(-1)
				releaseCapacity()
				releaseProbe()
				s.log(r.Context(), requestID, inbound, key.ProviderID, key.ID, modelName, result.status, start, result.usage, "")
				return
			}
			if protocolIndex == 0 && result.endpointUnsupported && len(protocols) > 1 {
				continue
			}
			break
		}
		last = result
		if router.CountsAgainstKeyHealth(result.errorType) {
			if err := s.router.RecordFailure(r.Context(), key, router.Failure{
				Status:            result.status,
				ErrorType:         result.errorType,
				Message:           result.message,
				RetryAfterSeconds: result.retryAfterSeconds,
			}); err != nil {
				slog.Warn("record upstream failure failed", "key_id", key.ID, "error", err)
			}
		}
		s.inFlight.Add(-1)
		releaseCapacity()
		releaseProbe()
		if result.committed {
			s.log(r.Context(), requestID, inbound, key.ProviderID, key.ID, modelName, result.status, start, result.usage, result.errorType)
			return
		}
		if !result.retryable || (result.ambiguous && !s.cfg.Routing.RetryAmbiguousErrors) {
			break
		}
	}
	if attempted == 0 && busy {
		s.rejectedRequests.Add(1)
		w.Header().Set("Retry-After", "1")
		s.log(r.Context(), requestID, inbound, "", "", modelName, http.StatusServiceUnavailable, start, protocol.Usage{}, "gateway_busy")
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]any{"message": "all matching upstream keys are busy or recovering", "type": "gateway_busy"}})
		return
	}
	status := last.status
	if status < 400 {
		status = http.StatusBadGateway
	}
	s.log(r.Context(), requestID, inbound, lastProviderID, lastKeyID, modelName, status, start, last.usage, last.errorType)
	writeJSON(w, status, map[string]any{"error": map[string]any{"message": last.message, "type": last.errorType}})
}

func (s *Service) forward(w http.ResponseWriter, r *http.Request, target string, key model.Key, inbound, upstream, gatewayRequestID string, converted protocol.ConvertedRequest) attemptResult {
	ctx := r.Context()
	cancel := func() {}
	if !converted.Stream {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(s.cfg.Routing.TimeoutSeconds)*time.Second)
	}
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(converted.Body))
	if err != nil {
		return attemptResult{status: 0, errorType: "upstream_error", message: sanitizeUpstreamMessage(err.Error(), key.Secret), retryable: true}
	}
	copyHeaders(req.Header, r.Header)
	req.Header.Set("X-Gateway-Request-ID", gatewayRequestID)
	if req.Header.Get("Idempotency-Key") == "" {
		req.Header.Set("Idempotency-Key", gatewayRequestID)
	}
	applyAuth(req.Header, key.Secret, upstream)

	resp, err := s.client.Do(req)
	if err != nil {
		errType := router.Classify(0, err.Error())
		return attemptResult{status: 0, errorType: errType, message: sanitizeUpstreamMessage(err.Error(), key.Secret), retryable: router.Retryable(errType), ambiguous: true}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		errType := router.Classify(resp.StatusCode, string(body))
		unsupported := isUnsupportedOpenAIEndpoint(resp.StatusCode, body)
		return attemptResult{status: resp.StatusCode, errorType: errType, message: sanitizeUpstreamMessage(errorMessage(body, resp.Status), key.Secret), retryable: unsupported || router.Retryable(errType), retryAfterSeconds: retryAfter(resp.Header.Get("Retry-After")), endpointUnsupported: unsupported, ambiguous: resp.StatusCode >= 500}
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return s.pipeStream(w, resp, inbound, upstream)
	}

	body, err := readResponseBody(resp.Body, maxProxyResponseBody)
	if err != nil {
		return attemptResult{status: http.StatusBadGateway, errorType: "upstream_error", message: sanitizeUpstreamMessage(err.Error(), key.Secret), retryable: true, ambiguous: true}
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return attemptResult{status: http.StatusBadGateway, errorType: "empty_response", message: "empty upstream response", retryable: true, ambiguous: true}
	}
	if converted.Stream {
		frames, usage, err := protocol.ConvertJSONResponseToStream(body, inbound, upstream)
		if err != nil {
			return attemptResult{status: http.StatusBadGateway, errorType: "protocol_error", message: err.Error(), retryable: true, ambiguous: true}
		}
		output := newStreamOutput(w, resp.StatusCode)
		for _, frame := range frames {
			if err := output.write(encodeSSEFrame(frame)); err != nil {
				return attemptResult{committed: output.committed, status: resp.StatusCode, errorType: "stream_interrupted", message: err.Error(), usage: usage}
			}
		}
		return attemptResult{ok: true, committed: output.committed, status: resp.StatusCode, usage: usage}
	}
	out := protocol.ConvertResponse(body, inbound, upstream)
	forwardResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := w.Write(out); err != nil {
		slog.Warn("write proxy response failed", "error", err)
	}
	return attemptResult{ok: true, status: resp.StatusCode, usage: protocol.ExtractUsage(out)}
}

func clientCanceledResult(result attemptResult) attemptResult {
	return attemptResult{
		committed: result.committed,
		status:    statusClientClosed,
		errorType: "client_canceled",
		message:   "request canceled by client",
		usage:     result.usage,
	}
}

func openAIFallbackProtocol(path, inbound, native string) string {
	if inbound == router.ProtocolOpenAIResponses && native == router.ProtocolOpenAIResponses {
		return router.ProtocolOpenAI
	}
	if inbound == router.ProtocolOpenAI && native == router.ProtocolOpenAI && strings.Contains(path, "/chat/completions") {
		return router.ProtocolOpenAIResponses
	}
	return ""
}

func isUnsupportedOpenAIEndpoint(status int, body []byte) bool {
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed || status == http.StatusNotImplemented {
		return true
	}
	if status != http.StatusBadRequest {
		return false
	}
	message := strings.ToLower(string(body))
	markers := []string{
		"unsupported legacy protocol", "unsupported protocol", "endpoint is not supported", "unsupported endpoint", "unknown endpoint", "no route for",
		"responses api is not supported", "does not support /v1/responses", "not support responses", "use /v1/responses", "use /v1/chat/completions",
	}
	for _, marker := range markers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func (s *Service) health(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	if err := s.store.Ping(r.Context()); err != nil {
		slog.Error("gateway health check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unhealthy"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Service) status(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid local gateway key"})
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		writeProxyStoreError(w, "load status statistics", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"service": map[string]any{"status": "ok", "proxyUrl": s.cfg.PublicURL()}, "stats": stats})
}

func (s *Service) metrics(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid local gateway key"})
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	stats, err := s.store.Stats(r.Context())
	if err != nil {
		writeProxyStoreError(w, "load metrics", err)
		return
	}
	body := fmt.Sprintf("# TYPE gateway_keys_total gauge\ngateway_keys_total %d\n# TYPE gateway_keys_active gauge\ngateway_keys_active %d\n# TYPE gateway_keys_cooling_down gauge\ngateway_keys_cooling_down %d\n# TYPE gateway_today_requests gauge\ngateway_today_requests %d\n# TYPE gateway_today_tokens gauge\ngateway_today_tokens %d\n# TYPE gateway_request_logs_dropped_total counter\ngateway_request_logs_dropped_total %d\n# TYPE gateway_in_flight_requests gauge\ngateway_in_flight_requests %d\n# TYPE gateway_upstream_attempts_total counter\ngateway_upstream_attempts_total %d\n# TYPE gateway_retry_attempts_total counter\ngateway_retry_attempts_total %d\n# TYPE gateway_rejected_requests_total counter\ngateway_rejected_requests_total %d\n", stats.TotalKeys, stats.ActiveKeys, stats.FailedKeys, stats.TodayRequests, stats.TodayTokens, stats.DroppedRequestLogs, s.inFlight.Load(), s.upstreamAttempts.Load(), s.retryAttempts.Load(), s.rejectedRequests.Load())
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if _, err := w.Write([]byte(body)); err != nil {
		slog.Warn("write metrics response failed", "error", err)
	}
}

func (s *Service) models(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid local gateway key"})
		return
	}
	if r.Method != http.MethodGet {
		requireMethod(w, r, http.MethodGet)
		return
	}
	providers, err := s.store.ListProviders(r.Context())
	if err != nil {
		writeProxyStoreError(w, "load models", err)
		return
	}
	discovered, err := s.store.ListProviderModels(r.Context())
	if err != nil {
		writeProxyStoreError(w, "load discovered models", err)
		return
	}
	set := make(map[string]bool)
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		for public := range p.ModelMap {
			public = strings.TrimSpace(public)
			if public != "" && public != "*" && public != "default" {
				set[public] = true
			}
		}
		if len(p.ModelMap) == 0 {
			for _, modelID := range discovered[p.ID] {
				if modelID = strings.TrimSpace(modelID); modelID != "" {
					set[modelID] = true
				}
			}
		}
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if strings.HasPrefix(r.URL.Path, "/v1beta/") || (r.URL.Path == "/v1/models" && r.Header.Get("x-goog-api-key") != "") {
		data := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			data = append(data, map[string]any{
				"name":                       "models/" + id,
				"displayName":                id,
				"supportedGenerationMethods": []string{"generateContent", "streamGenerateContent"},
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": data})
		return
	}
	data := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]any{"id": id, "object": "model", "owned_by": "local-gateway"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func (s *Service) authorized(r *http.Request) bool {
	token := requestGatewayToken(r)
	if token == "" {
		return false
	}
	if ok, err := s.store.VerifyGatewayKey(r.Context(), token); err == nil && ok {
		return true
	} else if err != nil {
		slog.Warn("verify gateway key failed", "error", err)
	}
	expected := s.cfg.Server.ProxyToken
	return expected != "" && subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func requestGatewayToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if fields := strings.Fields(auth); len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") {
		return fields[1]
	}
	for _, header := range []string{"x-api-key", "anthropic-auth-token", "x-goog-api-key"} {
		if v := strings.TrimSpace(r.Header.Get(header)); v != "" {
			return v
		}
	}
	return ""
}

func (s *Service) log(ctx context.Context, requestID, inbound, providerID, keyID, modelName string, status int, start time.Time, usage protocol.Usage, errType string) {
	if err := s.store.LogRequest(ctx, model.RequestLog{RequestID: requestID, InboundProtocol: inbound, ProviderID: providerID, KeyID: keyID, Model: modelName, Status: status, LatencyMS: time.Since(start).Milliseconds(), PromptTokens: usage.PromptTokens, CompletionTokens: usage.CompletionTokens, TotalTokens: usage.TotalTokens, ErrorType: errType}); err != nil {
		slog.Warn("record request log failed", "request_id", requestID, "error", err)
	}
}

func upstreamURL(baseURL, upstreamPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path := "/" + strings.TrimLeft(upstreamPath, "/")
	if base == "" {
		return path
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return base + path
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	for _, version := range []string{"/v1beta", "/v1"} {
		if (basePath == version || strings.HasSuffix(basePath, version)) && (path == version || strings.HasPrefix(path, version+"/")) {
			parsed.Path = strings.TrimRight(basePath+strings.TrimPrefix(path, version), "/")
			if parsed.Path == "" {
				parsed.Path = "/"
			}
			return parsed.String()
		}
	}
	return base + path
}

func withQuery(rawURL, key, value string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func copyHeaders(dst, src http.Header) {
	for _, name := range []string{"Content-Type", "Accept", "User-Agent", "OpenAI-Beta", "Anthropic-Version", "Anthropic-Beta", "Idempotency-Key"} {
		if v := src.Get(name); v != "" {
			dst.Set(name, v)
		}
	}
	if dst.Get("Content-Type") == "" {
		dst.Set("Content-Type", "application/json")
	}
}

func applyAuth(h http.Header, secret, upstream string) {
	h.Del("Authorization")
	h.Del("x-api-key")
	h.Del("x-goog-api-key")
	switch upstream {
	case router.ProtocolAnthropic:
		h.Set("x-api-key", secret)
		if h.Get("Anthropic-Version") == "" {
			h.Set("Anthropic-Version", "2023-06-01")
		}
	case router.ProtocolGemini:
		h.Set("x-goog-api-key", secret)
	default:
		h.Set("Authorization", "Bearer "+secret)
	}
}

func forwardResponseHeaders(dst, src http.Header) {
	if ct := src.Get("Content-Type"); ct != "" {
		dst.Set("Content-Type", ct)
	} else {
		dst.Set("Content-Type", "application/json; charset=utf-8")
	}
	forwardResponseMetadataHeaders(dst, src)
}

func forwardResponseMetadataHeaders(dst, src http.Header) {
	for _, name := range []string{
		"OpenAI-Request-ID", "X-Request-ID", "Request-ID", "Retry-After",
		"X-RateLimit-Limit-Requests", "X-RateLimit-Remaining-Requests", "X-RateLimit-Reset-Requests",
		"X-RateLimit-Limit-Tokens", "X-RateLimit-Remaining-Tokens", "X-RateLimit-Reset-Tokens",
	} {
		values := src.Values(name)
		if len(values) == 0 {
			continue
		}
		dst.Del(name)
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

func readResponseBody(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("upstream response exceeds %d MiB limit", limit>>20)
	}
	return body, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write JSON response failed", "error", err)
	}
}

func writeProxyStoreError(w http.ResponseWriter, action string, err error) {
	slog.Error("proxy storage operation failed", "action", action, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": action + " failed"})
}

func requireMethod(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, method := range allowed {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	return false
}

func requestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

func modelFromBody(body []byte) (string, error) {
	var raw struct {
		Model json.RawMessage `json:"model"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", err
	}
	if len(raw.Model) == 0 || string(raw.Model) == "null" {
		return "", nil
	}
	var modelName string
	if err := json.Unmarshal(raw.Model, &modelName); err != nil {
		return "", fmt.Errorf("model must be a string")
	}
	return strings.TrimSpace(modelName), nil
}

func retryAfter(v string) int {
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	if t, err := http.ParseTime(v); err == nil {
		return int(time.Until(t).Seconds())
	}
	return 0
}

func errorMessage(body []byte, fallback string) string {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err == nil {
		if e, ok := raw["error"].(map[string]any); ok {
			if msg, ok := e["message"].(string); ok {
				return msg
			}
		}
		if msg, ok := raw["message"].(string); ok {
			return msg
		}
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return fallback
	}
	if len(text) > 300 {
		return text[:300]
	}
	return text
}

func sanitizeUpstreamMessage(message, secret string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "upstream request failed"
	}
	if secret = strings.TrimSpace(secret); secret != "" {
		message = strings.ReplaceAll(message, secret, "***")
	}
	lower := strings.ToLower(message)
	for _, marker := range []string{"authorization:", "bearer ", "x-api-key:", "x-goog-api-key:", "anthropic-auth-token:"} {
		if strings.Contains(lower, marker) {
			return "upstream request failed; sensitive details redacted"
		}
	}
	if len(message) > 300 {
		return message[:300]
	}
	return message
}

func isStreamRequest(path string, body []byte) bool {
	if strings.Contains(path, ":streamGenerateContent") {
		return true
	}
	var raw struct {
		Stream bool `json:"stream"`
	}
	return json.Unmarshal(body, &raw) == nil && raw.Stream
}
