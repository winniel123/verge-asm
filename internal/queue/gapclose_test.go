package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

func gapClosingChange(kind, key, facet string, before, after []byte) spanChange {
	return spanChange{
		SubjectKind: kind, SubjectKey: key, Facet: facet, Opened: false,
		Previous: []byte(`{"outcome":"gap","cause":"vantage-unavailable"}`), PrevIsGap: true,
		Value: after, BeforeGap: before,
	}
}

func TestGapCloseFiresOnceAtTheCoveringScopeWithThePair(t *testing.T) {
	const svc1, svc2 = "10.1.0.1:443/tcp", "10.1.0.2:443/tcp"
	changes := []spanChange{
		gapClosingChange("service", svc1, "reachability", reachValue("not-reached"), reachValue("reached")),
		gapClosingChange("service", svc2, "reachability", reachValue("reached"), reachValue("reached")),
	}
	store := &fakeMessageStore{}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("10.1.0.0/24")}}
	var log []routed
	if err := produceMessages(context.Background(), store, 40, produceT0, changes, nil, nil, in, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 {
		t.Fatalf("a closing Gap is the sole carrier of the pair, got %d: %+v", len(store.inserted), store.inserted)
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseAperture) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a Gap closing is a coverage-class firing, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "seed" || m.FiredAt != "10.1.0.0/24" {
		t.Errorf("fires at the covering scope, got %q/%q", m.SubjectKind, m.FiredAt)
	}
	c, _ := message.ParseCensus(m.Census)
	if c.Len() != 2 || c.Entries[0].Key != svc1 || c.Entries[0].Detail != "reachability not-reached → reached" || c.Entries[1].Detail != "" {
		t.Errorf("the census names each subject and carries the pair only where it differs, got %+v", c)
	}
	if !strings.Contains(m.Headline, "1 differs from the last value seen: "+svc1) {
		t.Errorf("the headline states the difference, got %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassCoverage {
		t.Errorf("routed once on the coverage class, got %+v", log)
	}
	if store.citationsRead {
		t.Error("an address-seed-covered Service needs no citation read")
	}
}

func TestGapOpeningFiresNothing(t *testing.T) {
	changes := []spanChange{{
		SubjectKind: "service", SubjectKey: "10.1.0.1:443/tcp", Facet: "reachability", Opened: false,
		Previous: reachValue("reached"), Value: []byte(`{"outcome":"gap","cause":"vantage-unavailable"}`), IsGap: true,
	}, {
		SubjectKind: "service", SubjectKey: "10.1.0.3:443/tcp", Facet: "reachability", Opened: true,
		Value: []byte(`{"outcome":"gap","cause":"blanket"}`), IsGap: true,
	}}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("10.1.0.0/24")}}
	got, err := gapCloseMessages(context.Background(), &fakeMessageStore{}, produceT0, changes, in)
	if err != nil {
		t.Fatalf("gap close: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a Gap opening is inventory and never a message, got %+v", got)
	}
}

func TestGapCloseUnchangedResolutionIsOneLine(t *testing.T) {
	resolved := []byte(`{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	changes := []spanChange{gapClosingChange("name", "www.example.com", "resolution", resolved, resolved)}
	in := membershipInputs{seeds: []db.ListSeedsRow{plainNameSeed("example.com")}}
	got, err := gapCloseMessages(context.Background(), &fakeMessageStore{}, produceT0, changes, in)
	if err != nil {
		t.Fatalf("gap close: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("one message at the scope, got %+v", got)
	}
	want := "example.com · sight restored on 1 timeline across 1 subject · none differs from the last value seen"
	if got[0].Headline != want {
		t.Errorf("headline\n got %q\nwant %q", got[0].Headline, want)
	}
	if got[0].FiredAt != "example.com" || got[0].SubjectKind != "seed" {
		t.Errorf("fires at the name scope, got %q/%q", got[0].SubjectKind, got[0].FiredAt)
	}
}

func TestGapCloseReachesAnExtendedZoneThroughTheCitingName(t *testing.T) {
	const svc = "52.1.2.3:443/tcp"
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("api.example.com", produceT0.Add(-time.Hour), nil, "52.1.2.3"),
	}}
	changes := []spanChange{gapClosingChange("service", svc, "reachability", nil, reachValue("reached"))}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	got, err := gapCloseMessages(context.Background(), store, produceT0, changes, in)
	if err != nil {
		t.Fatalf("gap close: %v", err)
	}
	if !store.citationsRead {
		t.Fatal("an uncovered address resolves its scope through the citing Name")
	}
	if len(got) != 1 || got[0].FiredAt != "example.com" || got[0].SubjectKind != "seed" {
		t.Fatalf("fires at the extending scope, got %+v", got)
	}
	if !strings.HasSuffix(got[0].Headline, " · 1 has no earlier value") {
		t.Errorf("a timeline that opened as a Gap states no pair, got %q", got[0].Headline)
	}
}

func TestGapCloseWithNoCoveringScopeFiresAtTheSubject(t *testing.T) {
	changes := []spanChange{gapClosingChange("name", "cdn.example.net", "resolution",
		[]byte(`{"outcome":"Resolved","addresses":["198.51.100.1"]}`),
		[]byte(`{"outcome":"Resolved","addresses":["198.51.100.9","198.51.100.2"]}`))}
	in := membershipInputs{seeds: []db.ListSeedsRow{plainNameSeed("example.com")}}
	got, err := gapCloseMessages(context.Background(), &fakeMessageStore{}, produceT0, changes, in)
	if err != nil {
		t.Fatalf("gap close: %v", err)
	}
	if len(got) != 1 || got[0].SubjectKind != "name" || got[0].FiredAt != "cdn.example.net" {
		t.Fatalf("the closing is never lost, got %+v", got)
	}
	if !strings.Contains(got[0].Headline, "resolution Resolved 198.51.100.1 → Resolved 198.51.100.2, 198.51.100.9") {
		t.Errorf("the pair states both values with addresses in order, got %q", got[0].Headline)
	}
}

func TestGapCloseCarriesTheRulesThatOpenedAtFiredBeneathIt(t *testing.T) {
	const svc = "10.1.0.1:3306/tcp"
	changes := []spanChange{gapClosingChange("service", svc, "reachability", reachValue("not-reached"), reachValue("reached"))}
	store := &fakeMessageStore{
		current: []db.ListServiceReachabilitySpansByClassForServicesRow{internetReachRow(svc, "reached")},
	}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("10.1.0.0/24")}}
	if err := produceMessages(context.Background(), store, 41, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 1 || !strings.Contains(store.inserted[0].Headline, "sight restored") {
		t.Fatalf("the closing is the sole carrier, and the flagship has no decided before, got %+v", store.inserted)
	}
	rules, byKey := ruleKeys(t, store.inserted[0].Census)
	if strings.Join(rules, ",") != "sensitive-port-reached-from-internet" || byKey[rules[0]].Detail != "3306/tcp" {
		t.Errorf("the census names the rule the restored leg opened, with its port, got %v", rules)
	}
	if !strings.Contains(store.inserted[0].Headline, "1 rule opened at fired: sensitive-port-reached-from-internet (3306/tcp)") {
		t.Errorf("the headline names the rule for the channel body, got %q", store.inserted[0].Headline)
	}
}

func TestGapCloseNamesNoRuleOnAnInternalLeg(t *testing.T) {
	const svc = "10.1.0.1:3306/tcp"
	changes := []spanChange{gapClosingChange("service", svc, "reachability", reachValue("not-reached"), reachValue("reached"))}
	store := &fakeMessageStore{
		current: []db.ListServiceReachabilitySpansByClassForServicesRow{internalReachRow(svc, "reached")},
	}
	in := membershipInputs{seeds: []db.ListSeedsRow{addressSeed("10.1.0.0/24")}}
	got, err := gapCloseMessages(context.Background(), store, produceT0, changes, in)
	if err != nil {
		t.Fatalf("gap close: %v", err)
	}
	if len(got) != 1 || len(got[0].Census.Rules()) != 0 {
		t.Errorf("a rule reads the internet leg, and an internal reach opens none, got %+v", got)
	}
}

func TestLastValueBeforeGapReadsTheLatestClosedValueOnTheTimeline(t *testing.T) {
	v1 := pgtype.Int8{Int64: 1, Valid: true}
	v2 := pgtype.Int8{Int64: 2, Valid: true}
	key := drift.TimelineKey{Facet: "reachability", Source: "prober"}
	at := func(h int) pgtype.Timestamptz { return tstz(produceT0.Add(time.Duration(h) * time.Hour)) }
	rows := []db.ListSpansForSubjectRow{
		{ID: 1, Facet: "reachability", Source: "prober", VantageID: v1, Value: reachValue("reached"), OpenedAt: at(-10), ClosedAt: at(-8)},
		{ID: 2, Facet: "reachability", Source: "prober", VantageID: v1, Value: reachValue("not-reached"), OpenedAt: at(-8), ClosedAt: at(-4)},
		{ID: 3, Facet: "reachability", Source: "prober", VantageID: v1, IsGap: true, Value: []byte(`{"outcome":"gap"}`), OpenedAt: at(-4)},
		{ID: 4, Facet: "reachability", Source: "prober", VantageID: v2, Value: reachValue("reached"), OpenedAt: at(-3), ClosedAt: at(-2)},
		{ID: 5, Facet: "certificate", Source: "prober", VantageID: v1, Value: []byte(`{}`), OpenedAt: at(-3), ClosedAt: at(-1)},
	}
	got, broke := lastValueBeforeGap(rows, key, v1, drift.Vector{})
	if string(got) != `{"outcome":"not-reached"}` || broke {
		t.Errorf("the value before the Gap is the latest closed non-Gap span on this timeline, got %s broke=%v", got, broke)
	}
	if got, broke := lastValueBeforeGap(rows, key, pgtype.Int8{Int64: 3, Valid: true}, drift.Vector{}); got != nil || broke {
		t.Errorf("a timeline with no earlier value reads nil, got %s broke=%v", got, broke)
	}
}

func TestLastValueBeforeGapStatesNoPairAcrossADerivationMove(t *testing.T) {
	v1 := pgtype.Int8{Int64: 1, Valid: true}
	key := drift.TimelineKey{Facet: "reachability", Source: "prober"}
	before := drift.NewVector(drift.Component{Leaf: "connect-outcome", Version: "1"})
	after := drift.NewVector(drift.Component{Leaf: "connect-outcome", Version: "2"})
	rows := []db.ListSpansForSubjectRow{
		{ID: 1, Facet: "reachability", Source: "prober", VantageID: v1, Value: reachValue("not-reached"), Derivation: mustVectorJSON(before),
			OpenedAt: tstz(produceT0.Add(-8 * time.Hour)), ClosedAt: tstz(produceT0.Add(-4 * time.Hour))},
		{ID: 2, Facet: "reachability", Source: "prober", VantageID: v1, IsGap: true, Value: []byte(`{"outcome":"gap"}`), Derivation: mustVectorJSON(after),
			OpenedAt: tstz(produceT0.Add(-4 * time.Hour))},
	}
	got, broke := lastValueBeforeGap(rows, key, v1, after)
	if got != nil || !broke {
		t.Errorf("a value under another derivation is not compared, got %s broke=%v", got, broke)
	}
	if got, broke := lastValueBeforeGap(rows, key, v1, before); string(got) != `{"outcome":"not-reached"}` || broke {
		t.Errorf("the same derivation licenses the pair, got %s broke=%v", got, broke)
	}
}
