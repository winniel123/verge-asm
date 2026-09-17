package migrations

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"testing"
)

// goose's parser lives under its internal/ tree, so no test can call it. The rules below are
// transcribed from sqlparser.ParseSQLMigration and extractAnnotation at goose v3.28.0 (#2261).

var gooseAnnotations = []string{
	"Up",
	"Down",
	"StatementBegin",
	"StatementEnd",
	"NO TRANSACTION",
	"ENVSUB ON",
	"ENVSUB OFF",
}

func isGooseAnnotationLine(line string) bool {
	// goose matches its marker anywhere on a comment line, so backticks do not hide it (#2261).
	return strings.HasPrefix(strings.TrimSpace(line), "--") && strings.Contains(line, "+goose")
}

func extractGooseAnnotation(line string) (string, error) {
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return "", fmt.Errorf("%q contains leading whitespace", line)
	}
	cmd := strings.ReplaceAll(line, "--", "")
	cmd = strings.Replace(cmd, "+goose", "", 1)
	if strings.Contains(cmd, "+goose") {
		return "", fmt.Errorf("%q contains multiple '+goose' annotations", cmd)
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", errors.New("empty annotation")
	}
	for _, supported := range gooseAnnotations {
		if strings.EqualFold(supported, cmd) {
			return supported, nil
		}
	}
	return "", fmt.Errorf("%q is not a goose annotation; one of %s was expected",
		cmd, strings.Join(gooseAnnotations, ", "))
}

type gooseState int

const (
	gooseStart gooseState = iota
	gooseUp
	gooseStatementBeginUp
	gooseStatementEndUp
	gooseDown
	gooseStatementBeginDown
	gooseStatementEndDown
)

func checkGooseAnnotations(body string) error {
	// Annotation lines only, and no SQL. The required `test` job boots no Postgres (#2261).
	state := gooseStart
	for i, line := range strings.Split(body, "\n") {
		if !isGooseAnnotationLine(line) {
			continue
		}
		cmd, err := extractGooseAnnotation(line)
		if err != nil {
			return fmt.Errorf("line %d: failed to parse annotation line %q: %w", i+1, line, err)
		}
		switch cmd {
		case "Up":
			if state != gooseStart {
				return fmt.Errorf("line %d: duplicate '-- +goose Up' annotation", i+1)
			}
			state = gooseUp
		case "Down":
			switch state {
			case gooseUp, gooseStatementEndUp:
				state = gooseDown
			default:
				return fmt.Errorf("line %d: '-- +goose Down' must follow '-- +goose Up'", i+1)
			}
		case "StatementBegin":
			switch state {
			case gooseUp, gooseStatementEndUp:
				state = gooseStatementBeginUp
			case gooseDown, gooseStatementEndDown:
				state = gooseStatementBeginDown
			default:
				return fmt.Errorf(
					"line %d: '-- +goose StatementBegin' must follow '-- +goose Up' or '-- +goose Down'", i+1)
			}
		case "StatementEnd":
			switch state {
			case gooseStatementBeginUp:
				state = gooseStatementEndUp
			case gooseStatementBeginDown:
				state = gooseStatementEndDown
			default:
				return fmt.Errorf(
					"line %d: '-- +goose StatementEnd' must follow '-- +goose StatementBegin'", i+1)
			}
		}
	}
	switch state {
	case gooseStart:
		return errors.New("no '-- +goose Up' annotation")
	case gooseStatementBeginUp, gooseStatementBeginDown:
		return errors.New("missing '-- +goose StatementEnd' annotation")
	}
	return nil
}

func TestEveryMigrationCarriesAnnotationsGooseAccepts(t *testing.T) {
	names, err := fs.Glob(FS, "*.sql")
	if err != nil {
		t.Fatalf("glob the embedded migrations: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("the embedded migration set is empty, so this gate proves nothing")
	}
	for _, name := range names {
		body, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := checkGooseAnnotations(string(body)); err != nil {
			// goose.Up runs inside the web binary, so an unparseable file restarts the
			// container instead of serving it (#2261).
			t.Errorf("%s: goose would refuse this migration: %v", name, err)
		}
	}
}

func TestGooseAnnotationCheckRefusesWhatGooseRefuses(t *testing.T) {
	const good = "-- +goose Up\nCREATE TABLE t (id int);\n\n-- +goose Down\nDROP TABLE t;\n"

	cases := []struct {
		name string
		body string
		want string
	}{
		{"clean", good, ""},
		{
			// The #2261 migration quoted the annotation it had decided against.
			name: "marker quoted in prose",
			body: "-- +goose Up\n-- CONCURRENTLY needs `-- +goose NO TRANSACTION`, which we refuse.\n" +
				"CREATE TABLE t (id int);\n",
			want: "not a goose annotation",
		},
		{
			name: "marker named mid-sentence without backticks",
			body: "-- +goose Up\n-- We do not use +goose NO TRANSACTION here.\nCREATE TABLE t (id int);\n",
			want: "not a goose annotation",
		},
		{"misspelled annotation", "-- +goose Upp\nCREATE TABLE t (id int);\n", "not a goose annotation"},
		{"leading whitespace", "  -- +goose Up\nCREATE TABLE t (id int);\n", "leading whitespace"},
		{"no up annotation", "CREATE TABLE t (id int);\n", "no '-- +goose Up'"},
		{
			name: "unterminated statement block",
			body: "-- +goose Up\n-- +goose StatementBegin\nCREATE TABLE t (id int);\n",
			want: "missing '-- +goose StatementEnd'",
		},
		{
			name: "down before up",
			body: "-- +goose Down\nDROP TABLE t;\n",
			want: "must follow '-- +goose Up'",
		},
		{
			name: "no transaction is accepted",
			body: "-- +goose NO TRANSACTION\n" + good,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkGooseAnnotations(tc.body)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("goose accepts this migration, the gate refused it: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("goose refuses this migration, the gate accepted it")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error does not name its cause, want %q, got %q", tc.want, err)
			}
		})
	}
}
