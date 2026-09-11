package main

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func prgLanding(t *testing.T, c *http.Client, base string, resp *http.Response) string {
	// A 303 carries no body, so an act's message is asserted on this landing GET (ADR-0130 §1).
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("mutating act: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		t.Fatal("mutating act: 303 carries no Location")
	}
	return getBody(t, c, base+loc, http.StatusOK)
}

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

	vc := login(t, base, "viewer", "hunter2hunter2")
	refused := getBody(t, vc, base+"/settings", http.StatusForbidden)
	for _, want := range []string{"Admin only", "declared acts live", "Back to dashboard"} {
		if !strings.Contains(refused, want) {
			t.Errorf("settings-forbidden page missing %q; body: %s", want, refused)
		}
	}

	resp, err := newClient(t).Get(base + "/settings")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon GET /settings: status=%d loc=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	ac := login(t, base, "admin", "hunter2hunter2")
	page := settingsBody(t, ac, base)
	for _, tab := range []string{"tab=scans", "tab=vantages", "tab=sso", "tab=team", "tab=audit", "tab=sources", "tab=aperture", "tab=instance", "tab=channels", "tab=messages", "tab=delivery"} {
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
	if !ch.secret.Valid || openTestChannelSecret(t, ch.secret) != "s3cr3t-signing-key" {
		t.Errorf("secret not stored: %+v", ch.secret)
	}

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
		{"https metadata literal", url.Values{"url": {"https://169.254.169.254/"}, "drift": {"on"}}, false, "internal address"},
		{"https rfc1918 literal", url.Values{"url": {"https://10.0.0.5/"}, "drift": {"on"}}, false, "internal address"},
		{"https loopback literal", url.Values{"url": {"https://127.0.0.1:9200/"}, "drift": {"on"}}, false, "internal address"},
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
			if got := refusalPage(t, ac, base, resp); !strings.Contains(got, tc.wantMsg) {
				t.Fatalf("landing page missing %q; body: %s", tc.wantMsg, got)
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

	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "enabled": {"on"},
	}).Body.Close()
	ch := f.channels[0]
	if ch.url != "https://b.example.com" || !ch.clock || ch.drift {
		t.Fatalf("update did not persist url/classes: %+v", ch)
	}
	if openTestChannelSecret(t, ch.secret) != "first" {
		t.Fatalf("blank secret should keep existing; got %q", ch.secret.String)
	}

	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "secret": {"second"},
	}).Body.Close()
	if openTestChannelSecret(t, f.channels[0].secret) != "second" {
		t.Fatalf("secret not replaced; got %q", f.channels[0].secret.String)
	}

	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {idStr}, "url": {"https://b.example.com"}, "clock": {"on"}, "clear_secret": {"on"}, "secret": {"ignored"},
	}).Body.Close()
	if f.channels[0].secret.Valid {
		t.Fatalf("clear box should null the secret; got valid=%v", f.channels[0].secret.Valid)
	}

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

	postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(viewer.ID)}, "role": {roleAdmin},
	}).Body.Close()
	if f.accounts[viewer.ID].Role != roleAdmin {
		t.Fatalf("role not promoted; got %q", f.accounts[viewer.ID].Role)
	}

	postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(viewer.ID)}, "role": {roleViewer},
	}).Body.Close()

	resp := postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(admin.ID)}, "role": {roleViewer},
	})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "last admin") {
		t.Fatalf("last-admin demotion not refused; landing body: %s", got)
	}
	if f.accounts[admin.ID].Role != roleAdmin {
		t.Fatalf("last admin was demoted despite the guard")
	}
}

func TestInviteMintsAgainstInviteTable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	adminID := f.byName["admin"]
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := prgLanding(t, ac, base, postForm(t, ac, base+"/settings/accounts", url.Values{"role": {roleViewer}}))

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
	if len(f.accounts) != 1 {
		t.Fatalf("invite created an account directly; accounts=%d", len(f.accounts))
	}
	if !strings.Contains(page, "/invite?token=") {
		t.Errorf("join link not revealed; body: %s", page)
	}
	if strings.Contains(page, inv.TokenHash) {
		t.Errorf("the stored hash leaked into the page")
	}
}

func TestInviteRejectsUnknownRole(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/accounts", url.Values{"role": {"operator"}})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "admin or viewer") {
		t.Fatalf("bad-role invite not refused; landing body: %s", got)
	}
	if len(f.invites) != 0 {
		t.Fatalf("a rejected invite minted a row: %d", len(f.invites))
	}
}

func TestChangeRoleSaveDisabledUntilDiffers(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	viewer := seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/settings?tab=team&role="+itoa(viewer.ID), http.StatusOK)
	if !strings.Contains(page, `id="st-role-save"`) || !strings.Contains(page, "disabled>Save role") {
		t.Errorf("change-role Save not disabled by default; body: %s", page)
	}
	if !strings.Contains(page, `data-current="viewer"`) {
		t.Errorf("change-role select missing the current-role baseline; body: %s", page)
	}
}

func TestRequireReenrollmentResetsTOTP(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
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

func TestRemoveMemberTypedNameGate(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(member.ID)}, "confirm_name": {"wrong"},
	})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "did not match") {
		t.Fatalf("typed-name mismatch not caught; landing body: %s", got)
	}
	if _, ok := f.accounts[member.ID]; !ok {
		t.Fatalf("member removed despite a wrong confirmation")
	}

	resp = postForm(t, ac, base+"/settings/accounts/remove", url.Values{
		"id": {itoa(admin.ID)}, "confirm_name": {"admin"},
	})
	if got := refusalPage(t, ac, base, resp); !strings.Contains(got, "your own account") {
		t.Fatalf("self-removal not refused; landing body: %s", got)
	}

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
	invite := getBody(t, ac, base+"/settings?tab=team&invite=1", http.StatusOK)
	if strings.Contains(invite, ">operator<") || strings.Contains(invite, "value=\"operator\"") {
		t.Errorf("an operator role appeared in the Team surface")
	}
}

// The observation and dispatch dials moved to Coverage, and retentionpanel_test.go
// asserts their contract. Settings hosts the transcript dial alone.

func TestTranscriptRetentionPersistsAndValidates(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.retention = db.GetRetentionSettingsRow{ObservationCurrencyDays: 90, DispatchCadenceMultiple: 4}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/retention", url.Values{
		"transcript_currency_days": {"30"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("retention save: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.retention.TranscriptCurrencyDays != 30 {
		t.Fatalf("transcript dial not persisted: %+v", f.retention)
	}
	// This form no longer carries the other two dials, so it may not overwrite them.
	if f.retention.ObservationCurrencyDays != 90 || f.retention.DispatchCadenceMultiple != 4 {
		t.Fatalf("the Coverage dials were overwritten by the Settings form: %+v", f.retention)
	}
	if !f.retention.UpdatedBy.Valid {
		t.Errorf("updated_by not attributed")
	}

	const deliveryTab = "/settings?tab=delivery"
	refuse := func(what string, vals url.Values) string {
		t.Helper()
		if loc := submitLoc(t, postForm(t, ac, base+"/settings/retention", vals)); loc != deliveryTab {
			t.Fatalf("refused %s landed at %q, want %q", what, loc, deliveryTab)
		}
		return getBody(t, ac, base+deliveryTab, http.StatusOK)
	}

	got := refuse("negative transcript dial", url.Values{"transcript_currency_days": {"-1"}})
	if !strings.Contains(got, "zero or more") {
		t.Fatalf("negative transcript dial not refused; body: %s", got)
	}
	if f.retention.TranscriptCurrencyDays != 30 {
		t.Fatalf("rejected save mutated the dial: %+v", f.retention)
	}

	got = refuse("non-numeric transcript dial", url.Values{"transcript_currency_days": {"soon"}})
	if !strings.Contains(got, "zero or more") {
		t.Fatalf("non-numeric transcript dial not refused; body: %s", got)
	}
	if !strings.Contains(got, `value="soon"`) {
		t.Errorf("rejected transcript value not echoed back: %s", got)
	}
	if f.retention.TranscriptCurrencyDays != 30 {
		t.Fatalf("rejected save mutated the transcript dial: %+v", f.retention)
	}

	resp = postForm(t, ac, base+"/settings/retention", url.Values{"transcript_currency_days": {"0"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("unbounded (0) dial refused: status=%d (%s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()
	if f.retention.TranscriptCurrencyDays != 0 {
		t.Fatalf("unbounded dial not persisted: %+v", f.retention)
	}
}

func (f *fakeStore) GetInstanceHealth(context.Context) (db.GetInstanceHealthRow, error) {
	return f.instanceHealth, nil
}

func (f *fakeStore) ListVergeCoreFrequencyEditsWithAuthor(context.Context) ([]db.ListVergeCoreFrequencyEditsWithAuthorRow, error) {
	ports := make([]int, 0, len(f.freqEdits))
	for p := range f.freqEdits {
		ports = append(ports, int(p))
	}
	sort.Ints(ports)
	out := make([]db.ListVergeCoreFrequencyEditsWithAuthorRow, 0, len(ports))
	for i, p := range ports {
		e := f.freqEdits[int32(p)]
		out = append(out, db.ListVergeCoreFrequencyEditsWithAuthorRow{
			ID: int64(i + 1), Port: int32(p), Action: e.action, CreatedByUsername: "admin",
		})
	}
	return out, nil
}

func (f *fakeStore) TightestEnabledScanCadenceSeconds(context.Context) (int64, error) {
	var tightest int64
	for _, sc := range f.scans {
		if sc.Enabled && (tightest == 0 || sc.CadenceSeconds < tightest) {
			tightest = sc.CadenceSeconds
		}
	}
	return tightest, nil
}

func (f *fakeStore) DeleteAccount(_ context.Context, id int64) error {
	if _, ok := f.accounts[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(f.accounts, id)
	for name, nid := range f.byName {
		if nid == id {
			delete(f.byName, name)
		}
	}
	return nil
}

func (f *fakeStore) ResetAccountTOTP(_ context.Context, id int64) (db.ResetAccountTOTPRow, error) {
	acct, ok := f.accounts[id]
	if !ok {
		return db.ResetAccountTOTPRow{}, pgx.ErrNoRows
	}
	before := db.ResetAccountTOTPRow{Username: acct.Username, TotpEnabled: acct.TotpEnabled}
	acct.TotpSecret = pgtype.Text{}
	acct.TotpEnabled = false
	f.accounts[id] = acct
	return before, nil
}

func (f *fakeStore) CountAdmins(context.Context) (int64, error) {
	var n int64
	for _, a := range f.accounts {
		if a.Role == roleAdmin {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) UpdateAccountRole(_ context.Context, arg db.UpdateAccountRoleParams) error {
	a, ok := f.accounts[arg.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	a.Role = arg.Role
	f.accounts[arg.ID] = a
	return nil
}

func (f *fakeStore) CreateChannel(_ context.Context, arg db.CreateChannelParams) (int64, error) {
	c := fakeChannel{
		id: f.chanNextID, url: arg.Url, secret: arg.Secret,
		drift: arg.RouteDrift, coverage: arg.RouteCoverage, clock: arg.RouteClock,
		enabled: arg.Enabled, createdBy: arg.CreatedBy,
		createdAt: time.Now(), updatedAt: time.Now(),
	}
	f.channels = append(f.channels, c)
	f.chanNextID++
	return c.id, nil
}

func (f *fakeStore) UpdateChannel(_ context.Context, arg db.UpdateChannelParams) error {
	for i := range f.channels {
		if f.channels[i].id == arg.ID {
			f.channels[i].url = arg.Url
			f.channels[i].drift = arg.RouteDrift
			f.channels[i].coverage = arg.RouteCoverage
			f.channels[i].clock = arg.RouteClock
			f.channels[i].enabled = arg.Enabled
			f.channels[i].updatedAt = time.Now()
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (f *fakeStore) SetChannelSecret(_ context.Context, arg db.SetChannelSecretParams) error {
	for i := range f.channels {
		if f.channels[i].id == arg.ID {
			f.channels[i].secret = arg.Secret
			f.channels[i].updatedAt = time.Now()
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (f *fakeStore) DeleteChannel(_ context.Context, id int64) error {
	for i, c := range f.channels {
		if c.id == id {
			f.channels = append(f.channels[:i], f.channels[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeStore) GetRetentionSettings(context.Context) (db.GetRetentionSettingsRow, error) {
	if f.retentionErr != nil {
		return db.GetRetentionSettingsRow{}, f.retentionErr
	}
	return f.retention, nil
}

func (f *fakeStore) SetUpdateCheckEnabled(_ context.Context, arg db.SetUpdateCheckEnabledParams) error {
	f.instanceConfig.UpdateCheckEnabled = arg.UpdateCheckEnabled
	f.instanceConfig.UpdateCheckUpdatedBy = arg.UpdateCheckUpdatedBy
	f.instanceConfig.UpdateCheckUpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return nil
}

func (f *fakeStore) SetAPIEnabled(_ context.Context, arg db.SetAPIEnabledParams) error {
	f.instanceConfig.ApiEnabled = arg.ApiEnabled
	f.instanceConfig.ApiUpdatedBy = arg.ApiUpdatedBy
	f.instanceConfig.ApiUpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return nil
}

func (f *fakeStore) SetSeedAddressCap(_ context.Context, arg db.SetSeedAddressCapParams) error {
	f.instanceConfig.SeedAddressCap = arg.SeedAddressCap
	f.instanceConfig.SeedAddressCapUpdatedBy = arg.SeedAddressCapUpdatedBy
	f.instanceConfig.SeedAddressCapUpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return nil
}

func (f *fakeStore) UpdateRetentionSettings(_ context.Context, arg db.UpdateRetentionSettingsParams) error {
	f.retention.ObservationCurrencyDays = arg.ObservationCurrencyDays
	f.retention.DispatchCadenceMultiple = arg.DispatchCadenceMultiple
	f.retention.TranscriptCurrencyDays = arg.TranscriptCurrencyDays
	f.retention.UpdatedBy = arg.UpdatedBy
	f.retention.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return nil
}

func TestViewerIsRefusedEverySettingsTabButAPI(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	for _, tab := range settingsTabs {
		want := http.StatusForbidden
		if tab == "api" {
			want = http.StatusOK
		}
		resp, err := vc.Get(base + "/settings?tab=" + tab)
		if err != nil {
			t.Fatal(err)
		}
		got := body(t, resp)
		if resp.StatusCode != want {
			t.Errorf("viewer GET /settings?tab=%s: status = %d, want %d (body: %s)", tab, resp.StatusCode, want, got)
		}
	}
}
