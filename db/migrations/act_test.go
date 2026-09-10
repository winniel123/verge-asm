package migrations

import (
	"strings"
	"testing"
)

func actTable(t *testing.T) string {
	t.Helper()
	for _, s := range strings.Split(upMigrations(t), ";") {
		low := strings.ToLower(s)
		if strings.Contains(low, "create table act") {
			return low
		}
	}
	t.Fatal("no CREATE TABLE act found — the corpus is what the Act spec builds (audit-act §4.1)")
	return ""
}

// It returns the one declaration line, so a CHECK holding a comma survives the read.

func actColumn(t *testing.T, name string) string {
	t.Helper()
	for _, line := range strings.Split(actTable(t), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), name+" ") {
			return strings.TrimSpace(line)
		}
	}
	t.Fatalf("act declares no %s column", name)
	return ""
}

func TestActorKindCarriesTwoTokensAndNoSystem(t *testing.T) {
	// System was dropped because no migration writes an Act, so the variant would
	// be uninhabited (audit-act §3.4).
	col := actColumn(t, "actor_kind")

	for _, token := range []string{"'account'", "'grant_holder'"} {
		if !strings.Contains(col, token) {
			t.Errorf("actor_kind's CHECK omits %s, got: %s", token, col)
		}
	}
	if strings.Contains(col, "'system'") {
		t.Errorf("actor_kind's CHECK carries 'system'; §3.4 dropped it as uninhabited, got: %s", col)
	}
	if !strings.Contains(col, "not null") {
		t.Errorf("actor_kind must be NOT NULL, got: %s", col)
	}
}

func TestActionCarriesNoCheckConstraint(t *testing.T) {
	// A 61-token CHECK needs a migration per act class and fails at runtime rather
	// than in CI, so the Go encoder and the AST gate hold the set (audit-act §4.1).
	col := actColumn(t, "action")

	if strings.Contains(col, "check") {
		t.Errorf("action carries a CHECK; §4.1 diverges from transcript deliberately, got: %s", col)
	}
	if !strings.Contains(col, "not null") {
		t.Errorf("action must be NOT NULL, got: %s", col)
	}
}

func TestActPinsNoAccount(t *testing.T) {
	// The shipped attribution columns restrict, so copying them would make an
	// admin who has ever acted unremovable (audit-act §5.4).
	stmt := actTable(t)

	if strings.Contains(stmt, "references account") {
		t.Errorf("act carries an FK into account; §5.4 refuses one and carries the name instead:\n%s", stmt)
	}
}

func TestActStampsItsOwnRecordingTime(t *testing.T) {
	// created_at times the recording and not the act, and the caller cannot forge
	// it (audit-act §4.1).
	col := actColumn(t, "created_at")

	if !strings.Contains(col, "not null") || !strings.Contains(col, "default now()") {
		t.Errorf("created_at must be NOT NULL DEFAULT now(), got: %s", col)
	}
}

func TestActPayloadColumnsAreJSONB(t *testing.T) {
	// The payload is typed and rendered at read time, never a frozen sentence, and
	// no variant carries a list-valued subject (audit-act §4.1).
	for _, name := range []string{"actor", "subject"} {
		col := actColumn(t, name)
		if !strings.Contains(col, "jsonb") || !strings.Contains(col, "not null") {
			t.Errorf("%s must be JSONB NOT NULL, got: %s", name, col)
		}
	}
}
