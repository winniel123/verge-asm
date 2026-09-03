package custody

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// SharedEdgeThreshold is the fan-out count at which an edge reads as a SHARED
// foreign edge. `shared-edge` is true when the count of distinct registrable
// domains is AT LEAST this value (ADR-0129 §1, as set by the #955 amendment).
//
// The value is an absolute integer, never a fraction: fan-out has no natural
// denominator, unlike a certificate-expiry horizon. 100 sits between the two
// measured bands. A real shared edge presents dozens to thousands of unrelated
// registrable domains. A single estate rarely fronts hundreds on one address. The
// value favours the SAFE direction: a false veto of the operator's own edge stays
// rare, at the cost of probing small shared edges that present fewer than a
// hundred identities — the loud, wasteful direction ADR-0129 §2 accepts.
//
// It is a `const`, so no operator setting, no environment variable and no
// database row can write it. It is a declared parameter of the `Custody`
// derivation (ADR-0008): project-authored, fixed at the release, and moved only
// by a release that bumps the version and re-locks the golden corpus.
const SharedEdgeThreshold = 100

// _ proves at COMPILE TIME that SharedEdgeThreshold is a constant. An array
// length admits no variable, so a later change turning the threshold into a
// settable `var` — the shape an operator dial would need — fails the build here
// rather than silently opening the parameter (ADR-0129 §3, #55).
var _ [SharedEdgeThreshold]struct{}

// Params is the `Custody` derivation's declared-parameter set, in the shape the
// measure leaves already use for theirs. It carries the fan-out threshold and
// the identity of the Public Suffix List the reduction runs against, because the
// #954 amendment makes a PSL update a `Break` in the derivation exactly as a
// threshold move is.
//
// It is a RECORD OF the parameters, never the source the derivation reads. The
// derivation reads the constant and the linked PSL directly, exactly as
// `wildcard-discrimination` reads its `RandomLabelCount` rather than its
// `Params` field. So a hand-built `Params` moves no gate, and a zero value
// cannot quietly turn every edge into a shared one.
//
// The reduction itself — strip one wildcard label, drop an address, take the
// eTLD+1, deduplicate — is fixed in the code and moves with the derivation
// version, as `wildcard-discrimination`'s match predicate does.
type Params struct {
	SharedEdgeThreshold int `json:"shared_edge_threshold"`
	// PublicSuffixList identifies the PSL snapshot the reduction reads. It is
	// the list's own version string, so a dependency bump that ships a newer
	// list moves the digest.
	PublicSuffixList string `json:"public_suffix_list"`
}

func DefaultParams() Params {
	return Params{
		SharedEdgeThreshold: SharedEdgeThreshold,
		PublicSuffixList:    publicsuffix.List.String(),
	}
}

// Digest is a stable content hash of the declared parameters. The `Custody`
// golden corpus locks it, so a threshold move with no version bump fails the A6
// gate (ADR-0008, ADR-0021).
func (p Params) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("custody: marshal params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// SharedEdge reports whether the SAN set an edge presented reads as a shared
// foreign edge. It is the boolean the custody-extension veto reads (#985). The
// determination has two outcomes and no third: the *pending* state the #954
// amendment added is a CURRENCY state — no row measured yet — never a count
// band, so it is the caller's absence rule and not a value here.
//
// It takes no parameter argument ON PURPOSE. Reading the threshold off a value
// a caller supplies would let a zero value — an empty struct, or a JSON decode
// that omitted the field — compare `count >= 0` and veto EVERY edge, including
// one that presented no identity at all. That is the unsafe direction the value
// is chosen to avoid. The threshold reaches this comparison from the constant
// alone.
func SharedEdge(sans []string) bool { return FanOut(sans) >= SharedEdgeThreshold }

// FanOut is the count of distinct registrable domains an edge's SAN set reduces
// to. It is the raw measurement the §7 census may render for a human; the value
// that gates the veto is the boolean above.
func FanOut(sans []string) int { return len(registrableSet(sans)) }

// RegistrableDomains reduces a SAN set to its distinct registrable domains,
// sorted. This is the whole of the fan-out reduction (ADR-0129 §1):
//
//  1. Reduce each hostname to its registrable domain (eTLD+1) through the Public
//     Suffix List. The PSL is a registry-suffix dataset the system consumes to
//     compute a boundary, never a list of providers.
//  2. Deduplicate.
//  3. The count of what remains is the fan-out.
//
// "Unrelated" means "distinct registrable domain" and NOTHING MORE. The
// reduction applies no relatedness filter, no brand clustering and no ownership
// signal. Clustering domains that look like one brand is an ownership heuristic
// in disguise, and ADR-0129 §1 refuses ownership as the discriminator. A
// certificate brand string, a PTR pattern and an RDAP or ASN owner each carry
// ZERO weight here; they may label a finding for a human and no more.
func RegistrableDomains(sans []string) []string {
	set := registrableSet(sans)
	out := make([]string, 0, len(set))
	for d := range set {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}

// registrableSet is the reduction itself. FanOut reads it directly rather than
// through RegistrableDomains, so the gate path never builds and sorts a slice it
// only takes the length of — a SAN set on a real shared edge runs to thousands
// of entries, and the census count and the veto both reduce the same set.
func registrableSet(sans []string) map[string]struct{} {
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

// registrableDomain reduces one SAN to its registrable domain. It reports false
// for a SAN the reduction DROPS, and a drop is silent by design: a SAN set is
// third-party wire content, so an unreducible entry is not an error — it is
// simply not a registrable domain and cannot raise the count.
//
// Three kinds of SAN drop, tested in this order:
//
//   - Anything outside the LDH allowlist. A URI, an rfc822 mailbox and an IPv6
//     spelling all carry a character no domain name holds. The allowlist runs
//     before the PSL for the reason `seed.NormalizeDomain` states: the PSL's
//     wildcard rule passes unknown junk straight through.
//   - Anything numeric at the top. `iPAddress` is a SAN type of its own and
//     denotes no registrable domain, and its LDH spelling would otherwise reach
//     the PSL, whose wildcard rule reduces `192.0.2.1` to the nonsense eTLD+1
//     `2.1`. A parse alone does not close that: `999.999.999.999` parses as no
//     address and reduces to `999.999` by the same rule. So the test is the
//     top label, not the parse — no delegated TLD is all digits, and a SAN set
//     is third-party wire content that Go's x509 parser does not police, so a
//     bundle of dotted-numeric names would otherwise inflate the count that
//     gates the veto.
//   - A name with no registrable domain — a bare public suffix such as `co.uk`,
//     a single label such as `localhost`, or the empty string.
func registrableDomain(san string) (string, bool) {
	name := asciiLower(strings.TrimSpace(san))
	// A wildcard SAN reduces to the registrable domain of the name BENEATH the
	// wildcard: `*.example.com` counts as `example.com`, and `*.a.example.co.uk`
	// as `example.co.uk`. Only a leading `*.` is a wildcard SAN; a `*` anywhere
	// else fails the LDH allowlist below and drops.
	name = strings.TrimPrefix(name, "*.")
	name = strings.TrimSuffix(name, ".")
	if name == "" {
		return "", false
	}
	if !isLDHDomain(name) {
		return "", false
	}
	if numericTopLabel(name) {
		return "", false
	}
	reg, err := publicsuffix.EffectiveTLDPlusOne(name)
	if err != nil {
		return "", false
	}
	return reg, true
}

// isLDHDomain reports whether name holds only the characters a DNS domain name
// holds: ASCII letters, digits, hyphen, and the dot label separator (the LDH
// rule, RFC 1035). name is expected already lower-cased. It is an allowlist,
// never a blocklist. IDN is carried as punycode (`xn--…`), which is itself LDH,
// numericTopLabel reports whether name's last label is all digits. Every
// delegated top-level domain holds at least one letter, and a punycode TLD is
// itself `xn--`-prefixed, so this drops an address spelling and a dotted-numeric
// invention while it passes every real name. name is expected already LDH, so
// the labels hold only digits, letters and hyphens.
func numericTopLabel(name string) bool {
	top := name
	if i := strings.LastIndex(name, "."); i >= 0 {
		top = name[i+1:]
	}
	if top == "" {
		return false
	}
	for _, r := range top {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// isLDHDomain reports whether name holds only the characters a DNS domain name
// holds: ASCII letters, digits, hyphen, and the dot label separator (the LDH
// rule, RFC 1035). name is expected already lower-cased. It is an allowlist,
// never a blocklist. IDN is carried as punycode (`xn--…`), which is itself LDH,
// so every legitimate name passes.
func isLDHDomain(name string) bool {
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
