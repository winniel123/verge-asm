// Package httpexchange is the `http-exchange` leaf: one bounded `GET /` per reached
// Service, folded to the `http-identity` facet of an `Endpoint` (spec §3.3, ADR-0011).
// A redirect is never followed, and that is a declared parameter, not a dial (ADR-0025).
package httpexchange

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const Version = "http-exchange/v2" // moves only with a moved golden row (ADR-0021)

const Kind = "http-exchange"

const FacetHTTPIdentity = "http-identity"

type Params struct {
	Method           string `json:"method"`
	Path             string `json:"path"`
	BodyCapBytes     int    `json:"body_cap_bytes"`
	TimeoutMillis    int    `json:"timeout_millis"`
	PerHostReqPerSec int    `json:"per_host_req_per_sec"`
	FollowRedirects  bool   `json:"follow_redirects"`
}

func DefaultParams() Params {
	// Authored here, never a library default: the Batch records this set by content (ADR-0025).
	return Params{
		Method:           "GET",
		Path:             "/",
		BodyCapBytes:     64 * 1024,
		TimeoutMillis:    10000,
		PerHostReqPerSec: 10,
		// Always false, never an operator dial: a 3xx is recorded, never followed (ADR-0025).
		FollowRedirects: false,
	}
}

func (p Params) Digest() string {
	b, err := json.Marshal(p)
	if err != nil {
		panic("httpexchange: marshal params: " + err.Error())
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}
