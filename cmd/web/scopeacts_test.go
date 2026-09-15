package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

// A fold that reads an Act would call this, and the count is what the assertion reads.

type countingScopeActStore struct{ calls int }

func (c *countingScopeActStore) ListActsOfClassSince(context.Context, db.ListActsOfClassSinceParams) ([]db.Act, error) {
	c.calls++
	return []db.Act{}, nil
}

func exposureBoardFixture(t *testing.T, f *fakeStore, at time.Time) db.Account {
	t.Helper()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.vantages = append(f.vantages, db.Vantage{
		ID: f.vantageNextID, Name: "internet-prober", Class: "internet",
		Host:        pgtype.Text{String: "prober.example.com", Valid: true},
		DialledAddr: classPresentedDialled("internet"),
		CreatedBy:   pgtype.Int8{Int64: admin.ID, Valid: true},
	})
	f.vantageNextID++

	const svc = "198.51.100.10:443/tcp"
	f.addClassReachability(t, svc, "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", at, `{"outcome":"reached"}`)
	return admin
}

func declareScopeAct(t *testing.T, f *fakeStore, at time.Time, who, scope string) {
	t.Helper()
	recordActAt(t, f, at,
		act.Account{AccountID: 1, UsernameSnapshot: who},
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: scope}})
}

// The silence is stated where the figure moved, and the panel names no cause (ADR-1946 §3).

func TestExposurePanelStatesTheSilenceAndListsAddressScopes(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)
	declareScopeAct(t, f, now.Add(-4*time.Minute), "alice", "192.0.2.0/24")

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	for _, want := range []string{
		"Recent address scopes",
		"A declaration fires no message",
		"192.0.2.0/24",
		"@alice",
		">4m<",
		`title="2026-09-11T11:56:00Z"`,
		`href="/settings?tab=audit&amp;period=90d"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the scope panel is missing %q", want)
		}
	}
	// A causal claim is the one thing the panel may not make, because no tie exists (ADR-1946 §4).
	for _, refused := range []string{"changed this", "caused", "because you"} {
		if strings.Contains(page, refused) {
			t.Errorf("the scope panel claims a tie with %q", refused)
		}
	}
}

func TestExposurePanelDropsANameScopeAndOldAndSurplusRows(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	declareScopeAct(t, f, now.Add(-time.Minute), "alice", "acmecorp.io")
	declareScopeAct(t, f, now.Add(-8*24*time.Hour), "alice", "203.0.113.0/24")
	for i := range 6 {
		declareScopeAct(t, f, now.Add(-time.Duration(i+1)*time.Hour), "alice",
			fmt.Sprintf("10.%d.0.0/16", i))
	}

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if strings.Contains(page, "acmecorp.io") {
		t.Error("the panel listed a name scope, which moves no Vantage class")
	}
	if strings.Contains(page, "203.0.113.0/24") {
		t.Error("the panel listed a scope declared outside the seven-day window")
	}
	// Five newest-first, so the sixth hour is the one that falls off.
	for i := range 5 {
		if !strings.Contains(page, fmt.Sprintf("10.%d.0.0/16", i)) {
			t.Errorf("the panel dropped 10.%d.0.0/16, which is inside the five newest", i)
		}
	}
	if strings.Contains(page, "10.5.0.0/16") {
		t.Error("the panel listed a sixth row past its bound")
	}
}

// The corpus refuses a viewer at the audit tab, and the panel is no way around that gate.

func TestExposurePanelRefusesAViewer(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	declareScopeAct(t, f, now.Add(-4*time.Minute), "alice", "192.0.2.0/24")

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "viewer", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if !strings.Contains(page, "Service exposure") {
		t.Fatal("a viewer lost the board itself, which no gate asks for")
	}
	for _, refused := range []string{"Recent address scopes", "192.0.2.0/24", "@alice"} {
		if strings.Contains(page, refused) {
			t.Errorf("a viewer was served %q from the Act corpus", refused)
		}
	}
}

// An empty list would read as no scope was declared, which is a claim the failed read cannot make.

func TestExposurePanelNotesAReadThatDidNotResolve(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)
	f.actListErr = errors.New("act read down")

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if !strings.Contains(page, "did not resolve") {
		t.Error("the panel rendered no note for a read that failed")
	}
	if strings.Contains(page, "named an address scope") {
		t.Error("a failed read rendered the empty state, which claims the corpus is empty")
	}
	if !strings.Contains(page, "Service exposure") {
		t.Error("a failed panel read took the board with it")
	}
}

// No derivation may read an Act, so the fold is blind to the corpus (ADR-1946 §5).

func TestExposureFoldReadsNoActInput(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	spy := &countingScopeActStore{}
	s := &server{exposureStore: f, vantageClassStore: f, scopeActStore: spy, now: func() time.Time { return now }}
	req := httptest.NewRequest(http.MethodGet, "/exposure", nil)

	before, statsBefore, err := s.foldExposure(req)
	if err != nil {
		t.Fatalf("fold before the acts: %v", err)
	}

	declareScopeAct(t, f, now.Add(-time.Minute), "alice", "198.51.100.0/24")
	declareScopeAct(t, f, now.Add(-2*time.Minute), "alice", "10.200.0.0/16")

	after, statsAfter, err := s.foldExposure(req)
	if err != nil {
		t.Fatalf("fold after the acts: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Errorf("the Act corpus moved a fold row: before %+v, after %+v", before, after)
	}
	if statsBefore != statsAfter {
		t.Errorf("the Act corpus moved a count: before %+v, after %+v", statsBefore, statsAfter)
	}
	if spy.calls != 0 {
		t.Errorf("the fold reached the Act reader %d times, want 0", spy.calls)
	}
}

// A name-scope burst fills the read, and the empty panel may not then claim no scope was declared.

func TestExposurePanelNamesACappedRead(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)
	for i := range int(scopeActReadCap) + 4 {
		declareScopeAct(t, f, now.Add(-time.Duration(i+1)*time.Minute), "alice",
			fmt.Sprintf("host-%d.acmecorp.io", i))
	}

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if !strings.Contains(page, "No address scope is among the 50 newest seed declarations") {
		t.Error("a capped read claimed more than it read")
	}
	if strings.Contains(page, "named an address scope") {
		t.Error("a capped read rendered the uncapped empty state")
	}
}

// The scope that withholds the board is the one the reader most needs, so the panel survives it.

func TestExposurePanelRendersUnderAWithheldBoard(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	declareScopeAct(t, f, now.Add(-4*time.Minute), "alice", "192.0.2.0/24")

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if !strings.Contains(page, "Exposure withheld.") {
		t.Fatal("the fixture did not withhold the board")
	}
	for _, want := range []string{"Recent address scopes", "192.0.2.0/24"} {
		if !strings.Contains(page, want) {
			t.Errorf("the withheld board dropped %q, which is the act that can withhold it", want)
		}
	}
}
