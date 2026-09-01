package queue

import (
	"context"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
)

// This file is the ADDRESS limb of the withdrawal the membership fold owes
// (ADR-0133 §8, #1032). ADR-0012 §125 extended Exclusions to address scopes and
// `CONTEXT.md` states the consequence: leaving a declared scope by exclusion
// withdraws the address and takes its timelines with it, unless a current
// resolution still cites it, and the closure carries the `descoped` reason.
// ADR-0133 shipped the probe gate (#1022) and left this half open: nothing closed
// a span, and message.Narrowing — the function that renders the coverage message
// `POST /exclusions/preview` names — had no production caller. The preview
// therefore described an act the system did not perform.
//
// Two questions #1032 left open are decided here.
//
//   - **When it runs.** On the next membership fold, not at declaration time.
//     Two things force it. A closure cites the folding batch (drift.CloseSpan,
//     ADR-0111), and a web handler holds no batch. And ADR-0133 §3 stops
//     ENUMERATING an excluded address, so no observation about it ever arrives
//     again — a withdrawal scoped to the subjects a batch observed (the rule
//     foldEstateTransitions applies to Names) could never fire for one. So this
//     fold is driven from the DECLARATION side: it reads the exclusion corpus,
//     not the batch's observations. It is not a sweep of the estate; the set is
//     bounded by what the operator declared.
//
//   - **Whether the preview's counts must agree with the act's.** They are not
//     required to. ListAddressExclusionWithdrawals is the listing twin of
//     PreviewExclusionWithdrawal and reads the same two CTE shapes, so the two
//     agree over an estate that has not moved. The estate MAY move between the
//     preview and the fold — an address the preview counted may be cited by a
//     resolution by then, and survives. The receipt is a measurement at its own
//     instant, like every other count the product shows, and the withdrawal is a
//     measurement at the fold's.
//
// The act writes ONE message per exclusion, not one per subject. message.Narrowing
// is the count-carrying form ADR-0074 defines — "a scope and a count, no
// comparison and no rows" — so N per-subject `declared-input` rows for a single
// operator act would be exactly the census the receipt exists to replace. The Name
// limb keeps its per-subject declared-input message (produce.go): a name exclusion
// withdraws one Name, so there is no aggregate to state.

// narrowing is one declared address Exclusion's withdrawal, aggregated over every
// subject it took out of the estate in this batch — the payload
// message.PreviewNarrowing renders and message.Narrowing fires from. It is the
// twin of `departure` for the address limb: collected by the fold below, consumed
// by the producer in the same batch transaction.
type narrowing struct {
	// Scope is the Seed scope the message fires at — the address Seed the excluded
	// ground sits inside, which is the only object that survives the act.
	Scope string
	// Removed is the declared exclusion CIDR that narrowed the scope.
	Removed string
	// SubjectsWithdrawn counts the distinct subjects the act removed, and
	// TimelinesRemoved the open spans they held. Both are counted over what this
	// fold actually closed, never over what the preview projected.
	SubjectsWithdrawn int
	TimelinesRemoved  int
}

// foldAddressExclusionWithdrawals closes every open timeline a declared address
// Exclusion withdraws, with the `descoped` ground, citing the folding batch. It
// runs inside the batch transaction after foldEstateTransitions, so a subject the
// value fold just opened is visible to it.
//
// It is idempotent by construction rather than by a marker: the closure is what
// removes the row from the query's answer, so the next batch reads no row, closes
// nothing and fires no second message. ADR-0133 §3 keeps an excluded address out
// of the enumeration, so no new timeline opens there to be closed again.
//
// The closure happens whether or not the producer is on — a withdrawal is a fact
// about the estate, not a message. `out` is the narrowing collector and is nil on
// the measurement-only path, exactly as departureCollector is.
func foldAddressExclusionWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]narrowing) error {
	rows, err := qtx.ListAddressExclusionWithdrawals(ctx)
	if err != nil {
		return err
	}
	spanIDs, narrowings := composeAddressWithdrawals(rows, in)
	if err := closeSpansByID(ctx, qtx, spanIDs, observedAt, drift.ReasonDescoped, batchID); err != nil {
		return err
	}
	if out != nil {
		*out = append(*out, narrowings...)
	}
	return nil
}

// composeAddressWithdrawals is the pure heart of the address withdrawal: it
// attributes each withdrawn span to the Exclusion that covers it and to the Seed
// scope the message fires at, and returns the spans to close plus one narrowing
// per covering Exclusion, in first-seen order.
//
// A row it cannot attribute to a declared Exclusion is DROPPED, not closed. The
// query and this function read the same exclusion corpus, so that should not
// happen; where it does — an address key netip will not parse — the safe reading
// is to leave the timeline open. A closure with no mover to name is a withdrawal
// the operator cannot trace back to their own act.
func composeAddressWithdrawals(rows []db.ListAddressExclusionWithdrawalsRow, in membershipInputs) ([]int64, []narrowing) {
	var spanIDs []int64
	var order []string
	byExclusion := map[string]*narrowing{}
	subjects := map[string]map[string]bool{}

	for _, row := range rows {
		addr, ok := subjectAddress(row.SubjectKind, row.SubjectKey)
		if !ok {
			continue
		}
		p := coveringAddressExclusion(addr, in.exclusions)
		if p == nil {
			continue
		}
		key := p.String()
		n, seen := byExclusion[key]
		if !seen {
			n = &narrowing{Scope: narrowingScope(*p, in.seeds), Removed: key}
			byExclusion[key] = n
			subjects[key] = map[string]bool{}
			order = append(order, key)
		}
		spanIDs = append(spanIDs, row.ID)
		n.TimelinesRemoved++
		// A subject is counted once however many timelines it held — the receipt
		// states the two factors, never their product (message.NarrowingReceipt).
		if !subjects[key][row.SubjectKey] {
			subjects[key][row.SubjectKey] = true
			n.SubjectsWithdrawn++
		}
	}

	narrowings := make([]narrowing, 0, len(order))
	for _, key := range order {
		narrowings = append(narrowings, *byExclusion[key])
	}
	return spanIDs, narrowings
}

// narrowingScope is the Seed scope the narrowing message fires at — the declared
// address Seed the excluded ground sits inside. It mirrors the preview's
// FindCoveringAddressSeed, most-specific-covering-scope-first, so the act and the
// receipt name the same firing site. Where no declared scope covers the exclusion
// the excluded value itself is the site: nothing is enumerated there, so the act
// withdraws nothing and the message does not fire anyway.
func narrowingScope(excluded netip.Prefix, seeds []db.ListSeedsRow) string {
	best := ""
	bits := -1
	for _, s := range seeds {
		if s.Kind != "address" || s.AddressCidr == nil {
			continue
		}
		if !s.AddressCidr.Contains(excluded.Addr()) {
			continue
		}
		if s.AddressCidr.Bits() > bits {
			bits = s.AddressCidr.Bits()
			best = s.AddressCidr.String()
		}
	}
	if best == "" {
		return excluded.String()
	}
	return best
}

// closeSpansByID closes the listed spans at `at` with `reason`, citing the folding
// batch — closeSubjectTimelines addressed by id rather than by subject, because
// the address withdrawal reads its spans from the exclusion side and never holds a
// per-subject row. CloseSpan's `WHERE closed_at IS NULL` is the guard against
// rewriting a span another limb of the same fold already closed.
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
