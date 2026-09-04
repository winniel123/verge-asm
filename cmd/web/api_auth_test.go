package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

const apiTokenPlaintext = "vg_pat_deadbeefdeadbeefdeadbeef"

func runAPIBearer(t *testing.T, f *fakeStore, method, authz string) (*httptest.ResponseRecorder, *db.Account) {
	t.Helper()
	srv := newServer(f, testKey, "", fixedClock())
	var got *db.Account
	h := srv.apiBearer(func(w http.ResponseWriter, r *http.Request, acct db.Account) {
		a := acct
		got = &a
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(method, "/api/v1/subjects", nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec, got
}

func seedAPIToken(t *testing.T, f *fakeStore, role string) (db.Account, int64) {
	t.Helper()
	f.instanceConfig = db.GetInstanceConfigRow{ApiEnabled: true}
	acct := seedAccount(t, f, "ola", role, "hunter2hunter2")
	tok, err := f.CreatePersonalToken(t.Context(), db.CreatePersonalTokenParams{
		AccountID: acct.ID, Name: "cli", Prefix: "vg_pat_dead…", TokenHash: hashToken(apiTokenPlaintext),
	})
	if err != nil {
		t.Fatal(err)
	}
	return acct, tok.ID
}

func (f *fakeStore) tokenByID(id int64) (db.PersonalToken, bool) {
	for _, t := range f.personalTokens {
		if t.ID == id {
			return t, true
		}
	}
	return db.PersonalToken{}, false
}

func TestAPIBearerEnabledValidResolvesAndTouches(t *testing.T) {
	f := newFakeStore()
	acct, tokID := seedAPIToken(t, f, roleViewer)

	rec, got := runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got == nil {
		t.Fatal("read handler was not reached; caller not resolved")
	}
	if got.ID != acct.ID || got.Role != roleViewer {
		t.Fatalf("resolved account = %+v, want id=%d role=%s", got, acct.ID, roleViewer)
	}
	tok, _ := f.tokenByID(tokID)
	if !tok.LastUsedAt.Valid {
		t.Fatal("last_used_at not touched on the first authenticated request")
	}
	first := tok.LastUsedAt.Time

	rec2, got2 := runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if rec2.Code != http.StatusOK || got2 == nil {
		t.Fatalf("second request: status=%d resolved=%v, want 200 + resolved", rec2.Code, got2 != nil)
	}
	tok2, _ := f.tokenByID(tokID)
	if !tok2.LastUsedAt.Time.Equal(first) {
		t.Fatalf("last_used_at re-touched within the hour: %v -> %v", first, tok2.LastUsedAt.Time)
	}
}

func TestAPIBearerRoleReadLive(t *testing.T) {
	f := newFakeStore()
	acct, _ := seedAPIToken(t, f, roleAdmin)

	_, got := runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if got == nil || got.Role != roleAdmin {
		t.Fatalf("first resolve role = %v, want admin", got)
	}
	a := f.accounts[acct.ID]
	a.Role = roleViewer
	f.accounts[acct.ID] = a

	_, got2 := runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if got2 == nil || got2.Role != roleViewer {
		t.Fatalf("after demotion role = %v, want viewer (role must be read live)", got2)
	}
}

func TestAPIBearerDisabled404(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)
	f.instanceConfig = db.GetInstanceConfigRow{ApiEnabled: false}

	rec, got := runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("disabled GET status = %d, want 404", rec.Code)
	}
	if got != nil {
		t.Fatal("read handler reached on a disabled surface")
	}
	rec2, _ := runAPIBearer(t, f, http.MethodPost, "Bearer "+apiTokenPlaintext)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("disabled POST status = %d, want 404", rec2.Code)
	}
}

func TestAPIBearerRemovedAccount401(t *testing.T) {
	f := newFakeStore()
	f.instanceConfig = db.GetInstanceConfigRow{ApiEnabled: true}
	tok, err := f.CreatePersonalToken(t.Context(), db.CreatePersonalTokenParams{
		AccountID: 999, Name: "orphan", Prefix: "vg_pat_dead…", TokenHash: hashToken(apiTokenPlaintext),
	})
	if err != nil {
		t.Fatal(err)
	}

	rec, got := runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("removed-account status = %d, want 401", rec.Code)
	}
	if got != nil {
		t.Fatal("read handler reached for a removed account")
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("401 missing WWW-Authenticate challenge")
	}
	if cur, _ := f.tokenByID(tok.ID); cur.LastUsedAt.Valid {
		t.Fatal("last_used_at touched despite a failed resolution")
	}
}

func TestAPIBearerNonGET405(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec, got := runAPIBearer(t, f, method, "Bearer "+apiTokenPlaintext)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", method, rec.Code)
		}
		if got != nil {
			t.Fatalf("%s reached the read handler", method)
		}
		if rec.Header().Get("Allow") != http.MethodGet {
			t.Fatalf("%s Allow header = %q, want GET", method, rec.Header().Get("Allow"))
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("%s 405 wrote a body: %q (want no body detail)", method, rec.Body.String())
		}
	}
}

func TestAPIBearerRejectsNonBearerCredentials(t *testing.T) {
	f := newFakeStore()
	seedAPIToken(t, f, roleViewer)

	cases := []struct {
		name  string
		authz string
	}{
		{"missing header", ""},
		{"basic scheme", "Basic dXNlcjpwYXNz"},
		{"bare token no scheme", apiTokenPlaintext},
		{"wrong prefix", "Bearer not_a_pat_token"},
		{"unknown token", "Bearer vg_pat_0000000000000000000000"},
	}
	for _, tc := range cases {
		rec, got := runAPIBearer(t, f, http.MethodGet, tc.authz)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s: status = %d, want 401", tc.name, rec.Code)
		}
		if got != nil {
			t.Fatalf("%s: reached the read handler", tc.name)
		}
	}

	srv := newServer(f, testKey, "", fixedClock())
	h := srv.apiBearer(func(w http.ResponseWriter, r *http.Request, _ db.Account) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/subjects", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookie, Value: "whatever"})
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("session-cookie-only status = %d, want 401 (cookie must not be accepted)", rec.Code)
	}
}

func TestAPIBearerNeverUsedIsNull(t *testing.T) {
	f := newFakeStore()
	_, tokID := seedAPIToken(t, f, roleViewer)
	if tok, _ := f.tokenByID(tokID); tok.LastUsedAt != (pgtype.Timestamptz{}) {
		t.Fatalf("freshly minted token has non-null last_used_at: %+v", tok.LastUsedAt)
	}
	runAPIBearer(t, f, http.MethodGet, "Bearer "+apiTokenPlaintext)
	if tok, _ := f.tokenByID(tokID); !tok.LastUsedAt.Valid {
		t.Fatal("last_used_at still null after first use")
	}
}
