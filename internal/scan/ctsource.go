// The bulk-by-name `ct` Scan's two sources (docs/spec/ct-source-replacement.md §2).
// They differ only in query and decode; the admission filter is one and the same (§2.6).
// Exactly one is active per config, selected at worker wire-time by the operator key (§2.3).
package scan

import (
	"iter"
	"sort"
	"strings"
)

type CTSource interface {
	Slug() string
	DisplayName() string // a rendering for the operator, never the key (#780)
	QueryURL(domain, cursor string) string
	DecodePage(body []byte) (names iter.Seq[string], next string, err error)
}

type CTAdmitter struct {
	domain string
	seen   map[string]struct{}
	out    []string
}

func NewCTAdmitter(domain string) *CTAdmitter {
	return &CTAdmitter{domain: normaliseName(domain), seen: map[string]struct{}{}}
}

func (a *CTAdmitter) Add(candidates iter.Seq[string]) {
	for raw := range candidates {
		if len(a.out) >= MaxAdmittedNames {
			return
		}
		n := normaliseName(raw)
		if n == "" {
			continue
		}
		if strings.Contains(n, "*") {
			continue
		}
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

func (a *CTAdmitter) Full() bool {
	return len(a.out) >= MaxAdmittedNames
}

func (a *CTAdmitter) Names() []string {
	sort.Strings(a.out)
	return a.out
}
