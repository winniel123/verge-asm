package queue

import (
	"context"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

var produceT0 = time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)

type fakeMessageStore struct {
	prev        pgtype.Timestamptz
	current     []db.ListServiceReachabilitySpansByClassRow
	at          []db.ListServiceReachabilitySpansByClassAtRow
	inserted    []db.InsertMessageParams
	nextID      int64
	touchedRead bool

	addressExclusions []*netip.Prefix
}

func (f *fakeMessageStore) PreviousBatchTime(context.Context) (pgtype.Timestamptz, error) {
	f.touchedRead = true
	return f.prev, nil
}

func (f *fakeMessageStore) ListServiceReachabilitySpansByClass(context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error) {
	f.touchedRead = true
	return f.current, nil
}

func (f *fakeMessageStore) ListServiceReachabilitySpansByClassAt(context.Context, pgtype.Timestamptz) ([]db.ListServiceReachabilitySpansByClassAtRow, error) {
	f.touchedRead = true
	return f.at, nil
}

func (f *fakeMessageStore) InsertMessage(_ context.Context, arg db.InsertMessageParams) (db.Message, error) {
	f.inserted = append(f.inserted, arg)
	f.nextID++
	return db.Message{ID: f.nextID, Cause: arg.Cause, Class: arg.Class, SubjectKind: arg.SubjectKind, FiredAt: arg.FiredAt}, nil
}

func (f *fakeMessageStore) ListAddressScopeCidrs(context.Context) ([]*netip.Prefix, error) {
	p := netip.MustParsePrefix("10.0.0.0/8")
	return []*netip.Prefix{&p}, nil
}

func (f *fakeMessageStore) ListAddressExclusionCidrs(context.Context) ([]*netip.Prefix, error) {
	return f.addressExclusions, nil
}

const (
	dialledInternet = "198.51.100.200"
	dialledInternal = "10.200.0.1"
)

func internetReachRow(svc, outcome string) db.ListServiceReachabilitySpansByClassRow {
	return db.ListServiceReachabilitySpansByClassRow{
		SubjectKey: svc, VantageID: pgtype.Int8{Int64: 1, Valid: true}, Value: reachValue(outcome),
		DialledAddr: pgtype.Text{String: dialledInternet, Valid: true},
	}
}

func internetReachAtRow(svc, outcome string) db.ListServiceReachabilitySpansByClassAtRow {
	return db.ListServiceReachabilitySpansByClassAtRow{
		SubjectKey: svc, VantageID: pgtype.Int8{Int64: 1, Valid: true}, Value: reachValue(outcome),
		DialledAddr: pgtype.Text{String: dialledInternet, Valid: true},
	}
}

func internalReachRow(svc, outcome string) db.ListServiceReachabilitySpansByClassRow {
	return db.ListServiceReachabilitySpansByClassRow{
		SubjectKey: svc, VantageID: pgtype.Int8{Int64: 2, Valid: true}, Value: reachValue(outcome),
		DialledAddr: pgtype.Text{String: dialledInternal, Valid: true},
	}
}

func internalReachAtRow(svc, outcome string) db.ListServiceReachabilitySpansByClassAtRow {
	return db.ListServiceReachabilitySpansByClassAtRow{
		SubjectKey: svc, VantageID: pgtype.Int8{Int64: 2, Valid: true}, Value: reachValue(outcome),
		DialledAddr: pgtype.Text{String: dialledInternal, Valid: true},
	}
}

type routed struct {
	messageID int64
	class     message.Class
	made      int
}

func fakeEnqueuer(made int, log *[]routed) enqueueFunc {
	return func(_ context.Context, messageID int64, class message.Class) (int, error) {
		*log = append(*log, routed{messageID: messageID, class: class, made: made})
		return made, nil
	}
}

func prevAt(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func reachValue(outcome string) []byte { return []byte(`{"outcome":"` + outcome + `"}`) }

func batchMovingBothSignals() (changes []spanChange, store *fakeMessageStore) {
	const svc = "198.51.100.1:443/tcp"
	changes = []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: false, Value: reachValue("reached")},
		{SubjectKind: "endpoint", SubjectKey: "example.com|" + svc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "name", SubjectKey: "example.com", Facet: "resolution", Opened: true,
			Value: []byte(`{"outcome":"Resolved","addresses":["198.51.100.1"]}`)},
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
	}
	store = &fakeMessageStore{
		prev:    prevAt(produceT0.Add(-time.Hour)),
		current: []db.ListServiceReachabilitySpansByClassRow{internetReachRow(svc, "reached")},
		at:      []db.ListServiceReachabilitySpansByClassAtRow{internetReachAtRow(svc, "not-reached")},
	}
	return changes, store
}

func TestProduceWritesFlagshipAndMembershipAndEnqueues(t *testing.T) {
	changes, store := batchMovingBothSignals()
	var log []routed

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(store.inserted) != 2 {
		t.Fatalf("want 2 messages written (flagship + membership), got %d", len(store.inserted))
	}
	var flagship, membership *db.InsertMessageParams
	for i := range store.inserted {
		switch store.inserted[i].SubjectKind {
		case "service":
			flagship = &store.inserted[i]
		case "name":
			membership = &store.inserted[i]
		}
	}
	if flagship == nil {
		t.Fatal("no flagship message written")
	}
	if flagship.Cause != string(message.CauseDrift) || flagship.Class != string(message.ClassDrift) {
		t.Errorf("flagship is a drift firing, got cause=%q class=%q", flagship.Cause, flagship.Class)
	}
	if flagship.FiredAt != "198.51.100.1:443/tcp" {
		t.Errorf("flagship fires at the Service, got %q", flagship.FiredAt)
	}
	if c, _ := message.ParseCensus(flagship.Census); c.Len() != 1 || c.Entries[0].Key != "certificate" {
		t.Errorf("flagship census must carry the facet that opened beneath, got %+v", c)
	}
	if membership == nil {
		t.Fatal("no membership message written")
	}
	if membership.Cause != string(message.CauseDrift) || membership.FiredAt != "example.com" {
		t.Errorf("appeared membership fires drift at the Name, got cause=%q fired=%q", membership.Cause, membership.FiredAt)
	}
	mc, _ := message.ParseCensus(membership.Census)
	kinds := map[string]bool{}
	for _, e := range mc.Entries {
		kinds[e.Kind] = true
	}
	if !kinds["service"] || !kinds["endpoint"] {
		t.Errorf("membership census must enumerate the Service and Endpoint that entered beneath, got %+v", mc)
	}

	if len(log) != 2 {
		t.Fatalf("both messages must be routed to channels, got %d routings", len(log))
	}
	for _, r := range log {
		if r.made != 1 {
			t.Errorf("a bound channel receives one delivery, got %d", r.made)
		}
	}
}

func TestProduceIsNoOpUnderDevMode(t *testing.T) {
	changes, store := batchMovingBothSignals()
	var log []routed

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), true); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(store.inserted) != 0 {
		t.Errorf("a dev install writes no message, got %d", len(store.inserted))
	}
	if len(log) != 0 {
		t.Errorf("a dev install routes nothing, got %d", len(log))
	}
	if store.touchedRead {
		t.Error("the dev guard must short-circuit before any store read")
	}
}

func TestProduceUnboundConfigMakesNoDelivery(t *testing.T) {
	changes, store := batchMovingBothSignals()
	var log []routed

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(0, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(store.inserted) != 2 {
		t.Fatalf("the messages are still written, got %d", len(store.inserted))
	}
	if len(log) != 2 {
		t.Fatalf("each message is offered to the router, got %d", len(log))
	}
	for _, r := range log {
		if r.made != 0 {
			t.Errorf("an unbound config makes no delivery, got %d", r.made)
		}
	}
}

func TestProduceOpeningAtReachedIsNotFlagship(t *testing.T) {
	const svc = "198.51.100.1:443/tcp"
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
	}
	store := &fakeMessageStore{
		prev:    pgtype.Timestamptz{},
		current: []db.ListServiceReachabilitySpansByClassRow{internetReachRow(svc, "reached")},
	}
	var log []routed
	if err := produceMessages(context.Background(), store, 1, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 {
		t.Errorf("a leg opening at reached fires no flagship, got %d messages", len(store.inserted))
	}
}

func TestProduceInternalLegNeverFlagship(t *testing.T) {
	const svc = "10.0.0.1:22/tcp"

	// An internal port opening or closing is recorded and never alerted, either way (ADR-0029).
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: false, Value: reachValue("reached")},
	}
	store := &fakeMessageStore{
		prev:    prevAt(produceT0.Add(-time.Hour)),
		current: []db.ListServiceReachabilitySpansByClassRow{internalReachRow(svc, "reached")},
		at:      []db.ListServiceReachabilitySpansByClassAtRow{internalReachAtRow(svc, "not-reached")},
	}
	var log []routed
	if err := produceMessages(context.Background(), store, 2, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 {
		t.Errorf("an internal-leg move fires no flagship, got %d messages", len(store.inserted))
	}
}

func TestProduceDescopedDepartureFiresDeclaredInput(t *testing.T) {
	departures := []departure{
		{SubjectKind: "name", SubjectKey: "old.example.com", Reason: string(drift.ReasonDescoped), SourceKey: "example.com", Timelines: 3},
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 9, produceT0, nil, departures, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("want 1 declared-input message, got %d", len(store.inserted))
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseDeclaredInput) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a descope is a declared-input / coverage firing, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "source" || m.FiredAt != "example.com" {
		t.Errorf("the row links to the Exclusion as its Source, got kind=%q fired=%q", m.SubjectKind, m.FiredAt)
	}
	if !strings.Contains(m.Headline, "old.example.com") || !strings.Contains(m.Headline, "3 timelines") {
		t.Errorf("the headline states the withdrawn subject and its timeline count, got %q", m.Headline)
	}
	if message.ContainsValence(m.Headline) {
		t.Errorf("the headline carries a valence word: %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassCoverage {
		t.Errorf("the message is routed on the coverage class, got %+v", log)
	}
}

func TestProduceMeasuredAbsentDepartureFiresNoDeclaredInput(t *testing.T) {
	departures := []departure{
		{SubjectKind: "name", SubjectKey: "gone.example.com", Reason: string(drift.ReasonMeasuredAbsent), SourceKey: "", Timelines: 2},
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 10, produceT0, nil, departures, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 {
		t.Errorf("a world withdrawal fires no declared-input message, got %d", len(store.inserted))
	}
	if len(log) != 0 {
		t.Errorf("a world withdrawal routes nothing, got %d", len(log))
	}
}

func TestProduceDescopedDepartureIsNoOpUnderDevMode(t *testing.T) {
	departures := []departure{
		{SubjectKind: "name", SubjectKey: "old.example.com", Reason: string(drift.ReasonDescoped), SourceKey: "example.com", Timelines: 1},
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 11, produceT0, nil, departures, nil, membershipInputs{}, fakeEnqueuer(1, &log), true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 || len(log) != 0 {
		t.Errorf("a dev install writes and routes nothing, got %d messages / %d routings", len(store.inserted), len(log))
	}
}

func TestProduceNarrowingFiresOnceAtTheScope(t *testing.T) {
	narrowings := []message.NarrowingReceipt{
		message.PreviewNarrowing("198.51.100.0/24", "198.51.100.128/25", 3, 7),
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 12, produceT0, nil, nil, narrowings, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	if len(store.inserted) != 1 {
		t.Fatalf("want 1 narrowing message, got %d", len(store.inserted))
	}
	m := store.inserted[0]
	if m.Cause != string(message.CauseAperture) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a narrowing is an aperture / coverage firing, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.SubjectKind != "seed" || m.FiredAt != "198.51.100.0/24" {
		t.Errorf("the message fires at the Seed scope that moved, got kind=%q fired=%q", m.SubjectKind, m.FiredAt)
	}
	if !strings.Contains(m.Headline, "198.51.100.128/25") || !strings.Contains(m.Headline, "3") || !strings.Contains(m.Headline, "7") {
		t.Errorf("the headline states the removed value and both counts, got %q", m.Headline)
	}
	if len(m.Census) != 0 {
		t.Errorf("a narrowing carries a count, not a census of rows, got %d bytes", len(m.Census))
	}
	if message.ContainsValence(m.Headline) {
		t.Errorf("the headline carries a valence word: %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassCoverage {
		t.Errorf("the message is routed on the coverage class, got %+v", log)
	}
}

func TestProduceNarrowingSilentOverUninhabitedGround(t *testing.T) {
	narrowings := []message.NarrowingReceipt{
		message.PreviewNarrowing("198.51.100.0/24", "198.51.100.128/25", 0, 0),
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 13, produceT0, nil, nil, narrowings, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 || len(log) != 0 {
		t.Errorf("an empty withdrawal writes and routes nothing, got %d messages / %d routings", len(store.inserted), len(log))
	}
}

func TestProduceNarrowingIsNoOpUnderDevMode(t *testing.T) {
	narrowings := []message.NarrowingReceipt{
		message.PreviewNarrowing("198.51.100.0/24", "198.51.100.128/25", 3, 7),
	}
	store := &fakeMessageStore{}
	var log []routed

	if err := produceMessages(context.Background(), store, 14, produceT0, nil, nil, narrowings, membershipInputs{}, fakeEnqueuer(1, &log), true); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 || len(log) != 0 {
		t.Errorf("a dev install writes and routes nothing, got %d messages / %d routings", len(store.inserted), len(log))
	}
}
