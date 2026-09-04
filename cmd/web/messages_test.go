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
		"certificate", "http-identity",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("message panel missing %q\nbody: %s", want, page)
		}
	}
	for _, m := range f.messages {
		if message.ContainsValence(m.Headline) {
			t.Errorf("a rendered message headline carries a valence word: %q", m.Headline)
		}
	}
}

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
	if !strings.Contains(page, "hooks.example.net") {
		t.Errorf("undelivered channel host not shown; body: %s", page)
	}
	if strings.Contains(page, "token=secret") {
		t.Errorf("the full channel URL (with its token) leaked onto the panel; body: %s", page)
	}
	if !strings.Contains(page, "could not be delivered") {
		t.Errorf("the undelivered mark does not distinguish a delivery failure from nothing firing; body: %s", page)
	}
	if !strings.Contains(page, "HTTP 503") {
		t.Errorf("the delivery failure reason is not carried as a drill-down; body: %s", page)
	}
}

func TestUnreadCountAndMarkRead(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m1 := putMessage(t, f, message.CauseDrift, "name", "a.example.com", "a.example.com entered the estate · 1 timeline opened beneath it", nil)
	putMessage(t, f, message.CauseDrift, "name", "b.example.com", "b.example.com entered the estate · 1 timeline opened beneath it", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	seeds := getBody(t, ac, base+"/scope", http.StatusOK)
	if !strings.Contains(seeds, `href="/inbox"`) {
		t.Error("the global nav is missing the Inbox bell")
	}
	if !strings.Contains(seeds, `class="ud"`) {
		t.Errorf("the bell should wear the unread dot\nbody: %s", seeds)
	}
	if !strings.Contains(seeds, `2 unread`) {
		t.Errorf("the palette Inbox hint should carry the unread count 2\nbody: %s", seeds)
	}

	resp := postForm(t, ac, base+"/messages/read", url.Values{"id": {strconv.FormatInt(m1.ID, 10)}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("mark read: status = %d", resp.StatusCode)
	}
	resp.Body.Close()
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 1 {
		t.Errorf("unread after mark read = %d, want 1", n)
	}
}

func TestMarkAllReadIsPerAccount(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	viewer := seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	putMessage(t, f, message.CauseDrift, "name", "a.example.com", "a.example.com entered the estate", nil)

	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := postForm(t, vc, base+"/messages/read-all", url.Values{})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("viewer mark all read: status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	if n, _ := f.CountUnreadMessages(t.Context(), viewer.ID); n != 0 {
		t.Errorf("viewer unread after mark all read = %d, want 0", n)
	}
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 1 {
		t.Errorf("admin unread after viewer mark all read = %d, want 1 (a viewer must not clear an admin's badge)", n)
	}
}

func TestNarrowingPreviewWiredUp(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	f.previewResult = db.PreviewExclusionWithdrawalRow{SubjectsWithdrawn: 128, TimelinesRemoved: 17920}

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/exclusions/preview", url.Values{
		"kind": {"address"}, "value": {"198.51.100.128/25"},
	})
	page := refusalPage(t, ac, base, resp)
	for _, want := range []string{"128 subjects withdrawn", "17,920 timelines", "not seen"} {
		if !strings.Contains(page, want) {
			t.Errorf("narrowing preview missing %q\nbody: %s", want, page)
		}
	}

	resp = postForm(t, ac, base+"/exclusions/preview", url.Values{
		"kind": {"name"}, "value": {"api.example.com"},
	})
	page = refusalPage(t, ac, base, resp)
	if !strings.Contains(page, "Nothing is withdrawn") {
		t.Errorf("a non-firing name exclusion should say nothing is withdrawn\nbody: %s", page)
	}
}
