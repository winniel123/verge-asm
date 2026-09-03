// Package custody derives `Custody` (`operator` / `third-party`) for an
// `Address` from `Seed`s ALONE — never from registry expansion — and holds the
// probing gate that reads it. `Custody` means control of the listener, not
// registry title (ADR-0013), and it gates every active probe totally: a
// `third-party` address is connected to on no port, by no tier, at any rate,
// with no narrower-probe carve-out (ADR-0019).
//
// The package is database-free and pure. Its inputs are the two Declared `Seed`
// kinds — address scopes (direct custody, enumerating) and custody-extended
// name scopes — and two Observed facts: the addresses those scopes' names resolve
// to (ADR-0013 §5), and the `edge-fanout` measurement that narrows which of those
// addresses the custody extension reaches (ADR-0129 §4; see EdgeFanout). The
// second acts on the extension limb alone, never on the declaration limb.
// Nothing probes in v1 until ticket 14; this package produces
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
	Operator   Custody = "operator"
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
	AddressScopes []netip.Prefix
	ExtendedZones []string
	Resolutions   []Resolution
	// edgeFanout is the `edge-fanout` Scan's measured result. It narrows the
	// custody extension's reach and reaches NOTHING ELSE (ADR-0129 §4).
	//
	// It is UNEXPORTED and WithEdgeFanout is the only way to set it. The record
	// carries a floor that is read per limb (EdgeFanout.ExtensionErrored), and
	// resolving that floor needs the candidate set — which the assembler holds and
	// the read path does not. An exported field would let an assembler carry a
	// measurement in with the floor unresolved, and the failure that produces is
	// silent: every extension candidate held, for as long as the condition lasts.
	// The zero value is safe (the extension reaches what it reached before
	// ADR-0129), so an Estate that never takes a measurement is unharmed.
	edgeFanout EdgeFanout
	// addressExclusions are the CIDRs of declared `address` exclusions — the
	// operator's *not mine* claim over a range (ADR-0012 §125, ADR-0133 §1). They
	// narrow the ADDRESS-SCOPE limb and NOTHING ELSE. An address an exclusion covers
	// that a custody extension ALSO reaches still derives operator and is still
	// probed: the set an exclusion removes is never larger than the set the
	// declaration added, and *not mine* is a claim about the operator's own
	// declaration rather than about their own name resolving at the address.
	//
	// It is UNEXPORTED and WithAddressExclusions is the only way to set it, for the
	// reason edgeFanout gives above. Three sites build an Estate literal, and a new
	// EXPORTED field is silently zero at each of them with no compiler error. The
	// zero value means NO EXCLUSIONS, which is the safe reading for an assembler
	// that has not opted in: it derives what it derived before ADR-0133.
	addressExclusions []netip.Prefix
}

// WithAddressExclusions returns e carrying the declared `address` exclusions. It is
// the ONE way an exclusion enters an Estate (ADR-0133 §2).
//
// The prefixes need no masking. Containment is netip.Prefix.Contains, which compares
// the prefix-length bits alone, so a prefix carrying host bits covers the same
// addresses a masked one does.
func (e Estate) WithAddressExclusions(prefixes []netip.Prefix) Estate {
	e.addressExclusions = prefixes
	return e
}

// AddressExcluded reports whether a declared `address` exclusion covers addr.
// Containment is the family-matched prefix comparison coveringAddressScope already
// applies to a scope, never a comparison over a spelling (CONTEXT.md `Seed`).
//
// It is EXPORTED because the two fan-out enumerators outside this package need it.
// queue.candidateAddrs walks the COLD tier's opted-in prefixes rather than this
// Estate's scopes, so it cannot reach the exclusion through the coverage predicate
// (ADR-0133 §3).
//
// It answers about the SEED LIMB ALONE, and it is NOT a probing verdict. An address
// it returns true for is still probed where a custody extension reaches it, so no
// gate may read it in place of Derive or MayProbe.
func (e Estate) AddressExcluded(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, p := range e.addressExclusions {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (e Estate) WithEdgeFanout(f EdgeFanout) Estate {
	e.edgeFanout = f.overExtension(e.ExtensionCandidates())
	return e
}

// Derive returns the Custody of addr, reading the Estate's Seeds alone. The two
// operator limbs are disjunctive: an address scope covers it directly, or a
// custody extension covers it by resolution. Everything else is third-party.
//
// The `edge-fanout` measurement acts on the SECOND limb alone. An address a
// literal address-scope `Seed` covers returns from the first limb before the
// extension is asked, so it derives `operator` at any fan-out count (ADR-0129's
// #956 amendment). See EdgeFanout for the law that puts the veto there.
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

// CoversAddressScope reports whether a declared ADDRESS SCOPE contains addr — the
// public `covered` predicate the Vantage-class derivation binds (#711, ADR-0049).
// It is a thin exported wrapper over the private containment so batch gating, Custody,
// and Vantage-class coverage share ONE matcher and cannot diverge. It routes through
// coveredByAddressScope ALONE — deliberately NOT Derive (which also folds in the
// custody extension) and NOT MayProbe (which folds in non-global-reachability + class):
// a vantage's side of the boundary is decided by declared address scopes only, never
// by an extension (CONTEXT.md `Vantage class`), so admitting extension-covered
// addresses here would corrupt the class. addr is Unmap'ed to match the family-agnostic
// containment, mirroring exposure.VerifyClass's own covered(a.Unmap()).
//
// IT NARROWS WITH THE DERIVATION, and that is INTENDED (ADR-0133 §4). Routing through
// coveredByAddressScope means a declared `address` exclusion takes its addresses out of
// this predicate too, so a vantage whose egress sits inside an excluded range stops
// being covered and exposure.VerifyClass may reclassify it. The consequence is
// consistent on its own terms: the operator has said the range is not theirs, so a
// prober inside it is not inside the estate.
//
// DO NOT add a second, un-narrowed predicate to avoid the reclassification. #711's
// invariant is ONE binding used identically by batch gating and every render, and two
// coverage rules would have to be held in step by hand from then on.
func (e Estate) CoversAddressScope(addr netip.Addr) bool {
	return e.coveredByAddressScope(addr.Unmap())
}

// coveredByAddressScope reports whether a declared address scope contains addr.
// Containment is family-matched prefix comparison over the address, never over a
// spelling (CONTEXT.md `Seed`), so the lookup cannot turn on a rendering.
func (e Estate) coveredByAddressScope(addr netip.Addr) bool {
	_, covered := e.coveringAddressScope(addr)
	return covered
}

// coveringAddressScope returns the first declared address scope containing addr,
// and whether one does. It is coveredByAddressScope's answer plus the scope that
// gave it, which the custody-extension census names on its dual-limb row (#987).
// Both callers share this one matcher so the predicate and the row cannot
// disagree about which addresses a `Seed` covers.
//
// FIRST match, not most specific. Two overlapping scopes both cover the address
// and the derivation is the same either way; picking between them would be a
// specificity test, which this package refuses (see EdgeFanout).
//
// A declared `address` EXCLUSION answers first, and answers NOT COVERED (ADR-0133
// §1). This is the whole of the exclusion limb, and putting it HERE is what keeps
// the narrowing to the `Seed` limb: Derive asks coveredByExtension afterwards, so an
// excluded address a custody extension reaches still derives operator and is still
// probed. A refusal placed in MayProbe instead would shut the gate over that address
// too, which is the semantics ADR-0133 rejects.
func (e Estate) coveringAddressScope(addr netip.Addr) (netip.Prefix, bool) {
	if e.AddressExcluded(addr) {
		return netip.Prefix{}, false
	}
	for _, p := range e.AddressScopes {
		if p.Contains(addr) {
			return p, true
		}
	}
	return netip.Prefix{}, false
}

// coveredByExtension reports whether a custody extension covers addr. It is the
// extension's REACH narrowed by the `edge-fanout` measurement: the label-suffix
// test decides which in-zone-cited addresses the extension would pull in, and the
// fan-out result decides which of those it declines (ADR-0129 §4 as amended by
// #944).
//
// The two are a conjunction, never a ranking. A clear measurement admits no
// address the reach did not already hold, and a shared measurement drops one the
// reach did hold. So a measured shared edge and a CNAME-to-foreign edge reach the
// same resting state: outside the estate, never a `Subject`, holding no `Custody`
// value, opening no `Gap` and queueing no probe.
func (e Estate) coveredByExtension(addr netip.Addr) bool {
	return e.extensionReaches(addr) && e.edgeFanout.admits(addr)
}

// extensionReaches reports whether a custody extension REACHES addr, before the
// fan-out measurement narrows it. It is what the `edge-fanout` Scan measures over
// (ExtensionCandidates), so a vetoed edge stays a candidate and is handshaked
// again on the next tick — a veto read back into the population would freeze the
// last measurement and no later one could lift it.
//
// Two stopping conditions, both measurements rather than lists (ADR-0013 §3 as
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
func (e Estate) extensionReaches(addr netip.Addr) bool {
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
