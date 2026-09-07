package certcorpus

import (
	"crypto/dsa" //nolint:staticcheck // SA1019: ParseChainCert reads a DSA key's P and Q, so a witness needs the type.
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"testing"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// A scripted handshake bypasses ParseChainCert, so nothing else keeps a hand-written
// ChainCert inside the range the live parse can emit (ADR-0152 §1, #1439, #1505).

const sigAlgSweepBound = 64

func witnessKey(alg string, bits, paramN int) (any, bool) {
	// The witness carries fixed bytes: a corpus package draws no randomness (ADR-0142).
	switch alg {
	case "RSA":
		if bits < 1024 || paramN != 0 {
			return nil, false
		}
		n := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
		n.SetBit(n, 0, 1)
		return &rsa.PublicKey{N: n, E: 65537}, true
	case "ECDSA":
		if paramN != 0 {
			return nil, false
		}
		for _, c := range []elliptic.Curve{elliptic.P224(), elliptic.P256(), elliptic.P384(), elliptic.P521()} {
			if c.Params().BitSize == bits {
				return &ecdsa.PublicKey{Curve: c, X: c.Params().Gx, Y: c.Params().Gy}, true
			}
		}
		return nil, false
	case "Ed25519":
		if bits != 0 || paramN != 0 {
			return nil, false
		}
		return ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)), true
	case "DSA":
		if bits < 1 || paramN < 1 {
			return nil, false
		}
		return &dsa.PublicKey{Parameters: dsa.Parameters{
			P: new(big.Int).Lsh(big.NewInt(1), uint(bits-1)),
			Q: new(big.Int).Lsh(big.NewInt(1), uint(paramN-1)),
		}}, true
	}
	return nil, false
}

func sigAlgsYielding(key any, want co.ChainCert) []x509.SignatureAlgorithm {
	var out []x509.SignatureAlgorithm
	// The bound runs past crypto/x509's last constant, so a new one needs no edit here.
	for i := 0; i < sigAlgSweepBound; i++ {
		a := x509.SignatureAlgorithm(i)
		got := co.ParseChainCert(&x509.Certificate{SignatureAlgorithm: a, PublicKey: key})
		if got.KeyAlg == want.KeyAlg && got.KeyBits == want.KeyBits &&
			got.KeyParamN == want.KeyParamN && got.SigDigest == want.SigDigest {
			out = append(out, a)
		}
	}
	return out
}

func signableBy(key any, algs []x509.SignatureAlgorithm) bool {
	c := &x509.Certificate{PublicKey: key}
	for _, a := range algs {
		err := c.CheckSignature(a, nil, nil)
		var insecure x509.InsecureAlgorithmError
		if errors.Is(err, x509.ErrUnsupportedAlgorithm) || errors.As(err, &insecure) {
			continue
		}
		return true
	}
	return false
}

func scriptedChainCerts(r Row) []struct {
	endpoint string
	index    int
	cert     co.ChainCert
} {
	var out []struct {
		endpoint string
		index    int
		cert     co.ChainCert
	}
	if r.Step.Handshake == nil {
		return out
	}
	keys := make([]string, 0, len(r.Step.Handshake.byEndpoint))
	for k := range r.Step.Handshake.byEndpoint {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for i, cc := range r.Step.Handshake.byEndpoint[k].ChainCerts {
			out = append(out, struct {
				endpoint string
				index    int
				cert     co.ChainCert
			}{k, i, cc})
		}
	}
	return out
}

func unreachableWhy(cc co.ChainCert) string {
	if cc.SelfSignatureVerifies == nil {
		return "self_sig_verifies is unset, and ParseChainCert always sets it"
	}
	key, ok := witnessKey(cc.KeyAlg, cc.KeyBits, cc.KeyParamN)
	if !ok {
		return fmt.Sprintf("no public key yields key_alg=%q key_bits=%d key_param_n=%d",
			cc.KeyAlg, cc.KeyBits, cc.KeyParamN)
	}
	algs := sigAlgsYielding(key, cc)
	if len(algs) == 0 {
		return fmt.Sprintf("no x509.SignatureAlgorithm makes ParseChainCert emit key_alg=%q key_bits=%d key_param_n=%d sig_digest=%q",
			cc.KeyAlg, cc.KeyBits, cc.KeyParamN, cc.SigDigest)
	}
	if !*cc.SelfSignatureVerifies {
		return ""
	}
	if cc.Subject != cc.Issuer {
		return fmt.Sprintf("self_sig_verifies=true beside subject %q != issuer %q, a pair selfSignedOf() reads as not self-signed",
			cc.Subject, cc.Issuer)
	}
	// A key family that cannot sign the digest at all still passes: CheckSignature
	// reports that mismatch through an error crypto/x509 exports no handle on (#1505).
	if !signableBy(key, algs) {
		return fmt.Sprintf("self_sig_verifies=true beside sig_digest=%q, which CheckSignature refuses for a %s key",
			cc.SigDigest, cc.KeyAlg)
	}
	return ""
}

func TestScriptedChainCertsStayInTheLiveParseRange(t *testing.T) {
	for _, r := range Rows {
		for _, s := range scriptedChainCerts(r) {
			if why := unreachableWhy(s.cert); why != "" {
				t.Errorf("%s %s chain_certs[%d]: %s", r.Golden, s.endpoint, s.index, why)
			}
		}
	}
}

func TestReachabilityGuardRejectsWhatTheLiveParseCannotEmit(t *testing.T) {
	// Without these the guard could pass by asserting nothing (#1505).
	cases := []struct {
		name string
		cert co.ChainCert
	}{
		{"unset self_sig", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"}},
		{"lowercased digest", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "sha-256"}},
		{"digest no switch names", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-224"}},
		{"key_alg no branch emits", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: b(false), KeyAlg: "RSA-2048", KeyBits: 2048, SigDigest: "SHA-256"}},
		{"ECDSA off every curve", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: b(false), KeyAlg: "ECDSA", KeyBits: 2048, SigDigest: "SHA-256"}},
		{"Ed25519 carrying bits", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: b(false), KeyAlg: "Ed25519", KeyBits: 256, SigDigest: "Ed25519"}},
		{"key_param_n on an RSA key", co.ChainCert{Subject: "CN=l", Issuer: "CN=ca", SelfSignatureVerifies: b(false), KeyAlg: "RSA", KeyBits: 2048, KeyParamN: 224, SigDigest: "SHA-256"}},
		{"self_sig true, issuer differs", co.ChainCert{Subject: "CN=root", Issuer: "CN=other", SelfSignatureVerifies: b(true), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "SHA-256"}},
		{"self_sig true over MD5", co.ChainCert{Subject: "CN=root", Issuer: "CN=root", SelfSignatureVerifies: b(true), KeyAlg: "RSA", KeyBits: 2048, SigDigest: "MD5"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if why := unreachableWhy(tc.cert); why == "" {
				t.Errorf("the guard accepted %+v, which ParseChainCert cannot emit", tc.cert)
			}
		})
	}
}
