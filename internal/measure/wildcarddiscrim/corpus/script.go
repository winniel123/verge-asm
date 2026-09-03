// Package corpus is the golden corpus for the wildcard-discrimination leaf
// (docs/spec/golden-corpus.md §8, the 19-cell block ADR-0086 discharges here).
// It is never pooled with resolutionwalk/corpus (golden-corpus.md §8.4).
package corpus

import (
	"regexp"
	"strings"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

type labelShape int

const (
	anyShape labelShape = iota
	randomShape
	structuredShape
)

var structuredLabelRE = regexp.MustCompile(`^\d{1,3}-\d{1,3}-\d{1,3}-\d{1,3}$`)

type scriptRule struct {
	Name  string
	Under string
	Shape labelShape
	Qtype rw.Qtype
	Reply rw.Msg
}

type ScriptPeer struct {
	Rules []scriptRule
}

func (s ScriptPeer) Exchange(q rw.Query) rw.Msg {
	if q.Path != rw.PathDeclared {
		// Only the declared path is scripted; a delegation walk folds to an empty Delegation.
		return rw.Msg{Reached: false}
	}
	for _, r := range s.Rules {
		if r.matches(q) {
			return r.Reply
		}
	}
	// An unmatched query is silence, so a probe with no rule gaps rather than taking a value.
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

func noerror(answers ...rw.RR) rw.Msg {
	return rw.Msg{Reached: true, Rcode: rw.NOERROR, Answer: answers}
}
func nodata() rw.Msg   { return rw.Msg{Reached: true, Rcode: rw.NOERROR} }
func nxdomain() rw.Msg { return rw.Msg{Reached: true, Rcode: rw.NXDOMAIN} }

func rrA(name, ip string) rw.RR { return rw.RR{Name: name, Type: rw.QtypeA, Data: ip} }
func rrMX(name, mx string) rw.RR {
	return rw.RR{Name: name, Type: rw.QtypeMX, Data: mx}
}

type DeterministicLabels struct{}

func (DeterministicLabels) Labels() []string {
	// The labels never reach the output, so a fixed set still renders byte-identical NDJSON.
	out := make([]string, 0, wd.LabelCount)
	for i := 0; i < wd.RandomLabelCount; i++ {
		out = append(out, "ctl"+string(rune('a'+i)))
	}
	// One structured label over an RFC 5737 documentation address.
	out = append(out, "192-0-2-5")
	return out
}
