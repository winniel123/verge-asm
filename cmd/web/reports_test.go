package main

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// reportsClock is the fixed server instant the render tests read against
// (2026-08-15 12:00, matching fixedClock). A Dispatch dated inside the twelve-week
// window before it lands in the scans-per-day heatmap.
var reportsClock = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// The Reports screen is behind requireLogin — an anonymous GET is bounced to the
// sign-in page, never served the analytics.
func TestReportsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t) // does not follow redirects
	resp, err := c.Get(base + "/reports")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anonymous GET /reports status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("anonymous GET /reports location = %q, want /login", loc)
	}
}

// A signed-in operator gets the full composition: the KPI band with the real
// activity counts, the scans-per-day heatmap wired from Dispatch history, and the
// schedule wizard preview — the three structural elements the ticket checklists
// against 07-console.jpg (time-series card, heatmap, wizard).
func TestReportsRendersActivityAndComposition(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// Two Dispatches inside the window; one still has jobs running (in flight).
	day := reportsClock.AddDate(0, 0, -1)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1, "hot", day, 3, 1, 1, 1, 0, 0), // ready+running > 0 -> active
		progressRow(2, "dns", day, 2, 0, 0, 2, 0, 0), // complete
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	// The nav pill and the heading.
	if !strings.Contains(page, `class="navpill active" href="/reports"`) {
		t.Errorf("reports nav pill not marked active; body: %s", page)
	}
	for _, want := range []string{
		"Open signals", "Scans run", "In flight", // KPI band
		"Open signals over time",                     // time-series card title
		"Scans per day", "Scans per day, last 12 weeks", // heatmap card + grid aria-label
		"Recurring reports", "New report schedule", // recurring card + wizard preview
	} {
		if !strings.Contains(page, want) {
			t.Errorf("reports page missing %q; body: %s", want, page)
		}
	}

	// The two activity KPIs reflect the seeded Dispatches: two in the window, one in
	// flight.
	if !strings.Contains(page, ">2<") {
		t.Errorf("scans-run KPI should be 2; body: %s", page)
	}
	if !strings.Contains(page, ">1<") {
		t.Errorf("in-flight KPI should be 1; body: %s", page)
	}

	// The heatmap is wired (not the empty-state), so its intensity fill renders.
	if !strings.Contains(page, "color-mix(in srgb, var(--chart-1)") {
		t.Errorf("heatmap intensity fill missing; body: %s", page)
	}
	if strings.Contains(page, "No scans in the last 12 weeks") {
		t.Errorf("heatmap should be wired, not empty-stated; body: %s", page)
	}

	// Domain hygiene: signals are withdrawn by the world, never resolved, and carry
	// no severity ramp — the example's resolve/severity framings must not survive.
	low := strings.ToLower(page)
	if strings.Contains(low, "mean time to resolve") || strings.Contains(low, "time to resolve") {
		t.Errorf("no resolve metric may appear — signals are withdrawn, not resolved; body: %s", page)
	}
}

// With no Dispatch history and an empty estate, every data region falls back to a
// design-system empty-state that states the fact and the next action — the
// time-series and by-severity cards state the domain fact (a census is not a
// series and carries no severity), the heatmap says nothing was dispatched, and
// the recurring table says scheduling is not yet available. No series is
// fabricated.
func TestReportsEmptyStates(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	for _, want := range []string{
		"Signals are a current-state census",   // time-series region, domain fact
		"Signals carry no severity",            // by-severity region, domain fact
		"No scans in the last 12 weeks",        // heatmap empty-state
		"No recurring reports",                 // recurring table empty-state
	} {
		if !strings.Contains(page, want) {
			t.Errorf("reports empty-state missing %q; body: %s", want, page)
		}
	}
	// The empty heatmap must not emit any intensity fill (no fabricated activity).
	if strings.Contains(page, "color-mix(in srgb, var(--chart-1)") &&
		!strings.Contains(page, "28%, var(--surface)") {
		t.Errorf("empty heatmap should not draw wired cells; body: %s", page)
	}
}

// lastReportDelivery lights the "View last delivery" menu item only where a report
// actually delivered: a non-failed delivery yields the stable /reports/delivery
// artifact route (T3), while no deliveries — or only undelivered ones — yields no
// link, so the item renders disabled rather than fabricating a document.
func TestLastReportDelivery(t *testing.T) {
	// A report with a delivered outcome opens the artifact.
	if href, has := lastReportDelivery([]deliveryView{{State: "delivered"}}); !has || href != "/reports/delivery" {
		t.Errorf("delivered report: got (%q, %v), want (/reports/delivery, true)", href, has)
	}
	// No deliveries at all — the item is disabled, no link.
	if href, has := lastReportDelivery(nil); has || href != "" {
		t.Errorf("no delivery: got (%q, %v), want (\"\", false)", href, has)
	}
	// An undelivered (failed) outcome is not a delivery to view.
	if href, has := lastReportDelivery([]deliveryView{{State: "undelivered", Failed: true}}); has || href != "" {
		t.Errorf("undelivered only: got (%q, %v), want (\"\", false)", href, has)
	}
}

// The recurring-reports row menu ports Reports.jsx's per-row DropdownMenu: its
// "View last delivery" item opens T3's /reports/delivery artifact where a report
// has delivered, and renders disabled where none has — no fabrication. The menu
// only opens or confirms; it never fires destruction directly, so its "Delete
// schedule" item is inert (no form, no POST) on click.
func TestReportScheduleRowMenu(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"Title": "Reports", "NavActive": "reports", "IsAdmin": true,
		"Account": db.Account{},
		"Schedules": []reportScheduleRow{
			{Name: "Weekly exposure summary", Cadence: "weekly · mon 09:00", Format: "pdf", LastSent: "3d", HasDelivery: true, DeliveryHref: "/reports/delivery"},
			{Name: "Monthly asset inventory", Cadence: "monthly · 1st", Format: "csv", LastSent: "—", HasDelivery: false},
		},
	}
	if err := tmpl.ExecuteTemplate(&buf, "reports", data); err != nil {
		t.Fatalf("execute reports template: %v", err)
	}
	page := buf.String()

	// The delivered report's menu item opens the artifact at the stable T3 route.
	if !strings.Contains(page, `href="/reports/delivery"`) {
		t.Errorf("delivered row menu should link to /reports/delivery; body: %s", page)
	}
	if !strings.Contains(page, "View last delivery") {
		t.Errorf("row menu missing the ported 'View last delivery' item; body: %s", page)
	}
	// The undelivered report's item is disabled — no link fabricated.
	if !strings.Contains(page, `aria-disabled="true" title="No delivery yet"`) {
		t.Errorf("undelivered row should render a disabled 'View last delivery'; body: %s", page)
	}
	// The menu never destroys directly: "Delete schedule" is inert, not a POST form.
	if !strings.Contains(page, "Delete schedule") {
		t.Errorf("row menu missing the ported 'Delete schedule' item; body: %s", page)
	}
	// The menu never destroys directly — the delete item is a disabled span, carrying
	// no destructive form or POST action of its own. The wizard's own create form posts
	// to /reports/schedule (that is expected); what must NOT exist is a per-row mutation
	// route (delete / run-now / edit) firing off a menu click.
	if !strings.Contains(page, `aria-disabled="true" title="Report scheduling is not wired yet"`) {
		t.Errorf("row menu 'Delete schedule' should render as an inert disabled span; body: %s", page)
	}
	for _, dead := range []string{
		`action="/reports/schedule/delete"`,
		`action="/reports/schedule/run"`,
		`action="/reports/schedule/edit"`,
	} {
		if strings.Contains(page, dead) {
			t.Errorf("row menu must not carry a destructive per-row action (%s); body: %s", dead, page)
		}
	}

	// The rows themselves render — the table, not the empty-state.
	if strings.Contains(page, "No recurring reports") {
		t.Errorf("with schedules present the table should render, not the empty-state; body: %s", page)
	}
	if !strings.Contains(page, "Weekly exposure summary") || !strings.Contains(page, "Monthly asset inventory") {
		t.Errorf("schedule rows missing; body: %s", page)
	}
}

// Posting the schedule wizard persists one report_schedule and it then lists in the
// "Recurring reports" table — the empty-state gives way to a real row carrying the
// declared name, cadence and format. The chosen sections are stored as a canonical
// JSON array (only the checked, admitted keys, in wizard order), and the schedule is
// attributed to the admin who declared it.
func TestReportScheduleWizardPersistsAndLists(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A distinctive name — not the wizard's own placeholder ("Weekly exposure
	// summary"), which is always in the admin page and would give a false positive.
	resp := postForm(t, ac, base+"/reports/schedule", url.Values{
		"name":     {"Q3 exposure digest"},
		"sections": {"summary-kpis", "signal-changes"},
		"cadence":  {"weekly"},
		"format":   {"pdf"},
		"target":   {"ops@example.com"},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports" {
		t.Fatalf("wizard POST: status=%d location=%q, want 303 -> /reports (body: %s)",
			resp.StatusCode, resp.Header.Get("Location"), body(t, resp))
	}
	resp.Body.Close()

	// Exactly one schedule was filed, with the declared facts and the admin's id.
	if len(f.reportSchedules) != 1 {
		t.Fatalf("reportSchedules = %d, want 1", len(f.reportSchedules))
	}
	got := f.reportSchedules[0]
	if got.Name != "Q3 exposure digest" || got.Cadence != "weekly" || got.Format != "pdf" {
		t.Errorf("stored schedule = %+v, want name/cadence/format Q3.../weekly/pdf", got)
	}
	if got.DeliveryTarget != "ops@example.com" {
		t.Errorf("delivery target = %q, want ops@example.com", got.DeliveryTarget)
	}
	if got.CreatedBy != admin.ID {
		t.Errorf("created_by = %d, want %d (the declaring admin)", got.CreatedBy, admin.ID)
	}
	// Sections are stored as a canonical JSON array — only the two checked keys, in
	// wizard order (coverage-gaps was unchecked, so it is absent; no unknowns).
	if string(got.Sections) != `["summary-kpis","signal-changes"]` {
		t.Errorf("stored sections = %s, want [\"summary-kpis\",\"signal-changes\"]", got.Sections)
	}

	// The table now renders the row, not the empty-state.
	page := getBody(t, ac, base+"/reports", http.StatusOK)
	if strings.Contains(page, "No recurring reports") {
		t.Errorf("recurring table should list the schedule, not empty-state; body: %s", page)
	}
	for _, want := range []string{"Q3 exposure digest", ">weekly<"} {
		if !strings.Contains(page, want) {
			t.Errorf("reports table missing %q after persist; body: %s", want, page)
		}
	}
}

// An unnamed submission files nothing — a schedule needs a name (the input also
// carries `required`). The handler returns to /reports rather than persisting an
// unnamed row, and unknown section keys or bad cadence/format values never reach
// the store: they are dropped or defaulted at the closed sets.
func TestReportScheduleWizardValidates(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Empty name: nothing filed, redirect back.
	resp := postForm(t, ac, base+"/reports/schedule", url.Values{"name": {"   "}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("empty-name POST: status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 0 {
		t.Fatalf("empty name filed a schedule: %d", len(f.reportSchedules))
	}

	// Unknown section and bogus cadence/format are sanitised at the closed sets.
	resp = postForm(t, ac, base+"/reports/schedule", url.Values{
		"name":     {"Sane name"},
		"sections": {"summary-kpis", "not-a-section"},
		"cadence":  {"hourly-ish"},
		"format":   {"exe"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("sanitise POST: status=%d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 1 {
		t.Fatalf("reportSchedules = %d, want 1", len(f.reportSchedules))
	}
	got := f.reportSchedules[0]
	if string(got.Sections) != `["summary-kpis"]` {
		t.Errorf("unknown section leaked: sections = %s, want [\"summary-kpis\"]", got.Sections)
	}
	if got.Cadence != defaultReportCadence || got.Format != defaultReportFormat {
		t.Errorf("bogus cadence/format not defaulted: %q/%q, want %q/%q",
			got.Cadence, got.Format, defaultReportCadence, defaultReportFormat)
	}
}

// Declaring a schedule is an admin act: a viewer's POST is refused with 403 and
// files nothing, and a viewer never sees the wizard form (it renders admin-only).
func TestReportScheduleRequiresAdmin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/reports/schedule", url.Values{
		"name": {"Sneaky schedule"}, "cadence": {"weekly"}, "format": {"pdf"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer wizard POST: status=%d, want 403", resp.StatusCode)
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 0 {
		t.Fatalf("viewer's denied act still filed a schedule: %d", len(f.reportSchedules))
	}

	// The viewer's Reports page carries no create form (the wizard is admin-only).
	page := getBody(t, vc, base+"/reports", http.StatusOK)
	if strings.Contains(page, `action="/reports/schedule"`) {
		t.Errorf("wizard create form shown to a viewer; body: %s", page)
	}
}

// foldScanActivity buckets Dispatches by whole-day offset from today, oldest-first,
// and derives the window and in-flight counts. Out-of-window and undated rows drop
// from the grid; a running Dispatch counts as in flight regardless of its date.
func TestFoldScanActivity(t *testing.T) {
	s := &server{now: func() time.Time { return reportsClock }}
	rows := []db.ListDispatchProgressRow{
		progressRow(1, "hot", reportsClock, 2, 0, 1, 1, 0, 0),                 // today, in flight
		progressRow(2, "dns", reportsClock.AddDate(0, 0, -1), 2, 0, 0, 2, 0, 0), // yesterday, complete
		progressRow(3, "dns", reportsClock.AddDate(0, 0, -200), 1, 0, 0, 1, 0, 0), // out of window
	}
	// An undated row still contributes to the in-flight count but not the grid.
	rows = append(rows, db.ListDispatchProgressRow{DispatchID: 4, Ready: 1})

	cells, total, window, active := s.foldScanActivity(rows)
	if len(cells) != reportsHeatDays {
		t.Fatalf("cells = %d, want %d", len(cells), reportsHeatDays)
	}
	if total != 2 {
		t.Errorf("total in-window scans = %d, want 2 (the 200-day-old row drops)", total)
	}
	if window != 2 {
		t.Errorf("window count = %d, want 2", window)
	}
	if active != 2 {
		t.Errorf("active = %d, want 2 (the running today row + the undated ready row)", active)
	}
	// Today is the last cell and yesterday the one before; both were dispatched to,
	// so both carry a wired (non-sunken) fill.
	if string(cells[reportsHeatDays-1].Bg) == "var(--sunken)" {
		t.Errorf("today's cell should be wired, got sunken")
	}
	if string(cells[reportsHeatDays-2].Bg) == "var(--sunken)" {
		t.Errorf("yesterday's cell should be wired, got sunken")
	}
	// A day with no dispatch stays at the sunken step.
	if string(cells[0].Bg) != "var(--sunken)" {
		t.Errorf("the oldest, empty day should be sunken, got %q", cells[0].Bg)
	}
}
