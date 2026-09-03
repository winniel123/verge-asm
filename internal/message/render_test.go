package message

import (
	"strings"
	"testing"
)

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
		"acmecorp",
		"generated 2026-08-22T09:00:00Z",
		"verge v0.9.2",
		"Open signals", "47", "vs previous week",
		"Mean time to withdrawal", "2.4d",
		"Open signals by severity",
		"New this week", "edge-gw-03.acmecorp.io",
		"Endpoint reachable from the internet",
		"staging-4.acmecorp.io:8080",
		"delivered 2026-08-22T09:00Z · ops.acmecorp.io",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing %q", want)
		}
	}

	for _, want := range []string{
		"var(--sev-critical-dot)",
		"var(--sev-critical-fill)",
		"var(--sev-high-bg)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("populated artifact missing severity ramp %q", want)
		}
	}

	if !strings.Contains(out, "var(--drift-loss-fg)") {
		t.Error("change rows must use the drift palette")
	}

	for _, want := range []string{
		"--sev-critical-dot:",
		`[data-theme="dark"]`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("self-contained document missing %q", want)
		}
	}
}

func TestRenderArtifactEmpty(t *testing.T) {
	a := Artifact{}
	if !a.Empty() {
		t.Fatal("a zero artifact must report Empty")
	}
	out := string(RenderArtifact(a))

	for _, want := range []string{
		"No delivery yet",
		"This schedule has not delivered a report",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("empty artifact missing %q", want)
		}
	}
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
