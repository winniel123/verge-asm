// Package tlsacceptance is the `tls-acceptance` Derivation leaf: it enumerates which
// protocol versions and TLS 1.0–1.2 cipher suites a listener ACCEPTS, on its own weekly
// Scan rather than riding the `certificate` handshake (v1 spec §3.4, ADR-0028, #197).
package tlsacceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
)

// Widening the candidate set Breaks every certificate timeline too (ADR-0008, ADR-0021).

const Version = "tls-acceptance/v1"

const Kind = "tls-acceptance"

const Facet = "tls-acceptance"

// A 1.0 accept reads the v1 signal tls-1.0-accepted (measurement-offers §1.2).

const (
	TLS10 = tlsoffer.TLS10
	TLS11 = tlsoffer.TLS11
	TLS12 = tlsoffer.TLS12
	TLS13 = tlsoffer.TLS13
)

type CandidateSet struct {
	Versions                   []string `json:"versions"`
	Ciphers                    []string `json:"ciphers"`
	MaxHandshakesPerSecPerHost int      `json:"max_handshakes_per_sec_per_host"`
}

func DefaultCandidateSet() CandidateSet {
	// The certificate handshake offers the same list, so one edit moves both (ADR-0030 §3, #1680).
	return CandidateSet{
		Versions:                   tlsoffer.Versions(),
		Ciphers:                    tlsoffer.Ciphers(),
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
