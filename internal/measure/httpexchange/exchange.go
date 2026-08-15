package httpexchange

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

// Target is one `(Name, Service)` pair the leaf exchanges with: the Address and
// port of the Service, the scheme to speak, and the Name that resolved to the
// Address. Name may be empty — the nameless endpoint, a distinguished key variant
// (never an empty name): a Service reached with no citing Name still presents an
// HTTP identity, held under the nameless Endpoint key.
type Target struct {
	// Name is the DNS Name whose resolution cited the Address, or "" for the
	// nameless endpoint.
	Name string `json:"name"`
	// Address is the Service's Address (the `A`/`AAAA` the Name resolved to, or a
	// Seed-covered address).
	Address string `json:"address"`
	// Port is the Service's TCP port.
	Port uint16 `json:"port"`
	// Scheme is the URL scheme spoken — "http" or "https". It records how the
	// exchange was framed and never widens what was probed.
	Scheme string `json:"scheme"`
}

// ServiceKey renders the Target's `Service` subject key — `address:port/tcp` —
// the same triple rendering the connect-outcome leaf produces, so an Endpoint's
// Service leg names exactly the Service the reachability timeline is about. An
// address that parses is rendered from its netip form (bracketing IPv6); one that
// does not falls back to the raw string joined by port, so a malformed target
// still renders a legible key rather than panicking.
func (t Target) ServiceKey() string {
	if addr, err := netip.ParseAddr(t.Address); err == nil {
		return netip.AddrPortFrom(addr.Unmap(), t.Port).String() + "/tcp"
	}
	return t.Address + ":" + strconv.Itoa(int(t.Port)) + "/tcp"
}

// EndpointKey renders the `Endpoint` subject key for a `(Name, Service)` pair:
// `name@service` where Name is present, and `@service` for the nameless endpoint.
// The nameless form is a distinguished key variant — a leading `@` with no name
// segment — never an empty name masquerading as a named one. Neither a DNS Name
// nor a Service key contains `@`, so the key splits unambiguously at its first
// `@` on read.
func EndpointKey(name, serviceKey string) string {
	return name + "@" + serviceKey
}

// EndpointKey renders the Endpoint key for this Target.
func (t Target) EndpointKey() string { return EndpointKey(t.Name, t.ServiceKey()) }

// ExchangeResult is the raw outcome of one HTTP exchange the Exchanger reports,
// before the identity fold. A Failed result means no HTTP response was obtained
// (a transport error or a timeout): no exchange happened, so no Endpoint is
// created and no observation is emitted. A completed result carries the status
// line, the identity headers, the `Location` of a 3xx (recorded, never followed),
// and the — already 64 KB-bounded — body.
type ExchangeResult struct {
	// Failed is true where the exchange did not complete: no response was read.
	Failed bool
	// Err is the transport error text, carried on a Failed result for the operator.
	Err string
	// Status is the HTTP status code of the single `GET /`.
	Status int
	// Server is the `Server` response header, empty where absent.
	Server string
	// ContentType is the `Content-Type` response header, empty where absent.
	ContentType string
	// Location is the `Location` header of a 3xx — recorded as identity because
	// redirects are not followed, so the destination is a fact and not a next hop.
	Location string
	// Body is the response body. The production Exchanger bounds it at the cap on
	// read; the identity fold applies the cap again so the value is deterministic
	// regardless of the exchanger.
	Body []byte
}

// HTTPIdentity is the canonical `http-identity` value the leaf emits for an
// Endpoint: the status of the single `GET /`, the identifying response headers,
// the recorded (not followed) redirect destination, and a content digest over the
// capped body. The differ compares these fields structurally; the body itself is
// never stored — only its digest and length, so the value is small and stable.
type HTTPIdentity struct {
	Status           int    `json:"status"`
	Server           string `json:"server,omitempty"`
	ContentType      string `json:"content_type,omitempty"`
	RedirectLocation string `json:"redirect_location,omitempty"`
	BodySHA256       string `json:"body_sha256"`
	BodyBytes        int    `json:"body_bytes"`
	BodyTruncated    bool   `json:"body_truncated"`
}

// Identity folds a raw exchange result to the http-identity value. It is the pure
// heart of the leaf and the thing the golden corpus pins. A Failed exchange folds
// to no identity — the second return is false, and the caller emits nothing and
// creates no Endpoint. A completed exchange records its status and headers, caps
// the body at cap bytes (marking whether it was truncated), and digests the capped
// body: the identity is the same whether the body arrived capped or whole.
func Identity(r ExchangeResult, bodyCap int) (HTTPIdentity, bool) {
	if r.Failed {
		return HTTPIdentity{}, false
	}
	if bodyCap < 0 {
		bodyCap = 0
	}
	body := r.Body
	truncated := false
	if len(body) > bodyCap {
		body = body[:bodyCap]
		truncated = true
	}
	sum := sha256.Sum256(body)
	return HTTPIdentity{
		Status:           r.Status,
		Server:           r.Server,
		ContentType:      r.ContentType,
		RedirectLocation: r.Location,
		BodySHA256:       "sha256:" + hex.EncodeToString(sum[:]),
		BodyBytes:        len(body),
		BodyTruncated:    truncated,
	}, true
}

// Exchanger performs one `GET /` against a target and reports its raw result. The
// production adapter (NetExchanger) speaks real HTTP with the profile's timeout
// and redirects disabled; the golden corpus scripts an in-process Exchanger, so
// the identity fold runs hermetically with no network and no container. An
// Exchanger issues exactly one request per target — never a redirect's next hop —
// which is what makes "redirects not followed" a structural property rather than a
// discipline.
type Exchanger interface {
	Exchange(ctx context.Context, target Target) ExchangeResult
}

// NetExchanger is the production Exchanger: a single `GET /` with a bounded
// timeout, a body read capped at the profile's byte ceiling, and redirect
// following disabled so a 3xx returns its own response with the `Location` intact.
// It is not exercised by the hermetic golden corpus.
type NetExchanger struct {
	// Params carries the timeout and the body cap. Zero values fall back to the
	// shipped defaults, so a caller may hand a partial profile.
	Params Params
}

// Exchange implements Exchanger against the network. It builds an http.Client
// whose CheckRedirect refuses to follow (returning the last response), so a 3xx is
// read as-is. The body is read through an io.LimitReader at the cap plus one byte,
// so the identity fold can tell a body that exactly filled the cap from one that
// overran it. Any transport error — connect, TLS, or read — folds to a Failed
// result: our own blindness, never an identity value.
func (n NetExchanger) Exchange(ctx context.Context, target Target) ExchangeResult {
	p := n.Params
	if p.TimeoutMillis <= 0 {
		p.TimeoutMillis = DefaultParams().TimeoutMillis
	}
	if p.BodyCapBytes <= 0 {
		p.BodyCapBytes = DefaultParams().BodyCapBytes
	}
	scheme := target.Scheme
	if scheme == "" {
		scheme = "http"
	}
	url := scheme + "://" + net.JoinHostPort(target.Address, strconv.Itoa(int(target.Port))) + "/"

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMillis)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return ExchangeResult{Failed: true, Err: err.Error()}
	}
	client := &http.Client{
		// Redirects are not followed: return the 3xx response unfollowed so its
		// Location is recorded as identity and no next hop is ever requested.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return ExchangeResult{Failed: true, Err: err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(p.BodyCapBytes)+1))
	return ExchangeResult{
		Status:      resp.StatusCode,
		Server:      resp.Header.Get("Server"),
		ContentType: resp.Header.Get("Content-Type"),
		Location:    resp.Header.Get("Location"),
		Body:        body,
	}
}
