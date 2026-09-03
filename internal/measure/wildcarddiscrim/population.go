package wildcarddiscrim

import (
	"sort"
	"strings"

	"github.com/winniel123/verge-asm/internal/custody"
	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func ControlPopulation(names, seedScopes []string) []string {
	// A Batch aperture input recorded by content, never this leaf's declared parameter (ADR-0066).
	seeds := make([]string, 0, len(seedScopes))
	for _, s := range seedScopes {
		seeds = append(seeds, rw.CanonicalName(s))
	}
	seen := map[string]struct{}{}
	for _, name := range names {
		key := rw.CanonicalName(name)
		if strings.HasPrefix(key, "*.") || key == "*" {
			// A wildcard name is a subject nowhere and is never queried, so it has no parent here.
			continue
		}
		p, ok := parent(key)
		if !ok {
			continue
		}
		// Subtree containment is the model's one label-wise suffix test, never re-derived here.
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

func parent(name string) (string, bool) {
	i := strings.IndexByte(name, '.')
	if i < 0 || i == len(name)-1 {
		return "", false
	}
	return name[i+1:], true
}
