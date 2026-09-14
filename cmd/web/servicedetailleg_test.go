package main

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

const legProbeService = "198.51.100.1:5900/tcp"

func legProbeStore(t *testing.T) *fakeStore {
	t.Helper()
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	f.addResolution(t, admin.ID, "api.example.com", "dns", obsClock, `{"outcome":"Resolved","addresses":["198.51.100.1"]}`)
	return f
}

func serviceDetailBody(t *testing.T, f *fakeStore, status int) string {
	t.Helper()
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	return getBody(t, ac, base+"/subjects/service?key=198.51.100.1%3A5900%2Ftcp", status)
}

func TestServiceDetailHeaderRanksAnInternetReached(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	want := []legChip{{Tone: "danger", Label: "reached"}}
	if got := legChips(t, page); !slices.Equal(got, want) {
		t.Errorf("header leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

func TestServiceDetailInternalOnlyReachedNeverReadsExposed(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	for _, absent := range []string{"exposed", "firewalled"} {
		if strings.Contains(page, absent) {
			t.Errorf("an internal-only reach rendered the Exposure word %q; body: %s", absent, page)
		}
	}
	want := []legChip{{Tone: "absent", Label: "never looked"}}
	if got := legChips(t, page); !slices.Equal(got, want) {
		t.Errorf("header leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

func TestServiceDetailHeaderIgnoresALaterInternalLeg(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, legProbeService, "internal", obsClock.Add(time.Hour), `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	want := []legChip{{Tone: "neutral", Label: "not reached"}}
	if got := legChips(t, page); !slices.Equal(got, want) {
		t.Errorf("header leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

func TestServiceDetailAbsentInternetLegsKeepTheirTwoWords(t *testing.T) {
	// The two absences keep their two statements (ADR-0017 decision 4).
	gap := legProbeStore(t)
	gap.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"gap","reason":"an edge answers for the origin"}`)
	want := []legChip{{Tone: "warn", Label: "stopped looking"}}
	if got := legChips(t, serviceDetailBody(t, gap, http.StatusOK)); !slices.Equal(got, want) {
		t.Errorf("a Gap on the internet leg = %+v, want %+v", got, want)
	}

	never := legProbeStore(t)
	never.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"not-reached"}`)
	want = []legChip{{Tone: "absent", Label: "never looked"}}
	if got := legChips(t, serviceDetailBody(t, never, http.StatusOK)); !slices.Equal(got, want) {
		t.Errorf("a never-configured internet leg = %+v, want %+v", got, want)
	}
}

func TestServiceDetailFailsLoudlyWhenItsLegReadFails(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	f.reachSpansErr = errors.New("class-aware reach read failed")

	// A swallowed leg read renders the header with no chip, which reads as never looked (#1948).
	serviceDetailBody(t, f, http.StatusInternalServerError)
}
