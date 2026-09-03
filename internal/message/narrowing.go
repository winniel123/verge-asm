package message

import "time"

// A preview for a message that will not fire is a promise nothing has to make good on (ADR-0074).

type NarrowingReceipt struct {
	Scope             string
	Removed           string
	SubjectsWithdrawn int
	TimelinesRemoved  int
	Fires             bool
	Headline          string
	Loss              string
}

func PreviewNarrowing(scope, removed string, subjectsWithdrawn, timelinesRemoved int) NarrowingReceipt {
	// A count is stated with its factors and never as a bare product (ADR-0074).
	r := NarrowingReceipt{
		Scope:             scope,
		Removed:           removed,
		SubjectsWithdrawn: subjectsWithdrawn,
		TimelinesRemoved:  timelinesRemoved,
		Fires:             subjectsWithdrawn > 0,
	}
	if r.Fires {
		r.Headline = narrowingHeadline(scope, removed, subjectsWithdrawn, timelinesRemoved)
		r.Loss = narrowingLoss(removed)
	}
	return r
}

// Removal is the limiting case of narrowing, so it rides the same receipt and class (ADR-0134 §1).

func PreviewSeedWithdrawal(scope string, subjectsWithdrawn, timelinesRemoved int) NarrowingReceipt {
	// A name Seed withdrawal states the act once, never a row per Name (ADR-0135 §1).
	r := NarrowingReceipt{
		Scope:             scope,
		Removed:           scope,
		SubjectsWithdrawn: subjectsWithdrawn,
		TimelinesRemoved:  timelinesRemoved,
		Fires:             subjectsWithdrawn > 0,
	}
	if r.Fires {
		r.Headline = seedWithdrawalHeadline(scope, subjectsWithdrawn, timelinesRemoved)
		r.Loss = narrowingLoss(scope)
	}
	return r
}

func Narrowing(r NarrowingReceipt, seedKey string, instant time.Time) *Message {
	// A scope later narrowed out of existence is a dated fact here, never a broken join (ADR-0074).
	if !r.Fires {
		return nil
	}
	// A narrowing carries a count and not a census of rows, so Census stays nil (ADR-0074).
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassCoverage,
		SubjectKind: "seed",
		FiredAt:     seedKey,
		Instant:     instant,
		Headline:    r.Headline,
	}
}
