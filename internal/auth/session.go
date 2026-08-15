package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Kind distinguishes a fully authenticated session cookie from the short-lived
// intermediate cookie issued between a correct password and a TOTP code. The
// two are signed with the same key, so the kind is inside the signed payload:
// a KindPending cookie must never be accepted where a KindSession one is
// required, or password-only would defeat TOTP.
type Kind string

const (
	// KindSession is a completed login: the bearer is authenticated.
	KindSession Kind = "session"
	// KindPending is a half-login: the password was correct and a TOTP code
	// is still owed. It authorises only the TOTP-completion step.
	KindPending Kind = "totp"
)

// Session is the claim carried by a signed cookie. It holds no role: the role
// is read from the account row on every request, so a role change or an
// account deletion takes effect immediately rather than at cookie expiry.
type Session struct {
	AccountID int64     `json:"aid"`
	Kind      Kind      `json:"knd"`
	ExpiresAt time.Time `json:"exp"`
}

// ErrInvalidSession is returned by VerifySession for any token that is
// malformed, wrongly signed, expired, or of the wrong kind. It is deliberately
// single: a caller must not be able to distinguish "expired" from "forged"
// from "wrong kind" and leak which check failed.
var ErrInvalidSession = errors.New("auth: invalid session")

// SignSession encodes s and appends an HMAC-SHA256 tag over the encoding,
// producing a "<payload>.<mac>" token where both halves are base64url.
func SignSession(key []byte, s Session) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("auth: marshal session: %w", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	return b64 + "." + tag(key, b64), nil
}

// VerifySession checks the signature, expiry, and kind of token and returns
// the carried session. now is passed in so callers (and tests) control the
// clock. Every failure mode collapses to ErrInvalidSession.
func VerifySession(key []byte, token string, want Kind, now time.Time) (Session, error) {
	b64, mac, ok := strings.Cut(token, ".")
	if !ok {
		return Session{}, ErrInvalidSession
	}
	if !hmac.Equal([]byte(mac), []byte(tag(key, b64))) {
		return Session{}, ErrInvalidSession
	}
	payload, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return Session{}, ErrInvalidSession
	}
	var s Session
	if err := json.Unmarshal(payload, &s); err != nil {
		return Session{}, ErrInvalidSession
	}
	if s.Kind != want || !now.Before(s.ExpiresAt) {
		return Session{}, ErrInvalidSession
	}
	return s, nil
}

// tag returns the base64url HMAC-SHA256 of msg under key.
func tag(key []byte, msg string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
