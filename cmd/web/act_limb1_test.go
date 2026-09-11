package main

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Limb 1, first half — the estate's declaration (spec §1.1, §2.1). The gate proves a Record is
// reachable; it proves nothing about ordering, cardinality or content, so these do (spec §7.5).

func actClasses(f *fakeStore) []string {
	out := []string{}
	for _, a := range f.acts {
		out = append(out, a.Action)
	}
	return out
}

func actSubjects(t *testing.T, f *fakeStore, class string) []string {
	t.Helper()
	out := []string{}
	for _, a := range f.acts {
		if a.Action != class {
			continue
		}
		_, decoded := decodeAct(t, a)
		out = append(out, decoded.Subject())
	}
	return out
}

func wantOneAct(t *testing.T, f *fakeStore, class, subject string) {
	t.Helper()
	if got := actClasses(f); len(got) != 1 || got[0] != class {
		t.Fatalf("recorded %v, want one %s", got, class)
	}
	if got := actSubjects(t, f, class)[0]; got != subject {
		t.Errorf("subject = %q, want %q", got, subject)
	}
}

func adminSession(t *testing.T, f *fakeStore) (string, *http.Client) {
	t.Helper()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	return base, login(t, base, "admin", "hunter2hunter2")
}

func TestCustodyMoveRecordsTheScopeAndItsDisposition(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	declare(t, ac, base, "name", "example.com").Body.Close()
	id := f.seeds[0].ID
	f.acts = nil

	setCustody(t, ac, base, id, true).Body.Close()
	wantOneAct(t, f, "seed.custody.moved", "example.com · custody extended")

	f.acts = nil
	setCustody(t, ac, base, id, false).Body.Close()
	wantOneAct(t, f, "seed.custody.moved", "example.com · custody ended")
}

func TestCustodyMoveOnAnUnknownScopeRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	setCustody(t, ac, base, 4242, true).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown id recorded %v, want nothing", got)
	}
}

// A phantom row can never be retracted, so a no-op mutation may not write one (spec §7.6).

func TestCustodyMoveOnAnAddressScopeRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	declare(t, ac, base, "address", "10.0.0.0/24").Body.Close()
	id := f.seeds[0].ID
	f.acts = nil

	// A custody extension is a name scope's property, so the UPDATE matches no row.
	setCustody(t, ac, base, id, true).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an address scope recorded %v, want nothing", got)
	}
}

func TestZoneUploadRecordsOneActPerApex(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	declare(t, ac, base, "name", "example.com").Body.Close()
	seedID := f.seeds[0].ID
	f.acts = nil

	uploadZone(t, ac, base, seedID, testZone).Body.Close()

	wantOneAct(t, f, "zone.declared", "example.com")
}

func TestRefusedZoneUploadRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	declare(t, ac, base, "name", "example.com").Body.Close()
	seedID := f.seeds[0].ID
	f.acts = nil

	// The apex sits outside every declared name scope, so the file is refused (spec §7.6).
	uploadZone(t, ac, base, seedID, strings.ReplaceAll(testZone, "example.com", "other.test")).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused upload recorded %v, want nothing", got)
	}
}

func TestExclusionDeclarationRecordsTheNormalizedScope(t *testing.T) {
	for _, tc := range []struct{ kind, value, want string }{
		{"name", "API.Example.com", "name api.example.com"},
		{"subtree", "internal.example.com", "subtree internal.example.com"},
		{"address", "198.51.100.7", "address 198.51.100.7/32"},
	} {
		t.Run(tc.kind+" "+tc.value, func(t *testing.T) {
			f := newFakeStore()
			base, ac := adminSession(t, f)
			f.acts = nil

			exclude(t, ac, base, tc.kind, tc.value).Body.Close()

			wantOneAct(t, f, "exclusion.declared", tc.want)
		})
	}
}

func TestRefusedExclusionDeclarationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	exclude(t, ac, base, "address", "not-a-scope").Body.Close()
	exclude(t, ac, base, "kittens", "example.com").Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("refused declarations recorded %v, want nothing", got)
	}
}

func TestExclusionLiftRendersItsSubjectAfterTheRowIsGone(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	exclude(t, ac, base, "address", "203.0.113.0/24").Body.Close()
	id := f.exclusions[0].ID
	f.acts = nil

	unexclude(t, ac, base, id).Body.Close()

	wantOneAct(t, f, "exclusion.lifted", "address 203.0.113.0/24")
	if len(f.exclusions) != 0 {
		t.Fatalf("the exclusion row survived the lift: %+v", f.exclusions)
	}
	// The subject is a stored value, so the render reads no exclusion store (spec §4.2, §4.3).
	page := settingsTabBody(t, ac, base, "audit")
	if !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("the audit tab lost the lifted scope; body: %s", page)
	}
	if !strings.Contains(page, "Exclusion lifted") {
		t.Errorf("the audit tab is missing the Action cell; body: %s", page)
	}
}

func TestLiftOfAnUnknownExclusionRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	unexclude(t, ac, base, 4242).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown id recorded %v, want nothing", got)
	}
}

func TestColdMoveRecordsTheScopeAndItsDisposition(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	declare(t, ac, base, "address", "10.0.0.0/24").Body.Close()
	id := f.seeds[0].ID
	f.acts = nil

	setColdScope(t, ac, base, id, true).Body.Close()
	wantOneAct(t, f, "cold.moved", "10.0.0.0/24 · cold opt-in")

	f.acts = nil
	setColdScope(t, ac, base, id, false).Body.Close()
	wantOneAct(t, f, "cold.moved", "10.0.0.0/24 · cold opt-out")
}

func TestARepeatColdMoveRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	declare(t, ac, base, "address", "10.0.0.0/24").Body.Close()
	id := f.seeds[0].ID

	// An opt-out on an unenrolled scope withdraws nothing (spec §7.6).
	f.acts = nil
	setColdScope(t, ac, base, id, false).Body.Close()
	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unenrolled opt-out recorded %v, want nothing", got)
	}

	// A repeat opt-in hits ON CONFLICT DO NOTHING and enrols nothing (spec §7.6).
	setColdScope(t, ac, base, id, true).Body.Close()
	f.acts = nil
	setColdScope(t, ac, base, id, true).Body.Close()
	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a repeat opt-in recorded %v, want nothing", got)
	}
}

func TestColdMoveOnAnUnknownScopeRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	setColdScope(t, ac, base, 4242, true).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown id recorded %v, want nothing", got)
	}
}

func TestVantageDeclarationRecordsTheEndpoint(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	provision(t, ac, base, "edge-01.example.com", "22", "probe").Body.Close()

	wantOneAct(t, f, "vantage.declared", "probe@edge-01.example.com:22")
}

func TestRefusedVantageDeclarationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	// A vantage with no resolver dispatches nothing, so creation is refused (ADR-0202).
	provisionWithResolver(t, ac, base, "edge-01.example.com", "22", "probe", "").Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a refused provision recorded %v, want nothing", got)
	}
}

func TestResolverSetRecordsTheResolver(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	provision(t, ac, base, "edge-01.example.com", "22", "probe").Body.Close()
	id := f.vantages[0].ID
	f.acts = nil

	setResolver(t, ac, base, strconv.FormatInt(id, 10), "1.1.1.1:53").Body.Close()

	wantOneAct(t, f, "vantage.resolver.set", "1.1.1.1:53")
}

func newSchedule(t *testing.T, c *http.Client, base, name string) {
	t.Helper()
	postForm(t, c, base+"/reports/schedule/new", url.Values{
		"step": {"3"}, "name": {name}, "sections": {"kpis"},
		"cad": {reportDefaultCad}, "channel": {"0"},
	}).Body.Close()
}

func TestScheduleActsRecordTheScheduleName(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	newSchedule(t, ac, base, "Weekly exposure")
	wantOneAct(t, f, "schedule.declared", "Weekly exposure")

	id := f.reportSchedules[0].ID
	f.acts = nil
	postForm(t, ac, base+"/reports/schedule/"+itoa(id)+"/edit", url.Values{
		"step": {"3"}, "id": {itoa(id)}, "name": {"Monthly exposure"}, "sections": {"kpis"},
		"cad": {reportDefaultCad}, "channel": {"0"},
	}).Body.Close()
	wantOneAct(t, f, "schedule.edited", "Monthly exposure")

	f.acts = nil
	postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {itoa(id)}}).Body.Close()
	wantOneAct(t, f, "schedule.withdrawn", "Monthly exposure")
	if len(f.reportSchedules) != 0 {
		t.Fatalf("the schedule row survived its withdrawal: %+v", f.reportSchedules)
	}
}

func TestWithdrawalOfAnUnknownScheduleRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	postForm(t, ac, base+"/reports/schedule/delete", url.Values{"id": {"4242"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown id recorded %v, want nothing", got)
	}
}

func TestAnnotationActsRecordThePairAndNoProse(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	lameName(t, f, "lame.example.com")
	f.acts = nil

	const reason = "Accepted: the delegation is being retired under OPS-1."
	annotate(t, ac, base, "lame.example.com", "lame-delegation", reason).Body.Close()

	wantOneAct(t, f, "annotation.declared", "lame.example.com · lame-delegation")
	// Prose is #127 §5's one reopening condition, so no variant may carry it (spec §8 · D.6).
	if strings.Contains(string(f.acts[0].Subject), "OPS-1") {
		t.Errorf("the annotation act carried its reason: %s", f.acts[0].Subject)
	}

	id := f.annotations[0].ID
	f.acts = nil
	withdrawAnno(t, ac, base, id).Body.Close()

	wantOneAct(t, f, "annotation.withdrawn", "lame.example.com · lame-delegation")
	if len(f.annotations) != 0 {
		t.Fatalf("the annotation row survived its withdrawal: %+v", f.annotations)
	}
}

func TestRefusedAnnotationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	annotate(t, ac, base, "lame.example.com", "lame-delegation", "").Body.Close()
	annotate(t, ac, base, "", "lame-delegation", "accepted").Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("refused annotations recorded %v, want nothing", got)
	}
}

func TestWithdrawalOfAnUnknownAnnotationRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	withdrawAnno(t, ac, base, 4242).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("an unknown id recorded %v, want nothing", got)
	}
}

func TestProposalConfirmRecordsTheMaskedScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	f.acts = nil

	postForm(t, ac, base+"/proposals/confirm", url.Values{"id": {itoa(f.proposals[0].ID)}}).Body.Close()

	wantOneAct(t, f, "proposal.confirmed", "203.0.113.0/24")
}

func TestDeclineRecordsOneActPerSubject(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	f.acts = nil

	var ids []string
	for _, p := range f.proposals {
		ids = append(ids, itoa(p.ID))
	}
	postForm(t, ac, base+"/proposals/decline", url.Values{"ids": ids}).Body.Close()

	// One Act per subject, never one per request (spec §7.6, ruling 3).
	want := []string{"address 203.0.113.0/24", "address 198.51.100.8/29"}
	if got := actSubjects(t, f, "proposal.declined"); !reflect.DeepEqual(got, want) {
		t.Fatalf("recorded subjects = %v, want %v", got, want)
	}
}

func TestAPartialDeclineBatchRecordsOnlyTheAppliedSubjects(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	// The loop bails mid-batch and leaves the applied part committed (spec §7.6, ruling 3).
	f.addrExclFailScope = "198.51.100.8/29"
	f.acts = nil

	var ids []string
	for _, p := range f.proposals {
		ids = append(ids, itoa(p.ID))
	}
	postForm(t, ac, base+"/proposals/decline", url.Values{"ids": ids}).Body.Close()

	want := []string{"address 203.0.113.0/24"}
	if got := actSubjects(t, f, "proposal.declined"); !reflect.DeepEqual(got, want) {
		t.Fatalf("a partial batch recorded %v, want %v", got, want)
	}
}

func TestUndoDeclineRecordsTheLift(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	declined := f.proposals[0]
	declineOne(t, ac, base, declined.ID)
	f.acts = nil

	postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(declined.ID)}}).Body.Close()

	wantOneAct(t, f, "proposal.decline.undone", "address 203.0.113.0/24")
}

func TestUndoOfANonDeclinedProposalRecordsNothing(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithProposer(t, f, &fakeProposer{candidates: twoCandidates()})
	ac := login(t, base, "admin", "hunter2hunter2")
	lookup(t, ac, base, "Example").Body.Close()
	f.acts = nil

	// A pending Proposal is no decline to reverse, so nothing moves (ADR-0022).
	postForm(t, ac, base+"/proposals/undo-decline", url.Values{"id": {itoa(f.proposals[0].ID)}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a pending proposal recorded %v, want nothing", got)
	}
}

func TestFrequencyMoveRecordsThePortAndItsDisposition(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	for _, tc := range []struct{ action, port, want string }{
		{"add", "8443", "port 8443 · add"},
		{"remove", "443", "port 443 · remove"},
		{"reset", "8443", "port 8443 · reset"},
	} {
		f.acts = nil
		editFreq(t, ac, base, tc.action, tc.port).Body.Close()
		wantOneAct(t, f, "frequency.moved", tc.want)
	}
}

func TestAResetOfAnUneditedPortRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	// The DELETE matches no row, so the reset moved no frequency (spec §7.6).
	editFreq(t, ac, base, "reset", "8443").Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("a reset of an unedited port recorded %v, want nothing", got)
	}
}

func TestRefusedFrequencyMoveRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	editFreq(t, ac, base, "add", "70000").Body.Close()
	editFreq(t, ac, base, "kittens", "8443").Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("refused moves recorded %v, want nothing", got)
	}
}

func TestSourceMoveRecordsTheSlugAndItsDisposition(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	toggleSourceReq(t, ac, base, "arin", "false").Body.Close()
	wantOneAct(t, f, "source.moved", "arin · disabled")

	// Two routes, one class: the console form and the settings form move the same dial (§2.1).
	f.acts = nil
	postForm(t, ac, base+"/settings/sources", url.Values{"id": {"arin"}, "enable": {"true"}}).Body.Close()
	wantOneAct(t, f, "source.moved", "arin · enabled")
}

func TestRefusedSourceMoveRecordsNothing(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.acts = nil

	toggleSourceReq(t, ac, base, "no-such-source", "true").Body.Close()
	postForm(t, ac, base+"/settings/sources", url.Values{"id": {"arin"}, "enable": {"kittens"}}).Body.Close()

	if got := actClasses(f); len(got) != 0 {
		t.Fatalf("refused moves recorded %v, want nothing", got)
	}
}

// #11's own first example, which #127 §7 listed as unanswerable in v1 (spec §8 · A.1).

func TestTheAuditTabAnswersWhoChangedTheSeedList(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)

	exclude(t, ac, base, "address", "10.1.2.0/24").Body.Close()

	page := settingsTabBody(t, ac, base, "audit")
	for _, want := range []string{"@admin", "Exclusion declared", "address 10.1.2.0/24"} {
		if !strings.Contains(page, want) {
			t.Errorf("the audit tab is missing %q; body: %s", want, page)
		}
	}
}

// The recorder is not atomic with the act, so a failed insert leaves the act standing (spec §7.6).

func TestAFailedRecordLeavesTheActApplied(t *testing.T) {
	f := newFakeStore()
	base, ac := adminSession(t, f)
	f.actErr = errors.New("insert act: connection reset")

	exclude(t, ac, base, "address", "203.0.113.0/24").Body.Close()

	if len(f.exclusions) != 1 {
		t.Fatalf("exclusions = %d, want 1; a failed Record must not undo the act", len(f.exclusions))
	}
	if len(f.acts) != 0 {
		t.Fatalf("a failed insert stored %d acts, want 0", len(f.acts))
	}
}
