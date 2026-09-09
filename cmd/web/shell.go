package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

type shellStore interface {
	ListCurrentNameSubjects(ctx context.Context, arg db.ListCurrentNameSubjectsParams) ([]db.ListCurrentNameSubjectsRow, error)
	ListMessages(ctx context.Context) ([]db.Message, error)
}

type bellMessage struct {
	Class    string
	Headline string
	Rel      string
	Href     string
	Unread   bool
}

func (s *server) bellMessages(ctx context.Context, accountID int64, limit int) []bellMessage {
	rows, err := s.shellStore.ListMessages(ctx)
	if err != nil {
		log.Printf("web: shell: list messages for bell: %v", err)
		return nil
	}
	read, err := s.readSet(ctx, accountID)
	if err != nil {
		log.Printf("web: shell: read set for bell: %v", err)
		read = map[int64]bool{}
	}
	now := s.now()
	out := make([]bellMessage, 0, limit)
	for _, m := range rows {
		if len(out) >= limit {
			break
		}
		b := bellMessage{
			Class:    m.Class,
			Headline: m.Headline,
			Href:     "/inbox?id=" + strconv.FormatInt(m.ID, 10),
			Unread:   !read[m.ID],
		}
		if m.Instant.Valid {
			b.Rel = relTime(m.Instant.Time, now)
		}
		out = append(out, b)
	}
	return out
}

func accountInitials(name string) string {
	var b strings.Builder
	for _, field := range strings.Fields(name) {
		if b.Len() >= 2 {
			break
		}
		b.WriteString(strings.ToUpper(field[:1]))
	}
	if b.Len() == 0 {
		return "?"
	}
	return b.String()
}

func (s *server) toastRedirect(w http.ResponseWriter, r *http.Request, dest, tone, title, description string) {
	// A cookie carrier would break the HttpOnly invariant every cookie this server sets holds.
	payload, err := json.Marshal(map[string]string{"tone": tone, "title": title, "description": description})
	if err != nil {
		http.Redirect(w, r, dest, http.StatusSeeOther)
		return
	}
	sep := "?"
	if strings.Contains(dest, "?") {
		sep = "&"
	}
	http.Redirect(w, r, dest+sep+"toast="+base64.RawURLEncoding.EncodeToString(payload), http.StatusSeeOther)
}

func (s *server) flashRedirect(w http.ResponseWriter, r *http.Request, accountID int64, dest, tone, title, description string) {
	s.flash.set(accountID, toastVM{Tone: tone, Title: title, Description: description})
	http.Redirect(w, r, dest, http.StatusSeeOther)
}
