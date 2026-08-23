package main

import (
	"crypto/rand"
	"encoding/binary"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
)

// The error pages (T11, #306): the three full-screen states share one render path
// through the "error-page" template (templates_error.go). Each is a chrome-less
// frame, so the data map carries no "IsAdmin"/nav keys — injectUnread leaves it
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
// an admin widens it.
func (s *server) forbidden(w http.ResponseWriter, _ *http.Request) {
	s.renderError(w, http.StatusForbidden, "403", "Access denied", "")
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
