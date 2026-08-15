// Package httpexchange is the `http-exchange` Derivation leaf inside the shared
// measurement binary (v1 spec §3.3). It rides the reachability exchange: after
// the `connect-outcome` leaf's TCP connect to a `Service` reports `reached`, the
// HTTP exchange is the STEP that runs next over that same connection's target,
// deciding the `http-identity` facet for the `Endpoint` — the `(Name, Service)`
// pair, the only key under which HTTP identity is single-valued (CONTEXT.md
// `Endpoint`). The leaf is composed additively into that flow as its own function
// (RunWithExchanger); the orchestrator reconciles it with the TCP step and the
// concurrent TLS step so each leaf stays localized to its own file.
//
// The exchange is deliberately minimal and bounded (§3.3): a single `GET /`, a
// 64 KB-capped body read, a 10 s timeout, ≤ 10 req/s per host, and **redirects
// are not followed**. Not-following-redirects is a DECLARED PARAMETER of the leaf
// recorded on the Batch by content (ADR-0025), not an operator-configurable dial:
// a 3xx response has its `Location` recorded as identity and no second request is
// ever issued. Every cap lives in Params, so a change to any of them moves the
// leaf's params digest and forces a Version bump through the golden-corpus lock.
//
// The identity fold is pure and the exchange is behind an interface (Exchanger),
// so the golden corpus drives the leaf against a scripted in-process exchanger —
// no network, no container, arch-neutral.
package httpexchange

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Version is the leaf's Derivation version (ADR-0008/ADR-0021). It moves on an
// output-affecting change and only on one, gated bidirectionally by this leaf's
// own golden corpus (§4.4), separately from the sibling leaves so a break names
// its leaf.
const Version = "http-exchange/v1"

// Kind is the JobSpec.Kind that dispatches to this leaf. It is distinct from the
// `hot` Scan's DB kind and from the `connect-outcome` leaf it rides: the Scan is
// `hot`, the reachability step is `connect-outcome`, and the HTTP step is
// `http-exchange`.
const Kind = "http-exchange"

// FacetHTTPIdentity is the facet this leaf decides — HTTP identity, held on an
// `Endpoint` subject.
const FacetHTTPIdentity = "http-identity"

// Params is the leaf's declared-parameter set: the exact §3.3 HTTP safety table,
// recorded on the Batch by content (ADR-0025) so what governed the exchange is
// legible and never a library default. None of it is operator-configurable — in
// particular FollowRedirects is recorded here, never surfaced as a dial.
type Params struct {
	// Method is fixed: GET. The exchange asks for identity, never mutates.
	Method string `json:"method"`
	// Path is fixed: `/`. A single request to the root, never a crawl.
	Path string `json:"path"`
	// BodyCapBytes bounds the body read — 65536 (64 KB). A response longer than
	// this is truncated to the cap and the identity records that it was.
	BodyCapBytes int `json:"body_cap_bytes"`
	// TimeoutMillis bounds the whole exchange — 10000 (10 s).
	TimeoutMillis int `json:"timeout_millis"`
	// PerHostReqPerSec is the per-host request rate ceiling — 10 req/s.
	PerHostReqPerSec int `json:"per_host_req_per_sec"`
	// FollowRedirects records that redirects are NOT followed. It is always false
	// — the invariant, recorded by content — so a 3xx records its Location as
	// identity and no second request is issued.
	FollowRedirects bool `json:"follow_redirects"`
}

// DefaultParams is the v1 shipped HTTP safety table — the §3.3 values exactly.
// Authored here, not defaulted by any library, so the job spec carries it to the
// leaf and the Batch records exactly what governed the exchange.
func DefaultParams() Params {
	return Params{
		Method:           "GET",
		Path:             "/",
		BodyCapBytes:     64 * 1024,
		TimeoutMillis:    10000,
		PerHostReqPerSec: 10,
		FollowRedirects:  false,
	}
}

// Digest is a stable content hash of the params, used by the golden-corpus lock
// to bind a declared-parameter change to a Version bump.
func (p Params) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("httpexchange: marshal params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
