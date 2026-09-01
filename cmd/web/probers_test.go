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

// Prober provisioning relocated from /scope to Settings → Vantages (#21d): the act
// posts /settings/probers and the region renders under /settings?tab=vantages.
func provision(t *testing.T, c *http.Client, base, host, port, username string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/settings/probers", url.Values{
		"host": {host}, "port": {port}, "username": {username},
	})
}

// vantagesTab is the section's own URL, and the fallback destination of a provision
// submitted with no `return` field.
const vantagesTab = "/settings?tab=vantages"

// vantagesBody reads the Settings → Vantages tab, where prober provisioning + the
// prober listing now live.
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

	// The provisioned endpoint is listed, and its key is "not set" until the
	// worker publishes a public half.
	page := vantagesBody(t, ac, base)
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

// A refused provision is a post-redirect-get since ticket #978 (ADR-0130 §1): the 303
// goes back to the Vantages tab and the callout and the typed endpoint ride the session
// flash to that landing GET, so a long Vantages list keeps its scroll offset.
func TestProvisionRejectsRootAndBadPort(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	// Root username is refused, and the typed values are preserved.
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

	// Out-of-range port is refused.
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
	// The flash is single-consume, so a reload of the same tab shows no stale callout.
	if again := vantagesBody(t, ac, base); strings.Contains(again, "already provisioned") {
		t.Fatalf("the callout survived a reload; body: %s", again)
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
	// Nothing that looks like a private key may ever appear.
	for _, marker := range []string{"PRIVATE KEY", "BEGIN OPENSSH PRIVATE", "host_key", "AAAAHOSTKEY"} {
		if strings.Contains(page, marker) {
			t.Errorf("page leaked private/host material %q; body: %s", marker, page)
		}
	}
}

// The Settings → Vantages tab renders each provisioned prober's published PUBLIC
// key (reveal-once, for installing in authorized_keys) and its host-key pin status,
// but never a private key nor the host-key value (#313, ADR-0110).
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

	// Prober provisioning relocated to the admin-only Settings → Vantages surface (#21d),
	// so a viewer is bounced from the read too; the viewer-facing vantage read is
	// Settings' concern at map #21.
	resp = get(t, vc, base+"/settings?tab=vantages")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer reached the admin-only vantages surface: status=%d, want 403", resp.StatusCode)
	}
}
