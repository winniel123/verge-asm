package message

import (
	"strings"
	"testing"
)

func TestNarrowingFiresOverInhabitedGround(t *testing.T) {
	r := PreviewNarrowing("198.51.100.0/24", "198.51.100.128/25", 128, 17920)
	if !r.Fires {
		t.Fatal("a narrowing withdrawing 128 subjects must fire")
	}
	if !strings.Contains(r.Headline, "128 subjects withdrawn") {
		t.Errorf("headline should state the subjects withdrawn: %q", r.Headline)
	}
	if !strings.Contains(r.Headline, "17,920 timelines") {
		t.Errorf("headline should state the timelines taken out: %q", r.Headline)
	}
	if r.Loss == "" {
		t.Error("a narrowing receipt must name the loss — there is no repair")
	}

	msg := Narrowing(r, "198.51.100.0/24", t0)
	if msg == nil {
		t.Fatal("a firing receipt must produce a message")
	}
	if msg.Cause != CauseAperture || msg.Class != ClassCoverage {
		t.Errorf("a narrowing is a coverage-class aperture firing, got cause=%q class=%q", msg.Cause, msg.Class)
	}
	if msg.LinkKind() != LinkSeed {
		t.Error("a narrowing links to the Seed whose scope moved, never to Coverage")
	}
	if msg.Census != nil {
		t.Error("a narrowing carries a count, not a census of rows")
	}
}

func TestNarrowingSilentWhereNothingWithdrawn(t *testing.T) {
	r := PreviewNarrowing("AMAZON-04", "52.94.0.0/16", 0, 0)
	if r.Fires {
		t.Error("a narrowing withdrawing nothing must not fire — no preview is owed")
	}
	if r.Headline != "" || r.Loss != "" {
		t.Error("no copy is rendered for a receipt that does not fire")
	}
	if got := Narrowing(r, "AMAZON-04", t0); got != nil {
		t.Errorf("a non-firing receipt produces no message, got %+v", got)
	}
}

func TestSeedWithdrawalStatesItsOwnHeadline(t *testing.T) {
	r := PreviewSeedWithdrawal("10.0.0.0/24", 3, 7)
	if !r.Fires {
		t.Fatal("a withdrawal taking 3 subjects must fire")
	}
	want := "10.0.0.0/24 withdrawn · 3 subjects withdrawn · 7 timelines taken out of the estate"
	if r.Headline != want {
		t.Errorf("headline:\n got %q\nwant %q", r.Headline, want)
	}
	if strings.Contains(r.Headline, "narrowed") || strings.Contains(r.Headline, "excluded") {
		t.Errorf("the withdrawal names no narrowing and no exclusion: %q", r.Headline)
	}
	if r.Scope != r.Removed {
		t.Errorf("the scope that moved and the ground that left are one object, got %q / %q", r.Scope, r.Removed)
	}
	if r.Loss == "" {
		t.Error("a withdrawal receipt must name the loss — there is no repair")
	}
	if ContainsValence(r.Headline) || ContainsValence(r.Loss) {
		t.Error("a withdrawal is neither good news nor bad — the copy carries no valence")
	}

	msg := Narrowing(r, r.Scope, t0)
	if msg == nil {
		t.Fatal("a firing receipt must produce a message")
	}
	if msg.Cause != CauseAperture || msg.Class != ClassCoverage {
		t.Errorf("a withdrawal is a coverage-class aperture firing, got cause=%q class=%q", msg.Cause, msg.Class)
	}
	if msg.LinkKind() != LinkSeed {
		t.Error("a withdrawal links to the Seed whose scope moved")
	}
	if msg.Census != nil {
		t.Error("a withdrawal carries a count, not a census of rows")
	}
}

func TestSeedWithdrawalSilentWhereNothingWithdrawn(t *testing.T) {
	r := PreviewSeedWithdrawal("10.0.0.0/24", 0, 0)
	if r.Fires {
		t.Error("a withdrawal taking nothing must not fire")
	}
	if r.Headline != "" || r.Loss != "" {
		t.Error("no copy is rendered for a receipt that does not fire")
	}
	if got := Narrowing(r, "10.0.0.0/24", t0); got != nil {
		t.Errorf("a non-firing receipt produces no message, got %+v", got)
	}
}
