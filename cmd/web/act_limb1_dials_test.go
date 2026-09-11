package main

import (
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/act"
)

// Limb 1, second half — the operator dials, the channels and the integrations (spec §1.1, §2.1).
// The gate proves a Record is reachable; it proves nothing about ordering, cardinality or content,
// so these do (spec §7.5).

func TestTheZoneAndDNSCadenceDialsEachRecordTheirOwnDial(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/seeds/zone/interval", url.Values{"interval_days": {"24"}}).Body.Close()
	wantOneAct(t, f, "zone.cadence.set", "zone scan cadence · 24 days")

	f.acts = nil
	postForm(t, ac, base+"/seeds/dns/interval", url.Values{"interval_days": {"1"}}).Body.Close()
	wantOneAct(t, f, "dns.cadence.set", "dns scan cadence · 1 day")
}

// A refused act directed nothing, and a phantom row can never be retracted (spec §7.6).

func TestARefusedCadenceRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	for _, path := range []string{"/seeds/zone/interval", "/seeds/dns/interval"} {
		postForm(t, ac, base+path, url.Values{"interval_days": {"0"}}).Body.Close()
	}

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused interval recorded %v, want nothing", got)
	}
}

func TestTheTranscriptCurrencyDialRecordsItsOwnDialAlone(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/retention", url.Values{
		"transcript_currency_days": {"14"},
	}).Body.Close()

	wantOneAct(t, f, "transcript.currency.set", "transcript currency · 14 days")
}

func TestARefusedTranscriptCurrencyRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/retention", url.Values{
		"transcript_currency_days": {"soon"},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused currency recorded %v, want nothing", got)
	}
}

func TestTheAddressCapDialRecordsABareCount(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"1024"}}).Body.Close()

	wantOneAct(t, f, "address.cap.set", "address-scope cap · 1024")
}

func TestARefusedAddressCapRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/address-cap", url.Values{"address_cap": {"0"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused cap recorded %v, want nothing", got)
	}
}

func TestTheUpdateCheckRecordsTheSwitchTheOperatorThrew(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/updates/check", url.Values{"enabled": {"true"}}).Body.Close()
	wantOneAct(t, f, "update.check.moved", "update check · on")

	f.acts = nil
	postForm(t, ac, base+"/settings/updates/check", url.Values{"enabled": {"false"}}).Body.Close()
	wantOneAct(t, f, "update.check.moved", "update check · off")
}

// Two dials, two rows: one Subject cell holding both is the list-valued subject §4.1 bars (§2.2).

func TestCoverageRetentionWritesTwoRowsWhenBothDialsMove(t *testing.T) {
	f := newFakeStore()
	seedRetentionPanel(f)
	f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"30"}, "dispatch_cadence_multiple": {"8"},
	}).Body.Close()

	got := actClasses(f)
	sort.Strings(got)
	want := []string{"dispatch.cadence.set", "observation.currency.set"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recorded %v, want %v", got, want)
	}
	if s := actSubjects(t, f, "observation.currency.set")[0]; s != "observation currency · 30 days" {
		t.Errorf("observation subject = %q", s)
	}
	// The unit is the panel's own, so the cell cannot be misread as a day count (ADR-0081).
	if s := actSubjects(t, f, "dispatch.cadence.set")[0]; s != "dispatch cadence · 8 cadences" {
		t.Errorf("dispatch subject = %q", s)
	}
}

func TestCoverageRetentionWritesOneRowWhenOneDialMoves(t *testing.T) {
	f := newFakeStore()
	seedRetentionPanel(f)
	f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"90"}, "dispatch_cadence_multiple": {"8"},
	}).Body.Close()

	wantOneAct(t, f, "dispatch.cadence.set", "dispatch cadence · 8 cadences")
}

func TestCoverageRetentionWritesNothingWhenNeitherDialMoves(t *testing.T) {
	f := newFakeStore()
	seedRetentionPanel(f)
	f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"90"}, "dispatch_cadence_multiple": {"4"},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a submit that moved no dial recorded %v, want nothing", got)
	}
}

// Below the floor is raised, not refused, so the row carries what the store took (ADR-0081).

func TestCoverageRetentionRecordsTheClampedValueAndNotTheTypedOne(t *testing.T) {
	f := newFakeStore()
	seedRetentionPanel(f)
	f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"1"}, "dispatch_cadence_multiple": {"1"},
	}).Body.Close()

	if s := actSubjects(t, f, "observation.currency.set")[0]; s != "observation currency · 2 days" {
		t.Errorf("observation subject = %q, want the floor the store took", s)
	}
	if s := actSubjects(t, f, "dispatch.cadence.set")[0]; s != "dispatch cadence · 2 cadences" {
		t.Errorf("dispatch subject = %q, want the floor the store took", s)
	}
}

// The Subject cell carries the dial name, so the shared label reads once and not twice (§2.1).

func TestTheDialMovedClassesShareOneLabelAndKeepDistinctTokens(t *testing.T) {
	dials := []act.Act{
		act.ZoneCadenceSet{}, act.DNSCadenceSet{}, act.TranscriptCurrencySet{},
		act.ObservationCurrencySet{}, act.DispatchCadenceSet{}, act.AddressCapSet{},
	}
	seen := map[string]bool{}
	for _, a := range dials {
		if got := a.Label(); got != "Dial moved" {
			t.Errorf("%T renders %q, want Dial moved", a, got)
		}
		if seen[a.Class()] {
			t.Errorf("%T repeats the stored token %q", a, a.Class())
		}
		seen[a.Class()] = true
	}
	// update.check.moved shares the payload and not the label, so it is held apart (§2.1).
	if got := (act.UpdateCheckMoved{}).Label(); got != "Update check moved" {
		t.Errorf("update.check.moved renders %q", got)
	}
}

// Zero is the unbounded stop on all three retention dials, never a same-day retirement (ADR-0081).

func TestAnUnboundedRetentionDialDoesNotRecordAsZeroDays(t *testing.T) {
	f := newFakeStore()
	seedRetentionPanel(f)
	f.retention.TranscriptCurrencyDays = 30
	f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/settings/retention", url.Values{
		"transcript_currency_days": {"0"},
	}).Body.Close()
	wantOneAct(t, f, "transcript.currency.set", "transcript currency · keep everything")

	f.acts = nil
	postForm(t, ac, base+"/coverage/retention", url.Values{
		"observation_currency_days": {"0"}, "dispatch_cadence_multiple": {"0"},
	}).Body.Close()
	if s := actSubjects(t, f, "observation.currency.set")[0]; s != "observation currency · keep everything" {
		t.Errorf("observation subject = %q", s)
	}
	if s := actSubjects(t, f, "dispatch.cadence.set")[0]; s != "dispatch cadence · keep everything" {
		t.Errorf("dispatch subject = %q", s)
	}
}

func declareTestChannel(t *testing.T, ac *http.Client, base, rawURL, secret string) {
	t.Helper()
	postForm(t, ac, base+"/settings/channels", url.Values{
		"url": {rawURL}, "drift": {"on"}, "secret": {secret},
	}).Body.Close()
}

func TestTheThreeChannelActsRecordTheDeliveryEndpoint(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	declareTestChannel(t, ac, base, "https://hooks.example.com/services/T0/B0", "first")
	wantOneAct(t, f, "channel.declared", "hooks.example.com/services/T0/B0")

	id := itoa(f.channels[0].id)
	f.acts = nil
	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {id}, "url": {"https://hooks.example.com/services/T9/B9"}, "clock": {"on"},
	}).Body.Close()
	wantOneAct(t, f, "channel.updated", "hooks.example.com/services/T9/B9")

	f.acts = nil
	postForm(t, ac, base+"/settings/channels/delete", url.Values{"id": {id}}).Body.Close()
	// Resolved before the delete, because the row that holds the endpoint is gone after it (§4.2).
	wantOneAct(t, f, "channel.withdrawn", "hooks.example.com/services/T9/B9")
}

// The variant has one field and it is the endpoint, so a secret has nowhere to land (§4.2).

func TestNoChannelSecretReachesASubjectPayload(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	const secret = "correct-horse-battery-staple"

	declareTestChannel(t, ac, base, "https://hooks.example.com/services/T0/B0", secret)
	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {itoa(f.channels[0].id)}, "url": {"https://hooks.example.com/services/T0/B0"},
		"clock": {"on"}, "secret": {secret},
	}).Body.Close()

	if len(f.acts) != 2 {
		t.Fatalf("wrote %d acts, want 2", len(f.acts))
	}
	for _, a := range f.acts {
		if strings.Contains(string(a.Subject), secret) {
			t.Errorf("%s carried the secret in its subject: %s", a.Action, a.Subject)
		}
	}
}

func TestAnUpdateOrDeleteOfAnAbsentChannelRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/channels/update", url.Values{
		"id": {"4242"}, "url": {"https://hooks.example.com/x"}, "drift": {"on"},
	}).Body.Close()
	postForm(t, ac, base+"/settings/channels/delete", url.Values{"id": {"4242"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an absent channel recorded %v, want nothing", got)
	}
}

func TestTheThreeIntegrationActsRecordTheSlugAndItsChannel(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"pagerduty"}}).Body.Close()
	wantOneAct(t, f, "integration.installed", "pagerduty")

	declareTestChannel(t, ac, base, "https://events.pagerduty.com/x", "")
	f.acts = nil
	postForm(t, ac, base+"/settings/integrations/channel", url.Values{
		"id": {"pagerduty"}, "channel": {itoa(f.channels[0].id)},
	}).Body.Close()
	wantOneAct(t, f, "integration.channel.bound", "pagerduty · events.pagerduty.com/x")

	f.acts = nil
	postForm(t, ac, base+"/settings/integrations/remove", url.Values{"id": {"pagerduty"}}).Body.Close()
	wantOneAct(t, f, "integration.removed", "pagerduty")
}

// "Not connected" is a real option on the select, and §4.4 leaves no Subject cell empty.

func TestUnbindingAnIntegrationChannelRecordsTheNamedAbsence(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	postForm(t, ac, base+"/settings/integrations/install", url.Values{"slug": {"slack"}}).Body.Close()
	f.acts = nil

	postForm(t, ac, base+"/settings/integrations/channel", url.Values{
		"id": {"slack"}, "channel": {""},
	}).Body.Close()

	wantOneAct(t, f, "integration.channel.bound", "slack · not connected")
}

// The DELETE and the UPDATE are both WHERE slug, so an uninstalled slug moves nothing (§7.6).

func TestAnUninstalledIntegrationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	postForm(t, ac, base+"/settings/integrations/remove", url.Values{"id": {"jira"}}).Body.Close()
	postForm(t, ac, base+"/settings/integrations/channel", url.Values{
		"id": {"jira"}, "channel": {""},
	}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an uninstalled integration recorded %v, want nothing", got)
	}
}
