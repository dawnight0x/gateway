package admin

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/model"
	"local-ai-gateway/internal/store"
	"local-ai-gateway/internal/upstreamhttp"
	adminweb "local-ai-gateway/web/admin"
)

type providerInput struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Type                  string            `json:"type"`
	BaseURL               string            `json:"baseUrl"`
	Priority              *int              `json:"priority"`
	Enabled               *bool             `json:"enabled"`
	ModelMap              map[string]string `json:"modelMap"`
	ModelAllowlistEnabled *bool             `json:"modelAllowlistEnabled"`
	ModelAllowlist        []string          `json:"modelAllowlist"`
	BalancePath           *string           `json:"balancePath"`
}

type keyInput struct {
	ID              string `json:"id"`
	ProviderID      string `json:"providerId"`
	Name            string `json:"name"`
	Secret          string `json:"secret"`
	Priority        *int   `json:"priority"`
	Enabled         *bool  `json:"enabled"`
	ManualPreferred *bool  `json:"manualPreferred"`
}

type gatewayKeyInput struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

const maximumPriority = 1000

func validatePriority(label string, priority int) error {
	if priority < 0 || priority > maximumPriority {
		return fmt.Errorf("%s priority must be between 0 and %d", label, maximumPriority)
	}
	return nil
}

type Service struct {
	store      *store.Store
	cfg        config.Config
	files      http.Handler
	httpClient *http.Client

	discoveryMu      sync.Mutex
	discoveryCtx     context.Context
	discoveryCancel  context.CancelFunc
	discoveryWG      sync.WaitGroup
	discoveryRunning map[string]struct{}
	discoveryPending map[string]bool
}

func New(st *store.Store, cfg config.Config) *Service {
	sub, _ := fs.Sub(adminweb.FS, ".")
	client := upstreamhttp.New(time.Duration(cfg.Routing.TimeoutSeconds)*time.Second, 0, max(cfg.Routing.MaxConcurrentPerProvider, 4))
	return &Service{
		store: st, cfg: cfg, files: http.FileServer(http.FS(sub)), httpClient: client,
		discoveryRunning: make(map[string]struct{}),
		discoveryPending: make(map[string]bool),
	}
}

func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("/admin", s.index)
	mux.HandleFunc("/admin/", s.handle)
}

func (s *Service) index(w http.ResponseWriter, r *http.Request) {
	setAdminSecurityHeaders(w.Header())
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

func (s *Service) handle(w http.ResponseWriter, r *http.Request) {
	setAdminSecurityHeaders(w.Header())
	if strings.HasPrefix(r.URL.Path, "/admin/api/") {
		if !s.allowedAdminHost(r.Host) || !s.allowedAdminOrigin(r.Header.Get("Origin")) || !s.allowedAdminOrigin(r.Header.Get("Referer")) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "admin API only accepts local same-origin requests"})
			return
		}
		if !s.authorizedAdmin(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="local-ai-gateway-admin"`)
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid admin token"})
			return
		}
		s.api(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.StripPrefix("/admin/", s.files).ServeHTTP(w, r)
}

func setAdminSecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	header.Set("Cross-Origin-Resource-Policy", "same-origin")
	header.Set("Cache-Control", "no-store, max-age=0")
	header.Set("Pragma", "no-cache")
	header.Set("Expires", "0")
}

func (s *Service) api(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/admin/api/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch parts[0] {
	case "dashboard":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		s.dashboard(w, r)
	case "providers":
		s.providers(w, r, parts)
	case "keys":
		s.keys(w, r, parts)
	case "gateway-keys":
		s.gatewayKeys(w, r, parts)
	case "logs":
		s.logs(w, r)
	case "routing":
		if r.Method == http.MethodPatch {
			var routing config.RoutingConfig
			if !decodeJSONBody(w, r, &routing) {
				return
			}
			candidate := s.cfg
			candidate.Routing = routing
			if err := candidate.Validate(); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			if err := config.SaveRouting(config.ConfigPath(), routing); err != nil {
				writeAdminStoreError(w, "save routing configuration", err)
				return
			}
			s.cfg.Routing = routing
			writeJSON(w, http.StatusOK, map[string]any{"routing": routing, "restartRequired": true})
			return
		}
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		writeJSON(w, http.StatusOK, s.cfg.Routing)
	case "model-routes":
		s.modelRoutes(w, r, parts)
	case "model-states":
		s.modelStates(w, r, parts)
	case "setup-snippets":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		writeJSON(w, http.StatusOK, s.snippets())
	case "balance":
		s.balance(w, r, parts)
	case "maintenance":
		s.maintenance(w, r, parts)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (s *Service) logs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		deleted, err := s.store.DeleteRequestLogs(r.Context())
		if err != nil {
			writeAdminStoreError(w, "clear request logs", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	query, err := parseLogQuery(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	page, err := s.store.QueryLogs(r.Context(), query)
	if err != nil {
		writeAdminStoreError(w, "load request logs", err)
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		writeLogCSV(w, page.Items)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func parseLogQuery(r *http.Request) (store.LogQuery, error) {
	values := r.URL.Query()
	query := store.LogQuery{
		Limit:      100,
		ProviderID: values.Get("providerId"),
		KeyID:      values.Get("keyId"),
		Model:      values.Get("model"),
		ErrorType:  values.Get("errorType"),
		Search:     values.Get("q"),
	}
	for name, target := range map[string]*int{"limit": &query.Limit, "offset": &query.Offset} {
		if raw := strings.TrimSpace(values.Get(name)); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil {
				return query, fmt.Errorf("%s must be an integer", name)
			}
			*target = value
		}
	}
	if raw := strings.TrimSpace(values.Get("status")); raw != "" {
		status, err := strconv.Atoi(raw)
		if err != nil {
			return query, fmt.Errorf("status must be an integer")
		}
		query.Status = &status
	}
	return query, nil
}

func writeLogCSV(w http.ResponseWriter, logs []model.RequestLog) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="gateway-request-logs.csv"`)
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"request_id", "created_at", "protocol", "provider_id", "key_id", "model", "route_id", "upstream_model", "attempts", "attempt_trace", "trace_truncated", "status", "latency_ms", "prompt_tokens", "completion_tokens", "total_tokens", "error_type"})
	for _, item := range logs {
		attemptTrace, _ := json.Marshal(item.AttemptTrace)
		_ = writer.Write([]string{item.RequestID, item.CreatedAt.Format(time.RFC3339), item.InboundProtocol, item.ProviderID, item.KeyID, item.Model, item.RouteID, item.UpstreamModel, strconv.Itoa(item.Attempts), string(attemptTrace), strconv.FormatBool(item.TraceTruncated), strconv.Itoa(item.Status), strconv.FormatInt(item.LatencyMS, 10), optionalInt(item.PromptTokens), optionalInt(item.CompletionTokens), optionalInt(item.TotalTokens), item.ErrorType})
	}
	writer.Flush()
}

func optionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func (s *Service) maintenance(w http.ResponseWriter, r *http.Request, parts []string) {
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	switch action {
	case "backup":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		item, err := s.store.CreateBackup(r.Context(), "manual")
		if err != nil {
			writeAdminStoreError(w, "create database backup", err)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="`+item.Name+`"`)
		http.ServeFile(w, r, item.Path)
	case "portable-backup":
		if !requireMethod(w, r, http.MethodPost) {
			return
		}
		var input struct {
			Passphrase string `json:"passphrase"`
		}
		if !decodeJSONBody(w, r, &input) {
			return
		}
		if len([]byte(input.Passphrase)) < 12 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "backup passphrase must be at least 12 bytes"})
			return
		}
		item, err := s.store.CreatePortableBackup(r.Context(), input.Passphrase)
		if err != nil {
			writeAdminStoreError(w, "create portable backup", err)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+item.Name+`"`)
		http.ServeFile(w, r, item.Path)
	case "integrity":
		if !requireMethod(w, r, http.MethodGet, http.MethodPost) {
			return
		}
		if err := s.store.IntegrityCheck(r.Context()); err != nil {
			writeAdminStoreError(w, "check database integrity", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	case "backups":
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		items, err := s.store.ListBackups()
		if err != nil {
			writeAdminStoreError(w, "list database backups", err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (s *Service) authorizedAdmin(r *http.Request) bool {
	token := strings.TrimSpace(r.Header.Get("X-Admin-Token"))
	if token == "" {
		auth := r.Header.Get("Authorization")
		if fields := strings.Fields(auth); len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") {
			token = fields[1]
		}
	}
	expected := strings.TrimSpace(s.cfg.Server.AdminToken)
	if token == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func (s *Service) allowedAdminHost(host string) bool {
	return allowedLocalHost(host) ||
		sameHost(host, s.cfg.Server.Host) ||
		sameHost(host, s.cfg.Server.PublicHost)
}

func (s *Service) allowedAdminOrigin(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return s.allowedAdminHost(parsed.Host)
}

func allowedLocalHost(host string) bool {
	host = normalizedHost(host)
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func sameHost(a, b string) bool {
	a = normalizedHost(a)
	b = normalizedHost(b)
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

func normalizedHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return host
}

func (s *Service) dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := s.store.Stats(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load dashboard statistics"})
		return
	}
	providers, err := s.store.ListProviders(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load providers"})
		return
	}
	keys, err := s.store.ListKeys(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load keys"})
		return
	}
	gatewayKeys, err := s.store.ListGatewayKeys(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load gateway keys"})
		return
	}
	balances, err := s.store.ListBalances(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load balances"})
		return
	}
	logs, err := s.store.ListLogs(ctx, 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load request logs"})
		return
	}
	providerModels, err := s.store.ListProviderModels(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load discovered models"})
		return
	}
	modelDiscovery, err := s.store.ListProviderModelDiscoveries(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load model discovery state"})
		return
	}
	modelRoutes, err := s.store.ListModelRoutes(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load model routes"})
		return
	}
	modelStates, err := s.store.ListProviderModelStates(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to load model health state"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": map[string]any{
			"status":   "ok",
			"proxyUrl": s.cfg.PublicURL(),
			"adminUrl": s.cfg.PublicURL() + "/admin",
			"timezone": s.cfg.Storage.Timezone,
		},
		"stats":          stats,
		"providers":      providers,
		"keys":           keys,
		"gatewayKeys":    gatewayKeys,
		"balances":       balances,
		"logs":           logs,
		"providerModels": providerModels,
		"modelDiscovery": modelDiscovery,
		"modelRoutes":    modelRoutes,
		"modelStates":    modelStates,
		"routing":        s.cfg.Routing,
		"snippets":       s.snippets(),
	})
}

func (s *Service) gatewayKeys(w http.ResponseWriter, r *http.Request, parts []string) {
	id := ""
	if len(parts) > 1 {
		id = parts[1]
	}
	action := ""
	if len(parts) > 2 {
		action = parts[2]
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListGatewayKeys(r.Context())
		if err != nil {
			writeAdminStoreError(w, "load gateway keys", err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		if action == "rotate" {
			item, err := s.store.RotateGatewayKey(r.Context(), id)
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
		var input gatewayKeyInput
		if !decodeJSONBody(w, r, &input) {
			return
		}
		item, err := s.store.CreateGatewayKey(r.Context(), input.Name)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPatch:
		var input gatewayKeyInput
		if !decodeJSONBody(w, r, &input) {
			return
		}
		var name *string
		if strings.TrimSpace(input.Name) != "" {
			name = &input.Name
		}
		if err := s.store.PatchGatewayKey(r.Context(), id, name, input.Enabled); err != nil {
			status := http.StatusBadRequest
			if err == sql.ErrNoRows {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		items, err := s.store.ListGatewayKeys(r.Context())
		if err != nil {
			writeAdminStoreError(w, "reload gateway keys", err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodDelete:
		if err := s.store.DeleteGatewayKey(r.Context(), id); err != nil {
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

func (s *Service) providers(w http.ResponseWriter, r *http.Request, parts []string) {
	id := ""
	action := ""
	if len(parts) > 1 {
		id = parts[1]
	}
	if len(parts) > 2 {
		action = parts[2]
	}
	if r.Method == http.MethodPost && id != "" && action == "models" {
		result := s.refreshProviderModels(r.Context(), id)
		writeJSON(w, http.StatusOK, result)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if id != "" {
			p, err := s.store.GetProvider(r.Context(), id)
			if err != nil {
				status := http.StatusBadRequest
				if err == sql.ErrNoRows {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, p)
			return
		}
		items, err := s.store.ListProviders(r.Context())
		if err != nil {
			writeAdminStoreError(w, "load providers", err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost, http.MethodPatch:
		var input providerInput
		if !decodeJSONBody(w, r, &input) {
			return
		}
		p := model.Provider{
			ID:             input.ID,
			Name:           input.Name,
			Type:           input.Type,
			BaseURL:        input.BaseURL,
			ModelMap:       input.ModelMap,
			ModelAllowlist: input.ModelAllowlist,
			Enabled:        true,
		}
		if input.BalancePath != nil {
			p.BalancePath = *input.BalancePath
		}
		if input.Priority != nil {
			p.Priority = *input.Priority
		}
		if input.Enabled != nil {
			p.Enabled = *input.Enabled
		}
		if input.ModelAllowlistEnabled != nil {
			p.ModelAllowlistEnabled = *input.ModelAllowlistEnabled
		}
		if id != "" {
			p.ID = id
		}
		if r.Method == http.MethodPatch {
			current, err := s.store.GetProvider(r.Context(), id)
			if err != nil {
				status := http.StatusBadRequest
				if err == sql.ErrNoRows {
					status = http.StatusNotFound
				}
				writeJSON(w, status, map[string]any{"error": err.Error()})
				return
			}
			p = mergeProvider(*current, input)
		}
		if p.ModelMap == nil {
			p.ModelMap = map[string]string{}
		}
		p.Name = strings.TrimSpace(p.Name)
		p.Type = strings.TrimSpace(p.Type)
		p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
		p.BalancePath = strings.TrimSpace(p.BalancePath)
		p.ModelAllowlist = normalizeModelAllowlist(p.Type, p.ModelAllowlist)
		if err := validateProvider(p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if err := validateUpstreamSecurity(p.BaseURL, s.cfg.Routing.AllowInsecureUpstreams); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		if err := s.validateProviderRouteAllowlist(r.Context(), p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		item, err := s.store.UpsertProvider(r.Context(), p)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		s.scheduleProviderModelDiscovery(item.ID)
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := s.store.DeleteProvider(r.Context(), id); err != nil {
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

func (s *Service) keys(w http.ResponseWriter, r *http.Request, parts []string) {
	id := ""
	action := ""
	if len(parts) > 1 {
		id = parts[1]
	}
	if len(parts) > 2 {
		action = parts[2]
	}
	if r.Method == http.MethodPost && id != "" && action == "test" {
		result := s.testUpstreamKey(r.Context(), id)
		writeJSON(w, http.StatusOK, result)
		return
	}
	if r.Method == http.MethodPost && id != "" && action == "prefer" {
		if err := s.store.PreferKey(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if err == sql.ErrNoRows {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		keys, err := s.store.ListKeys(r.Context())
		if err != nil {
			writeAdminStoreError(w, "reload upstream keys", err)
			return
		}
		writeJSON(w, http.StatusOK, findKey(keys, id))
		return
	}
	if r.Method == http.MethodPost && id != "" && action == "reset" {
		if err := s.store.ResetKey(r.Context(), id); err != nil {
			status := http.StatusBadRequest
			if err == sql.ErrNoRows {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		keys, err := s.store.ListKeys(r.Context())
		if err != nil {
			writeAdminStoreError(w, "reload upstream keys", err)
			return
		}
		writeJSON(w, http.StatusOK, findKey(keys, id))
		return
	}
	switch r.Method {
	case http.MethodGet:
		keys, err := s.store.ListKeys(r.Context())
		if err != nil {
			writeAdminStoreError(w, "load upstream keys", err)
			return
		}
		writeJSON(w, http.StatusOK, keys)
	case http.MethodPost, http.MethodPatch:
		var input keyInput
		if !decodeJSONBody(w, r, &input) {
			return
		}
		k := model.Key{
			ID:         input.ID,
			ProviderID: input.ProviderID,
			Name:       input.Name,
			Secret:     input.Secret,
			Enabled:    true,
		}
		if input.Priority != nil {
			k.Priority = *input.Priority
		}
		if input.Enabled != nil {
			k.Enabled = *input.Enabled
		}
		if input.ManualPreferred != nil {
			k.ManualPreferred = *input.ManualPreferred
		}
		if id != "" {
			k.ID = id
		}
		if r.Method == http.MethodPatch {
			keys, err := s.store.ListKeys(r.Context())
			if err != nil {
				writeAdminStoreError(w, "load upstream key", err)
				return
			}
			current := findKey(keys, id)
			if current == nil {
				writeJSON(w, http.StatusNotFound, map[string]any{"error": sql.ErrNoRows.Error()})
				return
			}
			k = mergeKey(*current, input)
		}
		k.ProviderID = strings.TrimSpace(k.ProviderID)
		k.Name = strings.TrimSpace(k.Name)
		k.Secret = strings.TrimSpace(k.Secret)
		if err := validateKey(k); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		item, err := s.store.UpsertKey(r.Context(), k)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		s.scheduleProviderModelDiscovery(item.ProviderID)
		writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := s.store.DeleteKey(r.Context(), id); err != nil {
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

func (s *Service) snippets() map[string]any {
	root := s.cfg.PublicURL()
	key := s.cfg.Server.ProxyToken
	if key == "" {
		key = "CREATE_A_GATEWAY_KEY_IN_ADMIN"
	}
	return map[string]any{
		"openaiEnv":           "export OPENAI_BASE_URL=" + root + "/v1\nexport OPENAI_API_KEY=" + key,
		"anthropicEnv":        "export ANTHROPIC_BASE_URL=" + root + "\nexport ANTHROPIC_AUTH_TOKEN=" + key,
		"geminiEnv":           "export GEMINI_BASE_URL=" + root + "\nexport GEMINI_API_KEY=" + key,
		"powershellOpenAI":    "$env:OPENAI_BASE_URL=\"" + root + "/v1\"\n$env:OPENAI_API_KEY=\"" + key + "\"",
		"powershellAnthropic": "$env:ANTHROPIC_BASE_URL=\"" + root + "\"\n$env:ANTHROPIC_AUTH_TOKEN=\"" + key + "\"",
		"powershellGemini":    "$env:GEMINI_BASE_URL=\"" + root + "\"\n$env:GEMINI_API_KEY=\"" + key + "\"",
		"cursor":              map[string]any{"baseUrl": root + "/v1", "apiKey": key},
		"claudeCode":          "claude config set --global apiUrl " + root + "\nset ANTHROPIC_AUTH_TOKEN=" + key,
	}
}

func mergeProvider(current model.Provider, patch providerInput) model.Provider {
	if patch.Name != "" {
		current.Name = patch.Name
	}
	if patch.Type != "" {
		current.Type = patch.Type
	}
	if patch.BaseURL != "" {
		current.BaseURL = patch.BaseURL
	}
	if patch.Priority != nil {
		current.Priority = *patch.Priority
	}
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.ModelMap != nil {
		current.ModelMap = patch.ModelMap
	}
	if patch.ModelAllowlistEnabled != nil {
		current.ModelAllowlistEnabled = *patch.ModelAllowlistEnabled
	}
	if patch.ModelAllowlist != nil {
		current.ModelAllowlist = patch.ModelAllowlist
	}
	if patch.BalancePath != nil {
		current.BalancePath = *patch.BalancePath
	}
	return current
}

func validateProvider(p model.Provider) error {
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("provider name is required")
	}
	if err := validatePriority("provider", p.Priority); err != nil {
		return err
	}
	switch strings.TrimSpace(p.Type) {
	case model.ProviderOpenAICompatible, model.ProviderAnthropicCompatible, model.ProviderGeminiCompatible,
		model.ProviderNewAPI, model.ProviderSub2API, model.ProviderCustom:
	default:
		return fmt.Errorf("unsupported provider type %q", p.Type)
	}
	if err := validateHTTPURL("baseUrl", p.BaseURL); err != nil {
		return err
	}
	baseURL, _ := url.Parse(strings.TrimSpace(p.BaseURL))
	if baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return fmt.Errorf("baseUrl must not include a query string or fragment")
	}
	for publicName, upstreamName := range p.ModelMap {
		if strings.TrimSpace(publicName) == "" || strings.TrimSpace(upstreamName) == "" {
			return fmt.Errorf("modelMap names must not be empty")
		}
		if publicName != strings.TrimSpace(publicName) || upstreamName != strings.TrimSpace(upstreamName) {
			return fmt.Errorf("modelMap names must not have leading or trailing whitespace")
		}
	}
	if len(p.ModelAllowlist) > maxDiscoveredModelCount {
		return fmt.Errorf("modelAllowlist must contain at most %d models", maxDiscoveredModelCount)
	}
	for _, modelID := range p.ModelAllowlist {
		if modelID == "" || len(modelID) > 512 {
			return fmt.Errorf("modelAllowlist entries must be 1-512 characters")
		}
	}
	if balancePath := strings.TrimSpace(p.BalancePath); balancePath != "" {
		u, err := url.Parse(balancePath)
		if err != nil {
			return fmt.Errorf("balancePath is invalid: %w", err)
		}
		if u.Fragment != "" {
			return fmt.Errorf("balancePath must not include a fragment")
		}
		if u.Scheme == "" && u.Host != "" {
			return fmt.Errorf("balancePath must be relative or an absolute same-origin URL")
		}
		if u.Scheme != "" {
			if err := validateHTTPURL("balancePath", balancePath); err != nil {
				return err
			}
			if !sameURLOrigin(p.BaseURL, balancePath) {
				return fmt.Errorf("balancePath absolute URL must use the same origin as baseUrl")
			}
		}
	}
	return nil
}

func normalizeModelAllowlist(providerType string, values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = model.NormalizeModelID(providerType, value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validateHTTPURL(field, raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("%s is required", field)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be an absolute http(s) URL", field)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", field)
	}
	if u.User != nil {
		return fmt.Errorf("%s must not include user info", field)
	}
	if u.Fragment != "" {
		return fmt.Errorf("%s must not include a fragment", field)
	}
	return nil
}

func validateUpstreamSecurity(raw string, allowInsecure bool) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("baseUrl is invalid: %w", err)
	}
	if strings.EqualFold(u.Scheme, "https") || config.IsLoopbackHost(u.Hostname()) || allowInsecure {
		return nil
	}
	return fmt.Errorf("baseUrl must use https for non-loopback upstreams; set routing.allow_insecure_upstreams=true to override")
}

func validateKey(k model.Key) error {
	if strings.TrimSpace(k.ProviderID) == "" {
		return fmt.Errorf("providerId is required")
	}
	if strings.TrimSpace(k.Name) == "" {
		return fmt.Errorf("key name is required")
	}
	if strings.TrimSpace(k.Secret) == "" {
		return fmt.Errorf("key secret is required")
	}
	if err := validatePriority("key", k.Priority); err != nil {
		return err
	}
	return nil
}

func sameURLOrigin(a, b string) bool {
	left, err := url.Parse(strings.TrimSpace(a))
	if err != nil {
		return false
	}
	right, err := url.Parse(strings.TrimSpace(b))
	if err != nil {
		return false
	}
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func mergeKey(current model.Key, patch keyInput) model.Key {
	if patch.ProviderID != "" {
		current.ProviderID = patch.ProviderID
	}
	if patch.Name != "" {
		current.Name = patch.Name
	}
	if patch.Secret != "" {
		current.Secret = patch.Secret
	}
	if patch.Priority != nil {
		current.Priority = *patch.Priority
	}
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.ManualPreferred != nil {
		current.ManualPreferred = *patch.ManualPreferred
	}
	return current
}

func findKey(keys []model.Key, id string) *model.Key {
	for _, k := range keys {
		if k.ID == id {
			return &k
		}
	}
	return nil
}

const maxAdminRequestBody = 1 << 20

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxAdminRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeJSONDecodeError(w, err)
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "request body must contain a single JSON object"})
		} else {
			writeJSONDecodeError(w, err)
		}
		return false
	}
	return true
}

func writeJSONDecodeError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"error": "request body exceeds 1 MiB limit"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body: " + err.Error()})
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

func writeAdminStoreError(w http.ResponseWriter, action string, err error) {
	slog.Error("admin storage operation failed", "action", action, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": action + " failed"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write admin JSON response failed", "error", err)
	}
}
