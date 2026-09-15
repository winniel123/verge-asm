package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/signal"
)

const (
	signalsDidNotResolve = "The signal read did not resolve on this load."
	rulesDidNotResolve   = "The rule read did not resolve on this load."
	assetSignalsEmpty    = "No rule is firing on this asset right now."
	subjectRulesEmpty    = "No rule's predicate domain includes this subject yet."
)

func corpusReadFails(f *fakeStore) {
	// The corpus is the only reader of this facet, so no other region fails with it.
	f.tlsAcceptanceErr = errors.New("service tls acceptance read failed")
}

func assetHeaderOf(t *testing.T, page string) string {
	t.Helper()
	from := strings.Index(page, `aria-label="Breadcrumb"`)
	if from < 0 {
		t.Fatalf("asset page carries no breadcrumb, so its header cannot be isolated; body: %s", page)
	}
	hdr := page[from:]
	end := strings.Index(hdr, "</header>")
	if end < 0 {
		// An unclosed slice runs into the signals card, whose rows carry their own sevbadge.
		t.Fatalf("asset page header is not closed, so the header cannot be isolated; body: %s", page)
	}
	return hdr[:end]
}

func TestAssetDetailNamesAFailedSignalRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	// This name fires lame-delegation, so a swallow would hide a measured Medium verdict (#1951).
	lameName(t, f, "lame.example.com")
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

	if !strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("signals card does not name the failed read; body: %s", page)
	}
	if strings.Contains(page, assetSignalsEmpty) {
		t.Errorf("a failed read still asserts that no rule fires on this asset; body: %s", page)
	}
	if strings.Contains(page, "lame-delegation") {
		t.Errorf("a failed corpus read listed a rule anyway; body: %s", page)
	}
	if hdr := assetHeaderOf(t, page); strings.Contains(hdr, "var(--sev-") {
		t.Errorf("a failed read raised a header severity badge; header: %s", hdr)
	}
}

func TestAssetDetailKeepsItsSignalsEmptyStateWhenNoRuleFires(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/api.example.com", http.StatusOK)

	if !strings.Contains(page, assetSignalsEmpty) {
		t.Errorf("an asset with no firing rule lost its original empty state; body: %s", page)
	}
	if strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
}

func withdrawnNameStore(t *testing.T) *fakeStore {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved"}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)
	return f
}

func TestWithdrawnAssetNamesAFailedSignalRead(t *testing.T) {
	f := withdrawnNameStore(t)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)

	if !strings.Contains(page, "withdrawn") {
		t.Errorf("withdrawn asset lost its withdrawal notice; body: %s", page)
	}
	if !strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("signals card does not name the failed read; body: %s", page)
	}
	if strings.Contains(page, assetSignalsEmpty) {
		t.Errorf("a failed read still asserts that no rule fires on this asset; body: %s", page)
	}
	if hdr := assetHeaderOf(t, page); strings.Contains(hdr, "var(--sev-") {
		t.Errorf("a failed read raised a header severity badge; header: %s", hdr)
	}
}

func TestWithdrawnAssetKeepsItsSignalsEmptyStateWhenNoRuleFires(t *testing.T) {
	f := withdrawnNameStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusOK)

	// An empty list is the expected state for a withdrawn name, and it stays distinct from a fault.
	if !strings.Contains(page, assetSignalsEmpty) {
		t.Errorf("a withdrawn asset with no firing rule lost its original empty state; body: %s", page)
	}
	if strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
}

func serviceSubjectStore(t *testing.T) *fakeStore {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addClassReachability(t, "198.51.100.1:5900/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	return f
}

const serviceSubjectPath = "/subjects/service?key=198.51.100.1%3A5900%2Ftcp"

func TestServiceDetailNamesAFailedRuleRead(t *testing.T) {
	f := serviceSubjectStore(t)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+serviceSubjectPath, http.StatusOK)

	if !strings.Contains(page, rulesDidNotResolve) {
		t.Errorf("rules card does not name the failed read; body: %s", page)
	}
	if strings.Contains(page, subjectRulesEmpty) {
		t.Errorf("a failed read still asserts that no rule reads this subject; body: %s", page)
	}
	if strings.Contains(page, "sensitive-port-reached-from-internet") {
		t.Errorf("a failed corpus read listed a rule anyway; body: %s", page)
	}
	// The page serves its other regions, which is what a region read buys (ADR-0168 §1).
	for _, want := range []string{"Citation chain", "Current and closed timelines", "How it got here", "198.51.100.1:5900/tcp"} {
		if !strings.Contains(page, want) {
			t.Errorf("a failed rules read took away region %q; body: %s", want, page)
		}
	}
}

func TestServiceDetailKeepsItsRulesEmptyStateWhenNoRuleReadsTheSubject(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)

	if !strings.Contains(page, subjectRulesEmpty) {
		t.Errorf("a subject no rule reads lost its original empty state; body: %s", page)
	}
	if strings.Contains(page, rulesDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
}

func endpointSubjectStore(t *testing.T) *fakeStore {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addHTTPIdentity(t, "api.example.com@198.51.100.1:443/tcp", obsClock,
		`{"outcome":"responded","status":200,"server":"nginx","title":"Example API"}`)
	return f
}

func endpointSubjectPath() string {
	return "/subjects/endpoint?key=" + url.QueryEscape("api.example.com@198.51.100.1:443/tcp")
}

func TestEndpointDetailNamesAFailedRuleRead(t *testing.T) {
	f := endpointSubjectStore(t)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+endpointSubjectPath(), http.StatusOK)

	if !strings.Contains(page, rulesDidNotResolve) {
		t.Errorf("rules card does not name the failed read; body: %s", page)
	}
	if strings.Contains(page, subjectRulesEmpty) {
		t.Errorf("a failed read still asserts that no rule reads this subject; body: %s", page)
	}
	for _, want := range []string{"Citation chain", "HTTP identity", "How it got here"} {
		if !strings.Contains(page, want) {
			t.Errorf("a failed rules read took away region %q; body: %s", want, page)
		}
	}
}

func TestEndpointDetailKeepsItsRuleVerdictsOnAResolvedRead(t *testing.T) {
	f := endpointSubjectStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+endpointSubjectPath(), http.StatusOK)

	// Two rules read this endpoint, so the resolved state here is the table, not the empty state.
	if !strings.Contains(page, "unauthenticated-request-answered") {
		t.Errorf("a resolved read lost its rules table; body: %s", page)
	}
	if strings.Contains(page, rulesDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
	if strings.Contains(page, subjectRulesEmpty) {
		t.Errorf("a subject two rules read rendered the no-rule empty state; body: %s", page)
	}
}

func TestAssetDetailNamesAFailedSignalReadAtEveryCorpusRead(t *testing.T) {
	// Every read under buildSignalCorpus and deriveSignalInstances reached the swallow (#1951).
	for _, c := range []struct {
		name string
		set  func(*fakeStore)
	}{
		{"name resolutions", func(f *fakeStore) { f.nameResolutionsErr = errors.New("boom") }},
		{"dns records", func(f *fakeStore) { f.dnsRecordsErr = errors.New("boom") }},
		{"zone declarations", func(f *fakeStore) { f.zoneDeclarationsErr = errors.New("boom") }},
		{"tls acceptance", func(f *fakeStore) { f.tlsAcceptanceErr = errors.New("boom") }},
		{"endpoint certificates", func(f *fakeStore) { f.endpointCertsErr = errors.New("boom") }},
		{"endpoint subjects", func(f *fakeStore) { f.endpointSubjectsErr = errors.New("boom") }},
		{"signal instances", func(f *fakeStore) { f.signalInstancesErr = errors.New("boom") }},
		{"mint instances", func(f *fakeStore) { f.mintInstancesErr = errors.New("boom") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			lameName(t, f, "lame.example.com")
			c.set(f)

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+"/asset/lame.example.com", http.StatusOK)

			if !strings.Contains(page, signalsDidNotResolve) {
				t.Errorf("signals card does not name the failed read; body: %s", page)
			}
			if strings.Contains(page, assetSignalsEmpty) {
				t.Errorf("a failed read still asserts that no rule fires on this asset; body: %s", page)
			}
		})
	}
}

func TestSubjectRulesTableKeepsTheThreeReadStatesApart(t *testing.T) {
	render := func(t *testing.T, view subjectRulesView) string {
		t.Helper()
		var buf strings.Builder
		if err := tmpl.ExecuteTemplate(&buf, "subjectrules", view); err != nil {
			t.Fatalf("execute subjectrules template: %v", err)
		}
		return buf.String()
	}

	failed := render(t, subjectRulesView{Failed: true})
	if !strings.Contains(failed, rulesDidNotResolve) {
		t.Errorf("a failed read does not name itself; body: %s", failed)
	}
	if strings.Contains(failed, subjectRulesEmpty) {
		t.Errorf("a failed read still claims no rule reads the subject; body: %s", failed)
	}

	empty := render(t, subjectRulesView{})
	if !strings.Contains(empty, subjectRulesEmpty) {
		t.Errorf("a resolved-and-empty read lost its original empty state; body: %s", empty)
	}
	if strings.Contains(empty, rulesDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", empty)
	}

	rows := render(t, subjectRulesView{Rows: []subjectRule{
		{Rule: "fired-rule", Version: "3", Severity: "critical", SevLabel: "Critical", Verdict: signal.Fired},
	}})
	if !strings.Contains(rows, "fired-rule") {
		t.Errorf("a populated read lost its table; body: %s", rows)
	}
	for _, banned := range []string{rulesDidNotResolve, subjectRulesEmpty} {
		if strings.Contains(rows, banned) {
			t.Errorf("a populated read rendered %q; body: %s", banned, rows)
		}
	}
}
