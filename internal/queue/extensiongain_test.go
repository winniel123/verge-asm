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

func extendingSeed(domain string) db.ListSeedsRow {
	return db.ListSeedsRow{Kind: "name", NameDomain: pgtype.Text{String: domain, Valid: true}, CustodyExtension: true}
}

func plainNameSeed(domain string) db.ListSeedsRow {
	return db.ListSeedsRow{Kind: "name", NameDomain: pgtype.Text{String: domain, Valid: true}}
}

func citedSpan(name string, openedAt time.Time, closedAt *time.Time, addrs ...string) citationSpan {
	quoted := make([]string, 0, len(addrs))
	for _, a := range addrs {
		quoted = append(quoted, `"`+a+`"`)
	}
	r := citationSpan{
		SubjectKey: name, VantageID: pgtype.Int8{Int64: 1, Valid: true}, Facet: "resolution",
		Value:    []byte(`{"outcome":"Resolved","addresses":[` + strings.Join(quoted, ",") + `]}`),
		OpenedAt: tstz(openedAt),
	}
	if closedAt != nil {
		r.ClosedAt = tstz(*closedAt)
	}
	return r
}

func dnsRecordSpan(name string, openedAt time.Time, closedAt *time.Time, rrs string) citationSpan {
	r := citationSpan{
		SubjectKey: name, VantageID: pgtype.Int8{Int64: 1, Valid: true}, Facet: "dns-record",
		Value:    []byte(`{"rrs":[` + rrs + `]}`),
		OpenedAt: tstz(openedAt),
	}
	if closedAt != nil {
		r.ClosedAt = tstz(*closedAt)
	}
	return r
}

func nameResolutionChange(name string, addrs ...string) spanChange {
	return spanChange{
		SubjectKind: "name", SubjectKey: name, Facet: "resolution", Opened: true,
		Value: []byte(`{"outcome":"Resolved","addresses":["` + strings.Join(addrs, `","`) + `"]}`),
	}
}

func extensionGainMessagesOf(t *testing.T, store *fakeMessageStore) []db.InsertMessageParams {
	t.Helper()
	var out []db.InsertMessageParams
	for _, m := range store.inserted {
		if strings.Contains(m.Headline, "custody extension gained") {
			out = append(out, m)
		}
	}
	return out
}

func TestExtensionGainFiresOnceAtTheScopeWithTheDifferenceAndTheCount(t *testing.T) {
	earlier := produceT0.Add(-24 * time.Hour)
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("www.example.com", earlier, &produceT0, "52.1.2.1"),
		citedSpan("www.example.com", produceT0, nil, "52.1.2.1", "52.1.2.3"),
		citedSpan("api.example.com", produceT0, nil, "52.1.2.3", "52.1.2.4"),
	}}
	changes := []spanChange{nameResolutionChange("api.example.com", "52.1.2.3", "52.1.2.4")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	var log []routed
	if err := produceMessages(context.Background(), store, 30, produceT0, changes, nil, nil, in, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	got := extensionGainMessagesOf(t, store)
	if len(got) != 1 {
		t.Fatalf("one message per extending scope, got %d: %+v", len(got), got)
	}
	m := got[0]
	if m.Cause != string(message.CauseAperture) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a gain is a coverage-class aperture firing, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "seed" || m.FiredAt != "example.com" {
		t.Errorf("a gain fires at the extending scope, got %q/%q", m.SubjectKind, m.FiredAt)
	}
	c, _ := message.ParseCensus(m.Census)
	if c.Len() != 2 || c.Entries[0].Key != "52.1.2.3" || c.Entries[1].Key != "52.1.2.4" {
		t.Errorf("the census carries the gained addresses only, got %+v", c)
	}
	if c.Entries[0].Detail != "api.example.com, www.example.com" || c.Entries[1].Detail != "api.example.com" {
		t.Errorf("each entry names the citing names, got %+v", c)
	}
	if !strings.Contains(m.Headline, "3 addresses covered") {
		t.Errorf("the count is the extension's own after the gain, got %q", m.Headline)
	}
	if !strings.Contains(m.Headline, "52.1.2.3 (api.example.com, www.example.com)") || strings.Contains(m.Headline, "52.1.2.1") {
		t.Errorf("the headline states the difference and nothing else, got %q", m.Headline)
	}
	routedCoverage := 0
	for _, r := range log {
		if r.class == message.ClassCoverage {
			routedCoverage++
		}
	}
	if routedCoverage != 1 {
		t.Errorf("the gain is routed on the coverage class once, got %+v", log)
	}
}

func TestExtensionGainReadsNothingWithoutALiveExtension(t *testing.T) {
	store := &fakeMessageStore{citations: []citationSpan{citedSpan("api.example.com", produceT0, nil, "52.1.2.3")}}
	changes := []spanChange{nameResolutionChange("api.example.com", "52.1.2.3")}
	in := membershipInputs{seeds: []db.ListSeedsRow{plainNameSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 31, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if store.citationsRead {
		t.Error("no extension is declared, so the citation spans are never read")
	}
	if got := extensionGainMessagesOf(t, store); len(got) != 0 {
		t.Errorf("no extension, no message, got %+v", got)
	}
}

func TestExtensionGainReadsNothingWhenNoNameMoved(t *testing.T) {
	store := &fakeMessageStore{citations: []citationSpan{citedSpan("api.example.com", produceT0, nil, "52.1.2.3")}}
	changes := []spanChange{{SubjectKind: "service", SubjectKey: "52.1.2.3:443/tcp", Facet: "reachability", Opened: true, Value: reachValue("reached")}}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 32, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if store.citationsRead {
		t.Error("a batch that moved no Name timeline never reads the citation spans")
	}
}

func TestExtensionGainIsSilentForAnAddressAnotherNameAlreadyCited(t *testing.T) {
	earlier := produceT0.Add(-24 * time.Hour)
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("www.example.com", earlier, nil, "52.1.2.3"),
		citedSpan("api.example.com", produceT0, nil, "52.1.2.3"),
	}}
	changes := []spanChange{nameResolutionChange("api.example.com", "52.1.2.3")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 33, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if got := extensionGainMessagesOf(t, store); len(got) != 0 {
		t.Errorf("an address the extension already reached is no gain, got %+v", got)
	}
}

func TestExtensionGainIsSilentForAReEntryWithinTheCurrencyBound(t *testing.T) {
	earlier := produceT0.Add(-48 * time.Hour)
	left := produceT0.Add(-24 * time.Hour)
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("www.example.com", earlier, &left, "52.1.2.3"),
		citedSpan("www.example.com", left, &produceT0, "52.1.2.4"),
		citedSpan("www.example.com", produceT0, nil, "52.1.2.3"),
	}}
	changes := []spanChange{nameResolutionChange("www.example.com", "52.1.2.3")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 34, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if got := extensionGainMessagesOf(t, store); len(got) != 0 {
		t.Errorf("a blue-green flip inside the bound does not re-fire, got %+v", got)
	}
}

func TestExtensionGainIsSilentOnADeparture(t *testing.T) {
	earlier := produceT0.Add(-24 * time.Hour)
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("www.example.com", earlier, &produceT0, "52.1.2.3", "52.1.2.4"),
		citedSpan("www.example.com", produceT0, nil, "52.1.2.3"),
	}}
	changes := []spanChange{nameResolutionChange("www.example.com", "52.1.2.3")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 35, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if got := extensionGainMessagesOf(t, store); len(got) != 0 {
		t.Errorf("an address leaving is the gate narrowing, not a message, got %+v", got)
	}
}

func TestExtensionGainStopsWhereTheChainLeavesTheZone(t *testing.T) {
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("shop.example.com", produceT0, nil, "13.32.0.1"),
		dnsRecordSpan("shop.example.com", produceT0, nil,
			`{"name":"shop.example.com","type":"CNAME","data":"d1x2y3.cloudfront.net"},`+
				`{"name":"d1x2y3.cloudfront.net","type":"A","data":"13.32.0.1"}`),
	}}
	changes := []spanChange{nameResolutionChange("shop.example.com", "13.32.0.1")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 36, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if got := extensionGainMessagesOf(t, store); len(got) != 0 {
		t.Errorf("a CNAME target outside every extending zone is outside the extension, got %+v", got)
	}
}

func TestExtensionGainIgnoresANonGloballyReachableAddress(t *testing.T) {
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("db.example.com", produceT0, nil, "10.1.2.3"),
	}}
	changes := []spanChange{nameResolutionChange("db.example.com", "10.1.2.3")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 37, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if got := extensionGainMessagesOf(t, store); len(got) != 0 {
		t.Errorf("an extension reaches no non-globally-reachable address, got %+v", got)
	}
}

func TestExtensionGainFiresPerScopeAndTheMostSpecificScopeClaimsANestedName(t *testing.T) {
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("a.example.com", produceT0, nil, "52.1.2.3"),
		citedSpan("b.api.example.com", produceT0, nil, "52.1.2.4"),
		citedSpan("c.example.org", produceT0, nil, "52.1.2.5"),
	}}
	changes := []spanChange{nameResolutionChange("a.example.com", "52.1.2.3")}
	in := membershipInputs{seeds: []db.ListSeedsRow{
		extendingSeed("example.com"), extendingSeed("api.example.com"), extendingSeed("example.org"),
	}}
	if err := produceMessages(context.Background(), store, 38, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	got := extensionGainMessagesOf(t, store)
	if len(got) != 3 {
		t.Fatalf("one message per extending scope, got %d: %+v", len(got), got)
	}
	want := map[string]string{
		"api.example.com": "52.1.2.4 (b.api.example.com)",
		"example.com":     "52.1.2.3 (a.example.com)",
		"example.org":     "52.1.2.5 (c.example.org)",
	}
	for _, m := range got {
		entry, ok := want[m.FiredAt]
		if !ok {
			t.Errorf("unexpected scope %q", m.FiredAt)
			continue
		}
		if !strings.Contains(m.Headline, "gained 1 address: "+entry+" · 1 address covered") {
			t.Errorf("scope %q carries its own difference and count, got %q", m.FiredAt, m.Headline)
		}
	}
}

func TestExtensionGainPairsAnOwnerWithTheSpanOpenBesideIt(t *testing.T) {
	earlier := produceT0.Add(-24 * time.Hour)
	foreign := `{"name":"www.example.com","type":"CNAME","data":"d1x2y3.cloudfront.net"},` +
		`{"name":"d1x2y3.cloudfront.net","type":"A","data":"13.32.0.1"}`
	direct := `{"name":"www.example.com","type":"A","data":"13.32.0.1"}`
	store := &fakeMessageStore{citations: []citationSpan{
		citedSpan("www.example.com", earlier, &produceT0, "13.32.0.1"),
		citedSpan("www.example.com", produceT0, nil, "13.32.0.1"),
		dnsRecordSpan("www.example.com", earlier, &produceT0, foreign),
		dnsRecordSpan("www.example.com", produceT0, nil, direct),
	}}
	changes := []spanChange{nameResolutionChange("www.example.com", "13.32.0.1")}
	in := membershipInputs{seeds: []db.ListSeedsRow{extendingSeed("example.com")}}
	if err := produceMessages(context.Background(), store, 39, produceT0, changes, nil, nil, in, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	got := extensionGainMessagesOf(t, store)
	if len(got) != 1 || !strings.Contains(got[0].Headline, "13.32.0.1 (www.example.com)") {
		t.Fatalf("the extension first reaches the address when the in-zone owner cites it, got %+v", got)
	}
}
