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
