package wildcarddiscrim

import (
	"encoding/json"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

// OutcomeShadowed and OutcomeGap are the resolution/dns-record outcome strings
// this leaf writes. `Shadowed` cites no `Address`; `Gap` is the value an
// undiscriminated answer takes when the control probe did not complete.
const (
	OutcomeShadowed = "Shadowed"
	OutcomeGap      = "Gap"
)

// resolutionValue is the JSON payload of a composed resolution observation.
type resolutionValue struct {
	Outcome   string   `json:"outcome"`
	Addresses []string `json:"addresses,omitempty"`
}

// dnsRecordValue is the JSON payload of a composed dns-record observation. Under
// `Shadowed` and `Gap` it carries **no RRset at all** — the whole point of the
// citation pin (golden-corpus.md §8.2 W7): every `Address` held only by a
// suppressed citation leaves the estate.
type dnsRecordValue struct {
	Outcome    string         `json:"outcome,omitempty"`
	RRs        []rw.RR        `json:"rrs,omitempty"`
	Delegation *rw.Delegation `json:"delegation,omitempty"`
}

// Emit renders one Name's composed decision at one Vantage into NDJSON
// observations: one resolution line plus one dns-record line per queried qtype,
// tagged with this leaf's kind. `Shadowed` is all-or-nothing across a Name's
// qtypes (ADR-0068) — it holds on resolution and on every dns-record
// discriminator or on none — which this emission makes structural rather than
// asserted.
func Emit(batch, vantage string, res rw.Result, verdict Verdict) []wire.Observation {
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

// resolutionFor composes the resolution value: `Shadowed` cites nothing, `Gap`
// cites nothing, and a not-`Shadowed` verdict passes resolution-walk's own value
// through unchanged — the membership-deciding value is one or the other and the
// leaf that decided it is this one exactly when the value is Shadowed or Gap.
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
