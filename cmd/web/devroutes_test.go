package main

import (
	"net/http"
	"testing"
)

func TestDevRoutesAreUnroutedWithoutVergeDev(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, path := range []string{
		"/dev/403",
		"/dev/panic",
		"/dev/session/admin",
		"/dev/profile/session",
		"/dev/seed/empty",
		"/dev/seed/empty-authed",
	} {
		resp, err := ac.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		got := body(t, resp)
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s with VERGE_DEV off: status = %d, want 404 (body: %s)", path, resp.StatusCode, got)
		}
	}
}
