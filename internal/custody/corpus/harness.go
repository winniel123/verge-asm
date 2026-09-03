package corpus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"

	"github.com/winniel123/verge-asm/internal/custody"
)

func (s Step) Estate() custody.Estate {
	// A row is checked-in data, so a malformed literal is an authoring error, not a runtime value.
	scopes := make([]netip.Prefix, 0, len(s.AddressScopes))
	for _, c := range s.AddressScopes {
		scopes = append(scopes, netip.MustParsePrefix(c).Masked())
	}
	excluded := make([]netip.Prefix, 0, len(s.AddressExclusions))
	for _, c := range s.AddressExclusions {
		excluded = append(excluded, netip.MustParsePrefix(c).Masked())
	}
	res := make([]custody.Resolution, 0, len(s.Resolutions))
	for _, r := range s.Resolutions {
		res = append(res, custody.Resolution{Owner: r.Owner, Address: netip.MustParseAddr(r.Address).Unmap()})
	}
	fanout := custody.EdgeFanout{Enabled: s.ScanInForce, BatchCompleted: s.ScanBatchCompleted}
	if len(s.Observed) > 0 {
		// The verdict is derived here, never written into the row, so the threshold is inside the digest.
		fanout.Shared = make(map[netip.Addr]bool, len(s.Observed))
		for addr, sans := range s.Observed {
			fanout.Shared[netip.MustParseAddr(addr).Unmap()] = custody.SharedEdge(sans)
		}
	}
	// Built through the assemblers production uses, so no row renders an unreachable state (#1018).
	return custody.Estate{
		AddressScopes: scopes,
		ExtendedZones: s.ExtendedZones,
		Resolutions:   res,
	}.WithAddressExclusions(excluded).WithEdgeFanout(fanout)
}

func (s Step) observed() map[netip.Addr][]string {
	out := make(map[netip.Addr][]string, len(s.Observed))
	for addr, sans := range s.Observed {
		out[netip.MustParseAddr(addr).Unmap()] = sans
	}
	return out
}

type line struct {
	Address             string `json:"address"`
	Measured            bool   `json:"measured"`
	FanOut              *int   `json:"fanout"`
	SharedEdge          *bool  `json:"shared_edge"`
	ExtensionCandidate  bool   `json:"extension_candidate"`
	AddressScopeCovered bool   `json:"address_scope_covered"`
	Custody             string `json:"custody"`
	MayProbeInternet    bool   `json:"may_probe_internet"`
}

func RenderRow(r Row) ([]byte, error) {
	estate := r.Step.Estate()
	// The candidate set is the PRE-veto reach, so a vetoed edge still reads true and can be lifted.
	candidates := make(map[netip.Addr]struct{})
	for _, a := range estate.ExtensionCandidates() {
		candidates[a] = struct{}{}
	}
	observed := r.Step.observed()

	var buf bytes.Buffer
	// encoding/json renders fields in declaration order, so the golden is stable without a sort.
	enc := json.NewEncoder(&buf)
	for _, spelling := range r.Step.Under {
		addr, err := netip.ParseAddr(spelling)
		if err != nil {
			return nil, fmt.Errorf("corpus: render row %q: address %q: %w", r.Golden, spelling, err)
		}
		addr = addr.Unmap()
		_, isCandidate := candidates[addr]
		// The declaration limb the Scan also measures is no column, because there a result gates nothing.
		rec := line{
			Address:             addr.String(),
			ExtensionCandidate:  isCandidate,
			AddressScopeCovered: estate.CoversAddressScope(addr),
			Custody:             string(estate.Derive(addr)),
			MayProbeInternet:    estate.MayProbe(addr, custody.ClassInternet),
		}
		if sans, ok := observed[addr]; ok {
			count := custody.FanOut(sans)
			shared := custody.SharedEdge(sans)
			rec.Measured, rec.FanOut, rec.SharedEdge = true, &count, &shared
		}
		if err := enc.Encode(rec); err != nil {
			return nil, fmt.Errorf("corpus: render row %q: encode %s: %w", r.Golden, addr, err)
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

// Shipped parameters, so a threshold or list move shifts the lock even where no golden moved.

func ParamsDigest() string { return custody.DefaultParams().Digest() }

// golden-corpus.md §9's append-only register of bumps the corpus cannot itself justify.

type UncoveredMove struct {
	Leaf       string `json:"leaf"`
	BumpedTo   string `json:"bumped_to"`
	InputClass string `json:"input_class"`
	Ticket     string `json:"ticket"`
	Date       string `json:"date"`
}

// The JSON keys match the measure corpora's, so one corpus-version-gate script reads every lock.

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

func WriteLock(dir string, l Lock) error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(dir, "corpus.lock.json"), b, 0o600)
}
