package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
)

func TestReportDeliveryRendersRealDelivery(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	ctx := context.Background()

	sched, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly exposure summary", Cadence: "weekly", Format: "pdf",
		DeliveryTarget: "https://ops.example.test/hook/s3cr3t-token", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}

	periodStart := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	deliveredAt := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)
	if _, err := f.InsertReportDelivery(ctx, db.InsertReportDeliveryParams{
		ScheduleID:  sched.ID,
		PeriodStart: pgtype.Timestamptz{Time: periodStart, Valid: true},
		PeriodEnd:   pgtype.Timestamptz{Time: periodEnd, Valid: true},
		DeliveryNo:  7,
		State:       "delivered",
		DeliveredAt: pgtype.Timestamptz{Time: deliveredAt, Valid: true},
	}); err != nil {
		t.Fatalf("insert delivery: %v", err)
	}

	inPeriod := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	f.signalInstances = append(f.signalInstances,
		db.SignalInstance{ID: 1, SignalName: "certificate-expired", SubjectKey: "idp.example.test",
			FirstSeen: pgtype.Timestamptz{Time: inPeriod, Valid: true}},
		db.SignalInstance{ID: 2, SignalName: "certificate-not-yet-valid", SubjectKey: "api.example.test",
			FirstSeen: pgtype.Timestamptz{Time: inPeriod, Valid: true}},
		db.SignalInstance{ID: 3, SignalName: "cname-target-name-error", SubjectKey: "old.example.test",
			FirstSeen: pgtype.Timestamptz{Time: periodStart.AddDate(0, 0, -30), Valid: true}},
	)

	withdrewIn := time.Date(2026, 8, 18, 8, 0, 0, 0, time.UTC)
	f.withdrawalLifespans = append(f.withdrawalLifespans,
		db.ListWithdrawalLifespansRow{SubjectKind: "name", SubjectKey: "legacy.example.test",
			WithdrawnAt: pgtype.Timestamptz{Time: withdrewIn, Valid: true},
			FirstOpened: pgtype.Timestamptz{Time: periodStart.AddDate(0, 0, -60), Valid: true}},
		db.ListWithdrawalLifespansRow{SubjectKind: "name", SubjectKey: "ancient.example.test",
			WithdrawnAt: pgtype.Timestamptz{Time: periodStart.AddDate(0, 0, -10), Valid: true},
			FirstOpened: pgtype.Timestamptz{Time: periodStart.AddDate(0, 0, -90), Valid: true}},
	)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports/delivery", http.StatusOK)

	for _, want := range []string{
		"Weekly exposure summary",
		"2026-08-15 → 2026-08-22",
		"delivery #7",
		"Open signals by severity",
		"Critical",
		"New this week",
		"Certificate expired",
		"idp.example.test",
		"aug 20",
		"Withdrawn by the world",
		"legacy.example.test",
		"delivered 2026-08-22T09:00:00Z",
		"ops.example.test",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("real delivery page missing %q; body: %s", want, page)
		}
	}

	if strings.Contains(page, "No delivery yet") {
		t.Errorf("a delivered report must not render the empty-state; body: %s", page)
	}
	if strings.Contains(page, "s3cr3t-token") {
		t.Errorf("receipt leaked the raw delivery-target token; body: %s", page)
	}
	if strings.Contains(page, "old.example.test") {
		t.Errorf("a pre-period signal leaked into the period document; body: %s", page)
	}
	if strings.Contains(page, "ancient.example.test") {
		t.Errorf("a pre-period withdrawal leaked into the period document; body: %s", page)
	}
}

func TestReportDeliveryPDFPeriodNamedForRealDelivery(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	ctx := context.Background()

	sched, err := f.InsertReportSchedule(ctx, db.InsertReportScheduleParams{
		Name: "Weekly exposure summary", Cadence: "weekly", Format: "pdf",
		DeliveryTarget: "https://ops.example.test/hook", CreatedBy: admin.ID,
	})
	if err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	if _, err := f.InsertReportDelivery(ctx, db.InsertReportDeliveryParams{
		ScheduleID:  sched.ID,
		PeriodStart: pgtype.Timestamptz{Time: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), Valid: true},
		PeriodEnd:   pgtype.Timestamptz{Time: time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC), Valid: true},
		DeliveryNo:  1,
		State:       "delivered",
		DeliveredAt: pgtype.Timestamptz{Time: time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC), Valid: true},
	}); err != nil {
		t.Fatalf("insert delivery: %v", err)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/reports/delivery/pdf")
	if err != nil {
		t.Fatal(err)
	}
	b := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /reports/delivery/pdf status = %d, want 200 (body: %.80q)", resp.StatusCode, b)
	}
	if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, `filename="report-2026-08-15-to-2026-08-22.pdf"`) {
		t.Errorf("Content-Disposition = %q, want period-dated report filename", cd)
	}
	if !strings.HasPrefix(b, "%PDF-") {
		t.Errorf("body is not a PDF (no %%PDF- header); first bytes: %.8q", b)
	}
}

func TestReportDeliveryEmptyStateWithoutDelivery(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/reports/delivery", http.StatusOK)

	if !strings.Contains(page, "No delivery yet") {
		t.Errorf("no delivery: expected the design-system empty-state; body: %s", page)
	}
	if strings.Contains(page, "Open signals by severity") {
		t.Errorf("no delivery must not render a recomputed document; body: %s", page)
	}
}
