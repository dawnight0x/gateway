package store

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"local-ai-gateway/internal/fileutil"
)

const (
	portableBackupVersion    = 1
	portableBackupIterations = 600_000
	maxPortableBackupSize    = 4 << 30
)

type portableBackupManifest struct {
	Version        int       `json:"version"`
	CreatedAt      time.Time `json:"createdAt"`
	DatabaseSHA256 string    `json:"databaseSha256"`
	KDF            struct {
		Name       string `json:"name"`
		Iterations int    `json:"iterations"`
		Salt       string `json:"salt"`
	} `json:"kdf"`
	Encryption struct {
		Name  string `json:"name"`
		Nonce string `json:"nonce"`
	} `json:"encryption"`
}

// CreatePortableBackup creates a consistent database snapshot and includes the
// master key encrypted with a user-provided passphrase for cross-machine restore.
func (s *Store) CreatePortableBackup(ctx context.Context, passphrase string) (BackupInfo, error) {
	if err := validateBackupPassphrase(passphrase); err != nil {
		return BackupInfo{}, err
	}
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	if err := errors.Join(s.flushGatewayUsage(ctx), s.flushKeySuccesses(ctx), s.flushRequestLogs(ctx)); err != nil {
		return BackupInfo{}, err
	}
	dir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return BackupInfo{}, err
	}
	timestamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	snapshotPath := filepath.Join(dir, ".gateway-portable-"+timestamp+".db")
	defer os.Remove(snapshotPath)
	query := `VACUUM INTO '` + strings.ReplaceAll(snapshotPath, `'`, `''`) + `'`
	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return BackupInfo{}, err
	}
	database, err := os.ReadFile(snapshotPath)
	if err != nil {
		return BackupInfo{}, err
	}
	archiveName := "gateway-portable-" + timestamp + ".zip"
	archivePath := filepath.Join(dir, archiveName)
	if err := writePortableBackup(archivePath, database, s.crypto.key, passphrase); err != nil {
		return BackupInfo{}, err
	}
	if err := s.pruneBackupsLocked(); err != nil {
		return BackupInfo{}, err
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		return BackupInfo{}, err
	}
	return BackupInfo{Name: archiveName, Path: archivePath, Size: info.Size(), CreatedAt: info.ModTime().UTC(), Kind: "portable"}, nil
}

func writePortableBackup(path string, database, masterKey []byte, passphrase string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	derived := pbkdf2SHA256([]byte(passphrase), salt, portableBackupIterations, 32)
	block, err := aes.NewCipher(derived)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	encryptedKey := aead.Seal(nil, nonce, masterKey, []byte("local-ai-gateway-portable-backup-v1"))
	digest := sha256.Sum256(database)
	manifest := portableBackupManifest{Version: portableBackupVersion, CreatedAt: time.Now().UTC(), DatabaseSHA256: hex.EncodeToString(digest[:])}
	manifest.KDF.Name = "pbkdf2-sha256"
	manifest.KDF.Iterations = portableBackupIterations
	manifest.KDF.Salt = base64.StdEncoding.EncodeToString(salt)
	manifest.Encryption.Name = "aes-256-gcm"
	manifest.Encryption.Nonce = base64.StdEncoding.EncodeToString(nonce)
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	temporary := path + ".tmp"
	defer os.Remove(temporary)
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
	}()
	archive := zip.NewWriter(file)
	for name, content := range map[string][]byte{
		"manifest.json":  manifestJSON,
		"gateway.db":     database,
		"master-key.enc": encryptedKey,
	} {
		entry, createErr := archive.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if createErr != nil {
			return createErr
		}
		if _, writeErr := entry.Write(content); writeErr != nil {
			return writeErr
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	closed = true
	return os.Rename(temporary, path)
}

// RestorePortableBackup restores an encrypted archive. Existing destinations
// are preserved unless force is true; forced restores first create timestamped copies.
func RestorePortableBackup(archivePath, databasePath, secretPath, passphrase string, force bool) error {
	if err := validateBackupPassphrase(passphrase); err != nil {
		return err
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open portable backup: %w", err)
	}
	defer archive.Close()
	entries := make(map[string][]byte, 3)
	for _, item := range archive.File {
		if item.Name != "manifest.json" && item.Name != "gateway.db" && item.Name != "master-key.enc" {
			continue
		}
		if item.UncompressedSize64 > maxPortableBackupSize {
			return fmt.Errorf("portable backup entry %s is too large", item.Name)
		}
		reader, err := item.Open()
		if err != nil {
			return err
		}
		content, readErr := io.ReadAll(io.LimitReader(reader, maxPortableBackupSize+1))
		closeErr := reader.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		entries[item.Name] = content
	}
	for _, name := range []string{"manifest.json", "gateway.db", "master-key.enc"} {
		if len(entries[name]) == 0 {
			return fmt.Errorf("portable backup is missing %s", name)
		}
	}
	var manifest portableBackupManifest
	decoder := json.NewDecoder(bytes.NewReader(entries["manifest.json"]))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("decode portable backup manifest: %w", err)
	}
	if manifest.Version != portableBackupVersion || manifest.KDF.Name != "pbkdf2-sha256" || manifest.Encryption.Name != "aes-256-gcm" {
		return fmt.Errorf("unsupported portable backup format")
	}
	if manifest.KDF.Iterations < 100_000 || manifest.KDF.Iterations > 10_000_000 {
		return fmt.Errorf("portable backup KDF iterations are outside the supported range")
	}
	digest := sha256.Sum256(entries["gateway.db"])
	if !hmac.Equal([]byte(strings.ToLower(manifest.DatabaseSHA256)), []byte(hex.EncodeToString(digest[:]))) {
		return fmt.Errorf("portable backup database checksum mismatch")
	}
	salt, err := base64.StdEncoding.DecodeString(manifest.KDF.Salt)
	if err != nil || len(salt) < 16 {
		return fmt.Errorf("portable backup salt is invalid")
	}
	nonce, err := base64.StdEncoding.DecodeString(manifest.Encryption.Nonce)
	if err != nil {
		return fmt.Errorf("portable backup nonce is invalid")
	}
	derived := pbkdf2SHA256([]byte(passphrase), salt, manifest.KDF.Iterations, 32)
	block, err := aes.NewCipher(derived)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	if len(nonce) != aead.NonceSize() {
		return fmt.Errorf("portable backup nonce has an invalid length")
	}
	masterKey, err := aead.Open(nil, nonce, entries["master-key.enc"], []byte("local-ai-gateway-portable-backup-v1"))
	if err != nil || len(masterKey) != 32 {
		return fmt.Errorf("portable backup passphrase is incorrect or the key is corrupted")
	}
	if err := prepareRestoreDestination(databasePath, force); err != nil {
		return err
	}
	if err := prepareRestoreDestination(secretPath, force); err != nil {
		return err
	}
	if err := fileutil.WriteFileAtomic(databasePath, entries["gateway.db"], 0o600); err != nil {
		return fmt.Errorf("restore database: %w", err)
	}
	if err := writeStoredKey(secretPath, masterKey); err != nil {
		return fmt.Errorf("restore master key: %w", err)
	}
	return nil
}

func prepareRestoreDestination(path string, force bool) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if !force {
		return fmt.Errorf("destination %s already exists; use --force to preserve it and continue", path)
	}
	backup := path + ".before-restore-" + time.Now().UTC().Format("20060102T150405Z")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return fileutil.WriteFileAtomic(backup, data, 0o600)
}

func validateBackupPassphrase(passphrase string) error {
	if len([]byte(passphrase)) < 12 {
		return fmt.Errorf("backup passphrase must be at least 12 bytes")
	}
	return nil
}

func pbkdf2SHA256(password, salt []byte, iterations, length int) []byte {
	result := make([]byte, 0, length)
	for blockIndex := uint32(1); len(result) < length; blockIndex++ {
		counter := []byte{byte(blockIndex >> 24), byte(blockIndex >> 16), byte(blockIndex >> 8), byte(blockIndex)}
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		_, _ = mac.Write(counter)
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:length]
}
