package main

import (
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// The console shell/chrome is design-owned (design-system/templates/shell.tmpl,
// package v3.14.0, map #22). Its "chrome"/"foot" definitions read a single nullable
// .Chrome hole — auth surfaces pass none and get no chrome, exactly as today. This
// file assembles that .Chrome view-model. injectChrome (auth.go) wires it onto every
// chrome page's data. Two builders share one shape:
//
//   - production (chromeFromReads): the honest live reads a real deployment renders —
//     the nav slice the old hardcoded links encoded (Signals carries the open count),
//     the single-org static chip, the bell's recent messages, the palette groups, the
//     avatar identity, and the toast stack decoded from the PRG flash query (shell.go
//     toastRedirect).
//   - devMode (chromeFromFixture): the pinned fixtures.json shell slice, so the seeded
//     candidate renders the SAME chrome render-goldens composes for the golden — the
//     v4 pixel-parity contract. The active pill is set per the page's NavActive.
//
// Org switcher: single-org stands (ADR-0073 — the deployment IS the org; there is no org
// table, no org-scoped tenancy, no POST /org/switch). SPEC-CHANGE #33 RETIRED the switcher
// permanently (package v3.17.0): shell.tmpl renders only the static org chip, .Chrome.Orgs
// is gone from the contract, and the org-open golden is dropped. Nothing here models orgs.

// devShellToastVariant is the states.json shell "toasts" capture variant: a VERGE_DEV
// ?variant=flash-toast query folds the fixture's toast stack into .Chrome.Toasts so the
// candidate renders the same ToastStack the golden does (the real flash rides the
// `toast` PRG query, decoded by decodeToasts).
const devShellToastVariant = "flash-toast"

// chromeVM is the nullable .Chrome hole shell.tmpl's "chrome"/"foot" read.
type chromeVM struct {
	Nav           []chromeNav
	Org           string // the static single-org chip label (ADR-0073; switcher retired, #33)
	Version       string
	UserName      string
	UserInitials  string
	ScanRunning   bool
	Unread        bool
	Messages      []bellMessage
	PaletteGroups []paletteGroup
	Toasts        []toastVM
}

// chromeNav is one TopNav pill: id (active match), label, href, and an optional count
// pill (Signals carries the open-signal count). Count is a pre-formatted string so an
// absent count is the empty string the tmpl's {{if .Count}} drops.
type chromeNav struct {
	ID     string
	Label  string
	Href   string
	Active bool
	Count  string
}

// paletteGroup / paletteItem are the server-rendered command-palette groups
// (#27c): items are links except the one theme-toggle action; the search item stays
// visible through any filter and its href tracks the typed query to /search?q=.
type paletteGroup struct {
	Label string
	Items []paletteItem
}

type paletteItem struct {
	Label       string
	Icon        string
	Hint        string
	Href        string
	Search      bool
	ThemeToggle bool
}

// toastVM is one ToastStack entry (#27d): a tone dot, a title, and an optional
// description. It rides the PRG flash the shell's toastRedirect writes.
type toastVM struct {
	Tone        string
	Title       string
	Description string
}

// consoleNav is the primary-nav spine the old hardcoded chrome links encoded, in
// order (Signals carries the open count, threaded at build time). It is the one place
// the console's nav order lives now that the shell is design-owned.
var consoleNav = []chromeNav{
	{ID: "dashboard", Label: "Dashboard", Href: "/"},
	{ID: "scope", Label: "Scope", Href: "/scope"},
	{ID: "inventory", Label: "Inventory", Href: "/inventory"},
	{ID: "drift", Label: "Drift", Href: "/drift"},
	{ID: "signals", Label: "Signals", Href: "/signals"},
	{ID: "exposure", Label: "Exposure", Href: "/exposure"},
	{ID: "coverage", Label: "Coverage", Href: "/coverage"},
	{ID: "graph", Label: "Graph", Href: "/graph"},
	{ID: "reports", Label: "Reports", Href: "/reports"},
}

// navSlice builds the TopNav pills from consoleNav, marking the active pill and
// stamping the Signals open count.
func navSlice(active string, signalCount int) []chromeNav {
	out := make([]chromeNav, 0, len(consoleNav))
	for _, n := range consoleNav {
		item := n
		item.Active = n.ID == active
		if n.ID == "signals" && signalCount > 0 {
			item.Count = strconv.Itoa(signalCount)
		}
		out = append(out, item)
	}
	return out
}

// paletteGroupsProd builds the command-palette groups for a real deployment: the
// Screens group (the console screens + Inbox/Profile/Settings surfaces, the gated
// Integrations item, and the always-visible search-handoff item) and the Actions
// group (Run scan → /scans, Add seed → /scope#seed-form, and the theme toggle). The
// Signals/Inbox hints track the live open/unread counts. The Integrations item is
// emitted only when integrationsEnabled, per #27c.
func paletteGroupsProd(signalCount int, unread int64) []paletteGroup {
	screens := []paletteItem{
		{Label: "Dashboard", Icon: "layout-dashboard", Href: "/"},
		{Label: "Scope", Icon: "globe", Href: "/scope"},
		{Label: "Inventory", Icon: "server", Href: "/inventory"},
		{Label: "Drift", Icon: "git-branch", Href: "/drift"},
		{Label: "Signals", Icon: "shield-alert", Hint: countHint(int64(signalCount), "open"), Href: "/signals"},
		{Label: "Exposure", Icon: "eye", Href: "/exposure"},
		{Label: "Coverage", Icon: "gauge", Href: "/coverage"},
		{Label: "Graph", Icon: "network", Href: "/graph"},
		{Label: "Reports", Icon: "file-text", Href: "/reports"},
		{Label: "Inbox", Icon: "inbox", Hint: countHint(unread, "unread"), Href: "/inbox"},
		{Label: "Profile", Icon: "user", Href: "/profile"},
		{Label: "Sources", Icon: "database", Href: "/settings?tab=sources"},
		{Label: "Sessions", Icon: "monitor-smartphone", Href: "/settings?tab=sessions"},
		{Label: "Port aperture", Icon: "layout-grid", Href: "/settings?tab=aperture"},
	}
	if integrationsEnabled {
		screens = append(screens, paletteItem{Label: "Integrations", Icon: "puzzle", Href: "/settings?tab=integrations"})
	}
	screens = append(screens,
		paletteItem{Label: "Settings", Icon: "settings", Href: "/settings"},
		paletteItem{Label: "Search everything", Icon: "search", Href: "/search", Search: true},
	)
	return []paletteGroup{
		{Label: "Screens", Items: screens},
		{Label: "Actions", Items: []paletteItem{
			{Label: "Run scan", Icon: "play", Href: "/scans"},
			{Label: "Add seed", Icon: "plus", Href: "/scope#seed-form"},
			{Label: "Toggle theme", Icon: "moon", ThemeToggle: true},
		}},
	}
}

// countHint renders a palette item hint like "47 open" / "2 unread", or "" when the
// count is zero (the tmpl drops an empty hint).
func countHint(n int64, word string) string {
	if n <= 0 {
		return ""
	}
	return strconv.FormatInt(n, 10) + " " + word
}

// decodeToasts reads the PRG flash the shell's toastRedirect (shell.go) writes into
// the destination GET's `toast` query — a base64url JSON object {tone,title,
// description}. The spec ToastStack renders it server-side; the design-owned shell JS
// then auto-dismisses it. A missing or malformed query yields no toast (the page just
// renders without one). Text is rendered by html/template, which escapes it.
func decodeToasts(r *http.Request) []toastVM {
	if r == nil {
		return nil
	}
	raw := r.URL.Query().Get("toast")
	if raw == "" {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil
	}
	var t toastVM
	if err := json.Unmarshal(b, &t); err != nil || t.Title == "" {
		return nil
	}
	return []toastVM{t}
}

// --- devMode: the pinned fixtures.json shell slice ---------------------------------

// shellFixture is the on-disk fixtures.json → shell slice. render-goldens mirrors this
// shape one-for-one so golden and candidate compose the identical chrome.
type shellFixture struct {
	Title  string `json:"title"`
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

// loadShellFixture reads the pinned fixtures.json shell slice from the embedded design
// package (designfs). A read/parse failure degrades to the zero fixture.
func loadShellFixture() shellFixture {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		log.Printf("web: shell fixture read: %v", err)
		return shellFixture{}
	}
	var ff struct {
		Shell shellFixture `json:"shell"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		log.Printf("web: shell fixture parse: %v", err)
		return shellFixture{}
	}
	return ff.Shell
}

// chromeFromFixture shapes the pinned shell slice into the .Chrome view-model for a
// VERGE_DEV render. The active pill is set per the page's NavActive (each screen's
// candidate highlights its own pill); scanning lights .ScanRunning; showToast folds
// in the toasts variant. The org chip is static (the switcher retired, #33). The gated
// Integrations palette item is emitted only when integrationsEnabled. render-goldens
// composes the identical struct from the same bytes, so golden and candidate agree
// byte-for-byte.
func chromeFromFixture(navActive string, scanning, showToast bool) *chromeVM {
	fx := loadShellFixture()
	c := &chromeVM{
		Org:          fx.Chrome.Org,
		Version:      fx.Chrome.Version,
		UserName:     fx.Chrome.UserName,
		UserInitials: fx.Chrome.UserInitials,
		ScanRunning:  scanning,
		Unread:       fx.Chrome.Unread,
	}
	for _, n := range fx.Chrome.Nav {
		c.Nav = append(c.Nav, chromeNav{
			ID: n.ID, Label: n.Label, Href: n.Href,
			Active: n.ID == navActive, Count: n.Count,
		})
	}
	for _, m := range fx.Chrome.Messages {
		c.Messages = append(c.Messages, bellMessage{
			Class: m.Class, Rel: m.Rel, Headline: m.Headline, Unread: m.Unread, Href: m.Href,
		})
	}
	for _, g := range fx.Chrome.PaletteGroups {
		pg := paletteGroup{Label: g.Label}
		for _, it := range g.Items {
			if it.Gated == "integrationsEnabled" && !integrationsEnabled {
				continue
			}
			pg.Items = append(pg.Items, paletteItem{
				Label: it.Label, Icon: it.Icon, Hint: it.Hint, Href: it.Href,
				Search: it.Search, ThemeToggle: it.ThemeToggle,
			})
		}
		c.PaletteGroups = append(c.PaletteGroups, pg)
	}
	if showToast {
		for _, t := range fx.Chrome.ToastsVariant {
			c.Toasts = append(c.Toasts, toastVM{Tone: t.Tone, Title: t.Title, Description: t.Description})
		}
	}
	return c
}
