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
	// The hot tier probes every declared scope, so it walks all of them daily (ADR-0047).
	addrs := candidateAddrs(resolved, estate.AddressScopes, estate.AddressExcluded)
	// The gate runs inside BuildHotJobs, so no refused target is ever enqueued (ADR-0019).
	jobs := scan.BuildHotJobs(scanID, estate, addrs, vantages.scanVantages(), core)
	return streamEnqueue(ctx, d, jobs, func(ctx context.Context, qtx *db.Queries, j scan.HotJob) error {
		return enqueueHotJob(ctx, qtx, scanID, dispatchID, j)
	})
}

// The preview reads through this, so a second read set would state a count the fold breaks (#1046).

type EstateStore interface {
	AddressExclusionStore
	EdgeFanoutStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListExtendedZoneDomains(ctx context.Context) ([]pgtype.Text, error)
	NameCitedAddresses(ctx context.Context, arg db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error)
}

func hotEstate(ctx context.Context, q EstateStore, asOf time.Time) (custody.Estate, []netip.Addr, error) {
	// One assembler means every fan-out gates on the same veto (#985, ADR-0129 §4).
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

	// The live-tier gate keeps an evidential-only Address out of the probed estate (#237, ADR-0041).
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

	// No candidate set serves every consumer of this estate, so binding here would starve one (#1036).
	fanout, err := ReadEdgeFanout(ctx, q, EdgeFanoutUnbounded())
	if err != nil {
		return custody.Estate{}, nil, err
	}

	excluded, err := ReadAddressExclusions(ctx, q)
	if err != nil {
		return custody.Estate{}, nil, err
	}

	// One cuts the Seed limb, the other reads the extension limb, so they cannot mix (ADR-0133 §1).
	return custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
	}.WithAddressExclusions(excluded).WithEdgeFanout(fanout), addrs, nil
}

func candidateAddrs(resolved []netip.Addr, scopes []netip.Prefix, excluded func(netip.Addr) bool) iter.Seq[netip.Addr] {
	return func(yield func(netip.Addr) bool) {
		// The map holds the resolved set alone, so a scope above the cap streams bounded (ADR-0127).
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
		// Each tier passes only the scopes it probes, so neither enumerates a scope it discards.
		for _, p := range scopes {
			// The gate refuses either way, so this skip is for cost: an excluded /16 is 65,536 walks a tick.
			for a := range seed.EnumerateAddresses(p) {
				a = a.Unmap()
				// Overlapping scopes are not deduped, so the overlap probes twice, by choice (ADR-0216 §1).
				if _, ok := seen[a]; ok {
					continue
				}
				// An exclusion cuts the Seed limb alone, so the resolved set is never filtered (ADR-0133 §1).
				if excluded != nil && excluded(a) {
					continue
				}
				if !yield(a) {
					return
				}
			}
		}
	}
}

func hotCore(ctx context.Context, q *db.Queries) (vergecore.List, error) {
	// The sensitive half ships in the release, so no operator edit can reach it (v1-spec §3.5).
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
