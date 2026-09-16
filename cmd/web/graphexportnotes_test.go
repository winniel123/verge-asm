package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
)

func TestGraphExportSerialiserReadsEveryCalloutThePageDrew(t *testing.T) {
	raw, err := fs.ReadFile(designfs.FS, "templates/graph.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	tmplSrc := string(raw)

	// These read the export's source; its raster is measured in the docs-site lane (#2212).
	for _, want := range []string{
		`document.querySelectorAll(".gr-main .gr-callout")`,
		`exportNotes()`,
		`clone.appendChild(tn)`,
	} {
		if !strings.Contains(tmplSrc, want) {
			t.Errorf("graph.tmpl's export serialiser missing %q, so a note the page drew can leave the image", want)
		}
	}

	// Every note the page can draw is a .gr-callout, so the one selector must reach all three.
	for _, gate := range []string{".Graph.ScopesFailed", ".Graph.SignalsFailed", ".Graph.Capped"} {
		i := strings.Index(tmplSrc, "{{if "+gate+"}}")
		if i < 0 {
			t.Errorf("graph.tmpl draws no note for %s", gate)
			continue
		}
		if !strings.HasPrefix(tmplSrc[i:], "{{if "+gate+`}}<div class="gr-callout">`) {
			t.Errorf("the %s note is not a .gr-callout, so the export's note collection misses it", gate)
		}
	}
}

func TestGraphPageShipsTheExportNoteBandWithItsCapStatement(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		f.addResolution(t, admin.ID, fmt.Sprintf("n%02d.example.com", i), "dns", obsClock,
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, `class="gr-callout"`) || !strings.Contains(page, "Showing 20 of 25 names.") {
		t.Fatalf("graph page stated no cap, so this test proves nothing about the export; body: %s", page)
	}
	if !strings.Contains(page, `document.querySelectorAll(".gr-main .gr-callout")`) {
		t.Errorf("the page that states a cap ships no export serialiser that reads it; body: %s", page)
	}
}

func TestGraphExportLaysTheNoteBandOutAtTheScaleItRastersAt(t *testing.T) {
	src := graphTmplSource(t)

	// The band's type is sized PX / os, which is what makes the scale it is laid out
	// at decide what the note rasterizes at (#2186).
	layout := jsBlockAfter(t, src, "function layoutNotes(notes, plotW, os) {")
	if !strings.Contains(layout, "PX / os") {
		t.Fatalf("layoutNotes no longer sizes the note band's type from its scale argument, so this test proves nothing about #2186")
	}

	band := jsBlockAfter(t, src, "if (notes.length) {")
	calls := strings.Count(band, "layoutNotes(")
	if calls == 0 {
		t.Fatalf("the export lays no note band out; block: %s", band)
	}

	head := strings.Index(band, "for (")
	if head < 0 {
		t.Fatalf("the export lays the note band out once, from the pre-band scale, so the band's own height leaves its type rasterizing below the size it was measured for (#2186); block: %s", band)
	}
	loop := jsBlockAfter(t, band[head:], "for (")

	if got := strings.Count(loop, "layoutNotes("); got != calls {
		t.Errorf("%d of the export's %d layoutNotes calls sit outside the settling loop, so that band keeps whatever scale it was laid out at (#2186)", calls-got, calls)
	}
	if !regexp.MustCompile(`layoutNotes\([^()]*\bos\b[^()]*\)`).MatchString(loop) {
		t.Errorf("the settling loop hands layoutNotes no scale binding, so the band's type is sized from some other value (#2186); loop: %s", loop)
	}
	if !regexp.MustCompile(`\bos = `).MatchString(loop) || !strings.Contains(loop, "MAX_AREA") {
		t.Errorf("the settling loop never recomputes the scale from the banded frame, so it cannot reach the scale the export rasterizes at (#2186); loop: %s", loop)
	}
	if !regexp.MustCompile(`if \([^;]*\bos\b[^;]*\bbreak\b`).MatchString(loop) {
		t.Errorf("the settling loop stops on its pass count rather than on the scale, so it can end one step past the band it draws (#2186); loop: %s", loop)
	}

	for _, grow := range []string{`\bby\s*[-+]=`, `\bbh\s*[-+]=`, `\bby\s*=\s*by\b`, `\bbh\s*=\s*bh\b`} {
		if regexp.MustCompile(grow).MatchString(loop) {
			t.Errorf("the settling loop grows the frame from its own last value (%s), so a second pass stacks two bands (#2186); loop: %s", grow, loop)
		}
	}
	for _, restate := range []string{`\bby\s*=\s*[^;]*\bband\.height`, `\bbh\s*=\s*[^;]*\bband\.height`} {
		if !regexp.MustCompile(restate).MatchString(loop) {
			t.Errorf("the settling loop never restates the frame (%s), so the scale it settles on is not the banded frame's (#2186); loop: %s", restate, loop)
		}
	}
}

func graphTmplSource(t *testing.T) string {
	t.Helper()
	raw, err := fs.ReadFile(designfs.FS, "templates/graph.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func jsBlockAfter(t *testing.T, src, mark string) string {
	t.Helper()
	i := strings.Index(src, mark)
	if i < 0 {
		t.Fatalf("graph.tmpl holds no %q", mark)
	}
	open := strings.Index(src[i:], "{")
	if open < 0 {
		t.Fatalf("%q opens no block in graph.tmpl", mark)
	}
	open += i
	depth, quote := 0, byte(0)
	for j := open; j < len(src); j++ {
		c := src[j]
		if quote != 0 {
			switch {
			case c == '\\':
				j++
			case c == quote:
				quote = 0
			}
			continue
		}
		// An apostrophe in prose reads as a string quote, and every export comment
		// carries one, so step over a comment rather than lex it.
		if c == '/' && j+1 < len(src) && src[j+1] == '/' {
			nl := strings.IndexByte(src[j:], '\n')
			if nl < 0 {
				break
			}
			j += nl
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '{':
			depth++
		case '}':
			if depth--; depth == 0 {
				return src[open+1 : j]
			}
		}
	}
	t.Fatalf("%q opens a block graph.tmpl never closes", mark)
	return ""
}
