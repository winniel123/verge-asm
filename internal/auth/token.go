package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

const setupTokenBytes = 32 // 256 bits; it is written to first-boot logs before an admin claims it

func NewSetupToken() (string, error) {
	buf := make([]byte, setupTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generate setup token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func TokensEqual(a, b string) bool {
	// A closed setup carries no token, and an empty token must never open it.
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
