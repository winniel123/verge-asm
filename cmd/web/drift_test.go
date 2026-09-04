package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func driftOpenedRow(batchID int64, at time.Time, subject, value, prevValue string) db.ListRecentDriftEventsRow {
	const vec = `[{"leaf":"resolution-walk","version":"1"}]`
	row := db.ListRecentDriftEventsRow{
		Role: "opened", BatchID: batchID, BatchKind: "hot",
		BatchAt:       pgtype.Timestamptz{Time: at, Valid: true},
		RecordedScope: []byte(`{}`),
		SubjectKind:   "name", SubjectKey: subject, Facet: "resolution",
		Value: []byte(value), Derivation: []byte(vec),
		OpenedAt: pgtype.Timestamptz{Time: at, Valid: true},
	}
	if prevValue != "" {
		row.PrevValue = []byte(prevValue)
		row.PrevDerivation = []byte(vec)
	}
	return row
}

func TestBuildDriftFeedClassifiesAndGroups(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	newer := now.Add(-1 * time.Hour)
	older := now.Add(-24 * time.Hour)

	rows := []db.ListRecentDriftEventsRow{
		driftOpenedRow(2, newer, "api.example.com", `{"outcome":"NXDOMAIN"}`, `{"outcome":"Resolved"}`),
		driftOpenedRow(1, older, "api.example.com", `{"outcome":"Resolved"}`, ""),
	}

	groups, movement := buildDriftFeed(rows, now)

	if len(groups) != 2 {
		t.Fatalf("want 2 batch groups, got %d: %+v", len(groups), groups)
	}
	changed := groups[0].Events
	if len(changed) != 1 || changed[0].Change != "changed" {
		t.Fatalf("group 0 want one 'changed' event, got %+v", changed)
	}
	if len(changed[0].Diff) != 2 || changed[0].Diff[0].Type != "remove" || changed[0].Diff[0].Text != "Resolved" ||
		changed[0].Diff[1].Type != "add" || changed[0].Diff[1].Text != "NXDOMAIN" {
		t.Errorf("changed diff = %+v, want remove Resolved / add NXDOMAIN", changed[0].Diff)
	}
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

func TestBuildDriftFeedSkipsBreakCrossing(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	row := driftOpenedRow(2, now.Add(-time.Hour), "api.example.com", `{"outcome":"Resolved"}`, `{"outcome":"Resolved"}`)
	row.Derivation = []byte(`[{"leaf":"resolution-walk","version":"2"}]`)
	row.PrevDerivation = []byte(`[{"leaf":"resolution-walk","version":"1"}]`)

	groups, movement := buildDriftFeed([]db.ListRecentDriftEventsRow{row}, now)
	if len(groups) != 0 {
		t.Errorf("a Break-crossing opening must not be narrated; got groups %+v", groups)
	}
	if movement["changed"] != 0 {
		t.Errorf("a Break must not count as changed; movement %+v", movement)
	}
}

func TestClassifyDriftEventReasonedClose(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	for reason, wantKind := range map[string]string{
		"measured-absent": "withdrawn",
		"uncited":         "withdrawn",
		"descoped":        "descoped",
	} {
		row := db.ListRecentDriftEventsRow{
			Role: "closed", BatchID: 5, BatchKind: "hot",
			BatchAt:     pgtype.Timestamptz{Time: now, Valid: true},
			SubjectKind: "name", SubjectKey: "api.example.com", Facet: "resolution",
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

func TestClassifyDriftEventRevealed(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	revealed := driftOpenedRow(7, now.Add(-time.Hour), "api.example.com", `{"outcome":"not-reached"}`, "")
	revealed.OpenedAperture = true
	ev, ok := classifyDriftEvent(revealed, now)
	if !ok || ev.Change != "revealed" {
		t.Fatalf("aperture-marked first span => change %q (ok=%v), want revealed", ev.Change, ok)
	}
	if ev.Family != "gain" {
		t.Errorf("revealed family = %q, want gain", ev.Family)
	}

	appeared := driftOpenedRow(7, now.Add(-time.Hour), "discovered.other.net", `{"outcome":"Resolved"}`, "")
	ev, ok = classifyDriftEvent(appeared, now)
	if !ok || ev.Change != "appeared" {
		t.Fatalf("unmarked first span => change %q (ok=%v), want appeared", ev.Change, ok)
	}
}

func TestClassifyDriftEventReturned(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	returned := driftOpenedRow(9, now.Add(-time.Hour), "api.example.com", `{"outcome":"Resolved"}`, `{"outcome":"NameError"}`)
	returned.PrevClosureReason = pgtype.Text{String: "measured-absent", Valid: true}
	ev, ok := classifyDriftEvent(returned, now)
	if !ok || ev.Change != "returned" {
		t.Fatalf("re-open across a measured-absent closure => change %q (ok=%v), want returned", ev.Change, ok)
	}

	afterDescope := driftOpenedRow(9, now.Add(-time.Hour), "api.example.com", `{"outcome":"Resolved"}`, `{"outcome":"NameError"}`)
	afterDescope.PrevClosureReason = pgtype.Text{String: "descoped", Valid: true}
	ev, ok = classifyDriftEvent(afterDescope, now)
	if !ok || ev.Change != "appeared" {
		t.Fatalf("re-open across a descoped closure => change %q (ok=%v), want appeared", ev.Change, ok)
	}
}

func TestDriftPageRendersVocabularyAndEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	for _, want := range []string{"Drift", "Change is its own language", "By batch", "Movement"} {
		if !strings.Contains(page, want) {
			t.Errorf("drift page missing %q; body: %s", want, page)
		}
	}

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
	for _, pill := range []string{`class="sev sev-critical"`, `class="sev sev-high"`} {
		if strings.Contains(page, pill) {
			t.Errorf("drift page rendered a severity pill %q — change must ride the drift palette only", pill)
		}
	}

	if !strings.Contains(page, "dr-empty") {
		t.Errorf("drift page missing empty-state block; body: %s", page)
	}
	if !strings.Contains(page, "No change to show yet") {
		t.Errorf("drift page empty-state missing its fact; body: %s", page)
	}
	if !strings.Contains(page, "run twice") {
		t.Errorf("drift page empty-state missing its next action; body: %s", page)
	}

	if !strings.Contains(page, `href="/drift"`) || !strings.Contains(page, `sh-pill on`) {
		t.Errorf("drift page did not mark the Drift nav pill active; body: %s", page)
	}
}

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

func TestDriftBatchDetailLinksToRun(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

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
	if !strings.Contains(page, `href="/runs/88"`) {
		t.Errorf("Batch detail should link to the most recent batch /runs/88; body: %s", page)
	}
	if strings.Contains(page, `href="/runs/87"`) {
		t.Errorf("Batch detail linked an older batch /runs/87, not the latest; body: %s", page)
	}
	if !strings.Contains(page, "No change to show yet") {
		t.Errorf("Batch detail must not fabricate a transition feed; body: %s", page)
	}
}

func TestDriftBatchDetailOmittedWithoutBatch(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	if strings.Contains(page, "Batch detail") || strings.Contains(page, `href="/runs/`) {
		t.Errorf("drift page offered a Batch detail entry with no batch dispatched; body: %s", page)
	}
}

func TestDriftFeedRendersTransitions(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	t1 := time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)
	f.addResolution(t, admin.ID, "api.example.com", "hot", t1, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "api.example.com", "hot", t2, `{"outcome":"NXDOMAIN"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/drift", http.StatusOK)

	if strings.Contains(page, "No change to show yet") {
		t.Errorf("drift timeline still shows the empty-state with a real transition; body: %s", page)
	}
	for _, want := range []string{
		"api.example.com",
		`chip gain`,
		`chip change`,
		"Resolved",
		"NXDOMAIN",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("drift feed missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, `href="/drift/export?period=7d"`) {
		t.Errorf("drift page did not enable the CSV export link; body: %s", page)
	}
}

func TestDriftFeedPeriodFilter(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	old := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	f.addResolution(t, admin.ID, "api.example.com", "hot", old, `{"outcome":"Resolved"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	def := getBody(t, ac, base+"/drift", http.StatusOK)
	if !strings.Contains(def, "No change to show yet") {
		t.Errorf("default 7d window should exclude a >7d-old transition; body: %s", def)
	}
	wide := getBody(t, ac, base+"/drift?period=90d", http.StatusOK)
	if strings.Contains(wide, "No change to show yet") {
		t.Errorf("90d window should include the <90d-old transition; body: %s", wide)
	}
	if !strings.Contains(wide, "api.example.com") {
		t.Errorf("90d feed missing the transition's subject; body: %s", wide)
	}
}

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

	resp, err := ac.Get(base + "/drift/export?period=90d")
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

func TestDriftTransitionDelta(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	prevStart := now.Add(-14 * 24 * time.Hour)
	prevAt := now.Add(-10 * 24 * time.Hour)
	oldEnough := pgtype.Timestamptz{Time: now.Add(-30 * 24 * time.Hour), Valid: true}

	prevRows := []db.ListRecentDriftEventsRow{
		driftOpenedRow(1, prevAt, "a.example.com", `{"outcome":"Resolved"}`, ""),
		driftOpenedRow(1, prevAt, "b.example.com", `{"outcome":"Resolved"}`, ""),
		driftOpenedRow(1, prevAt, "c.example.com", `{"outcome":"Resolved"}`, ""),
	}

	if got := driftTransitionDelta(5, prevRows, oldEnough, prevStart, now); got != "+2" {
		t.Errorf("signed delta = %q, want %q", got, "+2")
	}
	// The want string is U+2212, not an ASCII hyphen — signedCount renders the true minus.
	if got := driftTransitionDelta(1, prevRows, oldEnough, prevStart, now); got != "−2" {
		t.Errorf("negative delta = %q, want %q", got, "−2")
	}
	if got := driftTransitionDelta(3, prevRows, oldEnough, prevStart, now); got != "0" {
		t.Errorf("zero delta = %q, want %q", got, "0")
	}
	if got := driftTransitionDelta(5, prevRows, pgtype.Timestamptz{}, prevStart, now); got != "" {
		t.Errorf("no-batch delta = %q, want empty", got)
	}
	tooYoung := pgtype.Timestamptz{Time: now.Add(-9 * 24 * time.Hour), Valid: true}
	if got := driftTransitionDelta(5, prevRows, tooYoung, prevStart, now); got != "" {
		t.Errorf("young-install delta = %q, want empty", got)
	}
}

func TestClassifyDriftEventRevealedAfterDescopedReEntry(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	widened := driftOpenedRow(11, now.Add(-time.Hour), "203.0.113.77", `{"outcome":"Resolved"}`, `{"outcome":"Resolved"}`)
	widened.SubjectKind = "address"
	widened.PrevClosureReason = pgtype.Text{String: "descoped", Valid: true}
	widened.OpenedAperture = true
	ev, ok := classifyDriftEvent(widened, now)
	if !ok || ev.Change != "revealed" {
		t.Fatalf("aperture-marked re-entry across a descoped closure => change %q (ok=%v), want revealed", ev.Change, ok)
	}
	if ev.Family != "gain" {
		t.Errorf("revealed family = %q, want gain", ev.Family)
	}

	recommissioned := driftOpenedRow(11, now.Add(-time.Hour), "203.0.113.78", `{"outcome":"Resolved"}`, `{"outcome":"NameError"}`)
	recommissioned.SubjectKind = "address"
	recommissioned.PrevClosureReason = pgtype.Text{String: "measured-absent", Valid: true}
	recommissioned.OpenedAperture = true
	ev, ok = classifyDriftEvent(recommissioned, now)
	if !ok || ev.Change != "returned" {
		t.Fatalf("aperture-marked re-entry across a measured-absent closure => change %q (ok=%v), want returned", ev.Change, ok)
	}
}
