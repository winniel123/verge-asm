// Package vantage holds the deployment-free, pure logic for provisioning a
// prober: generating the SSH keypair whose public half is the only thing that
// ever leaves the worker volume, and the trust-on-first-use host-key decision
// that pins a key on first connect and refuses to silently re-trust a changed
// one (v1 spec §4.2, CONTEXT.md "Vantage").
//
// Nothing here opens a network connection, so every rule is unit-testable
// without a live SSH server: keypair generation is deterministic in shape, and
// the host-key decision is a pure function over the pinned and presented keys.
package vantage

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// Keypair is a freshly generated SSH keypair. PrivatePEM is the OpenSSH-format
// private key that must be written only to the worker volume and never leave
// the instance; PublicKey is the authorized_keys line — the sole half exposed
// on the web surface and installed on the prober host.
type Keypair struct {
	PrivatePEM []byte
	PublicKey  string
}

func Generate() (Keypair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Keypair{}, fmt.Errorf("vantage: generate ed25519 key: %w", err)
	}

	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return Keypair{}, fmt.Errorf("vantage: marshal private key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return Keypair{}, fmt.Errorf("vantage: derive public key: %w", err)
	}

	return Keypair{
		PrivatePEM: pem.EncodeToMemory(block),
		PublicKey:  AuthorizedKey(sshPub),
	}, nil
}

// PublicKeyFromPrivatePEM re-derives the authorized_keys public line from a
// stored private key. The worker uses it to republish a public key for a
// vantage whose private half is already on the volume but whose public half
// never reached the database — a crash between writing the key file and the
// database write must not strand the vantage without a rendered public key.
func PublicKeyFromPrivatePEM(privatePEM []byte) (string, error) {
	signer, err := ssh.ParsePrivateKey(privatePEM)
	if err != nil {
		return "", fmt.Errorf("vantage: parse private key: %w", err)
	}
	return AuthorizedKey(signer.PublicKey()), nil
}

func AuthorizedKey(key ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}
