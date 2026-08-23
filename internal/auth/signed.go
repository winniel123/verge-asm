package auth

import (
	"crypto/hmac"
	"encoding/base64"
	"strings"
)

// Sign appends an HMAC-SHA256 tag over an arbitrary payload, producing a
// "<payload>.<mac>" token with both halves base64url — the same construction
// SignSession uses, exposed for callers that carry their own short-lived signed
// state rather than a Session. The OIDC login transaction (state / nonce / PKCE
// verifier) rides such a cookie between the redirect to the identity provider and
// the callback: it is signed with the same key as the session cookie, so the
// callback can trust the state it echoes back was minted here and not forged.
func Sign(key, payload []byte) string {
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	return b64 + "." + tag(key, b64)
}

// Verify checks the HMAC tag on a token produced by Sign and returns the payload.
// A malformed or wrongly-signed token yields ErrInvalidSession — the same opaque
// error the session path uses, so a caller cannot distinguish a forged tag from a
// truncated token. It does NOT check expiry: the caller carries and checks its own
// deadline inside the payload, exactly as it chose the payload's shape.
func Verify(key []byte, token string) ([]byte, error) {
	b64, mac, ok := strings.Cut(token, ".")
	if !ok {
		return nil, ErrInvalidSession
	}
	if !hmac.Equal([]byte(mac), []byte(tag(key, b64))) {
		return nil, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return nil, ErrInvalidSession
	}
	return payload, nil
}
