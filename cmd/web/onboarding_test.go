package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// The onboarding wizard is gated behind login — an anonymous GET is bounced to the
// sign-in page, never rendered (AC: requires-login).
func TestOnboardingRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")

	c := newClient(t)
	resp, err := c.Get(base + "/onboarding")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anon /onboarding: status = %d, want 303", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/login" {
		t.Fatalf("anon /onboarding redirect = %q, want /login", loc)
	}
}

// The four steps render in order and Next advances only past a satisfied valid
// gate: the seeds step blocks Next until a seed is entered, then walks seeds ->
// cadence -> channel -> review (AC: the 4 steps render/advance, controlled).
func TestOnboardingStepsAdvance(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	// Step 0 renders the seeds field.
	page := getBody(t, ac, base+"/onboarding", http.StatusOK)
	if !strings.Contains(page, "Set up this workspace") || !strings.Contains(page, `name="seedsadd"`) {
		t.Fatalf("step 0 did not render the seeds step; body: %s", page)
	}

	// Next on an empty seeds step does not advance — the valid gate holds it on step 0.
	resp := postForm(t, ac, base+"/onboarding", url.Values{"step": {"0"}, "action": {"next"}})
	held := body(t, resp)
	if !strings.Contains(held, `name="seedsadd"`) {
		t.Fatalf("empty seeds should hold on step 0; body: %s", held)
	}

	// Enter a seed and advance to cadence.
	resp = postForm(t, ac, base+"/onboarding", url.Values{"step": {"0"}, "action": {"next"}, "seedsadd": {"acmecorp.io"}})
	cadence := body(t, resp)
	if !strings.Contains(cadence, "Scan profile") || !strings.Contains(cadence, "Passive only") {
		t.Fatalf("did not advance to cadence step; body: %s", cadence)
	}
	// The committed seed rides forward as hidden state.
	if !strings.Contains(cadence, `name="seeds" value="acmecorp.io"`) {
		t.Fatalf("committed seed not carried forward; body: %s", cadence)
	}

	// Advance cadence -> channel.
	resp = postForm(t, ac, base+"/onboarding", url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
	})
	channel := body(t, resp)
	if !strings.Contains(channel, "Delivery URL (optional)") {
		t.Fatalf("did not advance to channel step; body: %s", channel)
	}

	// Advance channel -> review.
	resp = postForm(t, ac, base+"/onboarding", url.Values{
		"step": {"2"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"},
		"channel": {"https://ops.example/hook"},
	})
	review := body(t, resp)
	if !strings.Contains(review, "Start first scan") {
		t.Fatalf("did not advance to review step; body: %s", review)
	}
}

// The custom cadence gate holds Next until a cron is supplied, mirroring the
// example's valid predicate (cad != Custom || cron non-empty).
func TestOnboardingCustomCadenceGate(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	// Custom cadence with no cron: Next holds on the cadence step and reveals the cron field.
	resp := postForm(t, ac, base+"/onboarding", url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Custom…"},
	})
	held := body(t, resp)
	if !strings.Contains(held, "Scan profile") || !strings.Contains(held, `name="cron"`) {
		t.Fatalf("custom cadence without cron should hold on cadence and reveal cron; body: %s", held)
	}

	// With a cron supplied, Next advances to the channel step.
	resp = postForm(t, ac, base+"/onboarding", url.Values{
		"step": {"1"}, "action": {"next"}, "seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Custom…"}, "cron": {"0 8 * * 1"},
	})
	adv := body(t, resp)
	if !strings.Contains(adv, "Delivery URL (optional)") {
		t.Fatalf("custom cadence with cron should advance to channel; body: %s", adv)
	}
}

// The review step summarizes the real inputs — the seeds, profile, cadence and
// channel the operator entered, not sample data (AC: review reflects real inputs).
func TestOnboardingReviewReflectsInputs(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := startWithTrigger(t, f, &fakeTrigger{jobs: 2})
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/onboarding", url.Values{
		"step": {"2"}, "action": {"next"},
		"seeds": {"acmecorp.io,203.0.113.0/24"}, "profile": {"passive"}, "cad": {"Weekly · mon 09:00"},
		"channel": {"https://ops.example/hook"},
	})
	review := body(t, resp)
	for _, want := range []string{"acmecorp.io, 203.0.113.0/24", "passive", "weekly · mon 09:00", "https://ops.example/hook"} {
		if !strings.Contains(review, want) {
			t.Errorf("review missing real input %q; body: %s", want, review)
		}
	}
	// The finish button carries the passive-profile scan kind (dns discovery).
	if !strings.Contains(review, `name="kind" value="dns"`) {
		t.Errorf("passive profile should map the first scan to dns; body: %s", review)
	}
}

// Completion enqueues a real scan through the existing trigger path: the finish
// button posts the mapped kind to /onboarding/finish, which dispatches the same
// fan-out POST /scans/trigger uses and lands on the monitor (AC: completion
// enqueues a scan, asserted via the scan-trigger path; no fabricated scan).
func TestOnboardingFinishEnqueuesScan(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 4}
	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "admin", "hunter2hunter2")

	// A standard profile maps the first scan to the active hot port scan.
	resp := postForm(t, ac, base+"/onboarding/finish", url.Values{
		"seeds": {"acmecorp.io"}, "profile": {"standard"}, "cad": {"Daily · 08:00"}, "kind": {"hot"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("finish: status = %d, want 303", resp.StatusCode)
	}
	if !strings.HasPrefix(loc, "/scans?") || !strings.Contains(loc, "notice=triggered") || !strings.Contains(loc, "kind=hot") {
		t.Fatalf("finish redirect = %q, want /scans with a triggered hot notice", loc)
	}
	if len(trig.calls) != 1 || trig.calls[0] != "hot" {
		t.Fatalf("finish dispatcher calls = %v, want one hot fan-out", trig.calls)
	}
}

// Completion is an admin act — a viewer cannot enqueue the first scan, matching the
// admin-only guardrail on the trigger it reuses (AC: admin-only enqueue).
func TestOnboardingFinishViewerRefused(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	trig := &fakeTrigger{jobs: 4}
	base := startWithTrigger(t, f, trig)
	ac := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, ac, base+"/onboarding/finish", url.Values{"profile": {"standard"}, "kind": {"hot"}})
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer finish: status = %d, want 403", resp.StatusCode)
	}
	if len(trig.calls) != 0 {
		t.Fatalf("a viewer reached the dispatcher: %v", trig.calls)
	}
}
