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

const minGoFiles = 400

func TestGoCorpusLexes(t *testing.T) {
	files := inScopeGoFiles(t)
	// The Go surface measures 444 files, so a much smaller walk lost the
	// tree rather than the corpus (SPEC §1.3, #1133).
	if len(files) < minGoFiles {
		t.Fatalf("the walk found %d in-scope Go files, want at least %d", len(files), minGoFiles)
	}
	for _, rel := range files {
		src, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		if _, err := (Go{}).Lex(src); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
}

func inScopeGoFiles(t *testing.T) []string {
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
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if scope.Classify(rel) == scope.InScope {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return files
}
