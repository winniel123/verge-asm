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
)

// The cold Scan (v1 spec §3.4, ADR-0044) is the full 1–65535 TCP port tier,
// opt-in per `Seed` scope. It reuses the same Custody-derivation inputs as the
// hot Scan and the same connect-outcome enqueue path; the only differences are
// the port set (the full range) and the address scope (only addresses an
// opted-in `Seed` scope covers). It is dispatched only when the operator has
// opted a scope in — which flips the cold Scan enabled and puts it in
// ListEnabledScans — and even then produces no jobs when the opted-in scope
// admits no address, a legible empty state. Nothing here enables the tier or
// fires it on opt-in: the web opt-in handler reconciles the enabled flag, and
// this fan-out runs only on the monthly cadence tick.

// fanOutCold streams one connect-outcome job per (Vantage, admitted address) —
// one address per Batch (ADR-0005/ADR-0127) — over the addresses that are both
// Custody-admitted and inside an opted-in `Seed` scope, across the full TCP port
// range, and commits them in chunks. It reads the estate on the pool (the
// Dispatch row is already committed by claimDispatch), then consumes the streamed
// job sequence through streamEnqueue, so no record holds the whole scope.
//
// The cold tier probes only its opted-in scopes, so it enumerates ONLY those
// address scopes (ADR-0047), never the whole declared estate. BuildColdJobs then
// filters this set to the opted-in scope (address prefixes and the name-resolved
// addresses), so a resolved address outside every opted-in scope still drops out.
func (d *Dispatcher) fanOutCold(ctx context.Context, scanID, dispatchID int64) (int, error) {
	estate, resolved, err := hotEstate(ctx, d.q, d.now())
	if err != nil {
		return 0, err
	}
	scope, err := coldScope(ctx, d.q, estate.Resolutions)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, d.q)
	if err != nil {
		return 0, err
	}
	addrs := candidateAddrs(resolved, scope.AddressPrefixes)
	jobs := scan.BuildColdJobs(scanID, estate, addrs, vantages.scanVantages(), scope)
	return streamEnqueue(ctx, d, jobs, func(ctx context.Context, qtx *db.Queries, j scan.ColdJob) error {
		return enqueueColdJob(ctx, qtx, scanID, dispatchID, j)
	})
}

// coldScope reads the opted-in `Seed` scopes and folds them into the address
// membership the cold Scan probes within: an opted-in address scope contributes
// its CIDR, and an opted-in name scope contributes the addresses its names
// currently resolve to. With no scope opted in the result is empty and the sweep
// fires nothing (ADR-0044).
func coldScope(ctx context.Context, q *db.Queries, resolutions []custody.Resolution) (scan.ColdScope, error) {
	seeds, err := q.ListColdScopeSeeds(ctx)
	if err != nil {
		return scan.ColdScope{}, err
	}
	var prefixes []netip.Prefix
	var domains []string
	for _, s := range seeds {
		switch s.Kind {
		case "address":
			if s.AddressCidr != nil {
				prefixes = append(prefixes, *s.AddressCidr)
			}
		case "name":
			if s.NameDomain.Valid {
				// Raw zone key; custody.WithinAnyZone folds ASCII case and the
				// trailing dot label-wise, so no string-level normalisation here.
				domains = append(domains, s.NameDomain.String)
			}
		}
	}

	addrs := map[netip.Addr]bool{}
	for _, r := range resolutions {
		// The one label-wise, ASCII-folded containment test the model owns: a raw
		// strings.ToLower + HasSuffix here would fold Unicode octets the protocol
		// does not and compare a name as a string (ADR-0055, CONTEXT.md `Name`).
		if custody.WithinAnyZone(r.Owner, domains) {
			addrs[r.Address.Unmap()] = true
		}
	}
	return scan.ColdScope{AddressPrefixes: prefixes, Addresses: addrs}, nil
}

// enqueueColdJob enqueues one connect-outcome job for one Vantage across the full
// TCP range. Its recorded scope carries the admitted addresses and the port
// range by content; its offers carry the safety profile. It retries like a hot
// job — a connect is a network step that can transiently fail.
func enqueueColdJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.ColdJob) error {
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
