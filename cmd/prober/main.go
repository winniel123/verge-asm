// Command prober is the measurement binary: it reads one JobSpec from
// stdin and writes NDJSON Observations to stdout. It is exec'd locally for
// internal measurement and pushed over SSH for external vantages
// (ADR-0001).
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
		// The first live measurement: the resolution-walk leaf, dispatched
		// by the dns Scan (v1 spec §3.3/§3.4).
		return resolutionwalk.Run(spec, stdout)
	case wildcarddiscrim.Kind:
		// The wildcard-discrimination leaf: it reads the name's own answer and
		// a control probe under the name's parent on one declared path, deciding
		// Shadowed. It is versioned separately from resolution-walk and, with it,
		// is one of the two leaves membership composes (ADR-0086).
		return wildcarddiscrim.Run(spec, stdout)
	case connectoutcome.Kind:
		// The connect-outcome leaf: the daily hot Scan's TCP connect (never SYN),
		// deciding the reachability facet for each Service in scope. It is paced by
		// the §3.3 safety limiter, which never changes a verdict (ADR-0021).
		return connectoutcome.Run(spec, stdout)
	case tlsacceptance.Kind:
		// The tls-acceptance leaf: the weekly tls-acceptance Scan's TLS ENUMERATION
		// over the open Service population — version/cipher acceptance, its own
		// exchange, distinct from the certificate handshake that rides reachability
		// (ADR-0028). The candidate set travels in the job spec and is recorded on the
		// Batch by content (ADR-0025).
		return tlsacceptance.Run(spec, stdout)
	case httpexchange.Kind:
		// The http-exchange leaf: the STEP that rides the daily hot Scan's
		// reachability exchange — after connect-outcome reports `reached`, the
		// single `GET /` decides the `http-identity` facet for each `Endpoint`.
		// It sends the shared identifiable probe User-Agent and never follows a
		// 3xx (spec §3.3, ADR-0011/ADR-0025). Wiring this case is what lights the
		// four dormant HTTP identity rules.
		return httpexchange.Run(spec, stdout)
	case edgefanout.Kind:
		// The edge-fanout leaf: the seventh Scan's no-SNI TLS handshake, which
		// captures the certificate a candidate edge serves to a client that names
		// nothing. Like wildcard-discrimination it decides MEMBERSHIP rather than a
		// facet — it opens no timeline — and it runs BEFORE its target is a member,
		// which is why it rides neither the hot Scan nor tls-acceptance (ADR-0129 §6,
		// #954 amendment).
		return edgefanout.Run(spec, stdout)
	default:
		// The skeleton's job-spec-in / NDJSON-out proof for kinds whose leaf
		// a later ticket adds (TCP/TLS/HTTP).
		return wire.EncodeObservation(stdout, wire.Observation{
			Batch: spec.Batch,
			Kind:  spec.Kind,
		})
	}
}
