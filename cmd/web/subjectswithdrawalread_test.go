package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	withdrawalDidNotResolve = "The read did not resolve"
	subjectKeyedNowhere     = "No subject is keyed under that name"
)

func unresolvedWithdrawalFixture(t *testing.T, set func(*fakeStore)) string {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["203.0.113.7"]}`)
	f.addResolution(t, admin.ID, "gone.example.com", "dns", obsClock.Add(24*time.Hour), `{"outcome":"NameError"}`)
	f.withdrawSubject("name", "gone.example.com", obsClock.Add(24*time.Hour))
	set(f)
	return startAt(t, f, obsClock.Add(30*24*time.Hour))
}

func TestFailedSpanReadNeverRendersAWithdrawnNameAsMissing(t *testing.T) {
	base := unresolvedWithdrawalFixture(t, func(f *fakeStore) {
		f.subjectSpansErr = errors.New("span read failed")
	})
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusServiceUnavailable)

	if strings.Contains(page, "No such subject") || strings.Contains(page, subjectKeyedNowhere) {
		t.Errorf("a failed read rendered as a fact about the subject; body: %s", page)
	}
	if strings.Contains(page, "no current member") {
		t.Errorf("a failed read claimed the withdrawal it could not read; body: %s", page)
	}
	for _, want := range []string{withdrawalDidNotResolve, "did not resolve on this load", "gone.example.com", "Back to inventory"} {
		if !strings.Contains(page, want) {
			t.Errorf("unresolved-subject page missing %q; body: %s", want, page)
		}
	}
}

func TestFailedSpanReadNeverRendersAnUnmeasuredKeyAsMissing(t *testing.T) {
	base := unresolvedWithdrawalFixture(t, func(f *fakeStore) {
		f.subjectSpansErr = errors.New("span read failed")
	})
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/asset/never.measured.example", http.StatusServiceUnavailable)
	if strings.Contains(page, subjectKeyedNowhere) {
		t.Errorf("a failed read asserted that nothing was ever measured; body: %s", page)
	}
	if !strings.Contains(page, withdrawalDidNotResolve) {
		t.Errorf("unresolved-subject page missing its heading; body: %s", page)
	}
}

func TestNoSpanRowKeepsTheMissingSubjectPage(t *testing.T) {
	// No row is the honest empty answer, so it must not degrade (ADR-0168 §4).
	for _, empty := range []struct {
		name string
		set  func(*fakeStore)
	}{
		{"zero rows", func(f *fakeStore) { f.subjectSpansEmpty = true }},
		{"no rows error", func(f *fakeStore) { f.subjectSpansErr = pgx.ErrNoRows }},
	} {
		t.Run(empty.name, func(t *testing.T) {
			base := unresolvedWithdrawalFixture(t, empty.set)
			ac := login(t, base, "admin", "hunter2hunter2")

			page := getBody(t, ac, base+"/asset/gone.example.com", http.StatusNotFound)
			if !strings.Contains(page, subjectKeyedNowhere) {
				t.Errorf("a subject with no span lost the missing-subject page; body: %s", page)
			}
			if strings.Contains(page, withdrawalDidNotResolve) {
				t.Errorf("an empty read read as a failed one; body: %s", page)
			}
		})
	}
}
