package migrations

import (
	"regexp"
	"strings"
	"testing"
)

func TestSeedWithdrawalDoesNotPinItsAuthor(t *testing.T) {
	// Unlike seed and exclusion, a tombstone outlives every operator act (ADR-0134 §3).
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

func TestSeedWithdrawalCarriesBothLimbs(t *testing.T) {
	// ADR-0134 §7 left the name limb out, so the CREATE TABLE still reads NOT NULL (ADR-0135 §2).
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
