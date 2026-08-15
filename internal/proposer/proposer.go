// Package proposer holds the keyless registry proposer paths (v1 spec §3.1,
// ADR-0012). A proposer admits nothing: it does not observe a facet, so what it
// yields is never an Observation. It returns Candidate address scopes the
// operator may confirm into a Seed — a Proposal — and until that confirmation
// they are read by nothing.
//
// Every path here is keyless: it runs on availability alone, with no operator
// credential. The three shipped keyless paths are ARIN's entities?fn= org-name
// search and the AFRINIC and APNIC org->prefix paths built by joining CAIDA's
// org->opaque-id mapping to the RIR's delegated-extended-stats file.
//
// All network I/O goes through the injected Doer seam so the paths can be tested
// against fixtures with no real network, and the parsing is arch-neutral.
package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/netip"
)

// The two kinds of record a Proposal can be built from. They carry different
// caveats and the operator is the one judging them (ADR-0012), so the kind is
// recorded on every Candidate rather than erased.
const (
	// RecordRIRDelegation is a range delegated by the RIR to a holder — an
	// allocation or assignment in a delegated-stats file, or an ARIN org network.
	RecordRIRDelegation = "rir-delegation"
	// RecordCompelledReassignment is a range an upstream provider was compelled
	// to reassign downstream — an ARIN SWIP customer (C-handle) object, whose
	// name string is typed by the ISP rather than by the RIR.
	RecordCompelledReassignment = "compelled-reassignment"
)

// The slugs of the shipped keyless proposers. They match the source catalogue's
// slugs so the enablement state keys line up.
const (
	SlugARIN    = "arin"
	SlugAFRINIC = "afrinic"
	SlugAPNIC   = "apnic-caida"
)

// Candidate is one proposed address scope, before it is persisted as a Proposal.
// It is never an Observation: it records which kind of record produced it and
// the holder name it was offered under, and nothing about a measured facet.
type Candidate struct {
	SourceSlug string
	RecordKind string
	Scope      netip.Prefix
	OrgName    string
}

// Doer is the injected HTTP seam. Production supplies an *http.Client; tests
// supply a fake that returns fixture bodies, so no proposer touches the network
// under test.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Source is one keyless proposer path. It answers an operator's org-name search
// with the candidate scopes that path believes the org holds.
type Source interface {
	Slug() string
	Propose(ctx context.Context, orgName string) ([]Candidate, error)
}

// Registry is the set of shipped proposer paths. A lookup runs only the paths
// the operator has left enabled, so a source toggled off proposes nothing.
type Registry struct {
	sources []Source
}

// NewRegistry builds a registry over an explicit source set. Tests use it to
// inject fakes; DefaultRegistry wires the shipped paths.
func NewRegistry(sources ...Source) *Registry {
	return &Registry{sources: sources}
}

// DefaultRegistry wires the three shipped keyless paths against their real
// endpoints. The Doer carries whatever timeout the caller set.
func DefaultRegistry(doer Doer) *Registry {
	return NewRegistry(
		NewARIN(doer, "https://whois.arin.net/rest"),
		NewCAIDA(doer, SlugAFRINIC, "afrinic", "https://api.caida.org/as2org/v1", "https://ftp.afrinic.net/stats/afrinic"),
		NewCAIDA(doer, SlugAPNIC, "apnic", "https://api.caida.org/as2org/v1", "https://ftp.apnic.net/stats/apnic"),
	)
}

// Propose runs every enabled source for one org-name search and returns the
// union of their candidates. A source's failure is not fatal to the lookup — the
// others still answer — so a single registry outage costs coverage, never the
// whole search; the joined error is returned for the caller to surface.
func (r *Registry) Propose(ctx context.Context, orgName string, enabled map[string]bool) ([]Candidate, error) {
	var out []Candidate
	var errs []error
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

// rangeToPrefixes decomposes the half-open address range [start, start+count)
// into the minimal set of aligned CIDR prefixes that exactly cover it. A
// delegated-stats IPv4 record states a raw address count rather than a prefix
// length, and a holder's count need not be a single power of two, so a faithful
// conversion emits one prefix per aligned block rather than assuming a /n.
func rangeToPrefixes(start netip.Addr, count *big.Int) ([]netip.Prefix, error) {
	if !start.IsValid() || count.Sign() <= 0 {
		return nil, fmt.Errorf("empty or invalid range at %v", start)
	}
	bits := start.BitLen() // 32 for IPv4, 128 for IPv6
	remaining := new(big.Int).Set(count)
	cur := start
	var out []netip.Prefix
	for remaining.Sign() > 0 {
		// The largest aligned block startable at cur is bounded by cur's
		// alignment (its lowest set bit) and by how much is left to cover.
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

// alignmentBlock returns the size of the largest power-of-two block that can
// start at addr without crossing its natural alignment — i.e. the value of
// addr's lowest set bit, or the whole space when addr is the zero address.
func alignmentBlock(addr netip.Addr, bits int) *big.Int {
	v := addrToInt(addr)
	if v.Sign() == 0 {
		return new(big.Int).Lsh(big.NewInt(1), uint(bits))
	}
	// lowest set bit = v & -v, computed as the trailing-zero power of two.
	tz := v.TrailingZeroBits()
	return new(big.Int).Lsh(big.NewInt(1), tz)
}

// largestPow2AtMost returns the largest power of two that is <= n.
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

// addrAdd returns addr + delta, staying in addr's family. It reports false on
// overflow past the family's last address.
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
