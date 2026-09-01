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

// custodyCensusRow is one DECLINED row of the custody-extension census, shaped for
// scope.tmpl. It carries FACTS ALONE: the citing name, the edge, and the declared
// address scope that also covers it on a dual-limb row. The template writes the
// words.
//
// A pending candidate gets no row. It is counted in custodyCensusView.Pending and
// stated once — see that field for why.
//
// There is no count and no threshold here, and there must never be one. The fan-out
// figure and the boundary it is compared against are versioned parameters of the
// `Custody` derivation, locked by the `custody/v3` corpus, and a row that rendered
// either would put a product-chosen number in front of the operator (ADR-0129 §5).
type custodyCensusRow struct {
	// Name is the in-zone Name holding the A record — the name the operator
	// recognises and the one thing they can act on.
	Name string
	// Address is the edge that name's direct A record cites.
	Address string
	// Scope is the declared address scope that ALSO covers the edge, empty where
	// none does. Set, it is ADR-0129's dual-limb row: declined by the extension and
	// covered by an address-scope `Seed` at once. A bare *declined* would be true
	// about the extension and read as a contradiction to the person the census
	// exists for, and dropping the row would hide a decline they need if they later
	// withdraw the `Seed`.
	Scope string
}

// custodyCensusView is the whole section, shaped for scope.tmpl: the declined rows,
// and how many candidates still wait on their first measurement (#1015).
type custodyCensusView struct {
	// Rows are the declines, one per (citing name, edge) pair, in resolution order.
	Rows []custodyCensusRow
	// Pending is the number of DISTINCT edges the `edge-fanout` Scan holds and has
	// not measured yet. It is a COUNT and never a row, because a pending candidate
	// carries no remedy: the operator cannot act on it, and the state clears within
	// one Scan cadence with no act of theirs.
	//
	// It counts EDGES and not census entries, and the two differ. The derivation
	// emits one entry per (citing name, edge) pair, so a single unmeasured edge that
	// two hundred in-zone names front is two hundred entries and ONE held edge. A
	// count of entries would read as two hundred edges and inflate the one fact this
	// line carries. The declines count the other way round, one row per citing name,
	// because there the name is what the operator acts on.
	//
	// A row each would make the section's WORST render its FIRST one. The Scan ships
	// enabled and measures nothing until its first Batch completes, so a zone holding
	// thousands of in-estate names would render thousands of rows on the first load
	// of /scope, all of them the same fact, all of them gone by the next day (#1015).
	//
	// The number is a fact this install measured, never a product-chosen one, so it
	// does not collide with ADR-0129 §5. That rule bars the fan-out figure and the
	// boundary it is compared against, both of which stay inside the derivation.
	Pending int
}

// custodyCensusStore is the read surface the census needs beyond what the Scope
// render already holds. Three of the four reads mirror hotEstate's; the fourth is
// queue.ReadEdgeFanout's own, which this interface embeds so one value satisfies
// both.
type custodyCensusStore interface {
	queue.EdgeFanoutStore
	// AddressExclusionStore is the declared `address` exclusions. This census names
	// the address scope that ALSO covers a declined edge, through the same
	// coveringAddressScope an exclusion now narrows (ADR-0133 §1), so an assembler
	// that skipped this read would keep naming a scope that covers the address no
	// longer — and would contradict the address-scope census on the same screen.
	queue.AddressExclusionStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListExtendedZoneDomains(ctx context.Context) ([]pgtype.Text, error)
	NameCitedAddresses(ctx context.Context, arg db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error)
}

// custodyExtensionEstate assembles the Custody derivation's inputs for a render, at
// asOf. It mirrors internal/queue/hot.go:hotEstate read for read — the declared
// address scopes, the declared `address` exclusions, the custody-extended zones, the
// current cited addresses through the live-tier gate, and the `edge-fanout`
// measurement — so the census names the same declines the dispatch acts on.
//
// THE MEASUREMENT IS THE ONE READ THAT NO LONGER MIRRORS. hotEstate takes the whole
// store; this takes queue.EdgeFanoutOver(estate.ExtensionCandidates()), so the returned
// estate carries a measurement of the EXTENSION LIMB ALONE (#1036). The declines it
// names do not move — the census asks about nothing but those candidates, and the
// per-limb errored floor resolves over the same set — but the estate is narrower than
// hotEstate's and must not be treated as interchangeable with it.
//
// ExtensionCensus is its ONLY legal consumer. A reader that walks the measurement
// wholesale instead of looking a candidate's key up would read every declaration-limb
// row the bound left behind as an address the Scan never measured.
// custody.Estate.AddressScopeCensus is that reader, `/coverage` is its surface, and
// addressscopecensus.go assembles its own estate off an unbound read for exactly this
// reason. The record carries custody.EdgeFanout.Partial so that census refuses this
// estate rather than reporting a short count, but the refusal is a backstop and not a
// licence: a new consumer of this estate that needs the declaration limb needs an
// unbound read, not a narrower assertion.
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

	excluded, err := queue.ReadAddressExclusions(ctx, q)
	if err != nil {
		return custody.Estate{}, err
	}

	// The estate is assembled BEFORE the measurement is read, so the read can be BOUND
	// to this estate's own extension candidates (#1036). The exclusions go in here.
	// They narrow the address-scope limb alone, which is the limb this census reads to
	// name a declined edge's covering Scope: an excluded address is covered by no
	// scope, so the dual-limb row becomes a bare decline rather than naming a scope the
	// operator has withdrawn.
	estate := custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
	}.WithAddressExclusions(excluded)

	// BOUND to the extension candidates. This census reads the EXTENSION limb alone —
	// ExtensionCensus walks the resolutions under the two conditions ExtensionCandidates
	// itself applies, and nothing else on this estate reads the measurement — so the
	// unbound read pulled every measured address of every declared address scope to
	// answer a question about a handful of cited direct-A targets (#1036).
	//
	// It is the SAME set WithEdgeFanout resolves the errored floor over, so the bound
	// cannot drop a key either consumer looks up. Computing it twice is one linear pass
	// over the resolutions, and it buys back a read whose cost grows with the estate and
	// with time — `edge_fanout_observation` is never pruned (#985).
	fanout, err := queue.ReadEdgeFanout(ctx, q, queue.EdgeFanoutOver(estate.ExtensionCandidates()))
	if err != nil {
		return custody.Estate{}, err
	}

	// The measurement goes in LAST, through WithEdgeFanout: its errored floor is read
	// per limb, over the extension candidates this estate's own resolutions hold.
	return estate.WithEdgeFanout(fanout), nil
}

// toCustodyCensusView shapes the derivation's census entries for scope.tmpl. It is
// the pure half of the render, so the section's shape is tested without a database.
//
// It SPLITS the entries by state rather than mapping them one for one: a decline
// becomes a row, and a held edge becomes one increment of Pending however many names
// front it. The derivation still yields both — the collapse is a display choice, made
// here, and internal/custody keeps naming every candidate it holds.
//
// The switch names the two states of ADR-0129 §5 and SKIPS anything else. A later
// state absorbed into Pending would be asserted on screen as *awaiting a first
// measurement*, which is a claim about the Scan's progress that nothing measured —
// and this section must never state a fact it does not hold. A state it does not know
// goes unmentioned, which is a silence rather than a wrong number.
func toCustodyCensusView(entries []custody.ExtensionCensusEntry) custodyCensusView {
	var view custodyCensusView
	held := make(map[netip.Addr]struct{})
	for _, e := range entries {
		switch e.State {
		case custody.ExtensionDeclined:
			row := custodyCensusRow{Name: e.Name, Address: e.Address.String()}
			if e.Scope.IsValid() {
				row.Scope = e.Scope.String()
			}
			view.Rows = append(view.Rows, row)
		case custody.ExtensionPending:
			held[e.Address] = struct{}{}
		}
	}
	view.Pending = len(held)
	return view
}

// custodyCensus reads the custody-extension census for the Scope render. It is
// best-effort and additive: a read failure returns the error and renderSeeds
// degrades the section to an honest note rather than 500ing the whole screen or
// showing a row nothing measured.
func (s *server) custodyCensus(ctx context.Context) (custodyCensusView, error) {
	estate, err := custodyExtensionEstate(ctx, s.store, s.now().UTC())
	if err != nil {
		return custodyCensusView{}, err
	}
	return toCustodyCensusView(estate.ExtensionCensus()), nil
}
