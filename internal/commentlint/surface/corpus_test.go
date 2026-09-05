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
	minGoFiles   = 400
	minSQLFiles  = 35
	minCSSFiles  = 10
	minJSFiles   = 130
	minJSXFiles  = 135
	minTmplFiles = 20
	minAllSQL    = 100
	minAllCSS    = 15

	minCrossChecked = 15
)

func TestGoCorpusLexes(t *testing.T) {
	// The Go surface measures 444 files, so a smaller walk lost the tree, not the corpus (§1.3).
	lexCorpus(t, Go{}, inScopeFiles(t, ".go"), minGoFiles)
}

func TestSQLCorpusLexes(t *testing.T) {
	// §5.2 counts 106 `.sql` files, and §1.4 drops `db/migrations`, leaving 39 in queries (#1140).
	lexCorpus(t, SQL{}, inScopeFiles(t, ".sql"), minSQLFiles)
}

func TestSQLLexesEveryTrackedFile(t *testing.T) {
	// No sweep edits `db/migrations`, so `-- +goose` would otherwise reach a fixture alone (#1140).
	lexCorpus(t, SQL{}, walk(t, ".sql", false), minAllSQL)
}

func TestCSSLexesEveryTrackedFile(t *testing.T) {
	// `prototypes/` is out of the sweep, and §5.2 still measures its 7 files, so all 18 lex (#1140).
	lexCorpus(t, CSS{}, walk(t, ".css", false), minAllCSS)
}

func TestCSSCorpusLexes(t *testing.T) {
	// §5.2 counts 18 `.css` files, and §1.4 drops `prototypes/`, which leaves 11 (#1140).
	lexCorpus(t, CSS{}, inScopeFiles(t, ".css"), minCSSFiles)
}

func TestJSCorpusLexes(t *testing.T) {
	// §5.2 counts 135 `.mjs`, `.ts` and `.d.ts` files, 109 of them declaration files (#1141).
	files := append(inScopeFiles(t, ".mjs"), inScopeFiles(t, ".ts")...)
	lexCorpus(t, JS{}, files, minJSFiles)
}

func TestJSXCorpusLexes(t *testing.T) {
	// §5.2 counts 141 `.jsx` files and calls esbuild a fixed point on all of them (#1141).
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

func TestTmplCorpusLexes(t *testing.T) {
	// SPEC §5.2 counts 24 `.tmpl` files and 42 comments across them (#1142).
	lexCorpus(t, Tmpl{}, inScopeFiles(t, ".tmpl"), minTmplFiles)
}

func TestTmplByteRangeDeleteHoldsAcrossTheCorpus(t *testing.T) {
	// §5.4 measured the byte-range delete at 24 of 24 and the whole-line delete at 0 of 24 (#1142).
	measured := 0
	for _, rel := range corpusGuard(t, inScopeFiles(t, ".tmpl"), minTmplFiles) {
		src, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		base, err := Tmpl{}.Lex(src)
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if len(base.Blocks) == 0 {
			// The stage-D3 sweep emptied most templates, so a comment-free one measures nothing (#1236).
			continue
		}
		measured++
		all := append(append([]Block(nil), base.Blocks...), base.Trailing...)
		cut, err := Tmpl{}.Lex(TmplCut(src, all))
		if err != nil {
			t.Errorf("%s: the byte-range delete broke the parse: %v", rel, err)
			continue
		}
		if !sameSkeleton(base.Skeleton, cut.Skeleton) {
			t.Errorf("%s: the byte-range delete moved the skeleton", rel)
		}
		lines, err := Tmpl{}.Lex([]byte(cutWholeLines(string(src), base.Blocks)))
		if err == nil && sameSkeleton(base.Skeleton, lines.Skeleton) {
			t.Errorf("%s: the whole-line delete held, and §5.4 measured it failing", rel)
		}
	}
	if measured == 0 {
		t.Fatal("no .tmpl file carried a comment, so the byte-range delete went unmeasured")
	}
}

func TestJSCommentRangesAgreeWithEsbuild(t *testing.T) {
	// §5.5 records the circularity risk, and esbuild reads the comments apart from our lexer (#1141).
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
		// esbuild erases a `.d.ts` file to the empty string, so its canonical form proves nothing (§5.3).
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

func blankComments(src []byte, res Result) []byte {
	// A deletion would glue the neighbouring tokens, so each range fills with spaces (SPEC §5.1).
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
