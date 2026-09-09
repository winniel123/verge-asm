package message

import (
	"strings"
	"testing"
)

func TestRePointFiresDriftAtTheName(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "endpoint", Key: "www.example.com@203.0.113.5:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "www.example.com@203.0.113.5:80/tcp"},
	)
	msg := RePoint("www.example.com", census, t0)
	if msg == nil {
		t.Fatal("a non-empty residue must fire")
	}
	if msg.Cause != CauseDrift || msg.Class != ClassDrift {
		t.Errorf("the world moved, so this is drift; got cause=%q class=%q", msg.Cause, msg.Class)
	}
	if msg.SubjectKind != "name" || msg.FiredAt != "www.example.com" {
		t.Errorf("fires at the Name whose resolution moved, got %s/%s", msg.SubjectKind, msg.FiredAt)
	}
	if msg.CensusLen() != 2 {
		t.Errorf("census carries the residue Endpoints, got %d", msg.CensusLen())
	}
}

func TestRePointEmptyResidueFiresNothing(t *testing.T) {
	if got := RePoint("www.example.com", NewCensus(), t0); got != nil {
		t.Errorf("an empty residue is no message (ADR-0026 §2), got %+v", got)
	}
}

func TestRePointHeadlineReadsWithinTheEstate(t *testing.T) {
	census := NewCensus(
		CensusEntry{Kind: "endpoint", Key: "www.example.com@203.0.113.5:443/tcp"},
		CensusEntry{Kind: "endpoint", Key: "www.example.com@203.0.113.5:80/tcp"},
	)
	got := RePoint("www.example.com", census, t0).Headline
	want := "www.example.com re-pointed within the estate · 2 endpoints · 2 timelines opened beneath it"
	if got != want {
		t.Errorf("headline\n got %q\nwant %q", got, want)
	}
	entered := Membership(EntryAppeared, "address", "203.0.113.5", "", census, t0).Headline
	if !strings.HasPrefix(entered, "203.0.113.5 entered the estate") {
		t.Errorf("the Address half of the pair reads entered, got %q", entered)
	}
	if strings.Contains(got, "entered") {
		t.Errorf("the pair must not read as duplicates (ADR-0026 §2): %q", got)
	}
}
