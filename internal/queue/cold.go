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

func (d *Dispatcher) fanOutCold(ctx context.Context, scanID, dispatchID int64) (int, error) {
	// An opt-in only flips the tier enabled; the monthly cadence tick is the sole firing (ADR-0044).
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
	// MayProbe refuses these anyway, but a walk of an excluded range burns the whole cadence.
	addrs := candidateAddrs(resolved, scope.AddressPrefixes, estate.AddressExcluded)
	jobs := scan.BuildColdJobs(scanID, estate, addrs, vantages.scanVantages(), scope)
	return streamEnqueue(ctx, d, jobs, func(ctx context.Context, qtx *db.Queries, j scan.ColdJob) error {
		return enqueueColdJob(ctx, qtx, scanID, dispatchID, j)
	})
}

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
				domains = append(domains, s.NameDomain.String)
			}
		}
	}

	addrs := map[netip.Addr]bool{}
	for _, r := range resolutions {
		// A ToLower plus HasSuffix would fold Unicode octets the protocol does not (ADR-0055).
		if custody.WithinAnyZone(r.Owner, domains) {
			addrs[r.Address.Unmap()] = true
		}
	}
	return scan.ColdScope{AddressPrefixes: prefixes, Addresses: addrs}, nil
}

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
