package custody

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// An absolute count, never a fraction: fan-out has no natural denominator (ADR-0129 §1).

const SharedEdgeThreshold = 100 // a false veto stays rarer than probing a small shared edge

// An array length admits no variable, so a settable threshold fails the build here (ADR-0129 §3).

var _ [SharedEdgeThreshold]struct{}

// A PSL update is a Break in the derivation exactly as a threshold move is (ADR-0129, #954).

type Params struct {
	SharedEdgeThreshold int    `json:"shared_edge_threshold"`
	PublicSuffixList    string `json:"public_suffix_list"`
}

func DefaultParams() Params {
	// A record of the parameters, never the source: a hand-built value moves no gate.
	return Params{
		SharedEdgeThreshold: SharedEdgeThreshold,
		PublicSuffixList:    publicsuffix.List.String(),
	}
}

func (p Params) Digest() string {
	// The golden corpus locks it, so a threshold move with no version bump fails A6 (ADR-0008).
	b, err := json.Marshal(p)
	if err != nil {
		panic("custody: marshal params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// A caller-supplied threshold could be zero, and count >= 0 vetoes every edge (#985).

func SharedEdge(sans []string) bool { return FanOut(sans) >= SharedEdgeThreshold }

func FanOut(sans []string) int { return len(registrableSet(sans)) }

func RegistrableDomains(sans []string) []string {
	// Ownership is refused as the discriminator, so brand clustering carries no weight (ADR-0129 §1).
	set := registrableSet(sans)
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

func registrableSet(sans []string) map[string]struct{} {
	// A real shared edge presents thousands of SANs, so the gate path never sorts what it only counts.
	seen := make(map[string]struct{}, len(sans))
	for _, san := range sans {
		reg, ok := registrableDomain(san)
		if !ok {
			continue
		}
		seen[reg] = struct{}{}
	}
	return seen
}

func registrableDomain(san string) (string, bool) {
	// A SAN set is third-party wire content, so an unreducible entry is a silent drop, never an error.
	name := asciiLower(strings.TrimSpace(san))
	name = strings.TrimPrefix(name, "*.")
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return "", false
	}
	// The allowlist runs before the PSL, whose wildcard rule passes unknown junk straight through.
	if !isLDHDomain(name) {
		return "", false
	}
	// The PSL reduces 192.0.2.1 to "2.1", so dotted-numeric SANs would inflate the gating count.
	if numericTopLabel(name) {
		return "", false
	}
	reg, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err != nil {
		return "", false
	}
	return reg, true
}

func numericTopLabel(name string) bool {
	// A parse alone would miss 999.999.999.999, which the PSL still reduces to "999.999".
	top := name
	if i := strings.LastIndex(name, "."); i >= 0 {
		top = name[i+1:]
	}
	if top == "" {
		return false
	}
	// Every delegated TLD holds a letter and punycode is "xn--"-prefixed, so no real name drops.
	for _, r := range top {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isLDHDomain(name string) bool {
	// An allowlist, never a blocklist: IDN rides as punycode, which is itself LDH (RFC 1035).
	for _, r := range name {
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
