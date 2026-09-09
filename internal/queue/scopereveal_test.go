package queue

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

func darkOpening(svc string) spanChange {
	return spanChange{
		SubjectKind: "service", SubjectKey: svc, Facet: "reachability",
		Opened: true, OpenedAperture: true, Value: reachValue("not-reached"),
	}
}

func TestDeclaredAddressScopeFiresOneRevealedWithACount(t *testing.T) {
	changes := []spanChange{
		darkOpening("198.51.100.7:443/tcp"),
		darkOpening("198.51.100.7:80/tcp"),
		darkOpening("198.51.100.8:443/tcp"),
	}
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}
	var log []routed

	if err := produceMessages(context.Background(), store, 70, produceT0, changes, nil, nil, in, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("a declared address scope fires one message, never one per address, got %d: %+v", len(store.inserted), store.inserted)
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseAperture) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a widened aperture is an aperture / coverage firing, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "seed" || m.FiredAt != "198.51.100.0/24" {
		t.Errorf("the message fires at the Seed that declared the scope, got kind=%q fired=%q", m.SubjectKind, m.FiredAt)
	}
	c, err := message.ParseCensus(m.Census)
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if c.Len() != 3 {
		t.Errorf("the census counts the timelines opened, got %d: %+v", c.Len(), c.Entries)
	}
	for _, e := range c.Entries {
		if e.Kind == message.KindAddress {
			t.Errorf("the census carries a count, never a per-address list, got %+v", e)
		}
	}
	if !strings.Contains(m.Headline, "198.51.100.0/24") || !strings.Contains(m.Headline, "3 timelines") {
		t.Errorf("the headline states the scope and the count, got %q", m.Headline)
	}
	if strings.Contains(m.Headline, "came into view") || strings.Contains(m.Headline, "entered the estate") {
		t.Errorf("the scope wording must not read as a Name root's revealed, got %q", m.Headline)
	}
	if message.ContainsValence(m.Headline) {
		t.Errorf("the headline carries a valence word: %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassCoverage {
		t.Errorf("the message is routed on the coverage class, got %+v", log)
	}
}

func TestDeclaredAddressScopeSkipsAnOpeningANameCites(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "name", SubjectKey: "example.com", Facet: "resolution", Opened: true,
			Value: []byte(`{"outcome":"Resolved","addresses":["198.51.100.9"]}`)},
		darkOpening("198.51.100.9:443/tcp"),
		darkOpening("198.51.100.7:443/tcp"),
	}
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}

	msgs, err := buildMessages(context.Background(), store, 71, produceT0, changes, nil, nil, in)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var scope, root *message.Message
	for _, m := range msgs {
		switch m.SubjectKind {
		case "seed":
			scope = m
		case "name":
			root = m
		}
	}
	if root == nil || root.CensusLen() != 1 {
		t.Fatalf("the Name root still carries the Service it cites, got %+v", root)
	}
	if scope == nil {
		t.Fatal("the dark address still owes one message at the scope")
	}
	if scope.CensusLen() != 1 {
		t.Fatalf("a Name-cited opening is the root's, never the scope's, got %+v", scope.Census.Entries)
	}
	if scope.Census.Entries[0].Key != "198.51.100.7:443/tcp" {
		t.Errorf("the scope counts only what no Name cites, got %+v", scope.Census.Entries)
	}
}

func TestAnExcludedAddressFiresNoScopeReveal(t *testing.T) {
	changes := []spanChange{darkOpening("198.51.100.7:443/tcp")}
	store := &fakeMessageStore{}
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.0/28")},
	}

	msgs, err := buildMessages(context.Background(), store, 73, produceT0, changes, nil, nil, in)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, m := range msgs {
		if m.SubjectKind == "seed" {
			t.Fatalf("an operator who narrowed the range is owed no aperture message, got %+v", m)
		}
	}
}

func TestAnExclusionNarrowsTheScopeRevealCensus(t *testing.T) {
	changes := []spanChange{
		darkOpening("198.51.100.7:443/tcp"),
		darkOpening("198.51.100.200:443/tcp"),
	}
	store := &fakeMessageStore{}
	in := membershipInputs{
		seeds:      []db.ListSeedsRow{addressSeed("198.51.100.0/24")},
		exclusions: []db.ListExclusionsRow{addressExclusion("198.51.100.0/28")},
	}

	msgs, err := buildMessages(context.Background(), store, 74, produceT0, changes, nil, nil, in)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	var scope *message.Message
	for _, m := range msgs {
		if m.SubjectKind == "seed" {
			scope = m
		}
	}
	if scope == nil {
		t.Fatal("the address outside the exclusion still owes one message at the scope")
	}
	if scope.CensusLen() != 1 {
		t.Fatalf("only the unexcluded opening counts, got %+v", scope.Census.Entries)
	}
}

func TestDeclaredAddressScopeSkipsAnOpeningAnOpenSpanCites(t *testing.T) {
	changes := []spanChange{darkOpening("198.51.100.7:8443/tcp")}
	store := &fakeMessageStore{
		citers: map[string][]db.ListResolutionCitersForAddressesRow{"198.51.100.7": {citer("example.com")}},
	}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}

	msgs, err := buildMessages(context.Background(), store, 75, produceT0, changes, nil, nil, in)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, m := range msgs {
		if m.SubjectKind == "seed" {
			t.Fatalf("a Name cites the address through an open span, so the ground is not dark, got %+v", m)
		}
	}
	if len(store.citersAsked) != 1 {
		t.Fatalf("the producer reads the citers once for the fold, got %d reads: %+v", len(store.citersAsked), store.citersAsked)
	}
}

func TestDeclaredAddressScopeReadsTheCitersOncePerFold(t *testing.T) {
	changes := []spanChange{
		darkOpening("198.51.100.7:443/tcp"),
		darkOpening("198.51.100.7:80/tcp"),
		darkOpening("198.51.100.8:443/tcp"),
	}
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}

	if _, err := buildMessages(context.Background(), store, 76, produceT0, changes, nil, nil, in); err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(store.citersAsked) != 1 {
		t.Fatalf("three openings are one read, never one read each, got %d reads: %+v", len(store.citersAsked), store.citersAsked)
	}
	want := []string{"198.51.100.7", "198.51.100.8"}
	if !reflect.DeepEqual(store.citersAsked[0], want) {
		t.Errorf("the read carries every candidate address once, want %v, got %v", want, store.citersAsked[0])
	}
}

func TestARePointAndADarkOpeningReadTheCitersOncePerFold(t *testing.T) {
	changes := append(rePointChanges(rpNew, resolved(rpOld)), darkOpening("198.51.100.7:443/tcp"))
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("198.51.100.0/24")}}

	msgs, err := buildMessages(context.Background(), store, 77, produceT0, changes, nil, nil, in)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(store.citersAsked) != 1 {
		t.Fatalf("the two producers share one read of the fold's addresses, got %d reads: %+v", len(store.citersAsked), store.citersAsked)
	}
	want := []string{"198.51.100.7", rpNew}
	if !reflect.DeepEqual(store.citersAsked[0], want) {
		t.Errorf("the one read carries both producers' keys once, want %v, got %v", want, store.citersAsked[0])
	}
	got := byKind(msgs)
	if len(got["seed"]) != 1 || got["seed"][0].CensusLen() != 1 {
		t.Errorf("the dark opening still fires one scope message, got %+v", got["seed"])
	}
	if len(got["address"]) != 1 || got["address"][0].FiredAt != rpNew {
		t.Errorf("the move still roots on the address it reached, got %+v", got["address"])
	}
}
