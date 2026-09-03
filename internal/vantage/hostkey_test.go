package vantage

import (
	"errors"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

func testHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}
	return signer.PublicKey()
}

func TestCheckHostKey(t *testing.T) {
	key := testHostKey(t)
	other := testHostKey(t)

	if got := CheckHostKey("", key); got != HostKeyFirstUse {
		t.Errorf("empty pin: got %v, want HostKeyFirstUse", got)
	}
	if got := CheckHostKey(EncodeHostKey(key), key); got != HostKeyMatch {
		t.Errorf("matching pin: got %v, want HostKeyMatch", got)
	}
	if got := CheckHostKey(EncodeHostKey(other), key); got != HostKeyMismatch {
		t.Errorf("differing pin: got %v, want HostKeyMismatch", got)
	}
}

func TestPinningCallbackPinsOnFirstUse(t *testing.T) {
	key := testHostKey(t)

	var pinned string
	cb := PinningHostKeyCallback("", func(encoded string) error {
		pinned = encoded
		return nil
	})
	if err := cb("host:22", &net.TCPAddr{}, key); err != nil {
		t.Fatalf("first-use callback returned error: %v", err)
	}
	if pinned != EncodeHostKey(key) {
		t.Errorf("first use pinned %q, want %q", pinned, EncodeHostKey(key))
	}
}

func TestPinningCallbackTrustsMatch(t *testing.T) {
	key := testHostKey(t)

	called := false
	cb := PinningHostKeyCallback(EncodeHostKey(key), func(string) error {
		called = true
		return nil
	})
	if err := cb("host:22", &net.TCPAddr{}, key); err != nil {
		t.Fatalf("matching callback returned error: %v", err)
	}
	if called {
		t.Error("onFirstUse called when a matching key was already pinned")
	}
}

func TestPinningCallbackRejectsMismatch(t *testing.T) {
	pinnedKey := testHostKey(t)
	presented := testHostKey(t)

	cb := PinningHostKeyCallback(EncodeHostKey(pinnedKey), func(string) error {
		t.Error("onFirstUse called on a mismatch — silent re-trust")
		return nil
	})
	err := cb("host:22", &net.TCPAddr{}, presented)
	if !errors.Is(err, ErrHostKeyMismatch) {
		t.Fatalf("mismatch error = %v, want ErrHostKeyMismatch", err)
	}
}
