package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"local-ai-gateway/internal/admin"
	"local-ai-gateway/internal/config"
	"local-ai-gateway/internal/desktop"
	"local-ai-gateway/internal/proxy"
	"local-ai-gateway/internal/router"
	"local-ai-gateway/internal/store"
	"local-ai-gateway/internal/tray"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gateway stopped:", err)
		os.Exit(1)
	}
}

func run() (runErr error) {
	normalizeWorkingDirectory()
	if err := waitForRestartParent(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	trayEnabled := cfg.Tray.Enabled && tray.Supported()

	logPath := desktop.LogPath(cfg.Storage.Path)
	logFile, err := setupLogging(logPath, cfg.Logging.MaxSizeMB, cfg.Logging.MaxBackups)
	if err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, logFile.Close()) }()
	defer func() {
		if runErr != nil {
			slog.Error("gateway stopped", "error", runErr)
		}
	}()

	lockPath := desktop.LockPath(cfg.Storage.Path)
	if trayEnabled && !desktop.Exists(lockPath) && desktop.WaitHealthy(cfg.LocalURL()+"/health", 2, 150*time.Millisecond) {
		slog.Warn("healthy gateway without lock detected; replacing legacy/no-tray instance", "addr", cfg.Server.Addr())
		if err := desktop.TerminateMatchingListeners(cfg.Server.Port, os.Args[0]); err != nil {
			return fmt.Errorf("replace existing gateway: %w", err)
		}
		time.Sleep(700 * time.Millisecond)
	}

	lock, exists, err := desktop.AcquireLock(lockPath, false)
	if err != nil {
		return err
	}
	if exists {
		adminOpenURL := adminLoginURL(cfg)
		safeAdminURL := adminURL(cfg)
		if desktop.WaitHealthy(cfg.LocalURL()+"/health", 6, 250*time.Millisecond) {
			if !trayEnabled {
				if cfg.Server.OpenBrowserOnDuplicate {
					_ = desktop.OpenBrowser(adminOpenURL)
					slog.Info("gateway already running; opened existing admin page", "admin", safeAdminURL)
				} else {
					slog.Info("gateway already running; duplicate process exiting", "admin", safeAdminURL)
				}
				return nil
			}
			slog.Warn("healthy gateway already running; replacing it so this launch owns the system tray", "addr", cfg.Server.Addr())
			if err := desktop.TerminateMatchingListeners(cfg.Server.Port, os.Args[0]); err != nil {
				_ = desktop.OpenBrowser(adminOpenURL)
				slog.Warn("replace existing gateway failed; opened existing admin page", "admin", safeAdminURL, "error", err)
				return nil
			}
			time.Sleep(700 * time.Millisecond)
		}
		lock, exists, err = desktop.AcquireLock(desktop.LockPath(cfg.Storage.Path), true)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("gateway already appears to be running")
		}
	}
	defer func() { runErr = errors.Join(runErr, lock.Release()) }()

	db, err := store.OpenWithOptions(cfg.Storage.Path, cfg.Storage.SecretPath, store.Options{
		LogRetentionDays:      cfg.Storage.LogRetentionDays,
		LogMaxEntries:         cfg.Storage.LogMaxEntries,
		Timezone:              cfg.Storage.Timezone,
		BackupBeforeMigration: cfg.Storage.BackupBeforeMigration,
		BackupRetention:       cfg.Storage.BackupRetention,
		DisableRequestLogging: !cfg.Storage.RequestLoggingEnabled,
	})
	if err != nil {
		return err
	}
	defer func() { runErr = errors.Join(runErr, db.Close()) }()

	rt := router.New(db, cfg.Routing)
	px := proxy.New(db, rt, cfg)
	ad := admin.New(db, cfg)

	mux := http.NewServeMux()
	ad.Register(mux)
	px.Register(mux)

	server := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.Server.IdleTimeoutSeconds) * time.Second,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("local gateway listening", "addr", cfg.Server.Addr(), "admin", cfg.PublicURL()+"/admin")
		var err error
		if cfg.Server.TLSCertFile != "" {
			err = server.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	var once sync.Once
	var shutdownErr error
	shutdown := func() error {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			shutdownErr = server.Shutdown(ctx)
		})
		return shutdownErr
	}
	restart := func() error {
		slog.Info("restarting gateway")
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), "GATEWAY_RESTART_PARENT_PID="+strconv.Itoa(os.Getpid()))
		workingDirectory, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve restart working directory: %w", err)
		}
		cmd.Dir = workingDirectory
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		startReplacement := func() (func(), error) {
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			return func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
					_ = cmd.Wait()
				}
			}, nil
		}
		replacementStarted, err := performRestart(startReplacement, shutdown, db.Close, lock.Release)
		if replacementStarted && err != nil {
			slog.Warn("restart cleanup completed with errors", "error", err)
			return nil
		}
		return err
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)
	if trayEnabled {
		statusProvider := func() tray.Status {
			stats, err := db.Stats(context.Background())
			if err != nil {
				return tray.Status{Healthy: true, Error: err.Error()}
			}
			return tray.Status{
				Healthy:       true,
				ActiveKeys:    stats.ActiveKeys,
				FailedKeys:    stats.FailedKeys,
				TodayRequests: stats.TodayRequests,
				TodayTokens:   stats.TodayTokens,
			}
		}
		fatalServerErr := make(chan error, 1)
		go func() {
			select {
			case <-stop:
				if err := shutdown(); err != nil {
					slog.Error("shutdown failed", "error", err)
				}
				tray.Quit()
			case err := <-serverErr:
				fatalServerErr <- err
				if shutdownErr := shutdown(); shutdownErr != nil {
					slog.Error("shutdown after server failure failed", "error", shutdownErr)
				}
				tray.Quit()
			}
		}()
		tray.Run(tray.Options{
			AdminURL:        adminURL(cfg),
			AdminOpenURL:    adminLoginURL(cfg),
			DataDir:         desktop.DataDir(cfg.Storage.Path),
			LogPath:         logPath,
			Status:          statusProvider,
			OpenAIConfig:    "$env:OPENAI_BASE_URL=\"" + cfg.PublicURL() + "/v1\"\n$env:OPENAI_API_KEY=\"" + setupToken(cfg) + "\"",
			AnthropicConfig: "$env:ANTHROPIC_BASE_URL=\"" + cfg.PublicURL() + "\"\n$env:ANTHROPIC_AUTH_TOKEN=\"" + setupToken(cfg) + "\"",
			GeminiConfig:    "$env:GEMINI_BASE_URL=\"" + cfg.PublicURL() + "\"\n$env:GEMINI_API_KEY=\"" + setupToken(cfg) + "\"",
			Restart:         restart,
			Shutdown: func() {
				if err := shutdown(); err != nil {
					slog.Error("shutdown failed", "error", err)
				}
			},
		})
		select {
		case err := <-fatalServerErr:
			return err
		default:
			return nil
		}
	}

	select {
	case <-stop:
		return shutdown()
	case err := <-serverErr:
		return errors.Join(err, shutdown())
	}
}

func performRestart(startReplacement func() (func(), error), shutdown, closeDatabase, releaseLock func() error) (replacementStarted bool, restartErr error) {
	if startReplacement == nil {
		return false, fmt.Errorf("restart step %q is not configured", "start replacement process")
	}
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "shut down server", run: shutdown},
		{name: "close database", run: closeDatabase},
		{name: "release process lock", run: releaseLock},
	}
	for _, step := range steps {
		if step.run == nil {
			return false, fmt.Errorf("restart step %q is not configured", step.name)
		}
	}
	abortReplacement, err := startReplacement()
	if err != nil {
		return false, fmt.Errorf("start replacement process: %w", err)
	}
	completed := false
	defer func() {
		if !completed && abortReplacement != nil {
			abortReplacement()
		}
	}()
	for _, step := range steps {
		if err := step.run(); err != nil {
			restartErr = errors.Join(restartErr, fmt.Errorf("%s: %w", step.name, err))
		}
	}
	completed = true
	return true, restartErr
}

func waitForRestartParent() error {
	rawPID := strings.TrimSpace(os.Getenv("GATEWAY_RESTART_PARENT_PID"))
	if rawPID == "" {
		return nil
	}
	_ = os.Unsetenv("GATEWAY_RESTART_PARENT_PID")
	pid, err := strconv.Atoi(rawPID)
	if err != nil || pid <= 0 || pid == os.Getpid() {
		return fmt.Errorf("invalid restart parent PID %q", rawPID)
	}
	if !desktop.WaitForProcessExit(pid, 30*time.Second) {
		return fmt.Errorf("restart parent process %d did not exit within 30 seconds", pid)
	}
	return nil
}

func normalizeWorkingDirectory() {
	if target := strings.TrimSpace(os.Getenv("GATEWAY_WORKDIR")); target != "" {
		_ = os.Chdir(target)
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	exeDir := filepath.Dir(exe)
	if !samePath(cwd, exeDir) || !strings.EqualFold(filepath.Base(exeDir), "bin") {
		return
	}
	parent := filepath.Dir(exeDir)
	if desktop.Exists(filepath.Join(parent, "config.yaml")) ||
		desktop.Exists(filepath.Join(parent, "config.example.yaml")) ||
		desktop.Exists(filepath.Join(parent, "README.md")) {
		_ = os.Chdir(parent)
	}
}

func samePath(a, b string) bool {
	aa, err := filepath.Abs(a)
	if err != nil {
		aa = a
	}
	bb, err := filepath.Abs(b)
	if err != nil {
		bb = b
	}
	return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
}

func setupToken(cfg config.Config) string {
	if cfg.Server.ProxyToken != "" {
		return cfg.Server.ProxyToken
	}
	return "CREATE_A_GATEWAY_KEY_IN_ADMIN"
}

func adminURL(cfg config.Config) string {
	return cfg.PublicURL() + "/admin"
}

func adminLoginURL(cfg config.Config) string {
	token := strings.TrimSpace(cfg.Server.AdminToken)
	if token == "" {
		return adminURL(cfg)
	}
	return adminURL(cfg) + "#token=" + url.QueryEscape(token)
}
