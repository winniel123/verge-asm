// The `tls-acceptance` Scan enumerates version and cipher acceptance over every open
// `Service` (ADR-0028): that population plus the candidate set is the scope, never a port table.
// The enumeration is many handshakes per Service, which is the cost the weekly cadence buys.
package scan

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/vantageclass"
	"github.com/winniel123/verge-asm/internal/wire"
)

const TLSAcceptanceKind = tlsacceptance.Kind

type ReachedService struct {
	VantageID int64
	Address   string
	Port      uint16
}

type TLSAcceptanceJob struct {
	ScanID       int64
	VantageID    int64
	Vantage      string
	VantageClass string
	Kind         string
	Services     []tlsacceptance.ServiceTarget
	Candidates   tlsacceptance.CandidateSet
}

func BuildTLSAcceptanceJobs(scanID int64, estate custody.Estate, services []ReachedService, vantages []Vantage) []TLSAcceptanceJob {
	if len(services) == 0 || len(vantages) == 0 {
		return nil
	}
	byID := make(map[int64]Vantage, len(vantages))
	order := make([]int64, 0, len(vantages))
	for _, v := range vantages {
		if _, ok := byID[v.ID]; !ok {
			order = append(order, v.ID)
		}
		byID[v.ID] = v
	}

	covered := estate.CoversAddressScope
	vcByID := make(map[int64]custody.VantageClass, len(vantages))
	for _, v := range vantages {
		vcByID[v.ID] = vantageclass.Derive(v.Dialled, v.Egress, covered)
	}

	grouped := make(map[int64][]tlsacceptance.ServiceTarget)
	for _, s := range services {
		if _, ok := byID[s.VantageID]; !ok {
			continue
		}
		a, err := netip.ParseAddr(s.Address)
		if err != nil {
			continue
		}
		// A reached Service is a stale snapshot, so the ADR-0079 gate re-runs on the current Estate.
		if !estate.MayProbe(a, vcByID[s.VantageID]) {
			continue
		}
		grouped[s.VantageID] = append(grouped[s.VantageID], tlsacceptance.ServiceTarget{
			Address: s.Address,
			Port:    s.Port,
		})
	}

	var jobs []TLSAcceptanceJob
	for _, id := range order {
		svcs := grouped[id]
		if len(svcs) == 0 {
			continue
		}
		v := byID[id]
		jobs = append(jobs, TLSAcceptanceJob{
			ScanID:       scanID,
			VantageID:    id,
			Vantage:      v.Name,
			VantageClass: v.Class,
			Kind:         tlsacceptance.Kind,
			Services:     svcs,
			Candidates:   tlsacceptance.DefaultCandidateSet(),
		})
	}
	return jobs
}

func (j TLSAcceptanceJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(tlsacceptance.Scope{
		Vantage:      j.Vantage,
		VantageClass: j.VantageClass,
		Services:     j.Services,
		Candidates:   j.Candidates,
	})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal tls-acceptance scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: j.Kind, Scope: raw}, nil
}

func (j TLSAcceptanceJob) AttemptedScope() ([]byte, error) {
	// An offer of nineteen suites can never assert a twentieth was refused (CONTEXT.md).
	return json.Marshal(tlsAcceptanceScopeRecord{
		Vantage:    j.Vantage,
		Services:   j.Services,
		Candidates: j.Candidates,
	})
}

func (j TLSAcceptanceJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Candidates) }

func EmptyTLSAcceptanceScope(vantage string) ([]byte, error) {
	return json.Marshal(tlsAcceptanceScopeRecord{Vantage: vantage, Services: []tlsacceptance.ServiceTarget{}})
}

type tlsAcceptanceScopeRecord struct {
	Vantage    string                        `json:"vantage"`
	Services   []tlsacceptance.ServiceTarget `json:"services"`
	Candidates tlsacceptance.CandidateSet    `json:"candidates,omitempty"`
}
