package migrations

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var attributionColumns = []struct{ table, column string }{
	{"seed", "created_by"},
	{"exclusion", "created_by"},
	{"vantage", "created_by"},
	{"verge_core_frequency_edit", "created_by"},
	{"cold_scan_scope", "created_by"},
	{"zone_file", "uploaded_by"},
	{"channel", "created_by"},
	{"retention_settings", "updated_by"},
	{"proposer_lookup", "created_by"},
	{"report_schedule", "created_by"},
	{"sso_provider", "created_by"},
	{"instance_config", "api_updated_by"},
	{"instance_config", "update_check_updated_by"},
	{"instance_config", "seed_address_cap_updated_by"},
}

func flatUpMigrations(t *testing.T) string {
	t.Helper()
	// A multi-action ALTER TABLE spans lines, so the clauses match on one flat string.
	return strings.Join(strings.Fields(strings.ToLower(upMigrations(t))), " ")
}

func TestAttributionColumnsDoNotPinTheAccount(t *testing.T) {
	// The Act corpus captures the username as a value, so removal can no longer destroy
	// the record of what an account did (§5.4, §9). seed_withdrawal.created_by is the
	// fifteenth column and already shipped SET NULL, so its own test holds it.
	up := flatUpMigrations(t)

	for _, c := range attributionColumns {
		drop := "alter table " + c.table + " alter column " + c.column + " drop not null"
		if !strings.Contains(up, drop) {
			t.Errorf("%s.%s must be nullable; no statement matches %q", c.table, c.column, drop)
		}
		fk := "add constraint " + c.table + "_" + c.column + "_fkey foreign key (" + c.column +
			") references account (id) on delete set null"
		if !strings.Contains(up, fk) {
			t.Errorf("the FK from %s.%s to account(id) must be ON DELETE SET NULL; without it a live "+
				"declaration pins its author and DeleteAccount can never remove them; no statement matches %q",
				c.table, c.column, fk)
		}
		if strings.Contains(up, "alter table "+c.table+" alter column "+c.column+" set not null") {
			t.Errorf("re-tightening %s.%s to NOT NULL restores the refusal §9 withdraws", c.table, c.column)
		}
	}
}

func TestRetentionInstantIsNullable(t *testing.T) {
	// A rendered by-clause reads its meaning off the instant beside it, and the seeded
	// singleton carried an instant nobody wrote (docs/spec/audit-act.md §9.2).
	up := flatUpMigrations(t)

	for _, want := range []string{
		"alter table retention_settings alter column updated_at drop not null",
		"update retention_settings set updated_at = null where updated_by is null",
	} {
		if !strings.Contains(up, want) {
			t.Errorf("the never-moved retention row must carry no instant; no statement matches %q", want)
		}
	}
}

var accountJoinRe = regexp.MustCompile(`(?i)(left\s+(?:outer\s+)?)?join\s+account\s+\w+\s+on\s+([^\n;]*)`)

var attributionRefRe = regexp.MustCompile(`(?i)\b\w+_by\b`)

func TestAttributionJoinsAreLeftJoins(t *testing.T) {
	// An INNER JOIN turns a NULL author into a vanished object, which silently reverts a
	// frequency edit and unterminates a citation chain (§9.1). An attribution JOIN names
	// a *_by column; an identity JOIN (account_id) does not, and stays an INNER JOIN.
	found := map[string]int{}

	entries, err := os.ReadDir(filepath.Join("..", "queries"))
	if err != nil {
		t.Fatalf("read queries dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		body, rerr := os.ReadFile(filepath.Join("..", "queries", name))
		if rerr != nil {
			t.Fatalf("read %s: %v", name, rerr)
		}
		for _, m := range accountJoinRe.FindAllStringSubmatch(string(body), -1) {
			if !attributionRefRe.MatchString(m[2]) {
				continue
			}
			if strings.TrimSpace(m[1]) == "" {
				t.Errorf("%s: an attribution JOIN on account must be a LEFT JOIN, or the object "+
					"vanishes once its author is removed: %s", name, strings.TrimSpace(m[0]))
				continue
			}
			found[name]++
		}
	}

	// Named explicitly, so a deleted query fails here instead of passing vacuously. §9.1
	// deletes six of the eleven attribution JOINs and widens the five that render.
	want := map[string]int{"channels.sql": 1, "sso.sql": 1, "subjects.sql": 3}
	for name, n := range want {
		if found[name] != n {
			t.Errorf("expected %d LEFT JOIN account on an attribution column in %s, found %d",
				n, name, found[name])
		}
	}
	for name := range found {
		if _, ok := want[name]; !ok {
			t.Errorf("%s grew an attribution JOIN on account; §9.1 guards exactly five", name)
		}
	}
}
