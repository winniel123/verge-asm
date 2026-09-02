package migrations

import (
	"regexp"
	"strings"
	"testing"
)

// TestSeedWithdrawalDoesNotPinItsAuthor guards the one place ADR-0134 §3's
// tombstone parts from the `seed` and `exclusion` rows it otherwise mirrors.
//
// Those two carry `created_by BIGINT NOT NULL REFERENCES account (id)` with no ON
// DELETE, which deliberately refuses to remove an account that authored a live
// declaration (DeleteAccount, Settings -> Team). The operator can lift that refusal
// by removing the declaration.
//
// A tombstone cannot be removed by any operator act, so the same FK would make the
// admin who withdrew an address scope permanently undeletable, with no cleanup path
// even once the row is spent. The attribution is worth keeping while the account
// exists; it is not worth making a member undeletable. A later migration that
// tightens this column back to NOT NULL, or drops the ON DELETE SET NULL, fails
// here — no live database required.
func TestSeedWithdrawalDoesNotPinItsAuthor(t *testing.T) {
	up := upMigrations(t)

	stmt := ""
	for _, s := range strings.Split(up, ";") {
		if strings.Contains(strings.ToLower(s), "create table seed_withdrawal") {
			stmt = strings.ToLower(s)
			break
		}
	}
	if stmt == "" {
		t.Fatal("no CREATE TABLE seed_withdrawal found — the tombstone is the mover a Seed withdrawal reads (ADR-0134 §2)")
	}

	// The created_by column definition: from its name to the end of its clause.
	col := regexp.MustCompile(`(?s)created_by\s+bigint[^,]*`).FindString(stmt)
	if col == "" {
		t.Fatal("seed_withdrawal declares no created_by column")
	}
	if strings.Contains(col, "not null") {
		t.Errorf("seed_withdrawal.created_by must be nullable, got: %s", strings.TrimSpace(col))
	}
	if !strings.Contains(col, "on delete set null") {
		t.Errorf("the FK from seed_withdrawal.created_by to account(id) must be ON DELETE SET NULL; "+
			"without it a tombstone pins its author for ever and DeleteAccount can never remove them, got: %s",
			strings.TrimSpace(col))
	}
}
