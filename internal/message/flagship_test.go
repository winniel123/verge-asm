package message

import (
	"strings"
	"testing"
)

// The flagship fires on the internet leg going not-reached -> reached, whether
// or not an internal leg exists, carrying the census of facets that opened.
func TestFlagshipFiresRegardlessOfInternalLeg(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "facet", Key: "certificate"},
		CensusEntry{Kind: "facet", Key: "http-identity"},
		CensusEntry{Kind: "facet", Key: "tls-acceptance"},
	)
	msg := Flagship(ReachMove{
		ServiceKey: "198.51.100.1:443/tcp", Class: ClassInternet,
		From: NotReached, To: Reached,
	}, census, t0)
	if msg == nil {
		t.Fatal("the internet not-reached -> reached move is the flagship and must fire")
	}
	if msg.Cause != CauseDrift || msg.Class != ClassDrift {
		t.Errorf("the flagship is a drift firing, got cause=%q class=%q", msg.Cause, msg.Class)
	}
	if msg.SubjectKind != "service" || msg.FiredAt != "198.51.100.1:443/tcp" {
		t.Errorf("the flagship fires at the Service, got %q/%q", msg.SubjectKind, msg.FiredAt)
	}
	if msg.CensusLen() != 3 {
		t.Errorf("the flagship carries the census of every facet opening beneath, got %d", msg.CensusLen())
	}
	if msg.LinkKind() != LinkObject {
		t.Error("the flagship links to the object's page")
	}
	if !strings.Contains(msg.Headline, "reached from the internet") {
		t.Errorf("headline should name the move: %q", msg.Headline)
	}
}

// The internal leg never produces a message, in either direction — an internal
// port opening or closing is recorded and never alerted.
func TestInternalLegNeverMessages(t *testing.T) {
	for _, m := range []ReachMove{
		{ServiceKey: "10.0.0.1:22/tcp", Class: ClassInternal, From: NotReached, To: Reached},
		{ServiceKey: "10.0.0.1:22/tcp", Class: ClassInternal, From: Reached, To: NotReached},
	} {
		if got := Flagship(m, NewCensus(), t0); got != nil {
			t.Errorf("an internal-leg move (%s -> %s) must never message, got %+v", m.From, m.To, got)
		}
	}
}

// The internet leg's shrinking direction (reached -> not-reached) is silent — it
// is not the flagship.
func TestInternetShrinkIsSilent(t *testing.T) {
	if got := Flagship(ReachMove{
		ServiceKey: "198.51.100.1:443/tcp", Class: ClassInternet,
		From: Reached, To: NotReached,
	}, NewCensus(), t0); got != nil {
		t.Errorf("the internet reached -> not-reached move is silent, got %+v", got)
	}
}

// A leg that opened at reached — the first look ever — emits no Transition, so
// the flagship predicate does not match it; that news rides the membership
// census instead.
func TestOpenedAtReachedIsNotFlagship(t *testing.T) {
	if got := Flagship(ReachMove{
		ServiceKey: "198.51.100.1:443/tcp", Class: ClassInternet,
		To: Reached, Opened: true,
	}, NewCensus(), t0); got != nil {
		t.Errorf("a leg opening at reached is carried by the census, not the flagship, got %+v", got)
	}
}
