// Package wildcarddiscrim is the `wildcard-discrimination` Derivation leaf inside
// the shared measurement binary (v1 spec §3.3). It decides `Shadowed` — the
// value a `resolution` (and every `dns-record`) observation takes when a Name's
// answer was **not discriminated from its parent's synthesis**. It is versioned
// separately from `resolution-walk` and, with it, is one of the two leaves the
// membership vector composes (ADR-0086): `Shadowed` cites no `Address`, so the
// verdict decides whether an answer's address set enters the estate.
//
// The leaf reads two measurements on **one** query path inside one Batch
// (ADR-0070): the Name's own answer, and a **control probe** run under the
// Name's parent. A wildcard is discriminated only where its synthesis is
// **determinate**, measured per `(qtype asked, RR type answered)` component
// (ADR-0068): a component is `NoSynthesis` when no control label carried that RR,
// `Determinate(RRset)` when every control label carried the same one, and
// `Indeterminate` otherwise — and an `Indeterminate` component is never
// consulted. A Name is `Shadowed` unless it **differs at some determinate
// component**; where the probe found no wildcard at all it licenses everything
// beneath it, and where it did not complete the Name records a `Gap`, never a
// value.
//
// No `wildcard-synthesis` (wildcard-content) discrimination is implemented: the
// facet is priced out of v1 (§7), and DNSSEC's proof — the only sound second
// discriminator — is unavailable on most measured zones (§3.6). The leaf decides
// exactly the pass/fail `Shadowed` value and nothing about a wildcard's content.
package wildcarddiscrim

import (
	"net/netip"
	"sort"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// Version is the leaf's Derivation version (ADR-0008/ADR-0021). It moves on an
// output-affecting change and only on one, gated bidirectionally by this leaf's
// own golden corpus — separately from `resolution-walk`, so a break names its
// leaf (golden-corpus.md §8).
const Version = "wildcard-discrimination/v1"

// Signature is one component's determinacy verdict, the closed union of three
// (ADR-0068). Never *the* answer set, and never a union of the observed sets.
type Signature string

const (
	// NoSynthesis: no control label carried this RR — a determinate reading that
	// there is no wildcard synthesising this component, not an absent one.
	NoSynthesis Signature = "NoSynthesis"
	Determinate Signature = "Determinate"
	// Indeterminate: the control labels disagreed. Never consulted — it can
	// neither shadow a Name nor exempt one.
	Indeterminate Signature = "Indeterminate"
)

type Verdict string

const (
	VerdictShadowed    Verdict = "Shadowed"
	VerdictNotShadowed Verdict = "NotShadowed"
	// VerdictGap: the control probe under the Name's parent did not complete. An
	// undiscriminated answer is never a value (ADR-0066).
	VerdictGap Verdict = "Gap"
)

// compKey identifies one component: the qtype asked and the RR type answered.
// The answer to one qtype is a chain whose parts have different stabilities
// (ADR-0068), so determinacy is keyed finer than the qtype.
type compKey struct {
	Asked    rw.Qtype
	Answered rw.Qtype
}

// component is a per-compKey determinacy verdict, carrying the RRset only where
// Determinate — never a union, which is the object an intersection predicate
// would read back.
type component struct {
	Sig   Signature
	RRSet []string // canonical, sorted RDATA; set only when Sig == Determinate
}

type controlAnswers struct {
	perLabel []map[rw.Qtype][]rw.RR
	reached  bool // at least one control query was reached
}

// components folds a control probe's answers into the per-component signature
// union. A component appears here when some label carried that (asked, answered)
// RR; a component no label carried is determinately NoSynthesis and is consulted
// on the candidate side (differsAt) rather than enumerated here.
func (ca controlAnswers) components() map[compKey]component {
	// Gather, per component, each label's RDATA set (empty where the label had
	// no such RR). A component's key set is the union of RR types any label
	// carried under each asked qtype.
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

// wildcardMeasured is true when any control label carried any RR at any qtype —
// ADR-0069's qualified licence: a probe finds *no wildcard* only where no
// control label of any shape carried an RR at any qtype.
func (ca controlAnswers) wildcardMeasured() bool {
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
	// All equal; non-empty (anyNonEmpty and all equal to first ⇒ first non-empty).
	if len(first) == 0 {
		return component{Sig: Indeterminate}
	}
	return component{Sig: Determinate, RRSet: first}
}

func Discriminate(candidate map[compKey][]string, ctrl controlAnswers) Verdict {
	if !ctrl.reached {
		// The probe under the parent did not complete: a Gap, never a value.
		return VerdictGap
	}
	if !ctrl.wildcardMeasured() {
		// A probe that completed and found no wildcard licenses everything
		// beneath it — the modal case, and NameError beneath it stands.
		return VerdictNotShadowed
	}
	comps := ctrl.components()
	if differsAtDeterminate(candidate, comps) {
		return VerdictNotShadowed
	}
	return VerdictShadowed
}

// differsAtDeterminate is the match predicate: the candidate is discriminated
// when it differs at some determinate component — a different RRset where the
// control determinately had one, or an RRset where the control determinately had
// none. It checks every component the candidate carries, plus every component the
// control determinately carried (so a candidate empty where the wildcard
// synthesises an RRset still counts as differing).
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
			// No control label carried this component: determinately NoSynthesis.
			// The candidate differs iff it carries the RR the control did not.
			return len(cand) > 0
		}
		switch c.Sig {
		case NoSynthesis:
			return len(cand) > 0
		case Determinate:
			return !equalSet(cand, c.RRSet)
		default: // Indeterminate — never consulted.
			return false
		}
	}
	for k := range candidate {
		if check(k) {
			return true
		}
	}
	for k, c := range comps {
		if c.Sig == Determinate && check(k) {
			return true
		}
	}
	return false
}

// candidateComponents folds a resolution-walk Result's declared-path records into
// per-component RDATA sets, keyed the same way the control probe is, so the two
// are compared component for component.
func candidateComponents(rec []rw.Record) map[compKey][]string {
	// Group RRs by (asked qtype, answered RR type) first, then canonicalise.
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

// canonRDATA renders a set of RRs to canonical, sorted, de-duplicated RDATA
// strings: addresses by family and octets (so an IPv4-mapped AAAA folds to its A
// key), names ASCII-case-folded, everything else verbatim.
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
