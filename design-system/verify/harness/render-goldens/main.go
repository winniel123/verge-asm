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
	"github.com/winniel123/verge-asm/internal/message"
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
	screen := flag.String("screen", "inventory", "which screen to render: inventory | error | profile | signin | setup | coverage | exposure | drift | rundetail | scope | signals | dashboard | asset | subjectdetail | graph | reports | reportartifact | inbox | onboarding | firstrun | search")
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
	case "shell":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen shell")
		}
		files, err := renderShellStates(*bodyFlex)
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
	case "reports":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen reports")
		}
		files, err := renderReportsStates(*bodyFlex)
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
	case "settings":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen settings")
		}
		files, err := renderSettingsStates(*bodyFlex)
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
	case "reportartifact":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen reportartifact")
		}
		files, err := renderReportartifactStates(*bodyFlex)
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
	case "inbox":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen inbox")
		}
		files, err := renderInboxStates(*bodyFlex)
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
	case "onboarding":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen onboarding")
		}
		files, err := renderOnboardingStates(*bodyFlex)
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
	case "firstrun":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen firstrun")
		}
		files, err := renderFirstRunStates(*bodyFlex)
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
	case "search":
		if *outdir == "" {
			log.Fatal("render-goldens: -outdir is required for -screen search")
		}
		files, err := renderSearchStates(*bodyFlex)
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
	case "graph":
		if *out == "" {
			log.Fatal("render-goldens: -out is required for -screen graph")
		}
		html, err := renderGraph(*bodyFlex)
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
// two states.json rundetail states: default (the completed run at /runs/1407) and running (the
// live-tailing dispatch at /runs/1409, #35). The default data map mirrors runPage's
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

	// render composes one rundetail state through the frozen tmpl into golden HTML.
	render := func(d map[string]any) ([]byte, error) {
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
		if _, err := t.ParseFS(designfs.FS, "templates/rundetail.tmpl"); err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := execGolden(t, &buf, "run", d); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	defaultHTML, err := render(data)
	if err != nil {
		return nil, err
	}

	// The rundetail·running state (/runs/1409, #35): a live-tailing running dispatch, mirroring
	// cmd/web/devfixtures.go runningRunFixtureData EXACTLY — Status "running" (the accent LIVE
	// badge + streaming pulse), the six queue jobs folded through the same runStages/runLog/
	// runVantages the live path uses, the Outcome holes "—" (a running run's diff has not
	// concluded), no degraded callout, no job filter (the state carries no ?job=). So the cropped
	// `main` is byte-identical to what the seeded server renders for /runs/1409.
	jobs := rgRunningRunJobs
	running := map[string]any{
		"Title": "2026-08-22T14:00Z", "Status": "running", "Scope": "all scopes",
		"Meta": "standard profile · 3 vantages", "Transitions": "—", "NewSignals": "—",
		"Active": true, "Stages": rgRunStages(jobs), "Log": rgRunLog(jobs),
		"Vantages": rgRunVantages(jobs), "Degraded": nil,
		"Params": []map[string]any{
			{"K": "Profile", "V": "standard"},
			{"K": "Cadence", "V": "daily · 08:00 + 14:00"},
			{"K": "Dispatched", "V": "2026-08-22 14:00 UTC"},
			{"K": "Jobs", "V": "6"},
			{"K": "Vantages", "V": "3"},
		},
	}
	runningData := map[string]any{
		"Title": "batch 2026-08-22T14:00Z", "NavActive": "drift", "DesignTokens": true,
		"Refresh": 5, "Run": running,
	}
	runningHTML, err := render(runningData)
	if err != nil {
		return nil, err
	}

	return []errorGolden{
		{id: "default", html: defaultHTML},
		{id: "running", html: runningHTML},
	}, nil
}

// rgJob is the render-goldens mirror of cmd/web's jobView — only the fields the run folds
// read (kind, state, vantage, batch, and the retrying/superseded flags). The six pinned jobs
// below mirror cmd/web/devfixtures.go devRunningRunJobs one-for-one (the id split's #35
// running dispatch, fixtures.json → settings.scans.active[0].jobs at id 1409).
type rgJob struct {
	ID                   int64
	Kind, State          string
	Vantage, Batch       string
	Retrying, Superseded bool
}

var rgRunningRunJobs = []rgJob{
	{ID: 912, Kind: "dns-sweep", State: "done", Vantage: "eu-west-1", Batch: "1407"},
	{ID: 913, Kind: "reachability", State: "done", Vantage: "eu-west-1", Batch: "1407"},
	{ID: 914, Kind: "reachability", State: "done", Vantage: "us-east-2", Batch: "1407"},
	{ID: 915, Kind: "reachability", State: "ready", Vantage: "ap-south-1", Retrying: true},
	{ID: 916, Kind: "port-census", State: "running", Vantage: "eu-west-1"},
	{ID: 917, Kind: "tls-acceptance", State: "done", Vantage: "eu-west-1", Batch: "1407"},
}

// rgRunStages mirrors cmd/web/scans.go runStages: jobs grouped by kind in first-seen order,
// each stage done when nothing is in flight, current while a ready/running job remains.
func rgRunStages(jobs []rgJob) []map[string]any {
	var order []string
	idx := map[string]int{}
	type agg struct{ total, done, dead, inflight int }
	var aggs []agg
	for _, j := range jobs {
		if j.Superseded {
			continue
		}
		i, ok := idx[j.Kind]
		if !ok {
			i = len(order)
			idx[j.Kind] = i
			order = append(order, j.Kind)
			aggs = append(aggs, agg{})
		}
		aggs[i].total++
		switch j.State {
		case "done":
			aggs[i].done++
		case "dead":
			aggs[i].dead++
		case "ready", "running":
			aggs[i].inflight++
		}
	}
	out := make([]map[string]any, 0, len(order))
	for i, k := range order {
		a := aggs[i]
		detail := fmt.Sprintf("%d of %d done", a.done, a.total)
		if a.dead > 0 {
			detail += fmt.Sprintf(" · %d dead-lettered", a.dead)
		}
		out = append(out, map[string]any{
			"Num": i + 1, "Title": k, "Detail": detail,
			"Done": a.inflight == 0, "Current": a.inflight > 0, "Last": i == len(order)-1,
		})
	}
	return out
}

// rgRunLog mirrors cmd/web/scans.go runLog: one line per job, id as tag, a level from state
// (dead → error, superseded/retrying → warn), the terse kind · state · vantage · batch text.
func rgRunLog(jobs []rgJob) []map[string]any {
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		level := ""
		switch {
		case j.State == "dead":
			level = "error"
		case j.Superseded || j.Retrying:
			level = "warn"
		}
		text := j.Kind + " · " + j.State
		if j.Vantage != "" {
			text += " · " + j.Vantage
		}
		if j.Batch != "" {
			text += " · " + j.Batch
		}
		out = append(out, map[string]any{"Tag": "#" + strconv.FormatInt(j.ID, 10), "Level": level, "Text": text})
	}
	return out
}

// rgRunVantages mirrors cmd/web/scans.go runVantages: per-vantage health in first-seen order,
// degraded if any of its non-superseded jobs dead-lettered, latency unstored ("—").
func rgRunVantages(jobs []rgJob) []map[string]any {
	var order []string
	seen := map[string]bool{}
	dead := map[string]bool{}
	for _, j := range jobs {
		if j.Vantage == "" || j.Superseded {
			continue
		}
		if !seen[j.Vantage] {
			seen[j.Vantage] = true
			order = append(order, j.Vantage)
		}
		if j.State == "dead" {
			dead[j.Vantage] = true
		}
	}
	out := make([]map[string]any, 0, len(order))
	for _, n := range order {
		status := "ok"
		if dead[n] {
			status = "degraded"
		}
		out = append(out, map[string]any{"Name": n, "Latency": "—", "Status": status})
	}
	return out
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
		if err := execGolden(t, &buf, "coverage", st.data); err != nil {
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
		if err := execGolden(t, &buf, "exposure", st.data); err != nil {
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
	// DF-F1: the declare input paste-splits into one .Refusals[] callout per refused
	// token; this fixture pins a single over-cap token, so it renders as a one-element
	// list (mirrors cmd/web/devfixtures.go scopeFixtureDataRefusal, which sets
	// data["Refusals"] = []refusalView{ref}). Replaces the retired singular .Refusal.
	refusalData["Refusals"] = []map[string]any{{
		"Input":     fx.RefusalFixture.Input,
		"Reason":    fx.RefusalFixture.Reason,
		"Reachable": fx.RefusalFixture.Reachable,
	}}

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
		if err := execGolden(t, &buf, "scope", st.data); err != nil {
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
	if err := execGolden(t, &buf, "drift", data); err != nil {
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

// newStubbedTemplate returns a template set carrying the REAL design-owned shell
// (design-system/templates/shell.tmpl — "head"/"chrome"/"foot"/"cmdkicon"), so a
// screen render composes its FULL PAGE — head + TopNav chrome + main + footer —
// byte-for-byte the way the converted app serves it (#27f: with the shell landed the
// harness stops cropping to <main>; every landed screen re-renders full-page). The
// funcmap mirrors cmd/web/templates_shell.go: integrationsEnabled (the compile-time
// #388 flag — true, matching integrations.go), designTokens (the sorted-join of
// tokens/*.css the head inlines on EVERY page now), and signDelta. The `head` param
// is retained for call-site compatibility but is no longer used — the real shell.tmpl
// carries its own head.
func newStubbedTemplate(_ template.HTML) (*template.Template, error) {
	tokens, err := loadDesignTokens()
	if err != nil {
		return nil, err
	}
	t := template.New("root").Funcs(template.FuncMap{
		"integrationsEnabled": func() bool { return true },
		"designTokens":        func() template.CSS { return template.CSS(tokens) }, // #nosec G203 -- trusted design tokens from the embedded package
		// signDelta mirrors cmd/web/templates_shell.go byte-for-byte.
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
	if _, err := t.ParseFS(designfs.FS, "templates/shell.tmpl"); err != nil {
		return nil, err
	}
	return t, nil
}

// --- golden chrome (#27f) ----------------------------------------------------------
//
// The design-owned shell.tmpl "chrome"/"foot" read a single nullable .Chrome hole.
// For the full-page goldens the harness composes that .Chrome from the pinned
// fixtures.json shell slice — the SAME bytes cmd/web/chrome.go's chromeFromFixture
// reads in a VERGE_DEV candidate — so golden and candidate agree byte-for-byte. The
// struct mirrors cmd/web's chromeVM. The org chip is static — SPEC-CHANGE #33 retired
// the switcher permanently (ADR-0073 single-org); .Orgs is gone from the contract and
// the org-open golden is dropped (package v3.17.0).

// integrationsGolden mirrors the app's compile-time integrationsEnabled const
// (integrations.go = true): the gated Integrations palette item is emitted.
const integrationsGolden = true

type rgChrome struct {
	Nav           []rgNav
	Org           string
	Version       string
	UserName      string
	UserInitials  string
	ScanRunning   bool
	Unread        bool
	Messages      []rgMsg
	PaletteGroups []rgPaletteGroup
	Toasts        []rgToast
}
type rgNav struct {
	ID, Label, Href string
	Active          bool
	Count           string
}
type rgMsg struct {
	Class, Rel, Headline, Href string
	Unread                     bool
}
type rgPaletteGroup struct {
	Label string
	Items []rgPaletteItem
}
type rgPaletteItem struct {
	Label, Icon, Hint, Href string
	Search, ThemeToggle     bool
}
type rgToast struct {
	Tone, Title, Description string
}

// rgShellFixture mirrors cmd/web's shellFixture — the fixtures.json shell slice.
type rgShellFixture struct {
	Chrome struct {
		Nav []struct {
			ID     string `json:"id"`
			Label  string `json:"label"`
			Href   string `json:"href"`
			Active bool   `json:"active"`
			Count  string `json:"count"`
		} `json:"nav"`
		Org          string `json:"org"`
		Version      string `json:"version"`
		UserName     string `json:"user_name"`
		UserInitials string `json:"user_initials"`
		Unread       bool   `json:"unread"`
		Messages     []struct {
			Class    string `json:"class"`
			Rel      string `json:"rel"`
			Headline string `json:"headline"`
			Unread   bool   `json:"unread"`
			Href     string `json:"href"`
		} `json:"messages"`
		PaletteGroups []struct {
			Label string `json:"label"`
			Items []struct {
				Label       string `json:"label"`
				Icon        string `json:"icon"`
				Hint        string `json:"hint"`
				Href        string `json:"href"`
				Search      bool   `json:"search"`
				ThemeToggle bool   `json:"theme_toggle"`
				Gated       string `json:"gated"`
			} `json:"items"`
		} `json:"palette_groups"`
		ToastsVariant []struct {
			Tone        string `json:"tone"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"toasts_variant"`
	} `json:"chrome"`
}

func loadShellFixture() (rgShellFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return rgShellFixture{}, err
	}
	var ff struct {
		Shell rgShellFixture `json:"shell"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return rgShellFixture{}, err
	}
	return ff.Shell, nil
}

// goldenChrome composes the .Chrome view-model from the pinned shell slice, with the
// active pill set per navActive, scanning lighting .ScanRunning, and showToast folding
// in the toasts variant. The gated Integrations palette item is included (the app's
// integrationsEnabled const is true). The org chip is static (switcher retired, #33). It
// matches chromeFromFixture.
func goldenChrome(navActive string, scanning, showToast bool) *rgChrome {
	fx, err := loadShellFixture()
	if err != nil {
		log.Printf("render-goldens: shell fixture: %v", err)
	}
	c := &rgChrome{
		Org:          fx.Chrome.Org,
		Version:      fx.Chrome.Version,
		UserName:     fx.Chrome.UserName,
		UserInitials: fx.Chrome.UserInitials,
		ScanRunning:  scanning,
		Unread:       fx.Chrome.Unread,
	}
	for _, n := range fx.Chrome.Nav {
		c.Nav = append(c.Nav, rgNav{ID: n.ID, Label: n.Label, Href: n.Href, Active: n.ID == navActive, Count: n.Count})
	}
	for _, m := range fx.Chrome.Messages {
		c.Messages = append(c.Messages, rgMsg{Class: m.Class, Rel: m.Rel, Headline: m.Headline, Unread: m.Unread, Href: m.Href})
	}
	for _, g := range fx.Chrome.PaletteGroups {
		pg := rgPaletteGroup{Label: g.Label}
		for _, it := range g.Items {
			// The gated Integrations item is included: the app's integrationsEnabled
			// const is true (integrations.go), so the candidate emits it too.
			if it.Gated == "integrationsEnabled" && !integrationsGolden {
				continue
			}
			pg.Items = append(pg.Items, rgPaletteItem{Label: it.Label, Icon: it.Icon, Hint: it.Hint, Href: it.Href, Search: it.Search, ThemeToggle: it.ThemeToggle})
		}
		c.PaletteGroups = append(c.PaletteGroups, pg)
	}
	if showToast {
		for _, t := range fx.Chrome.ToastsVariant {
			c.Toasts = append(c.Toasts, rgToast{Tone: t.Tone, Title: t.Title, Description: t.Description})
		}
	}
	return c
}

// chromeInto stamps .Chrome onto a chrome-ful screen's golden data map, reading its
// own NavActive (each screen highlights its own pill) and its own Scanning flag (the
// dashboard's scanning state lights the chrome scan indicator). A map with no
// "NavActive" key (the chrome-less auth surfaces) is left untouched — the shell's
// {{with .Chrome}} then renders no chrome, exactly as the candidate does.
func chromeInto(data map[string]any) {
	nav, ok := data["NavActive"].(string)
	if !ok {
		return
	}
	scanning, _ := data["Scanning"].(bool)
	data["Chrome"] = goldenChrome(nav, scanning, false)
}

// execGolden is the single execute choke point for the full-page goldens: it stamps
// .Chrome onto any chrome-ful map (chromeInto is a no-op for chrome-less auth maps and
// non-map data) and then executes the named screen define into buf. Every render
// function routes its execute through here, so the full-page shell composes uniformly.
func execGolden(t *template.Template, buf *bytes.Buffer, name string, data any) error {
	if m, ok := data.(map[string]any); ok {
		chromeInto(m)
	}
	return t.ExecuteTemplate(buf, name, data)
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

	fx, err := loadFixture()
	if err != nil {
		return nil, err
	}
	// Full-page (#27f): wrap the inventory holes in a map so the shell chrome stamps on
	// (NavActive drives the active pill); the inventory.tmpl reads .HasData / .Groups.
	data := map[string]any{
		"Title": "Inventory", "NavActive": "inventory",
		"HasData": fx.HasData, "Groups": fx.Groups,
	}

	var buf bytes.Buffer
	if err := execGolden(t, &buf, "inventory", data); err != nil {
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
		if err := execGolden(t, &buf, "error-page", data[st.id]); err != nil {
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
			// Full-page (#27f): profile renders inside the chrome. NavActive is "" (no
			// nav pill is the Profile page) — matching auth.go's profile render — so
			// chromeInto stamps the chrome with no active pill.
			"NavActive":     "",
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
		if err := execGolden(t, &buf, "profile", data[id]); err != nil {
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
		if err := execGolden(t, &buf, st.tmpl, st.data); err != nil {
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
		if err := execGolden(t, &buf, "setup", st.data); err != nil {
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
		if err := execGolden(t, &buf, "signals", st.data); err != nil {
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
// dashboardData composes the dashboard "home" holes from the pinned dashboard+signals
// fixtures, per scanning / probe-dismissed. It mirrors cmd/web/devfixtures.go's
// dashboardFixtureData so golden and candidate agree byte-for-byte, and is shared by
// both the dashboard goldens and the shell-state goldens (which render `/` full-page).
func dashboardData(fx dashboardFixture, sfx signalsFixture, scanning, probeDismissed bool) map[string]any {
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

// renderShellStates composes the 6 captured shell-state goldens (#27f / SPEC-CHANGE
// #27), all on `/` (the dashboard) FULL-PAGE: default, palette-open, bell-open,
// acct-open (the popover states share the base HTML — capture.mjs opens each popover),
// scan-running (chrome .ScanRunning + the dashboard scanning content), and toasts (the
// fixture toast stack folded into .Chrome.Toasts). The org switcher is retired (SPEC-CHANGE
// #33, package v3.17.0): shell.tmpl renders only the static org chip, so there is no
// org-open state — it was dropped from states.json this round. Chrome is set explicitly (so
// the toasts state can carry showToast) rather than via chromeInto, and matches
// cmd/web/chrome.go's chromeFromFixture.
func renderShellStates(bodyFlex bool) ([]errorGolden, error) {
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
		return nil, fmt.Errorf("shell golden: signals.rows has %d rows, need >= 6", len(sfx.Rows))
	}
	states := []struct {
		id       string
		scanning bool
		toast    bool
	}{
		{"default", false, false},
		{"palette-open", false, false},
		{"bell-open", false, false},
		{"acct-open", false, false},
		{"scan-running", true, false},
		{"toasts", false, true},
	}
	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		data := dashboardData(fx, sfx, st.scanning, false)
		data["Chrome"] = goldenChrome("dashboard", st.scanning, st.toast)
		t, err := newStubbedTemplate(head)
		if err != nil {
			return nil, err
		}
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
		// Chrome is already set (with the per-state showToast), so execute directly
		// rather than through execGolden (whose chromeInto would drop the toast).
		if err := t.ExecuteTemplate(&buf, "home", data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

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

	states := []struct {
		id   string
		data map[string]any
	}{
		{"default", dashboardData(fx, sfx, false, false)},
		{"scanning", dashboardData(fx, sfx, true, false)},
		{"banner-dismissed", dashboardData(fx, sfx, false, true)},
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
		if err := execGolden(t, &buf, "home", st.data); err != nil {
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
	if err := execGolden(t, &buf, "asset", data); err != nil {
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
		if err := execGolden(t, &buf, st.define, st.data); err != nil {
			return nil, err
		}
		out = append(out, errorGolden{id: st.id, html: buf.Bytes()})
	}
	return out, nil
}

// graphFixture is the design-system/fixtures/fixtures.json → graph slice: the whole rendered graph
// (empty marker, canvas + minimap dims, the placed nodes and resolved edges). Its Go field names ARE
// the holes the frozen graph.tmpl reads (.Graph.{Empty,ViewW,ViewH,MiniW,MiniH,Nodes,Edges}), so a
// loaded value passes straight to the template. cmd/web/devfixtures.go pins the same values with a
// drift test (TestGraphFixtureMatchesPackage). The three golden states (default · node-drawer ·
// filtered-critical) are pure client-JS variants of this ONE server render, so this is a single
// shared golden HTML (--page, like drift) that states.json's JS drives on BOTH sides.
type graphFixture struct {
	Empty bool               `json:"empty"`
	ViewW int                `json:"view_w"`
	ViewH int                `json:"view_h"`
	MiniW int                `json:"mini_w"`
	MiniH int                `json:"mini_h"`
	Nodes []graphNodeFixture `json:"nodes"`
	Edges []graphEdgeFixture `json:"edges"`
}

type graphNodeFixture struct {
	ID          string               `json:"id"`
	Label       string               `json:"label"`
	Type        string               `json:"type"`
	X           int                  `json:"x"`
	Y           int                  `json:"y"`
	Mx          float64              `json:"mx"`
	My          float64              `json:"my"`
	Sev         string               `json:"sev"`
	HaloA       float64              `json:"halo_a"`
	HaloB       float64              `json:"halo_b"`
	LabelDX     int                  `json:"label_dx"`
	Ports       string               `json:"ports"`
	First       string               `json:"first"`
	OpenSignals []graphSignalFixture `json:"open_signals"`
}

type graphSignalFixture struct {
	Severity string `json:"severity"`
	SevLabel string `json:"sev_label"`
	Rule     string `json:"rule"`
	Subject  string `json:"subject"`
}

type graphEdgeFixture struct {
	X1        int  `json:"x1"`
	Y1        int  `json:"y1"`
	X2        int  `json:"x2"`
	Y2        int  `json:"y2"`
	ToService bool `json:"to_service"`
}

func loadGraphFixture() (graphFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return graphFixture{}, err
	}
	var ff struct {
		Graph graphFixture `json:"graph"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return graphFixture{}, err
	}
	return ff.Graph, nil
}

// renderGraph composes the Graph golden HTML from the frozen graph.tmpl, for the states.json graph
// states (default · node-drawer · filtered-critical). The data map mirrors graphPage's
// graphFixtureData EXACTLY (the .Graph holes the frozen tmpl reads) — the pinned fixtures.json graph
// slice — so the cropped `main`/`body` is byte-identical to what the seeded server renders. "graph"
// wraps head/chrome/graph/foot; it reuses the "sevbadge" define signals.tmpl declares (the drawer's
// sev-badged signal rows) — parsed into the set here. Chrome is the empty stub (default/filtered-
// critical crop `main`; the node-drawer state crops `body`, since the fixed scrim + drawer escape it).
func renderGraph(bodyFlex bool) ([]byte, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadGraphFixture()
	if err != nil {
		return nil, err
	}
	data := map[string]any{
		"Title": "Graph", "NavActive": "graph", "IsAdmin": true,
		"Graph": fx,
	}
	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/signals.tmpl", "templates/graph.tmpl"); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := execGolden(t, &buf, "graph", data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// --- screen 16: Reports (reports.tmpl, package v3.11.0, WORK-ORDER-16-18-BATCH5.md) -----------
//
// renderReportsStates composes the seven Reports golden HTMLs from the frozen reports.tmpl. The
// default / range-open / row-menu-open states are the SAME "reports" page HTML (their difference is
// the frozen tmpl's own JS — open the range popover / open a row kebab — driven over it by
// capture.mjs on BOTH sides, states.json). The wizard-1..4 states each execute the "schedulewizard"
// define for the PRG step the states.json GET URL addresses. Every data map mirrors reportsPage /
// reportsWizardFixtureData EXACTLY, read from the SAME fixtures.json reports slice (numbers ride as
// json.Number, heat backgrounds as template.CSS), so the cropped `main` is byte-identical to what
// the seeded server renders. Chrome is the empty stub (goldens crop to `main`).

type reportsFixtureDelta struct {
	Has  bool   `json:"has"`
	Text string `json:"text"`
	Dir  string `json:"dir"`
	Tone string `json:"tone"`
}

type reportsFixtureSpark struct {
	W     json.Number `json:"w"`
	H     json.Number `json:"h"`
	Area  string      `json:"area"`
	Line  string      `json:"line"`
	Color string      `json:"color"`
	DotX  json.Number `json:"dot_x"`
	DotY  json.Number `json:"dot_y"`
}

type reportsFixtureBar struct {
	HeightPct json.Number `json:"height_pct"`
	Title     string      `json:"title"`
	Last      bool        `json:"last"`
}

type reportsFixtureBars struct {
	Bars       []reportsFixtureBar `json:"bars"`
	LeftLabel  string              `json:"left_label"`
	RightLabel string              `json:"right_label"`
}

type reportsFixtureGrid struct {
	Y      json.Number `json:"y"`
	X1     json.Number `json:"x1"`
	X2     json.Number `json:"x2"`
	Stroke string      `json:"stroke"`
	LabelX json.Number `json:"label_x"`
	Label  string      `json:"label"`
}

type reportsFixtureXLabel struct {
	X    json.Number `json:"x"`
	Y    json.Number `json:"y"`
	Text string      `json:"text"`
}

type reportsFixtureSeries struct {
	W          json.Number            `json:"w"`
	H          json.Number            `json:"h"`
	N          json.Number            `json:"n"`
	Grid       []reportsFixtureGrid   `json:"grid"`
	AllOpen    string                 `json:"all_open"`
	CritHigh   string                 `json:"crit_high"`
	XLabels    []reportsFixtureXLabel `json:"x_labels"`
	LabelsAttr string                 `json:"labels_attr"`
	SeriesJSON string                 `json:"series_json"`
}

type reportsFixtureSev struct {
	Sev   string      `json:"sev"`
	Label string      `json:"label"`
	Pct   json.Number `json:"pct"`
	Count json.Number `json:"count"`
}

type reportsFixtureHeat struct {
	Title  string       `json:"title"`
	Bg     template.CSS `json:"bg"`
	Border template.CSS `json:"border"`
}

type reportsFixtureSchedule struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Cadence      string      `json:"cadence"`
	Delivery     string      `json:"delivery"`
	Format       string      `json:"format"`
	LastSent     string      `json:"last_sent"`
	LastMins     json.Number `json:"last_mins"`
	HasDelivery  bool        `json:"has_delivery"`
	DeliveryHref string      `json:"delivery_href"`
}

type reportsFixturePeriod struct {
	Token string `json:"token"`
	Label string `json:"label"`
}

type reportsFixtureWizardSection struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Checked bool   `json:"checked"`
}

type reportsFixtureWizardChannel struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
}

type reportsFixtureWizard struct {
	Title       string                        `json:"title"`
	FormAction  string                        `json:"form_action"`
	FinishLabel string                        `json:"finish_label"`
	Steps       []string                      `json:"steps"`
	Sections    []reportsFixtureWizardSection `json:"sections"`
	Cads        []string                      `json:"cads"`
	DefaultCad  string                        `json:"default_cad"`
	Channels    []reportsFixtureWizardChannel `json:"channels"`
}

type reportsFixture struct {
	Period            string                   `json:"period"`
	PeriodLabel       string                   `json:"period_label"`
	Periods           []reportsFixturePeriod   `json:"periods"`
	RangeLabel        string                   `json:"range_label"`
	RangeWeeks        json.Number              `json:"range_weeks"`
	HasOpenSignals    bool                     `json:"has_open_signals"`
	OpenSignals       string                   `json:"open_signals"`
	OpenDelta         reportsFixtureDelta      `json:"open_delta"`
	HasOpenSpark      bool                     `json:"has_open_spark"`
	OpenSpark         reportsFixtureSpark      `json:"open_spark"`
	HasDiscovery      bool                     `json:"has_discovery"`
	DiscoveryCount    string                   `json:"discovery_count"`
	DiscoveryDelta    reportsFixtureDelta      `json:"discovery_delta"`
	DiscoveryNames    json.Number              `json:"discovery_names"`
	DiscoveryServices json.Number              `json:"discovery_services"`
	DiscoveryBars     reportsFixtureBars       `json:"discovery_bars"`
	HasMTTW           bool                     `json:"has_mttw"`
	MTTW              string                   `json:"mttw"`
	MTTWDelta         reportsFixtureDelta      `json:"mttw_delta"`
	HasMTTWSpark      bool                     `json:"has_mttw_spark"`
	MTTWSpark         reportsFixtureSpark      `json:"mttw_spark"`
	HasSignalSeries   bool                     `json:"has_signal_series"`
	SignalSeries      reportsFixtureSeries     `json:"signal_series"`
	HasSeverity       bool                     `json:"has_severity"`
	BySeverity        []reportsFixtureSev      `json:"by_severity"`
	HasHeat           bool                     `json:"has_heat"`
	Heat              []reportsFixtureHeat     `json:"heat"`
	Schedules         []reportsFixtureSchedule `json:"schedules"`
	Wizard            reportsFixtureWizard     `json:"wizard"`
}

func loadReportsFixture() (reportsFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return reportsFixture{}, err
	}
	var ff struct {
		Reports reportsFixture `json:"reports"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return reportsFixture{}, err
	}
	return ff.Reports, nil
}

// reportsPageData mirrors reportsFixtureData (cmd/web/devfixtures.go) EXACTLY: the pinned reports
// slice passed straight through to the "reports" holes.
func reportsPageData(fx reportsFixture) map[string]any {
	return map[string]any{
		"Title": "Reports", "NavActive": "reports", "DesignTokens": true,
		"RangeLabel":  fx.RangeLabel,
		"RangeWeeks":  fx.RangeWeeks,
		"Periods":     fx.Periods,
		"Period":      fx.Period,
		"PeriodLabel": fx.PeriodLabel,

		"HasOpenSignals": fx.HasOpenSignals,
		"OpenSignals":    fx.OpenSignals,
		"OpenDelta":      fx.OpenDelta,
		"HasOpenSpark":   fx.HasOpenSpark,
		"OpenSpark":      fx.OpenSpark,

		"HasDiscovery":      fx.HasDiscovery,
		"DiscoveryCount":    fx.DiscoveryCount,
		"DiscoveryDelta":    fx.DiscoveryDelta,
		"DiscoveryNames":    fx.DiscoveryNames,
		"DiscoveryServices": fx.DiscoveryServices,
		"DiscoveryBars":     fx.DiscoveryBars,

		"HasMTTW":      fx.HasMTTW,
		"MTTW":         fx.MTTW,
		"MTTWDelta":    fx.MTTWDelta,
		"HasMTTWSpark": fx.HasMTTWSpark,
		"MTTWSpark":    fx.MTTWSpark,

		"HasSignalSeries": fx.HasSignalSeries,
		"SignalSeries":    fx.SignalSeries,

		"HasSeverity": fx.HasSeverity,
		"BySeverity":  fx.BySeverity,

		"HasHeat": fx.HasHeat,
		"Heat":    fx.Heat,

		"Schedules": fx.Schedules,
	}
}

// reportsCustomCad / reportsScheduleFmt mirror the cmd/web reports_schedule.go constants the wizard
// review row uses (a Custom cadence carries its cron; the delivered format is a fixed pdf).
const (
	reportsCustomCad   = "Custom…"
	reportsScheduleFmt = "pdf"
)

// reportsCadLabel mirrors cmd/web/reports_schedule.go reportCadLabel byte-for-byte: the stored
// cadence label (a custom cadence stores its cron or "custom"; otherwise the lower-cased preset).
func reportsCadLabel(cad, cron string) string {
	if cad == reportsCustomCad {
		if c := strings.TrimSpace(cron); c != "" {
			return c
		}
		return "custom"
	}
	return strings.ToLower(cad)
}

// reportsWizardMap mirrors cmd/web/devfixtures.go reportsWizardMap EXACTLY: it reconstructs the
// wizard's controlled state from the query (the PRG GET URL each wizard state addresses) using the
// fixture vocabulary and stamps every "schedulewizard" hole. Account/IsAdmin are omitted — the
// golden's chrome/head are the empty stubs, and the wizard body never reads them.
func reportsWizardMap(fx reportsFixtureWizard, q map[string][]string) map[string]any {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	step := 0
	if n, err := strconv.Atoi(get("step")); err == nil {
		step = n
	}
	if step < 0 {
		step = 0
	}
	if step > len(fx.Steps)-1 {
		step = len(fx.Steps) - 1
	}

	selected := map[string]bool{}
	if raw, ok := q["sections"]; ok {
		for _, k := range raw {
			selected[k] = true
		}
	} else {
		for _, sec := range fx.Sections {
			if sec.Checked {
				selected[sec.Key] = true
			}
		}
	}
	sectionOpts := make([]map[string]any, 0, len(fx.Sections))
	orderedKeys := make([]string, 0, len(fx.Sections))
	labels := make([]string, 0, len(fx.Sections))
	for _, sec := range fx.Sections {
		on := selected[sec.Key]
		sectionOpts = append(sectionOpts, map[string]any{"Key": sec.Key, "Label": sec.Label, "Checked": on})
		if on {
			orderedKeys = append(orderedKeys, sec.Key)
			labels = append(labels, sec.Label)
		}
	}

	cad := get("cad")
	if cad == "" {
		cad = fx.DefaultCad
	}
	cron := get("cron")
	cads := make([]map[string]any, 0, len(fx.Cads))
	for _, c := range fx.Cads {
		cads = append(cads, map[string]any{"Value": c, "Selected": c == cad})
	}

	channel := get("channel")
	if channel == "" && len(fx.Channels) > 0 {
		channel = fx.Channels[0].Value
	}
	channelLabel := ""
	channels := make([]map[string]any, 0, len(fx.Channels))
	for _, c := range fx.Channels {
		sel := c.Value == channel
		if sel {
			channelLabel = c.Label
		}
		channels = append(channels, map[string]any{"Value": c.Value, "Label": c.Label, "Hint": c.Hint, "Selected": sel})
	}

	steps := make([]map[string]any, 0, len(fx.Steps))
	for i, title := range fx.Steps {
		steps = append(steps, map[string]any{"Num": i + 1, "Title": title, "Done": i < step, "Current": i == step})
	}

	nameSummary := get("name")
	if nameSummary == "" {
		nameSummary = "—"
	}
	sectionsSummary := "—"
	if len(labels) > 0 {
		sectionsSummary = strings.Join(labels, ", ")
	}
	review := []map[string]any{
		{"K": "Report", "V": nameSummary},
		{"K": "Sections", "V": sectionsSummary},
		{"K": "Cadence", "V": reportsCadLabel(cad, cron)},
		{"K": "Format", "V": reportsScheduleFmt},
		{"K": "Delivery", "V": channelLabel},
	}

	last := step == len(fx.Steps)-1
	return map[string]any{
		"Title": fx.Title, "NavActive": "reports", "DesignTokens": true,

		"WizardTitle": fx.Title,
		"FormAction":  fx.FormAction,
		"FinishLabel": fx.FinishLabel,
		"EditMode":    false,
		"ID":          int64(0),

		"Step":      step,
		"StepNum":   step + 1,
		"StepTotal": len(fx.Steps),
		"Last":      last,
		"Steps":     steps,

		"Name":         get("name"),
		"Sections":     sectionOpts,
		"SectionsKeys": orderedKeys,
		"Cads":         cads,
		"Cad":          cad,
		"Cron":         cron,
		"Custom":       cad == reportsCustomCad,
		"Channels":     channels,
		"ChannelID":    channel,
		"ChannelLabel": channelLabel,

		"Review": review,
	}
}

func renderReportsStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadReportsFixture()
	if err != nil {
		return nil, err
	}

	exec := func(define string, data map[string]any) ([]byte, error) {
		t, terr := newStubbedTemplate(head)
		if terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS, "templates/reports.tmpl"); terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, define, data); terr != nil {
			return nil, terr
		}
		return buf.Bytes(), nil
	}

	// The page states (default / range-open / row-menu-open) are one HTML; capture.mjs drives the
	// frozen tmpl's own JS to reach the popover/menu-open forms on both sides.
	page, err := exec("reports", reportsPageData(fx))
	if err != nil {
		return nil, err
	}

	// The wizard states hit the PRG GET URLs directly (states.json); the accumulated query for each
	// step matches those URLs exactly.
	wizardQueries := []map[string][]string{
		{},
		{"step": {"1"}, "name": {"Weekly exposure summary"}, "sections": {"kpis", "new-assets", "signal-changes"}},
		{"step": {"2"}, "name": {"Weekly exposure summary"}, "sections": {"kpis", "new-assets", "signal-changes"}, "cad": {"Weekly · mon 09:00"}},
		{"step": {"3"}, "name": {"Weekly exposure summary"}, "sections": {"kpis", "new-assets", "signal-changes"}, "cad": {"Weekly · mon 09:00"}, "channel": {"ops"}},
	}
	out := []errorGolden{
		{id: "default", html: page},
		{id: "range-open", html: page},
		{id: "row-menu-open", html: page},
	}
	for i, q := range wizardQueries {
		wh, werr := exec("schedulewizard", reportsWizardMap(fx.Wizard, q))
		if werr != nil {
			return nil, werr
		}
		out = append(out, errorGolden{id: "wizard-" + strconv.Itoa(i+1), html: wh})
	}
	return out, nil
}

// --- screen 20: Onboarding (onboarding.tmpl, package v3.12.0, WORK-ORDER-19-20-BATCH6) ---------
//
// renderOnboardingStates composes the four Onboarding golden HTMLs from the frozen onboarding.tmpl,
// one per states.json wizard-N state. Each state is the PRG GET URL that step addresses (#25d): the
// query is reconstructed into the controlled view and shaped into the "onboarding" holes EXACTLY as
// cmd/web/onboarding.go's readOnboardView + renderOnboard do — the seeds accumulate, the profile /
// cadence default to standard / "Daily · 08:00", .StepValid computes the no-JS Next/Start gate, and
// the Review step maps the real inputs (lowercased cadence, "none — inbox only" for an empty
// channel). The onboarding fixture (steps / cads / default cad) is read from the SAME fixtures.json
// slice, so a fixture change flows through. Chrome/head are the empty stubs (goldens crop to `main`).

// onboardingFixture is the fixtures.json → onboarding slice the golden reads (the step titles, the
// cadence presets and the default cad); the per-state values ride the states.json query.
type onboardingFixture struct {
	Steps      []string `json:"steps"`
	Cads       []string `json:"cads"`
	DefaultCad string   `json:"default_cad"`
}

func loadOnboardingFixture() (onboardingFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return onboardingFixture{}, err
	}
	var ff struct {
		Onboarding onboardingFixture `json:"onboarding"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return onboardingFixture{}, err
	}
	return ff.Onboarding, nil
}

const onboardingCustomCad = "Custom…"

// onboardingSeedTokens mirrors cmd/web/onboarding.go parseSeedTokens: split a raw seed entry on
// commas and whitespace, dropping empties.
func onboardingSeedTokens(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

// onboardingGoldenMap mirrors cmd/web/onboarding.go renderOnboard EXACTLY: it reconstructs the
// controlled state from the wizard-N query and stamps every "onboarding" hole the frozen tmpl reads.
// Account/IsAdmin are omitted — the golden's chrome/head are the empty stubs, and the body never
// reads them.
func onboardingGoldenMap(fx onboardingFixture, q map[string][]string) map[string]any {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	last := len(fx.Steps) - 1
	step := 0
	if n, err := strconv.Atoi(get("step")); err == nil {
		step = n
	}
	if step < 0 {
		step = 0
	}
	if step > last {
		step = last
	}

	// Seeds accumulate (seeds + seedsadd), deduped in first-seen order — mirroring readOnboardView.
	seen := map[string]bool{}
	var seeds []string
	for _, tok := range append(onboardingSeedTokens(get("seeds")), onboardingSeedTokens(get("seedsadd"))...) {
		if !seen[tok] {
			seen[tok] = true
			seeds = append(seeds, tok)
		}
	}

	profile := get("profile")
	if profile != "passive" {
		profile = "standard"
	}
	cad := get("cad")
	if cad == "" {
		cad = fx.DefaultCad
	}
	cron := strings.TrimSpace(get("cron"))
	channel := get("channel")

	steps := make([]map[string]any, 0, len(fx.Steps))
	for i, title := range fx.Steps {
		steps = append(steps, map[string]any{"Num": i + 1, "Title": title, "Done": i < step, "Current": i == step})
	}
	cads := make([]map[string]any, 0, len(fx.Cads))
	for _, p := range fx.Cads {
		cads = append(cads, map[string]any{"Value": p, "Selected": p == cad})
	}

	// StepValid mirrors onboardStepValid: step 0 needs ≥1 seed; step 1 needs a cron when Custom…;
	// else valid.
	stepValid := true
	switch step {
	case 0:
		stepValid = len(seeds) > 0
	case 1:
		stepValid = cad != onboardingCustomCad || cron != ""
	}

	// Review summary mirrors renderOnboard: seeds joined (or an em dash), the profile, the cadence
	// (the cron when Custom…, else the lowercased preset), the channel (or inbox-only).
	seedsSummary := "—"
	if len(seeds) > 0 {
		seedsSummary = strings.Join(seeds, ", ")
	}
	cadence := strings.ToLower(cad)
	if cad == onboardingCustomCad {
		cadence = cron
	}
	channelSummary := strings.TrimSpace(channel)
	if channelSummary == "" {
		channelSummary = "none — inbox only"
	}
	review := []map[string]any{
		{"K": "Seeds", "V": seedsSummary},
		{"K": "Profile", "V": profile},
		{"K": "Cadence", "V": cadence},
		{"K": "Channel", "V": channelSummary},
	}

	kind := "hot"
	if profile == "passive" {
		kind = "dns"
	}

	return map[string]any{
		"Title": "Set up this workspace", "NavActive": "", "DesignTokens": true,

		"Step":      step,
		"StepNum":   step + 1,
		"StepTotal": len(fx.Steps),
		"Last":      step == last,
		"Steps":     steps,
		"StepValid": stepValid,

		"Seeds":      seeds,
		"SeedsField": strings.Join(seeds, ","),
		"Profile":    profile,
		"Cads":       cads,
		"Cad":        cad,
		"Cron":       cron,
		"Custom":     cad == onboardingCustomCad,
		"Channel":    channel,

		"Review": review,
		"Kind":   kind,
	}
}

func renderOnboardingStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadOnboardingFixture()
	if err != nil {
		return nil, err
	}

	exec := func(data map[string]any) ([]byte, error) {
		t, terr := newStubbedTemplate(head)
		if terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS, "templates/onboarding.tmpl"); terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, "onboarding", data); terr != nil {
			return nil, terr
		}
		return buf.Bytes(), nil
	}

	// The wizard states hit the PRG GET URLs directly (states.json); the accumulated query for each
	// step matches those URLs exactly.
	wizardQueries := []map[string][]string{
		{},
		{"step": {"1"}, "seeds": {"acmecorp.io"}},
		{"step": {"2"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"}},
		{"step": {"3"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"}, "channel": {"https://ops.example/hook"}},
	}
	out := make([]errorGolden, 0, len(wizardQueries))
	for i, q := range wizardQueries {
		h, herr := exec(onboardingGoldenMap(fx, q))
		if herr != nil {
			return nil, herr
		}
		out = append(out, errorGolden{id: "wizard-" + strconv.Itoa(i+1), html: h})
	}
	return out, nil
}

// --- screen 20: FirstRun (firstrun.tmpl, package v3.12.0, WORK-ORDER-19-20-BATCH6) -------------
//
// renderFirstRunStates composes the one FirstRun golden HTML — the empty-estate wrap of `/`. The
// bare "firstrun" define is wrapped by dashboard.tmpl's "home" define when .EmptyEstate is true, so
// "home" is executed with EmptyEstate lit over a template set carrying dashboard.tmpl + firstrun.tmpl
// (+ signals.tmpl for the "sevbadge" the dashboard body references but does not render here). Every
// hole mirrors firstRunFixtureData (cmd/web/devfixtures.go) EXACTLY, read from the SAME fixtures.json
// firstrun slice, so the cropped `main` is byte-identical to what the seeded server renders.

type firstRunGoldenFixture struct {
	FirstRunDone  int `json:"first_run_done"`
	FirstRunSteps []struct {
		Num         int    `json:"num"`
		Done        bool   `json:"done"`
		Title       string `json:"title"`
		Detail      string `json:"detail"`
		HasAction   bool   `json:"has_action"`
		ActionLabel string `json:"action_label"`
		ActionHref  string `json:"action_href"`
		ActionPost  string `json:"action_post"`
		Gated       bool   `json:"gated"`
		GateTitle   string `json:"gate_title"`
	} `json:"first_run_steps"`
}

func loadFirstRunGoldenFixture() (firstRunGoldenFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return firstRunGoldenFixture{}, err
	}
	var ff struct {
		FirstRun firstRunGoldenFixture `json:"firstrun"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return firstRunGoldenFixture{}, err
	}
	return ff.FirstRun, nil
}

func renderFirstRunStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadFirstRunGoldenFixture()
	if err != nil {
		return nil, err
	}

	steps := make([]map[string]any, 0, len(fx.FirstRunSteps))
	for _, st := range fx.FirstRunSteps {
		steps = append(steps, map[string]any{
			"Num": st.Num, "Done": st.Done, "Title": st.Title, "Detail": st.Detail,
			"HasAction": st.HasAction, "ActionLabel": st.ActionLabel, "ActionHref": st.ActionHref,
			"ActionPost": st.ActionPost, "Gated": st.Gated, "GateTitle": st.GateTitle,
		})
	}
	data := map[string]any{
		"Title": "Dashboard", "NavActive": "dashboard", "DesignTokens": true, "IsAdmin": true,
		"EmptyEstate":   true,
		"FirstRunDone":  fx.FirstRunDone,
		"FirstRunSteps": steps,
	}

	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	// "home" references "sevbadge" (signals.tmpl) via the dashboard body; parse it so the escaper's
	// static walk resolves, even though the empty-estate branch renders "firstrun", not the body.
	if _, err := t.ParseFS(designfs.FS, "templates/signals.tmpl"); err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/firstrun.tmpl"); err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/dashboard.tmpl"); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := execGolden(t, &buf, "home", data); err != nil {
		return nil, err
	}
	return []errorGolden{{id: "default", html: buf.Bytes()}}, nil
}

// --- screen 17: ReportArtifact (reportartifact.tmpl, package v3.11.0, WORK-ORDER-16-18-BATCH5) --
//
// renderReportartifactStates composes the two ReportArtifact golden HTMLs from the frozen
// reportartifact.tmpl. The page define "reportartifact" calls "artifactdoc", which in turn calls
// "deltachip" (reports.tmpl), "sevbadge" (signals.tmpl) and "changeglyph" (drift.tmpl) — one parse
// set — so all four tmpls are parsed into the stubbed template. Every data map mirrors
// reportartifactFixtureData (cmd/web/devfixtures.go) EXACTLY, read from the SAME fixtures.json
// reportartifact slice unmarshalled straight into message.ArtifactDoc, so the cropped `main` is
// byte-identical to what the seeded server renders. Chrome is the empty stub (goldens crop to `main`).

type reportartifactFixtureVariant struct {
	Period     string              `json:"period"`
	ScheduleID string              `json:"schedule_id"`
	Doc        message.ArtifactDoc `json:"doc"`
}

type reportartifactFixture struct {
	Heading        string                       `json:"heading"`
	Period         string                       `json:"period"`
	ScheduleID     string                       `json:"schedule_id"`
	Doc            message.ArtifactDoc          `json:"doc"`
	NeverDelivered reportartifactFixtureVariant `json:"never_delivered_variant"`
}

func loadReportartifactFixture() (reportartifactFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return reportartifactFixture{}, err
	}
	var ff struct {
		ReportArtifact reportartifactFixture `json:"reportartifact"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return reportartifactFixture{}, err
	}
	return ff.ReportArtifact, nil
}

// reportartifactVariantHeading mirrors cmd/web/devfixtures.go: the never-delivered variant's
// heading is the honest name of its schedule (s2), resolved from the reports schedules fixture.
func reportartifactVariantHeading(scheduleID string) string {
	fx, err := loadReportsFixture()
	if err != nil {
		return "Report delivery"
	}
	for _, sc := range fx.Schedules {
		if sc.ID == scheduleID {
			return sc.Name
		}
	}
	return "Report delivery"
}

// reportartifactPageData mirrors reportartifactFixtureData (cmd/web/devfixtures.go): the pinned
// slice passed straight through to the "reportartifact" holes for the given variant. Account/IsAdmin
// are omitted — the golden's chrome/head are the empty stubs, and the page body never reads them.
func reportartifactPageData(fx reportartifactFixture, variant string) map[string]any {
	heading, period, scheduleID, doc := fx.Heading, fx.Period, fx.ScheduleID, fx.Doc
	if variant == "never-delivered" {
		heading = reportartifactVariantHeading(fx.NeverDelivered.ScheduleID)
		period = fx.NeverDelivered.Period
		scheduleID = fx.NeverDelivered.ScheduleID
		doc = fx.NeverDelivered.Doc
	}
	var scheduleHole any
	if scheduleID != "" {
		scheduleHole = scheduleID
	}
	return map[string]any{
		"Title": "Report delivery", "NavActive": "reports", "DesignTokens": true,
		"Heading":    heading,
		"Period":     period,
		"ScheduleID": scheduleHole,
		"Doc":        doc,
	}
}

func renderReportartifactStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadReportartifactFixture()
	if err != nil {
		return nil, err
	}

	exec := func(data map[string]any) ([]byte, error) {
		t, terr := newStubbedTemplate(head)
		if terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS,
			"templates/reportartifact.tmpl",
			"templates/reports.tmpl",
			"templates/signals.tmpl",
			"templates/drift.tmpl",
		); terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, "reportartifact", data); terr != nil {
			return nil, terr
		}
		return buf.Bytes(), nil
	}

	def, err := exec(reportartifactPageData(fx, ""))
	if err != nil {
		return nil, err
	}
	nev, err := exec(reportartifactPageData(fx, "never-delivered"))
	if err != nil {
		return nil, err
	}
	return []errorGolden{
		{id: "default", html: def},
		{id: "never-delivered", html: nev},
	}, nil
}

// --- screen 18: Inbox (inbox.tmpl, package v3.11.1, WORK-ORDER-16-18-BATCH5) -----------------
//
// renderInboxStates composes the three Inbox golden HTMLs from the frozen inbox.tmpl, one per
// states.json inbox state (default /inbox, message-open /inbox?id=m1, unread-filter
// /inbox?filter=unread). Every data map mirrors inboxFixtureData (cmd/web/devfixtures.go) EXACTLY,
// read from the SAME fixtures.json inbox slice, so the cropped `main` is byte-identical to what the
// seeded server renders. Per SPEC-CHANGE #24 (ruled) there is no .Body hole — the detail form is the
// census + delivery receipts (the failed one flagged undelivered). The "inbox" define calls no
// cross-tmpl define, so only inbox.tmpl is parsed. Chrome is the empty stub (goldens crop to `main`).

type inboxFixture struct {
	Unread     int    `json:"unread"`
	Filter     string `json:"filter"`
	AllHref    string `json:"all_href"`
	UnreadHref string `json:"unread_href"`
	Messages   []struct {
		ID       string `json:"id"`
		Read     bool   `json:"read"`
		Cls      string `json:"cls"`
		Instant  string `json:"instant"`
		Rel      string `json:"rel"`
		Headline string `json:"headline"`
	} `json:"messages"`
	Selected struct {
		ID       string `json:"id"`
		Cls      string `json:"cls"`
		Headline string `json:"headline"`
		Rel      string `json:"rel"`
		Instant  string `json:"instant"`
		Census   []struct {
			Kind string `json:"kind"`
			Key  string `json:"key"`
			Href string `json:"href"`
		} `json:"census"`
		Deliveries []struct {
			State       string `json:"state"`
			ChannelHost string `json:"channel_host"`
			Failed      bool   `json:"failed"`
			LastError   string `json:"last_error"`
		} `json:"deliveries"`
		Href      string `json:"href"`
		JumpLabel string `json:"jump_label"`
	} `json:"selected_fixture"`
}

func loadInboxFixture() (inboxFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return inboxFixture{}, err
	}
	var ff struct {
		Inbox inboxFixture `json:"inbox"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return inboxFixture{}, err
	}
	return ff.Inbox, nil
}

// inboxStateData mirrors cmd/web/devfixtures.go inboxFixtureData byte-for-byte for one (id, filter)
// selection: the read-only fixture, the id marking a row .Selected and picking the detail, the
// unread filter trimming the list. Keeping this identical to the candidate is the point.
func inboxStateData(fx inboxFixture, selID, filter string) map[string]any {
	messages := make([]map[string]any, 0, len(fx.Messages))
	for _, m := range fx.Messages {
		if filter == "unread" && m.Read {
			continue
		}
		messages = append(messages, map[string]any{
			"ID":       m.ID,
			"Read":     m.Read,
			"Selected": selID != "" && m.ID == selID,
			"Class":    m.Cls,
			"Instant":  m.Instant,
			"Rel":      m.Rel,
			"Headline": m.Headline,
		})
	}

	var selected map[string]any
	if selID != "" && selID == fx.Selected.ID {
		census := make([]map[string]any, 0, len(fx.Selected.Census))
		for _, c := range fx.Selected.Census {
			census = append(census, map[string]any{"Kind": c.Kind, "Key": c.Key, "Href": c.Href})
		}
		deliveries := make([]map[string]any, 0, len(fx.Selected.Deliveries))
		for _, d := range fx.Selected.Deliveries {
			deliveries = append(deliveries, map[string]any{
				"State": d.State, "ChannelHost": d.ChannelHost, "Failed": d.Failed, "LastError": d.LastError,
			})
		}
		selected = map[string]any{
			"ID":         fx.Selected.ID,
			"Class":      fx.Selected.Cls,
			"Headline":   fx.Selected.Headline,
			"Rel":        fx.Selected.Rel,
			"Instant":    fx.Selected.Instant,
			"Census":     census,
			"Deliveries": deliveries,
			"Href":       fx.Selected.Href,
			"JumpLabel":  fx.Selected.JumpLabel,
		}
	}

	allHref, unreadHref := fx.AllHref, fx.UnreadHref
	if selID != "" {
		allHref = "/inbox?id=" + selID
		unreadHref = "/inbox?filter=unread&id=" + selID
	}

	return map[string]any{
		"Title": "Inbox", "NavActive": "inbox", "DesignTokens": true,
		"Messages":   messages,
		"Selected":   selected,
		"Unread":     fx.Unread,
		"Filter":     filter,
		"AllHref":    allHref,
		"UnreadHref": unreadHref,
	}
}

func renderInboxStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadInboxFixture()
	if err != nil {
		return nil, err
	}

	exec := func(data map[string]any) ([]byte, error) {
		t, terr := newStubbedTemplate(head)
		if terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS, "templates/inbox.tmpl"); terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, "inbox", data); terr != nil {
			return nil, terr
		}
		return buf.Bytes(), nil
	}

	type istate struct {
		id     string
		selID  string
		filter string
	}
	states := []istate{
		{"default", "", "all"},
		{"message-open", "m1", "all"},
		{"unread-filter", "", "unread"},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		html, herr := exec(inboxStateData(fx, st.selID, st.filter))
		if herr != nil {
			return nil, herr
		}
		out = append(out, errorGolden{id: st.id, html: html})
	}
	return out, nil
}

// searchHiSeg is one matched-field run in the golden: its literal text and whether it
// is the highlighted (query-matched) run. It carries the field names the "hisegs"
// define reads (.Text/.Hit).
type searchHiSeg struct {
	Text string
	Hit  bool
}

// searchSegG mirrors cmd/web searchSegs byte-for-byte: it splits text on the FIRST
// case-insensitive occurrence of q into the [{Text,Hit}] list "hisegs" renders (#25a),
// omitting empty edges and folding a non-match to a single un-hit seg. Keeping this
// identical to the candidate handler is the point — the golden must exercise the same
// builder over the reconstructed field text.
func searchSegG(text, q string) []searchHiSeg {
	if q == "" {
		return []searchHiSeg{{Text: text}}
	}
	i := strings.Index(strings.ToLower(text), strings.ToLower(q))
	if i < 0 {
		return []searchHiSeg{{Text: text}}
	}
	end := i + len(strings.ToLower(q))
	if i > len(text) || end > len(text) {
		return []searchHiSeg{{Text: text}}
	}
	segs := make([]searchHiSeg, 0, 3)
	if i > 0 {
		segs = append(segs, searchHiSeg{Text: text[:i]})
	}
	segs = append(segs, searchHiSeg{Text: text[i:end], Hit: true})
	if end < len(text) {
		segs = append(segs, searchHiSeg{Text: text[end:]})
	}
	return segs
}

// searchFixtureSeg is one fixtures.json search segment.
type searchFixtureSeg struct {
	Text string `json:"text"`
	Hit  bool   `json:"hit"`
}

// searchFixtureG is the design-system/fixtures/fixtures.json → search slice (the golden
// reads the SAME bytes cmd/web devfixtures.go loadSearchFixture does; a drift fails the
// pixel diff and TestBuildSearchMatchesDesignFixture).
type searchFixtureG struct {
	Query  string `json:"query"`
	Total  int    `json:"total"`
	Assets []struct {
		Href     string             `json:"href"`
		NameSegs []searchFixtureSeg `json:"name_segs"`
		Type     string             `json:"type"`
		Severity string             `json:"severity"`
		SevLabel string             `json:"sev_label"`
	} `json:"assets"`
	Signals []struct {
		Href        string             `json:"href"`
		Severity    string             `json:"severity"`
		SevLabel    string             `json:"sev_label"`
		RuleSegs    []searchFixtureSeg `json:"rule_segs"`
		SubjectSegs []searchFixtureSeg `json:"subject_segs"`
	} `json:"signals"`
	Batches []struct {
		Href      string             `json:"href"`
		Status    string             `json:"status"`
		LabelSegs []searchFixtureSeg `json:"label_segs"`
	} `json:"batches"`
	Docs []struct {
		TitleSegs []searchFixtureSeg `json:"title_segs"`
		SnipSegs  []searchFixtureSeg `json:"snip_segs"`
	} `json:"docs"`
	EmptyVariant struct {
		Query string `json:"query"`
		Total int    `json:"total"`
	} `json:"empty_variant"`
}

func loadSearchFixtureG() (searchFixtureG, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return searchFixtureG{}, err
	}
	var ff struct {
		Search searchFixtureG `json:"search"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return searchFixtureG{}, err
	}
	return ff.Search, nil
}

// joinSearchSegsG reconstructs a field's raw text from its authored segments so it can
// be re-segmented through searchSegG (mirrors cmd/web joinFixtureSegs).
func joinSearchSegsG(segs []searchFixtureSeg) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString(s.Text)
	}
	return b.String()
}

// searchStateData mirrors cmd/web/devfixtures.go searchFixtureData byte-for-byte for a
// given query: the canonical query renders the authored slice folded through searchSegG,
// any other query (the empty variant) renders the zero-result state. Keeping this
// identical to the candidate is the point.
func searchStateData(fx searchFixtureG, q string) map[string]any {
	base := map[string]any{
		"Title": "Search results", "NavActive": "", "DesignTokens": true,
		"Query": q,
	}
	if q != fx.Query {
		base["Total"] = 0
		base["Assets"] = []map[string]any{}
		base["Signals"] = []map[string]any{}
		base["Batches"] = []map[string]any{}
		base["Docs"] = []map[string]any{}
		return base
	}

	assets := make([]map[string]any, 0, len(fx.Assets))
	for _, a := range fx.Assets {
		assets = append(assets, map[string]any{
			"Href":     a.Href,
			"NameSegs": searchSegG(joinSearchSegsG(a.NameSegs), q),
			"Type":     a.Type,
			"Severity": a.Severity,
			"SevLabel": a.SevLabel,
		})
	}
	signals := make([]map[string]any, 0, len(fx.Signals))
	for _, sg := range fx.Signals {
		signals = append(signals, map[string]any{
			"Href":        sg.Href,
			"Severity":    sg.Severity,
			"SevLabel":    sg.SevLabel,
			"RuleSegs":    searchSegG(joinSearchSegsG(sg.RuleSegs), q),
			"SubjectSegs": searchSegG(joinSearchSegsG(sg.SubjectSegs), q),
		})
	}
	batches := make([]map[string]any, 0, len(fx.Batches))
	for _, b := range fx.Batches {
		batches = append(batches, map[string]any{
			"Href":      b.Href,
			"Status":    b.Status,
			"LabelSegs": searchSegG(joinSearchSegsG(b.LabelSegs), q),
		})
	}
	docs := make([]map[string]any, 0, len(fx.Docs))
	for _, d := range fx.Docs {
		docs = append(docs, map[string]any{
			"TitleSegs": searchSegG(joinSearchSegsG(d.TitleSegs), q),
			"SnipSegs":  searchSegG(joinSearchSegsG(d.SnipSegs), q),
		})
	}

	base["Total"] = len(assets) + len(signals) + len(batches) + len(docs)
	base["Assets"] = assets
	base["Signals"] = signals
	base["Batches"] = batches
	base["Docs"] = docs
	return base
}

// renderSearchStates renders the two search goldens — default (/search?q=acme) and
// empty (/search?q=zzz-none). search.tmpl calls the landed "sevbadge" (signals.tmpl),
// so the template set parses BOTH templates/signals.tmpl and templates/search.tmpl into
// the stubbed set for sevbadge to resolve.
func renderSearchStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadSearchFixtureG()
	if err != nil {
		return nil, err
	}

	exec := func(data map[string]any) ([]byte, error) {
		t, terr := newStubbedTemplate(head)
		if terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS, "templates/signals.tmpl"); terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS, "templates/search.tmpl"); terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, "search", data); terr != nil {
			return nil, terr
		}
		return buf.Bytes(), nil
	}

	type sstate struct {
		id string
		q  string
	}
	states := []sstate{
		{"default", fx.Query},
		{"empty", fx.EmptyVariant.Query},
	}

	out := make([]errorGolden, 0, len(states))
	for _, st := range states {
		html, herr := exec(searchStateData(fx, st.q))
		if herr != nil {
			return nil, herr
		}
		out = append(out, errorGolden{id: st.id, html: html})
	}
	return out, nil
}

// --- screen 21: Settings (settings.tmpl, package v3.13.0, WORK-ORDER-21-BATCH7) ---------
//
// renderSettingsStates composes the 19 Settings golden HTMLs from the frozen settings.tmpl,
// one per states.json state. It is the twin of cmd/web/settings_fixtures.go
// (settingsFixtureData): both read the SAME fixtures.json settings slice and stamp the SAME
// "settings" holes with VERBATIM fixture values, so the golden and the fixtures-seeded candidate
// are the same frozen tmpl fed the same data. The 18 chrome-hosted states execute "settings"; the
// 19th (forbidden) is the viewer's requireSettingsAdmin refusal — the error-page settings-forbidden
// kind, exactly as renderErrorStates renders it — so this renders that same error-page here.
//
// The stubbed template also defines an empty "scantrigger" (the repo-authored admin trigger panel
// the scans section calls, which renders nothing when .Trigger is unset — and the fixture omits it)
// and provides the integrationsEnabled func (true), so the frozen tmpl parses and executes.

type gsJob struct {
	ID          int64  `json:"id"`
	Href        string `json:"href"` // nullable — /runs/{run}?job={id} (DF-F3b); empty renders bare #id
	Kind        string `json:"kind"`
	Vantage     string `json:"vantage"`
	State       string `json:"state"`
	Retrying    bool   `json:"retrying"`
	Superseded  bool   `json:"superseded"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	Batch       string `json:"batch"`
}
type gsActive struct {
	ID           int64   `json:"id"`   // DF-F3: the active dispatch's run id (stop/terminate targets)
	Href         string  `json:"href"` // DF-F3: /runs/{id}
	ScanKind     string  `json:"scan_kind"`
	DispatchedAt string  `json:"dispatched_at"`
	Completed    int     `json:"completed"`
	Live         int     `json:"live"`
	Percent      int     `json:"percent"`
	Jobs         []gsJob `json:"jobs"`
}
type gsHistory struct {
	Href         string `json:"href"` // nullable — /runs/{run} (DF-F3b); empty renders bare scan kind
	ScanKind     string `json:"scan_kind"`
	DispatchedAt string `json:"dispatched_at"`
	Live         int    `json:"live"`
	Completed    int    `json:"completed"`
	Dead         int    `json:"dead"`
}
type gsColdScope struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	IsAddress bool   `json:"is_address"`
	OptedIn   bool   `json:"opted_in"`
}
type gsScans struct {
	Active      []gsActive    `json:"active"`
	History     []gsHistory   `json:"history"`
	ColdEnabled bool          `json:"cold_enabled"`
	ColdScopes  []gsColdScope `json:"cold_scopes"`
}
type gsVantage struct {
	Name         string `json:"name"`
	Class        string `json:"class"`
	Resolver     string `json:"resolver"`
	Latency      string `json:"latency"`
	Availability string `json:"availability"`
	Unverified   bool   `json:"unverified"`
}
type gsProber struct {
	Endpoint           string `json:"endpoint"`
	Username           string `json:"username"`
	Availability       string `json:"availability"`
	HostKeyPinned      bool   `json:"host_key_pinned"`
	HostKeyFingerprint string `json:"host_key_fingerprint"`
	Platform           string `json:"platform"`
	KeySet             bool   `json:"key_set"`
	PublicKey          string `json:"public_key"`
	Egress             string `json:"egress"`
}
type gsVantages struct {
	Vantages []gsVantage `json:"vantages"`
	Probers  []gsProber  `json:"probers"`
}
type gsProvider struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Issuer    string `json:"issuer"`
	ClientID  string `json:"client_id"`
	HasSecret bool   `json:"has_secret"`
	Enabled   bool   `json:"enabled"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}
type gsBinding struct {
	ID           string `json:"id"`
	ProviderName string `json:"provider_name"`
	Account      string `json:"account"`
	DisplayName  string `json:"display_name"`
	LinkedAt     string `json:"linked_at"`
}
type gsSSO struct {
	Providers []gsProvider `json:"providers"`
	Bindings  []gsBinding  `json:"bindings"`
}
type gsMember struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Initials    string `json:"initials"`
	Role        string `json:"role"`
	TotpEnabled bool   `json:"totp_enabled"`
	At          string `json:"at"`
	IsSelf      bool   `json:"is_self"`
}
type gsTeam struct {
	Members    []gsMember `json:"members"`
	InviteLink string     `json:"invite_link_fixture"`
}
type gsSession struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Account   string `json:"account"`
	Role      string `json:"role"`
	Device    string `json:"device"`
	IP        string `json:"ip"`
	LastSeen  string `json:"last_seen"`
	Current   bool   `json:"current"`
}
type gsSource struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Kind  string   `json:"kind"`
	What  string   `json:"what"`
	On    bool     `json:"on"`
	Terms []string `json:"terms"`
}
type gsSources struct {
	Unencumbered     []gsSource `json:"unencumbered"`
	OperatorAccepted []gsSource `json:"operator_accepted"`
	Barred           []gsSource `json:"barred"`
}
type gsCounts struct {
	Sensitive int `json:"sensitive"`
	Frequency int `json:"frequency"`
	Union     int `json:"union"`
	TCP       int `json:"tcp"`
	UDP       int `json:"udp"`
}
type gsSensitive struct {
	Port      int    `json:"port"`
	Transport string `json:"transport"`
	Service   string `json:"service"`
}
type gsFrequency struct {
	Port          int    `json:"port"`
	AlsoSensitive bool   `json:"also_sensitive"`
	Edited        bool   `json:"edited"`
	EditAction    string `json:"edit_action"`
}
type gsAperture struct {
	UDPCount  int           `json:"udp_count"`
	Counts    gsCounts      `json:"counts"`
	Sensitive []gsSensitive `json:"sensitive"`
	Frequency []gsFrequency `json:"frequency"`
}
type gsInstanceVantage struct {
	Name    string `json:"name"`
	Latency string `json:"latency"`
	Avail   string `json:"avail"`
}
type gsUpdate struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}
type gsInstance struct {
	Update     *gsUpdate           `json:"update"`
	Version    string              `json:"version"`
	License    string              `json:"license"`
	Uptime     string              `json:"uptime"`
	QueueDepth int                 `json:"queue_depth"`
	DiskPct    int                 `json:"disk_pct"`
	DiskDetail string              `json:"disk_detail"`
	PgLabel    string              `json:"pg_label"`
	PgDetail   string              `json:"pg_detail"`
	Vantages   []gsInstanceVantage `json:"vantages"`
}
type gsClassOption struct {
	Name    string `json:"name"`
	Checked bool   `json:"checked"`
}
type gsChannel struct {
	ID          string          `json:"id"`
	URL         string          `json:"url"`
	Classes     []string        `json:"classes"`
	ClassStates []gsClassOption `json:"class_states"`
	HasSecret   bool            `json:"has_secret"`
	Enabled     bool            `json:"enabled"`
	By          string          `json:"by"`
	At          string          `json:"at"`
}
type gsChannels struct {
	ClassOptions []gsClassOption `json:"class_options"`
	Channels     []gsChannel     `json:"channels"`
}
type gsTile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Mark        string `json:"mark"`
	Category    string `json:"category"`
	State       string `json:"state"`
	Description string `json:"description"`
}
type gsGrant struct {
	Scope  string `json:"scope"`
	Detail string `json:"detail"`
	Write  bool   `json:"write"`
}
type gsDrawer struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Mark         string    `json:"mark"`
	Category     string    `json:"category"`
	State        string    `json:"state"`
	Description  string    `json:"description"`
	Attention    string    `json:"attention"`
	Grants       []gsGrant `json:"grants"`
	Installed    string    `json:"installed"`
	LastDelivery string    `json:"last_delivery"`
	Classes      string    `json:"classes"`
}
type gsIntegrations struct {
	Cats   []string `json:"cats"`
	Cat    string   `json:"cat"`
	Q      string   `json:"q"`
	Tiles  []gsTile `json:"tiles"`
	Drawer gsDrawer `json:"drawer_fixture"`
}
type gsCensus struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
	Href string `json:"href"`
}
type gsMsgDelivery struct {
	State       string `json:"state"`
	ChannelHost string `json:"channel_host"`
	Failed      bool   `json:"failed"`
	LastError   string `json:"last_error"`
}
type gsMessage struct {
	ID         string          `json:"id"`
	Read       bool            `json:"read"`
	Cause      string          `json:"cause"`
	Class      string          `json:"class"`
	Instant    string          `json:"instant"`
	Headline   string          `json:"headline"`
	Href       string          `json:"href"`
	LinkText   string          `json:"link_text"`
	Census     []gsCensus      `json:"census"`
	Deliveries []gsMsgDelivery `json:"deliveries"`
}
type gsOutcome struct {
	ChannelHost string `json:"channel_host"`
	Class       string `json:"class"`
	Failed      bool   `json:"failed"`
	State       string `json:"state"`
	When        string `json:"when"`
}
type gsRetention struct {
	ObservationCurrencyDays int    `json:"observation_currency_days"`
	DispatchCadenceMultiple int    `json:"dispatch_cadence_multiple"`
	UpdatedAt               string `json:"updated_at"`
	UpdatedBy               string `json:"updated_by"`
}
type gsDeliverySection struct {
	Deliveries []gsOutcome `json:"deliveries"`
	Retention  gsRetention `json:"retention"`
}
type gsSettings struct {
	Scans        gsScans           `json:"scans"`
	Vantages     gsVantages        `json:"vantages"`
	SSO          gsSSO             `json:"sso"`
	Team         gsTeam            `json:"team"`
	Sessions     []gsSession       `json:"sessions"`
	Sources      gsSources         `json:"sources"`
	Aperture     gsAperture        `json:"aperture"`
	Instance     gsInstance        `json:"instance"`
	Channels     gsChannels        `json:"channels"`
	Integrations gsIntegrations    `json:"integrations"`
	Messages     []gsMessage       `json:"messages"`
	Delivery     gsDeliverySection `json:"delivery"`
}

func loadSettingsFixtureG() (gsSettings, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return gsSettings{}, err
	}
	var ff struct {
		Settings gsSettings `json:"settings"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return gsSettings{}, err
	}
	return ff.Settings, nil
}

// findActiveDispatchG returns the fixture active dispatch whose id matches the raw ?stop=/
// ?terminate= query value (id 1409, #35), or nil. Mirrors cmd/web/settings_fixtures.go findActiveDispatch.
func findActiveDispatchG(active []gsActive, raw string) *gsActive {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	for i := range active {
		if active[i].ID == id {
			return &active[i]
		}
	}
	return nil
}

// jobStateCountsG folds a dispatch's jobs into the confirm dialog's live counts: pending is the
// ready (not-yet-claimed) jobs a stop cancels, running is the running jobs a stop lets finish and a
// terminate kills. Mirrors cmd/web/settings_fixtures.go jobStateCounts.
func jobStateCountsG(jobs []gsJob) (pending, running int) {
	for _, j := range jobs {
		switch j.State {
		case "ready":
			pending++
		case "running":
			running++
		}
	}
	return pending, running
}

// settingsGoldenMap mirrors cmd/web/settings_fixtures.go settingsFixtureData one-for-one for the
// given tab and query, stamping the "settings" holes from the fixture with verbatim values.
func settingsGoldenMap(fx gsSettings, tab string, q map[string]string) map[string]any {
	data := map[string]any{
		"Title": "Settings", "NavActive": "settings", "Tab": tab,
		"IsAdmin": true, "DesignTokens": true,
	}
	switch tab {
	case "scans":
		data["Active"] = fx.Scans.Active
		data["History"] = fx.Scans.History
		data["ColdEnabled"] = fx.Scans.ColdEnabled
		data["ColdScopes"] = fx.Scans.ColdScopes
		data["ColdError"] = ""
		// DF-F4 stop / terminate confirm dialogs (states scans-stop-confirm /
		// scans-terminate-confirm at id 1409, #35). Mirrors cmd/web/settings_fixtures.go:
		// the target is the matching active dispatch, its Pending/Running folded live
		// from that dispatch's job states (jobStateCounts: ready→pending, running→running).
		if id := q["stop"]; id != "" {
			if a := findActiveDispatchG(fx.Scans.Active, id); a != nil {
				pending, running := jobStateCountsG(a.Jobs)
				data["StopTarget"] = map[string]any{
					"ID": a.ID, "ScanKind": a.ScanKind, "Pending": pending, "Running": running,
				}
			}
		}
		if id := q["terminate"]; id != "" {
			if a := findActiveDispatchG(fx.Scans.Active, id); a != nil {
				_, running := jobStateCountsG(a.Jobs)
				data["TerminateTarget"] = map[string]any{
					"ID": a.ID, "ScanKind": a.ScanKind, "Running": running,
				}
			}
		}
	case "vantages":
		data["Vantages"] = fx.Vantages.Vantages
		data["Probers"] = fx.Vantages.Probers
		data["ProberError"], data["ProberHost"], data["ProberPort"], data["ProberUser"] = "", "", "", ""
	case "sso":
		data["SSOProviders"] = fx.SSO.Providers
		data["SSOBindings"] = fx.SSO.Bindings
		data["SSOError"], data["SSOName"], data["SSOSlug"], data["SSOIssuer"], data["SSOClientID"] = "", "", "", "", ""
	case "team":
		data["Members"] = fx.Team.Members
		data["TeamError"], data["RoleError"], data["RemoveError"] = "", "", ""
		data["InviteLink"], data["InviteRole"] = "", ""
		data["InviteOpen"] = q["invite"] != ""
		if id := q["remove"]; id != "" {
			for i := range fx.Team.Members {
				if fx.Team.Members[i].ID == id {
					data["RemoveTarget"] = fx.Team.Members[i]
				}
			}
		}
	case "sessions":
		data["Sessions"] = fx.Sessions
		data["RevokeAccountError"] = ""
		if id := q["revoke-account"]; id != "" {
			for i := range fx.Sessions {
				if fx.Sessions[i].AccountID == id {
					data["RevokeAccountTarget"] = map[string]any{"AccountID": id, "Username": fx.Sessions[i].Account}
					break
				}
			}
		}
	case "audit":
		data["AuditRows"] = nil
	case "sources":
		data["Unencumbered"] = fx.Sources.Unencumbered
		data["OperatorAccepted"] = fx.Sources.OperatorAccepted
		data["Barred"] = fx.Sources.Barred
		data["SourceError"] = ""
		if id := q["consent"]; id != "" {
			for i := range fx.Sources.OperatorAccepted {
				src := fx.Sources.OperatorAccepted[i]
				if src.ID == id {
					data["Consent"] = map[string]any{"ID": src.ID, "Name": src.Name, "Terms": src.Terms}
					break
				}
			}
		}
	case "aperture":
		data["UDPCount"] = fx.Aperture.UDPCount
		data["Counts"] = fx.Aperture.Counts
		data["Sensitive"] = fx.Aperture.Sensitive
		data["Frequency"] = fx.Aperture.Frequency
		data["VCError"], data["VCPort"] = "", ""
	case "instance":
		data["Instance"] = fx.Instance
	case "channels":
		data["Channels"] = fx.Channels.Channels
		data["ClassOptions"] = fx.Channels.ClassOptions
		data["ChanError"], data["ChanURL"] = "", ""
	case "integrations":
		data["IntCats"] = fx.Integrations.Cats
		data["IntCat"] = fx.Integrations.Cat
		data["IntQ"] = fx.Integrations.Q
		data["Integrations"] = fx.Integrations.Tiles
		if id := q["view"]; id != "" && id == fx.Integrations.Drawer.ID {
			data["IntDrawer"] = fx.Integrations.Drawer
		}
	case "messages":
		data["Messages"] = fx.Messages
	case "delivery":
		data["Deliveries"] = fx.Delivery.Deliveries
		data["Retention"] = fx.Delivery.Retention
		data["RetError"], data["RetObs"], data["RetDispatch"] = "", "", ""
	}
	return data
}

// newSettingsTemplate carries the REAL design-owned shell (via newStubbedTemplate) plus
// settings.tmpl, so the Settings goldens render full-page inside the chrome (#27f). The
// repo-authored "scantrigger" define (cmd/web/scantrigger.go) is not a design-owned
// artifact, so it is stubbed empty here exactly as before — the candidate's settings
// scans tab renders no trigger panel either (its .Trigger is unset in the fixture path),
// so golden and candidate agree.
func newSettingsTemplate(head template.HTML) (*template.Template, error) {
	t, err := newStubbedTemplate(head)
	if err != nil {
		return nil, err
	}
	if _, err := t.Parse(`{{define "scantrigger"}}{{end}}`); err != nil {
		return nil, err
	}
	if _, err := t.ParseFS(designfs.FS, "templates/settings.tmpl"); err != nil {
		return nil, err
	}
	return t, nil
}

func renderSettingsStates(bodyFlex bool) ([]errorGolden, error) {
	head, err := goldenHead(bodyFlex)
	if err != nil {
		return nil, err
	}
	fx, err := loadSettingsFixtureG()
	if err != nil {
		return nil, err
	}

	execSettings := func(tab string, q map[string]string) ([]byte, error) {
		t, terr := newSettingsTemplate(head)
		if terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, "settings", settingsGoldenMap(fx, tab, q)); terr != nil {
			return nil, terr
		}
		return buf.Bytes(), nil
	}

	type sstate struct {
		id  string
		tab string
		q   map[string]string
	}
	states := []sstate{
		{"scans", "scans", nil},
		{"scans-stop-confirm", "scans", map[string]string{"stop": "1409"}},
		{"scans-terminate-confirm", "scans", map[string]string{"terminate": "1409"}},
		{"vantages", "vantages", nil},
		{"sso", "sso", nil},
		{"team", "team", nil},
		{"team-invite", "team", map[string]string{"invite": "1"}},
		{"team-remove", "team", map[string]string{"remove": "u3"}},
		{"sessions", "sessions", nil},
		{"sessions-revoke-all", "sessions", map[string]string{"revoke-account": "u2"}},
		{"audit", "audit", nil},
		{"sources", "sources", nil},
		{"sources-consent", "sources", map[string]string{"consent": "ripestat"}},
		{"aperture", "aperture", nil},
		{"instance", "instance", nil},
		{"channels", "channels", nil},
		{"integrations", "integrations", nil},
		{"integrations-drawer", "integrations", map[string]string{"view": "pagerduty"}},
		{"messages", "messages", nil},
		{"delivery", "delivery", nil},
	}

	out := make([]errorGolden, 0, len(states)+1)
	for _, st := range states {
		html, herr := execSettings(st.tab, st.q)
		if herr != nil {
			return nil, herr
		}
		out = append(out, errorGolden{id: st.id, html: html})
	}

	// forbidden: the viewer's requireSettingsAdmin refusal renders the error-page
	// settings-forbidden kind (the frozen landed behavior), so its golden is that
	// same error-page — rendered here exactly as renderErrorStates does.
	{
		t, terr := newStubbedTemplate(head)
		if terr != nil {
			return nil, terr
		}
		if _, terr := t.ParseFS(designfs.FS, "templates/error.tmpl"); terr != nil {
			return nil, terr
		}
		var buf bytes.Buffer
		if terr := execGolden(t, &buf, "error-page", map[string]any{
			"Kind": "settings-forbidden", "Code": "403",
			"ActionLabel": "Back to dashboard", "ActionHref": "/",
		}); terr != nil {
			return nil, terr
		}
		out = append(out, errorGolden{id: "forbidden", html: buf.Bytes()})
	}
	return out, nil
}
