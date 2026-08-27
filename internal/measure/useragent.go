// Package measure holds contracts shared across the measurement leaves that live
// in its subpackages (connectoutcome, httpexchange, resolutionwalk, …). It sits
// ABOVE the individual leaves and imports none of them, so a leaf may depend on a
// shared contract here without risking an import cycle.
package measure

// ProbeUserAgent is the single, identifiable User-Agent every active HTTP probe
// this project issues MUST send. The v1 spec's safety posture (README §"Probes
// safely", spec §3.3) requires an identifiable client so a target's operator can
// recognise the traffic as verge-asm's active-discovery probe and contact the
// project; the value carries a contact URL for exactly that. It is fixed and
// repo-owned — an implementation identifier, never an operator dial — so a target
// sees one stable string across every probe.
//
// It is the shared UA contract for the round: the http-exchange leaf sends it on
// its single `GET /`, and any sibling probe that issues its own HTTP request
// (e.g. the P0.8 identity path) reuses THIS constant rather than minting its own,
// so "an identifiable User-Agent" stays one string across the whole prober.
const ProbeUserAgent = "verge-asm-prober (+https://github.com/winniel123/verge-asm)"
