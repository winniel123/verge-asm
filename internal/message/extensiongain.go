package message

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// The unit is the scope, never the address: one act on the declaration (ADR-0022, ADR-0013 #55).

const KindAddress = "address"

type ExtensionGain struct {
	Address string
	Names   []string
}

func ExtensionGained(scope string, gains []ExtensionGain, covered int, instant time.Time) *Message {
	// A departure is the self-correction working, so only a gain fires (ADR-0013 #55).
	if len(gains) == 0 {
		return nil
	}
	entries := make([]CensusEntry, 0, len(gains))
	for _, g := range gains {
		names := append([]string(nil), g.Names...)
		sort.Strings(names)
		entries = append(entries, CensusEntry{Kind: KindAddress, Key: g.Address, Detail: strings.Join(names, ", ")})
	}
	c := NewCensus(entries...)
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassForCause(CauseAperture),
		SubjectKind: "seed",
		FiredAt:     scope,
		Instant:     instant,
		Census:      &c,
		Headline:    extensionGainHeadline(scope, c, covered),
	}
}

func extensionGainHeadline(scope string, census Census, covered int) string {
	// The difference and the count, no verdict: the census is the check (ADR-0013 #55).
	labels := make([]string, 0, census.Len())
	for _, e := range census.Entries {
		labels = append(labels, e.Label())
	}
	return fmt.Sprintf("%s custody extension gained %s: %s · %s covered",
		scope, plural(census.Len(), "address", "addresses"), strings.Join(labels, ", "),
		plural(covered, "address", "addresses"))
}
