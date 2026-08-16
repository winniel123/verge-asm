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
	// Two ticks in the same daily window must resolve to one key, so the second
	// fan-out conflicts and is skipped rather than run concurrently.
	a := scheduledTick(base, cadence)
	b := scheduledTick(base.Add(6*time.Hour), cadence)
	if !a.Equal(b) {
		t.Errorf("two ticks in one window differ: %s vs %s", a, b)
	}
	// A tick in the next window is a different key.
	c := scheduledTick(base.Add(25*time.Hour), cadence)
	if a.Equal(c) {
		t.Errorf("next window collapsed onto this one: %s", c)
	}
}

func TestBackoffGrowsAndCaps(t *testing.T) {
	if backoff(2) <= backoff(1) {
		t.Error("backoff should grow with attempt")
	}
	if backoff(20) != 32*time.Minute {
		t.Errorf("backoff should cap at 32m, got %s", backoff(20))
	}
}

// §4.5's retry budget: the waits before the five attempts span roughly an hour,
// the budget both measurement jobs and Channel deliveries share. A base of 30s
// summed to ~15m and quietly missed it.
func TestBackoffBudgetIsAboutAnHour(t *testing.T) {
	var total time.Duration
	for attempt := int32(2); attempt <= 5; attempt++ {
		total += backoff(attempt)
	}
	if total != 60*time.Minute {
		t.Errorf("five-attempt budget = %s, want ~1h", total)
	}
}

// AC #195: a reachability observation folds onto a `service` subject's timeline,
// sourced by the prober (never the resolver), under the connect-outcome vector —
// so the span fold is facet-generic, not a resolution hardcode.
func TestReachabilityFoldsToServiceProberTimeline(t *testing.T) {
	if got := subjectKindFor(connectoutcome.FacetReachability); got != "service" {
		t.Errorf("reachability subject kind = %q, want service", got)
	}
	if got := sourceFor(connectoutcome.FacetReachability); got != "prober" {
		t.Errorf("reachability source = %q, want prober", got)
	}
	// Since ADR-0104 the reachability vector composes two leaves — connect-outcome
	// and blanket-discrimination — so a bump of either Breaks the reach half once.
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
	// The resolution vector is unchanged — two leaves, not the reachability one.
	if r := facetVector(resolutionwalk.FacetResolution); len(r) != 2 {
		t.Errorf("resolution vector = %+v, want two leaves", r)
	}
}

// A blanketed reach's Gap observation folds to an is_gap span, while an ordinary
// reachability value does not — so a blanket responder's leg reads as absent
// downstream without a special case (ADR-0104). The resolution gap still folds too.
func TestReachabilityGapFoldsToIsGap(t *testing.T) {
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
		{Kind: "tcp-connect"}, // no facet — a kind whose leaf a later ticket adds
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
