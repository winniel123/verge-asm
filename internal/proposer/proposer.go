// Package proposer holds the keyless registry proposer paths (v1 spec §3.1, ADR-0012).
// A proposer admits nothing: it observes no facet, so what it yields is never an
// Observation, and until the operator confirms it, it is read by nothing.
package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/netip"
)

const ( // the operator judges the caveats, so the kind is recorded, never erased (ADR-0012)
	RecordRIRDelegation         = "rir-delegation"
	RecordCompelledReassignment = "compelled-reassignment" // the name is typed by the ISP, not the RIR
)

const ( // these match the source catalogue's slugs, so the enablement state keys line up
	SlugARIN    = "arin"
	SlugAFRINIC = "afrinic"
	SlugAPNIC   = "apnic-caida"
)

type Candidate struct {
	SourceSlug string
	RecordKind string
	Scope      netip.Prefix
	OrgName    string
}

type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Source interface {
	Slug() string
	Propose(ctx context.Context, orgName string) ([]Candidate, error)
}

type Registry struct {
	sources []Source
}

func NewRegistry(sources ...Source) *Registry {
	return &Registry{sources: sources}
}

func DefaultRegistry(doer Doer) *Registry {
	return NewRegistry(
		NewARIN(doer, "https://rdap.arin.net/registry"),
		NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic"),
		NewCAIDA(doer, SlugAPNIC, "apnic", "https://api.caida.org/as2org/v1", "https://ftp.apnic.net/stats/apnic"),
	)
}

func (r *Registry) Propose(ctx context.Context, orgName string, enabled map[string]bool) ([]Candidate, error) {
	var out []Candidate
	var errs []error
	// A source's failure costs its own coverage, never the whole search.
	for _, s := range r.sources {
		if !enabled[s.Slug()] {
			continue
		}
		cands, err := s.Propose(ctx, orgName)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.Slug(), err))
			continue
		}
		out = append(out, cands...)
	}
	return out, errors.Join(errs...)
}

func rangeToPrefixes(start netip.Addr, count *big.Int) ([]netip.Prefix, error) {
	// A delegated-stats count need not be a power of two, so a single /n would be invented.
	if !start.IsValid() || count.Sign() <= 0 {
		return nil, fmt.Errorf("empty or invalid range at %v", start)
	}
	bits := start.BitLen()
	remaining := new(big.Int).Set(count)
	cur := start
	var out []netip.Prefix
	for remaining.Sign() > 0 {
		maxByAlign := alignmentBlock(cur, bits)
		maxByCount := largestPow2AtMost(remaining, bits)
		block := maxByAlign
		if block.Cmp(maxByCount) > 0 {
			block = maxByCount
		}
		size := new(big.Int).Set(block)
		prefixLen := bits - size.BitLen() + 1
		out = append(out, netip.PrefixFrom(cur, prefixLen))
		remaining.Sub(remaining, size)
		if remaining.Sign() == 0 {
			break
		}
		next, ok := addrAdd(cur, size)
		if !ok {
			return out, fmt.Errorf("range overflowed the address space at %v", cur)
		}
		cur = next
	}
	return out, nil
}

func alignmentBlock(addr netip.Addr, bits int) *big.Int {
	v := addrToInt(addr)
	if v.Sign() == 0 {
		return new(big.Int).Lsh(big.NewInt(1), uint(bits))
	}
	tz := v.TrailingZeroBits()
	return new(big.Int).Lsh(big.NewInt(1), tz)
}

func largestPow2AtMost(n *big.Int, bits int) *big.Int {
	if n.Sign() <= 0 {
		return big.NewInt(0)
	}
	exp := uint(n.BitLen() - 1)
	if exp > uint(bits) {
		exp = uint(bits)
	}
	return new(big.Int).Lsh(big.NewInt(1), exp)
}

func addrToInt(addr netip.Addr) *big.Int {
	b := addr.As16()
	if addr.Is4() {
		v4 := addr.As4()
		return new(big.Int).SetBytes(v4[:])
	}
	return new(big.Int).SetBytes(b[:])
}

func addrAdd(addr netip.Addr, delta *big.Int) (netip.Addr, bool) {
	v := new(big.Int).Add(addrToInt(addr), delta)
	if addr.Is4() {
		if v.BitLen() > 32 {
			return netip.Addr{}, false
		}
		var b [4]byte
		v.FillBytes(b[:])
		return netip.AddrFrom4(b), true
	}
	if v.BitLen() > 128 {
		return netip.Addr{}, false
	}
	var b [16]byte
	v.FillBytes(b[:])
	return netip.AddrFrom16(b), true
}
