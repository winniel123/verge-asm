package queue

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

type uncitedStore struct {
	rows    []db.ListCitedAddressSpansForNamesRow
	beneath []db.ListOpenSpansBeneathAddressesRow

	askedNames []string
	askedAddrs []string
	closed     map[int64]string
}

func (s *uncitedStore) ListCitedAddressSpansForNames(_ context.Context, names []string) ([]db.ListCitedAddressSpansForNamesRow, error) {
	s.askedNames = names
	return s.rows, nil
}

func (s *uncitedStore) ListOpenSpansBeneathAddresses(_ context.Context, addrs []string) ([]db.ListOpenSpansBeneathAddressesRow, error) {
	s.askedAddrs = addrs
	return s.beneath, nil
}

func (s *uncitedStore) CloseSpan(_ context.Context, p db.CloseSpanParams) error {
	if s.closed == nil {
		s.closed = map[int64]string{}
	}
	s.closed[p.ID] = p.ClosureReason.String
	return nil
}

func TestCloseUncitedAddressesSoleCiterTakesAddressAndServices(t *testing.T) {
	// `a.example.com` was the only Name citing 203.0.113.9,
	// and its spans have already closed (#1689).
	store := &uncitedStore{
		rows: []db.ListCitedAddressSpansForNamesRow{{ID: 10, SubjectKey: "203.0.113.9"}},
		beneath: []db.ListOpenSpansBeneathAddressesRow{
			{ID: 20, SubjectKind: "service", SubjectKey: "203.0.113.9:443/tcp"},
			{ID: 21, SubjectKind: "endpoint", SubjectKey: "www.example.com@203.0.113.9:443/tcp"},
			{ID: 22, SubjectKind: "service", SubjectKey: "203.0.113.90:443/tcp"},
		},
	}
	var deps []departure
	err := closeUncitedAddresses(context.Background(), store, 7, time.Unix(1_700_000_000, 0), []string{"a.example.com"}, membershipInputs{}, &deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(store.askedNames) != 1 || store.askedNames[0] != "a.example.com" {
		t.Fatalf("candidate read asked for %v, want the departed Name alone", store.askedNames)
	}
	if len(store.askedAddrs) != 1 || store.askedAddrs[0] != "203.0.113.9" {
		t.Fatalf("descent asked for %v, want [203.0.113.9]", store.askedAddrs)
	}
	for _, id := range []int64{10, 20, 21} {
		if got := store.closed[id]; got != string(drift.ReasonUncited) {
			t.Errorf("span %d closed %q, want %q", id, got, drift.ReasonUncited)
		}
	}
	if _, closed := store.closed[22]; closed {
		t.Error("span 22 sits beneath 203.0.113.90, not the uncited Address, and must stay open")
	}
	if len(deps) != 3 {
		t.Fatalf("departures = %+v, want the Address, the Service and the Endpoint", deps)
	}
	if deps[0].SubjectKind != subjectKindAddress || deps[0].SubjectKey != "203.0.113.9" || deps[0].Reason != string(drift.ReasonUncited) || deps[0].Timelines != 1 {
		t.Errorf("address departure = %+v", deps[0])
	}
}

func TestCloseUncitedAddressesSharedCiterStaysOpen(t *testing.T) {
	store := &uncitedStore{
		rows: []db.ListCitedAddressSpansForNamesRow{{ID: 11, SubjectKey: "203.0.113.10", Citers: []string{"c.example.com"}}},
		beneath: []db.ListOpenSpansBeneathAddressesRow{
			{ID: 30, SubjectKind: "service", SubjectKey: "203.0.113.10:443/tcp"},
		},
	}
	var deps []departure
	if err := closeUncitedAddresses(context.Background(), store, 7, time.Now(), []string{"b.example.com"}, membershipInputs{}, &deps); err != nil {
		t.Fatal(err)
	}
	if len(store.closed) != 0 {
		t.Fatalf("closed %v, want nothing: a second live citer keeps the Address present", store.closed)
	}
	if store.askedAddrs != nil {
		t.Errorf("descent ran for %v, want no descent when no Address left", store.askedAddrs)
	}
	if len(deps) != 0 {
		t.Errorf("departures = %+v, want none", deps)
	}
}

func TestCloseUncitedAddressesSeedCoveredStaysOpen(t *testing.T) {
	// Presence is citation OR Seed cover, so a declared address outlives its last citer (ADR-0047).
	store := &uncitedStore{
		rows: []db.ListCitedAddressSpansForNamesRow{{ID: 12, SubjectKey: "198.51.100.7"}},
	}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}
	if err := closeUncitedAddresses(context.Background(), store, 7, time.Now(), []string{"a.example.com"}, in, nil); err != nil {
		t.Fatal(err)
	}
	if len(store.closed) != 0 {
		t.Fatalf("closed %v, want nothing", store.closed)
	}
}

func TestCloseUncitedAddressesNoDepartureReadsNothing(t *testing.T) {
	store := &uncitedStore{rows: []db.ListCitedAddressSpansForNamesRow{{ID: 13, SubjectKey: "203.0.113.9"}}}
	if err := closeUncitedAddresses(context.Background(), store, 7, time.Now(), nil, membershipInputs{}, nil); err != nil {
		t.Fatal(err)
	}
	if store.askedNames != nil || len(store.closed) != 0 {
		t.Fatalf("a batch with no Name departure must not read or close: asked %v, closed %v", store.askedNames, store.closed)
	}
}

func TestCloseUncitedAddressesReResolvedNameLeavesItsOldAddress(t *testing.T) {
	// `a.example.com` stays open but now resolves to 203.0.113.10 alone, so the
	// candidate read returns its pre-move Address with no live citer (#1706).
	store := &uncitedStore{
		rows: []db.ListCitedAddressSpansForNamesRow{
			{ID: 40, SubjectKey: "203.0.113.10", Citers: []string{"a.example.com"}},
			{ID: 41, SubjectKey: "203.0.113.9"},
		},
		beneath: []db.ListOpenSpansBeneathAddressesRow{
			{ID: 50, SubjectKind: "service", SubjectKey: "203.0.113.9:443/tcp"},
		},
	}
	var deps []departure
	if err := closeUncitedAddresses(context.Background(), store, 7, time.Now(), []string{"a.example.com"}, membershipInputs{}, &deps); err != nil {
		t.Fatal(err)
	}
	if len(store.askedAddrs) != 1 || store.askedAddrs[0] != "203.0.113.9" {
		t.Fatalf("descent asked for %v, want the pre-move Address alone", store.askedAddrs)
	}
	for _, id := range []int64{41, 50} {
		if got := store.closed[id]; got != string(drift.ReasonUncited) {
			t.Errorf("span %d closed %q, want %q", id, got, drift.ReasonUncited)
		}
	}
	if _, closed := store.closed[40]; closed {
		t.Error("the Address the Name now resolves to is still cited and must stay open")
	}
	if len(deps) != 2 {
		t.Fatalf("departures = %+v, want the old Address and the Service beneath it", deps)
	}
}

func TestUncitedCitersTakesDepartedAndDecidedNames(t *testing.T) {
	obs := []wire.Observation{
		resolutionObservation("moved.example.com", "A", `{"outcome":"Resolved","addresses":["203.0.113.10"]}`),
		resolutionObservation("gone.example.com", "A", `{"outcome":"NameError"}`),
		resolutionObservation("outage.example.com", "A", `{"outcome":"Gap"}`),
		resolutionObservation("outage.example.com", "AAAA", `{"outcome":"Resolved","addresses":["2001:db8::1"]}`),
		resolutionObservation("moved.example.com", "AAAA", `{"outcome":"Resolved"}`),
		{Facet: "reachability", Subject: "203.0.113.10:443/tcp", Data: json.RawMessage(`{"outcome":"Open"}`)},
	}
	got := uncitedCiters(obs, []string{"gone.example.com"})
	want := []string{"gone.example.com", "moved.example.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("uncitedCiters = %v, want %v", got, want)
	}
}

func TestUncitedCitersEmptyBatchReadsNothing(t *testing.T) {
	if got := uncitedCiters(nil, nil); got != nil {
		t.Fatalf("uncitedCiters = %v, want nil", got)
	}
	obs := []wire.Observation{resolutionObservation("outage.example.com", "A", `{"outcome":"Gap"}`)}
	if got := uncitedCiters(obs, nil); got != nil {
		t.Fatalf("a batch of Gaps re-decides no citation, got %v", got)
	}
}

func resolutionObservation(subject, qtype, data string) wire.Observation {
	return wire.Observation{
		Facet:         resolutionwalk.FacetResolution,
		Subject:       subject,
		Discriminator: qtype,
		Data:          json.RawMessage(data),
	}
}
