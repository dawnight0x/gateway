package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
)

// RotateMasterKey re-encrypts every upstream secret in one transaction. The
// previous key remains in the protected key bundle so local database snapshots
// created before rotation remain recoverable.
func (s *Store) RotateMasterKey(ctx context.Context) error {
	if os.Getenv("GATEWAY_MASTER_KEY") != "" {
		return fmt.Errorf("master key rotation is unavailable while GATEWAY_MASTER_KEY is set; rotate the external secret and re-encrypt through an offline migration")
	}
	s.maintenanceMu.Lock()
	defer s.maintenanceMu.Unlock()

	newKey := make([]byte, 32)
	if _, err := rand.Read(newKey); err != nil {
		return err
	}
	newCryptor, err := newCryptorFromKeys([][]byte{newKey, s.crypto.key})
	if err != nil {
		return err
	}
	if err := writeStoredKeys(s.secretPath, newCryptor.keys); err != nil {
		return fmt.Errorf("write crash-safe master key bundle: %w", err)
	}

	type encryptedSecret struct {
		id     string
		cipher string
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,secret_cipher FROM keys`)
	if err != nil {
		return err
	}
	var updates []encryptedSecret
	for rows.Next() {
		var id, oldCipher string
		if err := rows.Scan(&id, &oldCipher); err != nil {
			_ = rows.Close()
			return err
		}
		plain, err := s.crypto.decrypt(oldCipher)
		if err != nil {
			_ = rows.Close()
			return fmt.Errorf("decrypt upstream key %s: %w", id, err)
		}
		ciphertext, err := newCryptor.encrypt(plain)
		if err != nil {
			_ = rows.Close()
			return err
		}
		updates = append(updates, encryptedSecret{id: id, cipher: ciphertext})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	for _, update := range updates {
		if _, err := tx.ExecContext(ctx, `UPDATE keys SET secret_cipher=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, update.cipher, update.id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.crypto = newCryptor
	s.secretCacheMu.Lock()
	s.secretCache = make(map[string]cachedSecret)
	s.secretCacheMu.Unlock()
	return nil
}
