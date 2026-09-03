package tlsacceptance

import (
	"encoding/json"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/wire"
)

// The candidate set is the Batch's recorded scope, never part of the value (ADR-0025).

type acceptanceValue struct {
	Outcome  AcceptanceOutcome   `json:"outcome"`
	Versions []VersionAcceptance `json:"versions,omitempty"`
}

func ServiceKey(target netip.AddrPort, transport string) string {
	// The connect-outcome leaf renders the same triple, so the two timelines name one Service.
	return target.String() + "/" + transport
}

func EmitAcceptance(batch, vantage string, target netip.AddrPort, value acceptanceValue) wire.Observation {
	// The Scan's population is already-open Services, so there is no closed-port variant to withhold.
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
