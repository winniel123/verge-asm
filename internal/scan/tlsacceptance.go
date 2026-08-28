// This file builds the `tls-acceptance` Scan (v1 spec §3.4, ADR-0028): WEEKLY,
// over EVERY open `Service`, offering the full TLS candidate set — an enumeration,
// NOT a port tier. It has NO port list at all: the Services it enumerates carry
// their own ports, inherited from the `reachability` timelines that found them
// open, because the Scan's scope is the open `Service` population together with the
// candidate set (ADR-0028), never a curated implicit-TLS port table.
//
// It is a `Scan` of its OWN — the enumeration is many handshakes per Service, so it
// buys a weekly cadence against that cost, and it does not ride the `reachability`
// exchange the way `certificate` (#197) does. `certificate` reads the chain on the
// single default handshake; `tls-acceptance` enumerates version/cipher acceptance.
// Two exchanges, two subjects' facets, two cadences (measurement-offers §1.1).
//
// It dispatches its OWN leaf — the `tls-acceptance` leaf — rather than reusing
// `connect-outcome`, and is kept additive to hot.go/cold.go/zone.go: a new job
// builder and job type, no rewrite of scan.go's core.
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

// TLSAcceptanceKind is the DB Scan kind this file dispatches. Unlike the port
// tiers, whose Scan kind differs from the `connect-outcome` leaf they dispatch, the
// `tls-acceptance` Scan and its leaf share the kind — it is its own exchange on its
// own cadence, so there is one name for the pair.
const TLSAcceptanceKind = tlsacceptance.Kind

// ReachedService is one open `(Address, port)` at one Vantage — a member of the
// open `Service` population the Scan enumerates. It is read from the current
// `reachability` spans reading `reached`, so the Scan targets exactly the Services
// known open and no port list is consulted.
type ReachedService struct {
	VantageID int64
	Address   string
	Port      uint16
}

// TLSAcceptanceJob is one queue job the tls-acceptance Scan produces: one Vantage,
// the open Services reached from it, and the declared candidate set. Partitioning
// is per Vantage — acceptance is measured from a position, exactly as reachability
// is — while every Service stays enumerable in the recorded scope.
type TLSAcceptanceJob struct {
	ScanID       int64
	VantageID    int64
	Vantage      string
	VantageClass string
	Kind         string
	Services     []tlsacceptance.ServiceTarget
	Candidates   tlsacceptance.CandidateSet
}

// BuildTLSAcceptanceJobs fans a tls-acceptance Scan out into one job per Vantage
// over the Services reached from that Vantage. It is NOT gated by a port list — the
// Services ARE the scope, drawn from the open `Service` population.
//
// It IS re-gated by Custody here (ADR-0079, #742). The reached-Service population is
// a stale snapshot: an address admitted at connect time can lose its authorisation
// before this weekly enumeration runs — the covering address scope is withdrawn, or
// the vantage's derived class flips to `internet` — after which ADR-0079 forbids the
// probe. So the SAME denotation/class gate the connect-time dispatch enforces
// (hot.go/cold.go) is re-applied against the CURRENT Estate: any reached Service whose
// address no longer passes `MayProbe` for its vantage's freshly-derived class is
// dropped, closing the stale-population back door to the gate. A Vantage with no
// admitted reached Service yields no job, a legible empty scope rather than an error.
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

	// Class is DERIVED per batch from each vantage's presented-address facts against
	// the declared address scopes (#709), never the vestigial column — mirroring
	// hot.go/cold.go so the re-gate reads exactly what connect-time dispatch reads.
	covered := estate.CoversAddressScope
	vcByID := make(map[int64]custody.VantageClass, len(vantages))
	for _, v := range vantages {
		vcByID[v.ID] = vantageclass.Derive(v.Dialled, v.Egress, covered)
	}

	grouped := make(map[int64][]tlsacceptance.ServiceTarget)
	for _, s := range services {
		if _, ok := byID[s.VantageID]; !ok {
			continue // a reached Service at a Vantage no longer configured — dropped
		}
		a, err := netip.ParseAddr(s.Address)
		if err != nil {
			continue // an address that cannot be named cannot be gated — dropped
		}
		if !estate.MayProbe(a, vcByID[s.VantageID]) {
			continue // the authorising scope/class was withdrawn — re-gated out (#742)
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

// JobSpec renders a TLSAcceptanceJob into the wire JobSpec the tls-acceptance
// prober reads on stdin. The candidate set travels whole so the leaf offers exactly
// it and the Batch records it by content (ADR-0025).
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

// AttemptedScope is the by-content record of what the job set out to cover: the
// Vantage, the open Services enumerated, and the declared candidate set. The
// candidate set is the Batch's recorded scope (CONTEXT.md `tls-acceptance`), so an
// offer of nineteen suites can never assert a twentieth was refused, and a Service
// we could not reach on this run reads as an absence rather than a measured value
// (v1 spec §4.1).
func (j TLSAcceptanceJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(tlsAcceptanceScopeRecord{
		Vantage:    j.Vantage,
		Services:   j.Services,
		Candidates: j.Candidates,
	})
}

// OffersJSON is the candidate set recorded on the Batch by content — the same
// declared set the JobSpec carries, so the offer is legible on the durable Batch.
func (j TLSAcceptanceJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Candidates) }

// EmptyTLSAcceptanceScope is what a dead-lettered tls-acceptance Batch records —
// never the attempted scope, which would manufacture acceptance absences it never
// measured (v1 spec §4.1).
func EmptyTLSAcceptanceScope(vantage string) ([]byte, error) {
	return json.Marshal(tlsAcceptanceScopeRecord{Vantage: vantage, Services: []tlsacceptance.ServiceTarget{}})
}

type tlsAcceptanceScopeRecord struct {
	Vantage    string                        `json:"vantage"`
	Services   []tlsacceptance.ServiceTarget `json:"services"`
	Candidates tlsacceptance.CandidateSet    `json:"candidates,omitempty"`
}
