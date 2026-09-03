package corpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// RenderRow runs every step of a row through the leaf against its scripted peer
// and the deterministic control-label generator, returning the concatenated
// NDJSON. It is hermetic: the peer is in-process and the labels are fixed, so
// nothing here touches the network or a container.
func RenderRow(r Row) ([]byte, error) {
	var buf bytes.Buffer
	for _, step := range r.Steps {
		scope := wd.Scope{
			Vantage:    step.Vantage,
			Resolver:   step.Resolver,
			Names:      step.Names,
			SeedScopes: step.SeedScopes,
			Offers:     r.Params.Offers,
		}
		if err := wd.RunWithPeer(step.Peer, step.Batch, scope, DeterministicLabels{}, &buf); err != nil {
			return nil, fmt.Errorf("corpus: render row %q: %w", r.Golden, err)
		}
	}
	return buf.Bytes(), nil
}

func RenderAll() (map[string][]byte, error) {
	out := make(map[string][]byte, len(Rows))
	for _, r := range Rows {
		b, err := RenderRow(r)
		if err != nil {
			return nil, err
		}
		out[r.Golden] = b
	}
	return out, nil
}

func CorpusDigest(rendered map[string][]byte) string {
	names := make([]string, 0, len(rendered))
	for n := range rendered {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		h.Write([]byte(n))
		h.Write([]byte{0})
		h.Write(rendered[n])
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func ParamsDigest() string { return wd.DefaultParams().Digest() }

// UncoveredMove is one row of golden-corpus.md §9's register: a version bump
// justified by an input class the corpus cannot reach. Append-only.
type UncoveredMove struct {
	Leaf       string `json:"leaf"`
	BumpedTo   string `json:"bumped_to"`
	InputClass string `json:"input_class"`
	Ticket     string `json:"ticket"`
	Date       string `json:"date"`
}

type Lock struct {
	LeafVersion    string          `json:"leaf_version"`
	CorpusDigest   string          `json:"corpus_digest"`
	ParamsDigest   string          `json:"params_digest"`
	UncoveredMoves []UncoveredMove `json:"uncovered_moves"`
}

func LoadLock(dir string) (Lock, error) {
	b, err := os.ReadFile(filepath.Join(dir, "corpus.lock.json")) // #nosec G304 (test corpus loader; filename constant, dir is the fixed test corpus directory ".")
	if err != nil {
		return Lock{}, err
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return Lock{}, fmt.Errorf("corpus: decode lock: %w", err)
	}
	return l, nil
}

// WriteLock writes a freshly computed lock to dir. It is the deliberate "bless"
// action a maintainer takes (via the -update test flag) when an output or
// parameter change is intended and the version has been bumped to match.
func WriteLock(dir string, l Lock) error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(dir, "corpus.lock.json"), b, 0o600)
}
