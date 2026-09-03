// Package vergecore is the daily hot-tier port set `verge-core`, the union
// `frequency-set ∪ sensitive-list` (v1 spec §3.5, ADR-0009). Only the frequency
// half is operator-editable; moving the sensitive half would move a version.
package vergecore

import (
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Transport string

const (
	TCP Transport = "tcp"
	UDP Transport = "udp"
)

type Half string

const (
	Frequency Half = "frequency"
	Sensitive Half = "sensitive" // authored by the release, never operator-editable (§3.5)
)

type Pair struct {
	Port      uint16    `json:"port"`
	Transport Transport `json:"transport"`
}

func (p Pair) String() string { return strconv.Itoa(int(p.Port)) + "/" + string(p.Transport) }

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

var shippedList = mustParse(shipped)

func Default() List { return shippedList.clone() }

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

func (l List) IsSensitive(p Pair) bool {
	// The operator cannot move a sensitive pair, which is what the UI's edit guard reads (§3.5).
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

func (l List) TCPProbed() []Pair {
	out := map[Pair]struct{}{}
	for _, p := range l.Union() {
		if p.Transport == TCP {
			out[p] = struct{}{}
		}
	}
	return sortedPairs(out)
}

func (l List) UDPRecorded() []Pair {
	// Nothing probes these: connect-outcome cannot decide an honest UDP value (ADR-0083, §3.5).
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
	Frequency int
	Sensitive int
	Union     int
	TCP       int
	UDP       int
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
	Action string
}

const (
	ActionAdd    = "add"
	ActionRemove = "remove"
)

func (l List) WithFrequencyEdits(edits []FrequencyEdit) List {
	// A sensitive pair never moves: an operator edit would move a version without a release (§3.5).
	c := l.clone()
	for _, e := range edits {
		if e.Port == 0 {
			continue
		}
		// The frequency half is TCP-only, so a non-TCP edit names no pair.
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
