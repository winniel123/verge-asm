package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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
		"Recurring reports", "New schedule", // recurring card + the (disabled) schedule control
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

// Report scheduling has no dispatch or delivery backend (#344), so the create path
// must not persist a report_schedule row that would silently never run. An admin's
// POST — the one role that could reach the handler — is refused (501) and files
// nothing, so a hand-crafted request cannot re-introduce the inert row the wizard used
// to leak.
func TestReportScheduleCreateRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/reports/schedule", url.Values{
		"name":     {"Q3 exposure digest"},
		"sections": {"summary-kpis", "signal-changes"},
		"cadence":  {"weekly"},
		"format":   {"pdf"},
		"target":   {"ops@example.com"},
	})
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("admin schedule POST: status=%d, want 501 (scheduling unavailable); body: %s",
			resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 0 {
		t.Fatalf("a refused create still filed a schedule: %d", len(f.reportSchedules))
	}
}

// A viewer's POST is refused before the handler by requireAdmin (403), still filing
// nothing — the create path is defended at both the auth wrapper and the handler.
func TestReportScheduleCreateRefusesViewer(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/reports/schedule", url.Values{
		"name": {"Sneaky schedule"}, "cadence": {"weekly"}, "format": {"pdf"},
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer schedule POST: status=%d, want 403", resp.StatusCode)
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 0 {
		t.Fatalf("viewer's denied act still filed a schedule: %d", len(f.reportSchedules))
	}
}

// The Reports surface no longer offers an enabled control that can create a schedule:
// the "New schedule" wizard is gone and its entry point renders disabled, alongside
// its already-disabled sibling controls, presenting scheduling consistently as not
// available yet (#344).
func TestReportScheduleWizardDisabled(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	// No live create form — not even for an admin, who used to see the wizard.
	if strings.Contains(page, `action="/reports/schedule"`) {
		t.Errorf("Reports still renders a live schedule-create form; body: %s", page)
	}
	// The New schedule control is present but disabled, reading as not-yet-available.
	if !strings.Contains(page, "New schedule") {
		t.Errorf("Reports missing the New schedule control; body: %s", page)
	}
	if !strings.Contains(page, "Report scheduling is not available yet") {
		t.Errorf("New schedule control not marked unavailable; body: %s", page)
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

	cells, total, window, active := s.foldScanActivity(rows, reportsHeatDays)
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

// resolveReportsWeeks parses ?weeks= and clamps to the offered set, defaulting to
// twelve when the param is absent, unparseable, or not one of the offered spans.
func TestResolveReportsWeeks(t *testing.T) {
	cases := []struct {
		query string
		want  int
	}{
		{"", reportsHeatWeeks},           // absent -> default
		{"weeks=26", 26},                 // offered
		{"weeks=4", 4},                   // offered
		{"weeks=52", 52},                 // offered
		{"weeks=12", 12},                 // offered (the default, explicitly)
		{"weeks=9", reportsHeatWeeks},    // not offered -> default
		{"weeks=0", reportsHeatWeeks},    // not offered -> default
		{"weeks=-8", reportsHeatWeeks},   // not offered -> default
		{"weeks=abc", reportsHeatWeeks},  // unparseable -> default
		{"weeks=99999", reportsHeatWeeks}, // out of set -> default
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/reports?"+c.query, nil)
		if got := resolveReportsWeeks(r); got != c.want {
			t.Errorf("resolveReportsWeeks(%q) = %d, want %d", c.query, got, c.want)
		}
	}
}

// The range param changes the fold window: a Dispatch dated 100 days before today is
// outside the twelve-week (84-day) default window but inside a 26-week (182-day) one,
// so the window KPI counts it only for the wider range. The heatmap grid also grows
// to match the selected span.
func TestFoldScanActivityRangeWindow(t *testing.T) {
	s := &server{now: func() time.Time { return reportsClock }}
	rows := []db.ListDispatchProgressRow{
		progressRow(1, "dns", reportsClock.AddDate(0, 0, -100), 1, 0, 0, 1, 0, 0), // 100 days old
	}

	// Twelve weeks (84 days): the 100-day-old row is out of window.
	cells12, _, window12, _ := s.foldScanActivity(rows, reportsHeatWeeks*7)
	if len(cells12) != reportsHeatWeeks*7 {
		t.Fatalf("12wk cells = %d, want %d", len(cells12), reportsHeatWeeks*7)
	}
	if window12 != 0 {
		t.Errorf("12wk window = %d, want 0 (100-day-old row is out of the 84-day window)", window12)
	}

	// Twenty-six weeks (182 days): the same row now falls inside the window.
	cells26, _, window26, _ := s.foldScanActivity(rows, 26*7)
	if len(cells26) != 26*7 {
		t.Fatalf("26wk cells = %d, want %d", len(cells26), 26*7)
	}
	if window26 != 1 {
		t.Errorf("26wk window = %d, want 1 (100-day-old row is inside the 182-day window)", window26)
	}
}

// The header range control renders the offered spans, marks the active one selected,
// re-skins the range-aware captions, and carries the active span into the export
// links. The default (no param) stays exactly "last 12 weeks".
func TestReportsRangeControlRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Default: twelve weeks, and the export links carry weeks=12.
	def := getBody(t, ac, base+"/reports", http.StatusOK)
	if !strings.Contains(def, `<option value="12" selected>last 12 weeks</option>`) {
		t.Errorf("default range control should mark 12 weeks selected; body: %s", def)
	}
	if !strings.Contains(def, "/reports/export?format=csv&amp;weeks=12") {
		t.Errorf("default export link should carry weeks=12; body: %s", def)
	}

	// A selected range re-skins the caption, the heatmap aria-label, and the export
	// links to the chosen span.
	wide := getBody(t, ac, base+"/reports?weeks=26", http.StatusOK)
	if !strings.Contains(wide, `<option value="26" selected>last 26 weeks</option>`) {
		t.Errorf("range=26 should mark 26 weeks selected; body: %s", wide)
	}
	if !strings.Contains(wide, "last 26 weeks") {
		t.Errorf("range=26 should re-skin captions to 'last 26 weeks'; body: %s", wide)
	}
	if !strings.Contains(wide, "/reports/export?format=json&amp;weeks=26") {
		t.Errorf("range=26 JSON export link should carry weeks=26; body: %s", wide)
	}
}

// GET /reports/export?format=csv streams a text/csv attachment with the summary band
// and one scans_per_day row per day of the active range.
func TestReportsExportCSV(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	day := reportsClock.AddDate(0, 0, -1)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1, "hot", day, 3, 1, 1, 1, 0, 0), // in flight
		progressRow(2, "dns", day, 2, 0, 0, 2, 0, 0), // complete
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/reports/export?format=csv&weeks=12")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export csv status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("export csv Content-Type = %q, want text/csv", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") || !strings.Contains(cd, ".csv") {
		t.Errorf("export csv Content-Disposition = %q, want an attachment .csv filename", cd)
	}
	// Header row + the summary band, with the two-in-window scans-run figure.
	for _, want := range []string{"section,label,value", "summary,scans_run,2", "summary,in_flight,1", "scans_per_day,"} {
		if !strings.Contains(got, want) {
			t.Errorf("export csv missing %q; body:\n%s", want, got)
		}
	}
	// One scans_per_day row per day of the twelve-week (84-day) window.
	if n := strings.Count(got, "scans_per_day,"); n != reportsHeatWeeks*7 {
		t.Errorf("export csv scans_per_day rows = %d, want %d", n, reportsHeatWeeks*7)
	}
}

// GET /reports/export?format=json streams an application/json attachment whose range,
// KPI band, and per-day series reflect the active range.
func TestReportsExportJSON(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	day := reportsClock.AddDate(0, 0, -1)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1, "hot", day, 3, 1, 1, 1, 0, 0), // in flight
		progressRow(2, "dns", day, 2, 0, 0, 2, 0, 0), // complete
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/reports/export?format=json&weeks=26")
	if err != nil {
		t.Fatal(err)
	}
	raw := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export json status = %d, want 200 (body: %s)", resp.StatusCode, raw)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("export json Content-Type = %q, want application/json", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") || !strings.Contains(cd, ".json") {
		t.Errorf("export json Content-Disposition = %q, want an attachment .json filename", cd)
	}

	var doc reportsExportDoc
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("export json does not parse: %v\nbody: %s", err, raw)
	}
	if doc.Range.Weeks != 26 || doc.Range.Days != 26*7 {
		t.Errorf("range = %dw/%dd, want 26w/%dd", doc.Range.Weeks, doc.Range.Days, 26*7)
	}
	if len(doc.ScansPerDay) != 26*7 {
		t.Errorf("scans_per_day len = %d, want %d", len(doc.ScansPerDay), 26*7)
	}
	if doc.KPIs.ScansRun == nil || *doc.KPIs.ScansRun != 2 {
		t.Errorf("kpis.scans_run = %v, want 2", doc.KPIs.ScansRun)
	}
	if doc.KPIs.InFlight == nil || *doc.KPIs.InFlight != 1 {
		t.Errorf("kpis.in_flight = %v, want 1", doc.KPIs.InFlight)
	}
	// open_signals is present (empty estate -> a real zero census, not null).
	if doc.KPIs.OpenSignals == nil {
		t.Errorf("kpis.open_signals should be a real count, got null")
	}
	// The last day of the series is today.
	if last := doc.ScansPerDay[len(doc.ScansPerDay)-1].Date; last != reportsClock.Format("2006-01-02") {
		t.Errorf("last series day = %q, want today %q", last, reportsClock.Format("2006-01-02"))
	}
}

// An unrecognised export format is a 400 — the handler serves only csv and json.
func TestReportsExportBadFormat(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/reports/export?format=xml")
	if err != nil {
		t.Fatal(err)
	}
	body(t, resp)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("export bad format status = %d, want 400", resp.StatusCode)
	}
}

// The export is behind requireLogin — an anonymous GET is bounced to sign-in, never
// served the figures.
func TestReportsExportRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t) // does not follow redirects
	resp, err := c.Get(base + "/reports/export?format=csv")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anonymous export status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("anonymous export location = %q, want /login", loc)
	}
}

// The trend datum (P0.3, #444) is built and threaded through reportsPage: with a
// per-instance ledger and a withdrawal history seeded, the page still renders the
// full 200 composition. The series themselves are painted by P2.4, so this guards
// the web-layer glue — the severity lookup that splits the critical+high series and
// the pgtype handling of the withdrawal instants — against a real, populated read
// rather than only the empty path the other tests exercise. The derivations proper
// are unit-tested in internal/drift/trend_test.go.
func TestReportsBuildsTrendDatumWithoutError(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// A per-instance ledger: one high-severity rule (elevated) and one calm rule,
	// first-seen inside the window, so the signals-over-time fold has real raises.
	within := reportsClock.AddDate(0, 0, -7)
	f.signalInstances = []db.SignalInstance{
		{ID: 1000, SignalName: "cname-target-name-error", SubjectKey: "a.example.com", FirstSeen: pgtype.Timestamptz{Time: within, Valid: true}},
		{ID: 1001, SignalName: "some-unknown-calm-rule", SubjectKey: "b.example.com", FirstSeen: pgtype.Timestamptz{Time: within, Valid: true}},
	}

	// A subject withdrawal in the window: appeared three weeks before it was
	// withdrawn, so the mean-time-to-withdrawal KPI has a real interval to average.
	appeared := reportsClock.AddDate(0, 0, -28)
	withdrawn := reportsClock.AddDate(0, 0, -7)
	f.withdrawalLifespans = []db.ListWithdrawalLifespansRow{
		{SubjectKind: "name", SubjectKey: "gone.example.com",
			WithdrawnAt: pgtype.Timestamptz{Time: withdrawn, Valid: true},
			FirstOpened: pgtype.Timestamptz{Time: appeared, Valid: true}},
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)
	// The page composes as before — the heatmap card is the stable structural anchor
	// the datum wiring must not have disturbed.
	if !strings.Contains(page, "Scans per day") {
		t.Fatal("Reports page should still render the scans-per-day heatmap with the trend datum wired")
	}
}
