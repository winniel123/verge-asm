package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/seed"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The hot Scan (v1 spec §3.4/§3.5) fans out one connect-outcome job per Vantage
// over the addresses Custody admits for that Vantage's class, connecting to the
// verge-core TCP pairs. This file assembles the Custody derivation's inputs from
// the confirmed Seeds and the current resolutions, applies the operator's
// verge-core frequency edits over the shipped default, and enqueues the gated
// jobs. Nothing here probes a target the gate refuses — the gate runs in
// scan.BuildHotJobs before a job is ever enqueued (ADR-0019).

// fanOutHot streams one hot job per (Vantage, Custody-admitted address) — one
// address per Batch (ADR-0005/ADR-0127) — and commits them in chunks. It reads
// the estate on the pool (the Dispatch row is already committed by claimDispatch),
// then consumes the streamed job sequence through streamEnqueue, so no record
// holds the whole scope and an address scope above the cap fans out with bounded
// memory. The hot tier probes every declared address scope, so it enumerates all
// of them (ADR-0047): every declared CIDR is walked daily whether or not a name
// resolves into it.
func (d *Dispatcher) fanOutHot(ctx context.Context, scanID, dispatchID int64) (int, error) {
	estate, resolved, err := hotEstate(ctx, d.q, d.now())
	if err != nil {
		return 0, err
	}
	core, err := hotCore(ctx, d.q)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, d.q)
	if err != nil {
		return 0, err
	}
	addrs := candidateAddrs(resolved, estate.AddressScopes, estate.AddressExcluded)
	jobs := scan.BuildHotJobs(scanID, estate, addrs, vantages.scanVantages(), core)
	return streamEnqueue(ctx, d, jobs, func(ctx context.Context, qtx *db.Queries, j scan.HotJob) error {
		return enqueueHotJob(ctx, qtx, scanID, dispatchID, j)
	})
}

// EstateStore is the read set hotEstate needs. It is an interface because the
// chip-remove preview assembles the same estate from `cmd/web`, whose store is its
// own narrow interface rather than *db.Queries (#1046). A preview that built the
// derivation from a different read set could state a count the fold will not honour.
type EstateStore interface {
	AddressExclusionStore
	EdgeFanoutStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListExtendedZoneDomains(ctx context.Context) ([]pgtype.Text, error)
	NameCitedAddresses(ctx context.Context, arg db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error)
}

// hotEstate builds the Custody derivation's inputs from the confirmed Seeds, the
// current resolutions and the `edge-fanout` measurement, and returns the RESOLVED
// address set (the addresses names currently resolve to). It is the one Estate
// assembler, so every Scan's fan-out gates on the same veto (#985, ADR-0129 §4).
// Each fan-out then unions this set with the
// addresses its own scopes enumerate via candidateAddrs — the hot tier over every
// declared address scope, the cold tier over only its opted-in scopes — so
// neither tier enumerates a scope it does not probe. The gate then admits or
// refuses each address per Vantage class. asOf is the dispatcher's read instant:
// the current resolutions are read through the live-tier gate (#237, ADR-0041),
// so an Address held only by an evidential resolution — one no derivation may
// still read — is not admitted into the probed estate.
func hotEstate(ctx context.Context, q EstateStore, asOf time.Time) (custody.Estate, []netip.Addr, error) {
	scopes, err := q.ListAddressScopeCidrs(ctx)
	if err != nil {
		return custody.Estate{}, nil, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, *p)
		}
	}

	zones, err := q.ListExtendedZoneDomains(ctx)
	if err != nil {
		return custody.Estate{}, nil, err
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
		return custody.Estate{}, nil, err
	}
	var resolutions []custody.Resolution
	seen := map[netip.Addr]struct{}{}
	var addrs []netip.Addr
	for _, c := range cited {
		addr, perr := netip.ParseAddr(c.Address)
		if perr != nil {
			continue
		}
		addr = addr.Unmap()
		resolutions = append(resolutions, custody.Resolution{Owner: c.SubjectKey, Address: addr})
		if _, ok := seen[addr]; !ok {
			seen[addr] = struct{}{}
			addrs = append(addrs, addr)
		}
	}

	// UNBOUND, because this estate has no ONE limb (#1036). hotEstate serves five
	// dispatches — hot, cold, edge-fanout, http-identity, tls-acceptance — and
	// foldAddressExclusionWithdrawals on job completion, and its consumers reach both
	// limbs: the edge-fanout dispatch walks EdgeFanoutPopulation, and every gate over a
	// declared address reads the declaration limb. There is no candidate set to bind to
	// that would serve all six, so binding here could only starve one of them.
	//
	// It is NOT bound on the grounds of being rare. The withdrawal fold above runs on a
	// completion rather than on a tick, so this read is not once a day.
	//
	// SINCE #1046 IT IS ALSO REACHED FROM A CLICK — SeedWithdrawalReceipt, behind
	// `POST /seeds/preview` — so the earlier claim that it is off the render path no
	// longer holds. It is kept unbound anyway, on the same grounds as the paragraph
	// above and on cold.go's precedent, but the cost is now paid by an admin waiting on
	// a page. A bound that served all seven consumers is the fix if that cost bites.
	fanout, err := ReadEdgeFanout(ctx, q, EdgeFanoutUnbounded())
	if err != nil {
		return custody.Estate{}, nil, err
	}

	excluded, err := ReadAddressExclusions(ctx, q)
	if err != nil {
		return custody.Estate{}, nil, err
	}

	// The measurement goes in LAST, through WithEdgeFanout: its errored floor is read
	// per limb, over the extension candidates this estate's own resolutions hold.
	// The exclusions go in before it — they narrow the address-scope limb and the
	// floor reads the extension limb, so the two do not interact (ADR-0133 §1).
	return custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
	}.WithAddressExclusions(excluded).WithEdgeFanout(fanout), addrs, nil
}

// candidateAddrs is a tier's target set BEFORE the Custody gate, as a lazy
// sequence: the addresses names currently resolve to, followed by every address
// the given scopes enumerate (ADR-0047). Enumerating the scopes here — at the DB
// boundary — is what turns a declared CIDR into probe targets and dark
// `not-reached` subjects, closing the #779 gap. The caller passes the scopes its
// tier probes: the hot fan-out passes every declared address scope, the cold
// fan-out only its opted-in scopes, so neither enumerates a scope it discards.
//
// Dedup is against the SMALL resolved set only (ADR-0127): the `seen` map holds
// the resolved addresses, never the whole scope, so a scope above the cap streams
// with bounded memory. An address that is both resolved and inside a scope is
// yielded once — resolved-first — so it is probed once (single-probing). Two
// declared scopes that OVERLAP are not deduped against each other, so the overlap
// probes twice; that is the accepted cost of not holding the whole scope in a
// map, and each probe is its own idempotent Batch. The Custody gate
// (scan.BuildHotJobs / scan.BuildColdJobs) still runs over the result and remains
// total (ADR-0019): every enumerated candidate is an operator address, but the
// denotation precondition can still bar a non-globally-reachable one from an
// internet-class Vantage.
//
// excluded is custody.Estate.AddressExcluded — the declared `address` exclusions
// (ADR-0133 §3). It skips an excluded address out of the SCOPE ENUMERATION alone and
// never out of the resolved set: an exclusion cuts the `Seed` limb, so an excluded
// address a custody extension reaches is still an operator address and is still
// probed. Filtering the resolved set would take it out of the target set entirely,
// which is a removal larger than the declaration the exclusion narrows.
//
// The gate refuses an excluded address either way, so this skip is for COST: an
// excluded /16 inside a declared /8 is 65,536 addresses walked per tick and refused
// one at a time. A nil predicate means no exclusions.
//
// The CALLER passes it, and the cold tier has to pass it explicitly: fanOutCold
// enumerates coldScope's opted-in prefixes rather than the Estate's scopes, so a
// change that reached only the hot path would leave the cold sweep walking the
// excluded range.
func candidateAddrs(resolved []netip.Addr, scopes []netip.Prefix, excluded func(netip.Addr) bool) iter.Seq[netip.Addr] {
	return func(yield func(netip.Addr) bool) {
		seen := make(map[netip.Addr]struct{}, len(resolved))
		for _, a := range resolved {
			a = a.Unmap()
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			if !yield(a) {
				return
			}
		}
		for _, p := range scopes {
			for a := range seed.EnumerateAddresses(p) {
				a = a.Unmap()
				if _, ok := seen[a]; ok {
					continue // an address a name already resolved to — probed once
				}
				if excluded != nil && excluded(a) {
					continue // the operator declared this range is not theirs
				}
				if !yield(a) {
					return
				}
			}
		}
	}
}

// hotCore reads the shipped verge-core and applies the operator's frequency-half
// edits over it (v1 spec §3.5). The sensitive half is never read from the
// database — it ships in the release — so no operator edit can reach it.
func hotCore(ctx context.Context, q *db.Queries) (vergecore.List, error) {
	edits, err := q.ListVergeCoreFrequencyEdits(ctx)
	if err != nil {
		return vergecore.List{}, err
	}
	fe := make([]vergecore.FrequencyEdit, 0, len(edits))
	for _, e := range edits {
		fe = append(fe, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action}) // #nosec G115 (DB port written only via 1..65535-validated edit path)
	}
	return vergecore.Default().WithFrequencyEdits(fe), nil
}

// enqueueHotJob enqueues one connect-outcome job for one Vantage. Its recorded
// scope carries the admitted addresses and the verge-core port sets by content;
// its offers carry the safety profile. It retries like a dns job — a connect is
// a network step that can transiently fail.
func enqueueHotJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.HotJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:vantage:%d:addr:%s", scanID, j.VantageID, jobAddr(j.Addresses)))
	if err != nil {
		return err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	scopeJSON, err := j.AttemptedScope()
	if err != nil {
		return err
	}
	offersJSON, err := j.OffersJSON()
	if err != nil {
		return err
	}
	_, err = qtx.EnqueueJob(ctx, db.EnqueueJobParams{
		ScanID:         scanID,
		VantageID:      pgInt8(j.VantageID),
		DispatchID:     pgInt8(dispatchID),
		Kind:           j.Kind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         offersJSON,
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
