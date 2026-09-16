package main

import (
	"context"
	"fmt"
	"log"
	"maps"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

// An Act may be rendered beside a derived figure and may not be read into one (ADR-1946 §5).

type scopeActStore interface {
	ListActsOfClassesSince(ctx context.Context, arg db.ListActsOfClassesSinceParams) ([]db.Act, error)
}

const (
	scopeActWindow = 7 * 24 * time.Hour
	scopeActRows   = 5

	// One ANY() read now spans seven classes, and a bulk decline can fill it alone (#2221).
	scopeActReadCap int32 = 350

	exclusionKindAddress = "address"
)

// A decline writes an address exclusion, so it moves addressScopeCovered too (ADR-2171 §2).

var addressScopeActVerbs = map[string]string{
	act.SeedDeclared{}.Class():          "declared",
	act.SeedWithdrawn{}.Class():         "withdrawn",
	act.ProposalConfirmed{}.Class():     "confirmed",
	act.ExclusionDeclared{}.Class():     "excluded",
	act.ExclusionLifted{}.Class():       "exclusion lifted",
	act.ProposalDeclined{}.Class():      "declined",
	act.ProposalDeclineUndone{}.Class(): "decline lifted",
}

var addressScopeActClasses = slices.Sorted(maps.Keys(addressScopeActVerbs))

type scopeActRow struct {
	When  string
	ISO   string
	Actor string
	Scope string
	Verb  string
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
	case act.ProposalDeclined:
		return v.Scope, v.Kind == exclusionKindAddress
	case act.ProposalDeclineUndone:
		return v.Scope, v.Kind == exclusionKindAddress
	}
	return "", false
}

func (s *server) recentAddressScopeActs(ctx context.Context) (out []scopeActRow, capped bool, err error) {
	now := s.now()
	from := pgtype.Timestamptz{Time: now.UTC().Add(-scopeActWindow), Valid: true}

	rows, err := s.scopeActStore.ListActsOfClassesSince(ctx, db.ListActsOfClassesSinceParams{
		Actions:  addressScopeActClasses,
		FromTime: from,
		// Only an address scope renders, so the read caps above the render (ADR-1946 §3).
		MaxActs: scopeActReadCap,
	})
	if err != nil {
		return nil, false, err
	}
	// A name-scope burst can fill the read, and an empty panel would then read as no edit.
	capped = len(rows) == int(scopeActReadCap)

	out = make([]scopeActRow, 0, scopeActRows)
	for _, row := range rows {
		a, derr := act.DecodeSubject(row.Action, row.Subject)
		if derr != nil {
			return nil, false, fmt.Errorf("act %d: %w", row.ID, derr)
		}
		// The declare path decides the same question about the same string (ADR-1946 §7).
		scope, isAddress := addressScopeOf(a)
		if !isAddress {
			continue
		}
		actor, aerr := act.DecodeActor(row.ActorKind, row.Actor)
		if aerr != nil {
			return nil, false, fmt.Errorf("act %d: %w", row.ID, aerr)
		}
		out = append(out, scopeActRow{
			When:  relTime(row.CreatedAt.Time, now),
			ISO:   row.CreatedAt.Time.UTC().Format(time.RFC3339),
			Actor: act.ActorCell(actor),
			Scope: scope,
			Verb:  addressScopeActVerbs[row.Action],
		})
		// A filled render says five rendered, never that the read saw no more (#2188).
		if len(out) == scopeActRows {
			break
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
