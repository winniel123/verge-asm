package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The submitting-URL carrier (backurl.go, ADR-0130 §3, ticket #971). These tests hold
// three promises: the field carries the page's own URL, the guard admits only a
// same-origin relative path this router serves, and an annotation act lands the
// operator back on the exact list they acted from.

// backSrv builds a server whose route table is populated — handler() sets s.routes as
// it finishes the mux, and resolveBack answers "no" without one.
func backSrv(t *testing.T) *server {
	t.Helper()
	srv := newServer(newFakeStore(), testKey, "", fixedClock())
	_ = srv.handler()
	return srv
}

// backPost builds a POST carrying one submitting-URL field, as the "backfield" partial
// emits it.
func backPost(value string) *http.Request {
	form := url.Values{}
	if value != "" {
		form.Set(backField, value)
	}
	r := httptest.NewRequest(http.MethodPost, "/annotations", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// AC: the field carries the pathname plus the query, and nothing else. The single-
// consume `toast` receipt is dropped, so returning to the URL cannot re-fire a toast
// the operator has already seen.
func TestBackURLCarriesPathAndQuery(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare path", "/signals", "/signals"},
		{"path and query", "/signals?tab=open&sev=High", "/signals?tab=open&sev=High"},
		{"root", "/", "/"},
		{"toast receipt dropped", "/signals?tab=open&toast=abc", "/signals?tab=open"},
		{"toast dropped from the front", "/signals?toast=abc&tab=open", "/signals?tab=open"},
		{"toast dropped from the middle", "/signals?tab=open&toast=abc&q=lame", "/signals?tab=open&q=lame"},
		{"a percent-encoded toast key is dropped too", "/signals?tab=open&%74oast=abc", "/signals?tab=open"},
		{"toast-only query leaves a bare path", "/signals?toast=abc", "/signals"},
		{"a value that merely contains toast is kept", "/signals?q=toaster", "/signals?q=toaster"},
		{"escaped path segment kept", "/asset/host.example.com?tab=x", "/asset/host.example.com?tab=x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := backURL(httptest.NewRequest(http.MethodGet, tc.in, nil))
			if got != tc.want {
				t.Errorf("backURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
	if got := backURL(nil); got != "" {
		t.Errorf("backURL(nil) = %q, want the empty string", got)
	}
}

// The scroll key ticket #970 set is a raw string compare on `location.pathname +
// location.search`, so the carrier must not re-order the query. Rebuilding it through
// url.Values.Encode() sorts it alphabetically and the stash then misses on the
// landing — the class-C/E failure this map exists to close. These are the two orders
// /signals actually emits: the filter form's DOM order and the severity link's order.
// Neither is alphabetical.
func TestBackURLPreservesQueryOrder(t *testing.T) {
	for _, in := range []string{
		"/signals?tab=open&sev=High&sort=asset&dir=desc",
		"/signals?tab=open&q=lame&sev=High&sort=asset&dir=desc",
		"/signals?tab=open&sev=High&sort=asset&dir=desc&page=2",
	} {
		got := backURL(httptest.NewRequest(http.MethodGet, in, nil))
		if got != in {
			t.Errorf("backURL(%q) = %q; the query must survive in its own order, or the scroll key misses", in, got)
		}
	}
}

// AC: validation accepts only a same-origin relative path the router serves a GET at.
func TestResolveBackAcceptsRoutedRelativePaths(t *testing.T) {
	s := backSrv(t)
	const fallback = "/signals"
	for _, want := range []string{
		"/",
		"/signals",
		"/signals?tab=annotated&sev=High&page=3",
		"/settings?tab=scans",
		"/asset/host.example.com",
		"/scope",
		"/inventory?q=example",
	} {
		if got := s.resolveBack(backPost(want), fallback); got != want {
			t.Errorf("resolveBack(%q) = %q, want it accepted unchanged", want, got)
		}
	}
}

// AC: validation rejects an absolute URL, `//evil.example/x`, a backslash variant, and
// any unrouted path. Every rejection falls back to the caller's own path, so a handler
// is never left without a destination.
func TestResolveBackRejectsAndFallsBack(t *testing.T) {
	s := backSrv(t)
	const fallback = "/signals"
	cases := []struct {
		name string
		in   string
	}{
		{"absolute https", "https://evil.example/x"},
		{"absolute http", "http://evil.example/x"},
		{"scheme-relative", "//evil.example/x"},
		{"scheme-relative with userinfo", "//user@evil.example/x"},
		{"backslash after slash", `/\evil.example`},
		{"double backslash", `\\evil.example\x`},
		{"backslash inside a routed path", `/signals\@evil.example`},
		{"tab-obfuscated scheme", "http:/\t/evil.example/x"},
		{"unrouted path", "/nope/not-a-route"},
		{"a POST-only route serves no GET", "/annotations"},
		{"traversal is not folded", "/signals/../login"},
		{"relative, no leading slash", "signals?tab=open"},
		{"fragment", "/signals#row-3"},
		{"empty", ""},
		{"whitespace only", "   "},
		{"a catch-all match is not a route", "/definitely-not-a-page"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.resolveBack(backPost(tc.in), fallback); got != fallback {
				t.Errorf("resolveBack(%q) = %q, want the fallback %q", tc.in, got, fallback)
			}
		})
	}
}

// A hand-crafted field must not plant a toast on the landing page. decodeToasts reads
// only the FIRST `toast`, so a planted receipt would beat the real one the act fires
// and put an operator-chosen system message on the screen. backURL never emits a
// `toast`, so a legitimate value passes through untouched.
func TestResolveBackStripsAPlantedToast(t *testing.T) {
	s := backSrv(t)
	planted := "eyJ0b25lIjoiZGFuZ2VyIiwidGl0bGUiOiJTZXNzaW9uIGV4cGlyZWQifQ"
	cases := []struct {
		in   string
		want string
	}{
		{"/signals?toast=" + planted, "/signals"},
		{"/signals?tab=open&toast=" + planted, "/signals?tab=open"},
		{"/signals?toast=" + planted + "&tab=open", "/signals?tab=open"},
		{"/signals?tab=open&sev=High", "/signals?tab=open&sev=High"},
	}
	for _, tc := range cases {
		if got := s.resolveBack(backPost(tc.in), "/signals"); got != tc.want {
			t.Errorf("resolveBack(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// End to end: the act's own toast is the one that lands.
	w := httptest.NewRecorder()
	s.toastRedirectBack(w, backPost("/signals?tab=open&toast="+planted), "/signals", "ok", "Annotation declared", "")
	dest := httptest.NewRequest(http.MethodGet, w.Header().Get("Location"), nil)
	got := decodeToasts(dest)
	if len(got) != 1 || got[0].Title != "Annotation declared" {
		t.Errorf("a planted toast survived the guard: %+v (Location %q)", got, w.Header().Get("Location"))
	}
}

// A server whose handler() never ran has no route table, and the guard answers no
// rather than trusting a path nothing checked.
func TestResolveBackWithoutARouteTableFallsBack(t *testing.T) {
	s := newServer(newFakeStore(), testKey, "", fixedClock())
	if got := s.resolveBack(backPost("/signals?tab=open"), "/signals"); got != "/signals" {
		t.Errorf("resolveBack with no route table = %q, want the fallback", got)
	}
	if s.routeServesGET("/signals") {
		t.Error("routeServesGET said yes with no route table; it must fail closed")
	}
}

// The catch-all `GET /` matches every path, but its handler answers 404 for anything
// but the root. The guard must read it that way, or every unrouted path passes.
func TestRouteServesGETReadsTheCatchAllHonestly(t *testing.T) {
	s := backSrv(t)
	if !s.routeServesGET("/") {
		t.Error("routeServesGET(/) = false; the root is served")
	}
	if s.routeServesGET("/definitely-not-a-page") {
		t.Error("routeServesGET matched the catch-all for an unrouted path")
	}
	if !s.routeServesGET("/asset/host.example.com") {
		t.Error("routeServesGET(/asset/{key}) = false; a wildcard route is served")
	}
	if s.routeServesGET("/annotations") {
		t.Error("routeServesGET(/annotations) = true; that route serves POST only")
	}
}

// AC: `Referer` is not read anywhere in the new path. A request that carries only a
// Referer, and no field, falls back — the header is never a destination.
func TestResolveBackIgnoresReferer(t *testing.T) {
	s := backSrv(t)
	r := backPost("")
	r.Header.Set("Referer", "http://localhost/signals?tab=open&sev=High")
	if got := s.resolveBack(r, "/signals"); got != "/signals" {
		t.Errorf("resolveBack read the Referer: got %q, want the fallback", got)
	}
}

// AC: the helper composes with toastRedirect and does not drop the submitting URL's
// own query. The toast is appended with `&` onto a URL that already has a query, and
// with `?` onto one that does not.
func TestToastRedirectBackKeepsTheSubmittingQuery(t *testing.T) {
	s := backSrv(t)
	cases := []struct {
		name       string
		in         string
		wantPrefix string
	}{
		{"query kept, toast appended with &", "/signals?tab=open&sev=High", "/signals?tab=open&sev=High&toast="},
		{"no query, toast appended with ?", "/scope", "/scope?toast="},
		{"rejected value falls back, toast still fires", "https://evil.example/x", "/signals?toast="},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.toastRedirectBack(w, backPost(tc.in), "/signals", "ok", "Annotation declared", "")
			if w.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303", w.Code)
			}
			loc := w.Header().Get("Location")
			if !strings.HasPrefix(loc, tc.wantPrefix) {
				t.Fatalf("Location = %q, want the prefix %q", loc, tc.wantPrefix)
			}
			// The toast must be the destination's own, decodable where it lands.
			dest := httptest.NewRequest(http.MethodGet, loc, nil)
			if got := decodeToasts(dest); len(got) != 1 || got[0].Title != "Annotation declared" {
				t.Errorf("the toast did not survive the compose: %+v", got)
			}
		})
	}
}

// AC: the shared partial emits the hidden field on a mutating form, carrying the
// page's own pathname plus query. Both /signals annotation forms opt in.
func TestSignalsFormsCarryTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The declare form, opened on an open row's Drawer under a filtered view.
	const view = "/signals?tab=open&q=lame"
	list := getBody(t, ac, base+view, http.StatusOK)
	key := firstViewKey(t, list)
	drawer := getBody(t, ac, base+view+"&view="+url.QueryEscape(key), http.StatusOK)
	// The query keeps its own order, and html/template escapes the `&` in an attribute.
	const wantField = `name="return" value="/signals?tab=open&amp;q=lame&amp;view=`
	if !strings.Contains(drawer, wantField) {
		t.Errorf("the declare form carries no submitting-URL field starting %q; body: %s", wantField, drawer)
	}

	// AC: with JavaScript off both acts still work, so neither submit button may ship
	// disabled. The declare button's empty-reason disable is applied by the shell script
	// on load; the markup itself must be submittable.
	if strings.Contains(drawer, "disabled data-anno-submit") || strings.Contains(drawer, "data-anno-submit disabled") {
		t.Error("the declare submit button ships disabled; the act is then unreachable with JavaScript off")
	}

	// The withdraw form, on the Annotated tab's Drawer.
	annotate(t, ac, base, "lame.example.com", "lame-delegation", "Accepted under OPS-1.").Body.Close()
	annos, _ := f.ListAnnotations(t.Context())
	anno := getBody(t, ac, base+"/signals?tab=annotated&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	if !strings.Contains(anno, `action="/annotations/withdraw"`) || !strings.Contains(anno, `name="return" value="/signals?tab=annotated`) {
		t.Errorf("the withdraw form carries no submitting-URL field; body: %s", anno)
	}
}

// AC: declaring and withdrawing an annotation from `/signals?…` lands the operator
// back at that exact URL, with the filter intact. AC: with JavaScript off both acts
// still work — nothing here runs a script; the field is markup and the answer is a
// plain 303.
func TestAnnotationActsLandBackOnTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/signals?tab=open&sev=High&sort=asset&dir=desc&page=2"
	resp := postForm(t, ac, base+"/annotations", url.Values{
		"subject": {"lame.example.com"}, "signal": {"lame-delegation"},
		"reason":  {"Accepted: the delegation is being retired under OPS-1."},
		backField: {from},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("declare: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	if got := resp.Header.Get("Location"); got != from {
		t.Errorf("declare landed at %q, want the submitting URL %q", got, from)
	}

	annos, _ := f.ListAnnotations(t.Context())
	if len(annos) != 1 {
		t.Fatalf("annotations after declare = %d, want 1", len(annos))
	}
	const back = "/signals?tab=annotated&q=lame"
	wresp := postForm(t, ac, base+"/annotations/withdraw", url.Values{
		"id":      {strconv.FormatInt(annos[0].ID, 10)},
		backField: {back},
	})
	defer wresp.Body.Close()
	if wresp.StatusCode != http.StatusSeeOther {
		t.Fatalf("withdraw: status = %d, want 303 (body: %s)", wresp.StatusCode, body(t, wresp))
	}
	if got := wresp.Header.Get("Location"); got != back {
		t.Errorf("withdraw landed at %q, want the submitting URL %q", got, back)
	}
}

// A hostile field never reaches the Location header: the act still succeeds, and the
// operator lands on the handler's own fallback rather than off-origin.
func TestAnnotationActRefusesAnOffOriginReturn(t *testing.T) {
	for _, hostile := range []string{"https://evil.example/x", "//evil.example/x", `/\evil.example`} {
		t.Run(hostile, func(t *testing.T) {
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			lameName(t, f, "lame.example.com")
			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")

			resp := postForm(t, ac, base+"/annotations", url.Values{
				"subject": {"lame.example.com"}, "signal": {"lame-delegation"},
				"reason":  {"Accepted under OPS-1."},
				backField: {hostile},
			})
			got := resp.Header.Get("Location")
			resp.Body.Close()
			if resp.StatusCode != http.StatusSeeOther {
				t.Fatalf("declare with %q: status = %d, want 303", hostile, resp.StatusCode)
			}
			if got != "/signals" {
				t.Errorf("declare with %q landed at %q, want the fallback /signals", hostile, got)
			}
			// The guard rejects the destination, never the act itself.
			if n, _ := f.ListAnnotations(t.Context()); len(n) != 1 {
				t.Errorf("declare with %q recorded %d annotations, want 1", hostile, len(n))
			}
		})
	}
}

// A report schedule act lands the operator back on the /reports list it was submitted
// from, window and all (ticket #977). /reports carries the report window as ?start=&end=
// or ?period=, so the bare "/reports" destination dropped that window on every Run now and
// every Delete.
func TestReportScheduleActsLandBackOnTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/reports?start=2026-01-01&end=2026-03-31"
	// A stale id is the idempotent path — the act answers the contract's 303 rather than a
	// 500 — so this exercises the destination without seeding a schedule row.
	resp := postForm(t, ac, base+"/reports/schedule/delete", url.Values{
		"id": {"4242"}, backField: {from},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	if got := resp.Header.Get("Location"); got != from {
		t.Errorf("delete landed at %q, want the submitting URL %q", got, from)
	}

	rresp := postForm(t, ac, base+"/reports/schedule/run", url.Values{
		"id": {"4242"}, backField: {from},
	})
	defer rresp.Body.Close()
	if got := rresp.Header.Get("Location"); got != from {
		t.Errorf("run now landed at %q, want the submitting URL %q", got, from)
	}
}

// The wizard threads the entry URL across every step, so the finishing 303 lands on the
// list the operator opened it from (ticket #977). The wizard's own step URLs are never the
// destination — finishing must LEAVE the wizard — which is why this one act reads its
// return value from the entry link rather than from the page it is submitted on.
func TestReportScheduleWizardThreadsTheEntryURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/reports?period=90d"

	// The opening GET reads the entry URL off its own query and stamps it into the form.
	wiz := getBody(t, ac, base+"/reports/schedule/new?return="+url.QueryEscape(from), http.StatusOK)
	if !strings.Contains(wiz, `name="return" value="`+from+`"`) {
		t.Fatalf("the wizard did not stamp the entry URL into its form; body: %s", wiz)
	}

	// A step advance carries it on to the next step's GET URL.
	step := postForm(t, ac, base+"/reports/schedule/new", url.Values{
		"step": {"0"}, "action": {"next"}, "name": {"Quarterly exposure"},
		"sections": {"kpis"}, backField: {from},
	})
	defer step.Body.Close()
	if loc := step.Header.Get("Location"); !strings.Contains(loc, "return="+url.QueryEscape(from)) {
		t.Errorf("the step advance dropped the entry URL; Location = %q", loc)
	}

	// The finish leaves the wizard for that list.
	done := postForm(t, ac, base+"/reports/schedule/new", url.Values{
		"step": {"3"}, "name": {"Quarterly exposure"}, "sections": {"kpis"},
		"cad": {reportDefaultCad}, "channel": {"0"}, backField: {from},
	})
	defer done.Body.Close()
	if got := done.Header.Get("Location"); got != from {
		t.Errorf("the finish landed at %q, want the entry URL %q", got, from)
	}
}

// An Inbox read act lands back on the filtered, message-open Inbox it was submitted from
// (ticket #977). These three handlers used to pick between two allowlisted homes with a
// carrier of their own (messageReturn), so a mark-read from /inbox?id=3&filter=unread
// landed on a bare /inbox — the filter dropped and the open message closed.
func TestInboxReadActsLandBackOnTheFilteredInbox(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/inbox?id=3&filter=unread"
	for _, act := range []string{"/messages/read", "/messages/unread", "/messages/read-all"} {
		resp := postForm(t, ac, base+act, url.Values{"id": {"3"}, backField: {from}})
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("%s: status = %d, want 303", act, resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != from {
			t.Errorf("%s landed at %q, want the submitting URL %q", act, got, from)
		}
	}

	// With no field the historical /messages home still answers: the viewer-readable fold
	// these acts are shared with posts none.
	resp := postForm(t, ac, base+"/messages/read", url.Values{"id": {"3"}})
	resp.Body.Close()
	if got := resp.Header.Get("Location"); got != messagesFallback {
		t.Errorf("a post with no return landed at %q, want %q", got, messagesFallback)
	}
}

// firstViewKey pulls one row's Drawer key off a rendered Signals list, so a test can
// open a Drawer without hardcoding a minted SIG id.
func firstViewKey(t *testing.T, page string) string {
	t.Helper()
	// The row href is html-escaped, so the separator before `view=` reads `&amp;`.
	m := regexp.MustCompile(`view=([^"&]+)`).FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("no row Drawer link on the Signals page; body: %s", page)
	}
	key, err := url.QueryUnescape(m[1])
	if err != nil {
		t.Fatalf("row Drawer key %q does not unescape: %v", m[1], err)
	}
	return key
}
