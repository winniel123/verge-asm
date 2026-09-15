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

func subjectHeader(t *testing.T, page string) string {
	t.Helper()
	// The card renders its own leg cells, so a whole-page scrape no longer reads the header alone.
	_, main, ok := strings.Cut(page, `<main class="sd-main"`)
	if !ok {
		t.Fatalf("the page carries no subject main; body: %s", page)
	}
	// The chrome opens its own header first, so the cut starts inside the subject main.
	_, rest, ok := strings.Cut(main, "<header")
	if !ok {
		t.Fatalf("the page carries no header; body: %s", page)
	}
	head, _, ok := strings.Cut(rest, "</header>")
	if !ok {
		t.Fatalf("the page header does not close; body: %s", page)
	}
	return head
}

func headerLegChips(t *testing.T, page string) []legChip {
	t.Helper()
	return legChips(t, subjectHeader(t, page))
}

func assertNoHeaderLegChip(t *testing.T, page, what string) {
	t.Helper()
	if head := subjectHeader(t, page); strings.Contains(head, `<span class="vg-leg `) {
		t.Errorf("%s rendered a header leg chip; header: %s", what, head)
	}
}

func TestServiceDetailHeaderRanksAnInternetReached(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	want := []legChip{{Tone: "danger", Label: "reached"}}
	if got := headerLegChips(t, page); !slices.Equal(got, want) {
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
	if got := headerLegChips(t, page); !slices.Equal(got, want) {
		t.Errorf("header leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

func TestServiceDetailHeaderIgnoresALaterInternalLeg(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, legProbeService, "internal", obsClock.Add(time.Hour), `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	want := []legChip{{Tone: "neutral", Label: "not reached"}}
	if got := headerLegChips(t, page); !slices.Equal(got, want) {
		t.Errorf("header leg chips = %+v, want %+v; body: %s", got, want, page)
	}
}

func TestServiceDetailAbsentInternetLegsKeepTheirTwoWords(t *testing.T) {
	// The two absences keep their two statements (ADR-0017 decision 4).
	gap := legProbeStore(t)
	gap.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"gap","reason":"an edge answers for the origin"}`)
	want := []legChip{{Tone: "warn", Label: "stopped looking"}}
	if got := headerLegChips(t, serviceDetailBody(t, gap, http.StatusOK)); !slices.Equal(got, want) {
		t.Errorf("a Gap on the internet leg = %+v, want %+v", got, want)
	}

	never := legProbeStore(t)
	never.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"not-reached"}`)
	want = []legChip{{Tone: "absent", Label: "never looked"}}
	if got := headerLegChips(t, serviceDetailBody(t, never, http.StatusOK)); !slices.Equal(got, want) {
		t.Errorf("a never-configured internet leg = %+v, want %+v", got, want)
	}
}

func TestServiceDetailWithdrawsTheChipWhenNoSpanNamesAVantage(t *testing.T) {
	f := legProbeStore(t)
	// A vantage-less span leaves the join, and never looked would be a false word (#1985).
	f.addReachability(t, legProbeService, obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	if strings.Contains(page, `class="vg-leg`) {
		t.Errorf("a service no class-aware span reaches rendered a leg chip; body: %s", page)
	}
	if !strings.Contains(page, `<span class="sd-tag">service</span>`) {
		t.Errorf("the page withheld more than the chip; body: %s", page)
	}
}

const reachDidNotResolve = "The reachability read did not resolve on this load."

func TestServiceDetailReachReadFailureRendersDidNotResolve(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	f.reachSpansErr = errors.New("class-aware reach read failed")

	page := serviceDetailBody(t, f, http.StatusOK)

	card := reachCard(t, page)
	if !strings.Contains(card, reachDidNotResolve) {
		t.Errorf("a failed reach read rendered no did-not-resolve note; card: %s", card)
	}
	for _, label := range []string{"Internal leg", "Internet leg"} {
		if strings.Contains(card, `<span class="sd-micro">`+label+`</span>`) {
			t.Errorf("a failed reach read rendered the %q cell; card: %s", label, card)
		}
	}
	// The address and the port come from the subject key, so this read never withholds them.
	if got := reachCardCell(t, page, "Address"); got != "198.51.100.1" {
		t.Errorf("address cell = %q, want %q; card: %s", got, "198.51.100.1", card)
	}
	assertNoHeaderLegChip(t, page, "a failed reach read")
	// never looked is a claim about the estate's scanning, so a fault may not produce it (ADR-2030).
	if strings.Contains(page, "never looked") {
		t.Errorf("a failed reach read substituted a leg word; body: %s", page)
	}
}

func TestServiceDetailReachReadFailureKeepsEveryOtherRegion(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	f.reachSpansErr = errors.New("class-aware reach read failed")

	page := serviceDetailBody(t, f, http.StatusOK)

	// A 500 removed every one of these before (#2051).
	for _, want := range []string{
		`<ol class="sd-chain">`,
		"api.example.com",
		"<h3>Current and closed timelines</h3>",
		"<h3>How it got here</h3>",
		"<h3>Rules over this subject</h3>",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("a failed reach read removed %q; body: %s", want, page)
		}
	}
	if strings.Contains(page, "No timeline has been folded yet") {
		t.Errorf("a failed reach read emptied the timelines card; body: %s", page)
	}
}

func TestServiceDetailWithNoReachRowKeepsItsOriginalEmptyState(t *testing.T) {
	f := legProbeStore(t)
	// A vantage-less span leaves the join, so the card holds no leg and no failure (#1985).
	f.addReachability(t, legProbeService, obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	if strings.Contains(page, reachDidNotResolve) {
		t.Errorf("a service with no reach row rendered the did-not-resolve note; body: %s", page)
	}
	assertNoHeaderLegChip(t, page, "a service with no reach row")
}

const gapBannerCopy = "so this reach is undiscriminated"

func TestServiceDetailReachReadFailureWithholdsTheSubjectGapNote(t *testing.T) {
	// #2018 resolves the banner's prose from the cause, so a causeless Gap states the reason alone.
	gapValue := `{"outcome":"gap","reason":"an edge answers for the origin","cause":"blanket-responder"}`

	healthy := legProbeStore(t)
	healthy.addReachability(t, legProbeService, obsClock, gapValue)
	page := serviceDetailBody(t, healthy, http.StatusOK)
	if !strings.Contains(page, gapBannerCopy) {
		t.Fatalf("a Gap no leg carries rendered no gap note; body: %s", page)
	}

	failed := legProbeStore(t)
	failed.addReachability(t, legProbeService, obsClock, gapValue)
	failed.reachSpansErr = errors.New("class-aware reach read failed")
	page = serviceDetailBody(t, failed, http.StatusOK)

	card := reachCard(t, page)
	if !strings.Contains(card, reachDidNotResolve) {
		t.Errorf("a failed reach read rendered no did-not-resolve note; card: %s", card)
	}
	// One card states one thing, and a verdict beside the note would be the broader claim.
	if strings.Contains(card, gapBannerCopy) {
		t.Errorf("a failed reach read stated a reach verdict beside its note; card: %s", card)
	}
}
