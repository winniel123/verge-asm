package queue

import (
	"encoding/json"
	"log"
	"net/netip"
	"strings"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/wire"
)

// authorizedScope is the subject denotation a completed job's Batch authorised —
// parsed from the job's AttemptedScope (worker.go:357), the by-content record of
// what the job set out to cover. It is the recording-side re-gate for #773: a
// compromised prober can put ANY string in an observation's Subject (the field the
// batch writes as SubjectKey and the fold keys spans/estate/messages on), so before
// the write we drop observations whose Subject is NOT one the job authorised. This
// is the integrity twin of the egress re-gate custody.MayProbe already applies to
// new probe targets (per the audit): that bounds where we connect; this bounds what
// we are willing to record as measured.
//
// A dimension is gated ONLY where the scope denotes a non-empty authorised set for
// it, so the check can never over-reject a legitimate observation by testing it
// against an empty or absent denotation:
//
//   - addrs — the authorised Address set, from the address-scoped kinds: hot/cold
//     `addresses`, tls-acceptance `services[].address`, http-identity
//     `targets[].address`. Every address-bearing facet (reachability, certificate,
//     http-identity, tls-acceptance) keys its Service/Endpoint subject on one of
//     these addresses (connectoutcome.ServiceKey / EndpointKey, tlsacceptance /
//     httpexchange emit), rendered from the Address's netip form — so the authorised
//     denotation is exact under netip normalisation.
//   - names — the authorised Name set, from the dns kind's `names`. A resolution /
//     dns-record observation's Subject is exactly CanonicalName(one of the resolved
//     Names) (resolutionwalk.Resolve / emit): the walk emits its CNAME targets as RR
//     DATA, never as separate subjects, so the authorised denotation is exact under
//     normalizeDomain (which matches CanonicalName's trim-dot + lowercase).
//
// nil map => that dimension is not denoted by this scope (or is empty) and is left
// ungated — a job of that kind emits no observation on that dimension, and gating an
// empty set would drop every line. Kinds with no prober (zone, ct) never reach here.
type authorizedScope struct {
	addrs map[string]struct{}
	names map[string]struct{}
}

// scopeShape is the union of every address/name source an AttemptedScope record
// carries across the prober-measured kinds. Each kind fills its own subset; the
// others stay zero. Unmarshalling the union is safe because the field names do not
// collide across kinds (hot/cold `addresses`, tls `services`, http-identity
// `targets`, dns `names`).
type scopeShape struct {
	Names     []string `json:"names"`
	Addresses []string `json:"addresses"`
	Services  []struct {
		Address string `json:"address"`
	} `json:"services"`
	Targets []struct {
		Address string `json:"address"`
	} `json:"targets"`
}

// parseAuthorizedScope builds the authorised denotation from a job's AttemptedScope
// JSON. The scope is produced by our own trusted dispatcher (scan.*.AttemptedScope
// marshals a scope record), so a parse failure is an internal bug, not attacker
// input: it yields an all-nil scope that gates nothing, reverting to pre-#773
// behaviour rather than dropping legitimate observations.
func parseAuthorizedScope(raw []byte) authorizedScope {
	var s scopeShape
	if len(raw) == 0 || json.Unmarshal(raw, &s) != nil {
		return authorizedScope{}
	}
	var a authorizedScope
	addrs := map[string]struct{}{}
	for _, ip := range s.Addresses {
		addrs[normAddr(ip)] = struct{}{}
	}
	for _, svc := range s.Services {
		addrs[normAddr(svc.Address)] = struct{}{}
	}
	for _, t := range s.Targets {
		addrs[normAddr(t.Address)] = struct{}{}
	}
	if len(addrs) > 0 {
		a.addrs = addrs
	}
	names := map[string]struct{}{}
	for _, n := range s.Names {
		names[normalizeDomain(n)] = struct{}{}
	}
	if len(names) > 0 {
		a.names = names
	}
	return a
}

// normAddr canonicalises an address for set membership: a value that parses as an IP
// compares by its netip form (so a scope address and a subject address for the same
// host never differ by spelling — IPv4-mapped, zero-compression, case), and one that
// does not falls back to its trimmed string, symmetrically on both sides, so a
// non-IP scope entry can still match its own subject rather than being over-rejected.
func normAddr(s string) string {
	if a, err := netip.ParseAddr(strings.TrimSpace(s)); err == nil {
		return a.Unmap().String()
	}
	return strings.TrimSpace(s)
}

// admits reports whether observation o names a subject this scope authorised. It
// gates only the dimension the observation's facet lives on, and only when that
// dimension is denoted (non-nil map); every other facet, and every undenoted
// dimension, is admitted unchanged.
func (a authorizedScope) admits(o wire.Observation) bool {
	switch o.Facet {
	case resolutionwalk.FacetResolution, resolutionwalk.FacetDNSRecord:
		if a.names == nil {
			return true
		}
		_, ok := a.names[normalizeDomain(o.Subject)]
		return ok
	case connectoutcome.FacetReachability, connectoutcome.FacetCertificate,
		httpexchange.FacetHTTPIdentity, tlsacceptance.Facet:
		if a.addrs == nil {
			return true
		}
		_, ok := a.addrs[subjectAddrKey(o.Subject)]
		return ok
	default:
		// A facet with no address/name subject relation (or one a later wave adds):
		// nothing to gate it against here, so it is not this check's to reject.
		return true
	}
}

// subjectAddrKey extracts and normalises the Address limb of a Service or Endpoint
// subject key for set membership. A Service subject is `address:port/transport`; an
// Endpoint subject is `name@address:port/transport` (httpexchange.EndpointKey), so
// an `@` is split off first — neither a DNS Name nor a Service key contains `@`.
func subjectAddrKey(subject string) string {
	key := subject
	if i := strings.Index(key, "@"); i >= 0 {
		key = key[i+1:]
	}
	return normAddr(serviceAddress(key))
}

// gate drops the observations whose Subject the job did not authorise, returning the
// admitted ones in order. A drop is logged loudly (ADR-0001's fail-loud on a wire
// mismatch) and never fails the job: legitimate lines in the same batch commit, and a
// compromised prober cannot turn injected lines into a queue denial-of-service. When
// the scope denotes nothing gateable (all-nil) the input is returned as-is.
func (a authorizedScope) gate(obs []wire.Observation, logger *log.Logger, jobID int64) []wire.Observation {
	if a.addrs == nil && a.names == nil {
		return obs
	}
	out := make([]wire.Observation, 0, len(obs))
	for _, o := range obs {
		if a.admits(o) {
			out = append(out, o)
			continue
		}
		if logger != nil {
			logger.Printf("worker: job %d dropped out-of-scope observation: facet=%q subject=%q (#773)", jobID, o.Facet, o.Subject)
		}
	}
	return out
}
