package tlsacceptance

import (
	"encoding/json"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/wire"
)

// acceptanceValue is the JSON payload of a `tls-acceptance` observation: the
// outcome tag, plus — only on an `enumerated` result — the accepted versions and
// their accepted suites. The value space is a closed union: an enumeration carries
// its accepted set, and the two negatives (`tls-refused`, `no-tls`) carry none,
// because each is a value in its own right and not an absence (CONTEXT.md
// `tls-acceptance`).
//
// The candidate SET is NOT part of the value — it is the Batch's recorded scope
// (OffersJSON) — so an offer of nine ciphers can never assert the tenth was
// refused, and widening the offer moves the recorded scope rather than the value's
// shape (CONTEXT.md `tls-acceptance`, ADR-0025).
type acceptanceValue struct {
	Outcome  AcceptanceOutcome   `json:"outcome"`
	Versions []VersionAcceptance `json:"versions,omitempty"`
}

// ServiceKey renders the `Service` subject key for one `(Address, port, transport)`
// triple: `address:port/transport`, e.g. `198.51.100.1:443/tcp` — the same triple
// the `connect-outcome` leaf renders, so a Service's `tls-acceptance` timeline
// names exactly the Service its `reachability` timeline is about (CONTEXT.md
// `Service`). `tls-acceptance` keys on the `Service`, never an `Endpoint`: SNI is
// not a candidate, so no name selects the subject here (measurement-offers §1.6).
func ServiceKey(target netip.AddrPort, transport string) string {
	return target.String() + "/" + transport
}

// EmitAcceptance renders one Service's enumeration result at one Vantage into an
// NDJSON observation. The subject is the `Service` triple; the facet is
// `tls-acceptance`; the value is the closed union `enumerated(versions) │
// tls-refused │ no-tls`. It is emitted for EVERY Service the Scan's scope carries —
// the open `Service` population — because each negative is a value the enumeration
// measured, not an absence: the Scan targets only Services already known open, so
// there is no closed-port variant to withhold (CONTEXT.md `tls-acceptance`).
func EmitAcceptance(batch, vantage string, target netip.AddrPort, value acceptanceValue) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Facet:   Facet,
		Subject: ServiceKey(target, "tcp"),
		Vantage: vantage,
		Address: target.Addr().String(),
		Data:    mustJSON(value),
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("tlsacceptance: marshal observation value: " + err.Error())
	}
	return b
}
