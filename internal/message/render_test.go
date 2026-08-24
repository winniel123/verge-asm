package message

import (
	"strings"
	"testing"
)

// A populated artifact renders the delivered document: the identity row, the KPI
// band, the by-severity bars, the signals table with its SeverityBadge column, the
// withdrawn drift section, and the delivery receipt. The fixture is a test-local
// sample (as the reference JSX carries inline sample data); it is never shipped.
func TestRenderArtifactPopulated(t *testing.T) {
	a := Artifact{
		Title:       "Weekly exposure summary",
		Org:         "acmecorp",
		PeriodStart: "2026-08-15",
		PeriodEnd:   "2026-08-22",
		DeliveryNo:  42,
		GeneratedAt: "2026-08-22T09:00:00Z",
		Version:     "verge v0.9.2",
		Format:      "pdf",
		Stats: []ArtifactStat{
			{Label: "Open signals", Value: "47", Delta: "+3", DeltaTone: "bad", Caption: "vs previous week"},
			{Label: "New assets", Value: "12", Delta: "+8", DeltaTone: "neutral", Caption: "8 names · 4 addresses"},
			{Label: "Scans run", Value: "128", Caption: "this period"},
		},
		SeverityCounts: []ArtifactSeverityCount{
			{Level: "critical", Count: 3},
			{Level: "high", Count: 11},
			{Level: "medium", Count: 18},
			{Level: "low", Count: 9},
			{Level: "info", Count: 6},
		},
		Signals: []ArtifactSignal{
			{Severity: "critical", Signal: "Service answering without transport encryption", Asset: "edge-gw-03.acmecorp.io", Raised: "aug 22"},
			{Severity: "high", Signal: "Endpoint reachable from the internet", Asset: "api.acmecorp.io", Raised: "aug 22"},
			{Severity: "medium", Signal: "Certificate expires in 23 days", Asset: "idp-signing-2026", Raised: "aug 20"},
		},
		Withdrawn: []ArtifactChange{
			{Change: "withdrawn", Subject: "staging-4.acmecorp.io:8080", Detail: "closed since batch 14:00Z"},
		},
		Delivered:   "2026-08-22T09:00Z",
		ChannelHost: "ops.acmecorp.io",
	}
	out := string(RenderArtifact(a))

	for _, want := range []string{
		"acmecorp",                       // identity row
		"generated 2026-08-22T09:00:00Z", // provenance
		"verge v0.9.2",
		"Open signals", "47", "vs previous week", // KPI band
		"Scans run", "128",
		"Open signals by severity",                // severity bar breakdown header
		"New this week", "edge-gw-03.acmecorp.io", // signals table
		"Endpoint reachable from the internet",                 // a signal headline
		"Withdrawn by the world", "staging-4.acmecorp.io:8080", // withdrawn section
		"delivered 2026-08-22T09:00Z · ops.acmecorp.io", // receipt, host only
		"vg-artifact", // self-contained token scope for PDF/email
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing %q", want)
		}
	}

	// The severity ramp is present (P2.10): the by-severity bars take the
	// severity-dot tokens, and the signals table carries SeverityBadges keyed on the
	// five exact levels. Critical is the only solid fill; the dot levels tint.
	for _, want := range []string{
		"var(--sev-critical-dot)",  // a bar fill
		"var(--sev-critical-fill)", // the critical badge, the only solid fill
		"var(--sev-high-bg)",       // a tinted badge
		`data-sev="critical"`,      // the ramp is marked for the valence exemption
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing severity ramp %q", want)
		}
	}

	// Change still rides the drift palette, never the severity ramp.
	if !strings.Contains(out, "var(--drift-loss-fg)") {
		t.Error("change rows must use the drift palette")
	}
	// The delivered document must be self-contained (its own token styles) and
	// theme-aware in both directions, so it renders as a standalone PDF/email body.
	for _, want := range []string{
		"prefers-color-scheme:dark",
		`:root[data-theme="dark"] .vg-artifact`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("self-contained document missing %q", want)
		}
	}
	// No valence word grades the delivered copy.
	assertNoValence(t, out)
}

// An artifact with no delivered content renders the design-system empty-state
// rather than a fabricated document.
func TestRenderArtifactEmpty(t *testing.T) {
	a := Artifact{}
	if !a.Empty() {
		t.Fatal("a zero artifact must report Empty")
	}
	out := string(RenderArtifact(a))

	for _, want := range []string{
		"No report has been delivered yet", // empty-state headline
		"not delivered",                    // receipt states the fact
		"vg-artifact",                      // still a self-contained document frame
	} {
		if !strings.Contains(out, want) {
			t.Errorf("empty artifact missing %q", want)
		}
	}
	// Nothing is fabricated: no KPI numerals, no change chips.
	if strings.Contains(out, "var(--drift-gain-fg)") || strings.Contains(out, "var(--drift-loss-fg)") {
		t.Error("empty artifact must not draw change chips")
	}
	assertNoValence(t, out)
}

func TestArtifactPeriod(t *testing.T) {
	got := ArtifactPeriod(Artifact{PeriodStart: "2026-08-15", PeriodEnd: "2026-08-22", DeliveryNo: 42})
	if want := "2026-08-15 → 2026-08-22 · delivery #42"; got != want {
		t.Errorf("ArtifactPeriod = %q, want %q", got, want)
	}
	if got := ArtifactPeriod(Artifact{}); got != "" {
		t.Errorf("empty ArtifactPeriod = %q, want empty", got)
	}
}

// assertNoValence fails if any refused valence word appears in the rendered copy,
// after stripping the markup, inline styles and the token stylesheet the guard is
// not meant to read (a style token like --danger-* is not prose). It checks the
// human-visible text.
func assertNoValence(t *testing.T, htmlOut string) {
	t.Helper()
	text := stripTags(stripSev(stripStyle(htmlOut)))
	if ContainsValence(text) {
		t.Errorf("rendered artifact copy carries a valence word; text: %q", text)
	}
}

// stripSev removes every severity ramp element whole — the by-severity section
// title, the "Severity" column header, the badge labels, and the bar labels. That
// language is the severity scale, the one loud voice the system exempts from the
// valence rule, exactly as stripStyle exempts the token stylesheet. Ramp elements
// are marked data-sev on their opening tag; the removal reads the element's tag
// name from the opener and matches nested tags of the same name to its close (a
// badge wraps one nested dot <span>).
func stripSev(s string) string {
	isAlpha := func(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') }
	for {
		m := strings.Index(s, "data-sev")
		if m < 0 {
			return s
		}
		open := strings.LastIndex(s[:m], "<")
		if open < 0 {
			return s
		}
		j := open + 1
		for j < len(s) && isAlpha(s[j]) {
			j++
		}
		openTok, closeTok := "<"+s[open+1:j], "</"+s[open+1:j]+">"
		depth, i, end := 0, open, -1
		for i < len(s) && end < 0 {
			switch {
			case strings.HasPrefix(s[i:], openTok):
				depth++
				i += len(openTok)
			case strings.HasPrefix(s[i:], closeTok):
				depth--
				i += len(closeTok)
				if depth == 0 {
					end = i
				}
			default:
				i++
			}
		}
		if end < 0 {
			return s[:open]
		}
		s = s[:open] + s[end:]
	}
}

// stripStyle drops every <style>…</style> block whole, so the token stylesheet's
// custom-property names (--danger, --ok) never reach the prose guard.
func stripStyle(s string) string {
	for {
		i := strings.Index(s, "<style")
		if i < 0 {
			return s
		}
		j := strings.Index(s[i:], "</style>")
		if j < 0 {
			return s[:i]
		}
		s = s[:i] + s[i+j+len("</style>"):]
	}
}

// stripTags removes HTML tags so the valence guard reads only the visible text,
// not attribute values or the token stylesheet.
func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
