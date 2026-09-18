package migrations

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

var seedReference = regexp.MustCompile(`references\s+seed\s*\(`)

// A multi-action ALTER holds one clause per FK, so a clause ends at the next REFERENCES.

func seedClause(decl string) string {
	loc := seedReference.FindStringIndex(decl)
	if loc == nil {
		return ""
	}
	clause := decl[loc[0]+len("references"):]
	if next := seedReference.FindStringIndex(clause); next != nil {
		clause = clause[:next[0]]
	}
	return clause
}

func TestSeedForeignKeysCascadeOnDelete(t *testing.T) {
	// R4-R2 (#752): a seed-referencing FK at the default NO ACTION turns a Seed delete into a 500.
	schema := effectiveSchema(t)

	tables := make([]string, 0, len(schema))
	for table := range schema {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	carriers := map[string]bool{}
	for _, table := range tables {
		for _, action := range schema[table].unmodelled {
			if seedReference.MatchString(action) {
				t.Fatalf("a migration alters %s in a form this reader does not model and the "+
					"action names a reference to seed: %s", table, action)
			}
		}
		names := make([]string, 0, len(schema[table].constraints))
		for name := range schema[table].constraints {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			decl := schema[table].constraints[name]
			clause := seedClause(decl)
			if clause == "" {
				continue
			}
			carriers[table] = true
			if !strings.Contains(clause, "on delete cascade") {
				t.Errorf("%s on %s must be ON DELETE CASCADE; without it, deleting a Seed that "+
					"has a dependent %[2]s row returns a 500 (R4-R2 #752), got: %s", name, table, decl)
			}
		}
	}

	if len(carriers) == 0 {
		t.Fatal("no FK to seed(id) survives the migrations — either the parser matched nothing " +
			"or a later DROP CONSTRAINT removed them all, and both are regressions")
	}

	// Named explicitly, so a rename or a parser slip fails here instead of passing vacuously.
	for _, want := range []string{"cold_scan_scope", "zone_file", "admitted_name", "proposal"} {
		if tbl := schema[want]; tbl != nil {
			assertModelled(t, want, tbl)
		}
		if !carriers[want] {
			t.Errorf("expected an FK to seed(id) from %q to stand, but the migrations leave none", want)
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
