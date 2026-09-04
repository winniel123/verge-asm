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

func TestScopeAndOnboardingShareTokenizer(t *testing.T) {
	const raw = "a.com, b.com\n c.com\td.com,,e.com"
	want := []string{"a.com", "b.com", "c.com", "d.com", "e.com"}

	if got := parseSeedTokens(raw); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSeedTokens(%q) = %v, want %v", raw, got, want)
	}

	q := url.Values{"step": {"0"}, "seedsadd": {raw}}
	r := httptest.NewRequest(http.MethodGet, "/onboarding?"+q.Encode(), nil)
	if got := readOnboardView(r).Seeds; !reflect.DeepEqual(got, want) {
		t.Fatalf("onboarding tokenization = %v, want %v (must match parseSeedTokens)", got, want)
	}

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

func TestDeclareBulkPasteMixedSuccessRefusalDuplicate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declareScope(t, ac, base, "good1.com good2.com good1.com 10.0.0.0/21"))

	if len(f.seeds) != 2 {
		t.Fatalf("committed %d seeds, want 2 (good1.com, good2.com); seeds=%+v", len(f.seeds), f.seeds)
	}
	if !strings.Contains(got, "2 scopes declared") {
		t.Errorf("success flash missing; body: %s", got)
	}
	if !strings.Contains(got, "2 refused — see the callouts") {
		t.Errorf("flash refusal description missing; body: %s", got)
	}
	if !strings.Contains(got, "already declared") {
		t.Errorf("duplicate token not refused with 'already declared'; body: %s", got)
	}
	if !strings.Contains(got, "the cap is 1,024 per scope") {
		t.Errorf("over-cap refusal reason missing; body: %s", got)
	}
	if !strings.Contains(got, "10.0.0.0/22") {
		t.Errorf("over-cap reachable set not named; body: %s", got)
	}
}

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
	if !strings.Contains(got, "registrable domain example.com") {
		t.Errorf("subdomain refusal reason missing; body: %s", got)
	}
	if !strings.Contains(got, "the cap is 1,024 per scope") {
		t.Errorf("over-cap refusal reason missing; body: %s", got)
	}
}

func TestZoneBulkUploadMixedAcceptRefuse(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()

	first := "$ORIGIN example.com.\n@ IN A 203.0.113.10\n"
	second := "$ORIGIN example.com.\n@ IN A 203.0.113.20\n"
	foreign := "$ORIGIN notmine.example.\n@ IN A 203.0.113.10\n"
	garbage := "this is not a zone file at all\n"

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
	if !strings.Contains(got, "outside every declared name scope") {
		t.Errorf("foreign-apex refusal reason missing; body: %s", got)
	}
	if !strings.Contains(got, "not a zone file") {
		t.Errorf("unparseable file not refused with 'not a zone file'; body: %s", got)
	}
	rows, _ := f.ListZoneFileStatus(t.Context())
	if len(rows) != 1 {
		t.Fatalf("zone status rows = %d, want 1 (one apex)", len(rows))
	}
	if !strings.Contains(f.zoneFiles[len(f.zoneFiles)-1].content, "203.0.113.20") {
		t.Errorf("later file is not the current supply; files=%+v", f.zoneFiles)
	}
}

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
	// An empty zone row reads "no zone file supplied", so only the flash container is decisive.
	if strings.Contains(got, `<div class="sh-toasts"`) {
		t.Errorf("all-refused upload should fire no flash; body: %s", got)
	}
}

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
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	page := followString(t, ac, base+loc)
	if !strings.Contains(page, "Scope removed") {
		t.Errorf("removal flash title missing; body: %s", page)
	}
	if !strings.Contains(page, "leave the estate on the next completed job") {
		t.Errorf("removal flash body missing; body: %s", page)
	}
	if len(f.seedWithdrawals) != 1 {
		t.Fatalf("a name Seed records its withdrawal, got %+v", f.seedWithdrawals)
	}
	if got := f.seedWithdrawals[0].Kind; got != "name" {
		t.Errorf("the tombstone records the kind it withdrew, got %q", got)
	}
	if got := f.seedWithdrawals[0].NameDomain.String; got != "example.com" {
		t.Errorf("the tombstone names the withdrawn domain, got %q", got)
	}
	if f.seedWithdrawals[0].AddressCidr != nil {
		t.Errorf("a name tombstone carries no CIDR, got %v", f.seedWithdrawals[0].AddressCidr)
	}
	if strings.Contains(page, "existing subjects keep their citations") {
		t.Errorf("the flash must stop promising the opposite of the act; body: %s", page)
	}
}

func TestWithdrawAddressSeedRecordsItsTombstone(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "address", "198.51.100.0/24").Body.Close()
	id := f.seeds[0].ID

	resp := postForm(t, ac, base+"/seeds/delete", url.Values{"id": {intStr(id)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("withdraw: status=%d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()

	if len(f.seedWithdrawals) != 1 {
		t.Fatalf("the act must record its mover, got %+v", f.seedWithdrawals)
	}
	if f.seedWithdrawals[0].AddressCidr == nil {
		t.Fatalf("an address tombstone carries its CIDR, got %+v", f.seedWithdrawals[0])
	}
	if got := f.seedWithdrawals[0].AddressCidr.String(); got != "198.51.100.0/24" {
		t.Errorf("the tombstone names the withdrawn CIDR, got %q", got)
	}

	page := followString(t, ac, base+loc)
	if !strings.Contains(page, "leave the estate on the next completed job") {
		t.Errorf("the flash must state the withdrawal it now performs; body: %s", page)
	}
	if strings.Contains(page, "existing subjects keep their citations") {
		t.Errorf("the flash must stop promising the opposite of the act; body: %s", page)
	}
}

func declareScope(t *testing.T, c *http.Client, base, raw string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds", url.Values{"scope": {raw}})
}

type zonePart struct{ name, content string }

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

func followString(t *testing.T, c *http.Client, url string) string {
	t.Helper()
	resp, err := c.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return body(t, resp)
}
