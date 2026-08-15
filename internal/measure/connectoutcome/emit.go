package connectoutcome

import (
	"encoding/json"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/wire"
)

// FacetReachability is the facet this leaf covers.
const FacetReachability = "reachability"

// ServiceKey renders the `Service` subject key for one `(Address, port,
// transport)` triple: `address:port/transport`, e.g. `198.51.100.1:443/tcp`.
// The Address is the subject and is rendered from its Address key, never restated
// from a string, so one host never gets two spellings (CONTEXT.md `Service`).
func ServiceKey(target netip.AddrPort, transport string) string {
	return target.String() + "/" + transport
}

// reachabilityValue is the JSON payload of a reachability observation: the
// verdict, plus the raw connect result as evidence. The differ reads `outcome`;
// `result` is carried for the person asking *what did we actually measure*.
type reachabilityValue struct {
	Outcome Outcome    `json:"outcome"`
	Result  ConnResult `json:"result"`
}

// EmitService renders one TCP `Service`'s reachability verdict at one Vantage
// into an NDJSON observation. The subject is the `(Address, port, tcp)` triple;
// the facet is `reachability`; the value is `reached │ not-reached`. A `Service`
// exists — and therefore an observation is emitted — for the pair whether it is
// open or closed, which is what gives `unreachable` a subject to be a verdict
// about (CONTEXT.md `Service`).
func EmitService(batch, vantage string, target netip.AddrPort, outcome Outcome, raw ConnResult) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Facet:   FacetReachability,
		Subject: ServiceKey(target, "tcp"),
		Vantage: vantage,
		Address: target.Addr().String(),
		Data:    mustJSON(reachabilityValue{Outcome: outcome, Result: raw}),
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("connectoutcome: marshal observation value: " + err.Error())
	}
	return b
}
