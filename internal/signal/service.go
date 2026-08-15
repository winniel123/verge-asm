package signal

import (
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// This file holds the two `Signal` rules whose subject is a `Service` — the
// `(Address, port, transport)` triple the hot Scan turns into a subject (v1 spec
// §5.2, ADR-0024). Both are among the twelve rules bounded by which port tiers
// are enabled: the port tier bounds which `Service`s EXIST, never which rules may
// speak. Neither reads a `Name` fact, so each rides `ServiceFacts` rather than
// `NameFacts` — the engine is split by subject kind, not by rule.

// Reach outcome constants — the closed pair the `reachability` facet holds
// (CONTEXT.md `Reach`), mirroring internal/measure/connectoutcome. The web layer
// folds the internet-class leg into these values; the engine reads them.
const (
	Reached    = "reached"
	NotReached = "not-reached"
)

// ServiceFacts is the current Derived state about one `Service` that the two
// Service rules read — the only evidence they declare (ADR-0024). Every field is
// a value about *now*; a census is current state and never a comparison.
type ServiceFacts struct {
	// Subject is the Service key — `address:port/transport`, e.g.
	// `198.51.100.1:443/tcp`. It is what every census member drills to.
	Subject string

	// OnSensitiveList reports whether this Service's `(port, transport)` is on
	// `verge-core`'s sensitive half AND is a probed pair — the exact domain of
	// `sensitive-port-reached-from-internet` (ADR-0024: "Services whose
	// (port, transport) is on the sensitive list", restricted to verge-core's
	// probed pairs). A Service off the sensitive list is outside that rule's
	// domain: the fact *this sensitive port is reached from the internet* cannot be
	// true of a port that is not on the list.
	OnSensitiveList bool

	// The internet-class `Reach` leg, for `sensitive-port-reached-from-internet`
	// (ADR-0071/ADR-0080: a class-scoped internet, existential composition). The
	// rule's domain is the port list, not the vantage; the internet leg is read in
	// the predicate. HasInternetReach is false where no internet-class vantage
	// decided a value — a Gap, `not-evaluable`, never a silent not-fired.
	HasInternetReach bool
	InternetReach    string // Reached | NotReached

	// The `tls-acceptance` facet, for `tls-1.0-accepted`. TLSHandshakeCompleted is
	// the rule's domain — the Service completed at least one handshake in the
	// batch's candidate set (ADR-0024). It is read from the `tls-acceptance` facet
	// (#199), which lands concurrently: where that facet's producing data is not
	// present, TLSHandshakeCompleted is false and the Service is outside the domain
	// — the rule renders a no-population panel rather than taking a compile-time
	// dependency on #199's leaf. TLSVersionsReadable is false where a handshake
	// completed but which versions were accepted could not be read (`not-evaluable`).
	TLSHandshakeCompleted bool
	TLSVersionsReadable   bool
	TLS10Accepted         bool
}

// ServiceRule is one `Signal` whose subject is a `Service`. Same four-part shape
// as a Name Rule — name, version, predicate/domain/not-evaluable folded into Eval
// — over `ServiceFacts` instead of `NameFacts`.
type ServiceRule interface {
	Name() string
	Version() Version
	Eval(f ServiceFacts) Outcome
}

// AllServiceRules returns the shipped Service rules in a stable order — the order
// they render on the Signals page and the golden gate walks them.
func AllServiceRules() []ServiceRule {
	return []ServiceRule{
		tls10Accepted{},
		sensitivePortReachedFromInternet{},
	}
}

// EvaluateService runs one Service rule over the current Service snapshot,
// bucketing each subject into its census member and dropping the ones outside the
// domain (not rendered). Members are ordered by subject, never by verdict
// (ADR-0102), so the output is deterministic.
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

// --- tls-1.0-accepted -----------------------------------------------------
//
// Domain: the `Service` completed at least one handshake in the batch's candidate
// set (ADR-0024) — read from the `tls-acceptance` facet. A `Service` that
// accepted no TLS at all is outside the domain: *TLS 1.0 is accepted* cannot be
// true of a port that speaks no TLS. Predicate: the accepted-version set includes
// TLS 1.0. `not-evaluable`: a handshake completed but the version set could not be
// read. Where the `tls-acceptance` facet has no value at all (its leaf has not run
// — #199 lands concurrently) no `Service` can be confirmed in the domain, so the
// rule renders a no-population panel, NOT a compile-time dependency on #199.

type tls10Accepted struct{}

func (tls10Accepted) Name() string { return "tls-1.0-accepted" }
func (tls10Accepted) Version() Version {
	// Reads the `tls-acceptance` facet by NAME, not by importing #199's leaf
	// (the wave seam). The version composes that leaf's version string; the
	// concurrent coordination on its exact value rides the tickets, not a Go import.
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

// --- sensitive-port-reached-from-internet ---------------------------------
//
// Domain: `Service`s whose `(port, transport)` is on the sensitive list
// (ADR-0024), additionally restricted to `verge-core`'s probed pairs (the ticket
// AC): every other pair is outside — *this sensitive port is reached from the
// internet* cannot be true of a port that is not on the list. This is NOT a
// separate signal for port membership: the sensitive-list join fixes the domain,
// and the rule reads `Reach` like the rest. Predicate: the internet-class `Reach`
// leg is `reached` (ADR-0071: a class-scoped internet, existential composition —
// the internal twin is a different, refused rule). `not-evaluable`: no
// internet-class value at all (a Gap — we did not look from the internet).

type sensitivePortReachedFromInternet struct{}

func (sensitivePortReachedFromInternet) Name() string {
	return "sensitive-port-reached-from-internet"
}
func (sensitivePortReachedFromInternet) Version() Version {
	// Reads `Reach` (the connect-outcome leaf) and joins the release-coupled
	// sensitive list (§3.5); the reference table is what the attestation standard
	// governs, not the rule, so it adds no measured leaf to the vector.
	return Version{Rule: "v1", Composes: []string{co.Version}}
}
func (sensitivePortReachedFromInternet) Eval(f ServiceFacts) Outcome {
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
