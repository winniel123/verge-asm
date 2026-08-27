package main

import (
	"os"
	"testing"

	"github.com/winniel123/verge-asm/internal/wire"
)

func TestRunEchoesBatchAndKind(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		_ = wire.EncodeJobSpec(inW, wire.JobSpec{Batch: "b1", Kind: "tcp-connect"})
		inW.Close()
	}()

	if err := run(inR, outW); err != nil {
		t.Fatalf("run: %v", err)
	}
	outW.Close()

	sc := wire.NewObservationScanner(outR)
	if !sc.Next() {
		t.Fatalf("expected one observation, scanner error: %v", sc.Err())
	}
	got := sc.Observation()
	if got.Batch != "b1" || got.Kind != "tcp-connect" {
		t.Fatalf("unexpected observation: %+v", got)
	}
}

// TestRunDispatchesHTTPExchange proves the http-exchange kind reaches the
// httpexchange leaf rather than falling through to the skeleton echo. A scope with
// zero targets makes the leaf produce zero observations (one http-identity per
// target), whereas the default echo would emit exactly one observation carrying
// the Kind — so an empty output stream is proof the dispatch case, not the
// default, handled the spec.
func TestRunDispatchesHTTPExchange(t *testing.T) {
	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		_ = wire.EncodeJobSpec(inW, wire.JobSpec{
			Batch: "b2",
			Kind:  "http-exchange",
			Scope: []byte(`{"vantage":"v1","vantage_class":"internal","targets":[]}`),
		})
		inW.Close()
	}()

	if err := run(inR, outW); err != nil {
		t.Fatalf("run: %v", err)
	}
	outW.Close()

	sc := wire.NewObservationScanner(outR)
	if sc.Next() {
		t.Fatalf("http-exchange with no targets should emit no observations; got %+v (default echo was not bypassed)", sc.Observation())
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
}
