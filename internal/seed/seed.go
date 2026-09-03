// Package seed holds the Declared-layer validation for a Seed — the operator's
// assertion of where the estate ends (v1 spec §3.2). It is database-free and
// pure so the rules that decide what a valid name or address scope is can be
// tested in isolation: a name scope is a registrable domain, an address scope
// is a CIDR whose address count is within the operator's cap, family-agnostic.
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

// DefaultAddressCap is the default ceiling on the addresses an address scope
// may cover (v1 spec §3.2). It is operator-configurable; the value is applied
// at declaration and read by no rule (§5.3).
const DefaultAddressCap = 1024

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

// LargestPrefixLen returns the length of the widest (shortest-prefix) block whose
// address count is within maxAddrs for an address family of familyBits (32 for
// IPv4, 128 for IPv6) — the largest scope the cap admits. The host bits are
// floor(log2(maxAddrs)), so the prefix is familyBits minus that. It is exact only
// on a power-of-two cap; on any other value it names the largest FULL prefix that
// fits, since a prefix is always a power of two of addresses (a cap of 1000 admits
// a /23's 512, not a /22's 1024). A maxAddrs below 1 admits no scope and returns
// familyBits — the host route, a single address. The host bits are clamped to
// familyBits so an oversized cap never returns a negative prefix.
func LargestPrefixLen(maxAddrs, familyBits int) int {
	if maxAddrs < 1 {
		return familyBits
	}
	hostBits := bits.Len(uint(maxAddrs)) - 1 // floor(log2)
	if hostBits > familyBits {
		hostBits = familyBits
	}
	return familyBits - hostBits
}

// EnumerateAddresses yields every address a prefix covers, in ascending order,
// as a lazy sequence. An address scope is its own enumeration (ADR-0047): a
// declared CIDR produces a probe target for every address inside it — the
// network and broadcast addresses included, since exempting them would infer a
// subnetting we never measure. The prefix is masked first, so a host-bits-set
// input (10.0.0.5/30) enumerates its block (10.0.0.4..10.0.0.7). The per-scope
// address cap (WithinCap, applied at declaration, §5.3) is what bounds this:
// callers enumerate a whole scope and never truncate at scan time, which
// ADR-0047 refuses as a silent aperture. The walk stops when Next overflows to
// the invalid zero address, so a scope at the very top of the address space
// terminates cleanly rather than looping.
//
// The sequence is streamed — it never materializes the whole scope in a slice —
// so an address scope above the cap (ADR-0127) fans out with bounded memory: the
// ceiling is not bound by memory. A consumer that stops early (breaks the range)
// stops the walk.
func EnumerateAddresses(p netip.Prefix) iter.Seq[netip.Addr] {
	return func(yield func(netip.Addr) bool) {
		p = p.Masked()
		if !p.IsValid() {
			return
		}
		for a := p.Addr(); a.IsValid() && p.Contains(a); a = a.Next() {
			if !yield(a) {
				return
			}
		}
	}
}

// maxEnumCapHint bounds the capacity hint an enumeration returns. It caps only
// the size guess, never a walk itself — ADR-0047 refuses a scan-time cap on the
// enumeration, and EnumerateAddresses streams whole regardless. The bound stops
// a pathological oversized scope (one declared under a since-lowered cap) from
// reporting a huge capacity from a single AddressCount read.
const maxEnumCapHint = 1 << 16

// EnumCapHint is the exact address count of p as a capacity hint, bounded by
// maxEnumCapHint and 0 when the count does not fit an int. It lets a caller
// pre-size a buffer sized to a scope. It is a hint only, so an inexact value
// costs at most a few reallocations and never a wrong result. EnumerateAddresses
// itself no longer needs it — it streams — but the Settings cap control (#206)
// still prices a scope by its address count.
func EnumCapHint(p netip.Prefix) int {
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
