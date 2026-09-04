package main

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

// Proxies and referrer policies strip Referer, so the URL rides this field instead (ADR-0130 §3).

const backField = "return" // A rename must move shell.tmpl's backfield define too.

func backURL(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	p := r.URL.EscapedPath()
	if p == "" {
		p = "/"
	}
	q := stripToastParam(r.URL.RawQuery)
	if q == "" {
		return p
	}
	return p + "?" + q
}

func stripToastParam(raw string) string { return stripParams(raw, "toast") }

func stripParams(raw string, names ...string) string {
	if raw == "" || len(names) == 0 {
		return raw
	}
	drop := func(key string) bool {
		for _, n := range names {
			if key == n {
				return true
			}
		}
		return false
	}
	var kept []string
	// Re-encoding sorts the query, and shell.tmpl's scroll key is a raw string compare (#970).
	for _, pair := range strings.Split(raw, "&") {
		if pair == "" {
			continue
		}
		key := pair
		if i := strings.Index(pair, "="); i >= 0 {
			key = pair[:i]
		}
		if name, err := url.QueryUnescape(key); err == nil {
			key = name
		}
		if drop(key) {
			continue
		}
		kept = append(kept, pair)
	}
	return strings.Join(kept, "&")
}

func stripDestParams(dest string, names ...string) string {
	i := strings.IndexByte(dest, '?')
	if i < 0 || len(names) == 0 {
		return dest
	}
	q := stripParams(dest[i+1:], names...)
	if q == "" {
		return dest[:i]
	}
	return dest[:i+1] + q
}

func (s *server) resolveBack(r *http.Request, fallback string) string {
	if r == nil {
		return fallback
	}
	raw := strings.TrimSpace(r.FormValue(backField))
	if raw == "" {
		return fallback
	}
	// A browser folds \ to / in a URL, so /\evil.example reaches an origin this server does not own.
	if strings.Contains(raw, `\`) {
		return fallback
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return fallback
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fallback
	}
	if u.Scheme != "" || u.Host != "" || u.User != nil || u.Opaque != "" || u.Fragment != "" {
		return fallback
	}
	if u.Path == "" || path.Clean(u.Path) != u.Path {
		return fallback
	}
	if !s.routeServesGET(u.Path) {
		return fallback
	}
	// Only the first toast is read (chrome.go), so a planted one beats the real receipt.
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		q := stripToastParam(raw[i+1:])
		if q == "" {
			return raw[:i]
		}
		return raw[:i+1] + q
	}
	return raw
}

func (s *server) routeServesGET(p string) bool {
	if s == nil || s.routes == nil {
		return false
	}
	probe := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: p},
		Host:   "localhost",
	}
	_, pattern := s.routes.Handler(probe)
	// An unmatched path yields an empty pattern and a redirect hop a bare one, neither with a method.
	if !strings.HasPrefix(pattern, "GET ") {
		return false
	}
	// The catch-all matches every path, but auth.go home answers 404 for anything but the root.
	if pattern == "GET /" {
		return p == "/"
	}
	return true
}

// The submitting URL is arbitrary, so no literal destination can express it (ADR-0130 §3).

func (s *server) redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	// gosec raises no finding here under CI's flags, so a waiver would silence nothing.
	http.Redirect(w, r, s.resolveBack(r, fallback), http.StatusSeeOther)
}

func (s *server) toastRedirectBack(w http.ResponseWriter, r *http.Request, fallback, tone, title, description string) {
	s.toastRedirect(w, r, s.resolveBack(r, fallback), tone, title, description)
}
