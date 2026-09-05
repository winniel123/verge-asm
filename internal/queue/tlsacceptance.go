package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

func (d *Dispatcher) fanOutTLSAcceptance(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	// A scope withdrawn since a Service was reached must not be re-enumerated (ADR-0079, #742).
	estate, _, err := hotEstate(ctx, qtx, d.now())
	if err != nil {
		return 0, err
	}
	services, err := reachedServices(ctx, qtx)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	// The Services carry their own ports, so the aperture is the open-Service set (ADR-0028).
	for _, j := range scan.BuildTLSAcceptanceJobs(scanID, estate, services, vantages.scanVantages()) {
		if err := enqueueTLSAcceptanceJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

func reachedServices(ctx context.Context, q *db.Queries) ([]scan.ReachedService, error) {
	// A target the enumeration cannot name is skipped, never fabricated from a partial row (ADR-0207).
	rows, err := q.ListReachedServices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]scan.ReachedService, 0, len(rows))
	for _, r := range rows {
		if !r.VantageID.Valid {
			continue
		}
		ap, ok := parseServiceKey(r.ServiceKey)
		if !ok {
			continue
		}
		out = append(out, scan.ReachedService{
			VantageID: r.VantageID.Int64,
			Address:   ap.Addr().String(),
			Port:      ap.Port(),
		})
	}
	return out, nil
}

func parseServiceKey(key string) (netip.AddrPort, bool) {
	// A rendered key parsed back, which ADR-0208 refuses and its own ticket removes (#1320).
	base, ok := strings.CutSuffix(key, "/tcp")
	if !ok {
		return netip.AddrPort{}, false
	}
	ap, err := netip.ParseAddrPort(base)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return ap, true
}

func enqueueTLSAcceptanceJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.TLSAcceptanceJob) error {
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
