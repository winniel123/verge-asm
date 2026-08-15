// Package scan turns a Scan's configured intent into the concrete jobs the
// queue dispatches. This ticket builds the `dns` Scan (v1 spec §3.4, ADR-0084):
// daily, unconditional of Custody, no port list, run at every configured
// Vantage over the name-scope Seeds, covering `resolution` and the `dns-record`
// facet. Every offer the leaf will make is enumerated here and travels in the
// job spec so the Batch records it by content (ADR-0025).
package scan

import (
	"encoding/json"
	"fmt"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

// DNSKind is the scan kind this package dispatches.
const DNSKind = "dns"

// Vantage is the position a dns job resolves from. The resolver is part of the
// Vantage's identity (ADR-0070), so it is carried on the job rather than
// defaulted by the prober.
type Vantage struct {
	ID       int64
	Name     string
	Resolver string
}

// Job is one queue job the dns Scan produces: one Vantage, the full name-scope
// set, and the offers on the wire. Batch partitioning is per Vantage — a
// resolution's answer is a function of where it was asked from — while the
// resolver stays enumerable over each Name it lists.
type Job struct {
	ScanID    int64
	VantageID int64
	Vantage   string
	Kind      string
	Names     []string
	Offers    resolutionwalk.Offers
	resolver  string // the Vantage's recursive resolver, set by the dispatcher
}

// BuildDNSJobs fans a dns Scan out into one job per Vantage over the given
// name-scope domains. It produces no jobs when there is nothing to resolve — a
// Scan whose scope list is empty is a legible state, not an error — and none
// when there is no Vantage to resolve from, since Exposure and even resolution
// require a position to measure from.
func BuildDNSJobs(scanID int64, names []string, vantages []Vantage) []Job {
	if len(names) == 0 || len(vantages) == 0 {
		return nil
	}
	offers := resolutionwalk.DefaultOffers()
	jobs := make([]Job, 0, len(vantages))
	for _, v := range vantages {
		jobs = append(jobs, Job{
			ScanID:    scanID,
			VantageID: v.ID,
			Vantage:   v.Name,
			Kind:      resolutionwalk.Kind,
			Names:     names,
			Offers:    offers,
		})
	}
	return jobs
}

// JobSpec renders a Job into the wire JobSpec the prober reads on stdin. batch
// is a traceability token only; the durable Batch identity is the row the queue
// writes at the job's terminal outcome.
func (j Job) JobSpec(batch string) (wire.JobSpec, error) {
	scope := resolutionwalk.Scope{
		Vantage:  j.Vantage,
		Resolver: resolverFor(j),
		Names:    j.Names,
		Offers:   j.Offers,
	}
	raw, err := json.Marshal(scope)
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: j.Kind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the job set out to cover, used
// as the completed Batch scope on success and replaced by an empty scope on a
// dead-letter.
func (j Job) AttemptedScope() ([]byte, error) {
	return json.Marshal(scopeRecord{Vantage: j.Vantage, Names: j.Names})
}

// EmptyScope is what a dead-lettered Batch records — never the attempted scope,
// which would manufacture absences it never measured (v1 spec §4.1).
func EmptyScope(vantage string) ([]byte, error) {
	return json.Marshal(scopeRecord{Vantage: vantage, Names: []string{}})
}

// OffersJSON is the offers recorded on the Batch by content.
func (j Job) OffersJSON() ([]byte, error) { return json.Marshal(j.Offers) }

type scopeRecord struct {
	Vantage string   `json:"vantage"`
	Names   []string `json:"names"`
}

// resolverFor is where a job's resolver comes from. The Vantage row carries it;
// this indirection is a seam for a later per-vantage override.
func resolverFor(j Job) string {
	// The resolver is carried on the Vantage and copied onto the job's spec by
	// the dispatcher; jobs built without one fall back to the loopback stub the
	// migration ships, which the operator replaces.
	if j.resolver != "" {
		return j.resolver
	}
	return "127.0.0.1:53"
}

// WithResolver returns a copy of the job carrying the Vantage's resolver, set by
// the dispatcher from the Vantage row so the pure builder stays free of
// deployment detail.
func (j Job) WithResolver(resolver string) Job {
	j.resolver = resolver
	return j
}
