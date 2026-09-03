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

type Kind string // same key signs both, so a pending cookie must never pass as a session

const (
	KindSession Kind = "session"
	KindPending Kind = "totp"
)

type Session struct { // no role: read per request, so a change takes effect before cookie expiry
	AccountID int64     `json:"aid"`
	Kind      Kind      `json:"knd"`
	ExpiresAt time.Time `json:"exp"`
	Token     string    `json:"tok,omitempty"` // empty on a pending cookie: no registry row (ADR-0117)
}

// Deliberately single, so no caller can learn which check failed and leak it.

var ErrInvalidSession = errors.New("auth: invalid session")

func SignSession(key []byte, s Session) (string, error) {
	// The plaintext token is safe inside the HMAC-signed payload, so it is not sealed (ADR-0117).
	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("auth: marshal session: %w", err)
	}
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	return b64 + "." + tag(key, b64), nil
}

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

func tag(key []byte, msg string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
