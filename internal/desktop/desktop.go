package desktop

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Lock struct {
	path     string
	file     *os.File
	token    string
	mu       sync.Mutex
	released bool
}

func DataDir(dbPath string) string {
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "" {
		return "data"
	}
	return dir
}

func LogPath(dbPath string) string {
	return filepath.Join(DataDir(dbPath), "gateway.log")
}

func LockPath(dbPath string) string {
	return filepath.Join(DataDir(dbPath), "gateway.lock")
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func AcquireLock(path string, stale bool) (*Lock, bool, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, false, err
	}
	if stale {
		if pid, ok := lockPID(path); ok && processAlive(pid) {
			return nil, true, nil
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, false, err
		}
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, true, nil
		}
		return nil, false, err
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, false, err
	}
	token := hex.EncodeToString(tokenBytes)
	if _, err := fmt.Fprintf(f, "pid=%d\nstarted_at=%s\ntoken=%s\n", os.Getpid(), time.Now().Format(time.RFC3339), token); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, false, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, false, err
	}
	return &Lock{path: path, file: f, token: token}, false, nil
}

func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return nil
	}
	l.released = true
	closeErr := error(nil)
	if l.file != nil {
		closeErr = l.file.Close()
		l.file = nil
	}
	content, err := os.ReadFile(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return closeErr
	}
	if err != nil {
		return errors.Join(closeErr, err)
	}
	if !strings.Contains(string(content), "token="+l.token+"\n") {
		return closeErr
	}
	removeErr := os.Remove(l.path)
	if errors.Is(removeErr, os.ErrNotExist) {
		removeErr = nil
	}
	return errors.Join(closeErr, removeErr)
}

func lockPID(path string) (int, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.HasPrefix(line, "pid=") {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "pid=")))
		return pid, err == nil && pid > 0
	}
	return 0, false
}

func TerminateMatchingListeners(port int, exePath string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("terminating old listener is only implemented on windows")
	}
	absExe, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	script := `
$ErrorActionPreference = 'Stop'
$port = [int]$args[0]
$expected = [System.IO.Path]::GetFullPath($args[1])
$connections = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
foreach ($connection in $connections) {
  $process = Get-Process -Id $connection.OwningProcess -ErrorAction Stop
  $path = $process.Path
  $samePath = $false
  if ($path) {
    $samePath = ([System.IO.Path]::GetFullPath($path) -ieq $expected)
  }
  if ($samePath) {
    Stop-Process -Id $process.Id -Force
  } else {
    throw "Port $port is owned by $($process.ProcessName) pid=$($process.Id) path=$path"
  }
}
`
	return exec.Command(powerShellExecutable(), "-NoProfile", "-Command", script, strconv.Itoa(port), absExe).Run()
}

func WaitHealthy(healthURL string, attempts int, delay time.Duration) bool {
	client := &http.Client{Timeout: 650 * time.Millisecond}
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest(http.MethodGet, healthURL, nil)
		if err == nil {
			res, err := client.Do(req)
			if err == nil {
				_ = res.Body.Close()
				if res.StatusCode >= 200 && res.StatusCode < 300 {
					return true
				}
			}
		}
		time.Sleep(delay)
	}
	return false
}

func WaitForProcessExit(pid int, timeout time.Duration) bool {
	if pid <= 0 || !processAlive(pid) {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		if !processAlive(pid) {
			return true
		}
	}
	return !processAlive(pid)
}

func OpenBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	case "darwin":
		return exec.Command("open", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}

func OpenPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", abs).Start()
	case "darwin":
		return exec.Command("open", abs).Start()
	default:
		return exec.Command("xdg-open", abs).Start()
	}
}

func CopyText(text string) error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command(powerShellExecutable(), "-NoProfile", "-Command", "Set-Clipboard -Value $input")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command("wl-copy")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
}

func powerShellExecutable() string {
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path
	}
	return "powershell"
}
