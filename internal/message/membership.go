package message

import "time"

// Revealed is aperture, not membership, and rides here as an entry (CONTEXT.md Transition).

type Entry string

const (
	EntryAppeared Entry = "appeared"
	EntryReturned Entry = "returned"
	EntryRevealed Entry = "revealed"
)

func RootFires(subjectKind string) bool {
	// A Service or Endpoint brings no ground the model was not already accounting for (ADR-0031).
	switch subjectKind {
	case "name", "address":
		return true
	default:
		return false
	}
}

func Membership(entry Entry, rootKind, rootKey, seedKey string, census Census, instant time.Time) *Message {
	if !RootFires(rootKind) {
		return nil
	}
	cause := CauseDrift
	subjectKind, firedAt := rootKind, rootKey
	if entry == EntryRevealed {
		// A widened aperture makes a first run one coverage-class message, with no special case.
		cause = CauseAperture
		// A revealed firing is about the Seed whose scope moved, not the entering subject (§5.3).
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
