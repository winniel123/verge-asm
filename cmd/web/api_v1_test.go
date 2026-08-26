package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// apiV1Resources is the settled read-only resource list A3 mounts (#662). Each is a thin
// JSON projection of a session-authed read the HTML surface already wraps.
var apiV1Resources = []string{
	"/api/v1/inventory",
	"/api/v1/subjects",
	"/api/v1/drift",
	"/api/v1/signals",
	"/api/v1/coverage",
}

// serveAPI drives one request through the FULL server mux (so the real /api/v1 routing,
// the apiBearer wrap, and the mux's own unknown-path/method behavior are all exercised),
// returning the recorder.
func serveAPI(t *testing.T, f *fakeStore, method, path, authz string) *httptest.ResponseRecorder {
	t.Helper()
	h := newServer(f, testKey, "", fixedClock()).handler()
	req := httptest.NewRequest(method, path, nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// Enabled + a valid Bearer token: every resource returns 200 with an application/json
// body that decodes, and never an HTML page or a redirect-to-signin.
func TestAPIv1ResourcesReturnJSON(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	for _, path := range apiV1Resources {
		rec := serveAPI(t, f, http.MethodGet, path, "Bearer "+apiTokenPlaintext)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200 (body %q)", path, rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("%s: content-type = %q, want application/json", path, ct)
		}
		var v any
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatalf("%s: body is not valid JSON: %v (%q)", path, err, rec.Body.String())
		}
	}
}

// Disabled (api_enabled=false) ⇒ every resource path 404s even with a valid token —
// surface-off beats auth, byte-indistinguishable from a build with no API (ADR-0123 §2).
// No 401/403 that would confirm the surface exists.
func TestAPIv1DisabledReturns404(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)
	f.instanceConfig.ApiEnabled = false // flip the surface off; the token stays valid

	for _, path := range apiV1Resources {
		rec := serveAPI(t, f, http.MethodGet, path, "Bearer "+apiTokenPlaintext)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s disabled: status = %d, want 404", path, rec.Code)
		}
	}
}

// Read-only surface: any non-GET verb is refused 405 (ADR-0123 §1). Every verb reaches
// apiBearer (the routes are registered method-less), so this holds for a live surface
// on a registered resource path.
func TestAPIv1NonGetReturns405(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rec := serveAPI(t, f, method, "/api/v1/inventory", "Bearer "+apiTokenPlaintext)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s /api/v1/inventory: status = %d, want 405", method, rec.Code)
		}
	}
}

// An enabled + authenticated request to a path under /api/v1 that names no resource
// falls through to the mux's 404 — indistinguishable from a disabled surface.
func TestAPIv1UnknownPathReturns404(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	rec := serveAPI(t, f, http.MethodGet, "/api/v1/does-not-exist", "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown /api/v1 path: status = %d, want 404", rec.Code)
	}
}
