package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultUsesLocalhostEverywhere(t *testing.T) {
	cfg := Default()
	if got := cfg.Server.Host; got != "127.0.0.1" {
		t.Fatalf("host = %q, want 127.0.0.1", got)
	}
	if got := cfg.Server.PublicHost; got != "localhost" {
		t.Fatalf("public host = %q, want localhost", got)
	}
	if got := cfg.Server.Addr(); got != "127.0.0.1:18787" {
		t.Fatalf("addr = %q", got)
	}
	if got := cfg.LocalURL(); got != "http://127.0.0.1:18787" {
		t.Fatalf("local url = %q", got)
	}
	if got := cfg.PublicURL(); got != "http://localhost:18787" {
		t.Fatalf("public url = %q", got)
	}
	if got := cfg.Server.ProxyToken; got != "" {
		t.Fatalf("proxy token = %q, want empty by default", got)
	}
	if cfg.Storage.LogRetentionDays != 30 || cfg.Storage.LogMaxEntries != 100000 {
		t.Fatalf("log retention defaults = %#v", cfg.Storage)
	}
}

func TestDefaultTimezoneIsAvailable(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should validate without external timezone data: %v", err)
	}
}

func TestURLsSupportIPv6Hosts(t *testing.T) {
	cfg := Default()
	cfg.Server.Host = "::1"
	cfg.Server.PublicHost = "[::1]"
	if got := cfg.Server.Addr(); got != "[::1]:18787" {
		t.Fatalf("addr = %q", got)
	}
	if got := cfg.LocalURL(); got != "http://[::1]:18787" {
		t.Fatalf("local URL = %q", got)
	}
	if got := cfg.PublicURL(); got != "http://[::1]:18787" {
		t.Fatalf("public URL = %q", got)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("IPv6 config should validate: %v", err)
	}
}

func TestValidateRejectsHostWithSchemePathOrPort(t *testing.T) {
	for name, host := range map[string]string{
		"scheme":       "http://localhost",
		"path":         "localhost/admin",
		"port":         "localhost:18787",
		"bad brackets": "[::1",
	} {
		t.Run(name, func(t *testing.T) {
			cfg := Default()
			cfg.Server.PublicHost = host
			if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "server.public_host") {
				t.Fatalf("host %q validation error = %v", host, err)
			}
		})
	}
}

func TestValidateRequiresExplicitSecureRemoteMode(t *testing.T) {
	cfg := Default()
	cfg.Server.Host = "0.0.0.0"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "allow_remote") {
		t.Fatalf("remote validation error = %v", err)
	}
	cfg.Server.AllowRemote = true
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "requires TLS") {
		t.Fatalf("remote TLS validation error = %v", err)
	}
	cfg.Server.TLSCertFile = "server.crt"
	cfg.Server.TLSKeyFile = "server.key"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("secure remote config should validate: %v", err)
	}
	if got := cfg.PublicURL(); got != "https://localhost:18787" {
		t.Fatalf("TLS public URL = %q", got)
	}
}

func TestLoadReadsLogRetentionEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GATEWAY_CONFIG", filepath.Join(dir, "missing.yaml"))
	t.Setenv("GATEWAY_DB", filepath.Join(dir, "gateway.db"))
	t.Setenv("GATEWAY_SECRET_PATH", filepath.Join(dir, "secret.key"))
	t.Setenv("GATEWAY_ADMIN_TOKEN_FILE", filepath.Join(dir, "admin.token"))
	t.Setenv("GATEWAY_LOG_RETENTION_DAYS", "14")
	t.Setenv("GATEWAY_LOG_MAX_ENTRIES", "2500")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.LogRetentionDays != 14 || cfg.Storage.LogMaxEntries != 2500 {
		t.Fatalf("log retention config = %#v", cfg.Storage)
	}
}

func TestLoadGeneratesAdminTokenWhenDefaultIsWeak(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GATEWAY_CONFIG", filepath.Join(dir, "missing.yaml"))
	t.Setenv("GATEWAY_DB", filepath.Join(dir, "gateway.db"))
	t.Setenv("GATEWAY_SECRET_PATH", filepath.Join(dir, "secret.key"))
	t.Setenv("GATEWAY_ADMIN_TOKEN_FILE", filepath.Join(dir, "admin.token"))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.AdminToken == "" || cfg.Server.AdminToken == "change-me" {
		t.Fatalf("admin token was not generated: %q", cfg.Server.AdminToken)
	}
	if !strings.HasPrefix(cfg.Server.AdminToken, "gat-") {
		t.Fatalf("admin token prefix = %q", cfg.Server.AdminToken)
	}
	saved, err := os.ReadFile(cfg.Storage.AdminPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(saved)) != cfg.Server.AdminToken {
		t.Fatalf("saved token mismatch")
	}

	cfg2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Server.AdminToken != cfg.Server.AdminToken {
		t.Fatalf("generated token changed between loads")
	}
}

func TestLoadKeepsExplicitAdminToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GATEWAY_CONFIG", filepath.Join(dir, "missing.yaml"))
	t.Setenv("GATEWAY_DB", filepath.Join(dir, "gateway.db"))
	t.Setenv("GATEWAY_SECRET_PATH", filepath.Join(dir, "secret.key"))
	t.Setenv("GATEWAY_ADMIN_TOKEN_FILE", filepath.Join(dir, "admin.token"))
	t.Setenv("GATEWAY_ADMIN_TOKEN", "explicit-admin-token")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.AdminToken != "explicit-admin-token" {
		t.Fatalf("admin token = %q", cfg.Server.AdminToken)
	}
	if _, err := os.Stat(cfg.Storage.AdminPath); !os.IsNotExist(err) {
		t.Fatalf("admin token file should not be generated for explicit env token: %v", err)
	}
}

func TestLoadRejectsUnknownConfigFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  unknown_option: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATEWAY_CONFIG", path)
	t.Setenv("GATEWAY_ADMIN_TOKEN", "explicit-admin-token")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "unknown_option") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestLoadRejectsInvalidEnvironmentValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GATEWAY_CONFIG", filepath.Join(dir, "missing.yaml"))
	t.Setenv("GATEWAY_ADMIN_TOKEN", "explicit-admin-token")
	t.Setenv("GATEWAY_PORT", "not-a-number")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "GATEWAY_PORT") {
		t.Fatalf("expected invalid environment error, got %v", err)
	}
}

func TestLoadRejectsInvalidRuntimeMode(t *testing.T) {
	t.Setenv("GATEWAY_MODE", "server")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "GATEWAY_MODE") {
		t.Fatalf("expected invalid runtime mode error, got %v", err)
	}
}

func TestInstalledModeUsesExplicitDataDirectory(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config", "config.yaml")
	t.Setenv("GATEWAY_MODE", "installed")
	t.Setenv("GATEWAY_DATA_DIR", filepath.Join(dir, "state"))
	t.Setenv("GATEWAY_CONFIG", configPath)
	t.Setenv("GATEWAY_ADMIN_TOKEN", "explicit-admin-token")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage.Path != filepath.Join(dir, "state", "gateway.db") ||
		cfg.Storage.SecretPath != filepath.Join(dir, "state", "secret.key") ||
		cfg.Storage.AdminPath != filepath.Join(dir, "state", "admin.token") {
		t.Fatalf("installed storage paths = %#v", cfg.Storage)
	}
	if ConfigPath() != configPath {
		t.Fatalf("installed config path = %q", ConfigPath())
	}
}

func TestExampleConfigLoads(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GATEWAY_CONFIG", filepath.Join("..", "..", "config.example.yaml"))
	t.Setenv("GATEWAY_DB", filepath.Join(dir, "gateway.db"))
	t.Setenv("GATEWAY_SECRET_PATH", filepath.Join(dir, "secret.key"))
	t.Setenv("GATEWAY_ADMIN_TOKEN_FILE", filepath.Join(dir, "admin.token"))
	t.Setenv("GATEWAY_ADMIN_TOKEN", "explicit-admin-token")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAcceptsEmptyConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATEWAY_CONFIG", path)
	t.Setenv("GATEWAY_ADMIN_TOKEN", "explicit-admin-token")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}

func TestSaveRoutingPreservesOtherConfigSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  port: 19000\n# keep this section\nlogging:\n  max_size_mb: 8\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	routing := Default().Routing
	routing.MaxConcurrentPerKey = 7
	if err := SaveRouting(path, routing); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "port: 19000") || !strings.Contains(text, "max_size_mb: 8") || !strings.Contains(text, "max_concurrent_per_key: 7") {
		t.Fatalf("saved config = %s", text)
	}
}
