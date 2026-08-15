package resolutionwalk

import (
	"encoding/json"
	"io"

	"github.com/winniel123/verge-asm/internal/wire"
)

// Facet names, the two this leaf's Scan covers (ADR-0084).
const (
	FacetResolution = "resolution"
	FacetDNSRecord  = "dns-record"
)

// resolutionValue is the JSON payload of a resolution observation.
type resolutionValue struct {
	Outcome   Outcome  `json:"outcome"`
	Addresses []string `json:"addresses,omitempty"`
}

// dnsRecordValue is the JSON payload of a dns-record observation. Delegation is
// carried on the NS discriminator only — where the walk's Lame / per-nameserver
// verdict lives (measurement-offers.md §2, ADR-0011).
type dnsRecordValue struct {
	RRs        []RR        `json:"rrs"`
	Delegation *Delegation `json:"delegation,omitempty"`
}

// Emit renders one Result at one Vantage into the NDJSON observation lines the
// prober writes to stdout: one resolution line, plus one dns-record line per
// queried qtype. The lines are deterministic — the leaf sorts addresses, RRs and
// nameservers — so the golden corpus compares them byte for byte.
func Emit(batch, vantage string, res Result) []wire.Observation {
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

// WriteNDJSON writes observations to w as NDJSON, one object per line, in the
// order given. It is the leaf's half of ADR-0001's job-spec-in / NDJSON-out
// contract.
func WriteNDJSON(w io.Writer, obs []wire.Observation) error {
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
