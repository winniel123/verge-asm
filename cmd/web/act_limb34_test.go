package main

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
	"github.com/winniel123/verge-asm/internal/queue"
)

// Limb 3 — directing the instance to act on the network (spec §1.3) — and limb 4 — a disclosure
// from a corpus the model seals (§1.4). The gate proves a Record is reachable; it proves nothing
// about ordering, cardinality or content, so these do (spec §7.5).

func triggerStore(t *testing.T, kind string, enabled bool) *fakeStore {
	t.Helper()
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.scans = append(f.scans, db.Scan{ID: 7, Kind: kind, Enabled: enabled, CadenceSeconds: 3600})
	return f
}

func triggerAs(t *testing.T, f *fakeStore, trig scanTrigger, path, kind string) {
	t.Helper()
	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil
	postForm(t, ac, base+path, url.Values{"kind": {kind}}).Body.Close()
}

func wantNoActs(t *testing.T, f *fakeStore, why string) {
	t.Helper()
	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("%s recorded %v, want nothing", why, got)
	}
}

func TestATriggeredScanRecordsItsProfile(t *testing.T) {
	f := triggerStore(t, "hot", true)
	triggerAs(t, f, &fakeTrigger{jobs: 3}, "/scans/trigger", "hot")

	wantOneAct(t, f, "scan.triggered", "hot")
}

// The boundary is whether Trigger was reached, never how many jobs came out (spec §7.6 ruling 7).

func TestAZeroJobsTriggerStillRecords(t *testing.T) {
	f := triggerStore(t, "hot", true)
	triggerAs(t, f, &fakeTrigger{jobs: 0}, "/scans/trigger", "hot")

	wantOneAct(t, f, "scan.triggered", "hot")
}

// A skipped tick reached Trigger too, so the same boundary decides it (spec §7.6 ruling 7).

func TestASkippedTickStillRecords(t *testing.T) {
	for _, skip := range []queue.SkipReason{queue.SkipTickDispatched, queue.SkipCadenceLag} {
		f := triggerStore(t, "hot", true)
		triggerAs(t, f, &fakeTrigger{skip: skip}, "/scans/trigger", "hot")

		wantOneAct(t, f, "scan.triggered", "hot")
	}
}

func TestAnUnknownScanRecordsNothing(t *testing.T) {
	f := triggerStore(t, "hot", true)
	triggerAs(t, f, &fakeTrigger{jobs: 3}, "/scans/trigger", "nonesuch")

	wantNoActs(t, f, "an unknown scan")
}

func TestADisabledScanRecordsNothing(t *testing.T) {
	f := triggerStore(t, "cold", false)
	triggerAs(t, f, &fakeTrigger{jobs: 3}, "/scans/trigger", "cold")

	wantNoActs(t, f, "a disabled scan")
}

func TestAnInFlightScanRecordsNothing(t *testing.T) {
	f := triggerStore(t, "hot", true)
	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{progressRow(1408, "hot", tick, 6, 2, 1, 3, 0, 0)}
	triggerAs(t, f, &fakeTrigger{jobs: 3}, "/scans/trigger", "hot")

	wantNoActs(t, f, "an already in-flight scan")
}

// One helper serves both routes, so the wizard's own class is what proves they are two acts.

func TestFinishingOnboardingRecordsItsOwnClass(t *testing.T) {
	f := triggerStore(t, "hot", true)
	triggerAs(t, f, &fakeTrigger{jobs: 3}, "/onboarding/finish", "hot")

	wantOneAct(t, f, "onboarding.finished", "hot")
}

func TestStoppingADispatchRecordsItsProfile(t *testing.T) {
	f := activeDispatchStore(t, 2, 1)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/scans/stop", url.Values{"id": {"1408"}}).Body.Close()

	wantOneAct(t, f, "scan.stopped", "dispatch 1408 · standard")
}

func TestTerminatingADispatchRecordsItsProfile(t *testing.T) {
	f := activeDispatchStore(t, 2, 1)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/scans/terminate", url.Values{"id": {"1408"}}).Body.Close()

	wantOneAct(t, f, "scan.terminated", "dispatch 1408 · standard")
}

// A concluded dispatch is not in the active read, so nothing was stopped (spec §7.6 ruling 2).

func TestStoppingAConcludedDispatchRecordsNothing(t *testing.T) {
	f := activeDispatchStore(t, 0, 0)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	for _, path := range []string{"/scans/stop", "/scans/terminate"} {
		postForm(t, ac, base+path, url.Values{"id": {"1408"}}).Body.Close()
	}

	wantNoActs(t, f, "a concluded dispatch")
}

func oneAttempt(cands []proposer.Candidate) *fakeProposer {
	return &fakeProposer{candidates: cands, attempts: []proposer.Attempt{{SourceSlug: "crtsh"}}}
}

func lookupAs(t *testing.T, f *fakeStore, p proposerRunner, path, org string) {
	t.Helper()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, p)
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil
	postForm(t, ac, base+path, url.Values{"org": {org}}).Body.Close()
}

func TestARegistryLookupRecordsTheTerm(t *testing.T) {
	f := newFakeStore()
	lookupAs(t, f, oneAttempt(twoCandidates()), "/proposals", "Example Holdings Ltd")

	wantOneAct(t, f, "proposal.queried", "Example Holdings Ltd")
}

// The registries answered nothing, and they were still reached (spec §1.3).

func TestALookupThatMatchesNothingStillRecords(t *testing.T) {
	f := newFakeStore()
	lookupAs(t, f, oneAttempt(nil), "/proposals/search", "no-such-org-anywhere")

	wantOneAct(t, f, "proposal.queried", "no-such-org-anywhere")
}

func TestALookupWithNoTermRecordsNothing(t *testing.T) {
	f := newFakeStore()
	p := oneAttempt(twoCandidates())
	lookupAs(t, f, p, "/proposals", "   ")

	wantNoActs(t, f, "an empty lookup")
	if p.calls != 0 {
		t.Errorf("an empty lookup reached the registries %d times, want 0", p.calls)
	}
}

func TestAChannelTestRecordsItsEndpoint(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 5, "https://hooks.example.com/services/T0/B0", "sign-me")
	base := startWithChannelSender(t, f, &fakeChannelSender{status: http.StatusOK})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"5"}}).Body.Close()

	wantOneAct(t, f, "channel.tested", "hooks.example.com/services/T0/B0")
}

// An all-disabled registry set attempts nobody, so no limb-3 act happened (spec §1.3).

func TestALookupWithEveryRegistryDisabledRecordsNothing(t *testing.T) {
	f := newFakeStore()
	lookupAs(t, f, &fakeProposer{}, "/proposals", "Example Holdings Ltd")

	wantNoActs(t, f, "a lookup with no enabled registry")
}

// The endpoint answered, so the instance reached a third party (spec §1.3).

func TestARefusedChannelTestStillRecords(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 5, "https://hooks.example.com/services/T0/B0", "sign-me")
	base := startWithChannelSender(t, f, &fakeChannelSender{status: http.StatusInternalServerError})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"5"}}).Body.Close()

	wantOneAct(t, f, "channel.tested", "hooks.example.com/services/T0/B0")
}

// The SSRF guard refuses before a request is built, so nothing left the instance (spec §7.6).

func TestAGuardRefusedTestRecordsNothing(t *testing.T) {
	refused := errors.New("delivery: target host is not globally reachable")
	for _, tc := range []struct{ name, path, id string }{
		{"channel", "/settings/channels/test", "5"},
		{"integration", "/settings/integrations/test", "pagerduty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			skipIfIntegrationsHidden(t)
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			addFakeChannel(f, 5, "https://10.0.0.5/hook", "sign-me")
			f.integrationStates["pagerduty"] = db.IntegrationState{
				Slug: "pagerduty", State: integrationInstalled,
				ChannelID: pgtype.Int8{Int64: 5, Valid: true},
			}
			sender := &fakeChannelSender{err: refused}
			base := startWithChannelSender(t, f, sender)
			ac := login(t, base, "admin", "hunter2hunter2")
			f.acts = nil

			postForm(t, ac, base+tc.path, url.Values{"id": {tc.id}}).Body.Close()

			wantNoActs(t, f, "a guard-refused send")
		})
	}
}

// Nothing was sent, so no limb-3 act happened (spec §7.6 ruling 2).

func TestAChannelTestOnAnUnknownChannelRecordsNothing(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithChannelSender(t, f, &fakeChannelSender{status: http.StatusOK})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/channels/test", url.Values{"id": {"404"}}).Body.Close()

	wantNoActs(t, f, "an unknown channel")
}

func TestAnIntegrationTestRecordsItsSlugAndEndpoint(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addFakeChannel(f, 7, "https://events.pagerduty.com/verge", "sign-me")
	f.integrationStates["pagerduty"] = db.IntegrationState{
		Slug: "pagerduty", State: integrationInstalled,
		ChannelID: pgtype.Int8{Int64: 7, Valid: true},
	}
	base := startWithChannelSender(t, f, &fakeChannelSender{status: http.StatusOK})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/integrations/test", url.Values{"id": {"pagerduty"}}).Body.Close()

	wantOneAct(t, f, "integration.tested", "pagerduty · events.pagerduty.com/verge")
}

func TestAnIntegrationTestWithNoBoundChannelRecordsNothing(t *testing.T) {
	skipIfIntegrationsHidden(t)
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.integrationStates["pagerduty"] = db.IntegrationState{
		Slug: "pagerduty", State: integrationInstalled,
	}
	base := startWithChannelSender(t, f, &fakeChannelSender{status: http.StatusOK})
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	postForm(t, ac, base+"/settings/integrations/test", url.Values{"id": {"pagerduty"}}).Body.Close()

	wantNoActs(t, f, "an unbound integration")
}

func disclosureStore(t *testing.T, vantage string) *fakeStore {
	t.Helper()
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	job := db.ListJobsForDispatchRow{ID: 9182, Kind: "connect-outcome", State: "done", Attempt: 1, MaxAttempts: 5}
	if vantage != "" {
		job.VantageName = pgtype.Text{String: vantage, Valid: true}
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{412: {job}}
	return f
}

func TestOpeningRawOutputRecordsTheDisclosure(t *testing.T) {
	f := disclosureStore(t, "edge-01")
	seedTranscript(t, f, 9182, "connect-outcome", []byte("open=true"), []byte("probe: start"),
		[]byte(`{"kind":"connect-outcome"}`), `{"kind":"exited","code":0}`, `{}`,
		time.Second, time.Date(2026, 8, 29, 14, 22, 5, 0, time.UTC))
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	getBody(t, ac, base+"/run/412/raw?job=9182", http.StatusOK)

	wantOneAct(t, f, "transcript.disclosed", "job 9182 · run 412 · edge-01")
}

// A ct or zone job runs on no vantage, and a blank third cell would outlive the transcript (§4.3).

func TestARawOutputDisclosureNamesTheInstanceWhenNoVantageRanIt(t *testing.T) {
	f := disclosureStore(t, "")
	seedTranscript(t, f, 9182, "connect-outcome", []byte("open=true"), []byte("probe: start"),
		[]byte(`{"kind":"connect-outcome"}`), `{"kind":"exited","code":0}`, `{}`,
		time.Second, time.Date(2026, 8, 29, 14, 22, 5, 0, time.UTC))
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	getBody(t, ac, base+"/run/412/raw?job=9182", http.StatusOK)

	wantOneAct(t, f, "transcript.disclosed", "job 9182 · run 412 · local")
}

// GetTranscriptByJob keys on the job alone, so the run in the path would otherwise be a claim
// the admin chose, standing in the corpus after the transcript expires (spec §4.3).

func TestADisclosureRefusesAJobOfAnotherRun(t *testing.T) {
	f := disclosureStore(t, "edge-01")
	seedTranscript(t, f, 9182, "connect-outcome", []byte("open=true"), []byte("probe: start"),
		[]byte(`{"kind":"connect-outcome"}`), `{"kind":"exited","code":0}`, `{}`,
		time.Second, time.Date(2026, 8, 29, 14, 22, 5, 0, time.UTC))
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	getBody(t, ac, base+"/run/1/raw?job=9182", http.StatusNotFound)

	wantNoActs(t, f, "a job of another run")
}

// rawOutputPage renders a complete page on ErrNoRows and discloses nothing (spec §1.4).

func TestAMissingTranscriptRecordsNothing(t *testing.T) {
	f := disclosureStore(t, "edge-01")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	f.acts = nil

	getBody(t, ac, base+"/run/412/raw?job=9182", http.StatusOK)

	wantNoActs(t, f, "a page that opened no transcript")
}

// §1.4 worded the limb as a disclosure and refused the read register by name.

func TestTheDisclosureLabelNeverReadsAsARead(t *testing.T) {
	if got := (act.TranscriptDisclosed{}).Label(); got != "Raw output disclosed" {
		t.Errorf("limb 4 label = %q, want %q", got, "Raw output disclosed")
	}
}
