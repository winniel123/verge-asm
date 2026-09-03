package seed

import (
	"fmt"
	"net/netip"
	"strings"
)

func NormalizeExclusionName(input string) (string, error) {
	// Held in a Name key's form: label sequence, case-folded, trailing dot dropped (ADR-0055).
	d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(input)), ".")
	if d == "" {
		return "", fmt.Errorf("a name is required")
	}
	// *Not mine* is a different claim from *not there*, so shape alone is checked (§3.2, §6.4).
	if strings.ContainsAny(d, " /:*?_@") {
		return "", fmt.Errorf("%q is not a bare name — enter a name like api.example.com", input)
	}
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return "", fmt.Errorf("%q is not a fully-qualified name — enter something like api.example.com", input)
	}
	for _, l := range labels {
		if !validLabel(l) {
			return "", fmt.Errorf("%q is not a valid domain name", input)
		}
	}
	return d, nil
}

func validLabel(l string) bool {
	if len(l) == 0 || len(l) > 63 { // 63 is the RFC 1035 §2.3.1 label ceiling
		return false
	}
	if l[0] == '-' || l[len(l)-1] == '-' {
		return false
	}
	for i := 0; i < len(l); i++ {
		c := l[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

func NormalizeExclusionCIDR(input string) (netip.Prefix, error) {
	s := strings.TrimSpace(input)
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Masked(), nil
	}
	// A bare address is a single-host scope: carving one address out is the common case.
	if a, err := netip.ParseAddr(s); err == nil {
		return netip.PrefixFrom(a, a.BitLen()), nil
	}
	return netip.Prefix{}, fmt.Errorf("%q is not an address or CIDR block, e.g. 203.0.113.5 or 203.0.113.0/24", input)
}
