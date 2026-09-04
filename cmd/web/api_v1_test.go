package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

var apiV1Resources = []string{
	"/api/v1/inventory",
	"/api/v1/subjects",
	"/api/v1/drift",
	"/api/v1/signals",
	"/api/v1/coverage",
}

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

func TestAPIv1DisabledReturns404(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)
	f.instanceConfig.ApiEnabled = false

	for _, path := range apiV1Resources {
		rec := serveAPI(t, f, http.MethodGet, path, "Bearer "+apiTokenPlaintext)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s disabled: status = %d, want 404", path, rec.Code)
		}
	}
}

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

func TestAPIv1UnknownPathReturns404(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	rec := serveAPI(t, f, http.MethodGet, "/api/v1/does-not-exist", "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown /api/v1 path: status = %d, want 404", rec.Code)
	}
}
