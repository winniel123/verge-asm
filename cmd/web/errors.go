package main

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

// The error pages (T11, #306): the three full-screen states share one render path
// through the "error-page" template (templates_error.go). Each is a chrome-less
// frame, so the data map carries no "IsAdmin"/nav keys — injectChrome leaves it
// alone and no chrome read runs. Only the 500 carries an incident id.

// renderError writes the shared error page at the given HTTP status. kind is the
// "404" / "403" / "500" string the template branches on; incidentID is rendered
// only when non-empty (the 500 path). Title feeds the <head> so the browser tab
// names the state.
func (s *server) renderError(w http.ResponseWriter, status int, kind, title, incidentID string) {
	s.renderStatus(w, status, "error-page", map[string]any{
		"Title":      title,
		"Kind":       kind,
		"IncidentID": incidentID,
	})
}

// notFound answers an unknown path with the 404 error page. It replaces the
// scaffold's plain-text http.NotFound so an unmatched URL lands on the same frame
// as every other state, in both themes.
func (s *server) notFound(w http.ResponseWriter, _ *http.Request) {
	s.renderError(w, http.StatusNotFound, "404", "Page not found", "")
}

// forbidden answers an unauthorized request with the 403 error page. It is the one
// render behind requireAdmin: a viewer who reaches an admin act sees why, and how
// an admin widens it. The Settings destination renders the richer settingsForbidden
// instead (U4, #481); every other admin route keeps this plain 403.
func (s *server) forbidden(w http.ResponseWriter, _ *http.Request) {
	s.renderError(w, http.StatusForbidden, "403", "Access denied", "")
}

// renderMissingSubject renders the missing-subject ErrorPage kind (U3, #480) at 404,
// inside the console chrome (screenshot 26): the scan-search badge, the subject key
// the caller keyed but nothing ever measured shown big-mono, and the way back to
// Inventory. It is distinct from a withdrawn subject, which is still reachable by its
// own key — this state is only for a key that matched no subject at all. subject is
// the unmatched key the operator asked for.
func (s *server) renderMissingSubject(w http.ResponseWriter, acct db.Account, subject string) {
	s.renderStatus(w, http.StatusNotFound, "error-page", map[string]any{
		"Title":       "No such subject",
		"Kind":        "missing-subject",
		"Subject":     subject,
		"ActionLabel": "Back to inventory",
		"ActionHref":  "/inventory",
		"Account":     acct,
		"IsAdmin":     acct.Role == roleAdmin,
	})
}

// renderMissingRun renders the missing-run ErrorPage kind (U3, #480) at 404, inside
// the console chrome: the history badge, the run id shown big-mono as `run #<id>`,
// and the way back to Drift. It stands where a run id matched no Dispatch in recent
// history. run is the raw id the operator asked for.
func (s *server) renderMissingRun(w http.ResponseWriter, acct db.Account, run string) {
	s.renderStatus(w, http.StatusNotFound, "error-page", map[string]any{
		"Title":       "No such run",
		"Kind":        "missing-run",
		"Subject":     "run #" + run,
		"ActionLabel": "Back to drift",
		"ActionHref":  "/drift",
		"NavActive":   "drift",
		"Account":     acct,
		"IsAdmin":     acct.Role == roleAdmin,
	})
}

// settingsForbidden renders the settings-forbidden ErrorPage kind (U4, #481) at 403,
// inside the console chrome (screenshot 27): the lock badge, the 403 code, and the
// "admin only — Settings is where declared acts live" copy that names how an admin
// widens a viewer's role. It is the ONE admin surface that renders this richer copy;
// requireAdmin's plain 403 (forbidden) still stands behind every other admin route.
func (s *server) settingsForbidden(w http.ResponseWriter, acct db.Account) {
	s.renderStatus(w, http.StatusForbidden, "error-page", map[string]any{
		"Title":       "Admin only",
		"Kind":        "settings-forbidden",
		"Code":        "403",
		"ActionLabel": "Back to dashboard",
		"ActionHref":  "/",
		"Account":     acct,
		"IsAdmin":     acct.Role == roleAdmin,
	})
}

// newIncidentID mints the id the 500 page shows and the host log records. It is a
// short, copy-friendly token (err_ + 8 base36 chars from a crypto/rand draw), the
// shape ErrorPage.jsx samples (err_9f3ka72c). It is a real id minted at the point
// of failure, never fabricated for a page that did not actually fail.
func newIncidentID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A failed entropy draw must not swallow the incident — fall back to a
		// fixed-width marker so the id is still present and copyable.
		return "err_00000000"
	}
	s := strconv.FormatUint(binary.BigEndian.Uint64(b[:]), 36)
	for len(s) < 8 {
		s = "0" + s
	}
	return "err_" + s[len(s)-8:]
}

// recoverPanics wraps the mux so a handler panic becomes the 500 error page rather
// than a dropped connection. It mints one incident id, logs it with the recovered
// value and the stack (so the host log the page points to actually carries it), and
// renders the page carrying that same real id. Wired once at the mux-construction
// boundary (handlers.go).
func (s *server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				id := newIncidentID()
				log.Printf("web: incident %s: recovered panic on %s %s: %v\n%s",
					id, r.Method, r.URL.Path, rec, debug.Stack())
				s.renderError(w, http.StatusInternalServerError, "500", "Something broke", id)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
