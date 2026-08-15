package seed

import (
	"fmt"
	"net/netip"
	"strings"
)

// An exclusion draws the estate boundary inwards: an exact name, a name
// subtree, or an address scope the operator declares is not theirs (v1 spec
// §3.2, §6.4). Excluding a name that still resolves is legal — *not mine* is a
// different claim from *not there* — so these normalisers validate shape alone
// and never that the value is reachable or already declared.
//
// A name exclusion is held in the same form a Name's key is (ADR-0055): the
// label sequence, ASCII-case-folded, trailing dot dropped. The exact/subtree
// distinction is carried by the exclusion's kind, never by the stored string —
// subtree containment is the label-wise suffix comparison a later ticket runs
// over that key, not a property of the text.

// NormalizeExclusionName validates input as a fully-qualified name and returns
// it lowercased and trailing-dot-stripped. It accepts any depth of name (an
// exclusion may name `api.example.com` or the shallow `example.com`), and it
// refuses a wildcard, a single bare label, and anything that is not a bare
// name — the same refusals NormalizeDomain makes, minus the registrable-domain
// requirement, since an exclusion is not a scope.
func NormalizeExclusionName(input string) (string, error) {
	d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(input)), ".")
	if d == "" {
		return "", fmt.Errorf("a name is required")
	}
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

// validLabel reports whether l is a single valid DNS label: 1–63 characters of
// letters, digits and hyphens, not starting or ending with a hyphen. The
// caller has already rejected every character outside [a-z0-9.-], so this is
// only the length and hyphen-position check.
func validLabel(l string) bool {
	if len(l) == 0 || len(l) > 63 {
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

// NormalizeExclusionCIDR parses input as an address scope to exclude and returns
// it in canonical masked form. A bare address is accepted as a single-host scope
// (a `/32` or `/128`), since carving one address out of a declared CIDR is the
// common case; a CIDR has its host bits cleared, so `10.0.0.5/24` becomes
// `10.0.0.0/24`.
func NormalizeExclusionCIDR(input string) (netip.Prefix, error) {
	s := strings.TrimSpace(input)
	if p, err := netip.ParsePrefix(s); err == nil {
		return p.Masked(), nil
	}
	if a, err := netip.ParseAddr(s); err == nil {
		return netip.PrefixFrom(a, a.BitLen()), nil
	}
	return netip.Prefix{}, fmt.Errorf("%q is not an address or CIDR block, e.g. 203.0.113.5 or 203.0.113.0/24", input)
}
