package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/wire"
)

type blockingProber struct{ ran chan struct{} }

func (b blockingProber) Probe(ctx context.Context, _ wire.JobSpec) (wire.ProbeResult, error) {
	select {
	case b.ran <- struct{}{}:
	default:
	}
	// ExecProber returns ctx.Err() on the run, which is the shape this fake reproduces.
	<-ctx.Done()
	return wire.ProbeResult{}, ctx.Err()
}

func TestProbeTimeoutUnblocksHungProber(t *testing.T) {
	bp := blockingProber{ran: make(chan struct{}, 1)}
	w := &Worker{prober: bp, probeTimeout: 20 * time.Millisecond}

	done := make(chan error, 1)
	go func() {
		_, err := w.probe(context.Background(), pgtype.Int8{}, wire.JobSpec{})
		done <- err
	}()

	select {
	case <-bp.ran:
	case <-time.After(2 * time.Second):
		t.Fatal("the prober was never invoked")
	}

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("probe error = %v, want context.DeadlineExceeded", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe did not return after its deadline; the drain loop would be blocked")
	}
}

func TestProbeTimeoutZeroUsesParentContext(t *testing.T) {
	bp := blockingProber{ran: make(chan struct{}, 1)}
	w := &Worker{prober: bp, probeTimeout: 0}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := w.probe(ctx, pgtype.Int8{}, wire.JobSpec{})
		done <- err
	}()

	select {
	case <-bp.ran:
	case <-time.After(2 * time.Second):
		t.Fatal("the prober was never invoked")
	}

	select {
	case err := <-done:
		t.Fatalf("probe returned %v before the parent was cancelled; the bound was not disabled", err)
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("probe error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("probe did not return after the parent was cancelled")
	}
}
