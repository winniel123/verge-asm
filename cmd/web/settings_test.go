package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
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

// settingsTabBody fetches one Settings sub-tab (#281): the seven sections are
// query-param tabs, so a folded section renders at /settings?tab=<id>.
func settingsTabBody(t *testing.T, c *http.Client, base, tab string) string {
	t.Helper()
	resp, err := c.Get(base + "/settings?tab=" + tab)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /settings?tab=%s status = %d, want 200", tab, resp.StatusCode)
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

	// An admin reaches all seven sub-tabs, and each folded section renders on its
	// own tab.
	ac := login(t, base, "admin", "hunter2hunter2")
	page := settingsBody(t, ac, base)
	for _, tab := range []string{"tab=scans", "tab=vantages", "tab=sso", "tab=team", "tab=audit", "tab=sources", "tab=aperture", "tab=instance", "tab=channels", "tab=integrations", "tab=messages", "tab=delivery"} {
		if !strings.Contains(page, tab) {
			t.Errorf("settings tab bar missing %q", tab)
		}
	}
	if !strings.Contains(settingsTabBody(t, ac, base, "team"), "Who can sign in") {
		t.Error("team tab missing the members section")
	}
	if !strings.Contains(settingsTabBody(t, ac, base, "sso"), "Add an OpenID Connect provider") {
		t.Error("sso tab missing the add-provider form")
	}
	if !strings.Contains(settingsTabBody(t, ac, base, "channels"), "Declare a channel") {
		t.Error("channels tab missing the channel form")
	}
	if !strings.Contains(settingsTabBody(t, ac, base, "delivery"), "Retention dials") {
		t.Error("delivery tab missing the retention dials")
	}
	if !strings.Contains(settingsTabBody(t, ac, base, "aperture"), "Sensitive tier") {
		t.Error("aperture tab missing the port aperture")
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
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/settings?tab=channels" {
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
	page := settingsTabBody(t, ac, base, "channels")
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

// The Team invite dialog mints an invite against T19's invite table (#313) — it no
// longer creates an account directly. A minted invite carries the chosen role and a
// hashed token, and the plaintext join link is revealed once in the response.
func TestInviteMintsAgainstInviteTable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	adminID := f.byName["admin"]
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := body(t, postForm(t, ac, base+"/settings/accounts", url.Values{"role": {roleViewer}}))

	if len(f.invites) != 1 {
		t.Fatalf("invites minted = %d, want 1", len(f.invites))
	}
	inv := f.invites[0]
	if inv.Role != roleViewer {
		t.Errorf("invite role = %q, want viewer", inv.Role)
	}
	if inv.TokenHash == "" {
		t.Errorf("invite stored no token hash")
	}
	if !inv.InvitedBy.Valid || inv.InvitedBy.Int64 != adminID {
		t.Errorf("invite not attributed to the issuing admin: %+v", inv.InvitedBy)
	}
	// No account is created directly — the invitee accepts the link and chooses their
	// own credentials.
	if len(f.accounts) != 1 {
		t.Fatalf("invite created an account directly; accounts=%d", len(f.accounts))
	}
	// The join link is revealed once, and only its hash is stored (the plaintext
	// never appears in the store).
	if !strings.Contains(page, "/invite?token=") {
		t.Errorf("join link not revealed; body: %s", page)
	}
	if strings.Contains(page, inv.TokenHash) {
		t.Errorf("the stored hash leaked into the page")
	}
}

// An invite at an unknown role is refused, minting nothing.
func TestInviteRejectsUnknownRole(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/accounts", url.Values{"role": {"operator"}})
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "admin or viewer") {
		t.Fatalf("bad-role invite not refused: status=%d body=%s", resp.StatusCode, got)
	}
	if len(f.invites) != 0 {
		t.Fatalf("a rejected invite minted a row: %d", len(f.invites))
	}
}

// The change-role dialog's Save is disabled until the selected role differs from the
// current one (Settings.jsx): opened on a viewer, the Save button renders disabled
// and the select carries the current role as its baseline.
func TestChangeRoleSaveDisabledUntilDiffers(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	viewer := seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/settings?tab=team&role="+itoa(viewer.ID), http.StatusOK)
	if !strings.Contains(page, `id="rolesave"`) || !strings.Contains(page, "disabled>Save role") {
		t.Errorf("change-role Save not disabled by default; body: %s", page)
	}
	if !strings.Contains(page, `data-current="viewer"`) {
		t.Errorf("change-role select missing the current-role baseline; body: %s", page)
	}
}

// Require re-enrollment clears the member's second factor; the next sign-in re-enrols.
func TestRequireReenrollmentResetsTOTP(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	// Arm the member's second factor so the reset has something to clear.
	m := f.accounts[member.ID]
	m.TotpEnabled = true
	m.TotpSecret = pgtype.Text{String: "SECRET", Valid: true}
	f.accounts[member.ID] = m
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/accounts/reenroll", url.Values{"id": {itoa(member.ID)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("reenroll: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if got := f.accounts[member.ID]; got.TotpEnabled || got.TotpSecret.Valid {
		t.Fatalf("second factor not cleared: %+v", got)
	}
}

// Remove passes a typed-name gate, refuses self and the last admin, and removes on a
// correct confirmation.
func TestRemoveMemberTypedNameGate(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A wrong typed name re-opens the dialog with an error and removes nothing.
	resp := postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(member.ID)}, "confirm_name": {"wrong"},
	})
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "did not match") {
		t.Fatalf("typed-name mismatch not caught: status=%d body=%s", resp.StatusCode, got)
	}
	if _, ok := f.accounts[member.ID]; !ok {
		t.Fatalf("member removed despite a wrong confirmation")
	}

	// You cannot remove yourself.
	resp = postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(admin.ID)}, "confirm_name": {"admin"},
	})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body(t, resp), "your own account") {
		t.Fatalf("self-removal not refused")
	}

	// The exact username removes the member.
	resp = postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(member.ID)}, "confirm_name": {"member"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("remove: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if _, ok := f.accounts[member.ID]; ok {
		t.Fatalf("member not removed on a correct confirmation")
	}
}

// The two-role copy names admin and viewer only — never an operator role.
func TestTeamRolesCopyHasNoOperatorRole(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := settingsTabBody(t, ac, base, "team")
	for _, want := range []string{"admin", "viewer", "What each role can do"} {
		if !strings.Contains(page, want) {
			t.Errorf("roles card missing %q", want)
		}
	}
	// The invite dialog and roles card must never offer an "operator" role.
	invite := getBody(t, ac, base+"/settings?tab=team&invite=1", http.StatusOK)
	if strings.Contains(invite, ">operator<") || strings.Contains(invite, "value=\"operator\"") {
		t.Errorf("an operator role appeared in the Team surface")
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

	// An observation dial below the tightest bound in force is refused; the
	// previous value stands. With the dns Scan (daily) enabled the tightest bound
	// is k=2 daily cadences, so 1 day is below the floor. The floor is derived, not
	// an operator choice (#208, ADR-0094).
	resp = postForm(t, ac, base+"/settings/retention", url.Values{
		"observation_currency_days": {"1"}, "dispatch_cadence_multiple": {"4"},
	})
	got = body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "at least 2 days") {
		t.Fatalf("below-floor observation dial not refused: status=%d body=%s", resp.StatusCode, got)
	}
	if f.retention.ObservationCurrencyDays != 90 {
		t.Fatalf("rejected save mutated the observation dial: %+v", f.retention)
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
