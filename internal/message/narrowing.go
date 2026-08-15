package message

import "time"

// NarrowingReceipt is the preview a Seed-narrowing act shows before the operator
// commits, and the payload of the coverage-class message that fires at the scope
// (ADR-0074). It is the honestly-computable form of ticket 4's deferred
// narrowing receipt: it counts the subjects the narrowing withdraws and the
// timelines they take out of the estate, and it names the loss — a listener
// appearing in the removed ground after the act is invisible, and no later
// message recovers it.
//
// It fires ONLY where the withdrawn set is inhabited. Where nothing is withdrawn
// (an empty residue — declining a Proposal that was never a subject), or where a
// current resolution still cites the ground (the subject survives and its `Gap`
// carries it), Fires is false and no preview is owed: a preview for a message
// that will not fire is a promise the widening side never has to make good on.
type NarrowingReceipt struct {
	// Scope is the Seed scope the message fires at — the only object that
	// survives the act, and the firing site the widening message already uses.
	Scope string
	// Removed is the excluded value (an address scope, a name, or a subtree) that
	// narrows the scope.
	Removed string
	// SubjectsWithdrawn is the count of subjects the act removes — the trigger
	// (inhabitance of the withdrawn set) and half the payload.
	SubjectsWithdrawn int
	// TimelinesRemoved is the count of timelines those subjects take out of the
	// estate — the other half of the count, stated with its factors rather than
	// as a bare product.
	TimelinesRemoved int
	// Fires is true exactly where SubjectsWithdrawn > 0. It gates whether the
	// preview and the store message appear at all.
	Fires bool
	// Headline and Loss are the rendered copy, computed only where Fires. They
	// carry no valence word: a narrowing is neither good news nor bad.
	Headline string
	Loss     string
}

// PreviewNarrowing computes the receipt from the counts the caller measured — the
// subjects the narrowing would withdraw (ground nothing else cites) and the
// timelines they hold. The caller measures those against the current estate; this
// function decides whether the message fires and renders the copy in ADR-0064's
// grammar, so the act-time preview and the store message read as one sentence.
func PreviewNarrowing(scope, removed string, subjectsWithdrawn, timelinesRemoved int) NarrowingReceipt {
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

// Narrowing returns the coverage-class Message a firing receipt produces, or nil
// where the receipt does not fire. It is computed once, at the act, and stored:
// the message names a scope and a count, carries no comparison and no rows, and
// links to the Seed whose scope moved (the aperture mover). A dangling scope key
// — a scope later narrowed out of existence — is a dated fact, not a broken join,
// because a Message is written once and never recomputed.
func Narrowing(r NarrowingReceipt, seedKey string, instant time.Time) *Message {
	if !r.Fires {
		return nil
	}
	// A narrowing carries a count, not a census of rows (ADR-0074) — Census stays
	// nil.
	return &Message{
		Cause:       CauseAperture,
		Class:       ClassCoverage,
		SubjectKind: "seed",
		FiredAt:     seedKey,
		Instant:     instant,
		Headline:    r.Headline,
	}
}
