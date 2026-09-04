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

// A veto withholds a probe, the safe direction, so it is displayed, never messaged (ADR-0129 #944).

type custodyCensusRow struct {
	Name    string
	Address string

	// Dropping a dual-limb row hides a decline the operator needs if they withdraw the Seed (#987).

	Scope string
}

type custodyCensusView struct {
	Rows    []custodyCensusRow
	Pending int
}

type custodyCensusStore interface {
	queue.EdgeFanoutStore

	// Skipping this read would name a scope an exclusion has already withdrawn (ADR-0133 §1).

	queue.AddressExclusionStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListExtendedZoneDomains(ctx context.Context) ([]pgtype.Text, error)
	NameCitedAddresses(ctx context.Context, arg db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error)
}

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

	excluded, err := queue.ReadAddressExclusions(ctx, q)
	if err != nil {
		return custody.Estate{}, err
	}

	// Assembled before the measurement is read, so that read can bind to these candidates (#1036).
	estate := custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
	}.WithAddressExclusions(excluded)

	// edge_fanout_observation is never pruned, so an unbound read's cost grows without bound (#985).
	fanout, err := queue.ReadEdgeFanout(ctx, q, queue.EdgeFanoutOver(estate.ExtensionCandidates()))
	if err != nil {
		return custody.Estate{}, err
	}

	return estate.WithEdgeFanout(fanout), nil
}

func toCustodyCensusView(entries []custody.ExtensionCensusEntry) custodyCensusView {
	var view custodyCensusView
	held := make(map[netip.Addr]struct{})
	for _, e := range entries {
		// A default arm would count an unknown state as pending, which nothing measured (ADR-0110).
		switch e.State {
		case custody.ExtensionDeclined:
			// The row carries no fan-out count and no threshold: no product-chosen number (ADR-0129 #944).
			row := custodyCensusRow{Name: e.Name, Address: e.Address.String()}
			if e.Scope.IsValid() {
				row.Scope = e.Scope.String()
			}
			view.Rows = append(view.Rows, row)
		case custody.ExtensionPending:
			// A row per pending edge renders thousands of identical rows on a zone's first load (#1015).
			held[e.Address] = struct{}{}
		}
	}
	view.Pending = len(held)
	return view
}

func (s *server) custodyCensus(ctx context.Context) (custodyCensusView, error) {
	estate, err := custodyExtensionEstate(ctx, s.store, s.now().UTC())
	if err != nil {
		return custodyCensusView{}, err
	}
	return toCustodyCensusView(estate.ExtensionCensus()), nil
}
