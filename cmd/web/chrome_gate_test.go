package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

// The goldens crop to <main>, so nothing else covers the chrome band (#1358).

const chromeBandMark = `class="sh-topnav"`

func TestSignInPageRendersWithoutChrome(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")

	resp, err := newClient(t).Get(base + "/login")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /login status = %d, want 200 (body: %s)", resp.StatusCode, got)
	}
	if strings.Contains(got, chromeBandMark) {
		t.Errorf("the sign-in page carries the console chrome; body: %s", got)
	}
}

func TestSignedOutErrorPageRendersWithoutChrome(t *testing.T) {
	s := newServer(newFakeStore(), testKey, "", fixedClock())
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("kaboom") })
	ts := httptest.NewServer(s.recoverPanics(boom))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/anything")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("signed-out 500 status = %d, want 500 (body: %s)", resp.StatusCode, got)
	}
	if strings.Contains(got, chromeBandMark) {
		t.Errorf("a signed-out error page carries the console chrome; body: %s", got)
	}
}

func TestSignedInErrorPageKeepsChrome(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	c := login(t, base, "admin", "hunter2hunter2")

	resp, err := c.Get(base + "/no-such-screen")
	if err != nil {
		t.Fatal(err)
	}
	got := body(t, resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("signed-in 404 status = %d, want 404 (body: %s)", resp.StatusCode, got)
	}
	if !strings.Contains(got, chromeBandMark) {
		t.Errorf("a signed-in error page lost the console chrome; body: %s", got)
	}
}

func TestPageDataMarksTheShellAndBarePageDataDoesNot(t *testing.T) {
	if _, ok := barePageData("Sign in")[shellKey]; ok {
		t.Errorf("barePageData set the shell marker")
	}
	if _, ok := pageData(db.Account{Role: roleAdmin}, "Dashboard", "dashboard")[shellKey]; !ok {
		t.Errorf("pageData set no shell marker")
	}
}
