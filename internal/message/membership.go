package message

import "time"

// Entry is how a subject entered the estate — the membership-only half of the
// Transition grammar, plus `revealed` which is aperture (CONTEXT.md
// `Transition`). It decides the entering membership message's cause and class.
type Entry string

const (
	// EntryAppeared is discovery — the world moved. Drift class.
	EntryAppeared Entry = "appeared"
	// EntryReturned is a decommission undone — the world moved. Drift class.
	EntryReturned Entry = "returned"
	// EntryRevealed is a widened aperture — we started looking, the world did not
	// move. Coverage class. A first run is one coverage-class membership message
	// with no special case.
	EntryRevealed Entry = "revealed"
)

// RootFires reports whether a subject of the given kind is a membership root
// that fires its own message. A membership message fires on a `Name` or an
// `Address` alone (CONTEXT.md `Subject`, ADR-0031): the other two kinds can
// bring no ground the model was not already accounting for. So a `Service` or an
// `Endpoint` entering the estate is never itself a message — it rides the census
// of the membership message fired at the root of its entering sub-tree.
func RootFires(subjectKind string) bool {
	switch subjectKind {
	case "name", "address":
		return true
	default:
		return false
	}
}

// Membership returns the one membership Message fired at the root of an entering
// sub-tree, carrying the census of everything that entered beneath it, or nil
// where the root is not a `Name` or an `Address`. It fires once, at the root,
// because every timeline beneath a new subject opens and no alerting predicate
// in the product is opening-shaped (ADR-0031) — so the entering Services and
// Endpoints beneath appear only in this message's census, never as their own
// messages.
//
// The cause and class follow the Entry: `appeared` and `returned` describe a
// subject the world moved (CauseDrift / ClassDrift); `revealed` is a widened
// aperture (CauseAperture / ClassCoverage), which is what makes a first run one
// coverage-class message with no special case.
func Membership(entry Entry, rootKind, rootKey string, census Census, instant time.Time) *Message {
	if !RootFires(rootKind) {
		return nil
	}
	cause := CauseDrift
	if entry == EntryRevealed {
		cause = CauseAperture
	}
	c := census
	return &Message{
		Cause:       cause,
		Class:       ClassForCause(cause),
		SubjectKind: rootKind,
		FiredAt:     rootKey,
		Instant:     instant,
		Census:      &c,
		Headline:    membershipHeadline(entry, rootKey, census),
	}
}
