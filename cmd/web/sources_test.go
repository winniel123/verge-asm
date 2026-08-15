package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// --- fakeStore source-state methods ----------------------------------------

func (f *fakeStore) ListSourceStates(context.Context) ([]db.ListSourceStatesRow, error) {
	rows := make([]db.ListSourceStatesRow, 0, len(f.sourceStates))
	for _, st := range f.sourceStates {
		rows = append(rows, db.ListSourceStatesRow{
			Slug: st.Slug, Enabled: st.Enabled, ToggledBy: st.ToggledBy,
			ToggledAt: st.ToggledAt, ToggledByUsername: f.accounts[st.ToggledBy].Username,
		})
	}
	return rows, nil
}

func (f *fakeStore) UpsertSourceState(_ context.Context, arg db.UpsertSourceStateParams) (db.SourceState, error) {
	st := db.SourceState{
		Slug: arg.Slug, Enabled: arg.Enabled, ToggledBy: arg.ToggledBy,
		ToggledAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.sourceStates[arg.Slug] = st
	return st, nil
}

// --- helpers ----------------------------------------------------------------

func toggleSourceReq(t *testing.T, c *http.Client, base, slug, enabled string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/sources/toggle", url.Values{"slug": {slug}, "enabled": {enabled}})
}

func sourcesBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	resp, err := c.Get(base + "/sources")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sources status = %d, want 200", resp.StatusCode)
	}
	return body(t, resp)
}

// --- tests ------------------------------------------------------------------

// The modal renders every catalogued source, both marked groups, and the
// shipped defaults from §3.1 with no override in place.
func TestSourcesModalRendersCatalogueAndDefaults(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)

	// Every catalogued source appears.
	for _, c := range sourceCatalog {
		if !strings.Contains(page, c.Name) {
			t.Errorf("source %q missing from the modal", c.Name)
		}
	}
	// Both marked groups render.
	if !strings.Contains(page, "What you may be able to resolve") ||
		!strings.Contains(page, "What nobody has been able to resolve") {
		t.Errorf("the two marked groups are not both rendered; body: %s", page)
	}
	// A proposer is labelled a proposer, never a source (ADR-0012).
	if !strings.Contains(page, ">proposer<") || !strings.Contains(page, ">source<") {
		t.Errorf("source/proposer kinds not distinguished; body: %s", page)
	}
	// crt.sh ships on; RIPEstat ships off — the §3.1 defaults, unoverridden.
	if !strings.Contains(page, "crt.sh") || !strings.Contains(page, "RIPEstat") {
		t.Errorf("expected sources missing; body: %s", page)
	}
}

// LACNIC's actionable group is empty by construction, and it still renders — the
// #47 "render even when empty" requirement.
func TestEmptyMarkedGroupStillRenders(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := sourcesBody(t, ac, base)
	if !strings.Contains(page, "LACNIC registry") {
		t.Fatalf("LACNIC not rendered; body: %s", page)
	}
	if !strings.Contains(page, "every open question here is one nobody has been able to answer") {
		t.Errorf("empty actionable group did not render its empty state; body: %s", page)
	}
}

// Toggling persists the override and flips the effective state; the default is
// restored by toggling back.
func TestToggleSourcePersistsOverride(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// RIPEstat ships off; enable it.
	resp := toggleSourceReq(t, ac, base, "ripestat", "true")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/sources" {
		t.Fatalf("toggle: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if st, ok := f.sourceStates["ripestat"]; !ok || !st.Enabled {
		t.Fatalf("override not persisted: %+v", f.sourceStates["ripestat"])
	}

	// crt.sh ships on; disabling is safe and persists.
	toggleSourceReq(t, ac, base, "crtsh", "false").Body.Close()
	if st, ok := f.sourceStates["crtsh"]; !ok || st.Enabled {
		t.Fatalf("disable override not persisted: %+v", f.sourceStates["crtsh"])
	}
}

// A source excluded on terms has no consent instrument the modal operator can
// satisfy, so it cannot be toggled; an unknown slug is refused too.
func TestToggleRejectsBarredAndUnknown(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, slug := range []string{"hackertarget", "no-such-source"} {
		resp := toggleSourceReq(t, ac, base, slug, "true")
		got := resp.StatusCode
		resp.Body.Close()
		if got != http.StatusBadRequest {
			t.Errorf("toggle %q: status=%d, want 400", slug, got)
		}
	}
	if len(f.sourceStates) != 0 {
		t.Fatalf("no override should have been written; got %d", len(f.sourceStates))
	}
}

// Toggling is an admin act: a viewer is denied and no state is written, but a
// viewer may still read the modal — without a toggle control.
func TestViewerCannotToggleButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := toggleSourceReq(t, vc, base, "ripestat", "true")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer toggle: status=%d, want 403", resp.StatusCode)
	}
	if len(f.sourceStates) != 0 {
		t.Fatalf("viewer toggle wrote state; got %d", len(f.sourceStates))
	}

	page := sourcesBody(t, vc, base)
	if !strings.Contains(page, "crt.sh") {
		t.Errorf("viewer cannot read the modal; body: %s", page)
	}
	if strings.Contains(page, `action="/sources/toggle"`) {
		t.Errorf("a toggle control was shown to a viewer; body: %s", page)
	}
}

// The Coverage stub is the modal's entry point and links to it.
func TestCoverageStubLinksToModal(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp, err := ac.Get(base + "/coverage")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /coverage status=%d, want 200", resp.StatusCode)
	}
	if !strings.Contains(got, `href="/sources"`) {
		t.Errorf("Coverage stub does not link to the source modal; body: %s", got)
	}
}

// Both mutating and reading source routes require a login.
func TestSourceRoutesRequireLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)

	resp := toggleSourceReq(t, c, base, "ripestat", "true")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon toggle: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, err := c.Get(base + "/sources")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon GET /sources: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
}
