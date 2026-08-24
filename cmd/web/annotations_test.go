package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
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

	open := getBody(t, ac, base+"/signals", http.StatusOK)
	for _, name := range []string{"one.example.com", "two.example.com"} {
		if !strings.Contains(open, name) {
			t.Errorf("Open tab dropped fired signal %q; body: %s", name, open)
		}
	}
	if strings.Contains(open, "carries an annotation right now") {
		t.Errorf("census prose rendered on a partially-annotated set")
	}

	annotatedPage := getBody(t, ac, base+"/signals?tab=annotated", http.StatusOK)
	if !strings.Contains(annotatedPage, "one.example.com") {
		t.Errorf("Annotated tab missing the accepted signal; body: %s", annotatedPage)
	}
	if strings.Contains(annotatedPage, "two.example.com") {
		t.Errorf("Annotated tab wrongly lists the un-accepted signal")
	}
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
	// exist would be an acceptance with no reader.
	page := body(t, annotate(t, ac, base, "lame.example.com", "no-such-rule", "why"))
	if !strings.Contains(page, "Choose the signal") {
		t.Errorf("unknown signal not rejected; body: %s", page)
	}
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 0 {
		t.Fatalf("annotation created on an unknown signal; got %d", len(got))
	}

	// A duplicate pair is rejected — an Annotation cannot be edited.
	annotate(t, ac, base, "lame.example.com", "lame-delegation", "first").Body.Close()
	page = body(t, annotate(t, ac, base, "lame.example.com", "lame-delegation", "second"))
	if !strings.Contains(page, "already carries an annotation") {
		t.Errorf("duplicate pair not rejected; body: %s", page)
	}
	if got, _ := f.ListAnnotations(t.Context()); len(got) != 1 {
		t.Fatalf("duplicate declaration stored a second row; got %d", len(got))
	}
}
