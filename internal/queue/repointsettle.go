package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// The Endpoints a move opens belong to a later hot fold, so the move's own fold counts none.

type rePointSettleStore interface {
	membershipInputStore
	SettleRePointBatch(ctx context.Context, arg db.SettleRePointBatchParams) (int64, error)
	ListRePointMovesForBatch(ctx context.Context, batchID int64) ([]db.ListRePointMovesForBatchRow, error)
	ListResolutionCitersForAddressesAt(ctx context.Context, arg db.ListResolutionCitersForAddressesAtParams) ([]db.ListResolutionCitersForAddressesAtRow, error)
	ListNameRootsOpenedInBatch(ctx context.Context, batchID int64) ([]db.ListNameRootsOpenedInBatchRow, error)
	ListSubjectsOpenedSinceBatch(ctx context.Context, batchID int64) ([]db.ListSubjectsOpenedSinceBatchRow, error)
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
}

// A fold holding no move owes no message, so it settles in one statement beside the rest.

func splitSettleableBatches(rows []db.ListSettleableRePointBatchesRow) (folds, moveless []int64) {
	for _, r := range rows {
		if r.HasMove {
			folds = append(folds, r.ID)
			continue
		}
		moveless = append(moveless, r.ID)
	}
	return folds, moveless
}

func settleRePointFold(ctx context.Context, q rePointSettleStore, batchID int64, settledAt time.Time, enqueue enqueueFunc) (int, error) {
	// The guarded UPDATE is the claim, so a second pass takes no fold and owes no message.
	claimed, err := q.SettleRePointBatch(ctx, db.SettleRePointBatchParams{SettledAt: tstz(settledAt), ID: batchID})
	if err != nil {
		return 0, fmt.Errorf("queue: claim re-point batch %d: %w", batchID, err)
	}
	if claimed == 0 {
		return 0, nil
	}
	rows, err := q.ListRePointMovesForBatch(ctx, batchID)
	if err != nil {
		return 0, fmt.Errorf("queue: re-point moves of batch %d: %w", batchID, err)
	}
	fold := rePointFoldFrom(batchID, rows)
	// A Seed or an Exclusion the operator moved since the fold is read here (ADR-1806 §8).
	in, err := readMembershipInputs(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("queue: declared inputs for batch %d's residue: %w", batchID, err)
	}
	msgs, err := fold.messages(ctx, q, in)
	if err != nil {
		return 0, err
	}
	fired := 0
	for _, m := range msgs {
		params, err := insertParams(m)
		if err != nil {
			return 0, err
		}
		row, err := q.InsertMessage(ctx, params)
		if err != nil {
			return 0, fmt.Errorf("queue: write the re-point message for %s: %w", m.FiredAt, err)
		}
		if _, err := enqueue(ctx, row.ID, m.Class); err != nil {
			return 0, fmt.Errorf("queue: enqueue delivery for message %d: %w", row.ID, err)
		}
		fired++
	}
	return fired, nil
}

// One fold's moves settle together: they share the candidate set that decides the Address root.

type rePointFold struct {
	batchID int64
	instant time.Time
	moves   []rePoint
}

func rePointFoldFrom(batchID int64, rows []db.ListRePointMovesForBatchRow) rePointFold {
	out := rePointFold{batchID: batchID}
	for _, r := range rows {
		// One fold stamps one instant on every span it opened (internal/queue/worker.go).
		out.instant = r.OpenedAt.Time
		out.moves = append(out.moves, rePoint{
			name:          r.SubjectKey,
			discriminator: r.Discriminator,
			vantageID:     r.VantageID,
			source:        r.Source,
			before:        citedIn(r.Previous),
			after:         citedIn(r.Value),
		})
	}
	return out
}

func (f rePointFold) messages(ctx context.Context, q rePointSettleStore, in membershipInputs) ([]*message.Message, error) {
	fresh := map[string]bool{}
	// A fold that gained no address has no Address root, so it needs no estate read.
	if keys := rePointCandidateAddresses(f.moves); len(keys) > 0 {
		citers, err := q.ListResolutionCitersForAddressesAt(ctx, db.ListResolutionCitersForAddressesAtParams{
			Addresses: keys,
			At:        tstz(f.instant),
		})
		if err != nil {
			return nil, fmt.Errorf("queue: citers of batch %d's new addresses: %w", f.batchID, err)
		}
		// The Address root and this residue read one freshness test, so the two partition (#1730).
		for _, a := range addressesNewToEstate(f.moves, in, citersAtInstant(citers)) {
			fresh[a] = true
		}
	}
	// The roots are the move's own fold's, which is the unit ADR-0026 §2 words its test in.
	rootRows, err := q.ListNameRootsOpenedInBatch(ctx, f.batchID)
	if err != nil {
		return nil, fmt.Errorf("queue: membership roots of batch %d: %w", f.batchID, err)
	}
	roots := make([]spanChange, 0, len(rootRows))
	for _, r := range rootRows {
		roots = append(roots, spanChange{SubjectKind: subjectKindName, SubjectKey: r.SubjectKey, Value: r.Value})
	}
	opened, err := q.ListSubjectsOpenedSinceBatch(ctx, f.batchID)
	if err != nil {
		return nil, fmt.Errorf("queue: subjects opened since batch %d: %w", f.batchID, err)
	}
	subjects := openedSubjects(opened)
	var msgs []*message.Message
	for _, mv := range f.moves {
		// An empty residue is no firing, and RePoint refuses one (ADR-0026 §2).
		if m := message.RePoint(mv.name, rePointResidue(mv, subjects, fresh, roots), f.instant); m != nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

var _ rePointSettleStore = (*db.Queries)(nil)
