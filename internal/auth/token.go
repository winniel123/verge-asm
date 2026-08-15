package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

// setupTokenBytes is the entropy of a generated setup token: 256 bits, so a
// token written to container logs on first boot is not guessable before an
// admin claims it.
const setupTokenBytes = 32

// NewSetupToken returns a fresh URL-safe single-use setup token. On first boot
// with no accounts, web generates one (or takes the operator's VERGE_SETUP_TOKEN
// override) and writes it to its logs; visiting the setup URL with it creates
// the first admin (v1 spec §4.3).
func NewSetupToken() (string, error) {
	buf := make([]byte, setupTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generate setup token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// TokensEqual compares two tokens in constant time. Empty is never equal to
// empty: a closed setup (no active token) must not be openable by presenting
// an empty token.
func TokensEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
