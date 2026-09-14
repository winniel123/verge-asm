package main

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

func reachCard(t *testing.T, page string) string {
	t.Helper()
	_, rest, ok := strings.Cut(page, "<h3>Reachability</h3>")
	if !ok {
		t.Fatalf("the page carries no Reachability card; body: %s", page)
	}
	card, _, ok := strings.Cut(rest, "</section>")
	if !ok {
		t.Fatalf("the Reachability card does not close; body: %s", page)
	}
	return card
}

func reachCardCell(t *testing.T, page, label string) string {
	t.Helper()
	card := reachCard(t, page)
	_, rest, ok := strings.Cut(card, `<span class="sd-micro">`+label+`</span><span class="v">`)
	if !ok {
		t.Fatalf("the Reachability card carries no %q cell; card: %s", label, card)
	}
	value, _, ok := strings.Cut(rest, `</span></div>`)
	if !ok {
		t.Fatalf("the %q cell does not close; card: %s", label, card)
	}
	return value
}

func reachCardLeg(t *testing.T, page, label string) legChip {
	t.Helper()
	chips := legChips(t, reachCardCell(t, page, label))
	if len(chips) != 1 {
		t.Fatalf("the %q cell holds %d chips, want 1; card: %s", label, len(chips), reachCard(t, page))
	}
	return chips[0]
}

func assertReachCardLegs(t *testing.T, page string, internal, internet legChip) {
	t.Helper()
	if got := reachCardLeg(t, page, "Internal leg"); got != internal {
		t.Errorf("internal leg cell = %+v, want %+v; card: %s", got, internal, reachCard(t, page))
	}
	if got := reachCardLeg(t, page, "Internet leg"); got != internet {
		t.Errorf("internet leg cell = %+v, want %+v; card: %s", got, internet, reachCard(t, page))
	}
}

func TestServiceReachCardRendersAnInternalOnlyReachAsTwoLegs(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	// The card renders a class-scoped Reach per leg, never one pick (ADR-0017 decision 2).
	assertReachCardLegs(t, page,
		legChip{Tone: "neutral", Label: "reached"},
		legChip{Tone: "absent", Label: "never looked"})
	for _, absent := range []string{"exposed", "firewalled"} {
		if strings.Contains(page, absent) {
			t.Errorf("an internal-only reach rendered the Exposure word %q; body: %s", absent, page)
		}
	}
}

func TestServiceReachCardInternetLegIgnoresALaterInternalLeg(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, legProbeService, "internal", obsClock.Add(time.Hour), `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	assertReachCardLegs(t, page,
		legChip{Tone: "neutral", Label: "reached"},
		legChip{Tone: "neutral", Label: "not reached"})
}

func TestServiceReachCardRanksAnInternetReachedAsDanger(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"reached","result":"open"}`)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	assertReachCardLegs(t, page,
		legChip{Tone: "neutral", Label: "reached"},
		legChip{Tone: "danger", Label: "reached"})
}

func TestServiceReachCardSinceReadsTheEarliestOpenLeg(t *testing.T) {
	early := obsClock
	late := obsClock.Add(48 * time.Hour)
	want := early.UTC().Format(spanTimeFmt)

	for _, c := range []struct {
		name  string
		first string
		last  string
	}{
		{"internal first", "internal", "internet"},
		{"internet first", "internet", "internal"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := legProbeStore(t)
			f.addClassReachability(t, legProbeService, c.first, early, `{"outcome":"not-reached"}`)
			f.addClassReachability(t, legProbeService, c.last, late, `{"outcome":"reached","result":"open"}`)

			page := serviceDetailBody(t, f, http.StatusOK)

			// Since is the earliest open reachability span, so the seeding order cannot move it.
			if got := reachCardCell(t, page, "Since"); got != want {
				t.Errorf("Since = %q, want %q", got, want)
			}
		})
	}
}

func TestServiceReachCardAbsentLegsKeepTheirTwoWords(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"gap","reason":"the control probe did not complete"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	// The two absences keep their two statements (ADR-0017 decision 4).
	assertReachCardLegs(t, page,
		legChip{Tone: "warn", Label: "stopped looking"},
		legChip{Tone: "absent", Label: "never looked"})
	if card := reachCard(t, page); !strings.Contains(card, "the internal class") {
		t.Errorf("a Gap on the internal leg named no class; card: %s", card)
	}
}

func TestServiceReachCardGapBannerNamesTheClassThatHoldsIt(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"reached","result":"open"}`)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"gap","reason":"an edge answers for the origin"}`)

	card := reachCard(t, serviceDetailBody(t, f, http.StatusOK))

	for _, want := range []string{"an edge answers for the origin", "the internet class"} {
		if !strings.Contains(card, want) {
			t.Errorf("the Gap banner is missing %q; card: %s", want, card)
		}
	}
	// A leg speaks for a whole Vantage class, so the old per-vantage wording is wrong.
	if strings.Contains(card, "From this vantage") {
		t.Errorf("the Gap banner still speaks for one vantage; card: %s", card)
	}
	if strings.Contains(card, "internal or internet") {
		t.Errorf("a Gap on one leg named both classes; card: %s", card)
	}
}

func TestServiceReachCardGapOnBothLegsStatesOneBanner(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"gap","reason":"the control probe did not complete"}`)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"gap","reason":"an edge answers for the origin"}`)

	card := reachCard(t, serviceDetailBody(t, f, http.StatusOK))

	// The advice is one action, so two gapped legs state it once under both class names.
	if got := strings.Count(card, "address scope to measure the real surface"); got != 1 {
		t.Errorf("the card renders %d Gap banners, want 1; card: %s", got, card)
	}
	for _, want := range []string{
		"the internal or internet class",
		"the control probe did not complete",
		"an edge answers for the origin",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("the Gap banner is missing %q; card: %s", want, card)
		}
	}
	want := legChip{Tone: "warn", Label: "stopped looking"}
	assertReachCardLegs(t, serviceDetailBody(t, f, http.StatusOK), want, want)
}

func TestServiceReachCardRendersNoLegWhenNoSpanNamesAVantage(t *testing.T) {
	f := legProbeStore(t)
	// A vantage-less span leaves the join, and never looked would be a false word (#1985).
	f.addReachability(t, legProbeService, obsClock, `{"outcome":"reached","result":"open"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	card := reachCard(t, page)
	if legChipCell.MatchString(card) {
		t.Errorf("a service no class-aware span reaches rendered a leg chip; card: %s", card)
	}
	// The header drops its whole chip on this input, so the card drops its whole cell (#1985).
	for _, label := range []string{"Internal leg", "Internet leg"} {
		if strings.Contains(card, `<span class="sd-micro">`+label+`</span>`) {
			t.Errorf("the card still carries a %q cell; card: %s", label, card)
		}
	}
	if !strings.Contains(card, `<span class="sd-micro">Since</span>`) {
		t.Errorf("the card withheld more than its leg cells; card: %s", card)
	}
}

func TestServiceReachCardStatesAGapNoLegCarries(t *testing.T) {
	f := legProbeStore(t)
	// A vantage presenting no address derives the unverified class, which fills neither leg.
	f.addClassReachability(t, legProbeService, "unverified", obsClock,
		`{"outcome":"gap","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`)

	card := reachCard(t, serviceDetailBody(t, f, http.StatusOK))

	if !strings.Contains(card, "it is a proxy edge, not your origin") {
		t.Errorf("a Gap no leg carries lost its cause; card: %s", card)
	}
	if strings.Contains(card, " class can tell") {
		t.Errorf("a Gap no leg carries named a class; card: %s", card)
	}
}

func TestServiceReachCardWithdrawnServiceCarriesNoLegCell(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.addClassReachability(t, "203.0.113.99:5900/tcp", "internet", obsClock, `{"outcome":"reached","result":"open"}`)
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/subjects/service?key=203.0.113.99%3A5900%2Ftcp", http.StatusOK)

	card := reachCard(t, page)
	for _, label := range []string{"Internal leg", "Internet leg"} {
		if strings.Contains(card, `<span class="sd-micro">`+label+`</span>`) {
			t.Errorf("a withdrawn service carries a %q cell; card: %s", label, card)
		}
	}
}

func TestServiceReachCardCarriesNoVerdictCell(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"reached","result":"open"}`)

	card := reachCard(t, serviceDetailBody(t, f, http.StatusOK))

	// A latest-observation pick across every class is no well-formed Reach leg (CONTEXT.md, Reach).
	if strings.Contains(card, `<span class="sd-micro">Verdict</span>`) {
		t.Errorf("the card still carries the class-blind Verdict cell; card: %s", card)
	}
}

func TestServiceReachCardLegsFollowTheHeaderChip(t *testing.T) {
	f := legProbeStore(t)
	f.addClassReachability(t, legProbeService, "internal", obsClock, `{"outcome":"reached","result":"open"}`)
	f.addClassReachability(t, legProbeService, "internet", obsClock, `{"outcome":"not-reached"}`)

	page := serviceDetailBody(t, f, http.StatusOK)

	// The header and the card read one internet leg, so the page states one thing (#1984).
	header := headerLegChips(t, page)
	card := []legChip{reachCardLeg(t, page, "Internet leg")}
	if !slices.Equal(header, card) {
		t.Errorf("header = %+v, card internet leg = %+v", header, card)
	}
}
