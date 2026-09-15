package main

import (
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
	gaps, msgs := reachGapsAndMessages(rows)
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
	gapsA, msgsA := reachGapsAndMessages(forward)
	gapsB, msgsB := reachGapsAndMessages(back)

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
