package certcorpus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

func RenderRow(r Row) ([]byte, error) {
	var buf bytes.Buffer
	// An empty control-port set leaves ADR-0104's composition unable to move this output.
	if err := co.RunExchange(context.Background(), r.Step.Connect, r.Step.Handshake, blanketdiscrim.FixedPorts{}, r.Step.Batch, r.Step.Scope, &buf); err != nil {
		return nil, fmt.Errorf("certcorpus: render row %q: %w", r.Golden, err)
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

func ParamsDigest() string { return co.DefaultHandshakeParams().Digest() }

type UncoveredMove struct { // golden-corpus.md §9's append-only register
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
		return Lock{}, fmt.Errorf("certcorpus: decode lock: %w", err)
	}
	return l, nil
}

func WriteLock(dir string, l Lock) error {
	// The deliberate bless action: an output move needs an intended version bump first (ADR-0021).
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(dir, "corpus.lock.json"), b, 0o600)
}
