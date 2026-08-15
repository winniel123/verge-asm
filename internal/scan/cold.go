// This file builds the `cold` Scan (v1 spec §3.4, ADR-0044): the full 1–65535
// TCP port tier, monthly, opt-in **per `Seed` scope**. It ships configured and
// **disabled** with an empty scope list and never runs unasked — not at
// onboarding, not on config save — because a one-off measurement has no cadence
// and therefore no currency bound (ADR-0044). The only expressible form of
// "unasked" is shipping this Scan enabled at monthly cadence, and that is the
// option ADR-0044 refused: the tier is enabled by opting a `Seed` scope in, and
// then runs only on its own configured cadence.
//
// Cold REUSES the `connect-outcome` leaf (#194) rather than adding a new one:
// the difference from the `hot` Scan is the port set (the full range instead of
// `verge-core`) and the address scope (only Custody-admitted addresses that an
// opted-in `Seed` scope covers). It is gated by `Custody` exactly as `hot` is
// (ADR-0019): the opt-in gate can widen the tier but can never move an address
// past Custody, so no prober is ever handed a target it may not probe. It is
// kept additive to hot.go/zone.go — a new job builder and job type, no rewrite
// of scan.go's core.
package scan

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/wire"
)

// ColdKind is the DB Scan kind this file dispatches. The leaf it dispatches is
// `connect-outcome`, distinct from the Scan's kind exactly as `hot` dispatches
// the same leaf and `dns` dispatches `resolution-walk`.
const ColdKind = "cold"

// ColdPortLow and ColdPortHigh bound the cold Scan's port scope: the full TCP
// range (v1 spec §3.4). Unlike `hot`, which probes `verge-core`, `cold` probes
// every TCP port from 1 to 65535.
const (
	ColdPortLow  uint16 = 1
	ColdPortHigh uint16 = 65535
)

// ColdScope is the opted-in `Seed` scope the cold Scan probes within: the union
// of the address-scope Seeds opted in (as CIDR prefixes) and the addresses
// currently cited by names under an opted-in name-scope Seed. An address is in
// scope when it is enumerated by an opted-in prefix or listed in Addresses. An
// empty ColdScope is the shipped state — the tier configured with no scope — and
// produces no jobs at all.
type ColdScope struct {
	// AddressPrefixes are the opted-in address-scope Seeds. Every address inside
	// one is in scope, exactly as an address scope is its own enumeration.
	AddressPrefixes []netip.Prefix
	// Addresses are the addresses an opted-in name-scope Seed reaches: the
	// addresses names under its domain currently resolve to. The dispatcher folds
	// resolutions to this set; the pure builder only tests membership.
	Addresses map[netip.Addr]bool
}

// empty reports whether no Seed scope has opted in. An empty scope never fires:
// it is a legible configured-and-disabled state, not an error (ADR-0044).
func (c ColdScope) empty() bool {
	return len(c.AddressPrefixes) == 0 && len(c.Addresses) == 0
}

// contains reports whether an address falls inside the opted-in scope.
func (c ColdScope) contains(a netip.Addr) bool {
	a = a.Unmap()
	if c.Addresses[a] {
		return true
	}
	for _, p := range c.AddressPrefixes {
		if p.Contains(a) {
			return true
		}
	}
	return false
}

// ColdJob is one queue job the cold Scan produces: one Vantage, the Custody-
// admitted addresses within the opted-in scope for that Vantage's class, and the
// full TCP port range. It carries no UDP set — `connect-outcome` decides only a
// TCP `reachability` value (ADR-0083).
type ColdJob struct {
	ScanID       int64
	VantageID    int64
	Vantage      string
	VantageClass string
	Kind         string
	Addresses    []string
	TCPPorts     []uint16
	Profile      connectoutcome.SafetyProfile
}

// BuildColdJobs fans a cold Scan out into one job per Vantage over the addresses
// that are BOTH Custody-admitted for that Vantage's class AND covered by an
// opted-in `Seed` scope. The two gates compose: the opt-in gate bounds the tier
// to what the operator asked for, and the Custody gate (reused unchanged from
// `hot`, ADR-0019) refuses any address the derivation does not admit — a third-
// party address, or a non-globally-reachable one seen from an `internet`-class
// Vantage — so a wrongly opted-in address can never be smuggled past Custody. An
// empty opt-in scope, no address or no Vantage each yields no jobs: the shipped
// "configured and disabled" state fires nothing (ADR-0044).
func BuildColdJobs(scanID int64, estate custody.Estate, addrs []netip.Addr, vantages []Vantage, scope ColdScope) []ColdJob {
	if scope.empty() || len(addrs) == 0 || len(vantages) == 0 {
		return nil
	}
	tcp := coldTCPPorts()

	var jobs []ColdJob
	for _, v := range vantages {
		vc := custody.VantageClass(v.Class)
		admitted := make([]string, 0, len(addrs))
		for _, a := range addrs {
			if !scope.contains(a) {
				continue // not in an opted-in Seed scope — the tier was not asked for here
			}
			if estate.MayProbe(a, vc) {
				admitted = append(admitted, a.Unmap().String())
			}
		}
		if len(admitted) == 0 {
			continue
		}
		jobs = append(jobs, ColdJob{
			ScanID:       scanID,
			VantageID:    v.ID,
			Vantage:      v.Name,
			VantageClass: v.Class,
			Kind:         connectoutcome.Kind,
			Addresses:    admitted,
			TCPPorts:     tcp,
			Profile:      connectoutcome.DefaultProfile(),
		})
	}
	return jobs
}

// JobSpec renders a ColdJob into the wire JobSpec the connect-outcome prober
// reads on stdin. The full port range travels in the scope so the leaf connects
// to exactly the pairs the tier declares and the Batch records them by content.
func (j ColdJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(connectoutcome.Scope{
		Vantage:      j.Vantage,
		VantageClass: j.VantageClass,
		Addresses:    j.Addresses,
		TCPPorts:     j.TCPPorts,
		Profile:      j.Profile,
	})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal cold scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: j.Kind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the cold job set out to cover:
// the Vantage, the admitted addresses, and the full TCP port range as its
// bounds. The range is recorded as [low, high] rather than 65535 enumerated
// integers — it is the same closed statement, compactly — so a `Service` subject
// exists for every `(Address, port)` in the range and a pair we never connected
// to can never read as an absence we measured (v1 spec §4.1).
func (j ColdJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(coldScopeRecord{
		Vantage:       j.Vantage,
		Addresses:     j.Addresses,
		PortRangeLow:  ColdPortLow,
		PortRangeHigh: ColdPortHigh,
	})
}

// OffersJSON is the safety profile recorded on the Batch by content.
func (j ColdJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Profile) }

// EmptyColdScope is what a dead-lettered cold Batch records — never the
// attempted scope, which would manufacture reachability absences it never
// measured across the whole range (v1 spec §4.1).
func EmptyColdScope(vantage string) ([]byte, error) {
	return json.Marshal(coldScopeRecord{Vantage: vantage, Addresses: []string{}})
}

type coldScopeRecord struct {
	Vantage       string   `json:"vantage"`
	Addresses     []string `json:"addresses"`
	PortRangeLow  uint16   `json:"port_range_low,omitempty"`
	PortRangeHigh uint16   `json:"port_range_high,omitempty"`
}

// coldTCPPorts is the full 1–65535 TCP range the cold Scan probes. It is built
// fresh per fan-out; the slice is the aperture the tier declares, and it travels
// whole in the job spec so the leaf iterates exactly it.
func coldTCPPorts() []uint16 {
	ports := make([]uint16, 0, int(ColdPortHigh))
	for p := int(ColdPortLow); p <= int(ColdPortHigh); p++ {
		ports = append(ports, uint16(p))
	}
	return ports
}
