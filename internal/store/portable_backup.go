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
	portableBackupVersion    = 2
	portableBackupV1         = 1
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
		Name           string `json:"name"`
		Nonce          string `json:"nonce,omitempty"`
		DatabaseNonce  string `json:"databaseNonce,omitempty"`
		MasterKeyNonce string `json:"masterKeyNonce,omitempty"`
	} `json:"encryption"`
}

type restoreTransaction struct {
	Database restoreTransactionFile `json:"database"`
	Secret   restoreTransactionFile `json:"secret"`
}

type restoreTransactionFile struct {
	Target  string `json:"target"`
	Backup  string `json:"backup"`
	Existed bool   `json:"existed"`
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
	databaseNonce := make([]byte, aead.NonceSize())
	masterKeyNonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(databaseNonce); err != nil {
		return err
	}
	if _, err := rand.Read(masterKeyNonce); err != nil {
		return err
	}
	digest := sha256.Sum256(database)
	manifest := portableBackupManifest{Version: portableBackupVersion, CreatedAt: time.Now().UTC(), DatabaseSHA256: hex.EncodeToString(digest[:])}
	manifest.KDF.Name = "pbkdf2-sha256"
	manifest.KDF.Iterations = portableBackupIterations
	manifest.KDF.Salt = base64.StdEncoding.EncodeToString(salt)
	manifest.Encryption.Name = "aes-256-gcm"
	manifest.Encryption.DatabaseNonce = base64.StdEncoding.EncodeToString(databaseNonce)
	manifest.Encryption.MasterKeyNonce = base64.StdEncoding.EncodeToString(masterKeyNonce)
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	aad := append([]byte("local-ai-gateway-portable-backup-v2\n"), manifestJSON...)
	encryptedDatabase := aead.Seal(nil, databaseNonce, database, aad)
	encryptedKey := aead.Seal(nil, masterKeyNonce, masterKey, aad)

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
		"database.enc":   encryptedDatabase,
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
	if err := recoverRestoreTransaction(databasePath, secretPath); err != nil {
		return fmt.Errorf("recover interrupted restore: %w", err)
	}
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open portable backup: %w", err)
	}
	defer archive.Close()
	entries := make(map[string][]byte, 4)
	for _, item := range archive.File {
		if item.Name != "manifest.json" && item.Name != "gateway.db" && item.Name != "database.enc" && item.Name != "master-key.enc" {
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
	for _, name := range []string{"manifest.json", "master-key.enc"} {
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
	if (manifest.Version != portableBackupV1 && manifest.Version != portableBackupVersion) || manifest.KDF.Name != "pbkdf2-sha256" || manifest.Encryption.Name != "aes-256-gcm" {
		return fmt.Errorf("unsupported portable backup format")
	}
	if manifest.KDF.Iterations < 100_000 || manifest.KDF.Iterations > 10_000_000 {
		return fmt.Errorf("portable backup KDF iterations are outside the supported range")
	}
	salt, err := base64.StdEncoding.DecodeString(manifest.KDF.Salt)
	if err != nil || len(salt) < 16 {
		return fmt.Errorf("portable backup salt is invalid")
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
	database, masterKey, err := decryptPortableBackup(entries, manifest, aead)
	if err != nil {
		return err
	}
	if err := prepareRestoreDestination(databasePath, force); err != nil {
		return err
	}
	if err := prepareRestoreDestination(secretPath, force); err != nil {
		return err
	}
	encodedKey, err := encodeStoredKey(secretPath, masterKey)
	if err != nil {
		return fmt.Errorf("protect restored master key: %w", err)
	}
	return restoreFilesTransaction(databasePath, database, secretPath, []byte(encodedKey))
}

func decryptPortableBackup(entries map[string][]byte, manifest portableBackupManifest, aead cipher.AEAD) ([]byte, []byte, error) {
	if manifest.Version == portableBackupV1 {
		if len(entries["gateway.db"]) == 0 {
			return nil, nil, fmt.Errorf("portable backup is missing gateway.db")
		}
		digest := sha256.Sum256(entries["gateway.db"])
		if !hmac.Equal([]byte(strings.ToLower(manifest.DatabaseSHA256)), []byte(hex.EncodeToString(digest[:]))) {
			return nil, nil, fmt.Errorf("portable backup database checksum mismatch")
		}
		nonce, err := decodePortableNonce(manifest.Encryption.Nonce, aead.NonceSize())
		if err != nil {
			return nil, nil, err
		}
		masterKey, err := aead.Open(nil, nonce, entries["master-key.enc"], []byte("local-ai-gateway-portable-backup-v1"))
		if err != nil || len(masterKey) != 32 {
			return nil, nil, fmt.Errorf("portable backup passphrase is incorrect or the key is corrupted")
		}
		return entries["gateway.db"], masterKey, nil
	}
	if len(entries["database.enc"]) == 0 {
		return nil, nil, fmt.Errorf("portable backup is missing database.enc")
	}
	databaseNonce, err := decodePortableNonce(manifest.Encryption.DatabaseNonce, aead.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	masterKeyNonce, err := decodePortableNonce(manifest.Encryption.MasterKeyNonce, aead.NonceSize())
	if err != nil {
		return nil, nil, err
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	aad := append([]byte("local-ai-gateway-portable-backup-v2\n"), manifestJSON...)
	database, err := aead.Open(nil, databaseNonce, entries["database.enc"], aad)
	if err != nil {
		return nil, nil, fmt.Errorf("portable backup passphrase is incorrect or the database/manifest was modified")
	}
	masterKey, err := aead.Open(nil, masterKeyNonce, entries["master-key.enc"], aad)
	if err != nil || len(masterKey) != 32 {
		return nil, nil, fmt.Errorf("portable backup passphrase is incorrect or the key/manifest was modified")
	}
	digest := sha256.Sum256(database)
	if !hmac.Equal([]byte(strings.ToLower(manifest.DatabaseSHA256)), []byte(hex.EncodeToString(digest[:]))) {
		return nil, nil, fmt.Errorf("portable backup authenticated database checksum mismatch")
	}
	return database, masterKey, nil
}

func decodePortableNonce(encoded string, expected int) ([]byte, error) {
	nonce, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(nonce) != expected {
		return nil, fmt.Errorf("portable backup nonce is invalid")
	}
	return nonce, nil
}

func restoreFilesTransaction(databasePath string, database []byte, secretPath string, encodedKey []byte) error {
	databasePath, err := filepath.Abs(databasePath)
	if err != nil {
		return err
	}
	secretPath, err = filepath.Abs(secretPath)
	if err != nil {
		return err
	}
	originalDatabase, databaseExists, err := readOptionalFile(databasePath)
	if err != nil {
		return err
	}
	originalSecret, secretExists, err := readOptionalFile(secretPath)
	if err != nil {
		return err
	}
	suffix := time.Now().UTC().Format("20060102T150405.000000000Z")
	transaction := restoreTransaction{
		Database: restoreTransactionFile{Target: databasePath, Backup: databasePath + ".restore-old-" + suffix, Existed: databaseExists},
		Secret:   restoreTransactionFile{Target: secretPath, Backup: secretPath + ".restore-old-" + suffix, Existed: secretExists},
	}
	if databaseExists {
		if err := fileutil.WriteFileAtomic(transaction.Database.Backup, originalDatabase, 0o600); err != nil {
			return fmt.Errorf("stage original database: %w", err)
		}
	}
	if secretExists {
		if err := fileutil.WriteFileAtomic(transaction.Secret.Backup, originalSecret, 0o600); err != nil {
			_ = os.Remove(transaction.Database.Backup)
			return fmt.Errorf("stage original master key: %w", err)
		}
	}
	journalPath := restoreJournalPath(databasePath)
	journal, err := json.Marshal(transaction)
	if err != nil {
		return err
	}
	if err := fileutil.WriteFileAtomic(journalPath, journal, 0o600); err != nil {
		_ = os.Remove(transaction.Database.Backup)
		_ = os.Remove(transaction.Secret.Backup)
		return fmt.Errorf("create restore transaction journal: %w", err)
	}
	fail := func(cause error) error {
		return errors.Join(cause, recoverRestoreTransaction(databasePath, secretPath))
	}
	if err := fileutil.WriteFileAtomic(databasePath, database, 0o600); err != nil {
		return fail(fmt.Errorf("restore database: %w", err))
	}
	if err := fileutil.WriteFileAtomic(secretPath, encodedKey, 0o600); err != nil {
		return fail(fmt.Errorf("restore master key: %w", err))
	}
	// Verify both durable files before considering the transaction complete.
	if written, err := os.ReadFile(databasePath); err != nil || !bytes.Equal(written, database) {
		return fail(fmt.Errorf("verify restored database: %w", err))
	}
	if written, err := os.ReadFile(secretPath); err != nil || !bytes.Equal(written, encodedKey) {
		return fail(fmt.Errorf("verify restored master key: %w", err))
	}
	if err := os.Remove(journalPath); err != nil {
		return fail(fmt.Errorf("commit restore transaction: %w", err))
	}
	_ = os.Remove(transaction.Database.Backup)
	_ = os.Remove(transaction.Secret.Backup)
	return nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func restoreJournalPath(databasePath string) string {
	return databasePath + ".restore-transaction.json"
}

func recoverRestoreTransaction(databasePath, secretPath string) error {
	databasePath, err := filepath.Abs(databasePath)
	if err != nil {
		return err
	}
	secretPath, err = filepath.Abs(secretPath)
	if err != nil {
		return err
	}
	journalPath := restoreJournalPath(databasePath)
	journal, err := os.ReadFile(journalPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var transaction restoreTransaction
	decoder := json.NewDecoder(bytes.NewReader(journal))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&transaction); err != nil {
		return fmt.Errorf("decode restore transaction journal: %w", err)
	}
	if filepath.Clean(transaction.Database.Target) != filepath.Clean(databasePath) || filepath.Clean(transaction.Secret.Target) != filepath.Clean(secretPath) {
		return fmt.Errorf("restore transaction journal targets do not match configured storage paths")
	}
	restore := func(item restoreTransactionFile) error {
		if !item.Existed {
			if err := os.Remove(item.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}
		content, err := os.ReadFile(item.Backup)
		if err != nil {
			return err
		}
		return fileutil.WriteFileAtomic(item.Target, content, 0o600)
	}
	if err := errors.Join(restore(transaction.Database), restore(transaction.Secret)); err != nil {
		return err
	}
	if err := os.Remove(journalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = os.Remove(transaction.Database.Backup)
	_ = os.Remove(transaction.Secret.Backup)
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
