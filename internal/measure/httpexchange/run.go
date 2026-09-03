package httpexchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

type Scope struct {
	Vantage      string   `json:"vantage"`
	VantageClass string   `json:"vantage_class"`
	Targets      []Target `json:"targets"`
	Params       Params   `json:"params"`
}

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

func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	base := NetExchanger{Params: scope.Params}
	paced := &pacedExchanger{inner: base, pacer: NewPacer(scope.Params), now: time.Now, sleep: time.Sleep}
	return RunWithExchanger(context.Background(), paced, spec.Batch, scope, w)
}

func RunWithExchanger(ctx context.Context, ex Exchanger, batch string, scope Scope, w io.Writer) error {
	// The golden corpus drives this path with a scripted exchanger, so no test needs a network.
	bodyCap := scope.Params.BodyCapBytes
	if bodyCap <= 0 {
		bodyCap = DefaultParams().BodyCapBytes
	}
	var out []wire.Observation
	for _, target := range scope.Targets {
		res := ex.Exchange(ctx, target)
		id := Identity(res, bodyCap)
		// Vantage is recorded, never re-gated: ADR-0019's Custody gate ran at dispatch.
		out = append(out, EmitEndpoint(batch, scope.Vantage, target, id))
	}
	return writeNDJSON(w, out)
}

type pacedExchanger struct {
	inner Exchanger
	pacer *Pacer
	now   func() time.Time
	sleep func(time.Duration)
}

func (p *pacedExchanger) Exchange(ctx context.Context, target Target) ExchangeResult {
	// Pacing moves only when a request starts, never its deadline or its identity (ADR-0021).
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
