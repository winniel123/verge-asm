package main

import (
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func namedQuerySQL(t *testing.T, file, name string) string {
	t.Helper()
	raw, err := os.ReadFile("../../db/queries/" + file)
	if err != nil {
		t.Fatalf("read db/queries/%s: %v", file, err)
	}
	_, rest, ok := strings.Cut(string(raw), "-- name: "+name+" ")
	if !ok {
		t.Fatalf("db/queries/%s names no query %s", file, name)
	}
	if end := strings.Index(rest, "\n-- name: "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

func uncommentedQuerySQL(sql string) string {
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		if j := strings.Index(line, "--"); j >= 0 {
			lines[i] = line[:j]
		}
	}
	return strings.Join(lines, "\n")
}

func TestTheOutageReadCarriesTheCauseTheServiceLedgerDrops(t *testing.T) {
	// reachGapsAndMessages drops this cause because ListOutageReachGapVantages renders it per
	// vantage. A rename on either side alone restores the silent zero (#2147, #2180).
	q := uncommentedQuerySQL(namedQuerySQL(t, "signals.sql", "ListOutageReachGapVantages"))

	if !strings.Contains(q, `'cause' = '`+vantageUnavailableCause+`'`) {
		t.Fatalf("the read that carries %q must filter on it, or the Coverage carve-out drops "+
			"a Gap nothing renders (#2147); got:\n%s", vantageUnavailableCause, q)
	}
}

func TestTheOutageReadKeepsAvailabilityOutOfItsPredicate(t *testing.T) {
	// A vantage recovers long before a connect batch overwrites the reachability Gap it opened,
	// so an availability predicate here hides an open Gap (#2147, #2189).
	q := uncommentedQuerySQL(namedQuerySQL(t, "signals.sql", "ListOutageReachGapVantages"))

	// An inner join's ON clause filters exactly as WHERE does, so the scan starts at FROM
	// and leaves only the SELECT list, where the recovered projection legitimately reads it.
	i := strings.Index(q, "FROM")
	if i < 0 {
		t.Fatalf("the outage read must still name the table it rolls up; got:\n%s", q)
	}
	if strings.Contains(q[i:], "availability") {
		t.Errorf("availability is a projection and never a predicate: filtering a recovered "+
			"position out leaves its open Gap rendered nowhere (#2147, #2189); got:\n%s", q)
	}
}

func TestCoverageRendersAReachGapWhoseVantageHasRecovered(t *testing.T) {
	const svc = "198.51.100.9:443/tcp"
	f := newFakeStore()
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "alpha", Class: "internet", Resolver: "127.0.0.11:53",
		// Recovery retires the resolution facets alone, so this reach Gap outlives the outage.
		Availability: pgtype.Text{String: "available", Valid: true},
	})
	f.vantageNextID = 2
	f.addClassReachability(t, svc, "internet", obsClock, unavailableGap)

	// The premise under test: the per-service arm renders this cause nowhere at all.
	dropped, msgs, _ := reachGapsAndMessages([]db.ListOpenReachGapServicesRow{
		{SubjectKey: svc, Value: []byte(unavailableGap)},
	})
	if len(dropped) != 0 || len(msgs) != 0 {
		t.Fatalf("the carve-out must still drop this cause, or this test proves nothing "+
			"(#2147); gaps %+v messages %+v", dropped, msgs)
	}

	srv := newServer(f, testKey, "", fixedClock())
	ledger := srv.readCoverageGapLedger(t.Context())

	if len(ledger.Gaps) == 0 {
		t.Fatalf("a Gap the carve-out dropped and no other arm renders is a silent zero "+
			"(#2147); ledger: %+v", ledger)
	}
	if ledger.Gaps[0].Subject != "vantage alpha" {
		t.Errorf("the surviving row must name the position whose outage opened the Gap "+
			"(#2180); got: %+v", ledger.Gaps)
	}
	if ledger.GapsFailed || ledger.MessagesFailed {
		t.Errorf("every ledger read resolved, so neither card may read as degraded: %+v", ledger)
	}
}
