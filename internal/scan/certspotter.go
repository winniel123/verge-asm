// This file builds Cert Spotter as the operator-keyed bulk-by-name CT primary
// under the single `ct` Scan (spec docs/spec/ct-source-replacement.md §2, map #854,
// ticket #876). It is the pure half — the SSLMate CT Search API query URL, the
// issuance-page decoder, and the cursor that advances the pagination. The impure
// half (the Bearer-credentialed fetch, the throttle, the paginating completion
// path) is in internal/queue/crtsh.go, shared with crt.sh.
//
// Cert Spotter is name-indexed, so like crt.sh it answers a bulk-by-name query and
// admits the Names its answer carries without observing (ADR-0027). It is selected
// in place of crt.sh at worker wire-time when the operator supplies an API key; its
// authenticated tier clears the consent bar (ADR-0003), where its unauthenticated
// tier did not (docs/research/ct-bulk-primary-2026-08.md §2). The key is worker-only
// (ADR-0053); nothing here reads it — the queue fetcher carries it.
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strings"
)

// CertSpotterSource is the source every Cert Spotter admission is attributed to.
// It repurposes the barred `certspotter` catalogue slug into the operator-keyed
// primary (spec §2.1); consent moves from barred to operator-credentialed (the key
// is the instrument), completeness stays corroborative, authority stays inferred.
// It matches the source-catalogue slug so the enablement state keys line up.
const CertSpotterSource = "certspotter"

// certSpotterEndpoint is the SSLMate CT Search API "List Issuances for a Domain"
// endpoint (research §3.1). The query is the full-domain form the admission mapping
// needs: include_subdomains returns the whole subtree, and expand yields the SAN
// list (dns_names) the mapping reads and the issuer sub-object the query shape
// documents. match_wildcards is left default-off: a wildcard identity admits no
// Name (ADR-0060), so there is nothing to gain by asking for it.
const certSpotterEndpoint = "https://api.certspotter.com/v1/issuances"

// CertSpotterURL builds the issuances query for a registrable domain at a
// pagination cursor. after is the id of the last issuance already seen; it is
// omitted on the first page ("") and set to the running maximum id thereafter, so
// each page returns only issuances discovered after the previous one (research
// §3.2, cursor-based, not offset-based).
func CertSpotterURL(domain, after string) string {
	q := url.Values{}
	q.Set("domain", domain)
	q.Set("include_subdomains", "true")
	q.Add("expand", "dns_names")
	q.Add("expand", "issuer")
	if after != "" {
		q.Set("after", after)
	}
	return certSpotterEndpoint + "?" + q.Encode()
}

// CertSpotterIssuance is one issuance object of a Cert Spotter answer. Only the two
// fields the admission mapping needs are decoded: `id` is the cursor key
// (pagination and the between-runs high-water mark), and `dns_names` is the SAN
// list — the candidate Names. The fingerprint the `certificate` facet would need is
// not read here: CT admits without observing (ADR-0027), exactly as for crt.sh, so
// no other field is decoded.
type CertSpotterIssuance struct {
	ID       string   `json:"id"`
	DNSNames []string `json:"dns_names"`
}

// ParseCertSpotterPage decodes one Cert Spotter answer body into its issuances.
// The endpoint returns a JSON array of issuance objects; leading/trailing
// whitespace is tolerated. A body that is not a JSON array is an error, not an
// empty result — a malformed 200 is evidence of nothing (ADR-0027 §7), so the
// caller treats a parse failure as transient rather than as "no certificates".
func ParseCertSpotterPage(body []byte) ([]CertSpotterIssuance, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, fmt.Errorf("scan: empty Cert Spotter body")
	}
	var issuances []CertSpotterIssuance
	if err := json.Unmarshal([]byte(trimmed), &issuances); err != nil {
		return nil, fmt.Errorf("scan: decode Cert Spotter issuances: %w", err)
	}
	return issuances, nil
}

// maxIssuanceID returns the highest issuance id on a page — the cursor for the next
// page. It compares ids as non-negative decimal integers rather than trusting the
// page's order, so the cursor advances to the true maximum whichever way the answer
// is sorted. It returns "" for an empty page, which the completion path reads as
// "the tail is reached" and stops paginating.
func maxIssuanceID(issuances []CertSpotterIssuance) string {
	var max string
	for _, is := range issuances {
		if issuanceIDLess(max, is.ID) {
			max = is.ID
		}
	}
	return max
}

// issuanceIDLess orders two Cert Spotter issuance ids as non-negative decimal
// integers without leading zeros: the longer string is the larger number, and
// equal-length ids compare lexicographically. The empty string is treated as the
// smallest value, so a missing id never becomes the cursor.
func issuanceIDLess(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// certSpotterSource is Cert Spotter as a CTSource (spec §2): the operator-keyed
// bulk-by-name primary. Unlike crt.sh it paginates — a full-domain query returns
// issuances in cursor-addressed pages — so its next cursor is the running maximum
// id until a page comes back empty.
type certSpotterSource struct{}

func (certSpotterSource) Slug() string        { return CertSpotterSource }
func (certSpotterSource) DisplayName() string { return "Cert Spotter" }

func (certSpotterSource) QueryURL(domain, cursor string) string {
	return CertSpotterURL(domain, cursor)
}

func (certSpotterSource) DecodePage(body []byte) (iter.Seq[string], string, error) {
	issuances, err := ParseCertSpotterPage(body)
	if err != nil {
		return nil, "", err
	}
	next := maxIssuanceID(issuances)
	seq := func(yield func(string) bool) {
		for _, is := range issuances {
			for _, n := range is.DNSNames {
				if !yield(n) {
					return
				}
			}
		}
	}
	return seq, next, nil
}

// CertSpotterCTSource is the Cert Spotter bulk source, the operator-keyed primary
// selected when the operator key is configured (spec §2.3).
func CertSpotterCTSource() CTSource { return certSpotterSource{} }
