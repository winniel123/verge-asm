package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestAddressCapPersistsAndGovernsDeclaration covers #888 / ADR-0127: the operator
// address-scope cap persists, is read at declaration, has no upper bound, and a
// lowered cap never invalidates a scope declared under a higher one.
func TestAddressCapPersistsAndGovernsDeclaration(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Under the shipped default (1024), a /20 (4096 addresses) is over cap. The refusal
	// is a post-redirect-get (ADR-0130 §1), so its callout renders on the landing GET.
	if page := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/20")); !strings.Contains(page, "over the 1,024-address cap") {
		t.Fatalf("/20 over the default cap should be refused; body: %s", page)
	}
	var resp *http.Response

	// Raise the cap to 65536. It persists and attributes who set it.
	resp = postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"65536"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("cap raise: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.instanceConfig.SeedAddressCap != 65536 {
		t.Fatalf("cap not persisted: %d", f.instanceConfig.SeedAddressCap)
	}
	if !f.instanceConfig.SeedAddressCapUpdatedBy.Valid {
		t.Errorf("updated_by not attributed")
	}

	// The same /20 now declares — the raised cap took effect at declaration.
	resp = declare(t, ac, base, "address", "10.0.0.0/20")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("/20 under the raised cap should be accepted: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	// Lower the cap below the declared scope. ADR-0127: the cap is read at declaration
	// only, so the already-declared /20 is NOT invalidated.
	resp = postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"1024"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("cap lower: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if page := seedsBody(t, ac, base); !strings.Contains(page, "10.0.0.0/20") {
		t.Fatalf("scope declared under the higher cap was dropped after lowering the cap; body: %s", page)
	}

	// A NEW /20 is refused again now the cap is back to 1024 — the cap governs the
	// declaration, live.
	if page := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.1.0.0/20")); !strings.Contains(page, "over the 1,024-address cap") {
		t.Fatalf("/20 under the lowered cap should be refused; body: %s", page)
	}
}

// TestAddressCapHasNoUpperBound covers ADR-0127's load-bearing ruling: nothing gates a
// value above the operator's own cap — a value beyond 2^32 (no IPv4 purpose) is
// accepted, priced not gated.
func TestAddressCapHasNoUpperBound(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"4294967296"}}) // 2^32
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("a cap above 2^32 should be accepted (no ceiling): status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.instanceConfig.SeedAddressCap != 4294967296 {
		t.Fatalf("large cap not persisted: %d", f.instanceConfig.SeedAddressCap)
	}
}

// TestAddressCapRejectsInvalid covers the sole guard: a whole number of addresses, one
// or more. A rejected value leaves the previous cap standing.
//
// The refusal is a post-redirect-get since ticket #978 (ADR-0130 §1): the 303 goes back
// to the URL the dial was submitted from, and the callout and the typed value ride the
// session flash to that landing GET. Nothing the operator typed enters the URL.
func TestAddressCapRejectsInvalid(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"4096"}}).Body.Close()

	const tab = "/settings?tab=scans"
	for _, bad := range []string{"0", "-5", "abc", ""} {
		resp := postForm(t, ac, base+"/settings/address-cap", url.Values{
			"address_cap": {bad}, "return": {tab},
		})
		if loc := submitLoc(t, resp); loc != tab {
			t.Fatalf("refused cap %q landed at %q, want %q", bad, loc, tab)
		}
		page := getBody(t, ac, base+tab, http.StatusOK)
		if !strings.Contains(page, "one or more") {
			t.Fatalf("invalid cap %q: no callout on the landing page; body: %s", bad, page)
		}
		if f.instanceConfig.SeedAddressCap != 4096 {
			t.Fatalf("rejected cap %q mutated the stored value: %d", bad, f.instanceConfig.SeedAddressCap)
		}
	}
}

// TestEffectiveCadenceMatchesADR0047 pins the effective-cadence arithmetic (#891, decision
// #847) against ADR-0047's own worst-case figures: at a /22 cap (1,024 addresses) the hot
// pass is ~34 min and the full-range cold pass ~12 days — the upper ends of ADR-0047's
// "12-36 minutes" and "3.9-11.6 days" ranges, the pass where every probe exhausts its retry
// budget. It also guards projectedPassLabel's unit boundaries.
func TestEffectiveCadenceMatchesADR0047(t *testing.T) {
	// A /22 is 1,024 addresses. hot probes 131 ports, cold 65,535, both x3 attempts / 200 pkt/s.
	if got := projectedPassLabel(effectiveCadenceSeconds(1024, addressScopePorts["hot"])); got != "≈ 34 min" {
		t.Errorf("hot /22 effective cadence = %q, want ≈ 34 min", got)
	}
	if got := projectedPassLabel(effectiveCadenceSeconds(1024, addressScopePorts["cold"])); got != "≈ 12 days" {
		t.Errorf("cold /22 effective cadence = %q, want ≈ 12 days", got)
	}
	// projectedPassLabel unit boundaries: seconds, hours, days, months, years.
	for _, tc := range []struct {
		seconds float64
		want    string
	}{
		{30, "≈ under a minute"},
		{90, "≈ 2 min"},
		{7200, "≈ 2 h"},
		{3 * 86400, "≈ 3 days"},
		{60 * 86400, "≈ 2 months"},
		{800 * 86400, "≈ 2.2 years"},
	} {
		if got := projectedPassLabel(tc.seconds); got != tc.want {
			t.Errorf("projectedPassLabel(%v) = %q, want %q", tc.seconds, got, tc.want)
		}
	}
}

func TestAddressCapControlPricesTheCost(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// 262144 = 2^18, a /14 IPv4 (a /110 IPv6).
	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"262144"}}).Body.Close()

	page := body(t, get(t, ac, base+"/settings?tab=scans"))
	// The default fakeStore enables the hot scan on a daily cadence, so the sweep-load
	// readout prices a cap-sized scope against it. At 262,144 addresses the hot pass
	// (262144 x 131 ports x 3 attempts / 200 pkt/s ≈ 5.96 days) outpaces its daily
	// cadence, so the effective-cadence readout (#891) states the honest lag.
	for _, want := range []string{
		"Address-scope cap",
		"262,144 addresses",
		"/14 IPv4",
		"probes / cadence",
		"TB / year",
		"One full pass runs",
		"6 days",
		"longer than its daily cadence",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("cap control missing %q; body: %s", want, page)
		}
	}
	// The effective cadence is a Scans-surface figure only — it never leaks onto Coverage
	// (#891 acceptance: Coverage is evidential, Scans operational).
	coverage := body(t, get(t, ac, base+"/coverage"))
	if strings.Contains(coverage, "One full pass runs") {
		t.Errorf("effective cadence leaked onto Coverage; body: %s", coverage)
	}
}
