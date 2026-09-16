package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
)

type gapReadStore struct {
	*fakeStore
	reachErr   error
	vantageErr error
}

func (g gapReadStore) ListOpenReachGapServices(ctx context.Context) ([]db.ListOpenReachGapServicesRow, error) {
	if g.reachErr != nil {
		return nil, g.reachErr
	}
	return g.fakeStore.ListOpenReachGapServices(ctx)
}

func (g gapReadStore) ListUnavailableVantages(ctx context.Context) ([]db.ListUnavailableVantagesRow, error) {
	if g.vantageErr != nil {
		return nil, g.vantageErr
	}
	return g.fakeStore.ListUnavailableVantages(ctx)
}

func gapReadServer(t *testing.T, f *fakeStore, reachErr, vantageErr error) *server {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	srv.useTranscriptKey(testTranscriptKey)
	srv.coldStore = gapReadStore{fakeStore: f, reachErr: reachErr, vantageErr: vantageErr}
	return srv
}

func startGapRead(t *testing.T, srv *server) string {
	t.Helper()
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func TestCoverageNamesAFailedReachGapRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "104.21.61.6:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"`+blanketdiscrim.GapCause+`"}`)
	buf := captureLog(t)

	srv := gapReadServer(t, f, errors.New("list open reach gap services: connection reset"), nil)
	ledger := srv.readCoverageGapLedger(t.Context())

	if !ledger.GapsFailed {
		t.Error("a failed reach-gap read must raise GapsFailed, not render an empty gap ledger")
	}
	if !ledger.MessagesFailed {
		t.Error("the coverage messages card derives from the same read, so it must be flagged too")
	}
	if len(ledger.Gaps) != 0 {
		t.Errorf("a failed read must substitute no gap row, got %d", len(ledger.Gaps))
	}
	if !strings.Contains(buf.String(), "connection reset") {
		t.Errorf("a degradation the operator cannot see must log; log: %q", buf.String())
	}
}

func TestCoverageNamesAFailedUnavailableVantageRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	buf := captureLog(t)

	srv := gapReadServer(t, f, nil, errors.New("list unavailable vantages: connection reset"))
	ledger := srv.readCoverageGapLedger(t.Context())

	if ledger.GapsFailed {
		t.Error("the vantage read feeds no gap row, so it must not flag the gap ledger")
	}
	if !ledger.MessagesFailed {
		t.Error("a failed unavailable-vantage read must raise MessagesFailed, not render 'no coverage messages'")
	}
	if !strings.Contains(buf.String(), "connection reset") {
		t.Errorf("a degradation the operator cannot see must log; log: %q", buf.String())
	}
}

func TestCoverageLeavesTheLedgerFlagsDownWhenBothReadsResolve(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "104.21.61.6:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"`+blanketdiscrim.GapCause+`"}`)

	srv := gapReadServer(t, f, nil, nil)
	ledger := srv.readCoverageGapLedger(t.Context())

	if ledger.GapsFailed || ledger.MessagesFailed {
		t.Errorf("a resolved read read as a failed one: gaps=%v messages=%v", ledger.GapsFailed, ledger.MessagesFailed)
	}
	if len(ledger.Gaps) != 1 {
		t.Fatalf("the blanketed service must still reach the ledger, got %d rows", len(ledger.Gaps))
	}
}

func TestCoverageStillServesWhenBothLedgerReadsFail(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, 1, "example.com")
	captureLog(t)

	srv := gapReadServer(t, f, errors.New("reach read failed"), errors.New("vantage read failed"))
	base := startGapRead(t, srv)
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/coverage", http.StatusOK)
	for _, want := range []string{"Gaps did not resolve", "Coverage messages did not resolve"} {
		if !strings.Contains(page, want) {
			t.Errorf("a failed ledger read must name itself: missing %q; body: %s", want, page)
		}
	}
	for _, banned := range []string{"No gaps this batch", "No coverage messages"} {
		if strings.Contains(page, banned) {
			t.Errorf("a failed read still claims the ledger is clean: found %q; body: %s", banned, page)
		}
	}
	if !strings.Contains(page, "Aperture") {
		t.Errorf("a failed region read took a region it does not feed down; body: %s", page)
	}
}

func TestCoverageKeepsItsEmptyStateWhenBothLedgerReadsResolve(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, 1, "example.com")

	srv := gapReadServer(t, f, nil, nil)
	base := startGapRead(t, srv)
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/coverage", http.StatusOK)
	if !strings.Contains(page, "No gaps this batch") {
		t.Errorf("a resolved read lost the honest empty state; body: %s", page)
	}
	for _, banned := range []string{"Gaps did not resolve", "Coverage messages did not resolve"} {
		if strings.Contains(page, banned) {
			t.Errorf("a resolved read read as a failed one: found %q; body: %s", banned, page)
		}
	}
}

func TestCoverageRendersAGapOfAnUnexplainedCause(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"probe-failed","reason":"the control probe did not complete"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "198.51.100.10:443 tcp") {
		t.Errorf("a reach Gap of an unexplained cause never reached the ledger; body: %s", page)
	}
	if strings.Contains(page, "No gaps this batch") {
		t.Errorf("Coverage claimed no gaps while it held one; body: %s", page)
	}
	if strings.Contains(page, "proxy edge") {
		t.Errorf("a Gap that was never blanketed took the proxy-edge prose; body: %s", page)
	}
}

func TestReachGapsAndMessagesKeepARowForAnUnexplainedCause(t *testing.T) {
	rows := []db.ListOpenReachGapServicesRow{
		{SubjectKey: "104.21.61.6:443/tcp", Value: []byte(`{"outcome":"gap","cause":"` + blanketdiscrim.GapCause + `"}`)},
		{SubjectKey: "198.51.100.9:443/tcp", Value: []byte(`{"outcome":"gap","cause":"vantage-unavailable"}`)},
		{SubjectKey: "198.51.100.10:443/tcp", Value: []byte(`{"outcome":"gap","cause":"probe-failed","reason":"the control probe did not complete"}`)},
		{SubjectKey: "198.51.100.11:8443/tcp", Value: []byte(`{"outcome":"gap"}`)},
	}
	gaps, msgs := reachGapsAndMessages(rows)

	subjects := map[string]coverageGapView{}
	for _, g := range gaps {
		subjects[g.Subject] = g
	}
	if len(gaps) != 3 {
		t.Fatalf("want a row for the blanketed address and both unexplained causes, got %d: %+v", len(gaps), gaps)
	}
	for _, banned := range []string{"198.51.100.9", "198.51.100.9:443 tcp"} {
		if _, ok := subjects[banned]; ok {
			t.Error("a vantage-unavailable Gap is carried by its own message, so it must not double up in the ledger")
		}
	}
	for _, subject := range []string{"198.51.100.10:443 tcp", "198.51.100.11:8443 tcp"} {
		row, ok := subjects[subject]
		if !ok {
			t.Fatalf("a reach Gap of an unexplained cause must name its own service in the ledger; got %+v", gaps)
		}
		if strings.Contains(row.Gap, "origin") || strings.Contains(row.Expected, "proxy edge") {
			t.Errorf("an unexplained cause took the proxy-edge prose: %+v", row)
		}
	}
	if blanket := subjects["104.21.61.6"]; blanket.Expected != "origin behind the proxy edge" {
		t.Errorf("the blanketed row lost its own prose: %+v", blanket)
	}

	for _, m := range msgs {
		if m.Subject != "104.21.61.6" && strings.Contains(m.Text, "proxy edge") {
			t.Errorf("the proxy-edge explanation reached a Gap that was never blanketed: %+v", m)
		}
	}
	if !strings.Contains(messageFor(msgs, "198.51.100.10:443 tcp").Text, "the control probe did not complete") {
		t.Errorf("the recorded reason is the model's own operator prose and must survive: %+v", msgs)
	}
}

func TestReachGapsAndMessagesDoNotLetABlanketedAddressSwallowAnotherPort(t *testing.T) {
	rows := []db.ListOpenReachGapServicesRow{
		{SubjectKey: "104.21.61.6:443/tcp", Value: []byte(`{"outcome":"gap","cause":"` + blanketdiscrim.GapCause + `"}`)},
		{SubjectKey: "104.21.61.6:8443/tcp", Value: []byte(`{"outcome":"gap","cause":"probe-failed","reason":"the control probe did not complete"}`)},
	}
	gaps, msgs := reachGapsAndMessages(rows)

	if len(gaps) != 2 {
		t.Fatalf("the blanketed address and the unexplained service are two findings, got %d: %+v", len(gaps), gaps)
	}
	if gaps[0].Subject != "104.21.61.6" || gaps[1].Subject != "104.21.61.6:8443 tcp" {
		t.Errorf("one gap swallowed the other: %+v", gaps)
	}
	if !strings.Contains(messageFor(msgs, "104.21.61.6:8443 tcp").Text, "the control probe did not complete") {
		t.Errorf("the second service's recorded reason never rendered: %+v", msgs)
	}
}

func TestCoverageDoesNotClaimNoGapsWhileAnOutageGapStandsOpen(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "local", Class: "internet", Resolver: "127.0.0.11:53",
		Availability: pgtype.Text{String: "unavailable", Valid: true},
	})
	f.vantageNextID = 2
	for _, svc := range []string{"198.51.100.9:443/tcp", "198.51.100.10:443/tcp"} {
		f.addClassReachability(t, svc, "internet", obsClock, unavailableGap)
	}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if strings.Contains(page, "No gaps this batch") {
		t.Errorf("Coverage claimed no gaps while two outage Gaps stood open (#2180); body: %s", page)
	}
	if !strings.Contains(page, "2 services") {
		t.Errorf("the outage row must count the services beneath the dark vantage (#2180); body: %s", page)
	}
}

type outageReadStore struct {
	coldStore
	err error
}

func (o outageReadStore) ListOutageReachGapVantages(context.Context) ([]db.ListOutageReachGapVantagesRow, error) {
	return nil, o.err
}

func TestCoverageNamesAFailedOutageGapRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	buf := captureLog(t)

	srv := newServer(f, testKey, "", fixedClock())
	srv.useTranscriptKey(testTranscriptKey)
	srv.coldStore = outageReadStore{coldStore: f, err: errors.New("list outage reach gap vantages: connection reset")}
	ledger := srv.readCoverageGapLedger(t.Context())

	if !ledger.GapsFailed {
		t.Error("a failed outage read must raise GapsFailed, not render an empty gap ledger (#2180)")
	}
	if !strings.Contains(buf.String(), "connection reset") {
		t.Errorf("a degradation the operator cannot see must log; log: %q", buf.String())
	}
}

func messageFor(msgs []coverageMessageView, subject string) coverageMessageView {
	for _, m := range msgs {
		if m.Subject == subject {
			return m
		}
	}
	return coverageMessageView{}
}

func TestCoverageSaysARecoveredVantagesOutageGapIsPending(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "local", Class: "internet", Resolver: "127.0.0.11:53",
		Availability: pgtype.Text{String: "available", Valid: true},
	})
	f.vantageNextID = 2
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock, unavailableGap)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "1 service, pending") {
		t.Errorf("a recovered vantage's outage row must say the reading is pending (#2189); body: %s", page)
	}
	if !strings.Contains(page, "This position has recovered") {
		t.Errorf("Coverage must tell a recovered position from one we cannot look from (#2189); body: %s", page)
	}
}

func TestOutageGapViewsLeaveADarkVantagesRowUnqualified(t *testing.T) {
	gaps, msgs := outageGapViews([]db.ListOutageReachGapVantagesRow{
		{Vantage: "dark", Services: 2, Recovered: false},
		{Vantage: "back", Services: 1, Recovered: true},
	})

	if len(gaps) != 2 {
		t.Fatalf("both positions still hold an open Gap, so both take a row (#2189): %+v", gaps)
	}
	if gaps[0].Expected != "a reach reading for 2 services" {
		t.Errorf("a position we still cannot look from must not read as pending (#2189): %+v", gaps[0])
	}
	if gaps[1].Expected != "a reach reading for 1 service, pending" {
		t.Errorf("a recovered position's row must say the reading is pending (#2189): %+v", gaps[1])
	}
	if gaps[0].Gap != "outage" || gaps[1].Gap != "outage" {
		t.Errorf("the badge names the cause the span recorded, for both (#2180): %+v", gaps)
	}
	if len(msgs) != 1 || msgs[0].Subject != "vantage back" {
		t.Fatalf("only the recovered position earns the pending message (#2189): %+v", msgs)
	}
}
