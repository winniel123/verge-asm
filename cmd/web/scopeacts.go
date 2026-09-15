package main

import (
	"context"
	"fmt"
	"log"
	"sort"
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

	exclusionKindAddress = "address"
)

// The five classes ADR-2114 names, each with its own verb. Two more move the predicate (#2169).

var addressScopeActClasses = []struct {
	class string
	verb  string
}{
	{act.SeedDeclared{}.Class(), "declared"},
	{act.SeedWithdrawn{}.Class(), "withdrawn"},
	{act.ProposalConfirmed{}.Class(), "confirmed"},
	{act.ExclusionDeclared{}.Class(), "excluded"},
	{act.ExclusionLifted{}.Class(), "exclusion lifted"},
}

type scopeActRow struct {
	When  string
	ISO   string
	Actor string
	Scope string
	Verb  string
}

type scopeActAt struct {
	row  db.Act
	verb string
}

// An exclusion carries a Kind, so the address ones separate without re-parsing (ADR-2114 §4).

func addressScopeOf(a act.Act) (scope string, isAddress bool) {
	switch v := a.(type) {
	case act.SeedDeclared:
		return v.Scope, isAddressValue(v.Scope)
	case act.SeedWithdrawn:
		return v.Scope, isAddressValue(v.Scope)
	case act.ProposalConfirmed:
		return v.Scope, isAddressValue(v.Scope)
	case act.ExclusionDeclared:
		return v.Scope, v.Kind == exclusionKindAddress
	case act.ExclusionLifted:
		return v.Scope, v.Kind == exclusionKindAddress
	}
	return "", false
}

func (s *server) recentAddressScopeActs(ctx context.Context) (out []scopeActRow, capped bool, err error) {
	now := s.now()
	from := pgtype.Timestamptz{Time: now.UTC().Add(-scopeActWindow), Valid: true}

	var merged []scopeActAt
	for _, c := range addressScopeActClasses {
		rows, rerr := s.scopeActStore.ListActsOfClassSince(ctx, db.ListActsOfClassSinceParams{
			Action:   c.class,
			FromTime: from,
			// Only an address scope renders, so the read caps above the render (ADR-1946 §3).
			MaxActs: scopeActReadCap,
		})
		if rerr != nil {
			return nil, false, rerr
		}
		// A name-scope burst can fill the read, and an empty panel would then read as no edit.
		capped = capped || len(rows) == int(scopeActReadCap)
		for _, row := range rows {
			merged = append(merged, scopeActAt{row: row, verb: c.verb})
		}
	}
	// ADR-1946 §3 fixes the order, and five per-class reads must be merged to hold it.
	sort.SliceStable(merged, func(i, j int) bool {
		a, b := merged[i].row, merged[j].row
		if !a.CreatedAt.Time.Equal(b.CreatedAt.Time) {
			return a.CreatedAt.Time.After(b.CreatedAt.Time)
		}
		return a.ID > b.ID
	})

	out = make([]scopeActRow, 0, scopeActRows)
	for _, m := range merged {
		a, derr := act.DecodeSubject(m.row.Action, m.row.Subject)
		if derr != nil {
			return nil, false, fmt.Errorf("act %d: %w", m.row.ID, derr)
		}
		// The declare path decides the same question about the same string (ADR-1946 §7).
		scope, isAddress := addressScopeOf(a)
		if !isAddress {
			continue
		}
		actor, aerr := act.DecodeActor(m.row.ActorKind, m.row.Actor)
		if aerr != nil {
			return nil, false, fmt.Errorf("act %d: %w", m.row.ID, aerr)
		}
		out = append(out, scopeActRow{
			When:  relTime(m.row.CreatedAt.Time, now),
			ISO:   m.row.CreatedAt.Time.UTC().Format(time.RFC3339),
			Actor: act.ActorCell(actor),
			Scope: scope,
			Verb:  m.verb,
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
