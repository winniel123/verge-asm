package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

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

func TestEverySSOFormCarriesTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	alice := seedAccount(t, f, "alice", roleViewer, "unused-password-x")
	addSSOProvider(f, 1, "okta", "Okta")
	addSSOIdentity(f, 1, "okta-sub-alice", alice.ID, "alice@corp")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/settings?tab=sso", http.StatusOK)
	// A form that missed the field would fall back to the tab and drop the operator's place.
	if got := strings.Count(page, `name="return" value="/settings?tab=sso"`); got != 5 {
		t.Fatalf("the sso tab stamps the field %d times, want 5 (one per form); body: %s", got, page)
	}
}

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
	if again := getBody(t, ac, base+from, http.StatusOK); strings.Contains(again, "could not be found") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

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

func TestSourceStateChangeComesBackToTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"crtsh"}, "enable": {"false"}, "return": {"/sources"},
	})); loc != "/sources" {
		t.Fatalf("source disable landed at %q, want /sources", loc)
	}
	if st, ok := f.sourceStates["crtsh"]; !ok || st.Enabled {
		t.Fatalf("the disable did not persist: %+v", f.sourceStates["crtsh"])
	}

	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"crtsh"}, "enable": {"maybe"}, "return": {"/sources"},
	})); loc != "/sources" {
		t.Fatalf("refused source change landed at %q, want /sources", loc)
	}
	if page := getBody(t, ac, base+"/sources", http.StatusOK); !strings.Contains(page, "was not understood") {
		t.Fatalf("the refusal is not on the /sources landing; body: %s", page)
	}
}

func withConsentSource(t *testing.T, slug, name string) {
	// Every shipped operator-accepted source carries NoRunner, so the consent gate has no subject.
	t.Helper()
	before := sourceCatalog
	sourceCatalog = append(append([]catalogSource(nil), sourceCatalog...), catalogSource{
		Slug: slug, Name: name, IsProposer: true, Consent: consentAccepted,
		ShipNote:   "A test-only operator-accepted source that executes.",
		MayResolve: []string{"who asked for this lookup"},
	})
	t.Cleanup(func() { sourceCatalog = before })
}

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

	if loc := submitLoc(t, postForm(t, ac, base+"/settings/sources", url.Values{
		"id": {"test-registry"}, "enable": {"true"}, "accept_terms": {"true"}, "return": {dialog},
	})); loc != tab {
		t.Fatalf("accepted enable landed at %q, want %q with the dialog closed", loc, tab)
	}
	if st, ok := f.sourceStates["test-registry"]; !ok || !st.Enabled {
		t.Fatalf("the accepted enable did not persist: %+v", f.sourceStates["test-registry"])
	}
}

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
