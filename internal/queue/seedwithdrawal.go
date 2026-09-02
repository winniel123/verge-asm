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

// This file is the SEED limb of the withdrawal the membership fold owes
// (ADR-0134, #1040). `CONTEXT.md` names two acts that withdraw an Address from the
// estate — an exclusion, and a Seed that narrows or is withdrawn — and
// withdrawal.go answered the first alone. Withdrawing a Seed resolved the scope's
// display string, ran DELETE FROM seed and redirected with a toast. It closed no
// span, so an address the estate held ONLY because that Seed covered it kept its
// open timelines for ever: no resolution cited it, no declaration covered it, and
// nothing closed it.
//
// The exclusion fix does not generalise, and that is the whole reason this file
// exists rather than a branch in withdrawal.go. ListAddressExclusionWithdrawals
// reads the LIVE exclusion corpus, so its mover is a row the fold can still read
// after the act. A Seed withdrawal destroys the mover in the same statement that
// performs the act. So the act writes a `seed_withdrawal` tombstone beside the
// delete (ADR-0134 §2), and this fold reads that. It is the one place in the model
// where a declared input is read from a record of a past act rather than from live
// truth, and the survivor rules below are what make that safe.
//
// THREE SURVIVORS keep an address open (ADR-0134 §4). An address does not leave
// while any limb still holds it:
//
//   - A current resolution cites it. The disjunctive membership rule, applied in
//     SQL as the NOT EXISTS clause ListSeedWithdrawalCandidates carries.
//   - A LIVE Seed covers it. Read from the Seed corpus at fold time, never from
//     the tombstone. This one is load-bearing twice. It settles ground a second
//     declared Seed still covers, and it settles RE-DECLARATION: withdraw
//     10.0.0.0/24, declare it again later, and the addresses re-enter through the
//     Seed limb while a stale tombstone still names the CIDR. Reading live truth
//     is what stops the tombstone closing ground that is declared again.
//   - custody.Estate.Derive still calls it `operator`. The custody extension lives
//     on a NAME Seed and an address Seed can never carry one, so withdrawing an
//     address Seed leaves any extension standing, and such an address is still
//     enumerated, still probed and still measured. Without this survivor the
//     address closes, the next batch reopens it and the batch after closes it
//     again — a `descoped` departure and a coverage message every cadence, over an
//     address the gate never stopped probing. The test is Derive itself, so the
//     membership decision and the probe gate read ONE derivation and cannot
//     disagree.
//
// WHEN IT RUNS is ADR-0133 §8.1's ruling unchanged, for the same two reasons. A
// closure cites the folding batch (ADR-0111) and a web handler holds none. And a
// withdrawn scope stops being enumerated, so a fold scoped to the subjects a batch
// observed could never reach the address. The accepted bound is the same: the
// withdrawal lands on the next COMPLETED job, so an estate running no jobs holds
// its spans open until one completes.
//
// It runs LAST in the batch transaction, after the value fold, the Name estate
// fold and the exclusion withdrawal. An address the exclusion fold already closed
// is not open, so it is never counted or attributed twice.
//
// A TOMBSTONE IS SPENT LATE, once no open timeline is left under its CIDR, and not
// on the first fold that reads it. Two of the three survivors above are transient —
// a citing resolution goes away, a custody extension is turned off — and the
// exclusion twin can afford to act late because its live `exclusion` row is still
// there to re-read. A tombstone is the only mover its act will ever have. Spending
// it while its ground is still held would strand those addresses open for ever,
// which is the leak this file exists to close.
//
// The pending read takes FOR UPDATE SKIP LOCKED, because workers are
// multi-instance. Without it two folds completing at once both collect the same
// receipt, and the loser writes a coverage message for subjects its own batch
// withdrew none of. A Message is written once and never recomputed, so that
// duplicate would be permanent.

// foldSeedWithdrawals closes every open timeline a withdrawn address Seed takes
// with it, with the `descoped` ground, citing the folding batch — then spends the
// tombstones it read.
//
// The closure happens whether or not the producer is on. A withdrawal is a fact
// about the estate, not a message. `out` is the narrowing collector and is nil on
// the measurement-only path, exactly as departureCollector is.
func foldSeedWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]message.NarrowingReceipt) error {
	// The pending read is the guard. An estate that has withdrawn no Seed — or
	// whose withdrawals are all spent — reads one indexed row-less query per
	// completed job and stops here.
	pending, err := qtx.ListPendingSeedWithdrawals(ctx)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	rows, err := qtx.ListSeedWithdrawalCandidates(ctx)
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		// The estate is read ONLY where there is something to withdraw. A tombstone
		// over ground that holds no open timeline never builds the derivation.
		estate, _, err := hotEstate(ctx, qtx, observedAt)
		if err != nil {
			return err
		}
		spanIDs, narrowings := composeSeedWithdrawals(rows, pending, in, estate.Derive)
		if err := closeSpansByID(ctx, qtx, spanIDs, observedAt, drift.ReasonDescoped, batchID); err != nil {
			return err
		}
		if out != nil {
			*out = append(*out, narrowings...)
		}
	}
	// Spend the tombstones whose withdrawal is now EXHAUSTED — the ones with no open
	// timeline left under their CIDR. The query decides that, so it reads the
	// closures above from inside the same transaction. A tombstone whose ground a
	// transient survivor still holds stays pending and is retried on the next
	// completed job.
	//
	// Only the rows THIS fold claimed are offered. The pending read holds a row lock
	// on each of them, so the update never waits on a row another worker's fold is
	// holding.
	ids := make([]int64, 0, len(pending))
	for _, w := range pending {
		ids = append(ids, w.ID)
	}
	return qtx.SpendSeedWithdrawals(ctx, db.SpendSeedWithdrawalsParams{
		ConsumedAt:      tstz(observedAt),
		ConsumedBatchID: pgInt8(batchID),
		Ids:             ids,
	})
}

// composeSeedWithdrawals is the pure heart of the Seed withdrawal, and the twin of
// composeAddressWithdrawals. It attributes each candidate span to the tombstone
// whose withdrawn CIDR contains it, drops the spans that survive, and returns the
// spans to close plus one receipt per withdrawn CIDR, in first-seen order.
//
// It applies the two survivor rules the SQL cannot: a live Seed still covering the
// address, and `derive` — custody.Estate.Derive, the one derivation the probe gate
// reads — still calling it `operator`.
//
// A row it cannot attribute to a tombstone is DROPPED, not closed, exactly as the
// exclusion act drops an unattributable row. A closure with no mover to name is a
// withdrawal the operator cannot trace back to their own act.
//
// Receipts are keyed by CIDR, not by tombstone id, so two tombstones naming the
// same withdrawn scope state one act to the operator rather than two.
func composeSeedWithdrawals(rows []db.ListSeedWithdrawalCandidatesRow, pending []db.ListPendingSeedWithdrawalsRow, in membershipInputs, derive func(netip.Addr) custody.Custody) ([]int64, []message.NarrowingReceipt) {
	var spanIDs []int64
	var order []string
	counts := map[string]*withdrawalCount{}

	for _, row := range rows {
		addr, ok := subjectAddress(row.SubjectKind, row.SubjectKey)
		if !ok {
			continue
		}
		p := coveringSeedWithdrawal(addr, pending)
		if p == nil {
			continue
		}
		if addressSeedCovered(addr, in.seeds) {
			continue // a live Seed still declares it (ADR-0134 §4)
		}
		if derive != nil && derive(addr) == custody.Operator {
			continue // the custody extension still reaches it (ADR-0134 §4)
		}
		key := p.String()
		c, seen := counts[key]
		if !seen {
			// The withdrawn CIDR is its own firing site. An address Seed's display scope
			// IS its CIDR, so unlike an exclusion there is no wider declared scope to
			// find: the scope that moved and the ground that left are one object.
			c = &withdrawalCount{scope: key, subjects: map[string]bool{}}
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
		receipts = append(receipts, message.PreviewSeedWithdrawal(c.scope, len(c.subjects), c.timelines))
	}
	return spanIDs, receipts
}

// coveringSeedWithdrawal is the tombstone analogue of coveringAddressExclusion:
// the first pending withdrawal whose CIDR contains the address, in tombstone
// order, or nil where none does.
//
// Containment is the family-matched prefix test the coverage predicate already
// applies, so an IPv4 address is never read as inside an IPv6 scope. Withdrawals
// do not nest into a precedence the way Seeds do — each one is the same "I stopped
// declaring this" — so first match is the whole rule.
func coveringSeedWithdrawal(addr netip.Addr, pending []db.ListPendingSeedWithdrawalsRow) *netip.Prefix {
	for _, w := range pending {
		// The column became nullable when the table grew its name limb (ADR-0135 §2),
		// so a nil is skipped as an exclusion's CIDR is. The read is already filtered
		// to `kind = 'address'`, where the shape CHECK forbids a NULL, so nothing
		// reaches here with one.
		if w.AddressCidr == nil {
			continue
		}
		if w.AddressCidr.Contains(addr) {
			cidr := *w.AddressCidr
			return &cidr
		}
	}
	return nil
}
