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

// A refused declaration names a route and never takes it (ADR-0052).

type WildcardError struct {
	Input   string
	Subtree string
}

func (e *WildcardError) Error() string {
	if e.Subtree == "" {
		return fmt.Sprintf("%q is a pattern over names, not a name — the object that does this job is a subtree exclusion", e.Input)
	}
	return fmt.Sprintf("%q is a pattern over names, not a name — the object that does this job is a subtree exclusion on %s", e.Input, e.Subtree)
}

type ULabelError struct {
	Input string
}

func (e *ULabelError) Error() string {
	// The A-label is never computed here: a refused value may not be rendered as advice (ADR-0052).
	return fmt.Sprintf("%q is not a form the DNS carries — an internationalised label travels as an ASCII form beginning xn--, and your DNS provider shows that form beside the name", e.Input)
}

func FoldASCII(s string) string {
	// Folding 0x41–0x5A alone is what DNS folds; strings.ToLower would fold U+0130 (ADR-0055).
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func hasHighBit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return true
		}
	}
	return false
}

func NormalizeDomain(input string) (string, error) {
	d := strings.TrimSuffix(FoldASCII(strings.TrimSpace(input)), ".")
	if d == "" {
		return "", fmt.Errorf("a domain is required")
	}
	// Only a leftmost label of exactly * is a wildcard (RFC 4592 §2.1.2); the rest fall to isLDH.
	if d == "*" || strings.HasPrefix(d, "*.") {
		sub := strings.TrimPrefix(d, "*.")
		if !isLDH(sub) {
			sub = ""
		}
		return "", &WildcardError{Input: input, Subtree: sub}
	}
	if hasHighBit(d) {
		return "", &ULabelError{Input: input}
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
