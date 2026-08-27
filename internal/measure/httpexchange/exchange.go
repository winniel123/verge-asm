package httpexchange

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/measure"
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
// before the identity fold. A Failed result means the reached Service returned no
// valid HTTP response (a transport error, a timeout, or a non-HTTP protocol on the
// port): a determinate NEGATIVE, not an absence — it folds to the `no-http-response`
// value, never to nothing (ADR-0011, ADR-0015). A completed result carries the
// status line, the admitted identity headers, and the `Location` of a 3xx
// (recorded, never followed); the body is read only to lift the admitted `<title>`
// from it and is never itself stored.
type ExchangeResult struct {
	// Failed is true where the exchange did not complete: no valid HTTP response.
	Failed bool
	// Err is the transport error text, carried on a Failed result for the operator.
	Err string
	// Status is the HTTP status code of the single `GET /`.
	Status int
	// Server is the `Server` response header, empty where absent.
	Server string
	// WWWAuthenticate is the `WWW-Authenticate` response header (admitted on a 401),
	// empty where absent.
	WWWAuthenticate string
	// Location is the `Location` header of a 3xx — recorded as identity because
	// redirects are not followed, so the destination is a fact and not a next hop.
	Location string
	// Body is the response body, read only so the fold can lift the admitted
	// `<title>` from it. It is bounded at the cap on read and never stored: ADR-0011
	// refuses a body hash or content-length hardest, since their normalisation is an
	// unbounded corpus that would diff the Endpoint on every run.
	Body []byte
}

// Outcome is the closed union `http-identity` ranges over (ADR-0011, ADR-0015):
// a reached Service whose `GET /` returned an HTTP response (`responded`), or one
// reached over TCP that returned no valid HTTP response (`no-http-response`). The
// negative is a VALUE about the Endpoint, never a Gap — modelling it as an absence
// would make "the port does not speak HTTP" indistinguishable from "we did not
// look".
const (
	OutcomeResponded      = "responded"
	OutcomeNoHTTPResponse = "no-http-response"
)

// HTTPIdentity is the canonical `http-identity` value the leaf emits for an
// Endpoint. It is a CLOSED SET of admitted fields (ADR-0011): the outcome, the
// status of the single `GET /`, the `Server` header, the page `<title>`, the
// `WWW-Authenticate` challenge, and the recorded (not followed) redirect
// destination. It deliberately carries NO body hash, NO content-length, and NO
// `Content-Type`: the body hash's normalisation is an unbounded corpus that would
// diff the Endpoint on every run, and `Content-Type` is outside the closed
// admitted set. A `no-http-response` value carries only its outcome.
type HTTPIdentity struct {
	Outcome          string `json:"outcome"`
	Status           int    `json:"status,omitempty"`
	Server           string `json:"server,omitempty"`
	Title            string `json:"title,omitempty"`
	WWWAuthenticate  string `json:"www_authenticate,omitempty"`
	RedirectLocation string `json:"redirect_location,omitempty"`
}

// titleCap bounds the recorded `<title>`: a stable identifying snippet, never the
// document. A longer title is truncated to this many bytes.
const titleCap = 256

// Identity folds a raw exchange result to the http-identity value. It is the pure
// heart of the leaf and the thing the golden corpus pins. A Failed exchange folds
// to the `no-http-response` VALUE — the reached Service returned no valid HTTP
// response, a determinate negative the caller emits and whose Endpoint it creates,
// never an absence. A completed exchange records its outcome, status, the admitted
// headers, and the `<title>` lifted from the capped body; the body itself is never
// stored.
func Identity(r ExchangeResult, bodyCap int) HTTPIdentity {
	if r.Failed {
		return HTTPIdentity{Outcome: OutcomeNoHTTPResponse}
	}
	if bodyCap < 0 {
		bodyCap = 0
	}
	body := r.Body
	if len(body) > bodyCap {
		body = body[:bodyCap]
	}
	return HTTPIdentity{
		Outcome:          OutcomeResponded,
		Status:           r.Status,
		Server:           r.Server,
		Title:            extractTitle(body),
		WWWAuthenticate:  r.WWWAuthenticate,
		RedirectLocation: r.Location,
	}
}

// extractTitle lifts the text of the first `<title>...</title>` from the capped
// body, case-insensitively, collapsing inner whitespace and truncating to
// titleCap bytes. It is a small deterministic parse — never a full HTML parse —
// so the golden corpus can pin it exactly; a body with no title yields "".
func extractTitle(body []byte) string {
	s := string(body)
	lower := strings.ToLower(s)
	open := strings.Index(lower, "<title")
	if open < 0 {
		return ""
	}
	gt := strings.IndexByte(s[open:], '>')
	if gt < 0 {
		return ""
	}
	start := open + gt + 1
	end := strings.Index(lower[start:], "</title>")
	if end < 0 {
		return ""
	}
	title := strings.Join(strings.Fields(s[start:start+end]), " ")
	if len(title) > titleCap {
		title = title[:titleCap]
	}
	return title
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
	// Send the shared, identifiable probe User-Agent so a target's operator can
	// recognise the single GET / as verge-asm's active-discovery probe (README
	// §"Probes safely", spec §3.3). The value is the one repo-owned contract in
	// internal/measure, reused by every probe rather than minted per leaf.
	req.Header.Set("User-Agent", measure.ProbeUserAgent)
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
		Status:          resp.StatusCode,
		Server:          resp.Header.Get("Server"),
		WWWAuthenticate: resp.Header.Get("WWW-Authenticate"),
		Location:        resp.Header.Get("Location"),
		Body:            body,
	}
}
