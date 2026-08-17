package main

import (
	"net/http"
	"net/url"
	"strconv"

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
	rows, err := s.store.ListMessages(r.Context())
	if err != nil {
		s.serverError(w, "list messages", err)
		return
	}
	unread, err := s.store.CountUnreadMessages(r.Context())
	if err != nil {
		s.serverError(w, "count unread", err)
		return
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
		row := toMessageRow(m)
		row.Deliveries = byMessage[m.ID]
		for _, d := range row.Deliveries {
			if d.Failed {
				row.AnyUndelivered = true
			}
		}
		views = append(views, row)
	}
	s.render(w, "messages", map[string]any{
		"Title": "Messages", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Messages": views, "Unread": unread,
	})
}

// markMessageRead marks one message read at now and returns to the panel. Read
// state is a per-account fact; there is no un-read, since a message is read once
// the operator has seen it.
func (s *server) markMessageRead(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/messages", http.StatusSeeOther)
		return
	}
	if err := s.store.MarkMessageRead(r.Context(), db.MarkMessageReadParams{
		ID: id, ReadAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "mark message read", err)
		return
	}
	http.Redirect(w, r, "/messages", http.StatusSeeOther)
}

// markAllMessagesRead clears the unread count in one act.
func (s *server) markAllMessagesRead(w http.ResponseWriter, r *http.Request, _ db.Account) {
	if err := s.store.MarkAllMessagesRead(r.Context(), pgtype.Timestamptz{Time: s.now(), Valid: true}); err != nil {
		s.serverError(w, "mark all messages read", err)
		return
	}
	http.Redirect(w, r, "/messages", http.StatusSeeOther)
}

// toMessageRow renders one stored message for the panel, resolving its link per
// its mover and unpacking its census.
func toMessageRow(m db.Message) messageRow {
	row := messageRow{
		ID:       m.ID,
		Cause:    m.Cause,
		Class:    m.Class,
		Headline: m.Headline,
		Read:     m.ReadAt.Valid,
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
		// so the operator lands on the exact Seed whose scope moved.
		return "/seeds#seed-" + seedAnchor(firedAt), firedAt
	default: // LinkObject
		if h := subjectHref(subjectKind, firedAt); h != "" {
			return h, firedAt
		}
		return "/subjects", firedAt
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
