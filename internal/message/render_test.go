package message

import (
	"strings"
	"testing"
)

// A populated artifact renders the delivered document by executing the design-owned
// "artifactdoc" define (SPEC-CHANGE #23g): the identity row, the KPI band with its delta
// chips, the by-severity bars, the "new this week" table with its SeverityBadge column, the
// withdrawn drift section, and the delivery receipt. The fixture is a test-local sample (as
// the reference JSX carries inline sample data); it is never shipped. The standalone form
// prepends the design token vocabulary so its inline var(--…) styles resolve with no console
// stylesheet in scope.
func TestRenderArtifactPopulated(t *testing.T) {
	a := Artifact{
		Title:       "Weekly exposure summary",
		Org:         "acmecorp",
		PeriodStart: "2026-08-15",
		PeriodEnd:   "2026-08-22",
		DeliveryNo:  42,
		GeneratedAt: "2026-08-22T09:00:00Z",
		Version:     "v0.9.2",
		Format:      "pdf",
		Stats: []ArtifactStat{
			{Label: "Open signals", Value: "47", Delta: "+3", DeltaTone: "bad", Caption: "vs previous week"},
			{Label: "New assets", Value: "12", Delta: "+8", DeltaTone: "neutral", Caption: "8 names · 4 services"},
			{Label: "Mean time to withdrawal", Value: "2.4d", Delta: "−0.6d", DeltaTone: "good", Caption: "critical + high only"},
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
		"acmecorp",                               // identity row
		"generated 2026-08-22T09:00:00Z",         // provenance
		"verge v0.9.2",                           // the tmpl prepends "verge " to .Version
		"Open signals", "47", "vs previous week", // KPI band
		"Mean time to withdrawal", "2.4d", // the MTTW KPI (#23a)
		"Open signals by severity",                // severity bar breakdown header
		"New this week", "edge-gw-03.acmecorp.io", // signals table
		"Endpoint reachable from the internet",          // a signal headline
		"staging-4.acmecorp.io:8080",                    // withdrawn row subject
		"delivered 2026-08-22T09:00Z · ops.acmecorp.io", // receipt, host only
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing %q", want)
		}
	}

	// The severity ramp is present (P2.10): the by-severity bars take the severity-dot
	// tokens, and the signals table carries SeverityBadges keyed on the exact levels — the
	// design-owned "sevbadge"/bar markup. Critical is the only solid fill; the dot levels tint.
	for _, want := range []string{
		"var(--sev-critical-dot)",  // a bar fill
		"var(--sev-critical-fill)", // the critical badge, the only solid fill
		"var(--sev-high-bg)",       // a tinted badge
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing severity ramp %q", want)
		}
	}

	// Change still rides the drift palette, never the severity ramp.
	if !strings.Contains(out, "var(--drift-loss-fg)") {
		t.Error("change rows must use the drift palette")
	}

	// The standalone form is self-contained: the design token vocabulary is inlined so the
	// document renders as an email/print body, and the dark theme is defined for it.
	for _, want := range []string{
		"--sev-critical-dot:", // the token vocabulary is present
		`[data-theme="dark"]`, // the dark theme is defined for the standalone shell
	} {
		if !strings.Contains(out, want) {
			t.Errorf("self-contained document missing %q", want)
		}
	}
}

// An artifact with no delivered content renders the design-owned empty-state document
// (ADR-0110) rather than a fabricated one.
func TestRenderArtifactEmpty(t *testing.T) {
	a := Artifact{}
	if !a.Empty() {
		t.Fatal("a zero artifact must report Empty")
	}
	out := string(RenderArtifact(a))

	for _, want := range []string{
		"No delivery yet",                          // empty-state headline
		"This schedule has not delivered a report", // the honest body
	} {
		if !strings.Contains(out, want) {
			t.Errorf("empty artifact missing %q", want)
		}
	}
	// Nothing is fabricated: no KPI numerals, no change chips.
	if strings.Contains(out, "var(--drift-gain-fg)") || strings.Contains(out, "var(--drift-loss-fg)") {
		t.Error("empty artifact must not draw change chips")
	}
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
