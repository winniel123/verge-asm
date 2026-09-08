package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

func TestScheduledTickIsIdempotentWithinAWindow(t *testing.T) {
	cadence := 24 * time.Hour
	base := time.Date(2026, 8, 15, 3, 0, 0, 0, time.UTC)
	a := scheduledTick(base, cadence)
	b := scheduledTick(base.Add(6*time.Hour), cadence)
	if !a.Equal(b) {
		t.Errorf("two ticks in one window differ: %s vs %s", a, b)
	}
	c := scheduledTick(base.Add(25*time.Hour), cadence)
	if a.Equal(c) {
		t.Errorf("next window collapsed onto this one: %s", c)
	}
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	if jitteredBackoff(2, 0) <= jitteredBackoff(1, 0) {
		t.Error("backoff should grow with attempt")
	}
	if jitteredBackoff(20, 1) != 32*time.Minute {
		t.Errorf("backoff should cap at 32m, got %s", jitteredBackoff(20, 1))
	}
}

func TestBackoffJitterStaysInsideTheCurve(t *testing.T) {
	// ADR-0005 rules per-job exponential with jitter; the jitter only ever shortens a wait.
	for attempt := int32(1); attempt <= 20; attempt++ {
		ceiling := jitteredBackoff(attempt, 1)
		floor := jitteredBackoff(attempt, 0)
		if floor != ceiling/2 {
			t.Errorf("attempt %d: floor = %s, want half the ceiling %s", attempt, floor, ceiling)
		}
		for range 200 {
			got := backoff(attempt)
			if got < floor || got >= ceiling {
				t.Fatalf("attempt %d: backoff = %s, want in [%s, %s)", attempt, got, floor, ceiling)
			}
		}
	}
	if got := jitteredBackoff(3, 0.5); got != 6*time.Minute {
		t.Errorf("jitteredBackoff(3, 0.5) = %s, want 6m: half of 8m plus half the remainder", got)
	}
}

func TestExhaustedRetries(t *testing.T) {
	const max = 5

	if !exhaustedRetries(max, max) {
		t.Fatalf("attempt %d of %d not exhausted: the ct run would re-dispatch past its budget forever", max, max)
	}
	if !exhaustedRetries(max+1, max) {
		t.Errorf("attempt %d of %d not exhausted: the terminal bound leaks past the budget", max+1, max)
	}

	flips := 0
	prev := false
	for a := int32(1); a <= max; a++ {
		got := exhaustedRetries(a, max)
		if a < max && got {
			t.Errorf("attempt %d of %d read as exhausted: a legitimate retry was dropped", a, max)
		}
		if got && !prev {
			flips++
		}
		prev = got
	}
	if flips != 1 {
		t.Errorf("the retry budget flipped to exhausted %d times, want exactly 1 (a single clean terminal transition)", flips)
	}

	if !exhaustedRetries(1, 1) {
		t.Error("a max_attempts==1 job (zone) did not settle on its first failure")
	}
}

func TestMergeResolutionNamesUnionsAdmittedNames(t *testing.T) {
	seeds := []string{"example.com", "example.net"}
	admitted := []string{"vpn.example.com", "a.b.example.com"}
	got := mergeResolutionNames(seeds, admitted)
	want := []string{"example.com", "example.net", "vpn.example.com", "a.b.example.com"}
	if len(got) != len(want) {
		t.Fatalf("merged = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("merged[%d] = %q, want %q (Seeds lead, admitted appended)", i, got[i], want[i])
		}
	}
}

func TestMergeResolutionNamesDedupesAgainstSeedsAndItself(t *testing.T) {
	seeds := []string{"example.com"}
	admitted := []string{"Example.com.", "vpn.example.com", "vpn.example.com"}
	got := mergeResolutionNames(seeds, admitted)
	want := []string{"example.com", "vpn.example.com"}
	if len(got) != len(want) {
		t.Fatalf("merged = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("merged[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMergeResolutionNamesKeysOnResolverCanonicalName(t *testing.T) {
	// An inline ToLower folds a non-ASCII pair the resolver keeps apart, dropping one (#256).
	if got := resolutionwalk.CanonicalName("Ä.example.com"); got != "Ä.example.com" {
		t.Fatalf("precondition: CanonicalName folded a non-ASCII uppercase letter: %q", got)
	}
	seeds := []string{"ä.example.com"}
	admitted := []string{"Ä.example.com"}
	got := mergeResolutionNames(seeds, admitted)
	want := []string{"ä.example.com", "Ä.example.com"}
	if len(got) != len(want) {
		t.Fatalf("merged = %v, want %v — a non-ASCII-uppercase name must not fold onto its lowercase Seed", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("merged[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	if resolutionNameKey("Ä.example.com") != resolutionwalk.CanonicalName("Ä.example.com") {
		t.Errorf("resolutionNameKey diverged from the resolver's CanonicalName (#256)")
	}
}

func TestMergeResolutionNamesEmptyAdmittedIsSeedsUnchanged(t *testing.T) {
	seeds := []string{"example.com", "example.net"}
	got := mergeResolutionNames(seeds, nil)
	if len(got) != 2 || got[0] != "example.com" || got[1] != "example.net" {
		t.Errorf("no admitted names must leave the Seed set unchanged, got %v", got)
	}
}

func TestBackoffBudgetIsAboutAnHour(t *testing.T) {
	var longest, shortest time.Duration
	// Five attempts over roughly one hour, never more (notification-channels §4.2).
	for attempt := int32(2); attempt <= 5; attempt++ {
		longest += jitteredBackoff(attempt, 1)
		shortest += jitteredBackoff(attempt, 0)
	}
	if longest != 60*time.Minute {
		t.Errorf("five-attempt budget = %s at most, want 1h", longest)
	}
	if shortest != 30*time.Minute {
		t.Errorf("five-attempt budget = %s at least, want 30m", shortest)
	}
}

func TestReachabilityFoldsToServiceProberTimeline(t *testing.T) {
	if got := subjectKindFor(connectoutcome.FacetReachability); got != "service" {
		t.Errorf("reachability subject kind = %q, want service", got)
	}
	if got := sourceFor(connectoutcome.FacetReachability); got != "prober" {
		t.Errorf("reachability source = %q, want prober", got)
	}
	v := facetVector(connectoutcome.FacetReachability)
	if len(v) != 2 {
		t.Fatalf("reachability vector = %+v, want two leaves (connect-outcome + blanket-discrimination)", v)
	}
	leaves := map[string]string{v[0].Leaf: v[0].Version, v[1].Leaf: v[1].Version}
	if leaves[connectoutcome.Kind] != connectoutcome.Version {
		t.Errorf("reachability vector missing connect-outcome leaf: %+v", v)
	}
	if leaves[blanketdiscrim.Kind] != blanketdiscrim.Version {
		t.Errorf("reachability vector missing blanket-discrimination leaf: %+v", v)
	}
	if r := facetVector(resolutionwalk.FacetResolution); len(r) != 2 {
		t.Errorf("resolution vector = %+v, want two leaves", r)
	}
}

func TestReachabilityGapFoldsToIsGap(t *testing.T) {
	// is_gap reads a blanket responder's leg as absent downstream with no special case (ADR-0104).
	gap := json.RawMessage(`{"outcome":"gap","cause":"blanket-responder","reason":"proxy edge"}`)
	if !isGapValue(connectoutcome.FacetReachability, gap) {
		t.Error("a reachability gap observation must fold to is_gap=true")
	}
	for _, v := range []string{`{"outcome":"reached","result":"open"}`, `{"outcome":"not-reached","result":"refused"}`} {
		if isGapValue(connectoutcome.FacetReachability, json.RawMessage(v)) {
			t.Errorf("a reachability value %s must not be a gap", v)
		}
	}
	if !isGapValue(resolutionwalk.FacetResolution, json.RawMessage(`{"outcome":"Gap"}`)) {
		t.Error("a resolution gap must still fold to is_gap=true")
	}
}

func TestToObservationParamsAttributesTimelineAndSkipsFacetless(t *testing.T) {
	obs := []wire.Observation{
		{Kind: resolutionwalk.Kind, Facet: "resolution", Subject: "example.com", Vantage: "v1", Data: json.RawMessage(`{"outcome":"NameError"}`)},
		{Kind: resolutionwalk.Kind, Facet: "dns-record", Subject: "example.com", Discriminator: "A", Data: json.RawMessage(`{"rrs":null}`)},
		{Kind: "tcp-connect"},
	}
	params := toObservationParams(42, pgInt8(7), tstz(time.Now()), obs)
	if len(params) != 2 {
		t.Fatalf("got %d params, want 2 (the facet-less line skipped)", len(params))
	}
	if params[0].Facet != "resolution" || params[0].SubjectKey != "example.com" || params[0].BatchID != 42 {
		t.Errorf("resolution row mis-attributed: %+v", params[0])
	}
	if params[1].Discriminator != "A" || params[1].SubjectKind != "name" {
		t.Errorf("dns-record row mis-attributed: %+v", params[1])
	}
	if !params[0].VantageID.Valid || params[0].VantageID.Int64 != 7 {
		t.Errorf("vantage not carried: %+v", params[0].VantageID)
	}
}
