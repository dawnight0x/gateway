package desktop

import (
	"os"
	"path/filepath"
	"strings"
)

const startupAppName = "LocalAIGateway"

func startupCommand(exePath string) (string, error) {
	abs, err := filepath.Abs(exePath)
	if err != nil {
		return "", err
	}
	return `"` + strings.ReplaceAll(abs, `"`, `\"`) + `"`, nil
}

func currentExecutable() string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return os.Args[0]
}
