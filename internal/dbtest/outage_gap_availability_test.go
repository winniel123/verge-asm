package dbtest_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/dbtest"
)

func insertVantageWithAvailability(t *testing.T, tx pgx.Tx, name string, availability any) int64 {
	t.Helper()
	var id int64
	err := tx.QueryRow(context.Background(),
		`INSERT INTO vantage (name, resolver, host, port, username, availability)
		 VALUES ($1, '9.9.9.9', $1 || '.example', 22, 'verge', $2)
		 RETURNING id`, name, availability).Scan(&id)
	if err != nil {
		t.Fatalf("insert vantage %s: %v", name, err)
	}
	return id
}

func TestListOutageReachGapVantagesProjectsEveryAvailabilityState(t *testing.T) {
	q, tx := dbtest.Queries(t)
	ctx := context.Background()

	// The column admits four values, and the CHECK is the only writer-independent
	// statement of that (#2254).
	states := map[string]any{
		"outage-available":   "available",
		"outage-unavailable": "unavailable",
		"outage-pending":     "pending",
		"outage-null":        nil,
	}
	want := map[string]string{
		"outage-available":   "available",
		"outage-unavailable": "unavailable",
		"outage-pending":     "pending",
		"outage-null":        "unknown",
	}
	for name, availability := range states {
		id := insertVantageWithAvailability(t, tx, name, availability)
		insertSpan(t, tx, id, spanRow{
			subjectKind: "service",
			subjectKey:  "198.51.100.9:443/tcp",
			facet:       "reachability",
			source:      "prober",
			value:       `{"outcome":"gap","cause":"vantage-unavailable"}`,
			isGap:       true,
		})
	}

	rows, err := q.ListOutageReachGapVantages(ctx)
	if err != nil {
		t.Fatalf("ListOutageReachGapVantages: %v", err)
	}

	got := map[string]string{}
	for _, r := range rows {
		if _, ours := want[r.Vantage]; !ours {
			continue
		}
		got[r.Vantage] = r.Availability
		if r.Services != 1 {
			t.Errorf("%s counted %d services, want 1", r.Vantage, r.Services)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("read %d of the four positions, so a state was filtered out (#2180, #2254): %+v", len(got), got)
	}
	for name, state := range want {
		if got[name] != state {
			t.Errorf("%s projected %q, want %q (#2254)", name, got[name], state)
		}
	}
}
