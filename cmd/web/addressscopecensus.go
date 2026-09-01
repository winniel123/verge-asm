package main

import (
	"context"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/queue"
)

// The address-scope contradiction row (ADR-0129's #956 amendment, #989) is the
// display half of the declaration limb, and it is the OTHER half of custodycensus.go.
// That file renders the addresses the custody extension declines. This one renders the
// addresses a `Seed` DECLARES and fan-out measured as shared.
//
// The two limbs are disjoint, so the two surfaces are separate. An address-scope
// address is not an extension member, and the §7 custody-extension panel is one of the
// three surfaces ADR-0129 refuses for this evidence. The other two are `Coverage`'s
// aperture statement — which counts what the instrument CANNOT report, and a declared
// shared edge is looked at and reported fine — and a coverage-class message, whose
// safety justification does not transfer where a Declared act stands behind the
// address. So nothing here fires a message, and nothing here moves a gate.
//
// The measurement is reached through queue.ReadEdgeFanout, the ONE read path from the
// leaf's store to the derivation, so this surface and the gate cannot disagree about
// which edges fan-out measured as shared.
//
// It does NOT reuse custodyExtensionEstate. That assembler reads the current cited
// addresses, which the extension limb needs to name a citing name and this limb has no
// use for: a declared address is a subject by the declaration, whether or not any name
// resolves to it. Reading resolutions here would put a per-request walk on /coverage to
// populate a field nothing reads.

// addressScopeCensusStore is the read surface the contradiction row needs: the
// declared address scopes, and the `edge-fanout` measurement through its one reader.
type addressScopeCensusStore interface {
	queue.EdgeFanoutStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
}

// addressScopeSharedEdges reads how many addresses inside each declared address scope
// fan-out measured as shared, keyed by the masked scope. A scope with none is ABSENT
// from the map rather than present at zero, which is what makes the caller's lookup
// render a row only where the evidence exists.
//
// A read that FAILS returns the error, and the caller degrades to no row rather than
// fabricating one. Absence and zero are the same on this surface — that is the
// declaration limb's open-then-label absence rule (custody.Estate.AddressScopeCensus)
// — so a degraded read costs the operator a row they will see on the next load, and
// never a claim about a scope nothing measured.
func addressScopeSharedEdges(ctx context.Context, q addressScopeCensusStore) (map[netip.Prefix]int, error) {
	scopes, err := q.ListAddressScopeCidrs(ctx)
	if err != nil {
		return nil, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, p.Masked())
		}
	}

	fanout, err := queue.ReadEdgeFanout(ctx, q)
	if err != nil {
		return nil, err
	}

	// This estate carries no resolutions, so it holds NO extension candidates and
	// WithEdgeFanout resolves the extension limb's errored floor to *not errored*
	// (#1018). That is the right answer for this surface: the floor decides one
	// limb's reach, and the limb this census reads has none to open.
	estate := custody.Estate{AddressScopes: prefixes}.WithEdgeFanout(fanout)
	entries := estate.AddressScopeCensus()
	if len(entries) == 0 {
		return nil, nil
	}
	out := make(map[netip.Prefix]int, len(entries))
	for _, e := range entries {
		out[e.Scope] = e.SharedEdges
	}
	return out, nil
}
