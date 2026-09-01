package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// annotate declares an Annotation on one (subject, signal-name) pair.
func annotate(t *testing.T, c *http.Client, base, subject, signal, reason string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/annotations", url.Values{
		"subject": {subject}, "signal": {signal}, "reason": {reason},
	})
}

// withdrawAnno withdraws the annotation with the given id.
func withdrawAnno(t *testing.T, c *http.Client, base string, id int64) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/annotations/withdraw", url.Values{"id": {strconv.FormatInt(id, 10)}})
}

// lameName wires a name so lame-delegation fires on it: composed Lame (an
// all-refusing delegation over a Gap resolution), which the signals_test fixture
// uses too.
func lameName(t *testing.T, f *fakeStore, name string) {
	t.Helper()
	f.addClassResolution(t, name, "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, name, "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)
}

// AC1: an admin declares an Annotation on one pair, and the surface carries no
// status, expiry or author — only the pair, the reason (read in the Drawer) and the
// declared instant.
func TestDeclareAnnotation(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := annotate(t, ac, base, "lame.example.com", "lame-delegation", "Accepted: the delegation is being retired under OPS-1.")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("declare annotation: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	resp.Body.Close()

	if got, _ := f.ListAnnotations(t.Context()); len(got) != 1 {
		t.Fatalf("annotations after declare = %d, want 1", len(got))
	} else {
		a := got[0]
		if a.SubjectKey != "lame.example.com" || a.SignalName != "lame-delegation" {
			t.Fatalf("annotation keyed on wrong pair: %q / %q", a.SubjectKey, a.SignalName)
		}
		if !a.DeclaredAt.Valid {
			t.Errorf("annotation carries no declared instant; it is the one Declared term that must")
		}
	}

	// The accepted subject renders on the Annotated tab, and its reason + rule read
	// in the row Drawer.
	annos, _ := f.ListAnnotations(t.Context())
	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if !strings.Contains(annotatedPage, "lame.example.com") {
		t.Errorf("declared annotation not rendered on the Annotated tab; body: %s", annotatedPage)
	}
	drawer := getBody(t, ac, base+"/signals?tab=annotated&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	if !strings.Contains(drawer, "OPS-1") || !strings.Contains(drawer, "lame-delegation") {
		t.Errorf("annotation drawer missing its reason or rule; body: %s", drawer)
	}
	// No author, status or expiry field exists on the surface: no such form input
	// and no author column header.
	for _, banned := range []string{`name="author"`, `name="status"`, `name="expiry"`, `name="expires"`, ">Declared by<", ">Status<", ">Expiry<"} {
		if strings.Contains(drawer, banned) {
			t.Errorf("Signals annotation surface carries %q; an operator dial has no author/status/expiry field", banned)
		}
	}
}

// AC2: withdrawing an Annotation is a plain state change — the row is gone and no
// Message is minted (there is no message store in this sequence; the guarantee is
// structural — declaring and withdrawing mint no cause anywhere).
func TestWithdrawAnnotationIsPlainStateChange(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "temporary").Body.Close()
	got, _ := f.ListAnnotations(t.Context())
	if len(got) != 1 {
		t.Fatalf("precondition: annotations = %d, want 1", len(got))
	}

	resp := withdrawAnno(t, ac, base, got[0].ID)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("withdraw: status = %d, want 303", resp.StatusCode)
	}
	resp.Body.Close()

	if after, _ := f.ListAnnotations(t.Context()); len(after) != 0 {
		t.Fatalf("annotations after withdraw = %d, want 0", len(after))
	}
	// Idempotent: withdrawing an already-gone row is not an error.
	resp = withdrawAnno(t, ac, base, got[0].ID)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("idempotent withdraw: status = %d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
}

// AC3: an annotated fired signal is still open — an annotation moves the message,
// never the number. The fired instance stays on the Open tab and also lists on the
// Annotated tab; there is no census prose (that grouping has left the screen).
func TestAnnotatedFiredSignalStaysOpen(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// lame-delegation's whole fired census is one name.
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	before := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(before, "lame.example.com") {
		t.Fatalf("unannotated: the fired signal should list as an open row; body: %s", before)
	}

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "accepted, being retired").Body.Close()

	after := getBody(t, ac, base+"/signals", http.StatusOK)
	// Still open — the pair is still counted under fired.
	if !strings.Contains(after, "lame.example.com") {
		t.Errorf("annotated signal dropped from the Open tab; an annotation moves no number; body: %s", after)
	}
	// No census prose.
	if strings.Contains(after, "carries an annotation right now") {
		t.Errorf("census prose rendered; that grouping has left the screen")
	}
	// It also lists on the Annotated tab.
	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if !strings.Contains(annotatedPage, "lame.example.com") {
		t.Errorf("annotated signal not listed on the Annotated tab; body: %s", annotatedPage)
	}
}

// AC4: a partially-annotated set keeps both fired signals open; only the accepted
// one lists on the Annotated tab.
func TestPartiallyAnnotatedSignalsBothStayOpen(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// Two names fire on lame-delegation; only one is accepted.
	lameName(t, f, "one.example.com")
	lameName(t, f, "two.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "one.example.com", "lame-delegation", "accepted").Body.Close()

	// Scope assertions to the screen's <main> region: the shell's command
	// palette lists current Names (P1.5), so a page-wide substring check would
	// match a name in the palette chrome, not the signals table.
	open := signalsMain(getBody(t, ac, base+"/signals", http.StatusOK))
	for _, name := range []string{"one.example.com", "two.example.com"} {
		if !strings.Contains(open, name) {
			t.Errorf("Open tab dropped fired signal %q; body: %s", name, open)
		}
	}
	if strings.Contains(open, "carries an annotation right now") {
		t.Errorf("census prose rendered on a partially-annotated set")
	}

	annotatedPage := signalsMain(getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK))
	if !strings.Contains(annotatedPage, "one.example.com") {
		t.Errorf("Annotated tab missing the accepted signal; body: %s", annotatedPage)
	}
	if strings.Contains(annotatedPage, "two.example.com") {
		t.Errorf("Annotated tab wrongly lists the un-accepted signal")
	}
}

// signalsMain returns the screen's <main> region, excluding shell chrome (nav,
// command palette, footer). The palette lists current Names (P1.5), so a
// page-wide substring match would find a name outside the signals table.
func signalsMain(body string) string {
	i := strings.Index(body, "<main")
	j := strings.LastIndex(body, "</main>")
	if i < 0 || j < 0 || j < i {
		return body
	}
	return body[i:j]
}

// A declaration naming a subject that is in no current population of the rule is an
// orphan on read — it surfaces on the Withdrawn tab as withdrawn by the world, and
// nowhere else (ADR-0092).
func TestAnnotationOnAbsentSubjectMarkedOrphan(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A pair whose subject the rule never censuses.
	annotate(t, ac, base, "ghost.example.com", "lame-delegation", "kept for when it returns").Body.Close()

	page := getBody(t, ac, base+"/signals?tab=withdrawn", http.StatusOK)
	if !strings.Contains(page, "ghost.example.com") || !strings.Contains(page, "withdrawn") {
		t.Errorf("annotation on an absent subject not surfaced as withdrawn; body: %s", page)
	}
}

// Declaring is admin-only, and the shipped rule set bounds what may be named.
func TestAnnotationDeclareGuards(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	lameName(t, f, "lame.example.com")
	base := start(t, f, "")

	// A viewer may see the page but not declare.
	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := annotate(t, vc, base, "lame.example.com", "lame-delegation", "nope")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer declare: status = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 0 {
		t.Fatalf("viewer declared an annotation; got %d", len(got))
	}

	ac := login(t, base, "admin", "hunter2hunter2")
	// An unknown signal is refused: accepting a firing on a rule that does not
	// exist would be an acceptance with no reader. Since ADR-0130 §1 (#972) the
	// refusal is a redirect, so the message reads on the page it lands on.
	page := refusalPage(t, ac, base, annotate(t, ac, base, "lame.example.com", "no-such-rule", "why"))
	if !strings.Contains(page, "Choose the signal") {
		t.Errorf("unknown signal not rejected; body: %s", page)
	}
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 0 {
		t.Fatalf("annotation created on an unknown signal; got %d", len(got))
	}

	// A duplicate pair is rejected — an Annotation cannot be edited.
	annotate(t, ac, base, "lame.example.com", "lame-delegation", "first").Body.Close()
	page = refusalPage(t, ac, base, annotate(t, ac, base, "lame.example.com", "lame-delegation", "second"))
	if !strings.Contains(page, "already carries an annotation") {
		t.Errorf("duplicate pair not rejected; body: %s", page)
	}
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 1 {
		t.Fatalf("duplicate declaration stored a second row; got %d", len(got))
	}
}

// refusalPage follows a refused mutation's 303 and returns the page it lands on.
// ADR-0130 §1 (#972): a refusal renders nothing at the POST URL. It stashes its
// message in the session form flash and redirects, so every assertion about the
// message belongs on the GET that follows.
func refusalPage(t *testing.T, c *http.Client, base string, resp *http.Response) string {
	t.Helper()
	return prgLanding(t, c, base, resp)
}

// annotateFrom declares an Annotation the way the drawer's form does: carrying the
// `return` field that names the exact URL the operator submitted from (backurl.go).
func annotateFrom(t *testing.T, c *http.Client, base, from, subject, signal, reason string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/annotations", url.Values{
		"subject": {subject}, "signal": {signal}, "reason": {reason}, "return": {from},
	})
}

// openSigID reads the SIG id of the one fired instance on the Open tab. It is the
// row Drawer's ViewKey, so `/signals?…&view=<id>` re-opens that row's declare form.
func openSigID(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	page := getBody(t, c, base+"/signals", http.StatusOK)
	m := regexp.MustCompile(`SIG-\d+`).FindString(page)
	if m == "" {
		t.Fatalf("no SIG id on the Open tab; body: %s", page)
	}
	return m
}

// ADR-0130 §1 (#972): a REFUSED declaration is a post-redirect-get, exactly like an
// accepted one. It renders nothing at the POST URL, it answers 303 to the URL the
// form was submitted from, and it carries its message in a server-side flash rather
// than in the URL. That is what fixes failure class A — the landing is an ordinary
// navigation to the same URL, indistinguishable from a success, so the scroll key the
// shell stashed on submit hits and the operator keeps their place.
func TestRefusedAnnotationRedirectsBackWithNothingInTheURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The submitting URL is the shape the carrier exists to preserve: a tab, a sort,
	// and the row's drawer open. Its parameters are deliberately NOT alphabetical, so
	// a re-encode that sorted them would show up as a mismatch below.
	from := "/signals?tab=open&sort=id&dir=desc&view=" + openSigID(t, ac, base)

	const typed = "Accepted under OPS-1 until the delegation is retired."
	resp := annotateFrom(t, ac, base, from, "lame.example.com", "no-such-rule", typed)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("refused declaration: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	loc := resp.Header.Get("Location")
	if loc != from {
		t.Fatalf("refused declaration: Location = %q, want the submitting URL %q byte for byte", loc, from)
	}
	// The payload never enters the URL. A URL is written to the access log and kept in
	// browser history, and the typed reason can be sensitive.
	for _, leak := range []string{"OPS-1", "Accepted", "Choose+the+signal", "Choose%20the%20signal", "error="} {
		if strings.Contains(loc, leak) {
			t.Errorf("Location %q carries %q; no field error and no typed value may ride in a URL", loc, leak)
		}
	}

	// The message and the typed reason both come back on the page it lands on, so an
	// operator with JavaScript off sees the refusal inline exactly as before.
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "Choose the signal") {
		t.Errorf("landing page carries no callout; body: %s", page)
	}
	if !strings.Contains(page, typed) {
		t.Errorf("landing page dropped the typed reason; the operator must not retype it; body: %s", page)
	}

	// Single consume: a second load of the SAME URL shows no stale callout. This is
	// what keeps the scan-running auto-refresh from re-showing a spent refusal.
	again := getBody(t, ac, base+loc, http.StatusOK)
	if strings.Contains(again, "Choose the signal") || strings.Contains(again, typed) {
		t.Errorf("stale callout on a second load; the flash is not single-consume; body: %s", again)
	}
}

// The withdraw not-found path uses the same carrier: it stashes and 303s rather than
// re-rendering at the POST URL.
func TestWithdrawNotFoundRedirectsBack(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	from := "/signals?tab=annotated&sort=asset"
	resp := postForm(t, ac, base+"/annotations/withdraw", url.Values{
		"id": {"not-a-number"}, "return": {from},
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("withdraw not-found: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	if loc := resp.Header.Get("Location"); loc != from {
		t.Fatalf("withdraw not-found: Location = %q, want %q", loc, from)
	}
	if page := getBody(t, ac, base+from, http.StatusOK); !strings.Contains(page, "could not be found") {
		t.Errorf("withdraw not-found message did not land on the GET; body: %s", page)
	}
}

// The flash is keyed by SESSION, not by account. A session is one SIGN-IN, so a
// refusal on the operator's laptop must not be consumed by their phone, or by a
// private window, signed in as the same account. An account key would have made that
// the behaviour across every device at once.
//
// Note what this does NOT claim. Each login below gets its own cookie jar, so these
// are two sign-ins, not two tabs. Tabs of one browser share the session cookie, resolve
// to one session row, and so share one slot — see the formFlashStore doc comment.
func TestFormFlashIsScopedToOneSignInNotTheAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	laptop := login(t, base, "admin", "hunter2hunter2")
	phone := login(t, base, "admin", "hunter2hunter2")

	annotate(t, laptop, base, "lame.example.com", "no-such-rule", "why").Body.Close()

	// The other sign-in loads /signals and must see nothing.
	if page := getBody(t, phone, base+"/signals", http.StatusOK); strings.Contains(page, "Choose the signal") {
		t.Errorf("a second sign-in consumed the first one's refusal; body: %s", page)
	}
	// The submitting sign-in still has its message waiting.
	if page := getBody(t, laptop, base+"/signals", http.StatusOK); !strings.Contains(page, "Choose the signal") {
		t.Errorf("the submitting sign-in lost its own refusal; body: %s", page)
	}
}

// A flash whose landing GET never arrives — the operator closed the tab as they
// submitted, or the connection dropped — expires rather than waiting to ambush their
// next visit to the surface. Without the TTL that stale callout would fire on a page
// the operator did nothing to provoke, and would hold the pending() gate open for
// every other signed-in operator's GET in the meantime.
func TestFormFlashExpiresWhenItsLandingNeverComes(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	// A clock the test advances, so the TTL is exercised without a sleep.
	now := fixedClock()()
	srv := newServer(f, testKey, "", func() time.Time { return now })
	srv.transcriptKey = testTranscriptKey
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	base := ts.URL

	ac := login(t, base, "admin", "hunter2hunter2")
	annotate(t, ac, base, "lame.example.com", "no-such-rule", "why").Body.Close()

	// Still collectable inside the window, and the store still holds it.
	if !srv.formFlash.pending(now) {
		t.Fatal("precondition: the refusal was not stashed")
	}

	now = now.Add(formFlashTTL + time.Second)
	if page := getBody(t, ac, base+"/signals", http.StatusOK); strings.Contains(page, "Choose the signal") {
		t.Errorf("an expired refusal still rendered; body: %s", page)
	}
	// The prune ran, so the gate is closed again and no later GET pays the session read.
	if srv.formFlash.pending(now) {
		t.Error("an expired entry survived the prune and holds the pending gate open")
	}
}
