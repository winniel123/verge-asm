package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
)

const (
	outcomeOK    = "ok"
	outcomeError = "error"
)

type sourceHealthView struct {
	Label string
	Class string
}

type sourceHealthIndex map[string]db.SourceHealth

func (s *server) sourceHealthIndex(ctx context.Context) (sourceHealthIndex, error) {
	rows, err := s.sourcesStore.ListSourceHealth(ctx)
	if err != nil {
		return nil, err
	}
	idx := make(sourceHealthIndex, len(rows))
	for _, row := range rows {
		idx[row.Slug] = row
	}
	return idx, nil
}

func (idx sourceHealthIndex) view(v sourceView) sourceHealthView {
	if v.KindLabel != "proposer" || !v.Toggleable {
		// ADR-0223 does not rule on the admitting sources, so none reads here (#1583).
		return sourceHealthView{}
	}
	row, ok := idx[v.Slug]
	if !ok {
		// The query path is the only writer, so no row means never attempted (ADR-0223 §4).
		return sourceHealthView{Label: "never attempted", Class: "neutral"}
	}
	at := row.LastAttemptAt.Time.UTC().Format("2006-01-02 15:04 UTC")
	if row.LastOutcome != outcomeError {
		return sourceHealthView{Label: "last attempt succeeded · " + at, Class: "ok"}
	}
	// The count stands in for a derived word nobody has a threshold for (ADR-0223 §4).
	label := "last attempt failed · " + at
	if row.ConsecutiveFailures > 1 {
		label += fmt.Sprintf(" · %d in a row", row.ConsecutiveFailures)
	}
	return sourceHealthView{Label: label, Class: "danger"}
}

func (s *server) recordProposerAttempts(ctx context.Context, attempts []proposer.Attempt) {
	at := pgtype.Timestamptz{Time: s.now().UTC(), Valid: true}
	for _, a := range attempts {
		outcome := outcomeOK
		// A join gap is the source answering correctly, so it accrues no failure (ADR-0227, #1634).
		if a.Err != nil && !errors.Is(a.Err, proposer.ErrNoJoinKey) {
			outcome = outcomeError
		}
		if _, err := s.proposalsStore.RecordSourceAttempt(ctx, db.RecordSourceAttemptParams{
			Slug: a.SourceSlug, LastOutcome: outcome, LastAttemptAt: at,
		}); err != nil {
			// The candidates are the operator's answer, so a lost record never fails the lookup.
			log.Printf("web: record source attempt %s: %v", a.SourceSlug, err)
		}
	}
}
