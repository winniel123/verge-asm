package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// fakeTrigger stands in for the queue Dispatcher behind the on-demand trigger
// (#252). It records the kinds it was asked to dispatch and returns a scripted
// job count or error, so a handler test asserts the enqueue happened without a
// live Postgres or a running worker.
type fakeTrigger struct {
	calls   []string
	jobs    int
	err     error
	refused map[string]bool // kinds that answer with the disabled-scan refusal
}

func (t *fakeTrigger) Trigger(_ context.Context, kind string) (int, error) {
	t.calls = append(t.calls, kind)
	if t.refused[kind] {
		return 0, fmt.Errorf("queue: %s Scan is disabled", kind)
	}
	if t.err != nil {
		return 0, t.err
	}
	return t.jobs, nil
}

// startWithTrigger is start with a scan-trigger seam wired in, the way main.go
// wires the real Dispatcher over the pool. Redirects are not followed so the
// test reads the 303 and its destination.
func startWithTrigger(t *testing.T, f *fakeStore, trig scanTrigger) string {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	srv.dispatcher = trig
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

// An admin's trigger enqueues the same dispatch path the CLI trigger uses and
// lands back on the monitor with a receipt (AC: admin-only on-demand trigger,
// enqueues fanOut, paired with the running indicator).
func TestTriggerScanAdminEnqueues(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 3}

	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=scans"
	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"hot"}, backField: {from}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("trigger: status = %d, want 303 (body: %s)", resp.StatusCode, body(t, resp))
	}
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if loc != from {
		t.Fatalf("trigger landed at %q, want the submitting URL %q", loc, from)
	}
	if len(trig.calls) != 1 || trig.calls[0] != "hot" {
		t.Fatalf("dispatcher calls = %v, want one hot fan-out", trig.calls)
	}

	// The receipt is the single-consume flash toast, fired on the landing render.
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "hot scan dispatched") || !strings.Contains(page, "3 jobs fanned out") {
		t.Errorf("trigger receipt missing the job count; body: %s", page)
	}
}

// The trigger joins the ADR-0130 §3 submitting-URL carrier (ticket #1087). Pressing
// Run now from a filtered or scrolled page lands the operator back on that exact URL,
// so the scroll key ticket #970 stashes on submit hits on the landing. A form that
// carries no field falls back to the scans section.
func TestTriggerScanLandsBackOnTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := startWithTrigger(t, f, &fakeTrigger{jobs: 1})
	ac := login(t, base, "admin", "hunter2hunter2")

	// The panel stamps the page's own URL into the form, so the field is real markup.
	page := getBody(t, ac, base+"/scans", http.StatusOK)
	if !strings.Contains(page, `action="/scans/trigger"`) ||
		!strings.Contains(page, `name="return" value="/scans"`) {
		t.Fatalf("the trigger form does not carry the submitting URL; body: %s", page)
	}

	// The last case is the one the stop and terminate siblings do NOT keep: a trigger
	// answers no confirm, so an unanswered ?stop= opener stays on the destination
	// rather than being stripped off it and moving the scroll key.
	for _, from := range []string{"/scans", "/settings?tab=scans", "/scans?stop=12"} {
		resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"dns"}, backField: {from}})
		got := resp.Header.Get("Location")
		resp.Body.Close()
		if got != from {
			t.Errorf("trigger from %q landed at %q, want the submitting URL", from, got)
		}
	}

	// A hostile field never reaches the Location header; the act still lands somewhere
	// this server serves.
	for _, hostile := range []string{"https://evil.example/x", "//evil.example/x", `/\evil.example`} {
		resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"dns"}, backField: {hostile}})
		got := resp.Header.Get("Location")
		resp.Body.Close()
		if got != "/settings?tab=scans" {
			t.Errorf("trigger with %q landed at %q, want the fallback /settings?tab=scans", hostile, got)
		}
	}
}

// A viewer cannot trigger a scan — the endpoint is admin-only, like /sources
// toggling (AC: admin-only, respecting the existing admin gating).
func TestTriggerScanViewerRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 3}

	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"hot"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer trigger: status = %d, want 403", resp.StatusCode)
	}
	if len(trig.calls) != 0 {
		t.Fatalf("a viewer's request reached the dispatcher: %v", trig.calls)
	}
}

// A disabled scan (the shipped-off cold tier) is refused, not dispatched — the
// guardrail ADR-0044 forbids as an ad-hoc one-off (AC: honour the disabled-scan
// refusal). The seed cold Scan ships disabled.
func TestTriggerScanDisabledRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 3}

	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=scans"
	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"cold"}, backField: {from}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != from {
		t.Fatalf("cold trigger: status=%d loc=%q, want a 303 back to %q", resp.StatusCode, loc, from)
	}
	// The refusal is authoritative before the dispatcher: a disabled scan never
	// reaches the fan-out.
	if len(trig.calls) != 0 {
		t.Fatalf("a disabled scan reached the dispatcher: %v", trig.calls)
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "The cold scan is disabled") {
		t.Errorf("disabled receipt missing; body: %s", page)
	}
}

// A kind already in flight is not dispatched again — the overlap guard against
// accidental repeated kick-offs (AC: rate/overlap protection).
func TestTriggerScanOverlapRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0), // hot is active
	}
	trig := &fakeTrigger{jobs: 3}

	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=scans"
	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"hot"}, backField: {from}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != from {
		t.Fatalf("overlapping trigger: status=%d loc=%q, want a 303 back to %q", resp.StatusCode, loc, from)
	}
	if len(trig.calls) != 0 {
		t.Fatalf("an in-flight kind was dispatched again: %v", trig.calls)
	}
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "A hot scan is already in flight") {
		t.Errorf("overlap receipt missing; body: %s", page)
	}
}

// An unknown kind dispatches nothing rather than 500ing — a hand-crafted POST is
// refused at the door.
func TestTriggerScanUnknownKind(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 3}

	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=scans"
	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"nonsense"}, backField: {from}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != from {
		t.Fatalf("unknown trigger: status=%d loc=%q, want a 303 back to %q", resp.StatusCode, loc, from)
	}
	if len(trig.calls) != 0 {
		t.Fatalf("an unknown kind reached the dispatcher: %v", trig.calls)
	}
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "not one this deployment runs") {
		t.Errorf("unknown-kind receipt missing; body: %s", page)
	}
}

// A fresh dispatch that fanned out zero jobs (an empty scope — no seed or vantage
// covers the scan) is reported honestly as enqueued-nothing, never as a false
// "already dispatched" overlap.
func TestTriggerScanEmptyFanOut(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 0} // dispatched, but nothing to enqueue

	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	const from = "/settings?tab=scans"
	resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"dns"}, backField: {from}})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || loc != from {
		t.Fatalf("empty fan-out: status=%d loc=%q, want a 303 back to %q", resp.StatusCode, loc, from)
	}
	if len(trig.calls) != 1 {
		t.Fatalf("an empty fan-out should still have dispatched once: %v", trig.calls)
	}
	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "enqueued no jobs") {
		t.Errorf("empty fan-out receipt missing; body: %s", page)
	}
}

// A failure reading the scan list degrades the panel to absent rather than 500ing
// the read-only monitor an operator depends on (the graceful-degradation the
// scansPage comment promises).
func TestScansPageTriggerPanelDegrades(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.listScansErr = errors.New("scan: connection refused")

	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	// The monitor still renders — the panel is simply absent.
	page := getBody(t, ac, base+"/scans", http.StatusOK)
	if strings.Contains(page, `action="/scans/trigger"`) {
		t.Errorf("panel should be absent when its read fails; body: %s", page)
	}
	if !strings.Contains(page, "No scan running") {
		t.Errorf("the monitor itself should still render; body: %s", page)
	}
}

// The admin sees a trigger control for each enabled scan, and the disabled cold
// tier is shown as not-triggerable rather than hidden (AC: the trigger control,
// paired with the monitor; honour the disabled refusal transparently).
func TestScansPageAdminSeesTriggerPanel(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	if !strings.Contains(page, `action="/scans/trigger"`) {
		t.Errorf("admin trigger form missing; body: %s", page)
	}
	// An enabled scan carries a submit control; the disabled cold tier does not.
	if !strings.Contains(page, `value="hot"`) {
		t.Errorf("enabled hot scan has no trigger; body: %s", page)
	}
	if !strings.Contains(page, "cold") || !strings.Contains(page, "disabled") {
		t.Errorf("disabled cold scan should show as not-triggerable; body: %s", page)
	}
}

// A viewer does not see the trigger control at all — it is an admin act, and the
// monitor stays read-only for a viewer (AC: admin-only).
func TestScansPageViewerNoTriggerPanel(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")

	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "viewer", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	if strings.Contains(page, `action="/scans/trigger"`) {
		t.Errorf("a viewer must not see the trigger form; body: %s", page)
	}
}

// An active dispatch's kind is reported so the panel can refuse an overlapping
// trigger and mark the running scan — the seam the overlap guard rests on.
func TestActiveDispatchKinds(t *testing.T) {
	f := newFakeStore()
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0), // active
		progressRow(9, "dns", tick, 2, 0, 0, 2, 0, 0),  // complete
	}
	srv := newServer(f, testKey, "", fixedClock())

	got, err := srv.activeDispatchKinds(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !got["hot"] {
		t.Errorf("hot is in flight but not reported active: %v", got)
	}
	if got["dns"] {
		t.Errorf("dns is complete but reported active: %v", got)
	}
}
