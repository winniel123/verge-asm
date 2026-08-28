// Package seed holds the Declared-layer validation for a Seed — the operator's
// assertion of where the estate ends (v1 spec §3.2). It is database-free and
// pure so the rules that decide what a valid name or address scope is can be
// tested in isolation: a name scope is a registrable domain, an address scope
// is a CIDR whose address count is within the operator's cap, family-agnostic.
package seed

import (
	"fmt"
	"math/big"
	"net/netip"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// DefaultAddressCap is the default ceiling on the addresses an address scope
// may cover (v1 spec §3.2). It is operator-configurable; the value is applied
// at declaration and read by no rule (§5.3).
const DefaultAddressCap = 1024

// NormalizeDomain validates that input is a registrable domain (eTLD+1) and
// returns it lowercased and trailing-dot-stripped. It rejects a subdomain
// (`www.example.com`), a bare public suffix (`co.uk`), a URL, and a wildcard —
// anything that is not exactly its own registrable domain.
func NormalizeDomain(input string) (string, error) {
	d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(input)), ".")
	if d == "" {
		return "", fmt.Errorf("a domain is required")
	}
	// Strict LDH allowlist (RFC 1035): a registrable domain is only letters,
	// digits, hyphen and label-separating dots. Reject anything else BEFORE
	// trusting publicsuffix, whose wildcard rule would otherwise pass query-
	// injection characters (&, #, ;, ', %, whitespace) straight through into the
	// unencoded crt.sh query URL. IDN is carried as punycode (xn--…), which is
	// itself LDH, so this preserves every legitimate domain (#774).
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

// isLDH reports whether d contains only the characters legal in a DNS domain
// name: ASCII letters, digits, hyphen, and the dot label separator (the LDH
// rule, RFC 1035). d is expected already lowercased. It is an allowlist, not a
// blocklist: any character outside [a-z0-9.-] is rejected.
func isLDH(d string) bool {
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

// ParseCIDR parses input as a CIDR block and returns it in canonical masked
// form (host bits cleared), so `10.0.0.5/24` is stored as `10.0.0.0/24`.
func ParseCIDR(input string) (netip.Prefix, error) {
	p, err := netip.ParsePrefix(strings.TrimSpace(input))
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%q is not a valid CIDR block, e.g. 203.0.113.0/24", input)
	}
	return p.Masked(), nil
}

// AddressCount returns how many addresses a prefix covers. It uses big.Int
// because an IPv6 prefix can cover more addresses than any fixed-width integer
// holds.
func AddressCount(p netip.Prefix) *big.Int {
	hostBits := p.Addr().BitLen() - p.Bits()
	return new(big.Int).Lsh(big.NewInt(1), uint(hostBits))
}

// WithinCap reports whether p covers at most maxAddrs addresses. The cap
// counts addresses regardless of family, so a /22 and an equivalently-sized
// IPv6 block are treated the same.
func WithinCap(p netip.Prefix, maxAddrs int) bool {
	return AddressCount(p).Cmp(big.NewInt(int64(maxAddrs))) <= 0
}
