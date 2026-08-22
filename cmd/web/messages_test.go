package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// putMessage inserts a computed message straight into the fake store, standing
// in for the cause path that would have written it.
func putMessage(t *testing.T, f *fakeStore, cause message.Cause, subjectKind, firedAt, headline string, census []byte) db.Message {
	t.Helper()
	m, err := f.InsertMessage(t.Context(), db.InsertMessageParams{
		Cause:       string(cause),
		Class:       string(message.ClassForCause(cause)),
		SubjectKind: subjectKind,
		FiredAt:     firedAt,
		Instant:     pgtype.Timestamptz{Time: time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC), Valid: true},
		Census:      census,
		Headline:    headline,
	})
	if err != nil {
		t.Fatalf("insert message: %v", err)
	}
	return m
}

// Every subject kind resolves to a drill-down its route actually serves (#248):
// a Name or Address is a single path segment; a Service or Endpoint key carries a
// `/` (and an Endpoint an `@`), so it rides the `?key=` query page escaped rather
// than a second path segment the `/subjects/{key}` route would 404 on.
func TestSubjectHrefAllKinds(t *testing.T) {
	cases := []struct {
		kind string
		key  string
		want string
	}{
		{"name", "example.com", "/subjects/example.com"},
		{"address", "198.51.100.1", "/subjects/198.51.100.1"},
		{"service", "104.21.61.6:80/tcp", "/subjects/service?key=104.21.61.6%3A80%2Ftcp"},
		{"endpoint", "@104.21.61.6:9100/tcp", "/subjects/endpoint?key=%40104.21.61.6%3A9100%2Ftcp"},
		{"endpoint", "host.example.com@104.21.61.6:443/tcp", "/subjects/endpoint?key=host.example.com%40104.21.61.6%3A443%2Ftcp"},
	}
	for _, c := range cases {
		if got := subjectHref(c.kind, c.key); got != c.want {
			t.Errorf("subjectHref(%q, %q) = %q, want %q", c.kind, c.key, got, c.want)
		}
	}
}

// Each mover resolves to the right link (v1 spec §5.3): drift and threshold to an
// object page, declared-input to the Source, aperture to the Seed.
func TestMessageLinkPerMover(t *testing.T) {
	cases := []struct {
		cause       message.Cause
		subjectKind string
		firedAt     string
		wantHref    string
	}{
		{message.CauseDrift, "service", "198.51.100.1:443/tcp", "/subjects/service?key=198.51.100.1%3A443%2Ftcp"},
		{message.CauseDrift, "name", "example.com", "/subjects/example.com"},
		{message.CauseThreshold, "name", "expiry.example.com", "/subjects/expiry.example.com"},
		{message.CauseDeclaredInput, "source", "zone-file", "/sources"},
		{message.CauseAperture, "seed", "198.51.100.0/24", "/scope#seed-198-51-100-0-24"},
	}
	for _, c := range cases {
		href, text := messageLink(c.cause, c.subjectKind, c.firedAt)
		if href != c.wantHref {
			t.Errorf("messageLink(%q,%q,%q) href = %q, want %q", c.cause, c.subjectKind, c.firedAt, href, c.wantHref)
		}
		if text != c.firedAt {
			t.Errorf("link text = %q, want the fired-at key %q", text, c.firedAt)
		}
	}
}

// The panel renders every message newest-first with its headline and its
// per-mover link, and a flagship's census is enumerated beneath it.
func TestMessagePanelRendersRowsAndCensus(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	census, _ := message.NewCensus(
		message.CensusEntry{Kind: "facet", Key: "certificate"},
		message.CensusEntry{Kind: "facet", Key: "http-identity"},
	).Marshal()
	putMessage(t, f, message.CauseDrift, "service", "198.51.100.1:443/tcp",
		"198.51.100.1:443/tcp reached from the internet · 2 facets opened beneath it", census)
	putMessage(t, f, message.CauseAperture, "seed", "198.51.100.0/24",
		"198.51.100.0/24 narrowed · 198.51.100.128/25 excluded · 128 subjects withdrawn · 17,920 timelines taken out of the estate", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/messages", http.StatusOK)

	for _, want := range []string{
		"reached from the internet",
		"128 subjects withdrawn",
		"/subjects/service?key=198.51.100.1%3A443%2Ftcp",
		"certificate", "http-identity", // the flagship census, enumerated in full
	} {
		if !strings.Contains(page, want) {
			t.Errorf("message panel missing %q\nbody: %s", want, page)
		}
	}
	// The rendered headlines carry no valence word (the model guards the copy at
	// construction; here we confirm the stored, rendered sentences are clear).
	for _, m := range f.messages {
		if message.ContainsValence(m.Headline) {
			t.Errorf("a rendered message headline carries a valence word: %q", m.Headline)
		}
	}
}

// ADR-0108 / #244: an undelivered delivery is surfaced on the Message it failed
// to carry — the model's designated surface (ADR-0039/ADR-0081), never Coverage.
// A backend failure (the webhook was down) must read as *could not deliver*, not
// as *nothing fired*, and the reason is carried as a drill-down (#22).
func TestMessagePanelSurfacesUndeliveredDeliveries(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m := putMessage(t, f, message.CauseDrift, "name", "a.example.com",
		"a.example.com entered the estate · 1 timeline opened beneath it", nil)
	f.deliveryOutcomes = []db.ListDeliveryOutcomesRow{{
		MessageID: m.ID, ChannelID: 1, Url: "https://hooks.example.net/verge?token=secret",
		State: "undelivered", Attempt: 5, LastError: pgtype.Text{String: "HTTP 503", Valid: true},
	}}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/messages", http.StatusOK)

	if !strings.Contains(page, "undelivered") {
		t.Errorf("message panel does not surface the undelivered delivery; body: %s", page)
	}
	// The channel is shown by host only — the token in the URL path must not leak.
	if !strings.Contains(page, "hooks.example.net") {
		t.Errorf("undelivered channel host not shown; body: %s", page)
	}
	if strings.Contains(page, "token=secret") {
		t.Errorf("the full channel URL (with its token) leaked onto the panel; body: %s", page)
	}
	// The failure reads as a delivery failure, distinct from an empty result.
	if !strings.Contains(page, "could not be delivered") {
		t.Errorf("the undelivered mark does not distinguish a delivery failure from nothing firing; body: %s", page)
	}
	// The reason is carried as a drill-down, not a top-level log line.
	if !strings.Contains(page, "HTTP 503") {
		t.Errorf("the delivery failure reason is not carried as a drill-down; body: %s", page)
	}
}

// The global nav element carries the unread count on every screen, and marking a
// message read drops the count.
func TestUnreadCountAndMarkRead(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m1 := putMessage(t, f, message.CauseDrift, "name", "a.example.com", "a.example.com entered the estate · 1 timeline opened beneath it", nil)
	putMessage(t, f, message.CauseDrift, "name", "b.example.com", "b.example.com entered the estate · 1 timeline opened beneath it", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// The count rides a chrome page that never computed it itself.
	seeds := getBody(t, ac, base+"/scope", http.StatusOK)
	if !strings.Contains(seeds, `href="/inbox"`) {
		t.Error("the global nav is missing the Inbox bell")
	}
	if !strings.Contains(seeds, `class="count">2<`) {
		t.Errorf("the nav should carry the unread count 2\nbody: %s", seeds)
	}

	// Mark one read; the count drops to 1.
	resp := postForm(t, ac, base+"/messages/read", url.Values{"id": {strconv.FormatInt(m1.ID, 10)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("mark read: status = %d", resp.StatusCode)
	}
	resp.Body.Close()
	if n, _ := f.CountUnreadMessages(t.Context()); n != 1 {
		t.Errorf("unread after mark read = %d, want 1", n)
	}
}

// AC8: the narrowing receipt is honestly computable and wired up. An address
// exclusion over inhabited ground shows the count and names the loss; a name
// whose names still resolve withdraws nothing and shows no firing receipt.
func TestNarrowingPreviewWiredUp(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.previewResult = db.PreviewExclusionWithdrawalRow{SubjectsWithdrawn: 128, TimelinesRemoved: 17920}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Address exclusion: the receipt fires, carrying the counts and the loss.
	resp := postForm(t, ac, base+"/exclusions/preview", url.Values{
		"kind": {"address"}, "value": {"198.51.100.128/25"},
	})
	page := body(t, resp)
	for _, want := range []string{"128 subjects withdrawn", "17,920 timelines", "not seen"} {
		if !strings.Contains(page, want) {
			t.Errorf("narrowing preview missing %q\nbody: %s", want, page)
		}
	}

	// Name exclusion: nothing is withdrawn (the query is address-scoped), so the
	// receipt does not fire.
	resp = postForm(t, ac, base+"/exclusions/preview", url.Values{
		"kind": {"name"}, "value": {"api.example.com"},
	})
	page = body(t, resp)
	if !strings.Contains(page, "Nothing is withdrawn") {
		t.Errorf("a non-firing name exclusion should say nothing is withdrawn\nbody: %s", page)
	}
}
