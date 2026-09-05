package rule

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/commentlint/surface"
)

const repoRoot = "../../.."

func lintSource(t *testing.T, name, src string) []Finding {
	t.Helper()
	lexer, err := surface.For(name)
	if err != nil {
		t.Fatalf("For(%q): %v", name, err)
	}
	res, err := lexer.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex(%q): %v", name, err)
	}
	return Lint(res, false)
}

func ruleIDs(found []Finding) []string {
	out := make([]string, 0, len(found))
	for _, f := range found {
		out = append(out, f.Rule)
	}
	return out
}

func TestDTSFieldProseIsNotFlagged(t *testing.T) {
	// §4.3 keeps `.d.ts` field prose, and §7.7 condition 2 wants zero flags on the same file (#1406).
	cases := []struct {
		name string
		file string
		src  string
		want []string
	}{
		{
			name: "a field doc on its own line",
			file: "design-system/components/display/Widget.d.ts",
			src:  "export interface WidgetProps {\n  /** Default 140x36 */\n  width?: number;\n}\n",
			want: nil,
		},
		{
			name: "a field doc inline in a one-line interface body",
			file: "design-system/components/display/Widget.d.ts",
			src:  "export interface Column { key: string; /** Can't be hidden */ locked?: boolean; }\n",
			want: nil,
		},
		{
			name: "a required field doc",
			file: "design-system/components/display/Widget.d.ts",
			src:  "export interface WidgetProps {\n  /** Keys currently shown */\n  visible: string[];\n}\n",
			want: nil,
		},
		{
			name: "a one-liner above an interface is not field prose",
			file: "design-system/components/display/Widget.d.ts",
			src:  "/** @startingPoint console shell */\nexport interface WidgetProps {\n  width?: number;\n}\n",
			want: []string{"short-label"},
		},
		{
			name: "a one-liner above a top-level declaration is not field prose",
			file: "design-system/components/display/Widget.d.ts",
			src:  "/** The console gutter */\nexport declare const gutter: number;\n",
			want: []string{"short-label"},
		},
		{
			name: "the carve-out does not reach an implementation file",
			file: "docs-site/src/pipeline/toc.ts",
			src:  "export interface Entry {\n  /** Default 140x36 */\n  width?: number;\n}\n",
			want: []string{"short-label"},
		},
		{
			name: "a ratchet rule still reads a field doc",
			file: "design-system/components/display/Widget.d.ts",
			src:  "export interface WidgetProps {\n  /** TODO widen this */\n  width?: number;\n}\n",
			want: []string{"todo-marker"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ruleIDs(lintSource(t, c.file, c.src))
			if len(got) != len(c.want) {
				t.Fatalf("flags %v, want %v", got, c.want)
			}
			for i, r := range got {
				if r != c.want[i] {
					t.Fatalf("flags %v, want %v", got, c.want)
				}
			}
		})
	}
}

func TestSparklineFieldProseIsNotFlagged(t *testing.T) {
	// Appendix A row 15 is `Sparkline.d.ts:7` with verdict Keep, and line 4 is the same shape (#1406).
	rel := "design-system/components/display/Sparkline.d.ts"
	src, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	found := lintSource(t, rel, string(src))
	if len(found) != 0 {
		t.Fatalf("%s flags %+v, want none", rel, found)
	}
}

func TestDTSFieldProseClassifies(t *testing.T) {
	src := "export interface WidgetProps {\n  /** Default 140x36 */\n  width?: number;\n}\n"
	lexer, err := surface.For("design-system/components/display/Widget.d.ts")
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	res, err := lexer.Lex([]byte(src))
	if err != nil {
		t.Fatalf("Lex: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(res.Blocks))
	}
	if got := Classify(res.Blocks[0]); got != DTSFieldProse {
		t.Errorf("class is %s, want %s", got, DTSFieldProse)
	}
	if InDeleteSet(DTSFieldProse) {
		t.Errorf("%s is in the delete set, and §4.3 keeps it", DTSFieldProse)
	}
}

const minDTSFieldDocs = 150

func TestDTSCorpusReportsNoShortLabel(t *testing.T) {
	// A carve-out that goes dead reads as a pass here, so the count is asserted beside the flags.
	root := filepath.Join(repoRoot, "design-system", "components")
	docs := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".d.ts") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repoRoot, p)
		if err != nil {
			return err
		}
		lexer, err := surface.For(rel)
		if err != nil {
			return err
		}
		res, err := lexer.Lex(src)
		if err != nil {
			return err
		}
		blocks := append(append([]surface.Block{}, res.Blocks...), res.Trailing...)
		for _, b := range blocks {
			if Classify(b) == DTSFieldProse {
				docs++
			}
		}
		for _, f := range Lint(res, false) {
			t.Errorf("%s:%d flags %s", rel, f.Line, f.Rule)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	// The corpus held 284 field docs when the carve-out landed, 168 of them one-liners (#1406).
	if docs < minDTSFieldDocs {
		t.Errorf("the carve-out reached %d field docs, want at least %d", docs, minDTSFieldDocs)
	}
}
