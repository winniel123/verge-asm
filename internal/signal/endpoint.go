package signal

import (
	"net/url"
	"strings"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	hx "github.com/winniel123/verge-asm/internal/measure/httpexchange"
)

// This file holds the ten `Signal` rules whose subject is an `Endpoint` — the
// `(Name, Service)` pair, keyed `name@address:port/transport` (v1 spec §5.2,
// ADR-0024). Six read the `certificate` facet, four read `http-identity`, and
// `plaintext-http-no-https` reads both — its domain is one facet and its predicate
// the other, the case ADR-0024 used to show a domain is a property of the RULE,
// not of a facet. All ride `EndpointFacts`.

// Certificate outcome constants — the closed union the `certificate` facet holds
// (CONTEXT.md `Certificate`), mirroring internal/measure/connectoutcome. Only a
// presentation carries a chain; the two negatives are values in their own right.
const (
	CertPresented  = "presented"
	CertTLSRefused = "tls-refused"
	CertNoTLS      = "no-tls"
)

// CertDetails is the parsed attributes of a presented certificate chain that the
// five certificate predicates and the hostname-SAN rule read. It is a POINTER on
// EndpointFacts: nil means the chain was presented but NONE of its attributes could
// be read (no parsed leaf at all), so every certificate rule renders the subject
// `not-evaluable` rather than manufacturing a verdict from evidence it does not
// have.
//
// Each attribute is INDEPENDENTLY nullable (a `*bool`): a nil attribute means THAT
// rule's own input is absent, so THAT rule alone is `not-evaluable` — no single
// pointer gates all six cert rules, and a rule never emits a verdict from evidence
// it does not hold (collision #37, P0.10a). A non-nil attribute is the read
// verdict: `*attr == true` fires, `*attr == false` is a clean not-fired.
//
// SANMatchesName is held to the same discipline and it is the sharp case: absent
// SANs are nil → `not-evaluable`, NEVER a false that would manufacture a mismatch
// verdict (the hard constraint of the ruling). TDD supplies non-nil attributes to
// exercise each predicate; P0.10a's web wire sets only Expired/Expiring from the
// leaf's `not_after`, leaving the other four attributes nil (they land in P0.10b,
// #704) — see cmd/web/signals.go buildEndpointFacts.
type CertDetails struct {
	Expired            *bool
	NotYetValid        *bool
	Expiring           *bool
	SelfSigned         *bool
	WeakKeyOrSignature *bool
	// SANMatchesName reports whether the presented chain's SANs cover the
	// Endpoint's Name — its negation is `certificate-hostname-san-mismatch`'s
	// predicate. nil = SANs unread → not-evaluable (never a defaulted mismatch).
	SANMatchesName *bool
}

// EndpointFacts is the current Derived state about one `Endpoint` the ten
// Endpoint rules read — the evidence they declare across the `certificate` and
// `http-identity` facets (ADR-0024). Every field is a value about *now*.
type EndpointFacts struct {
	// Subject is the Endpoint key — `name@address:port/transport`, or
	// `@address:port/transport` for the nameless endpoint. HasName is false for the
	// nameless endpoint, which puts it outside `certificate-hostname-san-mismatch`'s
	// domain (a nameless endpoint has no hostname to mismatch — ADR-0011/ADR-0024).
	Subject string
	HasName bool

	// The `certificate` facet. CertMeasured is false where no certificate value
	// exists (the Service was never reached, or never handshaked); CertOutcome is
	// the closed union `presented | tls-refused | no-tls`. The five certificate
	// rules' domain is `certificate` is `presented`; `no-tls` and `tls-refused` are
	// outside. CertDetails carries the parsed leaf attributes: nil = NO attribute was
	// read at all (every cert rule not-evaluable); non-nil with per-attribute nil = that
	// one attribute is absent, so THAT rule alone is not-evaluable (collision #37).
	CertMeasured bool
	CertOutcome  string
	CertDetails  *CertDetails

	// The `http-identity` facet. HTTPResponded is the four HTTP rules' base domain
	// — `http-identity` is `Responded` (an Endpoint exists for a pair only where
	// its HTTP exchange completed, CONTEXT.md `Endpoint`); false is `NoHTTPResponse`,
	// outside every HTTP rule's domain. HTTPStatus is the single `GET /`'s status;
	// RedirectLocation is the `Location` of a 3xx, recorded and never followed.
	HTTPResponded    bool
	HTTPStatus       int
	RedirectLocation string

	// RedirectHostInEstate reports whether the host the 3xx `Location` names is a
	// subject in the estate — the pre-folded evidence `redirect-to-host-outside-estate`
	// reads (the estate membership is a Derived value the web layer folds, like
	// InDeclaredZone on NameFacts). It is meaningful only where the Endpoint is in
	// that rule's domain (a 3xx with a Location).
	RedirectHostInEstate bool
}

// EndpointRule is one `Signal` whose subject is an `Endpoint`.
type EndpointRule interface {
	Name() string
	Version() Version
	Severity() Severity
	Eval(f EndpointFacts) Outcome
}

// AllEndpointRules returns the shipped Endpoint rules in a stable order — the
// ADR-0024 table order (the five certificate rules, the hostname-SAN rule, then
// the four http-identity rules), the order they render and the gate walks them.
func AllEndpointRules() []EndpointRule {
	return []EndpointRule{
		certificateExpired,
		certificateNotYetValid,
		certificateExpiring,
		certificateSelfSigned,
		certificateWeakKeyOrSignature,
		certificateHostnameSANMismatch{},
		plaintextHTTPNoHTTPS{},
		redirectDoesNotUpgradeToTLS{},
		redirectToHostOutsideEstate{},
		unauthenticatedRequestAnswered{},
	}
}

// EvaluateEndpoint runs one Endpoint rule over the current Endpoint snapshot,
// bucketing each subject and dropping the ones outside the domain. Members are
// ordered by subject (ADR-0102).
func EvaluateEndpoint(r EndpointRule, endpoints []EndpointFacts) Census {
	c := Census{Rule: r.Name(), Version: r.Version()}
	for _, f := range endpoints {
		switch r.Eval(f) {
		case Fired:
			c.Fired = append(c.Fired, Member{Subject: f.Subject})
		case NotFired:
			c.NotFired = append(c.NotFired, Member{Subject: f.Subject})
		case NotEvaluable:
			c.NotEvaluable = append(c.NotEvaluable, Member{Subject: f.Subject})
		}
	}
	sortMembers(c.Fired)
	sortMembers(c.NotFired)
	sortMembers(c.NotEvaluable)
	return c
}

// certVersion is the version vector every certificate-reading rule composes: the
// `tls-handshake` leaf that decides the `certificate` facet. A bump of that leaf
// moves every rule that reads the value it decides.
func certVersion() Version { return Version{Rule: "v1", Composes: []string{co.CertVersion}} }

// presentedCert reports whether a certificate rule's domain is satisfied: a
// certificate was measured and it is `presented`. `no-tls` / `tls-refused` /
// unmeasured are all outside the certificate rules' domain (ADR-0024: NoTLS
// outside).
func presentedCert(f EndpointFacts) bool {
	return f.CertMeasured && f.CertOutcome == CertPresented
}

// --- the five certificate-detail rules ------------------------------------
//
// All five share one shape: domain `certificate` is `Presented`; `not-evaluable`
// where the chain is presented but this rule's own parsed attribute is unread —
// either the whole leaf is absent (CertDetails nil) or just this attribute is
// (its `*bool` is nil); Fired where the read attribute is true; otherwise NotFired.
// They differ ONLY in which parsed-leaf attribute they read, so one parameterised
// rule carries all five — a sixth of this kind is added by naming it and its
// picker, never by copying the control flow. (certificate-hostname-san-mismatch
// below keeps its own body: it adds a HasName domain guard and reads the *negation*
// of its attribute, so it is not this shape.)

type certDetailRule struct {
	name string
	sev  Severity
	pick func(CertDetails) *bool
}

func (r certDetailRule) Name() string       { return r.name }
func (r certDetailRule) Version() Version   { return certVersion() }
func (r certDetailRule) Severity() Severity { return r.sev }
func (r certDetailRule) Eval(f EndpointFacts) Outcome {
	if !presentedCert(f) {
		return OutsideDomain
	}
	if f.CertDetails == nil {
		return NotEvaluable
	}
	attr := r.pick(*f.CertDetails)
	if attr == nil {
		// This rule's own input is absent — not-evaluable, never a defaulted verdict.
		return NotEvaluable
	}
	if *attr {
		return Fired
	}
	return NotFired
}

// The five shipped certificate-detail rules, in ADR-0024 table order. Each names
// the one parsed-leaf boolean it reads; the domain, not-evaluable, and
// fired/not-fired control flow live once, in certDetailRule.Eval above.
// Severities, per rule: an expired or not-yet-valid leaf breaks TLS for clients
// today (critical / high); a weak key or signature is a high-value forgery risk;
// a self-signed leaf and an approaching expiry are medium warnings.
var (
	certificateExpired            = certDetailRule{"certificate-expired", SevCritical, func(d CertDetails) *bool { return d.Expired }}
	certificateNotYetValid        = certDetailRule{"certificate-not-yet-valid", SevHigh, func(d CertDetails) *bool { return d.NotYetValid }}
	certificateExpiring           = certDetailRule{"certificate-expiring", SevMedium, func(d CertDetails) *bool { return d.Expiring }}
	certificateSelfSigned         = certDetailRule{"certificate-self-signed", SevMedium, func(d CertDetails) *bool { return d.SelfSigned }}
	certificateWeakKeyOrSignature = certDetailRule{"certificate-weak-key-or-signature", SevHigh, func(d CertDetails) *bool { return d.WeakKeyOrSignature }}
)

// --- certificate-hostname-san-mismatch ------------------------------------
//
// Domain: `certificate` is `Presented` AND the `Endpoint` has a `Name` (ADR-0024)
// — a nameless endpoint has no hostname to mismatch (ADR-0011). Predicate: the
// presented chain's SANs do not cover the Endpoint's Name. `not-evaluable`: the
// chain is presented but its SANs are unreadable — the whole leaf is absent
// (CertDetails nil) OR the SANMatchesName attribute alone is (its `*bool` is nil).
// Absent SANs are NEVER read as a mismatch: the rule emits no verdict from evidence
// it does not hold (the hard constraint of collision #37 / P0.10a).

type certificateHostnameSANMismatch struct{}

func (certificateHostnameSANMismatch) Name() string     { return "certificate-hostname-san-mismatch" }
func (certificateHostnameSANMismatch) Version() Version { return certVersion() }

// Severity: high — a chain whose SANs do not cover the endpoint's name fails
// verification and is indistinguishable from a misissued or wrong certificate.
func (certificateHostnameSANMismatch) Severity() Severity { return SevHigh }
func (certificateHostnameSANMismatch) Eval(f EndpointFacts) Outcome {
	if !presentedCert(f) || !f.HasName {
		return OutsideDomain
	}
	if f.CertDetails == nil || f.CertDetails.SANMatchesName == nil {
		// No SANs read — not-evaluable, never a defaulted mismatch verdict.
		return NotEvaluable
	}
	if !*f.CertDetails.SANMatchesName {
		return Fired
	}
	return NotFired
}

// --- plaintext-http-no-https ----------------------------------------------
//
// Domain: `http-identity` is `Responded` (ADR-0024 — NOT a port: the `80/tcp`
// literal was withdrawn, an HTTP app on 8080 with no TLS is exactly what the rule
// is named for). Predicate: the same Endpoint's `certificate` is `NoTLS`. This is
// the rule ADR-0024 used to show a domain is a property of the RULE not the facet:
// its domain reads one facet and its predicate another. `not-evaluable`: HTTP
// responded but the certificate facet holds no value, so whether TLS is present
// cannot be read.

type plaintextHTTPNoHTTPS struct{}

func (plaintextHTTPNoHTTPS) Name() string { return "plaintext-http-no-https" }
func (plaintextHTTPNoHTTPS) Version() Version {
	return Version{Rule: "v1", Composes: sortedStrings(hx.Version, co.CertVersion)}
}

// Severity: medium — an HTTP app answering with no TLS exposes traffic to
// interception, but is a hardening gap rather than an immediate compromise.
func (plaintextHTTPNoHTTPS) Severity() Severity { return SevMedium }
func (plaintextHTTPNoHTTPS) Eval(f EndpointFacts) Outcome {
	if !f.HTTPResponded {
		return OutsideDomain
	}
	if !f.CertMeasured {
		return NotEvaluable
	}
	if f.CertOutcome == CertNoTLS {
		return Fired
	}
	return NotFired
}

// is3xxWithLocation reports whether an http-identity is a 3xx carrying a
// `Location` — the shared domain of the two redirect rules (ADR-0024).
func is3xxWithLocation(f EndpointFacts) bool {
	return f.HTTPResponded && f.HTTPStatus >= 300 && f.HTTPStatus <= 399 && f.RedirectLocation != ""
}

// RedirectTarget parses a 3xx `Location` into its scheme and host, lowercased.
// The scheme decides whether a redirect upgrades to TLS; the host is what the
// estate-membership test reads. A relative Location (no host) yields an empty
// host — a redirect that stays on the same origin, which never leaves the estate.
// Exported so the web layer folds RedirectHostInEstate against the SAME parse the
// engine's redirect rules use, keeping one truth for the target.
func RedirectTarget(location string) (scheme, host string) {
	u, err := url.Parse(strings.TrimSpace(location))
	if err != nil {
		return "", ""
	}
	return strings.ToLower(u.Scheme), strings.ToLower(u.Hostname())
}

// --- redirect-does-not-upgrade-to-tls -------------------------------------
//
// Domain: `http-identity` is `Responded` with a 3xx and a `Location` (ADR-0024).
// Predicate: the redirect does not upgrade to TLS — its `Location` scheme is not
// `https`. A relative Location keeps the current scheme, so a plaintext response
// redirecting to a relative path does not upgrade and fires.

type redirectDoesNotUpgradeToTLS struct{}

func (redirectDoesNotUpgradeToTLS) Name() string { return "redirect-does-not-upgrade-to-tls" }
func (redirectDoesNotUpgradeToTLS) Version() Version {
	return Version{Rule: "v1", Composes: []string{hx.Version}}
}

// Severity: low — a redirect that does not upgrade to TLS is a best-practice miss;
// the plaintext-http-no-https rule carries the underlying exposure.
func (redirectDoesNotUpgradeToTLS) Severity() Severity { return SevLow }
func (redirectDoesNotUpgradeToTLS) Eval(f EndpointFacts) Outcome {
	if !is3xxWithLocation(f) {
		return OutsideDomain
	}
	scheme, _ := RedirectTarget(f.RedirectLocation)
	if scheme != "https" {
		return Fired
	}
	return NotFired
}

// --- redirect-to-host-outside-estate --------------------------------------
//
// Domain: `http-identity` is `Responded` with a 3xx and a `Location` (ADR-0024).
// Predicate: the host the `Location` names is not a subject in the estate. A
// relative Location names no host — it stays on the same origin, which is in the
// estate — so it does not fire. The estate membership is Derived from the
// resolution leaves, so the vector composes http-exchange AND resolution.

type redirectToHostOutsideEstate struct{}

func (redirectToHostOutsideEstate) Name() string { return "redirect-to-host-outside-estate" }
func (redirectToHostOutsideEstate) Version() Version {
	return Version{Rule: "v1", Composes: sortedStrings(append([]string{hx.Version}, leafVersions...)...)}
}

// Severity: medium — a redirect leaving the estate can hand a session or a click
// to an unowned host, a phishing / open-redirect risk.
func (redirectToHostOutsideEstate) Severity() Severity { return SevMedium }
func (redirectToHostOutsideEstate) Eval(f EndpointFacts) Outcome {
	if !is3xxWithLocation(f) {
		return OutsideDomain
	}
	_, host := RedirectTarget(f.RedirectLocation)
	if host == "" {
		// A relative redirect stays on this origin — in the estate by construction.
		return NotFired
	}
	if f.RedirectHostInEstate {
		return NotFired
	}
	return Fired
}

// --- unauthenticated-request-answered -------------------------------------
//
// Domain: `http-identity` is `Responded` with a 2xx OR a 401/403 (ADR-0024) — a
// 3xx is outside (it is the redirect rules'), and every other status is outside.
// Predicate: the request was answered rather than challenged — the status is 2xx.
// A 401/403 is the not-fired case (the endpoint challenged the unauthenticated
// request, which is the healthy answer). There is no `not-evaluable` case: the
// status is a determinate value, never a fact about our own sight.

type unauthenticatedRequestAnswered struct{}

func (unauthenticatedRequestAnswered) Name() string { return "unauthenticated-request-answered" }
func (unauthenticatedRequestAnswered) Version() Version {
	return Version{Rule: "v1", Composes: []string{hx.Version}}
}

// Severity: high — an endpoint answering an unauthenticated request 2xx is an
// exposed surface (an admin panel or API reachable without a challenge).
func (unauthenticatedRequestAnswered) Severity() Severity { return SevHigh }
func (unauthenticatedRequestAnswered) Eval(f EndpointFacts) Outcome {
	if !f.HTTPResponded {
		return OutsideDomain
	}
	answered := f.HTTPStatus >= 200 && f.HTTPStatus <= 299
	challenged := f.HTTPStatus == 401 || f.HTTPStatus == 403
	if !answered && !challenged {
		return OutsideDomain
	}
	if answered {
		return Fired
	}
	return NotFired
}
