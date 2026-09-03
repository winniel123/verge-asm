package wildcarddiscrim

import (
	"encoding/json"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

const (
	OutcomeShadowed = "Shadowed"
	OutcomeGap      = "Gap"
)

type resolutionValue struct {
	Outcome   string   `json:"outcome"`
	Addresses []string `json:"addresses,omitempty"`
}

type dnsRecordValue struct {
	Outcome    string         `json:"outcome,omitempty"`
	RRs        []rw.RR        `json:"rrs,omitempty"`
	Delegation *rw.Delegation `json:"delegation,omitempty"`
}

func Emit(batch, vantage string, res rw.Result, verdict Verdict) []wire.Observation {
	// Shadowed is all-or-nothing across a Name's qtypes, made structural here (ADR-0068).
	obs := make([]wire.Observation, 0, len(res.Records)+1)

	obs = append(obs, wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Facet:   rw.FacetResolution,
		Subject: res.Name,
		Vantage: vantage,
		Data:    mustJSON(resolutionFor(res, verdict)),
	})

	for _, rec := range res.Records {
		obs = append(obs, wire.Observation{
			Batch:         batch,
			Kind:          Kind,
			Facet:         rw.FacetDNSRecord,
			Subject:       res.Name,
			Discriminator: string(rec.Qtype),
			Vantage:       vantage,
			Data:          mustJSON(dnsRecordFor(res, rec, verdict)),
		})
	}
	return obs
}

func resolutionFor(res rw.Result, verdict Verdict) resolutionValue {
	switch verdict {
	case VerdictShadowed:
		return resolutionValue{Outcome: OutcomeShadowed}
	case VerdictGap:
		return resolutionValue{Outcome: OutcomeGap}
	default:
		return resolutionValue{Outcome: string(res.Resolution.Outcome), Addresses: res.Resolution.Addresses}
	}
}

func dnsRecordFor(res rw.Result, rec rw.Record, verdict Verdict) dnsRecordValue {
	// A suppressed citation carries no RRset, so its Addresses leave the estate (§8.2 W7).
	switch verdict {
	case VerdictShadowed:
		return dnsRecordValue{Outcome: OutcomeShadowed}
	case VerdictGap:
		return dnsRecordValue{Outcome: OutcomeGap}
	default:
		val := dnsRecordValue{RRs: rec.RRs}
		if rec.Qtype == rw.QtypeNS {
			d := res.Delegation
			val.Delegation = &d
		}
		return val
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("wildcarddiscrim: marshal observation value: " + err.Error())
	}
	return b
}
