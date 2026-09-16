package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
)

func TestGraphExportSerialiserReadsEveryCalloutThePageDrew(t *testing.T) {
	raw, err := fs.ReadFile(designfs.FS, "templates/graph.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	tmplSrc := string(raw)

	for _, want := range []string{
		`document.querySelectorAll(".gr-main .gr-callout")`,
		`exportNotes()`,
		`clone.appendChild(tn)`,
	} {
		if !strings.Contains(tmplSrc, want) {
			t.Errorf("graph.tmpl's export serialiser missing %q, so a note the page drew can leave the image", want)
		}
	}

	// Every note the page can draw is a .gr-callout, so the one selector must reach all three.
	for _, gate := range []string{".Graph.ScopesFailed", ".Graph.SignalsFailed", ".Graph.Capped"} {
		i := strings.Index(tmplSrc, "{{if "+gate+"}}")
		if i < 0 {
			t.Errorf("graph.tmpl draws no note for %s", gate)
			continue
		}
		if !strings.HasPrefix(tmplSrc[i:], "{{if "+gate+`}}<div class="gr-callout">`) {
			t.Errorf("the %s note is not a .gr-callout, so the export's note collection misses it", gate)
		}
	}
}

func TestGraphPageShipsTheExportNoteBandWithItsCapStatement(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	if _, err := f.CreateNameSeed(context.Background(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: "example.com", Valid: true}, CreatedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		f.addResolution(t, admin.ID, fmt.Sprintf("n%02d.example.com", i), "dns", obsClock,
			`{"outcome":"Resolved","addresses":["203.0.113.5"]}`)
	}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/graph", http.StatusOK)

	if !strings.Contains(page, `class="gr-callout"`) || !strings.Contains(page, "Showing 20 of 25 names.") {
		t.Fatalf("graph page stated no cap, so this test proves nothing about the export; body: %s", page)
	}
	if !strings.Contains(page, `document.querySelectorAll(".gr-main .gr-callout")`) {
		t.Errorf("the page that states a cap ships no export serialiser that reads it; body: %s", page)
	}
}
