package act

import (
	"reflect"
	"strings"
	"testing"
)

// The Action cell §2.1 draws for every class. Six classes share Dial moved, because the
// Subject cell carries the dial name: the count in §2.1's own note predates the two-dial
// split #1790 added, and the table is the enumeration.

var labels = map[string]string{
	"setup.completed":           "Instance set up",
	"password.reset":            "Password reset",
	"invite.accepted":           "Invite accepted",
	"onboarding.finished":       "Scan dispatched",
	"seed.declared":             "Seed declared",
	"seed.withdrawn":            "Seed withdrawn",
	"seed.custody.moved":        "Custody moved",
	"zone.declared":             "Zone declared",
	"zone.cadence.set":          "Dial moved",
	"dns.cadence.set":           "Dial moved",
	"exclusion.declared":        "Exclusion declared",
	"exclusion.lifted":          "Exclusion lifted",
	"cold.moved":                "Cold scan moved",
	"vantage.declared":          "Vantage declared",
	"vantage.resolver.set":      "Resolver set",
	"schedule.declared":         "Schedule declared",
	"schedule.edited":           "Schedule edited",
	"schedule.withdrawn":        "Schedule withdrawn",
	"annotation.declared":       "Annotation declared",
	"annotation.withdrawn":      "Annotation withdrawn",
	"proposal.queried":          "Registries queried",
	"proposal.confirmed":        "Proposal confirmed",
	"proposal.declined":         "Proposal declined",
	"proposal.decline.undone":   "Decline lifted",
	"scan.triggered":            "Scan triggered",
	"scan.stopped":              "Scan stopped",
	"scan.terminated":           "Scan terminated",
	"frequency.moved":           "Frequency moved",
	"source.moved":              "Source moved",
	"password.changed":          "Password changed",
	"token.minted":              "Token minted",
	"token.revoked":             "Token revoked",
	"sso.unlinked":              "SSO unlinked",
	"account.created":           "Account created",
	"totp.enrolled":             "TOTP enrolled",
	"invite.minted":             "Invite minted",
	"account.role.moved":        "Role moved",
	"totp.stripped":             "TOTP stripped",
	"account.removed":           "Account removed",
	"channel.declared":          "Channel declared",
	"channel.updated":           "Channel updated",
	"channel.withdrawn":         "Channel withdrawn",
	"channel.tested":            "Channel tested",
	"transcript.currency.set":   "Dial moved",
	"observation.currency.set":  "Dial moved",
	"dispatch.cadence.set":      "Dial moved",
	"address.cap.set":           "Dial moved",
	"update.check.moved":        "Update check moved",
	"restore.applied":           "Restore applied",
	"api.access.moved":          "API access moved",
	"sso.provider.declared":     "SSO provider declared",
	"sso.provider.updated":      "SSO provider updated",
	"sso.provider.secret.set":   "SSO secret set",
	"sso.provider.withdrawn":    "SSO provider withdrawn",
	"sso.binding.removed":       "SSO binding removed",
	"integration.installed":     "Integration installed",
	"integration.removed":       "Integration removed",
	"integration.channel.bound": "Channel bound",
	"integration.tested":        "Integration tested",
	"sso.binding.created":       "SSO linked",
	"transcript.disclosed":      "Raw output disclosed",
}

// All 61 variants fill the four columns with no empty cell, so no fifth is owed (§4.4).

func TestEveryClassCarriesTheActionLabelSpecDraws(t *testing.T) {
	if len(labels) != classCount {
		t.Fatalf("the label table holds %d entries, §2.1 gives %d classes", len(labels), classCount)
	}
	for _, a := range Classes {
		want, drawn := labels[a.Class()]
		if !drawn {
			t.Errorf("§2.1 draws no Action label for class %q (%T)", a.Class(), a)
			continue
		}
		if got := a.Label(); got != want {
			t.Errorf("%T renders Action %q, §2.1 draws %q", a, got, want)
		}
	}
	for class := range labels {
		if _, known := registry[class]; !known {
			t.Errorf("the label table names class %q, which no variant claims", class)
		}
	}
}

// Both halves, because the prefix alone is not the property: a future label authored as
// "@something" would compile, round-trip and render cleanly (§3.5 fact 4).

func TestTheMarkAndTheGrantLabelsAreDisjoint(t *testing.T) {
	// Half one: every Account render carries the mark.
	for _, username := range []string{"alice", "setup token", "password-reset link", "invite 12", "@alice", ""} {
		got := ActorCell(Account{AccountID: 7, UsernameSnapshot: username})
		if !strings.HasPrefix(got, mark) {
			t.Errorf("ActorCell for username %q rendered %q, which carries no mark", username, got)
		}
		if got == "" {
			t.Errorf("ActorCell for username %q rendered an empty cell", username)
		}
	}
	for _, c := range []struct {
		act  Act
		want string
	}{
		{AccountRemoved{AccountRef{Username: "setup token"}}, "@setup token"},
		{AccountCreated{AccountAtRole{Username: "bob", Role: "viewer"}}, "@bob · viewer"},
		{SSOBindingCreated{BindingRef{Slug: "okta", Username: "alice"}}, "okta · @alice"},
	} {
		if got := SubjectCell(c.act); got != c.want {
			t.Errorf("SubjectCell(%T) = %q, want %q", c.act, got, c.want)
		}
	}

	// Half two: no authored grant label begins with the mark, in either union.
	for tag, typ := range grants {
		label := reflect.New(typ).Elem().Interface().(Grant).Label()
		if strings.HasPrefix(label, mark) {
			t.Errorf("grant %q renders %q, which collides with a marked account name", tag, label)
		}
		if got := ActorCell(GrantHolder{Grant: reflect.New(typ).Elem().Interface().(Grant)}); got != label {
			t.Errorf("ActorCell marked grant %q as %q; the mark reaches accounts alone", tag, got)
		}
	}
	for _, a := range Classes {
		if strings.HasPrefix(a.Label(), mark) {
			t.Errorf("class %q renders the Action label %q, which reads as a marked name", a.Class(), a.Label())
		}
	}
}

// Every payload that renders a bare username is marked, and the mark goes no further (§3.5 fact 3).

func TestTheMarkReachesExactlyTheThreeUsernamePayloads(t *testing.T) {
	marked := map[string]bool{}
	for _, s := range samples {
		if _, ok := s.act.(markedSubject); ok {
			marked[s.act.Class()] = true
		}
		// The store keeps the bare value, so an unmarked render stays available (§3.5 fact 1).
		if got := s.act.Subject(); got != s.subject {
			t.Errorf("%T's bare Subject() = %q, want %q", s.act, got, s.subject)
		}
	}
	want := map[string]bool{
		"setup.completed": true, "invite.accepted": true, "password.reset": true,
		"password.changed": true, "totp.enrolled": true, "account.created": true,
		"account.role.moved": true, "totp.stripped": true, "account.removed": true,
		"sso.binding.removed": true, "sso.binding.created": true,
	}
	if !reflect.DeepEqual(marked, want) {
		t.Errorf("the mark reaches %v, want %v", keys(marked), keys(want))
	}
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
