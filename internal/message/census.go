package message

import (
	"encoding/json"
	"sort"
)

// An opening reaches nobody on its own, so it rides here and fires no message (ADR-0031).

type Census struct {
	Entries []CensusEntry `json:"entries"`
}

type CensusEntry struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
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
