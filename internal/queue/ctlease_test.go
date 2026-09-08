package queue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

type leaseDBTX struct {
	cursorDBTX
	renewals int
	cancelAt int
}

func (l *leaseDBTX) Exec(_ context.Context, sql string, _ ...interface{}) (pgconn.CommandTag, error) {
	if !strings.HasPrefix(sql, "-- name: RenewJobLease") {
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	l.renewals++
	if l.cancelAt > 0 && l.renewals >= l.cancelAt {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func fullEntriesBody() []byte {
	type entry struct {
		LeafInput string `json:"leaf_input"`
		ExtraData string `json:"extra_data"`
	}
	entries := make([]entry, ctTailBatch)
	b, _ := json.Marshal(map[string]any{"entries": entries})
	return b
}

func fullDataTile() []byte {
	// Timestamp, x509 type, 24-bit-prefixed one-byte cert, two empties (static-ct-api).
	leaf := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0x30, 0, 0, 0, 0}
	tile := make([]byte, 0, len(leaf)*scan.CTTileWidth)
	for i := 0; i < scan.CTTileWidth; i++ {
		tile = append(tile, leaf...)
	}
	return tile
}

func fullWindowCases() []struct {
	name  string
	lg    scan.CTLog
	fetch CTFetcher
} {
	rfc := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(maxEntriesPerPoll, make([]byte, 32))},
		{sub: "get-entries", status: 200, body: fullEntriesBody()},
	}}
	tiled := routeFetcher{routes: []route{
		{sub: "checkpoint", status: 200, body: tiledCheckpoint(maxEntriesPerPoll)},
		{sub: "tile/data", status: 200, body: fullDataTile()},
	}}
	return []struct {
		name  string
		lg    scan.CTLog
		fetch CTFetcher
	}{
		{"rfc", scan.CTLog{LogID: "a", URL: "https://rfc.example"}, rfc},
		{"tiled", scan.CTLog{LogID: "b", URL: "https://tiled.example", Tiled: true}, tiled},
	}
}

func TestCTTailRenewsItsLeaseOncePerReservation(t *testing.T) {
	wantFetches := 1 + maxEntriesPerPoll/ctTailBatch
	for _, tc := range fullWindowCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := &countingFetcher{inner: tc.fetch}
			th := &slotThrottle{failAt: 1 << 30}
			lease := &leaseDBTX{}
			w := tailWorkerOn(lease, f, th)
			err := w.completeCTTail(context.Background(), db.ClaimJobRow{ID: 9}, tailSpec(t, tc.lg))
			if errors.Is(err, errJobCanceled) {
				t.Fatalf("a renewed lease must never read as a cancellation: %v", err)
			}
			if f.calls != wantFetches {
				t.Fatalf("a full %d-entry window fetched %d time(s), want %d (the head plus one per batch)", maxEntriesPerPoll, f.calls, wantFetches)
			}
			if th.calls != wantFetches {
				t.Fatalf("reserved %d slot(s), want %d", th.calls, wantFetches)
			}
			if lease.renewals != wantFetches {
				t.Fatalf("renewed the lease %d time(s), want one per reservation (%d): a %d-fetch window at %s per slot outlives "+
					"DefaultStaleJobThreshold (%s) unless the owner renews claimed_at as it goes (#1709)",
					lease.renewals, wantFetches, wantFetches, crtshInterval, DefaultStaleJobThreshold)
			}
		})
	}
}

func TestCTTailEndsWithoutAFetchWhenItsLeaseIsGone(t *testing.T) {
	for _, tc := range fullWindowCases() {
		t.Run(tc.name, func(t *testing.T) {
			f := &countingFetcher{inner: tc.fetch}
			th := &slotThrottle{failAt: 1 << 30}
			lease := &leaseDBTX{cancelAt: 2}
			w := tailWorkerOn(lease, f, th)
			// No pool: a transaction opened after the lost lease would panic here.
			err := w.completeCTTail(context.Background(), db.ClaimJobRow{ID: 9}, tailSpec(t, tc.lg))
			if err != nil {
				t.Fatalf("a job whose row was reaped or terminated owes nothing more; got %v, want nil", err)
			}
			if f.calls != 1 {
				t.Fatalf("fetched %d time(s) after the lease was lost, want 1: the head alone, before the zero-row renewal", f.calls)
			}
			if th.calls != 1 {
				t.Fatalf("reserved %d slot(s), want 1: a lost lease ends the loop before the next reservation", th.calls)
			}
			if lease.renewals != 2 {
				t.Fatalf("renewed %d time(s), want 2: the granted head renewal and the refused second", lease.renewals)
			}
		})
	}
}

func TestRenewJobLeaseReportsAZeroRowRenewalAsCanceled(t *testing.T) {
	w := tailWorkerOn(&leaseDBTX{cancelAt: 1}, nil, nil)
	if err := w.renewJobLease(context.Background(), 9); !errors.Is(err, errJobCanceled) {
		t.Fatalf("renewJobLease on a row no longer running = %v, want %v", err, errJobCanceled)
	}
	w = tailWorkerOn(&leaseDBTX{}, nil, nil)
	if err := w.renewJobLease(context.Background(), 9); err != nil {
		t.Fatalf("renewJobLease on a running row = %v, want nil", err)
	}
}
