// This file holds what the bulk-by-name `ct` Scan shares across its two sources
// (spec docs/spec/ct-source-replacement.md §2, map #854): the source protocol the
// worker drives, and the one admission filter both sources feed. crt.sh (the
// keyless fallback) and Cert Spotter (the operator-keyed primary) differ only in
// how they build a query and decode a response into candidate names; the filter —
// wildcard refusal, scope, dedup and the count cap — is one and the same (§2.6).
// Exactly one source is active per config, selected at worker wire-time by the
// presence of the operator key (§2.3).
package scan

import (
	"iter"
	"sort"
	"strings"
)

// CTSource is one bulk-by-name CT source's pure protocol under the single `ct`
// Scan (spec §2.6). It builds a query URL for a name-scope domain at a pagination
// cursor, decodes a 200 body into the candidate names the response carries plus
// the cursor for the next page, and names the source slug its admissions are
// stamped with. It performs no I/O — the impure fetch loop (queue.completeCT)
// drives it and feeds every candidate through the shared CTAdmitter. A single-shot
// source (crt.sh) returns an empty next cursor, so the loop fetches once.
type CTSource interface {
	// Slug is the source-catalogue slug stamped on every admission and consulted
	// by the dispatcher's per-slug enablement gate. It matches CrtshSource or
	// CertSpotterSource.
	Slug() string
	// DisplayName is the source's operator-facing name, surfaced verbatim in the
	// live progress stream on a non-200 (#780). It is a rendering, never the key.
	DisplayName() string
	// QueryURL builds the request URL for domain at cursor. cursor is "" on the
	// first page; a single-shot source ignores it.
	QueryURL(domain, cursor string) string
	// DecodePage decodes a well-formed 200 body into the candidate names it
	// carries (walked lazily so a hostile oversized body never materialises as one
	// slice, #741) and the cursor for the next page. next is "" when no page
	// remains — always so for a single-shot source, and for a paginated source
	// once its last page is read. A body that is not well-formed is an error, not
	// an empty result: a malformed 200 is evidence of nothing (ADR-0027 §7).
	DecodePage(body []byte) (names iter.Seq[string], next string, err error)
}

// CTAdmitter accumulates the Names a bulk CT source admits under one name-scope
// domain: in scope, deduped, wildcard-free, and capped at MaxAdmittedNames (#741).
// It is the shared pure filter every `ct` source feeds candidate names into
// (spec §2.6). A paginated source feeds it page by page across several DecodePage
// calls, so the dedup set and the cap carry across pages — the cap bounds the
// whole admission, not one page. It applies the two rulings the `ct` path already
// made: ADR-0060 (a value with an asterisk label is a wildcard or a partial
// wildcard, and admits no Name) and ADR-0047 (the name scope decides which names
// are inside, so a certificate's foreign SANs admit nothing under this scope).
type CTAdmitter struct {
	domain string
	seen   map[string]struct{}
	out    []string
}

// NewCTAdmitter builds an admitter for one name-scope domain. The domain is
// normalised to the resolver's key (ADR-0107) so the scope test matches a
// non-ASCII-uppercase name as well as an ASCII one.
func NewCTAdmitter(domain string) *CTAdmitter {
	return &CTAdmitter{domain: normaliseName(domain), seen: map[string]struct{}{}}
}

// Add applies the filter to each candidate the sequence yields, stopping once the
// cap is reached. The sequence is walked lazily (the caller yields off an
// iterator, never a materialised slice), so a single giant field is filtered
// element by element and never held whole (#741). Add is safe to call again for a
// further page; the dedup set and the cap persist.
func (a *CTAdmitter) Add(candidates iter.Seq[string]) {
	for raw := range candidates {
		if len(a.out) >= MaxAdmittedNames {
			return
		}
		n := normaliseName(raw)
		if n == "" {
			continue
		}
		// ADR-0060: an asterisk anywhere in a value denotes a set (a wildcard) or
		// has two denotations (a partial wildcard). Neither admits a Name.
		if strings.Contains(n, "*") {
			continue
		}
		// ADR-0047: the name scope filters. A cert legitimately carries SANs for
		// several estates; only the names under the queried domain are inside.
		if !withinScope(n, a.domain) {
			continue
		}
		if _, dup := a.seen[n]; dup {
			continue
		}
		a.seen[n] = struct{}{}
		a.out = append(a.out, n)
	}
}

// Full reports whether the admitter has reached MaxAdmittedNames, so the caller
// stops paginating a source that has already filled the cap and logs the truncation.
func (a *CTAdmitter) Full() bool {
	return len(a.out) >= MaxAdmittedNames
}

// Names returns the admitted set, deterministically sorted. It is the input to
// admitCT: one admitted_name row per Name, each citing the Batch (ADR-0027).
func (a *CTAdmitter) Names() []string {
	sort.Strings(a.out)
	return a.out
}
