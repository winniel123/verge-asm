package queue

import (
	"context"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

// This file is the ADDRESS limb of the withdrawal the membership fold owes
// (ADR-0133 §8, #1032). ADR-0012 §125 extended Exclusions to address scopes and
// `CONTEXT.md` states the consequence. Leaving a declared scope by exclusion
// withdraws the address and takes its timelines with it, unless a current
// resolution still cites it, and the closure carries the `descoped` reason.
// ADR-0133 shipped the probe gate (#1022) and left this half open. Nothing closed
// a span, and message.Narrowing — the function that renders the coverage message
// `POST /exclusions/preview` names — had no production caller, so the preview
// described an act the system did not perform.
//
// Three rulings hold this file up. The first two are the questions #1032 left
// open. The third is forced by ADR-0133 §1 and #1032 did not see it.
//
//   - **When it runs.** On the next membership fold, not at declaration time.
//     Two things force that. A closure cites the folding batch (drift.CloseSpan,
//     ADR-0111) and a web handler holds no batch. And ADR-0133 §3 stops
//     ENUMERATING an excluded address, so no observation about one need ever
//     arrive again. A withdrawal scoped to the subjects a batch observed — the
//     rule foldEstateTransitions applies to Names — could therefore never reach
//     it. So this fold is driven from the DECLARATION side and reads the
//     exclusion corpus, not the batch's observations. It is not a sweep of the
//     estate. The set is bounded by what the operator declared.
//
//     The bound this accepts: the withdrawal lands on the next COMPLETED job, so
//     an estate running no jobs at all holds its spans open until one completes.
//     A declared scope runs a cadence, so that is a quiet estate and not a stuck
//     one.
//
//   - **Whether the preview's counts must agree with the act's.** They are not
//     required to. ListAddressExclusionWithdrawals is the listing twin of
//     PreviewExclusionWithdrawal and reads the same two CTE shapes, so the two
//     agree over an estate that has not moved. The estate MAY move between the
//     preview and the fold. An address the preview counted may be cited by a
//     resolution by then, and it survives. Each count is a measurement at its own
//     instant, like every other count the product shows.
//
//   - **An address the custody extension still reaches does NOT leave.**
//     ADR-0133 §1 keeps the extension limb standing under an exclusion: such an
//     address derives `operator`, and candidateAddrs skips an excluded address
//     out of the SCOPE ENUMERATION alone, so it is still enumerated, still
//     probed and still measured. Closing its timelines would reopen them on the
//     next batch and close them again on the one after — a `descoped` departure
//     and a Narrowing message every cadence, for an address the gate never
//     stopped probing. The survivor test is custody.Estate.Derive itself, so the
//     membership decision and the probe gate read ONE derivation and cannot
//     disagree. This is the same shape as the resolution survivor the SQL
//     already applies: another limb still holds the address, so the narrowing
//     does not remove it, and the set an exclusion removes stays no larger than
//     the set the declaration added.
//
// The act writes ONE message per exclusion, not one per subject. message.Narrowing
// is the count-carrying form ADR-0074 defines — "a scope and a count, no
// comparison and no rows" — so N per-subject `declared-input` rows for a single
// operator act would be exactly the census the receipt exists to replace. The Name
// limb keeps its per-subject declared-input message (produce.go), because a name
// exclusion withdraws one Name and has no aggregate to state.

// foldAddressExclusionWithdrawals closes every open timeline a declared address
// Exclusion withdraws, with the `descoped` ground, citing the folding batch. It
// runs inside the batch transaction after foldEstateTransitions, so a subject the
// value fold just opened is visible to it.
//
// It is idempotent by construction rather than by a marker. The closure is what
// removes the row from the query's answer, so the next batch reads no row, closes
// nothing and fires no second message. The two survivor rules are what make that
// hold: an address another limb still carries is never closed in the first place,
// so it cannot reopen and be closed again.
//
// The closure happens whether or not the producer is on. A withdrawal is a fact
// about the estate, not a message. `out` is the narrowing collector and is nil on
// the measurement-only path, exactly as departureCollector is.
func foldAddressExclusionWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]message.NarrowingReceipt) error {
	// A corpus with no address exclusion withdraws nothing, so the read is skipped
	// outright. This is the shipped default, and the fold runs once per completed
	// job.
	if !in.hasAddressExclusion() {
		return nil
	}
	rows, err := qtx.ListAddressExclusionWithdrawals(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	// The estate is read ONLY where there is something to withdraw. In the steady
	// state the query above answers empty and the derivation is never built.
	estate, _, err := hotEstate(ctx, qtx, observedAt)
	if err != nil {
		return err
	}
	spanIDs, narrowings := composeAddressWithdrawals(rows, in, estate.Derive)
	if err := closeSpansByID(ctx, qtx, spanIDs, observedAt, drift.ReasonDescoped, batchID); err != nil {
		return err
	}
	if out != nil {
		*out = append(*out, narrowings...)
	}
	return nil
}

// composeAddressWithdrawals is the pure heart of the address withdrawal. It
// attributes each withdrawn span to the Exclusion that covers it and to the Seed
// scope the message fires at, drops the spans that survive, and returns the spans
// to close plus one receipt per covering Exclusion, in first-seen order.
//
// `derive` is custody.Estate.Derive — the one derivation the probe gate reads. An
// address it still calls `operator` under the exclusion is reached by the custody
// extension, so it has not left the estate and its timelines stay open.
//
// A row it cannot attribute to a declared Exclusion is DROPPED, not closed. The
// query and this function read the same exclusion corpus, so that should not
// happen. Where it does — an address key netip will not parse — the safe reading
// is to leave the timeline open. A closure with no mover to name is a withdrawal
// the operator cannot trace back to their own act.
func composeAddressWithdrawals(rows []db.ListAddressExclusionWithdrawalsRow, in membershipInputs, derive func(netip.Addr) custody.Custody) ([]int64, []message.NarrowingReceipt) {
	var spanIDs []int64
	var order []string
	counts := map[string]*withdrawalCount{}

	for _, row := range rows {
		addr, ok := subjectAddress(row.SubjectKind, row.SubjectKey)
		if !ok {
			continue
		}
		p := coveringAddressExclusion(addr, in.exclusions)
		if p == nil {
			continue
		}
		if derive != nil && derive(addr) == custody.Operator {
			continue // the custody extension still reaches it (ADR-0133 §1)
		}
		key := p.String()
		c, seen := counts[key]
		if !seen {
			c = &withdrawalCount{scope: narrowingScope(*p, in.seeds), subjects: map[string]bool{}}
			counts[key] = c
			order = append(order, key)
		}
		spanIDs = append(spanIDs, row.ID)
		c.timelines++
		// A subject is counted once however many timelines it held. The receipt states
		// the two factors and never their product (message.NarrowingReceipt).
		if !c.subjects[row.SubjectKey] {
			c.subjects[row.SubjectKey] = true
		}
	}

	receipts := make([]message.NarrowingReceipt, 0, len(order))
	for _, key := range order {
		c := counts[key]
		receipts = append(receipts, message.PreviewNarrowing(c.scope, key, len(c.subjects), c.timelines))
	}
	return spanIDs, receipts
}

// withdrawalCount accumulates one Exclusion's two factors while the fold walks the
// listed spans — the distinct subjects that left and the timelines they held. It
// is the mutable half of message.NarrowingReceipt, which is computed once at the
// end so the act and the preview render through the same constructor.
type withdrawalCount struct {
	scope     string
	subjects  map[string]bool
	timelines int
}

// hasAddressExclusion reports whether the declared input holds any `address`
// exclusion at all — the guard that keeps the withdrawal read off a batch that
// cannot possibly withdraw anything.
func (in membershipInputs) hasAddressExclusion() bool {
	for _, e := range in.exclusions {
		if e.Kind == exclusionKindAddress && e.AddressCidr != nil {
			return true
		}
	}
	return false
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
