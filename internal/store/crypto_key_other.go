//go:build !windows && !linux && !darwin

package store

import "fmt"

func protectSecretKey(key []byte) ([]byte, bool, error) {
	return nil, false, nil
}

func unprotectSecretKey(protected []byte) ([]byte, error) {
	return nil, fmt.Errorf("DPAPI-protected secret key can only be opened on Windows")
}
