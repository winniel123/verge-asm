package queue

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
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

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
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

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, membershipInputs{}, fakeEnqueuer(1, &log), true); err != nil {
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

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, membershipInputs{}, fakeEnqueuer(0, &log), false); err != nil {
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
	if err := produceMessages(context.Background(), store, 1, produceT0, changes, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
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
	if err := produceMessages(context.Background(), store, 2, produceT0, changes, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.inserted) != 0 {
		t.Errorf("an internal-leg move fires no flagship, got %d messages", len(store.inserted))
	}
}
