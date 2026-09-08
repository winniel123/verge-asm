package message

import (
	"encoding/json"
	"sort"
)

// An opening reaches nobody on its own, so it rides here and fires no message (ADR-0031).

type Census struct {
	Entries []CensusEntry `json:"entries"`
}

// A rule entry is the carrier of a rule that opened at fired beneath the cause (ADR-0033 §3).

const (
	KindFacet = "facet"
	KindRule  = "rule"
)

type CensusEntry struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`

	// The sensitive-port rule names its port here; every other entry leaves it empty (#1723).

	Detail string `json:"detail,omitempty"`
}

func (e CensusEntry) Label() string {
	if e.Detail == "" {
		return e.Key
	}
	return e.Key + " (" + e.Detail + ")"
}

func (c Census) Rules() []CensusEntry {
	var out []CensusEntry
	for _, e := range c.Entries {
		if e.Kind == KindRule {
			out = append(out, e)
		}
	}
	return out
}

func (c Census) Facets() []CensusEntry {
	var out []CensusEntry
	for _, e := range c.Entries {
		if e.Kind == KindFacet {
			out = append(out, e)
		}
	}
	return out
}

// Never sampled, ranked or truncated: a count is its own list's length (ADR-0102).

func (c Census) Len() int { return len(c.Entries) }

func (c Census) Marshal() ([]byte, error) { return json.Marshal(c) }

func ParseCensus(b []byte) (Census, error) {
	if len(b) == 0 {
		return Census{}, nil
	}
	var c Census
	if err := json.Unmarshal(b, &c); err != nil {
		return Census{}, err
	}
	return c, nil
}

func NewCensus(entries ...CensusEntry) Census {
	out := append([]CensusEntry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	// An empty census is legal: a Service reaching with no facet open carries length zero.
	return Census{Entries: out}
}
