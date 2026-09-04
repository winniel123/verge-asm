package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/message"
)

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
		`class="n">2<`,
		`class="ib-tag">drift`,
		`class="ib-tag">coverage`,
		"ib-dot unread",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inbox missing %q\nbody: %s", want, page)
		}
	}
	if !strings.Contains(page, `action="/messages/read-all"`) || !strings.Contains(page, `name="return" value="/inbox"`) {
		t.Errorf("inbox mark-all-read form not wired to return to /inbox\nbody: %s", page)
	}
	if strings.Contains(page, `disabled>Mark all read`) {
		t.Errorf("mark-all-read should be live while unread\nbody: %s", page)
	}
}

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
	if strings.Contains(page, "ib-dot unread") {
		t.Errorf("no message should carry the unread dot after mark all read\nbody: %s", page)
	}
	if !strings.Contains(page, "a.example.com entered the estate") {
		t.Errorf("read messages should still list under All\nbody: %s", page)
	}
}

func TestInboxSelectMarksReadAndShowsDetail(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m := putMessage(t, f, message.CauseDrift, "service", "198.51.100.1:443/tcp",
		"198.51.100.1:443/tcp reached from the internet · 2 facets opened beneath it", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/inbox?id="+strconv.FormatInt(m.ID, 10), http.StatusOK)

	for _, want := range []string{
		"198.51.100.1:443/tcp reached from the internet",
		`class="ib-micro">drift`,
		`href="/subjects/service?key=198.51.100.1%3A443%2Ftcp`,
		"Open subject",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inbox detail missing %q\nbody: %s", want, page)
		}
	}
	if strings.Contains(page, "No message selected.") {
		t.Errorf("a message is open, so the no-selection empty-state must not show\nbody: %s", page)
	}
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 0 {
		t.Errorf("unread after opening the only message = %d, want 0", n)
	}
}

func TestInboxMarkUnread(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m := putMessage(t, f, message.CauseDrift, "name", "a.example.com",
		"a.example.com entered the estate", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	page := getBody(t, ac, base+"/inbox?id="+strconv.FormatInt(m.ID, 10), http.StatusOK)
	if i := strings.Index(page, "</header>"); i >= 0 {
		page = page[i:]
	}
	for _, want := range []string{
		`action="/messages/unread"`,
		`>Mark unread</button>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("inbox detail missing the Mark-unread affordance %q\nbody: %s", want, page)
		}
	}
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 0 {
		t.Fatalf("unread after opening the only message = %d, want 0", n)
	}

	resp := postForm(t, ac, base+"/messages/unread", url.Values{"id": {strconv.FormatInt(m.ID, 10)}, "return": {"/inbox"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/inbox" {
		t.Fatalf("mark unread: status=%d location=%q, want 303 to /inbox", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()
	if n, _ := f.CountUnreadMessages(t.Context(), admin.ID); n != 1 {
		t.Fatalf("unread after mark unread = %d, want 1 (read is reversible)", n)
	}

	page = getBody(t, ac, base+"/inbox?filter=unread", http.StatusOK)
	if i := strings.Index(page, "</header>"); i >= 0 {
		page = page[i:]
	}
	if !strings.Contains(page, "a.example.com entered the estate") {
		t.Errorf("un-read message should reappear under the unread filter\nbody: %s", page)
	}
}

func TestInboxUnreadFilter(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	m1 := putMessage(t, f, message.CauseDrift, "name", "alpha.example.com", "alpha.example.com entered the estate", nil)
	putMessage(t, f, message.CauseDrift, "name", "bravo.example.com", "bravo.example.com entered the estate", nil)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	resp := postForm(t, ac, base+"/messages/read", url.Values{"id": {strconv.FormatInt(m1.ID, 10)}, "return": {"/inbox"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/inbox" {
		t.Fatalf("mark read: status=%d location=%q, want 303 to /inbox", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	page := getBody(t, ac, base+"/inbox?filter=unread", http.StatusOK)
	// The chrome bell lists messages whatever the filter, so the assertions scope past the header.
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
