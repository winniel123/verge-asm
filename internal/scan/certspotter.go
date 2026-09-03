// Cert Spotter is the operator-keyed bulk-by-name CT primary (ct-source-replacement.md §2).
// Its authenticated tier clears the consent bar; its unauthenticated tier did not (ADR-0003, §2.2).
// The API key is worker-only (ADR-0053), so the credentialed fetch stays in internal/queue.
package scan

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strings"
)

// The barred certspotter slug is repurposed — the operator's key is now the instrument (§2.1).

const CertSpotterSource = "certspotter"

const certSpotterEndpoint = "https://api.certspotter.com/v1/issuances"

func CertSpotterURL(domain, after string) string {
	// match_wildcards is left default-off: a wildcard identity admits no Name (ADR-0060).
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

type CertSpotterIssuance struct {
	ID       string   `json:"id"`
	DNSNames []string `json:"dns_names"`
}

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

func maxIssuanceID(issuances []CertSpotterIssuance) string {
	var max string
	// The API promises no page order, so the cursor is the maximum id, not the last row's.
	for _, is := range issuances {
		if issuanceIDLess(max, is.ID) {
			max = is.ID
		}
	}
	return max
}

func issuanceIDLess(a, b string) bool {
	// Cert Spotter ids are decimal with no leading zeros, so the longer string is the larger.
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

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

func CertSpotterCTSource() CTSource { return certSpotterSource{} }
