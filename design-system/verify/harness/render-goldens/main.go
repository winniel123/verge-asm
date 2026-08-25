// Command render-goldens produces the static golden HTML for the /inventory
// pixel-parity harness (ticket #526, P4.0 Inventory pilot).
//
// It parses the design-owned, frozen inventory.tmpl (design-system/templates/
// inventory.tmpl, embedded read-only via the designfs package) with STUB
// "head"/"chrome"/"foot" definitions, feeds it the design fixture
// (design-system/fixtures/fixtures.json) in authored array order, and executes
// the "inventory" template to a single deterministic HTML file.
//
// The stub "head" inlines the design token vocabulary exactly as the app does
// (cmd/web/templates_inventory.go loadDesignTokens): fs.Glob(designfs.FS,
// "tokens/*.css") -> sort.Strings -> read each -> strings.Join(parts,"\n"),
// wrapped in a <style data-design-tokens> block. No pageCSS, no localStorage
// theme script, no viewport meta: the capture harness sets the theme and the
// viewport deterministically, so the golden carries only the token cascade the
// frozen tmpl styles against. The stub "chrome" is empty (cropped out of the
// `main` screenshot); the stub "foot" only closes the document.
//
// This file is repo-owned harness glue, NOT a design-owned artifact: it lives
// under design-system/verify/harness/, which the designfs embed globs (by
// extension) and CI gate G1 (which covers templates/tokens/fixtures/verify/*.json
// + goldens) do not sweep in. It only READS the frozen files.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// The template data shape mirrors the holes the frozen inventory.tmpl reads and
// the inventory{Group,Subject,Facet} structs cmd/web/inventory.go emits:
// .Groups[{Kind,Label,Subjects[{Key,Type,Link,Facets[{Label,Summary,IsGap,
// Since,Details[{Type,Data}]}]}]}] plus .HasData.
type detail struct {
	Type string
	Data string
}

type facet struct {
	Label   string
	Summary string
	IsGap   bool
	Since   string
	Details []detail
}

type subject struct {
	Key    string
	Type   string
	Link   string
	Facets []facet
}

type group struct {
	Kind     string
	Label    string
	Subjects []subject
}

type pageData struct {
	HasData bool
	Groups  []group
}

// fixtureFile is the on-disk shape of design-system/fixtures/fixtures.json. The
// JSON is snake_case; the template data shape above is the Go/tmpl PascalCase.
type fixtureFile struct {
	Inventory struct {
		Groups []struct {
			Kind     string `json:"kind"`
			Label    string `json:"label"`
			Subjects []struct {
				Key    string `json:"key"`
				Type   string `json:"type"`
				Link   string `json:"link"`
				Facets []struct {
					Label   string `json:"label"`
					Summary string `json:"summary"`
					IsGap   bool   `json:"is_gap"`
					Since   string `json:"since"`
					Details []struct {
						Type string `json:"type"`
						Data string `json:"data"`
					} `json:"details"`
				} `json:"facets"`
			} `json:"subjects"`
		} `json:"groups"`
	} `json:"inventory"`
}

func main() {
	screen := flag.String("screen", "inventory", "which screen to render: inventory | error")
	out := flag.String("out", "", "inventory: path to write the single golden HTML")
	outdir := flag.String("outdir", "", "error: directory to write one golden HTML per state (<state>.html)")
	// -body-flex is a DIAGNOSTIC-ONLY toggle (never used for the canonical golden):
	// it injects the app shell's body layout context (body{display:flex;
	// flex-direction:column;margin:0}) so the golden's <main> shrink-wraps to its
	// content width exactly as the candidate's does under pageCSS + .inv-main's
	// margin:0 auto. It exists only to isolate content parity from the base-style
	// width delta the canonical (no-pageCSS) stub surfaces. Not part of the spec's
	// frozen stub contract.
	bodyFlex := flag.Bool("body-flex", false, "diagnostic: add body{display:flex;flex-direction:column} so main shrink-wraps like the app")
	flag.Parse()

	switch *screen {
	case "inventory":
		if *out == "" {
			log.Fatal("render-goldens: -out is required for -screen inventory")
		}
		html, err := render(*bodyFlex)
		if err != nil {
			log.Fatalf("render-goldens: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(*out), 0o750); err != nil {
			log.Fatalf("render-goldens: mkdir: %v", err)
		}
		if err := os.WriteFile(*out, html, 0o600); err != nil {
			log.Fatalf("render-goldens: write %s: %v", *out, err)
		}
		log.Printf("render-goldens: wrote %s (%d bytes)", *out, len(html))
	case "error":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen error")
		}
		files, err := renderErrorStates(*bodyFlex)
		if err != nil {
			log.Fatalf("render-goldens: %v", err)
		}
		if err := os.MkdirAll(*outdir, 0o750); err != nil {
			log.Fatalf("render-goldens: mkdir: %v", err)
		}
		for _, f := range files {
			path := filepath.Join(*outdir, f.id+".html")
			if err := os.WriteFile(path, f.html, 0o600); err != nil {
				log.Fatalf("render-goldens: write %s: %v", path, err)
			}
			log.Printf("render-goldens: wrote %s (%d bytes)", path, len(f.html))
		}
	default:
		log.Fatalf("render-goldens: unknown -screen %q (want inventory | error)", *screen)
	}
}

// goldenHead builds the golden's <head>…<body> shell: the frozen font @import hoisted
// to its own leading <style>, the concatenated design tokens, and the minimal reset
// that reconciles the candidate's effective body context for the cropped `main`. It is
// the SAME shell for every screen (see the reconciliation notes below), so both goldens
// carry the identical token cascade + font load the app inlines for design-served pages.
//
//   - body{margin:0}                — app pageCSS sets it; base.css does not.
//   - body{display:block}           — the app neutralizes its legacy flex shell for
//     design-served pages via a gated `<style data-design-shell>` shim; block flow is
//     already the golden's default, so this is a no-op but stated for parity of intent.
//   - *{box-sizing:border-box}      — app pageCSS applies this global reset; the design
//     components are authored for border-box. base.css (inlined via tokens) does NOT set
//     box-sizing, so without this padded controls grow content-box and diverge.
//
// FONT LOAD SYMMETRY: typography.css carries the webfont `@import url(...)` as its leading
// rule, valid there. Once tokens are CONCATENATED it is no longer first, so per CSS spec it
// is INVALID and dropped — the golden would fall back to system fonts while the candidate
// (whose pageCSS puts the same @import first) loads real Instrument Sans / Geist Mono,
// diverging glyph metrics. So hoist that exact @import into its own leading <style>, so BOTH
// sides attempt the identical webfont load (deterministic whether or not the CDN resolves).
func goldenHead(bodyFlex bool) (template.HTML, error) {
	tokens, err := loadDesignTokens()
	if err != nil {
		return "", err
	}
	fontImport, err := leadingFontImport()
	if err != nil {
		return "", err
	}
	diag := ""
	if bodyFlex {
		diag = "<style>body{display:flex;flex-direction:column;margin:0}</style>"
	}
	headHTML := "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">" +
		"<style data-design-fonts>" + fontImport + "</style>" +
		"<style data-design-tokens>" + tokens + "</style>" +
		"<style data-golden-shell>*,*::before,*::after{box-sizing:border-box}body{margin:0}</style>" + diag + "</head><body>"
	// The head is composed by this harness from the embedded design tokens and the
	// frozen font @import — no user input reaches it, so it is safe to mark trusted.
	return template.HTML(headHTML), nil // #nosec G203 -- trusted design CSS/HTML composed by the harness from embedded artifacts, no user input
}

// newStubbedTemplate returns a template set whose "head"/"chrome"/"foot" are the golden
// stubs the design tmpls call: "head" inlines the composed shell (so the token CSS's
// single braces are never parsed as template text), "chrome" is empty (cropped out of the
// `main` screenshot), and "foot" only closes the document.
func newStubbedTemplate(head template.HTML) (*template.Template, error) {
	t := template.New("root").Funcs(template.FuncMap{
		"stubhead": func() template.HTML { return head },
	})
	return t.Parse(`{{define "head"}}{{stubhead}}{{end}}{{define "chrome"}}{{end}}{{define "foot"}}</body></html>{{end}}`)
}

func render(bodyFlex bool) ([]byte, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/inventory.tmpl"); err != nil {
		return nil, err
	}

	data, err := loadFixture()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "inventory", data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// errorGolden is one rendered error-state golden: its state id (states.json) and the
// static HTML the capture harness snapshots in golden mode.
type errorGolden struct {
	id   string
	html []byte
}

// renderErrorStates composes the six ErrorPage golden HTMLs from the frozen error.tmpl,
// one per states.json state. The per-state data map mirrors errors.go's handlers EXACTLY
// (Kind/Code/Subject/IncidentID/ActionLabel/ActionHref) so the cropped `main` is
// byte-identical to what the seeded server renders — the golden and the candidate are the
// same tmpl fed the same holes. The incident id and the missing-subject/run keys are read
// from fixtures.json (never hardcoded here) so a fixture change flows through. .Chrome is
// unset: goldens crop to `main`, so the chrome band is excluded (shell #22 gates it).
func renderErrorStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadErrorFixture()
	if err != nil {
		return nil, err
	}

	// Order mirrors states.json. 404/403/500 carry no ActionLabel/Href — the tmpl
	// defaults ("Back to dashboard" → "/") apply, exactly as renderError leaves them.
	states := []errorGolden{
		{id: "404", html: nil},
		{id: "403", html: nil},
		{id: "500", html: nil},
		{id: "missing-subject", html: nil},
		{id: "missing-run", html: nil},
		{id: "settings-forbidden", html: nil},
	}
	data := map[string]map[string]any{
		"404": {"Kind": "404"},
		"403": {"Kind": "403"},
		"500": {"Kind": "500", "IncidentID": fx.IncidentID},
		"missing-subject": {
			"Kind": "missing-subject", "Subject": fx.MissingSubject,
			"ActionLabel": "Back to inventory", "ActionHref": "/inventory",
		},
		"missing-run": {
			"Kind": "missing-run", "Subject": "run #" + fx.MissingRun,
			"ActionLabel": "Back to drift", "ActionHref": "/drift",
		},
		"settings-forbidden": {
			"Kind": "settings-forbidden", "Code": "403",
			"ActionLabel": "Back to dashboard", "ActionHref": "/",
		},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/error.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "error-page", data[st.id]); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// loadDesignTokens replicates cmd/web/templates_inventory.go's loadDesignTokens
// byte-for-byte: sorted tokens/*.css glob, read each, join with "\n". Keeping
// this algorithm identical is the whole point — the golden must carry the exact
// token cascade the app inlines for /inventory.
func loadDesignTokens() (string, error) {
	names, err := fs.Glob(designfs.FS, "tokens/*.css")
	if err != nil {
		return "", err
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		b, err := fs.ReadFile(designfs.FS, name)
		if err != nil {
			return "", err
		}
		parts = append(parts, string(b))
	}
	return strings.Join(parts, "\n"), nil
}

// leadingFontImport extracts the webfont `@import url(...);` statement from
// tokens/typography.css verbatim, so the golden can emit it as the first rule of
// its own stylesheet (where @import is valid). Extracting rather than hardcoding
// keeps the golden's font load pinned to whatever the design package ships.
func leadingFontImport() (string, error) {
	b, err := fs.ReadFile(designfs.FS, "tokens/typography.css")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@import") {
			return trimmed, nil // includes the trailing ';'
		}
	}
	return "", fmt.Errorf("no @import found in tokens/typography.css")
}

// errorFixture is the design-system/fixtures/fixtures.json → error slice: the
// deterministic 500 incident id and the keys the missing-subject/run states show.
// The golden reads them here so a fixture change flows through instead of being pinned
// twice; the repo side pins the same values in code (devfixtures.go) with a drift test.
type errorFixture struct {
	IncidentID     string
	MissingSubject string
	MissingRun     string
}

func loadErrorFixture() (errorFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return errorFixture{}, err
	}
	var ff struct {
		Error struct {
			IncidentID     string `json:"incident_id"`
			MissingSubject string `json:"missing_subject"`
			MissingRun     string `json:"missing_run"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return errorFixture{}, err
	}
	return errorFixture{
		IncidentID:     ff.Error.IncidentID,
		MissingSubject: ff.Error.MissingSubject,
		MissingRun:     ff.Error.MissingRun,
	}, nil
}

func loadFixture() (pageData, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return pageData{}, err
	}
	var ff fixtureFile
	if err := json.Unmarshal(raw, &ff); err != nil {
		return pageData{}, err
	}

	groups := make([]group, 0, len(ff.Inventory.Groups))
	for _, g := range ff.Inventory.Groups { // authored array order preserved
		subs := make([]subject, 0, len(g.Subjects))
		for _, s := range g.Subjects {
			facets := make([]facet, 0, len(s.Facets))
			for _, f := range s.Facets {
				details := make([]detail, 0, len(f.Details))
				for _, d := range f.Details {
					details = append(details, detail{Type: d.Type, Data: d.Data})
				}
				facets = append(facets, facet{
					Label:   f.Label,
					Summary: f.Summary,
					IsGap:   f.IsGap,
					Since:   f.Since,
					Details: details,
				})
			}
			subs = append(subs, subject{Key: s.Key, Type: s.Type, Link: s.Link, Facets: facets})
		}
		groups = append(groups, group{Kind: g.Kind, Label: g.Label, Subjects: subs})
	}
	return pageData{HasData: len(groups) > 0, Groups: groups}, nil
}
