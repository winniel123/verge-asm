package exposure

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
)

func valued(v ReachValue) Leg { return Leg{Status: LegValued, Value: v} }

// --- ComposeReach: the class-scoped existential quantifier (ADR-0080) -------

func TestComposeReachExistential(t *testing.T) {
	cases := []struct {
		name     string
		outcomes []string
		want     ReachValue
		ok       bool
	}{
		{"one vantage reached is reached", []string{"not-reached", "reached"}, Reached, true},
		{"reached wins over any not-reached", []string{"reached", "not-reached", "not-reached"}, Reached, true},
		{"all not-reached is not-reached", []string{"not-reached", "not-reached"}, NotReached, true},
		{"single not-reached decides", []string{"not-reached"}, NotReached, true},
		{"empty in-scope set holds no value", nil, "", false},
		{"only connectionless-undecided holds no value", []string{"undecided", "no-answer"}, "", false},
		{"a decided not-reached beside noise is not-reached", []string{"undecided", "not-reached"}, NotReached, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ComposeReach(c.outcomes)
			if got != c.want || ok != c.ok {
				t.Fatalf("ComposeReach(%v) = (%q, %v), want (%q, %v)", c.outcomes, got, ok, c.want, c.ok)
			}
		})
	}
}

// AC #196: the 2×2 board renders all four Exposure values correctly.
func TestProjectAllFourValues(t *testing.T) {
	cases := []struct {
		internet, internal ReachValue
		want               ExposureValue
	}{
		{Reached, Reached, Exposed},
		{Reached, NotReached, EdgeOnly},
		{NotReached, Reached, Firewalled},
		{NotReached, NotReached, Unreachable},
	}
	for _, c := range cases {
		got, ok := Project(valued(c.internet), valued(c.internal))
		if !ok || got != c.want {
			t.Errorf("Project(internet=%s, internal=%s) = (%q, %v), want %q", c.internet, c.internal, got, ok, c.want)
		}
	}
}

// AC #196 (the two-legs precondition): Exposure is constructed ONLY where both
// legs hold a value — a one-legged reading yields no value and never a false
// internal-only reading (ADR-0017).
func TestProjectNeedsBothLegs(t *testing.T) {
	absents := []Leg{
		{Status: LegNeverConfigured},
		{Status: LegGap},
	}
	for _, missing := range absents {
		if _, ok := Project(valued(Reached), missing); ok {
			t.Errorf("a missing internal leg (%s) must yield no Exposure, got a value", missing.Status)
		}
		if _, ok := Project(missing, valued(Reached)); ok {
			t.Errorf("a missing internet leg (%s) must yield no Exposure, got a value", missing.Status)
		}
	}
	if _, ok := Project(Leg{Status: LegNeverConfigured}, Leg{Status: LegNeverConfigured}); ok {
		t.Error("two absent legs must yield no Exposure")
	}
}

// --- Flagship: the internet leg move, never an Exposure state (ADR-0029) ----

func TestFlagshipOnlyNotReachedToReached(t *testing.T) {
	if !Flagship(NotReached, Reached) {
		t.Error("not-reached -> reached is the flagship move")
	}
	for _, c := range []struct{ before, after ReachValue }{
		{Reached, NotReached},    // closing to the internet — recorded, not the flagship
		{Reached, Reached},       // no move
		{NotReached, NotReached}, // no move
	} {
		if Flagship(c.before, c.after) {
			t.Errorf("Flagship(%s, %s) must be false", c.before, c.after)
		}
	}
}

// AC #196: Vantage class is re-verified every Batch against the PRESENTED
// address, not a static config field (CONTEXT.md `Vantage class`). The quantifier
// is every-not-any and the closed direction is `internet`.
func TestVerifyClassFromPresentedAddress(t *testing.T) {
	// The operator's declared address scope — the boundary a presented address is
	// tested against, family-matched.
	scope := netip.MustParsePrefix("10.0.0.0/8")
	covered := func(a netip.Addr) bool { return scope.Contains(a.Unmap()) }

	cases := []struct {
		name      string
		presented []netip.Addr
		want      custody.VantageClass
	}{
		{
			name:      "no presented address observed is unverified (no prober)",
			presented: nil,
			want:      custody.ClassUnverified,
		},
		{
			name:      "every presented address covered verifies internal",
			presented: []netip.Addr{netip.MustParseAddr("10.1.2.3")},
			want:      custody.ClassInternal,
		},
		{
			name:      "one uncovered presented address verifies internet (closed direction)",
			presented: []netip.Addr{netip.MustParseAddr("10.1.2.3"), netip.MustParseAddr("52.1.2.3")},
			want:      custody.ClassInternet,
		},
		{
			name:      "a single public presented address verifies internet",
			presented: []netip.Addr{netip.MustParseAddr("203.0.113.9")},
			want:      custody.ClassInternet,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := VerifyClass(c.presented, covered); got != c.want {
				t.Fatalf("VerifyClass(%v) = %q, want %q", c.presented, got, c.want)
			}
		})
	}
}

// The class is a property of where the prober SITS, computed anew from whatever
// the batch presented — the same vantage re-presents a different address next
// batch and re-verifies to the other class, which is exactly what a static field
// could not do.
func TestVerifyClassIsRecomputedNotStatic(t *testing.T) {
	scope := netip.MustParsePrefix("10.0.0.0/8")
	covered := func(a netip.Addr) bool { return scope.Contains(a.Unmap()) }

	batch1 := []netip.Addr{netip.MustParseAddr("10.0.0.5")}
	batch2 := []netip.Addr{netip.MustParseAddr("198.51.100.5")}
	if VerifyClass(batch1, covered) != custody.ClassInternal {
		t.Error("batch 1 presented a covered address — internal")
	}
	if VerifyClass(batch2, covered) != custody.ClassInternet {
		t.Error("batch 2 presented an uncovered address — internet, re-verified not remembered")
	}
}

// AC #196: with no Service in the estate at all, the screen renders the
// no-Service precondition, never a blank grid.
func TestBuildNoServices(t *testing.T) {
	s := Build(nil, true, true)
	if !s.NoServices {
		t.Fatal("an estate with no Service must set the NoServices precondition")
	}
	if s.Board.Total() != 0 {
		t.Error("no Service means no board")
	}
}

// AC #196: below two Vantage classes holding a value, no Exposure is
// constructible — the surviving leg's raw Reach renders one-legged under
// "we never looked," never a false internal-only reading. This is the modal
// no-prober install: internal present, internet never configured.
func TestBuildOneLeggedNeverLooked(t *testing.T) {
	services := []ServiceInput{
		{
			Service:  "10.0.0.1:6379/tcp",
			Internet: Leg{Status: LegNeverConfigured},
			Internal: valued(Reached),
		},
	}
	s := Build(services, false /*internet*/, true /*internal*/)
	if s.Constructible {
		t.Error("one class present must not be Constructible")
	}
	if s.Board.Total() != 0 {
		t.Error("no Exposure is constructible, so the board is empty — never a fabricated value")
	}
	if len(s.OneLegged) != 1 {
		t.Fatalf("the surviving internal leg must render one-legged, got %d rows", len(s.OneLegged))
	}
	row := s.OneLegged[0]
	if row.Class != "internal" || row.Value != Reached {
		t.Errorf("the surviving leg is the internal Reach, got class=%s value=%s", row.Class, row.Value)
	}
	if row.Reason != NeverLooked {
		t.Errorf("a never-configured internet leg carries 'we never looked', got %q", row.Reason)
	}
}

// A configured leg that went silent renders one-legged too, but with the
// "we stopped looking" statement — the two absences keep their two statements
// (ADR-0017 decision 4).
func TestBuildOneLeggedStoppedLooking(t *testing.T) {
	services := []ServiceInput{
		{
			Service:  "203.0.113.5:443/tcp",
			Internet: valued(Reached),
			Internal: Leg{Status: LegGap},
		},
	}
	s := Build(services, true, true)
	if len(s.OneLegged) != 1 {
		t.Fatalf("a Gap on one leg renders the surviving leg one-legged, got %d", len(s.OneLegged))
	}
	if s.OneLegged[0].Reason != StoppedLooking {
		t.Errorf("a silenced internal leg carries 'we stopped looking', got %q", s.OneLegged[0].Reason)
	}
}

// AC #196: a populated board renders all four values; and a precondition render
// (a one-legged Service) co-exists with the populated board as distinct renders
// of the same screen.
func TestBuildPopulatedBoardCoexistsWithPrecondition(t *testing.T) {
	services := []ServiceInput{
		{Service: "a:443/tcp", Internet: valued(Reached), Internal: valued(Reached)},        // exposed
		{Service: "b:8080/tcp", Internet: valued(Reached), Internal: valued(NotReached)},    // edge-only
		{Service: "c:22/tcp", Internet: valued(NotReached), Internal: valued(Reached)},      // firewalled
		{Service: "d:9000/tcp", Internet: valued(NotReached), Internal: valued(NotReached)}, // unreachable
		{Service: "e:6379/tcp", Internet: valued(Reached), Internal: Leg{Status: LegGap}},   // one-legged (Gap)
	}
	s := Build(services, true, true)
	if s.Board.Total() != 4 {
		t.Fatalf("four both-legged Services populate the board, got %d", s.Board.Total())
	}
	if len(s.Board.Exposed) != 1 || len(s.Board.EdgeOnly) != 1 || len(s.Board.Firewalled) != 1 || len(s.Board.Unreachable) != 1 {
		t.Errorf("each of the four cells holds exactly one Service: %+v", s.Board)
	}
	if len(s.OneLegged) != 1 {
		t.Fatalf("the one-legged Service co-exists with the board, got %d one-legged rows", len(s.OneLegged))
	}
	if s.Board.Exposed[0] != "a:443/tcp" {
		t.Errorf("exposed cell holds the both-reached Service, got %v", s.Board.Exposed)
	}
}

// AC #196: a Break on the composing derivation renders "rules changed, nothing to
// compare yet" per Service — never a false Exposure value — and co-exists with
// the board the un-broken Services populate.
func TestBuildBreakCoexistsWithBoard(t *testing.T) {
	services := []ServiceInput{
		{Service: "clean:443/tcp", Internet: valued(Reached), Internal: valued(Reached)},
		{Service: "broke:443/tcp", Internet: valued(Reached), Internal: valued(Reached), Broken: true},
	}
	s := Build(services, true, true)
	if len(s.Broken) != 1 || s.Broken[0] != "broke:443/tcp" {
		t.Fatalf("the broken Service renders as rules-changed, got %v", s.Broken)
	}
	if s.Board.Total() != 1 || s.Board.Exposed[0] != "clean:443/tcp" {
		t.Errorf("the un-broken Service still populates the board: %+v", s.Board)
	}
}

// AC #196: the "what moved" panel shows the flagship internet not-reached ->
// reached transition when it occurs — computed on the internet leg alone, whether
// or not an Exposure exists (fires on a one-legged install too).
func TestBuildWhatMovedFlagship(t *testing.T) {
	services := []ServiceInput{
		{
			// Both legs valued: this one is on the board AND flagged as moved.
			Service:           "moved:443/tcp",
			Internet:          valued(Reached),
			Internal:          valued(NotReached),
			InternetBefore:    NotReached,
			InternetBeforeSet: true,
		},
		{
			// One-legged install (no internal leg) still fires the flagship.
			Service:           "oneleg:443/tcp",
			Internet:          valued(Reached),
			Internal:          Leg{Status: LegNeverConfigured},
			InternetBefore:    NotReached,
			InternetBeforeSet: true,
		},
		{
			// A leg that opened at reached (no prior value) emits no Transition.
			Service:           "opened:443/tcp",
			Internet:          valued(Reached),
			Internal:          valued(Reached),
			InternetBeforeSet: false,
		},
		{
			// A closing internet leg is recorded, never the flagship.
			Service:           "closed:443/tcp",
			Internet:          valued(NotReached),
			Internal:          valued(Reached),
			InternetBefore:    Reached,
			InternetBeforeSet: true,
		},
	}
	s := Build(services, true, true)
	if len(s.WhatMoved) != 2 {
		t.Fatalf("two Services crossed not-reached -> reached, got %v", s.WhatMoved)
	}
	want := map[string]bool{"moved:443/tcp": true, "oneleg:443/tcp": true}
	for _, svc := range s.WhatMoved {
		if !want[svc] {
			t.Errorf("unexpected flagship move for %q", svc)
		}
	}
}
