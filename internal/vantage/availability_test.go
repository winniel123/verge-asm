package vantage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/db/migrations"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// ADR-2087 rules Alternative B, write time. The rule is one SQL statement and no Go
// production code, so these read its text. internal/dbtest runs that statement against
// a real PostgreSQL, which is where its execution is proved (#2166).

const gapCause = `"cause":"vantage-unavailable"`

func queriesDir() string { return filepath.Join("..", "..", "db", "queries") }

// A predicate is what binds; prose naming a facet is not one.

func uncommented(sql string) string {
	var out []string
	for _, line := range strings.Split(sql, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func namedQuery(t *testing.T, file, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(queriesDir(), file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	marker := "-- name: " + name + " "
	i := strings.Index(string(body), marker)
	if i < 0 {
		t.Fatalf("%s declares no query named %s", file, name)
	}
	rest := string(body)[i:]
	end := statementEnd(rest)
	if end < 0 {
		t.Fatalf("%s.%s reaches the end of the file with no terminating semicolon outside a "+
			"comment or a literal", file, name)
	}
	return rest[:end]
}

func statementEnd(sql string) int {
	for i := 0; i < len(sql); {
		switch {
		case strings.HasPrefix(sql[i:], "--"):
			j := strings.IndexByte(sql[i:], '\n')
			if j < 0 {
				return -1
			}
			i += j + 1
		case strings.HasPrefix(sql[i:], "/*"):
			// Postgres nests a block comment, so the first */ need not end it.
			depth := 1
			i += 2
			for depth > 0 {
				switch {
				case i >= len(sql):
					return -1
				case strings.HasPrefix(sql[i:], "/*"):
					depth++
					i += 2
				case strings.HasPrefix(sql[i:], "*/"):
					depth--
					i += 2
				default:
					i++
				}
			}
		case sql[i] == '\'' || sql[i] == '"':
			// A doubled quote reads as a close and a reopen, so the run stays quoted.
			j := strings.IndexByte(sql[i+1:], sql[i])
			if j < 0 {
				return -1
			}
			i += j + 2
		case sql[i] == ';':
			return i + 1
		default:
			i++
		}
	}
	return -1
}

func TestStatementEndReadsNoSemicolonInsideACommentOrALiteral(t *testing.T) {
	// A cut at the first semicolon returned a fragment, and every negative assertion over the
	// tail passed because the text it forbids sat past the cut (#2187).
	for _, tc := range []struct {
		name string
		sql  string
		want string
	}{
		{"bare statement", "SELECT 1;\n-- name: Next :one\nSELECT 2;\n", "SELECT 1;"},
		{"line comment", "-- retires the Gap; the caller names the facets\nSELECT 1;\nSELECT 2;",
			"-- retires the Gap; the caller names the facets\nSELECT 1;"},
		{"trailing comment", "SELECT 1\n  AND a = 'b' -- one; two\n  AND c = 'd';\nSELECT 2;",
			"SELECT 1\n  AND a = 'b' -- one; two\n  AND c = 'd';"},
		{"block comment", "/* one;\ntwo */\nSELECT 1;\nSELECT 2;", "/* one;\ntwo */\nSELECT 1;"},
		{"nested block comment", "/* one /* two; */ three; */\nSELECT 1;", "/* one /* two; */ three; */\nSELECT 1;"},
		{"literal", "SELECT ';';\nSELECT 2;", "SELECT ';';"},
		{"doubled quote in a literal", "SELECT 'it''s; here';\nSELECT 2;", "SELECT 'it''s; here';"},
		{"quoted identifier", "SELECT \"a;b\";\nSELECT 2;", "SELECT \"a;b\";"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			end := statementEnd(tc.sql)
			if end < 0 {
				t.Fatalf("statementEnd found no end in:\n%s", tc.sql)
			}
			if got := tc.sql[:end]; got != tc.want {
				t.Errorf("statementEnd cut\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
	for _, unterminated := range []string{"SELECT 1\n", "-- one; two\nSELECT 1\n", "SELECT ';\n", "/* one;\n"} {
		if end := statementEnd(unterminated); end >= 0 {
			t.Errorf("an unterminated statement ends nowhere, so the helper must say so; got %d for %q",
				end, unterminated)
		}
	}
}

func TestMarkVantageUnavailableClosesTheVantagesOpenSpans(t *testing.T) {
	q := namedQuery(t, "vantages.sql", "MarkVantageUnavailable")

	if !strings.Contains(q, "UPDATE span") || !strings.Contains(q, "SET closed_at") {
		t.Fatalf("MarkVantageUnavailable must close the vantage's open spans, not only move the "+
			"column: one dead prober's open span otherwise pins a class to reached forever "+
			"(ADR-2087, #2060); got:\n%s", q)
	}
	if !strings.Contains(q, "span.closed_at IS NULL") {
		t.Errorf("the closure must reach the OPEN spans alone; got:\n%s", q)
	}
	if !strings.Contains(q, "span.vantage_id IN") {
		t.Errorf("the closure must be scoped to this vantage's own spans; got:\n%s", q)
	}
}

func TestMarkVantageUnavailableClosesEveryFacetIncludingDNSRecord(t *testing.T) {
	// A facet enumeration left the same dead vantage's dns-record spans open, which is the
	// defect the rule exists to kill one facet over (#2144). ADR-2087 now reads wide, so the
	// closure names no facet and one added later is covered on the day it is added.
	q := namedQuery(t, "vantages.sql", "MarkVantageUnavailable")
	closure := uncommented(q)
	if i := strings.Index(closure, "INSERT INTO span"); i >= 0 {
		closure = closure[:i]
	}
	for _, predicate := range []string{"facet IN", "facet ="} {
		if strings.Contains(closure, predicate) {
			t.Errorf("the closure restricts which facets it reaches, so a dns-record span stays "+
				"open behind a dead vantage (ADR-2087, #2144); got:\n%s", closure)
		}
	}
	// The Gap value still spells the outcome per facet, so the default must cover the rest.
	value := q[strings.Index(q, "CASE facet"):]
	if !strings.Contains(value, "ELSE") {
		t.Errorf("without a default branch a dns-record span would open a Gap holding NULL; got:\n%s", value)
	}
}

func TestMarkVantageUnavailableIsIdempotentOnTheSpansAndNotTheColumn(t *testing.T) {
	// Availability is cleared only by a COMPLETED resolution-walk batch, so a vantage can sit
	// at `unavailable` while its connect batches keep opening fresh `reached` spans. A guard on
	// the column would refuse the second write and leave those spans voting forever (#2060).
	q := namedQuery(t, "vantages.sql", "MarkVantageUnavailable")

	if strings.Contains(q, "availability IS DISTINCT FROM 'unavailable'") {
		t.Errorf("a column guard closes nothing on the second mark, and the spans opened since "+
			"keep voting under the existential fold (#2060); got:\n%s", q)
	}
	if !strings.Contains(q, `'cause' = 'vantage-unavailable') IS NOT TRUE`) {
		t.Errorf("a re-mark must close nothing it already closed, so the guard belongs on the "+
			"spans; IS NOT TRUE is what keeps a NULL cause in scope (ADR-2087); got:\n%s", q)
	}
}

func TestMarkVantageUnavailableOpensAGapCarryingItsCause(t *testing.T) {
	q := namedQuery(t, "vantages.sql", "MarkVantageUnavailable")

	if !strings.Contains(q, "INSERT INTO span") {
		t.Fatalf("closing the spans without opening a Gap reads as a withdrawal rather than as "+
			"a position that stopped answering (ADR-0017 decision 4); got:\n%s", q)
	}
	if !strings.Contains(q, gapCause) {
		t.Errorf("the opened span must carry cause vantage-unavailable, which cmd/web/cold.go "+
			"reads to keep the Coverage ledger from double-counting it (#2090); got:\n%s", q)
	}
	// Each facet decodes its own gap spelling, so one literal reads as an unknown outcome.
	for _, outcome := range []string{`"outcome":"Gap"`, `"outcome":"gap"`} {
		if !strings.Contains(q, outcome) {
			t.Errorf("the Gap value must spell %s for the facet that decodes it; got:\n%s", outcome, q)
		}
	}
}

func TestMarkVantageUnavailableNeverReadsANullAvailabilityAsUnavailable(t *testing.T) {
	q := namedQuery(t, "vantages.sql", "MarkVantageUnavailable")

	// A resolver-only vantage such as the shipped local holds NULL, and NULL is not unavailable.
	for _, wrong := range []string{"availability IS NOT NULL", "availability = 'available'"} {
		if strings.Contains(q, wrong) {
			t.Errorf("%q reads a NULL availability as already-unavailable, so the shipped "+
				"resolver-only vantage would never close its spans (ADR-2087); got:\n%s", wrong, q)
		}
	}
}

func TestNoCompositionReadGainedAnAvailabilityPredicate(t *testing.T) {
	// ADR-2087 refuses Alternative A: filtering the dispatch read would strand an
	// unavailable vantage as unavailable forever, because applyAvailability restores it
	// only from a completed resolution-walk batch.
	reads := map[string]string{
		"measurement.sql": "ListVantagesForDispatch",
		"signals.sql":     "ListNameResolutionsByClass",
	}
	for file, name := range reads {
		if q := namedQuery(t, file, name); strings.Contains(q, "availability") {
			t.Errorf("%s must keep every row and gain no availability predicate (ADR-2087); got:\n%s", name, q)
		}
	}
	// The four reads #2060 names, across the two files that declare them.
	for file, names := range map[string][]string{
		"signals.sql": {"ListServiceReachabilitySpansByClass", "ListServiceReachabilitySpansByClassForServices"},
		"span.sql":    {"ListServiceReachabilitySpansByClassAt", "ListServiceReachabilitySpansByClassAtForServices"},
	} {
		for _, name := range names {
			if q := namedQuery(t, file, name); strings.Contains(q, "availability") {
				t.Errorf("%s composes at read time and must stay unfiltered (ADR-2087); got:\n%s", name, q)
			}
		}
	}
}

func TestAMigrationBackfillsTheAlreadyUnavailableVantages(t *testing.T) {
	// The writer acts on the transition, so a vantage that went unavailable before it
	// landed would hold its open spans forever.
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	found, backfill := "", ""
	for _, e := range entries {
		body, err := migrations.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		up := string(body)
		if i := strings.Index(up, "-- +goose Down"); i >= 0 {
			up = up[:i]
		}
		if strings.Contains(up, "availability = 'unavailable'") &&
			strings.Contains(up, "SET closed_at") && strings.Contains(up, gapCause) {
			found, backfill = e.Name(), up
		}
	}
	if found == "" {
		t.Fatal("no migration closes the open spans of a vantage that was already unavailable, " +
			"so ADR-2087 holds for new marks alone and the standing corpus keeps voting (#2137)")
	}
	closure := uncommented(backfill)
	if i := strings.Index(closure, "INSERT INTO span"); i >= 0 {
		closure = closure[:i]
	}
	for _, predicate := range []string{"facet IN", "facet ="} {
		if strings.Contains(closure, predicate) {
			t.Errorf("%s backfills a facet enumeration, so the dns-record spans behind an "+
				"already dead vantage stay open (ADR-2087, #2144)", found)
		}
	}
}

func TestMarkVantageAvailableClosesTheGapsTheOutageOpened(t *testing.T) {
	// A Gap whose vantage recovered is invisible to both Coverage reads, because the ledger
	// skips this cause and the unavailable-vantage read lists a healthy position nowhere. So
	// a recovery that moved the column alone would leave that Gap open forever.
	q := namedQuery(t, "vantages.sql", "MarkVantageAvailable")

	if !strings.Contains(q, "UPDATE span") || !strings.Contains(q, "SET closed_at") {
		t.Fatalf("MarkVantageAvailable must close the Gaps MarkVantageUnavailable opened, not "+
			"only move the column back (ADR-2087); got:\n%s", q)
	}
	if !strings.Contains(q, "span.closed_at IS NULL") {
		t.Errorf("the closure must reach the OPEN Gaps alone; got:\n%s", q)
	}
	if !strings.Contains(q, "span.vantage_id IN") {
		t.Errorf("the closure must be scoped to this vantage's own spans; got:\n%s", q)
	}
}

func TestMarkVantageAvailableClosesTheOutagesGapAndNothingElse(t *testing.T) {
	// A connect batch keeps opening `reached` spans while the resolver is dark, so an
	// unguarded closure would end readings this recovery says nothing about (#2060).
	q := namedQuery(t, "vantages.sql", "MarkVantageAvailable")

	if !strings.Contains(q, "span.is_gap") {
		t.Errorf("a closure reaching a valued span ends a reading recovery never read; got:\n%s", q)
	}
	if !strings.Contains(q, `'cause' = 'vantage-unavailable'`) {
		t.Errorf("recovery may close only the cause the outage wrote: a blanket-responder Gap "+
			"outlives the outage and must stay open (ADR-2087); got:\n%s", q)
	}
}

func TestMarkVantageAvailableRetiresTheRecoveringBatchsFacetsAlone(t *testing.T) {
	// The unavailable writer closes every facet, because a dark position measures none of them.
	// Recovery is reached only from a completed resolution-walk batch, which re-reads two. An
	// unscoped closure retires the reachability Gap with nothing to replace it, the span leaves
	// ListServiceReachabilitySpansByClass, and the board reads never-configured (#2060).
	q := uncommented(namedQuery(t, "vantages.sql", "MarkVantageAvailable"))

	if !strings.Contains(q, "span.facet = ANY") {
		t.Fatalf("recovery must close the Gap on the facets the caller names, not on every facet "+
			"the outage opened (ADR-2087); got:\n%s", q)
	}
	for _, facet := range []string{"resolution", "dns-record", "reachability"} {
		if strings.Contains(q, "'"+facet+"'") {
			t.Errorf("the facet set is the recovering batch's own, so it comes from the caller "+
				"and %s is named nowhere in this query; got:\n%s", facet, q)
		}
	}
}

func TestMarkVantageAvailableOpensNothingBehindTheClosedGap(t *testing.T) {
	// Closing alone leaves the timeline silent, which is what holds: the position can look and
	// has not looked yet. A replacement Gap would write two rows per timeline on every flap,
	// and ADR-0041 bars ever deleting them.
	q := namedQuery(t, "vantages.sql", "MarkVantageAvailable")

	if strings.Contains(q, "INSERT INTO span") {
		t.Errorf("recovery states no reading, so it opens no span; got:\n%s", q)
	}
	if strings.Contains(strings.ToUpper(uncommented(q)), "DELETE") {
		t.Errorf("the Span corpus is never compacted or deleted (ADR-0041); got:\n%s", q)
	}
}

func TestTheTwoAvailabilityWritersAgreeOnTheCauseTheyMove(t *testing.T) {
	// The cause string is the only join between them, so a rename on one side would strand
	// every Gap the other wrote.
	down := namedQuery(t, "vantages.sql", "MarkVantageUnavailable")
	up := namedQuery(t, "vantages.sql", "MarkVantageAvailable")

	if !strings.Contains(down, gapCause) {
		t.Fatalf("MarkVantageUnavailable no longer writes %s; got:\n%s", gapCause, down)
	}
	if !strings.Contains(up, `'vantage-unavailable'`) {
		t.Errorf("MarkVantageAvailable must close the cause MarkVantageUnavailable writes; got:\n%s", up)
	}
}

func TestMarkVantageAvailableNeverReadsANullAvailabilityAsUnavailable(t *testing.T) {
	q := namedQuery(t, "vantages.sql", "MarkVantageAvailable")

	// The shipped resolver-only vantage holds NULL, and NULL is neither state.
	for _, wrong := range []string{"availability = 'unavailable'", "availability IS NOT NULL"} {
		if strings.Contains(q, wrong) {
			t.Errorf("%q gates recovery on a column a resolver-only vantage never sets "+
				"(ADR-2087); got:\n%s", wrong, q)
		}
	}
}

// Each facet's own writer owns the spelling, so the CASE cannot drift off production (#2183).

func TestTheGapSpellingMatchesEachFacetsOwnWriter(t *testing.T) {
	q := uncommented(namedQuery(t, "vantages.sql", "MarkVantageUnavailable"))

	lower := `"outcome":"` + connectoutcome.GapOutcome + `"`
	upper := `"outcome":"` + string(resolutionwalk.OutcomeGap) + `"`
	if lower == upper {
		t.Fatalf("reachability and resolution no longer disagree on the spelling, so this CASE has no job")
	}
	if wildcarddiscrim.OutcomeGap != string(resolutionwalk.OutcomeGap) {
		t.Fatalf("dns-record and resolution no longer share a spelling, so one default branch cannot "+
			"serve both; got %q and %q", wildcarddiscrim.OutcomeGap, resolutionwalk.OutcomeGap)
	}

	var when, els string
	for _, line := range strings.Split(q, "\n") {
		if strings.Contains(line, "WHEN 'reachability'") {
			when = line
		}
		if strings.Contains(line, "ELSE") {
			els = line
		}
	}
	if when == "" {
		t.Fatalf("reachability is the one facet spelling it %s, so it carries the guarded branch; got:\n%s", lower, q)
	}
	if !strings.Contains(when, lower) {
		t.Errorf("the reachability branch must spell the outcome %s, as connectoutcome.GapOutcome does; got:\n%s", lower, when)
	}
	if !strings.Contains(els, upper) {
		t.Errorf("every other facet spells it %s, as resolutionwalk and wildcarddiscrim both do, so the "+
			"default branch carries it; got:\n%s", upper, els)
	}
	for _, facet := range []string{"resolution", "dns-record"} {
		if strings.Contains(q, "WHEN '"+facet+"'") {
			t.Errorf("%s spells the outcome %s and takes the default branch, never one of its own; got:\n%s", facet, upper, q)
		}
	}
}

// A host-key pin is the third writer of the availability column, and it reads no facet (#2182).

func collapsed(sql string) string { return strings.Join(strings.Fields(sql), " ") }

func TestPinVantageHostKeyClearsNoOutageItCannotCloseTheGapsOf(t *testing.T) {
	// `host_key IS NULL` does not say the row holds no span: the shipped resolver-only `local`
	// vantage holds spans under a NULL host key. What keeps a stranded Gap out of reach today is
	// two other files — ListVantagesNeedingLatency filters `host IS NOT NULL`, and the router
	// refuses a host vantage whose key is unpinned, so that vantage completes no batch. The
	// guard belongs here, where a reader of this statement can see it.
	q := collapsed(uncommented(namedQuery(t, "vantages.sql", "PinVantageHostKey")))

	if !strings.Contains(q, "availability") {
		t.Fatalf("PinVantageHostKey writes availability nowhere, so this case has no job; got:\n%s", q)
	}
	if !strings.Contains(q, "WHEN availability = 'unavailable' THEN availability") {
		t.Errorf("the pin must leave an outage standing: it retires no Gap, and a Gap left open "+
			"under a healthy vantage is listed by no read and closed by nothing (ADR-2087); got:\n%s", q)
	}
	if !strings.Contains(q, "ELSE 'available'") {
		t.Errorf("a first connect still declares the position reachable, so every other prior "+
			"state must reach 'available'; got:\n%s", q)
	}
	if !strings.Contains(q, "host_key = ") {
		t.Errorf("the pin must still record the key the connect already trusted, or the next "+
			"connect trusts that host afresh; got:\n%s", q)
	}
}

func TestPinVantageHostKeyRetiresNoGapOfItsOwn(t *testing.T) {
	// Recovery retires the Gap on the facets the recovering batch re-read, and a connect re-reads
	// none. A pin that closed spans would end readings nothing replaces (ADR-2087, #2060).
	q := uncommented(namedQuery(t, "vantages.sql", "PinVantageHostKey"))

	for _, writing := range []string{"UPDATE span", "INSERT INTO span", "closed_at"} {
		if strings.Contains(q, writing) {
			t.Errorf("a host-key pin states no reading, so it touches no span; got:\n%s", q)
		}
	}
}
