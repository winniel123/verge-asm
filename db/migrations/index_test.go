package migrations

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

type sqlIndex struct {
	cols   []string
	where  string
	unique bool
}

func (ix sqlIndex) partial() bool { return ix.where != "" }

func (ix sqlIndex) String() string {
	s := "(" + strings.Join(ix.cols, ", ") + ")"
	if ix.unique {
		s = "unique " + s
	}
	if ix.partial() {
		s += " where " + ix.where
	}
	return s
}

func (ix sqlIndex) keys(cols ...string) bool {
	if len(ix.cols) < len(cols) {
		return false
	}
	for i, col := range cols {
		if ix.cols[i] != col {
			return false
		}
	}
	return true
}

var dropIndexStmt = regexp.MustCompile(
	`drop\s+index\s+(?:concurrently\s+)?(?:if\s+exists\s+)?(?:public\.)?(\w+)`)

func createIndexStmt(table string) *regexp.Regexp {
	// (?s) so a declaration broken across lines still yields its WHERE to parseIndex.
	return regexp.MustCompile(
		`(?s)create\s+(unique\s+)?index\s+(?:concurrently\s+)?(?:if\s+not\s+exists\s+)?` +
			`(\w+)\s+on\s+(?:public\.)?` + regexp.QuoteMeta(table) + `\s*\((.*)`)
}

// It returns the key list up to the first ), so a trailing NULLS NOT DISTINCT or WHERE
// cannot read as a column.

func parseIndex(unique, body string) sqlIndex {
	cols, rest := body, ""
	if i := strings.Index(body, ")"); i >= 0 {
		cols, rest = body[:i], body[i+1:]
	}
	ix := sqlIndex{unique: strings.TrimSpace(unique) == "unique"}
	if i := strings.Index(rest, "where"); i >= 0 {
		ix.where = strings.Join(strings.Fields(rest[i+len("where"):]), " ")
	}
	for _, c := range strings.Split(cols, ",") {
		ix.cols = append(ix.cols, strings.Join(strings.Fields(c), " "))
	}
	return ix
}

// upMigrations strips every -- comment, so prose cannot satisfy the callers below. It also
// concatenates every Up, so a later DROP INDEX is what decides whether an index stands.

func tableIndexes(t *testing.T, table string) map[string]sqlIndex {
	t.Helper()
	create := createIndexStmt(table)
	out := map[string]sqlIndex{}
	for _, s := range strings.Split(strings.ToLower(upMigrations(t)), ";") {
		if m := create.FindStringSubmatch(s); m != nil {
			out[m[2]] = parseIndex(m[1], m[3])
			continue
		}
		if m := dropIndexStmt.FindStringSubmatch(s); m != nil {
			delete(out, m[1])
			continue
		}
		// Dropping the table takes its indexes with it, and no DROP INDEX names them.
		if m := dropTableStmt.FindStringSubmatch(normSQL(s)); m != nil && m[1] == table {
			out = map[string]sqlIndex{}
		}
	}
	return out
}

func indexesLeadingOn(t *testing.T, table, col string) (map[string]sqlIndex, []string) {
	t.Helper()
	idx := tableIndexes(t, table)
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
