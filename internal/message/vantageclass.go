package message

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// The class is the aperture input, so a second vantage of a live class widens nothing (ADR-0029).

const KindVantageClass = "vantage-class"

func VantageClassWidened(class string, started []CensusEntry, instant time.Time) *Message {
	if class == "" {
		return nil
	}
	c := NewCensus(started...)
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassForCause(CauseAperture),
		SubjectKind: KindVantageClass,
		FiredAt:     class,
		Instant:     instant,
		Census:      &c,
		Headline:    vantageClassHeadline(class, c),
	}
}

func vantageClassHeadline(class string, census Census) string {
	head := fmt.Sprintf("%s %s vantage is now configured", article(class), class)
	if census.Len() == 0 {
		// Nothing composed is the honest reading of the widening, not a reason to stay silent.
		return head + " · no Exposure timeline opened"
	}
	// A census is a description and never a comparison, so no value is graded (ADR-0029).
	return fmt.Sprintf("%s · %s opened · %s", head,
		plural(census.Len(), "Exposure timeline", "Exposure timelines"), exposureTally(census))
}

func article(class string) string {
	if class != "" && strings.ContainsRune("aeiou", rune(class[0])) {
		return "an"
	}
	return "a"
}

func exposureTally(census Census) string {
	byValue := map[string]int{}
	for _, e := range census.Entries {
		byValue[e.Detail]++
	}
	values := make([]string, 0, len(byValue))
	for v := range byValue {
		values = append(values, v)
	}
	sort.Strings(values)
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%s %s", group(byValue[v]), v))
	}
	return strings.Join(parts, ", ")
}
