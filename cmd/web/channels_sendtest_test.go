package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// --- Settings · Channels "Send test" (#757, R4-D2) ---------------------------
//
// A channel always holds its own URL, so it is always testable — no binding gate
// (that gate is only for integrations, #38/#39). These exercise the real send
// through the channelTestSender seam (never the network) and the honest toast on
// both the success and failure paths, plus the missing-channel and admin gates.

// A channel's "Send test" POSTs a real signed payload through the channel's own URL
// via the transport seam and toasts the ok success copy.
func TestChannelsSendTestCallsTransport(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 5, "https://ops.example/hook", "sign-me")
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"5"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("channel test send: status=%d, want 303", resp.StatusCode)
	}
	if sender.calls != 1 {
		t.Fatalf("channel test did not call the transport exactly once; calls=%d", sender.calls)
	}
	if sender.lastURL != "https://ops.example/hook" {
		t.Errorf("channel test target = %q, want the channel's own URL", sender.lastURL)
	}
	if len(sender.lastBody) == 0 || !strings.Contains(string(sender.lastBody), "\"headline\"") {
		t.Errorf("channel test posted no formatted body; got %q", string(sender.lastBody))
	}
	if string(sender.lastSecret) != "sign-me" {
		t.Errorf("channel test did not carry the channel's signing secret; got %q", string(sender.lastSecret))
	}
	toast := decodeToast(t, loc)
	if toast["tone"] != "ok" || toast["title"] != "Test message sent" {
		t.Errorf("channel test toast = %+v, want the ok/Test message sent copy", toast)
	}
}

// A failing transport (non-2xx or transport error) still attempts the send once and
// toasts an honest non-ok degrade, never the success copy.
func TestChannelsSendTestFailureToasts(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 5, "https://ops.example/hook", "")
	sender := &fakeChannelSender{status: http.StatusInternalServerError}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"5"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if sender.calls != 1 {
		t.Fatalf("failure path should still attempt the send once; calls=%d", sender.calls)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] == "ok" || toast["title"] == "Test message sent" {
		t.Errorf("a failed channel send toasted success copy: %+v", toast)
	}
}

// A missing or unknown channel id makes NO POST and flashes-and-redirects (no crash),
// mirroring the sibling channel handlers.
func TestChannelsSendTestUnknownIdDoesNotPost(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"999"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("unknown-channel test send: status=%d, want 303", resp.StatusCode)
	}
	if sender.calls != 0 {
		t.Fatalf("an unknown-channel Send test POSTed to the transport; calls=%d", sender.calls)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] == "ok" {
		t.Errorf("unknown-channel Send test toasted success: %+v", toast)
	}
}

// The Send test is an admin act: a viewer is refused and nothing is sent.
func TestChannelsSendTestAdminGated(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	addFakeChannel(f, 5, "https://ops.example/hook", "")
	sender := &fakeChannelSender{}
	base := startWithChannelSender(t, f, sender)
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/settings/channels/test", url.Values{"id": {"5"}})
	got := resp.StatusCode
	resp.Body.Close()
	if got != http.StatusForbidden {
		t.Errorf("viewer POST /settings/channels/test: status=%d, want 403", got)
	}
	if sender.calls != 0 {
		t.Errorf("a viewer triggered a channel send; calls=%d", sender.calls)
	}
}
