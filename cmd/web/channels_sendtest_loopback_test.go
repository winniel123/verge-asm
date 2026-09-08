package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestChannelsSendTestDeliversToLoopbackHTTP(t *testing.T) {
	var got []byte
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer receiver.Close()

	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 5, receiver.URL+"/hook", "sign-me")
	base := startWithChannelSender(t, f, newHTTPChannelSender(fixedClock()))
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"5"}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("channel test send: status=%d, want 303", resp.StatusCode)
	}
	if !strings.Contains(string(got), "\"headline\"") {
		t.Fatalf("loopback receiver got no formatted body; got %q", got)
	}
	toast := decodeToast(t, loc)
	if toast["tone"] != "ok" || toast["title"] != "Test message sent" {
		t.Errorf("loopback test send toast = %+v, want ok/Test message sent", toast)
	}
}
