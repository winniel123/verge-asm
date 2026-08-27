// Package vantageclass derives a Vantage's class per read from its persisted
// presented-address facts, never from a stored column (#709 keystone (b): class is
// DERIVED per read, the `vantage.class` column is a vestige). It is the one shared
// seam every classification site calls — the batch-gating pass (internal/scan via
// internal/queue) and every render fold (cmd/web, internal/queue message producer) —
// so all sites agree on a vantage's class by construction.
//
// It takes the two presented facts as plain strings rather than a db row, so it stays
// storage-decoupled (like internal/exposure, which it defers to for the pure
// classification) and each call site adapts its own row type at the boundary. The
// facts are: the dialled peer address (vantage.dialled_addr, #710) and the SSH_CLIENT
// egress (vantage.egress, #683).
package vantageclass

import (
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/exposure"
)

// PresentedAddrs turns a vantage's two persisted presented-address facts — the dialled
// peer and the SSH_CLIENT egress — into the 0/1/2 addresses an outside observer saw of
// it, parsed and unmapped. Both are stored as canonical netip.Addr strings; a NULL/empty
// or unparseable fact simply drops out and is never fabricated (the dual-stack residue
// is disclosed by a smaller set, not closed — #710). This is exactly the set
// exposure.VerifyClass classifies.
func PresentedAddrs(dialled, egress string) []netip.Addr {
	out := make([]netip.Addr, 0, 2)
	for _, s := range [...]string{dialled, egress} {
		if s == "" {
			continue
		}
		if a, err := netip.ParseAddr(s); err == nil {
			out = append(out, a.Unmap())
		}
	}
	return out
}

// Derive classifies one vantage from its presented facts against the address-scope
// coverage predicate — exposure.VerifyClass over PresentedAddrs (#709). An empty
// presented set is `unverified` (no prober, or pre-connect); any one address uncovered
// by a declared address scope is `internet` (the closed direction); every address
// covered is `internal`. `covered` MUST be an address-scope-only predicate
// (custody.Estate.CoversAddressScope, #711) — never one that folds in the custody
// extension or non-global-reachability.
func Derive(dialled, egress string, covered func(netip.Addr) bool) custody.VantageClass {
	return exposure.VerifyClass(PresentedAddrs(dialled, egress), covered)
}
