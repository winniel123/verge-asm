package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// A delivery failure is legible on the Message it failed to carry, never on Coverage (ADR-0039).

type deliveryView struct {
	ChannelHost string

	// The outcomes projection carries no class or attempt time, so both stay empty (ADR-0110).

	Class     string
	When      string
	State     string
	Failed    bool
	LastError string
}

type messageRow struct {
	ID             int64
	Cause          string
	Class          string
	Headline       string
	Href           string
	LinkText       string
	Instant        string
	Read           bool
	Census         []censusRowView
	Deliveries     []deliveryView
	AnyUndelivered bool
}

type censusRowView struct {
	Kind string
	Key  string
	Href string
}

func (s *server) messagesPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: "messages"})
}

func (s *server) fillMessagesSection(r *http.Request, acct db.Account, data map[string]any) error {
	// The list is unbounded until an install shows the volume to size a cap (v1-spec §6.7).
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

func (s *server) markMessageRead(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.redirectBack(w, r, messagesFallback)
		return
	}
	if err := s.store.MarkMessageRead(r.Context(), db.MarkMessageReadParams{
		AccountID: acct.ID, MessageID: id, ReadAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "mark message read", err)
		return
	}
	s.redirectBack(w, r, messagesFallback)
}

func (s *server) markAllMessagesRead(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// Read state is per-account, so one operator's mark never clears another's unread badge (#327).
	if err := s.store.MarkAllMessagesRead(r.Context(), db.MarkAllMessagesReadParams{
		AccountID: acct.ID, ReadAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "mark all messages read", err)
		return
	}
	s.redirectBack(w, r, messagesFallback)
}

func (s *server) markMessageUnread(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.redirectBack(w, r, messagesFallback)
		return
	}
	if err := s.store.MarkMessageUnread(r.Context(), db.MarkMessageUnreadParams{
		AccountID: acct.ID, MessageID: id,
	}); err != nil {
		s.serverError(w, "mark message unread", err)
		return
	}
	s.redirectBack(w, r, messagesFallback)
}

const messagesFallback = "/messages"

type inboxView struct {
	messageRow
	Rel       string
	JumpLabel string
	Selected  bool
}

func (s *server) inboxPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "inbox", s.inboxFixtureData(acct, r))
		return
	}

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
			selected = &sel
		}
		if filter == "unread" && base.Read {
			continue
		}
		shown = append(shown, v)
	}

	allHref, unreadHref := "/inbox", "/inbox?filter=unread"
	if selID != 0 {
		idq := "id=" + strconv.FormatInt(selID, 10)
		allHref, unreadHref = "/inbox?"+idq, "/inbox?filter=unread&"+idq
	}

	s.render(w, r, "inbox", map[string]any{
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

func toDeliveryView(o db.ListDeliveryOutcomesRow) deliveryView {
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

func messageLink(cause message.Cause, subjectKind, firedAt string) (href, text string) {
	switch message.LinkKindForCause(cause) {
	case message.LinkSource:
		return "/sources", firedAt
	case message.LinkSeed:
		// Coverage's aperture is constant and cannot say which act fired (notification-channels §3.3).
		return "/scope#seed-" + seedAnchor(firedAt), firedAt
	default:
		if h := subjectHref(subjectKind, firedAt); h != "" {
			return h, firedAt
		}
		return "/inventory", firedAt
	}
}

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

// The artifact route is deliberately stable, so it carries no id of its own (#298).

const reportDeliveryHref = "/reports/delivery"

type reportScheduleRow struct {
	ID           int64
	Name         string
	Cadence      string
	Format       string
	LastSent     string
	LastMins     int
	Delivery     string
	HasDelivery  bool
	DeliveryHref string
}

func lastReportDelivery(deliveries []deliveryView) (href string, has bool) {
	for _, d := range deliveries {
		if !d.Failed {
			return reportDeliveryHref, true
		}
	}
	return "", false
}

func (s *server) reportScheduleRows(ctx context.Context) []reportScheduleRow {
	schedules, err := s.store.ListReportSchedules(ctx)
	if err != nil {
		log.Printf("web: reports: list report schedules: %v", err)
		return nil
	}
	channelURL := map[int64]string{}
	if channels, err := s.store.ListChannels(ctx); err != nil {
		log.Printf("web: reports: list channels for delivery column: %v", err)
	} else {
		for _, c := range channels {
			channelURL[c.ID] = c.Url
		}
	}

	now := s.now()
	rows := make([]reportScheduleRow, 0, len(schedules))
	for _, sc := range schedules {
		const reportScheduleNeverRunMins = 1 << 30
		// A never-run schedule sorts last, never as "just now" (ADR-0179 §1, #1363).
		lastSent, href, has := "—", "", false
		lastMins := reportScheduleNeverRunMins
		del, err := s.store.GetLatestReportDelivery(ctx, sc.ID)
		switch {
		case err == nil:
			inst := reportDeliveryInstant(del)
			lastSent = relTime(inst, now)
			if m := int(now.Sub(inst).Minutes()); m >= 0 {
				lastMins = m
			}
			href, has = reportDeliveryHref, true
		case !errors.Is(err, pgx.ErrNoRows):
			log.Printf("web: reports: latest delivery for schedule %d: %v", sc.ID, err)
		}
		delivery := "download only"
		if sc.ChannelID.Valid {
			if url, ok := channelURL[sc.ChannelID.Int64]; ok {
				delivery = url
			}
		}
		rows = append(rows, reportScheduleRow{
			ID:           sc.ID,
			Name:         sc.Name,
			Cadence:      sc.Cadence,
			Format:       sc.Format,
			LastSent:     lastSent,
			LastMins:     lastMins,
			Delivery:     delivery,
			HasDelivery:  has,
			DeliveryHref: href,
		})
	}
	return rows
}

func reportDeliveryInstant(d db.ReportDelivery) time.Time {
	if d.DeliveredAt.Valid {
		return d.DeliveredAt.Time
	}
	// The generated_at column is NOT NULL, so the fallback never needs a validity check.
	return d.GeneratedAt.Time
}
