package custody

// Version is the `Custody` derivation's version — the one name ADR-0008's
// version vector gives this derivation, and the name the golden corpus's lock
// binds its declared parameters to (golden-corpus.md §10).
//
// It moves on OUTPUT-AFFECTING CONTENT and never because a release shipped
// (ADR-0008). On this derivation that is the fan-out threshold, the reduction
// that feeds it, the Public Suffix List the reduction reads, the absence rule,
// and the two coverage limbs themselves. ADR-0129 §3 makes each of them a
// `Break` rather than drift.
//
// An operator act moves it NOT AT ALL. Declaring an address scope satisfies the
// reach; it does not change what the derivation computes. No install can move a
// `Custody` version without a release, because every input above is a `const` or
// a dependency (ADR-0129's #955 amendment).
//
// Nothing composes it into a `drift` component vector today. The measure leaves
// sit in that vector because a `Span` reads their outcomes. Whether `Custody`
// joins it is a separate decision with an estate-wide `Break` on the other side
// of it, and the corpus does not take it. Here the constant does ONE job: it
// names the derivation the lock binds, so a threshold move with no bump fails
// the A6 gate.
// v2 (#1018): the errored floor became PER LIMB. A `Scan` that completes a Batch
// and measures no EXTENSION CANDIDATE now reaches those candidates, where a
// declaration-limb row alone used to lift a whole-store floor and leave every
// candidate held. That is the absence rule moving, which this comment's list makes
// a `Break` rather than drift.
const Version = "custody/v2"
