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

// TestSeedWithdrawalCarriesBothLimbs guards ADR-0135 §2's shape.
//
// 24700 created the tombstone with `address_cidr CIDR NOT NULL`, because ADR-0134
// §7 deliberately left the name limb out. 24800 relaxes that column and adds `kind`
// and `name_domain`, so the row mirrors `seed` itself: one table, a discriminator,
// and a CHECK that exactly one scope column is populated.
//
// A reader who greps only the CREATE TABLE still sees the NOT NULL, so this test
// asserts the EFFECTIVE schema. A later migration that re-tightens the column, or
// drops the shape CHECK, silently stops a name Seed recording its mover — and a
// withdrawal with no mover is the leak the tombstone exists to close.
func TestSeedWithdrawalCarriesBothLimbs(t *testing.T) {
	up := strings.ToLower(upMigrations(t))

	for _, want := range []string{
		"alter table seed_withdrawal alter column address_cidr drop not null",
		"alter table seed_withdrawal add column name_domain",
		"alter table seed_withdrawal add column kind",
		"seed_withdrawal_shape",
	} {
		if !strings.Contains(up, want) {
			t.Errorf("the tombstone must carry both limbs (ADR-0135 §2); no statement matches %q", want)
		}
	}
	if strings.Contains(up, "alter table seed_withdrawal alter column address_cidr set not null") {
		t.Error("re-tightening seed_withdrawal.address_cidr to NOT NULL forbids the name limb its tombstone")
	}
}
