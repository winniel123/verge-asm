package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

func (d *Dispatcher) fanOutHTTPIdentity(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
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
	for _, j := range scan.BuildHTTPIdentityJobs(scanID, estate, services, vantages.scanVantages()) {
		if err := enqueueHTTPIdentityJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

func enqueueHTTPIdentityJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.HTTPIdentityJob) error {
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
