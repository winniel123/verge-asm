// Package delivery is the outbound half of a Notification Channel: it turns a
// stored Message into a signed https POST to each Channel that carries the
// Message's class, and records the per-Channel outcome (v1 spec §4.5, ADR-0039).
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

const (
	HeaderTimestamp = "X-Verge-Timestamp"
	HeaderSignature = "X-Verge-Signature"
	contentType     = "application/json"
	sigScheme       = "sha256="
)

type Firing struct {
	ID          int64
	Cause       message.Cause
	Class       message.Class
	SubjectKind string
	FiredAt     string
	Instant     time.Time
	Census      []byte
	Headline    string
}

// No field may enumerate the rows behind a census count (notification-channels.md §3.1).

type Body struct {
	Message  int64     `json:"message"`
	Class    string    `json:"class"`
	Cause    string    `json:"cause"`
	Subject  Subject   `json:"subject"`
	Instant  time.Time `json:"instant"`
	Headline string    `json:"headline"`
	Census   *int      `json:"census,omitempty"`
	Link     string    `json:"link,omitempty"`
}

type Subject struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}

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
		if c, err := message.ParseCensus(f.Census); err == nil {
			n := c.Len()
			b.Census = &n
		}
	}
	return b
}

// Field order is encoding/json's declaration order, so the signed bytes are stable across retries.

func MarshalBody(b Body) ([]byte, error) { return json.Marshal(b) }

func Routes(routeDrift, routeCoverage, routeClock bool, class message.Class) bool {
	// Class alone routes: there is no per-rule and no per-subject axis to consult (ADR-0091).
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

func Sign(secret, body []byte, ts time.Time) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func NewRequest(ctx context.Context, targetURL string, body, secret []byte, ts time.Time) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	// The receiver reconstructs the signed input from this header, even where no secret is set.
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(ts.Unix(), 10))
	if len(secret) > 0 {
		req.Header.Set(HeaderSignature, sigScheme+Sign(secret, body, ts))
	}
	// Nothing authenticates them to us, so no bearer header is ever set (§3.2).
	return req, nil
}

type Verdict int

const (
	VerdictDelivered Verdict = iota
	VerdictRetry
	VerdictUndelivered
)

func Decide(delivered bool, attempt, maxAttempts int32) Verdict {
	if delivered {
		return VerdictDelivered
	}
	if attempt < maxAttempts {
		return VerdictRetry
	}
	return VerdictUndelivered
}

func Delivered(statusCode int) bool { return statusCode >= 200 && statusCode < 300 }

func link(base string, cause message.Cause, subjectKind, firedAt string) string {
	if base == "" {
		return ""
	}
	base = trimSlash(base)
	// The per-mover link destinations are fixed by notification-channels.md §3.3.
	switch message.LinkKindForCause(cause) {
	case message.LinkSource:
		return base + "/sources"
	case message.LinkSeed:
		return base + "/seeds"
	default:
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
