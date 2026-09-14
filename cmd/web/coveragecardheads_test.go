package main

import (
	"strings"
	"testing"
)

// SPEC docs/spec/aperture-statement.md §7.4 and §7.5 — the statement, the meters and the rules
// card read three sources, so one word must not head two of them.

func TestCoverageSeparatesTheApertureLabelFromTheMeters(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	page := coverageBody(t, login(t, base, "admin", "hunter2hunter2"), base)

	for _, want := range []string{
		`<span class="cv-micro">Aperture</span><h3>What this install is configured to look at</h3>`,
		`<span class="cv-micro">Address scopes</span><h3>What the last batch walked</h3>`,
		`<span class="cv-micro">Rules</span><h3>Rules waiting on a reading</h3>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the coverage card head %q is missing; body: %s", want, page)
		}
	}
	if n := strings.Count(page, `<span class="cv-micro">Aperture</span>`); n != 1 {
		t.Errorf("the Aperture micro-label heads %d cards, want 1", n)
	}
	// ADR-0095 refuses a per-subject count under a per-rule name, so the old title cannot return.
	if strings.Contains(page, "Unevaluable this batch") {
		t.Error("the rules card still wears a per-rule name over a per-subject count")
	}
}
