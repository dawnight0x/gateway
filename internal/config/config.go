package config

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"local-ai-gateway/internal/fileutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server" json:"server"`
	Routing RoutingConfig `yaml:"routing" json:"routing"`
	Storage StorageConfig `yaml:"storage" json:"storage"`
	Logging LoggingConfig `yaml:"logging" json:"logging"`
	Tray    TrayConfig    `yaml:"tray" json:"tray"`
}

type ServerConfig struct {
	Host                   string `yaml:"host" json:"host"`
	PublicHost             string `yaml:"public_host" json:"publicHost"`
	Port                   int    `yaml:"port" json:"port"`
	AllowRemote            bool   `yaml:"allow_remote" json:"allowRemote"`
	AllowInsecureRemote    bool   `yaml:"allow_insecure_remote" json:"allowInsecureRemote"`
	TLSCertFile            string `yaml:"tls_cert_file" json:"tlsCertFile"`
	TLSKeyFile             string `yaml:"tls_key_file" json:"tlsKeyFile"`
	ProxyToken             string `yaml:"proxy_token" json:"-"`
	AdminToken             string `yaml:"admin_token" json:"-"`
	OpenBrowserOnDuplicate bool   `yaml:"open_browser_on_duplicate" json:"openBrowserOnDuplicate"`
	ReadTimeoutSeconds     int    `yaml:"read_timeout_seconds" json:"readTimeoutSeconds"`
	IdleTimeoutSeconds     int    `yaml:"idle_timeout_seconds" json:"idleTimeoutSeconds"`
	MaxHeaderBytes         int    `yaml:"max_header_bytes" json:"maxHeaderBytes"`
}

type RoutingConfig struct {
	FailureThreshold           int  `yaml:"failure_threshold" json:"failureThreshold"`
	CooldownSeconds            int  `yaml:"cooldown_seconds" json:"cooldownSeconds"`
	AuthCooldownSeconds        int  `yaml:"auth_cooldown_seconds" json:"authCooldownSeconds"`
	RetryPerRequest            int  `yaml:"retry_per_request" json:"retryPerRequest"`
	TimeoutSeconds             int  `yaml:"timeout_seconds" json:"timeoutSeconds"`
	StreamIdleTimeoutSeconds   int  `yaml:"stream_idle_timeout_seconds" json:"streamIdleTimeoutSeconds"`
	StreamWriteTimeoutSeconds  int  `yaml:"stream_write_timeout_seconds" json:"streamWriteTimeoutSeconds"`
	StreamRetryBeforeFirstByte bool `yaml:"stream_retry_before_first_byte" json:"streamRetryBeforeFirstByte"`
	RetryAmbiguousErrors       bool `yaml:"retry_ambiguous_errors" json:"retryAmbiguousErrors"`
	AllowInsecureUpstreams     bool `yaml:"allow_insecure_upstreams" json:"allowInsecureUpstreams"`
	MaxConcurrentRequests      int  `yaml:"max_concurrent_requests" json:"maxConcurrentRequests"`
	MaxConcurrentPerProvider   int  `yaml:"max_concurrent_per_provider" json:"maxConcurrentPerProvider"`
	MaxConcurrentPerKey        int  `yaml:"max_concurrent_per_key" json:"maxConcurrentPerKey"`
	QueueTimeoutMilliseconds   int  `yaml:"queue_timeout_milliseconds" json:"queueTimeoutMilliseconds"`
}

type StorageConfig struct {
	Path                  string `yaml:"path" json:"path"`
	SecretPath            string `yaml:"secret_path" json:"secretPath"`
	AdminPath             string `yaml:"admin_path" json:"adminPath"`
	LogRetentionDays      int    `yaml:"log_retention_days" json:"logRetentionDays"`
	LogMaxEntries         int    `yaml:"log_max_entries" json:"logMaxEntries"`
	Timezone              string `yaml:"timezone" json:"timezone"`
	BackupBeforeMigration bool   `yaml:"backup_before_migration" json:"backupBeforeMigration"`
	BackupRetention       int    `yaml:"backup_retention" json:"backupRetention"`
	RequestLoggingEnabled bool   `yaml:"request_logging_enabled" json:"requestLoggingEnabled"`
}

type LoggingConfig struct {
	MaxSizeMB  int `yaml:"max_size_mb" json:"maxSizeMB"`
	MaxBackups int `yaml:"max_backups" json:"maxBackups"`
}

type TrayConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			Host:                   "127.0.0.1",
			PublicHost:             "localhost",
			Port:                   18787,
			ProxyToken:             "",
			AdminToken:             "change-me",
			OpenBrowserOnDuplicate: true,
			ReadTimeoutSeconds:     30,
			IdleTimeoutSeconds:     120,
			MaxHeaderBytes:         1 << 20,
		},
		Routing: RoutingConfig{
			FailureThreshold:           5,
			CooldownSeconds:            300,
			AuthCooldownSeconds:        86400,
			RetryPerRequest:            3,
			TimeoutSeconds:             120,
			StreamIdleTimeoutSeconds:   120,
			StreamWriteTimeoutSeconds:  30,
			StreamRetryBeforeFirstByte: true,
			RetryAmbiguousErrors:       false,
			AllowInsecureUpstreams:     false,
			MaxConcurrentRequests:      64,
			MaxConcurrentPerProvider:   16,
			MaxConcurrentPerKey:        4,
			QueueTimeoutMilliseconds:   2000,
		},
		Storage: StorageConfig{
			Path:                  "data/gateway.db",
			SecretPath:            "data/secret.key",
			AdminPath:             "data/admin.token",
			LogRetentionDays:      30,
			LogMaxEntries:         100000,
			Timezone:              "Asia/Singapore",
			BackupBeforeMigration: true,
			BackupRetention:       5,
			RequestLoggingEnabled: true,
		},
		Logging: LoggingConfig{MaxSizeMB: 10, MaxBackups: 3},
		Tray:    TrayConfig{Enabled: runtime.GOOS == "windows"},
	}
}

func Load() (Config, error) {
	cfg := Default()
	path := ConfigPath()
	if b, err := os.ReadFile(path); err == nil {
		decoder := yaml.NewDecoder(bytes.NewReader(b))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
			return cfg, fmt.Errorf("decode config %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg.Server.Host = getenv("GATEWAY_HOST", cfg.Server.Host)
	cfg.Server.PublicHost = getenv("GATEWAY_PUBLIC_HOST", cfg.Server.PublicHost)
	cfg.Server.TLSCertFile = getenv("GATEWAY_TLS_CERT_FILE", cfg.Server.TLSCertFile)
	cfg.Server.TLSKeyFile = getenv("GATEWAY_TLS_KEY_FILE", cfg.Server.TLSKeyFile)
	cfg.Server.ProxyToken = getenv("GATEWAY_PROXY_TOKEN", cfg.Server.ProxyToken)
	cfg.Server.AdminToken = getenv("GATEWAY_ADMIN_TOKEN", cfg.Server.AdminToken)
	cfg.Storage.Path = getenv("GATEWAY_DB", cfg.Storage.Path)
	cfg.Storage.SecretPath = getenv("GATEWAY_SECRET_PATH", cfg.Storage.SecretPath)
	cfg.Storage.AdminPath = getenv("GATEWAY_ADMIN_TOKEN_FILE", cfg.Storage.AdminPath)
	cfg.Storage.Timezone = getenv("GATEWAY_TIMEZONE", cfg.Storage.Timezone)
	for _, item := range []struct {
		key string
		dst *int
	}{
		{"GATEWAY_PORT", &cfg.Server.Port},
		{"GATEWAY_READ_TIMEOUT_SECONDS", &cfg.Server.ReadTimeoutSeconds},
		{"GATEWAY_IDLE_TIMEOUT_SECONDS", &cfg.Server.IdleTimeoutSeconds},
		{"GATEWAY_MAX_HEADER_BYTES", &cfg.Server.MaxHeaderBytes},
		{"GATEWAY_FAILURE_THRESHOLD", &cfg.Routing.FailureThreshold},
		{"GATEWAY_COOLDOWN_SECONDS", &cfg.Routing.CooldownSeconds},
		{"GATEWAY_AUTH_COOLDOWN_SECONDS", &cfg.Routing.AuthCooldownSeconds},
		{"GATEWAY_RETRY_PER_REQUEST", &cfg.Routing.RetryPerRequest},
		{"GATEWAY_TIMEOUT_SECONDS", &cfg.Routing.TimeoutSeconds},
		{"GATEWAY_STREAM_IDLE_TIMEOUT_SECONDS", &cfg.Routing.StreamIdleTimeoutSeconds},
		{"GATEWAY_STREAM_WRITE_TIMEOUT_SECONDS", &cfg.Routing.StreamWriteTimeoutSeconds},
		{"GATEWAY_MAX_CONCURRENT_REQUESTS", &cfg.Routing.MaxConcurrentRequests},
		{"GATEWAY_MAX_CONCURRENT_PER_PROVIDER", &cfg.Routing.MaxConcurrentPerProvider},
		{"GATEWAY_MAX_CONCURRENT_PER_KEY", &cfg.Routing.MaxConcurrentPerKey},
		{"GATEWAY_QUEUE_TIMEOUT_MILLISECONDS", &cfg.Routing.QueueTimeoutMilliseconds},
		{"GATEWAY_LOG_RETENTION_DAYS", &cfg.Storage.LogRetentionDays},
		{"GATEWAY_LOG_MAX_ENTRIES", &cfg.Storage.LogMaxEntries},
		{"GATEWAY_BACKUP_RETENTION", &cfg.Storage.BackupRetention},
		{"GATEWAY_LOG_MAX_SIZE_MB", &cfg.Logging.MaxSizeMB},
		{"GATEWAY_LOG_MAX_BACKUPS", &cfg.Logging.MaxBackups},
	} {
		if err := overrideInt(item.key, item.dst); err != nil {
			return cfg, err
		}
	}
	for _, item := range []struct {
		key string
		dst *bool
	}{
		{"GATEWAY_TRAY", &cfg.Tray.Enabled},
		{"GATEWAY_OPEN_BROWSER_ON_DUPLICATE", &cfg.Server.OpenBrowserOnDuplicate},
		{"GATEWAY_STREAM_RETRY_BEFORE_FIRST_BYTE", &cfg.Routing.StreamRetryBeforeFirstByte},
		{"GATEWAY_RETRY_AMBIGUOUS_ERRORS", &cfg.Routing.RetryAmbiguousErrors},
		{"GATEWAY_ALLOW_INSECURE_UPSTREAMS", &cfg.Routing.AllowInsecureUpstreams},
		{"GATEWAY_ALLOW_REMOTE", &cfg.Server.AllowRemote},
		{"GATEWAY_ALLOW_INSECURE_REMOTE", &cfg.Server.AllowInsecureRemote},
		{"GATEWAY_BACKUP_BEFORE_MIGRATION", &cfg.Storage.BackupBeforeMigration},
		{"GATEWAY_REQUEST_LOGGING_ENABLED", &cfg.Storage.RequestLoggingEnabled},
	} {
		if err := overrideBool(item.key, item.dst); err != nil {
			return cfg, err
		}
	}

	cfg.Storage.Path = filepath.Clean(cfg.Storage.Path)
	cfg.Storage.SecretPath = filepath.Clean(cfg.Storage.SecretPath)
	cfg.Storage.AdminPath = filepath.Clean(cfg.Storage.AdminPath)
	if cfg.Server.TLSCertFile != "" {
		cfg.Server.TLSCertFile = filepath.Clean(cfg.Server.TLSCertFile)
	}
	if cfg.Server.TLSKeyFile != "" {
		cfg.Server.TLSKeyFile = filepath.Clean(cfg.Server.TLSKeyFile)
	}
	if os.Getenv("GATEWAY_ADMIN_TOKEN") == "" && weakAdminToken(cfg.Server.AdminToken) {
		token, err := loadOrCreateAdminToken(cfg.Storage.AdminPath)
		if err != nil {
			return cfg, err
		}
		cfg.Server.AdminToken = token
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func ConfigPath() string {
	return getenv("GATEWAY_CONFIG", "config.yaml")
}

func SaveRouting(path string, routing RoutingConfig) error {
	var document yaml.Node
	if data, err := os.ReadFile(path); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := yaml.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("decode config %s: %w", path, err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if len(document.Content) == 0 {
		document.Kind = yaml.DocumentNode
		document.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("config %s must contain a mapping", path)
	}
	var value yaml.Node
	if err := value.Encode(routing); err != nil {
		return err
	}
	replaced := false
	for index := 0; index+1 < len(root.Content); index += 2 {
		if root.Content[index].Value == "routing" {
			root.Content[index+1] = &value
			replaced = true
			break
		}
	}
	if !replaced {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "routing"},
			&value,
		)
	}
	data, err := yaml.Marshal(&document)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(path, data, 0o600)
}

func (c Config) Validate() error {
	if err := validateHost("server.host", c.Server.Host); err != nil {
		return err
	}
	if err := validateHost("server.public_host", c.Server.PublicHost); err != nil {
		return err
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be a valid TCP port")
	}
	if (c.Server.TLSCertFile == "") != (c.Server.TLSKeyFile == "") {
		return fmt.Errorf("server.tls_cert_file and server.tls_key_file must be configured together")
	}
	if !IsLoopbackHost(c.Server.Host) {
		if !c.Server.AllowRemote {
			return fmt.Errorf("server.host is not loopback; set server.allow_remote=true to expose the gateway")
		}
		if c.Server.TLSCertFile == "" && !c.Server.AllowInsecureRemote {
			return fmt.Errorf("remote listening requires TLS or server.allow_insecure_remote=true")
		}
	}
	if c.Server.AdminToken == "" {
		return fmt.Errorf("server.admin_token is required")
	}
	if c.Server.ReadTimeoutSeconds < 1 {
		return fmt.Errorf("server.read_timeout_seconds must be >= 1")
	}
	if c.Server.IdleTimeoutSeconds < 1 {
		return fmt.Errorf("server.idle_timeout_seconds must be >= 1")
	}
	if c.Server.MaxHeaderBytes < 1024 {
		return fmt.Errorf("server.max_header_bytes must be >= 1024")
	}
	if c.Routing.FailureThreshold < 1 {
		return fmt.Errorf("routing.failure_threshold must be >= 1")
	}
	if c.Routing.CooldownSeconds < 0 {
		return fmt.Errorf("routing.cooldown_seconds must be >= 0")
	}
	if c.Routing.AuthCooldownSeconds < 0 {
		return fmt.Errorf("routing.auth_cooldown_seconds must be >= 0")
	}
	if c.Routing.RetryPerRequest < 1 {
		return fmt.Errorf("routing.retry_per_request must be >= 1")
	}
	if c.Routing.TimeoutSeconds < 1 {
		return fmt.Errorf("routing.timeout_seconds must be >= 1")
	}
	if c.Routing.StreamIdleTimeoutSeconds < 1 {
		return fmt.Errorf("routing.stream_idle_timeout_seconds must be >= 1")
	}
	if c.Routing.StreamWriteTimeoutSeconds < 1 {
		return fmt.Errorf("routing.stream_write_timeout_seconds must be >= 1")
	}
	if c.Routing.MaxConcurrentRequests < 0 || c.Routing.MaxConcurrentPerProvider < 0 || c.Routing.MaxConcurrentPerKey < 0 {
		return fmt.Errorf("routing concurrency limits must be >= 0")
	}
	if c.Routing.QueueTimeoutMilliseconds < 0 {
		return fmt.Errorf("routing.queue_timeout_milliseconds must be >= 0")
	}
	if c.Storage.LogRetentionDays < 0 {
		return fmt.Errorf("storage.log_retention_days must be >= 0")
	}
	if c.Storage.LogMaxEntries < 0 {
		return fmt.Errorf("storage.log_max_entries must be >= 0")
	}
	if c.Storage.BackupRetention < 0 {
		return fmt.Errorf("storage.backup_retention must be >= 0")
	}
	if _, err := time.LoadLocation(c.Storage.Timezone); err != nil {
		return fmt.Errorf("storage.timezone is invalid: %w", err)
	}
	if c.Logging.MaxSizeMB < 1 {
		return fmt.Errorf("logging.max_size_mb must be >= 1")
	}
	if c.Logging.MaxBackups < 0 {
		return fmt.Errorf("logging.max_backups must be >= 0")
	}
	return nil
}

func (s ServerConfig) Addr() string {
	return joinHostPort(s.Host, s.Port)
}

func (c Config) PublicURL() string {
	return c.scheme() + "://" + joinHostPort(c.Server.PublicHost, c.Server.Port)
}

func (c Config) LocalURL() string {
	return c.scheme() + "://" + joinHostPort(c.Server.Host, c.Server.Port)
}

func (c Config) scheme() string {
	if c.Server.TLSCertFile != "" && c.Server.TLSKeyFile != "" {
		return "https"
	}
	return "http"
}

func IsLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if host == "localhost" {
		return true
	}
	if zone := strings.LastIndex(host, "%"); zone > 0 {
		host = host[:zone]
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func joinHostPort(host string, port int) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func validateHost(field, host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("%s is required", field)
	}
	if strings.ContainsAny(host, " \t\r\n/?#@") {
		return fmt.Errorf("%s must be a hostname or IP address without a scheme, path, or port", field)
	}
	if strings.HasPrefix(host, "[") != strings.HasSuffix(host, "]") {
		return fmt.Errorf("%s has invalid IPv6 brackets", field)
	}
	bare := strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	ipHost := bare
	if zone := strings.LastIndex(ipHost, "%"); zone > 0 {
		ipHost = ipHost[:zone]
	}
	if strings.Contains(bare, ":") && net.ParseIP(ipHost) == nil {
		return fmt.Errorf("%s must not include a port", field)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func overrideInt(key string, dst *int) error {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s must be an integer: %w", key, err)
	}
	*dst = n
	return nil
}

func overrideBool(key string, dst *bool) error {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return nil
	}
	switch v {
	case "1", "true", "yes", "on":
		*dst = true
	case "0", "false", "no", "off":
		*dst = false
	default:
		return fmt.Errorf("%s must be a boolean", key)
	}
	return nil
}

func weakAdminToken(token string) bool {
	token = strings.TrimSpace(token)
	return token == "" || token == "change-me"
}

func loadOrCreateAdminToken(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("storage.admin_path is required when using generated admin token")
	}
	if b, err := os.ReadFile(path); err == nil {
		token := strings.TrimSpace(string(b))
		if !weakAdminToken(token) {
			return token, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read admin token: %w", err)
	}
	token, err := randomToken("gat-")
	if err != nil {
		return "", err
	}
	if err := fileutil.WriteFileAtomic(path, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

func randomToken(prefix string) (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b[:]), nil
}
