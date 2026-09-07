package message

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// A clear is not always good news and a widening is neither, so the copy grades nothing (ADR-0064).

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

func ContainsValence(s string) bool { return valenceRe.MatchString(s) }

func Threshold(subjectKind, subjectKey, headline string, instant time.Time) *Message {
	// A rule whose span also moved fires CauseDrift instead of this (CONTEXT.md Message).
	return &Message{
		Cause:       CauseThreshold,
		Class:       ClassClock,
		SubjectKind: subjectKind,
		FiredAt:     subjectKey,
		Instant:     instant,
		Headline:    headline,
	}
}

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

func flagshipHeadline(serviceKey string, census Census) string {
	return fmt.Sprintf("%s reached from the internet · %s opened beneath it",
		serviceKey, plural(census.Len(), "facet", "facets"))
}

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

func narrowingHeadline(scope, removed string, subjects, timelines int) string {
	// The four-factor shape the narrowing-receipt prototype fixed (#167).
	return fmt.Sprintf("%s narrowed · %s excluded · %s withdrawn · %s taken out of the estate",
		scope, removed,
		plural(subjects, "subject", "subjects"),
		plural(timelines, "timeline", "timelines"))
}

func seedWithdrawalHeadline(scope string, subjects, timelines int) string {
	// Reusing narrowingHeadline would say EXCLUDED for an act that declared none (ADR-0134 §6).
	return fmt.Sprintf("%s withdrawn · %s withdrawn · %s taken out of the estate",
		scope,
		plural(subjects, "subject", "subjects"),
		plural(timelines, "timeline", "timelines"))
}

func narrowingLoss(removed string) string {
	// A narrowing cannot be undone, so naming the loss is the whole of the remedy (ADR-0074).
	return fmt.Sprintf("A listener answering inside %s after this act is not seen, "+
		"and no later message names it.", removed)
}

func plural(n int, one, many string) string {
	noun := many
	if n == 1 {
		noun = one
	}
	return fmt.Sprintf("%s %s", group(n), noun)
}

// A signal's severity is its rule's, against the older reading of ADR-0024.

type Artifact struct {
	Title          string
	Org            string
	PeriodStart    string
	PeriodEnd      string
	DeliveryNo     int
	GeneratedAt    string
	Version        string
	Format         string
	Stats          []ArtifactStat
	SeverityCounts []ArtifactSeverityCount
	Signals        []ArtifactSignal
	Withdrawn      []ArtifactChange
	Delivered      string
	ChannelHost    string
}

type ArtifactStat struct {
	Label     string
	Value     string
	Delta     string
	DeltaTone string
	Caption   string
}

type ArtifactChange struct {
	Change  string
	Subject string
	Detail  string
}

type ArtifactSeverityCount struct {
	Level string
	Count int
}

type ArtifactSignal struct {
	Severity string
	Signal   string
	Asset    string
	Raised   string
}

func (a Artifact) Empty() bool {
	// A screen with no backing data ships an empty-state and never invented data (ADR-0110).
	return len(a.Stats) == 0 && len(a.SeverityCounts) == 0 &&
		len(a.Signals) == 0 && len(a.Withdrawn) == 0
}

// Both render forms read these, so the layouts cannot drift in what the document says (ADR-0114).

const (
	artifactSeverityTitle  = "Open signals by severity"
	artifactSignalsTitle   = "New this week"
	artifactWithdrawnTitle = "Withdrawn by the world"
	artifactWithdrawnEmpty = "The world withdrew nothing in this period."
	artifactEmptyEyebrow   = "Nothing delivered"
	artifactEmptyHeadline  = "No report has been delivered yet"
	artifactEmptyBody      = "A delivered report is rendered here once report scheduling lands and a schedule runs. Until then there is no delivery to view."
)

func RenderArtifact(a Artifact) template.HTML {
	// The standalone form needs the token vocabulary inlined: no console stylesheet is in scope.
	doc, err := renderArtifactDoc(BuildArtifactDoc(a))
	if err != nil {
		// #nosec G203 -- err text is HTML-escaped via html.EscapeString and wrapped in an HTML comment; no unescaped data reaches output.
		return template.HTML("<!-- artifact render error: " + html.EscapeString(err.Error()) + " -->")
	}
	// #nosec G203 -- artifactDocTokens is a trusted internal <style> constant; doc is html/template output (auto-escaped) built from internal report data, not attacker input.
	return template.HTML(artifactDocTokens) + doc
}

var artifactSevLevels = []string{"critical", "high", "medium", "low", "info"}

func normSev(level string) string {
	// An unknown token folds to info, never manufacturing urgency (mirrors signal.SeverityFor).
	for _, l := range artifactSevLevels {
		if l == level {
			return l
		}
	}
	return "info"
}

func sevTitle(level string) string {
	l := normSev(level)
	return strings.ToUpper(l[:1]) + l[1:]
}

func artifactProvenance(a Artifact) string {
	var parts []string
	if a.GeneratedAt != "" {
		parts = append(parts, "generated "+a.GeneratedAt)
	}
	if a.Version != "" {
		parts = append(parts, a.Version)
	}
	return strings.Join(parts, " · ")
}

func artifactReceipt(a Artifact) string {
	if a.Delivered == "" {
		return "not delivered"
	}
	line := "delivered " + a.Delivered
	// The host only: an operator's embedded token rides in the raw URL (ADR-0114 #1456).
	if a.ChannelHost != "" {
		line += " · " + a.ChannelHost
	}
	return line
}

func changeFamily(change string) string {
	// Change rides its own drift vocabulary and palette, never the severity ramp.
	switch change {
	case "appeared", "returned", "revealed":
		return "gain"
	case "withdrawn", "descoped":
		return "loss"
	default:
		return "change"
	}
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func artifactPeriod(a Artifact) string {
	var line string
	switch {
	case a.PeriodStart != "" && a.PeriodEnd != "":
		line = a.PeriodStart + " → " + a.PeriodEnd
	case a.PeriodEnd != "":
		line = a.PeriodEnd
	}
	if a.DeliveryNo > 0 {
		if line != "" {
			line += " · "
		}
		line += "delivery #" + strconv.Itoa(a.DeliveryNo)
	}
	return line
}

func ArtifactPeriod(a Artifact) string { return artifactPeriod(a) }

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
