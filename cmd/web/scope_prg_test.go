package main

import (
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/proposer"
)

// The ADR-0130 contract on the /scope surface (map #969, ticket #976). Every mutating
// act on the Scope screen is a post-redirect-get back to the exact URL its form was
// submitted from, and a refusal carries its callouts to that landing GET through the
// session form flash rather than rendering at the POST URL.
//
// Every test here runs with no JavaScript at all — Go's HTTP client executes none — so
// each one is also the progressive-enhancement check the ticket asks for: the act works,
// and its refusals show, on plain markup alone.

// The scope tests read a refusal's message off its landing GET through the shared
// refusalPage / prgLanding helpers (annotations_test.go, settings_test.go), which is the
// shape every migrated surface's refusal now takes.

// declareFrom posts a scope declaration carrying an explicit submitting URL, the way the
// markup's own hidden field does.
func declareFrom(t *testing.T, c *http.Client, base, from, raw string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds", url.Values{"scope": {raw}, "return": {from}})
}

// TestScopeFormsCarryTheSubmittingURL is the emit half of ADR-0130 §3: every form on the
// Scope screen stamps the page's own path and query into the hidden `return` field, so
// each handler has the operator's exact URL to come back to.
//
// The screen has ten forms, and the count is asserted rather than sampled: a form added
// without the field is a mutating act that silently drops the operator back at bare
// /scope, which is the class-E failure this map exists to close.
func TestScopeFormsCarryTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// A declared name scope gives the custody, zone and seed-chip forms a subject to
	// render against.
	addNameSeed(t, f, admin.ID, "example.com")
	// A filed proposal gives the confirm row one, and the decline form its list.
	base := startWithProposer(t, f, &fakeProposer{candidates: []proposer.Candidate{
		{SourceSlug: proposer.SlugAFRINIC, RecordKind: proposer.RecordRIRDelegation,
			Scope: netip.MustParsePrefix("203.0.113.0/24"), OrgName: "Acme"},
	}})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Acme").Body.Close()
	// A declared exclusion gives the exclusion-row remove form one.
	postForm(t, ac, base+"/exclusions", url.Values{
		"kind": {"subtree"}, "value": {"old.example.com"},
	}).Body.Close()

	page := getBody(t, ac, base+"/scope", http.StatusOK)
	// The screen's own acts, not the chrome's: shell.tmpl's sign-out form posts to a
	// fixed destination by design and is out of scope for this map.
	var forms int
	for _, act := range []string{"/seeds", "/seeds/delete", "/seeds/custody", "/seeds/zone",
		"/seeds/zone/interval", "/exclusions", "/exclusions/delete", "/proposals/search",
		"/proposals/confirm", "/proposals/decline"} {
		if n := strings.Count(page, `action="`+act+`"`); n == 0 {
			t.Errorf("no form posts to %s on /scope", act)
		} else {
			forms += n
		}
	}
	if fields := strings.Count(page, `name="return" value="/scope"`); fields != forms {
		t.Fatalf("%d of %d POST forms on /scope carry the submitting-URL field; body: %s",
			fields, forms, page)
	}
}

// A refused declaration is a 303 back to the submitting URL, and its callout renders on
// the landing GET at 200 — not as a 400 body at the POST URL. This is failure classes A
// and E closed together.
//
// The destination is the submitting URL VERBATIM. /scope carries no dialog parameter of
// the kind the settings tabs drop (see backToScope), so nothing is stripped.
func TestRefusedDeclarationLandsBackAtTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/scope?seen=1"
	resp := declareFrom(t, ac, base, from, "10.0.0.0/21")
	if loc := submitLoc(t, resp); loc != from {
		t.Fatalf("refused declaration landed at %q, want %q", loc, from)
	}

	page := getBody(t, ac, base+from, http.StatusOK)
	if !strings.Contains(page, "over the 1,024-address cap") {
		t.Fatalf("the refusal callout is not on the landing page; body: %s", page)
	}
	// The operator's typed value is echoed back into the field, so they need not retype it.
	if !strings.Contains(page, `value="10.0.0.0/21"`) {
		t.Fatalf("the refused input was not echoed; body: %s", page)
	}
	// Nothing the operator typed entered the URL, and the flash is single-consume, so a
	// reload shows a clean page rather than a stale callout.
	if again := getBody(t, ac, base+from, http.StatusOK); strings.Contains(again, "over the 1,024-address cap") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

// The mixed bulk declare is the hard case the ticket names. A paste that mixes successes
// with refusals must show the success toast AND every per-token callout in ONE landing
// render, with the refused inputs still in the field.
//
// Before ADR-0130 that took an inline toast on a 200 rendered at the POST URL, because a
// redirect would have dropped the callouts. Now the callouts ride the session flash and
// the toast rides the `toast` query, so the whole result survives one redirect back to
// the operator's own URL — and their scroll offset survives with it.
func TestMixedBulkDeclareLandsWithBothTheToastAndEveryCallout(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/scope"
	resp := declareFrom(t, ac, base, from, "good1.com good2.com good1.com 10.0.0.0/21")
	loc := submitLoc(t, resp)
	// The receipt rides the query. The path and every other pair are the submitting URL's.
	if !strings.HasPrefix(loc, from+"?") || !strings.Contains(loc, "toast=") {
		t.Fatalf("mixed declare landed at %q, want %q with a toast receipt", loc, from)
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	for _, want := range []string{
		"2 scopes declared",            // the success toast
		"2 refused — see the callouts", // its description
		"already declared",             // the within-paste duplicate's callout
		"the cap is 1,024 per scope",   // the over-cap callout
		"10.0.0.0/22",                  // the reachable in-cap set it names
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the mixed landing is missing %q; body: %s", want, page)
		}
	}
	// The refused inputs are still in the field, so the operator can correct them in place.
	if !strings.Contains(page, `value="good1.com, 10.0.0.0/21"`) {
		t.Errorf("the refused inputs were not echoed; body: %s", page)
	}
	if len(f.seeds) != 2 {
		t.Fatalf("committed %d seeds, want 2", len(f.seeds))
	}
}

// An ALL-refused declare shows the callouts and no toast, and a PURE-success declare
// shows the toast and no callout. The two ends of the mixed case above.
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

// A refused org-name search shows its reason. The refusal already rode the flash before
// ticket #978, but seedsForms.proposalError had no hole to render into: the parity
// conversion (#574) dropped .ProposalError from scope.tmpl and the field from
// renderSeeds, so both refusals — an empty search and a scope already declared — landed
// on a clean page saying nothing at all. The sweep found the field with two setters and
// no reader, and restored the hole.
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
	// Single-consume, like every other callout on this surface.
	if again := getBody(t, ac, base+from, http.StatusOK); strings.Contains(again, "Enter an organisation name") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

// The narrowing preview is a receipt, not a refusal, and it must still reach the
// operator after the redirect. It rides the same session flash the callouts do.
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
	// The candidate value is echoed, so Add exclusion acts on what was previewed.
	if !strings.Contains(page, `value="203.0.113.0/25"`) {
		t.Fatalf("the previewed value was not echoed; body: %s", page)
	}
	// Nothing was committed: a preview is a read.
	if len(f.exclusions) != 0 {
		t.Fatalf("the preview committed %d exclusions, want 0", len(f.exclusions))
	}
}

// A SUCCEEDING act returns to the submitting URL too, not to bare /scope. The
// destination rule is the same for a success and a refusal, which is what lets the
// operator's scroll offset survive either.
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
		// The seed delete names the removed scope in a toast, so its destination carries
		// the receipt on top of the submitting URL.
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

// Every act on the screen has to work with JavaScript off, which means the markup has
// to ship the control that starts it. Two on this screen did not: the bulk-decline
// button was `disabled` in the markup and enabled only by script, and the zone form had
// no submit control at all — its only submit was the file input's `change` handler.
//
// Both are the ruling #971 and #975 each made on their own surface, applied here: the
// markup ships the control working, and the script takes over on load for a browser
// that has one. This test reads the markup, because a Go HTTP client runs no script and
// so cannot tell a script-enabled control from a broken one.
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

// The bulk decline works on plain markup: the ids ride the checkboxes' own form
// association, the act 303s back to the submitting URL, and each declined scope is
// recorded as the address exclusion that makes the decline durable.
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

// The guard refuses a `return` this server does not serve a GET at, and falls back to
// bare /scope rather than 303ing the operator off-site (backurl.go resolveBack). The
// scope forms are operator-controlled input like any other, so the check is asserted
// here as well as at the helper.
func TestScopeActsRefuseAForeignReturn(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, bad := range []string{"https://evil.example/x", "//evil.example/x", `/\evil.example`, "/nope"} {
		loc := submitLoc(t, declareFrom(t, ac, base, bad, "10.0.0.0/21"))
		if loc != "/scope" {
			t.Fatalf("return %q resolved to %q, want the /scope fallback", bad, loc)
		}
	}
}
