package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

// An Act may be rendered beside a derived figure and may not be read into one (ADR-1946 §5).

type scopeActStore interface {
	ListActsOfClassSince(ctx context.Context, arg db.ListActsOfClassSinceParams) ([]db.Act, error)
}

const (
	scopeActWindow = 7 * 24 * time.Hour
	scopeActRows   = 5

	scopeActReadCap int32 = 50
)

type scopeActRow struct {
	When  string
	ISO   string
	Actor string
	Scope string
}

func (s *server) recentAddressScopeActs(ctx context.Context) (out []scopeActRow, capped bool, err error) {
	now := s.now()
	rows, err := s.scopeActStore.ListActsOfClassSince(ctx, db.ListActsOfClassSinceParams{
		Action:   act.SeedDeclared{}.Class(),
		FromTime: pgtype.Timestamptz{Time: now.UTC().Add(-scopeActWindow), Valid: true},
		// Only an address scope renders, so the read caps above the render (ADR-1946 §3).
		MaxActs: scopeActReadCap,
	})
	if err != nil {
		return nil, false, err
	}
	// A name-scope burst can fill the read, and an empty panel would then read as no scope.
	capped = len(rows) == int(scopeActReadCap)
	out = make([]scopeActRow, 0, scopeActRows)
	for _, row := range rows {
		a, err := act.DecodeSubject(row.Action, row.Subject)
		if err != nil {
			return nil, false, fmt.Errorf("act %d: %w", row.ID, err)
		}
		scope := a.Subject()
		// The declare path decides the same question about the same string (ADR-1946 §7).
		if !isAddressValue(scope) {
			continue
		}
		actor, err := act.DecodeActor(row.ActorKind, row.Actor)
		if err != nil {
			return nil, false, fmt.Errorf("act %d: %w", row.ID, err)
		}
		out = append(out, scopeActRow{
			When:  relTime(row.CreatedAt.Time, now),
			ISO:   row.CreatedAt.Time.UTC().Format(time.RFC3339),
			Actor: act.ActorCell(actor),
			Scope: scope,
		})
		if len(out) == scopeActRows {
			return out, false, nil
		}
	}
	return out, capped, nil
}

func (s *server) fillScopeActPanel(ctx context.Context, acct db.Account, data map[string]any) {
	// The Act corpus refuses a viewer at the audit tab (ADR-0173 §1).
	if acct.Role != roleAdmin {
		return
	}
	data["ScopeActPanel"] = true
	rows, capped, err := s.recentAddressScopeActs(ctx)
	if err != nil {
		// An empty list would read as no scope declared (ADR-0168 §4).
		log.Printf("web: exposure: recent address scope acts: %v", err)
		data["ScopeActsUnread"] = true
		return
	}
	data["ScopeActs"] = rows
	data["ScopeActsCapped"] = capped
	data["ScopeActReadCap"] = scopeActReadCap
}
