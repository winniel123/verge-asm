package transcript

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

func Seal(key, plaintext []byte) ([]byte, error) {
	if plaintext == nil {
		// Nil stays SQL NULL, which the table distinguishes from captured-empty (migration 23700).
		return nil, nil
	}
	// Raw bytes, not base64: the column is bytea, unlike auth's text-column TOTP secret.
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("transcript: aead: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("transcript: nonce: %w", err)
	}
	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func Open(key, sealed []byte) ([]byte, error) {
	// A caller must fail closed on this error and never render an unauthenticated result.
	if sealed == nil {
		return nil, nil
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("transcript: aead: %w", err)
	}
	if len(sealed) < aead.NonceSize() {
		return nil, fmt.Errorf("transcript: sealed stream too short: %d bytes", len(sealed))
	}
	nonce, ciphertext := sealed[:aead.NonceSize()], sealed[aead.NonceSize():]
	// A non-nil zero-length dst keeps a captured-empty stream non-nil on open.
	plain, err := aead.Open(make([]byte, 0), nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("transcript: open stream: %w", err)
	}
	return plain, nil
}
