// The `ct` Scan reads certificate transparency through crt.sh (ADR-0106).
// Its source admits without observing: a completed Batch produces Names on
// `authority: inferred`, and no observation, facet or timeline (ADR-0027).
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strings"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

// Named for the exchange it reads, not the instrument that reads it (ADR-0084, ADR-0106).

const CTKind = "ct"

// It must equal the source-catalogue slug, or the enablement state keys stop lining up.

const CrtshSource = "crtsh"

// A CT admission's Citation chain terminates at the name-scope Seed (ADR-0027).

type CTSeed struct {
	SeedID int64
	Domain string
}

type CTJob struct {
	ScanID int64
	SeedID int64
	Domain string
}

// Enablement of the crtsh source is gated by the dispatcher, never here (ADR-0106).

func BuildCTJobs(scanID int64, seeds []CTSeed) []CTJob {
	// An aperture over an empty scope is a legible state, not an error (CONTEXT.md Scan).
	if len(seeds) == 0 {
		return nil
	}
	jobs := make([]CTJob, 0, len(seeds))
	for _, s := range seeds {
		jobs = append(jobs, CTJob{ScanID: scanID, SeedID: s.SeedID, Domain: s.Domain})
	}
	return jobs
}

type ctScope struct {
	Domain string `json:"domain"`
	SeedID int64  `json:"seed_id"`
}

func (j CTJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(ctScope{Domain: j.Domain, SeedID: j.SeedID})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal ct scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: CTKind, Scope: raw}, nil
}

func (j CTJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(ctScopeRecord{Domain: j.Domain})
}

func EmptyCTScope() ([]byte, error) {
	// A failed fetch of an append-only source asserts no absence (ADR-0005, ADR-0027 §7).
	return json.Marshal(ctScopeRecord{})
}

type ctScopeRecord struct {
	Domain string `json:"domain,omitempty"`
}

func CTScopeFromSpec(scope []byte) (CTSeed, error) {
	var cs ctScope
	if err := json.Unmarshal(scope, &cs); err != nil {
		return CTSeed{}, fmt.Errorf("scan: decode ct scope: %w", err)
	}
	return CTSeed{SeedID: cs.SeedID, Domain: cs.Domain}, nil
}

func CrtshURL(domain string) string {
	// %25 is crt.sh's SQL-LIKE wildcard, covering subdomains (passive-discovery-sources.md §2.2).
	return "https://crt.sh/?q=%25." + url.QueryEscape(domain) + "&output=json"
}

// crt.sh carries no fingerprint, so a CT answer admits a Name and observes nothing (ADR-0027).

type CrtshRow struct {
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"`
}

func ParseCrtshRows(body []byte) ([]CrtshRow, error) {
	// A malformed 200 evidences nothing, so an unreadable body errors, never empty (ADR-0027 §7).
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("scan: empty crt.sh body")
	}
	var rows []CrtshRow
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
		return nil, fmt.Errorf("scan: decode crt.sh rows: %w", err)
	}
	return rows, nil
}

// A hostile answer could mint unbounded admitted_name rows, a durable DB-bloat DoS (#741).

const MaxAdmittedNames = 100_000

func AdmittedNames(rows []CrtshRow, domain string) []string {
	a := NewCTAdmitter(domain)
	a.Add(crtshCandidates(rows))
	return a.Names()
}

func crtshCandidates(rows []CrtshRow) iter.Seq[string] {
	// A hostile multi-megabyte name_value is walked element by element, never held whole (#741).
	return func(yield func(string) bool) {
		for _, r := range rows {
			for line := range strings.SplitSeq(r.NameValue, "\n") {
				if !yield(line) {
					return
				}
			}
			if !yield(r.CommonName) {
				return
			}
		}
	}
}

// crt.sh answers in one shot, capped at 999 rows by the service, so nothing paginates.

type crtshCTSource struct{}

func (crtshCTSource) Slug() string        { return CrtshSource }
func (crtshCTSource) DisplayName() string { return "crt.sh" }

func (crtshCTSource) QueryURL(domain, _ string) string { return CrtshURL(domain) }

func (crtshCTSource) DecodePage(body []byte) (iter.Seq[string], string, error) {
	rows, err := ParseCrtshRows(body)
	if err != nil {
		return nil, "", err
	}
	return crtshCandidates(rows), "", nil
}

func CrtshCTSource() CTSource { return crtshCTSource{} }

func normaliseName(s string) string {
	// A parallel Unicode fold here breaks the admission hop's join on subject_key (ADR-0107, #256).
	return resolutionwalk.CanonicalName(strings.TrimSpace(s))
}

func withinScope(name, domain string) bool {
	if domain == "" {
		return false
	}
	return name == domain || strings.HasSuffix(name, "."+domain)
}
