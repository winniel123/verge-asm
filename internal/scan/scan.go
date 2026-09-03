// Package scan turns a Scan's configured intent into the jobs the queue dispatches.
// Every offer a leaf will make is enumerated here and travels in the job spec, so a
// Batch records what governed the measurement by content (ADR-0025).
package scan

import (
	"encoding/json"
	"fmt"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/wire"
)

const DNSKind = "dns"

// The stored class is vestigial: hot and cold derive it per batch from presented facts (#709).

type Vantage struct {
	ID       int64
	Name     string
	Resolver string // part of the Vantage's identity, never the prober's default (ADR-0070)
	Class    string
	Dialled  string
	Egress   string
}

type Job struct {
	ScanID    int64
	VantageID int64
	Vantage   string
	Kind      string
	Names     []string
	Offers    resolutionwalk.Offers
	resolver  string
	seeds     []string
}

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

func (j Job) AttemptedScope() ([]byte, error) {
	// A name whose parent went unprobed can never be Shadowed — a silence, not a value (ADR-0066).
	return json.Marshal(scopeRecord{
		Vantage:                j.Vantage,
		Names:                  j.Names,
		ControlProbePopulation: wildcarddiscrim.ControlPopulation(j.Names, j.seedScopes()),
	})
}

func (j Job) seedScopes() []string {
	// Names is wider than the Seeds since ADR-0107, so this fallback is only the no-Seeds case.
	if len(j.seeds) > 0 {
		return j.seeds
	}
	return j.Names
}

func (j Job) WithSeeds(seeds []string) Job {
	j.seeds = seeds
	return j
}

func EmptyScope(vantage string) ([]byte, error) {
	// An unmeasured pair must never read as a measured absence (v1 spec §4.1).
	return json.Marshal(scopeRecord{Vantage: vantage, Names: []string{}})
}

func (j Job) OffersJSON() ([]byte, error) { return json.Marshal(j.Offers) }

type scopeRecord struct {
	Vantage                string   `json:"vantage"`
	Names                  []string `json:"names"`
	ControlProbePopulation []string `json:"control_probe_population,omitempty"`
}

func resolverFor(j Job) string {
	// 127.0.0.11 is Docker's embedded DNS, the same default db/migrations/18800 ships.
	if j.resolver != "" {
		return j.resolver
	}
	return "127.0.0.11:53"
}

func (j Job) WithResolver(resolver string) Job {
	j.resolver = resolver
	return j
}
