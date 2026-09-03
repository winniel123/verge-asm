package wildcarddiscrim

import (
	"reflect"
	"testing"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func rrA(name, ip string) rw.RR  { return rw.RR{Name: name, Type: rw.QtypeA, Data: ip} }
func rrTXT(name, t string) rw.RR { return rw.RR{Name: name, Type: rw.QtypeTXT, Data: t} }
func rrMX(name, mx string) rw.RR { return rw.RR{Name: name, Type: rw.QtypeMX, Data: mx} }

func constWildcard(ip string) controlAnswers {
	ca := controlAnswers{reached: true}
	for i := 0; i < LabelCount; i++ {
		ca.perLabel = append(ca.perLabel, map[rw.Qtype][]rw.RR{
			rw.QtypeA: {rrA("l.example.com", ip)},
		})
	}
	return ca
}

func TestDiscriminateFictionalNameIsShadowed(t *testing.T) {
	ctrl := constWildcard("203.0.113.1")
	cand := map[compKey][]string{{rw.QtypeA, rw.QtypeA}: {"203.0.113.1"}}
	if got := Discriminate(cand, ctrl); got != VerdictShadowed {
		t.Errorf("fictional name = %q, want Shadowed", got)
	}
}

func TestDiscriminateRealNameDiffersAtDeterminate(t *testing.T) {
	ctrl := constWildcard("203.0.113.1")
	cand := map[compKey][]string{{rw.QtypeA, rw.QtypeA}: {"198.51.100.9"}}
	if got := Discriminate(cand, ctrl); got != VerdictNotShadowed {
		t.Errorf("differing name = %q, want NotShadowed", got)
	}
}

func TestDiscriminateDiffersAtNoSynthesisComponent(t *testing.T) {
	ctrl := constWildcard("203.0.113.1")
	cand := map[compKey][]string{
		{rw.QtypeA, rw.QtypeA}:     {"203.0.113.1"},
		{rw.QtypeTXT, rw.QtypeTXT}: {`"v=spf1"`},
	}
	if got := Discriminate(cand, ctrl); got != VerdictNotShadowed {
		t.Errorf("differ-at-NoSynthesis = %q, want NotShadowed", got)
	}
}

func TestDiscriminateIndeterminateNeverConsulted(t *testing.T) {
	ca := controlAnswers{reached: true}
	ca.perLabel = append(ca.perLabel, map[rw.Qtype][]rw.RR{rw.QtypeA: {rrA("l.example.com", "203.0.113.7")}})
	for i := 1; i < LabelCount; i++ {
		ca.perLabel = append(ca.perLabel, map[rw.Qtype][]rw.RR{})
	}
	cand := map[compKey][]string{{rw.QtypeA, rw.QtypeA}: {"203.0.113.7"}}
	if got := Discriminate(cand, ca); got != VerdictShadowed {
		t.Errorf("coincide-at-Indeterminate = %q, want Shadowed (never consulted)", got)
	}
}

func TestDiscriminateNoWildcardLicensesEverything(t *testing.T) {
	ca := controlAnswers{reached: true}
	for i := 0; i < LabelCount; i++ {
		ca.perLabel = append(ca.perLabel, map[rw.Qtype][]rw.RR{})
	}
	if got := Discriminate(map[compKey][]string{}, ca); got != VerdictNotShadowed {
		t.Errorf("no-wildcard licence = %q, want NotShadowed", got)
	}
}

func TestDiscriminateIncompleteProbeIsGap(t *testing.T) {
	if got := Discriminate(map[compKey][]string{}, controlAnswers{reached: false}); got != VerdictGap {
		t.Errorf("incomplete probe = %q, want Gap", got)
	}
}

func TestControlPopulationIsParentsInsideSeeds(t *testing.T) {
	got := ControlPopulation(
		[]string{"www.iana.org", "int.iana.org", "iana.org", "*.iana.org"},
		[]string{"iana.org"},
	)
	want := []string{"iana.org"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("population = %v, want %v", got, want)
	}
}

func TestControlPopulationKeepsDeeperParents(t *testing.T) {
	got := ControlPopulation(
		[]string{"a.int.iana.org", "b.rzm.iana.org"},
		[]string{"iana.org"},
	)
	want := []string{"int.iana.org", "rzm.iana.org"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("population = %v, want %v", got, want)
	}
}

func TestParamsDigestStableAndSensitive(t *testing.T) {
	a := DefaultParams()
	if a.Digest() != DefaultParams().Digest() {
		t.Error("digest is not stable across calls")
	}
	b := DefaultParams()
	b.RandomLabelCount = 5 // the #115 move, in reverse
	if a.Digest() == b.Digest() {
		t.Error("digest did not move when the control-label count changed")
	}
}

func TestCandidateComponentsSplitByAskedAndAnswered(t *testing.T) {
	recs := []rw.Record{{Qtype: rw.QtypeA, RRs: []rw.RR{
		{Name: "x.example.com", Type: rw.QtypeCNAME, Data: "c.example.net"},
		rrA("c.example.net", "203.0.113.9"),
	}}}
	comps := candidateComponents(recs)
	if _, ok := comps[compKey{rw.QtypeA, rw.QtypeCNAME}]; !ok {
		t.Error("missing (A,CNAME) component")
	}
	if got := comps[compKey{rw.QtypeA, rw.QtypeA}]; !reflect.DeepEqual(got, []string{"203.0.113.9"}) {
		t.Errorf("(A,A) component = %v, want [203.0.113.9]", got)
	}
	_ = rrTXT
	_ = rrMX
}
