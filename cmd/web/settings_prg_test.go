package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func submitLoc(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("mutating act: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	return resp.Header.Get("Location")
}

func TestSettingsFormsCarryTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	addFakeChannel(f, 5, "https://ops.example/hook", "")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	// The cold-tier region renders an opt-in form only against a declared scope.
	declare(t, ac, base, "name", "example.com").Body.Close()

	// Every Team form lives in a query-opened dialog, so the plain tab carries no form to stamp.
	for _, tc := range []struct {
		name string
		at   string
		want string
	}{
		{"channels", "/settings?tab=channels", `name="return" value="/settings?tab=channels"`},
		{"scans", "/settings?tab=scans", `name="return" value="/settings?tab=scans"`},
		{"role dialog", "/settings?tab=team&role=" + itoa(member.ID),
			`name="return" value="/settings?tab=team&amp;role=` + itoa(member.ID) + `"`},
		{"reenroll dialog", "/settings?tab=team&reenroll=" + itoa(member.ID),
			`name="return" value="/settings?tab=team&amp;reenroll=` + itoa(member.ID) + `"`},
		{"remove dialog", "/settings?tab=team&remove=" + itoa(member.ID),
			`name="return" value="/settings?tab=team&amp;remove=` + itoa(member.ID) + `"`},
		{"invite dialog", "/settings?tab=team&invite=1",
			`name="return" value="/settings?tab=team&amp;invite=1"`},
		{"folded scans", "/scans", `name="return" value="/scans"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := getBody(t, ac, base+tc.at, http.StatusOK)
			if !strings.Contains(page, tc.want) {
				t.Fatalf("%s carries no submitting-URL field %s; body: %s", tc.at, tc.want, page)
			}
		})
	}
}

func TestRefusedRoleChangeLandsBackOnTheTeamTabWithItsMessageVisible(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const tab = "/settings?tab=team"
	resp := postForm(t, ac, base+"/settings/accounts/role", url.Values{
		"id": {itoa(admin.ID)}, "role": {roleViewer},
		"return": {tab + "&role=" + itoa(admin.ID)},
	})
	if loc := submitLoc(t, resp); loc != tab {
		t.Fatalf("refused role change landed at %q, want %q with the dialog closed", loc, tab)
	}

	page := getBody(t, ac, base+tab, http.StatusOK)
	if !strings.Contains(page, "last admin") {
		t.Fatalf("the guard's message is not on the landing page; body: %s", page)
	}
	if strings.Contains(page, `class="st-scrim"`) {
		t.Fatalf("the landing re-opened a dialog over the message; body: %s", page)
	}
	if again := getBody(t, ac, base+tab, http.StatusOK); strings.Contains(again, "last admin") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

func TestSucceedingTeamActClosesItsDialog(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const tab = "/settings?tab=team"
	for _, tc := range []struct {
		name string
		at   string
		vals url.Values
	}{
		{"role change", "/settings/accounts/role", url.Values{
			"id": {itoa(member.ID)}, "role": {roleAdmin},
			"return": {tab + "&role=" + itoa(member.ID)},
		}},
		{"re-enrollment", "/settings/accounts/reenroll", url.Values{
			"id": {itoa(member.ID)}, "return": {tab + "&reenroll=" + itoa(member.ID)},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if loc := submitLoc(t, postForm(t, ac, base+tc.at, tc.vals)); loc != tab {
				t.Fatalf("landed at %q, want %q with the dialog closed", loc, tab)
			}
			if page := getBody(t, ac, base+tab, http.StatusOK); strings.Contains(page, `class="st-scrim"`) {
				t.Fatalf("the completed act left its dialog open; body: %s", page)
			}
		})
	}
}

func TestASettingsFlashIsOnlyConsumedByItsOwnTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/settings/channels", url.Values{
		"url": {"http://example.com/h"}, "drift": {"on"}, "return": {"/settings?tab=channels"},
	})
	resp.Body.Close()
	for _, other := range []string{"/scans", "/settings?tab=team", "/settings?tab=scans"} {
		if page := getBody(t, ac, base+other, http.StatusOK); strings.Contains(page, "loopback") {
			t.Fatalf("%s rendered another tab's callout; body: %s", other, page)
		}
	}
	page := getBody(t, ac, base+"/settings?tab=channels", http.StatusOK)
	if !strings.Contains(page, "loopback") {
		t.Fatalf("the operator's own landing lost the callout to another tab; body: %s", page)
	}
}

func TestRefusedChannelCreateEchoesTypedValueOnItsOwnTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=channels"
	resp := postForm(t, ac, base+"/settings/channels", url.Values{
		"url": {"http://example.com/h"}, "drift": {"on"}, "return": {from},
	})
	loc := submitLoc(t, resp)
	if loc != from {
		t.Fatalf("refused channel create landed at %q, want %q", loc, from)
	}
	if strings.Contains(loc, "example.com") {
		t.Fatalf("the typed value leaked into the URL: %q", loc)
	}

	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "loopback") {
		t.Fatalf("the refusal is not on the landing page; body: %s", page)
	}
	if !strings.Contains(page, `value="http://example.com/h"`) {
		t.Fatalf("the typed URL was not echoed back; body: %s", page)
	}
	if len(f.channels) != 0 {
		t.Fatalf("a refused create stored a channel; got %d", len(f.channels))
	}
}

func TestColdOptInComesBackToTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	resp := postForm(t, ac, base+"/settings/cold", url.Values{
		"id": {itoa(id)}, "opt_in": {"true"}, "return": {"/scans"},
	})
	if loc := submitLoc(t, resp); loc != "/scans" {
		t.Fatalf("cold opt-in landed at %q, want /scans", loc)
	}
	if !f.coldScopes[id] {
		t.Fatalf("the opt-in did not persist")
	}

	resp = postForm(t, ac, base+"/settings/cold", url.Values{
		"id": {"not-a-number"}, "opt_in": {"true"}, "return": {"/scans"},
	})
	if loc := submitLoc(t, resp); loc != "/scans" {
		t.Fatalf("refused cold opt-in landed at %q, want /scans", loc)
	}
	if page := getBody(t, ac, base+"/scans", http.StatusOK); !strings.Contains(page, "could not be found") {
		t.Fatalf("the refusal is not on the /scans landing; body: %s", page)
	}
}

func TestChannelSendTestLandsBackOnTheSubmittingURLWithItsToast(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 5, "https://ops.example/hook", "")
	sender := &fakeChannelSender{status: http.StatusOK}
	base := startWithChannelSender(t, f, sender)
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=channels"
	loc := submitLoc(t, postForm(t, ac, base+"/settings/channels/test", url.Values{
		"id": {"5"}, "return": {from},
	}))
	if !strings.HasPrefix(loc, from+"&toast=") {
		t.Fatalf("send test landed at %q, want %q with a toast appended", loc, from)
	}
	if toast := decodeToast(t, loc); toast["title"] != "Test message sent" {
		t.Errorf("send test toast = %+v, want the ok copy", toast)
	}
}

func TestForgedSubmittingURLFallsBackToTheTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The guard chooses the destination, never whether the operator's declared act runs.
	for _, forged := range []string{
		"https://evil.example/x",
		"//evil.example/x",
		`/\evil.example`,
		"/settings/../admin",
		"/not-a-route-this-server-serves",
	} {
		resp := postForm(t, ac, base+"/settings/accounts/reenroll", url.Values{
			"id": {itoa(member.ID)}, "return": {forged},
		})
		if loc := submitLoc(t, resp); loc != "/settings?tab=team" {
			t.Errorf("forged return %q landed at %q, want the /settings?tab=team fallback", forged, loc)
		}
	}
}
