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
	// Scope is the Seed scope the message fires at — the firing site the widening
	// message already uses. It survives an exclusion, which narrows it from
	// inside. It does NOT survive a Seed withdrawal (ADR-0134): there the scope
	// is the thing that went, and the key is a dated fact rather than a live join
	// — which Narrowing's doc already licenses.
	Scope             string
	Removed           string
	SubjectsWithdrawn int
	// TimelinesRemoved is the count of timelines those subjects take out of the
	// estate — the other half of the count, stated with its factors rather than
	// as a bare product.
	TimelinesRemoved int
	Fires            bool
	Headline         string
	Loss             string
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

// PreviewSeedWithdrawal computes the receipt for the OTHER narrowing act: a Seed
// the operator withdrew entirely (ADR-0134 §1, §6). Removal is the limiting case
// of narrowing, so it carries the same receipt, the same coverage class and the
// same `descoped` ground; only the rendered sentence differs.
//
// BOTH Seed limbs come here (ADR-0135 §1). A name Seed withdrawal removes many
// Names in one act, so it states the act once rather than writing a row per Name —
// which would be the census the receipt exists to replace (ADR-0074).
//
// Scope and Removed are both the withdrawn scope, because a Seed's display scope
// IS its scope column: a CIDR for the address limb and a domain for the name limb.
// The scope that moved and the ground that left are one object here. That is
// exactly why the copy cannot go through PreviewNarrowing, whose headline names
// the two separately.
func PreviewSeedWithdrawal(scope string, subjectsWithdrawn, timelinesRemoved int) NarrowingReceipt {
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
