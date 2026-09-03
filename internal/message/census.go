package message

import (
	"encoding/json"
	"sort"
)

// Census is the payload a flagship or membership message carries: the rows that
// opened beneath the fired-at subject (CONTEXT.md `Reach`, `Subject`, ADR-0031).
// It is enumerable in full — every entry is a real timeline that opened — and it
// carries no comparison, no rows of difference and no valence word. A count is
// stated with the factors that produced it, so the census is the factors and the
// length is the count.
//
// It is the only carrier a newly-entering Service or Endpoint has: an opening
// reaches nobody on its own (the facet layer is evidence, not a channel), so
// every timeline opening beneath a new subject rides here rather than firing its
// own message.
type Census struct {
	Entries []CensusEntry `json:"entries"`
}

type CensusEntry struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

// Len is the census size — the count a headline states. Each member's count IS
// the length of its own list (ADR-0102), so the census is never sampled, ranked
// or truncated.
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

// NewCensus returns a census over the given entries, sorted deterministically by
// (kind, key) so a stored census renders the same every read. A nil or empty
// entry list yields an empty census, which is legal — a Service reaching with no
// facet yet open carries a census of length zero.
func NewCensus(entries ...CensusEntry) Census {
	out := append([]CensusEntry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	return Census{Entries: out}
}
