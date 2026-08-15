package httpexchange

import (
	"encoding/json"

	"github.com/winniel123/verge-asm/internal/wire"
)

// EmitEndpoint renders one `(Name, Service)` pair's HTTP identity into an NDJSON
// observation. The subject is the `Endpoint` key — `name@service`, or `@service`
// for the nameless endpoint; the facet is `http-identity`; the value is the folded
// HTTPIdentity. An Endpoint observation is emitted only for a pair whose exchange
// completed, which is what makes "an Endpoint exists for every pair with a
// successful HTTP exchange" a structural property (CONTEXT.md `Endpoint`).
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
