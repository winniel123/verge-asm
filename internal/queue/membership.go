package queue

import (
	"context"
	"encoding/json"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/estate"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

// This file is the estate wiring into the spanfold closure (#637, SPEC-CHANGE
// ruling #32). foldObservationsIntoSpans tracks per-timeline VALUE movement — the
// `appeared` and `changed` transitions — and deliberately decides no withdrawal:
// whether a subject LEFT the estate is a subject-level cross-class composition
// (internal/estate), and closing its timelines with a `measured-absent` /
// `uncited` / `descoped` ground is that path's job (spanfold.go). Until this file
// existed, internal/estate had no production caller at all, so four of the drift
// legend's six transition words could never fire: `withdrawn` and `descoped` need
// a reasoned closure no path wrote, `returned` needs a re-open across such a
// closure, and `revealed` needs an aperture-widened opening the fold never marked.
//
// foldEstateTransitions runs after the value fold, inside the same batch
// transaction (ADR-0007), and closes the timelines of every subject the batch's
// observations show has left — composing the departure through internal/estate and
// applying it through drift's CloseWithdrawal grammar (drift.CloseSpan citing the
// folding batch, ADR-0111, so the estate-wide feed pairs the close into a
// transition). The aperture-widened OPENING half rides the value fold itself
// (foldOne stamps opened_aperture when a Seed-declared subject's timeline first
// opens), so this file owns only the closing half plus the membership decision the
// opening half consults.

// membershipInputs is the declared-input context a batch fold composes withdrawal
// against — the Seeds and Exclusions as they stand at fold time. It is read once
// per batch (readMembershipInputs) and threaded through both the value fold (for
// the aperture-widened opening marker) and the withdrawal closure below.
type membershipInputs struct {
	seeds      []db.ListSeedsRow
	exclusions []db.ListExclusionsRow
}

// readMembershipInputs loads the declared-input context for a batch fold. An error
// reading either corpus fails the fold — a withdrawal composed against a partial
// declared input could wrongly conclude a subject left (ADR-0001: never silently
// misparse into false exposure drift).
func readMembershipInputs(ctx context.Context, qtx *db.Queries) (membershipInputs, error) {
	seeds, err := qtx.ListSeeds(ctx)
	if err != nil {
		return membershipInputs{}, err
	}
	exclusions, err := qtx.ListExclusions(ctx)
	if err != nil {
		return membershipInputs{}, err
	}
	return membershipInputs{seeds: seeds, exclusions: exclusions}, nil
}

// foldEstateTransitions closes the timelines of every Name this batch's
// observations show has left the estate, composing the departure through
// internal/estate. It runs after foldObservationsIntoSpans, in the same
// transaction: the value fold has already opened the Name's current resolution
// span, so the cross-class composition reads the just-folded truth.
//
// It is scoped to the Names the batch actually observed (a `resolution` fold): a
// membership decision is only re-made where fresh evidence about the subject
// arrived, never as a background sweep of the whole estate. For each such Name the
// closure ground is decided by decideNameDeparture, and where it left every open
// timeline the Name holds is closed at the batch instant with that ground, citing
// the batch — exactly the shape drift.CloseWithdrawal describes, applied through
// the store.
func foldEstateTransitions(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, obs []wire.Observation, in membershipInputs, deps *[]departure) error {
	for _, name := range observedResolutionNames(obs) {
		open, err := qtx.ListOpenSpansForSubject(ctx, db.ListOpenSpansForSubjectParams{
			SubjectKind: subjectKindName,
			SubjectKey:  name,
		})
		if err != nil {
			return err
		}
		reason, left := decideNameDeparture(open, nameSeedCovered(name, in.seeds), nameExcluded(name, in.exclusions))
		if !left {
			continue
		}
		if err := closeSubjectTimelines(ctx, qtx, open, observedAt, reason, batchID); err != nil {
			return err
		}
		// Record the departure for the message producer (P0.7 / AL-2, #722): an
		// operator-caused `descoped` closure is a `declared-input` firing, linking to
		// the covering Exclusion as its Source; a world `measured-absent` closure
		// carries no source and fires no declared-input message. A nil collector is the
		// measurement-only path that produces no message.
		if deps != nil {
			*deps = append(*deps, departure{
				SubjectKind: subjectKindName,
				SubjectKey:  name,
				Reason:      string(reason),
				SourceKey:   coveringExclusionKey(name, reason, in.exclusions),
				Timelines:   len(open),
			})
		}
	}
	return nil
}

// coveringExclusionKey is the declared value of the Exclusion that descoped a
// subject — the Source identity a `declared-input` message links to
// (message.DeclaredInput → LinkSource). It is consulted only for a `descoped`
// departure: a world withdrawal (measured-absent) has no declared-input mover, so
// it returns "".
//
// It mirrors nameExcluded's coverage test (an exact `name` exclusion or a
// `subtree` exclusion of the Name or an ancestor) and returns the matching
// Exclusion's normalized declared name, so the same declared boundary that removed
// the Name is the key the message cites.
//
// It stays a NAME helper. #1032 reads its skip of every row carrying no name as
// half the address gap, and the address limb is answered by
// coveringAddressExclusion (below) rather than by a second branch here. The reason
// is the message shape: an address withdrawal fires ONE aggregate
// message.Narrowing per exclusion, not one declared-input row per subject, so the
// address side attributes its mover once, in composeAddressWithdrawals
// (internal/queue/withdrawal.go). A branch here would have no caller.
func coveringExclusionKey(name string, reason drift.ClosureReason, exclusions []db.ListExclusionsRow) string {
	if reason != drift.ReasonDescoped {
		return ""
	}
	name = normalizeDomain(name)
	for _, e := range exclusions {
		if !e.Name.Valid {
			continue
		}
		switch e.Kind {
		case "name":
			if name == normalizeDomain(e.Name.String) {
				return normalizeDomain(e.Name.String)
			}
		case "subtree":
			if nameWithinDomain(name, e.Name.String) {
				return normalizeDomain(e.Name.String)
			}
		}
	}
	return ""
}

// The subject kinds this fold tests by name. They are the stored `span.subject_kind`
// values, named here so the address limb reads them from one place.
const (
	subjectKindName     = "name"
	subjectKindAddress  = "address"
	subjectKindService  = "service"
	subjectKindEndpoint = "endpoint"

	// exclusionKindAddress is the stored `exclusion.kind` of an address-scope
	// exclusion — a CIDR the operator declares is not theirs (ADR-0012 §125).
	exclusionKindAddress = "address"
)

// observedResolutionNames is the distinct set of Names carried by a batch's
// `resolution` observations, in first-seen order — the Names whose membership this
// fold re-composes. Other facets (dns-record, reachability) do not re-decide a
// Name's membership; membership reads the `resolution` facet (internal/estate).
func observedResolutionNames(obs []wire.Observation) []string {
	seen := map[string]bool{}
	var out []string
	for _, o := range obs {
		if o.Facet != resolutionwalk.FacetResolution || o.Subject == "" {
			continue
		}
		if seen[o.Subject] {
			continue
		}
		seen[o.Subject] = true
		out = append(out, o.Subject)
	}
	return out
}

// decideNameDeparture composes whether a Name has left the estate and on what
// ground, from its currently-open spans and its declared-input standing. It is the
// pure heart of the estate wiring — the same three grounds drift's ClosureReason
// names (ADR-0087), decided in precedence order:
//
//   - excluded: the operator drew the boundary inward over this Name (an exact or
//     subtree Exclusion). The aperture stopped covering it, so it leaves `descoped`
//     — the one ground that blocks a later `returned`, because a narrowing is not a
//     decommission. This wins even over a resolution that still admits the Name:
//     the operator's narrowing is the mover.
//   - seedCovered: a Seed still declares this Name. Declared input keeps it in the
//     estate regardless of what resolution measured (estate.Membership), so it does
//     not leave — no closure.
//   - otherwise: the cross-class resolution composition decides. Where every
//     available vantage suppresses the Name (NameError / Shadowed) it has left by
//     measurement — `measured-absent`, the only closure that is independent
//     evidence. A single admitting or not-evaluable (Gap) vantage keeps it
//     (estate.WithdrawnCrossClass), so it does not leave.
//
// Returns the ground and left=true only where the Name leaves; ("", false) keeps
// it. A Name with no open spans cannot leave (there is nothing to close).
func decideNameDeparture(open []db.ListOpenSpansForSubjectRow, seedCovered, excluded bool) (drift.ClosureReason, bool) {
	if len(open) == 0 {
		return "", false
	}
	if excluded {
		return drift.ReasonDescoped, true
	}
	if seedCovered {
		return "", false
	}
	if estate.WithdrawnCrossClass(resolutionWitnesses(open)) {
		return drift.ReasonMeasuredAbsent, true
	}
	return "", false
}

// resolutionWitnesses groups a Name's open `resolution` spans into the per-class
// witness set estate.WithdrawnCrossClass composes over. Each distinct vantage is
// its own class witness keyed by vantage id (the shipped resolver position carries
// a NULL vantage, so it is the single default class ""): within-class absence is
// unanimous and across-class absence is unanimous, and for one class per vantage
// both collapse to "every available vantage suppresses" — the cross-vantage
// unanimity ADR-0080 requires. A vantage currently holding no open resolution span
// contributes no witness (a vantage that did not ask is not one that got nothing),
// exactly as estate.ClassWitness's empty-outcome rule intends.
func resolutionWitnesses(open []db.ListOpenSpansForSubjectRow) []estate.ClassWitness {
	byVantage := map[string][]string{}
	order := []string{}
	for _, s := range open {
		if s.Facet != resolutionwalk.FacetResolution {
			continue
		}
		key := vantageClassKey(s.VantageID)
		if _, seen := byVantage[key]; !seen {
			order = append(order, key)
		}
		byVantage[key] = append(byVantage[key], resolutionOutcome(s.Value))
	}
	out := make([]estate.ClassWitness, 0, len(order))
	for _, key := range order {
		out = append(out, estate.ClassWitness{Class: key, Outcomes: byVantage[key]})
	}
	return out
}

// vantageClassKey is the class key a vantage contributes under. A NULL vantage —
// the shipped resolver position — is the single default class "", so the
// cross-class rule collapses to the single-class case with no special path
// (ADR-0080).
func vantageClassKey(v pgtype.Int8) string {
	if !v.Valid {
		return ""
	}
	return "vantage:" + strconv.FormatInt(v.Int64, 10)
}

// resolutionOutcome reads the composed outcome tag off a `resolution` span value
// (`{"outcome": "...", "addresses": [...]}`). A value that will not parse reads as
// an empty outcome, which estate.suppresses treats as non-suppressing — the safe
// reading, since a malformed value is not affirmative evidence of absence.
func resolutionOutcome(value []byte) string {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(value, &v)
	return v.Outcome
}

// closeSubjectTimelines closes every open span in `open` at `at` with `reason`,
// citing the folding batch, so the estate-wide feed reads the close as a
// withdrawn / descoped transition (ADR-0111). This is drift.CloseWithdrawal applied
// through the store: a withdrawal takes every timeline the subject held, one cause
// recorded on n objects (ADR-0087). An already-closed span is never rewritten —
// CloseSpan's `WHERE closed_at IS NULL` is the guard, and the fold only ever hands
// it open spans.
func closeSubjectTimelines(ctx context.Context, qtx *db.Queries, open []db.ListOpenSpansForSubjectRow, at time.Time, reason drift.ClosureReason, batchID int64) error {
	ids := make([]int64, 0, len(open))
	for _, s := range open {
		ids = append(ids, s.ID)
	}
	return closeSpansByID(ctx, qtx, ids, at, reason, batchID)
}

// closeSpansByID is the write closeSubjectTimelines applies, addressed by id. The
// address withdrawal reads its spans from the exclusion side and holds no
// per-subject row, so both limbs of the fold close through this one call
// (internal/queue/withdrawal.go).
func closeSpansByID(ctx context.Context, qtx *db.Queries, ids []int64, at time.Time, reason drift.ClosureReason, batchID int64) error {
	for _, id := range ids {
		if err := qtx.CloseSpan(ctx, db.CloseSpanParams{
			ClosedAt:      tstz(at),
			ClosureReason: pgText(string(reason)),
			ClosedBatchID: pgInt8(batchID),
			ID:            id,
		}); err != nil {
			return err
		}
	}
	return nil
}

// openedByAperture reports whether a span opening was driven by the operator's
// aperture rather than the world — the signal the drift feed reads `revealed` from
// (ADR-0014). A subject the Declared aperture covers (a Name beneath a name Seed, an
// Address inside an address Seed's scope, or a Service on such an Address) is
// declared input: the fold looked at it because the aperture covers it, not because
// the world produced it. A subject no Seed declares — a resolved Address the world
// cited, a CT-admitted Name — is unmarked and opens `appeared`.
//
// An Exclusion cuts the subject back OUT of the Declared aperture, so an excluded
// subject is unmarked even where a Seed scope still contains it. The case is
// ordinary rather than defensive. An exclusion cuts the `Seed` limb alone, so an
// excluded Address a custody extension reaches still derives operator, is still
// probed, and still opens a span (ADR-0133 §3, internal/custody/candidates.go). The
// world's resolution is why we looked at that one, so it opens `appeared`. This is
// the disjunction decideNameDeparture composes on the way out, read on the way in.
//
// The fold consults it for an opening with no OPEN predecessor, which is a first
// span or a re-entry behind a closed one. A first span reads the marker as
// `revealed`. A re-entry behind a `descoped` closure reads it as the operator
// widening a Declared scope back over the subject, which is `revealed` as well
// (drift.ReEntryKind, ADR-0041, #1039). A value MOVE on an existing timeline is
// `changed` and never consults it.
func openedByAperture(subjectKind, subjectKey string, in membershipInputs) bool {
	if subjectKind == subjectKindName {
		return nameSeedCovered(subjectKey, in.seeds) && !nameExcluded(subjectKey, in.exclusions)
	}
	addr, ok := subjectAddress(subjectKind, subjectKey)
	return ok && addressSeedCovered(addr, in.seeds) && coveringAddressExclusion(addr, in.exclusions) == nil
}

// subjectAddress is the Address a subject key stands at — the key itself for an
// `address` subject, and the address limb of the key for a Service or Endpoint
// sitting on one. Any other subject kind, and any key netip will not parse, report
// false: they stand at no address, so neither an address Seed nor an address
// Exclusion has anything to test against them.
func subjectAddress(subjectKind, subjectKey string) (netip.Addr, bool) {
	switch subjectKind {
	case subjectKindAddress:
		addr, err := netip.ParseAddr(subjectKey)
		return addr, err == nil
	case subjectKindService, subjectKindEndpoint:
		addr, err := netip.ParseAddr(serviceAddress(subjectKey))
		return addr, err == nil
	default:
		return netip.Addr{}, false
	}
}

// nameSeedCovered reports whether a name Seed declares this Name — an exact match
// or a name beneath the Seed's domain (the domain apex and everything under it).
// Address Seeds declare no Name.
func nameSeedCovered(name string, seeds []db.ListSeedsRow) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	for _, s := range seeds {
		if s.Kind != "name" || !s.NameDomain.Valid {
			continue
		}
		if nameWithinDomain(name, s.NameDomain.String) {
			return true
		}
	}
	return false
}

// nameExcluded reports whether an Exclusion covers this Name — an exact `name`
// exclusion or a `subtree` exclusion of the Name or an ancestor. Address
// exclusions cover no Name.
func nameExcluded(name string, exclusions []db.ListExclusionsRow) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	for _, e := range exclusions {
		if !e.Name.Valid {
			continue
		}
		switch e.Kind {
		case "name":
			if name == normalizeDomain(e.Name.String) {
				return true
			}
		case "subtree":
			if nameWithinDomain(name, e.Name.String) {
				return true
			}
		}
	}
	return false
}

// coveringAddressExclusion is the address analogue of nameExcluded that ADR-0012
// §125 owed and #1032 supplies: the first declared `address` Exclusion whose scope
// contains the address, in ListExclusions order, or nil where none does.
//
// Containment is the family-matched prefix test the coverage predicate already
// applies, so an IPv4 address is never read as inside an IPv6 scope. Name and
// subtree exclusions cover no Address. Exclusions do not nest into a precedence
// the way Seeds do — every one of them is the same "not mine" claim — so first
// match is the whole rule, exactly as nameExcluded's loop already treats them.
//
// It returns the covering prefix rather than a bool because both its callers need
// it: the withdrawal names the Exclusion as the message's mover, and it finds the
// Seed scope the message fires at from the prefix.
func coveringAddressExclusion(addr netip.Addr, exclusions []db.ListExclusionsRow) *netip.Prefix {
	for _, e := range exclusions {
		if e.Kind != exclusionKindAddress || e.AddressCidr == nil {
			continue
		}
		if e.AddressCidr.Contains(addr) {
			return e.AddressCidr
		}
	}
	return nil
}

// addressSeedCovered reports whether an address Seed's scope contains the address.
func addressSeedCovered(addr netip.Addr, seeds []db.ListSeedsRow) bool {
	for _, s := range seeds {
		if s.Kind != "address" || s.AddressCidr == nil {
			continue
		}
		if s.AddressCidr.Contains(addr) {
			return true
		}
	}
	return false
}

// nameWithinDomain reports whether name is the domain apex or a subdomain beneath
// it — the subtree-coverage rule Seeds and subtree Exclusions share. Comparison is
// case-insensitive over trailing-dot-normalised labels, and a suffix match is
// gated on a label boundary so "notexample.com" is not read as within "example.com".
func nameWithinDomain(name, domain string) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	domain = normalizeDomain(domain)
	if domain == "" {
		return false
	}
	return name == domain || strings.HasSuffix(name, "."+domain)
}

func normalizeDomain(d string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(d)), ".")
}

// serviceAddress extracts the address limb of a Service/Endpoint subject key. A
// Service is keyed "address:port" (an IP and a port); the address is everything
// before the last colon, which leaves a bracketed or bare IPv6 host intact for the
// caller's netip.ParseAddr to accept or reject.
func serviceAddress(key string) string {
	if i := strings.LastIndex(key, ":"); i >= 0 {
		host := key[:i]
		host = strings.TrimPrefix(host, "[")
		host = strings.TrimSuffix(host, "]")
		return host
	}
	return key
}
