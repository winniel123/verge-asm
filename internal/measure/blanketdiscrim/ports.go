package blanketdiscrim

import (
	"crypto/rand"
	"math/big"
	"sort"
)

// The control-port set is a batch-generated set of random ports drawn from the
// **dynamic** range (RFC 6335 §6, 49152–65535). No service is ever assigned there,
// so a well-behaved origin refuses an inbound connect to one; a blanket responder
// completes the handshake regardless. The set exists to **falsify
// port-independence** exactly as ADR-0069's control labels falsify
// label-independence: an origin's open ports are port-specific, a blanket
// responder's are not, and the control set is what tells them apart. `62345` in
// the ADR-0104 repro is one draw from exactly this band.
//
// The count is a declared parameter (leaf.go Params) — eight ports, enough to
// falsify port-independence with margin (a honeypot answering one stray draw does
// not clear the unanimity Decide requires) while staying cheap against
// `connect-outcome`'s 50 conn/s per-host and 200 pkt/s global ceilings: eight
// extra connects per address, round-robin by host with the port tiers, is a small
// fraction of the budget the ADR-0066-priced control set is allotted. A structured
// decoy (ADR-0069's RFC-5737-style second label) is deliberately omitted in v1:
// the dynamic band alone carries *long random*'s defence against accidental
// existence, and a decoy is "at most" optional per ADR-0104 Decision §1.
const (
	// ControlPortCount is how many distinct control ports one batch draws.
	ControlPortCount = 8
	// portBandLow / portBandHigh bound the RFC 6335 dynamic range the ports are
	// drawn from — the range IANA assigns to no service.
	portBandLow  uint16 = 49152
	portBandHigh uint16 = 65535
)

// PortGen produces one batch's control-port set. The ports are drawn per batch as
// independent samples; they never appear on any timeline (a control port is an
// input to the decision and a value on nothing), so a deterministic generator
// produces byte-identical observations for the golden corpus while production
// draws from crypto/rand.
type PortGen interface {
	// Ports returns exactly ControlPortCount distinct ports from the dynamic range,
	// sorted, so the control probe is order-stable within a batch.
	Ports() []uint16
}

// CryptoPorts is the production PortGen: crypto/rand ports from the dynamic range.
type CryptoPorts struct{}

// Ports implements PortGen. It draws ControlPortCount distinct ports uniformly
// from [portBandLow, portBandHigh].
func (CryptoPorts) Ports() []uint16 {
	span := int64(portBandHigh) - int64(portBandLow) + 1
	seen := map[uint16]struct{}{}
	for len(seen) < ControlPortCount {
		n, err := rand.Int(rand.Reader, big.NewInt(span))
		if err != nil {
			// crypto/rand does not fail in practice; a draw we cannot make is our own
			// blindness, and the caller reads a short set as an incomplete probe.
			break
		}
		seen[uint16(int64(portBandLow)+n.Int64())] = struct{}{}
	}
	return sortedPorts(seen)
}

// FixedPorts is a deterministic PortGen for the golden corpus and tests: exactly
// the ports it is constructed with, sorted and de-duplicated. It never draws from
// crypto/rand, so a corpus row renders byte-identically across runs and
// architectures.
type FixedPorts struct{ P []uint16 }

// Ports implements PortGen.
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
