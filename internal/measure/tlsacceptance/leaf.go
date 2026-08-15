// Package tlsacceptance is the `tls-acceptance` Derivation leaf inside the shared
// measurement binary (v1 spec §3.4, ADR-0028). It is the ENUMERATION exchange —
// distinct from the `certificate` handshake (#197), which reads only the chain
// presented on the single default handshake riding the `reachability` exchange.
// This leaf enumerates what a listener ACCEPTS: which protocol versions, and — for
// TLS 1.0–1.2 — which cipher suites, deciding the `tls-acceptance` facet for a
// `Service` (CONTEXT.md `tls-acceptance`).
//
// It is dispatched by its OWN Scan — the weekly `tls-acceptance` Scan over every
// open `Service`, an enumeration with NO port list (ADR-0028) — never riding
// another exchange. `certificate` and `tls-acceptance` are two exchanges against
// two subjects on two cadences; this leaf is the second of them.
//
// The candidate set is DECLARED here and recorded on the `Batch` by content
// (ADR-0025), never taken from the TLS library's defaults: a default is not a
// declaration, and one left in place hides a TLS-1.0-only listener as `NoTLS` on
// exactly the estate `tls-1.0-accepted` exists for. The set is LITERAL, not derived
// from the linked library, so a Go upgrade cannot silently widen or narrow the
// offer (measurement-offers §1.4); an offerability test pins that every declared
// candidate is offerable by the linked `crypto/tls`.
//
// The enumeration is behind an interface (Enumerator) and the accept-fold is pure,
// so the golden corpus drives the leaf against a scripted in-process enumerator —
// no network, no container, arch-neutral.
package tlsacceptance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Version is the leaf's Derivation version (ADR-0008/ADR-0021). It moves on an
// output-affecting change and only on one, gated bidirectionally by this leaf's
// own golden corpus (golden-corpus.md §4.4), separately from the sibling leaves so
// a break names its leaf. Widening the candidate set is such a change — it costs a
// `Break` on every `tls-acceptance` timeline AND every `certificate` timeline,
// since the one declared set is carried by both TLS exchanges (CONTEXT.md
// `tls-acceptance`).
const Version = "tls-acceptance/v1"

// Kind is the JobSpec.Kind that dispatches to this leaf, and — unlike the port
// tiers, whose Scan kind differs from the `connect-outcome` leaf they dispatch —
// it is also the DB kind of the Scan that produces it: the `tls-acceptance` Scan
// dispatches the `tls-acceptance` leaf, its own exchange on its own cadence.
const Kind = "tls-acceptance"

// Facet is the facet this leaf decides — the versions and suites a `Service`
// accepted, held on a `Service` subject (not an `Endpoint`: SNI is the subject of
// the `certificate` handshake, never a candidate here — measurement-offers §1.6).
const Facet = "tls-acceptance"

// Protocol-version candidates, TLS 1.0–1.3 (measurement-offers §1.2). Versions are
// enumerated one handshake each with the version pinned; acceptance of 1.0 reads
// the v1 signal `tls-1.0-accepted`, 1.2/1.3 are offered so a correctly-configured
// listener is not misfiled `TLSRefused`.
const (
	TLS10 = "1.0"
	TLS11 = "1.1"
	TLS12 = "1.2"
	TLS13 = "1.3"
)

// CandidateSet is the leaf's declared-parameter set: the exact enumeration offer
// (measurement-offers §1.2/§1.3), recorded on the `Batch` by content (ADR-0025) so
// the offer is legible and never a library default. None of it is
// operator-configurable in v1 (measurement-offers §1.7).
//
// Cipher suites are declared for TLS 1.0–1.2 ONLY: Go's `Config.CipherSuites` is
// documented as ignored for TLS 1.3, so the three TLS 1.3 suites are the library's
// choice and may not sit inside the value — a per-candidate negative over
// candidates we did not choose would move estate-wide on a library upgrade
// (measurement-offers §1.3). Under TLS 1.3 the leaf records that the version was
// accepted and nothing about suites.
type CandidateSet struct {
	// Versions is the ordered protocol-version candidate list, TLS 1.0–1.3.
	Versions []string `json:"versions"`
	// Ciphers is the cipher-suite candidate list by Go constant name, offered for
	// the TLS 1.0–1.2 handshakes only.
	Ciphers []string `json:"ciphers"`
	// MaxHandshakesPerSecPerHost records the per-host handshake pacing the
	// enumeration honours (#4 §6: 5 handshakes/s). It is recorded by content and,
	// like every other pacing knob, sits OUTSIDE the verdict — it changes the rate
	// and never which versions or suites a listener accepted (ADR-0021).
	MaxHandshakesPerSecPerHost int `json:"max_handshakes_per_sec_per_host"`
}

// DefaultCandidateSet is the v1 shipped enumeration offer — measurement-offers
// §1.2 (versions) and §1.3 (the nineteen TLS 1.0–1.2 suites, both limbs) exactly.
// Authored here, not defaulted by any library, so the job spec carries it to the
// leaf and the Batch records exactly what went on the wire.
func DefaultCandidateSet() CandidateSet {
	return CandidateSet{
		Versions: []string{TLS10, TLS11, TLS12, TLS13},
		Ciphers: []string{
			// Limb 1 — acceptance is a finding (measurement-offers §1.3).
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
			// Limb 2 — absence would make the measurement false (the modal 1.2 set).
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

// Digest is a stable content hash of the candidate set, used by the golden-corpus
// lock to bind a declared-parameter change to a Version bump. Widening the set is
// the one aperture change this leaf makes, so binding it to the version is what
// makes a silent widening impossible.
func (c CandidateSet) Digest() string {
	b, err := json.Marshal(c)
	if err != nil {
		panic("tlsacceptance: marshal candidate set: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
