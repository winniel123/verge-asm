// This file builds the `http-identity` Scan (v1 spec §3.3/§3.4, ADR-0011/ADR-0024):
// DAILY, over every reached `Endpoint`, issuing a single `GET /` — an enumeration of
// the open `Service` population, NOT a port tier. Like `tls-acceptance` (#199) it has
// NO port list at all: the Services it enumerates carry their own ports, inherited
// from the `reachability` timelines that found them open, because the Scan's scope is
// the reached `Service` population (ADR-0028's enumeration shape), never a curated
// implicit-HTTP port table.
//
// It rides the daily hot-Scan reachability the way `certificate` (#197) rides the
// default handshake and `tls-acceptance` rides the open-Service population: after
// `connect-outcome` reports `reached`, the HTTP step decides the `http-identity`
// facet for the `Endpoint`. Modelled as a `Scan` of its OWN so the dispatch stays
// additive to hot.go/cold.go/tlsacceptance.go — a new job builder and job type, no
// rewrite of scan.go's core.
//
// It dispatches the `http-exchange` leaf (httpexchange.Kind) rather than reusing
// `connect-outcome`: the Scan kind (`http-identity`, the facet it decides) differs
// from the leaf kind (`http-exchange`, the exchange it runs), exactly as `hot`/`cold`
// dispatch `connect-outcome`. Wiring this dispatch is what lights the four dormant
// HTTP-identity rules — the prober case, the probe User-Agent, the measurer, the
// drift fold and the rules already shipped; only the job that feeds them was absent.
package scan

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/vantageclass"
	"github.com/winniel123/verge-asm/internal/wire"
)

// HTTPIdentityKind is the DB Scan kind this file dispatches. Unlike `tls-acceptance`
// — whose Scan and leaf share one name — the `http-identity` Scan decides the
// `http-identity` facet by dispatching the `http-exchange` leaf, so the Scan kind and
// the leaf kind differ, exactly as the port tiers' `hot`/`cold` dispatch
// `connect-outcome`.
const HTTPIdentityKind = "http-identity"

// HTTPIdentityJob is one queue job the http-identity Scan produces: one Vantage, the
// reached `Endpoint`s exchanged with from it, and the declared parameter set.
// Partitioning is per Vantage — an HTTP identity is measured from a position, exactly
// as reachability is — while every Endpoint stays enumerable in the recorded scope.
type HTTPIdentityJob struct {
	ScanID       int64
	VantageID    int64
	Vantage      string
	VantageClass string
	Kind         string
	Targets      []httpexchange.Target
	Params       httpexchange.Params
}

// BuildHTTPIdentityJobs fans an http-identity Scan out into one job per Vantage over
// the Services reached from that Vantage, each rendered as the nameless `Endpoint` on
// that reached Service (a reached Service presents an HTTP identity whether or not a
// Name cited it — httpexchange.Target with an empty Name is the distinguished nameless
// key). It is NOT gated by a port list — the reached Services ARE the scope.
//
// It IS re-gated by Custody here (ADR-0079, #742), mirroring BuildTLSAcceptanceJobs:
// the reached-Service population is a stale snapshot, so an address whose authorising
// address scope was withdrawn — or whose vantage's derived class flipped to `internet`
// — since it was reached must not be re-enumerated with a daily `GET /`. The SAME
// denotation/class gate connect-time dispatch enforces is re-applied against the
// current Estate, dropping any reached Service that no longer passes `MayProbe` for its
// vantage's freshly-derived class. A Vantage with no admitted reached Service yields no
// job, a legible empty scope rather than an error.
func BuildHTTPIdentityJobs(scanID int64, estate custody.Estate, services []ReachedService, vantages []Vantage) []HTTPIdentityJob {
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

	grouped := make(map[int64][]httpexchange.Target)
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
		grouped[s.VantageID] = append(grouped[s.VantageID], httpexchange.Target{
			Name:    "", // the nameless Endpoint — the reached Service's own HTTP identity
			Address: s.Address,
			Port:    s.Port,
			Scheme:  schemeForPort(s.Port),
		})
	}

	var jobs []HTTPIdentityJob
	for _, id := range order {
		targets := grouped[id]
		if len(targets) == 0 {
			continue
		}
		v := byID[id]
		jobs = append(jobs, HTTPIdentityJob{
			ScanID:       scanID,
			VantageID:    id,
			Vantage:      v.Name,
			VantageClass: v.Class,
			Kind:         httpexchange.Kind,
			Targets:      targets,
			Params:       httpexchange.DefaultParams(),
		})
	}
	return jobs
}

// schemeForPort frames the single `GET /` as `https` on the well-known implicit-TLS
// ports and `http` everywhere else. It is a framing of how the exchange is spoken, not
// a widening of what is probed (httpexchange.Target.Scheme doc): the reached-Service
// population is unchanged, and a port that speaks neither still folds to the
// determinate `no-http-response` value, never an absence.
func schemeForPort(port uint16) string {
	switch port {
	case 443, 8443, 6443:
		return "https"
	default:
		return "http"
	}
}

// JobSpec renders an HTTPIdentityJob into the wire JobSpec the http-exchange prober
// reads on stdin. The declared parameter set travels whole so the leaf exchanges under
// exactly it and the Batch records what governed the exchange by content (ADR-0025).
func (j HTTPIdentityJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(httpexchange.Scope{
		Vantage:      j.Vantage,
		VantageClass: j.VantageClass,
		Targets:      j.Targets,
		Params:       j.Params,
	})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal http-identity scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: j.Kind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the job set out to cover: the
// Vantage, the reached Endpoints exchanged with, and the declared parameter set. The
// params are the Batch's recorded offer (ADR-0025), so an Endpoint we could not reach
// on this run reads as an absence rather than a measured value (v1 spec §4.1).
func (j HTTPIdentityJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(httpIdentityScopeRecord{
		Vantage: j.Vantage,
		Targets: j.Targets,
		Params:  j.Params,
	})
}

// OffersJSON is the declared parameter set recorded on the Batch by content — the same
// params the JobSpec carries, so the offer is legible on the durable Batch.
func (j HTTPIdentityJob) OffersJSON() ([]byte, error) { return json.Marshal(j.Params) }

// EmptyHTTPIdentityScope is what a dead-lettered http-identity Batch records — never
// the attempted scope, which would manufacture HTTP-identity absences it never
// measured (v1 spec §4.1).
func EmptyHTTPIdentityScope(vantage string) ([]byte, error) {
	return json.Marshal(httpIdentityScopeRecord{Vantage: vantage, Targets: []httpexchange.Target{}})
}

type httpIdentityScopeRecord struct {
	Vantage string                `json:"vantage"`
	Targets []httpexchange.Target `json:"targets"`
	Params  httpexchange.Params   `json:"params,omitempty"`
}
