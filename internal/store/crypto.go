package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"local-ai-gateway/internal/fileutil"
)

const protectedKeyPrefix = "dpapi:"
const keyBundlePrefix = "keybundle:"
const osKeyringPrefix = "keyring:"

type cryptor struct {
	aead      cipher.AEAD
	fallbacks []cipher.AEAD
	key       []byte
	keys      [][]byte
}

func newCryptor(secretPath string) (*cryptor, error) {
	keys, err := loadOrCreateKeys(secretPath)
	if err != nil {
		return nil, err
	}
	return newCryptorFromKeys(keys)
}

func newCryptorFromKeys(keys [][]byte) (*cryptor, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one master key is required")
	}
	result := &cryptor{keys: make([][]byte, 0, len(keys))}
	for index, key := range keys {
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		if index == 0 {
			result.aead = aead
			result.key = append([]byte(nil), key...)
		} else {
			result.fallbacks = append(result.fallbacks, aead)
		}
		result.keys = append(result.keys, append([]byte(nil), key...))
	}
	return result, nil
}

func (c *cryptor) encrypt(text string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := c.aead.Seal(nonce, nonce, []byte(text), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

func (c *cryptor) decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(raw) < c.aead.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := raw[:c.aead.NonceSize()]
	data := raw[c.aead.NonceSize():]
	plain, err := c.aead.Open(nil, nonce, data, nil)
	if err == nil {
		return string(plain), nil
	}
	for _, fallback := range c.fallbacks {
		plain, fallbackErr := fallback.Open(nil, nonce, data, nil)
		if fallbackErr == nil {
			return string(plain), nil
		}
	}
	return "", err
}

func loadOrCreateKey(secretPath string) ([]byte, error) {
	keys, err := loadOrCreateKeys(secretPath)
	if err != nil {
		return nil, err
	}
	return keys[0], nil
}

func loadOrCreateKeys(secretPath string) ([][]byte, error) {
	if raw := strings.TrimSpace(os.Getenv("GATEWAY_MASTER_KEY")); raw != "" {
		key, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("decode GATEWAY_MASTER_KEY: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("GATEWAY_MASTER_KEY must decode to 32 bytes")
		}
		return [][]byte{key}, nil
	}
	b, err := os.ReadFile(secretPath)
	if err == nil {
		raw := strings.TrimSpace(string(b))
		keys, err := decodeStoredKeys(raw)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(raw, protectedKeyPrefix) && !strings.HasPrefix(raw, osKeyringPrefix) && !strings.HasPrefix(raw, keyBundlePrefix) {
			if err := writeStoredKeys(secretPath, keys); err != nil {
				return nil, fmt.Errorf("protect existing secret key: %w", err)
			}
		}
		return keys, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read secret key: %w", err)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := writeStoredKey(secretPath, key); err != nil {
		return nil, err
	}
	return [][]byte{key}, nil
}

func decodeStoredKeys(raw string) ([][]byte, error) {
	if !strings.HasPrefix(raw, keyBundlePrefix) {
		key, err := decodeStoredKey(raw)
		if err != nil {
			return nil, err
		}
		return [][]byte{key}, nil
	}
	encoded := strings.TrimPrefix(raw, keyBundlePrefix)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode master key bundle: %w", err)
	}
	var items []string
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode master key bundle: %w", err)
	}
	if len(items) == 0 || len(items) > 2 {
		return nil, fmt.Errorf("master key bundle must contain one or two keys")
	}
	keys := make([][]byte, 0, len(items))
	for _, item := range items {
		key, err := decodeStoredKey(item)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func decodeStoredKey(raw string) ([]byte, error) {
	if strings.HasPrefix(raw, protectedKeyPrefix) || strings.HasPrefix(raw, osKeyringPrefix) {
		prefix := protectedKeyPrefix
		if strings.HasPrefix(raw, osKeyringPrefix) {
			prefix = osKeyringPrefix
		}
		if prefix == protectedKeyPrefix && runtime.GOOS != "windows" {
			return nil, fmt.Errorf("DPAPI-protected secret key can only be opened on Windows; use a portable backup for migration")
		}
		if prefix == osKeyringPrefix && runtime.GOOS == "windows" {
			return nil, fmt.Errorf("OS-keyring protected secret key can only be opened on Linux or macOS; use a portable backup for migration")
		}
		encoded := strings.TrimPrefix(raw, prefix)
		protected, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, err
		}
		key, err := unprotectSecretKey(protected)
		if err != nil {
			return nil, err
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("secret key must be 32 bytes")
		}
		return key, nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("secret key must be 32 bytes")
	}
	return key, nil
}

func writeStoredKey(secretPath string, key []byte) error {
	return writeStoredKeys(secretPath, [][]byte{key})
}

func writeStoredKeys(secretPath string, keys [][]byte) error {
	if len(keys) == 0 || len(keys) > 2 {
		return fmt.Errorf("master key bundle must contain one or two keys")
	}
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		item, err := encodeStoredKey(secretPath, key)
		if err != nil {
			return err
		}
		items = append(items, item)
	}
	if len(items) == 1 {
		return fileutil.WriteFileAtomic(secretPath, []byte(items[0]), 0o600)
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	raw := keyBundlePrefix + base64.StdEncoding.EncodeToString(data)
	return fileutil.WriteFileAtomic(secretPath, []byte(raw), 0o600)
}

func encodeStoredKey(secretPath string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", fmt.Errorf("secret key must be 32 bytes")
	}
	raw := base64.StdEncoding.EncodeToString(key)
	if protected, ok, err := protectSecretKey(key); err != nil {
		return "", err
	} else if ok {
		prefix := protectedKeyPrefix
		if runtime.GOOS != "windows" {
			prefix = osKeyringPrefix
		}
		raw = prefix + base64.StdEncoding.EncodeToString(protected)
	} else {
		slog.Warn("master key is protected only by file permissions; set GATEWAY_MASTER_KEY from an OS secret manager for stronger protection", "path", secretPath)
	}
	return raw, nil
}
