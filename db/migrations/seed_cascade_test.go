package migrations

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestSeedForeignKeysCascadeOnDelete(t *testing.T) {
	// R4-R2 (#752): a seed-referencing FK at the default NO ACTION turns a Seed delete into a 500.
	up := upMigrations(t)

	tableRe := regexp.MustCompile(`(?is)\b(?:create\s+table|alter\s+table)\s+([a-z_][a-z0-9_]*)`)

	// Statements arrive in migration order, so a later ALTER overrides the definition it replaces.
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
		// ON DELETE CASCADE always trails REFERENCES, so the clause ends at the first comma.
		clause := low[idx:]
		if c := strings.IndexByte(clause, ','); c >= 0 {
			clause = clause[:c]
		}
		cascades[table] = strings.Contains(clause, "on delete cascade")
	}

	if len(cascades) == 0 {
		t.Fatal("no FK to seed(id) found in the migrations — the parser matched nothing, which is itself a regression")
	}

	// Named explicitly, so a rename or a parser slip fails here instead of passing vacuously.
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
		// A Down's non-cascading re-add is not the effective schema, so it is dropped.
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
