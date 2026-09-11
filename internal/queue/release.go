package queue

import (
	"context"
	"fmt"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// Root determination reads gap-crossing history only the fold holds, so only the census moves here.

type releaseStore interface {
	ListSubjectsOpenedSinceBatch(ctx context.Context, batchID int64) ([]db.ListSubjectsOpenedSinceBatchRow, error)
	ReleaseHeldMessage(ctx context.Context, arg db.ReleaseHeldMessageParams) (int64, error)
}

func releaseHeldMessage(ctx context.Context, q releaseStore, row db.ListReleasableHeldMessagesRow, enqueue enqueueFunc) (bool, error) {
	basis, err := message.ParseCensusBasis(row.CensusBasis)
	if err != nil {
		return false, fmt.Errorf("queue: census basis of message %d: %w", row.ID, err)
	}
	batchID := row.CensusPendingAfterBatch.Int64
	subjects, err := q.ListSubjectsOpenedSinceBatch(ctx, batchID)
	if err != nil {
		return false, fmt.Errorf("queue: subjects opened since batch %d: %w", batchID, err)
	}
	census := censusBeneathRoot(basisRoot(basis), openedSubjects(subjects))
	payload, err := census.Marshal()
	if err != nil {
		return false, fmt.Errorf("queue: census of message %d: %w", row.ID, err)
	}
	n, err := q.ReleaseHeldMessage(ctx, db.ReleaseHeldMessageParams{
		ID:       row.ID,
		Census:   payload,
		Headline: message.ReleasedMembershipHeadline(row.Headline, census),
	})
	if err != nil {
		return false, fmt.Errorf("queue: release message %d: %w", row.ID, err)
	}
	// A pass that took no row lost the claim, so the winner owns the one delivery (ADR-1806 §2).
	if n == 0 {
		return false, nil
	}
	if _, err := enqueue(ctx, row.ID, message.Class(row.Class)); err != nil {
		return false, fmt.Errorf("queue: enqueue delivery for message %d: %w", row.ID, err)
	}
	return true, nil
}

// A revealed firing fires at the Seed and names no root, so the root is read from the basis (§5.3).

func basisRoot(b message.CensusBasis) spanChange {
	return spanChange{SubjectKind: b.RootKind, SubjectKey: b.RootKey, Value: b.RootValue}
}

func openedSubjects(rows []db.ListSubjectsOpenedSinceBatchRow) []subjectRef {
	out := make([]subjectRef, 0, len(rows))
	for _, r := range rows {
		out = append(out, subjectRef{kind: r.SubjectKind, key: r.SubjectKey})
	}
	return out
}

var _ releaseStore = (*db.Queries)(nil)
