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

const devShellToastVariant = "flash-toast"

type chromeVM struct {
	Nav           []chromeNav
	Org           string
	Version       string
	UserName      string
	UserInitials  string
	ScanRunning   bool
	Unread        bool
	Messages      []bellMessage
	PaletteGroups []paletteGroup
	Toasts        []toastVM
}

type chromeNav struct {
	ID     string
	Label  string
	Href   string
	Active bool
	Count  string
}

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

type toastVM struct {
	Tone        string
	Title       string
	Description string
}

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

func navSlice(active string, signalCount int) []chromeNav {
	out := make([]chromeNav, 0, len(consoleNav))
	for _, n := range consoleNav {
		item := n
		item.Active = n.ID == active
		if n.ID == "signals" && signalCount > 0 {
			// An absent count must render as the empty string the tmpl drops, so it is pre-formatted here.
			item.Count = strconv.Itoa(signalCount)
		}
		out = append(out, item)
	}
	return out
}

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

func countHint(n int64, word string) string {
	if n <= 0 {
		return ""
	}
	return strconv.FormatInt(n, 10) + " " + word
}

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
