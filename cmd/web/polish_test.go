package main

import (
	"strings"
	"testing"
)

func TestFootCarriesScrollPreservation(t *testing.T) {
	// Losing the script returns a form action to a top-of-page reload (#246).
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsBody(t, ac, base)
	for _, want := range []string{`sessionStorage`, `"verge:scroll:"`, `window.scrollTo`} {
		if !strings.Contains(page, want) {
			t.Fatalf("rendered page missing scroll-preservation marker %q", want)
		}
	}
}

func TestFootScrollRestoreHonoursADR0130(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsBody(t, ac, base)
	for _, want := range []string{
		`return "verge:scroll:" + u.pathname + (q ? "?" + q : "");`,
		`var K = keyFor(location);`,
		// A toast receipt is not part of a page's identity, so dropping it on both ends
		// keeps a toasting act from landing under a key it never stashed.
		`parts[i].split("=")[0] !== "toast"`,
		`getEntriesByType("navigation")`,
		`nav.type === "navigate"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("rendered page missing ADR-0130 §2 marker %q", want)
		}
	}
	if strings.Contains(page, "FRESH_MS") {
		t.Fatal("the FRESH_MS freshness window is back; ADR-0130 §2 replaced it with a navigation-type gate")
	}
}

func TestFootClickStashCoversClassB(t *testing.T) {
	// A plain link never reaches the submit listener, so the click listener stashes for
	// it, under the target's key because that is what the landing page reads (ADR-0130 §3).
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsBody(t, ac, base)
	for _, want := range []string{
		`document.addEventListener("click"`,
		`closest("a[href], button[type=button], [data-href]")`,
		`if (u.origin !== location.origin || u.pathname !== location.pathname) return;`,
		`stash(keyFor(u));`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("rendered page missing ADR-0130 §3 class-B marker %q", want)
		}
	}
}
