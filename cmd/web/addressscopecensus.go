package main

import (
	"context"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/queue"
)

type addressScopeCensusStore interface {
	queue.EdgeFanoutStore
	queue.AddressExclusionStore
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
}

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

	fanout, err := queue.ReadEdgeFanout(ctx, q, queue.EdgeFanoutUnbounded())
	if err != nil {
		return nil, err
	}

	// An exclusion narrows the count on read and prunes no observation (ADR-0133 §7).
	excluded, err := queue.ReadAddressExclusions(ctx, q)
	if err != nil {
		return nil, err
	}

	// Reusing custodyExtensionEstate would walk resolutions per request for a field this limb ignores.
	estate := custody.Estate{AddressScopes: prefixes}.
		WithAddressExclusions(excluded).
		WithEdgeFanout(fanout)
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
