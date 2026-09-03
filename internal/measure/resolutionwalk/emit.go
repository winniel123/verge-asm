package resolutionwalk

import (
	"encoding/json"
	"io"

	"github.com/winniel123/verge-asm/internal/wire"
)

const (
	FacetResolution = "resolution"
	FacetDNSRecord  = "dns-record"
)

type resolutionValue struct {
	Outcome   Outcome  `json:"outcome"`
	Addresses []string `json:"addresses,omitempty"`
}

type dnsRecordValue struct {
	RRs        []RR        `json:"rrs"`
	Delegation *Delegation `json:"delegation,omitempty"`
}

func Emit(batch, vantage string, res Result) []wire.Observation {
	// The golden corpus compares these lines byte for byte, so the order is fixed.
	obs := make([]wire.Observation, 0, len(res.Records)+1)

	obs = append(obs, wire.Observation{
		Batch:   batch,
		Kind:    "resolution-walk",
		Facet:   FacetResolution,
		Subject: res.Name,
		Vantage: vantage,
		Data:    mustJSON(resolutionValue{Outcome: res.Resolution.Outcome, Addresses: res.Resolution.Addresses}),
	})

	for _, rec := range res.Records {
		val := dnsRecordValue{RRs: rec.RRs}
		if rec.Qtype == QtypeNS {
			d := res.Delegation
			val.Delegation = &d
		}
		obs = append(obs, wire.Observation{
			Batch:         batch,
			Kind:          "resolution-walk",
			Facet:         FacetDNSRecord,
			Subject:       res.Name,
			Discriminator: string(rec.Qtype),
			Vantage:       vantage,
			Data:          mustJSON(val),
		})
	}
	return obs
}

func WriteNDJSON(w io.Writer, obs []wire.Observation) error {
	// The leaf's half of ADR-0001's job-spec-in / NDJSON-out contract.
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("resolutionwalk: marshal observation value: " + err.Error())
	}
	return b
}
