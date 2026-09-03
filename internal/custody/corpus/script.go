// Package corpus is the golden corpus for the `Custody` derivation
// (golden-corpus.md §10, ADR-0008, ADR-0129). It is a sibling of the six measure
// corpora and is never pooled with them: A6 is evaluated per lock.
package corpus

import "fmt"

const sanTemplate = "edge.d%03d.invalid"

func sanSet(n int) []string {
	if n <= 0 {
		panic("corpus: sanSet needs a positive count")
	}
	out := make([]string, 0, n+2)
	// RFC 2606 reserves `.invalid` and delegates it to nobody, so no fixture name can be registered.
	// No Public Suffix List section lists it, so each index is one distinct registrable domain.
	for i := range n {
		out = append(out, fmt.Sprintf(sanTemplate, i))
	}
	// A wildcard folds onto the name beneath it and an iPAddress SAN is dropped, so both add zero.
	// The pair makes a row a claim about the reduction rather than about a list length.
	out = append(out,
		fmt.Sprintf("*."+sanTemplate, 0),
		"192.0.2.1",
	)
	return out
}

// A boundary is pinned by two rows, one each side, authored and failing as a pair (ADR-0085).

func SANsBelowThreshold() []string { return sanSet(belowThreshold) }

func SANsAtThreshold() []string { return sanSet(atThreshold) }
