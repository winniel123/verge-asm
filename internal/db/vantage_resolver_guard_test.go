package db

import (
	"strings"
	"testing"
)

func TestSetVantageResolverRefusesAnObservedVantage(t *testing.T) {
	sql := strings.ToLower(setVantageResolver)
	// Retention deletes observations while spans persist, so the guard reads both (ADR-0070, #1716).
	for _, want := range []string{
		"not exists (select 1 from observation",
		"not exists (select 1 from span",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("SetVantageResolver must guard on %q; got:\n%s", want, setVantageResolver)
		}
	}
	if !strings.Contains(setVantageResolver, ":execrows") {
		t.Errorf("SetVantageResolver must report rows affected so a held guard is distinguishable; got:\n%s", setVantageResolver)
	}
	if !strings.Contains(strings.ToLower(listVantages), "as observed") {
		t.Errorf("ListVantages must expose observed so the card can hide the form; got:\n%s", listVantages)
	}
}
