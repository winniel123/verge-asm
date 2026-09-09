package signal

import (
	"net/url"
	"strings"
	"time"

	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	hx "github.com/winniel123/verge-asm/internal/measure/httpexchange"
)

// Mirrors internal/measure/connectoutcome: the two negatives are values, never an absence.

const (
	CertPresented  = "presented"
	CertTLSRefused = "tls-refused"
	CertNoTLS      = "no-tls"
)

// A nil attribute is evidence we do not hold: that rule alone is not-evaluable (collision #37).

type CertDetails struct {
	Clock              *CertClock
	SelfSigned         *bool
	WeakKeyOrSignature *bool

	// The rule fires on this field's negation, so an unread SAN set may never default to false.

	SANMatchesName *bool
}

type EndpointFacts struct {
	// A nameless endpoint keys as @address:port/transport, with no hostname to mismatch (ADR-0011).

	Subject string
	HasName bool

	// False where the Service was never reached or handshaked, so no certificate value exists.

	CertMeasured bool
	CertOutcome  string
	CertDetails  *CertDetails

	// An Endpoint exists for a pair only where its HTTP exchange completed (CONTEXT.md `Endpoint`).

	HTTPResponded    bool
	HTTPStatus       int
	RedirectLocation string // A 3xx Location is recorded and never followed.

	RedirectHostInEstate bool
}

type EndpointRule interface {
	Name() string
	Version() Version
	Severity() Severity
	Eval(f EndpointFacts) Outcome
}

func AllEndpointRules() []EndpointRule {
	// The ADR-0024 table order is what renders and what the gate walks, so it may not be resorted.
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

func certVersion() Version { return Version{Rule: "v1", Composes: []string{co.CertVersion}} }

type CertClock struct {
	NotBefore   time.Time
	NotAfter    time.Time
	ObservedAt  time.Time
	EvaluatedAt time.Time
}

// N is the issuer's replacement window; a flat 30 days cannot split a six-day cert (ADR-0004 #67).

const certHorizonVersion = "cert-horizon/v1"

func CertHorizon(notBefore, notAfter time.Time) (time.Duration, bool) {
	validity := notAfter.Sub(notBefore)
	if validity <= 0 {
		return 0, false
	}
	if validity <= 10*24*time.Hour {
		return validity / 2, true
	}
	return validity / 3, true
}

// An observation older than N is surely superseded, so the clock class declines it (ADR-0043).

type clockRule struct {
	name string
	sev  Severity
	read func(c CertClock, horizon time.Duration) bool
}

func (r clockRule) Name() string { return r.name }
func (r clockRule) Version() Version {
	return Version{Rule: "v2", Composes: sortedStrings(co.CertVersion, certHorizonVersion)}
}
func (r clockRule) Severity() Severity { return r.sev }
func (r clockRule) Eval(f EndpointFacts) Outcome {
	if !presentedCert(f) {
		return OutsideDomain
	}
	if f.CertDetails == nil || f.CertDetails.Clock == nil {
		return NotEvaluable
	}
	c := *f.CertDetails.Clock
	horizon, ok := CertHorizon(c.NotBefore, c.NotAfter)
	if !ok || c.EvaluatedAt.Sub(c.ObservedAt) > horizon {
		return NotEvaluable
	}
	if r.read(c, horizon) {
		return Fired
	}
	return NotFired
}

// Read-side floors, so an edit Breaks this rule alone, never every certificate timeline (#715 §6).

const weakKeyFloorVersion = "weak-key-floor/v1"

type weakKeyRule struct{ certDetailRule }

func (r weakKeyRule) Version() Version {
	return Version{Rule: "v1", Composes: sortedStrings(co.CertVersion, weakKeyFloorVersion)}
}

func presentedCert(f EndpointFacts) bool {
	return f.CertMeasured && f.CertOutcome == CertPresented
}

// A sixth rule of this shape is added by naming it and its picker, not by copying control flow.

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
		return NotEvaluable
	}
	if *attr {
		return Fired
	}
	return NotFired
}

// Rated by what breaks TLS for a client today, not by how bad the certificate looks (ADR-0186 §2).

var (
	certificateExpired = clockRule{"certificate-expired", SevCritical, func(c CertClock, _ time.Duration) bool {
		return !c.NotAfter.After(c.EvaluatedAt)
	}}
	certificateNotYetValid = clockRule{"certificate-not-yet-valid", SevHigh, func(c CertClock, _ time.Duration) bool {
		return c.NotBefore.After(c.EvaluatedAt)
	}}
	certificateExpiring = clockRule{"certificate-expiring", SevMedium, func(c CertClock, horizon time.Duration) bool {
		return c.NotAfter.After(c.EvaluatedAt) && !c.NotAfter.After(c.EvaluatedAt.Add(horizon))
	}}
	certificateSelfSigned         = certDetailRule{"certificate-self-signed", SevMedium, func(d CertDetails) *bool { return d.SelfSigned }}
	certificateWeakKeyOrSignature = weakKeyRule{certDetailRule{"certificate-weak-key-or-signature", SevHigh, func(d CertDetails) *bool { return d.WeakKeyOrSignature }}}
)

type certificateHostnameSANMismatch struct{}

func (certificateHostnameSANMismatch) Name() string     { return "certificate-hostname-san-mismatch" }
func (certificateHostnameSANMismatch) Version() Version { return certVersion() }

func (certificateHostnameSANMismatch) Severity() Severity { return SevHigh }
func (certificateHostnameSANMismatch) Eval(f EndpointFacts) Outcome {
	if !presentedCert(f) || !f.HasName {
		return OutsideDomain
	}
	if f.CertDetails == nil || f.CertDetails.SANMatchesName == nil {
		return NotEvaluable
	}
	if !*f.CertDetails.SANMatchesName {
		return Fired
	}
	return NotFired
}

// The domain reads one facet and the predicate another, which is a property of the rule (ADR-0024).

type plaintextHTTPNoHTTPS struct{}

func (plaintextHTTPNoHTTPS) Name() string { return "plaintext-http-no-https" }
func (plaintextHTTPNoHTTPS) Version() Version {
	return Version{Rule: "v1", Composes: sortedStrings(hx.Version, co.CertVersion)}
}

// A hardening gap rather than an immediate compromise, so this is rated below a live exposure.

func (plaintextHTTPNoHTTPS) Severity() Severity { return SevMedium }
func (plaintextHTTPNoHTTPS) Eval(f EndpointFacts) Outcome {
	// Not a port: the 80/tcp literal was withdrawn, so an HTTP app on 8080 with no TLS fires here.
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

func is3xxWithLocation(f EndpointFacts) bool {
	return f.HTTPResponded && f.HTTPStatus >= 300 && f.HTTPStatus <= 399 && f.RedirectLocation != ""
}

func RedirectTarget(location string) (scheme, host string) {
	// Exported so the web layer folds RedirectHostInEstate against one parse, never a second.
	u, err := url.Parse(strings.TrimSpace(location))
	if err != nil {
		return "", ""
	}
	return strings.ToLower(u.Scheme), strings.ToLower(u.Hostname())
}

type redirectDoesNotUpgradeToTLS struct{}

func (redirectDoesNotUpgradeToTLS) Name() string { return "redirect-does-not-upgrade-to-tls" }
func (redirectDoesNotUpgradeToTLS) Version() Version {
	return Version{Rule: "v1", Composes: []string{hx.Version}}
}

func (redirectDoesNotUpgradeToTLS) Severity() Severity { return SevLow }
func (redirectDoesNotUpgradeToTLS) Eval(f EndpointFacts) Outcome {
	if !is3xxWithLocation(f) {
		return OutsideDomain
	}
	scheme, _ := RedirectTarget(f.RedirectLocation)
	// A relative Location keeps the current scheme, so a plaintext page still fires.
	if scheme != "https" {
		return Fired
	}
	return NotFired
}

type redirectToHostOutsideEstate struct{}

func (redirectToHostOutsideEstate) Name() string { return "redirect-to-host-outside-estate" }
func (redirectToHostOutsideEstate) Version() Version {
	// Estate membership is Derived from the resolution leaves, so the vector composes those too.
	return Version{Rule: "v1", Composes: sortedStrings(append([]string{hx.Version}, leafVersions...)...)}
}

func (redirectToHostOutsideEstate) Severity() Severity { return SevMedium }
func (redirectToHostOutsideEstate) Eval(f EndpointFacts) Outcome {
	if !is3xxWithLocation(f) {
		return OutsideDomain
	}
	_, host := RedirectTarget(f.RedirectLocation)
	if host == "" {
		// A relative redirect stays on this origin, which is in the estate by construction.
		return NotFired
	}
	if f.RedirectHostInEstate {
		return NotFired
	}
	return Fired
}

type unauthenticatedRequestAnswered struct{}

func (unauthenticatedRequestAnswered) Name() string { return "unauthenticated-request-answered" }
func (unauthenticatedRequestAnswered) Version() Version {
	return Version{Rule: "v1", Composes: []string{hx.Version}}
}

func (unauthenticatedRequestAnswered) Severity() Severity { return SevHigh }
func (unauthenticatedRequestAnswered) Eval(f EndpointFacts) Outcome {
	if !f.HTTPResponded {
		return OutsideDomain
	}
	// A 3xx is outside because it is the redirect rules' domain, not this one's.
	answered := f.HTTPStatus >= 200 && f.HTTPStatus <= 299
	challenged := f.HTTPStatus == 401 || f.HTTPStatus == 403
	if !answered && !challenged {
		return OutsideDomain
	}
	// No not-evaluable case: a status is a determinate value, never a fact about our own sight.
	if answered {
		return Fired
	}
	return NotFired
}
