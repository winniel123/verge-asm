package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// driftEventRow is a small builder for a raw feed row in the classifier unit tests.
func driftOpenedRow(batchID int64, at time.Time, subject, value, prevValue string) db.ListRecentDriftEventsRow {
	const vec = `[{"leaf":"resolution-walk","version":"1"}]`
	row := db.ListRecentDriftEventsRow{
		Role: "opened", BatchID: batchID, BatchKind: "hot",
		BatchAt:     pgtype.Timestamptz{Time: at, Valid: true},
		RecordedScope: []byte(`{}`),
		SubjectKind: "name", SubjectKey: subject, Facet: "resolution",
		Value: []byte(value), Derivation: []byte(vec),
		OpenedAt: pgtype.Timestamptz{Time: at, Valid: true},
	}
	if prevValue != "" {
		row.PrevValue = []byte(prevValue)
		row.PrevDerivation = []byte(vec)
	}
	return row
}

// buildDriftFeed classifies raw span events into the six change kinds and groups them
// by batch (#288, ADR-0111): a first span on a timeline is `appeared`, a later value
// move under the same Derivation vector is `changed` with a before/after diff, and the
// movement tally counts each kind. Rows arrive newest-batch-first.
func TestBuildDriftFeedClassifiesAndGroups(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	newer := now.Add(-1 * time.Hour)
	older := now.Add(-24 * time.Hour)

	rows := []db.ListRecentDriftEventsRow{
		// Newest batch: the value moved Resolved -> NXDOMAIN.
		driftOpenedRow(2, newer, "api.example.com", `{"outcome":"NXDOMAIN"}`, `{"outcome":"Resolved"}`),
		// Older batch: the first span for the timeline.
		driftOpenedRow(1, older, "api.example.com", `{"outcome":"Resolved"}`, ""),
	}

	groups, movement := buildDriftFeed(rows, now)

	if len(groups) != 2 {
		t.Fatalf("want 2 batch groups, got %d: %+v", len(groups), groups)
	}
	// Newest batch first: the changed transition, with a two-line before/after diff.
	changed := groups[0].Events
	if len(changed) != 1 || changed[0].Change != "changed" {
		t.Fatalf("group 0 want one 'changed' event, got %+v", changed)
	}
	if len(changed[0].Diff) != 2 || changed[0].Diff[0].Type != "remove" || changed[0].Diff[0].Text != "Resolved" ||
		changed[0].Diff[1].Type != "add" || changed[0].Diff[1].Text != "NXDOMAIN" {
		t.Errorf("changed diff = %+v, want remove Resolved / add NXDOMAIN", changed[0].Diff)
	}
	// Older batch: the appeared event, no diff.
	appeared := groups[1].Events
	if len(appeared) != 1 || appeared[0].Change != "appeared" {
		t.Fatalf("group 1 want one 'appeared' event, got %+v", appeared)
	}
	if len(appeared[0].Diff) != 0 {
		t.Errorf("appeared event should carry no diff, got %+v", appeared[0].Diff)
	}
	if movement["appeared"] != 1 || movement["changed"] != 1 {
		t.Errorf("movement = %+v, want appeared:1 changed:1", movement)
	}
	if movement["withdrawn"] != 0 || movement["descoped"] != 0 || movement["returned"] != 0 {
		t.Errorf("dormant kinds should tally zero, got %+v", movement)
	}
}

// A transition across a Break — the predecessor sits under a different Derivation
// vector — is a version bump, not a value move (ADR-0008), so it is not narrated as
// `changed`; nothing compares across a Break.
func TestBuildDriftFeedSkipsBreakCrossing(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	row := driftOpenedRow(2, now.Add(-time.Hour), "api.example.com", `{"outcome":"Resolved"}`, `{"outcome":"Resolved"}`)
	row.Derivation = []byte(`[{"leaf":"resolution-walk","version":"2"}]`)      // moved
	row.PrevDerivation = []byte(`[{"leaf":"resolution-walk","version":"1"}]`) // from

	groups, movement := buildDriftFeed([]db.ListRecentDriftEventsRow{row}, now)
	if len(groups) != 0 {
		t.Errorf("a Break-crossing opening must not be narrated; got groups %+v", groups)
	}
	if movement["changed"] != 0 {
		t.Errorf("a Break must not count as changed; movement %+v", movement)
	}
}

// classifyDriftEvent reads a reasoned close as withdrawn (measured-absent / uncited)
// or descoped, carrying the human reason — the dormant exit kinds, classified for when
// withdrawal persistence is wired.
func TestClassifyDriftEventReasonedClose(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for reason, wantKind := range map[string]string{
		"measured-absent": "withdrawn",
		"uncited":         "withdrawn",
		"descoped":        "descoped",
	} {
		row := db.ListRecentDriftEventsRow{
			Role: "closed", BatchID: 5, BatchKind: "hot",
			BatchAt:       pgtype.Timestamptz{Time: now, Valid: true},
			SubjectKind:   "name", SubjectKey: "api.example.com", Facet: "resolution",
			Value:         []byte(`{"outcome":"Resolved"}`),
			ClosedAt:      pgtype.Timestamptz{Time: now.Add(-time.Hour), Valid: true},
			ClosureReason: pgtype.Text{String: reason, Valid: true},
		}
		ev, ok := classifyDriftEvent(row, now)
		if !ok || ev.Change != wantKind {
			t.Errorf("reason %q => change %q (ok=%v), want %q", reason, ev.Change, ok, wantKind)
		}
		if ev.Reason == "" {
			t.Errorf("reason %q => empty reason label", reason)
		}
	}
}

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

// With two batches folding a value that moved for the same subject, the timeline
// leaves the empty-state and renders the batch-grouped transitions: an `appeared`
// event for the first batch and a `changed` event with a before/after diff for the
// second (#288, ADR-0111). The movement summary counts the kinds, and the export
// button lights up.
func TestDriftFeedRendersTransitions(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// The value moved between two batches, one day apart — both within the default 7d.
	t1 := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	f.addResolution(t, admin.ID, "api.example.com", "hot", t1, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "api.example.com", "hot", t2, `{"outcome":"NXDOMAIN"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	// The empty-state is gone once a real transition exists.
	if strings.Contains(page, "No change to show yet") {
		t.Errorf("drift timeline still shows the empty-state with a real transition; body: %s", page)
	}
	// Both transitions render, on the subject, with the changed diff's before/after.
	for _, want := range []string{
		"api.example.com",
		`chip gain`,   // appeared rides the gain family
		`chip change`, // changed rides the change family
		"Resolved",    // the diff's before value
		"NXDOMAIN",    // the diff's after value
	} {
		if !strings.Contains(page, want) {
			t.Errorf("drift feed missing %q; body: %s", want, page)
		}
	}
	// The export button is a live link now that there is something to export.
	if !strings.Contains(page, `href="/drift/export?period=7d"`) {
		t.Errorf("drift page did not enable the CSV export link; body: %s", page)
	}
}

// The period selector bounds the feed: a change older than the default 7d window is
// excluded from the default view (the timeline falls back to the empty-state) and
// included under ?period=all.
func TestDriftFeedPeriodFilter(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// An appearance well over 7d before the fixed clock (2026-08-15).
	old := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	f.addResolution(t, admin.ID, "api.example.com", "hot", old, `{"outcome":"Resolved"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Default 7d: the old appearance is out of window, so the timeline is empty.
	def := getBody(t, ac, base+"/drift", http.StatusOK)
	if !strings.Contains(def, "No change to show yet") {
		t.Errorf("default 7d window should exclude a >7d-old transition; body: %s", def)
	}
	// All time: the appearance is in window and renders.
	all := getBody(t, ac, base+"/drift?period=all", http.StatusOK)
	if strings.Contains(all, "No change to show yet") {
		t.Errorf("all-time window should include the transition; body: %s", all)
	}
	if !strings.Contains(all, "api.example.com") {
		t.Errorf("all-time feed missing the transition's subject; body: %s", all)
	}
}

// GET /drift/export streams a text/csv attachment of the transition feed for the
// active period: a header row plus one row per transition, times absolute.
func TestDriftExportCSV(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	t1 := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	f.addResolution(t, admin.ID, "api.example.com", "hot", t1, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "api.example.com", "hot", t2, `{"outcome":"NXDOMAIN"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/drift/export?period=all")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drift export status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("drift export Content-Type = %q, want text/csv", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") || !strings.Contains(cd, ".csv") {
		t.Errorf("drift export Content-Disposition = %q, want an attachment .csv filename", cd)
	}
	// Header row, an appeared row, and the changed row with its before/after values.
	for _, want := range []string{
		"batch,scope,change,subject,detail,time,reason,before,after",
		",appeared,api.example.com,",
		",changed,api.example.com,",
		"Resolved",
		"NXDOMAIN",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("drift export CSV missing %q; body:\n%s", want, got)
		}
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
