package queue

import (
	"context"
	"net/netip"
)

// AddressExclusionStore is the narrow read surface ReadAddressExclusions needs. It
// exists so the render paths — which hold the web layer's own store interface and not
// a *db.Queries — reach the exclusions through the SAME reader the dispatcher uses,
// rather than growing a second one that could drop a row differently (the shape
// EdgeFanoutStore already established for the fan-out measurement).
type AddressExclusionStore interface {
	ListAddressExclusionCidrs(ctx context.Context) ([]*netip.Prefix, error)
}

// ReadAddressExclusions reads the declared `address` exclusion CIDRs in the form
// custody.Estate.WithAddressExclusions consumes (ADR-0133 §2). It is the ONE read
// path from the exclusion table to the derivation, so no assembler can narrow the
// address-scope limb by a different set than the gate does.
//
// Every assembler that builds an Estate over declared address scopes MUST pass the
// result through WithAddressExclusions. An assembler that skips it derives what it
// derived before ADR-0133, which is the safe reading and also the defect #1022
// recorded — so the omission is silent, and this comment is where it is written down.
//
// A NULL row is dropped, exactly as the scope read drops one. The column is
// constrained NOT NULL for an `address` row, so a NULL here is a row the query's own
// WHERE clause already refuses; the check is the same belt-and-braces the scope read
// carries.
func ReadAddressExclusions(ctx context.Context, q AddressExclusionStore) ([]netip.Prefix, error) {
	rows, err := q.ListAddressExclusionCidrs(ctx)
	if err != nil {
		return nil, err
	}
	var out []netip.Prefix
	for _, p := range rows {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}
