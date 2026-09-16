package main

import (
	"strings"
	"testing"
)

// The handler composes a comparison one level above the two functions ADR-1945 fences: the stat
// band renders a figure from the fold beside a change from the deltas. Both legs of that pair are
// one comparison, so the whole handler binds the address scope once (ADR-1945 §1, #2046).
//
// This gate reads the same static tree as TestEveryWebComparisonBindsTheAddressScopeOnce and
// carries its coverage gaps 1 to 3. It holds the shape and never the value.
//
// It sits here rather than in webComparisonRoots because the handler pays only half of that
// table's gap 4: dashboardData still binds five times. An entry there would also earn
// TestWebBindingGateNamesAreLive's rename protection, so this gate is a stand-in until the
// dashboard lands and gap 4 can be struck.

func TestExposurePageBindsTheAddressScopeOnce(t *testing.T) {
	c := parseWebPackage(t)
	sites := c.bindingSites("s.exposurePage", webBindingProducers)
	switch {
	case len(sites) == 1:
	case len(sites) == 0:
		t.Error("s.exposurePage reaches no address-scope binding, so this gate proves nothing;\n" +
			"either the handler stopped classifying or the producer left webBindingProducers")
	default:
		t.Errorf("s.exposurePage acquires %d address-scope bindings (ADR-1945 §2):\n  %s\n"+
			"The band's figure and its change classify under one boundary. Acquire the binding in\n"+
			"the handler and pass it down, the way foldExposureUnder takes it.",
			len(sites), strings.Join(sites, "\n  "))
	}
}
