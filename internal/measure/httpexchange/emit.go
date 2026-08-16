package httpexchange

import (
	"encoding/json"

	"github.com/winniel123/verge-asm/internal/wire"
)

// EmitEndpoint renders one `(Name, Service)` pair's HTTP identity into an NDJSON
// observation. The subject is the `Endpoint` key — `name@service`, or `@service`
// for the nameless endpoint; the facet is `http-identity`; the value is the folded
// HTTPIdentity, `responded` or `no-http-response`. An Endpoint observation is
// emitted for every reached pair — a reached Service is an Endpoint whether or not
// it speaks HTTP (CONTEXT.md `Endpoint`, ADR-0011) — so its membership never turns
// on whether the HTTP exchange happened to complete.
func EmitEndpoint(batch, vantage string, target Target, id HTTPIdentity) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Facet:   FacetHTTPIdentity,
		Subject: target.EndpointKey(),
		Vantage: vantage,
		Address: target.Address,
		Data:    mustJSON(id),
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("httpexchange: marshal observation value: " + err.Error())
	}
	return b
}
