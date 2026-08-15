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

// AC1: an admin declares an Annotation on one pair, and the row carries no
// status, expiry or author — only the pair, the reason and the declared instant.
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

	// The declaration renders in the management list, and the surface carries no
	// author, status or expiry — every operator dial is unattributed (ADR-0073)
	// and an Annotation carries no timeline.
	page := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(page, "lame-delegation") || !strings.Contains(page, "OPS-1") {
		t.Errorf("declared annotation not rendered on Signals; body: %s", page)
	}
	// No author, status or expiry field exists on the surface: no such form input
	// and no author column header.
	for _, banned := range []string{`name="author"`, `name="status"`, `name="expiry"`, `name="expires"`, ">Declared by<", ">Status<", ">Expiry<"} {
		if strings.Contains(page, banned) {
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

// AC3: a rule whose entire fired census is annotated renders as categorical
// prose, never a bare count.
func TestFullyAnnotatedFiredCensusRendersAsProse(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// lame-delegation's whole fired census is one name.
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Before annotating, the fired member is a listed, drillable count row.
	before := getBody(t, ac, base+"/signals", http.StatusOK)
	if strings.Count(before, `href="/subjects/lame.example.com"`) != 1 {
		t.Fatalf("unannotated: fired member should list its subject once; body: %s", before)
	}
	if strings.Contains(before, "carries an annotation right now") {
		t.Fatalf("prose rendered while the census is not annotated")
	}

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "accepted, being retired").Body.Close()

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	// The categorical sentence renders...
	if !strings.Contains(page, "carries an annotation right now") {
		t.Errorf("fully-annotated fired census did not render as prose; body: %s", page)
	}
	if !strings.Contains(page, "Fired · every subject accepted") {
		t.Errorf("fully-annotated fired census missing its categorical label; body: %s", page)
	}
	// ...and the fired census no longer lists the subject as a bare count: the
	// only drill link to it now is the one in the Annotations list.
	if n := strings.Count(page, `href="/subjects/lame.example.com"`); n != 1 {
		t.Errorf("fully-annotated fired census still lists its member as a count (drill links = %d, want 1); body: %s", n, page)
	}
}

// AC4: a partially-annotated census still renders the normal three-member
// breakdown — fired is not partitioned into accepted and outstanding.
func TestPartiallyAnnotatedCensusRendersNormalBreakdown(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// Two names fire on lame-delegation; only one is accepted.
	lameName(t, f, "one.example.com")
	lameName(t, f, "two.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	annotate(t, ac, base, "one.example.com", "lame-delegation", "accepted").Body.Close()

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	// No prose — the census is only partially annotated.
	if strings.Contains(page, "carries an annotation right now") {
		t.Errorf("partially-annotated census wrongly rendered as prose; body: %s", page)
	}
	// The normal three-member breakdown stands: both fired members are still
	// listed in the fired census (once there, once in the annotations list for the
	// accepted one), and the fired register is a single list, not partitioned.
	for _, name := range []string{"one.example.com", "two.example.com"} {
		if !strings.Contains(page, `href="/subjects/`+name+`"`) {
			t.Errorf("partial census dropped fired member %q; body: %s", name, page)
		}
	}
	if strings.Count(page, `href="/subjects/two.example.com"`) != 1 {
		t.Errorf("the unannotated fired member should appear once, in the fired census")
	}
	// two.example.com is only in the fired census; one.example.com is in both the
	// fired census and the annotations list.
	if strings.Count(page, `href="/subjects/one.example.com"`) != 2 {
		t.Errorf("the accepted member should appear in both the fired census and the annotations list")
	}
}

// A declaration naming a subject that is in no current population of the rule is
// marked as orphan on read — it names a withdrawn or never-measured subject and
// matches nothing right now (ADR-0092).
func TestAnnotationOnAbsentSubjectMarkedOrphan(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	lameName(t, f, "lame.example.com")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// A pair whose subject the rule never censuses.
	annotate(t, ac, base, "ghost.example.com", "lame-delegation", "kept for when it returns").Body.Close()

	page := getBody(t, ac, base+"/signals", http.StatusOK)
	if !strings.Contains(page, "names no current member") {
		t.Errorf("annotation on an absent subject not marked orphan; body: %s", page)
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
