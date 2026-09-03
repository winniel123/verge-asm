package edgefanout

import (
	"encoding/json"
	"fmt"
	"net/netip"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/wire"
)

type Value struct {
	Outcome     Outcome `json:"outcome"`
	Fingerprint string  `json:"fingerprint,omitempty"`
}

func DecodeValue(raw json.RawMessage) (Value, error) {
	var v Value
	// internal/queue reads this back and re-gates the prober, so a bad line is dropped (#773).
	if err := json.Unmarshal(raw, &v); err != nil {
		return Value{}, fmt.Errorf("edgefanout: decode value: %w", err)
	}
	if !v.Outcome.Valid() {
		return Value{}, fmt.Errorf("edgefanout: outcome %q is outside the closed union", v.Outcome)
	}
	// A presented with no fingerprint is a half-read certificate; the store refuses the pair too.
	if (v.Outcome == Presented) != (v.Fingerprint != "") {
		return Value{}, fmt.Errorf("edgefanout: outcome %q carries fingerprint %q", v.Outcome, v.Fingerprint)
	}
	return v, nil
}

func Emit(batch string, target netip.AddrPort, res Result) wire.Observation {
	// No facet and no subject: this leaf decides membership and opens no timeline (ADR-0129 §6).
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind,
		Address: target.Addr().String(),
		Data: mustJSON(Value{
			Outcome:     res.Outcome,
			Fingerprint: res.Fingerprint,
		}),
		CertMaterial: presentedMaterial(res),
	}
}

func presentedMaterial(res Result) *wire.CertMaterial {
	// No SCTs and no issuer SPKI: CT rides the certificate facet, never this leaf (ADR-0129).
	if res.Outcome != Presented || len(res.LeafDER) == 0 {
		return nil
	}
	// The DER rides beside the value, never inside it, so ADR-0027's fence stays closed.
	return &wire.CertMaterial{
		// Recomputed from these bytes, never copied: no path can land a DER under a wrong key.
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
