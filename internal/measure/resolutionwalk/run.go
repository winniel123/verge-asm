package resolutionwalk

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/winniel123/verge-asm/internal/wire"
)

const Kind = "resolution-walk"

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
			// A failed batch licenses no absence, so this dead-letters (ADR-0108, CONTEXT.md).
			return fmt.Errorf("resolutionwalk: declared resolver %q unreachable; batch covers nothing", scope.Resolver)
		}
		out = append(out, Emit(batch, scope.Vantage, res)...)
	}
	return WriteNDJSON(w, out)
}
