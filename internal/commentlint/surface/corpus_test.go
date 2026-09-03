package surface

import (
	"bytes"
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
	minJSFiles  = 130
	minJSXFiles = 135
	minAllSQL   = 100
	minAllCSS   = 15

	// A cross-check that reaches no file passes for the wrong reason, so the
	// vacuous canonical forms are counted out and a floor holds (SPEC §5.5).
	minCrossChecked = 15
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

func TestJSCorpusLexes(t *testing.T) {
	// SPEC §5.2 counts 135 `.mjs`, `.ts` and `.d.ts` files, 109 of them
	// declaration files (#1141).
	files := append(inScopeFiles(t, ".mjs"), inScopeFiles(t, ".ts")...)
	lexCorpus(t, JS{}, files, minJSFiles)
}

func TestJSXCorpusLexes(t *testing.T) {
	// SPEC §5.2 counts 141 `.jsx` files and calls esbuild a fixed point on all
	// of them (#1141).
	esbuildOrSkip(t)
	for _, rel := range corpusGuard(t, inScopeFiles(t, ".jsx"), minJSXFiles) {
		src, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if _, err := (JSX{Path: filepath.Join(repoRoot, rel)}).Lex(src); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
}

func TestJSCommentRangesAgreeWithEsbuild(t *testing.T) {
	// §5.5 records the circularity risk: one hand lexer both finds the
	// comments and builds the skeleton. esbuild is the independent reader, so
	// blanking every range the lexer calls a comment must leave its canonical
	// form untouched (#1141).
	esbuildOrSkip(t)
	checked := 0
	for _, rel := range append(inScopeFiles(t, ".mjs"), inScopeFiles(t, ".ts")...) {
		path := filepath.Join(repoRoot, rel)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		loader := "ts"
		if strings.HasSuffix(rel, ".mjs") {
			loader = "js"
		}
		before, err := esbuildCanonical(path, src, loader)
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		// esbuild erases a `.d.ts` file to the empty string, so its canonical
		// form proves nothing and the file is counted out (SPEC §5.3).
		if len(bytes.TrimSpace(before)) == 0 {
			continue
		}
		res, err := JS{}.Lex(src)
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		after, err := esbuildCanonical(path, blankComments(src, res), loader)
		if err != nil {
			t.Errorf("%s: blanking the comments broke the parse: %v", rel, err)
			continue
		}
		if !bytes.Equal(before, after) {
			t.Errorf("%s: a range the lexer calls a comment carries code", rel)
		}
		checked++
	}
	if checked < minCrossChecked {
		t.Errorf("the cross-check reached %d file(s), want at least %d", checked, minCrossChecked)
	}
}

// blankComments overwrites every comment range with spaces. A deletion would
// glue the neighbouring tokens, which is a source change of its own (§5.1).
func blankComments(src []byte, res Result) []byte {
	out := append([]byte(nil), src...)
	for _, b := range append(append([]Block(nil), res.Blocks...), res.Trailing...) {
		for i := b.Start; i < b.End && i < len(out); i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	return out
}

func corpusGuard(t *testing.T, files []string, min int) []string {
	t.Helper()
	if len(files) < min {
		t.Fatalf("the walk found %d files, want at least %d", len(files), min)
	}
	return files
}

func lexCorpus(t *testing.T, lexer Lexer, files []string, min int) {
	t.Helper()
	for _, rel := range corpusGuard(t, files, min) {
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
