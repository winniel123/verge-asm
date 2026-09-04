package main

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/proposer"
)

func declareFrom(t *testing.T, c *http.Client, base, from, raw string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds", url.Values{"scope": {raw}, "return": {from}})
}

func TestScopeFormsCarryTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	base := startWithProposer(t, f, &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Acme"},
	}})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Acme").Body.Close()
	postForm(t, ac, base+"/exclusions", url.Values{
		"kind": {"subtree"}, "value": {"old.example.com"},
	}).Body.Close()

	page := getBody(t, ac, base+"/scope", http.StatusOK)
	var forms int
	// The withdrawal is two-step, so the chip's remove control posts to /seeds/preview.
	// A confirm state is the only render that ships /seeds/delete, and this is not one.
	for _, act := range []string{"/seeds", "/seeds/preview", "/seeds/custody", "/seeds/zone",
		"/seeds/zone/interval", "/exclusions", "/exclusions/delete", "/proposals/search",
		"/proposals/confirm", "/proposals/decline"} {
		if n := strings.Count(page, `action="`+act+`"`); n == 0 {
			t.Errorf("no form posts to %s on /scope", act)
		} else {
			forms += n
		}
	}
	// A form without the field drops the operator at bare /scope, so count rather than sample.
	if fields := strings.Count(page, `name="return" value="/scope"`); fields != forms {
		t.Fatalf("%d of %d POST forms on /scope carry the submitting-URL field; body: %s",
			fields, forms, page)
	}
}

func TestRefusedDeclarationLandsBackAtTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/scope?seen=1"
	resp := declareFrom(t, ac, base, from, "10.0.0.0/21")
	// The settings tabs drop a dialog parameter and /scope carries none, so nothing is stripped.
	if loc := submitLoc(t, resp); loc != from {
		t.Fatalf("refused declaration landed at %q, want %q", loc, from)
	}

	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "over the 1,024-address cap") {
		t.Fatalf("the refusal callout is not on the landing page; body: %s", page)
	}
	if !strings.Contains(page, `value="10.0.0.0/21"`) {
		t.Fatalf("the refused input was not echoed; body: %s", page)
	}
	if again := getBody(t, ac, base+from, http.StatusOK); strings.Contains(again, "over the 1,024-address cap") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

func TestMixedBulkDeclareLandsWithBothTheToastAndEveryCallout(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/scope"
	resp := declareFrom(t, ac, base, from, "good1.com good2.com good1.com 10.0.0.0/21")
	loc := submitLoc(t, resp)
	if !strings.HasPrefix(loc, from+"?") || !strings.Contains(loc, "toast=") {
		t.Fatalf("mixed declare landed at %q, want %q with a toast receipt", loc, from)
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	for _, want := range []string{
		"2 scopes declared",
		"2 refused — see the callouts",
		"already declared",
		"the cap is 1,024 per scope",
		"10.0.0.0/22",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the mixed landing is missing %q; body: %s", want, page)
		}
	}
	if !strings.Contains(page, `value="good1.com, 10.0.0.0/21"`) {
		t.Errorf("the refused inputs were not echoed; body: %s", page)
	}
	if len(f.seeds) != 2 {
		t.Fatalf("committed %d seeds, want 2", len(f.seeds))
	}
}

func TestBulkDeclareEndsCarryOnlyWhatTheyEarned(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	t.Run("all refused", func(t *testing.T) {
		resp := declareScope(t, ac, base, "www.example.com, 10.0.0.0/21")
		loc := submitLoc(t, resp)
		if strings.Contains(loc, "toast=") {
			t.Fatalf("an all-refused declare fired a toast: %q", loc)
		}
		page := getBody(t, ac, base+loc, http.StatusOK)
		if !strings.Contains(page, "registrable domain example.com") ||
			!strings.Contains(page, "the cap is 1,024 per scope") {
			t.Fatalf("an all-refused declare lost its callouts; body: %s", page)
		}
		if len(f.seeds) != 0 {
			t.Fatalf("an all-refused declare committed %d seeds, want 0", len(f.seeds))
		}
	})

	t.Run("pure success", func(t *testing.T) {
		resp := declareScope(t, ac, base, "acmecorp.io")
		loc := submitLoc(t, resp)
		if !strings.Contains(loc, "toast=") {
			t.Fatalf("a pure-success declare fired no toast: %q", loc)
		}
		page := getBody(t, ac, base+loc, http.StatusOK)
		if !strings.Contains(page, "1 scope declared") {
			t.Fatalf("the success toast is not on the landing; body: %s", page)
		}
		if strings.Contains(page, "see the callouts") || strings.Contains(page, `class="sc-refusal"`) {
			t.Fatalf("a pure-success declare rendered a callout; body: %s", page)
		}
	})
}

func TestRefusedOrgSearchShowsItsReason(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/scope"
	resp := postForm(t, ac, base+"/proposals/search", url.Values{"org": {"   "}, "return": {from}})
	if loc := submitLoc(t, resp); loc != from {
		t.Fatalf("refused org search landed at %q, want %q", loc, from)
	}
	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "Enter an organisation name to search") {
		t.Fatalf("the refusal is not on the landing page; body: %s", page)
	}
	if again := getBody(t, ac, base+from, http.StatusOK); strings.Contains(again, "Enter an organisation name") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

func TestNarrowingPreviewSurvivesTheRedirect(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "address", "203.0.113.0/24").Body.Close()

	const from = "/scope"
	resp := postForm(t, ac, base+"/exclusions/preview", url.Values{
		"kind": {"address"}, "value": {"203.0.113.0/25"}, "return": {from},
	})
	if loc := submitLoc(t, resp); loc != from {
		t.Fatalf("preview landed at %q, want %q", loc, from)
	}
	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "What this exclusion would withdraw") {
		t.Fatalf("the narrowing receipt did not survive the redirect; body: %s", page)
	}
	if !strings.Contains(page, `value="203.0.113.0/25"`) {
		t.Fatalf("the previewed value was not echoed; body: %s", page)
	}
	if len(f.exclusions) != 0 {
		t.Fatalf("the preview committed %d exclusions, want 0", len(f.exclusions))
	}
}

func TestSucceedingScopeActsReturnToTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedID := addNameSeed(t, f, admin.ID, "example.com")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/scope?seen=1"
	for _, tc := range []struct {
		name  string
		path  string
		form  url.Values
		toast bool
	}{
		{"custody toggle", "/seeds/custody",
			url.Values{"id": {intStr(seedID)}, "extend": {"true"}, "return": {from}}, false},
		{"exclusion create", "/exclusions",
			url.Values{"kind": {"subtree"}, "value": {"old.example.com"}, "return": {from}}, false},
		{"zone interval", "/seeds/zone/interval",
			url.Values{"interval_days": {"14"}, "return": {from}}, false},
		{"seed delete", "/seeds/delete",
			url.Values{"id": {intStr(seedID)}, "return": {from}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loc := submitLoc(t, postForm(t, ac, base+tc.path, tc.form))
			if tc.toast {
				if !strings.HasPrefix(loc, from+"&") || !strings.Contains(loc, "toast=") {
					t.Fatalf("%s landed at %q, want %q plus a toast receipt", tc.name, loc, from)
				}
				return
			}
			if loc != from {
				t.Fatalf("%s landed at %q, want %q", tc.name, loc, from)
			}
		})
	}
}

func TestScopeControlsShipUsableWithoutScript(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	base := startWithProposer(t, f, &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Acme"},
	}})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Acme").Body.Close()

	// A Go HTTP client runs no script, so only the markup tells a usable control from a broken one.
	page := getBody(t, ac, base+"/scope", http.StatusOK)
	if !strings.Contains(page, `id="sc-decline-btn"`) {
		t.Fatal("the bulk-decline button is not on the page")
	}
	if strings.Contains(page, `id="sc-decline-btn" style="margin-left:auto" disabled`) {
		t.Error("the bulk-decline button ships disabled, so it cannot be used without script")
	}
	if !strings.Contains(page, `data-zone-submit`) {
		t.Error("the zone form ships no submit control, so an upload cannot be started without script")
	}
}

func TestBulkDeclineWorksWithoutScript(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Acme"},
	}})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Acme").Body.Close()

	const from = "/scope?seen=1"
	loc := submitLoc(t, postForm(t, ac, base+"/proposals/decline", url.Values{
		"ids": {itoa(f.proposals[0].ID)}, "return": {from},
	}))
	if loc != from {
		t.Fatalf("the decline landed at %q, want %q", loc, from)
	}
	if f.proposals[0].Status != "declined" {
		t.Fatalf("proposal status = %q, want declined", f.proposals[0].Status)
	}
	if len(f.exclusions) != 1 {
		t.Fatalf("the decline recorded %d exclusions, want 1", len(f.exclusions))
	}
}

func TestScopeActsRefuseAForeignReturn(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// These forms are operator-controlled input, so the guard is asserted here and at resolveBack.
	for _, bad := range []string{"https://evil.example/x", "//evil.example/x", `/\evil.example`, "/nope"} {
		loc := submitLoc(t, declareFrom(t, ac, base, bad, "10.0.0.0/21"))
		if loc != "/scope" {
			t.Fatalf("return %q resolved to %q, want the /scope fallback", bad, loc)
		}
	}
}
