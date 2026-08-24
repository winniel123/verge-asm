package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// deliveryView is one Message's outcome to one Channel, rendered on the Message
// panel — the model's designated surface for a delivery failure (ADR-0039,
// ADR-0081): a delivery has no cause and never touches Coverage, so an
// undelivered POST is legible HERE and joined to the Message it failed to carry,
// never collapsed into silence (#244). Failed reports whether the state is
// undelivered; LastError is the drill-down reason, never a top-level log (#22).
type deliveryView struct {
	ChannelHost string
	State       string
	Failed      bool
	LastError   string
}

// messageRow is one message shaped for the panel: the rendered headline, the
// per-mover link, the cause and class as read-only labels, the instant, the
// read-state, and the census rows where the firing carries one.
type messageRow struct {
	ID       int64
	Cause    string
	Class    string
	Headline string
	// Href and LinkText are the row's link, resolved per its mover (v1 spec
	// §5.3): the object's page, the Source, or the Seed. LinkText is the key the
	// link points at, rendered in mono.
	Href     string
	LinkText string
	Instant  string
	Read     bool
	Census   []censusRowView
	// Deliveries is this message's outcome to each routed Channel. A message with
	// no channel configured carries none; an undelivered one carries a Failed row.
	Deliveries []deliveryView
	// AnyUndelivered is true where at least one delivery is undelivered, so the
	// panel can flag the message without the operator opening every row.
	AnyUndelivered bool
}

// censusRowView is one entry in a message's census — a thing that opened beneath
// the fired-at subject. A subject entry (service/endpoint/name/address) links to
// its own page; a facet or signal entry is a plain label, since it is evidence
// rather than a subject with a page.
type censusRowView struct {
	Kind string
	Key  string
	Href string // empty for a facet or signal — no page to link to
}

// messagesPage renders the global message panel: every message, newest-first and
// unbounded (no cap or load-more ships, v1 spec §6.7). Each row links per its
// mover, and the census of a flagship or membership firing is enumerated in full
// beneath it.
func (s *server) messagesPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: "messages"})
}

// fillMessagesSection assembles the Settings messages sub-tab: every message
// newest-first, each with its per-mover link and census, and the per-message
// delivery outcomes joined in one pass. A delivery read failure degrades to no
// annotation rather than dropping the messages themselves.
func (s *server) fillMessagesSection(r *http.Request, acct db.Account, data map[string]any) error {
	rows, err := s.store.ListMessages(r.Context())
	if err != nil {
		return err
	}
	unread, err := s.store.CountUnreadMessages(r.Context(), acct.ID)
	if err != nil {
		return err
	}
	read, err := s.readSet(r.Context(), acct.ID)
	if err != nil {
		return err
	}
	// A delivery failure is surfaced on the Message it failed to carry (ADR-0039,
	// ADR-0081), never on Coverage. Read every outcome in one pass and group by
	// message so the panel renders each row's own deliveries without an N+1 walk.
	// Best-effort: a read failure degrades to no delivery annotation rather than a
	// 500 — the messages themselves must still render.
	byMessage := map[int64][]deliveryView{}
	if outcomes, derr := s.store.ListDeliveryOutcomes(r.Context()); derr == nil {
		for _, o := range outcomes {
			byMessage[o.MessageID] = append(byMessage[o.MessageID], toDeliveryView(o))
		}
	}

	views := make([]messageRow, 0, len(rows))
	for _, m := range rows {
		row := toMessageRow(m, read[m.ID])
		row.Deliveries = byMessage[m.ID]
		for _, d := range row.Deliveries {
			if d.Failed {
				row.AnyUndelivered = true
			}
		}
		views = append(views, row)
	}
	data["Messages"] = views
	data["Unread"] = unread
	return nil
}

// markMessageRead marks one message read at now and returns to the panel. Read
// state is a per-account fact.
//
// WITHDRAWN by ADR-0116 / #473: the former clause "there is no un-read, since a
// message is read once the operator has seen it" no longer holds. The design
// package is normative for functionality, and its Inbox renders a "Mark unread"
// affordance (Inbox.jsx:59), so read is reversible — see markMessageUnread and
// the MarkMessageUnread mutation. Marked at this site per ADR-0058 (a superseded
// mechanism is withdrawn where it is specified), rather than only at ADR-0116.
func (s *server) markMessageRead(w http.ResponseWriter, r *http.Request, acct db.Account) {
	dest := messageReturn(r)
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, dest, http.StatusSeeOther)
		return
	}
	if err := s.store.MarkMessageRead(r.Context(), db.MarkMessageReadParams{
		AccountID: acct.ID, MessageID: id, ReadAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "mark message read", err)
		return
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// markAllMessagesRead clears the caller's own unread count in one act. Read-state
// is per-account (#327): this marks read only for the calling account, so it can
// never clear another operator's — or an admin's — unread badge.
func (s *server) markAllMessagesRead(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if err := s.store.MarkAllMessagesRead(r.Context(), db.MarkAllMessagesReadParams{
		AccountID: acct.ID, ReadAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "mark all messages read", err)
		return
	}
	http.Redirect(w, r, messageReturn(r), http.StatusSeeOther)
}

// markMessageUnread returns one message to unread for the caller and redirects
// back to the panel (#473, ADR-0116). It is the inverse of markMessageRead: the
// design's Inbox renders a "Mark unread" affordance (Inbox.jsx:59), so read is a
// reversible per-account fact rather than a monotonic one. Clearing the caller's
// own read-mark can never touch another operator's unread badge (#327), and the
// delete is idempotent, so re-marking an already-unread message is harmless. The
// unread count and the shell bell reflect the flip on the next read.
func (s *server) markMessageUnread(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// The unread control posts return="/inbox" (a constant); resolve the target to
	// a bool here and redirect to a string LITERAL at each call site below, so no
	// request-derived value ever reaches http.Redirect. This satisfies gosec's
	// G107 open-redirect taint analyzer by construction (a #nosec its taint pass
	// ignores would not), and preserves markMessageRead's /inbox-or-/messages home.
	toInbox := false
	if ret := r.FormValue("return"); ret == "/inbox" || strings.HasPrefix(ret, "/inbox") {
		toInbox = true
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		if toInbox {
			http.Redirect(w, r, "/inbox", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/messages", http.StatusSeeOther)
		}
		return
	}
	if err := s.store.MarkMessageUnread(r.Context(), db.MarkMessageUnreadParams{
		AccountID: acct.ID, MessageID: id,
	}); err != nil {
		s.serverError(w, "mark message unread", err)
		return
	}
	if toInbox {
		http.Redirect(w, r, "/inbox", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/messages", http.StatusSeeOther)
	}
}

// messageReturn is where a message-read POST returns to. The two read handlers are
// shared by the viewer-readable /messages fold and the V3 /inbox surface (#299);
// an /inbox form carries a `return` field so the redirect lands back on the Inbox,
// while /messages posts (which carry none) keep their historical /messages home. The
// value is admitted only when it names the Inbox, so the field is not an open
// redirect into an arbitrary URL.
func messageReturn(r *http.Request) string {
	if ret := r.FormValue("return"); ret == "/inbox" || strings.HasPrefix(ret, "/inbox?") {
		return ret
	}
	return "/messages"
}

// inboxView is one message shaped for the Inbox screen (#299): the shared
// messageRow plus the relative instant the list and detail render, the jump-link
// label, and whether this row is the one the operator has open.
type inboxView struct {
	messageRow
	Rel       string // relative instant ("3h"), with the absolute Instant on hover
	JumpLabel string // the detail card's per-mover jump-link label
	Selected  bool
}

// inboxPage renders the Inbox (#299, ADR-0110), the V3 primary message surface the
// shell bell deep-links to. It is the console port of
// design-system/examples/console/Inbox.jsx: the read/unread list beside the
// per-message detail, an all/unread filter, mark-all-read, and the per-mover jump
// link. Selecting a message (an ?id link, the port's open()/initialId) marks it
// read, so the unread count is read back after that act. Sample data is swapped for
// real Message rows of the same shape — the class vocabulary is the store's own
// (drift / coverage / clock), never the example's placeholder classes. Where there
// are no messages, the design-system inbox-zero empty-state renders; nothing is
// fabricated.
func (s *server) inboxPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// Opening a message marks it read (the port's open() and initialId both do), so
	// resolve the selection first and mark before the counts are read back.
	var selID int64
	if v := r.URL.Query().Get("id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			selID = id
			if err := s.store.MarkMessageRead(r.Context(), db.MarkMessageReadParams{
				AccountID: acct.ID, MessageID: selID, ReadAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
			}); err != nil {
				s.serverError(w, "mark message read", err)
				return
			}
		}
	}
	filter := "all"
	if r.URL.Query().Get("filter") == "unread" {
		filter = "unread"
	}

	rows, err := s.store.ListMessages(r.Context())
	if err != nil {
		s.serverError(w, "list messages", err)
		return
	}
	unread, err := s.store.CountUnreadMessages(r.Context(), acct.ID)
	if err != nil {
		s.serverError(w, "count unread messages", err)
		return
	}
	read, err := s.readSet(r.Context(), acct.ID)
	if err != nil {
		s.serverError(w, "list read messages", err)
		return
	}
	// A delivery failure is surfaced on the Message it failed to carry (ADR-0039,
	// ADR-0081), never on Coverage. Best-effort, as on the /messages fold: a read
	// failure degrades to no delivery annotation rather than dropping the messages.
	byMessage := map[int64][]deliveryView{}
	if outcomes, derr := s.store.ListDeliveryOutcomes(r.Context()); derr == nil {
		for _, o := range outcomes {
			byMessage[o.MessageID] = append(byMessage[o.MessageID], toDeliveryView(o))
		}
	}

	now := s.now()
	shown := make([]inboxView, 0, len(rows))
	var selected *inboxView
	for _, m := range rows {
		base := toMessageRow(m, read[m.ID])
		base.Deliveries = byMessage[m.ID]
		for _, d := range base.Deliveries {
			if d.Failed {
				base.AnyUndelivered = true
			}
		}
		v := inboxView{messageRow: base, JumpLabel: jumpLabel(message.Cause(m.Cause), m.SubjectKind)}
		if m.Instant.Valid {
			v.Rel = relTime(m.Instant.Time, now)
		}
		if m.ID == selID {
			v.Selected = true
			sel := v
			selected = &sel // the open message is shown whatever the filter
		}
		if filter == "unread" && base.Read {
			continue
		}
		shown = append(shown, v)
	}

	// The filter toggle preserves the open message, so its links carry the id.
	allHref, unreadHref := "/inbox", "/inbox?filter=unread"
	if selID != 0 {
		idq := "id=" + strconv.FormatInt(selID, 10)
		allHref, unreadHref = "/inbox?"+idq, "/inbox?filter=unread&"+idq
	}

	s.render(w, "inbox", map[string]any{
		"Title": "Inbox", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":  "inbox",
		"Messages":   shown,
		"Selected":   selected,
		"Unread":     unread,
		"Filter":     filter,
		"AllHref":    allHref,
		"UnreadHref": unreadHref,
	})
}

// jumpLabel is the detail card's jump-link label, keyed on the message's mover so
// the button names where it lands — the Source, the Seed's scope, or the subject
// that moved — mirroring the per-mover link messageLink resolves.
func jumpLabel(cause message.Cause, subjectKind string) string {
	switch message.LinkKindForCause(cause) {
	case message.LinkSource:
		return "Open source"
	case message.LinkSeed:
		return "Open scope"
	default:
		switch subjectKind {
		case "service", "endpoint", "name", "address":
			return "Open subject"
		}
		return "Open inventory"
	}
}

// relTime renders an instant as a terse relative age ("now", "3h", "2d"), the
// design system's relative-timestamp convention; the absolute instant rides the
// element's title for the exact reading.
func relTime(t, now time.Time) string {
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	case d < 7*24*time.Hour:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d"
	default:
		return strconv.Itoa(int(d/(7*24*time.Hour))) + "w"
	}
}

// readSet resolves the set of message ids the account has read (#327), keyed for
// O(1) per-row lookup while shaping the panel and Inbox. Read-state is a
// per-account fact held in message_read; this is the read-side companion to the
// per-account CountUnreadMessages and mark handlers.
func (s *server) readSet(ctx context.Context, accountID int64) (map[int64]bool, error) {
	ids, err := s.store.ListReadMessageIDs(ctx, accountID)
	if err != nil {
		return nil, err
	}
	set := make(map[int64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// toMessageRow renders one stored message for the panel, resolving its link per
// its mover and unpacking its census. read is the CALLER's read-state for this
// message (#327) — a per-account fact resolved from message_read, never the
// retired global message.read_at column — so the badge and unread filter reflect
// whether THIS operator has seen the message.
func toMessageRow(m db.Message, read bool) messageRow {
	row := messageRow{
		ID:       m.ID,
		Cause:    m.Cause,
		Class:    m.Class,
		Headline: m.Headline,
		Read:     read,
	}
	if m.Instant.Valid {
		row.Instant = m.Instant.Time.UTC().Format("2006-01-02 15:04 UTC")
	}
	row.Href, row.LinkText = messageLink(message.Cause(m.Cause), m.SubjectKind, m.FiredAt)

	if c, err := message.ParseCensus(m.Census); err == nil {
		for _, e := range c.Entries {
			row.Census = append(row.Census, censusRowView{
				Kind: e.Kind, Key: e.Key, Href: subjectHref(e.Kind, e.Key),
			})
		}
	}
	return row
}

// toDeliveryView renders one delivery outcome for the panel. The channel is
// shown by host only — never the full URL — so a token an operator embedded in a
// webhook path is not printed on a viewer-visible surface. The last error is the
// drill-down reason on a failed delivery, carried but not rendered as a top-level
// log (#22).
func toDeliveryView(o db.ListDeliveryOutcomesRow) deliveryView {
	// Never echo the raw URL: an operator may have embedded a token in the path
	// or query. Render the host alone, and a neutral placeholder if the URL does
	// not parse to one — never the raw string, which is where a token would sit.
	host := "(channel)"
	if u, err := url.Parse(o.Url); err == nil && u.Host != "" {
		host = u.Host
	}
	v := deliveryView{
		ChannelHost: host,
		State:       o.State,
		Failed:      o.State == "undelivered",
	}
	if v.Failed && o.LastError.Valid {
		v.LastError = o.LastError.String
	}
	return v
}

// messageLink resolves a message row's link target per §5.3's per-mover rule:
//
//   - drift and threshold link to an object's own page (the object that moved,
//     or the object whose span the rule read);
//   - declared-input links to the Source the rule reads;
//   - aperture-widening links to the Seed whose scope moved — the seed-scoped
//     anchor on the Seeds list, never the bare list and never Coverage's standing
//     aperture statement (which is constant and would lose which act the message
//     was about).
//
// LinkText is the key the link points at.
func messageLink(cause message.Cause, subjectKind, firedAt string) (href, text string) {
	switch message.LinkKindForCause(cause) {
	case message.LinkSource:
		return "/sources", firedAt
	case message.LinkSeed:
		// firedAt is the Seed's scope key (subject_kind "seed"); anchor to its row
		// on the canonical Scope page (#286) so the operator lands on the exact
		// Seed whose scope moved.
		return "/scope#seed-" + seedAnchor(firedAt), firedAt
	default: // LinkObject
		if h := subjectHref(subjectKind, firedAt); h != "" {
			return h, firedAt
		}
		// The subject list folded into Inventory (#286); a subject with no detail
		// page lands there rather than on the retired /subjects list.
		return "/inventory", firedAt
	}
}

// subjectHref builds the drill-down URL for a subject of the given kind, or ""
// where the kind has no page (a facet or signal in a census). A Service key
// carries a `/` and an Endpoint key a `/` and an `@`, so each rides a query
// parameter escaped; a Name or Address is a single path segment. Routing an
// Endpoint through the path form yields a two-segment URL the `/subjects/{key}`
// route does not match, falling through to the root 404 (#248).
func subjectHref(kind, key string) string {
	switch kind {
	case "service":
		return "/subjects/service?key=" + url.QueryEscape(key)
	case "endpoint":
		return "/subjects/endpoint?key=" + url.QueryEscape(key)
	case "name", "address":
		return "/subjects/" + url.PathEscape(key)
	default:
		return ""
	}
}

// reportDeliveryHref is the stable route the Reports screen's "View last delivery"
// row-menu item opens — the delivered report artifact built in T3 (#298). It is a
// constant because the artifact view resolves the account's latest delivery
// itself; the menu item only decides whether a delivery exists to open.
const reportDeliveryHref = "/reports/delivery"

// reportScheduleRow is one recurring report shaped for the Reports screen's
// "Recurring reports" table (T17, after design-system/examples/console/Reports.jsx).
// Name / Cadence / Format / LastSent are the schedule's facts; the row-action menu
// carries "View last delivery", which opens the delivered artifact when this report
// has a delivery and is disabled where it has none (no fabrication).
type reportScheduleRow struct {
	// ID keys the row's mutations — the Run now / Edit / Delete row-menu actions post
	// it back so the handler resolves which schedule to act on (P0.6/T4).
	ID       int64
	Name     string
	Cadence  string
	Format   string
	LastSent string
	// HasDelivery is true where a last delivery exists for this report; DeliveryHref
	// is the artifact route the menu item opens (reportDeliveryHref when a delivery
	// exists, empty otherwise so the item renders disabled).
	HasDelivery  bool
	DeliveryHref string
}

// lastReportDelivery resolves the "View last delivery" target for one report from
// its deliveries. Deliveries are messages (ADR-0039, ADR-0081): a report whose run
// was actually delivered — at least one outcome that is not undelivered — opens the
// delivered artifact at the stable /reports/delivery route (T3). A report with no
// delivery yet returns no link, so the menu item renders disabled rather than
// fabricating one; a menu never destroys and never invents (ADR-0110).
func lastReportDelivery(deliveries []deliveryView) (href string, has bool) {
	for _, d := range deliveries {
		if !d.Failed {
			return reportDeliveryHref, true
		}
	}
	return "", false
}

// reportScheduleRows assembles the "Recurring reports" table for the Reports screen
// (T17, wired in #290). It lists the declared schedules newest-first and maps each
// to a render row. A schedule's "last sent" and "View last delivery" now resolve
// from the report_delivery receipts store (#291/T2): GetLatestReportDelivery returns
// the newest non-failed run, whose instant fills the last-sent cell and lights the
// menu item at the stable /reports/delivery route. A schedule that has never run (or
// only failed) has no row to read, so the cell stays an em dash and the item renders
// disabled rather than fabricating a delivery (ADR-0110). Where there are no
// schedules the table renders the design-system empty-state. A read failure — of the
// schedule list or a per-schedule delivery — degrades rather than 500ing the
// analytics page a viewer depends on, matching the other best-effort reads here.
func (s *server) reportScheduleRows(ctx context.Context) []reportScheduleRow {
	schedules, err := s.store.ListReportSchedules(ctx)
	if err != nil {
		log.Printf("web: reports: list report schedules: %v", err)
		return nil
	}
	now := s.now()
	rows := make([]reportScheduleRow, 0, len(schedules))
	for _, sc := range schedules {
		// The latest non-failed run backs both the last-sent cell and the menu item.
		// No such run (pgx.ErrNoRows) is the genuine empty-state — an em dash and a
		// disabled item, never an invented delivery. Any other read error degrades the
		// row to that same empty-state rather than failing the whole page.
		lastSent, href, has := "—", "", false
		del, err := s.store.GetLatestReportDelivery(ctx, sc.ID)
		switch {
		case err == nil:
			lastSent = relTime(reportDeliveryInstant(del), now)
			href, has = reportDeliveryHref, true
		case !errors.Is(err, pgx.ErrNoRows):
			log.Printf("web: reports: latest delivery for schedule %d: %v", sc.ID, err)
		}
		rows = append(rows, reportScheduleRow{
			ID:           sc.ID,
			Name:         sc.Name,
			Cadence:      sc.Cadence,
			Format:       sc.Format,
			LastSent:     lastSent,
			HasDelivery:  has,
			DeliveryHref: href,
		})
	}
	return rows
}

// reportDeliveryInstant is the instant a receipt reads as "last sent": the delivery
// stamp where the run left (delivered_at), else the instant the artifact was cut
// (generated_at) for a run that generated without delivering. Both are present on a
// non-failed run, so the caller always has an instant to render.
func reportDeliveryInstant(d db.ReportDelivery) time.Time {
	if d.DeliveredAt.Valid {
		return d.DeliveredAt.Time
	}
	return d.GeneratedAt.Time
}
