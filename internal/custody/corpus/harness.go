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

// Estate builds the row's `custody.Estate`. The `edge-fanout` input is DERIVED
// from the row's observed SAN sets through `custody.SharedEdge`, never written
// into the row: that is what puts the shipped threshold inside the corpus digest,
// so a value move rewrites every boundary golden.
//
// An address absent from Observed is absent from the Shared map, which is the
// measurement-pending state the absence rule reads.
//
// The parses are Must-shaped, matching the measure corpora. A row is checked-in
// data, so a malformed literal is a build-time authoring error rather than a
// runtime condition worth a value.
func (s Step) Estate() custody.Estate {
	scopes := make([]netip.Prefix, 0, len(s.AddressScopes))
	for _, c := range s.AddressScopes {
		scopes = append(scopes, netip.MustParsePrefix(c).Masked())
	}
	res := make([]custody.Resolution, 0, len(s.Resolutions))
	for _, r := range s.Resolutions {
		res = append(res, custody.Resolution{Owner: r.Owner, Address: netip.MustParseAddr(r.Address).Unmap()})
	}
	fanout := custody.EdgeFanout{Enabled: s.ScanInForce}
	if len(s.Observed) > 0 {
		fanout.Shared = make(map[netip.Addr]bool, len(s.Observed))
		for addr, sans := range s.Observed {
			fanout.Shared[netip.MustParseAddr(addr).Unmap()] = custody.SharedEdge(sans)
		}
	}
	return custody.Estate{
		AddressScopes: scopes,
		ExtendedZones: s.ExtendedZones,
		Resolutions:   res,
		EdgeFanout:    fanout,
	}
}

// observed keys the row's SAN sets by parsed address, so a row's spelling of an
// address cannot drift from the derivation's family-agnostic comparison.
func (s Step) observed() map[netip.Addr][]string {
	out := make(map[netip.Addr][]string, len(s.Observed))
	for addr, sans := range s.Observed {
		out[netip.MustParseAddr(addr).Unmap()] = sans
	}
	return out
}

// line is one rendered NDJSON record: everything the `Custody` derivation
// concluded about one address, from the measurement in to the gate out. Fields
// render in declaration order, so the encoding is stable without a sort.
type line struct {
	Address string `json:"address"`
	// Measured reports whether the `edge-fanout` Scan recorded a row for this
	// address. False is measurement PENDING, and it is not a count band.
	Measured bool `json:"measured"`
	// FanOut is the count of distinct registrable domains the SAN set reduces to,
	// and SharedEdge the verdict the threshold gives that count. Both are null
	// where nothing measured, because an absence is never a value.
	FanOut     *int  `json:"fanout"`
	SharedEdge *bool `json:"shared_edge"`
	// ExtensionCandidate reports membership of the EXTENSION LIMB of the population
	// the Scan measures. It reads the PRE-veto reach, so a vetoed edge is still true
	// here and a later handshake can lift the veto. Since #988 the Scan also measures
	// the declaration limb, which this column does NOT report: there the result labels
	// and gates nothing, and AddressScopeCovered below is that limb's column.
	ExtensionCandidate bool `json:"extension_candidate"`
	// AddressScopeCovered reports whether a declared address-scope `Seed` covers
	// the address — the OTHER limb of subject membership, which the veto never
	// reaches.
	AddressScopeCovered bool `json:"address_scope_covered"`
	// Custody is the derived value, and MayProbeInternet the total probing gate
	// that reads it from an `internet`-class Vantage.
	Custody          string `json:"custody"`
	MayProbeInternet bool   `json:"may_probe_internet"`
}

// RenderRow runs a row's step through the `Custody` derivation and returns the
// NDJSON it renders, one line per address under test in the row's declared order.
// It is hermetic: the estate is in-process data, so nothing here touches the
// network, a database or a container.
func RenderRow(r Row) ([]byte, error) {
	estate := r.Step.Estate()
	candidates := make(map[netip.Addr]struct{})
	for _, a := range estate.ExtensionCandidates() {
		candidates[a] = struct{}{}
	}
	observed := r.Step.observed()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, spelling := range r.Step.Under {
		addr, err := netip.ParseAddr(spelling)
		if err != nil {
			return nil, fmt.Errorf("corpus: render row %q: address %q: %w", r.Golden, spelling, err)
		}
		addr = addr.Unmap()
		_, isCandidate := candidates[addr]
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
// order. It moves exactly when a row's expected output moves, which binds an
// output change to a derivation-version bump through the lock.
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

// ParamsDigest is the digest of the derivation's declared-parameter set — the
// fan-out threshold and the Public Suffix List revision the reduction reads — the
// second thing a version bump may be justified by. It reflects the SHIPPED
// parameters (custody.DefaultParams), so a threshold move or a dependency bump
// that ships a newer list moves the lock even where no row's output happened to
// cross the boundary.
func ParamsDigest() string { return custody.DefaultParams().Digest() }

// UncoveredMove is one row of golden-corpus.md §9's register: a version bump
// justified by an input class the corpus cannot reach. Append-only.
type UncoveredMove struct {
	Leaf       string `json:"leaf"`
	BumpedTo   string `json:"bumped_to"`
	InputClass string `json:"input_class"`
	Ticket     string `json:"ticket"`
	Date       string `json:"date"`
}

// Lock is the checked-in manifest that binds the derivation version to the corpus
// and parameter digests. A lock edit that bumps the version with no digest move
// and no new uncovered move is what CI's version gate refuses.
//
// The JSON keys match the measure corpora's, so CI's `corpus-version-gate` reads
// every lock with one script. `leaf_version` is ADR-0008's own word: the version
// vector holds one leaf per named `Derivation`, and this is `Custody`'s.
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
