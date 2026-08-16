package wildcarddiscrim

import (
	"sort"
	"strings"

	"github.com/winniel123/verge-asm/internal/custody"
	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// ControlPopulation is the control-probe population of a Batch (ADR-0066): the
// set of **immediate parents** of the Names in the batch's resolution scope,
// deduplicated, and intersected with the Seed name scopes those names sit inside
// — recorded on the Batch by content as the seventh aperture input, never a
// declared parameter of this leaf.
//
// A name is discriminated **at its parent**, because a control label constructed
// under P falls off the tree at exactly the encloser the names under P do —
// whatever depth the wildcard sits at, and whether or not P itself exists. The
// probing gate stops the population at the Seed: a parent at or above the
// operator's own apex (e.g. a TLD) sits inside no Seed scope and is dropped, so a
// wildcard at or above the apex is out of reach by an existing rule rather than a
// carve-out.
func ControlPopulation(names, seedScopes []string) []string {
	seeds := make([]string, 0, len(seedScopes))
	for _, s := range seedScopes {
		seeds = append(seeds, rw.CanonicalName(s))
	}
	seen := map[string]struct{}{}
	for _, name := range names {
		key := rw.CanonicalName(name)
		if strings.HasPrefix(key, "*.") || key == "*" {
			// A name whose leftmost label is `*` is a subject nowhere and is
			// never queried; it contributes no parent.
			continue
		}
		p, ok := parent(key)
		if !ok {
			continue
		}
		// Subtree containment is the one label-wise suffix test the model owns
		// (custody.WithinAnyZone) — parents and seeds are already CanonicalName'd,
		// so this only single-sources the rule rather than re-deriving it.
		if !custody.WithinAnyZone(p, seeds) {
			continue
		}
		seen[p] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// parent drops the leftmost label. A single-label name (or empty) has no parent
// in the estate — its parent is a TLD or the root, which the Seed gate excludes.
func parent(name string) (string, bool) {
	i := strings.IndexByte(name, '.')
	if i < 0 || i == len(name)-1 {
		return "", false
	}
	return name[i+1:], true
}
