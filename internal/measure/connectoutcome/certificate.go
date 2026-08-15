package connectoutcome

import (
	"context"
	"io"
	"net/netip"

	"github.com/winniel123/verge-asm/internal/wire"
)

// FacetCertificate is the facet the tls-handshake step decides.
const FacetCertificate = "certificate"

// EndpointKey renders the `Endpoint` subject key for one `(Name, Service)` pair:
// `name@address:port/transport`, e.g. `api.example.com@198.51.100.1:443/tcp`. The
// Endpoint is the only key under which a presented certificate chain is
// single-valued — two names on one address and port serve, under SNI, different
// certificates, so keying `certificate` on the `Service` would manufacture drift
// on every virtual host every run (CONTEXT.md `Endpoint`).
//
// The `Name` may be ABSENT — *the default response to a client that names
// nothing* — a distinguished variant of the key and never an empty name. It
// renders with an empty name segment, `@address:port/transport`, and because a
// real name is always non-empty it can never collide with a named endpoint. Both
// legs are subjects, so this key inherits the Service (and its Address) form
// rather than restating it.
func EndpointKey(serverName string, target netip.AddrPort, transport string) string {
	return serverName + "@" + ServiceKey(target, transport)
}

// certificateValue is the JSON payload of a certificate observation: the outcome
// tag, plus — only on a presented chain — the ordered fingerprints, leaf first.
// The value space is a closed union: a presentation carries its chain, and the two
// negatives (`tls-refused`, `no-tls`) carry none, because each is a value in its
// own right and not an absence (CONTEXT.md `Certificate`).
type certificateValue struct {
	Outcome TLSOutcome `json:"outcome"`
	Chain   []string   `json:"chain,omitempty"`
}

// EmitCertificate renders one Endpoint's presented-certificate value at one
// Vantage into an NDJSON observation. The subject is the `(Name, Service)` pair;
// the facet is `certificate`; the value is the closed union
// `presented(chain) │ tls-refused │ no-tls`. It is emitted ONLY for a Service the
// connect reached — neither negative can be read without knowing the port was
// open, and the value space has no variant meaning *the port was shut*
// (CONTEXT.md `Certificate`).
func EmitCertificate(batch, vantage string, target netip.AddrPort, serverName string, res HandshakeResult) wire.Observation {
	return wire.Observation{
		Batch:   batch,
		Kind:    Kind, // shares the reachability dispatch — the handshake is a step in it
		Facet:   FacetCertificate,
		Subject: EndpointKey(serverName, target, "tcp"),
		Vantage: vantage,
		Address: target.Addr().String(),
		Data:    mustJSON(certificateValue{Outcome: res.Outcome, Chain: res.Chain}),
	}
}

// endpointNames folds a scope's declared server names to the set of Endpoints to
// hand the handshake for a reached Service. An empty set is the single NAMELESS
// endpoint — the only mode available on an address-scope Seed where no name is
// known yet — represented by an empty server name (no SNI). A named scope hands
// one Endpoint per name, each sending its own SNI.
func (s Scope) endpointNames() []string {
	if len(s.Names) == 0 {
		return []string{""} // the nameless endpoint
	}
	return s.Names
}

// RunExchange runs the reachability exchange with the certificate handshake step
// composed into it, writing both facets' NDJSON to w. For each TCP Service in
// scope it emits a `reachability` observation exactly as RunWithConnector does;
// and for each Service the connect REACHED it performs the TLS handshake — one
// per Endpoint (nameless, or one per declared name) — and emits a `certificate`
// observation. The handshake is a STEP inside this one exchange, not a second
// dispatch (AC #197): `certificate` has no Scan and no cadence of its own, it
// rides whichever port tier made the connect.
//
// A later ticket adds an HTTP step to the same reached-Service branch (#198); this
// function is the composition point the orchestrator reconciles.
func RunExchange(ctx context.Context, c Connector, h Handshaker, batch string, scope Scope, w io.Writer) error {
	var out []wire.Observation
	for _, target := range scope.targets() {
		outcome, raw := Probe(ctx, c, scope.Profile, target)
		out = append(out, EmitService(batch, scope.Vantage, target, outcome, raw))
		if outcome != Reached {
			continue
		}
		for _, name := range scope.endpointNames() {
			res := h.Handshake(ctx, target, name)
			out = append(out, EmitCertificate(batch, scope.Vantage, target, name, res))
		}
	}
	return writeNDJSON(w, out)
}
