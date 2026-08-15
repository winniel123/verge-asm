package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
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
