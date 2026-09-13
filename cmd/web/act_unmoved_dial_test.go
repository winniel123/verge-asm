package main

import (
	"net/url"
	"testing"
)

// A dial submitted at its current value moved nothing, so it writes no row (spec §2.2, #1901).

// The mutation stays unguarded, so the attribution columns still take the no-op (ADR-1909 §5.1).

func TestASubmitThatLeavesADialWhereItStoodRecordsNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		seed  func(*fakeStore)
		path  string
		form  url.Values
		wrote func(*fakeStore) bool
	}{
		{
			name: "zone cadence",
			seed: func(f *fakeStore) { f.zoneCadence = 24 * 86400 },
			path: "/seeds/zone/interval",
			form: url.Values{"interval_days": {"24"}},
		},
		{
			name: "dns cadence",
			seed: func(f *fakeStore) { f.dnsCadence = 6 * 86400 },
			path: "/seeds/dns/interval",
			form: url.Values{"interval_days": {"6"}},
		},
		{
			name:  "transcript currency",
			seed:  func(f *fakeStore) { f.retention.TranscriptCurrencyDays = 14 },
			path:  "/settings/retention",
			form:  url.Values{"transcript_currency_days": {"14"}},
			wrote: func(f *fakeStore) bool { return f.retention.UpdatedBy.Valid },
		},
		{
			name:  "address cap",
			seed:  func(f *fakeStore) { f.instanceConfig.SeedAddressCap = 1024 },
			path:  "/settings/address-cap",
			form:  url.Values{"address_cap": {"1024"}},
			wrote: func(f *fakeStore) bool { return f.instanceConfig.SeedAddressCapUpdatedBy.Valid },
		},
		{
			name:  "update check",
			seed:  func(f *fakeStore) { f.instanceConfig.UpdateCheckEnabled = true },
			path:  "/settings/updates/check",
			form:  url.Values{"enabled": {"true"}},
			wrote: func(f *fakeStore) bool { return f.instanceConfig.UpdateCheckUpdatedBy.Valid },
		},
		{
			name:  "api access",
			seed:  func(f *fakeStore) { f.instanceConfig.ApiEnabled = true },
			path:  "/settings/api",
			form:  url.Values{"enabled": {"true"}},
			wrote: func(f *fakeStore) bool { return f.instanceConfig.ApiUpdatedBy.Valid },
		},
		{
			name: "observation currency and dispatch cadence",
			seed: func(f *fakeStore) {
				f.retention.ObservationCurrencyDays, f.retention.DispatchCadenceMultiple = 90, 4
			},
			path:  "/coverage/retention",
			form:  url.Values{"observation_currency_days": {"90"}, "dispatch_cadence_multiple": {"4"}},
			wrote: func(f *fakeStore) bool { return f.retention.UpdatedBy.Valid },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			seedRetentionPanel(f)
			tc.seed(f)
			base, ac := adminSession(t, f)
			f.acts = nil

			postForm(t, ac, base+tc.path, tc.form).Body.Close()

			if got := actClasses(f); len(got) != 0 {
				t.Fatalf("a dial left where it stood recorded %v, want nothing", got)
			}
			if tc.wrote != nil && !tc.wrote(f) {
				t.Error("the mutation was skipped, so the attribution columns lost the no-op")
			}
		})
	}
}

// The guard reads the stored value, so a dial that did move must still record (spec §2.2).

func TestADialThatMovedOffItsStoredValueStillRecords(t *testing.T) {
	for _, tc := range []struct {
		name           string
		seed           func(*fakeStore)
		path           string
		form           url.Values
		class, subject string
	}{
		{
			name:  "zone cadence",
			seed:  func(f *fakeStore) { f.zoneCadence = 24 * 86400 },
			path:  "/seeds/zone/interval",
			form:  url.Values{"interval_days": {"25"}},
			class: "zone.cadence.set", subject: "zone scan cadence · 25 days",
		},
		{
			name:  "dns cadence",
			seed:  func(f *fakeStore) { f.dnsCadence = 6 * 86400 },
			path:  "/seeds/dns/interval",
			form:  url.Values{"interval_days": {"7"}},
			class: "dns.cadence.set", subject: "dns scan cadence · 7 days",
		},
		{
			name:  "transcript currency",
			seed:  func(f *fakeStore) { f.retention.TranscriptCurrencyDays = 14 },
			path:  "/settings/retention",
			form:  url.Values{"transcript_currency_days": {"21"}},
			class: "transcript.currency.set", subject: "transcript currency · 21 days",
		},
		{
			name:  "address cap",
			seed:  func(f *fakeStore) { f.instanceConfig.SeedAddressCap = 1024 },
			path:  "/settings/address-cap",
			form:  url.Values{"address_cap": {"2048"}},
			class: "address.cap.set", subject: "address-scope cap · 2048",
		},
		{
			name:  "update check",
			seed:  func(f *fakeStore) { f.instanceConfig.UpdateCheckEnabled = true },
			path:  "/settings/updates/check",
			form:  url.Values{"enabled": {"false"}},
			class: "update.check.moved", subject: "update check · off",
		},
		{
			name:  "api access",
			seed:  func(f *fakeStore) { f.instanceConfig.ApiEnabled = true },
			path:  "/settings/api",
			form:  url.Values{"enabled": {"false"}},
			class: "api.access.moved", subject: "API access · off",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			seedRetentionPanel(f)
			tc.seed(f)
			base, ac := adminSession(t, f)
			f.acts = nil

			postForm(t, ac, base+tc.path, tc.form).Body.Close()

			wantOneAct(t, f, tc.class, tc.subject)
		})
	}
}
