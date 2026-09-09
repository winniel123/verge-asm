// Package secretseal seals an operator-typed credential for a TEXT column under a
// labelled sub-key of the shared transcript-key volume, so Postgres holds ciphertext and
// never the key (ADR-0172 §2, #1679).
package secretseal

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	LabelChannelSecret   = "channel-secret-aead" // one label per purpose (ADR-0172 §1)
	LabelSSOClientSecret = "sso-client-secret-aead"
)

func DeriveKey(parent []byte, label string) ([]byte, error) {
	if len(parent) == 0 {
		return nil, errors.New("secretseal: empty parent key")
	}
	if label == "" {
		return nil, errors.New("secretseal: empty label")
	}
	// The nil salt keeps this deterministic, so a restart still opens values sealed earlier.
	out := make([]byte, chacha20poly1305.KeySize)
	r := hkdf.New(sha256.New, parent, nil, []byte(label))
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, fmt.Errorf("secretseal: derive %s: %w", label, err)
	}
	return out, nil
}

func Seal(key []byte, secret string) (string, error) {
	if secret == "" {
		return "", nil
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("secretseal: aead: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secretseal: nonce: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func Open(key []byte, stored string) (string, error) {
	if stored == "" {
		return "", nil
	}
	// A caller must fail closed on any error here; a legacy cleartext row is a fault (ADR-0172 §5).
	raw, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return "", fmt.Errorf("secretseal: decode: %w", err)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", fmt.Errorf("secretseal: aead: %w", err)
	}
	if len(raw) < aead.NonceSize() {
		return "", errors.New("secretseal: sealed value too short")
	}
	nonce, ct := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("secretseal: open: %w", err)
	}
	return string(plain), nil
}

func SealText(key []byte, raw string) (pgtype.Text, error) {
	if strings.TrimSpace(raw) == "" {
		return pgtype.Text{}, nil
	}
	sealed, err := Seal(key, raw)
	if err != nil {
		return pgtype.Text{}, err
	}
	return pgtype.Text{String: sealed, Valid: true}, nil
}

func OpenText(key []byte, stored pgtype.Text) ([]byte, error) {
	if !stored.Valid {
		return nil, nil
	}
	plain, err := Open(key, stored.String)
	if err != nil {
		return nil, err
	}
	return []byte(plain), nil
}
