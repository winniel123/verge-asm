// The hot Scan is the only tier that ships enabled: daily, over the verge-core port
// set, dispatched as the connect-outcome leaf, and kept additive to the dns and zone
// Scans so each tier registers independently (v1 spec §3.4, §3.5).
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/vantageclass"
	"github.com/winniel123/verge-asm/internal/vergecore"
	"github.com/winniel123/verge-asm/internal/wire"
)

const HotKind = "hot"

// The single admitted address rides as a one-element slice so the wire scope stays a list (ADR-0150).

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

func BuildHotJobs(scanID int64, estate custody.Estate, addrs iter.Seq[netip.Addr], vantages []Vantage, core vergecore.List) iter.Seq[HotJob] {
	return func(yield func(HotJob) bool) {
		tcp := portsOf(core.TCPProbed())
		udp := portsOf(core.UDPRecorded())
		if len(tcp) == 0 || len(vantages) == 0 {
			return
		}

		// The class is derived from presented-address facts, never the vestigial column (#709, #711).
		covered := estate.CoversAddressScope
		classes := make([]custody.VantageClass, len(vantages))
		for i, v := range vantages {
			classes[i] = vantageclass.Derive(v.Dialled, v.Egress, covered)
		}

		for a := range addrs {
			for i, v := range vantages {
				// The gate applies here at dispatch, so no prober gets a target it may not probe (ADR-0019).
				if !estate.MayProbe(a, classes[i]) {
					continue
				}
				// A Batch carries exactly one address, which is the execution gap ADR-0127 names (ADR-0005).
				job := HotJob{
					ScanID:       scanID,
					VantageID:    v.ID,
					Vantage:      v.Name,
					VantageClass: string(classes[i]),
					Kind:         connectoutcome.Kind,
					Addresses:    []string{a.Unmap().String()},
					TCPPorts:     tcp,
					UDPPorts:     udp,
					Profile:      connectoutcome.DefaultProfile(),
				}
				if !yield(job) {
					return
				}
			}
		}
	}
}

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

func (j HotJob) AttemptedScope() ([]byte, error) {
	// A UDP pair is recorded-not-probed, so it must not read as an absence we measured (v1 spec §4.1).
	return json.Marshal(hotScopeRecord{
		Vantage:   j.Vantage,
		Addresses: j.Addresses,
		TCPPorts:  j.TCPPorts,
		UDPPorts:  j.UDPPorts,
	})
}

func (j HotJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Profile) }

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
