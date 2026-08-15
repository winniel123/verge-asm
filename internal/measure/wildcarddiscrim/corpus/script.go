// Package corpus is the golden corpus for the wildcard-discrimination leaf
// (docs/spec/golden-corpus.md §8 — the 19-cell block ADR-0086 discharges into
// this leaf). It holds a checked-in enumeration of the W-cells the leaf owes a
// pinning row, each row being a (job-spec fragment, authored peer script,
// expected NDJSON) triple run hermetically against an in-process scripted peer
// and a deterministic control-label generator — no network, no containers.
//
// It is a sibling of resolutionwalk/corpus and never pooled with it: a row
// protects the leaf whose gate runs it and nothing else (golden-corpus.md §8.4),
// so the two blocks are counted together and run apart.
package corpus

import (
	"regexp"
	"strings"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// labelShape selects which control labels a rule answers, so a row can script an
// address-parsing authority (the structured label answers where the random ones
// do not) that reads Indeterminate.
type labelShape int

const (
	anyShape labelShape = iota
	randomShape
	structuredShape
)

var structuredLabelRE = regexp.MustCompile(`^\d{1,3}-\d{1,3}-\d{1,3}-\d{1,3}$`)

// scriptRule matches a subset of a declared-path Query and returns a scripted
// reply. An exact Name match is tried before an Under (parent-suffix) match, so a
// real name's own answer overrides the wildcard synthesis a fictional label gets.
type scriptRule struct {
	Name  string     // exact canonical name (highest precedence)
	Under string     // parent; matches any name ending ".<Under>"
	Shape labelShape // which leftmost-label shapes the Under rule answers
	Qtype rw.Qtype   // "" matches any qtype
	Reply rw.Msg
}

// ScriptPeer is a deterministic, in-process Peer built from ordered rules. An
// unmatched query returns a silent (not-reached) response, so a control probe
// with no matching rule reads as *did not complete* — a Gap, never a wrong value.
type ScriptPeer struct {
	Rules []scriptRule
}

// Exchange implements resolutionwalk.Peer.
func (s ScriptPeer) Exchange(q rw.Query) rw.Msg {
	if q.Path != rw.PathDeclared {
		// The leaf and its candidate resolution use only the declared path here;
		// a delegation walk finds no authority and folds to an empty Delegation.
		return rw.Msg{Reached: false}
	}
	for _, r := range s.Rules {
		if r.matches(q) {
			return r.Reply
		}
	}
	return rw.Msg{Reached: false}
}

func (r scriptRule) matches(q rw.Query) bool {
	if r.Qtype != "" && r.Qtype != q.Qtype {
		return false
	}
	name := rw.CanonicalName(q.Name)
	if r.Name != "" {
		return name == rw.CanonicalName(r.Name)
	}
	if r.Under != "" {
		under := rw.CanonicalName(r.Under)
		if !strings.HasSuffix(name, "."+under) {
			return false
		}
		leftmost := strings.TrimSuffix(name, "."+under)
		// Only an immediate child (one label) is a control label under Under.
		if strings.Contains(leftmost, ".") {
			return false
		}
		switch r.Shape {
		case structuredShape:
			return structuredLabelRE.MatchString(leftmost)
		case randomShape:
			return !structuredLabelRE.MatchString(leftmost)
		default:
			return true
		}
	}
	return false
}

// --- reply constructors, so a row's script reads like a zone answer ---

func noerror(answers ...rw.RR) rw.Msg {
	return rw.Msg{Reached: true, Rcode: rw.NOERROR, Answer: answers}
}
func nodata() rw.Msg   { return rw.Msg{Reached: true, Rcode: rw.NOERROR} }
func nxdomain() rw.Msg { return rw.Msg{Reached: true, Rcode: rw.NXDOMAIN} }

func rrA(name, ip string) rw.RR { return rw.RR{Name: name, Type: rw.QtypeA, Data: ip} }
func rrMX(name, mx string) rw.RR {
	return rw.RR{Name: name, Type: rw.QtypeMX, Data: mx}
}

// DeterministicLabels is the corpus LabelGen: a fixed set of the shipped shape —
// RandomLabelCount random labels plus StructuredLabelCount structured ones. The
// labels never appear in the leaf's output, so a fixed set renders byte-identical
// golden NDJSON while still exercising the structured-vs-random distinction the
// Indeterminate rows turn on.
type DeterministicLabels struct{}

// Labels implements wildcarddiscrim.LabelGen.
func (DeterministicLabels) Labels() []string {
	out := make([]string, 0, wd.LabelCount)
	for i := 0; i < wd.RandomLabelCount; i++ {
		out = append(out, "ctl"+string(rune('a'+i)))
	}
	// One structured label over an RFC 5737 documentation address.
	out = append(out, "192-0-2-5")
	return out
}
