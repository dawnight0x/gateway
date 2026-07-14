package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"local-ai-gateway/internal/model"

	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Store struct {
	db         *sql.DB
	crypto     *cryptor
	options    Options
	location   *time.Location
	path       string
	secretPath string

	secretCacheMu sync.RWMutex
	secretCache   map[string]cachedSecret

	gatewayKeyCacheMu sync.RWMutex
	gatewayKeyCache   map[string]string

	gatewayUsageMu      sync.Mutex
	gatewayUsageFlushMu sync.Mutex
	pendingGatewayUsage map[string]gatewayUsage
	gatewayUsageStop    chan struct{}
	gatewayUsageWG      sync.WaitGroup
	keySuccessMu        sync.Mutex
	keySuccessFlushMu   sync.Mutex
	pendingKeySuccesses map[string]keySuccess

	requestLogMu          sync.Mutex
	requestLogFlushMu     sync.Mutex
	pendingRequestLogs    []model.RequestLog
	requestMetricsMu      sync.Mutex
	requestMetricsFlushMu sync.Mutex
	pendingRequestMetrics map[string]dailyRequestMetric
	droppedRequestLogs    atomic.Uint64
	maintenanceMu         sync.Mutex

	closeOnce sync.Once
	closeErr  error
}

type Options struct {
	LogRetentionDays      int
	LogMaxEntries         int
	Timezone              string
	BackupBeforeMigration bool
	BackupRetention       int
	DisableRequestLogging bool
}

type FailurePolicy struct {
	Threshold         int
	Cooldown          time.Duration
	ThresholdCooldown time.Duration
	ForceCooldown     bool
	OverrideCooldown  time.Duration
}

type cachedSecret struct {
	cipher string
	secret string
}

type gatewayUsage struct {
	count    int
	lastUsed time.Time
}

type keySuccess struct {
	count       int
	lastUsed    time.Time
	resetHealth bool
}

type dailyRequestMetric struct {
	requests int
	tokens   int
}

const (
	gatewayUsageFlushInterval = 2 * time.Second
	requestLogFlushInterval   = 250 * time.Millisecond
	requestLogBatchSize       = 64
	maxPendingRequestLogs     = 8192
	maintenanceInterval       = time.Hour
)

var schemaMigrations = []string{
	`CREATE TABLE IF NOT EXISTS providers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		base_url TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER NOT NULL DEFAULT 1,
		model_map TEXT NOT NULL DEFAULT '{}',
		balance_path TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS keys (
		id TEXT PRIMARY KEY,
		provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		secret_cipher TEXT NOT NULL,
		priority INTEGER NOT NULL DEFAULT 0,
		enabled INTEGER NOT NULL DEFAULT 1,
		manual_preferred INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS key_state (
		key_id TEXT PRIMARY KEY REFERENCES keys(id) ON DELETE CASCADE,
		consecutive_failures INTEGER NOT NULL DEFAULT 0,
		cooldown_until TEXT,
		last_error TEXT NOT NULL DEFAULT '',
		last_status_code INTEGER,
		success_count INTEGER NOT NULL DEFAULT 0,
		failure_count INTEGER NOT NULL DEFAULT 0,
		last_used_at TEXT
	);

	CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT NOT NULL,
		inbound_protocol TEXT NOT NULL,
		provider_id TEXT,
		key_id TEXT,
		model TEXT,
		status INTEGER NOT NULL,
		latency_ms INTEGER NOT NULL,
		prompt_tokens INTEGER,
		completion_tokens INTEGER,
		total_tokens INTEGER,
		error_type TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS gateway_keys (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL UNIQUE,
		key_hint TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		request_count INTEGER NOT NULL DEFAULT 0,
		last_used_at TEXT,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS balances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id TEXT NOT NULL,
		key_id TEXT NOT NULL DEFAULT '',
		balance REAL,
		currency TEXT NOT NULL DEFAULT '',
		quota_used REAL,
		quota_limit REAL,
		source TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'unknown',
		error TEXT NOT NULL DEFAULT '',
		refreshed_at TEXT,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(provider_id,key_id)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_keys_provider_id ON keys(provider_id);
	CREATE INDEX IF NOT EXISTS idx_key_state_cooldown_until ON key_state(cooldown_until);
	CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
	CREATE INDEX IF NOT EXISTS idx_request_logs_provider_created_at ON request_logs(provider_id,created_at);
	CREATE INDEX IF NOT EXISTS idx_request_logs_key_created_at ON request_logs(key_id,created_at);
	CREATE INDEX IF NOT EXISTS idx_balances_updated_at ON balances(updated_at);`,
	`CREATE TABLE balances_v3 (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
		key_id TEXT NOT NULL DEFAULT '',
		balance REAL,
		currency TEXT NOT NULL DEFAULT '',
		quota_used REAL,
		quota_limit REAL,
		source TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'unknown',
		error TEXT NOT NULL DEFAULT '',
		refreshed_at TEXT,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(provider_id,key_id)
	);
	INSERT INTO balances_v3 (id,provider_id,key_id,balance,currency,quota_used,quota_limit,source,status,error,refreshed_at,created_at,updated_at)
	SELECT b.id,b.provider_id,b.key_id,b.balance,b.currency,b.quota_used,b.quota_limit,b.source,b.status,b.error,b.refreshed_at,b.created_at,b.updated_at
	FROM balances b JOIN providers p ON p.id=b.provider_id;
	DROP TABLE balances;
	ALTER TABLE balances_v3 RENAME TO balances;
	CREATE INDEX IF NOT EXISTS idx_balances_updated_at ON balances(updated_at);`,
	`CREATE TABLE IF NOT EXISTS provider_models (
		provider_id TEXT NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
		model_id TEXT NOT NULL,
		discovered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(provider_id,model_id)
	);
	CREATE INDEX IF NOT EXISTS idx_provider_models_discovered_at ON provider_models(discovered_at);`,
	`CREATE TABLE IF NOT EXISTS request_metrics_daily (
		day TEXT PRIMARY KEY,
		request_count INTEGER NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS upstream_affinity (
		resource_id TEXT PRIMARY KEY,
		provider_id TEXT NOT NULL,
		key_id TEXT NOT NULL REFERENCES keys(id) ON DELETE CASCADE,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_upstream_affinity_expires_at ON upstream_affinity(expires_at);`,
}

func Open(path, secretPath string) (*Store, error) {
	return OpenWithOptions(path, secretPath, Options{LogRetentionDays: 30, LogMaxEntries: 100000, Timezone: "Asia/Singapore", BackupBeforeMigration: true, BackupRetention: 5})
}

func OpenWithOptions(path, secretPath string, options Options) (*Store, error) {
	if strings.TrimSpace(options.Timezone) == "" {
		options.Timezone = "Asia/Singapore"
	}
	location, err := time.LoadLocation(options.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load storage timezone: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := recoverRestoreTransaction(path, secretPath); err != nil {
		return nil, fmt.Errorf("recover interrupted portable restore: %w", err)
	}
	c, err := newCryptor(secretPath)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(1)")
	dsn := path + "?" + query.Encode()
	databaseExisted := false
	if info, statErr := os.Stat(path); statErr == nil && info.Size() > 0 {
		databaseExisted = true
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("stat database: %w", statErr)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	s := &Store{
		db:                    db,
		crypto:                c,
		options:               options,
		location:              location,
		path:                  path,
		secretPath:            secretPath,
		secretCache:           make(map[string]cachedSecret),
		gatewayKeyCache:       make(map[string]string),
		pendingGatewayUsage:   make(map[string]gatewayUsage),
		pendingKeySuccesses:   make(map[string]keySuccess),
		pendingRequestLogs:    make([]model.RequestLog, 0, requestLogBatchSize),
		pendingRequestMetrics: make(map[string]dailyRequestMetric),
		gatewayUsageStop:      make(chan struct{}),
	}
	if err := s.migrate(context.Background(), databaseExisted); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := s.PruneRequestLogs(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	s.gatewayUsageWG.Add(1)
	go s.flushGatewayUsageLoop()
	return s, nil
}

func (s *Store) Close() error {
	s.closeOnce.Do(func() {
		close(s.gatewayUsageStop)
		s.gatewayUsageWG.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.closeErr = errors.Join(s.flushGatewayUsage(ctx), s.flushKeySuccesses(ctx), s.flushRequestLogs(ctx), s.flushRequestMetrics(ctx), s.Checkpoint(ctx), s.db.Close())
	})
	return s.closeErr
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Store) migrate(ctx context.Context, databaseExisted bool) error {
	if _, err := s.db.ExecContext(ctx, `
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;
		PRAGMA busy_timeout=5000;
	`); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			checksum TEXT NOT NULL DEFAULT '',
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return err
	}
	if err := ensureMigrationChecksumColumn(ctx, s.db); err != nil {
		return err
	}
	applied, err := loadAppliedMigrations(ctx, s.db)
	if err != nil {
		return err
	}
	for version := range applied {
		if version > len(schemaMigrations) {
			return fmt.Errorf("database schema version %d is newer than this gateway supports", version)
		}
	}
	for i, migration := range schemaMigrations {
		version := i + 1
		checksum := migrationChecksum(migration)
		if recorded, ok := applied[version]; ok {
			if recorded != "" && recorded != checksum {
				return fmt.Errorf("schema migration %d checksum mismatch", version)
			}
			if recorded == "" {
				if _, err := s.db.ExecContext(ctx, `UPDATE schema_migrations SET checksum=? WHERE version=?`, checksum, version); err != nil {
					return fmt.Errorf("record schema migration %d checksum: %w", version, err)
				}
			}
		}
	}
	if databaseExisted && len(applied) < len(schemaMigrations) && s.options.BackupBeforeMigration {
		if _, err := s.CreateBackup(ctx, "pre-migration"); err != nil {
			return fmt.Errorf("backup database before migration: %w", err)
		}
	}
	for i, migration := range schemaMigrations {
		version := i + 1
		if _, ok := applied[version]; ok {
			continue
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migration); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply schema migration %d: %w", version, err)
		}
		if version == 5 {
			if err := backfillRequestMetricsMigration(ctx, tx, s.location); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("backfill request metrics migration: %w", err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version,checksum) VALUES (?,?)`, version, migrationChecksum(migration)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record schema migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit schema migration %d: %w", version, err)
		}
	}
	return nil
}

func backfillRequestMetricsMigration(ctx context.Context, tx *sql.Tx, location *time.Location) error {
	rows, err := tx.QueryContext(ctx, `SELECT total_tokens,created_at FROM request_logs`)
	if err != nil {
		return err
	}
	metrics := make(map[string]dailyRequestMetric)
	for rows.Next() {
		var tokens sql.NullInt64
		var created string
		if err := rows.Scan(&tokens, &created); err != nil {
			_ = rows.Close()
			return err
		}
		day := parseTime(created).In(location).Format("2006-01-02")
		metric := metrics[day]
		metric.requests++
		if tokens.Valid {
			metric.tokens += int(tokens.Int64)
		}
		metrics[day] = metric
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for day, metric := range metrics {
		if _, err := tx.ExecContext(ctx, `INSERT INTO request_metrics_daily (day,request_count,total_tokens) VALUES (?,?,?)`, day, metric.requests, metric.tokens); err != nil {
			return err
		}
	}
	return nil
}

func ensureMigrationChecksumColumn(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(schema_migrations)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == "checksum" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = db.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN checksum TEXT NOT NULL DEFAULT ''`)
	return err
}

func loadAppliedMigrations(ctx context.Context, db *sql.DB) (map[int]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT version,checksum FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int]string)
	for rows.Next() {
		var version int
		var checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		out[version] = checksum
	}
	return out, rows.Err()
}

func migrationChecksum(migration string) string {
	sum := sha256.Sum256([]byte(migration))
	return hex.EncodeToString(sum[:])
}

func (s *Store) ListProviders(ctx context.Context) ([]model.Provider, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,type,base_url,priority,enabled,model_map,balance_path,created_at,updated_at FROM providers ORDER BY priority DESC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Provider, 0)
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProvider(ctx context.Context, id string) (*model.Provider, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,name,type,base_url,priority,enabled,model_map,balance_path,created_at,updated_at FROM providers WHERE id=?`, id)
	p, err := scanProvider(row)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	return &p, err
}

func (s *Store) ListGatewayKeys(ctx context.Context) ([]model.GatewayKey, error) {
	if err := s.flushGatewayUsage(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,key_hint,enabled,request_count,last_used_at,created_at,updated_at FROM gateway_keys ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.GatewayKey, 0)
	for rows.Next() {
		item, err := scanGatewayKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) ListBalances(ctx context.Context) ([]model.Balance, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,provider_id,key_id,balance,currency,quota_used,quota_limit,source,status,error,refreshed_at,created_at,updated_at FROM balances ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Balance, 0)
	for rows.Next() {
		item, err := scanBalance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) UpsertBalance(ctx context.Context, b model.Balance) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO balances (provider_id,key_id,balance,currency,quota_used,quota_limit,source,status,error,refreshed_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(provider_id,key_id) DO UPDATE SET
			balance=excluded.balance,currency=excluded.currency,quota_used=excluded.quota_used,quota_limit=excluded.quota_limit,
			source=excluded.source,status=excluded.status,error=excluded.error,refreshed_at=excluded.refreshed_at,updated_at=CURRENT_TIMESTAMP
	`, b.ProviderID, b.KeyID, floatNil(b.Balance), b.Currency, floatNil(b.QuotaUsed), floatNil(b.QuotaLimit), b.Source, b.Status, b.Error, now)
	return err
}

func (s *Store) CreateGatewayKey(ctx context.Context, name string) (model.GatewayKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "gateway-key"
	}
	plain, err := randomGatewayKey()
	if err != nil {
		return model.GatewayKey{}, err
	}
	item := model.GatewayKey{
		ID:        slug(name),
		Name:      name,
		KeyHint:   MaskSecret(plain),
		Enabled:   true,
		Plaintext: plain,
	}
	hash := hashGatewayKey(plain)
	base := item.ID
	var insertErr error
	for attempt := 0; attempt < 6; attempt++ {
		if attempt > 0 {
			suffix, err := randomIDSuffix()
			if err != nil {
				return model.GatewayKey{}, err
			}
			item.ID = base + "-" + suffix
		}
		_, insertErr = s.db.ExecContext(ctx, `INSERT INTO gateway_keys (id,name,key_hash,key_hint,enabled) VALUES (?,?,?,?,1)`, item.ID, item.Name, hash, item.KeyHint)
		if insertErr == nil {
			break
		}
		// Only an ID collision is worth retrying with a fresh suffix; any other
		// error (disk full, closed connection) is real and should surface at once.
		if !isConstraintViolation(insertErr) {
			return model.GatewayKey{}, insertErr
		}
	}
	if insertErr != nil {
		return model.GatewayKey{}, insertErr
	}
	s.clearGatewayKeyCache()
	row := s.db.QueryRowContext(ctx, `SELECT id,name,key_hint,enabled,request_count,last_used_at,created_at,updated_at FROM gateway_keys WHERE id=?`, item.ID)
	got, err := scanGatewayKey(row)
	if err != nil {
		return model.GatewayKey{}, err
	}
	got.Plaintext = plain
	return got, nil
}

func (s *Store) PatchGatewayKey(ctx context.Context, id string, name *string, enabled *bool) error {
	current := id
	if strings.TrimSpace(current) == "" {
		return fmt.Errorf("gateway key id is required")
	}
	if ok, err := s.gatewayKeyExists(ctx, current); err != nil {
		return err
	} else if !ok {
		return sql.ErrNoRows
	}
	if name != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE gateway_keys SET name=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, strings.TrimSpace(*name), current); err != nil {
			return err
		}
	}
	if enabled != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE gateway_keys SET enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, boolInt(*enabled), current); err != nil {
			return err
		}
	}
	s.clearGatewayKeyCache()
	return nil
}

func (s *Store) gatewayKeyExists(ctx context.Context, id string) (bool, error) {
	var found int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM gateway_keys WHERE id=?`, id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) RotateGatewayKey(ctx context.Context, id string) (model.GatewayKey, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.GatewayKey{}, fmt.Errorf("gateway key id is required")
	}
	plain, err := randomGatewayKey()
	if err != nil {
		return model.GatewayKey{}, err
	}
	hash := hashGatewayKey(plain)
	hint := MaskSecret(plain)
	res, err := s.db.ExecContext(ctx, `UPDATE gateway_keys SET key_hash=?, key_hint=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, hash, hint, id)
	if err != nil {
		return model.GatewayKey{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return model.GatewayKey{}, err
	}
	if affected == 0 {
		return model.GatewayKey{}, sql.ErrNoRows
	}
	s.clearGatewayKeyCache()
	row := s.db.QueryRowContext(ctx, `SELECT id,name,key_hint,enabled,request_count,last_used_at,created_at,updated_at FROM gateway_keys WHERE id=?`, id)
	item, err := scanGatewayKey(row)
	if err != nil {
		return model.GatewayKey{}, err
	}
	item.Plaintext = plain
	return item, nil
}

func (s *Store) DeleteGatewayKey(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM gateway_keys WHERE id=?`, id)
	if err := requireRowsAffected(res, err); err != nil {
		return err
	}
	s.clearGatewayKeyCache()
	return nil
}

func (s *Store) VerifyGatewayKey(ctx context.Context, plain string) (bool, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return false, nil
	}
	hash := hashGatewayKey(plain)
	s.gatewayKeyCacheMu.RLock()
	id, cached := s.gatewayKeyCache[hash]
	s.gatewayKeyCacheMu.RUnlock()
	if cached {
		s.recordGatewayUsage(id)
		return true, nil
	}
	err := s.db.QueryRowContext(ctx, `SELECT id FROM gateway_keys WHERE key_hash=? AND enabled=1`, hash).Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	s.gatewayKeyCacheMu.Lock()
	s.gatewayKeyCache[hash] = id
	s.gatewayKeyCacheMu.Unlock()
	s.recordGatewayUsage(id)
	return true, nil
}

func (s *Store) clearGatewayKeyCache() {
	s.gatewayKeyCacheMu.Lock()
	s.gatewayKeyCache = make(map[string]string)
	s.gatewayKeyCacheMu.Unlock()
}

func (s *Store) recordGatewayUsage(id string) {
	s.gatewayUsageMu.Lock()
	usage := s.pendingGatewayUsage[id]
	usage.count++
	usage.lastUsed = time.Now().UTC()
	s.pendingGatewayUsage[id] = usage
	s.gatewayUsageMu.Unlock()
}

func (s *Store) flushGatewayUsageLoop() {
	defer s.gatewayUsageWG.Done()
	ticker := time.NewTicker(gatewayUsageFlushInterval)
	defer ticker.Stop()
	logTicker := time.NewTicker(requestLogFlushInterval)
	defer logTicker.Stop()
	maintenance := time.NewTicker(maintenanceInterval)
	defer maintenance.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), gatewayUsageFlushInterval)
			if err := errors.Join(s.flushGatewayUsage(ctx), s.flushKeySuccesses(ctx)); err != nil {
				slog.Warn("flush usage counters failed", "error", err)
			}
			cancel()
		case <-logTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), gatewayUsageFlushInterval)
			if err := s.flushRequestLogs(ctx); err != nil {
				slog.Warn("flush request logs failed", "error", err)
			}
			cancel()
		case <-maintenance.C:
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			if _, err := s.PruneRequestLogs(ctx); err != nil {
				slog.Warn("prune request logs failed", "error", err)
			}
			if err := s.Checkpoint(ctx); err != nil {
				slog.Warn("checkpoint database failed", "error", err)
			}
			cancel()
		case <-s.gatewayUsageStop:
			return
		}
	}
}

func (s *Store) flushGatewayUsage(ctx context.Context) error {
	s.gatewayUsageFlushMu.Lock()
	defer s.gatewayUsageFlushMu.Unlock()

	s.gatewayUsageMu.Lock()
	if len(s.pendingGatewayUsage) == 0 {
		s.gatewayUsageMu.Unlock()
		return nil
	}
	batch := s.pendingGatewayUsage
	s.pendingGatewayUsage = make(map[string]gatewayUsage)
	s.gatewayUsageMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.restoreGatewayUsage(batch)
		return err
	}
	for id, usage := range batch {
		if _, err := tx.ExecContext(ctx, `
			UPDATE gateway_keys
			SET request_count=request_count+?,last_used_at=?
			WHERE id=?
		`, usage.count, usage.lastUsed.Format(time.RFC3339), id); err != nil {
			_ = tx.Rollback()
			s.restoreGatewayUsage(batch)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		s.restoreGatewayUsage(batch)
		return err
	}
	return nil
}

func (s *Store) restoreGatewayUsage(batch map[string]gatewayUsage) {
	s.gatewayUsageMu.Lock()
	defer s.gatewayUsageMu.Unlock()
	for id, usage := range batch {
		pending := s.pendingGatewayUsage[id]
		pending.count += usage.count
		if usage.lastUsed.After(pending.lastUsed) {
			pending.lastUsed = usage.lastUsed
		}
		s.pendingGatewayUsage[id] = pending
	}
}

func (s *Store) flushKeySuccesses(ctx context.Context) error {
	s.keySuccessFlushMu.Lock()
	defer s.keySuccessFlushMu.Unlock()

	s.keySuccessMu.Lock()
	if len(s.pendingKeySuccesses) == 0 {
		s.keySuccessMu.Unlock()
		return nil
	}
	batch := s.pendingKeySuccesses
	s.pendingKeySuccesses = make(map[string]keySuccess)
	s.keySuccessMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.restoreKeySuccesses(batch)
		return err
	}
	for id, success := range batch {
		if success.resetHealth {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO key_state (key_id,consecutive_failures,cooldown_until,last_error,last_status_code,success_count,last_used_at)
				VALUES (?,0,NULL,'',NULL,?,?)
				ON CONFLICT(key_id) DO UPDATE SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL,success_count=success_count+excluded.success_count,last_used_at=excluded.last_used_at
			`, id, success.count, success.lastUsed.Format(time.RFC3339))
		} else {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO key_state (key_id,success_count,last_used_at)
				VALUES (?,?,?)
				ON CONFLICT(key_id) DO UPDATE SET success_count=success_count+excluded.success_count,last_used_at=excluded.last_used_at
			`, id, success.count, success.lastUsed.Format(time.RFC3339))
		}
		if err != nil {
			_ = tx.Rollback()
			s.restoreKeySuccesses(batch)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		s.restoreKeySuccesses(batch)
		return err
	}
	return nil
}

func (s *Store) restoreKeySuccesses(batch map[string]keySuccess) {
	s.keySuccessMu.Lock()
	defer s.keySuccessMu.Unlock()
	for id, success := range batch {
		pending := s.pendingKeySuccesses[id]
		pending.count += success.count
		pending.resetHealth = pending.resetHealth || success.resetHealth
		if success.lastUsed.After(pending.lastUsed) {
			pending.lastUsed = success.lastUsed
		}
		s.pendingKeySuccesses[id] = pending
	}
}

func (s *Store) UpsertProvider(ctx context.Context, p model.Provider) (model.Provider, error) {
	if p.ID == "" {
		p.ID = slug(p.Name)
	}
	if p.Name == "" {
		p.Name = p.ID
	}
	if p.Type == "" {
		p.Type = model.ProviderOpenAICompatible
	}
	p.BaseURL = strings.TrimRight(p.BaseURL, "/")
	b, _ := json.Marshal(p.ModelMap)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO providers (id,name,type,base_url,priority,enabled,model_map,balance_path,updated_at)
		VALUES (?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,type=excluded.type,base_url=excluded.base_url,priority=excluded.priority,
			enabled=excluded.enabled,model_map=excluded.model_map,balance_path=excluded.balance_path,updated_at=CURRENT_TIMESTAMP
	`, p.ID, p.Name, p.Type, p.BaseURL, p.Priority, boolInt(p.Enabled), string(b), p.BalancePath)
	if err != nil {
		return p, err
	}
	got, err := s.GetProvider(ctx, p.ID)
	if err != nil {
		return p, err
	}
	return *got, nil
}

func (s *Store) DeleteProvider(ctx context.Context, id string) error {
	if err := s.flushKeySuccesses(ctx); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM providers WHERE id=?`, id)
	return requireRowsAffected(res, err)
}

func (s *Store) ListKeys(ctx context.Context) ([]model.Key, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT k.id,k.provider_id,p.name,p.type,p.base_url,p.priority,p.enabled,p.model_map,p.balance_path,
			k.name,k.secret_cipher,k.priority,k.enabled,k.manual_preferred,
			COALESCE(st.consecutive_failures,0),st.cooldown_until,COALESCE(st.last_error,''),st.last_status_code,
			COALESCE(st.success_count,0),COALESCE(st.failure_count,0),st.last_used_at
		FROM keys k
		JOIN providers p ON p.id=k.provider_id
		LEFT JOIN key_state st ON st.key_id=k.id
		ORDER BY k.manual_preferred DESC,p.priority DESC,k.priority DESC,k.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.Key, 0)
	for rows.Next() {
		k, err := s.scanKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) UpsertKey(ctx context.Context, k model.Key) (model.Key, error) {
	if k.ID == "" {
		k.ID = slug(k.ProviderID + "-" + k.Name)
	}
	if k.Name == "" {
		k.Name = k.ID
	}
	if strings.TrimSpace(k.Secret) == "" {
		if existing, ok, err := s.getKeySecret(ctx, k.ID); err != nil {
			return k, err
		} else if ok {
			k.Secret = existing
		}
	}
	cipher, err := s.crypto.encrypt(k.Secret)
	if err != nil {
		return k, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return k, err
	}
	defer func() { _ = tx.Rollback() }()
	if k.ManualPreferred {
		if _, err := tx.ExecContext(ctx, `UPDATE keys SET manual_preferred=0 WHERE manual_preferred=1 AND id<>?`, k.ID); err != nil {
			return k, err
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO keys (id,provider_id,name,secret_cipher,priority,enabled,manual_preferred,updated_at)
		VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			provider_id=excluded.provider_id,name=excluded.name,secret_cipher=excluded.secret_cipher,
			priority=excluded.priority,enabled=excluded.enabled,manual_preferred=excluded.manual_preferred,updated_at=CURRENT_TIMESTAMP
	`, k.ID, k.ProviderID, k.Name, cipher, k.Priority, boolInt(k.Enabled), boolInt(k.ManualPreferred))
	if err != nil {
		return k, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO key_state (key_id) VALUES (?)`, k.ID); err != nil {
		return k, err
	}
	if err := tx.Commit(); err != nil {
		return k, err
	}
	keys, err := s.ListKeys(ctx)
	if err != nil {
		return k, err
	}
	for _, item := range keys {
		if item.ID == k.ID {
			return item, nil
		}
	}
	return k, nil
}

func (s *Store) getKeySecret(ctx context.Context, id string) (string, bool, error) {
	var secretCipher string
	err := s.db.QueryRowContext(ctx, `SELECT secret_cipher FROM keys WHERE id=?`, id).Scan(&secretCipher)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	secret, err := s.crypto.decrypt(secretCipher)
	if err != nil {
		return "", false, err
	}
	return secret, true, nil
}

func (s *Store) DeleteKey(ctx context.Context, id string) error {
	if err := s.flushKeySuccesses(ctx); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM keys WHERE id=?`, id)
	if err := requireRowsAffected(res, err); err != nil {
		return err
	}
	s.secretCacheMu.Lock()
	delete(s.secretCache, id)
	s.secretCacheMu.Unlock()
	return nil
}

func (s *Store) PreferKey(ctx context.Context, id string) error {
	if ok, err := s.keyExists(ctx, id); err != nil {
		return err
	} else if !ok {
		return sql.ErrNoRows
	}
	_, err := s.db.ExecContext(ctx, `UPDATE keys SET manual_preferred=CASE WHEN id=? THEN 1 ELSE 0 END`, id)
	return err
}

func (s *Store) ResetKey(ctx context.Context, id string) error {
	if ok, err := s.keyExists(ctx, id); err != nil {
		return err
	} else if !ok {
		return sql.ErrNoRows
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO key_state (key_id,consecutive_failures,cooldown_until,last_error,last_status_code)
		VALUES (?,0,NULL,'',NULL)
		ON CONFLICT(key_id) DO UPDATE SET consecutive_failures=0,cooldown_until=NULL,last_error='',last_status_code=NULL
	`, id)
	return err
}

func (s *Store) keyExists(ctx context.Context, id string) (bool, error) {
	var found int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM keys WHERE id=?`, id).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func requireRowsAffected(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) RecordSuccess(ctx context.Context, keyID string, resetHealth bool) error {
	s.keySuccessMu.Lock()
	pending := s.pendingKeySuccesses[keyID]
	pending.count++
	pending.lastUsed = time.Now().UTC()
	pending.resetHealth = pending.resetHealth || resetHealth
	s.pendingKeySuccesses[keyID] = pending
	s.keySuccessMu.Unlock()
	if resetHealth {
		return s.flushKeySuccesses(ctx)
	}
	return nil
}

func (s *Store) RecordFailure(ctx context.Context, keyID string, status *int, message string, policy FailurePolicy) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var failures int
	err = tx.QueryRowContext(ctx, `
		INSERT INTO key_state (key_id,consecutive_failures,last_error,last_status_code,failure_count,last_used_at)
		VALUES (?,1,?,?,1,CURRENT_TIMESTAMP)
		ON CONFLICT(key_id) DO UPDATE SET consecutive_failures=consecutive_failures+1,last_error=excluded.last_error,last_status_code=excluded.last_status_code,failure_count=failure_count+1,last_used_at=CURRENT_TIMESTAMP
		RETURNING consecutive_failures
	`, keyID, message, status).Scan(&failures)
	if err != nil {
		return err
	}

	if policy.ForceCooldown || failures >= policy.Threshold {
		duration := policy.Cooldown
		if failures == policy.Threshold && policy.ThresholdCooldown > 0 {
			duration = policy.ThresholdCooldown
		}
		if policy.OverrideCooldown > 0 {
			duration = policy.OverrideCooldown
		}
		until := time.Now().Add(duration).UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `UPDATE key_state SET cooldown_until=? WHERE key_id=?`, until, keyID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) LogRequest(ctx context.Context, l model.RequestLog) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}
	day := l.CreatedAt.In(s.location).Format("2006-01-02")
	tokens := 0
	if l.TotalTokens != nil {
		tokens = *l.TotalTokens
	}
	s.requestMetricsMu.Lock()
	metric := s.pendingRequestMetrics[day]
	metric.requests++
	metric.tokens += tokens
	s.pendingRequestMetrics[day] = metric
	flushMetrics := metric.requests >= requestLogBatchSize
	s.requestMetricsMu.Unlock()
	if s.options.DisableRequestLogging {
		if flushMetrics {
			return s.flushRequestMetrics(ctx)
		}
		return nil
	}
	s.requestLogMu.Lock()
	s.pendingRequestLogs = append(s.pendingRequestLogs, l)
	dropped := s.trimPendingRequestLogsLocked()
	flush := len(s.pendingRequestLogs) == requestLogBatchSize
	s.requestLogMu.Unlock()
	s.noteDroppedRequestLogs(dropped)
	if flush {
		return s.flushRequestLogs(ctx)
	}
	return nil
}

func (s *Store) flushRequestLogs(ctx context.Context) error {
	if err := s.flushRequestMetrics(ctx); err != nil {
		return err
	}
	s.requestLogFlushMu.Lock()
	defer s.requestLogFlushMu.Unlock()

	s.requestLogMu.Lock()
	if len(s.pendingRequestLogs) == 0 {
		s.requestLogMu.Unlock()
		return nil
	}
	batch := s.pendingRequestLogs
	s.pendingRequestLogs = make([]model.RequestLog, 0, requestLogBatchSize)
	s.requestLogMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.restoreRequestLogs(batch)
		return err
	}
	for _, l := range batch {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO request_logs (request_id,inbound_protocol,provider_id,key_id,model,status,latency_ms,prompt_tokens,completion_tokens,total_tokens,error_type,created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		`, l.RequestID, l.InboundProtocol, emptyNil(l.ProviderID), emptyNil(l.KeyID), emptyNil(l.Model), l.Status, l.LatencyMS, l.PromptTokens, l.CompletionTokens, l.TotalTokens, l.ErrorType, l.CreatedAt.UTC().Format("2006-01-02 15:04:05")); err != nil {
			_ = tx.Rollback()
			s.restoreRequestLogs(batch)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		s.restoreRequestLogs(batch)
		return err
	}
	return nil
}

func (s *Store) flushRequestMetrics(ctx context.Context) error {
	s.requestMetricsFlushMu.Lock()
	defer s.requestMetricsFlushMu.Unlock()

	s.requestMetricsMu.Lock()
	if len(s.pendingRequestMetrics) == 0 {
		s.requestMetricsMu.Unlock()
		return nil
	}
	batch := s.pendingRequestMetrics
	s.pendingRequestMetrics = make(map[string]dailyRequestMetric)
	s.requestMetricsMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.restoreRequestMetrics(batch)
		return err
	}
	for day, metric := range batch {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO request_metrics_daily (day,request_count,total_tokens,updated_at)
			VALUES (?,?,?,CURRENT_TIMESTAMP)
			ON CONFLICT(day) DO UPDATE SET
				request_count=request_count+excluded.request_count,
				total_tokens=total_tokens+excluded.total_tokens,
				updated_at=CURRENT_TIMESTAMP
		`, day, metric.requests, metric.tokens); err != nil {
			_ = tx.Rollback()
			s.restoreRequestMetrics(batch)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		s.restoreRequestMetrics(batch)
		return err
	}
	return nil
}

func (s *Store) restoreRequestMetrics(batch map[string]dailyRequestMetric) {
	s.requestMetricsMu.Lock()
	defer s.requestMetricsMu.Unlock()
	for day, metric := range batch {
		pending := s.pendingRequestMetrics[day]
		pending.requests += metric.requests
		pending.tokens += metric.tokens
		s.pendingRequestMetrics[day] = pending
	}
}

func (s *Store) restoreRequestLogs(batch []model.RequestLog) {
	s.requestLogMu.Lock()
	restored := make([]model.RequestLog, 0, len(batch)+len(s.pendingRequestLogs))
	restored = append(restored, batch...)
	restored = append(restored, s.pendingRequestLogs...)
	s.pendingRequestLogs = restored
	dropped := s.trimPendingRequestLogsLocked()
	s.requestLogMu.Unlock()
	s.noteDroppedRequestLogs(dropped)
}

func (s *Store) trimPendingRequestLogsLocked() int {
	if len(s.pendingRequestLogs) <= maxPendingRequestLogs {
		return 0
	}
	dropped := len(s.pendingRequestLogs) - maxPendingRequestLogs
	copy(s.pendingRequestLogs, s.pendingRequestLogs[dropped:])
	clear(s.pendingRequestLogs[maxPendingRequestLogs:])
	s.pendingRequestLogs = s.pendingRequestLogs[:maxPendingRequestLogs]
	return dropped
}

func (s *Store) noteDroppedRequestLogs(dropped int) {
	if dropped == 0 {
		return
	}
	count := uint64(dropped)
	total := s.droppedRequestLogs.Add(count)
	if total == count || total/1000 != (total-count)/1000 {
		slog.Warn("request log buffer full; oldest entries dropped", "dropped_total", total, "buffer_limit", maxPendingRequestLogs)
	}
}

func (s *Store) PruneRequestLogs(ctx context.Context) (int64, error) {
	if err := s.flushRequestLogs(ctx); err != nil {
		return 0, err
	}
	var deleted int64
	if s.options.LogRetentionDays > 0 {
		modifier := fmt.Sprintf("-%d days", s.options.LogRetentionDays)
		res, err := s.db.ExecContext(ctx, `DELETE FROM request_logs WHERE created_at<datetime('now',?)`, modifier)
		if err != nil {
			return deleted, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return deleted, err
		}
		deleted += count
	}
	if s.options.LogMaxEntries > 0 {
		res, err := s.db.ExecContext(ctx, `
			DELETE FROM request_logs
			WHERE id<COALESCE((
				SELECT id FROM request_logs ORDER BY id DESC LIMIT 1 OFFSET ?
			),0)
		`, s.options.LogMaxEntries-1)
		if err != nil {
			return deleted, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			return deleted, err
		}
		deleted += count
	}
	return deleted, nil
}

func (s *Store) ListLogs(ctx context.Context, limit int) ([]model.RequestLog, error) {
	if err := s.flushRequestLogs(ctx); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,request_id,inbound_protocol,provider_id,key_id,model,status,latency_ms,prompt_tokens,completion_tokens,total_tokens,error_type,created_at FROM request_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.RequestLog, 0)
	for rows.Next() {
		var l model.RequestLog
		var providerID, keyID, mod sql.NullString
		var pt, ct, tt sql.NullInt64
		var created string
		if err := rows.Scan(&l.ID, &l.RequestID, &l.InboundProtocol, &providerID, &keyID, &mod, &l.Status, &l.LatencyMS, &pt, &ct, &tt, &l.ErrorType, &created); err != nil {
			return nil, err
		}
		l.ProviderID = providerID.String
		l.KeyID = keyID.String
		l.Model = mod.String
		l.PromptTokens = intPtr(pt)
		l.CompletionTokens = intPtr(ct)
		l.TotalTokens = intPtr(tt)
		l.CreatedAt = parseTime(created)
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) Stats(ctx context.Context) (model.Stats, error) {
	st := model.Stats{DroppedRequestLogs: s.droppedRequestLogs.Load()}
	if err := s.flushRequestLogs(ctx); err != nil {
		return st, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM keys`).Scan(&st.TotalKeys); err != nil {
		return st, err
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM keys k
		JOIN providers p ON p.id=k.provider_id
		LEFT JOIN key_state state ON state.key_id=k.id
		WHERE k.enabled=1 AND p.enabled=1
			AND (state.cooldown_until IS NULL OR datetime(state.cooldown_until)<=datetime('now'))
	`).Scan(&st.ActiveKeys); err != nil {
		return st, err
	}
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM keys k
		JOIN providers p ON p.id=k.provider_id
		JOIN key_state state ON state.key_id=k.id
		WHERE k.enabled=1 AND p.enabled=1
			AND state.cooldown_until IS NOT NULL AND datetime(state.cooldown_until)>datetime('now')
	`).Scan(&st.FailedKeys); err != nil {
		return st, err
	}
	now := time.Now().In(s.location)
	day := now.Format("2006-01-02")
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(request_count,0),COALESCE(total_tokens,0)
		FROM request_metrics_daily WHERE day=?
	`, day).Scan(&st.TodayRequests, &st.TodayTokens); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return st, err
	}
	recent, err := s.ListLogs(ctx, 10)
	if err != nil {
		return st, err
	}
	st.Recent = recent
	return st, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProvider(row rowScanner) (model.Provider, error) {
	var p model.Provider
	var modelMap, created, updated string
	var enabled int
	if err := row.Scan(&p.ID, &p.Name, &p.Type, &p.BaseURL, &p.Priority, &enabled, &modelMap, &p.BalancePath, &created, &updated); err != nil {
		return p, err
	}
	p.Enabled = enabled == 1
	p.ModelMap = map[string]string{}
	if err := json.Unmarshal([]byte(modelMap), &p.ModelMap); err != nil {
		return p, fmt.Errorf("decode provider %s model map: %w", p.ID, err)
	}
	p.CreatedAt = parseTime(created)
	p.UpdatedAt = parseTime(updated)
	return p, nil
}

func scanGatewayKey(row rowScanner) (model.GatewayKey, error) {
	var k model.GatewayKey
	var enabled int
	var lastUsed sql.NullString
	var created, updated string
	if err := row.Scan(&k.ID, &k.Name, &k.KeyHint, &enabled, &k.RequestCount, &lastUsed, &created, &updated); err != nil {
		return k, err
	}
	k.Enabled = enabled == 1
	if lastUsed.Valid {
		t := parseTime(lastUsed.String)
		k.LastUsedAt = &t
	}
	k.CreatedAt = parseTime(created)
	k.UpdatedAt = parseTime(updated)
	return k, nil
}

func scanBalance(row rowScanner) (model.Balance, error) {
	var b model.Balance
	var balance, used, limit sql.NullFloat64
	var refreshed sql.NullString
	var created, updated string
	if err := row.Scan(&b.ID, &b.ProviderID, &b.KeyID, &balance, &b.Currency, &used, &limit, &b.Source, &b.Status, &b.Error, &refreshed, &created, &updated); err != nil {
		return b, err
	}
	b.Balance = floatPtr(balance)
	b.QuotaUsed = floatPtr(used)
	b.QuotaLimit = floatPtr(limit)
	if refreshed.Valid {
		t := parseTime(refreshed.String)
		b.RefreshedAt = &t
	}
	b.CreatedAt = parseTime(created)
	b.UpdatedAt = parseTime(updated)
	return b, nil
}

func (s *Store) scanKey(row rowScanner) (model.Key, error) {
	var k model.Key
	var secretCipher, modelMap string
	var pEnabled, enabled, preferred int
	var cooldown, lastUsed sql.NullString
	var lastStatus sql.NullInt64
	if err := row.Scan(&k.ID, &k.ProviderID, &k.ProviderName, &k.ProviderType, &k.ProviderBaseURL, &k.ProviderPriority, &pEnabled, &modelMap, &k.ProviderBalancePath,
		&k.Name, &secretCipher, &k.Priority, &enabled, &preferred,
		&k.ConsecutiveFailures, &cooldown, &k.LastError, &lastStatus, &k.SuccessCount, &k.FailureCount, &lastUsed); err != nil {
		return k, err
	}
	secret, err := s.decryptKeySecret(k.ID, secretCipher)
	if err != nil {
		return k, err
	}
	k.Secret = secret
	k.KeyHint = MaskSecret(secret)
	k.ProviderEnabled = pEnabled == 1
	k.Enabled = enabled == 1
	k.ManualPreferred = preferred == 1
	k.ProviderModelMap = map[string]string{}
	if err := json.Unmarshal([]byte(modelMap), &k.ProviderModelMap); err != nil {
		return k, fmt.Errorf("decode provider %s model map: %w", k.ProviderID, err)
	}
	if cooldown.Valid {
		t := parseTime(cooldown.String)
		k.CooldownUntil = &t
	}
	if lastUsed.Valid {
		t := parseTime(lastUsed.String)
		k.LastUsedAt = &t
	}
	if lastStatus.Valid {
		v := int(lastStatus.Int64)
		k.LastStatusCode = &v
	}
	return k, nil
}

func (s *Store) decryptKeySecret(id, cipher string) (string, error) {
	s.secretCacheMu.RLock()
	cached, ok := s.secretCache[id]
	s.secretCacheMu.RUnlock()
	if ok && cached.cipher == cipher {
		return cached.secret, nil
	}
	secret, err := s.crypto.decrypt(cipher)
	if err != nil {
		return "", err
	}
	s.secretCacheMu.Lock()
	s.secretCache[id] = cachedSecret{cipher: cipher, secret: secret}
	s.secretCacheMu.Unlock()
	return secret, nil
}

func MaskSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if len(secret) <= 4 {
		return "***"
	}
	if len(secret) <= 8 {
		return secret[:2] + "*****" + secret[len(secret)-2:]
	}

	prefixLen := 4
	if dash := strings.Index(secret, "-"); dash > 0 && dash <= 4 {
		prefixLen = dash + 1
	}
	if prefixLen > len(secret)-4 {
		prefixLen = 2
	}
	return secret[:prefixLen] + "*****" + secret[len(secret)-4:]
}

func randomGatewayKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + base64.RawURLEncoding.EncodeToString(b), nil
}

func randomIDSuffix() (string, error) {
	var b [5]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// isConstraintViolation reports whether err is a SQLite constraint failure (e.g. a
// primary-key or UNIQUE collision). The low byte of every SQLITE_CONSTRAINT_* extended
// code is SQLITE_CONSTRAINT, so masking to the primary result code covers all variants.
func isConstraintViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return sqliteErr.Code()&0xff == sqlite3.SQLITE_CONSTRAINT
}

func hashGatewayKey(plain string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(plain)))
	return hex.EncodeToString(sum[:])
}

func slug(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	var b strings.Builder
	lastDash := false
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return fmt.Sprintf("id-%d", time.Now().Unix())
	}
	return out
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func parseTime(v string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Time{}
}

func intPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func floatPtr(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func floatNil(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func emptyNil(v string) any {
	if v == "" {
		return nil
	}
	return v
}
