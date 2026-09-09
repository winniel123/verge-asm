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

func TestRePointToANewAddressFiresAddressAppeared(t *testing.T) {
	store := &fakeMessageStore{}
	msgs, err := rePointMessages(context.Background(), store, produceT0, rePointChanges(rpNew, resolved(rpOld)), membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["address"]) != 1 || len(got["name"]) != 0 {
		t.Fatalf("a new address is the root and the residue is empty, got %+v", msgs)
	}
	m := got["address"][0]
	if m.FiredAt != rpNew || m.Cause != message.CauseDrift || m.Class != message.ClassDrift {
		t.Errorf("fires appeared, drift, at the Address; got %+v", m)
	}
	if k := censusKinds(m); k["service"] != 1 || k["endpoint"] != 1 {
		t.Errorf("the census carries the Service and the Endpoint that opened beneath, got %v", k)
	}
	if !strings.HasPrefix(m.Headline, rpNew+" entered the estate") {
		t.Errorf("headline = %q", m.Headline)
	}
}

func TestRePointOntoAKnownAddressFiresTheNameResidue(t *testing.T) {
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpOther)}}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, rePointChanges(rpNew, resolved(rpOld)), membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["address"]) != 0 || len(got["name"]) != 1 {
		t.Fatalf("an address another Name cites is no root, and the Endpoint is residue (ADR-0026 §2); got %+v", msgs)
	}
	m := got["name"][0]
	if m.FiredAt != rpName || censusKinds(m)["endpoint"] != 1 || m.Census.Len() != 1 {
		t.Errorf("the residue is exactly the Endpoint beneath the known address, got %+v", m)
	}
	if !strings.Contains(m.Headline, "re-pointed within the estate") {
		t.Errorf("headline = %q", m.Headline)
	}
}

func TestRePointTwoNamesOneAddressInOneFoldRootsOnce(t *testing.T) {
	changes := append(rePointChanges(rpNew, resolved(rpOld)), rePointMove(rpOther, resolved(rpOld), resolved(rpNew)))
	changes = append(changes, openedBeneath(rpOther, rpNew, "443")...)
	// Both new spans are open when the fold reads, so each sees the other as a citer.
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpName), citer(rpOther)}}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["address"]) != 1 || len(got["name"]) != 0 {
		t.Fatalf("one new address is one root however many Names reached it, got %+v", msgs)
	}
	if k := censusKinds(got["address"][0]); k["endpoint"] != 2 || k["service"] != 1 {
		t.Errorf("the one census carries both Endpoints and the one Service, got %v", k)
	}
}

func TestRePointStandingCitationAtAnotherVantageIsNotNew(t *testing.T) {
	// The same Name's timeline at vantage 2 has cited the address for weeks.
	other := citer(rpName)
	other.VantageID = pgtype.Int8{Int64: 2, Valid: true}
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpName), other}}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, rePointChanges(rpNew, resolved(rpOld)), membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["address"]) != 0 || len(got["name"]) != 1 {
		t.Fatalf("only the moved timeline is dropped, so a sibling vantage keeps the address in the estate; got %+v", msgs)
	}
}

func TestRePointSwapWithinOneFoldIsNotNewGround(t *testing.T) {
	changes := append(rePointChanges(rpNew, resolved(rpOld)), rePointMove(rpOther, resolved(rpNew), resolved(rpOld)))
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpName)}}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["address"]) != 0 || len(got["name"]) != 1 || got["name"][0].FiredAt != rpName {
		t.Fatalf("an address a sibling just dropped was in the estate, so the move is residue; got %+v", msgs)
	}
}

func TestRePointIntoADeclaredScopeIsNoRoot(t *testing.T) {
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("203.0.113.0/24")}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, rePointChanges(rpNew, resolved(rpOld)), in)
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["address"]) != 0 || len(got["name"]) != 1 {
		t.Fatalf("a Seed-covered address never appears (ADR-0047), so the Endpoint is residue; got %+v", msgs)
	}
}

func TestRePointIntoAnExclusionIsNoRoot(t *testing.T) {
	store := &fakeMessageStore{}
	in := membershipInputs{exclusions: []db.ListExclusionsRow{addressExclusion("203.0.113.0/24")}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, []spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}, in)
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("an excluded address is refused ground and nothing opened beneath it, got %+v", msgs)
	}
}

func TestRePointLeavesAGapCloseToCoverage(t *testing.T) {
	changes := rePointChanges(rpNew, nil)
	changes[0].PrevIsGap = true
	changes[0].BeforeGap = resolved(rpOld)
	store := &fakeMessageStore{}
	msgs, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	if len(msgs) != 0 || len(store.citersAsked) != 0 {
		t.Fatalf("a Gap-closing edge is coverage by construction and gapclose carries it (ADR-0014); got %+v", msgs)
	}
}

func TestRePointReCitedAddressIsNotNew(t *testing.T) {
	store := &fakeMessageStore{}
	msgs, err := rePointMessages(context.Background(), store, produceT0, rePointChanges(rpNew, resolved(rpNew, rpOld)), membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("the closed span already cited it, so nothing beneath it is the move's consequence; got %+v", msgs)
	}
	if len(store.citersAsked) != 0 {
		t.Errorf("no candidate means no estate read, got %v", store.citersAsked)
	}
}

func TestRePointResidueIsOnlyBeneathTheNewlyCitedAddresses(t *testing.T) {
	// The move keeps rpOld and adds rpNew, which another Name already cites.
	changes := []spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpOld, rpNew))}
	changes = append(changes, openedBeneath(rpName, rpOld, "8443")...)
	changes = append(changes, openedBeneath(rpName, rpNew, "443")...)
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpOther)}}}
	msgs, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	got := byKind(msgs)
	if len(got["name"]) != 1 || got["name"][0].Census.Len() != 1 {
		t.Fatalf("a new port on an address the Name already cited is not the move's consequence; got %+v", msgs)
	}
	if e := got["name"][0].Census.Entries[0]; !strings.Contains(e.Key, rpNew) {
		t.Errorf("the residue is the Endpoint beneath the newly cited address, got %+v", e)
	}
}

func TestRePointShrinkingToNoDataFiresNothing(t *testing.T) {
	changes := []spanChange{rePointMove(rpName, resolved(rpOld), []byte(`{"outcome":"NoData"}`))}
	changes = append(changes, openedBeneath(rpName, rpOld, "8443")...)
	store := &fakeMessageStore{}
	msgs, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	if len(msgs) != 0 || len(store.citersAsked) != 0 {
		t.Fatalf("the shrinking direction is silent (ADR-0026 §2), got %+v", msgs)
	}
}

func TestRePointIgnoresAResolutionOpening(t *testing.T) {
	changes := rePointChanges(rpNew, nil)
	changes[0].Opened = true
	store := &fakeMessageStore{}
	msgs, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{})
	if err != nil {
		t.Fatalf("repoint: %v", err)
	}
	if len(msgs) != 0 || len(store.citersAsked) != 0 {
		t.Fatalf("an opening roots on the Name and membership carries it (ADR-0031), got %+v", msgs)
	}
}

func TestRePointReadsTheEstateOnceForEveryCandidate(t *testing.T) {
	const third = "203.0.113.10"
	changes := append(rePointChanges(rpNew, resolved(rpOld)), rePointMove(rpOther, resolved(rpOld), resolved(third)))
	store := &fakeMessageStore{}
	if _, err := rePointMessages(context.Background(), store, produceT0, changes, membershipInputs{}); err != nil {
		t.Fatalf("repoint: %v", err)
	}
	if len(store.citersAsked) != 1 || strings.Join(store.citersAsked[0], ",") != third+","+rpNew {
		t.Errorf("one batched read of every candidate, sorted; got %v", store.citersAsked)
	}
}

func TestProduceWritesAnAddressAppearedAndRoutesItAsDrift(t *testing.T) {
	store := &fakeMessageStore{prev: prevAt(produceT0.Add(-time.Hour))}
	var log []routed
	if err := produceMessages(context.Background(), store, 31, produceT0, rePointChanges(rpNew, resolved(rpOld)), nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	var found *db.InsertMessageParams
	for i := range store.inserted {
		if store.inserted[i].SubjectKind == "address" {
			found = &store.inserted[i]
		}
	}
	if found == nil {
		t.Fatalf("no Address appeared written, got %+v", store.inserted)
	}
	if found.FiredAt != rpNew || found.Class != string(message.ClassDrift) {
		t.Errorf("fires drift at the Address, got %+v", found)
	}
	if len(log) == 0 || log[0].class != message.ClassDrift {
		t.Errorf("routed as drift, got %+v", log)
	}
}
