package queue

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

func TestHotLagGateAppliesToHotAndCTTailAlone(t *testing.T) {
	if !hotLagGateApplies(scan.HotKind) {
		t.Error("the hot Scan carries the per-target ceiling, so the gate must apply to it")
	}
	if !hotLagGateApplies(scan.CTTailKind) {
		t.Error("the ct-tail Scan's throttle floor exceeds its cadence, so an ungated tick stacks without bound (#1658)")
	}
	ungated := []string{
		scan.ColdKind, scan.EdgeFanoutKind, scan.DNSKind, scan.ZoneKind,
		scan.TLSAcceptanceKind, scan.HTTPIdentityKind, scan.CTKind,
	}
	for _, kind := range ungated {
		if hotLagGateApplies(kind) {
			t.Errorf("the %s Scan must dispatch exactly as it did before the gate", kind)
		}
	}
}

type fakeHotLagStore struct {
	called    bool
	scanID    int64
	dispatch  int64
	lagging   bool
	returnErr error
}

func (f *fakeHotLagStore) ScanHasNonTerminalJobs(_ context.Context, arg db.ScanHasNonTerminalJobsParams) (bool, error) {
	f.called = true
	f.scanID = arg.ScanID
	f.dispatch = arg.DispatchID
	return f.lagging, f.returnErr
}

func TestHotLagGateArmed(t *testing.T) {
	if HotLagGateArmed(0) {
		t.Error("a zero stale-job timeout disables the reaper, so the gate must not arm")
	}
	if HotLagGateArmed(-time.Minute) {
		t.Error("a negative stale-job timeout disables the reaper, so the gate must not arm")
	}
	if !HotLagGateArmed(DefaultStaleJobThreshold) {
		t.Error("the default stale-job timeout runs the reaper, so the gate must arm")
	}
}

func TestHotLagGateArmingMatchesStaleCutoff(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	for _, threshold := range []time.Duration{-time.Hour, -time.Nanosecond, 0, time.Nanosecond, time.Minute, DefaultStaleJobThreshold} {
		_, bounded := StaleCutoff(now, threshold)
		if got := HotLagGateArmed(threshold); got != bounded {
			t.Errorf("threshold %s: gate armed = %v, reaper bounded = %v; the two must agree", threshold, got, bounded)
		}
	}
}

func TestHotTickLagsWhenAnEarlierDispatchHasNotDrained(t *testing.T) {
	f := &fakeHotLagStore{lagging: true}
	lagging, err := hotTickLags(context.Background(), f, 7, 42, DefaultStaleJobThreshold, nil)
	if err != nil {
		t.Fatalf("hotTickLags: %v", err)
	}
	if !lagging {
		t.Fatal("a Scan holding non-terminal jobs from an earlier dispatch must hold the tick")
	}
	if f.scanID != 7 || f.dispatch != 42 {
		t.Errorf("gate asked about scan %d / dispatch %d, want scan 7 / dispatch 42", f.scanID, f.dispatch)
	}
}

func TestHotTickDoesNotLagWhenTheQueueIsDrained(t *testing.T) {
	f := &fakeHotLagStore{lagging: false}
	lagging, err := hotTickLags(context.Background(), f, 7, 42, DefaultStaleJobThreshold, nil)
	if err != nil {
		t.Fatalf("hotTickLags: %v", err)
	}
	if lagging {
		t.Fatal("a drained Scan must dispatch normally")
	}
	if !f.called {
		t.Error("an armed gate must ask the store")
	}
}

func TestHotTickFallsThroughWhenTheReaperIsDisabled(t *testing.T) {
	for _, threshold := range []time.Duration{0, -5 * time.Minute} {
		f := &fakeHotLagStore{lagging: true}
		var buf bytes.Buffer
		lagging, err := hotTickLags(context.Background(), f, 7, 42, threshold, log.New(&buf, "", 0))
		if err != nil {
			t.Fatalf("threshold %s: hotTickLags: %v", threshold, err)
		}
		if lagging {
			t.Errorf("threshold %s: an unarmed gate must never hold a tick", threshold)
		}
		if f.called {
			t.Errorf("threshold %s: an unarmed gate must not read the queue", threshold)
		}
		if !strings.Contains(buf.String(), "not armed") {
			t.Errorf("threshold %s: the fall-through must warn, logged %q", threshold, buf.String())
		}
	}
}

func TestHotTickLagsReportsAReadFailure(t *testing.T) {
	// A gate that did not answer must never fall through to dispatching anyway.
	want := errors.New("boom")
	f := &fakeHotLagStore{returnErr: want}
	if _, err := hotTickLags(context.Background(), f, 7, 42, DefaultStaleJobThreshold, nil); !errors.Is(err, want) {
		t.Fatalf("hotTickLags error = %v, want %v", err, want)
	}
}

type fakeDispatchGateStore struct {
	fakeHotLagStore
	statuses  []db.SetDispatchStatusParams
	statusErr error
}

func (f *fakeDispatchGateStore) SetDispatchStatus(_ context.Context, arg db.SetDispatchStatusParams) error {
	f.statuses = append(f.statuses, arg)
	return f.statusErr
}

func TestCadenceLagSkipRecordsSkippedNotFannedOut(t *testing.T) {
	f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: true}}
	skip, err := gateHotTick(context.Background(), f, scan.HotKind, 7, 42, DefaultStaleJobThreshold, nil)
	if err != nil {
		t.Fatalf("gateHotTick: %v", err)
	}
	if skip != SkipCadenceLag {
		t.Fatalf("skip = %q, want %q", skip, SkipCadenceLag)
	}
	if len(f.statuses) != 1 {
		t.Fatalf("the skip must stamp its claimed Dispatch exactly once, wrote %d", len(f.statuses))
	}
	got := f.statuses[0]
	if got.Status != DispatchStatusSkipped {
		t.Errorf("recorded status = %q, want %q", got.Status, DispatchStatusSkipped)
	}
	if got.Status == "fanned-out" {
		t.Error("a tick that fanned nothing out must not be recorded as fanned-out")
	}
	if got.ID != 42 {
		t.Errorf("stamped dispatch %d, want the claimed dispatch 42", got.ID)
	}
}

func TestDrainedHotTickRecordsNoSkip(t *testing.T) {
	f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: false}}
	skip, err := gateHotTick(context.Background(), f, scan.HotKind, 7, 42, DefaultStaleJobThreshold, nil)
	if err != nil {
		t.Fatalf("gateHotTick: %v", err)
	}
	if skip != SkipNone {
		t.Fatalf("skip = %q, want none", skip)
	}
	if len(f.statuses) != 0 {
		t.Errorf("a dispatched tick keeps the status TryFanOut wrote, wrote %+v", f.statuses)
	}
}

func TestUndrainedCTTailTickRecordsSkipped(t *testing.T) {
	f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: true}}
	skip, err := gateHotTick(context.Background(), f, scan.CTTailKind, 7, 42, DefaultStaleJobThreshold, nil)
	if err != nil {
		t.Fatalf("gateHotTick: %v", err)
	}
	if skip != SkipCadenceLag {
		t.Fatalf("skip = %q, want %q: an undrained ct-tail dispatch must skip the tick instead of stacking another (#1658)", skip, SkipCadenceLag)
	}
	if len(f.statuses) != 1 {
		t.Fatalf("the skip must stamp its claimed Dispatch exactly once, wrote %d", len(f.statuses))
	}
	if got := f.statuses[0]; got.ID != 42 || got.Status != DispatchStatusSkipped {
		t.Errorf("recorded %+v, want dispatch 42 as %q", got, DispatchStatusSkipped)
	}
	if f.scanID != 7 || f.dispatch != 42 {
		t.Errorf("gate asked about scan %d / dispatch %d, want scan 7 / dispatch 42", f.scanID, f.dispatch)
	}
}

func TestDrainedCTTailTickRecordsNoSkip(t *testing.T) {
	f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: false}}
	skip, err := gateHotTick(context.Background(), f, scan.CTTailKind, 7, 42, DefaultStaleJobThreshold, nil)
	if err != nil {
		t.Fatalf("gateHotTick: %v", err)
	}
	if skip != SkipNone {
		t.Fatalf("skip = %q, want none: a drained ct-tail dispatch fans out one job per log as before", skip)
	}
	if !f.called {
		t.Error("an armed gate must ask the store")
	}
	if len(f.statuses) != 0 {
		t.Errorf("a dispatched tick keeps the status TryFanOut wrote, wrote %+v", f.statuses)
	}
}

func TestUnarmedCTTailGateFallsThrough(t *testing.T) {
	for _, threshold := range []time.Duration{0, -5 * time.Minute} {
		f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: true}}
		var buf bytes.Buffer
		skip, err := gateHotTick(context.Background(), f, scan.CTTailKind, 7, 42, threshold, log.New(&buf, "", 0))
		if err != nil {
			t.Fatalf("threshold %s: gateHotTick: %v", threshold, err)
		}
		if skip != SkipNone || len(f.statuses) != 0 {
			t.Errorf("threshold %s: the reaper-disabled fall-through dispatches ct-tail as it does hot, got skip %q and %d status writes", threshold, skip, len(f.statuses))
		}
		if f.called {
			t.Errorf("threshold %s: an unarmed gate must not read the queue", threshold)
		}
		if !strings.Contains(buf.String(), "not armed") {
			t.Errorf("threshold %s: the fall-through must warn, logged %q", threshold, buf.String())
		}
	}
}

func TestUngatedKindsRecordNoSkip(t *testing.T) {
	ungated := []string{
		scan.ColdKind, scan.EdgeFanoutKind, scan.DNSKind, scan.ZoneKind,
		scan.TLSAcceptanceKind, scan.HTTPIdentityKind, scan.CTKind,
	}
	for _, kind := range ungated {
		f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: true}}
		skip, err := gateHotTick(context.Background(), f, kind, 7, 42, DefaultStaleJobThreshold, nil)
		if err != nil {
			t.Fatalf("%s: gateHotTick: %v", kind, err)
		}
		if skip != SkipNone || len(f.statuses) != 0 {
			t.Errorf("%s: the gate holds hot and ct-tail alone, got skip %q and %d status writes", kind, skip, len(f.statuses))
		}
		if f.called {
			t.Errorf("%s: an ungated kind must not read the queue", kind)
		}
	}
}

func TestUnarmedGateRecordsNoSkip(t *testing.T) {
	f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: true}}
	var buf bytes.Buffer
	skip, err := gateHotTick(context.Background(), f, scan.HotKind, 7, 42, 0, log.New(&buf, "", 0))
	if err != nil {
		t.Fatalf("gateHotTick: %v", err)
	}
	if skip != SkipNone || len(f.statuses) != 0 {
		t.Errorf("the reaper-disabled fall-through dispatches, got skip %q and %d status writes", skip, len(f.statuses))
	}
}

func TestCadenceLagSkipReportsAFailedRecording(t *testing.T) {
	// A skip the row does not carry is the unrecorded tick ADR-0137 §4 refuses.
	want := errors.New("boom")
	f := &fakeDispatchGateStore{fakeHotLagStore: fakeHotLagStore{lagging: true}, statusErr: want}
	if _, err := gateHotTick(context.Background(), f, scan.HotKind, 7, 42, DefaultStaleJobThreshold, nil); !errors.Is(err, want) {
		t.Fatalf("gateHotTick error = %v, want %v", err, want)
	}
}
