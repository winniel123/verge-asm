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

// The gate is hot-only. cold and edge-fanout stream their fan-out too and lag the same
// way, but neither connects to a target, so gating them would be a scheduling change
// under a safety ticket. Every other kind must be left exactly as it was.
func TestHotLagGateAppliesToTheHotTierAlone(t *testing.T) {
	if !hotLagGateApplies(scan.HotKind) {
		t.Error("the hot Scan carries the per-target ceiling, so the gate must apply to it")
	}
	ungated := []string{
		scan.ColdKind, scan.EdgeFanoutKind, scan.DNSKind, scan.ZoneKind,
		scan.TLSAcceptanceKind, scan.HTTPIdentityKind, scan.CTKind, scan.CTTailKind,
	}
	for _, kind := range ungated {
		if hotLagGateApplies(kind) {
			t.Errorf("the %s Scan must dispatch exactly as it did before the gate", kind)
		}
	}
}

// fakeHotLagStore is the whole surface the gate can reach: the one non-terminal-job
// read. It records what the gate asked so a test can prove the current dispatch is
// excluded from the question.
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

// The gate and the reaper must agree about what "disabled" means, or the gate could
// arm over a reaper that will never terminate a wedged 'running' job.
func TestHotLagGateArmingMatchesStaleCutoff(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	for _, threshold := range []time.Duration{-time.Hour, -time.Nanosecond, 0, time.Nanosecond, time.Minute, DefaultStaleJobThreshold} {
		_, bounded := StaleCutoff(now, threshold)
		if got := HotLagGateArmed(threshold); got != bounded {
			t.Errorf("threshold %s: gate armed = %v, reaper bounded = %v; the two must agree", threshold, got, bounded)
		}
	}
}

// Cadence lag: the Scan still holds a 'ready' or 'running' job from an earlier
// dispatch, so the tick is held.
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

// A drained queue dispatches normally. The store answers false when every earlier job
// reached a terminal state — done, dead, retried or cancelled — so a dead-lettered
// backlog is covered by the same case and never wedges the next tick.
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

// The reaper-disabled configuration falls through to the pre-#1114 behaviour: the
// gate does not arm, it does not even ask, and it logs why the protection is off. A
// gate armed here would skip every future hot tick forever on one wedged 'running'
// row, which is a silent stop of all active measurement.
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

// A read failure is reported, never swallowed into "dispatch anyway": the caller
// aborts the tick rather than fan out past a gate that did not answer.
func TestHotTickLagsReportsAReadFailure(t *testing.T) {
	want := errors.New("boom")
	f := &fakeHotLagStore{returnErr: want}
	if _, err := hotTickLags(context.Background(), f, 7, 42, DefaultStaleJobThreshold, nil); !errors.Is(err, want) {
		t.Fatalf("hotTickLags error = %v, want %v", err, want)
	}
}
