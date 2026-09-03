package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/exposure"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/vantageclass"
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
	// ListAddressScopeCidrs is the declared address-scope corpus the flagship's Vantage
	// class is DERIVED against per read (#709/#711) — the same read hotEstate uses.
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	// AddressExclusionStore is the declared `address` exclusions, which narrow that
	// same predicate (ADR-0133 §4). It is embedded rather than restated, so this
	// surface and the gate cannot read a different exclusion set.
	AddressExclusionStore
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
}

type spanChange struct {
	SubjectKind    string
	SubjectKey     string
	Facet          string
	Opened         bool
	OpenedAperture bool
	Value          []byte
}

// departure is one subject the batch's estate fold closed out of the estate — the
// withdrawal/exclusion feed the declared-input producer consumes (AL-2, #722,
// lighting the `declared-input` cause the "what fires" table promised). It is
// collected by foldEstateTransitions (internal/queue/membership.go) alongside the
// closure it applies, so the producer folds the operator-caused departure into a
// message in the same batch transaction the closure committed in.
//
// Only an operator-caused departure is a `declared-input` firing: a `descoped`
// closure is the operator's own declared Exclusion narrowing the estate over the
// subject (CauseDeclaredInput — "the operator's own declared input moved",
// message.DeclaredInput, links to the Exclusion as the Source). A `measured-absent`
// closure is the WORLD withdrawing the subject — drift, not declared input — so it
// carries SourceKey "" and folds into no message here (there is no drift-exit
// constructor, and the `drift` cause is already lit by the flagship leg).
type departure struct {
	SubjectKind string
	SubjectKey  string
	Reason      string
	SourceKey   string
	// Timelines is the count of open timelines the withdrawal closed at once
	// (ADR-0082) — one cause recorded on n objects, the count the headline states.
	Timelines int
}

// produceMessages folds a batch's estate/drift transitions into messages, writes
// each once, and routes it to its bound channels — all in the batch's transaction.
//
// The VERGE_DEV guard is first and unconditional (AL-25): a devMode worker returns
// before any read or write, so a fixture-only install writes no message and enqueues
// no delivery, and the golden fixtures stay message-free. A batch with no transition
// is likewise a no-op.
func produceMessages(ctx context.Context, store messageStore, batchID int64, observedAt time.Time, changes []spanChange, departures []departure, narrowings []message.NarrowingReceipt, in membershipInputs, enqueue enqueueFunc, devMode bool) error {
	_ = batchID // the message links by fired-at subject key, not the batch id
	if devMode || (len(changes) == 0 && len(departures) == 0 && len(narrowings) == 0) {
		return nil
	}
	msgs, err := buildMessages(ctx, store, observedAt, changes, departures, narrowings, in)
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
func buildMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, departures []departure, narrowings []message.NarrowingReceipt, in membershipInputs) ([]*message.Message, error) {
	var msgs []*message.Message

	flagship, err := flagshipMessages(ctx, store, observedAt, changes)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, flagship...)

	msgs = append(msgs, membershipMessages(observedAt, changes, in)...)
	msgs = append(msgs, declaredInputMessages(observedAt, departures)...)
	msgs = append(msgs, narrowingMessages(observedAt, narrowings)...)
	return msgs, nil
}

// narrowingMessages fires one coverage-class message.Narrowing per narrowing act
// whose withdrawal this batch applied (ADR-0074). Both acts CONTEXT.md names
// arrive here on one slice: an address Exclusion (ADR-0133 §8, #1032) and a
// withdrawn address Seed (ADR-0134, #1040). They differ only in the sentence their
// receipt already carries — the two constructors render it — so this producer has
// no branch. Until #1032 only PreviewNarrowing was called, so
// `POST /exclusions/preview` promised a message that no path ever wrote.
//
// It fires at the Seed scope the narrowing moved, carries the two counts and no
// rows, and stays silent over an uninhabited withdrawn set — message.Narrowing
// returns nil where the receipt does not fire, which is the same gate the preview
// applies, so the operator is never shown a receipt for a message that will not
// come. One message per act, not one per subject: the count IS the payload
// (message.NarrowingReceipt), and a row per withdrawn subject would be the census
// the receipt exists to replace.
func narrowingMessages(observedAt time.Time, narrowings []message.NarrowingReceipt) []*message.Message {
	var msgs []*message.Message
	for _, r := range narrowings {
		if m := message.Narrowing(r, r.Scope, observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

// declaredInputMessages fires one coverage-class message.DeclaredInput per subject
// the operator's own declared input withdrew this batch — a `descoped` departure,
// where an Exclusion the operator declared narrowed the estate over the subject
// (AL-2, #722). This is the `declared-input` cause the "what fires" table names
// ("the operator's own declared input moved") and lights the producer wire that was
// dark: produce.go read only the openings/moves change-feed, so a withdrawal by
// declared exclusion never became a message.
//
// Only a `descoped` closure fires here. A `measured-absent` closure is the world
// withdrawing the subject — a drift departure, not a declared-input one — and it
// carries no SourceKey and no message (there is no drift-exit constructor; the
// `drift` cause is already carried by the flagship leg). The message links to the
// Exclusion as its Source (message.DeclaredInput → LinkSource, ADR §5.3), and the
// departed subject and the count of timelines it took out ride the headline.
func declaredInputMessages(observedAt time.Time, departures []departure) []*message.Message {
	var msgs []*message.Message
	for _, d := range departures {
		if d.Reason != string(drift.ReasonDescoped) || d.SourceKey == "" {
			continue
		}
		m := message.DeclaredInput(d.SourceKey, declaredInputHeadline(d.SubjectKey, d.SourceKey, d.Timelines), observedAt)
		if m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

// declaredInputHeadline states the operator's declared exclusion taking a subject
// out of the estate, with the count of timelines it closed as its factor. The words
// are the drift/coverage vocabulary's own (`withdrawn`, `narrowed`, `taken out`) and
// carry no valence — a narrowing is neither good news nor bad (ADR-0064) — mirroring
// message.narrowingHeadline's shape.
func declaredInputHeadline(subjectKey, sourceKey string, timelines int) string {
	tl := "timelines"
	if timelines == 1 {
		tl = "timeline"
	}
	return fmt.Sprintf("%s withdrawn by declared exclusion %s · %d %s taken out of the estate",
		subjectKey, sourceKey, timelines, tl)
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

	// Vantage class is DERIVED per read from each vantage's presented-address facts
	// against the declared address scopes (#709), so the flagship's internet leg is
	// composed over the vantages that verify `internet` this read — never a stored
	// column. covered is the address-scope-only predicate (#711), assembled once from
	// the same corpus hotEstate uses, and shared across the current and previous reads.
	covered, err := coveredAddressScope(ctx, store)
	if err != nil {
		return nil, err
	}

	current, err := store.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		return nil, err
	}
	curLegs := legsFromCurrent(current, covered)

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
		prevLegs = legsFromAt(past, covered)
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

type classLeg struct {
	subject string
	class   string
	outcome string
}

// legsFromCurrent normalizes the current per-vantage reachability rows into classLegs,
// DERIVING each vantage's class from its presented-address facts (#709) rather than a
// stored column. Every internet-class vantage's leg survives — composeInternetLeg then
// applies the existential quantifier over them (ADR-0080), instead of the SQL
// pre-collapsing to a single row per class.
func legsFromCurrent(rows []db.ListServiceReachabilitySpansByClassRow, covered func(netip.Addr) bool) []classLeg {
	out := make([]classLeg, 0, len(rows))
	for _, r := range rows {
		class := string(vantageclass.Derive(r.DialledAddr.String, r.Egress.String, covered))
		out = append(out, classLeg{subject: r.SubjectKey, class: class, outcome: reachOutcome(r.Value)})
	}
	return out
}

func legsFromAt(rows []db.ListServiceReachabilitySpansByClassAtRow, covered func(netip.Addr) bool) []classLeg {
	out := make([]classLeg, 0, len(rows))
	for _, r := range rows {
		class := string(vantageclass.Derive(r.DialledAddr.String, r.Egress.String, covered))
		out = append(out, classLeg{subject: r.SubjectKey, class: class, outcome: reachOutcome(r.Value)})
	}
	return out
}

// coveredAddressScope assembles the address-scope coverage predicate the flagship's
// Vantage-class derivation binds (#711) from the declared address-scope Seeds — the same
// corpus and the same family-matched containment hotEstate uses at the fan-out side, so
// batch gating and the flagship classify against one identical corpus. It routes through
// custody.Estate.CoversAddressScope (address scopes ALONE — never the extension).
//
// It reads the declared `address` EXCLUSIONS too, so the predicate narrows with the
// derivation (ADR-0133 §4). A vantage whose egress sits inside an excluded range
// stops being covered and exposure.VerifyClass may reclassify it. That is intended:
// the operator has said the range is not theirs, so a prober inside it is not inside
// the estate. A second, un-narrowed predicate for classification alone would leave
// two coverage rules a later session has to hold in step, which is what #711's
// one-binding invariant refuses.
func coveredAddressScope(ctx context.Context, store messageStore) (func(netip.Addr) bool, error) {
	scopes, err := store.ListAddressScopeCidrs(ctx)
	if err != nil {
		return nil, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, *p)
		}
	}
	excluded, err := ReadAddressExclusions(ctx, store)
	if err != nil {
		return nil, err
	}
	return custody.Estate{AddressScopes: prefixes}.WithAddressExclusions(excluded).CoversAddressScope, nil
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

func reachOutcome(value []byte) string {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(value, &v)
	return v.Outcome
}
