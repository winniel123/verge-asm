package migrations

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"
)

// goose's parser sits under its internal/ tree, so no test can call it, and cmd/goose pulls eight
// drivers this module's go.sum does not carry. The rules below are transcribed from
// sqlparser.ParseSQLMigration and extractAnnotation at the version named here (#2261).

const transcribedGooseVersion = "v3.28.0"

const (
	annotationUp             = "Up"
	annotationDown           = "Down"
	annotationStatementBegin = "StatementBegin"
	annotationStatementEnd   = "StatementEnd"
	annotationNoTransaction  = "NO TRANSACTION"
	annotationEnvsubOn       = "ENVSUB ON"
	annotationEnvsubOff      = "ENVSUB OFF"
)

var gooseAnnotations = []string{
	annotationUp,
	annotationDown,
	annotationStatementBegin,
	annotationStatementEnd,
	annotationNoTransaction,
	annotationEnvsubOn,
	annotationEnvsubOff,
}

// A migration that carries one of these runs outside a transaction, or with $VAR interpolation
// live. goose accepts such a line wherever it sits, so prose that spells one changes behaviour
// and reports nothing. No migration has ever used one, which is what makes a spelled one a
// mistake the gate below can name (#2261).

var behaviourChangingAnnotations = []string{
	annotationNoTransaction,
	annotationEnvsubOn,
	annotationEnvsubOff,
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

func endsWithSemicolon(line string) bool {
	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}
	return strings.HasSuffix(prev, ";")
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

type gooseDirection int

const (
	directionUp gooseDirection = iota
	directionDown
)

func missingSemicolonError(rest string) error {
	return fmt.Errorf("unexpected unfinished SQL query %q: missing semicolon?", rest)
}

func checkGooseMigration(body string) error {
	// goose parses each file once per direction, and only the down pass reaches a down
	// statement, so a missing semicolon there surfaces in that pass alone (#2261).
	if err := parseGooseMigration(body, directionUp); err != nil {
		return err
	}
	return parseGooseMigration(body, directionDown)
}

func parseGooseMigration(body string, direction gooseDirection) error {
	state := gooseStart
	var buf strings.Builder

	for _, line := range strings.Split(body, "\n") {
		if state == gooseStart && strings.TrimSpace(line) == "" {
			continue
		}
		if isGooseAnnotationLine(line) {
			cmd, err := extractGooseAnnotation(line)
			if err != nil {
				return fmt.Errorf("failed to parse annotation line %q: %w", line, err)
			}
			consumed := true
			switch cmd {
			case annotationUp:
				if state != gooseStart {
					return errors.New("duplicate '-- +goose Up' annotation")
				}
				state = gooseUp
			case annotationDown:
				switch state {
				case gooseUp, gooseStatementEndUp:
					if rest := strings.TrimSpace(buf.String()); rest != "" {
						return missingSemicolonError(rest)
					}
					state = gooseDown
				default:
					return errors.New("'-- +goose Down' must follow '-- +goose Up'")
				}
			case annotationStatementBegin:
				switch state {
				case gooseUp, gooseStatementEndUp:
					state = gooseStatementBeginUp
				case gooseDown, gooseStatementEndDown:
					state = gooseStatementBeginDown
				default:
					return errors.New(
						"'-- +goose StatementBegin' must follow '-- +goose Up' or '-- +goose Down'")
				}
			case annotationStatementEnd:
				switch state {
				case gooseStatementBeginUp:
					state = gooseStatementEndUp
				case gooseStatementBeginDown:
					state = gooseStatementEndDown
				default:
					return errors.New(
						"'-- +goose StatementEnd' must follow '-- +goose StatementBegin'")
				}
				// goose does not skip this line: the flush below closes the statement on it.
				consumed = false
			}
			if consumed {
				continue
			}
		}
		if buf.Len() == 0 {
			if strings.HasPrefix(strings.TrimSpace(line), "--") || line == "" {
				continue
			}
		}
		switch state {
		case gooseStatementEndUp, gooseStatementEndDown:
		default:
			buf.WriteString(line + "\n")
		}
		switch state {
		case gooseUp, gooseStatementBeginUp, gooseStatementEndUp:
			if direction == directionDown {
				buf.Reset()
				continue
			}
		case gooseDown, gooseStatementBeginDown, gooseStatementEndDown:
			if direction == directionUp {
				buf.Reset()
				continue
			}
		default:
			return fmt.Errorf("no '-- +goose Up' annotation above the statement %q", line)
		}
		switch state {
		case gooseUp, gooseDown:
			if endsWithSemicolon(line) {
				buf.Reset()
			}
		case gooseStatementEndUp:
			buf.Reset()
			state = gooseUp
		case gooseStatementEndDown:
			buf.Reset()
			state = gooseDown
		}
	}

	switch state {
	case gooseStart:
		return errors.New("no '-- +goose Up' annotation")
	case gooseStatementBeginUp, gooseStatementBeginDown:
		return errors.New("missing '-- +goose StatementEnd' annotation")
	}
	if rest := strings.TrimSpace(buf.String()); rest != "" {
		return missingSemicolonError(rest)
	}
	return nil
}

func migrationNames(t *testing.T) []string {
	t.Helper()
	names, err := fs.Glob(FS, "*.sql")
	if err != nil {
		t.Fatalf("glob the embedded migrations: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("the embedded migration set is empty, so these gates prove nothing")
	}
	return names
}

func TestEveryMigrationParsesUnderGoose(t *testing.T) {
	for _, name := range migrationNames(t) {
		body, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := checkGooseMigration(string(body)); err != nil {
			// goose.Up runs inside the web binary, which calls log.Fatalf on its error, so
			// the container restarts instead of serving (#2261).
			t.Errorf("%s: goose would refuse this migration: %v", name, err)
		}
	}
}

func TestNoMigrationSpellsABehaviourChangingAnnotation(t *testing.T) {
	for _, name := range migrationNames(t) {
		body, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(body), "\n") {
			if !isGooseAnnotationLine(line) {
				continue
			}
			cmd, err := extractGooseAnnotation(line)
			if err != nil {
				continue
			}
			for _, banned := range behaviourChangingAnnotations {
				if cmd != banned {
					continue
				}
				t.Errorf("%s:%d carries `-- +goose %s`, which goose honours wherever it "+
					"sits. No migration has used one, so this reads as prose that spelled "+
					"the annotation rather than naming it. Update this test when a "+
					"migration means it (#2261): %q", name, i+1, banned, line)
			}
		}
	}
}

var gooseRequireLine = regexp.MustCompile(`(?m)^\s*github\.com/pressly/goose/v3 (v\S+)`)

func TestTheTranscribedGooseVersionIsThePinnedOne(t *testing.T) {
	body, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	match := gooseRequireLine.FindSubmatch(body)
	if match == nil {
		t.Fatal("go.mod names no github.com/pressly/goose/v3 requirement")
	}
	if got := string(match[1]); got != transcribedGooseVersion {
		// A bump can add an annotation or change the parser, and this file would then
		// refuse a migration goose accepts, blaming the migration (#2261).
		t.Errorf("go.mod pins goose %s and this file transcribes %s; re-read "+
			"internal/sqlparser and move transcribedGooseVersion", got, transcribedGooseVersion)
	}
}

func TestGooseMigrationCheckRefusesWhatGooseRefuses(t *testing.T) {
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
			name: "statement above the up annotation",
			body: "CREATE TABLE early (id int);\n-- +goose Up\nCREATE TABLE t (id int);\n",
			want: "no '-- +goose Up' annotation above the statement",
		},
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
			name: "up statement missing its semicolon",
			body: "-- +goose Up\nCREATE TABLE t (id int)\n\n-- +goose Down\nDROP TABLE t;\n",
			want: "missing semicolon?",
		},
		{
			name: "down statement missing its semicolon",
			body: "-- +goose Up\nCREATE TABLE t (id int);\n\n-- +goose Down\nDROP TABLE t\n",
			want: "missing semicolon?",
		},
		{
			name: "last statement missing its semicolon with no down section",
			body: "-- +goose Up\nCREATE TABLE t (id int)\n",
			want: "missing semicolon?",
		},
		{
			name: "statement block",
			body: "-- +goose Up\n-- +goose StatementBegin\nCREATE FUNCTION f() RETURNS int AS $$\n" +
				"BEGIN; RETURN 1; END;\n$$ LANGUAGE plpgsql;\n-- +goose StatementEnd\n",
			want: "",
		},
		{
			name: "duplicate up annotation",
			body: "-- +goose Up\nCREATE TABLE a (id int);\n-- +goose Up\nCREATE TABLE b (id int);\n",
			want: "duplicate",
		},
		{
			name: "second down annotation after a down section",
			body: "-- +goose Up\nCREATE TABLE a (id int);\n-- +goose Down\nDROP TABLE a;\n" +
				"-- +goose Down\nDROP TABLE b;\n",
			want: "must follow '-- +goose Up'",
		},
		{
			name: "no transaction is accepted",
			body: "-- +goose NO TRANSACTION\n" + good,
			want: "",
		},
		{
			// The residual gap §2.3 of docs/spec/comment-policy.md records: goose honours a
			// lone Down anywhere, so every statement below it becomes a down statement.
			name: "a lone down annotation with no down section is accepted",
			body: "-- +goose Up\nCREATE TABLE a (id int);\n-- +goose Down\nCREATE TABLE b (id int);\n",
			want: "",
		},
		{
			name: "trailing comment after the semicolon",
			body: "-- +goose Up\nCREATE TABLE t (id int); -- a trailing note\n",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkGooseMigration(tc.body)
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
