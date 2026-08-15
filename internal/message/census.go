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

// CensusEntry is one thing that opened beneath the fired-at subject: a facet
// whose handshake rode the newly-reached Service, a Service or Endpoint that
// entered beneath a membership root, or a rule that opened at `fired` and has no
// firing edge of its own (a move carries the rule that opens beneath it).
type CensusEntry struct {
	// Kind is what opened: "facet", "service", "endpoint", "name", "address" or
	// "signal". It is display-only — the census is a flat enumerable payload, not
	// a tree to walk.
	Kind string `json:"kind"`
	// Key is the facet name, subject key or rule name that opened.
	Key string `json:"key"`
}

// Len is the census size — the count a headline states. Each member's count IS
// the length of its own list (ADR-0102), so the census is never sampled, ranked
// or truncated.
func (c Census) Len() int { return len(c.Entries) }

// Marshal renders the census to the JSONB bytes the store holds. A census with
// no entries marshals to a present-but-empty payload, which is distinct from the
// SQL NULL a firing with no census carries at all.
func (c Census) Marshal() ([]byte, error) { return json.Marshal(c) }

// ParseCensus reads a census back from the store's JSONB bytes. Empty or nil
// input yields an empty census — a stored NULL means the firing carried none.
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
