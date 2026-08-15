// Command prober is the measurement binary: it reads one JobSpec from
// stdin and writes NDJSON Observations to stdout. It is exec'd locally for
// internal measurement and pushed over SSH for external vantages
// (ADR-0001).
package main

import (
	"io"
	"log"
	"os"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("prober: %v", err)
	}
}

func run(stdin io.Reader, stdout io.Writer) error {
	spec, err := wire.DecodeJobSpec(stdin)
	if err != nil {
		return err
	}

	switch spec.Kind {
	case resolutionwalk.Kind:
		// The first live measurement: the resolution-walk leaf, dispatched
		// by the dns Scan (v1 spec §3.3/§3.4).
		return resolutionwalk.Run(spec, stdout)
	default:
		// The skeleton's job-spec-in / NDJSON-out proof for kinds whose leaf
		// a later ticket adds (TCP/TLS/HTTP).
		return wire.EncodeObservation(stdout, wire.Observation{
			Batch: spec.Batch,
			Kind:  spec.Kind,
		})
	}
}
