package scope

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		path string
		want Verdict
	}{
		{"cmd/web/main.go", InScope},
		{"cmd/web/main_test.go", InScope},
		{"db/migrations/00100_init.go", InScope},
		{"db/queries/scans.sql", InScope},
		{"design-system/tokens/core.css", InScope},
		{"design-system/templates/shell.tmpl", InScope},
		{"docs-site/scripts/doclint.mjs", InScope},
		{"design-system/components/button.d.ts", InScope},
		{"docs-site/src/components/Nav.jsx", InScope},
		{"internal/db/scans.sql.go", OutOfScope},
		{"db/migrations/00100_init.sql", OutOfScope},
		{"prototypes/inbox/app.jsx", OutOfScope},
		{"docs-site/node_modules/pkg/index.mjs", OutOfScope},
		{"docs/spec/comment-policy.md", OutOfScope},
		{".github/workflows/ci.yml", OutOfScope},
		{"go.mod", OutOfScope},
		{"prototypes/inbox/index.html", Refused},
		{"docs-site/src/pages/index.astro", Refused},
		{"design-system/preview/index.html", Refused},
		{"./cmd/web/main.go", InScope},
		{"cmd\\web\\main.go", InScope},
	}

	for _, c := range cases {
		if got := Classify(c.path); got != c.want {
			t.Errorf("Classify(%q) = %d, want %d", c.path, got, c.want)
		}
	}
}
