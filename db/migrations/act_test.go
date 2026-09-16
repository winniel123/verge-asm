package migrations

import (
	"regexp"
	"sort"
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

type actIndex struct {
	cols    []string
	partial bool
}

func (ix actIndex) String() string {
	s := "(" + strings.Join(ix.cols, ", ") + ")"
	if ix.partial {
		s += " partial"
	}
	return s
}

var (
	createActIndex = regexp.MustCompile(
		`create\s+(?:unique\s+)?index\s+(?:concurrently\s+)?(?:if\s+not\s+exists\s+)?` +
			`(\w+)\s+on\s+(?:public\.)?act\s*\((.*)`)
	dropActIndex = regexp.MustCompile(
		`drop\s+index\s+(?:concurrently\s+)?(?:if\s+exists\s+)?(?:public\.)?(\w+)`)
)

func parseActIndex(body string) actIndex {
	cols, rest := body, ""
	if i := strings.Index(body, ")"); i >= 0 {
		cols, rest = body[:i], body[i+1:]
	}
	ix := actIndex{partial: strings.Contains(rest, "where")}
	for _, c := range strings.Split(cols, ",") {
		ix.cols = append(ix.cols, strings.Join(strings.Fields(c), " "))
	}
	return ix
}

// upMigrations strips every -- comment, so prose cannot satisfy the tests below. It also
// concatenates every Up, so a later DROP INDEX is what decides whether an index stands.

func actIndexes(t *testing.T) map[string]actIndex {
	t.Helper()
	out := map[string]actIndex{}
	for _, s := range strings.Split(strings.ToLower(upMigrations(t)), ";") {
		if m := createActIndex.FindStringSubmatch(s); m != nil {
			out[m[1]] = parseActIndex(m[2])
			continue
		}
		if m := dropActIndex.FindStringSubmatch(s); m != nil {
			delete(out, m[1])
		}
	}
	return out
}

func actIndexesLeadingOn(t *testing.T, col string) (map[string]actIndex, []string) {
	t.Helper()
	idx := actIndexes(t)
	var names []string
	for name, ix := range idx {
		if len(ix.cols) > 0 && ix.cols[0] == col {
			names = append(names, name)
		}
	}
	// A map ranges in no fixed order, so a second matching index must not decide the run.
	sort.Strings(names)
	return idx, names
}

func TestActIndexesTheClassTheScopePanelFiltersOn(t *testing.T) {
	// One ANY() read runs on every admin Exposure load, and an index leading on
	// created_at makes it walk the whole window to return its matches (#2073).
	idx, names := actIndexesLeadingOn(t, "action")

	if len(names) == 0 {
		t.Fatalf("no act index leads on action; ListActsOfClassesSince filters on it and "+
			"act_created_at_idx cannot seek to a class, got: %v", idx)
	}
	want := []string{"action", "created_at desc", "id desc"}
	for _, name := range names {
		ix := idx[name]
		// PG 16 sorts a ScalarArrayOp match, so these trailing columns buy ordering on a later PG.
		for i, col := range want {
			if i >= len(ix.cols) || ix.cols[i] != col {
				t.Errorf("%s must key (%s) so the read seeks to a class, got: %v",
					name, strings.Join(want, ", "), ix)
				break
			}
		}
		// A partial index needs a migration per class added, and the set already moved (#2169).
		if ix.partial {
			t.Errorf("%s is partial; the act class set is not fixed (#2169), got: %v", name, ix)
		}
	}
}

func TestActKeepsItsDateRangeIndex(t *testing.T) {
	// ListActsInRange carries no action predicate, so an index leading on action cannot
	// serve it: the action index is an addition and never a replacement (#2073).
	idx, names := actIndexesLeadingOn(t, "created_at desc")

	if len(names) == 0 {
		t.Fatalf("no act index leads on created_at desc; ListActsInRange filters on the "+
			"range alone, got: %v", idx)
	}
}
