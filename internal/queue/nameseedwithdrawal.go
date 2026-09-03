package queue

import (
	"context"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
)

func foldNameSeedWithdrawals(ctx context.Context, qtx *db.Queries, batchID int64, observedAt time.Time, in membershipInputs, out *[]message.NarrowingReceipt) error {
	// A withdrawn Seed stops its Names being enumerated, so no observation reaches them (ADR-0135 §5).
	pending, err := qtx.ListPendingNameSeedWithdrawals(ctx)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	// The cascade takes the admissions but no span, so the tombstone holds the domain (ADR-0135 §4).
	rows, err := qtx.ListNameSeedWithdrawalCandidates(ctx, pendingWithdrawnDomains(pending))
	if err != nil {
		return err
	}
	if len(rows) > 0 {
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
	// An in-flight job frozen before the withdrawal still opens a span after it (ADR-0135 §5).
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

func composeNameSeedWithdrawals(rows []db.ListNameSeedWithdrawalCandidatesRow, pending []db.ListPendingNameSeedWithdrawalsRow, in membershipInputs, admitted []string) ([]int64, []message.NarrowingReceipt) {
	spanIDs, order, counts := composeWithdrawnNameGround(rows, func(name string) string {
		return coveringNameSeedWithdrawal(name, pending)
	}, in.seeds, admitted)

	// A Name act that removes many subjects at once takes the aggregate form (ADR-0135 §1).
	receipts := make([]message.NarrowingReceipt, 0, len(order))
	for _, key := range order {
		c := counts[key]
		receipts = append(receipts, message.PreviewSeedWithdrawal(c.scope, len(c.subjects), c.timelines))
	}
	return spanIDs, receipts
}

func composeWithdrawnNameGround(rows []db.ListNameSeedWithdrawalCandidatesRow, covering func(string) string, seeds []db.ListSeedsRow, admitted []string) ([]int64, []string, map[string]*withdrawalCount) {
	// A SQL test keying names its own way would drop a Name the estate still walks (ADR-0135 §3).
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
		domain := covering(row.SubjectKey)
		if domain == "" {
			continue
		}
		if nameSeedCovered(row.SubjectKey, seeds) {
			continue
		}
		// A surviving Seed's admission keeps the Name enumerated, so closing it would flap (ADR-0135 §3).
		if stillAdmitted[resolutionNameKey(row.SubjectKey)] {
			continue
		}
		c, seen := counts[domain]
		if !seen {
			c = &withdrawalCount{scope: domain, subjects: map[string]bool{}}
			counts[domain] = c
			order = append(order, domain)
		}
		spanIDs = append(spanIDs, row.ID)
		c.timelines++
		c.subjects[row.SubjectKey] = true
	}
	return spanIDs, order, counts
}

func pendingWithdrawnDomains(pending []db.ListPendingNameSeedWithdrawalsRow) []string {
	out := make([]string, 0, len(pending))
	for _, w := range pending {
		if !w.NameDomain.Valid {
			continue
		}
		out = append(out, normalizeDomain(w.NameDomain.String))
	}
	return out
}

type NameSeedWithdrawalPreviewStore interface {
	ListSeeds(ctx context.Context) ([]db.ListSeedsRow, error)
	ListNameSeedWithdrawalCandidates(ctx context.Context, domains []string) ([]db.ListNameSeedWithdrawalCandidatesRow, error)
	ListAdmittedNamesOutsideSeed(ctx context.Context, seedID int64) ([]string, error)
}

func NameSeedWithdrawalReceipt(ctx context.Context, q NameSeedWithdrawalPreviewStore, seedID int64, domain string) (message.NarrowingReceipt, error) {
	domain = normalizeDomain(domain)
	rows, err := q.ListNameSeedWithdrawalCandidates(ctx, []string{domain})
	if err != nil {
		return message.NarrowingReceipt{}, err
	}
	seeds, err := q.ListSeeds(ctx)
	if err != nil {
		return message.NarrowingReceipt{}, err
	}
	// The delete has not cascaded at preview time, so this Seed's own admissions go (ADR-0135 §4).
	admitted, err := q.ListAdmittedNamesOutsideSeed(ctx, seedID)
	if err != nil {
		return message.NarrowingReceipt{}, err
	}

	_, _, counts := composeWithdrawnNameGround(rows, func(name string) string {
		if !nameWithinDomain(name, domain) {
			return ""
		}
		return domain
	}, withoutNameSeed(seeds, domain), admitted)

	if c := counts[domain]; c != nil {
		return message.PreviewSeedWithdrawal(domain, len(c.subjects), c.timelines), nil
	}
	return message.PreviewSeedWithdrawal(domain, 0, 0), nil
}

func withoutNameSeed(seeds []db.ListSeedsRow, drop string) []db.ListSeedsRow {
	out := make([]db.ListSeedsRow, 0, len(seeds))
	for _, s := range seeds {
		if s.Kind == "name" && s.NameDomain.Valid && normalizeDomain(s.NameDomain.String) == drop {
			continue
		}
		out = append(out, s)
	}
	return out
}

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
