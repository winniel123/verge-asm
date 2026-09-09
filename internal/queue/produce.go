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

// delivery imports queue, so the reverse edge would cycle (ADR-0199 §1, #1316).

type enqueueFunc func(ctx context.Context, messageID int64, class message.Class) (int, error)

type messageStore interface {
	PreviousBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	ListServiceReachabilitySpansByClassForServices(ctx context.Context, serviceKeys []string) ([]db.ListServiceReachabilitySpansByClassForServicesRow, error)
	ListServiceReachabilitySpansByClassAtForServices(ctx context.Context, arg db.ListServiceReachabilitySpansByClassAtForServicesParams) ([]db.ListServiceReachabilitySpansByClassAtForServicesRow, error)
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	AddressExclusionStore
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
	ListOpenSpansForSubject(ctx context.Context, arg db.ListOpenSpansForSubjectParams) ([]db.ListOpenSpansForSubjectRow, error)
	ListAnnotations(ctx context.Context) ([]db.Annotation, error)
	ListNameCitationSpansWithinCurrency(ctx context.Context, arg db.ListNameCitationSpansWithinCurrencyParams) ([]db.ListNameCitationSpansWithinCurrencyRow, error)
	FoldedBatchWindow(ctx context.Context, unfoldedKinds []string) (db.FoldedBatchWindowRow, error)
	ListOpenEndpointCertificateSpans(ctx context.Context) ([]db.ListOpenEndpointCertificateSpansRow, error)
}

type spanChange struct {
	SubjectKind    string
	SubjectKey     string
	Facet          string
	Opened         bool
	OpenedAperture bool
	Value          []byte
	Previous       []byte // The closed span's value; nil where the timeline opened (ADR-0033 §2).
	Vector         drift.Vector
	PrevVector     drift.Vector
	PriorClosure   *drift.Span
	WitnessBroke   bool
}

type departure struct {
	SubjectKind string
	SubjectKey  string
	Reason      string
	SourceKey   string
	Timelines   int
}

func produceMessages(ctx context.Context, store messageStore, batchID int64, observedAt time.Time, changes []spanChange, departures []departure, narrowings []message.NarrowingReceipt, in membershipInputs, enqueue enqueueFunc, devMode bool) error {
	_ = batchID // a message links by fired-at subject key, never the batch id (ADR-0064)
	// A devMode worker produces nothing, so a fixture install never pages anyone (ADR-0197 §1).
	if devMode {
		return nil
	}
	// A message is computed once at the cause and committed with its spans (ADR-0064).
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
	// A membership withdrawal is written and never routed, so it skips the enqueue (ADR-0087).
	for _, m := range withdrawalMessages(observedAt, departures) {
		if _, err := store.InsertMessage(ctx, insertParams(m)); err != nil {
			return err
		}
	}
	return nil
}

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

func buildMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange, departures []departure, narrowings []message.NarrowingReceipt, in membershipInputs) ([]*message.Message, error) {
	var msgs []*message.Message

	flagship, err := flagshipMessages(ctx, store, observedAt, changes)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, flagship...)

	msgs = append(msgs, membershipMessages(observedAt, changes, in)...)

	// The gate opening under a standing declaration is coverage, as revealed is (ADR-0013 #55).
	gains, err := extensionGainMessages(ctx, store, observedAt, changes, in)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, gains...)

	msgs = append(msgs, rebaselineMessages(observedAt, changes)...)

	// Composed after every census producer, so the residue clause can consult them (ADR-0033 §3).
	moves, err := facetMoveMessages(ctx, store, observedAt, changes, msgs)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, moves...)

	// A census above the edge names it, so this runs after every census producer (ADR-0026 §6).
	edges, err := signalEdgeMessages(ctx, store, observedAt, changes, msgs)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, edges...)

	// A batch that moved nothing still folds, so the clock's crossings are read here (#1728).
	clock, err := certificateLifetimeMessages(ctx, store, observedAt, changes, msgs)
	if err != nil {
		return nil, err
	}
	msgs = append(msgs, clock...)

	msgs = append(msgs, declaredInputMessages(observedAt, departures)...)
	msgs = append(msgs, narrowingMessages(observedAt, narrowings)...)
	return msgs, nil
}

func narrowingMessages(observedAt time.Time, narrowings []message.NarrowingReceipt) []*message.Message {
	var msgs []*message.Message
	// Both narrowing acts differ only in the receipt wording, so this needs no branch (ADR-0134).
	for _, r := range narrowings {
		// This nil gate must match the preview's, or an operator sees a receipt for nothing.
		if m := message.Narrowing(r, r.Scope, observedAt); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

func declaredInputMessages(observedAt time.Time, departures []departure) []*message.Message {
	var msgs []*message.Message
	for _, d := range departures {
		// Only a descoped closure fires here: measured-absent is drift, not declared input (#722).
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

func declaredInputHeadline(subjectKey, sourceKey string, timelines int) string {
	// A narrowing is neither good news nor bad, so the headline carries no valence (ADR-0064).
	tl := "timelines"
	if timelines == 1 {
		tl = "timeline"
	}
	return fmt.Sprintf("%s withdrawn by declared exclusion %s · %d %s taken out of the estate",
		subjectKey, sourceKey, timelines, tl)
}

func flagshipMessages(ctx context.Context, store messageStore, observedAt time.Time, changes []spanChange) ([]*message.Message, error) {
	services := flagshipCandidateServices(changes)
	if len(services) == 0 {
		return nil, nil
	}

	covered, err := coveredAddressScope(ctx, store)
	if err != nil {
		return nil, err
	}

	current, err := store.ListServiceReachabilitySpansByClassForServices(ctx, services)
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
		past, err := store.ListServiceReachabilitySpansByClassAtForServices(ctx, db.ListServiceReachabilitySpansByClassAtForServicesParams{
			ServiceKeys: services,
			At:          prev,
		})
		if err != nil {
			return nil, err
		}
		prevLegs = legsFromAt(past, covered)
	}

	var msgs []*message.Message
	for _, svc := range services {
		after, aok := composeInternetLeg(curLegs, svc)
		before, bok := composeInternetLeg(prevLegs, svc)
		// A leg that only opened has no decided "before", so the census carries it (ADR-0029).
		if !aok || !bok || !exposure.Flagship(before, after) {
			continue
		}
		census, err := flagshipCensusWithRules(ctx, store, observedAt, changes, svc, flagshipCensus(changes, svc))
		if err != nil {
			return nil, err
		}
		m := message.Flagship(message.ReachMove{
			ServiceKey: svc,
			Class:      message.ClassInternet,
			From:       message.NotReached,
			To:         message.Reached,
		}, census, observedAt)
		if m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

func membershipMessages(observedAt time.Time, changes []spanChange, in membershipInputs) []*message.Message {
	var msgs []*message.Message
	// Membership rides the resolution facet, so a dns-record opening is no second root (ADR-0031).
	for _, root := range changes {
		if !root.Opened || root.Facet != resolutionwalk.FacetResolution || !message.RootFires(root.SubjectKind) {
			continue
		}
		entry := membershipEntry(root)
		seedKey := ""
		if entry == message.EntryRevealed {
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

func legsFromCurrent(rows []db.ListServiceReachabilitySpansByClassForServicesRow, covered func(netip.Addr) bool) []classLeg {
	// The class is derived per read from presented-address facts, never a stored column (#709).
	out := make([]classLeg, 0, len(rows))
	// No leg is pre-collapsed here, so the existential quantifier applies later (ADR-0080).
	for _, r := range rows {
		class := string(vantageclass.Derive(r.DialledAddr.String, r.Egress.String, covered))
		out = append(out, classLeg{subject: r.SubjectKey, class: class, outcome: reachOutcome(r.Value)})
	}
	return out
}

func legsFromAt(rows []db.ListServiceReachabilitySpansByClassAtForServicesRow, covered func(netip.Addr) bool) []classLeg {
	out := make([]classLeg, 0, len(rows))
	for _, r := range rows {
		class := string(vantageclass.Derive(r.DialledAddr.String, r.Egress.String, covered))
		out = append(out, classLeg{subject: r.SubjectKey, class: class, outcome: reachOutcome(r.Value)})
	}
	return out
}

func coveredAddressScope(ctx context.Context, store messageStore) (func(netip.Addr) bool, error) {
	// One coverage rule only: an excluded egress may reclassify a vantage, as intended (#711).
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

func composeInternetLeg(legs []classLeg, service string) (exposure.ReachValue, bool) {
	var outcomes []string
	for _, l := range legs {
		if l.subject == service && l.class == "internet" {
			outcomes = append(outcomes, l.outcome)
		}
	}
	return exposure.ComposeReach(outcomes)
}

func flagshipCandidateServices(changes []spanChange) []string {
	// A Service the batch did not touch is never re-composed, so an unchanged leg never re-fires.
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

func flagshipCensus(changes []spanChange, service string) message.Census {
	// An opening reaches nobody on its own, so it rides this census instead (CONTEXT.md Reach).
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
	// An Endpoint key carries its Service key, so containment is how a facet nests (CONTEXT.md).
	return key == service || strings.Contains(key, service)
}

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
