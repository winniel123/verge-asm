package migrations

import (
	"regexp"
	"strings"
	"testing"
)

// The schema holds the invariant, so a reader that inner-joins vantage can never
// disagree with a reader that does not (ADR-1985 §3, #1985).

var perVantageConstraints = map[string]string{
	"span":        "span_per_vantage_facet_needs_vantage",
	"observation": "observation_per_vantage_facet_needs_vantage",
}

func perVantageCheck(t *testing.T, table string) string {
	t.Helper()
	up := strings.ToLower(upMigrations(t))
	re := regexp.MustCompile(`(?s)add constraint ` + perVantageConstraints[table] + `\s+check\s*\((.*?)\)\s*;`)
	m := re.FindStringSubmatch(up)
	if m == nil {
		t.Fatalf("%s declares no %s constraint; a per-vantage row could then be stored "+
			"with no vantage, and the asset page would read `never looked` over a measured "+
			"service (#1985)", table, perVantageConstraints[table])
	}
	return m[1]
}

func TestPerVantageRowRequiresAVantage(t *testing.T) {
	for _, table := range []string{"span", "observation"} {
		pred := perVantageCheck(t, table)
		if !strings.Contains(pred, "vantage_id is not null") {
			t.Errorf("%s: the predicate must require a vantage, got: %s", table, strings.TrimSpace(pred))
		}
	}
}

func TestPerVantageCheckIsADenyList(t *testing.T) {
	// An allow-list admits a new per-vantage facet in silence (ADR-1985 §4).
	for _, table := range []string{"span", "observation"} {
		pred := perVantageCheck(t, table)
		for _, facet := range []string{"reachability", "resolution", "tls-acceptance", "http-identity"} {
			if strings.Contains(pred, facet) {
				t.Errorf("%s: the predicate names the per-vantage facet %q, so it is an "+
					"allow-list; a facet added later would pass unconstrained: %s",
					table, facet, strings.TrimSpace(pred))
			}
		}
	}
}

func TestPerVantageCheckAdmitsItsTwoExceptions(t *testing.T) {
	for _, table := range []string{"span", "observation"} {
		pred := perVantageCheck(t, table)
		// The zone reader restates a file from no network position (ADR-0027).
		if !strings.Contains(pred, "source = 'zone'") {
			t.Errorf("%s: a zone-sourced dns-record row carries no vantage and must be "+
				"admitted, got: %s", table, strings.TrimSpace(pred))
		}
		// A default certificate is not per-vantage (ADR-0129, #954).
		if !strings.Contains(pred, "facet = 'certificate'") {
			t.Errorf("%s: an edge fan-out certificate row carries no vantage and must be "+
				"admitted, got: %s", table, strings.TrimSpace(pred))
		}
	}
}

func TestPerVantageCheckIsValidated(t *testing.T) {
	// A NOT VALID constraint is silent about stored rows, which is the weaker
	// guarantee and the harder one to reason about later (ADR-1985 §6).
	up := strings.ToLower(upMigrations(t))
	for table, name := range perVantageConstraints {
		re := regexp.MustCompile(`(?s)add constraint ` + name + `\s+check\s*\(.*?\)\s*(not valid)?\s*;`)
		m := re.FindStringSubmatch(up)
		if m == nil {
			t.Fatalf("%s declares no %s constraint", table, name)
		}
		if m[1] != "" {
			t.Errorf("%s: %s is NOT VALID; it must validate the stored rows", table, name)
		}
	}
}
