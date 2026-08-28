package migrations

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestSeedForeignKeysCascadeOnDelete is the R4-R2 (#752) regression guard.
//
// Deleting a Seed is a hard DELETE FROM seed (deleteSeed, the Scope chip-remove
// act #21a). Every table that references seed(id) with the default NO ACTION
// therefore BLOCKS the delete the moment a dependent row exists — Postgres
// raises a foreign_key_violation and the handler turns it into a 500. The bug
// fired on the realistic shapes: a name scope with an uploaded zone file
// (zone_file), a name scope CT had admitted names under (admitted_name), and an
// address scope confirmed from a Proposal (proposal); cold_scan_scope already
// cascaded.
//
// This asserts the EFFECTIVE schema (the last FK definition per table wins, so a
// later ALTER that adds ON DELETE CASCADE counts): every table that references
// seed(id) must cascade on delete. A future migration that introduces a
// seed-referencing FK without ON DELETE CASCADE — reintroducing the 500 — fails
// here, no live database required.
func TestSeedForeignKeysCascadeOnDelete(t *testing.T) {
	up := upMigrations(t)

	tableRe := regexp.MustCompile(`(?is)\b(?:create\s+table|alter\s+table)\s+([a-z_][a-z0-9_]*)`)

	// cascades[table] = whether the current FK from <table> to seed cascades on
	// delete. Statements are visited in migration order, so a later ALTER
	// overrides the inline definition it replaces.
	cascades := map[string]bool{}
	for _, stmt := range strings.Split(up, ";") {
		low := strings.ToLower(stmt)
		idx := strings.Index(low, "references seed")
		if idx < 0 {
			continue
		}
		m := tableRe.FindStringSubmatch(low)
		if m == nil {
			t.Fatalf("could not identify the table for a REFERENCES seed statement:\n%s", strings.TrimSpace(stmt))
		}
		table := m[1]
		// Look at the FK clause only: from the reference to the end of its clause
		// — a comma inside a CREATE TABLE column list, or the statement end for an
		// ALTER ... ADD CONSTRAINT. ON DELETE CASCADE always trails REFERENCES.
		clause := low[idx:]
		if c := strings.IndexByte(clause, ','); c >= 0 {
			clause = clause[:c]
		}
		cascades[table] = strings.Contains(clause, "on delete cascade")
	}

	if len(cascades) == 0 {
		t.Fatal("no FK to seed(id) found in the migrations — the parser matched nothing, which is itself a regression")
	}

	// The known seed-referencing tables, so a table rename or a parser slip that
	// makes one silently vanish is caught rather than passing vacuously.
	for _, want := range []string{"cold_scan_scope", "zone_file", "admitted_name", "proposal"} {
		if _, ok := cascades[want]; !ok {
			t.Errorf("expected a FK to seed(id) from %q but the migrations declare none", want)
		}
	}

	tables := make([]string, 0, len(cascades))
	for tbl := range cascades {
		tables = append(tables, tbl)
	}
	sort.Strings(tables)
	for _, tbl := range tables {
		if !cascades[tbl] {
			t.Errorf("FK from %q to seed(id) must be ON DELETE CASCADE; without it, deleting a Seed that has a dependent %[1]s row returns a 500 (R4-R2 #752)", tbl)
		}
	}
}

// upMigrations returns every migration file's +goose Up body concatenated in
// filename order, with SQL comments stripped so prose never matches. The Down
// bodies are dropped so a Down's non-cascading re-add is not mistaken for the
// effective schema.
func upMigrations(t *testing.T) string {
	t.Helper()
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		data, err := FS.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		body := string(data)
		if i := strings.Index(body, "-- +goose Down"); i >= 0 {
			body = body[:i]
		}
		for _, line := range strings.Split(body, "\n") {
			if i := strings.Index(line, "--"); i >= 0 {
				line = line[:i]
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
