package queue

import (
	"context"
	"net/netip"
)

type AddressExclusionStore interface {
	ListAddressExclusionCidrs(ctx context.Context) ([]*netip.Prefix, error)
}

func ReadAddressExclusions(ctx context.Context, q AddressExclusionStore) ([]netip.Prefix, error) {
	// A second reader could narrow this limb by a different set than the gate does (ADR-0133 §2).
	rows, err := q.ListAddressExclusionCidrs(ctx)
	if err != nil {
		return nil, err
	}
	var out []netip.Prefix
	// An Estate assembler that skips WithAddressExclusions silently derives pre-ADR-0133 (#1022).
	for _, p := range rows {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}
