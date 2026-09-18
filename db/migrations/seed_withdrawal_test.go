package migrations

import (
	"strings"
	"testing"
)

func TestSeedWithdrawalDoesNotPinItsAuthor(t *testing.T) {
	// Unlike seed and exclusion, a tombstone outlives every operator act (ADR-0134 §3).
	col := tableColumn(t, "seed_withdrawal", "created_by")

	if !strings.Contains(col, "bigint") {
		t.Errorf("seed_withdrawal.created_by must be a BIGINT, got: %s", col)
	}
	if strings.Contains(col, "not null") {
		t.Errorf("seed_withdrawal.created_by must be nullable, got: %s", col)
	}

	fk, ok := tableConstraints(t, "seed_withdrawal")["seed_withdrawal_created_by_fkey"]
	if !ok {
		t.Fatal("no seed_withdrawal_created_by_fkey stands; without an FK the column stops " +
			"being an attribution at all (ADR-0134 §3)")
	}
	if !strings.Contains(fk, "references account (id)") || !strings.Contains(fk, "on delete set null") {
		t.Errorf("the FK from seed_withdrawal.created_by to account(id) must be ON DELETE SET NULL; "+
			"without it a tombstone pins its author for ever and DeleteAccount can never remove them, got: %s", fk)
	}
}

func TestSeedWithdrawalCarriesBothLimbs(t *testing.T) {
	// ADR-0134 §7 left the name limb out, so the CREATE TABLE still reads NOT NULL (ADR-0135 §2).
	cols := tableColumns(t, "seed_withdrawal")

	for _, want := range []string{"address_cidr", "name_domain", "kind"} {
		if _, ok := cols[want]; !ok {
			t.Errorf("the tombstone must carry both limbs (ADR-0135 §2); seed_withdrawal leaves "+
				"no %s column standing", want)
		}
	}
	if strings.Contains(cols["address_cidr"], "not null") {
		t.Errorf("a NOT NULL seed_withdrawal.address_cidr forbids the name limb its tombstone, got: %s",
			cols["address_cidr"])
	}
	if _, ok := tableConstraints(t, "seed_withdrawal")["seed_withdrawal_shape"]; !ok {
		t.Error("no seed_withdrawal_shape constraint stands; nothing then ties the limb to its column")
	}
}
