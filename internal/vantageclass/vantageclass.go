// Package vantageclass derives a Vantage's class per read from its persisted
// presented-address facts, never from the vestigial vantage.class column (#709).
// It is the one seam every classification site calls, so all sites agree.
package vantageclass

import (
	"net/netip"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/exposure"
)

func PresentedAddrs(dialled, egress string) []netip.Addr {
	out := make([]netip.Addr, 0, 2)
	// An unparseable fact drops out; the residue is disclosed by a smaller set, not closed (#710).
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

func Derive(dialled, egress string, covered func(netip.Addr) bool) custody.VantageClass {
	// The predicate must be address-scope only, never the custody extension or reachability (#711).
	return exposure.VerifyClass(PresentedAddrs(dialled, egress), covered)
}
