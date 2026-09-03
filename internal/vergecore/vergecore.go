// Package vergecore is the daily hot-tier port set: `verge-core`, the union
// `frequency-set ∪ sensitive-list` (v1 spec §3.5, ADR-0009). It ships as an
// editable list file (`verge-core.tsv`, embedded), and only the frequency half
// is operator-editable — the sensitive half is authored by the project and
// ships in the release, because moving one would move a version and `Break` the
// estate without a release and without a golden-corpus row moving.
//
// Composed, `verge-core` is 136 pairs (131 TCP, 5 UDP). Only the TCP pairs are
// probed on default settings, since UDP is off — `connect-outcome` cannot
// decide an honest UDP value at all (ADR-0083), so the UDP pairs are recorded in
// scope but never produce a probe. The package is database-free and pure: the
// operator's frequency edits arrive as a `FrequencyEdit` slice a caller reads
// from wherever it stores them, and the union is computed here.
package vergecore

import (
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Transport is the wire transport of a port pair. Only `tcp` is probed by
// default; `udp` is recorded and never probed.
type Transport string

const (
	TCP Transport = "tcp"
	UDP Transport = "udp"
)

type Half string

const (
	Frequency Half = "frequency"
	// Sensitive is the curated never-legitimately-internet-facing list. Authored
	// by the release and never operator-editable (§3.5).
	Sensitive Half = "sensitive"
)

type Pair struct {
	Port      uint16    `json:"port"`
	Transport Transport `json:"transport"`
}

func (p Pair) String() string { return strconv.Itoa(int(p.Port)) + "/" + string(p.Transport) }

// less orders pairs by port then transport, so every enumeration this package
// produces is deterministic (tcp before udp on a tie).
func (p Pair) less(o Pair) bool {
	if p.Port != o.Port {
		return p.Port < o.Port
	}
	return p.Transport < o.Transport
}

type List struct {
	frequency map[Pair]struct{}
	sensitive map[Pair]struct{}
}

//go:embed verge-core.tsv
var shipped string

// shippedList is the parsed shipped file, computed once. A parse failure here is
// a build-time authoring error in the checked-in file, so it panics rather than
// shipping a half-read set.
var shippedList = mustParse(shipped)

func Default() List { return shippedList.clone() }

// Parse reads a `verge-core.tsv` body into a List. Lines are
// `port<TAB>transport<TAB>half`; blank lines and `#` comments are skipped. It is
// exported so a test can parse an alternative body, but production reads only the
// embedded shipped file through Default.
func Parse(body string) (List, error) {
	l := List{frequency: map[Pair]struct{}{}, sensitive: map[Pair]struct{}{}}
	for n, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 3 {
			return List{}, fmt.Errorf("vergecore: line %d: want 3 tab-separated fields, got %d", n+1, len(fields))
		}
		port, err := strconv.ParseUint(strings.TrimSpace(fields[0]), 10, 16)
		if err != nil || port == 0 {
			return List{}, fmt.Errorf("vergecore: line %d: bad port %q", n+1, fields[0])
		}
		tr := Transport(strings.TrimSpace(fields[1]))
		if tr != TCP && tr != UDP {
			return List{}, fmt.Errorf("vergecore: line %d: bad transport %q", n+1, fields[1])
		}
		pair := Pair{Port: uint16(port), Transport: tr}
		switch Half(strings.TrimSpace(fields[2])) {
		case Frequency:
			l.frequency[pair] = struct{}{}
		case Sensitive:
			l.sensitive[pair] = struct{}{}
		default:
			return List{}, fmt.Errorf("vergecore: line %d: bad half %q", n+1, fields[2])
		}
	}
	return l, nil
}

func mustParse(body string) List {
	l, err := Parse(body)
	if err != nil {
		panic(err)
	}
	return l
}

func (l List) clone() List {
	c := List{frequency: make(map[Pair]struct{}, len(l.frequency)), sensitive: make(map[Pair]struct{}, len(l.sensitive))}
	for p := range l.frequency {
		c.frequency[p] = struct{}{}
	}
	for p := range l.sensitive {
		c.sensitive[p] = struct{}{}
	}
	return c
}

// IsSensitive reports whether a pair is on the sensitive half. It is what the
// UI's edit guard reads: a pair the operator cannot move.
func (l List) IsSensitive(p Pair) bool {
	_, ok := l.sensitive[p]
	return ok
}

func (l List) IsFrequency(p Pair) bool {
	_, ok := l.frequency[p]
	return ok
}

func (l List) Union() []Pair {
	seen := map[Pair]struct{}{}
	for p := range l.frequency {
		seen[p] = struct{}{}
	}
	for p := range l.sensitive {
		seen[p] = struct{}{}
	}
	return sortedPairs(seen)
}

// TCPProbed is the union's TCP pairs, sorted — the pairs actually connected to
// on default settings. UDP is off, so this is the set the connect-outcome leaf
// receives per address.
func (l List) TCPProbed() []Pair {
	out := map[Pair]struct{}{}
	for _, p := range l.Union() {
		if p.Transport == TCP {
			out[p] = struct{}{}
		}
	}
	return sortedPairs(out)
}

// UDPRecorded is the union's UDP pairs, sorted — recorded in scope but never
// probed (§3.5; ADR-0083).
func (l List) UDPRecorded() []Pair {
	out := map[Pair]struct{}{}
	for _, p := range l.Union() {
		if p.Transport == UDP {
			out[p] = struct{}{}
		}
	}
	return sortedPairs(out)
}

func (l List) FrequencyPairs() []Pair { return sortedSet(l.frequency) }

func (l List) SensitivePairs() []Pair { return sortedSet(l.sensitive) }

type Counts struct {
	Frequency int // pairs on the frequency half
	Sensitive int // pairs on the sensitive half
	Union     int // distinct pairs
	TCP       int // distinct TCP pairs (probed)
	UDP       int // distinct UDP pairs (recorded, never probed)
}

func (l List) Count() Counts {
	u := l.Union()
	tcp, udp := 0, 0
	for _, p := range u {
		if p.Transport == TCP {
			tcp++
		} else {
			udp++
		}
	}
	return Counts{
		Frequency: len(l.frequency),
		Sensitive: len(l.sensitive),
		Union:     len(u),
		TCP:       tcp,
		UDP:       udp,
	}
}

type FrequencyEdit struct {
	Port   uint16
	Action string // "add" | "remove"
}

const (
	ActionAdd    = "add"
	ActionRemove = "remove"
)

// WithFrequencyEdits returns a copy of the list with the operator's frequency
// edits applied. It touches ONLY the frequency half: an edit can never add or
// remove a sensitive pair, so a removed pair that is also sensitive stays in the
// Union (moving the sensitive half would move a version — §3.5). Non-TCP or
// zero-port edits are ignored, since the frequency half is TCP-only. Edits are
// applied in slice order, so a later edit wins over an earlier one on the same
// port.
func (l List) WithFrequencyEdits(edits []FrequencyEdit) List {
	c := l.clone()
	for _, e := range edits {
		if e.Port == 0 {
			continue
		}
		p := Pair{Port: e.Port, Transport: TCP}
		switch e.Action {
		case ActionAdd:
			c.frequency[p] = struct{}{}
		case ActionRemove:
			delete(c.frequency, p)
		}
	}
	return c
}

func sortedPairs(set map[Pair]struct{}) []Pair { return sortedSet(set) }

func sortedSet(set map[Pair]struct{}) []Pair {
	out := make([]Pair, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].less(out[j]) })
	return out
}
