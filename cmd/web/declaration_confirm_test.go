package main

import (
	"strings"
	"testing"
)

func TestDeclarationFlatConfirmScaffolding(t *testing.T) {
	// The live line's count is filled client-side, so only its scaffolding is assertable.
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := seedsBody(t, ac, base)

	if !strings.Contains(page, "data-declare-confirm") || !strings.Contains(page, ">Confirm<") {
		t.Errorf("declaration Confirm control missing; body: %s", page)
	}
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

func TestOverCapRefusalNamesRaiseRoute(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/20"))
	if !strings.Contains(got, "Raise your cap") || !strings.Contains(got, "Settings") {
		t.Errorf("over-cap refusal does not name the raise route; body: %s", got)
	}
	if !strings.Contains(got, "10.0.0.0/22") || !strings.Contains(got, "nothing is auto-corrected") {
		t.Errorf("over-cap refusal drops the reachable set; body: %s", got)
	}
	for _, forbidden := range []string{"will not finish", "cannot finish", "won't finish", "completion warning"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("declaration refusal carries a forbidden completion warning %q; body: %s", forbidden, got)
		}
	}
}
