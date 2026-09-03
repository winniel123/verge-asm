package signal

import (
	"net/url"
	"strings"

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
	Expired            *bool
	NotYetValid        *bool
	Expiring           *bool
	SelfSigned         *bool
	WeakKeyOrSignature *bool

	// The rule fires on this field's negation, so an unread SAN set may never default to false.

	SANMatchesName *bool
}

type EndpointFacts struct {
	// A nameless endpoint keys as @address:port/transport and has no hostname to mismatch (ADR-0011).

	Subject string
	HasName bool

	// False where the Service was never reached or never handshaked, so no certificate value exists.

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

// Rated by what breaks TLS for a client today, not by how bad the certificate looks.

var (
	certificateExpired            = certDetailRule{"certificate-expired", SevCritical, func(d CertDetails) *bool { return d.Expired }}
	certificateNotYetValid        = certDetailRule{"certificate-not-yet-valid", SevHigh, func(d CertDetails) *bool { return d.NotYetValid }}
	certificateExpiring           = certDetailRule{"certificate-expiring", SevMedium, func(d CertDetails) *bool { return d.Expiring }}
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
	// Exported so the web layer folds RedirectHostInEstate against this same parse, keeping one truth.
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
	// A relative Location keeps the current scheme, so a plaintext page redirecting relatively fires.
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
