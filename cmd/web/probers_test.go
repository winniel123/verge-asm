package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func provision(t *testing.T, c *http.Client, base, host, port, username string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/probers", url.Values{
		"host": {host}, "port": {port}, "username": {username},
	})
}

func TestProvisionProber(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := provision(t, ac, base, "prober.example.com", "2222", "scanner")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/seeds" {
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

	// The provisioned endpoint is listed, and its key is "not set" until the
	// worker publishes a public half.
	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "prober.example.com:2222") {
		t.Errorf("prober endpoint not listed; body: %s", page)
	}
	if !strings.Contains(page, "not set") {
		t.Errorf("public key status not shown as 'not set'; body: %s", page)
	}
	// Availability starts pending — no connection has pinned a host key yet.
	if !strings.Contains(page, ">pending<") {
		t.Errorf("availability not shown as pending; body: %s", page)
	}
}

func TestProvisionRejectsRootAndBadPort(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Root username is refused, and the typed values are preserved.
	resp := provision(t, ac, base, "host", "22", "root")
	got := body(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "non-root") {
		t.Fatalf("root username not refused: status=%d body=%s", resp.StatusCode, got)
	}
	if !strings.Contains(got, `value="host"`) {
		t.Errorf("rejected host not retained; body: %s", got)
	}

	// Out-of-range port is refused.
	resp = provision(t, ac, base, "host", "70000", "scanner")
	if got := body(t, resp); resp.StatusCode != http.StatusBadRequest || !strings.Contains(got, "between 1 and 65535") {
		t.Fatalf("bad port not refused: status=%d body=%s", resp.StatusCode, got)
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
	resp := provision(t, ac, base, "host", "22", "scanner")
	if got := body(t, resp); !strings.Contains(got, "already provisioned") {
		t.Fatalf("duplicate endpoint not reported; body: %s", got)
	}
}

// TestPublicKeyShownAndPrivateNeverIs asserts the whole exposure rule: once the
// worker has published a public key, web renders it (status "set" plus the key),
// and no private key material is ever present on the page.
func TestPublicKeyShownAndPrivateNeverIs(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	adminID := f.byName["admin"]

	// A vantage the worker has already keyed: public half set, host key pinned.
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
	page := seedsBody(t, ac, base)

	if !strings.Contains(page, pub) {
		t.Errorf("public key not rendered; body: %s", page)
	}
	if !strings.Contains(page, ">set<") {
		t.Errorf("public key status not shown as 'set'; body: %s", page)
	}
	if !strings.Contains(page, ">available<") {
		t.Errorf("availability not shown; body: %s", page)
	}
	// Nothing that looks like a private key may ever appear.
	for _, marker := range []string{"PRIVATE KEY", "BEGIN OPENSSH PRIVATE", "host_key", "AAAAHOSTKEY"} {
		if strings.Contains(page, marker) {
			t.Errorf("page leaked private/host material %q; body: %s", marker, page)
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

	// The viewer can see the list but is not offered the provisioning form.
	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "prober.example.com:22") {
		t.Errorf("viewer cannot see provisioned probers; body: %s", page)
	}
	if strings.Contains(page, `action="/probers"`) {
		t.Errorf("provision form shown to a viewer; body: %s", page)
	}
}
