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
	gaps, msgs, _ := reachGapsAndMessages(rows)

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
	gaps, msgs, _ := reachGapsAndMessages(rows)

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

func TestOutageGapViewsQualifiesEveryAvailabilityState(t *testing.T) {
	gaps, msgs := outageGapViews([]db.ListOutageReachGapVantagesRow{
		{Vantage: "dark", Services: 2, Availability: "unavailable"},
		{Vantage: "back", Services: 1, Availability: "available"},
		{Vantage: "fresh", Services: 3, Availability: "pending"},
		{Vantage: "proberless", Services: 4, Availability: "unknown"},
	}, map[string]bool{"vantage dark": true})

	if len(gaps) != 4 {
		t.Fatalf("every position still holds an open Gap, so every state takes a row (#2254): %+v", gaps)
	}
	want := map[string]string{
		"vantage dark":       "a reach reading for 2 services, not evaluable",
		"vantage back":       "a reach reading for 1 service, pending",
		"vantage fresh":      "a reach reading for 3 services, unconfirmed",
		"vantage proberless": "a reach reading for 4 services, unconfirmed",
	}
	for _, g := range gaps {
		if g.Gap != "outage" {
			t.Errorf("the badge names the cause the span recorded, for every state (#2180): %+v", g)
		}
		// A dark position must not read as pending, and neither fall-through state may read bare.
		if g.Expected != want[g.Subject] {
			t.Errorf("%s reads %q, want %q (#2254)", g.Subject, g.Expected, want[g.Subject])
		}
	}
	if m := messageFor(msgs, "vantage dark"); m.Text != "" {
		t.Errorf("unavailableVantageMessages still names this position, so its richer message "+
			"owns the card and the row writes none (#2254, #2363): %+v", m)
	}
	if m := messageFor(msgs, "vantage back"); !strings.Contains(m.Text, "has recovered") {
		t.Errorf("a recovered position's row must say the reading is pending (#2189): %+v", m)
	}
	if m := messageFor(msgs, "vantage fresh"); !strings.Contains(m.Text, "no pinned host key") {
		t.Errorf("a pending position must name its state rather than render bare (#2254): %+v", m)
	}
	if m := messageFor(msgs, "vantage proberless"); !strings.Contains(m.Text, "no availability at all") {
		t.Errorf("a position with no availability must name that rather than render bare (#2254): %+v", m)
	}
}

func TestOutageGapViewsReturnsARowForEveryStateMix(t *testing.T) {
	states := []string{"available", "unavailable", "pending", "unknown"}
	for mix := 0; mix < 1<<len(states); mix++ {
		var rows []db.ListOutageReachGapVantagesRow
		for i, s := range states {
			if mix&(1<<i) != 0 {
				rows = append(rows, db.ListOutageReachGapVantagesRow{Vantage: s, Services: 1, Availability: s})
			}
		}
		gaps, _ := outageGapViews(rows, nil)
		if len(gaps) != len(rows) {
			t.Fatalf("mix %04b: read %d rows and returned %d gap rows (#2254)", mix, len(rows), len(gaps))
		}
	}
}

func TestCoverageQualifiesAPendingVantagesOutageGap(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "local", Class: "internet", Resolver: "127.0.0.11:53",
		Availability: pgtype.Text{String: "pending", Valid: true},
	})
	f.vantageNextID = 2
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock, unavailableGap)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "1 service, unconfirmed") {
		t.Errorf("a pending position's outage row must name the state, not render bare (#2254); body: %s", page)
	}
	if !strings.Contains(page, "no pinned host key") {
		t.Errorf("a pending position must read as neither recovered nor dark (#2254); body: %s", page)
	}
}

func TestOutageGapViewsMessagesADarkPositionTheSilentReadMissed(t *testing.T) {
	// No snapshot spans the two reads, so the row must not delegate blind: here the silent read
	// names no position and the row alone knows this one is dark (#2363).
	gaps, msgs := outageGapViews([]db.ListOutageReachGapVantagesRow{
		{Vantage: "dark", Services: 2, Availability: "unavailable"},
	}, map[string]bool{})

	if len(gaps) != 1 || gaps[0].Expected != "a reach reading for 2 services, not evaluable" {
		t.Fatalf("a dark position's row must qualify its own state, never read bare (#2363): %+v", gaps)
	}
	m := messageFor(msgs, "vantage dark")
	if !strings.Contains(m.Text, "not evaluable") {
		t.Errorf("the row must write its own message where the silent read no longer names the "+
			"position, or it renders with nothing beside it (#2254, #2363): %+v", msgs)
	}
	if m.Badge != "outage" || m.Kind != "gap" {
		t.Errorf("the fallback message belongs to the outage row, so it carries that row's badge: %+v", m)
	}
}

func TestCoverageGivesADarkPositionOneMessageUnderTheRowsSubject(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "dark", Class: "internet", Resolver: "127.0.0.11:53",
		Availability: pgtype.Text{String: "unavailable", Valid: true},
	})
	f.vantageNextID = 2
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock, unavailableGap)

	srv := newServer(f, testKey, "", fixedClock())
	ledger := srv.readCoverageGapLedger(t.Context())

	if len(ledger.Gaps) != 1 || ledger.Gaps[0].Subject != "vantage dark" {
		t.Fatalf("the outage row must name the position whose outage opened the Gap (#2180): %+v", ledger.Gaps)
	}
	var subjects []string
	for _, m := range ledger.Messages {
		subjects = append(subjects, m.Subject)
	}
	// One position takes one message: both reads name it, so the richer one stands alone (#2363).
	if len(subjects) != 1 || subjects[0] != ledger.Gaps[0].Subject {
		t.Fatalf("the message card and the Gaps table must agree on the subject, and neither read "+
			"may double it (#2363); row %q, messages %v", ledger.Gaps[0].Subject, subjects)
	}
	if !strings.Contains(ledger.Messages[0].Text, "127.0.0.11:53") {
		t.Errorf("the surviving message must be the one naming the resolver (#2363): %+v", ledger.Messages[0])
	}
}

type ledgerRaceStore struct {
	coldStore
	outage []db.ListOutageReachGapVantagesRow
	silent []db.ListUnavailableVantagesRow
}

func (s ledgerRaceStore) ListOutageReachGapVantages(context.Context) ([]db.ListOutageReachGapVantagesRow, error) {
	return s.outage, nil
}

func (s ledgerRaceStore) ListUnavailableVantages(context.Context) ([]db.ListUnavailableVantagesRow, error) {
	return s.silent, nil
}

func raceLedger(t *testing.T, outage []db.ListOutageReachGapVantagesRow, silent []db.ListUnavailableVantagesRow) coverageGapLedger {
	t.Helper()
	f := newFakeStore()
	srv := newServer(f, testKey, "", fixedClock())
	srv.coldStore = ledgerRaceStore{coldStore: f, outage: outage, silent: silent}
	return srv.readCoverageGapLedger(t.Context())
}

func TestCoverageMessagesADarkOutageRowTheVantageReadNoLongerNames(t *testing.T) {
	// The two reads disagree: the silent read names no position, and the outage read projects
	// one as dark. No snapshot spans them, so the row cannot delegate its message and wait.
	ledger := raceLedger(t,
		[]db.ListOutageReachGapVantagesRow{{Vantage: "dark", Services: 2, Availability: "unavailable"}},
		nil)

	if len(ledger.Gaps) != 1 {
		t.Fatalf("the outage row stands whatever the other read says (#2147): %+v", ledger.Gaps)
	}
	if m := messageFor(ledger.Messages, ledger.Gaps[0].Subject); m.Text == "" {
		t.Errorf("a dark position the silent read does not name leaves its outage row with "+
			"nothing beside it, which is the failure #2254 closed (#2363): %+v", ledger)
	}
}

func TestCoverageDoesNotContradictItselfWhenAPositionRecoversBetweenTheReads(t *testing.T) {
	// The mirror window: the silent read still named the position, and the later outage read
	// projects the recovery. Both would otherwise write a message under the one subject.
	ledger := raceLedger(t,
		[]db.ListOutageReachGapVantagesRow{{Vantage: "dark", Services: 2, Availability: "available"}},
		[]db.ListUnavailableVantagesRow{{Name: "dark", Resolver: "127.0.0.11:53"}})

	if len(ledger.Messages) != 1 {
		t.Fatalf("one position took %d messages, so the card states a recovery and an outage at "+
			"once (#2363): %+v", len(ledger.Messages), ledger.Messages)
	}
	if !strings.Contains(ledger.Messages[0].Text, "has recovered") {
		t.Errorf("the later of the two reads saw the recovery, so its message is the one that "+
			"stands (#2363): %+v", ledger.Messages[0])
	}
}

func TestEveryOutageRowTakesExactlyOneMessageUnderItsOwnSubject(t *testing.T) {
	// The invariant the two un-snapshotted reads must hold between them: whichever of the two
	// names the position, and whichever state the row projects, the row takes one message.
	for _, state := range []string{"available", "unavailable", "pending", "unknown", "surprise"} {
		for _, stillDark := range []bool{false, true} {
			var silent []db.ListUnavailableVantagesRow
			if stillDark {
				silent = []db.ListUnavailableVantagesRow{{Name: "p", Resolver: "127.0.0.11:53"}}
			}
			ledger := raceLedger(t,
				[]db.ListOutageReachGapVantagesRow{{Vantage: "p", Services: 1, Availability: state}},
				silent)

			if len(ledger.Gaps) != 1 {
				t.Fatalf("%s/%v: one row read must render one row (#2254): %+v", state, stillDark, ledger.Gaps)
			}
			if len(ledger.Messages) != 1 {
				t.Fatalf("%s/%v: the row took %d messages, want 1 (#2363): %+v",
					state, stillDark, len(ledger.Messages), ledger.Messages)
			}
			if ledger.Messages[0].Subject != ledger.Gaps[0].Subject {
				t.Errorf("%s/%v: message %q sits under a subject its Gap row does not use (#2363)",
					state, stillDark, ledger.Messages[0].Subject)
			}
		}
	}
}

func TestCoverageUnavailableVantageMessageCarriesTheGapsTableSubject(t *testing.T) {
	msgs := unavailableVantageMessages([]db.ListUnavailableVantagesRow{{Name: "dark", Resolver: "127.0.0.11:53"}})
	gaps, _ := outageGapViews([]db.ListOutageReachGapVantagesRow{
		{Vantage: "dark", Services: 1, Availability: "unavailable"},
	}, nil)

	if len(msgs) != 1 || len(gaps) != 1 {
		t.Fatalf("one position must take one row and one message: gaps %+v messages %+v", gaps, msgs)
	}
	// sortCoverageMessages orders on Subject, so a disagreement also splits the two apart (#2363).
	if msgs[0].Subject != gaps[0].Subject {
		t.Errorf("the message reads %q and its Gap row reads %q, so a reader scanning the card "+
			"for the row's subject finds nothing (#2363)", msgs[0].Subject, gaps[0].Subject)
	}
}
