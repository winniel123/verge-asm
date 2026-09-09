package queue

import (
	"context"
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
