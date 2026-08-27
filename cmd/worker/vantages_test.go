package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/remoteexec"
	"github.com/winniel123/verge-asm/internal/vantage"
)

// fakeVantageStore is an in-memory vantageKeyStore for the worker key loop.
type fakeVantageStore struct {
	needing   []db.Vantage
	published map[int64]string
}

func (f *fakeVantageStore) ListVantagesNeedingKey(context.Context) ([]db.Vantage, error) {
	return f.needing, nil
}

func (f *fakeVantageStore) SetVantagePublicKey(_ context.Context, arg db.SetVantagePublicKeyParams) error {
	if f.published == nil {
		f.published = map[int64]string{}
	}
	f.published[arg.ID] = arg.PublicKey.String
	return nil
}

func TestProvisionVantageKeysGeneratesAndPublishes(t *testing.T) {
	dir := t.TempDir()
	store := &fakeVantageStore{needing: []db.Vantage{{ID: 7}, {ID: 9}}}

	provisionVantageKeys(context.Background(), store, dir)

	for _, id := range []int64{7, 9} {
		pub, ok := store.published[id]
		if !ok || pub == "" {
			t.Fatalf("vantage %d: no public key published", id)
		}
		if !strings.HasPrefix(pub, "ssh-ed25519 ") {
			t.Errorf("vantage %d: published key is not an ed25519 line: %q", id, pub)
		}

		// The private half is on the worker volume, and its public half matches
		// what was published.
		keyPath := filepath.Join(dir, "vantages", strconv.FormatInt(id, 10), "id_ed25519")
		data, err := os.ReadFile(keyPath)
		if err != nil {
			t.Fatalf("vantage %d: private key not written: %v", id, err)
		}
		if strings.Contains(pub, "PRIVATE") || strings.Contains(string(data), pub) {
			// A weak but real guard that the two halves are distinct.
			t.Errorf("vantage %d: private and public material overlap", id)
		}
		derived, err := vantage.PublicKeyFromPrivatePEM(data)
		if err != nil {
			t.Fatalf("vantage %d: stored private key does not parse: %v", id, err)
		}
		if derived != pub {
			t.Errorf("vantage %d: published %q does not match stored key %q", id, pub, derived)
		}

		// The private key file is not world-readable (skipped on Windows, whose
		// permission bits do not model POSIX modes).
		if runtime.GOOS != "windows" {
			info, err := os.Stat(keyPath)
			if err != nil {
				t.Fatal(err)
			}
			if perm := info.Mode().Perm(); perm&0o077 != 0 {
				t.Errorf("vantage %d: private key mode %o is group/world accessible", id, perm)
			}
		}
	}
}

func TestProvisionVantageKeysIsIdempotent(t *testing.T) {
	dir := t.TempDir()

	// First pass generates and publishes.
	store := &fakeVantageStore{needing: []db.Vantage{{ID: 3}}}
	provisionVantageKeys(context.Background(), store, dir)
	first := store.published[3]

	keyPath := filepath.Join(dir, "vantages", "3", "id_ed25519")
	before, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}

	// A second pass (e.g. the public write failed and the row still lists as
	// needing a key) reuses the on-disk private key rather than regenerating,
	// so the key installed on the prober host stays valid.
	store2 := &fakeVantageStore{needing: []db.Vantage{{ID: 3}}}
	provisionVantageKeys(context.Background(), store2, dir)

	after, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("private key was regenerated on the second pass")
	}
	if store2.published[3] != first {
		t.Errorf("re-published key %q differs from first %q", store2.published[3], first)
	}
}

// fakeLatencyStore is an in-memory vantageLatencyStore for the worker latency loop.
type fakeLatencyStore struct {
	needing  []db.Vantage
	pinned   map[int64]string
	latency  map[int64]int32
	platform map[int64]pgtype.Text
	egress   map[int64]pgtype.Text
}

func (f *fakeLatencyStore) ListVantagesNeedingLatency(context.Context) ([]db.Vantage, error) {
	return f.needing, nil
}

func (f *fakeLatencyStore) PinVantageHostKey(_ context.Context, arg db.PinVantageHostKeyParams) error {
	if f.pinned == nil {
		f.pinned = map[int64]string{}
	}
	f.pinned[arg.ID] = arg.HostKey.String
	return nil
}

func (f *fakeLatencyStore) SetVantageLatency(_ context.Context, arg db.SetVantageLatencyParams) error {
	if f.latency == nil {
		f.latency = map[int64]int32{}
	}
	f.latency[arg.ID] = arg.LatencyMs.Int32
	return nil
}

func (f *fakeLatencyStore) SetVantageProbeFacts(_ context.Context, arg db.SetVantageProbeFactsParams) error {
	if f.platform == nil {
		f.platform = map[int64]pgtype.Text{}
		f.egress = map[int64]pgtype.Text{}
	}
	f.platform[arg.ID] = arg.Platform
	f.egress[arg.ID] = arg.Egress
	return nil
}

// fakeProber stands in for the SSH dial so the orchestration is exercised without
// a live server: it pins the given host key via onFirstUse and returns a fixed
// round-trip and lifecycle facts, or an error to model an unreachable prober.
type fakeProber struct {
	rtt     time.Duration
	encoded string
	facts   remoteexec.Facts
	err     error
}

func (p fakeProber) Connect(_ context.Context, _ db.Vantage, _ string, onFirstUse func(string) error) (vantageProbe, error) {
	if p.err != nil {
		return vantageProbe{}, p.err
	}
	if onFirstUse != nil {
		if err := onFirstUse(p.encoded); err != nil {
			return vantageProbe{}, err
		}
	}
	return vantageProbe{rtt: p.rtt, facts: p.facts}, nil
}

func provisionedVantage(id int64) db.Vantage {
	return db.Vantage{
		ID:       id,
		Host:     pgtype.Text{String: "prober.example.com", Valid: true},
		Port:     pgtype.Int4{Int32: 22, Valid: true},
		Username: pgtype.Text{String: "verge", Valid: true},
	}
}

// A successful connect pins the presented host key trust-on-first-use, records the
// round-trip in whole milliseconds — the unit the Dashboard renders — and persists the
// off-host lifecycle facts (platform, egress) the same connect observed.
func TestMeasureVantageLatenciesPinsAndRecords(t *testing.T) {
	store := &fakeLatencyStore{needing: []db.Vantage{provisionedVantage(5)}}
	prober := fakeProber{
		rtt: 34 * time.Millisecond, encoded: "ssh-ed25519 AAAAhostkey",
		facts: remoteexec.Facts{
			Platform:  remoteexec.Platform{GOOS: "linux", GOARCH: "amd64", Label: "linux · x86_64"},
			Egress:    "203.0.113.5",
			HasEgress: true,
		},
	}

	measureVantageLatencies(context.Background(), store, prober, t.TempDir())

	if got := store.pinned[5]; got != "ssh-ed25519 AAAAhostkey" {
		t.Errorf("vantage 5: host key pinned = %q, want the presented key", got)
	}
	if got := store.latency[5]; got != 34 {
		t.Errorf("vantage 5: latency recorded = %dms, want 34ms", got)
	}
	if got := store.platform[5]; !got.Valid || got.String != "linux · x86_64" {
		t.Errorf("vantage 5: platform recorded = %+v, want linux · x86_64", got)
	}
	if got := store.egress[5]; !got.Valid || got.String != "203.0.113.5" {
		t.Errorf("vantage 5: egress recorded = %+v, want 203.0.113.5", got)
	}
}

// A connect that could not read the egress (host does not export SSH_CLIENT) persists
// a NULL egress rather than a fabricated one — the chip stays collapsed.
func TestMeasureVantageLatenciesBlankEgressStaysNull(t *testing.T) {
	store := &fakeLatencyStore{needing: []db.Vantage{provisionedVantage(6)}}
	prober := fakeProber{
		rtt: 10 * time.Millisecond, encoded: "ssh-ed25519 AAAAhostkey",
		facts: remoteexec.Facts{
			Platform: remoteexec.Platform{Label: "linux · aarch64"},
			// no egress observed
		},
	}

	measureVantageLatencies(context.Background(), store, prober, t.TempDir())

	if got := store.egress[6]; got.Valid {
		t.Errorf("vantage 6: egress recorded = %q despite no read, want NULL", got.String)
	}
	if got := store.platform[6]; !got.Valid || got.String != "linux · aarch64" {
		t.Errorf("vantage 6: platform recorded = %+v, want linux · aarch64", got)
	}
}

// An unreachable prober records nothing: its latency stays NULL and the Dashboard
// keeps rendering the pending em dash — never a fabricated value.
func TestMeasureVantageLatenciesSkipsUnreachable(t *testing.T) {
	store := &fakeLatencyStore{needing: []db.Vantage{provisionedVantage(8)}}
	prober := fakeProber{err: errors.New("dial: connection refused")}

	measureVantageLatencies(context.Background(), store, prober, t.TempDir())

	if _, ok := store.latency[8]; ok {
		t.Error("vantage 8: latency recorded for an unreachable prober")
	}
	if _, ok := store.pinned[8]; ok {
		t.Error("vantage 8: host key pinned despite the connect failing")
	}
}
