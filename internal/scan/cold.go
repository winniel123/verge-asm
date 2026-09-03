// The `cold` full-range TCP tier ships configured and disabled, and never runs unasked (ADR-0044).
// A one-off sweep has no cadence and therefore no currency bound, which is why none is offered.
// Opting a `Seed` scope in enables it, after which it runs on its own cadence (v1 spec §3.4).
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/vantageclass"
	"github.com/winniel123/verge-asm/internal/wire"
)

const ColdKind = "cold"

const (
	ColdPortLow  uint16 = 1
	ColdPortHigh uint16 = 65535
)

type ColdScope struct {
	AddressPrefixes []netip.Prefix
	Addresses       map[netip.Addr]bool
}

func (c ColdScope) empty() bool {
	return len(c.AddressPrefixes) == 0 && len(c.Addresses) == 0
}

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

// No UDP set travels: `connect-outcome` decides only a TCP `reachability` value (ADR-0083).

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

func BuildColdJobs(scanID int64, estate custody.Estate, addrs iter.Seq[netip.Addr], vantages []Vantage, scope ColdScope) iter.Seq[ColdJob] {
	return func(yield func(ColdJob) bool) {
		if scope.empty() || len(vantages) == 0 {
			return
		}
		tcp := coldTCPPorts()

		covered := estate.CoversAddressScope
		classes := make([]custody.VantageClass, len(vantages))
		for i, v := range vantages {
			classes[i] = vantageclass.Derive(v.Dialled, v.Egress, covered)
		}

		for a := range addrs {
			if !scope.contains(a) {
				continue
			}
			for i, v := range vantages {
				// The opt-in gate widens the tier but never moves an address past Custody (ADR-0019).
				if !estate.MayProbe(a, classes[i]) {
					continue
				}
				// One address per Batch, so a dead-letter withdraws one address's scope (ADR-0005).
				job := ColdJob{
					ScanID:       scanID,
					VantageID:    v.ID,
					Vantage:      v.Name,
					VantageClass: string(classes[i]),
					Kind:         connectoutcome.Kind,
					Addresses:    []string{a.Unmap().String()},
					TCPPorts:     tcp,
					Profile:      connectoutcome.DefaultProfile(),
				}
				if !yield(job) {
					return
				}
			}
		}
	}
}

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

func (j ColdJob) AttemptedScope() ([]byte, error) {
	// The range is recorded as bounds, not 65535 integers: the same closed statement (v1 spec §4.1).
	return json.Marshal(coldScopeRecord{
		Vantage:       j.Vantage,
		Addresses:     j.Addresses,
		PortRangeLow:  ColdPortLow,
		PortRangeHigh: ColdPortHigh,
	})
}

func (j ColdJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Profile) }

func EmptyColdScope(vantage string) ([]byte, error) {
	return json.Marshal(coldScopeRecord{Vantage: vantage, Addresses: []string{}})
}

type coldScopeRecord struct {
	Vantage       string   `json:"vantage"`
	Addresses     []string `json:"addresses"`
	PortRangeLow  uint16   `json:"port_range_low,omitempty"`
	PortRangeHigh uint16   `json:"port_range_high,omitempty"`
}

func coldTCPPorts() []uint16 {
	ports := make([]uint16, 0, int(ColdPortHigh))
	for p := int(ColdPortLow); p <= int(ColdPortHigh); p++ {
		ports = append(ports, uint16(p))
	}
	return ports
}
