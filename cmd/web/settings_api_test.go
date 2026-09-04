package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAPIToggleEnableDisableRecordsWhoWhen(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	if f.instanceConfig.ApiEnabled {
		t.Fatalf("api enabled before any toggle: %+v", f.instanceConfig)
	}

	resp := postForm(t, ac, base+"/settings/api", url.Values{"enabled": {"true"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("enable toggle: status=%d, want 303", resp.StatusCode)
	}
	if !strings.HasPrefix(loc, "/settings?tab=api") {
		t.Fatalf("enable toggle redirected to %q, want a PRG back to the api tab", loc)
	}
	if !f.instanceConfig.ApiEnabled {
		t.Fatalf("enable did not persist api_enabled: %+v", f.instanceConfig)
	}
	if !f.instanceConfig.ApiUpdatedBy.Valid || f.instanceConfig.ApiUpdatedBy.Int64 != admin.ID {
		t.Fatalf("enable did not record who: got %+v, want by=%d", f.instanceConfig.ApiUpdatedBy, admin.ID)
	}
	if !f.instanceConfig.ApiUpdatedAt.Valid {
		t.Fatalf("enable did not stamp when: %+v", f.instanceConfig.ApiUpdatedAt)
	}

	page := getBody(t, ac, base+"/settings?tab=api", http.StatusOK)
	if !strings.Contains(page, "enabled") || !strings.Contains(page, "Enabled by admin") {
		t.Fatalf("enabled api tab missing badge/record; body: %s", page)
	}
	if !strings.Contains(page, "Live") {
		t.Fatalf("enabled api tab missing the live callout; body: %s", page)
	}

	resp = postForm(t, ac, base+"/settings/api", url.Values{"enabled": {"false"}})
	loc = resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("disable toggle: status=%d, want 303", resp.StatusCode)
	}
	if !strings.HasPrefix(loc, "/settings?tab=api") {
		t.Fatalf("disable toggle redirected to %q, want a PRG back to the api tab", loc)
	}
	if f.instanceConfig.ApiEnabled {
		t.Fatalf("disable did not clear api_enabled: %+v", f.instanceConfig)
	}
}

func TestViewerCannotToggleAPI(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := postForm(t, vc, base+"/settings/api", url.Values{"enabled": {"true"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer toggle: status=%d, want 403", resp.StatusCode)
	}
	if f.instanceConfig.ApiEnabled || f.instanceConfig.ApiUpdatedBy.Valid {
		t.Fatalf("viewer toggle wrote config: %+v", f.instanceConfig)
	}

	page := getBody(t, vc, base+"/settings?tab=api", http.StatusOK)
	if !strings.Contains(page, "disabled") {
		t.Fatalf("viewer cannot read the api tab; body: %s", page)
	}
	if strings.Contains(page, `action="/settings/api"`) {
		t.Fatalf("a toggle control was shown to a viewer; body: %s", page)
	}
}
