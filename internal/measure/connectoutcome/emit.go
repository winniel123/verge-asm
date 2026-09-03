package connectoutcome

import (
	"encoding/json"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/wire"
)

const FacetReachability = "reachability"

func ServiceKey(target netip.AddrPort, transport string) string {
	// The Address renders from its own key, never a restated string, so one host has one spelling.
	return target.String() + "/" + transport
}

type reachabilityValue struct {
	Outcome Outcome    `json:"outcome"`
	Result  ConnResult `json:"result"`
}

// A marker outside reachability's closed pair: an undiscriminated reach holds no value (ADR-0104).

const GapOutcome = "gap"

// A blanketed Service's own port is never probed, which saves the budget ADR-0104 prices.

type reachabilityGapValue struct {
	Outcome string `json:"outcome"`
	Cause   string `json:"cause"`
	Reason  string `json:"reason"`
}

func EmitServiceGap(batch, vantage string, target netip.AddrPort, cause, reason string) wire.Observation {
	// The fold opens an is_gap span, so a downstream reader sees an absent leg with no special case.
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

func EmitService(batch, vantage string, target netip.AddrPort, outcome Outcome, raw ConnResult) wire.Observation {
	// A Service exists open or closed, so an unreachable verdict has a subject (CONTEXT.md Service).
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
