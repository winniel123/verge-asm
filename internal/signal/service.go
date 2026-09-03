package signal

import (
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

const (
	Reached    = "reached"
	NotReached = "not-reached"
)

type ServiceFacts struct {
	Subject string

	// The list is verge-core's sensitive half restricted to its probed pairs (ADR-0024).

	OnSensitiveList bool

	// The domain is the port list and not the vantage, so the internet leg is read in the predicate.

	HasInternetReach bool
	InternetReach    string

	// An absent tls-acceptance facet leaves no Service in the domain (#199).

	TLSHandshakeCompleted bool
	TLSVersionsReadable   bool
	TLS10Accepted         bool
}

type ServiceRule interface {
	Name() string
	Version() Version
	Severity() Severity
	Eval(f ServiceFacts) Outcome
}

func AllServiceRules() []ServiceRule {
	// A port tier bounds which Services exist, never which rules may speak.
	return []ServiceRule{
		tls10Accepted{},
		sensitivePortReachedFromInternet{},
	}
}

func EvaluateService(r ServiceRule, services []ServiceFacts) Census {
	c := Census{Rule: r.Name(), Version: r.Version()}
	for _, f := range services {
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

type tls10Accepted struct{}

func (tls10Accepted) Name() string { return "tls-1.0-accepted" }

func (tls10Accepted) Severity() Severity { return SevMedium }
func (tls10Accepted) Version() Version {
	// Named and never imported, so the leaf's version bump is coordinated on tickets (#199).
	return Version{Rule: "v1", Composes: []string{"tls-acceptance/v1"}}
}
func (tls10Accepted) Eval(f ServiceFacts) Outcome {
	if !f.TLSHandshakeCompleted {
		return OutsideDomain
	}
	if !f.TLSVersionsReadable {
		return NotEvaluable
	}
	if f.TLS10Accepted {
		return Fired
	}
	return NotFired
}

type sensitivePortReachedFromInternet struct{}

func (sensitivePortReachedFromInternet) Name() string {
	return "sensitive-port-reached-from-internet"
}

func (sensitivePortReachedFromInternet) Severity() Severity { return SevCritical }
func (sensitivePortReachedFromInternet) Version() Version {
	// A release-coupled reference table, so the sensitive list adds no measured leaf (v1 spec §3.5).
	return Version{Rule: "v1", Composes: []string{co.Version}}
}
func (sensitivePortReachedFromInternet) Eval(f ServiceFacts) Outcome {
	// The sensitive-list join fixes the domain: this is not a separate port-membership signal.
	if !f.OnSensitiveList {
		return OutsideDomain
	}
	if !f.HasInternetReach {
		return NotEvaluable
	}
	if f.InternetReach == Reached {
		return Fired
	}
	return NotFired
}
