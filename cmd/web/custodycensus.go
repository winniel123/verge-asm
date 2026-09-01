package main

import (
	"context"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/queue"
	"github.com/winniel123/verge-asm/internal/retention"
)

// The custody-extension census (ADR-0129 §5, #987) is the display half of the
// `edge-fanout` veto. A vetoed edge never becomes a `Subject`, holds no `Custody`,
// opens no `Gap` and queues no probe — so without this section an operator asking
// *why is that address not covered?* finds nothing at all on screen.
//
// The register is fixed at DISPLAY. It is not silence: the answer and the remedy
// are both here. It is NOT a message: a veto WITHHOLDS a probe, which is the safe
// direction, and the coverage-class message exists for the dangerous one — the
// probing gate opening with no Declared act behind it. That justification does not
// transfer, so nothing on this path fires a message.
//
// The web layer has no Estate assembler of its own (see vantageclass.go for the
// same note): internal/queue/hot.go:hotEstate builds one at batch time. This file
// assembles the same inputs from the same four reads for the render, and reaches
// the measurement through queue.ReadEdgeFanout — the ONE read path from the leaf's
// store to the derivation — so the render and the gate cannot apply different
// absence rules.

// custodyCensusRow is one row of the custody-extension census, shaped for
// scope.tmpl. It carries FACTS ALONE: the citing name, the edge, why the extension
// does not reach it, and the declared address scope that also covers it on a
// dual-limb row. The template writes the words.
//
// There is no count and no threshold here, and there must never be one. The fan-out
// figure and the boundary it is compared against are versioned parameters of the
// `Custody` derivation, locked by the `custody/v1` corpus, and a row that rendered
// either would put a product-chosen number in front of the operator (ADR-0129 §5).
type custodyCensusRow struct {
	// Name is the in-zone Name holding the A record — the name the operator
	// recognises and the one thing they can act on.
	Name string
	// Address is the edge that name's direct A record cites.
	Address string
	// Declined is true where fan-out measured the edge as shared and the extension
	// declined the reach; false where the Scan is in force and has not measured the
	// candidate yet, which is the *pending* row. The census carries no third state
	// — a reached edge is an ordinary covered address and is not this section's
	// business.
	Declined bool
	// Scope is the declared address scope that ALSO covers the edge, empty where
	// none does. Set, it is ADR-0129's dual-limb row: declined by the extension and
	// covered by an address-scope `Seed` at once. A bare *declined* would be true
	// about the extension and read as a contradiction to the person the census
	// exists for, and dropping the row would hide a decline they need if they later
	// withdraw the `Seed`.
	Scope string
}

// custodyCensusStore is the read surface the census needs beyond what the Scope
// render already holds. Three of the four reads mirror hotEstate's; the fourth is
// queue.ReadEdgeFanout's own, which this interface embeds so one value satisfies
// both.
type custodyCensusStore interface {
	queue.EdgeFanoutStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListExtendedZoneDomains(ctx context.Context) ([]pgtype.Text, error)
	NameCitedAddresses(ctx context.Context, arg db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error)
}

// custodyExtensionEstate assembles the Custody derivation's inputs for a render, at
// asOf. It mirrors internal/queue/hot.go:hotEstate read for read — the declared
// address scopes, the custody-extended zones, the current cited addresses through
// the live-tier gate, and the `edge-fanout` measurement — so the census names the
// same declines the dispatch acts on.
//
// A read that FAILS returns the error. The caller degrades the section rather than
// rendering a row: a census that fabricated one on a database error would name a
// decline that did not happen.
func custodyExtensionEstate(ctx context.Context, q custodyCensusStore, asOf time.Time) (custody.Estate, error) {
	scopes, err := q.ListAddressScopeCidrs(ctx)
	if err != nil {
		return custody.Estate{}, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, *p)
		}
	}

	zones, err := q.ListExtendedZoneDomains(ctx)
	if err != nil {
		return custody.Estate{}, err
	}
	var extended []string
	for _, z := range zones {
		if z.Valid {
			extended = append(extended, z.String)
		}
	}

	cited, err := q.NameCitedAddresses(ctx, db.NameCitedAddressesParams{
		AsOf:          pgtype.Timestamptz{Time: asOf.UTC(), Valid: true},
		FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		return custody.Estate{}, err
	}
	var resolutions []custody.Resolution
	for _, c := range cited {
		addr, perr := netip.ParseAddr(c.Address)
		if perr != nil {
			continue
		}
		resolutions = append(resolutions, custody.Resolution{Owner: c.SubjectKey, Address: addr.Unmap()})
	}

	fanout, err := queue.ReadEdgeFanout(ctx, q)
	if err != nil {
		return custody.Estate{}, err
	}

	return custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
		EdgeFanout:    fanout,
	}, nil
}

// toCustodyCensusRows shapes the derivation's census entries for scope.tmpl. It is
// the pure half of the render, so the row states are tested without a database.
func toCustodyCensusRows(entries []custody.ExtensionCensusEntry) []custodyCensusRow {
	out := make([]custodyCensusRow, 0, len(entries))
	for _, e := range entries {
		row := custodyCensusRow{
			Name:     e.Name,
			Address:  e.Address.String(),
			Declined: e.State == custody.ExtensionDeclined,
		}
		if e.Scope.IsValid() {
			row.Scope = e.Scope.String()
		}
		out = append(out, row)
	}
	return out
}

// custodyCensus reads the custody-extension census for the Scope render. It is
// best-effort and additive: a read failure returns the error and renderSeeds
// degrades the section to an honest note rather than 500ing the whole screen or
// showing a row nothing measured.
func (s *server) custodyCensus(ctx context.Context) ([]custodyCensusRow, error) {
	estate, err := custodyExtensionEstate(ctx, s.store, s.now().UTC())
	if err != nil {
		return nil, err
	}
	return toCustodyCensusRows(estate.ExtensionCensus()), nil
}
