package edgefanout

import (
	"encoding/json"
	"net/netip"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/wire"
)

// edgeValue is the JSON payload of an `edge-fanout` observation: the outcome tag, plus
// — only on `presented` — the fingerprint of the certificate the edge served. The three
// negatives carry no fingerprint, because each is a value in its own right and not a
// half-read certificate.
type edgeValue struct {
	Outcome     Outcome `json:"outcome"`
	Fingerprint string  `json:"fingerprint,omitempty"`
}

// Emit renders one candidate edge's measurement into an NDJSON observation.
//
// The line carries NO facet, NO subject and NO discriminator: this leaf decides
// membership and opens no timeline, so there is nothing for a facet's four-part key to
// name (ADR-0129 §6, #954 amendment). It carries no vantage either — the default
// certificate is not a function of vantage, and vantage-varying fan-out is anycast, out
// of v1 (§5). What it names is the `Address` it measured; the port and transport are
// the leaf's fixed 443/tcp, not a per-line datum. The result is recorded on the `Batch`
// by content, and the `Custody` derivation composes it from there.
//
// On `presented` the leaf DER rides BESIDE the value as CertMaterial, never inside it:
// the observation records only the fingerprint, so ADR-0027's fence stays closed, and
// the worker lands the DER in the `certificate_material` side store keyed by that same
// fingerprint. The SAN set the fan-out reduction counts is read back from there (#984).
// A scripted handshaker with no DER carries no material, so a hermetic row stays
// byte-identical.
func Emit(batch string, target netip.AddrPort, res Result) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Address: target.Addr().String(),
		Data: mustJSON(edgeValue{
			Outcome:     res.Outcome,
			Fingerprint: res.Fingerprint,
		}),
		CertMaterial: presentedMaterial(res),
	}
}

// presentedMaterial captures the served leaf's DER for the fingerprint-keyed side
// store, carried only on a presented handshake whose DER we actually read. It carries
// no SCTs and no issuer SPKI: this leaf exists to read a SAN set off the leaf, and CT
// verification rides the `certificate` facet's handshake, never this one (ADR-0129,
// "CT never binds to the IP").
func presentedMaterial(res Result) *wire.CertMaterial {
	if res.Outcome != Presented || len(res.LeafDER) == 0 {
		return nil
	}
	return &wire.CertMaterial{
		// The side store's key is DEFINED as the sha-256 of the DER it holds, so it is
		// computed here from those very bytes rather than copied from the presented
		// chain. In production the two agree; recomputing makes the invariant
		// structural, so no path can land a DER under a key that is not its hash and
		// silently break the fingerprint join.
		Fingerprint: co.Fingerprint(res.LeafDER),
		DER:         res.LeafDER,
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic("edgefanout: marshal observation value: " + err.Error())
	}
	return b
}
