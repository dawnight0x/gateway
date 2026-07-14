//go:build darwin

package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const macOSKeychainService = "local-ai-gateway"

func protectSecretKey(key []byte) ([]byte, bool, error) {
	tool, err := exec.LookPath("security")
	if err != nil {
		return nil, false, nil
	}
	token, err := keyringToken()
	if err != nil {
		return nil, false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := exec.CommandContext(ctx, tool, "add-generic-password", "-a", token, "-s", macOSKeychainService, "-w", encoded, "-U").Run(); err != nil {
		return nil, false, nil
	}
	return []byte(token), true, nil
}

func unprotectSecretKey(protected []byte) ([]byte, error) {
	tool, err := exec.LookPath("security")
	if err != nil {
		return nil, fmt.Errorf("macOS security tool is required to open the Keychain-protected master key")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, tool, "find-generic-password", "-a", string(protected), "-s", macOSKeychainService, "-w").Output()
	if err != nil {
		return nil, fmt.Errorf("read master key from macOS Keychain: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(output)))
	if err != nil {
		return nil, fmt.Errorf("decode master key from macOS Keychain: %w", err)
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
