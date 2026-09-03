package vantage

import (
	"errors"
	"net"

	"golang.org/x/crypto/ssh"
)

var ErrHostKeyMismatch = errors.New("vantage: host key mismatch — refusing to re-trust")

type HostKeyResult int

const (
	HostKeyFirstUse HostKeyResult = iota
	HostKeyMatch
	HostKeyMismatch
)

func EncodeHostKey(key ssh.PublicKey) string {
	return AuthorizedKey(key)
}

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
			// The caller marks the vantage unavailable rather than silently re-trust (v1 spec §4.2).
			return ErrHostKeyMismatch
		}
	}
}
