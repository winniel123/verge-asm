package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/wire"
)

// A unit test over a hand-built slice models a batch one job never assembles (ADR-1806 §1, #1774).

const (
	proofName     = "example.net"
	proofAddr     = "203.0.113.31"
	proofMoveAddr = "203.0.113.32"
	proofPort     = 443

	proofDNSBatch     = 7100
	proofHotBatch     = 7200
	proofLateBatch    = 7300
	proofMoveBatch    = 7400
	proofMoveHotBatch = 7500
)

var (
	proofTarget     = netip.AddrPortFrom(netip.MustParseAddr(proofAddr), proofPort)
	proofMoveTarget = netip.AddrPortFrom(netip.MustParseAddr(proofMoveAddr), proofPort)
)

func proofSvc() string        { return connectoutcome.ServiceKey(proofTarget, "tcp") }
func proofNamelessEP() string { return connectoutcome.EndpointKey("", proofTarget, "tcp") }
func proofNamedEP() string    { return connectoutcome.EndpointKey(proofName, proofTarget, "tcp") }

type foldedSpan struct {
	db.OpenSpanParams
	id            int64
	closedAt      pgtype.Timestamptz
	closureReason pgtype.Text
	closedBatchID pgtype.Int8
}

// One store answers the fold and the release, so the census reads spans the fold really wrote.

type foldSpanStore struct {
	spans  []*foldedSpan
	nextID int64
}

func (s *foldSpanStore) GetOpenSpan(_ context.Context, arg db.GetOpenSpanParams) (db.GetOpenSpanRow, error) {
	for _, sp := range s.spans {
		if sp.closedAt.Valid || sp.SubjectKey != arg.SubjectKey || sp.Facet != arg.Facet ||
			sp.Discriminator != arg.Discriminator || sp.Source != arg.Source ||
			sp.VantageID != arg.VantageID {
			continue
		}
		return db.GetOpenSpanRow{
			ID: sp.id, SubjectKind: sp.SubjectKind, SubjectKey: sp.SubjectKey, Facet: sp.Facet,
			Discriminator: sp.Discriminator, VantageID: sp.VantageID, Source: sp.Source,
			Value: sp.Value, IsGap: sp.IsGap, Derivation: sp.Derivation, OpenedAt: sp.OpenedAt,
		}, nil
	}
	return db.GetOpenSpanRow{}, pgx.ErrNoRows
}

func (s *foldSpanStore) CloseSpan(_ context.Context, arg db.CloseSpanParams) error {
	for _, sp := range s.spans {
		// The query closes an open span only, and re-entry reads the reason back (ADR-0041).
		if sp.id != arg.ID || sp.closedAt.Valid {
			continue
		}
		sp.closedAt = arg.ClosedAt
		sp.closureReason = arg.ClosureReason
		sp.closedBatchID = arg.ClosedBatchID
	}
	return nil
}

func (s *foldSpanStore) OpenSpan(_ context.Context, arg db.OpenSpanParams) (int64, error) {
	s.nextID++
	s.spans = append(s.spans, &foldedSpan{OpenSpanParams: arg, id: s.nextID})
	return s.nextID, nil
}

func (s *foldSpanStore) ListSpansForSubject(_ context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error) {
	var out []db.ListSpansForSubjectRow
	for _, sp := range s.spans {
		if sp.SubjectKind != arg.SubjectKind || sp.SubjectKey != arg.SubjectKey {
			continue
		}
		out = append(out, db.ListSpansForSubjectRow{
			ID: sp.id, SubjectKind: sp.SubjectKind, SubjectKey: sp.SubjectKey, Facet: sp.Facet,
			Discriminator: sp.Discriminator, VantageID: sp.VantageID, Source: sp.Source,
			Value: sp.Value, IsGap: sp.IsGap, Derivation: sp.Derivation,
			OpenedAt: sp.OpenedAt, ClosedAt: sp.closedAt, ClosureReason: sp.closureReason,
		})
	}
	return out, nil
}

// The lower bound is the root's own batch, inclusive (db/queries/span.sql, #1816).

func (s *foldSpanStore) ListSubjectsOpenedSinceBatch(_ context.Context, batchID int64) ([]db.ListSubjectsOpenedSinceBatchRow, error) {
	seen := map[subjectRef]bool{}
	var out []db.ListSubjectsOpenedSinceBatchRow
	for _, sp := range s.spans {
		if !sp.OpenedBatchID.Valid || sp.OpenedBatchID.Int64 < batchID {
			continue
		}
		if sp.SubjectKind != subjectKindService && sp.SubjectKind != subjectKindEndpoint {
			continue
		}
		ref := subjectRef{kind: sp.SubjectKind, key: sp.SubjectKey}
		if seen[ref] {
			continue
		}
		seen[ref] = true
		out = append(out, db.ListSubjectsOpenedSinceBatchRow{SubjectKind: ref.kind, SubjectKey: ref.key})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SubjectKind != out[j].SubjectKind {
			return out[i].SubjectKind < out[j].SubjectKind
		}
		return out[i].SubjectKey < out[j].SubjectKey
	})
	return out, nil
}

type releaseRecorder struct {
	*foldSpanStore
	released []db.ReleaseHeldMessageParams
	claimed  map[int64]bool
}

// The guarded UPDATE is the claim, so a row already released takes none (db/queries/messages.sql).

func (r *releaseRecorder) ReleaseHeldMessage(_ context.Context, arg db.ReleaseHeldMessageParams) (int64, error) {
	if r.claimed[arg.ID] {
		return 0, nil
	}
	if r.claimed == nil {
		r.claimed = map[int64]bool{}
	}
	r.claimed[arg.ID] = true
	r.released = append(r.released, arg)
	return 1, nil
}

// A Kind is one job's whole population, so each of these is a separate dispatch (ADR-1806 §1).

func proofDNSObservations(addr string) []wire.Observation {
	return resolutionwalk.Emit("proof", "vantage", resolutionwalk.Result{
		Name: proofName,
		Resolution: resolutionwalk.Resolution{
			Outcome:   resolutionwalk.OutcomeResolved,
			Addresses: []string{addr},
		},
	})
}

func proofHotObservations(target netip.AddrPort) []wire.Observation {
	return []wire.Observation{
		connectoutcome.EmitService("proof", "vantage", target, connectoutcome.Reached, connectoutcome.ConnOpen),
		// A hot dispatch enumerates addresses, so the Endpoint it opens has no Name leg (#1774).
		connectoutcome.EmitCertificate("proof", "vantage", target, "", connectoutcome.HandshakeResult{
			Outcome: connectoutcome.TLSPresented,
			Chain:   []string{"sha256:0"},
		}),
	}
}

func proofLateScanObservations(t *testing.T) []wire.Observation {
	t.Helper()
	// tls-acceptance exports no acceptance value, so this observation is built from its parts.
	acceptance, err := json.Marshal(struct {
		Outcome  tlsacceptance.AcceptanceOutcome   `json:"outcome"`
		Versions []tlsacceptance.VersionAcceptance `json:"versions,omitempty"`
	}{Outcome: tlsacceptance.Enumerated, Versions: []tlsacceptance.VersionAcceptance{{Version: "1.3"}}})
	if err != nil {
		t.Fatalf("marshal the acceptance value: %v", err)
	}
	return []wire.Observation{
		httpexchange.EmitEndpoint("proof", "vantage",
			httpexchange.Target{Name: proofName, Address: proofAddr, Port: proofPort, Scheme: "https"},
			httpexchange.HTTPIdentity{Outcome: httpexchange.OutcomeResponded, Status: 200}),
		{
			Kind:    tlsacceptance.Kind,
			Facet:   tlsacceptance.Facet,
			Subject: tlsacceptance.ServiceKey(proofTarget, "tcp"),
			Vantage: "vantage",
			Address: proofAddr,
			Data:    acceptance,
		},
	}
}

func foldProofBatch(t *testing.T, spans *foldSpanStore, msgs *fakeMessageStore, batchID int64, at time.Time, obs []wire.Observation) {
	t.Helper()
	ctx := context.Background()
	var changes []spanChange
	if err := foldObservationsIntoSpans(ctx, spans, batchID, pgInt8(1), at, obs, membershipInputs{}, &changes); err != nil {
		t.Fatalf("fold batch %d: %v", batchID, err)
	}
	var log []routed
	first := len(msgs.inserted)
	// The fold holds its census exactly while the reaper runs, so it reads that knob (#1114).
	hold := HotLagGateArmed(DefaultStaleJobThreshold)
	if err := produceMessages(ctx, msgs, batchID, at, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false, hold); err != nil {
		t.Fatalf("produce for batch %d: %v", batchID, err)
	}
	routedHere := map[int64]bool{}
	for _, r := range log {
		routedHere[r.messageID] = true
	}
	for i := first; i < len(msgs.inserted); i++ {
		if msgs.inserted[i].CensusPendingAfterBatch.Valid && routedHere[int64(i+1)] {
			t.Errorf("the release poll alone routes a held row, got %+v", log)
		}
	}
}

func heldMembershipRow(t *testing.T, msgs *fakeMessageStore, rootKind string, batchID int64) db.ListReleasableHeldMessagesRow {
	t.Helper()
	for i := range msgs.inserted {
		m := msgs.inserted[i]
		if m.SubjectKind != rootKind || m.CensusPendingAfterBatch.Int64 != batchID {
			continue
		}
		if len(m.Census) != 0 {
			t.Errorf("a held row carries no census until release, got %s", m.Census)
		}
		return db.ListReleasableHeldMessagesRow{
			ID:                      int64(i + 1),
			Class:                   m.Class,
			Headline:                m.Headline,
			CensusPendingAfterBatch: m.CensusPendingAfterBatch,
			CensusBasis:             m.CensusBasis,
		}
	}
	t.Fatalf("the dns fold wrote no held %s membership row, got %+v", rootKind, msgs.inserted)
	return db.ListReleasableHeldMessagesRow{}
}

func censusKeys(t *testing.T, payload []byte) map[string]string {
	t.Helper()
	census, err := message.ParseCensus(payload)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	keys := make(map[string]string, census.Len())
	for _, e := range census.Entries {
		keys[e.Key] = e.Kind
	}
	return keys
}

func openedFacets(spans *foldSpanStore, batchID int64) map[string]bool {
	out := map[string]bool{}
	for _, sp := range spans.spans {
		if sp.OpenedBatchID.Valid && sp.OpenedBatchID.Int64 == batchID {
			out[sp.Facet] = true
		}
	}
	return out
}

func foldedFacets(spans *foldSpanStore) map[string]bool {
	out := map[string]bool{}
	for _, sp := range spans.spans {
		out[sp.Facet] = true
	}
	return out
}

// ADR-1806's proof: two folds, a drain, and the census the release computes from them.

func TestTheDeferredCensusNamesTheHotFoldsServiceAndNamelessEndpoint(t *testing.T) {
	spans := &foldSpanStore{}
	store := &releaseRecorder{foldSpanStore: spans}
	msgs := &fakeMessageStore{prev: prevAt(produceT0.Add(-time.Hour))}

	// The dns fold enters the Name and cites the address. Its own census would be empty.
	foldProofBatch(t, spans, msgs, proofDNSBatch, produceT0, proofDNSObservations(proofAddr))
	held := heldMembershipRow(t, msgs, subjectKindName, proofDNSBatch)
	cause := held.Headline

	// The hot fold opens the Service and the nameless Endpoint on the cited address.
	foldProofBatch(t, spans, msgs, proofHotBatch, produceT0.Add(time.Minute), proofHotObservations(proofTarget))

	var log []routed
	took, err := releaseHeldMessage(context.Background(), store, held, fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if !took {
		t.Fatal("the hot dispatch drained, so the pass releases the row")
	}

	keys := censusKeys(t, store.released[0].Census)
	if keys[proofSvc()] != subjectKindService {
		t.Errorf("the census names the Service the hot fold opened beneath the root, got %v", keys)
	}
	if keys[proofNamelessEP()] != subjectKindEndpoint {
		t.Errorf("the census names the nameless Endpoint the hot fold opened, got %v", keys)
	}
	if len(keys) != 2 {
		t.Errorf("the hot fold opened those two subjects and no other, got %v", keys)
	}

	want := cause + " · 1 endpoint + 1 service · 2 timelines opened beneath it"
	if got := store.released[0].Headline; got != want {
		t.Errorf("headline = %q, want the fold's cause clause plus the census clause %q", got, want)
	}
	if len(log) != 1 || log[0].messageID != held.ID || log[0].class != message.Class(held.Class) {
		t.Errorf("a released row enqueues its one delivery, got %+v", log)
	}
}

// An Address root is read from a move, so its census is a later hot fold's too (ADR-1806 §4).

func TestTheDeferredCensusOfAnAddressRootNamesWhatOpenedBeneathTheMove(t *testing.T) {
	spans := &foldSpanStore{}
	store := &releaseRecorder{foldSpanStore: spans}
	msgs := &fakeMessageStore{prev: prevAt(produceT0.Add(-time.Hour))}

	// The Name enters on the first address, and the hot tier opens beneath it.
	foldProofBatch(t, spans, msgs, proofDNSBatch, produceT0, proofDNSObservations(proofAddr))
	foldProofBatch(t, spans, msgs, proofHotBatch, produceT0.Add(time.Minute), proofHotObservations(proofTarget))

	// The Name re-points. The second address is new to the estate, so the move roots on it.
	foldProofBatch(t, spans, msgs, proofMoveBatch, produceT0.Add(2*time.Minute), proofDNSObservations(proofMoveAddr))
	held := heldMembershipRow(t, msgs, subjectKindAddress, proofMoveBatch)
	if held.Headline != proofMoveAddr+" entered the estate" {
		t.Errorf("the held row carries the fold's cause clause alone, got %q", held.Headline)
	}
	cause := held.Headline

	// The next hot fold opens the Service and the nameless Endpoint on the new address.
	foldProofBatch(t, spans, msgs, proofMoveHotBatch, produceT0.Add(3*time.Minute), proofHotObservations(proofMoveTarget))

	var log []routed
	took, err := releaseHeldMessage(context.Background(), store, held, fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if !took {
		t.Fatal("the hot dispatch drained, so the pass releases the row")
	}

	moveSvc := connectoutcome.ServiceKey(proofMoveTarget, "tcp")
	moveEP := connectoutcome.EndpointKey("", proofMoveTarget, "tcp")
	keys := censusKeys(t, store.released[0].Census)
	if keys[moveSvc] != subjectKindService || keys[moveEP] != subjectKindEndpoint {
		t.Errorf("the census names what the hot tier opened on the new address, got %v", keys)
	}
	if len(keys) != 2 {
		t.Errorf("only the new address is beneath this root, got %v", keys)
	}

	want := cause + " · 1 endpoint + 1 service · 2 timelines opened beneath it"
	if got := store.released[0].Headline; got != want {
		t.Errorf("headline = %q, want the fold's cause clause plus the census clause %q", got, want)
	}
	if len(log) != 1 || log[0].messageID != held.ID {
		t.Errorf("a released row enqueues its one delivery, got %+v", log)
	}
}

// The price ADR-0031 accepted and ADR-1806 §5 keeps, pinned so no later session widens it.

func TestTheDeferredCensusNamesNoHTTPIdentityAndNoTLSAcceptanceSubject(t *testing.T) {
	spans := &foldSpanStore{}
	store := &releaseRecorder{foldSpanStore: spans}
	msgs := &fakeMessageStore{prev: prevAt(produceT0.Add(-time.Hour))}

	foldProofBatch(t, spans, msgs, proofDNSBatch, produceT0, proofDNSObservations(proofAddr))
	held := heldMembershipRow(t, msgs, subjectKindName, proofDNSBatch)
	foldProofBatch(t, spans, msgs, proofHotBatch, produceT0.Add(time.Minute), proofHotObservations(proofTarget))

	// The row leaves on the drained hot dispatch, so neither Scan has folded yet (§3).
	before := foldedFacets(spans)
	if before[httpexchange.FacetHTTPIdentity] || before[tlsacceptance.Facet] {
		t.Fatalf("the release waits on the hot tier alone, got %v", before)
	}
	var log []routed
	took, err := releaseHeldMessage(context.Background(), store, held, fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if !took {
		t.Fatal("the hot dispatch drained, so the pass releases the row")
	}

	foldProofBatch(t, spans, msgs, proofLateBatch, produceT0.Add(2*time.Minute), proofLateScanObservations(t))
	facets := openedFacets(spans, proofLateBatch)
	if !facets[httpexchange.FacetHTTPIdentity] || !facets[tlsacceptance.Facet] {
		t.Fatalf("both Scans fold here, or this test asserts nothing; got %v", facets)
	}

	// tls-acceptance renders the Service key connect-outcome renders, so it enters no subject.
	fresh := subjectsFirstOpenedBy(spans, proofLateBatch)
	if len(fresh) != 1 || !fresh[proofNamedEP()] {
		t.Fatalf("the named Endpoint is the one subject those Scans enter, got %v", fresh)
	}
	root := basisRoot(mustBasis(t, held.CensusBasis))
	if !subjectBeneathRoot(root, citedAddresses(root), subjectKindEndpoint, proofNamedEP()) {
		t.Fatal("the named Endpoint sits beneath the root, or this test asserts nothing")
	}
	keys := censusKeys(t, store.released[0].Census)
	if _, ok := keys[proofNamedEP()]; ok {
		t.Errorf("the census carries no http-identity subject, got %v", keys)
	}

	// A second pass takes no row, so the one delivery stands (ADR-1806 §2).
	took, err = releaseHeldMessage(context.Background(), store, held, fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("second release: %v", err)
	}
	if took {
		t.Error("the row was released, so a later pass loses the claim")
	}
	if len(store.released) != 1 || len(log) != 1 {
		t.Errorf("a later Scan re-releases nothing, got %+v and %+v", store.released, log)
	}
}

// What this batch entered, rather than what it opened a further span on.

func subjectsFirstOpenedBy(spans *foldSpanStore, batchID int64) map[string]bool {
	earlier := map[string]bool{}
	for _, sp := range spans.spans {
		if sp.OpenedBatchID.Valid && sp.OpenedBatchID.Int64 < batchID {
			earlier[sp.SubjectKey] = true
		}
	}
	out := map[string]bool{}
	for _, sp := range spans.spans {
		if sp.OpenedBatchID.Valid && sp.OpenedBatchID.Int64 == batchID && !earlier[sp.SubjectKey] {
			out[sp.SubjectKey] = true
		}
	}
	return out
}

func mustBasis(t *testing.T, payload []byte) message.CensusBasis {
	t.Helper()
	basis, err := message.ParseCensusBasis(payload)
	if err != nil {
		t.Fatalf("parse basis: %v", err)
	}
	return basis
}
