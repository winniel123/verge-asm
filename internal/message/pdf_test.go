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
// band, the two change sections and their subjects, and the delivery receipt — and
// it grades nothing (no valence word reaches the print copy).
func TestArtifactPDFStringsPopulated(t *testing.T) {
	got := strings.Join(artifactPDFStrings(sampleArtifact()), "\n")

	for _, want := range []string{
		"acmecorp",                       // identity
		"generated 2026-08-22T09:00:00Z", // provenance
		"verge v0.9.2",
		"Open signals", "47", "+3", "vs previous week", // KPI band, delta appended to numeral
		"Scans run", "128",
		"New this week", "edge-gw-03.acmecorp.io", // appeared section
		"Withdrawn by the world", "staging-4.acmecorp.io:8080", // withdrawn section
		"delivered 2026-08-22T09:00Z · ops.acmecorp.io", // receipt, host only
	} {
		if !strings.Contains(got, want) {
			t.Errorf("PDF content missing %q; text:\n%s", want, got)
		}
	}

	// No valence word grades the print copy — the same guarantee the HTML render
	// makes. The strings are visible text only, so the guard reads them directly.
	if ContainsValence(got) {
		t.Errorf("PDF copy carries a valence word; text:\n%s", got)
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
