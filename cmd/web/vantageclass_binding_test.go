package main

import (
	"go/ast"
	"sort"
	"strings"
	"testing"
)

// What this gate proves, stated honestly (ADR-1945 §3).
//
// It proves that each named comparison below reaches an address-scope binding producer at exactly
// one site on its static call graph. It does not prove that both legs read the same value. A
// comparison that binds once, then rebinds the name before its second leg, passes here. ADR-1895
// §10's two per-site tests hold the value. This gate holds the shape, and neither replaces the
// other.
//
// Three coverage gaps, named rather than smoothed (ADR-1945 §3).
//
//  1. The root set is a list, not a derivation. A wholly new comparison escapes this gate,
//     because nobody adds its function to webComparisonRoots. What the gate does change is the
//     failure mode of a stale entry: a root naming a function this package no longer declares
//     fails, rather than passing in silence.
//  2. It counts acquisition sites, never executions. The walk visits each function once, so a
//     helper that binds once and is called from two places reads as one site. A loop around one
//     acquisition reads as one site too.
//  3. Dynamic dispatch is invisible. A binding reached through a function value, a struct field
//     or an interface method is not a call on the static graph. Every site in this package calls
//     the producer directly today, and the gate reads that tree.
//
// This gate carries no exemption list. internal/queue's twin does, for rulesOpenedByGapClose.
// Nothing in cmd/web acquires a binding inside a comparison and is right to.

// The one producer in this package. The gate keys on the name, so a rename that left this map
// behind would read every comparison as binding zero times, which the count check below refuses.

var webBindingProducers = map[string]string{
	"s.addressScopeCovered": "reads the address-scope Seeds and the live exclusion corpus (ADR-0133 §4)",
}

// Each entry compares across time, so ADR-1945's rule governs its whole call graph. Each must
// bind exactly once: a count of zero means the gate lost the tree rather than that the rule holds.

var webComparisonRoots = map[string]string{
	"s.exposureCountDeltas": "the exposed, firewalled and not-reached deltas, over two snapshots",
	"s.dashboardDeltas":     "the dashboard fold, which holds the exposure comparison and three more",
}

func (c *contractPkg) bindingSites(root string, producers map[string]string) []string {
	var sites []string
	c.reach(root, map[string]bool{}, func(short string, fn *ast.FuncDecl) {
		for _, callee := range c.calleesOf(fn) {
			if _, ok := producers[callee]; ok {
				sites = append(sites, short+" → "+shortName(callee))
			}
		}
	})
	sort.Strings(sites)
	return sites
}

func TestEveryWebComparisonBindsTheAddressScopeOnce(t *testing.T) {
	c := parseWebPackage(t)
	for root, why := range webComparisonRoots {
		sites := c.bindingSites(root, webBindingProducers)
		switch {
		case len(sites) == 1:
		case len(sites) == 0:
			t.Errorf("%s (%s) reaches no address-scope binding (ADR-1945 §2).\n"+
				"Either the comparison no longer classifies, and the entry must go, or the\n"+
				"producer moved out of webBindingProducers and this gate now proves nothing.",
				root, why)
		default:
			t.Errorf("%s (%s) acquires %d address-scope bindings (ADR-1945 §2):\n  %s\n"+
				"Every leg of one comparison classifies under one binding. Build it once and\n"+
				"pass it down, the way readExposureLegs does.",
				root, why, len(sites), strings.Join(sites, "\n  "))
		}
	}
}

// An entry naming a function the package no longer declares would sit here reading as compliance,
// so the gate asserts both tables against the live tree.

func TestWebBindingGateNamesAreLive(t *testing.T) {
	c := parseWebPackage(t)
	for name := range webBindingProducers {
		if _, ok := c.decl(name); !ok {
			t.Errorf("webBindingProducers names %q, which this package no longer declares; "+
				"re-point the entry at the producer's new name", name)
		}
	}
	for name := range webComparisonRoots {
		if _, ok := c.decl(name); !ok {
			t.Errorf("webComparisonRoots names %q, which this package no longer declares; "+
				"delete the entry, or re-point it at the comparison's new name", name)
		}
	}
}
