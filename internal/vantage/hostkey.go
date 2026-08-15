package vantage

import (
	"errors"
	"net"

	"golang.org/x/crypto/ssh"
)

// ErrHostKeyMismatch is returned by the pinning callback when the host presents
// a key that differs from the one pinned on first connect. It is a hard failure:
// the caller marks the vantage unavailable rather than silently re-trusting the
// new key (v1 spec §4.2).
var ErrHostKeyMismatch = errors.New("vantage: host key mismatch — refusing to re-trust")

// HostKeyResult is the trust-on-first-use decision for a presented host key.
type HostKeyResult int

const (
	// HostKeyFirstUse means no key is pinned yet, so the presented key is
	// pinned on this first successful connection.
	HostKeyFirstUse HostKeyResult = iota
	// HostKeyMatch means the presented key equals the pinned key — trusted.
	HostKeyMatch
	// HostKeyMismatch means the presented key differs from the pinned key —
	// a hard failure, never a silent re-trust.
	HostKeyMismatch
)

// EncodeHostKey renders a host key as its known_hosts key field (type plus
// base64 body, no host prefix and no newline), the form pinned in the database
// and compared byte-for-byte on later connects.
func EncodeHostKey(key ssh.PublicKey) string {
	return AuthorizedKey(key)
}

// CheckHostKey is the pure trust-on-first-use decision. pinned is the stored
// known_hosts key field, empty when nothing is pinned yet; presented is the key
// the host offered this connection. It compares the encoded forms, so it is a
// test over the keys themselves and never over an incidental rendering.
func CheckHostKey(pinned string, presented ssh.PublicKey) HostKeyResult {
	switch {
	case pinned == "":
		return HostKeyFirstUse
	case pinned == EncodeHostKey(presented):
		return HostKeyMatch
	default:
		return HostKeyMismatch
	}
}

// PinningHostKeyCallback builds an ssh.HostKeyCallback enforcing
// trust-on-first-use against a pinned key.
//
//   - With no key pinned (pinned == ""), it pins the presented key by calling
//     onFirstUse with its encoded form and trusts the connection.
//   - With a pinned key that matches, it trusts silently.
//   - With a pinned key that differs, it returns ErrHostKeyMismatch and never
//     re-pins — the caller marks the vantage unavailable.
//
// It opens no connection itself, so a test drives it by invoking the returned
// callback with a generated ssh.PublicKey; no live SSH server is needed.
func PinningHostKeyCallback(pinned string, onFirstUse func(encoded string) error) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		switch CheckHostKey(pinned, key) {
		case HostKeyFirstUse:
			if onFirstUse == nil {
				return nil
			}
			return onFirstUse(EncodeHostKey(key))
		case HostKeyMatch:
			return nil
		default:
			return ErrHostKeyMismatch
		}
	}
}
