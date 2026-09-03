// Package blanketdiscrim is the `blanket-discrimination` Derivation leaf inside the
// shared measurement binary (v1 spec §3.3, ADR-0104). An undiscriminated reach is the
// sixth gap cause, never a third `reachability` value (ADR-0104 Decision §2, ADR-0015).
package blanketdiscrim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const Version = "blanket-discrimination/v1" // moves only with a moved golden row (ADR-0021)

const Kind = "blanket-discrimination" // composed by connect-outcome, not dispatched (ADR-0104 §2)

type ControlResult string // local so this leaf never imports connect-outcome; the two stay acyclic

const (
	ControlAnswered   ControlResult = "answered"
	ControlClosed     ControlResult = "closed"
	ControlIncomplete ControlResult = "incomplete"
)

type Verdict string

const (
	VerdictBlanket    Verdict = "Blanket"
	VerdictNotBlanket Verdict = "NotBlanket"
	VerdictGap        Verdict = "Gap"
)

func Decide(results []ControlResult) Verdict {
	// A false Gap withholds one reach; a false reach fabricates surface (ADR-0068, ADR-0104).
	if len(results) == 0 {
		// An empty set was never probed, not a probe that could not decide, so nothing is gapped.
		// Production always draws a full CryptoPorts set, so only a test or an opt-out reaches here.
		return VerdictNotBlanket
	}
	allAnswered := true
	for _, r := range results {
		switch r {
		case ControlClosed:
			// One port-specific refusal witnesses behaviour a blanket responder cannot have.
			return VerdictNotBlanket
		case ControlAnswered:
		default:
			allAnswered = false
		}
	}
	if allAnswered {
		return VerdictBlanket
	}
	return VerdictGap
}

type Params struct { // a field change may need a Version bump (ADR-0021)
	ControlPortCount int    `json:"control_port_count"`
	PortBandLow      uint16 `json:"port_band_low"`
	PortBandHigh     uint16 `json:"port_band_high"`
}

func DefaultParams() Params {
	return Params{
		ControlPortCount: ControlPortCount,
		PortBandLow:      portBandLow,
		PortBandHigh:     portBandHigh,
	}
}

func (p Params) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("blanketdiscrim: marshal params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

const ( // the sixth gap cause: two operator reasons, one cause (CONTEXT.md Gap, ADR-0104 §2)
	GapCause         = "blanket-responder" // the operator surfacing reads this tag (#254)
	ReasonBlanket    = "this address answers on all ports — it is a proxy edge, not your origin"
	ReasonIncomplete = "the control-port probe did not complete, so this reach could not be discriminated from a blanket responder"
)

func ReasonFor(v Verdict) string {
	switch v {
	case VerdictBlanket:
		return ReasonBlanket
	case VerdictGap:
		return ReasonIncomplete
	default:
		return ""
	}
}

func (v Verdict) Gaps() bool { return v == VerdictBlanket || v == VerdictGap }
