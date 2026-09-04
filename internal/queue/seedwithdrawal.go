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

func foldSeedWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]message.NarrowingReceipt) error {
	// The act destroys the Seed it withdraws, so a tombstone is the only mover left (ADR-0134 §2).
	pending, err := qtx.ListPendingSeedWithdrawals(ctx)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	// The candidate query is shared with the preview, which has no tombstone to read (#1046).
	rows, err := qtx.ListSeedWithdrawalCandidates(ctx, pendingWithdrawnCIDRs(pending))
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		estate, _, err := hotEstate(ctx, qtx, observedAt)
		if err != nil {
			return err
		}
		spanIDs, narrowings := composeSeedWithdrawals(rows, pending, in, estate.Derive)
		// Running last is what stops a span an earlier fold closed being attributed twice (ADR-0134 §5).
		if err := closeSpansByID(ctx, qtx, spanIDs, observedAt, drift.ReasonDescoped, batchID); err != nil {
			return err
		}
		if out != nil {
			*out = append(*out, narrowings...)
		}
	}
	// A tombstone is the only mover its act will have, so an unexhausted one stays (ADR-0134 §5.1).
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

func composeSeedWithdrawals(rows []db.ListSeedWithdrawalCandidatesRow, pending []db.ListPendingSeedWithdrawalsRow, in membershipInputs, derive func(netip.Addr) custody.Custody) ([]int64, []message.NarrowingReceipt) {
	spanIDs, order, counts := composeWithdrawnGround(rows, func(addr netip.Addr) *netip.Prefix {
		return coveringSeedWithdrawal(addr, pending)
	}, in.seeds, derive)

	receipts := make([]message.NarrowingReceipt, 0, len(order))
	for _, key := range order {
		c := counts[key]
		receipts = append(receipts, message.PreviewSeedWithdrawal(c.scope, len(c.subjects), c.timelines))
	}
	return spanIDs, receipts
}

func composeWithdrawnGround(rows []db.ListSeedWithdrawalCandidatesRow, covering func(netip.Addr) *netip.Prefix, seeds []db.ListSeedsRow, derive func(netip.Addr) custody.Custody) ([]int64, []string, map[string]*withdrawalCount) {
	var spanIDs []int64
	var order []string
	// The fold and the preview must not disagree, so both survivor rules live here alone (#1046).
	counts := map[string]*withdrawalCount{}

	for _, row := range rows {
		addr, ok := subjectAddress(row.SubjectKind, row.SubjectKey)
		if !ok {
			continue
		}
		p := covering(addr)
		// A closure with no mover to name cannot be traced to the operator's own act (ADR-0134).
		if p == nil {
			continue
		}
		// Read live, so a re-declared CIDR is not closed by its own stale tombstone (ADR-0134 §4).
		if addressSeedCovered(addr, seeds) {
			continue
		}
		// The extension outlives an address Seed, so closing a still-probed address flaps (ADR-0134 §4).
		if derive != nil && derive(addr) == custody.Operator {
			continue
		}
		key := p.String()
		c, seen := counts[key]
		if !seen {
			// An address Seed's scope IS its CIDR, so there is no wider declared scope to find (ADR-0074).
			c = &withdrawalCount{scope: key, subjects: map[string]bool{}}
			counts[key] = c
			order = append(order, key)
		}
		spanIDs = append(spanIDs, row.ID)
		c.timelines++
		if !c.subjects[row.SubjectKey] {
			c.subjects[row.SubjectKey] = true
		}
	}
	return spanIDs, order, counts
}

func pendingWithdrawnCIDRs(pending []db.ListPendingSeedWithdrawalsRow) []string {
	out := make([]string, 0, len(pending))
	for _, w := range pending {
		// The read filters to kind='address', where the shape CHECK forbids a NULL (ADR-0135 §2).
		if w.AddressCidr == nil {
			continue
		}
		out = append(out, w.AddressCidr.String())
	}
	return out
}

type SeedWithdrawalPreviewStore interface {
	EstateStore
	ListSeeds(ctx context.Context) ([]db.ListSeedsRow, error)
	ListSeedWithdrawalCandidates(ctx context.Context, cidrs []string) ([]db.ListSeedWithdrawalCandidatesRow, error)
}

func SeedWithdrawalReceipt(ctx context.Context, q SeedWithdrawalPreviewStore, asOf time.Time, cidr netip.Prefix) (message.NarrowingReceipt, error) {
	rows, err := q.ListSeedWithdrawalCandidates(ctx, []string{cidr.String()})
	if err != nil {
		return message.NarrowingReceipt{}, err
	}
	seeds, err := q.ListSeeds(ctx)
	if err != nil {
		return message.NarrowingReceipt{}, err
	}
	estate, _, err := hotEstate(ctx, q, asOf)
	if err != nil {
		return message.NarrowingReceipt{}, err
	}
	// The Seed is still declared at preview time, so leaving it in would spare every address (#1046).
	estate.AddressScopes = withoutPrefix(estate.AddressScopes, cidr)

	_, _, counts := composeWithdrawnGround(rows, func(addr netip.Addr) *netip.Prefix {
		if !cidr.Contains(addr) {
			return nil
		}
		return &cidr
	}, withoutAddressSeed(seeds, cidr), estate.Derive)

	// The estate moves between preview and fold, so the two counts need not agree (#1046).
	if c := counts[cidr.String()]; c != nil {
		return message.PreviewSeedWithdrawal(cidr.String(), len(c.subjects), c.timelines), nil
	}
	return message.PreviewSeedWithdrawal(cidr.String(), 0, 0), nil
}

func withoutPrefix(prefixes []netip.Prefix, drop netip.Prefix) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(prefixes))
	for _, p := range prefixes {
		if p != drop {
			out = append(out, p)
		}
	}
	return out
}

func withoutAddressSeed(seeds []db.ListSeedsRow, drop netip.Prefix) []db.ListSeedsRow {
	out := make([]db.ListSeedsRow, 0, len(seeds))
	for _, s := range seeds {
		if s.Kind == "address" && s.AddressCidr != nil && *s.AddressCidr == drop {
			continue
		}
		out = append(out, s)
	}
	return out
}

func coveringSeedWithdrawal(addr netip.Addr, pending []db.ListPendingSeedWithdrawalsRow) *netip.Prefix {
	for _, w := range pending {
		if w.AddressCidr == nil {
			continue
		}
		// A withdrawal carries no precedence the way a Seed does, so first match is the whole rule.
		if w.AddressCidr.Contains(addr) {
			cidr := *w.AddressCidr
			return &cidr
		}
	}
	return nil
}
