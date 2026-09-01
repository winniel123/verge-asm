package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// TestScopeAndOnboardingShareTokenizer proves the paste-split declare (DF-F1) and the
// onboarding seedsadd field tokenize an identical raw string identically — they call
// the ONE parseSeedTokens (cmd/web/onboarding.go), never a fork. The scope side is
// checked end-to-end (every valid token becomes a seed); the onboarding side through
// readOnboardView; and both are pinned to parseSeedTokens' own output.
func TestScopeAndOnboardingShareTokenizer(t *testing.T) {
	// Commas, spaces, tabs and newlines all split; empty tokens (the double comma) drop.
	const raw = "a.com, b.com\n c.com\td.com,,e.com"
	want := []string{"a.com", "b.com", "c.com", "d.com", "e.com"}

	// The shared tokenizer itself.
	if got := parseSeedTokens(raw); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSeedTokens(%q) = %v, want %v", raw, got, want)
	}

	// Onboarding's seedsadd rides parseSeedTokens through readOnboardView.
	q := url.Values{"step": {"0"}, "seedsadd": {raw}}
	r := httptest.NewRequest(http.MethodGet, "/onboarding?"+q.Encode(), nil)
	if got := readOnboardView(r).Seeds; !reflect.DeepEqual(got, want) {
		t.Fatalf("onboarding tokenization = %v, want %v (must match parseSeedTokens)", got, want)
	}

	// The scope declare paste-splits with the SAME tokenizer: every valid token commits
	// as its own seed.
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := declareScope(t, ac, base, raw)
	resp.Body.Close()
	if len(f.seeds) != len(want) {
		t.Fatalf("declared %d seeds, want %d (one per token); seeds=%+v", len(f.seeds), len(want), f.seeds)
	}
	got := map[string]bool{}
	for _, s := range f.seeds {
		got[s.NameDomain.String] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("token %q was not declared; declared=%v", w, got)
		}
	}
}

// TestDeclareBulkPasteMixedSuccessRefusalDuplicate covers the DF-F1 paste with a mix of
// success, refusal, and a within-paste duplicate: two good names declare, the repeat of
// the first refuses `already declared` (the first wins), and an over-cap block refuses
// with its reachable set named. The response carries BOTH the success flash AND the
// per-token callouts on one render (status 200, since successes committed).
func TestDeclareBulkPasteMixedSuccessRefusalDuplicate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// good1 declares, good2 declares, good1 again is a within-paste duplicate, and the
	// /21 block is over the 1,024 cap.
	// The result is one post-redirect-get (ADR-0130 §1): the callouts ride the session
	// form flash and the toast the `toast` query, so both land on one GET.
	got := refusalPage(t, ac, base, declareScope(t, ac, base, "good1.com good2.com good1.com 10.0.0.0/21"))

	// Exactly the two distinct valid names committed — the duplicate did not create a
	// second row, the over-cap block created none.
	if len(f.seeds) != 2 {
		t.Fatalf("committed %d seeds, want 2 (good1.com, good2.com); seeds=%+v", len(f.seeds), f.seeds)
	}
	// The success flash names the count, and its description names the refusals.
	if !strings.Contains(got, "2 scopes declared") {
		t.Errorf("success flash missing; body: %s", got)
	}
	if !strings.Contains(got, "2 refused — see the callouts") {
		t.Errorf("flash refusal description missing; body: %s", got)
	}
	// The within-paste duplicate refuses `already declared`.
	if !strings.Contains(got, "already declared") {
		t.Errorf("duplicate token not refused with 'already declared'; body: %s", got)
	}
	// The over-cap block refuses and NAMES its reachable in-cap set (never auto-applied).
	if !strings.Contains(got, "the cap is 1,024 per scope") {
		t.Errorf("over-cap refusal reason missing; body: %s", got)
	}
	if !strings.Contains(got, "10.0.0.0/22") {
		t.Errorf("over-cap reachable set not named; body: %s", got)
	}
}

// TestDeclareBulkPasteAllRefusedNoFlash: when no token declares, there is no toast —
// only the callouts, on the landing GET of the refusal's own post-redirect-get.
func TestDeclareBulkPasteAllRefusedNoFlash(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declareScope(t, ac, base, "www.example.com, 10.0.0.0/21"))
	if len(f.seeds) != 0 {
		t.Fatalf("all-refused committed %d seeds, want 0", len(f.seeds))
	}
	if strings.Contains(got, `<div class="sh-toasts"`) {
		t.Errorf("all-refused should fire no success flash; body: %s", got)
	}
	// The subdomain is refused toward its registrable domain; the /21 over the cap.
	if !strings.Contains(got, "registrable domain example.com") {
		t.Errorf("subdomain refusal reason missing; body: %s", got)
	}
	if !strings.Contains(got, "the cap is 1,024 per scope") {
		t.Errorf("over-cap refusal reason missing; body: %s", got)
	}
}

// TestZoneBulkUploadMixedAcceptRefuse covers DF-F2: one multipart POST with several
// zonefile parts — an in-scope apex accepted, a foreign apex refused, an unparseable
// file refused `not a zone file`, and a second file for the accepted apex (both acts
// recorded, the later the current supply). ≥1 accepted fires the "zone files supplied"
// flash while the refusal rows render on the same response.
func TestZoneBulkUploadMixedAcceptRefuse(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()

	first := "$ORIGIN example.com.\n@ IN A 203.0.113.10\n"
	second := "$ORIGIN example.com.\n@ IN A 203.0.113.20\n" // same apex, later supply
	foreign := "$ORIGIN notmine.example.\n@ IN A 203.0.113.10\n"
	garbage := "this is not a zone file at all\n"

	// Two accepted, two refused: one post-redirect-get carries the toast and the refusal
	// rows to the landing GET (ADR-0130 §1).
	got := refusalPage(t, ac, base, uploadZones(t, ac, base, []zonePart{
		{"one.example.com.zone", first},
		{"foreign.zone", foreign},
		{"garbage.zone", garbage},
		{"two.example.com.zone", second},
	}))
	if len(f.zoneFiles) != 2 {
		t.Fatalf("stored %d zone files, want 2 (both example.com acts recorded)", len(f.zoneFiles))
	}
	if !strings.Contains(got, "2 zone files supplied") {
		t.Errorf("zone success flash missing; body: %s", got)
	}
	if !strings.Contains(got, "2 refused") {
		t.Errorf("zone flash refusal count missing; body: %s", got)
	}
	// The foreign apex is refused with the reason; the garbage file is `not a zone file`.
	if !strings.Contains(got, "outside every declared name scope") {
		t.Errorf("foreign-apex refusal reason missing; body: %s", got)
	}
	if !strings.Contains(got, "not a zone file") {
		t.Errorf("unparseable file not refused with 'not a zone file'; body: %s", got)
	}
	// Two files for one apex → the later is the current supply (last content wins).
	rows, _ := f.ListZoneFileStatus(t.Context())
	if len(rows) != 1 {
		t.Fatalf("zone status rows = %d, want 1 (one apex)", len(rows))
	}
	if !strings.Contains(f.zoneFiles[len(f.zoneFiles)-1].content, "203.0.113.20") {
		t.Errorf("later file is not the current supply; files=%+v", f.zoneFiles)
	}
}

// TestZoneBulkUploadAllRefusedNoFlash: zero accepted → refusal rows only, no toast, on
// the landing GET of the refusal's own post-redirect-get.
func TestZoneBulkUploadAllRefusedNoFlash(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()

	got := refusalPage(t, ac, base, uploadZones(t, ac, base, []zonePart{
		{"foreign.zone", "$ORIGIN notmine.example.\n@ IN A 203.0.113.10\n"},
		{"garbage.zone", "definitely not a zone\n"},
	}))
	if len(f.zoneFiles) != 0 {
		t.Fatalf("all-refused upload stored %d files, want 0", len(f.zoneFiles))
	}
	// No accepted file → no ToastStack at all (the empty zone rows say "no zone file
	// supplied", so match on the flash container, not a substring).
	if strings.Contains(got, `<div class="sh-toasts"`) {
		t.Errorf("all-refused upload should fire no flash; body: %s", got)
	}
}

// TestDeleteSeedWritesRemovalFlash: /seeds/delete names the removed scope in a flash
// (WORK-ORDER-DOGFOOD-R1 item 2).
func TestDeleteSeedWritesRemovalFlash(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID

	resp := postForm(t, ac, base+"/seeds/delete", url.Values{"id": {intStr(id)}})
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/scope") {
		t.Fatalf("delete: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	// The flash rides the PRG toast query; decode it on the destination GET.
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	page := followString(t, ac, base+loc)
	if !strings.Contains(page, "Scope removed") {
		t.Errorf("removal flash title missing; body: %s", page)
	}
	if !strings.Contains(page, "existing subjects keep their citations") {
		t.Errorf("removal flash body missing; body: %s", page)
	}
}

// --- helpers ---------------------------------------------------------------

// declareScope posts a raw (possibly multi-token) scope string to the paste-split
// declare endpoint.
func declareScope(t *testing.T, c *http.Client, base, raw string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds", url.Values{"scope": {raw}})
}

type zonePart struct{ name, content string }

// uploadZones posts one multipart POST /seeds/zone with N `zonefile` parts (DF-F2).
func uploadZones(t *testing.T, c *http.Client, base string, parts []zonePart) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for _, p := range parts {
		fw, err := mw.CreateFormFile("zonefile", p.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(p.content)); err != nil {
			t.Fatal(err)
		}
	}
	mw.Close()
	req, err := http.NewRequest(http.MethodPost, base+"/seeds/zone", &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	return resp
}

func intStr(id int64) string {
	return strconv.FormatInt(id, 10)
}

// followString GETs a URL and returns the body — used to read the destination page a
// PRG toast flash lands on.
func followString(t *testing.T, c *http.Client, url string) string {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return body(t, resp)
}
