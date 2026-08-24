package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/message"
)

// The Inbox (#299, T4) renders the message list read/unread — an unread dot and the
// unread count — with each message's class tag, and offers a mark-all-read control
// that is live while anything is unread.
func TestInboxRendersReadUnread(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	putMessage(t, f, message.CauseDrift, "name", "a.example.com",
		"a.example.com entered the estate · 1 timeline opened beneath it", nil)
	putMessage(t, f, message.CauseAperture, "seed", "198.51.100.0/24",
		"198.51.100.0/24 narrowed · 128 subjects withdrawn", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inbox", http.StatusOK)

	for _, want := range []string{
		"Inbox",
		"Everything Verge told you, by class.",
		"a.example.com entered the estate",
		"198.51.100.0/24 narrowed",
		`class="n">2<`,             // two unread, rendered mono in the subtitle
		`class="ibx-tag">drift`,    // the store's own class vocabulary, not the example's
		`class="ibx-tag">coverage`, // an aperture firing routes to the coverage class
		"ibx-dot unread",           // the unread affordance
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inbox missing %q\nbody: %s", want, page)
		}
	}
	// Mark-all-read is live (not disabled) while messages are unread, and returns to
	// the Inbox rather than the /messages fold.
	if !strings.Contains(page, `action="/messages/read-all"`) || !strings.Contains(page, `name="return" value="/inbox"`) {
		t.Errorf("inbox mark-all-read form not wired to return to /inbox\nbody: %s", page)
	}
	if strings.Contains(page, `disabled>Mark all read`) {
		t.Errorf("mark-all-read should be live while unread\nbody: %s", page)
	}
}

// Marking all read clears the unread count and disables the control; the message
// rows stay (the All filter shows read messages too), now without the unread dot.
func TestInboxMarkAllRead(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	putMessage(t, f, message.CauseDrift, "name", "a.example.com", "a.example.com entered the estate", nil)
	putMessage(t, f, message.CauseDrift, "name", "b.example.com", "b.example.com entered the estate", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := postForm(t, ac, base+"/messages/read-all", url.Values{"return": {"/inbox"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/inbox" {
		t.Fatalf("mark all read: status=%d location=%q, want 303 to /inbox", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 0 {
		t.Errorf("unread after mark all read = %d, want 0", n)
	}

	page := getBody(t, ac, base+"/inbox", http.StatusOK)
	if !strings.Contains(page, `class="n">0<`) {
		t.Errorf("inbox should read zero unread after mark all read\nbody: %s", page)
	}
	if !strings.Contains(page, `disabled>Mark all read`) {
		t.Errorf("mark-all-read should be disabled at inbox zero\nbody: %s", page)
	}
	if strings.Contains(page, "ibx-dot unread") {
		t.Errorf("no message should carry the unread dot after mark all read\nbody: %s", page)
	}
	// The messages themselves are still listed under the All filter.
	if !strings.Contains(page, "a.example.com entered the estate") {
		t.Errorf("read messages should still list under All\nbody: %s", page)
	}
}

// Opening a message (an ?id link, the ported open()/initialId) marks it read and
// renders the per-class detail — the class micro-label and the per-mover jump link.
func TestInboxSelectMarksReadAndShowsDetail(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m := putMessage(t, f, message.CauseDrift, "service", "198.51.100.1:443/tcp",
		"198.51.100.1:443/tcp reached from the internet · 2 facets opened beneath it", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inbox?id="+strconv.FormatInt(m.ID, 10), http.StatusOK)

	for _, want := range []string{
		"198.51.100.1:443/tcp reached from the internet",       // the headline, as the detail title
		`class="microlabel">drift`,                             // per-class detail micro-label
		`href="/subjects/service?key=198.51.100.1%3A443%2Ftcp`, // the per-mover jump link
		"Open subject",                                         // the jump-link label
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inbox detail missing %q\nbody: %s", want, page)
		}
	}
	if strings.Contains(page, "No message selected.") {
		t.Errorf("a message is open, so the no-selection empty-state must not show\nbody: %s", page)
	}
	// Opening marked it read: the count drops to zero.
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 0 {
		t.Errorf("unread after opening the only message = %d, want 0", n)
	}
}

// The unread filter shows only unread messages; a read one drops out of the list.
func TestInboxUnreadFilter(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m1 := putMessage(t, f, message.CauseDrift, "name", "alpha.example.com", "alpha.example.com entered the estate", nil)
	putMessage(t, f, message.CauseDrift, "name", "bravo.example.com", "bravo.example.com entered the estate", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	// Mark the first read out of band, then view the unread filter.
	resp := postForm(t, ac, base+"/messages/read", url.Values{"id": {strconv.FormatInt(m1.ID, 10)}, "return": {"/inbox"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/inbox" {
		t.Fatalf("mark read: status=%d location=%q, want 303 to /inbox", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	page := getBody(t, ac, base+"/inbox?filter=unread", http.StatusOK)
	// The chrome's inbox bell lists recent messages regardless of the inbox filter
	// (TopNav.jsx onOpenMessage), so a read message legitimately still shows in the
	// bell menu. Scope the filter assertions to the inbox itself — the region past
	// the chrome header — so they test the list, not the always-on bell.
	if i := strings.Index(page, "</header>"); i >= 0 {
		page = page[i:]
	}
	if !strings.Contains(page, "bravo.example.com entered the estate") {
		t.Errorf("unread filter should show the unread message\nbody: %s", page)
	}
	if strings.Contains(page, "alpha.example.com entered the estate") {
		t.Errorf("unread filter should hide the read message\nbody: %s", page)
	}
	if !strings.Contains(page, `href="/inbox?filter=unread">Unread`) {
		t.Errorf("the Unread filter tab is missing\nbody: %s", page)
	}
}

// With no messages the Inbox ships the design-system inbox-zero empty-state and the
// no-selection empty-state, and fabricates nothing.
func TestInboxEmptyState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inbox", http.StatusOK)

	for _, want := range []string{
		"Nothing unread.",
		"New messages land here as batches conclude.",
		"No message selected.",
		"Pick a message on the left to read it here.",
		`class="n">0<`,
		`disabled>Mark all read`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inbox empty-state missing %q\nbody: %s", want, page)
		}
	}
}

// The Inbox is behind requireLogin: an unauthenticated request redirects to the
// login form rather than rendering the screen.
func TestInboxRequiresLogin(t *testing.T) {
	f := newFakeStore()
	base := start(t, f, "")
	c := newClient(t)

	resp, err := c.Get(base + "/inbox")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("unauthenticated /inbox: status=%d location=%q, want redirect to /login",
			resp.StatusCode, resp.Header.Get("Location"))
	}
}
