// Package tlsacceptance is the `tls-acceptance` Derivation leaf: it enumerates which
// protocol versions and TLS 1.0–1.2 cipher suites a listener ACCEPTS, on its own weekly
// Scan rather than riding the `certificate` handshake (v1 spec §3.4, ADR-0028, #197).
package tlsacceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Widening the candidate set Breaks every certificate timeline too (ADR-0008, ADR-0021).

const Version = "tls-acceptance/v1"

const Kind = "tls-acceptance"

const Facet = "tls-acceptance"

// A 1.0 accept reads the v1 signal tls-1.0-accepted (measurement-offers §1.2).

// 1.2 and 1.3 are offered so a correctly-configured listener is never misfiled TLSRefused.

const (
	TLS10 = "1.0"
	TLS11 = "1.1"
	TLS12 = "1.2"
	TLS13 = "1.3"
)

// The set is literal so a Go upgrade cannot silently widen the offer (measurement-offers §1.4).

// Go ignores Config.CipherSuites under TLS 1.3, so no 1.3 suite may sit in the value (§1.3).

type CandidateSet struct {
	Versions                   []string `json:"versions"`
	Ciphers                    []string `json:"ciphers"`
	MaxHandshakesPerSecPerHost int      `json:"max_handshakes_per_sec_per_host"`
}

func DefaultCandidateSet() CandidateSet {
	// measurement-offers §1.2 (versions) and §1.3 (the nineteen suites), verbatim.
	return CandidateSet{
		Versions: []string{TLS10, TLS11, TLS12, TLS13},
		Ciphers: []string{
			// Limb 1 — accepting any of these is itself a finding (measurement-offers §1.3).
			"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
			"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
			"TLS_RSA_WITH_AES_128_CBC_SHA",
			"TLS_RSA_WITH_AES_256_CBC_SHA",
			"TLS_RSA_WITH_AES_128_CBC_SHA256",
			"TLS_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
			"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
			"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
			"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
			"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
			// Limb 2 — their absence would make the measurement false: the modal 1.2 set (§1.3).
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
			"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
		},
		MaxHandshakesPerSecPerHost: 5,
	}
}

func (c CandidateSet) Digest() string {
	// The golden-corpus lock reads this to bind a declared-parameter change to a Version bump.
	b, err := json.Marshal(c)
	if err != nil {
		panic("tlsacceptance: marshal candidate set: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
