// This file builds the `ct` Scan (ADR-0106): the covering execution path for
// certificate transparency, queried through crt.sh. It is worker-read like the
// zone Scan — no port list, no vantage — but unlike every other Scan its source
// ADMITS WITHOUT OBSERVING (ADR-0027): a completed Batch produces no observation,
// no facet and no timeline; it produces the set of `Name`s the certificates
// carry, each admitted on `authority: inferred` and citing the Batch. This file
// holds the pure half — the job shape, the crt.sh URL, and the name extraction
// and filtering that decide what a crt.sh answer admits. The impure half (the
// throttled fetch, the retry/dead-letter, the admission write) is in
// internal/queue/crtsh.go.
package scan

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/winniel123/verge-asm/internal/wire"
)

// CTKind is the scan kind this file dispatches — the exchange (certificate
// transparency), not the instrument (ADR-0084's `dns`-not-`discovery` move,
// ADR-0106). Kept additive so the queue and the measurement binary register it
// independently.
const CTKind = "ct"

// CrtshSource is the source every CT admission is attributed to: crt.sh, the
// keyless CT front door (ADR-0003), `authority: inferred`, `completeness:
// corroborative` (CONTEXT.md `Source`). It matches the source-catalogue slug so
// the enablement state keys line up.
const CrtshSource = "crtsh"

// CTSeed is one name-scope Seed the CT Scan queries: the registrable domain and
// the Seed id its admissions' Citation chain terminates at (ADR-0027).
type CTSeed struct {
	SeedID int64
	Domain string
}

// CTJob is one queue job the CT Scan produces: one crt.sh query for one
// name-scope Seed. It carries no Vantage — a logged certificate is not a function
// of where we read the log from — and no offers.
type CTJob struct {
	ScanID int64
	SeedID int64
	Domain string
}

// BuildCTJobs fans a CT Scan out into one job per name-scope Seed. It produces no
// jobs when no name scope is declared — an aperture over an empty scope is a
// legible state, not an error (CONTEXT.md `Scan`). Enablement of the crtsh source
// is gated by the dispatcher (ADR-0106), not here: this is the pure fan-out.
func BuildCTJobs(scanID int64, seeds []CTSeed) []CTJob {
	if len(seeds) == 0 {
		return nil
	}
	jobs := make([]CTJob, 0, len(seeds))
	for _, s := range seeds {
		jobs = append(jobs, CTJob{ScanID: scanID, SeedID: s.SeedID, Domain: s.Domain})
	}
	return jobs
}

// ctScope is the wire payload a CT job carries: the domain to query and the Seed
// its admissions terminate at.
type ctScope struct {
	Domain string `json:"domain"`
	SeedID int64  `json:"seed_id"`
}

// JobSpec renders a CTJob into the wire JobSpec the worker reads. Like the zone
// Scan there is no prober exec — the worker itself fetches crt.sh and reads the
// names off — so the domain travels in the job rather than a vantage and offers.
func (j CTJob) JobSpec(batch string) (wire.JobSpec, error) {
	raw, err := json.Marshal(ctScope{Domain: j.Domain, SeedID: j.SeedID})
	if err != nil {
		return wire.JobSpec{}, fmt.Errorf("scan: marshal ct scope: %w", err)
	}
	return wire.JobSpec{Batch: batch, Kind: CTKind, Scope: raw}, nil
}

// AttemptedScope is the by-content record of what the CT job set out to cover: the
// domain it queried. It is the completed Batch's recorded scope on success. A
// dead-lettered CT Batch records EmptyCTScope instead — never this — because a
// failed fetch of an append-only, corroborative source asserts no absence
// (ADR-0005, ADR-0027).
func (j CTJob) AttemptedScope() ([]byte, error) {
	return json.Marshal(ctScopeRecord{Domain: j.Domain})
}

// EmptyCTScope is what a dead-lettered CT Batch records — never the attempted
// domain, which would read as "we covered this domain and found no certificates",
// the false-absence a non-200 must never assert (ADR-0005, ADR-0027 §7).
func EmptyCTScope() ([]byte, error) {
	return json.Marshal(ctScopeRecord{})
}

type ctScopeRecord struct {
	Domain string `json:"domain,omitempty"`
}

// CTScopeFromSpec decodes a CT job's wire scope back into its domain and Seed, so
// the worker fetches and attributes from the same values the dispatcher enqueued.
func CTScopeFromSpec(scope []byte) (CTSeed, error) {
	var cs ctScope
	if err := json.Unmarshal(scope, &cs); err != nil {
		return CTSeed{}, fmt.Errorf("scan: decode ct scope: %w", err)
	}
	return CTSeed{SeedID: cs.SeedID, Domain: cs.Domain}, nil
}

// CrtshURL is the crt.sh query for a registrable domain: a wildcard identity
// match that includes subdomains, in JSON (passive-discovery-sources.md §2.2).
// `%25` is a URL-encoded `%`, crt.sh's SQL-LIKE wildcard, so `%.example.com`
// matches every subdomain identity.
func CrtshURL(domain string) string {
	return "https://crt.sh/?q=%25." + domain + "&output=json"
}

// CrtshRow is one row of a crt.sh `output=json` answer. Only the two identity
// fields are read: `name_value` is the newline-separated SAN list (the field you
// actually want) and `common_name` is the subject CN. The fingerprint the
// `certificate` facet would need is absent from the instrument (ADR-0027), which
// is the whole reason CT observes nothing — so no other field is decoded.
type CrtshRow struct {
	CommonName string `json:"common_name"`
	NameValue  string `json:"name_value"`
}

// ParseCrtshRows decodes a crt.sh `output=json` body into its rows. crt.sh
// returns a JSON array of objects; leading/trailing whitespace is tolerated. A
// body that is not a JSON array is an error, not an empty result — a malformed
// 200 is not evidence of anything (ADR-0027 §7), so the caller treats a parse
// failure as transient rather than as "no certificates".
func ParseCrtshRows(body []byte) ([]CrtshRow, error) {
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

// AdmittedNames is the pure admission decision: the set of `Name`s a crt.sh
// answer admits under one name-scope domain. It applies the two rulings already
// made — ADR-0060 (no value with an asterisk label admits a Name; a wildcard
// denotes a set and a partial wildcard has two denotations, both refused) and
// ADR-0047 (the Seed decides which names are inside, so a certificate's foreign
// SANs admit nothing under this scope) — and dedupes, returning a deterministic
// sorted set. A row's `name_value` is split on newlines; `common_name` is one
// more candidate.
func AdmittedNames(rows []CrtshRow, domain string) []string {
	domain = normaliseName(domain)
	seen := map[string]struct{}{}
	var out []string
	consider := func(raw string) {
		n := normaliseName(raw)
		if n == "" {
			return
		}
		// ADR-0060: an asterisk anywhere in a dNSName value denotes a set (a
		// wildcard) or has two denotations (a partial wildcard). Neither admits a
		// Name, and the operative rule on the SAN path is the blunt one.
		if strings.Contains(n, "*") {
			return
		}
		// ADR-0047: the name scope filters. A cert legitimately carries SANs for
		// several estates; only the names under the domain we queried are inside.
		if !withinScope(n, domain) {
			return
		}
		if _, dup := seen[n]; dup {
			return
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	for _, r := range rows {
		for _, line := range strings.Split(r.NameValue, "\n") {
			consider(line)
		}
		consider(r.CommonName)
	}
	sort.Strings(out)
	return out
}

// normaliseName lowercases, trims surrounding whitespace, and strips a trailing
// dot, so `Example.COM.` and `example.com` key alike. It does not attempt full
// IDNA or the ADR-0055 label-sequence key — a name admitted here acquires its
// canonical key when the resolver measures it (ADR-0027); this is a membership
// candidate, not a subject key.
func normaliseName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".")
	return s
}

// withinScope reports whether name falls under the name-scope domain — equal to
// it or a label-wise subdomain of it (ADR-0047). The suffix test is on a label
// boundary so `notexample.com` is not read as under `example.com`.
func withinScope(name, domain string) bool {
	if domain == "" {
		return false
	}
	return name == domain || strings.HasSuffix(name, "."+domain)
}
