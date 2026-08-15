// Package delivery is the outbound half of a Notification Channel (v1 spec §4.5,
// notification-channels.md, ADR-0039): it turns a stored Message into a signed
// `https` POST to each Channel that carries the Message's class, and records the
// per-Channel outcome as a Delivery.
//
// The shape is fixed by the spec and by the surrounding model:
//
//   - The body is built from the FROZEN Message — the same value the in-app panel
//     renders — and is byte-identical in content to it: the headline verbatim, the
//     census as a COUNT, never recomputed. It reaches no other table, so no row
//     sits behind a census count (§3.1's "no rows, in any field").
//   - Authentication is an HMAC-SHA256 signature over the body and a timestamp
//     WHEN a secret is set, and nothing otherwise — the URL is the credential.
//     There is NO bearer header, ever (§3.2): a bearer sits in the receiver's
//     access log and a signature does not, and nothing authenticates them to us.
//   - Routing is by class alone (ADR-0091): Routes is the whole predicate, and it
//     reads three booleans and the firing's class and nothing finer. There is no
//     per-rule and no per-subject axis.
//   - Retry runs on the queue's own retry/backoff/dead-letter machinery (#188):
//     five attempts over roughly an hour, then dead-lettered. A dead-lettered
//     Delivery marks the Delivery undelivered and leaves the Message untouched —
//     it licenses no silence and is never itself a Message.
//
// This file is the pure core — body, signing, request shape, response verdict and
// routing — with no database and no live network, so it is exercised in full by a
// fake HTTP doer. runner.go is the thin worker-side loop that drives it over the
// queue.
package delivery

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/message"
)

// Header names carried on every delivery POST. The signature header is present
// only when a secret is configured; the timestamp is always present because it is
// part of the signed input the receiver reconstructs.
const (
	// HeaderTimestamp carries the Unix-seconds instant the signature was computed
	// over — the instant of the ATTEMPT, distinct from the instant of the cause in
	// the body. Always present.
	HeaderTimestamp = "X-Verge-Timestamp"
	// HeaderSignature carries `sha256=<hex>`, the HMAC over timestamp and body.
	// Present only when the Channel has a secret.
	HeaderSignature = "X-Verge-Signature"
	// contentType is the one media type: the body is always a single JSON document.
	contentType = "application/json"
	// sigScheme prefixes the signature value, so a receiver reads the algorithm off
	// the header rather than assuming it.
	sigScheme = "sha256="
)

// Firing is the frozen Message a delivery body is built from — the arch-neutral
// input the runner maps a stored row onto. It carries exactly what the in-app
// Message carries: the identity, the class and cause, the fired-at subject KEY
// (never a rendering), the instant of the cause, the census bytes (read for their
// COUNT only), and the headline (carried verbatim). It reaches nothing else.
type Firing struct {
	ID          int64
	Cause       message.Cause
	Class       message.Class
	SubjectKind string
	FiredAt     string
	Instant     time.Time
	// Census is the stored census JSONB. Nil where the firing carries none; where
	// present, only its LENGTH crosses the wire, never its entries.
	Census   []byte
	Headline string
}

// Body is the JSON document posted to a Channel — one per Message, identical
// across every Channel and every retry. It carries exactly what the in-app
// Message carries and no rows (notification-channels.md §3.1): the census is a
// count, and no field enumerates the services, addresses or evidence behind it.
type Body struct {
	// Message is the stable, unique identifier — unchanged across retries, the
	// receiver's de-duplication key.
	Message int64 `json:"message"`
	// Class is the routing class the receiver may filter on: drift/coverage/clock.
	Class string `json:"class"`
	// Cause is which of the four causes fired — a field the operator reads; the
	// router never keys on it (ADR-0091).
	Cause string `json:"cause"`
	// Subject is the KEY of the thing the message fired at, never a rendering of it.
	Subject Subject `json:"subject"`
	// Instant is the instant of the CAUSE, read from the frozen Message — not the
	// instant of the delivery attempt.
	Instant time.Time `json:"instant"`
	// Headline is the rendered sentence, byte-identical to the in-app Message's own
	// headline: one computation at the cause, two renderings.
	Headline string `json:"headline"`
	// Census is the count where the firing carries one, and nil (omitted) otherwise.
	// It is a count and never a list — no rows sit behind it.
	Census *int `json:"census,omitempty"`
	// Link is an absolute URL into this instance, at the object or scope the message
	// fired at (§3.3). Empty where no base URL is configured.
	Link string `json:"link,omitempty"`
}

// Subject is the fired-at subject as a bare (kind, key) pair — the key the
// receiver would follow the link on, not a rendered label.
type Subject struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

// BuildBody renders the delivery body from a frozen firing and the instance's
// base URL. It reads only the firing — never the estate — so no row can appear
// behind a census count. The headline is carried verbatim, and the census (where
// the firing has one) contributes only its length.
func BuildBody(f Firing, baseURL string) Body {
	b := Body{
		Message:  f.ID,
		Class:    string(f.Class),
		Cause:    string(f.Cause),
		Subject:  Subject{Kind: f.SubjectKind, Key: f.FiredAt},
		Instant:  f.Instant.UTC(),
		Headline: f.Headline,
		Link:     link(baseURL, f.Cause, f.SubjectKind, f.FiredAt),
	}
	if len(f.Census) > 0 {
		// A present census contributes its COUNT alone. Its entries never cross the
		// wire — that is the "no services behind a census count" rule.
		if c, err := message.ParseCensus(f.Census); err == nil {
			n := c.Len()
			b.Census = &n
		}
	}
	return b
}

// MarshalBody renders the body to the exact bytes that are posted AND signed, so
// the signature covers precisely what the receiver reads. It is compact and
// deterministic — encoding/json emits struct fields in declaration order — so the
// dedup key and the signed bytes are stable across retries.
func MarshalBody(b Body) ([]byte, error) { return json.Marshal(b) }

// Routes is the WHOLE routing predicate (ADR-0091): a Channel receives a firing
// exactly when it carries that firing's class. It reads three booleans and the
// class and nothing finer — there is no per-rule and no per-subject axis for it
// to consult. An unknown class routes nowhere.
func Routes(routeDrift, routeCoverage, routeClock bool, class message.Class) bool {
	switch class {
	case message.ClassDrift:
		return routeDrift
	case message.ClassCoverage:
		return routeCoverage
	case message.ClassClock:
		return routeClock
	default:
		return false
	}
}

// Sign computes the HMAC-SHA256 over the timestamp and the body — the signature
// value (without its scheme prefix) a Channel with a secret carries. The signed
// input is `<unix-seconds>.<body>`, so the receiver reconstructs it from the
// timestamp header and the raw body it received.
func Sign(secret, body []byte, ts time.Time) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// NewRequest builds the outbound POST for one delivery. It always carries the
// timestamp header and the JSON content type; it carries the signature header
// ONLY when secret is non-empty (the URL alone is the credential otherwise); and
// it carries NO Authorization/bearer header under any condition. The body bytes
// are posted verbatim and are exactly the bytes Sign covered.
func NewRequest(ctx context.Context, targetURL string, body, secret []byte, ts time.Time) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(ts.Unix(), 10))
	if len(secret) > 0 {
		req.Header.Set(HeaderSignature, sigScheme+Sign(secret, body, ts))
	}
	// No Authorization header is ever set. The signature authenticates us to them;
	// nothing authenticates them to us, so there is nothing for a bearer to carry.
	return req, nil
}

// Verdict is what to do with a delivery after one attempt — the terminal fork the
// runner records. It keeps the retry-vs-dead-letter decision a pure function of
// the outcome and the attempt budget, so "five attempts then dead-lettered" is
// exercised without a database.
type Verdict int

const (
	// VerdictDelivered: a 2xx. Nothing more is sent.
	VerdictDelivered Verdict = iota
	// VerdictRetry: a failure with attempts remaining. The delivery returns to
	// pending on the shared backoff.
	VerdictRetry
	// VerdictUndelivered: a failure with the attempt budget spent. Dead-lettered —
	// the undelivered mark — leaving the Message untouched.
	VerdictUndelivered
)

// Decide is the terminal fork for one attempt: a delivered POST is done; a failed
// one retries while this attempt is below the budget and dead-letters once it
// reaches it. With the default budget of five, attempts 1–4 retry and the fifth
// dead-letters, so a receiver has five tries over the shared backoff before the
// delivery is marked undelivered.
func Decide(delivered bool, attempt, maxAttempts int32) Verdict {
	if delivered {
		return VerdictDelivered
	}
	if attempt < maxAttempts {
		return VerdictRetry
	}
	return VerdictUndelivered
}

// Delivered reports whether a status code counts as a delivery. Only a 2xx does:
// a 3xx is a redirect we never follow (it would move our attack surface to a host
// the operator never declared), and every 4xx/5xx is a failure — §4's table.
func Delivered(statusCode int) bool { return statusCode >= 200 && statusCode < 300 }

// link builds an absolute URL into this instance at the fired-at object or scope,
// per the per-mover rule (notification-channels.md §3.3): declared-input links to
// the Source, an aperture widening to the Seed, and drift/threshold to the
// subject's own page. It returns "" when no base URL is configured.
func link(base string, cause message.Cause, subjectKind, firedAt string) string {
	if base == "" {
		return ""
	}
	base = trimSlash(base)
	switch message.LinkKindForCause(cause) {
	case message.LinkSource:
		return base + "/sources"
	case message.LinkSeed:
		return base + "/seeds"
	default: // LinkObject
		switch subjectKind {
		case "service":
			return base + "/subjects/service?key=" + url.QueryEscape(firedAt)
		case "name", "address":
			return base + "/subjects/" + url.PathEscape(firedAt)
		default:
			return base + "/subjects"
		}
	}
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
