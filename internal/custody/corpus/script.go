// Package corpus is the golden corpus for the `Custody` derivation
// (golden-corpus.md §10, ADR-0008, ADR-0129). It holds a checked-in enumeration
// of the cells the derivation owes a pinning row, each row an
// (estate, observed SAN set per address, expected NDJSON) tuple rendered
// hermetically through `custody.Estate` — no network, no database, no
// containers.
//
// It is a SIBLING of the six measure corpora and is never pooled with them. Those
// discharge ADR-0041's membership obligation over two measure leaves. This one
// discharges a different obligation: ADR-0008's rule that a declared parameter is
// pinned by a corpus row whose output the value decides. The parameter is the
// fan-out threshold, and the rows straddle it (ADR-0129 §3, the #955 amendment).
//
// A row renders the DERIVATION, never a leaf's wire output. Each line carries the
// measurement the row observed, the reach the derivation gave the address, and the
// gate that reads it, so a row fails on the whole path rather than on one boolean:
//
//   - the fan-out count and the `shared-edge` verdict the SAN set reduces to,
//   - whether the address is an `edge-fanout` candidate (the PRE-veto reach),
//   - whether a declared address scope covers it,
//   - the derived `Custody`, and
//   - whether the probing gate opens from an `internet`-class Vantage.
//
// The verdict is computed by `custody.SharedEdge` at render time rather than
// written into the row. That is what puts the threshold inside the digest: a row
// declares a SAN set, and the shipped constant alone decides which side of the
// boundary the row lands on. A threshold move rewrites the goldens.
package corpus

import "fmt"

// sanTemplate renders one SAN under its own registrable domain. `.invalid` is
// reserved by RFC 2606 and is delegated to nobody, so it can never be registered
// and the fixture reaches no real estate. It is in no section of the Public
// Suffix List, so the PSL's wildcard rule reduces `dNNN.invalid` to itself and
// each index is one distinct registrable domain.
//
// The fixture asserts its own counts (TestFixtureStraddlesTheThreshold), so a
// PSL revision that ever listed `invalid` fails with that test's message rather
// than as an unexplained digest move.
const sanTemplate = "edge.d%03d.invalid"

// sanSet renders a SAN set that reduces to exactly n distinct registrable
// domains, plus two entries that raise the count by ZERO. The two are what make
// the row a claim about the REDUCTION rather than about a list length:
//
//   - a wildcard SAN over an already-counted domain, which reduces to the name
//     beneath the wildcard and deduplicates away, and
//   - an IPv4 SAN, which the reduction drops for its numeric top label.
//
// So a row at the threshold carries n+2 SANs and measures n. n must be positive;
// the corpus calls it at 99 and at 100 alone.
func sanSet(n int) []string {
	if n <= 0 {
		panic("corpus: sanSet needs a positive count")
	}
	out := make([]string, 0, n+2)
	for i := range n {
		out = append(out, fmt.Sprintf(sanTemplate, i))
	}
	out = append(out,
		fmt.Sprintf("*."+sanTemplate, 0), // already counted; deduplicates away
		"192.0.2.1",                      // an iPAddress SAN; dropped
	)
	return out
}

// SANsBelowThreshold and SANsAtThreshold are the two boundary fixtures, exported
// so the threshold-move test can assert the pair straddles the constant. They are
// ADR-0085's rule in force — *a boundary between two outcomes is pinned by two
// rows, one on each side, authored as a pair and failing as a pair* — and the
// pair is authored HERE so neither side can be edited without the other in view.
func SANsBelowThreshold() []string { return sanSet(belowThreshold) }

func SANsAtThreshold() []string { return sanSet(atThreshold) }
