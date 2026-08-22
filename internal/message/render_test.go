package message

import (
	"strings"
	"testing"
)

// A populated artifact renders the delivered document: the identity row, the KPI
// band, the two change sections with drift-palette chips, and the delivery
// receipt. The fixture is a test-local sample (as the reference JSX carries inline
// sample data); it is never shipped by a handler.
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
		Appeared: []ArtifactChange{
			{Change: "appeared", Subject: "edge-gw-03.acmecorp.io", Detail: "entered the estate aug 22"},
			{Change: "revealed", Subject: "api.acmecorp.io", Detail: "came into view aug 22"},
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
		"New this week", "edge-gw-03.acmecorp.io", // appeared section
		"Withdrawn by the world", "staging-4.acmecorp.io:8080", // withdrawn section
		"delivered 2026-08-22T09:00Z · ops.acmecorp.io", // receipt, host only
		"vg-artifact", // self-contained token scope for PDF/email
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing %q", want)
		}
	}

	// Change rides the drift palette, never the severity ramp.
	if !strings.Contains(out, "var(--drift-gain-fg)") || !strings.Contains(out, "var(--drift-loss-fg)") {
		t.Error("change rows must use the drift palette")
	}
	if strings.Contains(out, "--sev-") || strings.Contains(out, "SeverityBadge") {
		t.Error("the artifact must carry no severity ramp — a signal is not scored")
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
	text := stripTags(stripStyle(htmlOut))
	if ContainsValence(text) {
		t.Errorf("rendered artifact copy carries a valence word; text: %q", text)
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
