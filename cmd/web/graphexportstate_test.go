package main

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
)

func TestGraphExportBandNamesTheScopeAndSeverityTheControlsHold(t *testing.T) {
	src := graphTmplSource(t)

	notes := jsBlockAfter(t, src, "function exportNotes() {")
	if !strings.Contains(notes, "exportState()") {
		t.Fatalf("the export's note list never reads the graph's own controls, so a filtered or scoped export still reads as the whole estate (ADR-0168 §2, #2152); block: %s", notes)
	}
	// The callout loop carries its own guarded push, so the caption's guard is read
	// from the statements ahead of it.
	head := notes
	if i := strings.Index(notes, "forEach"); i >= 0 {
		head = notes[:i]
	}
	if !regexp.MustCompile(`if \(\w+\)[^;]*\bout\.push\b`).MatchString(head) {
		t.Errorf("the export pushes the control caption unguarded, so a page with no scope and no severity control still bands an empty line (#2152); block: %s", head)
	}

	state := jsBlockAfter(t, src, "function exportState() {")
	for _, want := range []struct{ sel, label, control string }{
		{"#gr-scope-btn .cv", "Scope", "scope the plot is bounded by"},
		{"#gr-sev-btn .cv", "Severity", "severity filter the halos are hidden by"},
	} {
		if !strings.Contains(state, want.sel) {
			t.Errorf("exportState reads no %s, so the exported image never names the %s (#2152); block: %s", want.sel, want.control, state)
		}
		if !strings.Contains(state, `"`+want.label+`"`) {
			t.Errorf("exportState names no %q, so the band's caption does not say which control the value came from (#2152); block: %s", want.label, state)
		}
	}
	if !strings.Contains(state, `": "`) {
		t.Errorf("exportState joins no label to its value, so the caption reads as a bare pair of words (#2152); block: %s", state)
	}
}

func TestGraphExportCloneKeepsTheHaloTheSeverityFilterHid(t *testing.T) {
	src := graphTmplSource(t)

	sweep := jsBlockAfter(t, src, "for (var i = 0; i < src.length; i++) {")
	// The sweep replaces the clone's whole style attribute, so a filtered halo's
	// display:none survives only when the sweep restates it.
	if !strings.Contains(sweep, `setAttribute("style", decl)`) {
		t.Fatalf("the export's style sweep no longer writes the clone's style attribute, so this test proves nothing about #2152; block: %s", sweep)
	}
	// A comment naming display:none would satisfy a bare substring, and an inverted
	// comparison would hide every element, so guard, operator and append are matched together.
	if !regexp.MustCompile(`if \([^)!]*\bdisplay\b[^)!]*===\s*"none"\)[^;]*decl \+= "display:none;"`).MatchString(sweep) {
		t.Errorf("the export's style sweep does not restate a hidden element's display:none, so the clone draws every halo while the band's caption names one severity (ADR-0168 §2, #2152); block: %s", sweep)
	}
}

func TestGraphPageServesTheExportsScopeAndSeverityCaption(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	f.addResolution(t, admin.ID, "a.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.5"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, `id="gr-sev-btn"`) {
		t.Fatalf("the graph page drew no severity control, so this test proves nothing about the export caption; body: %s", page)
	}
	// html/template rewrites a script body, so the caption is asserted on what the
	// page serves and not only on the template source.
	for _, want := range []string{
		`"#gr-scope-btn .cv", "Scope"`,
		`"#gr-sev-btn .cv", "Severity"`,
		`parts.join(" · ")`,
		`decl += "display:none;"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the served graph page's export carries no %s, so a filtered or scoped export still reads as the whole estate (#2152); body: %s", want, page)
		}
	}
}
