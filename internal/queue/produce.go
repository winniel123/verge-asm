package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
)

// This file is P0.7 — the message PRODUCER wire (AUDIT-LEDGER AL-2). Every other
// piece of the message pipeline already existed and was tested — the pure
// constructors (message.Flagship, message.Membership), the store write
// (InsertMessage), the channel router (delivery.EnqueueForMessage), the Inbox,
// the shell bell, the transport and the retry/dead-letter curve — and only the
// producing wire was absent: no production code folded a signal/drift transition
// into a message. This is that wire:
//
//	signal / drift transition → message.Flagship / message.Membership
//	                          → InsertMessage → EnqueueForMessage
//
// It runs INSIDE the batch's own transaction (worker.complete, ADR-0007), after
// the value fold (foldObservationsIntoSpans) and the estate-transition fold
// (foldEstateTransitions, #637): a message is computed once, at the cause, and
// committed together with the spans that caused it, so a delivery later reads the
// one stored computation and never recomputes across a Break (CONTEXT.md
// `Message`, ADR-0064).
//
// Two legs, exactly the two the constructors model, and no new message class:
//
//   - the flagship — an internet `Reach` leg going not-reached → reached, the move
//     the product exists to catch (ADR-0029). It rides the class-composed internet
//     leg, so it is decided through the same pure engine the Exposure screen and
//     its vs-last-batch delta use (exposure.ComposeReach / exposure.Flagship over
//     ListServiceReachabilitySpansByClass now vs ...At the previous batch), never a
//     per-vantage move — a service reachable from one internet position is reachable
//     from the internet, and a leg that opened at reached is carried by the census,
//     not the flagship (message.Flagship returns nil for an opening).
//
//   - the membership entry — a Name (or Address) root entering the estate, carrying
//     the census of what opened beneath it (CONTEXT.md `Subject` §; ADR-0031). It
//     fires once, at the root of the entering sub-tree, because the entering
//     Services / Endpoints beneath open no message of their own and ride this
//     message's census instead.
//
// The producer is off by default: a Worker built without WithMessages writes no
// message, which keeps the measurement-only construction (and its tests) unchanged.
// cmd/worker opts in with the live delivery enqueuer. **VERGE_DEV guard (AL-25):**
// even opted-in, a devMode worker produces NOTHING — a real deployment never serves
// fixtures, and the golden fixtures must stay message-free so G2 does not move.

// enqueueFunc routes a freshly-written Message to its bound Channels by class and
// returns how many Deliveries it enqueued. It is delivery.EnqueueForMessage bound to
// the batch's own transaction at the call site, injected so internal/queue never
// imports internal/delivery (delivery imports queue for the shared backoff curve —
// the reverse edge would be an import cycle). A default install binds no Channel, so
// it enqueues zero and no channel POST is ever made.
type enqueueFunc func(ctx context.Context, messageID int64, class message.Class) (int, error)

// messageStore is the slice of the store the producer reads and writes: the
// reachability legs it composes the flagship transition from, the previous-batch
// boundary it reads the "before" leg at, and the message write itself. *db.Queries
// satisfies it (so the batch's qtx is passed straight in); a fake drives the unit
// tests with no Postgres.
type messageStore interface {
	PreviousBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	ListServiceReachabilitySpansByClass(ctx context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error)
	ListServiceReachabilitySpansByClassAt(ctx context.Context, at pgtype.Timestamptz) ([]db.ListServiceReachabilitySpansByClassAtRow, error)
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
}

// spanChange is one timeline the batch's value fold opened or moved — the estate
// feed the producer consumes (collected by foldObservationsIntoSpans). Opened marks
// a first span on the timeline (a membership / census candidate); a reachability
// change, opened or moved, is a flagship candidate. Value is the opened value, read
// only for a resolution root's cited addresses.
type spanChange struct {
	SubjectKind    string
	SubjectKey     string
	Facet          string
	Opened         bool
	OpenedAperture bool
	Value          []byte
}

// produceMessages folds a batch's estate/drift transitions into messages, writes
// each once, and routes it to its bound channels — all in the batch's transaction.
//
// The VERGE_DEV guard is first and unconditional (AL-25): a devMode worker returns
// before any read or write, so a fixture-only install writes no message and enqueues
// no delivery, and the golden fixtures stay message-free. A batch with no transition
// is likewise a no-op.
func produceMessages(ctx context.Context, store messageStore, batchID int64, observedAt time.Time, changes []spanChange, in membershipInputs, enqueue enqueueFunc, devMode bool) error {
	_ = batchID // the message links by fired-at subject key, not the batch id
	if devMode || len(changes) == 0 {
		return nil
	}
	msgs, err := buildMessages(ctx, store, observedAt, changes, in)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		row, err := store.InsertMessage(ctx, insertParams(m))
		if err != nil {
			return err
		}
		if enqueue == nil {
			continue
		}
		if _, err := enqueue(ctx, row.ID, m.Class); err != nil {
			return err
		}
	}
	return nil
}

// insertParams renders a computed Message as the store write. census is left NULL
// only where the firing carries none; a flagship or membership firing always carries
// one (possibly empty), so its bytes are the present-but-empty payload the panel reads.
func insertParams(m *message.Message) db.InsertMessageParams {
	p := db.InsertMessageParams{
		Cause:       string(m.Cause),
		Class:       string(m.Class),
		SubjectKind: m.SubjectKind,
		FiredAt:     m.FiredAt,
		Instant:     pgtype.Timestamptz{Time: m.Instant, Valid: !m.Instant.IsZero()},
		Headline:    m.Headline,
	}
	if m.Census != nil {
		if b, err := m.Census.Marshal(); err == nil {
			p.Census = b
		}
	}
	return p
}

// buildMessages composes the messages a batch's changes fire. Given the store's
// answers it is deterministic, so the whole producer is driven by a fake store in
// the unit tests. The flagship leg reads the class-composed internet leg; the
// membership leg reads only the changes and the declared-input context.
func buildMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, in membershipInputs) ([]*message.Message, error) {
	var msgs []*message.Message

	flagship, err := flagshipMessages(ctx, store, observedAt, changes)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, flagship...)

	msgs = append(msgs, membershipMessages(observedAt, changes, in)...)
	return msgs, nil
}

// flagshipMessages fires one message.Flagship per Service whose class-composed
// internet leg moved not-reached → reached in this batch. Candidates are the
// Services this batch actually touched (a reachability change), so a rescan that
// moved nothing fires nothing and a batch on one Service never fires for another.
// The transition is decided against the internet leg as it stood a batch ago
// (ListServiceReachabilitySpansByClassAt at PreviousBatchTime) versus now, through
// the pure exposure engine — the same reconstruction the Exposure delta uses. A leg
// that only opened this batch has no decided "before", so exposure.Flagship reports
// no move and the news rides the membership census instead (ADR-0029).
func flagshipMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange) ([]*message.Message, error) {
	services := flagshipCandidateServices(changes)
	if len(services) == 0 {
		return nil, nil
	}

	current, err := store.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		return nil, err
	}
	curLegs := legsFromCurrent(current)

	var prevLegs []classLeg
	prev, err := store.PreviousBatchTime(ctx)
	if err != nil {
		return nil, err
	}
	if prev.Valid {
		past, err := store.ListServiceReachabilitySpansByClassAt(ctx, prev)
		if err != nil {
			return nil, err
		}
		prevLegs = legsFromAt(past)
	}

	var msgs []*message.Message
	for _, svc := range services {
		after, aok := composeInternetLeg(curLegs, svc)
		before, bok := composeInternetLeg(prevLegs, svc)
		if !aok || !bok || !exposure.Flagship(before, after) {
			continue
		}
		m := message.Flagship(message.ReachMove{
			ServiceKey: svc,
			Class:      message.ClassInternet,
			From:       message.NotReached,
			To:         message.Reached,
		}, flagshipCensus(changes, svc), observedAt)
		if m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

// membershipMessages fires one message.Membership at the root of each entering
// sub-tree — a Name (or Address) whose membership-bearing timeline first opened this
// batch. Membership rides the `resolution` facet (internal/estate), so a Name's
// entry is its first `resolution` span; its sibling `dns-record` opening is not a
// second root. The Entry follows the aperture marker the fold stamped: an
// aperture-driven opening `revealed` (coverage class, fired at the Seed whose scope
// moved), a world-brought opening `appeared` (drift class, fired at the subject).
//
// `returned` (a re-entry across a withdrawal closure) shares `appeared`'s cause,
// class, census and fired-at — they differ only in the headline word — so a first
// span with no open predecessor reads `appeared` here without the extra prior-closed
// -span read; the distinction is a headline refinement, never a routing one.
func membershipMessages(observedAt time.Time, changes []spanChange, in membershipInputs) []*message.Message {
	var msgs []*message.Message
	for _, root := range changes {
		if !root.Opened || root.Facet != resolutionwalk.FacetResolution || !message.RootFires(root.SubjectKind) {
			continue
		}
		entry := message.EntryAppeared
		seedKey := ""
		if root.OpenedAperture {
			entry = message.EntryRevealed
			seedKey = coveringSeedKey(root.SubjectKind, root.SubjectKey, in)
		}
		m := message.Membership(entry, root.SubjectKind, root.SubjectKey, seedKey, membershipCensus(changes, root), observedAt)
		if m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

// classLeg is one (service, class) reachability leg normalized across the current
// and as-of reads, which return distinct row types with identical fields.
type classLeg struct {
	subject string
	class   string
	outcome string
}

func legsFromCurrent(rows []db.ListServiceReachabilitySpansByClassRow) []classLeg {
	out := make([]classLeg, 0, len(rows))
	for _, r := range rows {
		out = append(out, classLeg{subject: r.SubjectKey, class: r.Class, outcome: reachOutcome(r.Value)})
	}
	return out
}

func legsFromAt(rows []db.ListServiceReachabilitySpansByClassAtRow) []classLeg {
	out := make([]classLeg, 0, len(rows))
	for _, r := range rows {
		out = append(out, classLeg{subject: r.SubjectKey, class: r.Class, outcome: reachOutcome(r.Value)})
	}
	return out
}

// composeInternetLeg applies the existential internet-class Reach composition over
// one Service's internet-class legs (exposure.ComposeReach, ADR-0080): reached where
// any internet vantage reached it, not-reached where one decided so and none reached,
// and undecided ("", false) where no internet vantage decided at all — an empty class
// or a leg that only just opened, which is not vacuously not-reached.
func composeInternetLeg(legs []classLeg, service string) (exposure.ReachValue, bool) {
	var outcomes []string
	for _, l := range legs {
		if l.subject == service && l.class == "internet" {
			outcomes = append(outcomes, l.outcome)
		}
	}
	return exposure.ComposeReach(outcomes)
}

// flagshipCandidateServices is the distinct set of Service keys this batch moved on
// the `reachability` facet, in first-seen order — the Services whose internet leg the
// flagship pass re-composes. A Service the batch did not touch is never re-composed,
// so an unchanged leg never re-fires.
func flagshipCandidateServices(changes []spanChange) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range changes {
		if c.Facet != connectoutcome.FacetReachability || c.SubjectKey == "" {
			continue
		}
		if seen[c.SubjectKey] {
			continue
		}
		seen[c.SubjectKey] = true
		out = append(out, c.SubjectKey)
	}
	return out
}

// flagshipCensus is the census a flagship message carries: every facet timeline that
// opened beneath the newly-reached Service this batch — certificate, http-identity,
// tls-acceptance and the rest — since an opening reaches nobody on its own and rides
// the flagship census instead (CONTEXT.md `Reach`; message.Census). The reachability
// leg that fired is not itself a census facet. Facets nest under the Service by key:
// tls-acceptance keys on the Service, certificate/http-identity on an Endpoint whose
// key carries the Service key.
func flagshipCensus(changes []spanChange, service string) message.Census {
	seen := map[string]bool{}
	var entries []message.CensusEntry
	for _, c := range changes {
		if !c.Opened || c.Facet == "" || c.Facet == connectoutcome.FacetReachability {
			continue
		}
		if !keyNestsService(service, c.SubjectKey) || seen[c.Facet] {
			continue
		}
		seen[c.Facet] = true
		entries = append(entries, message.CensusEntry{Kind: "facet", Key: c.Facet})
	}
	return message.NewCensus(entries...)
}

// membershipCensus is the census a membership message carries: every Subject
// (Service / Endpoint) that entered beneath the root this batch — the only carrier a
// new subject has, since every timeline beneath the root opens and no alerting
// predicate is opening-shaped, so the entering subjects appear here rather than as
// their own messages (CONTEXT.md `Subject`, ADR-0031). A subject sits beneath a Name
// root where its key carries the Name, or where it sits on one of the Addresses the
// Name's resolution cited; beneath an Address root where its key carries the Address.
// One entry per distinct subject, in deterministic order (NewCensus sorts).
func membershipCensus(changes []spanChange, root spanChange) message.Census {
	cited := citedAddresses(root)
	seen := map[[2]string]bool{}
	var entries []message.CensusEntry
	for _, c := range changes {
		if !c.Opened || c.SubjectKey == root.SubjectKey {
			continue
		}
		if c.SubjectKind != "service" && c.SubjectKind != "endpoint" {
			continue
		}
		if !subjectBeneathRoot(root, cited, c.SubjectKey) {
			continue
		}
		key := [2]string{c.SubjectKind, c.SubjectKey}
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, message.CensusEntry{Kind: c.SubjectKind, Key: c.SubjectKey})
	}
	return message.NewCensus(entries...)
}

// citedAddresses reads the Addresses a Name root's `resolution` value cites, so the
// census can gather the Services standing on them. A value that carries none (a
// NameError / NoData / Shadowed / Gap outcome, or an address root) yields no cited
// set, and the census then rests on key containment alone.
func citedAddresses(root spanChange) map[string]bool {
	if root.SubjectKind != subjectKindName || len(root.Value) == 0 {
		return nil
	}
	var v struct {
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(root.Value, &v); err != nil || len(v.Addresses) == 0 {
		return nil
	}
	out := make(map[string]bool, len(v.Addresses))
	for _, a := range v.Addresses {
		out[a] = true
	}
	return out
}

// subjectBeneathRoot reports whether a Service/Endpoint key sits beneath a membership
// root — under the Name (its key carries the Name, or it stands on a cited Address)
// or under the Address (its key carries the Address).
func subjectBeneathRoot(root spanChange, cited map[string]bool, key string) bool {
	switch root.SubjectKind {
	case subjectKindName:
		if strings.Contains(key, root.SubjectKey) {
			return true
		}
		return cited[serviceAddress(key)]
	case "address":
		return serviceAddress(key) == root.SubjectKey || strings.HasPrefix(key, root.SubjectKey)
	default:
		return false
	}
}

// keyNestsService reports whether a facet's subject key sits on the Service — the
// Service itself (its own facets) or an Endpoint on it (whose key carries the Service
// key). Used to gather the facets that opened beneath a newly-reached Service.
func keyNestsService(service, key string) bool {
	return key == service || strings.Contains(key, service)
}

// coveringSeedKey is the scope key of the Seed whose aperture reveals a root — the
// key a `revealed` membership message fires at and links to (ADR §5.3). A Name root
// is revealed by the covering name Seed (its declared domain); an Address root by the
// covering address Seed (its declared CIDR). Empty where no Seed covers it, in which
// case the opening was `appeared`, not `revealed`, and this is not consulted.
func coveringSeedKey(rootKind, rootKey string, in membershipInputs) string {
	switch rootKind {
	case subjectKindName:
		name := normalizeDomain(rootKey)
		for _, s := range in.seeds {
			if s.Kind == "name" && s.NameDomain.Valid && nameWithinDomain(name, s.NameDomain.String) {
				return normalizeDomain(s.NameDomain.String)
			}
		}
	case "address":
		if addr, err := netip.ParseAddr(rootKey); err == nil {
			for _, s := range in.seeds {
				if s.Kind == "address" && s.AddressCidr != nil && s.AddressCidr.Contains(addr) {
					return s.AddressCidr.String()
				}
			}
		}
	}
	return ""
}

// reachOutcome reads the `outcome` tag off a `reachability` span value
// (`{"outcome":"reached"│"not-reached"│"gap", ...}`). A value that will not parse
// reads as an empty outcome, which exposure.ComposeReach treats as undecided — the
// safe reading, since a malformed value is not affirmative evidence of a reach.
func reachOutcome(value []byte) string {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(value, &v)
	return v.Outcome
}
