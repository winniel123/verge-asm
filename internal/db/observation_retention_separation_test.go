package db

import (
	"strings"
	"testing"
)

// These guard the structural half of the Observation retention ACs (v1 spec §4.6,
// ADR-0041, ADR-0094) at the SQL layer, the way retention_separation_test.go
// guards the Dispatch delete. The row-level RETENTION rule itself is proved in
// internal/retention/observation_test.go; here we hold the two queries to the
// shape those proofs assume — a per-timeline bound that is never collapsed, an
// undefined bound that is never retired, a withdrawn subject governed by the dial
// alone, and a delete that touches only the Observation corpus.

// TestDeleteExpiredObservationsEvaluatesEachRowsOwnBound holds the deletion query
// to its defining property: it reads EACH ROW'S OWN per-timeline bound rather than
// one collapsed number (ADR-0094). The bound is k cadences of the tightest covering
// Scan, resolved by grouping observations on their full timeline key and joining
// through batch to scan — so the query must name that join and that grouping.
func TestDeleteExpiredObservationsEvaluatesEachRowsOwnBound(t *testing.T) {
	sql := strings.ToLower(deleteExpiredObservations)

	if !strings.Contains(sql, "delete from observation") {
		t.Fatalf("must delete from observation, got:\n%s", deleteExpiredObservations)
	}
	// The per-timeline bound: grouped on the full timeline key, tightest covering
	// cadence via batch -> scan. If any of these is missing the bound has been
	// collapsed or hard-coded.
	for _, need := range []string{
		"min(s.cadence_seconds)", // tightest covering cadence
		"join batch",             // observation -> batch
		"join scan",              // batch -> scan (the cadence)
		"s.enabled = true",       // only ENABLED Scans cover
		"group by o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source", // the full timeline key
		"* c.tightest_cadence", // the per-row bound: k (a parameter) times this row's own cadence
	} {
		if !strings.Contains(sql, need) {
			t.Errorf("deletion query must contain %q — its per-timeline bound depends on it:\n%s", need, deleteExpiredObservations)
		}
	}
}

// TestDeleteExpiredObservationsHonoursTheTwoExceptions holds the query to the two
// populations that fall outside the ordinary rule: an undefined-bound timeline
// (the cover LEFT JOIN misses) is never retired, and a withdrawn subject (every
// span closed) is governed by the dial alone.
func TestDeleteExpiredObservationsHonoursTheTwoExceptions(t *testing.T) {
	sql := strings.ToLower(deleteExpiredObservations)

	// Undefined bound: a LEFT JOIN onto cover so an uncovered timeline yields NULL
	// and is excluded from the delete rather than aged out on some default.
	if !strings.Contains(sql, "left join cover") {
		t.Errorf("must LEFT JOIN cover so an undefined bound is never retired, got:\n%s", deleteExpiredObservations)
	}
	if !strings.Contains(sql, "c.tightest_cadence is not null") {
		t.Errorf("the defined-bound branch must gate on a non-null cover row (undefined => never retired):\n%s", deleteExpiredObservations)
	}
	// Withdrawn subject: composed from the span table (all spans closed), dial alone.
	if !strings.Contains(sql, "withdrawn") || !strings.Contains(sql, "bool_and(closed_at is not null)") {
		t.Errorf("must compose a withdrawn subject (all spans closed) so the dial alone governs it:\n%s", deleteExpiredObservations)
	}
}

// TestObservationRetentionTouchesOnlyObservation proves the delete moves no value
// on any timeline: it may READ batch, scan and span to resolve a bound and a
// membership, but it deletes from observation and touches no other corpus — no
// dispatch, queue_job or channel row, and it writes nothing to batch, scan or span.
func TestObservationRetentionTouchesOnlyObservation(t *testing.T) {
	sql := strings.ToLower(deleteExpiredObservations)

	// The only DELETE/UPDATE/INSERT verb in the statement is the observation delete.
	if strings.Count(sql, "delete from") != 1 || !strings.Contains(sql, "delete from observation") {
		t.Errorf("exactly one delete, and it must be from observation:\n%s", deleteExpiredObservations)
	}
	for _, verb := range []string{"update ", "insert into"} {
		if strings.Contains(sql, verb) {
			t.Errorf("deletion query must not %q — it may read supporting tables but write only the observation delete:\n%s", verb, deleteExpiredObservations)
		}
	}
	// Corpora that have nothing to do with observation currency must not appear.
	for _, forbidden := range []string{"dispatch", "queue_job", "channel"} {
		if strings.Contains(sql, forbidden) {
			t.Errorf("deletion query must never name %q:\n%s", forbidden, deleteExpiredObservations)
		}
	}
}

// TestLiveDerivationReadFiltersByOwnBound holds the read gate to the other half of
// the separation: a derivation reads only live-tier rows, so the query INNER JOINs
// cover (dropping uncovered timelines) and keeps only rows within k cadences of
// their own bound — an evidential row is unreadable through it.
func TestLiveDerivationReadFiltersByOwnBound(t *testing.T) {
	sql := strings.ToLower(listLiveObservationsForDerivation)

	if strings.Contains(sql, "left join cover") {
		t.Errorf("the live read must INNER JOIN cover so an uncovered (undefined-bound) timeline yields no live row:\n%s", listLiveObservationsForDerivation)
	}
	if !strings.Contains(sql, "join cover") {
		t.Fatalf("the live read must join cover to know each row's bound:\n%s", listLiveObservationsForDerivation)
	}
	for _, need := range []string{"* c.tightest_cadence", "<="} {
		if !strings.Contains(sql, need) {
			t.Errorf("the live read must bound each row by k cadences of its own timeline (%q missing):\n%s", need, listLiveObservationsForDerivation)
		}
	}
	// A read, never a write.
	for _, verb := range []string{"delete", "update ", "insert into"} {
		if strings.Contains(sql, verb) {
			t.Errorf("the derivation read must not %q:\n%s", verb, listLiveObservationsForDerivation)
		}
	}
}
