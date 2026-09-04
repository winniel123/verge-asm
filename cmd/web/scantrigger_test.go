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

type fakeTrigger struct {
	calls   []string
	jobs    int
	err     error
	refused map[string]bool
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

func startWithTrigger(t *testing.T, f *fakeStore, trig scanTrigger) string {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	srv.dispatcher = trig
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)
	return ts.URL
}

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

	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "hot scan dispatched") || !strings.Contains(page, "3 jobs fanned out") {
		t.Errorf("trigger receipt missing the job count; body: %s", page)
	}
}

func TestTriggerScanLandsBackOnTheSubmittingURL(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := startWithTrigger(t, f, &fakeTrigger{jobs: 1})
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/scans", http.StatusOK)
	if !strings.Contains(page, `action="/scans/trigger"`) ||
		!strings.Contains(page, `name="return" value="/scans"`) {
		t.Fatalf("the trigger form does not carry the submitting URL; body: %s", page)
	}

	for _, from := range []string{"/scans", "/settings?tab=scans", "/scans?stop=12"} {
		resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"dns"}, backField: {from}})
		got := resp.Header.Get("Location")
		resp.Body.Close()
		if got != from {
			t.Errorf("trigger from %q landed at %q, want the submitting URL", from, got)
		}
	}

	for _, hostile := range []string{"https://evil.example/x", "//evil.example/x", `/\evil.example`} {
		resp := postForm(t, ac, base+"/scans/trigger", url.Values{"kind": {"dns"}, backField: {hostile}})
		got := resp.Header.Get("Location")
		resp.Body.Close()
		if got != "/settings?tab=scans" {
			t.Errorf("trigger with %q landed at %q, want the fallback /settings?tab=scans", hostile, got)
		}
	}
}

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
	if len(trig.calls) != 0 {
		t.Fatalf("a disabled scan reached the dispatcher: %v", trig.calls)
	}

	page := getBody(t, ac, base+loc, http.StatusOK)
	if !strings.Contains(page, "The cold scan is disabled") {
		t.Errorf("disabled receipt missing; body: %s", page)
	}
}

func TestTriggerScanOverlapRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0),
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

func TestTriggerScanEmptyFanOut(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 0}

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

func TestScansPageTriggerPanelDegrades(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.listScansErr = errors.New("scan: connection refused")

	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)
	if strings.Contains(page, `action="/scans/trigger"`) {
		t.Errorf("panel should be absent when its read fails; body: %s", page)
	}
	if !strings.Contains(page, "No scan running") {
		t.Errorf("the monitor itself should still render; body: %s", page)
	}
}

func TestScansPageAdminSeesTriggerPanel(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	if !strings.Contains(page, `action="/scans/trigger"`) {
		t.Errorf("admin trigger form missing; body: %s", page)
	}
	if !strings.Contains(page, `value="hot"`) {
		t.Errorf("enabled hot scan has no trigger; body: %s", page)
	}
	if !strings.Contains(page, "cold") || !strings.Contains(page, "disabled") {
		t.Errorf("disabled cold scan should show as not-triggerable; body: %s", page)
	}
}

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

func TestActiveDispatchKinds(t *testing.T) {
	f := newFakeStore()
	tick := time.Date(2026, 8, 16, 9, 30, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(10, "hot", tick, 3, 1, 1, 1, 0, 0),
		progressRow(9, "dns", tick, 2, 0, 0, 2, 0, 0),
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
