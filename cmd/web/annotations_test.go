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

func annotate(t *testing.T, c *http.Client, base, subject, signal, reason string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/annotations", url.Values{
		"subject": {subject}, "signal": {signal}, "reason": {reason},
	})
}

func withdrawAnno(t *testing.T, c *http.Client, base string, id int64) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/annotations/withdraw", url.Values{"id": {strconv.FormatInt(id, 10)}})
}

func lameName(t *testing.T, f *fakeStore, name string) {
	t.Helper()
	f.addClassResolution(t, name, "internet", obsClock, `{"outcome":"Gap"}`)
	f.addDNSRecord(t, name, "NS", obsClock, `{"rrs":[],"delegation":{"lame":true}}`)
}

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

	annos, _ := f.ListAnnotations(t.Context())
	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if !strings.Contains(annotatedPage, "lame.example.com") {
		t.Errorf("declared annotation not rendered on the Annotated tab; body: %s", annotatedPage)
	}
	drawer := getBody(t, ac, base+"/signals?tab=annotated&view="+strconv.FormatInt(annos[0].ID, 10), http.StatusOK)
	if !strings.Contains(drawer, "OPS-1") || !strings.Contains(drawer, "lame-delegation") {
		t.Errorf("annotation drawer missing its reason or rule; body: %s", drawer)
	}
	for _, banned := range []string{`name="author"`, `name="status"`, `name="expiry"`, `name="expires"`, ">Declared by<", ">Status<", ">Expiry<"} {
		if strings.Contains(drawer, banned) {
			t.Errorf("Signals annotation surface carries %q; an operator dial has no author/status/expiry field", banned)
		}
	}
}

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
	resp = withdrawAnno(t, ac, base, got[0].ID)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("idempotent withdraw: status = %d, want 303", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAnnotatedFiredSignalStaysOpen(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	before := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(before, "lame.example.com") {
		t.Fatalf("unannotated: the fired signal should list as an open row; body: %s", before)
	}

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "accepted, being retired").Body.Close()

	after := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(after, "lame.example.com") {
		t.Errorf("annotated signal dropped from the Open tab; an annotation moves no number; body: %s", after)
	}
	if strings.Contains(after, "carries an annotation right now") {
		t.Errorf("census prose rendered; that grouping has left the screen")
	}
	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if !strings.Contains(annotatedPage, "lame.example.com") {
		t.Errorf("annotated signal not listed on the Annotated tab; body: %s", annotatedPage)
	}
}

func TestPartiallyAnnotatedSignalsBothStayOpen(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "one.example.com")
	lameName(t, f, "two.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "one.example.com", "lame-delegation", "accepted").Body.Close()

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

func signalsMain(body string) string {
	i := strings.Index(body, "<main")
	j := strings.LastIndex(body, "</main>")
	if i < 0 || j < 0 || j < i {
		return body
	}
	return body[i:j]
}

func TestAnnotationOnAbsentSubjectMarkedOrphan(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "ghost.example.com", "lame-delegation", "kept for when it returns").Body.Close()

	page := getBody(t, ac, base+"/signals?tab=withdrawn", http.StatusOK)
	if !strings.Contains(page, "ghost.example.com") || !strings.Contains(page, "withdrawn") {
		t.Errorf("annotation on an absent subject not surfaced as withdrawn; body: %s", page)
	}
}

func TestAnnotationDeclareGuards(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	lameName(t, f, "lame.example.com")
	base := start(t, f, "")

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
	page := refusalPage(t, ac, base, annotate(t, ac, base, "lame.example.com", "no-such-rule", "why"))
	if !strings.Contains(page, "Choose the signal") {
		t.Errorf("unknown signal not rejected; body: %s", page)
	}
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 0 {
		t.Fatalf("annotation created on an unknown signal; got %d", len(got))
	}

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "first").Body.Close()
	page = refusalPage(t, ac, base, annotate(t, ac, base, "lame.example.com", "lame-delegation", "second"))
	if !strings.Contains(page, "already carries an annotation") {
		t.Errorf("duplicate pair not rejected; body: %s", page)
	}
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 1 {
		t.Fatalf("duplicate declaration stored a second row; got %d", len(got))
	}
}

func refusalPage(t *testing.T, c *http.Client, base string, resp *http.Response) string {
	// A refusal renders nothing at the POST URL, so its message reads on the GET (ADR-0130 §1).
	t.Helper()
	return prgLanding(t, c, base, resp)
}

func annotateFrom(t *testing.T, c *http.Client, base, from, subject, signal, reason string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/annotations", url.Values{
		"subject": {subject}, "signal": {signal}, "reason": {reason}, "return": {from},
	})
}

func openSigID(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	page := getBody(t, c, base+"/signals", http.StatusOK)
	m := regexp.MustCompile(`SIG-\d+`).FindString(page)
	if m == "" {
		t.Fatalf("no SIG id on the Open tab; body: %s", page)
	}
	return m
}

func TestRefusedAnnotationRedirectsBackWithNothingInTheURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The parameters are not alphabetical, so a re-encode that sorted them shows up as a mismatch.
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
	for _, leak := range []string{"OPS-1", "Accepted", "Choose+the+signal", "Choose%20the%20signal", "error="} {
		if strings.Contains(loc, leak) {
			t.Errorf("Location %q carries %q; no field error and no typed value may ride in a URL", loc, leak)
		}
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "Choose the signal") {
		t.Errorf("landing page carries no callout; body: %s", page)
	}
	if !strings.Contains(page, typed) {
		t.Errorf("landing page dropped the typed reason; the operator must not retype it; body: %s", page)
	}

	again := getBody(t, ac, base+loc, http.StatusOK)
	if strings.Contains(again, "Choose the signal") || strings.Contains(again, typed) {
		t.Errorf("stale callout on a second load; the flash is not single-consume; body: %s", again)
	}
}

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

func TestFormFlashIsScopedToOneSignInNotTheAccount(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	laptop := login(t, base, "admin", "hunter2hunter2")
	phone := login(t, base, "admin", "hunter2hunter2")

	annotate(t, laptop, base, "lame.example.com", "no-such-rule", "why").Body.Close()

	if page := getBody(t, phone, base+"/signals", http.StatusOK); strings.Contains(page, "Choose the signal") {
		t.Errorf("a second sign-in consumed the first one's refusal; body: %s", page)
	}
	if page := getBody(t, laptop, base+"/signals", http.StatusOK); !strings.Contains(page, "Choose the signal") {
		t.Errorf("the submitting sign-in lost its own refusal; body: %s", page)
	}
}

func TestFormFlashExpiresWhenItsLandingNeverComes(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	now := fixedClock()()
	srv := newServer(f, testKey, "", func() time.Time { return now })
	srv.transcriptKey = testTranscriptKey
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	base := ts.URL

	ac := login(t, base, "admin", "hunter2hunter2")
	annotate(t, ac, base, "lame.example.com", "no-such-rule", "why").Body.Close()

	if !srv.formFlash.pending(now) {
		t.Fatal("precondition: the refusal was not stashed")
	}

	now = now.Add(formFlashTTL + time.Second)
	if page := getBody(t, ac, base+"/signals", http.StatusOK); strings.Contains(page, "Choose the signal") {
		t.Errorf("an expired refusal still rendered; body: %s", page)
	}
	if srv.formFlash.pending(now) {
		t.Error("an expired entry survived the prune and holds the pending gate open")
	}
}
