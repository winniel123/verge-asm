// Package wildcarddiscrim is the wildcard-discrimination Derivation leaf (v1 spec
// §3.3, ADR-0068, ADR-0086). No wildcard-content discrimination is implemented: the
// facet is priced out of v1 (§7) and DNSSEC's proof is unavailable on most zones (§3.6).
package wildcarddiscrim

import (
	"net/netip"
	"sort"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// Versioned apart from resolution-walk, so a break names its own leaf (golden-corpus.md §8).

const Version = "wildcard-discrimination/v1"

type Signature string // never the answer set, and never a union of the observed sets (ADR-0068)

const (
	NoSynthesis   Signature = "NoSynthesis" // a determinate reading, never an absent one
	Determinate   Signature = "Determinate"
	Indeterminate Signature = "Indeterminate" // never consulted: it can neither shadow nor exempt
)

type Verdict string

const (
	VerdictShadowed    Verdict = "Shadowed"
	VerdictNotShadowed Verdict = "NotShadowed"
	VerdictGap         Verdict = "Gap"
)

// An answer chain's parts differ in stability, so the key is finer than the qtype (ADR-0068).

type compKey struct {
	Asked    rw.Qtype
	Answered rw.Qtype
}

type component struct {
	Sig   Signature
	RRSet []string // canonical, sorted RDATA, so equalSet may compare element-wise
}

type controlAnswers struct {
	perLabel []map[rw.Qtype][]rw.RR
	reached  bool
}

func (ca controlAnswers) components() map[compKey]component {
	keys := map[compKey]struct{}{}
	for _, ans := range ca.perLabel {
		for asked, rrs := range ans {
			for _, rr := range rrs {
				keys[compKey{Asked: asked, Answered: rr.Type}] = struct{}{}
			}
		}
	}
	out := make(map[compKey]component, len(keys))
	for k := range keys {
		sets := make([][]string, 0, len(ca.perLabel))
		for _, ans := range ca.perLabel {
			sets = append(sets, rdataSet(ans[k.Asked], k.Answered))
		}
		out[k] = signatureOf(sets)
	}
	return out
}

func (ca controlAnswers) wildcardMeasured() bool {
	// ADR-0069's qualified licence: no wildcard means no label carried an RR at any qtype.
	for _, ans := range ca.perLabel {
		for _, rrs := range ans {
			if len(rrs) > 0 {
				return true
			}
		}
	}
	return false
}

func signatureOf(sets [][]string) component {
	anyNonEmpty := false
	for _, s := range sets {
		if len(s) > 0 {
			anyNonEmpty = true
			break
		}
	}
	if !anyNonEmpty {
		return component{Sig: NoSynthesis}
	}
	first := sets[0]
	for _, s := range sets[1:] {
		if !equalSet(first, s) {
			return component{Sig: Indeterminate}
		}
	}
	if len(first) == 0 {
		return component{Sig: Indeterminate}
	}
	return component{Sig: Determinate, RRSet: first}
}

func Discriminate(candidate map[compKey][]string, ctrl controlAnswers) Verdict {
	if !ctrl.reached {
		// An undiscriminated answer is never a value (ADR-0066).
		return VerdictGap
	}
	if !ctrl.wildcardMeasured() {
		// A completed probe that found no wildcard licenses everything beneath it (ADR-0069).
		return VerdictNotShadowed
	}
	comps := ctrl.components()
	if differsAtDeterminate(candidate, comps) {
		return VerdictNotShadowed
	}
	return VerdictShadowed
}

func differsAtDeterminate(candidate map[compKey][]string, comps map[compKey]component) bool {
	seen := map[compKey]struct{}{}
	check := func(k compKey) bool {
		if _, ok := seen[k]; ok {
			return false
		}
		seen[k] = struct{}{}
		cand := candidate[k]
		c, ok := comps[k]
		if !ok {
			// A component absent from the map is determinately NoSynthesis, never unknown.
			return len(cand) > 0
		}
		switch c.Sig {
		case NoSynthesis:
			return len(cand) > 0
		case Determinate:
			return !equalSet(cand, c.RRSet)
		default:
			return false
		}
	}
	for k := range candidate {
		if check(k) {
			return true
		}
	}
	// A candidate empty where the wildcard synthesises an RRset still differs (ADR-0068).
	for k, c := range comps {
		if c.Sig == Determinate && check(k) {
			return true
		}
	}
	return false
}

func candidateComponents(rec []rw.Record) map[compKey][]string {
	grouped := map[compKey][]rw.RR{}
	for _, r := range rec {
		for _, rr := range r.RRs {
			k := compKey{Asked: r.Qtype, Answered: rr.Type}
			grouped[k] = append(grouped[k], rr)
		}
	}
	out := make(map[compKey][]string, len(grouped))
	for k, rrs := range grouped {
		out[k] = canonRDATA(rrs)
	}
	return out
}

func rdataSet(rrs []rw.RR, answered rw.Qtype) []string {
	filtered := make([]rw.RR, 0, len(rrs))
	for _, rr := range rrs {
		if rr.Type == answered {
			filtered = append(filtered, rr)
		}
	}
	return canonRDATA(filtered)
}

func canonRDATA(rrs []rw.RR) []string {
	seen := map[string]struct{}{}
	for _, rr := range rrs {
		seen[canonOne(rr)] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func canonOne(rr rw.RR) string {
	// An IPv4-mapped AAAA folds onto its A key, so the two never read as a difference.
	switch rr.Type {
	case rw.QtypeA, rw.QtypeAAAA:
		if addr, err := netip.ParseAddr(rr.Data); err == nil {
			return addr.Unmap().String()
		}
		return rr.Data
	case rw.QtypeCNAME, rw.QtypeNS:
		return rw.CanonicalName(rr.Data)
	default:
		return rr.Data
	}
}

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
