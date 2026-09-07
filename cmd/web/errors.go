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

// A signed-out reader has no console to return to, so this page carries no chrome (#533, #1358).

func (s *server) renderError(w http.ResponseWriter, r *http.Request, status int, kind, title, incidentID string) {
	data := barePageData(title)
	if acct, ok := s.currentAccount(r); ok {
		data = pageData(acct, title, "")
	}
	data["Kind"] = kind
	data["IncidentID"] = incidentID
	s.renderStatus(w, r, status, "error-page", data)
}

func (s *server) notFound(w http.ResponseWriter, r *http.Request) {
	s.renderError(w, r, http.StatusNotFound, "404", "Page not found", "")
}

func (s *server) forbidden(w http.ResponseWriter, r *http.Request) {
	s.renderError(w, r, http.StatusForbidden, "403", "Access denied", "")
}

// A withdrawn subject is still reachable by its key, so this state is only an unmatched key (#480).

func (s *server) renderMissingSubject(w http.ResponseWriter, r *http.Request, acct db.Account, subject string) {
	s.renderStatus(w, r, http.StatusNotFound, "error-page", pageData(acct, "No such subject", "", map[string]any{
		"Kind":        "missing-subject",
		"Subject":     subject,
		"ActionLabel": "Back to inventory",
		"ActionHref":  "/inventory",
	}))
}

func (s *server) renderMissingRun(w http.ResponseWriter, r *http.Request, acct db.Account, run string) {
	s.renderStatus(w, r, http.StatusNotFound, "error-page", pageData(acct, "No such run", "drift", map[string]any{
		"Kind":        "missing-run",
		"Subject":     "run #" + run,
		"ActionLabel": "Back to drift",
		"ActionHref":  "/drift",
	}))
}

// Settings alone renders the richer refusal; every other admin route keeps the plain 403 (#481).

func (s *server) settingsForbidden(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderStatus(w, r, http.StatusForbidden, "error-page", pageData(acct, "Admin only", "", map[string]any{
		"Kind":        "settings-forbidden",
		"Code":        "403",
		"ActionLabel": "Back to dashboard",
		"ActionHref":  "/",
	}))
}

func newIncidentID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A failed entropy draw must not swallow the incident, so the id stays copyable.
		return "err_00000000"
	}
	s := strconv.FormatUint(binary.BigEndian.Uint64(b[:]), 36)
	for len(s) < 8 {
		s = "0" + s
	}
	return "err_" + s[len(s)-8:]
}

func (s *server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				id := newIncidentID()
				if s.devMode {
					id = devFixtureIncidentID
				}
				log.Printf("web: incident %s: recovered panic on %s %s: %v\n%s", // #nosec G706 (sanitized via logSafe)
					id, r.Method, logSafe(r.URL.Path), rec, debug.Stack())
				s.renderError(w, r, http.StatusInternalServerError, "500", "Something broke", id)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
