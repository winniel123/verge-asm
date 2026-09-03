package vantage

import (
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateProducesUsableKeypair(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	if !strings.HasPrefix(kp.PublicKey, "ssh-ed25519 ") {
		t.Errorf("public key is not an ed25519 authorized_keys line: %q", kp.PublicKey)
	}
	if strings.ContainsAny(kp.PublicKey, "\n\r") {
		t.Errorf("public key carries a newline: %q", kp.PublicKey)
	}

	signer, err := ssh.ParsePrivateKey(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("private PEM does not parse: %v", err)
	}
	if got := AuthorizedKey(signer.PublicKey()); got != kp.PublicKey {
		t.Errorf("public half %q does not match private half %q", kp.PublicKey, got)
	}

	if strings.Contains(string(kp.PrivatePEM), kp.PublicKey) {
		t.Error("private PEM contains the public key line verbatim")
	}
}

func TestGenerateIsUnique(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if a.PublicKey == b.PublicKey {
		t.Error("two generated keypairs share a public key")
	}
}

func TestPublicKeyFromPrivatePEM(t *testing.T) {
	kp, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	got, err := PublicKeyFromPrivatePEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivatePEM: %v", err)
	}
	if got != kp.PublicKey {
		t.Errorf("re-derived public key %q, want %q", got, kp.PublicKey)
	}

	if _, err := PublicKeyFromPrivatePEM([]byte("not a key")); err == nil {
		t.Error("PublicKeyFromPrivatePEM accepted garbage")
	}
}
