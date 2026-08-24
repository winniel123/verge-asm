package message

import (
	"bytes"
	"strings"
	"testing"
)

// sampleArtifact is the populated fixture the PDF tests read — the same delivered
// report shape the HTML render test uses, kept test-local (never shipped).
func sampleArtifact() Artifact {
	return Artifact{
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
}

// A populated artifact renders a real PDF document: the bytes carry the PDF magic
// header and the end-of-file marker, and the body is non-trivial.
func TestRenderArtifactPDFPopulated(t *testing.T) {
	out, err := RenderArtifactPDF(sampleArtifact())
	if err != nil {
		t.Fatalf("RenderArtifactPDF: %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output is not a PDF (no %%PDF- header); first bytes: %q", firstBytes(out, 8))
	}
	if !bytes.Contains(out, []byte("%%EOF")) {
		t.Error("PDF output missing EOF trailer")
	}
	if len(out) < 800 {
		t.Errorf("PDF output suspiciously small: %d bytes", len(out))
	}
}

// The PDF says the same things the delivered document says: the identity, the KPI
// band, the by-severity breakdown, the signals table, the withdrawn section and its
// subjects, and the delivery receipt — and it grades nothing in prose (no valence
// word reaches the print copy; the severity ramp is the exempt one loud voice).
func TestArtifactPDFStringsPopulated(t *testing.T) {
	got := strings.Join(artifactPDFStrings(sampleArtifact()), "\n")

	for _, want := range []string{
		"acmecorp",                       // identity
		"generated 2026-08-22T09:00:00Z", // provenance
		"verge v0.9.2",
		"Open signals", "47", "+3", "vs previous week", // KPI band, delta appended to numeral
		"Scans run", "128",
		"Open signals by severity",                // severity breakdown header
		"New this week", "edge-gw-03.acmecorp.io", // signals table
		"Endpoint reachable from the internet",                 // a signal headline
		"Withdrawn by the world", "staging-4.acmecorp.io:8080", // withdrawn section
		"delivered 2026-08-22T09:00Z · ops.acmecorp.io", // receipt, host only
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PDF content missing %q; text:\n%s", want, got)
		}
	}

	// No valence word grades the print copy — the same guarantee the HTML render
	// makes. The severity ramp is the one loud voice, exempt from the graded-prose
	// guard exactly as on screen: its labels are never emitted to this view, and its
	// section title (which names the scale) is dropped here as the HTML test strips
	// the marked ramp elements. What remains is pure prose and grades nothing.
	var prose []string
	for _, s := range artifactPDFStrings(sampleArtifact()) {
		if s == artifactSeverityTitle {
			continue
		}
		prose = append(prose, s)
	}
	if joined := strings.Join(prose, "\n"); ContainsValence(joined) {
		t.Errorf("PDF copy carries a valence word; text:\n%s", joined)
	}
}

// An artifact with no delivered content renders the empty-state, never a
// fabricated document, and still produces a valid PDF.
func TestRenderArtifactPDFEmpty(t *testing.T) {
	a := Artifact{}
	if !a.Empty() {
		t.Fatal("a zero artifact must report Empty")
	}

	out, err := RenderArtifactPDF(a)
	if err != nil {
		t.Fatalf("RenderArtifactPDF(empty): %v", err)
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Error("empty-artifact output is not a PDF")
	}

	got := strings.Join(artifactPDFStrings(a), "\n")
	for _, want := range []string{
		"No report has been delivered yet", // empty-state headline
		"not delivered",                    // receipt states the fact
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty PDF content missing %q; text:\n%s", want, got)
		}
	}
	// Nothing fabricated: no sample org, no KPI numerals, no change subjects.
	for _, absent := range []string{"acmecorp", "Open signals", "edge-gw-03"} {
		if strings.Contains(got, absent) {
			t.Errorf("empty PDF must not fabricate content, found %q; text:\n%s", absent, got)
		}
	}
	if ContainsValence(got) {
		t.Errorf("empty PDF copy carries a valence word; text:\n%s", got)
	}
}

func firstBytes(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
