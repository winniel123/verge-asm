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
