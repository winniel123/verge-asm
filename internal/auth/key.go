package auth

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const keyFile = "session.key"

const keyLen = 32

func LoadOrCreateKey(dir string) ([]byte, error) {
	// Never in Postgres: a dump discloses no key and a restore rotates none (v1 spec §4.3).
	path := filepath.Join(dir, keyFile)

	key, err := os.ReadFile(path) // #nosec G304 (session-key file: constant basename under operator-configured state dir, not request input)
	switch {
	case err == nil:
		if len(key) != keyLen {
			return nil, fmt.Errorf("auth: session key %s is %d bytes, want %d", path, len(key), keyLen)
		}
		return key, nil
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("auth: read session key: %w", err)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("auth: create state dir: %w", err)
	}
	key = make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("auth: generate session key: %w", err)
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("auth: write session key: %w", err)
	}
	return key, nil
}

func RotateKey(dir string) ([]byte, error) {
	// A restore must not carry a prior instance's signing key, so old sessions lapse (ADR-0124).
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("auth: create state dir: %w", err)
	}
	key := make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("auth: generate session key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, keyFile), key, 0o600); err != nil {
		return nil, fmt.Errorf("auth: write session key: %w", err)
	}
	// The caller must swap the in-memory key and its sub-keys, or old cookies keep verifying.
	return key, nil
}
