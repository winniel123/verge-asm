// Package resolutionwalk is the `resolution-walk` Derivation leaf inside the
// shared measurement binary (v1 spec §3.3). It decides the `resolution`
// facet's value — Resolved │ NoData │ NameError │ Lame │ Gap — plus the
// delegation walk's per-nameserver serves/does-not-serve RRset, from two
// queries read off two different peers (ADR-0070): the declared query path
// for Resolved/NoData/NameError, and the delegation walk for Lame.
//
// Every offer the leaf makes — the qtype set, the EDNS option set and buffer
// size, and the DNS transport/fallback policy — is enumerated here and
// recorded on the Batch by content, never taken from a library default
// (ADR-0025; docs/spec/measurement-offers.md). None is operator-configurable.
package resolutionwalk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Version is the leaf's Derivation version (ADR-0008/ADR-0021). It moves on an
// output-affecting change and only on one, gated bidirectionally by the golden
// corpus (§4.4). Bumping it without a moved corpus row, a changed declared
// parameter, or a registered uncovered move fails CI.
const Version = "resolution-walk/v1"

// Qtype is a DNS query type the leaf asks. The set is a declared parameter,
// recorded on the Batch by content.
type Qtype string

// The seven qtypes v1 queries, explicitly and never as ANY
// (docs/spec/measurement-offers.md §2). A is first so the address set the
// membership predicate reads is stable in the record.
const (
	QtypeA     Qtype = "A"
	QtypeAAAA  Qtype = "AAAA"
	QtypeCNAME Qtype = "CNAME"
	QtypeNS    Qtype = "NS"
	QtypeSOA   Qtype = "SOA"
	QtypeMX    Qtype = "MX"
	QtypeTXT   Qtype = "TXT"
)

// EDNSOffer is the EDNS(0) option set and advertised buffer size the leaf puts
// on the wire (measurement-offers.md §4). A declared parameter: under the rule
// that a truncated answer is never a value, none of it can silently move a
// value, only how often the §5 fallback path is taken.
type EDNSOffer struct {
	Version      int  `json:"version"`         // EDNS(0), version 0
	UDPBufSize   int  `json:"udp_buf_size"`    // 1232, the DNS Flag Day 2020 position
	DNSSECOK     bool `json:"dnssec_ok"`       // DO bit — clear in v1
	Cookie       bool `json:"cookie"`          // RFC 7873 cookie sent, one BADCOOKIE retry
	ClientSubnet bool `json:"client_subnet"`   // ECS — never sent, in either form (ADR-0025)
}

// TransportPolicy is the DNS transport and fallback policy (measurement-offers.md
// §5). A declared parameter of resolution-walk. The TCP fallback edge it produces
// is Gap → value, which ADR-0014 rules is not `revealed`.
type TransportPolicy struct {
	UDPAttempts   int  `json:"udp_attempts"`    // initial plus retries
	TCPAttempts   int  `json:"tcp_attempts"`    // after UDP exhausted or TC=1
	FallbackOnTC  bool `json:"fallback_on_tc"`  // retry over TCP when TC bit set
	EDNSlessRetry bool `json:"ednsless_retry"`  // retry without OPT on FORMERR
}

// Offers is the complete set of offers the leaf makes on one Batch, recorded by
// content. It is the leaf's declared-parameter set for the golden-corpus gate:
// a change here is a declared-parameter change that may justify a Version bump.
type Offers struct {
	Qtypes    []Qtype         `json:"qtypes"`
	EDNS      EDNSOffer       `json:"edns"`
	Transport TransportPolicy `json:"transport"`
}

// DefaultOffers is the v1 shipped offer set. It is authored here, not defaulted
// by any library, so the job spec carries it explicitly to the leaf and the
// Batch records exactly what went on the wire.
func DefaultOffers() Offers {
	return Offers{
		Qtypes: []Qtype{QtypeA, QtypeAAAA, QtypeCNAME, QtypeNS, QtypeSOA, QtypeMX, QtypeTXT},
		EDNS: EDNSOffer{
			Version:      0,
			UDPBufSize:   1232,
			DNSSECOK:     false,
			Cookie:       true,
			ClientSubnet: false,
		},
		Transport: TransportPolicy{
			UDPAttempts:   2,
			TCPAttempts:   1,
			FallbackOnTC:  true,
			EDNSlessRetry: true,
		},
	}
}

// Digest is a stable content hash of the offers, used by the golden-corpus lock
// to bind a declared-parameter change to a Version bump. Canonical JSON keys are
// sorted by encoding/json, so the digest moves exactly when a recorded offer
// moves.
func (o Offers) Digest() string {
	b, err := json.Marshal(o)
	if err != nil {
		// Offers is a fixed struct of JSON-safe fields; marshalling cannot fail.
		panic("resolutionwalk: marshal offers: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
