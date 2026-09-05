// Package seed holds the Declared-layer validation for a Seed — the operator's
// assertion of where the estate ends (v1 spec §3.2). It is database-free and pure, so
// the rules that decide a valid name or address scope are testable in isolation.
package seed

import (
	"fmt"
	"iter"
	"math/big"
	"math/bits"
	"net/netip"
	"strings"

	"golang.org/x/net/publicsuffix"
)

const DefaultAddressCap = 1024 // applied at declaration and read by no rule (§5.3)

func NormalizeDomain(input string) (string, error) {
	d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(input)), ".")
	if d == "" {
		return "", fmt.Errorf("a domain is required")
	}
	// Runs before publicsuffix, whose wildcard rule would pass crt.sh query injection.
	if !isLDH(d) {
		return "", fmt.Errorf("%q is not a bare domain — enter a registrable domain like example.com", input)
	}
	reg, err := publicsuffix.EffectiveTLDPlusOne(d)
	if err != nil {
		return "", fmt.Errorf("%q is not a registrable domain", input)
	}
	if reg != d {
		return "", fmt.Errorf("declare the registrable domain %s, not %s", reg, d)
	}
	return reg, nil
}

func isLDH(d string) bool {
	// IDN arrives as punycode, itself LDH, so this RFC 1035 allowlist loses no domain.
	for _, r := range d {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func ParseCIDR(input string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(strings.TrimSpace(input))
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%q is not a valid CIDR block, e.g. 203.0.113.0/24", input)
	}
	return p.Masked(), nil
}

func AddressCount(p netip.Prefix) *big.Int {
	// An IPv6 prefix covers more addresses than any fixed-width integer holds.
	hostBits := p.Addr().BitLen() - p.Bits()
	return new(big.Int).Lsh(big.NewInt(1), uint(hostBits))
}

func WithinCap(p netip.Prefix, maxAddrs int) bool {
	return AddressCount(p).Cmp(big.NewInt(int64(maxAddrs))) <= 0
}

func LargestPrefixLen(maxAddrs, familyBits int) int {
	if maxAddrs < 1 {
		return familyBits
	}
	hostBits := bits.Len(uint(maxAddrs)) - 1
	if hostBits > familyBits {
		hostBits = familyBits
	}
	return familyBits - hostBits
}

func EnumerateAddresses(p netip.Prefix) iter.Seq[netip.Addr] {
	// Streaming, so a scope above the cap fans out with bounded memory (ADR-0127).
	return func(yield func(netip.Addr) bool) {
		// A scope enumerates whole — broadcast included, never truncated at scan time (ADR-0047).
		p = p.Masked()
		if !p.IsValid() {
			return
		}
		// Next overflows to the invalid zero address, so the top of the space terminates.
		for a := p.Addr(); a.IsValid() && p.Contains(a); a = a.Next() {
			if !yield(a) {
				return
			}
		}
	}
}

const maxEnumCapHint = 1 << 16 // caps the size guess only, never a walk (ADR-0047)

func EnumCapHint(p netip.Prefix) int {
	// The Settings cap control prices a scope by its address count (#206).
	c := AddressCount(p)
	if !c.IsInt64() {
		return 0
	}
	n := c.Int64()
	if n <= 0 {
		return 0
	}
	if n > maxEnumCapHint {
		return maxEnumCapHint
	}
	return int(n)
}
