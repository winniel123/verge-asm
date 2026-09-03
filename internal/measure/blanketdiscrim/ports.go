package blanketdiscrim

import (
	"crypto/rand"
	"math/big"
	"sort"
)

const (
	ControlPortCount = 8 // eight falsifies with margin, cheap against the rate ceilings (ADR-0066)

	portBandLow  uint16 = 49152 // IANA assigns no service here, so an origin refuses (RFC 6335 §6)
	portBandHigh uint16 = 65535
)

type PortGen interface { // a control port is an input to the decision and a value on no timeline
	Ports() []uint16 // sorted, so the control probe is order-stable for the golden corpus (ADR-0021)
}

type CryptoPorts struct{} // a vendor CDN prefix list is refused as the detector (ADR-0013)

func (CryptoPorts) Ports() []uint16 {
	span := int64(portBandHigh) - int64(portBandLow) + 1
	seen := map[uint16]struct{}{}
	// A structured decoy is omitted in v1; the random band alone carries the defence (ADR-0104 §1).
	for len(seen) < ControlPortCount {
		n, err := rand.Int(rand.Reader, big.NewInt(span))
		if err != nil {
			// A short draw errs safe: an empty set Decides NotBlanket, so no Gap is fabricated.
			break
		}
		seen[uint16(int64(portBandLow)+n.Int64())] = struct{}{} // #nosec G115 (result in [portBandLow,portBandHigh], both <= 65535)
	}
	return sortedPorts(seen)
}

type FixedPorts struct{ P []uint16 }

func (f FixedPorts) Ports() []uint16 {
	seen := make(map[uint16]struct{}, len(f.P))
	for _, p := range f.P {
		seen[p] = struct{}{}
	}
	return sortedPorts(seen)
}

func sortedPorts(seen map[uint16]struct{}) []uint16 {
	out := make([]uint16, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
