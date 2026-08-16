// Package custody derives `Custody` (`operator` / `third-party`) for an
// `Address` from `Seed`s ALONE — never from registry expansion — and holds the
// probing gate that reads it. `Custody` means control of the listener, not
// registry title (ADR-0013), and it gates every active probe totally: a
// `third-party` address is connected to on no port, by no tier, at any rate,
// with no narrower-probe carve-out (ADR-0019).
//
// The package is database-free and pure. Its inputs are the two Declared `Seed`
// kinds — address scopes (direct custody, enumerating) and custody-extended
// name scopes — and one Observed fact, the addresses those scopes' names resolve
// to (ADR-0013 §5). Nothing probes in v1 until ticket 14; this package produces
// the derivation and the gate, and its tests prove the gate holds by attempting
// to probe fixture data and observing the total block.
//
// A `Proposal` — a registry-proposed address scope the operator has not
// confirmed — is not a `Seed` and is read by nothing (ADR-0002 as amended by
// #27; ADR-0012). That property is structural here: the only inputs an Estate
// accepts are confirmed `Seed`s, so no registry-expansion input can reach the
// derivation.
package custody

import (
	"net/netip"
	"strings"
)

// Custody is the derived value that gates probing. It takes exactly two values:
// `unknown` was deleted once the derivation read `Seed`s alone, because *is this
// address covered by a `Seed`?* is a total question with no lookup left to fail
// (ADR-0013 §2).
type Custody string

const (
	// Operator: the operator controls the listener, so the gate may open.
	Operator Custody = "operator"
	// ThirdParty: everything a `Seed` does not cover — the closed direction.
	ThirdParty Custody = "third-party"
)

// Resolution is one observed direct A/AAAA record: an owner Name holding an
// Address. Custody-extension transitivity is decided by *which name holds the
// address record*: a direct A record inside a declared, extended zone extends
// custody; a CNAME to a foreign name does not, because after the CNAME the A
// record's owner is that foreign name and it is within no extended scope
// (ADR-0013 §3). Modelling resolution at the owner-of-the-A-record granularity
// is exactly how DNS records the fact and is what a dns-record A/AAAA
// observation carries.
type Resolution struct {
	Owner   string     // canonical Name key holding the record
	Address netip.Addr // the address the record points at
}

// Estate is the confirmed-Seed input the derivation reads. Every field is a
// `Seed` fact or an observation of a `Seed`'s names; there is no field a
// registry proposal could populate.
type Estate struct {
	// AddressScopes are the CIDRs of declared address-scope Seeds, canonical
	// (masked). Every address inside one derives operator directly.
	AddressScopes []netip.Prefix
	// ExtendedZones are the registrable domains of name-scope Seeds carrying a
	// custody extension. A name-scope Seed with the extension off contributes
	// nothing to the derivation.
	ExtendedZones []string
	// Resolutions are the observed direct A/AAAA records of names in the estate.
	Resolutions []Resolution
}

// Derive returns the Custody of addr, reading the Estate's Seeds alone. The two
// operator limbs are disjunctive: an address scope covers it directly, or a
// custody extension covers it by resolution. Everything else is third-party.
func (e Estate) Derive(addr netip.Addr) Custody {
	addr = addr.Unmap()
	if e.coveredByAddressScope(addr) {
		return Operator
	}
	if e.coveredByExtension(addr) {
		return Operator
	}
	return ThirdParty
}

// coveredByAddressScope reports whether a declared address scope contains addr.
// Containment is family-matched prefix comparison over the address, never over a
// spelling (CONTEXT.md `Seed`), so the lookup cannot turn on a rendering.
func (e Estate) coveredByAddressScope(addr netip.Addr) bool {
	for _, p := range e.AddressScopes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// coveredByExtension reports whether a custody extension covers addr. Two
// stopping conditions, both measurements rather than lists (ADR-0013 §3 as
// amended by ADR-0079):
//
//   - The extension does not cover a non-globally-reachable address, whether or
//     not the chain stayed inside the zone — such an address denotes one machine
//     per realm and an extension declares no realm (ADR-0079). `api.example.com
//     A 10.0.0.5` is a direct A record inside the declared zone and extends
//     nothing.
//   - Transitivity stops where the chain leaves the declared zone: the A
//     record's owner name must itself be within a custody-extended name scope.
//     A CNAME to a foreign name puts the A record on that foreign name, which is
//     within no extended scope, so it does not extend.
func (e Estate) coveredByExtension(addr netip.Addr) bool {
	if IsNonGloballyReachable(addr) {
		return false
	}
	for _, r := range e.Resolutions {
		if r.Address.Unmap() != addr {
			continue
		}
		if e.withinExtendedZone(r.Owner) {
			return true
		}
	}
	return false
}

// withinExtendedZone reports whether name is within some custody-extended name
// scope, by label-wise suffix comparison over the Name key — the candidate's
// labels end with the zone's labels, compared label by label (CONTEXT.md
// `Custody extension`; ADR-0013 §3). This is the name-side twin of the
// address-containment test and, like it, never compares a name as a string, so
// evilexample.com does not read as inside example.com.
func (e Estate) withinExtendedZone(name string) bool {
	return WithinAnyZone(name, e.ExtendedZones)
}

// WithinAnyZone reports whether name falls within any of zones — the subtree
// containment CONTEXT.md defines, `name` being a zone's apex or a subdomain of
// it. It is the ONE label-wise suffix test the model owns: every "a name is
// under a declared domain" question (custody extension, cold-scan opt-in,
// signal InDeclaredZone, wildcard-discrimination population) routes here rather
// than re-deriving `name == d || strings.HasSuffix(name, "."+d)` on raw strings,
// which folds Unicode and compares a name as a string — both of which CONTEXT.md
// and ADR-0055 forbid.
func WithinAnyZone(name string, zones []string) bool {
	for _, zone := range zones {
		if LabelSuffix(name, zone) {
			return true
		}
	}
	return false
}

// LabelSuffix reports whether candidate's labels end with zone's labels. A
// name's own labels end with its own labels, so a zone covers its own apex as
// well as everything beneath it. Comparison is over lower-cased ASCII labels,
// matching the Name key's fold (ADR-0055) — never a raw string HasSuffix, so
// evilexample.com does not read as inside example.com.
func LabelSuffix(candidate, zone string) bool {
	cl := labels(candidate)
	zl := labels(zone)
	if len(zl) == 0 || len(cl) < len(zl) {
		return false
	}
	off := len(cl) - len(zl)
	for i := range zl {
		if cl[off+i] != zl[i] {
			return false
		}
	}
	return true
}

// labels splits a Name into its lower-cased ASCII labels, dropping a trailing
// dot. It folds only what the protocol folds — the 26 ASCII letters — so the
// test never turns on case or a trailing-dot spelling.
func labels(name string) []string {
	trimmed := strings.TrimSuffix(name, ".")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(asciiLower(trimmed), ".")
	return parts
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
