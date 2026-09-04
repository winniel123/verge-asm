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
	"github.com/winniel123/verge-asm/internal/drift"
)

var reportsClock = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

func TestReportsRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
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

func TestReportsRendersActivityAndComposition(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	day := reportsClock.AddDate(0, 0, -1)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1, "hot", day, 3, 1, 1, 1, 0, 0),
		progressRow(2, "dns", day, 2, 0, 0, 2, 0, 0),
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	if !strings.Contains(page, `class="sh-pill on" href="/reports"`) {
		t.Errorf("reports nav pill not marked active; body: %s", page)
	}
	for _, want := range []string{
		"Open signals", "New assets discovered", "Mean time to withdrawal",
		"Open signals over time",
		"Scans per day", "Scans per day, Last 7d",
		"Recurring reports", "New schedule",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("reports page missing %q; body: %s", want, page)
		}
	}

	if !strings.Contains(page, "color-mix(in srgb, var(--chart-1)") {
		t.Errorf("heatmap intensity fill missing; body: %s", page)
	}
	if strings.Contains(page, "No scans in the Last 7d") {
		t.Errorf("heatmap should be wired, not empty-stated; body: %s", page)
	}

	low := strings.ToLower(page)
	if strings.Contains(low, "mean time to resolve") || strings.Contains(low, "time to resolve") {
		t.Errorf("no resolve metric may appear — signals are withdrawn, not resolved; body: %s", page)
	}
}

func TestReportsRendersNewAssetsDiscovered(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	at := reportsClock.AddDate(0, 0, -2)
	f.addResolution(t, admin.ID, "api.example.com", "dns", at, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", at, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	for _, want := range []string{
		"New assets discovered",
		"1 names",
		"1 services",
		"border-bottom:1px solid var(--chart-grid)",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("discovery card missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, "&#43;2</span>") {
		t.Errorf("discovery card should show a +2 vs-previous-period delta; body: %s", page)
	}
	if strings.Contains(page, "Assets watched") {
		t.Errorf("the interim \"Assets watched\" card must be replaced by \"New assets discovered\"; body: %s", page)
	}
}

func TestReportsEmptyStates(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports", http.StatusOK)

	for _, want := range []string{
		"No signal history",
		"No signals firing",
		"No scans in the Last 7d",
		"No recurring reports",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("reports empty-state missing %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "color-mix(in srgb, var(--chart-1)") &&
		!strings.Contains(page, "28%, var(--surface)") {
		t.Errorf("empty heatmap should not draw wired cells; body: %s", page)
	}
}

func TestLastReportDelivery(t *testing.T) {
	if href, has := lastReportDelivery([]deliveryView{{State: "delivered"}}); !has || href != "/reports/delivery" {
		t.Errorf("delivered report: got (%q, %v), want (/reports/delivery, true)", href, has)
	}
	if href, has := lastReportDelivery(nil); has || href != "" {
		t.Errorf("no delivery: got (%q, %v), want (\"\", false)", href, has)
	}
	if href, has := lastReportDelivery([]deliveryView{{State: "undelivered", Failed: true}}); has || href != "" {
		t.Errorf("undelivered only: got (%q, %v), want (\"\", false)", href, has)
	}
}

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

func TestReportScheduleRowsLastSent(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	ctx := context.Background()

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
	if rows[0].Name != "Monthly asset inventory" {
		t.Fatalf("row[0] = %q, want newest-first Monthly asset inventory", rows[0].Name)
	}
	if rows[0].HasDelivery || rows[0].LastSent != "—" || rows[0].DeliveryHref != "" {
		t.Errorf("un-run schedule should be empty-stated; got %+v", rows[0])
	}
	if !rows[1].HasDelivery || rows[1].LastSent != "3d" || rows[1].DeliveryHref != reportDeliveryHref {
		t.Errorf("delivered schedule should read its receipt (3d, lit); got %+v", rows[1])
	}
}

func TestReportScheduleRowMenu(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"Title": "Reports", "NavActive": "reports", "IsAdmin": true,
		"Account": db.Account{},
		"BackURL": "/reports?period=30d",
		"Schedules": []reportScheduleRow{
			{ID: 7, Name: "Weekly exposure summary", Cadence: "weekly · mon 09:00", Format: "pdf", LastSent: "3d", HasDelivery: true, DeliveryHref: "/reports/delivery"},
			{ID: 8, Name: "Monthly asset inventory", Cadence: "monthly · 1st", Format: "csv", LastSent: "—", HasDelivery: false},
		},
	}
	if err := tmpl.ExecuteTemplate(&buf, "reports", data); err != nil {
		t.Fatalf("execute reports template: %v", err)
	}
	page := buf.String()

	if !strings.Contains(page, `href="/reports/delivery"`) {
		t.Errorf("delivered row menu should link to /reports/delivery; body: %s", page)
	}
	if !strings.Contains(page, "View last delivery") {
		t.Errorf("row menu missing the ported 'View last delivery' item; body: %s", page)
	}
	if !strings.Contains(page, `aria-disabled="true" title="No delivery yet"`) {
		t.Errorf("undelivered row should render a disabled 'View last delivery'; body: %s", page)
	}

	for _, want := range []string{
		`action="/reports/schedule/run"`,
		`action="/reports/schedule/delete"`,
		`href="/reports/schedule/7/edit?return=`,
		`href="/reports/schedule/8/edit?return=`,
		`<input type="hidden" name="id" value="7">`,
		`<input type="hidden" name="return" value="/reports?period=30d">`,
		"Run now", "Edit schedule", "Delete schedule",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("row menu missing live action %q; body: %s", want, page)
		}
	}

	if strings.Contains(page, "No recurring reports") {
		t.Errorf("with schedules present the table should render, not the empty-state; body: %s", page)
	}
	if !strings.Contains(page, "Weekly exposure summary") || !strings.Contains(page, "Monthly asset inventory") {
		t.Errorf("schedule rows missing; body: %s", page)
	}
}

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

	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, base+"/reports/schedule/run", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer run status = %d, want 403", vr.StatusCode)
	}
	vr.Body.Close()
	if len(f.reportDeliveries) != 0 {
		t.Fatalf("viewer's denied run still stamped a delivery: %d", len(f.reportDeliveries))
	}

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
	resp2 := postForm(t, ac, base+"/reports/schedule/run", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	resp2.Body.Close()
	if f.reportDeliveries[1].DeliveryNo != 2 {
		t.Errorf("second run delivery_no = %d, want 2", f.reportDeliveries[1].DeliveryNo)
	}
}

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

	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, editURL, url.Values{
		"action": {"finish"}, "step": {"2"}, "id": {strconv.FormatInt(sched.ID, 10)},
		"name": {"Hijacked"}, "sections": {"kpis"}, "cad": {"Daily · 08:00"},
	})
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer edit status = %d, want 403", vr.StatusCode)
	}
	vr.Body.Close()

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

	vc := login(t, base, "viewer", "hunter2hunter2")
	vr := postForm(t, vc, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if vr.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer delete status = %d, want 403", vr.StatusCode)
	}
	vr.Body.Close()
	if len(f.reportSchedules) != 1 {
		t.Fatalf("viewer's denied delete removed the row: %d left", len(f.reportSchedules))
	}

	ac := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports" {
		t.Fatalf("admin delete: status=%d loc=%q, want 303 /reports", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if len(f.reportSchedules) != 0 {
		t.Fatalf("delete left %d schedules, want 0", len(f.reportSchedules))
	}

	resp2 := postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {strconv.FormatInt(sched.ID, 10)}})
	if resp2.StatusCode != http.StatusSeeOther {
		t.Fatalf("stale delete status = %d, want 303 (idempotent)", resp2.StatusCode)
	}
	resp2.Body.Close()
}

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
	if !strings.Contains(loc, "name=Weekly") || !strings.Contains(loc, "sections=kpis") {
		t.Errorf("redirect should carry the accumulated name + sections; location: %q", loc)
	}
	if len(f.reportSchedules) != 0 {
		t.Fatalf("stepping filed a schedule (%d); only finish persists", len(f.reportSchedules))
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "Cadence") || !strings.Contains(page, `name="cad"`) {
		t.Errorf("the step-1 GET should render the Cadence step; body: %s", page)
	}
}

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

func TestReportScheduleWizardLive(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/reports", http.StatusOK)
	if !strings.Contains(page, `href="/reports/schedule/new?return=`) {
		t.Errorf("Reports should link the New schedule control to the wizard; body: %s", page)
	}
	if strings.Contains(page, "Report scheduling is not available yet") {
		t.Errorf("New schedule control still marked unavailable; body: %s", page)
	}

	wiz := getBody(t, ac, base+"/reports/schedule/new", http.StatusOK)
	if !strings.Contains(wiz, `action="/reports/schedule/new"`) {
		t.Errorf("wizard should post to the live create route; body: %s", wiz)
	}
	for _, want := range []string{
		`name="name"`,
		`name="sections" value="kpis"`,
		"Scope", "Cadence", "Review",
	} {
		if !strings.Contains(wiz, want) {
			t.Errorf("wizard Scope step missing %q; body: %s", want, wiz)
		}
	}
}

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

func TestFoldScanActivity(t *testing.T) {
	s := &server{now: func() time.Time { return reportsClock }}
	rows := []db.ListDispatchProgressRow{
		progressRow(1, "hot", reportsClock, 2, 0, 1, 1, 0, 0),
		progressRow(2, "dns", reportsClock.AddDate(0, 0, -1), 2, 0, 0, 2, 0, 0),
		progressRow(3, "dns", reportsClock.AddDate(0, 0, -200), 1, 0, 0, 1, 0, 0),
	}
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
	if string(cells[reportsHeatDays-1].Bg) == "var(--surface-sunken)" {
		t.Errorf("today's cell should be wired, got sunken")
	}
	if string(cells[reportsHeatDays-2].Bg) == "var(--surface-sunken)" {
		t.Errorf("yesterday's cell should be wired, got sunken")
	}
	if string(cells[0].Bg) != "var(--surface-sunken)" {
		t.Errorf("the oldest, empty day should be sunken, got %q", cells[0].Bg)
	}
	// The legend swatch beside this grid uses --row-sep, so the empty cell's border pins it (#1088).
	if string(cells[0].Border) != "var(--row-sep)" {
		t.Errorf("the empty day's border = %q, want var(--row-sep)", cells[0].Border)
	}
	if string(cells[reportsHeatDays-1].Border) != "transparent" {
		t.Errorf("a wired day's border = %q, want transparent", cells[reportsHeatDays-1].Border)
	}
}

func TestReportsTimeSeriesGridStrokes(t *testing.T) {
	pts := []drift.SignalPoint{
		{Start: reportsClock.AddDate(0, 0, -14), Standing: 0, StandingElevated: 0},
		{Start: reportsClock.AddDate(0, 0, -7), Standing: 40, StandingElevated: 12},
		{Start: reportsClock, Standing: 20, StandingElevated: 4},
	}
	ts, ok := buildReportsTimeSeries(pts)
	if !ok {
		t.Fatal("buildReportsTimeSeries declined a three-point series")
	}
	if len(ts.Grid) < 2 {
		t.Fatalf("grid has %d lines, want at least a baseline and one rule", len(ts.Grid))
	}
	if ts.Grid[0].Label != "0" {
		t.Fatalf("first grid line is %q, want the zero baseline", ts.Grid[0].Label)
	}
	if ts.Grid[0].Stroke != "var(--border-default)" {
		t.Errorf("zero baseline stroke = %q, want var(--border-default)", ts.Grid[0].Stroke)
	}
	for _, g := range ts.Grid[1:] {
		if g.Stroke != "var(--row-sep)" {
			t.Errorf("gridline at %q stroke = %q, want var(--row-sep)", g.Label, g.Stroke)
		}
	}
}

func TestResolveReportsWindow(t *testing.T) {
	cases := []struct {
		query     string
		wantToken string
		wantLabel string
		wantWeeks int
	}{
		{"", "7d", "Last 7d", reportsHeatWeeks},
		{"period=24h", "24h", "Last 24h", 4},
		{"period=30d", "30d", "Last 30d", 26},
		{"period=90d", "90d", "Last 90d", 52},
		{"period=7d", "7d", "Last 7d", reportsHeatWeeks},
		{"period=nope", "7d", "Last 7d", reportsHeatWeeks},
		{"start=2026-08-01&end=2026-08-14", "custom_2026-08-01_2026-08-14", "2026-08-01 – 2026-08-14", 2},
		{"period=custom_2026-08-01_2026-08-07", "custom_2026-08-01_2026-08-07", "2026-08-01 – 2026-08-07", 1},
		{"start=bogus&end=2026-08-14", "7d", "Last 7d", reportsHeatWeeks},
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

func TestFoldScanActivityRangeWindow(t *testing.T) {
	s := &server{now: func() time.Time { return reportsClock }}
	rows := []db.ListDispatchProgressRow{
		progressRow(1, "dns", reportsClock.AddDate(0, 0, -100), 1, 0, 0, 1, 0, 0),
	}

	cells12, _, window12, _ := s.foldScanActivity(rows, reportsHeatWeeks*7)
	if len(cells12) != reportsHeatWeeks*7 {
		t.Fatalf("12wk cells = %d, want %d", len(cells12), reportsHeatWeeks*7)
	}
	if window12 != 0 {
		t.Errorf("12wk window = %d, want 0 (100-day-old row is out of the 84-day window)", window12)
	}

	cells26, _, window26, _ := s.foldScanActivity(rows, 26*7)
	if len(cells26) != 26*7 {
		t.Fatalf("26wk cells = %d, want %d", len(cells26), 26*7)
	}
	if window26 != 1 {
		t.Errorf("26wk window = %d, want 1 (100-day-old row is inside the 182-day window)", window26)
	}
}

func TestFoldScanActivitySparseCorpusIsContiguous(t *testing.T) {
	s := &server{now: func() time.Time { return reportsClock }}
	activeOffsets := []int{0, 10, 40}
	rows := make([]db.ListDispatchProgressRow, 0, len(activeOffsets))
	for i, off := range activeOffsets {
		rows = append(rows, progressRow(int64(i+1), "dns", reportsClock.AddDate(0, 0, -off), 1, 0, 0, 1, 0, 0))
	}

	cells, total, _, _ := s.foldScanActivity(rows, reportsHeatDays)

	if len(cells) != reportsHeatDays {
		t.Fatalf("cells = %d, want %d (one per day incl. zeros)", len(cells), reportsHeatDays)
	}
	if total != len(activeOffsets) {
		t.Errorf("total in-window scans = %d, want %d", total, len(activeOffsets))
	}

	wired, zeros := 0, 0
	for i, c := range cells {
		if c.Title == "" {
			t.Fatalf("cell %d has no title — a day was omitted from the series", i)
		}
		switch string(c.Bg) {
		case "var(--surface-sunken)":
			zeros++
		default:
			wired++
		}
	}
	if wired != len(activeOffsets) {
		t.Errorf("wired (non-zero) cells = %d, want %d (the active days)", wired, len(activeOffsets))
	}
	if zeros != reportsHeatDays-len(activeOffsets) {
		t.Errorf("zero cells = %d, want %d (every silent day present, not dropped)", zeros, reportsHeatDays-len(activeOffsets))
	}

	for _, off := range activeOffsets {
		idx := reportsHeatDays - 1 - off
		if string(cells[idx].Bg) == "var(--surface-sunken)" {
			t.Errorf("cell at offset %d (index %d) should be wired, got sunken", off, idx)
		}
	}
}

func TestReportsPeriodPickerRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

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

	wide := getBody(t, ac, base+"/reports?period=30d", http.StatusOK)
	if !strings.Contains(wide, "Last 30d") {
		t.Errorf("period=30d should re-skin the header/captions to 'Last 30d'; body: %s", wide)
	}
	if !strings.Contains(wide, "/reports/export?format=json&period=30d") {
		t.Errorf("period=30d JSON export link should carry period=30d; body: %s", wide)
	}
}

func TestReportsExportCSV(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	day := reportsClock.AddDate(0, 0, -1)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1, "hot", day, 3, 1, 1, 1, 0, 0),
		progressRow(2, "dns", day, 2, 0, 0, 2, 0, 0),
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
	for _, want := range []string{"section,label,value", "summary,scans_run,2", "summary,in_flight,1", "scans_per_day,"} {
		if !strings.Contains(got, want) {
			t.Errorf("export csv missing %q; body:\n%s", want, got)
		}
	}
	if n := strings.Count(got, "scans_per_day,"); n != reportsHeatWeeks*7 {
		t.Errorf("export csv scans_per_day rows = %d, want %d", n, reportsHeatWeeks*7)
	}
}

func TestReportsExportJSON(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	day := reportsClock.AddDate(0, 0, -1)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(1, "hot", day, 3, 1, 1, 1, 0, 0),
		progressRow(2, "dns", day, 2, 0, 0, 2, 0, 0),
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
	if doc.KPIs.OpenSignals == nil {
		t.Errorf("kpis.open_signals should be a real count, got null")
	}
	if last := doc.ScansPerDay[len(doc.ScansPerDay)-1].Date; last != reportsClock.Format("2006-01-02") {
		t.Errorf("last series day = %q, want today %q", last, reportsClock.Format("2006-01-02"))
	}
}

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
	if !strings.HasPrefix(got, "%PDF") {
		t.Errorf("export pdf body should be a real PDF (%%PDF header); got %.16q", got)
	}
}

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

func TestReportsExportRequiresLogin(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	c := newClient(t)
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

func TestReportsBuildsTrendDatumWithoutError(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	// The assertion passes on an empty read, so this populated ledger is what exercises the glue.
	within := reportsClock.AddDate(0, 0, -7)
	f.signalInstances = []db.SignalInstance{
		{ID: 1000, SignalName: "cname-target-name-error", SubjectKey: "a.example.com", FirstSeen: pgtype.Timestamptz{Time: within, Valid: true}},
		{ID: 1001, SignalName: "some-unknown-calm-rule", SubjectKey: "b.example.com", FirstSeen: pgtype.Timestamptz{Time: within, Valid: true}},
	}

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
	if !strings.Contains(page, "Scans per day") {
		t.Fatal("Reports page should still render the scans-per-day heatmap with the trend datum wired")
	}
}

func TestReportsBarChartCapsAtThirtyOneBars(t *testing.T) {
	series := func(days int) []drift.DiscoveryPoint {
		pts := make([]drift.DiscoveryPoint, days)
		base := reportsClock.AddDate(0, 0, -(days - 1))
		for i := range pts {
			pts[i] = drift.DiscoveryPoint{Start: base.AddDate(0, 0, i), Count: 1}
		}
		return pts
	}

	cases := []struct {
		name     string
		weeks    int
		wantBars int
		wantAgg  bool
	}{
		{"24h preset — 28 days, daily", 4, 28, false},
		{"7d default — 84 days → weekly", 12, 12, true},
		{"30d preset — 182 days → weekly", 26, 26, true},
		{"90d preset — 364 days → 2-week", 52, 26, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			days := tc.weeks * 7
			chart := buildReportsBarChart(series(days), tc.weeks)
			if len(chart.Bars) > reportsMaxBars {
				t.Fatalf("bars = %d, exceeds the %d-bar contract", len(chart.Bars), reportsMaxBars)
			}
			if len(chart.Bars) != tc.wantBars {
				t.Errorf("bars = %d, want %d", len(chart.Bars), tc.wantBars)
			}
			if chart.LeftLabel != strconv.Itoa(tc.weeks)+"w ago" || chart.RightLabel != "today" {
				t.Errorf("labels = %q/%q, want %dw ago/today", chart.LeftLabel, chart.RightLabel, tc.weeks)
			}
			last := chart.Bars[len(chart.Bars)-1]
			if !last.Last {
				t.Error("the last bar should be emphasised (today)")
			}
			if len(chart.Bars) > 1 && chart.Bars[0].Last {
				t.Error("only the last bar should be emphasised")
			}
		})
	}

	chart := buildReportsBarChart(series(84), 12)
	for i, b := range chart.Bars {
		if b.Title != "7 assets" {
			t.Fatalf("weekly bar %d title = %q, want %q (7 daily discoveries summed)", i, b.Title, "7 assets")
		}
		if b.HeightPct != 100 {
			t.Errorf("weekly bar %d height = %d, want 100 (all weeks equal)", i, b.HeightPct)
		}
	}

	short := series(21)
	if got := aggregateDiscoveryBars(short); len(got) != 21 {
		t.Errorf("21-day series aggregated to %d bars, want 21 (unchanged within a month)", len(got))
	}
}
