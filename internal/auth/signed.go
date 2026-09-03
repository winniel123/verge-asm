package auth

import (
	"crypto/hmac"
	"encoding/base64"
	"strings"
)

func Sign(key []byte, domain string, payload []byte) string {
	// The domain is mixed in and never emitted, so one domain's token cannot verify under another.
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	return b64 + "." + tag(key, domain+"."+b64)
}

func Verify(key []byte, domain, token string) ([]byte, error) {
	// Expiry is not checked here; the caller carries its own deadline inside the payload.
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
