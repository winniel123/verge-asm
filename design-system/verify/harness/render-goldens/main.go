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
	screen := flag.String("screen", "inventory", "which screen to render: inventory | error | profile | signin")
	out := flag.String("out", "", "inventory: path to write the single golden HTML")
	outdir := flag.String("outdir", "", "error|profile: directory to write one golden HTML per state (<state>.html)")
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
	default:
		log.Fatalf("render-goldens: unknown -screen %q (want inventory | error | profile | signin)", *screen)
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
