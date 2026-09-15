package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

const serviceSignalsNoOpenClaim = "so nothing here says this subject holds no open signal"

func TestServiceDetailNamesAFailedInstancesReadBesideAFiredVerdict(t *testing.T) {
	for _, c := range []struct {
		name string
		set  func(*fakeStore)
	}{
		{"signal instances", func(f *fakeStore) { f.signalInstancesErr = errors.New("boom") }},
		{"mint instances", func(f *fakeStore) { f.mintInstancesErr = errors.New("boom") }},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := serviceSubjectStore(t)
			c.set(f)

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+serviceSubjectPath, http.StatusOK)

			// The corpus still builds, so the rules card renders the fired verdict this pairs with.
			if !strings.Contains(page, "sensitive-port-reached-from-internet") {
				t.Fatalf("the rules card lost its own resolved read; body: %s", page)
			}
			if strings.Contains(page, rulesDidNotResolve) {
				t.Fatalf("a failed instances read degraded the rules card too; body: %s", page)
			}
			if !strings.Contains(page, signalsDidNotResolve) {
				t.Errorf("the signals card does not name the failed read; body: %s", page)
			}
			if !strings.Contains(page, serviceSignalsNoOpenClaim) {
				t.Errorf("the signals card does not deny the no-open-signal claim; body: %s", page)
			}
		})
	}
}

func TestServiceDetailDegradesBothCardsOnAFailedCorpusRead(t *testing.T) {
	f := serviceSubjectStore(t)
	corpusReadFails(f)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+serviceSubjectPath, http.StatusOK)

	// The corpus feeds both cards, so neither may claim the other's read resolved.
	for _, want := range []string{rulesDidNotResolve, signalsDidNotResolve} {
		if !strings.Contains(page, want) {
			t.Errorf("a failed corpus read does not carry %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, subjectRulesEmpty) {
		t.Errorf("a failed read still asserts that no rule reads this subject; body: %s", page)
	}
	if strings.Contains(page, "sensitive-port-reached-from-internet") {
		t.Errorf("a failed corpus read listed a rule anyway; body: %s", page)
	}
}

func TestWithdrawnServiceNamesAFailedInstancesRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "203.0.113.99:5900/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	f.signalInstancesErr = errors.New("boom")

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/subjects/service?key=203.0.113.99%3A5900%2Ftcp", http.StatusOK)

	if !strings.Contains(page, "Withdrawn by the world") {
		t.Fatalf("the withdrawn service lost its withdrawal notice; body: %s", page)
	}
	// The rules card renders for a withdrawn service too, so the pairing reaches this page.
	if !strings.Contains(page, "sensitive-port-reached-from-internet") {
		t.Fatalf("the rules card lost its own resolved read; body: %s", page)
	}
	if !strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("the signals card does not name the failed read; body: %s", page)
	}
	if !strings.Contains(page, serviceSignalsNoOpenClaim) {
		t.Errorf("the signals card does not deny the no-open-signal claim; body: %s", page)
	}
}

func TestServiceDetailOmitsItsSignalsCardWhenTheReadResolvesEmpty(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	f.addReachability(t, "198.51.100.1:443/tcp", obsClock, `{"outcome":"reached","result":"open"}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A443%2Ftcp", http.StatusOK)

	if strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
	if strings.Contains(page, "Signals here") {
		t.Errorf("a resolved-and-empty read raised the signals card; body: %s", page)
	}
}

func TestServiceDetailKeepsItsSignalsListOnAResolvedRead(t *testing.T) {
	f := serviceSubjectStore(t)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+serviceSubjectPath, http.StatusOK)

	if !strings.Contains(page, "Signals here") {
		t.Fatalf("a fired rule raised no signals card; body: %s", page)
	}
	if strings.Contains(page, signalsDidNotResolve) {
		t.Errorf("a resolved read read as a failed one; body: %s", page)
	}
}
