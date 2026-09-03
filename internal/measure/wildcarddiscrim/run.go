package wildcarddiscrim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

const Kind = "wildcard-discrimination"

type Scope struct {
	Vantage    string    `json:"vantage"`
	Resolver   string    `json:"resolver"`
	Names      []string  `json:"names"`
	SeedScopes []string  `json:"seed_scopes"`
	Offers     rw.Offers `json:"offers"`
}

// A change here is a declared-parameter change that may justify a Version bump (ADR-0021).

type Params struct {
	RandomLabelCount     int       `json:"random_label_count"`
	StructuredLabelCount int       `json:"structured_label_count"`
	Offers               rw.Offers `json:"offers"`
}

func DefaultParams() Params {
	return Params{
		RandomLabelCount:     RandomLabelCount,
		StructuredLabelCount: StructuredLabelCount,
		Offers:               rw.DefaultOffers(),
	}
}

func (p Params) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("wildcarddiscrim: marshal params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func DecodeScope(spec wire.JobSpec) (Scope, error) {
	var s Scope
	if len(spec.Scope) == 0 {
		return Scope{}, fmt.Errorf("wildcarddiscrim: job spec has no scope")
	}
	if err := json.Unmarshal(spec.Scope, &s); err != nil {
		return Scope{}, fmt.Errorf("wildcarddiscrim: decode scope: %w", err)
	}
	return s, nil
}

func Run(spec wire.JobSpec, w io.Writer) error {
	scope, err := DecodeScope(spec)
	if err != nil {
		return err
	}
	// Candidate and control share one declared path, so no two vantages are compared (ADR-0070).
	peer := rw.NetPeer{Resolver: scope.Resolver}
	return RunWithPeer(peer, spec.Batch, scope, CryptoLabels{}, w)
}

func RunWithPeer(peer rw.Peer, batch string, scope Scope, gen LabelGen, w io.Writer) error {
	pop := ControlPopulation(scope.Names, scope.SeedScopes)
	popSet := make(map[string]struct{}, len(pop))
	for _, p := range pop {
		popSet[p] = struct{}{}
	}

	labels := gen.Labels()
	ctrlByParent := make(map[string]controlAnswers, len(pop))
	for _, parentName := range pop {
		ctrlByParent[parentName] = probeControl(peer, scope.Offers, parentName, labels)
	}

	var out []wire.Observation
	for _, name := range scope.Names {
		res := rw.Resolve(peer, scope.Offers, name)
		verdict := decide(res, popSet, ctrlByParent)
		out = append(out, Emit(batch, scope.Vantage, res, verdict)...)
	}
	return writeNDJSON(w, out)
}

func decide(res rw.Result, popSet map[string]struct{}, ctrlByParent map[string]controlAnswers) Verdict {
	p, ok := parent(res.Name)
	if !ok {
		return VerdictNotShadowed
	}
	if _, probed := popSet[p]; !probed {
		// A parent outside the Seed gate was never probed, so the walk's value stands.
		return VerdictNotShadowed
	}
	return Discriminate(candidateComponents(res.Records), ctrlByParent[p])
}

func probeControl(peer rw.Peer, offers rw.Offers, parentName string, labels []string) controlAnswers {
	ca := controlAnswers{perLabel: make([]map[rw.Qtype][]rw.RR, 0, len(labels))}
	for _, label := range labels {
		name := rw.CanonicalName(label + "." + parentName)
		ans := make(map[rw.Qtype][]rw.RR, len(offers.Qtypes))
		for _, qt := range offers.Qtypes {
			msg := peer.Exchange(rw.Query{
				Path:      rw.PathDeclared,
				Name:      name,
				Qtype:     qt,
				Transport: rw.UDP,
				EDNS:      true,
				Cookie:    offers.EDNS.Cookie,
			})
			if msg.Reached {
				ca.reached = true
			}
			if len(msg.Answer) > 0 {
				ans[qt] = msg.Answer
			}
		}
		ca.perLabel = append(ca.perLabel, ans)
	}
	return ca
}

func writeNDJSON(w io.Writer, obs []wire.Observation) error {
	for _, o := range obs {
		if err := wire.EncodeObservation(w, o); err != nil {
			return err
		}
	}
	return nil
}
