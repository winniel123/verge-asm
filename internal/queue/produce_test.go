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

// fakeMessageStore drives the producer with no Postgres: it answers the reach reads
// from canned rows, records every InsertMessage, and flags whether any read ran (so a
// guard test can prove the producer short-circuited before touching the store).
type fakeMessageStore struct {
	prev        pgtype.Timestamptz
	current     []db.ListServiceReachabilitySpansByClassRow
	at          []db.ListServiceReachabilitySpansByClassAtRow
	inserted    []db.InsertMessageParams
	nextID      int64
	touchedRead bool
	// addressExclusions are the declared `address` exclusions the class predicate
	// narrows by (ADR-0133 §4). Nil in every fixture here, so the convention scope
	// covers what it covered before.
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

// ListAddressScopeCidrs returns the fixed 10.0.0.0/8 convention scope the flagship's
// class derivation binds against (#711): the reachability-row constructors below present
// an uncovered public dialled address for an internet vantage and a covered 10.x address
// for an internal one, so each row's class derives (#709) to the intended value.
func (f *fakeMessageStore) ListAddressScopeCidrs(context.Context) ([]*netip.Prefix, error) {
	p := netip.MustParsePrefix("10.0.0.0/8")
	return []*netip.Prefix{&p}, nil
}

// ListAddressExclusionCidrs returns the declared `address` exclusions, which narrow
// that same predicate (ADR-0133 §4). The fixtures declare none, so the convention
// scope covers what it covered before.
func (f *fakeMessageStore) ListAddressExclusionCidrs(context.Context) ([]*netip.Prefix, error) {
	return f.addressExclusions, nil
}

// dialledInternet is an uncovered public address (derives `internet`); dialledInternal
// is a 10.x address covered by the convention scope (derives `internal`).
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

// routed is one message the fake enqueuer routed, and how many deliveries it made —
// the count delivery.EnqueueForMessage returns (zero on an unbound/default install).
type routed struct {
	messageID int64
	class     message.Class
	made      int
}

// fakeEnqueuer models delivery.EnqueueForMessage: it records the routing and returns a
// fixed delivery count — >0 for a bound channel, 0 for an unbound / download-only
// config where no channel POST is ever made.
func fakeEnqueuer(made int, log *[]routed) enqueueFunc {
	return func(_ context.Context, messageID int64, class message.Class) (int, error) {
		*log = append(*log, routed{messageID: messageID, class: class, made: made})
		return made, nil
	}
}

func prevAt(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func reachValue(outcome string) []byte { return []byte(`{"outcome":"` + outcome + `"}`) }

// batchMovingBothSignals is a batch that moves a flagship (an internet leg going
// not-reached → reached on a Service) AND a membership signal (a Name entering the
// estate), with a facet opening beneath the Service and the Service opening beneath
// the Name — the census each carries.
func batchMovingBothSignals() (changes []spanChange, store *fakeMessageStore) {
	const svc = "198.51.100.1:443/tcp"
	changes = []spanChange{
		// The internet leg moved to reached (a flagship candidate).
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: false, Value: reachValue("reached")},
		// A facet opened beneath the newly-reached Service (its flagship census).
		{SubjectKind: "endpoint", SubjectKey: "example.com|" + svc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		// A Name entered the estate (a membership root), citing the Service's address.
		{SubjectKind: "name", SubjectKey: "example.com", Facet: "resolution", Opened: true,
			Value: []byte(`{"outcome":"Resolved","addresses":["198.51.100.1"]}`)},
		// The Service opened beneath the entering Name (its membership census).
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
	}
	store = &fakeMessageStore{
		prev:    prevAt(produceT0.Add(-time.Hour)),
		current: []db.ListServiceReachabilitySpansByClassRow{internetReachRow(svc, "reached")},
		at:      []db.ListServiceReachabilitySpansByClassAtRow{internetReachAtRow(svc, "not-reached")},
	}
	return changes, store
}

// A real (non-dev) install, on a batch that moves both a flagship and a membership
// signal, writes both message rows and routes each to its bound channels.
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
	// The census carries every Subject that entered beneath the Name — the Service
	// and the Endpoint both ride it, never their own message (ADR-0031).
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

// A VERGE_DEV / fixture install writes NO message and routes nothing — a real
// deployment never serves fixtures, so the golden fixtures stay message-free (AL-25).
// The guard is unconditional: no store read runs either.
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

// A default install with no channel bound (download-only / unbound config) still
// writes the message but enqueues no delivery — nothing is POSTed until an admin
// declares a Channel.
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

// A leg that only OPENED at reached this batch is not a flagship — it emits no
// Transition, so its news rides the membership census, never a flagship message
// (ADR-0029). With no previous batch there is no decided "before" leg to move from.
func TestProduceOpeningAtReachedIsNotFlagship(t *testing.T) {
	const svc = "198.51.100.1:443/tcp"
	changes := []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
	}
	store := &fakeMessageStore{
		// No previous batch: the leg has no decided predecessor.
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

// An internal-leg move never fires a flagship, in either direction — an internal port
// opening or closing is recorded and never alerted (ADR-0029).
func TestProduceInternalLegNeverFlagship(t *testing.T) {
	const svc = "10.0.0.1:22/tcp"
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

// A `descoped` departure — a Name the operator's own declared Exclusion narrowed out
// of the estate — fires one coverage-class declared-input message that links to the
// Exclusion as its Source and states the withdrawn subject and the count of timelines
// it took out (AL-2, #722). This is the producer wire the "what fires" table promised
// and that produce.go's openings/moves change-feed left dark.
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

// A `measured-absent` departure — a Name the WORLD stopped resolving — is a drift
// withdrawal, not the operator's declared input moving, so it fires NO declared-input
// message (there is no drift-exit constructor, and the drift cause is already carried
// by the flagship leg). It carries no SourceKey, so declaredInputMessages skips it.
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

// A VERGE_DEV / fixture install writes NO declared-input message either — the dev
// guard short-circuits the whole producer before any departure is folded (AL-25).
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

// An address exclusion's withdrawal fires ONE coverage-class message.Narrowing at
// the Seed scope it narrowed, carrying the two counts and no rows (ADR-0074,
// ADR-0133 §8, #1032). This is the first production call to message.Narrowing:
// until #1032 only PreviewNarrowing was called, so `POST /exclusions/preview`
// described an act the system never performed.
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

// A withdrawal over uninhabited ground fires NOTHING. message.Narrowing returns nil
// where the receipt does not fire, which is the same gate the preview applies — so
// the operator is never shown a receipt for a message that will not come.
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

// A VERGE_DEV / fixture install writes no narrowing message either — the dev guard
// short-circuits the whole producer before any narrowing is folded (AL-25).
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
