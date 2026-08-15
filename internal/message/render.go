package message

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValenceWords are the words the message vocabulary refuses (v1 spec §5.3,
// ADR-0064): nothing is resolved, fixed, improved, critical or OK, because a
// clear is not always good news and a widening is neither. The list is exported
// so a test can assert every rendered sentence in the product is clear of them,
// and so the store's read path can be checked too. Every value is a lowercase
// whole word; ContainsValence matches on word boundaries, so a value can never
// falsely fire on a substring (`ok` inside `looked`, say).
var ValenceWords = []string{
	"resolved", "resolve", "fixed", "fix", "improved", "improve",
	"critical", "ok", "okay", "good", "bad", "safe", "unsafe",
	"danger", "dangerous", "severe", "severity", "healthy", "unhealthy",
	"clean", "warning", "warn", "urgent", "vulnerable", "vulnerability",
	"success", "failure", "risk", "threat", "worse", "better",
}

var valenceRe = func() *regexp.Regexp {
	quoted := make([]string, len(ValenceWords))
	for i, w := range ValenceWords {
		quoted[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\b`)
}()

// ContainsValence reports whether s carries any refused valence word, matched on
// word boundaries and case-insensitively. It is the guard behind the model's
// promise that no rendered message copy grades the news.
func ContainsValence(s string) bool { return valenceRe.MatchString(s) }

// Threshold returns a clock-class Message for a rule whose threshold was crossed
// with no measurement moving — the estate did not move and we did not change how
// we look, only time passed (CONTEXT.md `Message`). It links to the object whose
// span the rule read. Where the same rule instead finds the span it reads has
// moved, the caller fires CauseDrift (a drift-class firing) rather than this —
// class is a property of the firing.
func Threshold(subjectKind, subjectKey, headline string, instant time.Time) *Message {
	return &Message{
		Cause:       CauseThreshold,
		Class:       ClassClock,
		SubjectKind: subjectKind,
		FiredAt:     subjectKey,
		Instant:     instant,
		Headline:    headline,
	}
}

// DeclaredInput returns a Message whose mover is the operator's own declared
// input — a Source they toggled, a zone file they re-supplied. It links to the
// Source the rule reads, and rides the coverage class (we changed what we are
// told). sourceKey is the Source identity the row links to.
func DeclaredInput(sourceKey, headline string, instant time.Time) *Message {
	return &Message{
		Cause:       CauseDeclaredInput,
		Class:       ClassCoverage,
		SubjectKind: "source",
		FiredAt:     sourceKey,
		Instant:     instant,
		Headline:    headline,
	}
}

// flagshipHeadline states the internet leg reaching, with the count of facets
// that opened beneath as its factor. `reached` is a Reach value, not a valence
// word — the sentence names what moved and grades nothing.
func flagshipHeadline(serviceKey string, census Census) string {
	return fmt.Sprintf("%s reached from the internet · %s opened beneath it",
		serviceKey, plural(census.Len(), "facet", "facets"))
}

// membershipHeadline states a root entering the estate, with the count of
// timelines that opened beneath as its factor. `appeared` / `returned` /
// `revealed` are the Transition's own words and carry no valence.
func membershipHeadline(entry Entry, rootKey string, census Census) string {
	verb := map[Entry]string{
		EntryAppeared: "entered the estate",
		EntryReturned: "returned to the estate",
		EntryRevealed: "came into view",
	}[entry]
	if verb == "" {
		verb = "entered the estate"
	}
	return fmt.Sprintf("%s %s · %s opened beneath it",
		rootKey, verb, plural(census.Len(), "timeline", "timelines"))
}

// narrowingHeadline states a scope narrowing with its two counts as factors, in
// the shape the narrowing-receipt prototype fixed (#167): the scope narrowed,
// the value excluded, the subjects withdrawn, the timelines taken out.
func narrowingHeadline(scope, removed string, subjects, timelines int) string {
	return fmt.Sprintf("%s narrowed · %s excluded · %s withdrawn · %s taken out of the estate",
		scope, removed,
		plural(subjects, "subject", "subjects"),
		plural(timelines, "timeline", "timelines"))
}

// narrowingLoss names what can no longer be told — the one payload element that
// is not a mirror of the widening receipt, because the narrowing cannot be
// corrected afterwards and naming it is the whole of the remedy (ADR-0074).
func narrowingLoss(removed string) string {
	return fmt.Sprintf("A listener answering inside %s after this act is not seen, "+
		"and no later message names it.", removed)
}

// plural renders a count with its noun, thousands-separated so a large factor
// reads (17,920 rather than 17920).
func plural(n int, one, many string) string {
	noun := many
	if n == 1 {
		noun = one
	}
	return fmt.Sprintf("%s %s", group(n), noun)
}

// group inserts thousands separators into a non-negative integer.
func group(n int) string {
	s := fmt.Sprintf("%d", n)
	if n < 1000 {
		return s
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
