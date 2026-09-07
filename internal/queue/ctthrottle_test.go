package queue

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

var errSlotRefused = errors.New("slot refused")

type slotThrottle struct {
	failAt int
	calls  int
}

func (t *slotThrottle) Reserve(context.Context) (time.Time, error) {
	t.calls++
	if t.calls >= t.failAt {
		return time.Time{}, errSlotRefused
	}
	return time.Time{}, nil
}

type countingFetcher struct {
	inner CTFetcher
	calls int
}

func (f *countingFetcher) Fetch(ctx context.Context, url string) (int, []byte, error) {
	f.calls++
	return f.inner.Fetch(ctx, url)
}

type noRowsDBTX struct{}

func (noRowsDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, pgx.ErrNoRows
}

func (noRowsDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func (noRowsDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row { return noRowsRow{} }

type noRowsRow struct{}

func (noRowsRow) Scan(...interface{}) error { return pgx.ErrNoRows }

type cursorDBTX struct{ noRowsDBTX }

func (cursorDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row { return cursorRow{} }

type cursorRow struct{}

func (cursorRow) Scan(dest ...interface{}) error {
	for _, d := range dest {
		switch p := d.(type) {
		case *int64:
			*p = 0
		case *[]byte:
			*p = nil
		}
	}
	return nil
}

func tailWorker(f CTFetcher, t CTThrottle) *Worker {
	return tailWorkerOn(noRowsDBTX{}, f, t)
}

func tailWorkerOn(dbtx db.DBTX, f CTFetcher, t CTThrottle) *Worker {
	return &Worker{
		q:              db.New(dbtx),
		log:            log.New(io.Discard, "", 0),
		now:            time.Now,
		ctTailFetcher:  f,
		ctTailThrottle: t,
	}
}

func verifyWorker(f CTFetcher, t CTThrottle) *Worker {
	return &Worker{
		log:              log.New(io.Discard, "", 0),
		now:              time.Now,
		ctVerifyFetcher:  f,
		ctVerifyThrottle: t,
	}
}

func tailSpec(t *testing.T, lg scan.CTLog) wire.JobSpec {
	t.Helper()
	spec, err := scan.CTTailJob{Log: lg}.JobSpec("test")
	if err != nil {
		t.Fatal(err)
	}
	return spec
}

func tiledCheckpoint(treeSize int) []byte {
	return []byte("example.log\n" + strconv.Itoa(treeSize) + "\n" + base64.StdEncoding.EncodeToString(make([]byte, 32)) + "\n\n— example sig\n")
}

func TestCTTailReservesBeforeEveryFetch(t *testing.T) {
	rfc := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(2, make([]byte, 32))},
		{sub: "get-entries", status: 200, body: []byte("[]")},
	}}
	tiled := routeFetcher{routes: []route{
		{sub: "checkpoint", status: 200, body: tiledCheckpoint(1)},
		{sub: "tile/data", status: 200, body: nil},
	}}
	cases := []struct {
		name      string
		lg        scan.CTLog
		fetch     CTFetcher
		failAt    int
		wantFetch int
		wantSite  string
	}{
		{"rfc get-sth", scan.CTLog{LogID: "a", URL: "https://rfc.example"}, rfc, 1, 0, "get-sth"},
		{"rfc get-entries", scan.CTLog{LogID: "a", URL: "https://rfc.example"}, rfc, 2, 1, "get-entries"},
		{"tiled checkpoint", scan.CTLog{LogID: "b", URL: "https://tiled.example", Tiled: true}, tiled, 1, 0, "checkpoint"},
		{"tiled data tile", scan.CTLog{LogID: "b", URL: "https://tiled.example", Tiled: true}, tiled, 2, 1, "tile/data"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &countingFetcher{inner: tc.fetch}
			th := &slotThrottle{failAt: tc.failAt}
			// With a cursor the second fetch is reachable (#1654).
			err := tailWorkerOn(cursorDBTX{}, f, th).completeCTTail(context.Background(), db.ClaimJobRow{}, tailSpec(t, tc.lg))
			if !errors.Is(err, errSlotRefused) {
				t.Fatalf("err = %v, want the refused reservation to surface", err)
			}
			if f.calls != tc.wantFetch {
				t.Fatalf("%s fetched %d time(s) after a refused reservation, want %d — the reserve must precede it", tc.wantSite, f.calls, tc.wantFetch)
			}
			if th.calls != tc.failAt {
				t.Fatalf("reserve called %d time(s), want %d", th.calls, tc.failAt)
			}
		})
	}
}

func TestCTVerifyReservesBeforeEveryFetch(t *testing.T) {
	_, rfcLogID := firstLog(t, false)
	_, tiledLogID := firstLog(t, true)
	der, _ := selfSignedLeaf(t)
	ts := uint64(1700000000000)

	rfcSCT := serializeSCT(rfcLogID, ts, nil)
	rfcLeaf := scan.LeafHashX509(der, nil, ts)
	rfcFetch := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(1, rfcLeaf)},
		{sub: "get-proof-by-hash", status: 200, body: proofBody(0, nil)},
	}}

	ext := leafIndexExtension(0)
	tiledSCT := serializeSCT(tiledLogID, ts, ext)
	tiledLeaf := scan.LeafHashX509(der, ext, ts)
	tiledFetch := routeFetcher{routes: []route{
		{sub: "checkpoint", status: 200, body: tiledCheckpoint(1)},
		{sub: "tile/0/", status: 200, body: tiledLeaf},
	}}

	logs, err := scan.AllLogs()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		sct       []byte
		fetch     CTFetcher
		failAt    int
		wantFetch int
		want      VerifyOutcome
	}{
		{"rfc get-sth", rfcSCT, rfcFetch, 1, 0, VerifyUnverifiable},
		{"rfc proof-by-hash", rfcSCT, rfcFetch, 2, 1, VerifyUnverifiable},
		{"rfc unthrottled control", rfcSCT, rfcFetch, 3, 2, VerifyLogged},
		{"tiled checkpoint", tiledSCT, tiledFetch, 1, 0, VerifyUnverifiable},
		{"tiled hash tile", tiledSCT, tiledFetch, 2, 1, VerifyUnverifiable},
		{"tiled unthrottled control", tiledSCT, tiledFetch, 3, 2, VerifyLogged},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &countingFetcher{inner: tc.fetch}
			th := &slotThrottle{failAt: tc.failAt}
			blob := wire.EncodeSCTCapture(wire.SCTCapture{TLSExt: [][]byte{tc.sct}})
			res := verifyWorker(f, th).verifyMaterial(context.Background(), logs, der, blob, nil)
			if res.Outcome != tc.want {
				t.Fatalf("outcome = %v (%s), want %v", res.Outcome, res.Reason, tc.want)
			}
			if f.calls != tc.wantFetch {
				t.Fatalf("fetched %d time(s), want %d — every verify fetch reserves first", f.calls, tc.wantFetch)
			}
		})
	}
}

type materialDBTX struct {
	der  []byte
	scts []byte
}

func (materialDBTX) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (materialDBTX) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func (m materialDBTX) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return materialRow{der: m.der, scts: m.scts}
}

type materialRow struct {
	der  []byte
	scts []byte
}

func (r materialRow) Scan(dest ...interface{}) error {
	if len(dest) != 4 {
		return errors.New("queue: unexpected certificate_material column count")
	}
	fp, ok := dest[0].(*string)
	if !ok {
		return errors.New("queue: unexpected fingerprint destination")
	}
	*fp = "sha256:test"
	for i, v := range [][]byte{r.der, r.scts, nil} {
		p, ok := dest[i+1].(*[]byte)
		if !ok {
			return errors.New("queue: unexpected bytea destination")
		}
		*p = v
	}
	return nil
}

func TestVerifyByFingerprintReservesBeforeFetch(t *testing.T) {
	_, logID := firstLog(t, false)
	der, _ := selfSignedLeaf(t)
	ts := uint64(1700000000000)
	blob := wire.EncodeSCTCapture(wire.SCTCapture{TLSExt: [][]byte{serializeSCT(logID, ts, nil)}})

	f := &countingFetcher{inner: routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(1, scan.LeafHashX509(der, nil, ts))},
		{sub: "get-proof-by-hash", status: 200, body: proofBody(0, nil)},
	}}}
	th := &slotThrottle{failAt: 1}
	w := verifyWorker(f, th)
	w.q = db.New(materialDBTX{der: der, scts: blob})

	res, err := w.VerifyByFingerprint(context.Background(), "sha256:test")
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != VerifyUnverifiable {
		t.Fatalf("outcome = %v (%s), want unverifiable when no slot is granted", res.Outcome, res.Reason)
	}
	// The path has no production caller, so this is what keeps the gap from returning (#1117).
	if f.calls != 0 {
		t.Fatalf("VerifyByFingerprint fetched %d time(s) with no reservation granted, want 0", f.calls)
	}
}

func TestCTTailFreshLogFetchesOnlyTheHead(t *testing.T) {
	rfc := routeFetcher{routes: []route{
		{sub: "get-sth", status: 200, body: sthBody(1_000_000, make([]byte, 32))},
		{sub: "get-entries", status: 200, body: []byte("[]")},
	}}
	tiled := routeFetcher{routes: []route{
		{sub: "checkpoint", status: 200, body: tiledCheckpoint(1_000_000)},
		{sub: "tile/data", status: 200, body: nil},
	}}
	cases := []struct {
		name  string
		lg    scan.CTLog
		fetch CTFetcher
	}{
		{"rfc", scan.CTLog{LogID: "a", URL: "https://rfc.example"}, rfc},
		{"tiled", scan.CTLog{LogID: "b", URL: "https://tiled.example", Tiled: true}, tiled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &countingFetcher{inner: tc.fetch}
			th := &slotThrottle{failAt: 1 << 30}
			_ = tailWorker(f, th).completeCTTail(context.Background(), db.ClaimJobRow{}, tailSpec(t, tc.lg))
			if f.calls != 1 {
				t.Fatalf("a log with no cursor fetched %d time(s), want 1: the signed head alone, never history", f.calls)
			}
		})
	}
}
