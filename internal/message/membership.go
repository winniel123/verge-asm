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
//
// Where a message fires and where its row links are two different things for the
// aperture case. A `revealed` firing is *about* the Seed whose scope moved, so it
// fires at that Seed (SubjectKind "seed", FiredAt the seed scope key) and its row
// links there per §5.3 — never to the entering subject and never to Coverage's
// standing aperture statement. The entering Name/Address root still names the
// headline and the census still carries everything that entered beneath it;
// seedKey is read only for a `revealed` entry and ignored otherwise.
func Membership(entry Entry, rootKind, rootKey, seedKey string, census Census, instant time.Time) *Message {
	if !RootFires(rootKind) {
		return nil
	}
	cause := CauseDrift
	subjectKind, firedAt := rootKind, rootKey
	if entry == EntryRevealed {
		cause = CauseAperture
		// The aperture mover is the Seed, so the message fires at it: the row must
		// link to the exact Seed whose scope moved (ADR carried on Cause -> LinkSeed).
		subjectKind, firedAt = "seed", seedKey
	}
	c := census
	return &Message{
		Cause:       cause,
		Class:       ClassForCause(cause),
		SubjectKind: subjectKind,
		FiredAt:     firedAt,
		Instant:     instant,
		Census:      &c,
		Headline:    membershipHeadline(entry, rootKey, census),
	}
}
