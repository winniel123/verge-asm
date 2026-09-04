package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

type fakeChannelSender struct {
	calls      int
	lastURL    string
	lastBody   []byte
	lastSecret []byte
	status     int
	err        error
}

func (f *fakeChannelSender) Send(_ context.Context, targetURL string, body, secret []byte) (int, error) {
	f.calls++
	f.lastURL = targetURL
	f.lastBody = append([]byte(nil), body...)
	f.lastSecret = append([]byte(nil), secret...)
	if f.err != nil {
		return 0, f.err
	}
	st := f.status
	if st == 0 {
		st = http.StatusOK
	}
	return st, nil
}

func startWithChannelSender(t *testing.T, f *fakeStore, sender channelTestSender) string {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	srv.channelSender = sender
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

func addFakeChannel(f *fakeStore, id int64, rawURL, secret string) {
	c := fakeChannel{
		id: id, url: rawURL, drift: true, coverage: true, clock: true, enabled: true,
		createdBy: 1, createdAt: time.Now(), updatedAt: time.Now(),
	}
	if secret != "" {
		c.secret = pgtype.Text{String: secret, Valid: true}
	}
	f.channels = append(f.channels, c)
	if id >= f.chanNextID {
		f.chanNextID = id + 1
	}
}

func TestIntegrationsChannelBindPersistsAndShowsInDrawer(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 1, "https://ops.acmecorp.io/hook", "sk-secret")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"slack"}}).Body.Close()

	before := integrationsBody(t, ac, base, "&view=slack")
	for _, want := range []string{
		`action="/settings/integrations/channel"`, "Delivery channel",
		`value=""`, "Not connected", `value="1"`, "ops.acmecorp.io/hook",
		"Connect a channel to test",
	} {
		if !strings.Contains(before, want) {
			t.Errorf("unbound drawer missing %q; body: %s", want, before)
		}
	}

	resp := postForm(t, ac, base+"/settings/integrations/channel", url.Values{"id": {"slack"}, "channel": {"1"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != "/settings?tab=integrations&view=slack" {
		t.Fatalf("bind: status=%d loc=%q, want 303 -> the drawer", resp.StatusCode, loc)
	}
	if st := f.integrationStates["slack"]; !st.ChannelID.Valid || st.ChannelID.Int64 != 1 {
		t.Fatalf("binding not persisted: %+v", f.integrationStates["slack"])
	}

	after := integrationsBody(t, ac, base, "&view=slack")
	if !strings.Contains(after, `value="1" selected`) {
		t.Errorf("bound channel not rendered selected; body: %s", after)
	}
	if !strings.Contains(after, `action="/settings/integrations/test"`) {
		t.Errorf("bound drawer missing the enabled Send-test form; body: %s", after)
	}
	if strings.Contains(after, "Connect a channel to test") {
		t.Errorf("bound drawer still shows the disabled Send-test hint; body: %s", after)
	}
}

func TestIntegrationsTestSendBoundCallsTransport(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 7, "https://pager.example/verge", "sign-me")
	f.integrationStates["pagerduty"] = db.IntegrationState{
		Slug: "pagerduty", State: integrationInstalled,
		ChannelID: pgtype.Int8{Int64: 7, Valid: true},
	}
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/integrations/test", url.Values{"id": {"pagerduty"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("test send: status=%d, want 303", resp.StatusCode)
	}
	if sender.calls != 1 {
		t.Fatalf("test send did not call the transport exactly once; calls=%d", sender.calls)
	}
	if sender.lastURL != "https://pager.example/verge" {
		t.Errorf("test send target = %q, want the bound channel's URL", sender.lastURL)
	}
	if len(sender.lastBody) == 0 || !strings.Contains(string(sender.lastBody), "\"headline\"") {
		t.Errorf("test send posted no formatted body; got %q", string(sender.lastBody))
	}
	if string(sender.lastSecret) != "sign-me" {
		t.Errorf("test send did not carry the channel's signing secret; got %q", string(sender.lastSecret))
	}
	toast := decodeToast(t, loc)
	if toast["tone"] != "ok" || toast["title"] != "Test message sent" ||
		toast["description"] != "Check PagerDuty for the delivery." {
		t.Errorf("test send toast = %+v, want the spec ok/Test message sent copy", toast)
	}
}

func TestIntegrationsTestSendBoundFailureToasts(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 7, "https://pager.example/verge", "")
	f.integrationStates["pagerduty"] = db.IntegrationState{
		Slug: "pagerduty", State: integrationInstalled,
		ChannelID: pgtype.Int8{Int64: 7, Valid: true},
	}
	sender := &fakeChannelSender{status: http.StatusInternalServerError}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/integrations/test", url.Values{"id": {"pagerduty"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if sender.calls != 1 {
		t.Fatalf("failure path should still attempt the send once; calls=%d", sender.calls)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] == "ok" || toast["title"] == "Test message sent" {
		t.Errorf("a failed send toasted success copy: %+v", toast)
	}
}

func TestIntegrationsTestSendUnboundDoesNotPost(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.integrationStates["slack"] = db.IntegrationState{Slug: "slack", State: integrationInstalled}
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/integrations/test", url.Values{"id": {"slack"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("unbound test send: status=%d, want 303", resp.StatusCode)
	}
	if sender.calls != 0 {
		t.Fatalf("an unbound Send-test POSTed to the transport; calls=%d", sender.calls)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] == "ok" {
		t.Errorf("unbound Send-test toasted success: %+v", toast)
	}
}

func TestIntegrationsChannelUnbindAndUnknownRefused(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 1, "https://ops.acmecorp.io/hook", "")
	f.integrationStates["slack"] = db.IntegrationState{
		Slug: "slack", State: integrationInstalled,
		ChannelID: pgtype.Int8{Int64: 1, Valid: true},
	}
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/integrations/channel", url.Values{"id": {"slack"}, "channel": {""}})
	resp.Body.Close()
	if st := f.integrationStates["slack"]; st.ChannelID.Valid {
		t.Fatalf("empty channel did not unbind: %+v", st)
	}

	const drawer = "/settings?tab=integrations&view=slack"
	resp = postForm(t, ac, base+"/settings/integrations/channel", url.Values{
		"id": {"slack"}, "channel": {"999"}, "return": {drawer},
	})
	loc := submitLoc(t, resp)
	if !strings.HasPrefix(loc, drawer) {
		t.Errorf("refused bind landed at %q, want the drawer %q", loc, drawer)
	}
	if !strings.Contains(loc, "toast=") {
		t.Errorf("refused bind carried no toast: %q", loc)
	}
	if st := f.integrationStates["slack"]; st.ChannelID.Valid {
		t.Fatalf("a refused bind wrote a dangling reference: %+v", st)
	}
}

func TestIntegrationsChannelActsAdminGated(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	addFakeChannel(f, 1, "https://ops.acmecorp.io/hook", "")
	f.integrationStates["slack"] = db.IntegrationState{Slug: "slack", State: integrationInstalled}
	sender := &fakeChannelSender{}
	base := startWithChannelSender(t, f, sender)
	vc := login(t, base, "viewer", "hunter2hunter2")

	for _, path := range []string{"/settings/integrations/channel", "/settings/integrations/test"} {
		resp := postForm(t, vc, base+path, url.Values{"id": {"slack"}, "channel": {"1"}})
		got := resp.StatusCode
		resp.Body.Close()
		if got != http.StatusForbidden {
			t.Errorf("viewer POST %s: status=%d, want 403", path, got)
		}
	}
	if st := f.integrationStates["slack"]; st.ChannelID.Valid {
		t.Errorf("a viewer bound a channel: %+v", st)
	}
	if sender.calls != 0 {
		t.Errorf("a viewer triggered a send; calls=%d", sender.calls)
	}
}
