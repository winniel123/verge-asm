package main

import (
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

var (
	nameSeedDeclaredAt    = time.Date(2026, 5, 2, 17, 30, 0, 0, time.UTC)
	addressSeedDeclaredAt = time.Date(2026, 3, 19, 8, 5, 0, 0, time.UTC)
)

func coveredSinceChip(at time.Time) string {
	return "covered since " + at.UTC().Format("2006-01-02")
}

func pinEverySeedDate(f *fakeStore, at time.Time) {
	for i := range f.seeds {
		f.seeds[i].CreatedAt = pgtype.Timestamptz{Time: at, Valid: true}
	}
}

func pinSeedDate(t *testing.T, f *fakeStore, id int64, at time.Time) {
	t.Helper()
	for i := range f.seeds {
		if f.seeds[i].ID == id {
			f.seeds[i].CreatedAt = pgtype.Timestamptz{Time: at, Valid: true}
			return
		}
	}
	t.Fatalf("the fixture holds no seed %d", id)
}

func TestAssetInsideADeclaredScopeRendersTheCoveredSinceChip(t *testing.T) {
	// The absence assertion at #2050 discriminates only once the covered branch is pinned (#2173).
	f, base := assetRegionFixture(t, "api.example.com")
	pinEverySeedDate(f, nameSeedDeclaredAt)

	page := assetRegionPage(t, f, base, "api.example.com")

	wantIn(t, page, coveredSinceChip(nameSeedDeclaredAt), "name inside a declared scope")
	wantNotIn(t, page, seedChipDidNotResolve, "name inside a declared scope")
}

func TestSubjectDetailInsideADeclaredScopeRendersTheCoveredSinceChip(t *testing.T) {
	for _, c := range subjectSpanReadCases() {
		t.Run(c.name, func(t *testing.T) {
			f := c.store(t)
			admin, err := f.GetAccountByUsername(t.Context(), "admin")
			if err != nil {
				t.Fatal(err)
			}
			scope := netip.MustParsePrefix("198.51.100.0/24")
			seed, err := f.CreateAddressSeed(t.Context(), db.CreateAddressSeedParams{
				AddressCidr: &scope, CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			// The name seed holds the other date, so a chip from that read fails (#2173).
			pinEverySeedDate(f, nameSeedDeclaredAt)
			pinSeedDate(t, f, seed.ID, addressSeedDeclaredAt)

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+c.path, http.StatusOK)

			wantIn(t, page, coveredSinceChip(addressSeedDeclaredAt), "address inside a declared scope")
			wantNotIn(t, page, coveredSinceChip(nameSeedDeclaredAt), "address inside a declared scope")
			wantNotIn(t, page, seedChipDidNotResolve, "address inside a declared scope")
		})
	}
}
