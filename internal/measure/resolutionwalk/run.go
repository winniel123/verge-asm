package resolutionwalk

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/winniel123/verge-asm/internal/wire"
)

const Kind = "resolution-walk"

// Scope is the resolution-walk-specific payload of a JobSpec. It carries the
// Vantage and its recursive resolver (part of the Vantage identity, ADR-0070),
// the Names to resolve, and the Offers put on the wire — all enumerated in the
// job spec so the Batch records what went on the wire by content (ADR-0025).
type Scope struct {
	Vantage  string   `json:"vantage"`
	Resolver string   `json:"resolver"`
	Names    []string `json:"names"`
	Offers   Offers   `json:"offers"`
}

func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("resolutionwalk: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("resolutionwalk: decode scope: %w", err)
	}
	return s, nil
}

func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	peer := NetPeer{Resolver: scope.Resolver}
	return RunWithPeer(peer, spec.Batch, scope, w)
}

func RunWithPeer(peer Peer, batch string, scope Scope, w io.Writer) error {
	var out []wire.Observation
	for _, name := range scope.Names {
		res := Resolve(peer, scope.Offers, name)
		if res.Unreachable {
			// The declared resolver could not be reached. A batch that failed
			// outright covers nothing and licenses no absence (CONTEXT.md Batch),
			// so we emit nothing and return an error: the worker routes this
			// through retry → dead-letter, which records the empty scope, and the
			// vantage's Availability is marked unavailable (ADR-0108).
			return fmt.Errorf("resolutionwalk: declared resolver %q unreachable; batch covers nothing", scope.Resolver)
		}
		out = append(out, Emit(batch, scope.Vantage, res)...)
	}
	return WriteNDJSON(w, out)
}
