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

	// Under the shipped default (1024), a /20 (4096 addresses) is over cap.
	resp := declare(t, ac, base, "address", "10.0.0.0/20")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("/20 over the default cap should be refused: status=%d", resp.StatusCode)
	}
	resp.Body.Close()

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
	resp = declare(t, ac, base, "address", "10.1.0.0/20")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("/20 under the lowered cap should be refused: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
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
func TestAddressCapRejectsInvalid(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"4096"}}).Body.Close()

	for _, bad := range []string{"0", "-5", "abc", ""} {
		resp := postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {bad}})
		got := body(t, resp)
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "one or more") {
			t.Fatalf("invalid cap %q not refused: status=%d body=%s", bad, resp.StatusCode, got)
		}
		if f.instanceConfig.SeedAddressCap != 4096 {
			t.Fatalf("rejected cap %q mutated the stored value: %d", bad, f.instanceConfig.SeedAddressCap)
		}
	}
}

// TestAddressCapControlPricesTheCost covers the Variant C readout on the Scans tab: the
// largest scope the cap admits, the per-cadence sweep load on each enabled address-scope
// scan, and the projected evidential disk growth.
func TestAddressCapControlPricesTheCost(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// 262144 = 2^18, a /14 IPv4 (a /110 IPv6).
	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"262144"}}).Body.Close()

	page := body(t, get(t, ac, base+"/settings?tab=scans"))
	// The default fakeStore enables the hot scan on a daily cadence, so the sweep-load
	// readout prices a cap-sized scope against it.
	for _, want := range []string{
		"Address-scope cap",
		"262,144 addresses",
		"/14 IPv4",
		"probes / cadence",
		"TB / year",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("cap control missing %q; body: %s", want, page)
		}
	}
}
