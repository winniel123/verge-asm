package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

const (
	rpName  = "www.example.com"
	rpOther = "api.example.com"
	rpNew   = "203.0.113.9"
	rpOld   = "198.51.100.1"
	rpBatch = int64(61)
)

func resolved(addrs ...string) []byte {
	if len(addrs) == 0 {
		return []byte(`{"outcome":"Resolved","addresses":[]}`)
	}
	return []byte(`{"outcome":"Resolved","addresses":["` + strings.Join(addrs, `","`) + `"]}`)
}

func rePointMove(name string, prev, next []byte) spanChange {
	return spanChange{SubjectKind: "name", SubjectKey: name, Facet: "resolution", Value: next, Previous: prev}
}

func openedBeneath(name, addr string, port string) []spanChange {
	svc := addr + ":" + port + "/tcp"
	return []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
		{SubjectKind: "endpoint", SubjectKey: name + "@" + svc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
	}
}

func rePointChanges(addr string, prev []byte) []spanChange {
	return append([]spanChange{rePointMove(rpName, prev, resolved(addr))}, openedBeneath(rpName, addr, "443")...)
}

func citer(name string) db.ListResolutionCitersForAddressesRow {
	return db.ListResolutionCitersForAddressesRow{SubjectKey: name}
}

func rePointFrom(t *testing.T, store *fakeMessageStore, changes []spanChange, in membershipInputs) []*message.Message {
	t.Helper()
	return rePointHeld(t, store, changes, in, true)
}

func rePointHeld(t *testing.T, store *fakeMessageStore, changes []spanChange, in membershipInputs, hold bool) []*message.Message {
	t.Helper()
	moves := rePoints(changes)
	citers, err := readResolutionCiters(context.Background(), store, nil, moves)
	if err != nil {
		t.Fatalf("citers: %v", err)
	}
	return rePointMessages(rpBatch, produceT0, changes, moves, in, citers, hold)
}

func byKind(msgs []*message.Message) map[string][]*message.Message {
	out := map[string][]*message.Message{}
	for _, m := range msgs {
		out[m.SubjectKind] = append(out[m.SubjectKind], m)
	}
	return out
}

func censusKinds(m *message.Message) map[string]int {
	out := map[string]int{}
	for _, e := range m.Census.Entries {
		out[e.Kind]++
	}
	return out
}

func heldBasis(t *testing.T, m *message.Message) message.CensusBasis {
	t.Helper()
	if m.CensusPending == nil {
		t.Fatalf("a membership root owes a held row (ADR-1806 §2), got %+v", m)
	}
	return m.CensusPending.Basis
}

func TestRePointToANewAddressFiresAddressAppeared(t *testing.T) {
	store := &fakeMessageStore{}
	msgs := rePointFrom(t, store, rePointChanges(rpNew, resolved(rpOld)), membershipInputs{})
	got := byKind(msgs)
	if len(got["address"]) != 1 || len(got["name"]) != 0 {
		t.Fatalf("a new address is the root, and the fold writes no residue (#1818); got %+v", msgs)
	}
	m := got["address"][0]
	if m.FiredAt != rpNew || m.Cause != message.CauseDrift || m.Class != message.ClassDrift {
		t.Errorf("fires appeared, drift, at the Address; got %+v", m)
	}
	if m.Headline != rpNew+" entered the estate" {
		t.Errorf("headline = %q, want the cause clause the release appends its count to", m.Headline)
	}
}

func TestTheAddressRootHoldsItsCensusForTheAdmittingTier(t *testing.T) {
	// A dns fold enters the address and opens nothing beneath it, so a census read here would
	// tell the operator that nothing is there (ADR-1806 §2, #1774).
	store := &fakeMessageStore{}
	changes := []spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	if len(msgs) != 1 {
		t.Fatalf("the address is new ground, so it roots once; got %+v", msgs)
	}
	m := msgs[0]
	if m.Census != nil {
		t.Errorf("a held row carries no census until release, got %+v", m.Census)
	}
	if b := heldBasis(t, m); b.RootKind != subjectKindAddress || b.RootKey != rpNew {
		t.Errorf("the basis names the Address the move cited, got %+v", b)
	}
	// An Address cites no address of its own, so the frozen value is the root key alone (§4).
	if b := heldBasis(t, m); len(b.RootValue) != 0 {
		t.Errorf("an Address root freezes no resolution value, got %s", b.RootValue)
	}
	if m.CensusPending.AfterBatch != rpBatch {
		t.Errorf("the release reads what opened since the move's own batch, got %d", m.CensusPending.AfterBatch)
	}
}

func TestWithNoReaperTheAddressRootWritesItsCensusAtTheCause(t *testing.T) {
	// The drain test reads a job set nothing reaps, so one wedged row would hold every message
	// forever. The census is written at the cause instead (ADR-1806 §6 row 1).
	store := &fakeMessageStore{}
	changes := []spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}
	msgs := rePointHeld(t, store, changes, membershipInputs{}, false)
	if len(msgs) != 1 {
		t.Fatalf("the root fires on either configuration, got %+v", msgs)
	}
	m := msgs[0]
	if m.CensusPending != nil {
		t.Errorf("this configuration holds nothing, got %+v", m.CensusPending)
	}
	// The dns fold that moved the Name opened nothing, which is the price §6 row 1 accepts.
	if m.Census == nil || m.Census.Len() != 0 {
		t.Errorf("the census is this fold's own, and this fold opened nothing; got %+v", m.Census)
	}
	if m.Headline != rpNew+" entered the estate · 0 timelines opened beneath it" {
		t.Errorf("headline = %q, want the count the degraded arm can reach", m.Headline)
	}
}

func TestRePointOntoAKnownAddressRootsNothingInTheFold(t *testing.T) {
	// ADR-0026 §2's residue is the poll's, so the fold writes neither root nor residue (#1818).
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpOther)}}}
	msgs := rePointFrom(t, store, rePointChanges(rpNew, resolved(rpOld)), membershipInputs{})
	if len(msgs) != 0 {
		t.Fatalf("an address another Name cites is no root (ADR-0026 §2), got %+v", msgs)
	}
}

func TestRePointTwoNamesOneAddressInOneFoldRootsOnce(t *testing.T) {
	changes := append(rePointChanges(rpNew, resolved(rpOld)), rePointMove(rpOther, resolved(rpOld), resolved(rpNew)))
	changes = append(changes, openedBeneath(rpOther, rpNew, "443")...)
	// Both new spans are open when the fold reads, so each sees the other as a citer.
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpName), citer(rpOther)}}}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	got := byKind(msgs)
	if len(got["address"]) != 1 || len(got["name"]) != 0 {
		t.Fatalf("one new address is one root however many Names reached it, got %+v", msgs)
	}
	if b := heldBasis(t, got["address"][0]); b.RootKey != rpNew {
		t.Errorf("the one root is the one address both Names reached, got %+v", b)
	}
}

func TestRePointStandingCitationAtAnotherVantageIsNotNew(t *testing.T) {
	// The same Name's timeline at vantage 2 has cited the address for weeks.
	other := citer(rpName)
	other.VantageID = pgtype.Int8{Int64: 2, Valid: true}
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpName), other}}}
	msgs := rePointFrom(t, store, rePointChanges(rpNew, resolved(rpOld)), membershipInputs{})
	if len(msgs) != 0 {
		t.Fatalf("only the moved timeline is dropped, so a sibling vantage keeps the address in the estate; got %+v", msgs)
	}
}

func TestRePointSwapWithinOneFoldIsNotNewGround(t *testing.T) {
	changes := append(rePointChanges(rpNew, resolved(rpOld)), rePointMove(rpOther, resolved(rpNew), resolved(rpOld)))
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpName)}}}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	if len(msgs) != 0 {
		t.Fatalf("an address a sibling just dropped was in the estate, so it is no root; got %+v", msgs)
	}
}

func TestRePointIntoADeclaredScopeIsNoRoot(t *testing.T) {
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("203.0.113.0/24")}}
	msgs := rePointFrom(t, store, rePointChanges(rpNew, resolved(rpOld)), in)
	if len(msgs) != 0 {
		t.Fatalf("a Seed-covered address never appears (ADR-0047), so the fold roots nothing; got %+v", msgs)
	}
}

func TestRePointIntoAnExclusionIsNoRoot(t *testing.T) {
	store := &fakeMessageStore{}
	in := membershipInputs{exclusions: []db.Exclusion{addressExclusion("203.0.113.0/24")}}
	msgs := rePointFrom(t, store, []spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}, in)
	if len(msgs) != 0 {
		t.Fatalf("an excluded address is refused ground and nothing opened beneath it, got %+v", msgs)
	}
}

func TestRePointLeavesAGapCloseToCoverage(t *testing.T) {
	changes := rePointChanges(rpNew, nil)
	changes[0].PrevIsGap = true
	changes[0].BeforeGap = resolved(rpOld)
	store := &fakeMessageStore{}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	if len(msgs) != 0 || len(store.citersAsked) != 0 {
		t.Fatalf("a Gap-closing edge is coverage by construction and gapclose carries it (ADR-0014); got %+v", msgs)
	}
}

func TestRePointReCitedAddressIsNotNew(t *testing.T) {
	store := &fakeMessageStore{}
	msgs := rePointFrom(t, store, rePointChanges(rpNew, resolved(rpNew, rpOld)), membershipInputs{})
	if len(msgs) != 0 {
		t.Fatalf("the closed span already cited it, so nothing beneath it is the move's consequence; got %+v", msgs)
	}
	if len(store.citersAsked) != 0 {
		t.Errorf("no candidate means no estate read, got %v", store.citersAsked)
	}
}

func TestRePointShrinkingToNoDataFiresNothing(t *testing.T) {
	changes := []spanChange{rePointMove(rpName, resolved(rpOld), []byte(`{"outcome":"NoData"}`))}
	changes = append(changes, openedBeneath(rpName, rpOld, "8443")...)
	store := &fakeMessageStore{}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	if len(msgs) != 0 || len(store.citersAsked) != 0 {
		t.Fatalf("the shrinking direction is silent (ADR-0026 §2), got %+v", msgs)
	}
}

func TestRePointIgnoresAResolutionOpening(t *testing.T) {
	changes := rePointChanges(rpNew, nil)
	changes[0].Opened = true
	store := &fakeMessageStore{}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	if len(msgs) != 0 || len(store.citersAsked) != 0 {
		t.Fatalf("an opening roots on the Name and membership carries it (ADR-0031), got %+v", msgs)
	}
}

func TestRePointReadsTheEstateOnceForEveryCandidate(t *testing.T) {
	const third = "203.0.113.10"
	changes := append(rePointChanges(rpNew, resolved(rpOld)), rePointMove(rpOther, resolved(rpOld), resolved(third)))
	store := &fakeMessageStore{}
	rePointFrom(t, store, changes, membershipInputs{})
	if len(store.citersAsked) != 1 || strings.Join(store.citersAsked[0], ",") != third+","+rpNew {
		t.Errorf("one batched read of every candidate, sorted; got %v", store.citersAsked)
	}
}

func TestProduceHoldsAnAddressAppearedAndRoutesItNowhere(t *testing.T) {
	store := &fakeMessageStore{prev: prevAt(produceT0.Add(-time.Hour))}
	var log []routed
	if err := produceMessages(context.Background(), store, 31, produceT0, rePointChanges(rpNew, resolved(rpOld)), nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	var found []db.InsertMessageParams
	for _, p := range store.inserted {
		if p.SubjectKind == "address" {
			found = append(found, p)
		}
	}
	if len(found) != 1 {
		t.Fatalf("one new address is one root, got %+v", store.inserted)
	}
	if found[0].FiredAt != rpNew || found[0].Class != string(message.ClassDrift) {
		t.Errorf("fires drift at the Address, got %+v", found[0])
	}
	if found[0].CensusPendingAfterBatch.Int64 != 31 || len(found[0].CensusBasis) == 0 {
		t.Errorf("the row is held on its own batch and owes its basis, got %+v", found[0])
	}
	// The held row is this fold's only message, so an empty log is the whole claim (ADR-1806 §2).
	if len(log) != 0 {
		t.Errorf("the release poll alone routes a held row, got %+v", log)
	}
}
