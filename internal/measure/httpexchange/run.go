package httpexchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

// Scope is the http-exchange-specific payload of a JobSpec. It carries the Vantage
// and its class (recorded — the Custody gate has already run at dispatch,
// ADR-0019), the `(Name, Service)` targets to exchange with — each already known
// to have a `reached` TCP connect, since the HTTP step rides the reachability
// exchange — and the declared parameter set, every offer enumerated so the Batch
// records what governed the exchange by content (ADR-0025).
type Scope struct {
	Vantage      string   `json:"vantage"`
	VantageClass string   `json:"vantage_class"`
	Targets      []Target `json:"targets"`
	Params       Params   `json:"params"`
}

// DecodeScope reads a Scope from a JobSpec's opaque Scope payload.
func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("httpexchange: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("httpexchange: decode scope: %w", err)
	}
	return s, nil
}

// Run executes the leaf against live HTTP for one JobSpec, writing NDJSON
// http-identity observations to w. It is the production entrypoint; the golden
// corpus calls RunWithExchanger against a scripted Exchanger instead. The
// production Exchanger is paced to the profile's per-host request ceiling (§3.3),
// pacing that never changes an identity, only its timing.
func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	base := NetExchanger{Params: scope.Params}
	paced := &pacedExchanger{inner: base, pacer: NewPacer(scope.Params), now: time.Now, sleep: time.Sleep}
	return RunWithExchanger(context.Background(), paced, spec.Batch, scope, w)
}

// RunWithExchanger executes the leaf against an arbitrary Exchanger. Separating
// the exchanger from Run is what lets one code path be driven by the paced network
// adapter in production and by a scripted exchanger in a hermetic test. It produces
// one `http-identity` observation per target whose exchange COMPLETED — creating
// the Endpoint subject for that `(Name, Service)` pair — and no observation for a
// target whose exchange failed, which records no identity and creates no Endpoint.
func RunWithExchanger(ctx context.Context, ex Exchanger, batch string, scope Scope, w io.Writer) error {
	bodyCap := scope.Params.BodyCapBytes
	if bodyCap <= 0 {
		bodyCap = DefaultParams().BodyCapBytes
	}
	var out []wire.Observation
	for _, target := range scope.Targets {
		res := ex.Exchange(ctx, target)
		id, ok := Identity(res, bodyCap)
		if !ok {
			// The exchange did not complete: no Endpoint, no observation.
			continue
		}
		out = append(out, EmitEndpoint(batch, scope.Vantage, target, id))
	}
	return writeNDJSON(w, out)
}

// pacedExchanger wraps an Exchanger with the per-host request limiter: before each
// exchange it sleeps until the pacer says the host's next slot is due, honouring
// the ≤ 10 req/s per-host ceiling (§3.3). The timeout is untouched — the wrapped
// exchange keeps its own deadline — so pacing changes only when a request starts,
// never how long it may run (ADR-0021).
type pacedExchanger struct {
	inner Exchanger
	pacer *Pacer
	now   func() time.Time
	sleep func(time.Duration)
}

func (p *pacedExchanger) Exchange(ctx context.Context, target Target) ExchangeResult {
	due := p.pacer.Next(target.Address, p.now())
	if wait := due.Sub(p.now()); wait > 0 {
		p.sleep(wait)
	}
	return p.inner.Exchange(ctx, target)
}

func writeNDJSON(w io.Writer, obs []wire.Observation) error {
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}
