package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

// This file is the NAME limb of the Seed withdrawal (ADR-0135, #1045).
// seedwithdrawal.go answered the address limb and ADR-0134 §7 named this one as
// the gap it was leaving open.
//
// WHY IT IS NOT foldEstateTransitions. That fold iterates the Names the batch
// OBSERVED, and a withdrawn name Seed stops its Names being enumerated: the dns
// Scan's resolution set is the live seed domains plus the distinct admitted names
// (mergeResolutionNames), and the delete removes the domain from the first limb
// while the FK cascade removes its admissions from the second. So no observation
// about those Names ever arrives again and the fold can never revisit them. This
// is ADR-0133 §8.1's shape a third time: a withdrawal scoped to observed subjects
// can never reach a subject the act stopped enumerating.
//
// WHY THE CASCADE DOES NOT COST US THE GROUND. Deleting the Seed cascades away its
// `admitted_name` and `zone_file` rows, so the evidence that ADMITTED the Names is
// gone in the same transaction as the declaration. It touches no `span`, and
// `span` is where the candidates are read. The record of a Name's timelines is not
// the evidence that admitted it, so the tombstone needs to carry the domain and
// nothing else (ADR-0135 §4).
//
// TWO SURVIVORS keep a Name open, not the address limb's three (ADR-0135 §3):
//
//   - A live name Seed still covers it. Read from the Seed corpus at fold time,
//     never from the tombstone. As on the address side this also settles
//     re-declaration: withdraw example.com, declare it again, and the Names
//     re-enter through the Seed limb while a stale tombstone still names the
//     domain.
//   - A live `admitted_name` row still admits it. Those rows carry their own
//     `seed_id`, so a Name a SURVIVING Seed also admitted keeps its row through
//     the cascade, stays in the resolution set, and is still walked every batch.
//     Closing it would be a `descoped` departure over a Name the estate never
//     stopped measuring, and the next batch would reopen it and the batch after
//     close it again.
//
// The address limb's third survivor has no Name counterpart. `custody.Estate`
// derives ADDRESSES, and its extension lives on a name Seed — so withdrawing that
// Seed takes the extension with it and there is nothing left standing to test.
// CONTEXT.md's "unless a current resolution still cites it" is likewise an ADDRESS
// rule: an address is in the estate while a resolution cites it, and a Name is the
// thing that resolves rather than a thing resolutions cite.
//
// BOTH SURVIVOR TESTS RUN IN GO, and both must. Each has to use the same key
// function the resolution set uses — nameSeedCovered for the Seed corpus,
// resolutionNameKey over the admitted names — so that "does this Name survive" and
// "does this Name stay enumerated" cannot disagree. A SQL test keying names its own
// way would drop a Name the estate still walks, or hold one it stopped walking.
// This is ADR-0134 §4's argument for deciding the custody survivor through Derive
// itself, applied to the pair of limbs that decide a Name.
//
// ONE AGGREGATE MESSAGE, not one per Name (ADR-0135 §1). ADR-0133 §8.1 kept a
// per-subject `declared-input` on the Name side because a name exclusion withdraws
// ONE Name; the discriminator was one-per-act against many-per-act, never the
// subject kind. A name Seed withdrawal is the first Name act that removes many
// subjects at once, so it takes the aggregate contract for the reason ADR-0074
// gives: a row per withdrawn subject would be the census the receipt exists to
// replace.

// foldNameSeedWithdrawals closes every open timeline a withdrawn name Seed takes
// with it, with the `descoped` ground, citing the folding batch — then spends the
// tombstones whose ground is exhausted.
//
// It runs after foldEstateTransitions, so a Name that fold already closed is not
// open and is never counted or attributed twice.
func foldNameSeedWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]message.NarrowingReceipt) error {
	pending, err := qtx.ListPendingNameSeedWithdrawals(ctx)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	rows, err := qtx.ListNameSeedWithdrawalCandidates(ctx)
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		// The admitted names are read ONLY where there is ground to withdraw, so a
		// tombstone over an empty domain never pays for the corpus.
		admitted, err := admittedNames(ctx, qtx)
		if err != nil {
			return err
		}
		spanIDs, narrowings := composeNameSeedWithdrawals(rows, pending, in, admitted)
		if err := closeSpansByID(ctx, qtx, spanIDs, observedAt, drift.ReasonDescoped, batchID); err != nil {
			return err
		}
		if out != nil {
			*out = append(*out, narrowings...)
		}
	}
	// Spend only the tombstones whose withdrawal is EXHAUSTED and whose ground no
	// in-flight job can re-open. A dns Scan fans out one job PER VANTAGE, each
	// freezing the whole resolution set into its own scope gate, so a job enqueued
	// before the withdrawal still opens a fresh span for a withdrawn Name when it
	// completes after it. Spending on the first exhausted fold would leave the
	// sibling vantage's span open with no mover left (ADR-0135 §5). The query decides
	// it, so it reads the closures above from inside the same transaction.
	//
	// Only the rows THIS fold claimed are offered. The pending read holds a row lock
	// on each of them.
	ids := make([]int64, 0, len(pending))
	for _, w := range pending {
		ids = append(ids, w.ID)
	}
	return qtx.SpendNameSeedWithdrawals(ctx, db.SpendNameSeedWithdrawalsParams{
		ConsumedAt:      tstz(observedAt),
		ConsumedBatchID: pgInt8(batchID),
		Ids:             ids,
	})
}

// composeNameSeedWithdrawals is the pure heart of the name Seed withdrawal and the
// twin of composeSeedWithdrawals. It attributes each candidate span to the
// tombstone whose withdrawn domain covers it, drops the Names that survive, and
// returns the spans to close plus one receipt per withdrawn domain, in first-seen
// order.
//
// A row it cannot attribute to a tombstone is DROPPED, not closed, exactly as both
// other narrowing acts drop an unattributable row. A closure with no mover to name
// is a withdrawal the operator cannot trace back to their own act.
func composeNameSeedWithdrawals(rows []db.ListNameSeedWithdrawalCandidatesRow, pending []db.ListPendingNameSeedWithdrawalsRow, in membershipInputs, admitted []string) ([]int64, []message.NarrowingReceipt) {
	stillAdmitted := make(map[string]bool, len(admitted))
	for _, n := range admitted {
		if key := resolutionNameKey(n); key != "" {
			stillAdmitted[key] = true
		}
	}

	var spanIDs []int64
	var order []string
	counts := map[string]*withdrawalCount{}

	for _, row := range rows {
		domain := coveringNameSeedWithdrawal(row.SubjectKey, pending)
		if domain == "" {
			continue
		}
		if nameSeedCovered(row.SubjectKey, in.seeds) {
			continue // a live Seed still declares it (ADR-0135 §3)
		}
		if stillAdmitted[resolutionNameKey(row.SubjectKey)] {
			continue // a surviving Seed's admission still enumerates it (ADR-0135 §3)
		}
		c, seen := counts[domain]
		if !seen {
			// The withdrawn domain is its own firing site. A name Seed's display scope IS
			// its domain, so the scope that moved and the ground that left are one object,
			// as they are for an address Seed's CIDR.
			c = &withdrawalCount{scope: domain, subjects: map[string]bool{}}
			counts[domain] = c
			order = append(order, domain)
		}
		spanIDs = append(spanIDs, row.ID)
		c.timelines++
		// A subject is counted once however many timelines it held. The receipt states
		// the two factors and never their product (message.NarrowingReceipt).
		c.subjects[row.SubjectKey] = true
	}

	receipts := make([]message.NarrowingReceipt, 0, len(order))
	for _, key := range order {
		c := counts[key]
		receipts = append(receipts, message.PreviewSeedWithdrawal(c.scope, len(c.subjects), c.timelines))
	}
	return spanIDs, receipts
}

// coveringNameSeedWithdrawal is the name analogue of coveringSeedWithdrawal: the
// normalized domain of the first pending withdrawal that covers the Name — the
// apex itself or anything beneath it — in tombstone order, or "" where none does.
//
// Withdrawals do not nest into a precedence the way Seeds do, each one being the
// same "I stopped declaring this", so first match is the whole rule.
func coveringNameSeedWithdrawal(name string, pending []db.ListPendingNameSeedWithdrawalsRow) string {
	for _, w := range pending {
		if !w.NameDomain.Valid {
			continue
		}
		if nameWithinDomain(name, w.NameDomain.String) {
			return normalizeDomain(w.NameDomain.String)
		}
	}
	return ""
}
