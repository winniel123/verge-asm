package main

import (
	"math"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

func TestAddressCapPersistsAndGovernsDeclaration(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	if page := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/20")); !strings.Contains(page, "over the 1,024-address cap") {
		t.Fatalf("/20 over the default cap should be refused; body: %s", page)
	}
	var resp *http.Response

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

	resp = declare(t, ac, base, "address", "10.0.0.0/20")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("/20 under the raised cap should be accepted: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	resp = postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"1024"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("cap lower: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if page := seedsBody(t, ac, base); !strings.Contains(page, "10.0.0.0/20") {
		t.Fatalf("scope declared under the higher cap was dropped after lowering the cap; body: %s", page)
	}

	if page := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.1.0.0/20")); !strings.Contains(page, "over the 1,024-address cap") {
		t.Fatalf("/20 under the lowered cap should be refused; body: %s", page)
	}
}

func TestAddressCapHasNoUpperBound(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"4294967296"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("a cap above 2^32 should be accepted (no ceiling): status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.instanceConfig.SeedAddressCap != 4294967296 {
		t.Fatalf("large cap not persisted: %d", f.instanceConfig.SeedAddressCap)
	}
}

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

func TestEffectiveCadencePricesACapSizedScope(t *testing.T) {
	// The wants pace a cap-sized scope at the governing per-host rate, every probe retried (#1127).
	if got := projectedPassLabel(effectiveCadenceSeconds(1024, addressScopePorts["hot"])); got != "≈ 2 h" {
		t.Errorf("hot /22 effective cadence = %q, want ≈ 2 h", got)
	}
	if got := projectedPassLabel(effectiveCadenceSeconds(1024, addressScopePorts["cold"])); got != "≈ 2 months" {
		t.Errorf("cold /22 effective cadence = %q, want ≈ 2 months", got)
	}
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

func TestEffectiveCadenceDividesByTheGoverningInterval(t *testing.T) {
	p := connectoutcome.DefaultProfile()
	host := netip.MustParseAddr("198.51.100.7")
	pacer := connectoutcome.NewPacer(p)
	// A hot job carries one Address, so one host exercises the whole pacer (ADR-0005).
	start := time.Unix(0, 0).UTC()
	first := pacer.Next(host, start)
	governing := pacer.Next(host, start).Sub(first)

	if want := time.Second / time.Duration(p.PerHostConnPerSec); governing != want {
		t.Fatalf("pacer spaced one host's probes %v apart, want the per-host interval %v", governing, want)
	}
	if aggregate := time.Second / time.Duration(p.PerVantageConnPerSec); governing <= aggregate {
		t.Fatalf("per-host interval %v no longer outlasts the per-vantage interval %v, so the "+
			"projection's divisor must be re-derived", governing, aggregate)
	}

	const addresses, ports = int64(1024), int64(131)
	want := float64(addresses) * float64(ports) * float64(1+p.Retries) * governing.Seconds()
	got := effectiveCadenceSeconds(addresses, ports)
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("effectiveCadenceSeconds(%d, %d) = %v s, want %v s at one probe every %v",
			addresses, ports, got, want, governing)
	}
}

func TestAddressCapControlPricesTheCost(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"262144"}}).Body.Close()

	page := body(t, get(t, ac, base+"/settings?tab=scans"))
	for _, want := range []string{
		"Address-scope cap",
		"262,144 addresses",
		"/14 IPv4",
		"probes / cadence",
		"TB / year",
		"One full pass runs",
		"24 days",
		"50 conn/s per-host",
		"longer than its daily cadence",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("cap control missing %q; body: %s", want, page)
		}
	}
	// Coverage stays evidential and Scans operational, so lag never leaks here (#891).
	coverage := body(t, get(t, ac, base+"/coverage"))
	if strings.Contains(coverage, "One full pass runs") {
		t.Errorf("effective cadence leaked onto Coverage; body: %s", coverage)
	}
}
