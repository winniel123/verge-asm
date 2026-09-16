package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

// A fold that reads an Act would call this, and the count is what the assertion reads.

type countingScopeActStore struct {
	calls int
	args  db.ListActsOfClassesSinceParams
}

func (c *countingScopeActStore) ListActsOfClassesSince(_ context.Context, arg db.ListActsOfClassesSinceParams) ([]db.Act, error) {
	c.calls++
	c.args = arg
	return []db.Act{}, nil
}

// One read covers every class, so a class added to the panel costs no extra round trip (#2167).

func TestExposurePanelReadsEveryScopeActClassInOneQuery(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	spy := &countingScopeActStore{}
	s := &server{scopeActStore: spy, now: func() time.Time { return now }}

	if _, _, err := s.recentAddressScopeActs(context.Background()); err != nil {
		t.Fatalf("recentAddressScopeActs: %v", err)
	}
	if spy.calls != 1 {
		t.Errorf("the panel issued %d act reads, want 1", spy.calls)
	}
	want := []string{
		act.ExclusionDeclared{}.Class(),
		act.ExclusionLifted{}.Class(),
		act.ProposalConfirmed{}.Class(),
		act.ProposalDeclineUndone{}.Class(),
		act.ProposalDeclined{}.Class(),
		act.SeedDeclared{}.Class(),
		act.SeedWithdrawn{}.Class(),
	}
	got := slices.Clone(spy.args.Actions)
	slices.Sort(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the one read asked for %v, want %v", got, want)
	}
	if spy.args.MaxActs != scopeActReadCap {
		t.Errorf("MaxActs = %d, want %d: the cap bounds the whole read", spy.args.MaxActs, scopeActReadCap)
	}
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

// The panel covers every class that moves addressScopeCovered, and each renders its own verb,
// because a withdrawal and a declaration are different facts (ADR-2114, #2072).

func TestExposurePanelCoversEveryClassThatMovesCovered(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	who := act.Account{AccountID: 1, UsernameSnapshot: "alice"}
	// Newest first, so the panel lists them in this order.
	recordActAt(t, f, now.Add(-1*time.Minute), who,
		act.ExclusionLifted{ExclusionRef: act.ExclusionRef{Kind: "address", Scope: "192.0.2.128/25"}})
	recordActAt(t, f, now.Add(-2*time.Minute), who,
		act.ExclusionDeclared{ExclusionRef: act.ExclusionRef{Kind: "address", Scope: "192.0.2.64/26"}})
	recordActAt(t, f, now.Add(-3*time.Minute), who,
		act.ProposalConfirmed{SeedScope: act.SeedScope{Scope: "203.0.113.0/24"}})
	recordActAt(t, f, now.Add(-4*time.Minute), who,
		act.SeedWithdrawn{SeedScope: act.SeedScope{Scope: "10.42.0.0/16"}})
	recordActAt(t, f, now.Add(-5*time.Minute), who,
		act.SeedDeclared{SeedScope: act.SeedScope{Scope: "198.51.100.0/24"}})
	// A name-kind exclusion moves no address scope, so it stays out (ADR-2114 §4).
	recordActAt(t, f, now.Add(-30*time.Second), who,
		act.ExclusionDeclared{ExclusionRef: act.ExclusionRef{Kind: "name", Scope: "acmecorp.io"}})

	s := &server{scopeActStore: f, now: func() time.Time { return now }}
	rows, capped, err := s.recentAddressScopeActs(context.Background())
	if err != nil {
		t.Fatalf("recentAddressScopeActs: %v", err)
	}
	if capped {
		t.Error("six acts filled no 50-row read")
	}
	want := []scopeActRow{
		{Scope: "192.0.2.128/25", Verb: "exclusion lifted"},
		{Scope: "192.0.2.64/26", Verb: "excluded"},
		{Scope: "203.0.113.0/24", Verb: "confirmed"},
		{Scope: "10.42.0.0/16", Verb: "withdrawn"},
		{Scope: "198.51.100.0/24", Verb: "declared"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %d, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		if rows[i].Scope != w.Scope || rows[i].Verb != w.Verb {
			t.Errorf("row %d = %q/%q, want %q/%q", i, rows[i].Scope, rows[i].Verb, w.Scope, w.Verb)
		}
	}
}

// Declining an address proposal writes an address exclusion, and un-declining removes it, so both
// move addressScopeCovered and both render (ADR-2171, #2169).

func TestExposurePanelCoversADeclinedAddressProposal(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	who := act.Account{AccountID: 1, UsernameSnapshot: "alice"}
	recordActAt(t, f, now.Add(-1*time.Minute), who,
		act.ProposalDeclineUndone{ExclusionRef: act.ExclusionRef{Kind: "address", Scope: "203.0.113.0/24"}})
	recordActAt(t, f, now.Add(-2*time.Minute), who,
		act.ProposalDeclined{ExclusionRef: act.ExclusionRef{Kind: "address", Scope: "192.0.2.0/24"}})
	// A name-kind decline reaches no address scope, so it stays out (ADR-2114 §4, ADR-2171 §6).
	recordActAt(t, f, now.Add(-30*time.Second), who,
		act.ProposalDeclined{ExclusionRef: act.ExclusionRef{Kind: "name", Scope: "acmecorp.io"}})

	s := &server{scopeActStore: f, now: func() time.Time { return now }}
	rows, _, err := s.recentAddressScopeActs(context.Background())
	if err != nil {
		t.Fatalf("recentAddressScopeActs: %v", err)
	}
	want := []scopeActRow{
		{Scope: "203.0.113.0/24", Verb: "decline lifted"},
		{Scope: "192.0.2.0/24", Verb: "declined"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %d, want %d: %+v", len(rows), len(want), rows)
	}
	for i, w := range want {
		if rows[i].Scope != w.Scope || rows[i].Verb != w.Verb {
			t.Errorf("row %d = %q/%q, want %q/%q", i, rows[i].Scope, rows[i].Verb, w.Scope, w.Verb)
		}
	}
}

// ADR-2114 named a principle and then a list, and the list went stale (#2169). A scope payload is
// all this guard sees, so a class that moves the predicate without one stays open (ADR-2171 §6).

func TestEveryScopeShapedActClassIsInThePanel(t *testing.T) {
	const addr = "192.0.2.0/24"
	seedScope := reflect.TypeOf(act.SeedScope{})
	exclusionRef := reflect.TypeOf(act.ExclusionRef{})

	for _, a := range act.Classes {
		filled := reflect.New(reflect.TypeOf(a)).Elem()
		var carries bool
		for i := range filled.NumField() {
			switch filled.Field(i).Type() {
			case seedScope:
				filled.Field(i).Set(reflect.ValueOf(act.SeedScope{Scope: addr}))
				carries = true
			case exclusionRef:
				filled.Field(i).Set(reflect.ValueOf(act.ExclusionRef{Kind: "address", Scope: addr}))
				carries = true
			}
		}
		if !carries {
			continue
		}
		if _, ok := addressScopeActVerbs[a.Class()]; !ok {
			t.Errorf("%s carries an address-capable scope payload, so it moves addressScopeCovered, "+
				"and the panel does not read it (ADR-2114, ADR-2171)", a.Class())
			continue
		}
		scope, isAddress := addressScopeOf(filled.Interface().(act.Act))
		if !isAddress || scope != addr {
			t.Errorf("%s is in the read set, and addressScopeOf returned %q/%t: the switch has no "+
				"case for it, so every row it returns is dropped", a.Class(), scope, isAddress)
		}
	}
	for class := range addressScopeActVerbs {
		if !slices.ContainsFunc(act.Classes, func(a act.Act) bool { return a.Class() == class }) {
			t.Errorf("the panel reads %q, which no act class records", class)
		}
	}
}

// The five-row render cap holds over every class the read covers, not over one class (ADR-2114).

func TestExposurePanelCapsTheReadAtFiveRows(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	who := act.Account{AccountID: 1, UsernameSnapshot: "alice"}
	for i := range 4 {
		recordActAt(t, f, now.Add(-time.Duration(i+1)*time.Hour), who,
			act.SeedDeclared{SeedScope: act.SeedScope{Scope: fmt.Sprintf("10.%d.0.0/16", i)}})
		recordActAt(t, f, now.Add(-time.Duration(i+1)*time.Hour-time.Minute), who,
			act.SeedWithdrawn{SeedScope: act.SeedScope{Scope: fmt.Sprintf("172.%d.0.0/16", i)}})
	}

	s := &server{scopeActStore: f, now: func() time.Time { return now }}
	rows, _, err := s.recentAddressScopeActs(context.Background())
	if err != nil {
		t.Fatalf("recentAddressScopeActs: %v", err)
	}
	if len(rows) != scopeActRows {
		t.Fatalf("rows = %d, want %d", len(rows), scopeActRows)
	}
	want := []string{"10.0.0.0/16", "172.0.0.0/16", "10.1.0.0/16", "172.1.0.0/16", "10.2.0.0/16"}
	for i, w := range want {
		if rows[i].Scope != w {
			t.Errorf("row %d = %q, want %q; the read did not return newest-first", i, rows[i].Scope, w)
		}
	}
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
		"Recent address-scope edits",
		"An edit fires no message",
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
	for _, refused := range []string{"Recent address-scope edits", "192.0.2.0/24", "@alice"} {
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
	ctx := context.Background()
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		t.Fatalf("addressScopeCovered: %v", err)
	}

	before, statsBefore, err := s.foldExposureUnder(ctx, covered)
	if err != nil {
		t.Fatalf("fold before the acts: %v", err)
	}

	declareScopeAct(t, f, now.Add(-time.Minute), "alice", "198.51.100.0/24")
	declareScopeAct(t, f, now.Add(-2*time.Minute), "alice", "10.200.0.0/16")

	after, statsAfter, err := s.foldExposureUnder(ctx, covered)
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

// #2167 made one LIMIT bound every class at once, so a bulk class can evict the others (#2221).
// declineLookup records one act per checked proposal and ListPendingProposals has no LIMIT, so
// proposal.declined is the class that can do it.

func TestABulkDeclineDoesNotEvictEveryOtherScopeActClass2221(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	who := act.Account{AccountID: 1, UsernameSnapshot: "alice"}
	// The address scope is the oldest act, so only a cap wide enough to hold the burst reaches it.
	recordActAt(t, f, now.Add(-6*time.Hour), who,
		act.ExclusionDeclared{ExclusionRef: act.ExclusionRef{Kind: "address", Scope: "198.51.100.7/32"}})

	// Name scopes carry no address scope, so they render nothing and only consume the read.
	const burst = 200
	for i := range burst {
		declareScopeAct(t, f, now.Add(-time.Duration(i+1)*time.Minute), "alice",
			fmt.Sprintf("host-%d.acmecorp.io", i))
	}

	if int(scopeActReadCap) <= burst {
		t.Fatalf("scopeActReadCap = %d, want more than the %d-act burst: one class would evict the rest",
			scopeActReadCap, burst)
	}

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if !strings.Contains(page, "198.51.100.7/32") {
		t.Error("a bulk decline evicted the address scope, so the panel hid an act the cap should reach")
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

	if !strings.Contains(page, fmt.Sprintf("No address scope is among the %d newest scope acts", scopeActReadCap)) {
		t.Error("a capped read claimed more than it read")
	}
	if strings.Contains(page, "named an address scope") {
		t.Error("a capped read rendered the uncapped empty state")
	}
}

// A read that returned its whole LIMIT may hide a newer address scope behind a name-scope burst,
// and filling the five render rows from another class says nothing about what it hid (#2188).

func TestACappedReadSurvivesAFullRender(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	who := act.Account{AccountID: 1, UsernameSnapshot: "alice"}
	// Name scopes take every slot the five render rows leave, so the read returns its whole LIMIT.
	for i := range int(scopeActReadCap) - scopeActRows {
		declareScopeAct(t, f, now.Add(-time.Duration(i+1)*time.Minute), "alice",
			fmt.Sprintf("host-%d.acmecorp.io", i))
	}
	// Six, so five fill the render and the sixth is the address scope the LIMIT really drops.
	for i := range scopeActRows + 1 {
		recordActAt(t, f, now.Add(-time.Duration(i+1)*time.Hour), who,
			act.ExclusionDeclared{ExclusionRef: act.ExclusionRef{
				Kind: "address", Scope: fmt.Sprintf("192.0.2.%d/32", i)}})
	}

	s := &server{scopeActStore: f, now: func() time.Time { return now }}
	rows, capped, err := s.recentAddressScopeActs(context.Background())
	if err != nil {
		t.Fatalf("recentAddressScopeActs: %v", err)
	}
	if len(rows) != scopeActRows {
		t.Fatalf("rows = %d, want %d: the render must fill or this proves nothing", len(rows), scopeActRows)
	}
	if !capped {
		t.Error("a filled render reported an uncapped read: a read returning its LIMIT means " +
			"there may be more, and five rendered rows is a different fact (#2188)")
	}
}

func TestExposurePanelNamesACappedReadBesideFiveRows(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	exposureBoardFixture(t, f, now)

	who := act.Account{AccountID: 1, UsernameSnapshot: "alice"}
	for i := range int(scopeActReadCap) - scopeActRows {
		declareScopeAct(t, f, now.Add(-time.Duration(i+1)*time.Minute), "alice",
			fmt.Sprintf("host-%d.acmecorp.io", i))
	}
	for i := range scopeActRows + 1 {
		recordActAt(t, f, now.Add(-time.Duration(i+1)*time.Hour), who,
			act.ExclusionDeclared{ExclusionRef: act.ExclusionRef{
				Kind: "address", Scope: fmt.Sprintf("192.0.2.%d/32", i)}})
	}

	base := startAt(t, f, now)
	page := getBody(t, login(t, base, "admin", "hunter2hunter2"), base+"/exposure", http.StatusOK)

	if !strings.Contains(page, fmt.Sprintf("This read stopped at the %d newest scope acts", scopeActReadCap)) {
		t.Error("a full list claimed a completeness its capped read cannot support, and the " +
			"operator is told nothing (#2188)")
	}
	if !strings.Contains(page, "192.0.2.0/32") {
		t.Fatal("the fixture did not render the five address rows the note sits beside")
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
	for _, want := range []string{"Recent address-scope edits", "192.0.2.0/24"} {
		if !strings.Contains(page, want) {
			t.Errorf("the withheld board dropped %q, which is the act that can withhold it", want)
		}
	}
}
