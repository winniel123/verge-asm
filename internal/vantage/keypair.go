// Package vantage holds the deployment-free logic for provisioning a prober:
// SSH keypair generation and the trust-on-first-use host-key decision (v1 spec
// §4.2, CONTEXT.md "Vantage"). Nothing here opens a network connection.
package vantage

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Keypair struct {
	PrivatePEM []byte // never leaves the worker volume (v1 spec §4.2)
	PublicKey  string // the only half exposed on the web surface (v1 spec §4.2)
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

func PublicKeyFromPrivatePEM(privatePEM []byte) (string, error) {
	// A crash between the key file write and the database write must not strand a vantage.
	signer, err := ssh.ParsePrivateKey(privatePEM)
	if err != nil {
		return "", fmt.Errorf("vantage: parse private key: %w", err)
	}
	return AuthorizedKey(signer.PublicKey()), nil
}

func AuthorizedKey(key ssh.PublicKey) string {
	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
}
