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

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// RenderRow runs every step of a row through the leaf against its scripted peer
// and returns the concatenated NDJSON. It is hermetic: the peer is in-process,
// so nothing here touches the network or a container.
func RenderRow(r Row) ([]byte, error) {
	var buf bytes.Buffer
	for _, step := range r.Steps {
		scope := resolutionwalk.Scope{
			Vantage:  step.Vantage,
			Resolver: step.Resolver,
			Names:    step.Names,
			Offers:   r.Offers,
		}
		if err := resolutionwalk.RunWithPeer(step.Peer, step.Batch, scope, &buf); err != nil {
			return nil, fmt.Errorf("corpus: render row %q: %w", r.Golden, err)
		}
	}
	return buf.Bytes(), nil
}

// RenderAll renders every row, keyed by its golden filename.
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

// CorpusDigest is a stable hash over the rendered corpus, in golden-filename
// order. It moves exactly when a row's expected output moves, which is what
// binds an output change to a leaf-version bump through the lock.
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

// ParamsDigest is the digest of the leaf's declared-parameter set (the offers),
// the second thing a version bump may be justified by.
func ParamsDigest() string { return resolutionwalk.DefaultOffers().Digest() }

// UncoveredMove is one row of golden-corpus.md §9's register: a version bump
// justified by an input class the corpus cannot reach. Append-only.
type UncoveredMove struct {
	Leaf       string `json:"leaf"`
	BumpedTo   string `json:"bumped_to"`
	InputClass string `json:"input_class"`
	Ticket     string `json:"ticket"`
	Date       string `json:"date"`
}

// Lock is the checked-in manifest that binds the leaf version to the corpus and
// parameter digests. Editing an output or a parameter forces a lock edit; a lock
// edit that bumps the version with no digest move and no new uncovered move is
// what A6 (in CI, against the base commit) refuses.
type Lock struct {
	LeafVersion    string          `json:"leaf_version"`
	CorpusDigest   string          `json:"corpus_digest"`
	ParamsDigest   string          `json:"params_digest"`
	UncoveredMoves []UncoveredMove `json:"uncovered_moves"`
}

// LoadLock reads corpus.lock.json from dir.
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
