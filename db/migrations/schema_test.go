package migrations

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type sqlTable struct {
	cols        map[string]string
	constraints map[string]string
	unmodelled  []string
}

func newSQLTable() *sqlTable {
	return &sqlTable{cols: map[string]string{}, constraints: map[string]string{}}
}

// Postgres spells IF EXISTS before ONLY, and the other order names a table called `only`.

var (
	createTableStmt = regexp.MustCompile(`(?s)^create\s+table\s+(?:if\s+not\s+exists\s+)?(?:public\.)?(\w+)\s*\(`)
	alterTableStmt  = regexp.MustCompile(`(?s)^alter\s+table\s+(?:if\s+exists\s+)?(?:only\s+)?(?:public\.)?(\w+)\s+(.*)`)
	dropTableStmt   = regexp.MustCompile(`(?s)^drop\s+table\s+(?:if\s+exists\s+)?(?:public\.)?(\w+)`)
	defaultClause   = regexp.MustCompile(`\s+default\s+(?:\([^)]*\)|'[^']*'|[^\s,]+)(?:::[\w.]+(?:\[\])?)*`)
	parenCols       = regexp.MustCompile(`\(([^)]*)\)`)
)

// A greedy regex would run a table's column list on into a trailing WITH or PARTITION BY.

func balancedBody(s string, open int) (string, bool) {
	depth, quoted := 0, false
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '\'':
			quoted = !quoted
		case '(':
			if !quoted {
				depth++
			}
		case ')':
			if !quoted {
				depth--
				if depth == 0 {
					return s[open+1 : i], true
				}
			}
		}
	}
	return "", false
}

// Splitting a DDL list on every comma would cut a CHECK's IN-list and a multi-column key.

func splitTopLevel(s string) []string {
	var out []string
	depth, start, quoted := 0, 0, false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			quoted = !quoted
		case '(':
			if !quoted {
				depth++
			}
		case ')':
			if !quoted {
				depth--
			}
		case ',':
			if !quoted && depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	return append(out, s[start:])
}

func normSQL(s string) string { return strings.Join(strings.Fields(s), " ") }

func (tbl *sqlTable) put(name, decl string) {
	key := name
	for i := 1; ; i++ {
		if _, taken := tbl.constraints[key]; !taken {
			break
		}
		key = fmt.Sprintf("%s%d", name, i)
	}
	tbl.constraints[key] = decl
}

func firstParenCol(s string) string {
	m := parenCols.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return strings.Fields(strings.Split(m[1], ",")[0])[0]
}

// Postgres derives these names, and 23500 drops an inline REFERENCES by the name it derived.

func (tbl *sqlTable) inlineConstraints(table, col, decl string) {
	if strings.Contains(decl, "references ") {
		tbl.constraints[table+"_"+col+"_fkey"] = decl
	}
	if strings.Contains(decl, "primary key") {
		tbl.constraints[table+"_pkey"] = decl
	}
	if strings.Contains(decl, "check (") {
		tbl.constraints[table+"_"+col+"_check"] = decl
	}
	if strings.Contains(decl, " unique") {
		tbl.constraints[table+"_"+col+"_key"] = decl
	}
}

// An inline CHECK lives in the column text, so dropping it by name must cut it from there too.

func stripCheck(decl string) string {
	i := strings.Index(decl, "check (")
	if i < 0 {
		return decl
	}
	depth := 0
	for j := i + len("check "); j < len(decl); j++ {
		switch decl[j] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return normSQL(decl[:i] + decl[j+1:])
			}
		}
	}
	return decl
}

// A CHECK predicate spells IS NOT NULL, so the nullability clause is the one at depth zero.

func stripNotNull(decl string) string {
	const want = " not null"
	depth := 0
	for i := 0; i+len(want) <= len(decl); i++ {
		switch decl[i] {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth == 0 && decl[i:i+len(want)] == want {
			return normSQL(decl[:i] + decl[i+len(want):])
		}
	}
	return decl
}

func (tbl *sqlTable) createItem(table, item string) {
	f := strings.Fields(item)
	if len(f) == 0 {
		return
	}
	switch f[0] {
	case "constraint":
		if len(f) < 2 {
			tbl.unmodelled = append(tbl.unmodelled, item)
			return
		}
		tbl.constraints[f[1]] = item
	case "primary":
		tbl.put(table+"_pkey", item)
	case "unique":
		tbl.put(table+"_"+firstParenCol(item)+"_key", item)
	case "check":
		tbl.put(table+"_check", item)
	case "foreign":
		tbl.put(table+"_"+firstParenCol(item)+"_fkey", item)
	case "exclude", "like":
		tbl.unmodelled = append(tbl.unmodelled, item)
	default:
		tbl.cols[f[0]] = item
		tbl.inlineConstraints(table, f[0], item)
	}
}

func dropTarget(f []string) string {
	if len(f) >= 2 && f[0] == "if" && f[1] == "exists" {
		f = f[2:]
	}
	if len(f) == 0 {
		return ""
	}
	return f[0]
}

func (tbl *sqlTable) alterColumn(f []string, action string) {
	if len(f) < 2 {
		tbl.unmodelled = append(tbl.unmodelled, action)
		return
	}
	col, sub := f[0], strings.Join(f[1:], " ")
	decl, declared := tbl.cols[col]
	if !declared {
		tbl.unmodelled = append(tbl.unmodelled, action)
		return
	}
	switch {
	case sub == "drop not null":
		tbl.cols[col] = stripNotNull(decl)
	case sub == "set not null":
		tbl.cols[col] = decl + " not null"
	case sub == "drop default":
		tbl.cols[col] = normSQL(defaultClause.ReplaceAllString(decl, ""))
	case strings.HasPrefix(sub, "set default "):
		tbl.cols[col] = normSQL(defaultClause.ReplaceAllString(decl, "")) + " " + strings.TrimPrefix(sub, "set ")
	default:
		tbl.unmodelled = append(tbl.unmodelled, action)
	}
}

func (tbl *sqlTable) alterAction(table, action string) {
	f := strings.Fields(action)
	if len(f) < 2 {
		tbl.unmodelled = append(tbl.unmodelled, action)
		return
	}
	verb, rest := f[0]+" "+f[1], f[2:]
	switch verb {
	case "add column":
		if len(rest) >= 3 && rest[0] == "if" {
			rest = rest[3:]
		}
		if len(rest) == 0 {
			tbl.unmodelled = append(tbl.unmodelled, action)
			return
		}
		decl := strings.Join(rest, " ")
		tbl.cols[rest[0]] = decl
		tbl.inlineConstraints(table, rest[0], decl)
	case "drop column":
		col := dropTarget(rest)
		delete(tbl.cols, col)
		for _, suffix := range []string{"_fkey", "_check", "_key"} {
			delete(tbl.constraints, table+"_"+col+suffix)
		}
	case "add constraint":
		if len(rest) == 0 {
			tbl.unmodelled = append(tbl.unmodelled, action)
			return
		}
		tbl.constraints[rest[0]] = action
	case "drop constraint":
		name := dropTarget(rest)
		delete(tbl.constraints, name)
		if col, inline := strings.CutSuffix(name, "_check"); inline {
			col = strings.TrimPrefix(col, table+"_")
			if decl, declared := tbl.cols[col]; declared {
				tbl.cols[col] = stripCheck(decl)
			}
		}
	case "alter column":
		tbl.alterColumn(rest, action)
	case "validate constraint":
		name := dropTarget(rest)
		if decl, known := tbl.constraints[name]; known {
			tbl.constraints[name] = normSQL(strings.Replace(decl, " not valid", "", 1))
		}
	default:
		tbl.unmodelled = append(tbl.unmodelled, action)
	}
}

// upMigrations concatenates every Up, so a later DROP or ALTER is what decides the schema.

func effectiveSchema(t *testing.T) map[string]*sqlTable {
	t.Helper()
	tables := map[string]*sqlTable{}
	for _, raw := range strings.Split(strings.ToLower(upMigrations(t)), ";") {
		stmt := normSQL(raw)
		if m := createTableStmt.FindStringSubmatchIndex(stmt); m != nil {
			table := stmt[m[2]:m[3]]
			tbl := newSQLTable()
			tables[table] = tbl
			body, closed := balancedBody(stmt, m[1]-1)
			if !closed {
				tbl.unmodelled = append(tbl.unmodelled, "an unterminated CREATE TABLE body")
				continue
			}
			for _, item := range splitTopLevel(body) {
				tbl.createItem(table, normSQL(item))
			}
			continue
		}
		if m := dropTableStmt.FindStringSubmatch(stmt); m != nil {
			delete(tables, m[1])
			continue
		}
		if m := alterTableStmt.FindStringSubmatch(stmt); m != nil {
			tbl := tables[m[1]]
			if tbl == nil {
				tbl = newSQLTable()
				tables[m[1]] = tbl
			}
			for _, action := range splitTopLevel(m[2]) {
				tbl.alterAction(m[1], normSQL(action))
			}
		}
	}
	return tables
}

func assertModelled(t *testing.T, table string, tbl *sqlTable) {
	t.Helper()
	if len(tbl.unmodelled) > 0 {
		sort.Strings(tbl.unmodelled)
		t.Fatalf("a migration alters %s in a form this reader does not model, so every "+
			"assertion over that table would read a stale schema; teach effectiveSchema "+
			"these actions first: %v", table, tbl.unmodelled)
	}
}

func schemaOf(t *testing.T, table string) *sqlTable {
	t.Helper()
	tbl := effectiveSchema(t)[table]
	if tbl == nil {
		t.Fatalf("the migrations leave no table %s standing; a later DROP TABLE removes "+
			"what the assertions below read", table)
	}
	assertModelled(t, table, tbl)
	return tbl
}

func tableColumns(t *testing.T, table string) map[string]string {
	t.Helper()
	return schemaOf(t, table).cols
}

func tableConstraints(t *testing.T, table string) map[string]string {
	t.Helper()
	return schemaOf(t, table).constraints
}

func tableColumn(t *testing.T, table, col string) string {
	t.Helper()
	decl, ok := tableColumns(t, table)[col]
	if !ok {
		t.Fatalf("%s declares no %s column, or a later migration drops it", table, col)
	}
	return decl
}

func constraintsMatching(t *testing.T, table, want string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for name, decl := range tableConstraints(t, table) {
		if strings.Contains(decl, want) {
			out[name] = decl
		}
	}
	return out
}
