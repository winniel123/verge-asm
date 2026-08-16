package main

import (
	"strings"
	"testing"
)

// The UI-polish fixes from #246 live in the shared template CSS and the foot
// script. They render on every screen, so a coarse regression guard here keeps
// them from being dropped by a later edit to the stylesheet block.

// The scroll-preservation script rides every chrome page, delivered through the
// shared "foot" template. Its absence would return every form action to the
// top-of-page reload #246 set out to fix.
func TestFootCarriesScrollPreservation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsBody(t, ac, base)
	for _, want := range []string{`sessionStorage`, `"verge:scroll:"`, `window.scrollTo`} {
		if !strings.Contains(page, want) {
			t.Fatalf("rendered page missing scroll-preservation marker %q", want)
		}
	}
}

// A badge must never wrap mid-box: multi-token states like "1 name · 0 address"
// split the bordered pill across two lines without this (#246, SS1).
func TestBadgeDoesNotWrap(t *testing.T) {
	if !strings.Contains(pageCSS, ".badge {") {
		t.Fatal("pageCSS has no .badge rule")
	}
	rule := pageCSS[strings.Index(pageCSS, ".badge {"):]
	rule = rule[:strings.Index(rule, "}")]
	if !strings.Contains(rule, "white-space: nowrap") {
		t.Fatalf(".badge rule lacks white-space: nowrap; got: %s", rule)
	}
}

// A dial submit button aligns to its input, not the input-plus-helper stack: the
// helper unit renders inline beside a content-width input rather than as a block
// below it (#246, SS4/SS5).
func TestDialUnitIsInline(t *testing.T) {
	if !strings.Contains(pageCSS, ".dial .unit {") {
		t.Fatal("pageCSS has no .dial .unit rule")
	}
	rule := pageCSS[strings.Index(pageCSS, ".dial .unit {"):]
	rule = rule[:strings.Index(rule, "}")]
	if !strings.Contains(rule, "display: inline") {
		t.Fatalf(".dial .unit rule lacks display: inline; got: %s", rule)
	}
}
