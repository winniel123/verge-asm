package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
)

// The console shell (templates_shell.go, PARITY-CHART P1) is normative for look AND
// functionality (ADR-0116). Three of its affordances ride global reads the shell
// renders on every screen — the inbox bell's recent-messages menu (P1.3), the org
// switcher's asset count (P1.4), and the command palette's Assets group (P1.5). This
// file assembles those reads and the toast-flash channel (P1.7); injectChrome wires
// them onto every chrome page's data. Every read is best-effort: a failure degrades
// the affordance to its empty form (the spec's own pattern) rather than 500ing a page.

// bellMessage is one message shaped for the top-nav bell menu — TopNav.jsx's
// MessageList, whose onOpenMessage deep-links each row into the Inbox. It carries
// the class tag, the headline, a relative instant, the per-message /inbox?id= link,
// and the caller's own unread state (read-state is per-account, #327).
type bellMessage struct {
	Class    string
	Headline string
	Rel      string
	Href     string
	Unread   bool
}

// bellMessages resolves the recent messages the bell menu shows — the newest few,
// each deep-linking into the Inbox. It is the read behind TopNav's onOpenMessage /
// "All messages" menu. Best-effort: on a read failure the menu renders its empty
// state rather than failing the page it rides.
func (s *server) bellMessages(ctx context.Context, accountID int64, limit int) []bellMessage {
	rows, err := s.store.ListMessages(ctx)
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

// paletteAsset is one current Name subject shaped for the command palette's Assets
// group (ConsoleApp.jsx): its key and the /asset/{key} detail link. "Asset" is the
// UI collective noun for a current Name subject — the same corpus the search screen
// indexes as Assets (search.go).
type paletteAsset struct {
	Key  string
	Href string
}

// currentAssets reads the current Name subjects the org switcher counts (P1.4) and
// the palette's Assets group lists (P1.5). It returns the top-N for the palette and
// the full count for the switcher, from the one census so the shell reads it once.
// Best-effort: on a read failure the count is zero and the group is empty, and both
// affordances degrade to the spec's own absent-count / absent-group forms.
func (s *server) currentAssets(ctx context.Context, limit int) (top []paletteAsset, count int) {
	rows, err := s.store.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	})
	if err != nil {
		log.Printf("web: shell: list name subjects for palette: %v", err)
		return nil, 0
	}
	top = make([]paletteAsset, 0, limit)
	for _, row := range rows {
		if len(top) >= limit {
			break
		}
		top = append(top, paletteAsset{Key: row.SubjectKey, Href: "/asset/" + url.PathEscape(row.SubjectKey)})
	}
	return top, len(rows)
}

// accountInitials derives the avatar's initials from the signed-in account's name,
// exactly as design-system Avatar.jsx does: the first letter of up to two
// whitespace-separated words, uppercased, with "?" as the empty fallback. It
// replaces the literal "VA" placeholder the shell shipped (P1.8).
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

// toastRedirect issues the post-redirect-get for an act and carries a toast for the
// destination page to fire (PARITY-CHART P1.7; ConsoleApp.jsx fires the same toasts
// on scan/save). The toast rides the redirect URL's `toast` query — a base64url
// JSON blob — rather than a cookie: the shell's script reads it on load, fires the
// toast, and strips it from the address bar (history.replaceState) so a refresh does
// not re-toast. Carrying it in the URL (not a cookie) keeps every cookie this server
// sets HttpOnly, and the text is rendered as textContent (never HTML) so it cannot
// inject. tone is one of neutral / ok / warn / danger (Toast.jsx). On any encoding
// failure the redirect still lands, just without the toast.
func (s *server) toastRedirect(w http.ResponseWriter, r *http.Request, dest, tone, title, description string) {
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
