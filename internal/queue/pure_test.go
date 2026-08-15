package queue

import (
	"encoding/json"
	"testing"
	"time"

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
	if backoff(20) != 16*time.Minute {
		t.Errorf("backoff should cap at 16m, got %s", backoff(20))
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
