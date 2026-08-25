package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
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
		"Open signals", "New assets discovered", "Mean time to withdrawal", // KPI band — the three trend cards
		"Open signals over time",                // time-series card title
		"Scans per day", "Scans per day, Last 7d", // heatmap card + grid aria-label (period-labelled, #23b)
		"Recurring reports", "New schedule", // recurring card + the schedule wizard control
	} {
		if !strings.Contains(page, want) {
			t.Errorf("reports page missing %q; body: %s", want, page)
		}
	}

	// The operational scans-run / in-flight scalars moved off the band (the spec's
	// band is three trend cards, not operational counters) and now live only in the
	// export — see TestReportsExportCSV/JSON. The heatmap still reflects the seeded
	// Dispatches: two dated in the window, so its intensity fill renders.
	if !strings.Contains(page, "color-mix(in srgb, var(--chart-1)") {
		t.Errorf("heatmap intensity fill missing; body: %s", page)
	}
	if strings.Contains(page, "No scans in the Last 7d") {
		t.Errorf("heatmap should be wired, not empty-stated; body: %s", page)
	}

	// Domain hygiene: signals are withdrawn by the world, never resolved, and carry
	// no severity ramp — the example's resolve/severity framings must not survive.
	low := strings.ToLower(page)
	if strings.Contains(low, "mean time to resolve") || strings.Contains(low, "time to resolve") {
		t.Errorf("no resolve metric may appear — signals are withdrawn, not resolved; body: %s", page)
	}
}

// The KPI band's second card renders the spec's "New assets discovered" datum
// (P2.4b, #468): the per-period count of Name/Service subjects that first appeared
// in the range (MIN(opened_at)), its name/service split caption, a vs-previous-period
// delta, and the daily-discovery bar series. A Name (resolution) and a Service
// (reachability) that first appear inside the window are two newly discovered assets.
func TestReportsRendersNewAssetsDiscovered(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// One Name and one Service first appearing a couple of days before the render
	// clock — both inside the default twelve-week window, none in the prior window.
	at := reportsClock.AddDate(0, 0, -2)
	f.addResolution(t, admin.ID, "api.example.com", "dns", at, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", at, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	for _, want := range []string{
		"New assets discovered",                       // card title (not the interim "Assets watched")
		"1 names",                                     // the name half of the split caption
		"1 services",                                  // the service half
		"border-bottom:1px solid var(--chart-grid)",   // the daily-discovery BarChart baseline
	} {
		if !strings.Contains(page, want) {
			t.Errorf("discovery card missing %q; body: %s", want, page)
		}
	}
	// vs-previous-period delta: two appeared this window, none before it -> +2 (the
	// signed "+" is HTML-escaped to &#43; in the rendered text node).
	if !strings.Contains(page, "&#43;2</span>") {
		t.Errorf("discovery card should show a +2 vs-previous-period delta; body: %s", page)
	}
	// The interim P0.2 "Assets watched" card must not survive alongside the spec card.
	if strings.Contains(page, "Assets watched") {
		t.Errorf("the interim \"Assets watched\" card must be replaced by \"New assets discovered\"; body: %s", page)
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
		"No signal history",    // time-series region — no raises yet
		"No signals firing",    // by-severity region — nothing firing
		"No scans in the Last 7d", // heatmap empty-state (period-labelled, #23b)
		"No recurring reports", // recurring table empty-state
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

// The report_delivery receipts store's read semantics (#291/T2): NextReportDeliveryNo
// hands out a dense 1-based per-schedule sequence, GetLatestReportDelivery returns the
// newest NON-failed run (a later failed run does not shadow an earlier delivery) and
// signals pgx.ErrNoRows where a schedule has never run, and ListReportDeliveries lists
// every run newest-first.
func TestReportDeliveryStore(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	ctx := context.Background()
	sched, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly", Cadence: "weekly", Format: "pdf", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}

	// No run yet: the first sequence number is 1, and the latest read is empty.
	if no, _ := f.NextReportDeliveryNo(ctx, sched.ID); no != 1 {
		t.Fatalf("first next-no = %d, want 1", no)
	}
	if _, err := f.GetLatestReportDelivery(ctx, sched.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("latest with no runs = %v, want pgx.ErrNoRows", err)
	}

	mk := func(no int32, state string, at time.Time) db.InsertReportDeliveryParams {
		return db.InsertReportDeliveryParams{
			ScheduleID:  sched.ID,
			PeriodStart: pgtype.Timestamptz{Time: at.AddDate(0, 0, -7), Valid: true},
			PeriodEnd:   pgtype.Timestamptz{Time: at, Valid: true},
			DeliveryNo:  no,
			State:       state,
			DeliveredAt: pgtype.Timestamptz{Time: at, Valid: state == "delivered"},
		}
	}

	// Run 1 delivered, run 2 failed. The sequence advances to 2, and the latest
	// non-failed read must still surface run 1 — a failed run is not a delivery.
	n1, _ := f.NextReportDeliveryNo(ctx, sched.ID)
	d1, err := f.InsertReportDelivery(ctx, mk(n1, "delivered", reportsClock.AddDate(0, 0, -2)))
	if err != nil {
		t.Fatalf("insert delivered run: %v", err)
	}
	n2, _ := f.NextReportDeliveryNo(ctx, sched.ID)
	if n2 != 2 {
		t.Fatalf("second next-no = %d, want 2", n2)
	}
	if _, err := f.InsertReportDelivery(ctx, mk(n2, "failed", reportsClock.AddDate(0, 0, -1))); err != nil {
		t.Fatalf("insert failed run: %v", err)
	}

	latest, err := f.GetLatestReportDelivery(ctx, sched.ID)
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if latest.ID != d1.ID || latest.State != "delivered" {
		t.Errorf("latest = %+v, want the delivered run %d (a later failed run does not shadow it)", latest, d1.ID)
	}

	all, err := f.ListReportDeliveries(ctx, sched.ID)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if len(all) != 2 || all[0].DeliveryNo != 2 || all[1].DeliveryNo != 1 {
		t.Errorf("list should be newest-first [2,1]; got %+v", all)
	}
}

// reportScheduleRows now sources "last sent" and "View last delivery" from the
// report_delivery store (#291/T2): a schedule with a non-failed run reads its instant
// as a relative age and lights the menu item at the stable route, while a schedule that
// has never run keeps the em-dash empty-state and a disabled item — no fabrication
// (ADR-0110). The two schedules are listed newest-first (id DESC).
func TestReportScheduleRowsLastSent(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	ctx := context.Background()

	// Schedule A delivered three days before the fixed render clock; schedule B has
	// never run. B is filed last, so it sorts first (id DESC).
	schedA, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly exposure summary", Cadence: "weekly", Format: "pdf", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule A: %v", err)
	}
	if _, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Monthly asset inventory", Cadence: "monthly", Format: "csv", CreatedBy: admin.ID,
	}); err != nil {
		t.Fatalf("insert schedule B: %v", err)
	}
	delivered := reportsClock.AddDate(0, 0, -3)
	no, _ := f.NextReportDeliveryNo(ctx, schedA.ID)
	if _, err := f.InsertReportDelivery(ctx, db.InsertReportDeliveryParams{
		ScheduleID:  schedA.ID,
		PeriodStart: pgtype.Timestamptz{Time: delivered.AddDate(0, 0, -7), Valid: true},
		PeriodEnd:   pgtype.Timestamptz{Time: delivered, Valid: true},
		DeliveryNo:  no,
		State:       "delivered",
		DeliveredAt: pgtype.Timestamptz{Time: delivered, Valid: true},
	}); err != nil {
		t.Fatalf("insert delivery: %v", err)
	}

	srv := newServer(f, testKey, "", fixedClock())
	rows := srv.reportScheduleRows(ctx)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	// B (never run) sorts first: the genuine empty-state — em dash, disabled item.
	if rows[0].Name != "Monthly asset inventory" {
		t.Fatalf("row[0] = %q, want newest-first Monthly asset inventory", rows[0].Name)
	}
	if rows[0].HasDelivery || rows[0].LastSent != "—" || rows[0].DeliveryHref != "" {
		t.Errorf("un-run schedule should be empty-stated; got %+v", rows[0])
	}
	// A read its delivered run: three days ago, menu lit at the stable route.
	if !rows[1].HasDelivery || rows[1].LastSent != "3d" || rows[1].DeliveryHref != reportDeliveryHref {
		t.Errorf("delivered schedule should read its receipt (3d, lit); got %+v", rows[1])
	}
}

// The recurring-reports row menu ports Reports.jsx's per-row DropdownMenu with its
// four live actions (#290, P0.6/T4): "View last delivery" opens T3's /reports/delivery
// artifact where a report has delivered and renders disabled where none has; Run now
// and Delete are per-row POST forms carrying the schedule id; Edit links to the
// prefilled wizard at /reports/schedule/{id}/edit.
func TestReportScheduleRowMenu(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"Title": "Reports", "NavActive": "reports", "IsAdmin": true,
		"Account": db.Account{},
		"Schedules": []reportScheduleRow{
			{ID: 7, Name: "Weekly exposure summary", Cadence: "weekly · mon 09:00", Format: "pdf", LastSent: "3d", HasDelivery: true, DeliveryHref: "/reports/delivery"},
			{ID: 8, Name: "Monthly asset inventory", Cadence: "monthly · 1st", Format: "csv", LastSent: "—", HasDelivery: false},
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

	// The four live actions are wired per row, each carrying the schedule id.
	for _, want := range []string{
		`action="/reports/schedule/run"`,     // Run now POST form
		`action="/reports/schedule/delete"`,  // Delete POST form
		`href="/reports/schedule/7/edit"`,    // Edit link, first row's id
		`href="/reports/schedule/8/edit"`,    // Edit link, second row's id
		`<input type="hidden" name="id" value="7">`, // the id rides the row-menu forms
		"Run now", "Edit schedule", "Delete schedule",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("row menu missing live action %q; body: %s", want, page)
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

// Run now dispatches an on-demand run: an admin's POST stamps a report_delivery
// receipt for the current period and redirects to /reports. The wizard declares no
// recipient, so the run generates without delivering — state "generated", no
// delivered_at (a download-only schedule). A viewer is refused before the handler.
func TestReportScheduleRunNow(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	ctx := context.Background()
	sched, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly exposure summary", Cadence: "weekly · mon 09:00", Format: "pdf", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	base := start(t, f, "")

	// A viewer cannot run it.
	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, base+"/reports/schedule/run", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer run status = %d, want 403", vr.StatusCode)
	}
	vr.Body.Close()
	if len(f.reportDeliveries) != 0 {
		t.Fatalf("viewer's denied run still stamped a delivery: %d", len(f.reportDeliveries))
	}

	// The admin runs it: one receipt is stamped, redirect to /reports.
	ac := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, ac, base+"/reports/schedule/run", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports" {
		t.Fatalf("admin run: status=%d loc=%q, want 303 /reports; body: %s", resp.StatusCode, resp.Header.Get("Location"), body(t, resp))
	}
	resp.Body.Close()

	if len(f.reportDeliveries) != 1 {
		t.Fatalf("run stamped %d deliveries, want 1", len(f.reportDeliveries))
	}
	d := f.reportDeliveries[0]
	if d.ScheduleID != sched.ID || d.DeliveryNo != 1 {
		t.Errorf("delivery = {schedule %d, no %d}, want {schedule %d, no 1}", d.ScheduleID, d.DeliveryNo, sched.ID)
	}
	if d.State != "generated" {
		t.Errorf("state = %q, want generated (download-only run, no recipient)", d.State)
	}
	if d.DeliveredAt.Valid {
		t.Errorf("delivered_at should be null on a generated (undelivered) run; got %v", d.DeliveredAt.Time)
	}
	// The receipt now backs the row's last-sent read: a second run advances the sequence.
	resp2 := postForm(t, ac, base+"/reports/schedule/run", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	resp2.Body.Close()
	if f.reportDeliveries[1].DeliveryNo != 2 {
		t.Errorf("second run delivery_no = %d, want 2", f.reportDeliveries[1].DeliveryNo)
	}
}

// Edit updates a schedule in place: an admin's finishing edit POST rewrites the
// declared contents of the target row (same id, same created_by) rather than filing a
// new one. A stale id is not persisted. A viewer is refused.
func TestReportScheduleEdit(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	ctx := context.Background()
	sched, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly exposure summary", Cadence: "weekly · mon 09:00", Format: "pdf", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	base := start(t, f, "")

	editURL := base + "/reports/schedule/" + strconv.FormatInt(sched.ID, 10) + "/edit"

	// A viewer cannot edit.
	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, editURL, url.Values{
		"action": {"finish"}, "step": {"2"}, "id": {strconv.FormatInt(sched.ID, 10)},
		"name": {"Hijacked"}, "sections": {"kpis"}, "cad": {"Daily · 08:00"},
	})
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer edit status = %d, want 403", vr.StatusCode)
	}
	vr.Body.Close()

	// The admin edits it: the row is updated in place, not appended.
	ac := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, ac, editURL, url.Values{
		"action": {"finish"}, "step": {"2"}, "id": {strconv.FormatInt(sched.ID, 10)},
		"name": {"Daily exposure summary"}, "sections": {"kpis", "coverage-gaps"}, "cad": {"Daily · 08:00"},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports" {
		t.Fatalf("admin edit: status=%d loc=%q, want 303 /reports; body: %s", resp.StatusCode, resp.Header.Get("Location"), body(t, resp))
	}
	resp.Body.Close()

	if len(f.reportSchedules) != 1 {
		t.Fatalf("edit changed the row count to %d, want 1 (update in place, not insert)", len(f.reportSchedules))
	}
	got := f.reportSchedules[0]
	if got.ID != sched.ID || got.CreatedBy != admin.ID {
		t.Errorf("edit changed identity: id %d created_by %d, want id %d created_by %d", got.ID, got.CreatedBy, sched.ID, admin.ID)
	}
	if got.Name != "Daily exposure summary" || got.Cadence != "daily · 08:00" {
		t.Errorf("edit did not rewrite contents: name=%q cadence=%q", got.Name, got.Cadence)
	}
}

// Delete removes a schedule: an admin's POST drops the row and redirects; a stale id
// is a no-op, not an error. A viewer is refused.
func TestReportScheduleDelete(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	ctx := context.Background()
	sched, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly exposure summary", Cadence: "weekly", Format: "pdf", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	base := start(t, f, "")

	// A viewer cannot delete.
	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer delete status = %d, want 403", vr.StatusCode)
	}
	vr.Body.Close()
	if len(f.reportSchedules) != 1 {
		t.Fatalf("viewer's denied delete removed the row: %d left", len(f.reportSchedules))
	}

	// The admin deletes it.
	ac := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports" {
		t.Fatalf("admin delete: status=%d loc=%q, want 303 /reports", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 0 {
		t.Fatalf("delete left %d schedules, want 0", len(f.reportSchedules))
	}

	// A repeat delete of the now-stale id is a no-op, not an error.
	resp2 := postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if resp2.StatusCode != http.StatusSeeOther {
		t.Fatalf("stale delete status = %d, want 303 (idempotent)", resp2.StatusCode)
	}
	resp2.Body.Close()
}

// Report scheduling is live (#290, P0.6/T4): an admin's finishing wizard POST files
// a report_schedule from the parsed Scope + Cadence and redirects to /reports. The
// stored row carries the trimmed name, the chosen section keys, the cadence label,
// the fixed pdf format, and — the wizard has no recipient field — an empty
// delivery_target (download-only), never an invented recipient.
func TestReportScheduleCreateLive(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/reports/schedule/new", url.Values{
		"action":   {"finish"},
		"step":     {"2"},
		"name":     {"  Q3 exposure digest  "},
		"sections": {"kpis", "signal-changes"},
		"cad":      {"Weekly · mon 09:00"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("admin schedule finish POST: status=%d, want 303; body: %s", resp.StatusCode, body(t, resp))
	}
	if loc := resp.Header.Get("Location"); loc != "/reports" {
		t.Fatalf("create redirect = %q, want /reports", loc)
	}
	resp.Body.Close()

	if len(f.reportSchedules) != 1 {
		t.Fatalf("live create filed %d schedules, want 1", len(f.reportSchedules))
	}
	got := f.reportSchedules[0]
	if got.Name != "Q3 exposure digest" {
		t.Errorf("name = %q, want trimmed \"Q3 exposure digest\"", got.Name)
	}
	if got.Cadence != "weekly · mon 09:00" {
		t.Errorf("cadence = %q, want the lower-cased preset label", got.Cadence)
	}
	if got.Format != "pdf" {
		t.Errorf("format = %q, want pdf", got.Format)
	}
	if got.DeliveryTarget != "" {
		t.Errorf("delivery_target = %q, want empty (no recipient field in the wizard)", got.DeliveryTarget)
	}
	var sections []string
	if err := json.Unmarshal(got.Sections, &sections); err != nil {
		t.Fatalf("sections is not a JSON array: %v (%s)", err, got.Sections)
	}
	if len(sections) != 2 || sections[0] != "kpis" || sections[1] != "signal-changes" {
		t.Errorf("sections = %v, want [kpis signal-changes] in canonical order", sections)
	}
}

// The Delivery step binds a schedule to a Channel (P0.6c/T7, #508): a finishing wizard
// POST carrying a channel id stores it as the schedule's channel_id, and the recurring-
// reports Delivery cell renders the bound channel's URL. A schedule with no channel
// (the download-only default) stores a NULL channel_id and reads "download only". The
// channel binding — not the superseded delivery_target — is the destination of record.
func TestReportScheduleChannelBinding(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	ctx := context.Background()
	chID, err := f.CreateChannel(ctx, db.CreateChannelParams{
		Url: "https://ops.example/hook", RouteDrift: true, Enabled: true, CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Finish the wizard with the channel chosen on the Delivery step.
	resp := postForm(t, ac, base+"/reports/schedule/new", url.Values{
		"action":   {"finish"},
		"step":     {"3"},
		"name":     {"Weekly exposure summary"},
		"sections": {"kpis"},
		"cad":      {"Weekly · mon 09:00"},
		"channel":  {strconv.FormatInt(chID, 10)},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("channel-bound finish POST: status=%d, want 303; body: %s", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	if len(f.reportSchedules) != 1 {
		t.Fatalf("filed %d schedules, want 1", len(f.reportSchedules))
	}
	bound := f.reportSchedules[0]
	if !bound.ChannelID.Valid || bound.ChannelID.Int64 != chID {
		t.Errorf("channel_id = %+v, want a valid binding to channel %d", bound.ChannelID, chID)
	}
	if bound.DeliveryTarget != "" {
		t.Errorf("delivery_target = %q, want empty (superseded by the channel binding)", bound.DeliveryTarget)
	}

	// A download-only schedule (no channel field) binds nothing.
	resp2 := postForm(t, ac, base+"/reports/schedule/new", url.Values{
		"action":   {"finish"},
		"step":     {"3"},
		"name":     {"Monthly asset inventory"},
		"sections": {"kpis"},
		"cad":      {"Monthly · 1st"},
	})
	if resp2.StatusCode != http.StatusSeeOther {
		t.Fatalf("download-only finish POST: status=%d, want 303", resp2.StatusCode)
	}
	resp2.Body.Close()

	var downloadOnly db.ReportSchedule
	for _, sc := range f.reportSchedules {
		if sc.Name == "Monthly asset inventory" {
			downloadOnly = sc
		}
	}
	if downloadOnly.ChannelID.Valid {
		t.Errorf("download-only schedule channel_id = %+v, want NULL", downloadOnly.ChannelID)
	}

	// The Delivery column renders the bound channel's URL, or "download only".
	srv := newServer(f, testKey, "", fixedClock())
	rows := srv.reportScheduleRows(ctx)
	byName := map[string]string{}
	for _, r := range rows {
		byName[r.Name] = r.Delivery
	}
	if byName["Weekly exposure summary"] != "https://ops.example/hook" {
		t.Errorf("bound schedule Delivery = %q, want the channel URL", byName["Weekly exposure summary"])
	}
	if byName["Monthly asset inventory"] != "download only" {
		t.Errorf("download-only schedule Delivery = %q, want \"download only\"", byName["Monthly asset inventory"])
	}
}

// A stepping (non-finishing) wizard POST 303-redirects to the next step's GET URL rather
// than filing a schedule (the PRG post-back, #23f): Next advances only when the current
// step's gate passes, and the accumulated values ride the redirect query so the target is
// bookmarkable. An admin advancing Scope with a name and sections lands on step 1 (Cadence)
// and nothing is persisted until the finishing submit.
func TestReportScheduleWizardStepping(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/reports/schedule/new", url.Values{
		"action":   {"next"},
		"step":     {"0"},
		"name":     {"Weekly exposure summary"},
		"sections": {"kpis"},
		"cad":      {"Weekly · mon 09:00"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("stepping POST status = %d, want 303 (PRG redirect to the next step)", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/reports/schedule/new?") || !strings.Contains(loc, "step=1") {
		t.Errorf("Next from Scope should redirect to the step-1 GET URL; location: %q", loc)
	}
	// The accumulated values ride the redirect query so the target renders the full state.
	if !strings.Contains(loc, "name=Weekly") || !strings.Contains(loc, "sections=kpis") {
		t.Errorf("redirect should carry the accumulated name + sections; location: %q", loc)
	}
	if len(f.reportSchedules) != 0 {
		t.Fatalf("stepping filed a schedule (%d); only finish persists", len(f.reportSchedules))
	}

	// Following the redirect renders the Cadence step.
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "Cadence") || !strings.Contains(page, `name="cad"`) {
		t.Errorf("the step-1 GET should render the Cadence step; body: %s", page)
	}
}

// A viewer's create POST is refused before the handler by requireAdmin (403), filing
// nothing — declaring a schedule is an admin config act, so the role gate stops a
// viewer at the auth wrapper regardless of the body.
func TestReportScheduleCreateRefusesViewer(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/reports/schedule/new", url.Values{
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

// The Reports surface offers a live "New schedule" control that opens the wizard, and
// the wizard renders its first (Scope) step with the real create form. Scheduling is
// no longer the disabled not-yet-available surface (#290, P0.6/T4).
func TestReportScheduleWizardLive(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The New schedule control on the Reports page opens the wizard, and the old
	// disabled copy is gone.
	page := getBody(t, ac, base+"/reports", http.StatusOK)
	if !strings.Contains(page, `href="/reports/schedule/new"`) {
		t.Errorf("Reports should link the New schedule control to the wizard; body: %s", page)
	}
	if strings.Contains(page, "Report scheduling is not available yet") {
		t.Errorf("New schedule control still marked unavailable; body: %s", page)
	}

	// The wizard itself renders the Scope step with the live create form: a name input,
	// the section checkbox group, and a finish target of /reports/schedule/new (the PRG
	// post-back route, #23f).
	wiz := getBody(t, ac, base+"/reports/schedule/new", http.StatusOK)
	if !strings.Contains(wiz, `action="/reports/schedule/new"`) {
		t.Errorf("wizard should post to the live create route; body: %s", wiz)
	}
	for _, want := range []string{
		`name="name"`,                   // the Report name input
		`name="sections" value="kpis"`,  // a section checkbox
		"Scope", "Cadence", "Review",    // the three step titles
	} {
		if !strings.Contains(wiz, want) {
			t.Errorf("wizard Scope step missing %q; body: %s", want, wiz)
		}
	}
}

// The wizard is admin-gated: a viewer's GET is bounced by requireAdmin (403), never
// served the create form.
func TestReportScheduleWizardRefusesViewer(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp, err := vc.Get(base + "/reports/schedule/new")
	if err != nil {
		t.Fatal(err)
	}
	body(t, resp)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer wizard GET status = %d, want 403", resp.StatusCode)
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

// resolveReportsWindow parses ?period= (or a custom ?start=&end= pair) into the reporting
// window, defaulting to the design's 7d preset (twelve-week span) when the token is absent
// or unrecognised, and resolving a valid custom pair to a stable token + span.
func TestResolveReportsWindow(t *testing.T) {
	cases := []struct {
		query      string
		wantToken  string
		wantLabel  string
		wantWeeks  int
	}{
		{"", "7d", "Last 7d", reportsHeatWeeks},                 // absent -> default preset
		{"period=24h", "24h", "Last 24h", 4},                    // preset
		{"period=30d", "30d", "Last 30d", 26},                   // preset
		{"period=90d", "90d", "Last 90d", 52},                   // preset
		{"period=7d", "7d", "Last 7d", reportsHeatWeeks},        // the default, explicitly
		{"period=nope", "7d", "Last 7d", reportsHeatWeeks},      // unrecognised -> default
		{"start=2026-08-01&end=2026-08-14", "custom_2026-08-01_2026-08-14", "2026-08-01 – 2026-08-14", 2}, // custom pair
		{"period=custom_2026-08-01_2026-08-07", "custom_2026-08-01_2026-08-07", "2026-08-01 – 2026-08-07", 1}, // custom token from an export link
		{"start=bogus&end=2026-08-14", "7d", "Last 7d", reportsHeatWeeks}, // malformed custom -> preset fallback
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/reports?"+c.query, nil)
		win := resolveReportsWindow(r)
		if win.Token != c.wantToken || win.Label != c.wantLabel || win.Weeks != c.wantWeeks {
			t.Errorf("resolveReportsWindow(%q) = {%q,%q,%d}, want {%q,%q,%d}",
				c.query, win.Token, win.Label, win.Weeks, c.wantToken, c.wantLabel, c.wantWeeks)
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

// The header period picker (#23b) renders the preset links, marks the active one, re-skins
// the period-aware captions, and carries the active period token into the export links. The
// default (no param) is the design's 7d preset.
func TestReportsPeriodPickerRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Default: 7d, the preset link is marked active, and the export links carry period=7d.
	def := getBody(t, ac, base+"/reports", http.StatusOK)
	if !strings.Contains(def, `href="/reports?period=7d"`) {
		t.Errorf("period picker should offer the 7d preset link; body: %s", def)
	}
	if !strings.Contains(def, "/reports/export?format=csv&period=7d") {
		t.Errorf("default export link should carry period=7d; body: %s", def)
	}
	if !strings.Contains(def, "/reports/export?format=pdf&period=7d") {
		t.Errorf("export SplitButton should offer the PDF route carrying period=7d; body: %s", def)
	}

	// A selected preset re-skins the header/captions and the export links to the period.
	wide := getBody(t, ac, base+"/reports?period=30d", http.StatusOK)
	if !strings.Contains(wide, "Last 30d") {
		t.Errorf("period=30d should re-skin the header/captions to 'Last 30d'; body: %s", wide)
	}
	if !strings.Contains(wide, "/reports/export?format=json&period=30d") {
		t.Errorf("period=30d JSON export link should carry period=30d; body: %s", wide)
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

	resp, err := ac.Get(base + "/reports/export?format=csv&period=7d")
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

	resp, err := ac.Get(base + "/reports/export?format=json&period=30d")
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

// GET /reports/export?format=pdf streams an application/pdf attachment — the delivered-
// report document recomputed from the period bounds (#23c), rendered by the SAME
// internal/message.RenderArtifactPDF the delivery PDF uses. An empty estate still renders
// the empty-state document, never a fabricated one.
func TestReportsExportPDF(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/reports/export?format=pdf&period=7d")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("export pdf status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("export pdf Content-Type = %q, want application/pdf", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment; filename=") || !strings.Contains(cd, ".pdf") {
		t.Errorf("export pdf Content-Disposition = %q, want an attachment .pdf filename", cd)
	}
	// A real PDF document, not a stub — it begins with the %PDF signature.
	if !strings.HasPrefix(got, "%PDF") {
		t.Errorf("export pdf body should be a real PDF (%%PDF header); got %.16q", got)
	}
}

// An unrecognised export format is a 400 — the handler serves csv, json and pdf.
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
