// Command prober reads one JobSpec from stdin and writes NDJSON Observations to
// stdout. It runs locally for an internal vantage and over SSH for an external
// one (ADR-0001).
package main

import (
	"io"
	"log"
	"os"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
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
		return resolutionwalk.Run(spec, stdout)
	case wildcarddiscrim.Kind:
		// Versioned apart from resolution-walk, one of the two leaves membership composes (ADR-0086).
		return wildcarddiscrim.Run(spec, stdout)
	case connectoutcome.Kind:
		// Paced by the §3.3 safety limiter, which never changes a verdict (ADR-0021).
		return connectoutcome.Run(spec, stdout)
	case tlsacceptance.Kind:
		// Its own exchange, distinct from the certificate handshake that rides reachability (ADR-0028).
		return tlsacceptance.Run(spec, stdout)
	case httpexchange.Kind:
		return httpexchange.Run(spec, stdout)
	case edgefanout.Kind:
		// It runs before its target is a member, so neither Scan can carry it (ADR-0129 §6, #954).
		return edgefanout.Run(spec, stdout)
	default:
		// A kind with no leaf yet still answers job-spec-in / NDJSON-out (ADR-0001).
		return wire.EncodeObservation(stdout, wire.Observation{
			Batch: spec.Batch,
			Kind:  spec.Kind,
		})
	}
}
