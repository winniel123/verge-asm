package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func provision(t *testing.T, c *http.Client, base, host, port, username string) *http.Response {
	t.Helper()
	return provisionWithResolver(t, c, base, host, port, username, "9.9.9.9:53")
}

func provisionWithResolver(t *testing.T, c *http.Client, base, host, port, username, resolver string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/settings/probers", url.Values{
		"host": {host}, "port": {port}, "username": {username}, "resolver": {resolver},
	})
}

func setResolver(t *testing.T, c *http.Client, base, id, resolver string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/settings/vantages/resolver", url.Values{
		"id": {id}, "resolver": {resolver},
	})
}

const vantagesTab = "/settings?tab=vantages"

func vantagesBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	return getBody(t, c, base+vantagesTab, http.StatusOK)
}

func TestProvisionProber(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := provision(t, ac, base, "prober.example.com", "2222", "scanner")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != vantagesTab {
		t.Fatalf("provision: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	if len(f.vantages) != 1 {
		t.Fatalf("vantages = %d, want 1", len(f.vantages))
	}
	v := f.vantages[0]
	if v.Host.String != "prober.example.com" || v.Port.Int32 != 2222 || v.Username.String != "scanner" {
		t.Errorf("vantage row = %+v, want host/port/username as provisioned", v)
	}
	if v.Resolver != "9.9.9.9:53" {
		t.Errorf("resolver = %q, want the declared 9.9.9.9:53 — a vantage without one must not exist", v.Resolver)
	}

	page := vantagesBody(t, ac, base)
	if !strings.Contains(page, "prober.example.com:2222") {
		t.Errorf("prober endpoint not listed; body: %s", page)
	}
	if !strings.Contains(page, "not set") {
		t.Errorf("public key status not shown as 'not set'; body: %s", page)
	}
	if !strings.Contains(page, ">pending<") {
		t.Errorf("availability not shown as pending; body: %s", page)
	}
}

func TestProvisionRejectsRootAndBadPort(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	if loc := submitLoc(t, provision(t, ac, base, "host", "22", "root")); loc != vantagesTab {
		t.Fatalf("refused provision landed at %q, want %q", loc, vantagesTab)
	}
	got := vantagesBody(t, ac, base)
	if !strings.Contains(got, "non-root") {
		t.Fatalf("root username not refused; body: %s", got)
	}
	if !strings.Contains(got, `value="host"`) {
		t.Errorf("rejected host not retained; body: %s", got)
	}

	if loc := submitLoc(t, provision(t, ac, base, "host", "70000", "scanner")); loc != vantagesTab {
		t.Fatalf("refused port landed at %q, want %q", loc, vantagesTab)
	}
	if got := vantagesBody(t, ac, base); !strings.Contains(got, "between 1 and 65535") {
		t.Fatalf("bad port not refused; body: %s", got)
	}

	if len(f.vantages) != 0 {
		t.Fatalf("vantages after rejected provisions = %d, want 0", len(f.vantages))
	}
}

func TestProvisionDuplicateRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	provision(t, ac, base, "host", "22", "scanner").Body.Close()
	provision(t, ac, base, "host", "22", "scanner").Body.Close()
	got := vantagesBody(t, ac, base)
	if !strings.Contains(got, "already provisioned") {
		t.Fatalf("duplicate endpoint not reported; body: %s", got)
	}
	if again := vantagesBody(t, ac, base); strings.Contains(again, "already provisioned") {
		t.Fatalf("the callout survived a reload; body: %s", again)
	}
}

func TestPublicKeyShownAndPrivateNeverIs(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	adminID := f.byName["admin"]

	pub := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITESTPUBLICKEYVALUE example"
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "scanner@prober.example.com:22", Class: "unverified",
		Host:         pgtype.Text{String: "prober.example.com", Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "scanner", Valid: true},
		Availability: pgtype.Text{String: "available", Valid: true},
		PublicKey:    pgtype.Text{String: pub, Valid: true},
		HostKey:      pgtype.Text{String: "ssh-ed25519 AAAAHOSTKEY", Valid: true},
		CreatedBy:    pgtype.Int8{Int64: adminID, Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	f.vantageNextID = 2

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := vantagesBody(t, ac, base)

	if !strings.Contains(page, pub) {
		t.Errorf("public key not rendered; body: %s", page)
	}
	if !strings.Contains(page, "authorized_keys") {
		t.Errorf("public key not presented for install (authorized_keys block); body: %s", page)
	}
	if !strings.Contains(page, ">available<") {
		t.Errorf("availability not shown; body: %s", page)
	}
	for _, marker := range []string{"PRIVATE KEY", "BEGIN OPENSSH PRIVATE", "host_key", "AAAAHOSTKEY"} {
		if strings.Contains(page, marker) {
			t.Errorf("page leaked private/host material %q; body: %s", marker, page)
		}
	}
}

func TestVantagesTabRevealsPublicKeyNotHostKey(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	adminID := f.byName["admin"]
	pub := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITESTPUBLICKEYVALUE example"
	f.vantages = append(f.vantages, db.Vantage{
		ID: 1, Name: "scanner@prober.example.com:22", Class: "unverified",
		Host:         pgtype.Text{String: "prober.example.com", Valid: true},
		Port:         pgtype.Int4{Int32: 22, Valid: true},
		Username:     pgtype.Text{String: "scanner", Valid: true},
		Availability: pgtype.Text{String: "available", Valid: true},
		PublicKey:    pgtype.Text{String: pub, Valid: true},
		HostKey:      pgtype.Text{String: "ssh-ed25519 AAAAHOSTKEY", Valid: true},
		CreatedBy:    pgtype.Int8{Int64: adminID, Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	f.vantageNextID = 2

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/settings?tab=vantages", http.StatusOK)

	if !strings.Contains(page, pub) {
		t.Errorf("public key not revealed on the vantages tab; body: %s", page)
	}
	if !strings.Contains(page, "pinned") {
		t.Errorf("host-key pin status not shown; body: %s", page)
	}
	for _, marker := range []string{"PRIVATE KEY", "AAAAHOSTKEY", "host_key"} {
		if strings.Contains(page, marker) {
			t.Errorf("vantages tab leaked private/host material %q", marker)
		}
	}
}

func TestViewerCannotProvisionButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	provision(t, ac, base, "prober.example.com", "22", "scanner").Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := provision(t, vc, base, "other.example.com", "22", "scanner")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer provision: status=%d, want 403", resp.StatusCode)
	}
	if len(f.vantages) != 1 {
		t.Fatalf("vantages after denied provision = %d, want 1", len(f.vantages))
	}

	resp = get(t, vc, base+"/settings?tab=vantages")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer reached the admin-only vantages surface: status=%d, want 403", resp.StatusCode)
	}
}

func TestProvisionRequiresAResolver(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	for _, bad := range []string{"", "   ", "https://dns.example/x", "9.9.9.9:0"} {
		if loc := submitLoc(t, provisionWithResolver(t, ac, base, "probe.example.net", "22", "scanner", bad)); loc != vantagesTab {
			t.Fatalf("refused provision landed at %q, want %q", loc, vantagesTab)
		}
		if got := vantagesBody(t, ac, base); !strings.Contains(got, "resolver") && !strings.Contains(got, "1 and 65535") {
			t.Errorf("resolver %q was not refused with a reason; body: %s", bad, got)
		}
		if len(f.vantages) != 0 {
			t.Fatalf("resolver %q created a vantage; vantages = %d, want 0", bad, len(f.vantages))
		}
	}
}

func TestSetVantageResolver(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	provision(t, ac, base, "probe.example.net", "22", "scanner").Body.Close()
	id := strconv.FormatInt(f.vantages[0].ID, 10)

	if loc := submitLoc(t, setResolver(t, ac, base, id, " 1.1.1.1:53 ")); loc != vantagesTab {
		t.Fatalf("set resolver landed at %q, want %q", loc, vantagesTab)
	}
	if got := f.vantages[0].Resolver; got != "1.1.1.1:53" {
		t.Fatalf("resolver = %q, want the trimmed 1.1.1.1:53", got)
	}
	if page := vantagesBody(t, ac, base); !strings.Contains(page, "1.1.1.1:53") {
		t.Errorf("the new resolver is not shown; body: %s", page)
	}

	if loc := submitLoc(t, setResolver(t, ac, base, id, "")); loc != vantagesTab {
		t.Fatalf("refused resolver landed at %q, want %q", loc, vantagesTab)
	}
	if got := f.vantages[0].Resolver; got != "1.1.1.1:53" {
		t.Errorf("a refused edit changed the row to %q", got)
	}
	if page := vantagesBody(t, ac, base); !strings.Contains(page, "recursive resolver is required") {
		t.Errorf("blank resolver not refused with a reason; body: %s", page)
	}
}

func TestViewerCannotSetAResolver(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	provision(t, ac, base, "probe.example.net", "22", "scanner").Body.Close()
	id := strconv.FormatInt(f.vantages[0].ID, 10)

	vc := login(t, base, "viewer", "hunter2hunter2")
	resp := setResolver(t, vc, base, id, "1.1.1.1:53")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer set resolver: status=%d, want 403", resp.StatusCode)
	}
	if got := f.vantages[0].Resolver; got != "9.9.9.9:53" {
		t.Errorf("a viewer changed the resolver to %q", got)
	}
}
