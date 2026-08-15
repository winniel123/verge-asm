// Command prober is the measurement binary: it reads one JobSpec from
// stdin and writes NDJSON Observations to stdout. It is exec'd locally for
// internal measurement and pushed over SSH for external vantages
// (ADR-0001).
package main

import (
	"log"
	"os"

	"github.com/winniel123/verge-asm/internal/wire"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("prober: %v", err)
	}
}

func run(stdin *os.File, stdout *os.File) error {
	spec, err := wire.DecodeJobSpec(stdin)
	if err != nil {
		return err
	}

	// The skeleton emits no real measurement yet — later tickets add the
	// DNS/TCP/TLS/HTTP kinds. This proves the stdin JobSpec -> stdout
	// NDJSON contract end to end.
	return wire.EncodeObservation(stdout, wire.Observation{
		Batch: spec.Batch,
		Kind:  spec.Kind,
	})
}
