package screen

import "regexp"

const (
	SignalCitation     = "citation"
	SignalExternalSpec = "external-spec"
	SignalWhyMarker    = "why-marker"
	SignalBareURL      = "bare-url"
)

var (
	citationRe     = regexp.MustCompile(`(?i)\bADR-\d{4}\b|#\d+`)
	externalSpecRe = regexp.MustCompile(`\bRFC\b|\bRFC\d+|\bIANA\b|\bX\.509\b|\bBCP\b|\bNIST\b|\bPKIX\b|\bISO\s+\d{4}\b`)
	urlRe          = regexp.MustCompile(`\bhttps?://\S`)
	whyRe          = regexp.MustCompile(`(?i)\b(because|otherwise|so that|avoid|avoids|work ?around|race|races|panic|panics|deliberate|deliberately|intentional|intentionally|on purpose|hazard|gotcha|beware|deadlock|must not|cannot|prevent|prevents|load-bearing|unsafe|breaks|corrupts)\b`)
	historyRe      = regexp.MustCompile(`(?i)\b(no longer|previously|renamed|superseded|supersedes|deprecated|used to|formerly|as of \d{4}|since \d{4}|replaced by)\b`)
	looseRe        = regexp.MustCompile(`(?i)\b(now|was)\b`)
)

func Signal(payload string) string {
	// The §3.2 table's order, so one block yields one stable manifest reason.
	switch {
	case HasCitation(payload):
		return SignalCitation
	case HasExternalSpec(payload):
		return SignalExternalSpec
	case HasWhyMarker(payload):
		return SignalWhyMarker
	case urlRe.MatchString(payload):
		return SignalBareURL
	}
	return ""
}

func HasCitation(payload string) bool {
	return citationRe.MatchString(payload)
}

func HasExternalSpec(payload string) bool {
	return externalSpecRe.MatchString(payload)
}

func HasWhyMarker(payload string) bool {
	// The §3.2 word list is a floor. A revision may widen it. Narrowing it
	// loses a reason permanently.
	return whyRe.MatchString(payload)
}

func HasHistoryMarker(payload string) bool {
	// A history hit is a delete signal, never a screen signal (SPEC §3.2).
	return historyRe.MatchString(payload)
}

func HasLooseNarration(payload string) bool {
	// The survey's named false-positive trap, which no rule may flag (§3.5).
	return looseRe.MatchString(payload)
}
