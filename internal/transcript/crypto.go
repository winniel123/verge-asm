package transcript

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// Seal encrypts a single transcript stream for storage in its bytea column,
// returning a fresh 24-byte random nonce prepended to the XChaCha20-Poly1305
// ciphertext. The result is raw bytes, not base64 — the column is bytea, so no
// text encoding is needed (unlike auth.EncryptTOTPSecret, which targets a text
// column).
//
// A nil plaintext returns nil: a stream the variant does not carry stays SQL NULL,
// preserving the table's NULL-vs-captured-empty distinction (migration 23700). A
// non-nil but empty stream seals to real ciphertext, so a captured-empty stream is
// retained as captured, not collapsed to NULL. Each call draws a fresh random
// nonce, so sealing the same bytes twice yields different ciphertext.
func Seal(key, plaintext []byte) ([]byte, error) {
	if plaintext == nil {
		return nil, nil
	}
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

// Open reverses Seal, recovering a stream's verbatim bytes from the stored
// nonce||ciphertext. A nil input returns nil (a NULL column stays absent).
//
// A too-short input, a wrong key, or any tampering returns an error. Callers MUST
// treat that as a hard fault and fail closed and loudly, never rendering a partial
// or unauthenticated result — a correctly-sealed store holds only valid ciphertext
// (mirrors auth.DecryptTOTPSecret's contract). A successful open always returns a
// non-nil slice, so a captured-empty stream round-trips to non-nil empty and stays
// distinct from a NULL column.
func Open(key, sealed []byte) ([]byte, error) {
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
