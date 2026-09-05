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

const totpAEADLabel = "totp-secret-aead" // no key is reused for two purposes (ADR-0172 §1, #337)

func DeriveTOTPKey(sessionKey []byte) ([]byte, error) {
	// The nil salt keeps this deterministic, so a restart still decrypts secrets written earlier.
	out := make([]byte, chacha20poly1305.KeySize)
	r := hkdf.New(sha256.New, sessionKey, nil, []byte(totpAEADLabel))
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("auth: derive totp key: %w", err)
	}
	return out, nil
}

func EncryptTOTPSecret(key []byte, secret string) (string, error) {
	// A not-yet-enrolled account stores no ciphertext, so an empty secret stays empty (#337).
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

func DecryptTOTPSecret(key []byte, stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	// A caller must fail closed on any error here; a legacy cleartext row is a fault (#337).
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
