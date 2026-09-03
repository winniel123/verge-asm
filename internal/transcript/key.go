// Package transcript manages the instance key and the AEAD seal/open helper that
// protect the Transcript corpus at rest (raw-job-output spec §5.3; ADR-0126
// reverses ADR-0053 for this one corpus).
package transcript

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

const keyFile = "transcript.key"

const keyLen = 32

func LoadOrCreateKey(dir string) ([]byte, error) {
	// A third volume both web and the worker mount, so either may create the key (spec §5.3).
	path := filepath.Join(dir, keyFile)

	key, err := os.ReadFile(path) // #nosec G304 (transcript-key file: constant basename under an operator-configured key dir, not request input)
	switch {
	case err == nil:
		return validateKey(path, key)
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("transcript: read key: %w", err)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("transcript: create key dir: %w", err)
	}
	// The key never enters Postgres, so no dump discloses it and no restore rotates it (spec §5.3).
	fresh := make([]byte, keyLen)
	if _, err := rand.Read(fresh); err != nil {
		return nil, fmt.Errorf("transcript: generate key: %w", err)
	}
	if err := atomicCreate(dir, path, fresh); err != nil {
		if os.IsExist(err) {
			winner, rerr := os.ReadFile(path) // #nosec G304 (same constant path as above)
			if rerr != nil {
				return nil, fmt.Errorf("transcript: read key after create race: %w", rerr)
			}
			return validateKey(path, winner)
		}
		return nil, err
	}
	return fresh, nil
}

func EnsureKey(dir string) error {
	// Both mains call this, so the volume is proven before the first seal or open (#865, #866).
	if _, err := LoadOrCreateKey(dir); err != nil {
		return err
	}
	return nil
}

func validateKey(path string, key []byte) ([]byte, error) {
	if len(key) != keyLen {
		return nil, fmt.Errorf("transcript: key %s is %d bytes, want %d", path, len(key), keyLen)
	}
	return key, nil
}

func atomicCreate(dir, path string, key []byte) error {
	// Link, never rename: a link fails when path exists, so exactly one concurrent caller wins.
	tmp, err := os.CreateTemp(dir, keyFile+".tmp.*")
	if err != nil {
		return fmt.Errorf("transcript: create temp key: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(key); err != nil {
		tmp.Close()
		return fmt.Errorf("transcript: write temp key: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("transcript: close temp key: %w", err)
	}
	// Returned unwrapped: os.IsExist does not unwrap, and the caller's race check depends on it.
	return os.Link(tmpPath, path)
}
