//go:build linux

package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func protectSecretKey(key []byte) ([]byte, bool, error) {
	tool, err := exec.LookPath("secret-tool")
	if err != nil {
		return nil, false, nil
	}
	token, err := keyringToken()
	if err != nil {
		return nil, false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, tool, "store", "--label=Local AI Gateway master key", "application", "local-ai-gateway", "token", token)
	cmd.Stdin = strings.NewReader(base64.StdEncoding.EncodeToString(key))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, false, nil
	}
	return []byte(token), true, nil
}

func unprotectSecretKey(protected []byte) ([]byte, error) {
	tool, err := exec.LookPath("secret-tool")
	if err != nil {
		return nil, fmt.Errorf("secret-tool is required to open the OS-keyring protected master key")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, tool, "lookup", "application", "local-ai-gateway", "token", string(protected)).Output()
	if err != nil {
		return nil, fmt.Errorf("read master key from Secret Service: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return nil, fmt.Errorf("decode master key from Secret Service: %w", err)
	}
	return key, nil
}

func keyringToken() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
