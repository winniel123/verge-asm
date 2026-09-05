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

func foldAddressExclusionWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]message.NarrowingReceipt) error {
	// No observation reaches an excluded address, so this fold reads the declaration (ADR-0133 §3).
	if !in.hasAddressExclusion() {
		return nil
	}
	// The closure removes the row from this query, so idempotency needs no marker (ADR-0218 §1).
	rows, err := qtx.ListAddressExclusionWithdrawals(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	// The estate is built only where there is something to withdraw. The steady state answers empty.
	estate, _, err := hotEstate(ctx, qtx, observedAt)
	if err != nil {
		return err
	}
	// The probe gate reads this same derivation, so membership and probing cannot disagree.
	spanIDs, narrowings := composeAddressWithdrawals(rows, in, estate.Derive)
	if err := closeSpansByID(ctx, qtx, spanIDs, observedAt, drift.ReasonDescoped, batchID); err != nil {
		return err
	}
	// A withdrawal is a fact about the estate, not a message, so it closes whether or not out is set (ADR-0219 §1).
	if out != nil {
		*out = append(*out, narrowings...)
	}
	return nil
}

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
		// A closure with no mover to name is untraceable, so an unattributable row is dropped.
		if p == nil {
			continue
		}
		// An address the extension still reaches is still probed, so closing it would flap (ADR-0133 §1).
		if derive != nil && derive(addr) == custody.Operator {
			continue
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
		if !c.subjects[row.SubjectKey] {
			c.subjects[row.SubjectKey] = true
		}
	}

	// One act writes one counted receipt; per-subject rows would be the census it replaces (ADR-0074).
	receipts := make([]message.NarrowingReceipt, 0, len(order))
	for _, key := range order {
		c := counts[key]
		receipts = append(receipts, message.PreviewNarrowing(c.scope, key, len(c.subjects), c.timelines))
	}
	// The estate may move between the preview and the fold, so the two counts need not agree.
	return spanIDs, receipts
}

// The act renders through the preview's constructor, so the two can never state it differently (ADR-0218 §4).

type withdrawalCount struct {
	scope     string
	subjects  map[string]bool
	timelines int
}

func (in membershipInputs) hasAddressExclusion() bool {
	for _, e := range in.exclusions {
		if e.Kind == exclusionKindAddress && e.AddressCidr != nil {
			return true
		}
	}
	return false
}

func narrowingScope(excluded netip.Prefix, seeds []db.ListSeedsRow) string {
	// This must mirror the preview's FindCoveringAddressSeed, or the two name different sites (ADR-0218 §3).
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
