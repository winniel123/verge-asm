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

// ADR-0130 §2 hardens that script. The stash keys on the full URL, and the
// restore fires on a navigation-type gate instead of a short freshness budget.
// A regression to either half silently returns a failure class the map closed.
func TestFootScrollRestoreHonoursADR0130(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsBody(t, ac, base)
	for _, want := range []string{
		`"verge:scroll:" + location.pathname + location.search`,
		`getEntriesByType("navigation")`,
		`nav.type === "navigate"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("rendered page missing ADR-0130 §2 marker %q", want)
		}
	}
	if strings.Contains(page, "FRESH_MS") {
		t.Fatal("the FRESH_MS freshness window is back; ADR-0130 §2 replaced it with a navigation-type gate")
	}
}

// The #246 .badge / .dial pageCSS-rule guards retired with pageCSS itself under P4.4
// (the shell conversion, map #22): the badge and dial styling now lives in the
// design-owned frozen templates + tokens, not a repo stylesheet const, so a repo-side
// CSS-substring guard no longer has anything to assert. The design-system G1/G2 gates
// hold that styling now.
