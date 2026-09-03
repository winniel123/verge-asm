// Package measure holds contracts shared across the measurement leaves in its
// subpackages (connectoutcome, httpexchange, resolutionwalk, …). It imports none
// of them, so a leaf may share a contract here with no import cycle.
package measure

// The safety posture requires an identifiable client the operator can recognise (spec §3.3).

const ProbeUserAgent = "verge-asm-prober (+https://github.com/winniel123/verge-asm)"
