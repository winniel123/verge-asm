package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The ADR-0130 contract on the Settings SSO, sources and aperture tabs (map #969,
// ticket #975). It is the same contract settings_prg_test.go asserts for the team,
// channels and scans tabs, and it uses that file's submitLoc helper.
//
// Two of these tabs have a FOLDED read surface as well: /sources renders the same
// section /settings?tab=sources does, and /verge-core the same one
// /settings?tab=aperture does. Each folded page is therefore a legitimate submitting
// URL and a legitimate landing, and both are exercised here.
//
// Every test runs with no JavaScript at all — Go's HTTP client executes none — so each
// one is also the progressive-enhancement check the ticket asks for.

// TestSSOSourcesAndAperturePagesCarryTheSubmittingURL is the emit half of ADR-0130 §3:
// each of the five pages stamps its OWN path into the hidden field, so a folded surface
// never sends the operator to the settings one.
func TestSSOSourcesAndAperturePagesCarryTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, at := range []string{
		"/settings?tab=sso", "/settings?tab=sources", "/settings?tab=aperture",
		"/sources", "/verge-core",
	} {
		t.Run(at, func(t *testing.T) {
			page := getBody(t, ac, base+at, http.StatusOK)
			if want := `name="return" value="` + at + `"`; !strings.Contains(page, want) {
				t.Fatalf("%s carries no submitting-URL field %s; body: %s", at, want, page)
			}
		})
	}
}

// EVERY form on the SSO tab stamps the field, not only the first. Each of the five acts
// has its own form, and one that missed the field would 303 to the tab fallback and drop
// the operator's place — the class-E failure this map exists to close.
func TestEverySSOFormCarriesTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The five forms: add provider, edit provider, replace secret, remove provider,
	// remove identity binding.
	page := getBody(t, ac, base+"/settings?tab=sso", http.StatusOK)
	if got := strings.Count(page, `name="return" value="/settings?tab=sso"`); got != 5 {
		t.Fatalf("the sso tab stamps the field %d times, want 5 (one per form); body: %s", got, page)
	}
}

// A refused provider declaration is a 303 back to the operator's own tab, and the
// message AND everything they typed render on the landing GET.
//
// The client secret is why the payload rides the session flash rather than the query: a
// URL is written to the access log and kept in browser history. The secret is
// write-only, so it is echoed nowhere at all.
func TestRefusedSSOProviderCreateEchoesTypedValuesOnItsOwnTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=sso"
	resp := postForm(t, ac, base+"/settings/sso", url.Values{
		"slug": {"okta"}, "name": {"Okta"}, "issuer": {"http://idp.example"},
		"client_id": {"cid"}, "client_secret": {"super-secret-value"}, "return": {from},
	})
	loc := submitLoc(t, resp)
	if loc != from {
		t.Fatalf("refused provider create landed at %q, want %q", loc, from)
	}
	if strings.Contains(loc, "idp.example") || strings.Contains(loc, "super-secret-value") {
		t.Fatalf("a typed value leaked into the URL: %q", loc)
	}
	if len(f.ssoProviders) != 0 {
		t.Fatalf("a refused create stored a provider; got %d", len(f.ssoProviders))
	}

	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "must be an https URL") {
		t.Fatalf("the refusal is not on the landing page; body: %s", page)
	}
	for _, echo := range []string{`value="okta"`, `value="Okta"`, `value="http://idp.example"`, `value="cid"`} {
		if !strings.Contains(page, echo) {
			t.Fatalf("the typed form was not echoed back (%s); body: %s", echo, page)
		}
	}
	if strings.Contains(page, "super-secret-value") {
		t.Fatalf("the client secret was echoed back; body: %s", page)
	}
}

// A refused provider EDIT keeps the operator on the SSO tab with the message visible.
// The edit form is a row disclosure rather than a query-opened dialog, so there is no
// modal to drop from the destination and nothing covers the callout.
func TestRefusedSSOProviderEditLandsBackOnTheSSOTab(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addSSOProvider(f, 1, "okta", "Okta")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=sso"
	resp := postForm(t, ac, base+"/settings/sso/update", url.Values{
		"id": {"404"}, "slug": {"okta"}, "name": {"Okta"},
		"issuer": {"https://idp.example"}, "client_id": {"cid"}, "return": {from},
	})
	if loc := submitLoc(t, resp); loc != from {
		t.Fatalf("refused provider edit landed at %q, want %q", loc, from)
	}
	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "could not be found") {
		t.Fatalf("the refusal is not on the landing page; body: %s", page)
	}
	// The flash is single-consume, so a reload of the same URL shows a clean tab.
	if again := getBody(t, ac, base+from, http.StatusOK); strings.Contains(again, "could not be found") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

// A succeeding SSO act comes back to the submitting URL too. That is the point of the
// contract: one destination rule for both outcomes, so a refusal is indistinguishable
// from a success and the scroll offset survives either.
func TestSucceedingSSOActsComeBackToTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=sso"
	for _, tc := range []struct {
		name string
		at   string
		vals url.Values
	}{
		{"replace secret", "/settings/sso/secret", url.Values{
			"id": {"1"}, "client_secret": {"s3cret"}, "return": {from},
		}},
		{"remove identity", "/settings/sso/identity/remove", url.Values{
			"id": {"1"}, "return": {from},
		}},
		{"remove provider", "/settings/sso/delete", url.Values{
			"id": {"1"}, "return": {from},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if loc := submitLoc(t, postForm(t, ac, base+tc.at, tc.vals)); loc != from {
				t.Fatalf("landed at %q, want %q", loc, from)
			}
		})
	}
}

// A source state change comes back to the surface it was made on. The sources tab has
// two landing GETs, and an act from the folded one must return there — on the success
// and on the refusal alike.
func TestSourceStateChangeComesBackToTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// crt.sh ships on and executing (ADR-0106), so disabling it is an ordinary success.
	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"crtsh"}, "enable": {"false"}, "return": {"/sources"},
	})); loc != "/sources" {
		t.Fatalf("source disable landed at %q, want /sources", loc)
	}
	if st, ok := f.sourceStates["crtsh"]; !ok || st.Enabled {
		t.Fatalf("the disable did not persist: %+v", f.sourceStates["crtsh"])
	}

	// A refusal is a 303 to the same place, and the folded surface reads the flash, so
	// the callout renders there rather than being dropped on the floor.
	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"crtsh"}, "enable": {"maybe"}, "return": {"/sources"},
	})); loc != "/sources" {
		t.Fatalf("refused source change landed at %q, want /sources", loc)
	}
	if page := getBody(t, ac, base+"/sources", http.StatusOK); !strings.Contains(page, "was not understood") {
		t.Fatalf("the refusal is not on the /sources landing; body: %s", page)
	}
}

// withConsentSource appends an operator-accepted source WITH a runner to the release
// catalogue for the length of one test. Every operator-accepted entry the release ships
// today also carries NoRunner (#241), so no live source opens the consent dialog and the
// terms gate has no subject. The dialog and the gate are both still in the tree and both
// still have to hold, so the test supplies the subject the catalogue does not.
func withConsentSource(t *testing.T, slug, name string) {
	t.Helper()
	before := sourceCatalog
	sourceCatalog = append(append([]catalogSource(nil), sourceCatalog...), catalogSource{
		Slug: slug, Name: name, IsProposer: true, Consent: consentAccepted,
		ShipNote:   "A test-only operator-accepted source that executes.",
		MayResolve: []string{"who asked for this lookup"},
	})
	t.Cleanup(func() { sourceCatalog = before })
}

// The consent dialog's own form obeys the contract on both outcomes, and its acceptance
// box is a real gate with JavaScript off.
//
// The submit button ships ENABLED — the ruling #971 made for the /signals declare
// button — so an operator with no JavaScript can reach it. What stops an unaccepted
// enable is the handler, not the disabled attribute: the box itself carries
// `accept_terms`, and without it settingsSources refuses rather than applies.
//
// The `consent` parameter is dropped from the destination on either outcome
// (dialogParams). A success would otherwise re-open the terms of a source just enabled,
// and a refusal would leave its callout behind .st-scrim.
func TestConsentDialogEnableObeysTheContract(t *testing.T) {
	withConsentSource(t, "test-registry", "Test registry")
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const tab = "/settings?tab=sources"
	const dialog = tab + "&consent=test-registry"

	page := getBody(t, ac, base+dialog, http.StatusOK)
	if !strings.Contains(page, "Accept and enable") {
		t.Fatalf("the consent dialog did not open; body: %s", page)
	}
	if !strings.Contains(page, `name="return" value="/settings?tab=sources&amp;consent=test-registry"`) {
		t.Fatalf("the consent form carries no submitting-URL field; body: %s", page)
	}
	if strings.Contains(page, "disabled data-st-consent-submit") {
		t.Fatalf("the consent submit ships disabled, so it is unusable with JavaScript off; body: %s", page)
	}
	if !strings.Contains(page, `name="accept_terms" value="true"`) {
		t.Fatalf("the acceptance box does not carry accept_terms; body: %s", page)
	}

	// An unticked submit — what a JavaScript-off operator can send — is refused, and its
	// callout lands on the tab with the dialog closed.
	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"test-registry"}, "enable": {"true"}, "return": {dialog},
	})); loc != tab {
		t.Fatalf("unaccepted enable landed at %q, want %q with the dialog closed", loc, tab)
	}
	if st, ok := f.sourceStates["test-registry"]; ok && st.Enabled {
		t.Fatalf("an unaccepted enable turned the source on: %+v", st)
	}
	land := getBody(t, ac, base+tab, http.StatusOK)
	if !strings.Contains(land, "Accept the terms before you enable Test registry.") {
		t.Fatalf("the refusal is not on the landing page; body: %s", land)
	}
	if strings.Contains(land, `class="st-scrim"`) {
		t.Fatalf("the landing re-opened the dialog over the message; body: %s", land)
	}

	// A ticked submit enables the source and closes the dialog.
	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"test-registry"}, "enable": {"true"}, "accept_terms": {"true"}, "return": {dialog},
	})); loc != tab {
		t.Fatalf("accepted enable landed at %q, want %q with the dialog closed", loc, tab)
	}
	if st, ok := f.sourceStates["test-registry"]; !ok || !st.Enabled {
		t.Fatalf("the accepted enable did not persist: %+v", f.sourceStates["test-registry"])
	}
}

// The aperture port dial is this ticket's check on the flash carrier: a rejected port
// comes back with the operator's typed value still in the input, AFTER a redirect rather
// than from an inline render. Both of the tab's landing GETs serve it.
func TestRejectedAperturePortEchoesTypedValueOnBothSurfaces(t *testing.T) {
	for _, from := range []string{"/settings?tab=aperture", "/verge-core"} {
		t.Run(from, func(t *testing.T) {
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")

			resp := postForm(t, ac, base+"/verge-core/frequency", url.Values{
				"action": {"add"}, "port": {"70000"}, "return": {from},
			})
			loc := submitLoc(t, resp)
			if loc != from {
				t.Fatalf("rejected port landed at %q, want %q", loc, from)
			}
			if strings.Contains(loc, "70000") {
				t.Fatalf("the typed port leaked into the URL: %q", loc)
			}
			if len(f.freqEdits) != 0 {
				t.Fatalf("a rejected edit stored a row: %+v", f.freqEdits)
			}
			page := getBody(t, ac, base+from, http.StatusOK)
			if !strings.Contains(page, "between 1 and 65535") {
				t.Fatalf("the refusal is not on the landing page; body: %s", page)
			}
			if !strings.Contains(page, `value="70000"`) {
				t.Fatalf("the typed port was not echoed back; body: %s", page)
			}
		})
	}
}

// A succeeding port dial comes back to the surface it was dialled from.
func TestSucceedingAperturePortComesBackToTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=aperture"
	if loc := submitLoc(t, postForm(t, ac, base+"/verge-core/frequency", url.Values{
		"action": {"add"}, "port": {"12345"}, "return": {from},
	})); loc != from {
		t.Fatalf("port add landed at %q, want %q", loc, from)
	}
	if e, ok := f.freqEdits[12345]; !ok || e.action != "add" {
		t.Fatalf("add edit not stored: %+v", f.freqEdits)
	}
}
