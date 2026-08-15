package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The hot Scan (v1 spec §3.4/§3.5) fans out one connect-outcome job per Vantage
// over the addresses Custody admits for that Vantage's class, connecting to the
// verge-core TCP pairs. This file assembles the Custody derivation's inputs from
// the confirmed Seeds and the current resolutions, applies the operator's
// verge-core frequency edits over the shipped default, and enqueues the gated
// jobs. Nothing here probes a target the gate refuses — the gate runs in
// scan.BuildHotJobs before a job is ever enqueued (ADR-0019).

// fanOutHot enqueues one hot job per Vantage over the Custody-admitted addresses.
func (d *Dispatcher) fanOutHot(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	estate, addrs, err := hotEstate(ctx, qtx)
	if err != nil {
		return 0, err
	}
	core, err := hotCore(ctx, qtx)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, j := range scan.BuildHotJobs(scanID, estate, addrs, vantages.scanVantages(), core) {
		if err := enqueueHotJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

// hotEstate builds the Custody derivation's inputs from the confirmed Seeds and
// the current resolutions, and returns the candidate address set the hot Scan
// would probe (the addresses names currently resolve to). The gate then admits
// or refuses each per Vantage class.
func hotEstate(ctx context.Context, q *db.Queries) (custody.Estate, []netip.Addr, error) {
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

	cited, err := q.NameCitedAddresses(ctx)
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

	return custody.Estate{
		AddressScopes: prefixes,
		ExtendedZones: extended,
		Resolutions:   resolutions,
	}, addrs, nil
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
		fe = append(fe, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action})
	}
	return vergecore.Default().WithFrequencyEdits(fe), nil
}

// enqueueHotJob enqueues one connect-outcome job for one Vantage. Its recorded
// scope carries the admitted addresses and the verge-core port sets by content;
// its offers carry the safety profile. It retries like a dns job — a connect is
// a network step that can transiently fail.
func enqueueHotJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.HotJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:vantage:%d", scanID, j.VantageID))
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
