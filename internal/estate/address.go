package estate

import "github.com/winniel123/verge-asm/internal/drift"

// This file is the Address half of membership. Alone among the four subjects an
// Address has no lifecycle of its own — nothing ever observes an address's
// *existence* — so it is in the estate exactly while a current resolution cites
// it OR a Seed's address scope covers it (CONTEXT.md `Address`, ADR-0047). The
// two limbs are disjunctive, and the second is not redundant with the first: an
// address inside a declared address scope is a subject *from the declaration*,
// before anything has resolved to it, with its Citation hopping straight to the
// Seed. Its Services then hold `not-reached` — a measured value — until
// something answers there.
//
// drift decides nothing about membership; it only applies a departure once
// membership concludes it (drift.CloseWithdrawal). estate is that decision, so
// the ground an Address's closure records is decided here and handed to drift,
// exactly as WithdrawnCrossClass hands it the Name departure (ADR-0087). estate
// may import drift because drift imports nothing of ours — the dependency runs
// one way.

// AddressPresent reports whether an Address is in the estate. The rule is the
// disjunction of its two membership limbs: a current resolution citing it, or a
// Seed's address scope covering it. Because the limbs are disjunctive, an
// Address that a resolution stopped citing is still present while a Seed covers
// it, and one no Seed covers is still present while a resolution cites it.
func AddressPresent(cited, seedCovered bool) bool {
	return cited || seedCovered
}

// AddressClosure decides the ground an Address's departure rests on, or reports
// that the Address stays. It composes the disjunctive membership rule with the
// closed union of three closure grounds (ADR-0087):
//
//   - present (cited OR seedCovered): the Address stays; left is false and the
//     reason is empty.
//   - a Seed stopped covering it (seedDescoped) and nothing cites it:
//     `descoped` — our aperture stopped covering the address by an exclusion or a
//     narrower scope. That ground alone blocks a later `returned`, because a
//     narrowing is not a decommission and a re-citation must not read as the
//     world bringing the address back.
//   - otherwise it left because a resolution stopped citing it: `uncited` — the
//     departure is grounded in evidence about *another* subject, the Name whose
//     answer changed, never in an observation about the address itself, which
//     has no existence to measure. This is the ground the `reachability` facet's
//     Services close under when their Address falls out of every current
//     resolution.
//
// descoped takes precedence where a resolution retirement and a Seed narrowing
// coincide: the operator's narrowing is the mover, and recording the ground that
// suppresses a spurious `returned` is the safe reading.
func AddressClosure(cited, seedCovered, seedDescoped bool) (reason drift.ClosureReason, left bool) {
	if AddressPresent(cited, seedCovered) {
		return "", false
	}
	if seedDescoped {
		return drift.ReasonDescoped, true
	}
	return drift.ReasonUncited, true
}
