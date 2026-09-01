package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The ADR-0130 contract on the Settings team, channels and scans tabs (map #969,
// ticket #974). Every mutating act on those tabs is a post-redirect-get back to the
// exact URL its form was submitted from, and a refusal carries its callout to that
// landing GET through the session form flash rather than rendering at the POST URL.
//
// Every test here runs with no JavaScript at all — Go's HTTP client executes none —
// so each one is also the progressive-enhancement check the ticket asks for: the act
// works, and its error is shown, on plain markup alone.

// submitLoc asserts a mutating act answered 303 and returns its Location.
func submitLoc(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("mutating act: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	return resp.Header.Get("Location")
}

// TestSettingsFormsCarryTheSubmittingURL is the emit half of ADR-0130 §3: each
// migrated form on these tabs stamps the page's own path AND query into the hidden
// `return` field, so the handler has the operator's exact list to come back to.
//
// The query matters more than the path. A bare `/settings` would drop the tab, which is
// the class-E failure the ticket exists to close. The field carries the page's whole
// query, dialog parameter included; what a handler does with each pair is the redirect's
// business, not the form's (see dialogParams).
func TestSettingsFormsCarryTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	addFakeChannel(f, 5, "https://ops.example/hook", "")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	// The cold-tier region renders an opt-in form only against a declared scope.
	declare(t, ac, base, "name", "example.com").Body.Close()

	// The Team tab's own forms all live inside a dialog, and every dialog is opened by
	// a query parameter (fillTeamSection) rather than by a menu click — so the plain
	// tab carries no form to stamp, and each dialog URL is exercised on its own below.
	for _, tc := range []struct {
		name string
		at   string
		want string
	}{
		{"channels", "/settings?tab=channels", `name="return" value="/settings?tab=channels"`},
		{"scans", "/settings?tab=scans", `name="return" value="/settings?tab=scans"`},
		// A dialog rides its own query parameter, so the field carries it too — verbatim,
		// like every other pair. backToSection is what decides to drop it again.
		{"role dialog", "/settings?tab=team&role=" + itoa(member.ID),
			`name="return" value="/settings?tab=team&amp;role=` + itoa(member.ID) + `"`},
		{"reenroll dialog", "/settings?tab=team&reenroll=" + itoa(member.ID),
			`name="return" value="/settings?tab=team&amp;reenroll=` + itoa(member.ID) + `"`},
		{"remove dialog", "/settings?tab=team&remove=" + itoa(member.ID),
			`name="return" value="/settings?tab=team&amp;remove=` + itoa(member.ID) + `"`},
		{"invite dialog", "/settings?tab=team&invite=1",
			`name="return" value="/settings?tab=team&amp;invite=1"`},
		// The folded read surface renders the same Scans section, so its cold-tier
		// opt-in names /scans as the URL to come back to, not /settings.
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

// A refused role change is a 303 back to the operator's own tab, and the guard's
// message renders on the landing GET at 200 — not as a 400 body at the POST URL.
// This is failure classes A and E closed together: the load is an ordinary navigation,
// so the scroll offset the shell stashed on submit is restored.
//
// The dialog parameter is dropped from the destination (dialogParams). It has to be:
// the role callout renders at page level and .st-scrim covers the page, so landing back
// on ?role=<id> would show the operator a dimmed page with the message hidden behind
// the modal.
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
	// Nothing the act carried entered the URL, and the flash is single-consume, so a
	// reload of the same URL shows the operator a clean page rather than a stale one.
	if again := getBody(t, ac, base+tab, http.StatusOK); strings.Contains(again, "last admin") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

// A SUCCEEDING team act closes the dialog it was submitted from. Returning to the
// dialog's own query parameter would re-open the confirm the operator just accepted,
// which reads as "nothing happened" and offers the act a second time.
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

// The settings surface has more than one landing GET, and a stash belongs to exactly
// one of them. A GET on another tab must leave it alone — and this is not a rare race:
// /scans re-requests itself every six seconds while a scan is in flight, so an
// unclaimed take there would eat the session's every refusal for the length of the scan.
func TestASettingsFlashIsOnlyConsumedByItsOwnTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Refuse a channel create, then load every other landing before the operator's own.
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

// A refused channel create lands back on the Channels tab with the operator's typed
// URL still in the input. The echo rides the session flash, so it is on the landing
// page and not in the query.
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

// The cold-tier opt-in renders on the folded /scans surface as well as on
// /settings?tab=scans, so both are legitimate submitting URLs. Both the success and
// the refusal must come back to the one the operator actually acted from.
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

	// A refusal is a 303 to the same place, and /scans reads the flash so the callout
	// renders there rather than being dropped on the floor.
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

// The channel Send test lands back on the submitting URL and still fires its toast
// there. The two carriers compose: the submitting URL owns the destination, the
// `toast` query owns the receipt.
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

// A submitting URL the operator forged is refused and the act falls back to its own
// tab, so the carrier can never become an open redirect (backurl.go resolveBack).
// The act itself still happens — the guard chooses the destination, never whether the
// operator's declared act runs.
func TestForgedSubmittingURLFallsBackToTheTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	member := seedAccount(t, f, "member", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

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
