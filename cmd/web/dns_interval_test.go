package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDnsIntervalIsConfigurable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, `action="/seeds/dns/interval"`) {
		t.Errorf("no dns interval form on /scope; body: %s", page)
	}

	resp := postForm(t, ac, base+"/seeds/dns/interval", url.Values{"interval_days": {"7"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("set interval: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if f.dnsCadence != 7*86400 {
		t.Errorf("dns cadence = %d, want 7 days in seconds", f.dnsCadence)
	}

	resp = postForm(t, ac, base+"/seeds/dns/interval", url.Values{"interval_days": {"0"}})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "between 1 and 3,650 days") {
		t.Fatalf("zero interval not rejected; body=%s", got)
	}
}

func TestDnsIntervalIsCappedAtTenYears(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, raw := range []string{"3651", "99999999999999999999"} {
		resp := postForm(t, ac, base+"/seeds/dns/interval", url.Values{"interval_days": {raw}})
		got := refusalPage(t, ac, base, resp)
		if !strings.Contains(got, "between 1 and 3,650 days") {
			t.Fatalf("interval %s not refused as a form error; body=%s", raw, got)
		}
		if !strings.Contains(got, `value="`+raw+`"`) {
			t.Errorf("refused interval %s not retained in the form", raw)
		}
		if f.dnsCadence != 0 {
			t.Fatalf("interval %s reached the store: cadence=%d", raw, f.dnsCadence)
		}
	}

	resp := postForm(t, ac, base+"/seeds/dns/interval", url.Values{"interval_days": {"3650"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || f.dnsCadence != 3650*86400 {
		t.Fatalf("3650 days at the cap: status=%d cadence=%d", resp.StatusCode, f.dnsCadence)
	}
}

func TestDnsRowStatesTheCurrencyBound(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	if page := seedsBody(t, ac, base); !strings.Contains(page, "current for 2 days") {
		t.Errorf("shipped daily cadence does not state a 2-day currency bound; body: %s", page)
	}

	postForm(t, ac, base+"/seeds/dns/interval", url.Values{"interval_days": {"7"}}).Body.Close()
	// The bound is k cadences and k is 2, so the row moves with the dial (ADR-0084).
	if page := seedsBody(t, ac, base); !strings.Contains(page, "current for 14 days") {
		t.Errorf("7-day cadence does not state a 14-day currency bound; body: %s", page)
	}
}

func TestDnsIntervalRendersFromFixturesInDevMode(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	srv := newServer(f, testKey, "", fixedClock())
	srv.devMode = true
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)

	ac := login(t, ts.URL, "admin", "hunter2hunter2")
	page := seedsBody(t, ac, ts.URL)

	if !strings.Contains(page, `value="1"`) || !strings.Contains(page, "current for 2 days") {
		t.Errorf("dev scope does not render the pinned dns interval; body: %s", page)
	}
	// A key the fixture map omits renders as "<no value>" and raises no error (ADR-0167 §1).
	if strings.Contains(page, "<no value>") {
		t.Errorf("dev scope renders a template key the fixture map omits; body: %s", page)
	}
}
