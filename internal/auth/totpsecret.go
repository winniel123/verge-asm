package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// totpAEADLabel domain-separates the key HKDF derives for the TOTP-secret AEAD from
// the raw session signing key. The session key signs cookies; a distinct sub-key
// encrypts the stored TOTP secret, so the same file-backed material is never reused
// for two unrelated cryptographic purposes (#337, ADR-0053).
const totpAEADLabel = "totp-secret-aead"

// DeriveTOTPKey derives the 256-bit XChaCha20-Poly1305 key that encrypts TOTP
// secrets at rest, via HKDF-SHA256 over the file-backed session key with a
// domain-separation label (#337). It never reads or writes Postgres: the input is
// the same web-only volume key the session path holds, so a database dump discloses
// neither the session key nor this derivative (v1 spec §4.3, ADR-0053). The result
// is deterministic for a given session key, so a restart re-derives the identical
// sub-key and previously-written secrets still decrypt.
func DeriveTOTPKey(sessionKey []byte) ([]byte, error) {
	out := make([]byte, chacha20poly1305.KeySize)
	r := hkdf.New(sha256.New, sessionKey, nil, []byte(totpAEADLabel))
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("auth: derive totp key: %w", err)
	}
	return out, nil
}

// EncryptTOTPSecret seals a cleartext base32 TOTP secret under key, returning the
// base64 of a fresh 24-byte random nonce prepended to the XChaCha20-Poly1305
// ciphertext — the form stored in the account.totp_secret TEXT column (#337). An
// empty secret encrypts to empty, so a not-yet-enrolled account stores no ciphertext.
func EncryptTOTPSecret(key []byte, secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("auth: totp aead: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("auth: totp nonce: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptTOTPSecret reverses EncryptTOTPSecret, recovering the cleartext base32
// secret from the stored base64(nonce||ciphertext) (#337). An empty stored value
// yields an empty secret. Every malformed, wrong-key, or tampered input returns an
// error rather than a partial result, so the caller can treat a decryption failure
// as a plain verification failure — a pre-#337 cleartext row (never valid base64 of
// a sealed message) simply fails to verify rather than crashing the login path.
func DecryptTOTPSecret(key []byte, stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", fmt.Errorf("auth: decode totp secret: %w", err)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("auth: totp aead: %w", err)
	}
	if len(raw) < aead.NonceSize() {
		return "", fmt.Errorf("auth: totp secret too short")
	}
	nonce, ct := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("auth: open totp secret: %w", err)
	}
	return string(plain), nil
}
