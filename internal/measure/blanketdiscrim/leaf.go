// Package blanketdiscrim is the `blanket-discrimination` Derivation leaf inside
// the shared measurement binary (v1 spec §3.3, ADR-0104). It is the reachability
// twin of `wildcard-discrimination`: where that leaf decides whether a DNS answer
// can be trusted or is a parent's wildcard synthesis, this one decides whether a
// `reached` connect can be trusted or is a **blanket responder**'s artefact — a
// CDN, anycast front, or reverse-proxy edge that completes the TCP handshake on
// **every** port before deciding what to do with the connection.
//
// It decides one fact about an `Address`: does it answer TCP on ports that should
// be closed? The leaf sends a **control-port probe** — a batch-generated set of
// random high ports a well-behaved origin refuses (ports.go) — and the address is
// a **blanket responder** exactly where the whole control set answers. On such an
// address every `Service`'s `reachability` is **undiscriminated**, and an
// undiscriminated reach is a `Gap` (ADR-0066's rule, one facet over): not
// `reached`, not `not-reached`, but the absence of a value recording the sixth
// gap cause. The `reachability` value space is untouched — the closed pair
// `reached │ not-reached` (CONTEXT.md `Reach`) — because the undiscriminated case
// is a `Gap`, never a third value (ADR-0104 Decision §2, ADR-0015).
//
// This package is the pure decision half: the control-port set generator and the
// `Decide` verdict. The connect that drives the control probe, and the emission of
// the `reachability` `Gap`, live in `internal/measure/connectoutcome`, which
// imports this leaf and composes the control probe as a step in its exchange
// exactly as it composes the certificate handshake — so the leaf count moves from
// five to six with no import cycle and no change to the hot Scan's dispatch. The
// leaf is versioned separately and gated by its own golden corpus (ADR-0021,
// ADR-0085), so a break in the blanket verdict names this leaf and never
// `connect-outcome`.
//
// A published CDN/anycast prefix list is **refused** as the detector, and the
// refusal is the project's standing one (ADR-0002/0013/0089): a vendor's file on
// the critical path of the product's sharpest signal ages, false-negatives on a
// self-hosted reverse proxy, and asserts about the world what an instrument must
// measure. The control set measures the exact behaviour the false positive rests
// on and needs no list to maintain.
package blanketdiscrim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Version is the leaf's Derivation version (ADR-0008/ADR-0021). It moves on an
// output-affecting change and only on one, gated bidirectionally by this leaf's
// own golden corpus — separately from `connect-outcome`, so a break names its
// leaf.
const Version = "blanket-discrimination/v1"

// Kind is the leaf's name in the `reachability` Derivation vector (spanfold's
// facetVector). Unlike the other leaves it is NOT a JobSpec dispatch kind: the
// control probe and the service's own connect are decided inside one batch by the
// `connect-outcome` leaf's exchange (ADR-0104 Decision §2, rider 3), so the hot
// Scan keeps dispatching `connect-outcome` and this leaf composes into it. The
// constant names the vector component and this leaf's golden corpus.
const Kind = "blanket-discrimination"

// ControlResult is one control port's decided outcome, the closed union the
// verdict reads. It is `connect-outcome`'s raw connect result projected to the
// three cases the blanket decision turns on — kept local so this package imports
// nothing from `connect-outcome` and the two leaves stay acyclic. The caller
// (connect-outcome) maps its `ConnResult` onto these.
type ControlResult string

const (
	ControlAnswered ControlResult = "answered"
	// ControlClosed: the control port was refused (RST) — a decided negative. One
	// refused control port witnesses port-specific behaviour, which is exactly what
	// a blanket responder does not have, so it rules the address NOT blanket.
	ControlClosed ControlResult = "closed"
	// ControlIncomplete: the control connect did not decide — it timed out after
	// its retries, or a local error blinded us. We could not run the discrimination,
	// so the reach is a `Gap` for the same sixth cause (ADR-0104 Decision §2, rider
	// 3), exactly as an incomplete wildcard control probe yields a `Gap`.
	ControlIncomplete ControlResult = "incomplete"
)

type Verdict string

const (
	// VerdictBlanket: the whole control set answered. The address completes the
	// handshake on ports that should be closed, so every reach it offers is
	// undiscriminated and folds to a `Gap`.
	VerdictBlanket Verdict = "Blanket"
	// VerdictNotBlanket: at least one control port was refused. The address
	// discriminates by port — a real origin — so its reaches are trustworthy values.
	VerdictNotBlanket Verdict = "NotBlanket"
	// VerdictGap: the control probe did not complete (no refusal, and not every
	// control port answered). We could not discriminate, so the reach is a `Gap` —
	// never a value.
	VerdictGap Verdict = "Gap"
)

// Decide reads a Verdict from one address's control-port results. It errs by
// construction toward NOT fabricating surface: a single refused control port —
// one witness of port-specific behaviour — rules the address NotBlanket, and only
// a control set that answers with no dissent is a blanket responder. A probe that
// neither refuses nor fully answers (silence on some ports) is a `Gap`: we could
// not run the discrimination, and an undiscriminated reach is never a value
// (ADR-0104 Decision §2).
//
// The direction of caution is the model's own (ADR-0068, ADR-0104 Thin ground): a
// false `Gap` withholds one reach value; a false `reached` fabricates surface and
// fires the sharpest signal on noise. Requiring unanimity to call blanket keeps
// the false-blanket rate low; requiring a refusal to clear it keeps a filtered
// host from reading as a clean origin.
func Decide(results []ControlResult) Verdict {
	if len(results) == 0 {
		// No control ports were probed at all — the discrimination was not requested
		// (an empty PortGen), not a probe that ran and could not decide. Nothing to
		// gap: the connect value passes through. Production always draws a full set
		// (CryptoPorts), so this is the test/opt-out path, never a live estate.
		return VerdictNotBlanket
	}
	allAnswered := true
	for _, r := range results {
		switch r {
		case ControlClosed:
			// One port-specific refusal is enough: this is not a blanket responder.
			return VerdictNotBlanket
		case ControlAnswered:
			// keep scanning; unanimity is required for a blanket verdict.
		default: // ControlIncomplete
			allAnswered = false
		}
	}
	if allAnswered {
		return VerdictBlanket
	}
	return VerdictGap
}

// Params is this leaf's declared-parameter set for the golden-corpus gate
// (ADR-0021): the control-port count and the port band it draws from. The verdict
// predicate (Decide) is fixed in code and moves with the leaf version. A change
// here is a declared-parameter change that may justify a Version bump.
type Params struct {
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

// Gap cause and reasons. The reach `Gap` records the **sixth** gap cause
// (CONTEXT.md `Gap`): we could not discriminate this connect from an address that
// answers on every port. Cause is the machine tag every blanketed / undiscriminated
// reach `Gap` carries; the two reasons distinguish a measured blanket responder
// from a control probe that could not run, which are the same cause (ADR-0104
// Decision §2, rider 3) worded for the operator.
const (
	// GapCause is the sixth gap cause's tag, carried on every reach `Gap` this leaf
	// produces. It is what the surfacing (#254) reads to render the proxy-edge prose.
	GapCause = "blanket-responder"
	// ReasonBlanket is the reason on a reach gapped because the address is a
	// measured blanket responder.
	ReasonBlanket = "this address answers on all ports — it is a proxy edge, not your origin"
	// ReasonIncomplete is the reason on a reach gapped because the control probe did
	// not complete, so blanket-ness could not be decided.
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
