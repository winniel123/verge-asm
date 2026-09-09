package message

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type GapClosure struct {
	Kind   string
	Key    string
	Facet  string
	Before string
	After  string
	Broke  bool
}

// A Gap closing is neither drift nor a clear, so it says we resumed and grades nothing (ADR-0014).

func GapClosed(scope, scopeKind string, closures []GapClosure, rules []CensusEntry, instant time.Time) *Message {
	if len(closures) == 0 {
		return nil
	}
	type subject struct {
		kind, key      string
		pairs          []string
		noEarlierValue bool
		broke          bool
	}
	index := map[[2]string]int{}
	var subjects []subject
	for _, c := range closures {
		id := [2]string{c.Kind, c.Key}
		i, ok := index[id]
		if !ok {
			i = len(subjects)
			index[id] = i
			subjects = append(subjects, subject{kind: c.Kind, key: c.Key})
		}
		s := &subjects[i]
		switch {
		case c.Broke:
			s.broke = true
		case c.Before == "":
			s.noEarlierValue = true
		case c.Before != c.After:
			s.pairs = append(s.pairs, fmt.Sprintf("%s %s → %s", c.Facet, c.Before, c.After))
		}
	}
	entries := make([]CensusEntry, 0, len(subjects))
	var differ []string
	noEarlierValue, broke := 0, 0
	for _, s := range subjects {
		e := CensusEntry{Kind: s.kind, Key: s.key}
		switch {
		case len(s.pairs) > 0:
			sort.Strings(s.pairs)
			e.Detail = strings.Join(s.pairs, "; ")
			differ = append(differ, e.Label())
		case s.broke:
			broke++
		case s.noEarlierValue:
			noEarlierValue++
		}
		entries = append(entries, e)
	}
	sort.Strings(differ)
	seenRule := map[[2]string]bool{}
	for _, r := range rules {
		if id := [2]string{r.Key, r.Detail}; !seenRule[id] {
			seenRule[id] = true
			entries = append(entries, r)
		}
	}
	c := NewCensus(entries...)
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassForCause(CauseAperture),
		SubjectKind: scopeKind,
		FiredAt:     scope,
		Instant:     instant,
		Census:      &c,
		Headline:    gapClosedHeadline(scope, len(closures), len(subjects), differ, noEarlierValue, broke, c),
	}
}

func gapClosedHeadline(scope string, timelines, subjects int, differ []string, noEarlierValue, broke int, census Census) string {
	head := fmt.Sprintf("%s · sight restored on %s across %s", scope,
		plural(timelines, "timeline", "timelines"), plural(subjects, "subject", "subjects"))
	if len(differ) == 0 {
		head += " · none differs from the last value seen"
	} else {
		head += fmt.Sprintf(" · %s from the last value seen: %s",
			plural(len(differ), "differs", "differ"), strings.Join(differ, ", "))
	}
	if noEarlierValue > 0 {
		// A timeline that opened as a Gap has no earlier value, so no pair is stated (ADR-0014).
		head += fmt.Sprintf(" · %s no earlier value", plural(noEarlierValue, "has", "have"))
	}
	if broke > 0 {
		// A break forbids the comparison, so the copy grades nothing (ADR-0008).
		head += fmt.Sprintf(" · %s not compared across a derivation move", plural(broke, "is", "are"))
	}
	return head + rulesOpenedClause(census)
}
