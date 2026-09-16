package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
)

func TestCoverageGapExpectedNamesTheProxyEdge(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "104.21.61.6:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "origin behind the proxy edge") {
		t.Errorf("the Coverage gap row must expect an origin behind the proxy edge; body: %s", page)
	}
	if strings.Contains(page, "origin behind the edge") {
		t.Errorf("bare \"edge\" is reserved for Exposure's edge-only; body: %s", page)
	}
}

func TestReachGapsAndMessagesReadTheGapCause(t *testing.T) {
	rows := []db.ListOpenReachGapServicesRow{
		{SubjectKey: "104.21.61.6:443/tcp", Value: []byte(`{"outcome":"gap","cause":"` + blanketdiscrim.GapCause + `"}`)},
		{SubjectKey: "198.51.100.9:443/tcp", Value: []byte(`{"outcome":"gap","cause":"vantage-unavailable"}`)},
	}
	gaps, msgs, _ := reachGapsAndMessages(rows)
	if len(gaps) != 1 || len(msgs) != 1 {
		t.Fatalf("one blanket-responder row must yield one gap and one message, got %d and %d", len(gaps), len(msgs))
	}
	if gaps[0].Subject != "104.21.61.6" {
		t.Errorf("gap subject = %q, want the blanketed address", gaps[0].Subject)
	}
	if msgs[0].Subject != "104.21.61.6" {
		t.Errorf("message subject = %q, want the blanketed address", msgs[0].Subject)
	}
}

func TestCoverageOmitsProxyEdgeProseForAnotherGapCause(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "198.51.100.9:443/tcp", "internet", obsClock,
		`{"outcome":"gap","cause":"vantage-unavailable","reason":"we could not look from this position"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := coverageBody(t, ac, base)
	if strings.Contains(page, "proxy edge") {
		t.Errorf("a reach Gap of another cause must not render as a proxy edge; body: %s", page)
	}
	if strings.Contains(page, "198.51.100.9") {
		t.Errorf("a reach Gap of another cause must not reach the blanket-responder rows; body: %s", page)
	}
}

func TestOneServicesGapMessageIsTheSameWhateverOrderTheRowsArrive(t *testing.T) {
	const svc = "198.51.100.9:443/tcp"
	// Two vantages can hold an open Gap on one service and record different reasons. The page
	// states one message per service, so the row that survives the dedupe must be settled by
	// the rows themselves and never by the order the database happened to return them.
	gap := func(reason string) db.ListOpenReachGapServicesRow {
		return db.ListOpenReachGapServicesRow{
			SubjectKey: svc,
			Value:      []byte(`{"outcome":"gap","cause":"probe-timeout","reason":"` + reason + `"}`),
		}
	}
	first, second := gap("a resolver answered no query"), gap("z the dial never completed")

	forward, back := []db.ListOpenReachGapServicesRow{first, second}, []db.ListOpenReachGapServicesRow{second, first}
	gapsA, msgsA, _ := reachGapsAndMessages(forward)
	gapsB, msgsB, _ := reachGapsAndMessages(back)

	if len(gapsA) != 1 || len(msgsA) != 1 || len(gapsB) != 1 || len(msgsB) != 1 {
		t.Fatalf("one service must state one gap and one message, got %d/%d and %d/%d",
			len(gapsA), len(msgsA), len(gapsB), len(msgsB))
	}
	if msgsA[0].Text != msgsB[0].Text {
		t.Errorf("the message text varies with the row order: %q then %q", msgsA[0].Text, msgsB[0].Text)
	}
	if !strings.Contains(msgsA[0].Text, "a resolver answered no query") {
		t.Errorf("the surviving row is the one whose recorded value sorts first; got %q", msgsA[0].Text)
	}
}

func TestABlanketedAddressTakesNoServiceLevelGapRow(t *testing.T) {
	const svc = "104.21.61.6:443/tcp"
	// Two vantages can hold an open Gap on one service and disagree on the cause. The blanket
	// row raises the finding to the address, so a "no reading" row for the same service would
	// state that address a second time (#2184).
	rows := []db.ListOpenReachGapServicesRow{
		{SubjectKey: svc, Value: []byte(`{"outcome":"gap","cause":"` + blanketdiscrim.GapCause + `"}`)},
		{SubjectKey: svc, Value: []byte(`{"outcome":"gap","cause":"probe-timeout","reason":"the dial never completed"}`)},
	}
	gaps, msgs, _ := reachGapsAndMessages(rows)

	if len(gaps) != 1 || len(msgs) != 1 {
		t.Fatalf("a blanketed address states one gap and one message, got %d and %d: %+v", len(gaps), len(msgs), gaps)
	}
	if gaps[0].Subject != "104.21.61.6" || gaps[0].Gap != "no origin" {
		t.Errorf("gap = %q/%q, want the blanketed address and \"no origin\"", gaps[0].Subject, gaps[0].Gap)
	}
	if msgs[0].Subject != "104.21.61.6" {
		t.Errorf("message subject = %q, want the blanketed address", msgs[0].Subject)
	}
}

func TestSilencingTheBlanketedServiceLeavesAnotherPortOfItsAddressStanding(t *testing.T) {
	// Dedupe on the service key, never on the address: an address-wide skip would take the
	// third row too, which TestReachGapsAndMessagesDoNotLetABlanketedAddressSwallowAnotherPort
	// forbids (#2184).
	rows := []db.ListOpenReachGapServicesRow{
		{SubjectKey: "104.21.61.6:443/tcp", Value: []byte(`{"outcome":"gap","cause":"` + blanketdiscrim.GapCause + `"}`)},
		{SubjectKey: "104.21.61.6:443/tcp", Value: []byte(`{"outcome":"gap","cause":"probe-timeout","reason":"the dial never completed"}`)},
		{SubjectKey: "104.21.61.6:8080/tcp", Value: []byte(`{"outcome":"gap","cause":"probe-timeout","reason":"the dial never completed"}`)},
	}
	gaps, msgs, _ := reachGapsAndMessages(rows)

	if len(gaps) != 2 || len(msgs) != 2 {
		t.Fatalf("want the address row and the unblanketed port, got %d gaps and %d messages: %+v", len(gaps), len(msgs), gaps)
	}
	want := []string{"104.21.61.6", serviceCopyKey("104.21.61.6", "8080", "tcp")}
	for i, subject := range want {
		if gaps[i].Subject != subject {
			t.Errorf("gap %d subject = %q, want %q", i, gaps[i].Subject, subject)
		}
	}
}

func noReadingRows(n int) []db.ListOpenReachGapServicesRow {
	rows := make([]db.ListOpenReachGapServicesRow, 0, n)
	for i := range n {
		rows = append(rows, db.ListOpenReachGapServicesRow{
			SubjectKey: fmt.Sprintf("198.51.100.%d:443/tcp", i),
			Value:      []byte(`{"outcome":"gap","cause":"probe-timeout","reason":"the dial never completed"}`),
		})
	}
	return rows
}

func TestReachGapsAndMessagesBoundTheNoReadingList(t *testing.T) {
	const extra = 7
	// A wide connect failure Gaps every service at once, and each drew a row and a block (#2181).
	gaps, msgs, omitted := reachGapsAndMessages(noReadingRows(coverageGapListCap + extra))

	if len(gaps) != coverageGapListCap || len(msgs) != coverageGapListCap {
		t.Fatalf("got %d gaps and %d messages, want %d of each — the list is unbounded",
			len(gaps), len(msgs), coverageGapListCap)
	}
	if omitted != extra {
		t.Errorf("omitted = %d, want %d — the page must state the remainder it did not list", omitted, extra)
	}
}

func TestReachGapsAndMessagesOmitNothingAtExactlyTheCap(t *testing.T) {
	// A flag set by len == cap claims a truncation that did not happen, and the reader then
	// hunts for a service that is already on screen (#2222).
	gaps, _, omitted := reachGapsAndMessages(noReadingRows(coverageGapListCap))

	if len(gaps) != coverageGapListCap {
		t.Fatalf("got %d gaps, want every one of the %d services", len(gaps), coverageGapListCap)
	}
	if omitted != 0 {
		t.Errorf("omitted = %d, want 0 — a window holding exactly the cap lost nothing", omitted)
	}
}

func TestReachGapsAndMessagesCountOneServiceOnceAgainstTheCap(t *testing.T) {
	// Two vantages can hold an open Gap on one service. The cap bounds what the page draws,
	// so it must count the rendered service, never the row the dedupe threw away (#2181).
	rows := noReadingRows(coverageGapListCap)
	gaps, _, omitted := reachGapsAndMessages(append(rows, rows...))

	if len(gaps) != coverageGapListCap || omitted != 0 {
		t.Errorf("got %d gaps and %d omitted, want %d and 0 — a duplicate row consumed the cap",
			len(gaps), omitted, coverageGapListCap)
	}
}

func TestCoverageStatesTheNoReadingRemainderItDidNotList(t *testing.T) {
	for _, tc := range []struct {
		name    string
		extra   int
		wantEnd string
	}{
		{name: "several", extra: 3, wantEnd: "3 more are not listed."},
		{name: "one", extra: 1, wantEnd: "1 more is not listed."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			for i := range coverageGapListCap + tc.extra {
				f.addClassReachability(t, fmt.Sprintf("198.51.100.%d:443/tcp", i), "internet", obsClock,
					`{"outcome":"gap","cause":"probe-timeout","reason":"the dial never completed"}`)
			}
			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")

			page := coverageBody(t, ac, base)
			want := fmt.Sprintf("Showing the first %d services with no reading. %s", coverageGapListCap, tc.wantEnd)
			if n := strings.Count(page, want); n != 2 {
				t.Errorf("the Gaps table and the messages card each state the remainder %q %d times, want 2; body: %s",
					want, n, page)
			}
		})
	}
}
