package main

import (
	"context"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
)

// Vantage class is DERIVED per read from each vantage's persisted presented-address
// facts, never from the vestigial `class` column (#709 keystone (b)). The web layer
// has no Estate assembler of its own — only internal/queue/hot.go:hotEstate builds one
// at batch time — so the render/compose path assembles the same address-scope corpus
// here, from the same ListAddressScopeCidrs query, and offers the identical family-
// matched containment predicate to the class derivation (#711). One binding, used
// identically by batch gating and every render.

// addressScopeCovered reads the declared address-scope Seed CIDRs and returns the
// `covered` predicate the Vantage-class derivation binds — custody.Estate's family-
// matched CoversAddressScope over ADDRESS SCOPES ALONE (never the extension or
// MayProbe, #711). It mirrors internal/queue/hot.go:hotEstate's scope load exactly
// (drop nil rows into custody.Estate{AddressScopes}). A read failure returns the error;
// callers degrade the affected screen rather than 500ing.
func (s *server) addressScopeCovered(ctx context.Context) (func(netip.Addr) bool, error) {
	scopes, err := s.store.ListAddressScopeCidrs(ctx)
	if err != nil {
		return nil, err
	}
	var prefixes []netip.Prefix
	for _, p := range scopes {
		if p != nil {
			prefixes = append(prefixes, *p)
		}
	}
	estate := custody.Estate{AddressScopes: prefixes}
	return estate.CoversAddressScope, nil
}
