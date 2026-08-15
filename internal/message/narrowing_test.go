package message

import (
	"strings"
	"testing"
)

// A narrowing over inhabited ground fires: the receipt carries the subject and
// timeline counts, names the loss, and produces a coverage-class message at the
// scope linking to the Seed.
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

// A narrowing that withdraws no subject is silent: an empty residue (declining a
// Proposal that was never a subject) or ground a current resolution still cites.
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
