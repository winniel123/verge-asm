package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func settingsBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	resp, err := c.Get(base + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /settings status = %d, want 200", resp.StatusCode)
	}
	return body(t, resp)
}

func TestSettingsIsAdminOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	// A viewer is refused the whole destination.
	vc := login(t, base, "viewer", "hunter2hunter2")
	resp, err := vc.Get(base + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer GET /settings: status=%d, want 403", resp.StatusCode)
	}

	// An anonymous request is bounced to login.
	resp, err = newClient(t).Get(base + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon GET /settings: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	// An admin sees the three sections.
	ac := login(t, base, "admin", "hunter2hunter2")
	page := settingsBody(t, ac, base)
	for _, want := range []string{"Accounts", "Channels", "Retention dials"} {
		if !strings.Contains(page, want) {
			t.Errorf("settings page missing %q section", want)
		}
	}
}

func TestChannelCreateListAndSecretWriteOnly(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Default: all three classes when none is unchecked would be sent by the
	// form; here we send an explicit subset and a secret.
	resp := postForm(t, ac, base+"/settings/channels", url.Values{
		"url": {"https://hooks.example.com/verge"}, "coverage": {"on"}, "secret": {"s3cr3t-signing-key"},
	})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings" {
		t.Fatalf("create channel: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	if len(f.channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(f.channels))
	}
	ch := f.channels[0]
	if ch.drift || !ch.coverage || ch.clock {
		t.Errorf("routing subset not persisted: %+v", ch)
	}
	if ch.secret.String != "s3cr3t-signing-key" || !ch.secret.Valid {
		t.Errorf("secret not stored: %+v", ch.secret)
	}

	// The secret is write-only: the page shows it is set, never the value.
	page := settingsBody(t, ac, base)
	if strings.Contains(page, "s3cr3t-signing-key") {
		t.Errorf("secret value leaked into the rendered page")
	}
	if !strings.Contains(page, "https://hooks.example.com/verge") {
		t.Errorf("channel URL not listed; body: %s", page)
	}
	if !strings.Contains(page, ">set<") {
		t.Errorf("secret set-state not shown; body: %s", page)
	}
}

func TestChannelURLValidation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	cases := []struct {
		name    string
		vals    url.Values
		wantOK  bool
		wantMsg string
	}{
		{"relative", url.Values{"url": {"/hook"}, "drift": {"on"}}, false, "absolute URL"},
		{"plain http", url.Values{"url": {"http://example.com/h"}, "drift": {"on"}}, false, "loopback"},
		{"loopback http", url.Values{"url": {"http://127.0.0.1:9000/h"}, "drift": {"on"}}, true, ""},
		{"https ok", url.Values{"url": {"https://ok.example.com"}, "drift": {"on"}}, true, ""},
		{"no class", url.Values{"url": {"https://ok.example.com"}}, false, "at least one routing class"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := postForm(t, ac, base+"/settings/channels", tc.vals)
			if tc.wantOK {
				if resp.StatusCode != http.StatusSeeOther {
					t.Fatalf("status=%d, want 303 (%s)", resp.StatusCode, body(t, resp))
				}
				resp.Body.Close()
				return
			}
			got := body(t, resp)
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, tc.wantMsg) {
				t.Fatalf("status=%d body=%s, want 400 containing %q", resp.StatusCode, got, tc.wantMsg)
			}
		})
	}
}

func TestChannelUpdateAndSecretLifecycle(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/settings/channels", url.Values{
		"url": {"https://a.example.com"}, "drift": {"on"}, "secret": {"first"},
	}).Body.Close()
	id := f.channels[0].id
	idStr := itoa(id)

	// A blank secret field keeps the stored one; the URL and classes update.
	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "enabled": {"on"},
	}).Body.Close()
	ch := f.channels[0]
	if ch.url != "https://b.example.com" || !ch.clock || ch.drift {
		t.Fatalf("update did not persist url/classes: %+v", ch)
	}
	if ch.secret.String != "first" {
		t.Fatalf("blank secret should keep existing; got %q", ch.secret.String)
	}

	// A typed value replaces it.
	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "secret": {"second"},
	}).Body.Close()
	if f.channels[0].secret.String != "second" {
		t.Fatalf("secret not replaced; got %q", f.channels[0].secret.String)
	}

	// The clear box removes it.
	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "clear_secret": {"on"}, "secret": {"ignored"},
	}).Body.Close()
	if f.channels[0].secret.Valid {
		t.Fatalf("clear box should null the secret; got valid=%v", f.channels[0].secret.Valid)
	}

	// Delete removes the row.
	postForm(t, ac, base+"/settings/channels/delete", url.Values{"id": {idStr}}).Body.Close()
	if len(f.channels) != 0 {
		t.Fatalf("channel not deleted; %d remain", len(f.channels))
	}
}

func TestRoleAssignmentAndLastAdminGuard(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	viewer := seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Promote the viewer to admin.
	postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(viewer.ID)}, "role": {roleAdmin},
	}).Body.Close()
	if f.accounts[viewer.ID].Role != roleAdmin {
		t.Fatalf("role not promoted; got %q", f.accounts[viewer.ID].Role)
	}

	// Now two admins: demoting one is allowed.
	postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(viewer.ID)}, "role": {roleViewer},
	}).Body.Close()

	// Demoting the last admin is refused.
	resp := postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(admin.ID)}, "role": {roleViewer},
	})
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "last admin") {
		t.Fatalf("last-admin demotion not refused: status=%d body=%s", resp.StatusCode, got)
	}
	if f.accounts[admin.ID].Role != roleAdmin {
		t.Fatalf("last admin was demoted despite the guard")
	}
}

func TestInviteAccountFromSettings(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/accounts", url.Values{
		"username": {"reviewer"}, "password": {"hunter2hunter2"}, "role": {roleViewer},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("invite: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if _, ok := f.byName["reviewer"]; !ok {
		t.Fatalf("account not created")
	}

	page := settingsBody(t, ac, base)
	if !strings.Contains(page, "reviewer") {
		t.Errorf("new account not listed; body: %s", page)
	}
}

func TestRetentionPersistsAndValidates(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Valid values persist.
	resp := postForm(t, ac, base+"/settings/retention", url.Values{
		"observation_currency_days": {"90"}, "dispatch_cadence_multiple": {"4"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("retention save: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.retention.ObservationCurrencyDays != 90 || f.retention.DispatchCadenceMultiple != 4 {
		t.Fatalf("dials not persisted: %+v", f.retention)
	}
	if !f.retention.UpdatedBy.Valid {
		t.Errorf("updated_by not attributed")
	}

	// A negative value is refused and the previous value stands.
	resp = postForm(t, ac, base+"/settings/retention", url.Values{
		"observation_currency_days": {"-1"}, "dispatch_cadence_multiple": {"4"},
	})
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "zero or more") {
		t.Fatalf("negative dial not refused: status=%d body=%s", resp.StatusCode, got)
	}
	if f.retention.ObservationCurrencyDays != 90 {
		t.Fatalf("rejected save mutated the dial: %+v", f.retention)
	}

	// A Dispatch multiple below the k=2 floor is refused; the previous value
	// stands. The dial is a multiple of the slowest enabled Scan's cadence, so
	// one cadence is below the floor.
	resp = postForm(t, ac, base+"/settings/retention", url.Values{
		"observation_currency_days": {"90"}, "dispatch_cadence_multiple": {"1"},
	})
	got = body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "at least 2 cadences") {
		t.Fatalf("below-floor dispatch dial not refused: status=%d body=%s", resp.StatusCode, got)
	}
	if f.retention.DispatchCadenceMultiple != 4 {
		t.Fatalf("rejected save mutated the dispatch dial: %+v", f.retention)
	}

	// Zero (unbounded, the v1 default) is always allowed.
	resp = postForm(t, ac, base+"/settings/retention", url.Values{
		"observation_currency_days": {"90"}, "dispatch_cadence_multiple": {"0"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("unbounded (0) dispatch dial refused: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.retention.DispatchCadenceMultiple != 0 {
		t.Fatalf("unbounded dial not persisted: %+v", f.retention)
	}
}
