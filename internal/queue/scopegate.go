package queue

import (
	"encoding/json"
	"log"
	"net/netip"
	"strings"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
	"github.com/winniel123/verge-asm/internal/wire"
)

// A compromised prober names any Subject, so this re-gates what we record as measured (ADR-0217).

type authorizedScope struct {
	addrs map[string]struct{}
	names map[string]struct{}
}

// The kinds' scope records share no field name, which is what makes one union safe to unmarshal.

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

func parseAuthorizedScope(raw []byte) authorizedScope {
	var s scopeShape
	// Our dispatcher writes this scope, so a parse failure is a bug and fails open (ADR-0217 §1).
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

func normAddr(s string) string {
	// One address has many spellings, so both sides normalise or the check over-rejects a line.
	if a, err := netip.ParseAddr(strings.TrimSpace(s)); err == nil {
		return a.Unmap().String()
	}
	return strings.TrimSpace(s)
}

func (a authorizedScope) admits(o wire.Observation) bool {
	// An undenoted dimension gates nothing; gating it would drop every real line (ADR-0217 §1).
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
	case "":
		// An injected edge-fanout line feeds the custody veto an answer nothing measured (#985).
		if o.Kind != edgefanout.Kind {
			return true
		}
		// This arm alone fails closed: edge-fanout scope always denotes addresses (ADR-0217 §3).
		if a.addrs == nil {
			return false
		}
		_, ok := a.addrs[normAddr(o.Address)]
		return ok
	default:
		return true
	}
}

func subjectAddrKey(subject string) string {
	// An Endpoint key prefixes a Name with @, and neither a Name nor a Service key holds one.
	key := subject
	if i := strings.Index(key, "@"); i >= 0 {
		key = key[i+1:]
	}
	return normAddr(serviceAddress(key))
}

func (a authorizedScope) gate(obs []wire.Observation, logger *log.Logger, jobID int64) []wire.Observation {
	// Failing the job on a drop would hand a compromised prober a denial of service (ADR-0217 §4).
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
			logger.Printf("worker: job %d dropped out-of-scope observation: kind=%q facet=%q subject=%q address=%q",
				jobID, o.Kind, o.Facet, o.Subject, o.Address)
		}
	}
	return out
}
