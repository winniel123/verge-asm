package main

import (
	"strings"
	"testing"
)

// TestDeclarationFlatConfirmScaffolding covers #889 / ADR-0127: the declaration box
// carries a flat confirm — a visible Confirm control beside the input and a live line
// that reads "N addresses · within your cap of M" as an in-cap address scope is typed.
// The count and the "within your cap of M" text are filled client-side (the live line is
// a preview; declareSeed stays authoritative), so this asserts the scaffolding the
// script drives: the Confirm control, the live element, and the operator cap it reads.
func TestDeclarationFlatConfirmScaffolding(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := seedsBody(t, ac, base)

	// One Confirm control on the declaration form.
	if !strings.Contains(page, "data-declare-confirm") || !strings.Contains(page, ">Confirm<") {
		t.Errorf("declaration Confirm control missing; body: %s", page)
	}
	// The live line element, wired to submit the declare form and carrying the operator
	// cap (default 1024) the "within your cap of M" text reads.
	if !strings.Contains(page, "data-declare-live") {
		t.Errorf("declaration live line element missing; body: %s", page)
	}
	if !strings.Contains(page, `data-cap="1024"`) {
		t.Errorf("live line does not carry the operator cap; body: %s", page)
	}
	if !strings.Contains(page, `form="sc-declare"`) {
		t.Errorf("Confirm control not wired to the declare form; body: %s", page)
	}
}

// TestOverCapRefusalNamesRaiseRoute covers #889 / ADR-0127 + ADR-0052: above the
// operator cap the declaration is refused and NAMES the raise route (Settings), never
// taking it. The reachable in-cap set stays named (ADR-0052 route), and no cadence or
// completion warning appears — the number is priced at policy time, not gated here.
func TestOverCapRefusalNamesRaiseRoute(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// /20 is 4096 addresses, over the default 1024 cap. The refusal is a
	// post-redirect-get (ADR-0130 §1), so the callout is read off the landing GET.
	got := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/20"))
	// The refusal names the raise route.
	if !strings.Contains(got, "Raise your cap") || !strings.Contains(got, "Settings") {
		t.Errorf("over-cap refusal does not name the raise route; body: %s", got)
	}
	// The reachable in-cap set stays named (a /22 for the 1024 cap), never auto-applied.
	if !strings.Contains(got, "10.0.0.0/22") || !strings.Contains(got, "nothing is auto-corrected") {
		t.Errorf("over-cap refusal drops the reachable set; body: %s", got)
	}
	// No cadence line or completion warning at declaration (AC#4). The refusal prices the
	// raise, it does not warn the scope will not finish.
	for _, forbidden := range []string{"will not finish", "cannot finish", "won't finish", "completion warning"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("declaration refusal carries a forbidden completion warning %q; body: %s", forbidden, got)
		}
	}
}
