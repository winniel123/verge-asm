package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
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

// T16 delta (#311): with a real batch dispatched, the Drift header offers a "Batch
// detail" entry into the Run detail screen at GET /run/{id} — id being the most
// recent Dispatch id. The entry is real data (a dispatch exists), never a fabricated
// change event, and it stands even while the transition timeline is still the
// empty-state (change and batches are distinct feeds).
func TestDriftBatchDetailLinksToRun(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// Two dispatches; the header links the most recent (id DESC → 88), not 87.
	older := time.Date(2026, 8, 22, 8, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(88, "hot", newer, 2, 0, 0, 2, 0, 0),
		progressRow(87, "hot", older, 2, 0, 0, 2, 0, 0),
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	if !strings.Contains(page, "Batch detail") {
		t.Errorf("drift page missing the Batch detail entry; body: %s", page)
	}
	if !strings.Contains(page, `href="/run/88"`) {
		t.Errorf("Batch detail should link to the most recent batch /run/88; body: %s", page)
	}
	if strings.Contains(page, `href="/run/87"`) {
		t.Errorf("Batch detail linked an older batch /run/87, not the latest; body: %s", page)
	}
	// The timeline is still the empty-state — the entry does not fabricate change.
	if !strings.Contains(page, "No change to show yet") {
		t.Errorf("Batch detail must not fabricate a transition feed; body: %s", page)
	}
}

// With no scan yet dispatched there is no batch to open, so the header offers no
// Batch detail entry rather than fabricate a run id — no /run/ link is rendered.
func TestDriftBatchDetailOmittedWithoutBatch(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	if strings.Contains(page, "Batch detail") || strings.Contains(page, `href="/run/`) {
		t.Errorf("drift page offered a Batch detail entry with no batch dispatched; body: %s", page)
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
