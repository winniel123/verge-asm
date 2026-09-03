package httpexchange

import (
	"encoding/json"

	"github.com/winniel123/verge-asm/internal/wire"
)

func EmitEndpoint(batch, vantage string, target Target, id HTTPIdentity) wire.Observation {
	// A reached Service is an Endpoint whether or not it speaks HTTP (CONTEXT.md, ADR-0011).
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
