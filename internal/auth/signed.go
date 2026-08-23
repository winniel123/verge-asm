package auth

import (
	"crypto/hmac"
	"encoding/base64"
	"strings"
)

// Sign appends an HMAC-SHA256 tag over a domain label and an arbitrary payload,
// producing a "<payload>.<mac>" token with both halves base64url, for callers that
// carry their own short-lived signed state rather than a Session. The domain is mixed
// into the MAC (never emitted), so a token signed for one domain does not verify under
// another even though both use the same key: it is the type tag that keeps, say, an
// OIDC login-transaction cookie from ever verifying as some other signed value, the
// way Session carries its Kind. The OIDC transaction (state / nonce / PKCE verifier)
// rides such a cookie between the redirect to the identity provider and the callback,
// so the callback can trust the state it echoes back was minted here and not forged.
func Sign(key []byte, domain string, payload []byte) string {
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	return b64 + "." + tag(key, domain+"."+b64)
}

// Verify checks the HMAC tag on a token produced by Sign under the SAME domain and
// returns the payload. A malformed, wrongly-signed, or wrong-domain token yields
// ErrInvalidSession — the same opaque error the session path uses, so a caller cannot
// distinguish a forged tag from a truncated token or a domain mismatch. It does NOT
// check expiry: the caller carries and checks its own deadline inside the payload.
func Verify(key []byte, domain, token string) ([]byte, error) {
	b64, mac, ok := strings.Cut(token, ".")
	if !ok {
		return nil, ErrInvalidSession
	}
	if !hmac.Equal([]byte(mac), []byte(tag(key, domain+"."+b64))) {
		return nil, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return nil, ErrInvalidSession
	}
	return payload, nil
}
