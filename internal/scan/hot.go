// This file builds the `hot` Scan (v1 spec §3.4/§3.5): daily, the only tier
// that ships enabled, its scope the `verge-core` port set, dispatched as the
// `connect-outcome` leaf. It is gated TOTALLY by `Custody` (ADR-0019): the hot
// Scan probes ONLY the addresses the `Custody` derivation admits, per Vantage
// class, and no other. It is kept additive to the dns and zone Scans — a new
// job builder and job type, no rewrite of scan.go's core — so the measurement
// binary and the queue register the tiers independently.
package scan

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/vantageclass"
	"github.com/winniel123/verge-asm/internal/vergecore"
	"github.com/winniel123/verge-asm/internal/wire"
)

// HotKind is the DB Scan kind this file dispatches. The leaf it dispatches is
// `connect-outcome`, distinct from the Scan's kind exactly as `dns` dispatches
// `resolution-walk`.
const HotKind = "hot"

// HotJob is one queue job the hot Scan produces: one Vantage, the addresses
// Custody admitted for that Vantage's class, and the `verge-core` TCP/UDP sets.
// Only the TCP pairs are probed; the UDP pairs travel so the Batch records them
// in scope (open or closed for TCP; recorded-not-probed for UDP).
type HotJob struct {
	ScanID       int64
	VantageID    int64
	Vantage      string
	VantageClass string
	Kind         string
	Addresses    []string
	TCPPorts     []uint16
	UDPPorts     []uint16
	Profile      connectoutcome.SafetyProfile
}

// BuildHotJobs fans a hot Scan out into one job per Vantage over the addresses
// Custody admits for that Vantage's class. It is the sole place the gate is
// applied at dispatch: an address the derivation refuses — a `third-party`
// address, or a non-globally-reachable one seen from an `internet`-class Vantage
// — never enters a job, so no prober is ever handed a target it may not probe
// (ADR-0019). A Vantage with no admitted address yields no job, a legible empty
// scope rather than an error.
func BuildHotJobs(scanID int64, estate custody.Estate, addrs []netip.Addr, vantages []Vantage, core vergecore.List) []HotJob {
	tcp := portsOf(core.TCPProbed())
	udp := portsOf(core.UDPRecorded())
	if len(tcp) == 0 || len(addrs) == 0 || len(vantages) == 0 {
		return nil
	}

	// Class is DERIVED per batch from each vantage's presented-address facts against the
	// declared address scopes (#709), never the vestigial column: covered is the
	// address-scope-only predicate (#711) the same Estate carries.
	covered := estate.CoversAddressScope
	var jobs []HotJob
	for _, v := range vantages {
		vc := vantageclass.Derive(v.Dialled, v.Egress, covered)
		admitted := make([]string, 0, len(addrs))
		for _, a := range addrs {
			if estate.MayProbe(a, vc) {
				admitted = append(admitted, a.Unmap().String())
			}
		}
		if len(admitted) == 0 {
			continue
		}
		jobs = append(jobs, HotJob{
			ScanID:       scanID,
			VantageID:    v.ID,
			Vantage:      v.Name,
			VantageClass: string(vc),
			Kind:         connectoutcome.Kind,
			Addresses:    admitted,
			TCPPorts:     tcp,
			UDPPorts:     udp,
			Profile:      connectoutcome.DefaultProfile(),
		})
	}
	return jobs
}

// JobSpec renders a HotJob into the wire JobSpec the prober reads on stdin.
func (j HotJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(connectoutcome.Scope{
		Vantage:      j.Vantage,
		VantageClass: j.VantageClass,
		Addresses:    j.Addresses,
		TCPPorts:     j.TCPPorts,
		UDPPorts:     j.UDPPorts,
		Profile:      j.Profile,
	})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal hot scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: j.Kind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the hot job set out to cover:
// the Vantage, the admitted addresses, and the `verge-core` TCP and UDP port
// sets. A `Service` subject exists for every `(Address, port, transport)` in
// this record — open or closed for TCP, recorded-not-probed for UDP — which is
// what the recorded Batch scope must state so a pair we never connected to can
// never read as an absence we measured (v1 spec §4.1).
func (j HotJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(hotScopeRecord{
		Vantage:   j.Vantage,
		Addresses: j.Addresses,
		TCPPorts:  j.TCPPorts,
		UDPPorts:  j.UDPPorts,
	})
}

// OffersJSON is the safety profile recorded on the Batch by content.
func (j HotJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Profile) }

// EmptyHotScope is what a dead-lettered hot Batch records — never the attempted
// scope, which would manufacture reachability absences it never measured.
func EmptyHotScope(vantage string) ([]byte, error) {
	return json.Marshal(hotScopeRecord{Vantage: vantage, Addresses: []string{}})
}

type hotScopeRecord struct {
	Vantage   string   `json:"vantage"`
	Addresses []string `json:"addresses"`
	TCPPorts  []uint16 `json:"tcp_ports,omitempty"`
	UDPPorts  []uint16 `json:"udp_ports,omitempty"`
}

func portsOf(pairs []vergecore.Pair) []uint16 {
	out := make([]uint16, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.Port)
	}
	return out
}
