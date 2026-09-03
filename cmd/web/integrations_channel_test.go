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

// --- Integrations channel binding + real Send test (#39b, P0.14) -------------
//
// These exercise the delivery-Channel binding the ruling added to the Integrations
// drawer: a "Delivery channel" select that binds an installed integration to a Channel
// (a reference, not a fold), and a "Send test" that POSTs a real payload through the
// bound Channel's transport. They run only when the surface is live (integrationsEnabled).

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

// startWithChannelSender starts a server whose Send-test egress is the given fake, so a
// test asserts the send through the seam rather than the live network.
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

// Binding an installed integration to a Channel persists the reference and renders it
// selected in the drawer, and gates "Send test" from disabled to a real POST form.
func TestIntegrationsChannelBindPersistsAndShowsInDrawer(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 1, "https://ops.acmecorp.io/hook", "sk-secret")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Install so there is an installed row to bind.
	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"slack"}}).Body.Close()

	// Before binding, the drawer offers the "Delivery channel" select with a leading
	// "Not connected" option and the declared channel, and gates "Send test" off.
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

	// The drawer now renders the channel selected and offers the real Send-test form.
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

// A bound integration's "Send test" POSTs a real payload through the bound Channel's
// transport (via the seam, never the network) and toasts the spec's exact copy.
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

// A failing transport (non-2xx or transport error) toasts an honest non-ok degrade,
// never the success copy.
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

// An unbound integration's "Send test" makes NO POST (the template already disables the
// button; the handler defends it) and toasts a "connect a channel" warn.
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

// Clearing the binding (empty channel) unbinds, and an unknown channel id is refused
// rather than stored as a dangling reference.
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

	// An unknown channel id is refused, and no dangling reference is written.
	//
	// An operator CAN reach this: the select's options are the Channels that existed when
	// the drawer rendered, so another admin deleting one between the render and the submit
	// lands here. It is a refusal, so since ticket #978 it answers the way every refusal on
	// this surface answers (ADR-0130 §1) — a 303 back to the drawer with a toast — rather
	// than a 400 text body at the POST URL that loses the drawer, the tab and the offset.
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

// The channel-bind and test acts are admin acts: a viewer is refused and nothing is
// written or sent.
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
