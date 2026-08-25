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
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/qr"
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
	screen := flag.String("screen", "inventory", "which screen to render: inventory | error | profile | signin | setup | coverage | exposure | drift | rundetail | scope | signals | dashboard | asset | subjectdetail")
	out := flag.String("out", "", "inventory|drift: path to write the single golden HTML")
	outdir := flag.String("outdir", "", "error|profile|…: directory to write one golden HTML per state (<state>.html)")
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
	case "profile":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen profile")
		}
		files, err := renderProfileStates(*bodyFlex)
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
	case "signin":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen signin")
		}
		files, err := renderSigninStates(*bodyFlex)
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
	case "setup":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen setup")
		}
		files, err := renderSetupStates(*bodyFlex)
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
	case "coverage":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen coverage")
		}
		files, err := renderCoverageStates(*bodyFlex)
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
	case "exposure":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen exposure")
		}
		files, err := renderExposureStates(*bodyFlex)
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
	case "scope":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen scope")
		}
		files, err := renderScopeStates(*bodyFlex)
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
	case "rundetail":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen rundetail")
		}
		files, err := renderRunDetailStates(*bodyFlex)
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
	case "signals":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen signals")
		}
		files, err := renderSignalsStates(*bodyFlex)
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
	case "dashboard":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen dashboard")
		}
		files, err := renderDashboardStates(*bodyFlex)
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
	case "asset":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen asset")
		}
		files, err := renderAssetStates(*bodyFlex)
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
	case "subjectdetail":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen subjectdetail")
		}
		files, err := renderSubjectDetailStates(*bodyFlex)
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
	case "drift":
		if *out == "" {
			log.Fatal("render-goldens: -out is required for -screen drift")
		}
		html, err := renderDrift(*bodyFlex)
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
	default:
		log.Fatalf("render-goldens: unknown -screen %q (want inventory | error | profile | signin | setup | coverage | exposure | drift)", *screen)
	}
}

// runDetailFixture is the design-system/fixtures/fixtures.json → rundetail slice: the run header +
// Outcome figures (the #20a batch join, carried as strings), the four stages, the seven log lines
// (one warn, one error), the nullable degraded callout, the five params and the three vantages. The
// golden reads them here (never re-hardcoded) so a fixture change flows through; cmd/web/devfixtures.go
// pins the same values with a drift test (TestRunDetailFixtureMatchesPackage).
type runDetailFixture struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Scope       string `json:"scope"`
	Meta        string `json:"meta"`
	Active      bool   `json:"active"`
	Transitions string `json:"transitions"`
	NewSignals  string `json:"new_signals"`
	Stages      []struct {
		Num     int    `json:"num"`
		Title   string `json:"title"`
		Detail  string `json:"detail"`
		Done    bool   `json:"done"`
		Current bool   `json:"current"`
		Last    bool   `json:"last"`
	} `json:"stages"`
	Log []struct {
		Tag   string `json:"tag"`
		Level string `json:"level"`
		Text  string `json:"text"`
	} `json:"log"`
	Degraded *struct {
		Vantage string `json:"vantage"`
		Detail  string `json:"detail"`
	} `json:"degraded"`
	Params []struct {
		K string `json:"k"`
		V string `json:"v"`
	} `json:"params"`
	Vantages []struct {
		Name    string `json:"name"`
		Latency string `json:"latency"`
		Status  string `json:"status"`
	} `json:"vantages"`
}

func loadRunDetailFixture() (runDetailFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return runDetailFixture{}, err
	}
	var ff struct {
		RunDetail runDetailFixture `json:"rundetail"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return runDetailFixture{}, err
	}
	return ff.RunDetail, nil
}

// renderRunDetailStates composes the RunDetail golden HTML from the frozen rundetail.tmpl, for the
// one states.json rundetail state (default, /runs/1407). The data map mirrors runPage's
// runDetailFixtureData EXACTLY (the .Run holes the frozen tmpl reads): the header, the four done
// stages, the seven-line log (levels feeding the colored-text treatment #20e), the Outcome batch
// join (7 transitions · 3 new signals as strings), the nullable degraded callout, the five params
// and the three vantages — all in fixtures.json authored order — so the cropped `main` is
// byte-identical to what the seeded server renders (golden and candidate = same tmpl, same holes).
// Chrome is the empty stub (goldens crop to `main`).
func renderRunDetailStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadRunDetailFixture()
	if err != nil {
		return nil, err
	}

	stages := make([]map[string]any, 0, len(fx.Stages))
	for _, st := range fx.Stages {
		stages = append(stages, map[string]any{
			"Num": st.Num, "Title": st.Title, "Detail": st.Detail,
			"Done": st.Done, "Current": st.Current, "Last": st.Last,
		})
	}
	logLines := make([]map[string]any, 0, len(fx.Log))
	for _, l := range fx.Log {
		logLines = append(logLines, map[string]any{"Tag": l.Tag, "Level": l.Level, "Text": l.Text})
	}
	params := make([]map[string]any, 0, len(fx.Params))
	for _, p := range fx.Params {
		params = append(params, map[string]any{"K": p.K, "V": p.V})
	}
	vantages := make([]map[string]any, 0, len(fx.Vantages))
	for _, vt := range fx.Vantages {
		vantages = append(vantages, map[string]any{"Name": vt.Name, "Latency": vt.Latency, "Status": vt.Status})
	}
	run := map[string]any{
		"Title": fx.Title, "Status": fx.Status, "Scope": fx.Scope, "Meta": fx.Meta,
		"Transitions": fx.Transitions, "NewSignals": fx.NewSignals, "Active": fx.Active,
		"Stages": stages, "Log": logLines, "Params": params, "Vantages": vantages,
	}
	// Nullable degraded (#20): the frozen tmpl's {{with .Degraded}} renders the callout only when
	// present. A nil map value renders nothing, matching runDetailFixtureData's *runDegraded.
	if fx.Degraded != nil {
		run["Degraded"] = map[string]any{"Vantage": fx.Degraded.Vantage, "Detail": fx.Degraded.Detail}
	} else {
		run["Degraded"] = nil
	}

	data := map[string]any{
		"Title": "batch " + fx.Title, "NavActive": "drift", "DesignTokens": true,
		"Run": run,
	}

	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/rundetail.tmpl"); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "run", data); err != nil {
		return nil, err
	}
	return []errorGolden{{id: "default", html: buf.Bytes()}}, nil
}

// coverageFixture is the design-system/fixtures/fixtures.json coverage slice: the aperture meters
// (address counted/total, name census), the currency messages (with relative When + ISO tooltip),
// the gaps register, the unevaluable rules and the per-zone stale callouts. The golden reads them
// here (never re-hardcoded) so a fixture change flows through; cmd/web/devfixtures.go pins the same
// values with a drift test (TestCoverageFixtureMatchesPackage).
type coverageFixture struct {
	Meters []struct {
		Label   string `json:"label"`
		Counted int    `json:"counted"`
		Total   *int   `json:"total"`
		Unit    string `json:"unit"`
		Detail  string `json:"detail"`
	} `json:"meters"`
	Messages []struct {
		Kind    string `json:"kind"`
		Badge   string `json:"badge"`
		Bound   string `json:"bound"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
		When    string `json:"when"`
		ISO     string `json:"iso"`
	} `json:"messages"`
	Gaps []struct {
		Subject  string `json:"subject"`
		Gap      string `json:"gap"`
		Expected string `json:"expected"`
		Since    string `json:"since"`
	} `json:"gaps"`
	Unevaluable []struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
		Why     string `json:"why"`
	} `json:"unevaluable"`
	StaleZones []struct {
		Zone string `json:"zone"`
		Age  string `json:"age"`
	} `json:"stale_zones"`
}

func loadCoverageFixture() (coverageFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return coverageFixture{}, err
	}
	var ff struct {
		Coverage coverageFixture `json:"coverage"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return coverageFixture{}, err
	}
	return ff.Coverage, nil
}

// coveragePct replicates cmd/web/cold.go coveragePct byte-for-byte: the ADDRESS-scope meter fill
// (counted/total, rounded to the nearest whole percent, clamped 0–100). Keeping this arithmetic
// identical is the point — fixtures.json carries counted/total but no precomputed pct, so the
// golden must compute the same fill the seeded candidate does.
func coveragePct(counted, total int) int {
	if total <= 0 {
		return 0
	}
	p := int(math.Round(float64(counted) / float64(total) * 100))
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// renderCoverageStates composes the two Coverage golden HTMLs from the frozen coverage.tmpl, one
// per states.json coverage state (default, empty). The "default" data map mirrors coveragePage
// coverageFixtureData EXACTLY (the holes the frozen tmpl reads) — the aperture meters with the
// #19c address counted/total + computed Pct and the name census, the four relative-time messages,
// the gaps, the unevaluable rules and the per-zone stale callout, all in fixtures.json authored
// order — so the cropped `main` is byte-identical to what the seeded server renders. The "empty"
// state renders every region empty (the tmpl draws its own empty states, no stale callout),
// mirroring the /dev/seed/empty-authed capture. Chrome is the empty stub (goldens crop to `main`).
func renderCoverageStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadCoverageFixture()
	if err != nil {
		return nil, err
	}

	meters := make([]map[string]any, 0, len(fx.Meters))
	for _, m := range fx.Meters {
		mv := map[string]any{
			"Label": m.Label, "Counted": strconv.Itoa(m.Counted), "Unit": m.Unit, "Detail": m.Detail,
		}
		if m.Total != nil {
			mv["Total"] = strconv.Itoa(*m.Total)
			mv["Pct"] = coveragePct(m.Counted, *m.Total)
		}
		meters = append(meters, mv)
	}
	messages := make([]map[string]any, 0, len(fx.Messages))
	for _, m := range fx.Messages {
		messages = append(messages, map[string]any{
			"Kind": m.Kind, "Badge": m.Badge, "Bound": m.Bound, "Subject": m.Subject, "Text": m.Text, "When": m.When, "ISO": m.ISO,
		})
	}
	gaps := make([]map[string]any, 0, len(fx.Gaps))
	for _, g := range fx.Gaps {
		gaps = append(gaps, map[string]any{"Subject": g.Subject, "Gap": g.Gap, "Expected": g.Expected, "Since": g.Since})
	}
	unevaluable := make([]map[string]any, 0, len(fx.Unevaluable))
	for _, u := range fx.Unevaluable {
		unevaluable = append(unevaluable, map[string]any{"ID": u.ID, "Version": strconv.Itoa(u.Version), "Why": u.Why})
	}
	stale := make([]map[string]any, 0, len(fx.StaleZones))
	for _, z := range fx.StaleZones {
		stale = append(stale, map[string]any{"Zone": z.Zone, "Age": z.Age})
	}

	type cstate struct {
		id   string
		data map[string]any
	}
	states := []cstate{
		{"default", map[string]any{
			"Title": "Coverage", "NavActive": "coverage", "DesignTokens": true,
			"Meters": meters, "Messages": messages, "Gaps": gaps, "Unevaluable": unevaluable, "StaleZones": stale,
		}},
		{"empty", map[string]any{
			"Title": "Coverage", "NavActive": "coverage", "DesignTokens": true,
			"Meters": []map[string]any{}, "Messages": []map[string]any{}, "Gaps": []map[string]any{},
			"Unevaluable": []map[string]any{}, "StaleZones": []map[string]any{},
		}},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/coverage.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "coverage", st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// exposureFixture is the design-system/fixtures/fixtures.json exposure slice: the summary band
// (exposed with a vs-last-batch delta, firewalled, not reached), the withheld variant token, and
// the both-legs board rows (asset, ":port transport", the internal + internet leg display states,
// and the "since"). The golden reads them here (never re-hardcoded) so a fixture change flows
// through; cmd/web/devfixtures.go pins the same values with a drift test
// (TestExposureFixtureMatchesPackage).
type exposureFixture struct {
	Exposed         int    `json:"exposed"`
	HasDeltas       bool   `json:"has_deltas"`
	ExposedDelta    int    `json:"exposed_delta"`
	Firewalled      int    `json:"firewalled"`
	NotReached      int    `json:"not_reached"`
	WithheldVariant string `json:"withheld_variant"`
	Rows            []struct {
		Asset    string `json:"asset"`
		Svc      string `json:"svc"`
		Internal string `json:"internal"`
		Internet string `json:"internet"`
		Since    string `json:"since"`
	} `json:"rows"`
}

func loadExposureFixture() (exposureFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return exposureFixture{}, err
	}
	var ff struct {
		Exposure exposureFixture `json:"exposure"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return exposureFixture{}, err
	}
	return ff.Exposure, nil
}

// renderExposureStates composes the two Exposure golden HTMLs from the frozen exposure.tmpl, one
// per states.json exposure state (default, withheld). The "default" data map mirrors exposurePage
// exposureFixtureData EXACTLY (the holes the frozen tmpl reads) — the six board rows in fixtures.json
// authored order, the summary counts and the +2 exposed delta fed as .ExposedDelta.Change (the tmpl's
// signDelta formats it) — so the cropped `main` is byte-identical to what the seeded server renders.
// The "withheld" state sets only .Withheld (the no-internet-vantage branch), so the tmpl draws its
// WITHHELD card. Chrome is the empty stub (goldens crop to `main`).
func renderExposureStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadExposureFixture()
	if err != nil {
		return nil, err
	}

	rows := make([]map[string]any, 0, len(fx.Rows))
	for _, r := range fx.Rows {
		rows = append(rows, map[string]any{
			"Asset": r.Asset, "Svc": r.Svc, "Internal": r.Internal, "Internet": r.Internet, "Since": r.Since,
		})
	}

	defaultData := map[string]any{
		"Title": "Exposure", "NavActive": "exposure", "DesignTokens": true,
		"Withheld": false, "Rows": rows,
		"Exposed": fx.Exposed, "Firewalled": fx.Firewalled, "NotReached": fx.NotReached,
	}
	if fx.HasDeltas {
		defaultData["HasDeltas"] = true
		defaultData["ExposedDelta"] = map[string]any{"Change": fx.ExposedDelta}
	}

	type estate struct {
		id   string
		data map[string]any
	}
	states := []estate{
		{"default", defaultData},
		{"withheld", map[string]any{
			"Title": "Exposure", "NavActive": "exposure", "DesignTokens": true, "Withheld": true,
		}},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/exposure.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "exposure", st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// scopeFixture is the design-system/fixtures/fixtures.json scope slice: the seeds, the refusal +
// exclusion-preview + org-search fixtures, custody, zone, name tree, coverage messages, proposals
// and exclusions. The golden reads them here (never re-hardcoded) so a fixture change flows
// through; cmd/web/devfixtures.go pins the same values with a drift test
// (TestScopeFixtureMatchesPackage).
type scopeFixture struct {
	AddressCap int `json:"address_cap"`
	Seeds      []struct {
		ID        string `json:"id"`
		Anchor    string `json:"anchor"`
		Scope     string `json:"scope"`
		IsAddress bool   `json:"is_address"`
	} `json:"seeds"`
	RefusalFixture struct {
		PostValue string `json:"post_value"`
		Input     string `json:"input"`
		Reason    string `json:"reason"`
		Reachable string `json:"reachable"`
		FormError string `json:"form_error"`
	} `json:"refusal_fixture"`
	CustodyScopes []struct {
		ID               string `json:"id"`
		Scope            string `json:"scope"`
		CustodyExtension bool   `json:"custody_extension"`
		Census           int    `json:"census"`
	} `json:"custody_scopes"`
	ZoneScopes []struct {
		ID            string `json:"id"`
		Domain        string `json:"domain"`
		HasFile       bool   `json:"has_file"`
		SuppliedAt    string `json:"supplied_at"`
		IntervalLabel string `json:"interval_label"`
		AgingLabel    string `json:"aging_label"`
	} `json:"zone_scopes"`
	ZoneIntervalDays int `json:"zone_interval_days"`
	NameTree         []struct {
		Label    string `json:"label"`
		Count    int    `json:"count"`
		Sev      string `json:"sev"`
		Children []struct {
			Label string `json:"label"`
			Sev   string `json:"sev"`
		} `json:"children"`
	} `json:"name_tree"`
	CoverageMsgs []struct {
		Kind    string `json:"kind"`
		Badge   string `json:"badge"`
		Bound   string `json:"bound"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
		When    string `json:"when"`
		ISO     string `json:"iso"`
	} `json:"coverage_msgs"`
	Proposals []struct {
		ID     string `json:"id"`
		Value  string `json:"value"`
		Kind   string `json:"kind"`
		Source string `json:"source"`
	} `json:"proposals"`
	Exclusions []struct {
		ID    string `json:"id"`
		Kind  string `json:"kind"`
		Value string `json:"value"`
	} `json:"exclusions"`
	ExclusionPreviewFixture struct {
		PostKind  string `json:"post_kind"`
		PostValue string `json:"post_value"`
		Fires     bool   `json:"fires"`
		Headline  string `json:"headline"`
		Loss      string `json:"loss"`
	} `json:"exclusion_preview_fixture"`
}

func loadScopeFixture() (scopeFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return scopeFixture{}, err
	}
	var ff struct {
		Scope scopeFixture `json:"scope"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return scopeFixture{}, err
	}
	return ff.Scope, nil
}

// renderScopeStates composes the three Scope golden HTMLs from the frozen scope.tmpl, one per
// states.json scope state (default, refusal, exclusion-preview). Every state's data map mirrors
// seedsPage's scopeFixtureData EXACTLY (the holes the frozen tmpl reads) — the pinned fixtures.json
// scope slice in authored order — so the cropped `main` is byte-identical to what the seeded server
// renders. The "refusal" state adds the RefusalCallout + FormError (read from refusal_fixture, the
// same values declareSeed derives) with the posted value echoed in FormScope; the "exclusion-preview"
// state adds the firing Preview receipt (read from exclusion_preview_fixture) with the typed kind +
// value echoed. Chrome is the empty stub (goldens crop to `main`).
func renderScopeStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadScopeFixture()
	if err != nil {
		return nil, err
	}

	seeds := make([]map[string]any, 0, len(fx.Seeds))
	for _, s := range fx.Seeds {
		seeds = append(seeds, map[string]any{"ID": s.ID, "Anchor": s.Anchor, "Scope": s.Scope, "IsAddress": s.IsAddress})
	}
	custody := make([]map[string]any, 0, len(fx.CustodyScopes))
	for _, c := range fx.CustodyScopes {
		custody = append(custody, map[string]any{"ID": c.ID, "Scope": c.Scope, "CustodyExtension": c.CustodyExtension, "Census": c.Census})
	}
	zones := make([]map[string]any, 0, len(fx.ZoneScopes))
	for _, z := range fx.ZoneScopes {
		zones = append(zones, map[string]any{"ID": z.ID, "Domain": z.Domain, "HasFile": z.HasFile, "SuppliedAt": z.SuppliedAt, "IntervalLabel": z.IntervalLabel, "AgingLabel": z.AgingLabel})
	}
	tree := make([]map[string]any, 0, len(fx.NameTree))
	for _, root := range fx.NameTree {
		kids := make([]map[string]any, 0, len(root.Children))
		for _, leaf := range root.Children {
			kids = append(kids, map[string]any{"Label": leaf.Label, "Sev": leaf.Sev})
		}
		tree = append(tree, map[string]any{"Label": root.Label, "Count": root.Count, "Sev": root.Sev, "Children": kids})
	}
	msgs := make([]map[string]any, 0, len(fx.CoverageMsgs))
	for _, m := range fx.CoverageMsgs {
		msgs = append(msgs, map[string]any{"Kind": m.Kind, "Badge": m.Badge, "Bound": m.Bound, "Subject": m.Subject, "Text": m.Text, "When": m.When, "ISO": m.ISO})
	}
	proposals := make([]map[string]any, 0, len(fx.Proposals))
	for _, p := range fx.Proposals {
		proposals = append(proposals, map[string]any{"ID": p.ID, "Value": p.Value, "Kind": p.Kind, "Source": p.Source})
	}
	exclusions := make([]map[string]any, 0, len(fx.Exclusions))
	for _, e := range fx.Exclusions {
		exclusions = append(exclusions, map[string]any{"ID": e.ID, "Kind": e.Kind, "Value": e.Value})
	}

	base := func() map[string]any {
		return map[string]any{
			"Title": "Scope", "NavActive": "scope", "DesignTokens": true,
			// states.json scope states run session=admin, so the golden renders the admin
			// view (seed chip-remove + declare form, custody toggle form, zone FileDrop,
			// proposals confirm/decline, exclusion add) — the same IsAdmin the seeded
			// candidate carries. Without it the golden would render the viewer view and
			// diverge by the whole admin control surface.
			"IsAdmin":          true,
			"AddressCap":       fx.AddressCap,
			"Seeds":            seeds,
			"FormScope":        "",
			"FormError":        "",
			"CustodyScopes":    custody,
			"ZoneScopes":       zones,
			"ZoneIntervalDays": strconv.Itoa(fx.ZoneIntervalDays),
			"NameTree":         tree,
			"CoverageMsgs":     msgs,
			"Proposals":        proposals,
			"OrgQuery":         "",
			"Exclusions":       exclusions,
			"ExclKind":         "",
			"ExclValue":        "",
		}
	}

	defaultData := base()

	refusalData := base()
	refusalData["FormScope"] = fx.RefusalFixture.PostValue
	refusalData["FormError"] = fx.RefusalFixture.FormError
	refusalData["Refusal"] = map[string]any{
		"Input":     fx.RefusalFixture.Input,
		"Reason":    fx.RefusalFixture.Reason,
		"Reachable": fx.RefusalFixture.Reachable,
	}

	previewData := base()
	previewData["ExclKind"] = fx.ExclusionPreviewFixture.PostKind
	previewData["ExclValue"] = fx.ExclusionPreviewFixture.PostValue
	previewData["ExclPreview"] = map[string]any{
		"Fires":    fx.ExclusionPreviewFixture.Fires,
		"Headline": fx.ExclusionPreviewFixture.Headline,
		"Loss":     fx.ExclusionPreviewFixture.Loss,
	}

	type sstate struct {
		id   string
		data map[string]any
	}
	states := []sstate{
		{"default", defaultData},
		{"refusal", refusalData},
		{"exclusion-preview", previewData},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/scope.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "scope", st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// driftFixture is the design-system/fixtures/fixtures.json drift slice: the range-picker presets,
// the change vocabulary, the trigger + tally scalars (period, period_label, batch_id/label, the
// transition count + signed delta), the movement map, and the batch groups (each with its collapsed
// flag and transition events, some carrying a before/after diff). The golden reads them here (never
// re-hardcoded) so a fixture change flows through; cmd/web/devfixtures.go pins the same values with
// a drift test (TestDriftFixtureMatchesPackage).
type driftFixture struct {
	Period          string `json:"period"`
	PeriodLabel     string `json:"period_label"`
	HasEvents       bool   `json:"has_events"`
	Truncated       bool   `json:"truncated"`
	FeedLimit       int    `json:"feed_limit"`
	BatchID         string `json:"batch_id"`
	BatchLabel      string `json:"batch_label"`
	TransitionCount int    `json:"transition_count"`
	TransitionDelta string `json:"transition_delta"`
	Periods         []struct {
		Token string `json:"token"`
		Label string `json:"label"`
	} `json:"periods"`
	Kinds []struct {
		Change string `json:"change"`
		Family string `json:"family"`
	} `json:"kinds"`
	Movement map[string]int `json:"movement"`
	Groups   []struct {
		Label     string `json:"label"`
		Meta      string `json:"meta"`
		Collapsed bool   `json:"collapsed"`
		Events    []struct {
			Change  string `json:"change"`
			Family  string `json:"family"`
			Subject string `json:"subject"`
			Detail  string `json:"detail"`
			Time    string `json:"time"`
			Reason  string `json:"reason"`
			Diff    []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"diff"`
		} `json:"events"`
	} `json:"groups"`
}

func loadDriftFixture() (driftFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return driftFixture{}, err
	}
	var ff struct {
		Drift driftFixture `json:"drift"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return driftFixture{}, err
	}
	return ff.Drift, nil
}

// renderDrift composes the single Drift golden HTML from the frozen drift.tmpl. Its data map
// mirrors driftPage driftFixtureData EXACTLY (the holes the frozen tmpl reads) — the preset +
// change vocabularies, the pinned batch groups (each with its Collapsed flag and events), the
// movement tally, and the trigger + tally scalars, all in fixtures.json authored order — so the
// cropped `main` is byte-identical to what the seeded server renders for the DEFAULT state. The
// feed-expanded and range-open states are the SAME HTML with the frozen tmpl's own JS driven over
// it by capture.mjs (states.json), exactly as inventory's expanded/columns-open states — so drift
// is a single-file golden (--page), not a per-state dir. Chrome is the empty stub (crop to `main`).
func renderDrift(bodyFlex bool) ([]byte, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadDriftFixture()
	if err != nil {
		return nil, err
	}

	periods := make([]map[string]any, 0, len(fx.Periods))
	for _, p := range fx.Periods {
		periods = append(periods, map[string]any{"Token": p.Token, "Label": p.Label})
	}
	kinds := make([]map[string]any, 0, len(fx.Kinds))
	for _, k := range fx.Kinds {
		kinds = append(kinds, map[string]any{"Change": k.Change, "Family": k.Family})
	}
	groups := make([]map[string]any, 0, len(fx.Groups))
	for _, g := range fx.Groups {
		events := make([]map[string]any, 0, len(g.Events))
		for _, e := range g.Events {
			diff := make([]map[string]any, 0, len(e.Diff))
			for _, d := range e.Diff {
				diff = append(diff, map[string]any{"Type": d.Type, "Text": d.Text})
			}
			events = append(events, map[string]any{
				"Change": e.Change, "Family": e.Family, "Subject": e.Subject,
				"Detail": e.Detail, "Time": e.Time, "Reason": e.Reason, "Diff": diff,
			})
		}
		groups = append(groups, map[string]any{
			"Label": g.Label, "Meta": g.Meta, "Collapsed": g.Collapsed, "Events": events,
		})
	}

	data := map[string]any{
		"Title": "Drift", "NavActive": "drift", "DesignTokens": true,
		"Kinds": kinds, "Periods": periods,
		"Period": fx.Period, "PeriodLabel": fx.PeriodLabel,
		"Groups": groups, "Movement": fx.Movement,
		"HasEvents": fx.HasEvents, "Truncated": fx.Truncated, "FeedLimit": fx.FeedLimit,
		"BatchID": fx.BatchID, "BatchLabel": fx.BatchLabel,
		"TransitionCount": fx.TransitionCount, "TransitionDelta": fx.TransitionDelta,
	}

	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/drift.tmpl"); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "drift", data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
		// signDelta mirrors cmd/web/templates_shell.go byte-for-byte: the exposure.tmpl's
		// exposed-tile chip formats its vs-last-batch change (.ExposedDelta.Change) through it,
		// so the golden must carry the identical "+N" / "−N" / "0" label the app renders. A
		// screen that never calls signDelta (inventory, coverage, …) simply ignores it.
		"signDelta": func(n int) template.HTML {
			var s string
			switch {
			case n > 0:
				s = "+" + strconv.Itoa(n)
			case n < 0:
				s = "−" + strconv.Itoa(-n)
			default:
				s = "0"
			}
			return template.HTML(s) // #nosec G203 -- a sign and digits only, from an int; no user input
		},
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

// profileFixture is the design-system/fixtures/fixtures.json → profile slice: the account,
// its live sessions, linked SSO identity + linkable provider, personal tokens, and the
// deterministic minted-token plaintext. The golden reads them here (never re-hardcoded) so a
// fixture change flows through; cmd/web/devfixtures.go pins the same values with a drift test.
type profileFixture struct {
	Account struct {
		Username    string `json:"username"`
		Role        string `json:"role"`
		Created     string `json:"created"`
		TotpEnabled bool   `json:"totp_enabled"`
		Initials    string `json:"initials"`
	} `json:"account"`
	Sessions []struct {
		ID         string `json:"id"`
		Device     string `json:"device"`
		IP         string `json:"ip"`
		LastActive string `json:"last_active"`
		Current    bool   `json:"current"`
	} `json:"sessions"`
	SSOIdentities []struct {
		ID          string `json:"id"`
		Provider    string `json:"provider"`
		DisplayName string `json:"display_name"`
		LinkedAt    string `json:"linked_at"`
	} `json:"sso_identities"`
	SSOProviders []struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"sso_providers"`
	Tokens []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Prefix  string `json:"prefix"`
		Created string `json:"created"`
		Last    string `json:"last"`
	} `json:"tokens"`
	MintedToken string `json:"minted_token_fixture"`
}

func loadProfileFixture() (profileFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return profileFixture{}, err
	}
	var ff struct {
		Profile profileFixture `json:"profile"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return profileFixture{}, err
	}
	return ff.Profile, nil
}

// renderProfileStates composes the six Profile golden HTMLs from the frozen profile.tmpl, one
// per states.json state. Each state's data map mirrors renderProfile's output EXACTLY (the holes
// the frozen tmpl reads): the persistent surface (account, sessions, tokens, SSO) is the same
// across all six, and each state flips only its own transient dialog flag — so the cropped `main`
// is byte-identical to what the seeded server renders (golden and candidate = same tmpl, same
// holes). Tokens are emitted in fixture order (created-ASC: laptop-cli → grafana-readonly), which
// is the order renderProfile now sorts to; the minted state appends the fixture's ci-golden token
// last (created 2026-08-24, never-used "—"), mirroring the live create → re-list. IDs feed only
// form values + hrefs, never text in the `main` crop.
func renderProfileStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadProfileFixture()
	if err != nil {
		return nil, err
	}

	sessions := make([]map[string]any, 0, len(fx.Sessions))
	for _, s := range fx.Sessions {
		sessions = append(sessions, map[string]any{
			"ID": s.ID, "Device": s.Device, "IP": s.IP, "LastActive": s.LastActive, "Current": s.Current,
		})
	}
	baseTokens := make([]map[string]any, 0, len(fx.Tokens))
	for _, t := range fx.Tokens {
		baseTokens = append(baseTokens, map[string]any{
			"ID": t.ID, "Name": t.Name, "Prefix": t.Prefix, "Created": t.Created, "Last": t.Last,
		})
	}
	ssoIdent := make([]map[string]any, 0, len(fx.SSOIdentities))
	for _, i := range fx.SSOIdentities {
		ssoIdent = append(ssoIdent, map[string]any{
			"ID": i.ID, "Provider": i.Provider, "DisplayName": i.DisplayName, "LinkedAt": i.LinkedAt,
		})
	}
	ssoProv := make([]map[string]any, 0, len(fx.SSOProviders))
	for _, p := range fx.SSOProviders {
		ssoProv = append(ssoProv, map[string]any{"Slug": p.Slug, "Name": p.Name})
	}

	// The minted state's tokens table gains the freshly-minted ci-golden row LAST — mirroring
	// createPersonalToken's fixture mint (devFixtureMintedToken) + the created-ASC re-list. Its
	// prefix is plaintext[:11]+"…" exactly as fixtureMintedToken forms it; created is the pinned
	// fixture clock's date; last is "—" (never used). A separate slice so it never leaks into the
	// other five states.
	mintedPrefix := fx.MintedToken
	if len(mintedPrefix) >= 11 {
		mintedPrefix = mintedPrefix[:11] + "…"
	}
	mintedTokens := make([]map[string]any, 0, len(baseTokens)+1)
	mintedTokens = append(mintedTokens, baseTokens...)
	mintedTokens = append(mintedTokens, map[string]any{
		"ID": "new", "Name": "ci-golden", "Prefix": mintedPrefix, "Created": "2026-08-24", "Last": "—",
	})

	base := func(tokens []map[string]any) map[string]any {
		return map[string]any{
			"Initials":      fx.Account.Initials,
			"Username":      fx.Account.Username,
			"Role":          fx.Account.Role,
			"CreatedISO":    fx.Account.Created,
			"TotpEnabled":   fx.Account.TotpEnabled,
			"Notice":        "",
			"PwError":       "",
			"Sessions":      sessions,
			"Tokens":        tokens,
			"SSOIdentities": ssoIdent,
			"SSOProviders":  ssoProv,
			"SSONotice":     "",
			"SSOError":      "",
			"CreateOpen":    false,
			"Minted":        "",
			"TokName":       "",
			"TokError":      "",
			"MintedName":    "",
			"RevokeID":      "",
			"RevokeName":    "",
			"RevokeErr":     "",
			"EndSession":    false,
			"SignOutOthers": false,
		}
	}

	// Order mirrors states.json's profile block.
	newTok := base(baseTokens)
	newTok["CreateOpen"] = true

	minted := base(mintedTokens)
	minted["Minted"] = fx.MintedToken
	minted["TokName"] = "ci-golden"
	minted["MintedName"] = "ci-golden"

	revoke := base(baseTokens)
	revoke["RevokeID"] = "t1"
	revoke["RevokeName"] = "laptop-cli"

	endSession := base(baseTokens)
	endSession["EndSession"] = true

	signOutOthers := base(baseTokens)
	signOutOthers["SignOutOthers"] = true

	data := map[string]map[string]any{
		"default":        base(baseTokens),
		"new-token":      newTok,
		"minted":         minted,
		"revoke-token":   revoke,
		"end-session":    endSession,
		"signout-others": signOutOthers,
	}
	order := []string{"default", "new-token", "minted", "revoke-token", "end-session", "signout-others"}

	out := make([]errorGolden, 0, len(order))
	for _, id := range order {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/profile.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "profile", data[id]); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: id, html: buf.Bytes()})
	}
	return out, nil
}

// signinFixture is the design-system/fixtures/fixtures.json → signin slice plus the two login
// accounts (for the totp step's mid-login username and the enroll screen's account): the build
// version, the login provider set (slug/name/mark), the well-known reset/invite tokens + invite
// role, the enroll secret, and the recovery-code set. The golden reads them here (never
// re-hardcoded) so a fixture change flows through; cmd/web/devfixtures.go pins the same values
// with a drift test (TestSigninFixtureMatchesPackage).
type signinFixture struct {
	Version      string `json:"version"`
	SSOProviders []struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
		Mark string `json:"mark"`
	} `json:"sso_providers"`
	ResetToken    string   `json:"reset_token"`
	InviteToken   string   `json:"invite_token"`
	InviteRole    string   `json:"invite_role"`
	EnrollSecret  string   `json:"enroll_secret"`
	RecoveryCodes []string `json:"recovery_codes"`
	// AdminUser / ViewerUser are read from the top-level accounts slice: the totp step names the
	// mid-login admin account, the enroll/recovery screens run as the viewer session account.
	AdminUser  string
	ViewerUser string
}

func loadSigninFixture() (signinFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return signinFixture{}, err
	}
	var ff struct {
		Signin   signinFixture `json:"signin"`
		Accounts []struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return signinFixture{}, err
	}
	sf := ff.Signin
	for _, a := range ff.Accounts {
		switch a.Role {
		case "admin":
			if sf.AdminUser == "" {
				sf.AdminUser = a.Username
			}
		case "viewer":
			if sf.ViewerUser == "" {
				sf.ViewerUser = a.Username
			}
		}
	}
	return sf, nil
}

// renderSigninStates composes the SignIn-family golden HTMLs from the frozen signin.tmpl, one per
// states.json signin state (states.json is authoritative: 11 states, no reset-done). Each state's
// data map mirrors the handler output EXACTLY (the holes the frozen tmpl reads) so the chrome-less
// `body` crop is byte-identical to what the seeded server renders — golden and candidate = same
// tmpl, same holes. Every page emits authfoot, so every map carries .Version. The enroll QR is
// built with the SAME auth.OtpauthURI + qr.SVG the handler's totpEnrollData uses, over the same
// secret + viewer username + issuer, so the two encodings are byte-identical.
func renderSigninStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadSigninFixture()
	if err != nil {
		return nil, err
	}

	// issuer mirrors cmd/web's `issuer` const (auth.go); the enroll otpauth URI names it.
	const issuer = "Verge ASM"

	providers := make([]map[string]any, 0, len(fx.SSOProviders))
	for _, p := range fx.SSOProviders {
		providers = append(providers, map[string]any{"Slug": p.Slug, "Name": p.Name, "Mark": p.Mark})
	}

	// Enroll: build the QR from the pinned secret + viewer account, exactly as totpEnrollData does.
	enrollURI := auth.OtpauthURI(fx.EnrollSecret, fx.ViewerUser, issuer)
	enrollSVG, err := qr.SVG([]byte(enrollURI), "Two-factor enrollment QR code for "+fx.ViewerUser)
	if err != nil {
		return nil, fmt.Errorf("signin: build enroll QR: %w", err)
	}

	v := func(m map[string]any) map[string]any {
		m["Version"] = fx.Version
		return m
	}

	type sstate struct {
		id   string
		tmpl string
		data map[string]any
	}
	states := []sstate{
		{"login", "login", v(map[string]any{"Notice": "", "Error": "", "SSOProviders": providers})},
		{"login-sso-none", "login", v(map[string]any{"Notice": "", "Error": "", "SSOProviders": []map[string]any{}})},
		{"totp", "totp", v(map[string]any{"Error": "", "Username": fx.AdminUser})},
		{"forgot", "forgot", v(map[string]any{})},
		{"forgot-sent", "forgot-sent", v(map[string]any{})},
		{"reset", "reset", v(map[string]any{"Error": "", "Token": fx.ResetToken})},
		{"reset-invalid", "reset-invalid", v(map[string]any{})},
		{"invite", "invite", v(map[string]any{"Error": "", "Token": fx.InviteToken, "Role": fx.InviteRole, "Username": ""})},
		{"invite-invalid", "invite-invalid", v(map[string]any{})},
		{"enroll", "totp-enroll", v(map[string]any{"Error": "", "Secret": fx.EnrollSecret, "OtpauthQR": template.HTML(enrollSVG)})}, // #nosec G203 -- trusted QR SVG built by our own encoder, no user input
		{"recovery", "totp-recovery", v(map[string]any{"Codes": fx.RecoveryCodes})},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/signin.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, st.tmpl, st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// renderSetupStates composes the Setup screen's golden HTMLs from the frozen setup.tmpl, one per
// states.json setup state (default, error). setup.tmpl's single "setup" define reuses the SignIn
// family's shared authcss / authbrand / authfoot partials, so BOTH signin.tmpl and setup.tmpl are
// parsed into the stub set for those refs to resolve — mirroring the app, where both parse into the
// one shared set. Each state's data map mirrors the setupForm / setupSubmit handler output EXACTLY
// (the .Error / .Token / .Version holes): "default" is the open first-run form (no error, empty
// token) and "error" is the invalid-token re-render the states.json script drives (POST with
// token="wrong" → "Invalid setup token." with the rejected token echoed back). .Version is the
// SignIn fixture's build version (the authfoot the app fills via buildVersion→devFixtureVersion), so
// the chrome-less footer matches the candidate. The `body` crop is byte-identical to the seeded
// server's render — golden and candidate = same tmpl, same holes.
func renderSetupStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	// The version hole is filled by authfoot on every SignIn-family page, Setup included; it comes
	// from the SignIn fixture slice (the app's buildVersion returns devFixtureVersion in dev).
	sf, err := loadSigninFixture()
	if err != nil {
		return nil, err
	}

	type sstate struct {
		id   string
		data map[string]any
	}
	states := []sstate{
		{"default", map[string]any{"Error": "", "Token": "", "Version": sf.Version}},
		{"error", map[string]any{"Error": "Invalid setup token.", "Token": "wrong", "Version": sf.Version}},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		// signin.tmpl carries the shared authcss/authbrand/authfoot setup.tmpl calls; parse both.
		if _, err := t.ParseFS(designfs.FS, "templates/signin.tmpl", "templates/setup.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "setup", st.data); err != nil {
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

// signalsFixture is the design-system/fixtures/fixtures.json signals slice: the open-tab scalars
// (open count, shown, page info/count), the detecting vantage, the ten open rows + three withdrawn
// rows (each carrying its rule metadata: tags, nullable CVE, description, the [name, version] rule
// ref), the annotations and the drift diffs. The golden reads them here (never re-hardcoded) so a
// fixture change flows through; cmd/web/devfixtures.go pins the same values with a drift test
// (TestSignalsFixtureMatchesPackage).
type signalsFixture struct {
	OpenCount   int                 `json:"open_count"`
	Shown       int                 `json:"shown"`
	PageInfo    string              `json:"page_info"`
	PageCount   int                 `json:"page_count"`
	DetectedBy  string              `json:"detected_by"`
	Rows        []signalsFixtureRow `json:"rows"`
	Withdrawn   []signalsFixtureRow `json:"withdrawn"`
	Annotations map[string]struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	} `json:"annotations"`
	Diffs map[string]struct {
		Title string `json:"title"`
		Lines []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"lines"`
	} `json:"diffs"`
}

type signalsFixtureRow struct {
	ID       string            `json:"id"`
	Severity string            `json:"severity"`
	SevLabel string            `json:"sev_label"`
	Title    string            `json:"title"`
	Asset    string            `json:"asset"`
	IP       string            `json:"ip"`
	Port     string            `json:"port"`
	Seen     string            `json:"seen"`
	First    string            `json:"first"`
	Last     string            `json:"last"`
	CVE      *string           `json:"cve"`
	Tags     []string          `json:"tags"`
	Desc     string            `json:"desc"`
	Rule     []json.RawMessage `json:"rule"`
	ViewKey  string            `json:"view_key"`
}

// signalsDiscovered is the fixed discovery instant the span-derived "Asset discovered" history
// entry renders (embedded in fixtures.json signals.history_rule). cmd/web/devfixtures.go pins the
// identical literal (devSignalsDiscovered), asserted present in history_rule by the drift test.
const signalsDiscovered = "2026-08-12"

func loadSignalsFixture() (signalsFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return signalsFixture{}, err
	}
	var ff struct {
		Signals signalsFixture `json:"signals"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return signalsFixture{}, err
	}
	return ff.Signals, nil
}

// ruleParts splits the fixture's [name, version] rule ref into the id + version strings the drawer
// renders as "id@version".
func ruleParts(raw []json.RawMessage) (id, version string) {
	if len(raw) >= 1 {
		_ = json.Unmarshal(raw[0], &id)
	}
	if len(raw) >= 2 {
		var n json.Number
		if err := json.Unmarshal(raw[1], &n); err == nil {
			version = n.String()
		}
	}
	return id, version
}

func signalsCVE(row signalsFixtureRow) string {
	if row.CVE != nil {
		return *row.CVE
	}
	return ""
}

// signalsHistoryG mirrors cmd/web/devfixtures.go signalsHistory byte-for-byte (span-derived per
// fixtures.json signals.history_rule): Still present · Drift detected (only when a diff exists) ·
// Signal raised · Asset discovered.
func signalsHistoryG(row signalsFixtureRow, detectedBy string, hasDiff bool) []map[string]any {
	raisedTone := "neutral"
	if row.Severity == "critical" || row.Severity == "high" {
		raisedTone = "danger"
	}
	hist := []map[string]any{
		{"Title": "Still present", "Detail": detectedBy + " re-confirmed", "Time": row.Seen, "Tone": "accent", "Mono": false},
	}
	if hasDiff {
		hist = append(hist, map[string]any{"Title": "Drift detected", "Detail": row.Asset + " changed", "Time": row.Seen, "Tone": "warn", "Mono": true})
	}
	hist = append(hist,
		map[string]any{"Title": "Signal raised", "Detail": row.ID, "Time": row.First, "Tone": raisedTone, "Mono": true},
		map[string]any{"Title": "Asset discovered", "Detail": row.Asset, "Time": signalsDiscovered, "Tone": "neutral", "Mono": true},
	)
	return hist
}

// signalsRowMapG mirrors cmd/web/devfixtures.go signalsRowMap: one table row's holes.
func signalsRowMapG(row signalsFixtureRow, closeHref string, withdrawn bool) map[string]any {
	return map[string]any{
		"Severity":    row.Severity,
		"SevLabel":    row.SevLabel,
		"Title":       row.Title,
		"Asset":       row.Asset,
		"Port":        row.Port,
		"SigID":       row.ID,
		"Seen":        row.Seen,
		"Last":        row.Last,
		"Withdrawn":   withdrawn,
		"ViewKey":     row.ID,
		"DescopeHref": closeHref + "&descope=" + row.ID,
	}
}

// signalsDrawerMapG mirrors cmd/web/devfixtures.go signalsDrawerMap: the full spec drawer (#21j).
func signalsDrawerMapG(fx signalsFixture, row signalsFixtureRow, withdrawn bool) map[string]any {
	ruleID, ruleVersion := ruleParts(row.Rule)
	d := map[string]any{
		"Title":       row.Title,
		"Seen":        row.Seen,
		"SigID":       row.ID,
		"Severity":    row.Severity,
		"SevLabel":    row.SevLabel,
		"Withdrawn":   withdrawn,
		"Tags":        row.Tags,
		"CVE":         signalsCVE(row),
		"Desc":        row.Desc,
		"Asset":       row.Asset,
		"IP":          row.IP,
		"RuleID":      ruleID,
		"RuleVersion": ruleVersion,
		"Port":        row.Port,
		"DetectedBy":  fx.DetectedBy,
		"First":       row.First,
		"Last":        row.Last,
	}
	diff, hasDiff := fx.Diffs[row.ID]
	if hasDiff {
		lines := make([]map[string]any, 0, len(diff.Lines))
		for _, l := range diff.Lines {
			lines = append(lines, map[string]any{"Type": l.Type, "Text": l.Text})
		}
		d["Diff"] = map[string]any{"Title": diff.Title, "Lines": lines}
	}
	if anno, ok := fx.Annotations[row.ID]; ok {
		d["Annotated"] = true
		d["AnnoID"] = anno.ID
		d["AnnoReason"] = anno.Reason
	}
	d["History"] = signalsHistoryG(row, fx.DetectedBy, hasDiff)
	return d
}

// renderSignalsStates composes the six Signals golden HTMLs from the frozen signals.tmpl, one per
// states.json signals state (default, drawer-open, drawer-annotated, descope-confirm, withdrawn-tab,
// menu-open). Every state's data map mirrors signalsPage's signalsFixtureData EXACTLY (the holes the
// frozen tmpl reads) — the pinned fixtures.json signals slice in authored order, the 47-of-47 open
// count, and the drawer's rule metadata + diff + span history — so the cropped `main`/`body` is
// byte-identical to what the seeded server renders. The drawer / descope states select the row by
// its view key; withdrawn-tab lists the three withdrawn rows; menu-open is the default page whose
// kebab the state JS opens on both sides (capture.mjs runs it against the frozen tmpl's own handler).
// Chrome is the empty stub (drawer/descope crop `body`, the rest crop `main`).
func renderSignalsStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadSignalsFixture()
	if err != nil {
		return nil, err
	}

	withdrawnSet := map[string]bool{}
	for _, w := range fx.Withdrawn {
		withdrawnSet[w.ID] = true
	}
	byKey := map[string]signalsFixtureRow{}
	for _, row := range fx.Rows {
		byKey[row.ID] = row
	}
	for _, row := range fx.Withdrawn {
		byKey[row.ID] = row
	}

	// base builds the tab's data map exactly as signalsFixtureData does for the given tab.
	base := func(tab string) map[string]any {
		closeHref := "/signals?tab=" + tab + "&sort=sev&dir=asc"
		viewPrefix := closeHref + "&view="

		var tabRows []signalsFixtureRow
		switch tab {
		case "withdrawn":
			tabRows = fx.Withdrawn
		case "annotated":
			for _, row := range fx.Rows {
				if _, ok := fx.Annotations[row.ID]; ok {
					tabRows = append(tabRows, row)
				}
			}
		default:
			tabRows = fx.Rows
		}
		rows := make([]map[string]any, 0, len(tabRows))
		for _, row := range tabRows {
			rows = append(rows, signalsRowMapG(row, closeHref, withdrawnSet[row.ID]))
		}

		sortHref := func(col string) string {
			nd := "asc"
			if col == "sev" {
				nd = "desc"
			}
			return "/signals?tab=" + tab + "&sort=" + col + "&dir=" + nd
		}

		data := map[string]any{
			"Title": "Signals", "NavActive": "signals", "DesignTokens": true, "IsAdmin": true,
			"Tab":            tab,
			"OpenCount":      fx.OpenCount,
			"AnnotatedCount": len(fx.Annotations),
			"WithdrawnCount": len(fx.Withdrawn),
			"Q":              "",
			"Sev":            "All severities",
			"SevOptions":     []string{"All severities", "Critical", "High", "Medium", "Low", "Info"},
			"HasAny":         true,
			"ClearHref":      "/signals?tab=" + tab,
			"HasExport":      true,
			"ExportHref":     "/signals/export?tab=" + tab,
			"AnnoError":      "",
			"ViewPrefix":     viewPrefix,
			"CloseHref":      closeHref,
			"Rows":           rows,
			"SelKey":         "",
			"Sort": map[string]any{
				"Key": "sev", "Dir": "asc",
				"SevHref": sortHref("sev"), "AssetHref": sortHref("asset"),
				"IDHref": sortHref("id"), "SeenHref": sortHref("seen"),
			},
		}
		if tab == "open" {
			data["Shown"] = fx.Shown
			data["Total"] = fx.OpenCount
			data["ShowPagination"] = true
			data["PageInfo"] = fx.PageInfo
			data["PrevDisabled"] = true
			data["PrevHref"] = closeHref
			data["NextDisabled"] = false
			data["NextHref"] = closeHref + "&page=2"
			pages := make([]map[string]any, 0, fx.PageCount)
			for p := 1; p <= fx.PageCount; p++ {
				href := closeHref
				if p > 1 {
					href = closeHref + "&page=" + strconv.Itoa(p)
				}
				pages = append(pages, map[string]any{"Ellipsis": false, "Href": href, "Num": p, "Active": p == 1})
			}
			data["Pages"] = pages
		} else {
			data["Shown"] = len(tabRows)
			data["Total"] = len(tabRows)
			data["ShowPagination"] = false
			data["Pages"] = []map[string]any{}
		}
		return data
	}

	defaultData := base("open")

	drawerOpen := base("open")
	if row, ok := byKey["SIG-1042"]; ok {
		drawerOpen["SelKey"] = "SIG-1042"
		drawerOpen["Drawer"] = signalsDrawerMapG(fx, row, withdrawnSet["SIG-1042"])
	}

	drawerAnnotated := base("open")
	if row, ok := byKey["SIG-1027"]; ok {
		drawerAnnotated["SelKey"] = "SIG-1027"
		drawerAnnotated["Drawer"] = signalsDrawerMapG(fx, row, withdrawnSet["SIG-1027"])
	}

	descopeConfirm := base("open")
	if row, ok := byKey["SIG-1042"]; ok {
		descopeConfirm["Descope"] = map[string]any{"Asset": row.Asset, "CloseHref": "/signals?tab=open&sort=sev&dir=asc"}
	}

	withdrawnTab := base("withdrawn")

	// menu-open is the default page; its kebab is opened by the state JS on both golden and
	// candidate (capture.mjs runs the state's js against the frozen tmpl's own handler).
	menuOpen := base("open")

	type sstate struct {
		id   string
		data map[string]any
	}
	states := []sstate{
		{"default", defaultData},
		{"drawer-open", drawerOpen},
		{"drawer-annotated", drawerAnnotated},
		{"descope-confirm", descopeConfirm},
		{"withdrawn-tab", withdrawnTab},
		{"menu-open", menuOpen},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/signals.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "signals", st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// dashboardFixture is the design-system/fixtures/fixtures.json → dashboard slice: the header
// schedule, the running-scan detail + `scanning` variant token, the missed-check vantage set, the
// five-cell stat band (with per-cell delta + the live_when_scanning flag), the by-severity ramp, the
// two-shape coverage meters (address counted/total + pct, name census), the silent-zone callout and
// the three vantages. The most-recent register reuses the signals.rows slice (fixtures.json says
// "first 6 of signals.rows"). The golden reads them here (never re-hardcoded) so a fixture change
// flows through; cmd/web/devfixtures.go pins the same values with a drift test
// (TestDashboardFixtureMatchesPackage).
type dashboardFixture struct {
	ScanSchedule struct {
		HasLast bool   `json:"has_last"`
		LastAgo string `json:"last_ago"`
		HasNext bool   `json:"has_next"`
		NextIn  string `json:"next_in"`
	} `json:"scan_schedule"`
	ScanDetail  string   `json:"scan_detail"`
	Unavailable []string `json:"unavailable"`
	StatBand    []struct {
		Label            string `json:"label"`
		Value            string `json:"value"`
		LiveWhenScanning bool   `json:"live_when_scanning"`
		HasDelta         bool   `json:"has_delta"`
		Change           int    `json:"change"`
		Tone             string `json:"tone"`
		Caption          string `json:"caption"`
	} `json:"stat_band"`
	SevBars []struct {
		Sev   string `json:"sev"`
		Pct   int    `json:"pct"`
		Count int    `json:"count"`
	} `json:"sev_bars"`
	CoverageMeters []struct {
		Label   string          `json:"label"`
		Counted json.RawMessage `json:"counted"`
		Total   *int            `json:"total"`
		Pct     int             `json:"pct"`
		Unit    string          `json:"unit"`
	} `json:"coverage_meters"`
	SilentZone struct {
		Bound string `json:"bound"`
		Text  string `json:"text"`
	} `json:"silent_zone"`
	Vantages []struct {
		Name    string `json:"name"`
		Latency string `json:"latency"`
		Avail   string `json:"avail"`
	} `json:"vantages"`
}

func loadDashboardFixture() (dashboardFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return dashboardFixture{}, err
	}
	var ff struct {
		Dashboard dashboardFixture `json:"dashboard"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return dashboardFixture{}, err
	}
	return ff.Dashboard, nil
}

// dashRawStr renders a fixtures.json coverage-meter `counted` value verbatim — the field is mixed
// (a JSON number 212 for the address scope, a pre-formatted string "1,284" for the name scope), so
// it is read as a raw message and unquoted only when it is a JSON string.
func dashRawStr(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if len(s) >= 1 && s[0] == '"' {
		var out string
		if err := json.Unmarshal(raw, &out); err == nil {
			return out
		}
	}
	return s
}

// renderDashboardStates composes the three Dashboard golden HTMLs from the frozen dashboard.tmpl, one
// per states.json dashboard state (default, scanning, banner-dismissed). Every state's data map
// mirrors home()'s dashboardFixtureData EXACTLY (the holes the frozen tmpl reads) — the pinned
// fixtures.json dashboard slice in authored order plus the first six signals.rows for the most-recent
// register — so the cropped `main` is byte-identical to what the seeded server renders. The `scanning`
// state lights .Scanning + .ScanDetail and the first cell's live pulse; banner-dismissed sets
// .ProbeDismissed. "home" wraps head/chrome/dashboard/foot; it references the repo-authored "firstrun"
// (stubbed empty here — never executed with EmptyEstate false) and the "sevbadge" define signals.tmpl
// declares (parsed in for the register rows). Chrome is the empty stub (goldens crop to `main`).
func renderDashboardStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadDashboardFixture()
	if err != nil {
		return nil, err
	}
	sfx, err := loadSignalsFixture()
	if err != nil {
		return nil, err
	}
	if len(sfx.Rows) < 6 {
		return nil, fmt.Errorf("dashboard golden: signals.rows has %d rows, need >= 6", len(sfx.Rows))
	}

	sevBars := make([]map[string]any, 0, len(fx.SevBars))
	for _, b := range fx.SevBars {
		sevBars = append(sevBars, map[string]any{"Sev": b.Sev, "Pct": b.Pct, "Count": b.Count})
	}
	meters := make([]map[string]any, 0, len(fx.CoverageMeters))
	for _, m := range fx.CoverageMeters {
		mv := map[string]any{"Label": m.Label, "Counted": dashRawStr(m.Counted), "Unit": m.Unit}
		if m.Total != nil {
			mv["Total"] = strconv.Itoa(*m.Total)
			mv["Pct"] = m.Pct
		}
		meters = append(meters, mv)
	}
	vantages := make([]map[string]any, 0, len(fx.Vantages))
	for _, v := range fx.Vantages {
		vantages = append(vantages, map[string]any{"Name": v.Name, "Latency": v.Latency, "Avail": v.Avail})
	}
	recent := make([]map[string]any, 0, 6)
	for _, row := range sfx.Rows[:6] {
		recent = append(recent, map[string]any{
			"Severity": row.Severity, "SevLabel": row.SevLabel, "Title": row.Title,
			"Asset": row.Asset, "Port": row.Port, "Seen": row.Seen, "ViewKey": row.ViewKey,
		})
	}

	buildData := func(scanning, probeDismissed bool) map[string]any {
		statBand := make([]map[string]any, 0, len(fx.StatBand))
		for _, st := range fx.StatBand {
			statBand = append(statBand, map[string]any{
				"Label": st.Label, "Value": st.Value, "Live": scanning && st.LiveWhenScanning,
				"HasDelta": st.HasDelta, "Change": st.Change, "Tone": st.Tone, "Caption": st.Caption,
			})
		}
		data := map[string]any{
			"Title": "Dashboard", "NavActive": "dashboard", "DesignTokens": true, "IsAdmin": true,
			"EmptyEstate": false,
			"ScanSchedule": map[string]any{
				"HasLast": fx.ScanSchedule.HasLast, "LastAgo": fx.ScanSchedule.LastAgo,
				"HasNext": fx.ScanSchedule.HasNext, "NextIn": fx.ScanSchedule.NextIn,
			},
			"Scanning":       scanning,
			"Unavailable":    fx.Unavailable,
			"ProbeDismissed": probeDismissed,
			"StatBand":       statBand,
			"HasSignals":     true,
			"SevBars":        sevBars,
			"CoverageMeters": meters,
			"SilentZone":     map[string]any{"Bound": fx.SilentZone.Bound, "Text": fx.SilentZone.Text},
			"Vantages":       vantages,
			"RecentSignals":  recent,
		}
		if scanning {
			data["ScanDetail"] = fx.ScanDetail
		}
		return data
	}

	states := []struct {
		id   string
		data map[string]any
	}{
		{"default", buildData(false, false)},
		{"scanning", buildData(true, false)},
		{"banner-dismissed", buildData(false, true)},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		// "home" references the repo-authored "firstrun" define; stub it empty so the escaper (which
		// statically walks every {{template}} target) resolves it — EmptyEstate is false, so it is
		// never executed. signals.tmpl carries the "sevbadge" define the register rows call.
		if _, err := t.Parse(`{{define "firstrun"}}{{end}}`); err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/signals.tmpl"); err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/dashboard.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, "home", st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// assetFixture is the design-system/fixtures/fixtures.json → asset slice: the header identity, the
// three-port census (with the joined Service strings), the DNS records, the parsed TLS certificate
// card, the provenance facts, the signals-here and the drift trail. Its Go field names ARE the holes
// the frozen asset.tmpl reads, so the loaded value passes straight to the template as `.Asset`.
// cmd/web/devfixtures.go pins the same values with a drift test (TestAssetFixtureMatchesPackage).
type assetFixture struct {
	Key          string             `json:"key"`
	Type         string             `json:"type"`
	Severity     string             `json:"severity"`
	SevLabel     string             `json:"sev_label"`
	Exposure     string             `json:"exposure"`
	Seen         string             `json:"seen"`
	InScopeSince string             `json:"in_scope_since"`
	Withdrawn    bool               `json:"withdrawn"`
	Ports        []assetFixturePort `json:"ports"`
	DNS          []assetFixtureDNS  `json:"dns"`
	Cert         *assetFixtureCert  `json:"cert"`
	Provenance   []assetFixtureKV   `json:"provenance"`
	Signals      []assetFixtureSig  `json:"signals"`
	Drift        []assetFixtureDrft `json:"drift"`
}

type assetFixturePort struct {
	Port     string `json:"port"`
	Service  string `json:"service"`
	Exposure string `json:"exposure"`
	Since    string `json:"since"`
}

type assetFixtureDNS struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Seen  string `json:"seen"`
}

type assetFixtureCert struct {
	Name        string `json:"name"`
	Issuer      string `json:"issuer"`
	Algorithm   string `json:"algorithm"`
	NotAfter    string `json:"not_after"`
	Label       string `json:"label"`
	Tone        string `json:"tone"`
	Fingerprint string `json:"fingerprint"`
}

type assetFixtureKV struct {
	K string `json:"k"`
	V string `json:"v"`
}

type assetFixtureSig struct {
	Severity string `json:"severity"`
	SevLabel string `json:"sev_label"`
	Rule     string `json:"rule"`
	SigID    string `json:"sig_id"`
	Time     string `json:"time"`
}

type assetFixtureDrft struct {
	Change  string `json:"change"`
	Family  string `json:"family"`
	Subject string `json:"subject"`
	Detail  string `json:"detail"`
	Time    string `json:"time"`
}

func loadAssetFixture() (assetFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return assetFixture{}, err
	}
	var ff struct {
		Asset assetFixture `json:"asset"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return assetFixture{}, err
	}
	return ff.Asset, nil
}

// renderAssetStates composes the AssetDetail golden HTML from the frozen asset.tmpl, for the single
// states.json asset state (default). The data map mirrors assetPage's assetFixtureData EXACTLY (the
// holes the frozen tmpl reads) — the pinned fixtures.json asset slice — so the cropped `main` is
// byte-identical to what the seeded server renders. "asset" wraps head/chrome/asset/foot; it reuses
// the "sevbadge" define signals.tmpl declares (header + signals-here badges) and the "changeglyph"
// define drift.tmpl declares (drift-trail glyphs) — both parsed into the set here. Chrome is the empty
// stub (goldens crop to `main`).
func renderAssetStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadAssetFixture()
	if err != nil {
		return nil, err
	}
	data := map[string]any{
		"Title": fx.Key, "NavActive": "inventory", "IsAdmin": true,
		"Asset": fx,
	}
	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/signals.tmpl", "templates/drift.tmpl", "templates/asset.tmpl"); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "asset", data); err != nil {
		return nil, err
	}
	return []errorGolden{{id: "default", html: buf.Bytes()}}, nil
}

// subjectDetailFixture is the design-system/fixtures/fixtures.json → subjectdetail slice: the two
// service states (reachable + withdrawn) and the endpoint state. Its Go field names ARE the holes
// the frozen subjectdetail.tmpl reads (.Service / .Endpoint), so a loaded value passes straight to
// the template. cmd/web/devfixtures.go pins the same values with a drift test
// (TestSubjectDetailFixtureMatchesPackage).
type subjectDetailFixture struct {
	Service          subjectServiceFixture  `json:"service"`
	ServiceWithdrawn subjectServiceFixture  `json:"service_withdrawn"`
	Endpoint         subjectEndpointFixture `json:"endpoint"`
}

type subjectServiceFixture struct {
	Key                string                   `json:"key"`
	CopyKey            string                   `json:"copy_key"`
	Withdrawn          bool                     `json:"withdrawn"`
	Exposure           string                   `json:"exposure"`
	Seen               string                   `json:"seen"`
	InScopeSince       string                   `json:"in_scope_since"`
	Citation           []subjectCitationFixture `json:"citation"`
	CitationTerminated bool                     `json:"citation_terminated"`
	Address            string                   `json:"address"`
	Port               string                   `json:"port"`
	Transport          string                   `json:"transport"`
	Reach              string                   `json:"reach"`
	ReachGap           bool                     `json:"reach_gap"`
	ReachGapReason     string                   `json:"reach_gap_reason"`
	Since              string                   `json:"since"`
	Timelines          []subjectTimelineFixture `json:"timelines"`
	Rules              []subjectRuleFixture     `json:"rules"`
	Provenance         []subjectKVFixture       `json:"provenance"`
	Signals            []subjectSignalFixture   `json:"signals"`
}

type subjectEndpointFixture struct {
	Key                string                   `json:"key"`
	CopyKey            string                   `json:"copy_key"`
	Nameless           bool                     `json:"nameless"`
	Withdrawn          bool                     `json:"withdrawn"`
	Seen               string                   `json:"seen"`
	InScopeSince       string                   `json:"in_scope_since"`
	Citation           []subjectCitationFixture `json:"citation"`
	CitationTerminated bool                     `json:"citation_terminated"`
	Name               string                   `json:"name"`
	Service            string                   `json:"service"`
	HasIdentity        bool                     `json:"has_identity"`
	Status             string                   `json:"status"`
	Server             string                   `json:"server"`
	Title              string                   `json:"title"`
	RedirectLocation   string                   `json:"redirect_location"`
	WWWAuthenticate    string                   `json:"www_authenticate"`
	Timelines          []subjectTimelineFixture `json:"timelines"`
	Rules              []subjectRuleFixture     `json:"rules"`
	Provenance         []subjectKVFixture       `json:"provenance"`
}

type subjectCitationFixture struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
}

type subjectTimelineFixture struct {
	Label   string                `json:"label"`
	Current *subjectSpanFixture   `json:"current"`
	Breaks  []subjectBreakFixture `json:"breaks"`
	Closed  []subjectSpanFixture  `json:"closed"`
}

type subjectSpanFixture struct {
	IsGap      bool                       `json:"is_gap"`
	Value      string                     `json:"value"`
	Details    []subjectSpanDetailFixture `json:"details"`
	OpenedAt   string                     `json:"opened_at"`
	OpenedFull string                     `json:"opened_full"`
	ClosedAt   string                     `json:"closed_at"`
	ClosedFull string                     `json:"closed_full"`
	Reason     string                     `json:"reason"`
}

type subjectSpanDetailFixture struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type subjectBreakFixture struct {
	At          string `json:"at"`
	MovedLeaves string `json:"moved_leaves"`
}

type subjectRuleFixture struct {
	Rule     string `json:"rule"`
	Version  int    `json:"version"`
	Severity string `json:"severity"`
	SevLabel string `json:"sev_label"`
	Fired    bool   `json:"fired"`
}

type subjectKVFixture struct {
	K string `json:"k"`
	V string `json:"v"`
}

type subjectSignalFixture struct {
	Severity string `json:"severity"`
	SevLabel string `json:"sev_label"`
	Rule     string `json:"rule"`
	SigID    string `json:"sig_id"`
	Time     string `json:"time"`
}

func loadSubjectDetailFixture() (subjectDetailFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return subjectDetailFixture{}, err
	}
	var ff struct {
		SubjectDetail subjectDetailFixture `json:"subjectdetail"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return subjectDetailFixture{}, err
	}
	return ff.SubjectDetail, nil
}

// renderSubjectDetailStates composes the SubjectDetail golden HTMLs from the frozen
// subjectdetail.tmpl, one per states.json subjectdetail state (service · endpoint ·
// service-withdrawn). Each data map mirrors servicePage / endpointPage's fixture data EXACTLY (the
// holes the frozen tmpl reads) — the pinned fixtures.json subjectdetail slice — so the cropped
// `main` is byte-identical to what the seeded server renders. "service"/"endpoint" wrap
// head/chrome/…/foot; they reuse the "assetexposure" define asset.tmpl declares, the "sevbadge"
// define signals.tmpl declares and the "recordrows" define inventory.tmpl declares — all parsed
// into the set here (drift.tmpl is pulled in for asset.tmpl's "changeglyph" reference so the set
// parses, though the executed defines never reach it). Chrome is the empty stub (goldens crop `main`).
func renderSubjectDetailStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadSubjectDetailFixture()
	if err != nil {
		return nil, err
	}
	states := []struct {
		id     string
		define string
		data   map[string]any
	}{
		{id: "service", define: "service", data: map[string]any{
			"Title": fx.Service.Key, "NavActive": "inventory", "IsAdmin": true, "Service": fx.Service,
		}},
		{id: "endpoint", define: "endpoint", data: map[string]any{
			"Title": fx.Endpoint.Key, "NavActive": "inventory", "IsAdmin": true, "Endpoint": fx.Endpoint,
		}},
		{id: "service-withdrawn", define: "service", data: map[string]any{
			"Title": fx.ServiceWithdrawn.Key, "NavActive": "inventory", "IsAdmin": true, "Service": fx.ServiceWithdrawn,
		}},
	}
	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/signals.tmpl", "templates/drift.tmpl", "templates/asset.tmpl", "templates/inventory.tmpl", "templates/subjectdetail.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := t.ExecuteTemplate(&buf, st.define, st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}
