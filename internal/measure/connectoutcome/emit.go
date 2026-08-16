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

// GapOutcome is the `outcome` tag a reachability `Gap` carries — the marker the
// span fold reads to set `is_gap`, and NOT a member of `reachability`'s value
// space (that stays the closed pair `reached │ not-reached`, CONTEXT.md `Reach`).
// An undiscriminated reach on a blanket responder holds no value; this tag records
// its absence and the sixth gap cause (ADR-0104 Decision §2, blanketdiscrim).
const GapOutcome = "gap"

// reachabilityGapValue is the JSON payload of a reachability `Gap` observation: the
// gap marker, the sixth-cause tag, and the operator-facing reason. It carries no
// `reached │ not-reached` value, because a blanketed connect witnesses no listener
// we can attribute to the origin — the reach is undiscriminated, and an
// undiscriminated reach is never a value (ADR-0104). It carries no per-port connect
// result either: a blanketed Service's own port is deliberately NOT probed (the
// control probe already decided the verdict, and skipping the port connect saves
// the budget the ADR prices), so there is no raw result to record.
type reachabilityGapValue struct {
	Outcome string `json:"outcome"` // always GapOutcome
	Cause   string `json:"cause"`   // the sixth gap cause tag (blanketdiscrim.GapCause)
	Reason  string `json:"reason"`  // operator-facing prose (blanketdiscrim.ReasonFor)
}

// EmitServiceGap renders one TCP `Service`'s reachability `Gap` at one Vantage into
// an NDJSON observation — the value a `Service` takes on a blanket responder (or
// where the blanket control probe did not complete). The subject and facet are the
// Service's exactly as EmitService's are; the value is a `Gap`, not `reached` /
// `not-reached`, so the span fold opens an `is_gap` span and every downstream
// reader (the signal, inventory, Exposure) sees the leg as absent without a special
// case (ADR-0104 Decision §3). The Service's own port is not probed — the verdict is
// the control probe's — so the value carries no connect result.
func EmitServiceGap(batch, vantage string, target netip.AddrPort, cause, reason string) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Facet:   FacetReachability,
		Subject: ServiceKey(target, "tcp"),
		Vantage: vantage,
		Address: target.Addr().String(),
		Data:    mustJSON(reachabilityGapValue{Outcome: GapOutcome, Cause: cause, Reason: reason}),
	}
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
