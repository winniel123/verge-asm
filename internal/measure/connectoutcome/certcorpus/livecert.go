package certcorpus

import (
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

//go:embed testdata/sha1_self_signed_root.pem
var sha1SelfSignedRootPEM []byte

func sha1SelfSignedRoot() co.ChainCert {
	// Checked in rather than generated: a corpus input draws no randomness (ADR-0142).
	blk, _ := pem.Decode(sha1SelfSignedRootPEM)
	if blk == nil {
		panic("certcorpus: sha1_self_signed_root.pem carries no PEM block")
	}
	c, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		panic(fmt.Sprintf("certcorpus: parse sha1_self_signed_root.pem: %v", err))
	}
	// The live adapter parses it: a scripted ChainCert cannot cover this (ADR-0152 §1, #1426).
	return co.ParseChainCert(c)
}
