// Package transcript manages the instance key and the AEAD seal/open helper that
// protect the Transcript corpus (raw job output) at rest. The Transcript is the
// first secret-bearing data Postgres holds, so its three bytea streams (stdout,
// stderr, sent-scope) are sealed with a symmetric key that lives on a service
// volume and never enters Postgres (raw-job-output spec §5.3; ADR-0126 reverses
// ADR-0053 for this one corpus).
//
// The worker writes transcripts and web reads them, from two different state
// volumes, so the key lives on a THIRD volume that both mount. LoadOrCreateKey is
// therefore race-safe: either service may create the key on first boot.
package transcript

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
)

// keyFile is the basename, inside the shared key directory, of the instance
// transcript key. The directory is a volume mounted by BOTH web (the reader) and
// the worker (the writer) — distinct from each service's own state volume — so one
// symmetric key seals on write and opens on read (spec §5.3).
const keyFile = "transcript.key"

const keyLen = 32

// LoadOrCreateKey returns the instance transcript key held in dir, creating it on
// first boot. The key is generated on the shared key volume and is never persisted
// to Postgres, so a database dump does not disclose it and a database restore does
// not silently rotate it (spec §5.3).
//
// Both web and the worker call this and may boot together, so creation is
// race-safe: the fresh key is written to a temp file and hard-linked into place.
// The link fails if the final key already exists, so exactly one key ever wins and
// a losing caller adopts the winner's key. A concurrent reader never observes a
// half-written file. A key of the wrong length is treated as corruption and
// reported rather than used.
func LoadOrCreateKey(dir string) ([]byte, error) {
	path := filepath.Join(dir, keyFile)

	key, err := os.ReadFile(path) // #nosec G304 (transcript-key file: constant basename under an operator-configured key dir, not request input)
	switch {
	case err == nil:
		return validateKey(path, key)
	case !os.IsNotExist(err):
		return nil, fmt.Errorf("transcript: read key: %w", err)
	}

	// The key is absent: generate one and try to win the create.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("transcript: create key dir: %w", err)
	}
	fresh := make([]byte, keyLen)
	if _, err := rand.Read(fresh); err != nil {
		return nil, fmt.Errorf("transcript: generate key: %w", err)
	}
	if err := atomicCreate(dir, path, fresh); err != nil {
		if os.IsExist(err) {
			// Another process won the create; adopt its key.
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

// EnsureKey provisions the instance key on dir at boot, creating it if absent and
// proving the shared volume is writable, then discards it. Both mains call this so
// the key exists before the first write or read; the writer (#865) and reader
// (#866) re-load it with LoadOrCreateKey when they seal or open.
func EnsureKey(dir string) error {
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

// atomicCreate writes key to a uniquely-named temp file in dir, then hard-links it
// to path. The link fails with an os.IsExist error if path already exists, so only
// the first caller to link wins and its fully-written file becomes the key. The
// temp file is always removed. The link error is returned unwrapped so the caller's
// os.IsExist check still sees it. dir must already exist.
func atomicCreate(dir, path string, key []byte) error {
	tmp, err := os.CreateTemp(dir, keyFile+".tmp.*") // 0600 by default
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
	return os.Link(tmpPath, path) // may be an os.IsExist error: caller adopts the winner's key
}
