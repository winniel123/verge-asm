package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	spansDidNotResolve     = "The span read did not resolve on this load."
	firstSeenDidNotResolve = "the span read did not resolve"
	timelinesEmpty         = "No timeline has been folded yet."
)

type subjectSpanCase struct {
	name  string
	store func(*testing.T) *fakeStore
	path  string
}

func subjectSpanReadCases() []subjectSpanCase {
	return []subjectSpanCase{
		{"endpoint", endpointSubjectStore, endpointSubjectPath()},
		{"service", serviceSubjectStore, serviceSubjectPath},
	}
}

func TestSubjectDetailNamesAFailedSpanReadAtBothEnds(t *testing.T) {
	for _, c := range subjectSpanReadCases() {
		t.Run(c.name, func(t *testing.T) {
			f := c.store(t)
			f.subjectSpansErr = errors.New("span read failed")

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+c.path, http.StatusOK)

			if !strings.Contains(page, spansDidNotResolve) {
				t.Errorf("timelines card does not name the failed read; body: %s", page)
			}
			if strings.Contains(page, timelinesEmpty) {
				t.Errorf("a failed read still asserts that this subject has no timeline; body: %s", page)
			}
			if !strings.Contains(page, firstSeenDidNotResolve) {
				t.Errorf("provenance card does not name the failed read; body: %s", page)
			}
			if !strings.Contains(page, "First seen") {
				t.Errorf("a failed read dropped the First seen row with nothing said; body: %s", page)
			}
			for _, want := range []string{"Citation chain", "How it got here", "Rules over this subject"} {
				if !strings.Contains(page, want) {
					t.Errorf("a failed span read took away region %q; body: %s", want, page)
				}
			}
		})
	}
}

func TestSubjectDetailKeepsItsEmptyStatesWhenNoSpanExists(t *testing.T) {
	for _, empty := range []struct {
		name string
		set  func(*fakeStore)
	}{
		{"zero rows", func(f *fakeStore) { f.subjectSpansEmpty = true }},
		// No row is the honest empty answer, so it must not raise the failed flag (ADR-0168 §4).
		{"no rows error", func(f *fakeStore) { f.subjectSpansErr = pgx.ErrNoRows }},
	} {
		for _, c := range subjectSpanReadCases() {
			t.Run(empty.name+"/"+c.name, func(t *testing.T) {
				f := c.store(t)
				empty.set(f)

				base := start(t, f, "")
				ac := login(t, base, "admin", "hunter2hunter2")
				page := getBody(t, ac, base+c.path, http.StatusOK)

				if !strings.Contains(page, timelinesEmpty) {
					t.Errorf("a subject with no span lost its original timelines empty state; body: %s", page)
				}
				if strings.Contains(page, "did not resolve") {
					t.Errorf("an empty read read as a failed one; body: %s", page)
				}
				if strings.Contains(page, "First seen") {
					t.Errorf("a subject with no span gained a First seen row; body: %s", page)
				}
			})
		}
	}
}

func TestSubjectDetailRendersFirstSeenOnAResolvedSpanRead(t *testing.T) {
	for _, c := range subjectSpanReadCases() {
		t.Run(c.name, func(t *testing.T) {
			f := c.store(t)

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+c.path, http.StatusOK)

			if !strings.Contains(page, "First seen") {
				t.Errorf("a resolved read lost its First seen row; body: %s", page)
			}
			if strings.Contains(page, "did not resolve") {
				t.Errorf("a resolved read read as a failed one; body: %s", page)
			}
		})
	}
}
