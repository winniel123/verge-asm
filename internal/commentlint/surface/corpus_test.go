package surface

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/commentlint/scope"
)

const repoRoot = "../../.."

const (
	minGoFiles  = 400
	minSQLFiles = 35
	minCSSFiles = 10
	minAllSQL   = 100
	minAllCSS   = 15
)

func TestGoCorpusLexes(t *testing.T) {
	// The Go surface measures 444 files, so a much smaller walk lost the
	// tree rather than the corpus (SPEC §1.3, #1133).
	lexCorpus(t, Go{}, inScopeFiles(t, ".go"), minGoFiles)
}

func TestSQLCorpusLexes(t *testing.T) {
	// SPEC §5.2 counts 106 `.sql` files across the tree. §1.4 then puts
	// `db/migrations` out of scope, which leaves 39 in `db/queries` (#1140).
	lexCorpus(t, SQL{}, inScopeFiles(t, ".sql"), minSQLFiles)
}

func TestSQLLexesEveryTrackedFile(t *testing.T) {
	// The goose markers sit in `db/migrations`, which no sweep edits, so the
	// tokenizer would otherwise meet `-- +goose` in a fixture alone (#1140).
	lexCorpus(t, SQL{}, walk(t, ".sql", false), minAllSQL)
}

func TestCSSLexesEveryTrackedFile(t *testing.T) {
	// `prototypes/` is out of the sweep, and SPEC §5.2 still measures its 7
	// files, so the tokenizer answers for all 18 (#1140).
	lexCorpus(t, CSS{}, walk(t, ".css", false), minAllCSS)
}

func TestCSSCorpusLexes(t *testing.T) {
	// SPEC §5.2 counts 18 `.css` files. §1.4 puts `prototypes/` out of scope,
	// which leaves 11 (#1140).
	lexCorpus(t, CSS{}, inScopeFiles(t, ".css"), minCSSFiles)
}

func lexCorpus(t *testing.T, lexer Lexer, files []string, min int) {
	t.Helper()
	if len(files) < min {
		t.Fatalf("the walk found %d files, want at least %d", len(files), min)
	}
	for _, rel := range files {
		src, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if _, err := lexer.Lex(src); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
}

func inScopeFiles(t *testing.T, ext string) []string {
	t.Helper()
	return walk(t, ext, true)
}

func walk(t *testing.T, ext string, inScopeOnly bool) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(repoRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != repoRoot && (strings.HasPrefix(name, ".") || name == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ext) {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !inScopeOnly || scope.Classify(rel) == scope.InScope {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return files
}
