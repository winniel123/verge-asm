package main

import (
	"net/http"
	"strings"
	"testing"
)

// The Drift screen renders the change vocabulary (the legend) on the drift palette
// and, with no transition feed yet, the empty-state timeline — never a fabricated
// change event. It is a first-class screen (nav item 4 of 7): the full composition
// is present even where the data is thin.
func TestDriftPageRendersVocabularyAndEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	// Title and the change-not-severity subtitle.
	for _, want := range []string{"Drift", "Change is its own language", "By batch", "Movement"} {
		if !strings.Contains(page, want) {
			t.Errorf("drift page missing %q; body: %s", want, page)
		}
	}

	// The full change vocabulary renders as chips on the drift palette — never the
	// severity ramp. Each kind rides its family's chip class (gain/change/loss).
	for _, kind := range []string{"appeared", "revealed", "withdrawn", "descoped", "returned", "changed"} {
		if !strings.Contains(page, kind) {
			t.Errorf("drift page missing change kind %q; body: %s", kind, page)
		}
	}
	for _, cls := range []string{"chip gain", "chip change", "chip loss"} {
		if !strings.Contains(page, cls) {
			t.Errorf("drift page missing drift-palette chip class %q; body: %s", cls, page)
		}
	}
	// Change is its own palette, never the severity ramp: the screen body carries no
	// severity pill. (The shared stylesheet in <head> defines the .sev-* classes for
	// every page, so we assert on a rendered pill element, not the class name.)
	for _, pill := range []string{`class="sev sev-critical"`, `class="sev sev-high"`} {
		if strings.Contains(page, pill) {
			t.Errorf("drift page rendered a severity pill %q — change must ride the drift palette only", pill)
		}
	}

	// No transition feed exists yet, so the timeline is the design-system empty-state
	// (fact + next action), not a fabricated batch.
	if !strings.Contains(page, "emptystate") {
		t.Errorf("drift page missing empty-state block; body: %s", page)
	}
	if !strings.Contains(page, "No change to show yet") {
		t.Errorf("drift page empty-state missing its fact; body: %s", page)
	}
	if !strings.Contains(page, "run twice") {
		t.Errorf("drift page empty-state missing its next action; body: %s", page)
	}

	// The Drift nav pill is the active one (keyed on NavActive, not Active).
	if !strings.Contains(page, `href="/drift"`) || !strings.Contains(page, `navpill active`) {
		t.Errorf("drift page did not mark the Drift nav pill active; body: %s", page)
	}
}

// The Drift route is behind requireLogin: an unauthenticated request redirects to
// the login form rather than rendering the screen.
func TestDriftRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/drift")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /drift: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}

// driftFamily maps each change kind to its drift palette family exactly as
// ChangeBadge.jsx's FAMILY does — gain (appeared/revealed/returned), loss
// (withdrawn/descoped), change (changed) — and never onto the severity ramp.
func TestDriftFamilyMapsToDriftPalette(t *testing.T) {
	for kind, want := range map[string]string{
		"appeared":  "gain",
		"revealed":  "gain",
		"returned":  "gain",
		"withdrawn": "loss",
		"descoped":  "loss",
		"changed":   "change",
	} {
		if got := driftFamily(kind); got != want {
			t.Errorf("driftFamily(%q) = %q, want %q", kind, got, want)
		}
	}
}
