// Package resolutionwalk is the `resolution-walk` Derivation leaf inside the
// shared measurement binary (v1 spec §3.3). It reads two peers (ADR-0070) and
// enumerates every offer it puts on the wire, never a library default (ADR-0025).
package resolutionwalk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const Version = "resolution-walk/v1" // moves only with a moved golden row (ADR-0021)

type Qtype string

const (
	QtypeA     Qtype = "A"
	QtypeAAAA  Qtype = "AAAA"
	QtypeCNAME Qtype = "CNAME"
	QtypeNS    Qtype = "NS"
	QtypeSOA   Qtype = "SOA"
	QtypeMX    Qtype = "MX"
	QtypeTXT   Qtype = "TXT"
)

type EDNSOffer struct {
	Version      int  `json:"version"`
	UDPBufSize   int  `json:"udp_buf_size"` // 1232, the DNS Flag Day 2020 position
	DNSSECOK     bool `json:"dnssec_ok"`
	Cookie       bool `json:"cookie"`        // RFC 7873 cookie sent, one BADCOOKIE retry
	ClientSubnet bool `json:"client_subnet"` // ECS — never sent, in either form (ADR-0025)
}

type TransportPolicy struct {
	UDPAttempts   int  `json:"udp_attempts"`
	TCPAttempts   int  `json:"tcp_attempts"`
	FallbackOnTC  bool `json:"fallback_on_tc"`
	EDNSlessRetry bool `json:"ednsless_retry"`
}

type Offers struct {
	Qtypes    []Qtype         `json:"qtypes"`
	EDNS      EDNSOffer       `json:"edns"`
	Transport TransportPolicy `json:"transport"`
}

func DefaultOffers() Offers {
	return Offers{
		// Enumerated explicitly, never ANY, A first for a stable set (measurement-offers.md §2).
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

func (o Offers) Digest() string {
	// encoding/json sorts object keys, so the digest moves exactly when an offer moves.
	b, err := json.Marshal(o)
	if err != nil {
		panic("resolutionwalk: marshal offers: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
