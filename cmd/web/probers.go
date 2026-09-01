package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/remoteexec"
	"github.com/winniel123/verge-asm/internal/vantage"
)

// proberView is a provisioned Vantage shaped for rendering. Only the public key
// is ever carried to the web surface — never a private key, which lives on the
// worker volume alone. KeySet collapses "has the worker published a public key
// yet" to a boolean so the template can show a "set/not set" status beside the
// key itself.
type proberView struct {
	Endpoint     string // host:port
	Username     string
	Availability string
	KeySet       bool
	PublicKey    string
	// HostKeyPinned reports whether the worker has pinned this vantage's host key on
	// first connection. The host key VALUE never reaches the web surface — a later
	// change is a hard failure, never a prompt — so this is only the pinned/awaiting
	// status, exactly as PublicKey exposes the public half but never the private one.
	HostKeyPinned bool
	// HostKeyFingerprint, Platform and Egress carry the pinned prober's lifecycle
	// facts to the spec VantageCard (#26c): the host-key fingerprint chip, the
	// accepted platform, and the observed egress address (which links /scope). They
	// are the real off-host facts as of P0.8 (#683): the fingerprint is derived from
	// the pinned host key, and platform/egress are what the worker observed over SSH
	// (`uname` and SSH_CLIENT) on the connect that pinned the key. Each renders empty
	// and its region collapses until that read lands — an un-pinned or not-yet-probed
	// vantage shows no fabricated fact — while the VERGE_DEV golden path keeps rendering
	// the design fixture's pinned stubs, never these live reads.
	HostKeyFingerprint string
	Platform           string
	Egress             string
	By                 string
	At                 string
}

// provisionProber declares a prober: the operator supplies host, port and a
// non-root username, and the row created here DECLARES "this vantage is on the
// internet" (CONTEXT.md "Vantage class") — there is no network_position field
// and no setup-wizard step. It is reached only through requireAdmin.
//
// No key material is generated here. The worker owns the SSH keypair volume and
// generates the pair out of band, publishing only the public half back to this
// row; web never touches a private key.
func (s *server) provisionProber(w http.ResponseWriter, r *http.Request, acct db.Account) {
	host := r.FormValue("host")
	port := r.FormValue("port")
	username := r.FormValue("username")
	fail := func(msg string) {
		// #21d: prober provisioning relocated to Settings → Vantages. A rejected provision
		// is a post-redirect-get like every other console refusal (ADR-0130 §1, ticket
		// #978): the error and the typed endpoint ride the session flash, and the 303 goes
		// back to the URL the form was submitted from, so the operator keeps their place in
		// a long Vantages list.
		s.failSettings(w, r, settingsForms{
			section: "vantages", proberError: msg, proberHost: host, proberPort: port, proberUser: username,
		})
	}

	ep, err := vantage.ParseEndpoint(host, port, username)
	if err != nil {
		fail(err.Error())
		return
	}
	// A provisioned Vantage still needs a mandatory measurement identity. Derive
	// its name from the endpoint (username@host:port) so it is unique per
	// provisioned endpoint — matching the (host, port, username) endpoint index —
	// while resolver ships blank (set in SQL) for the operator to fill in.
	if _, err := s.store.CreateVantage(r.Context(), db.CreateVantageParams{
		Name: fmt.Sprintf("%s@%s:%d", ep.Username, ep.Host, ep.Port),
		Host: ep.Host, Port: int32(ep.Port), Username: ep.Username, CreatedBy: acct.ID, // #nosec G115 (ep.Port validated 1..65535 by vantage.ParseEndpoint)
	}); err != nil {
		if isUniqueViolation(err) {
			fail("That prober endpoint is already provisioned.")
			return
		}
		fail("Could not provision the prober.")
		return
	}
	// #21d: prober provisioning relocated to Settings → Vantages. The 303 goes back to the
	// URL the form was submitted from (ADR-0130 §3, ticket #977), falling back to the tab
	// that hosts the section, so a provision made from a scrolled Vantages list keeps the
	// operator's place.
	s.backToSection(w, r, "vantages")
}

func toProberViews(rows []db.ListVantagesRow) []proberView {
	out := make([]proberView, 0, len(rows))
	for _, row := range rows {
		v := proberView{
			Endpoint:      endpointString(row.Host.String, row.Port.Int32),
			Username:      row.Username.String,
			Availability:  row.Availability.String,
			KeySet:        row.PublicKey.Valid && row.PublicKey.String != "",
			PublicKey:     row.PublicKey.String,
			HostKeyPinned: row.HostKey.Valid && row.HostKey.String != "",
			// The host-key fingerprint chip is DERIVED from the pinned known_hosts key
			// (the value itself never reaches web); it renders the canonical SHA256 form,
			// or empty when nothing is pinned yet. Platform and egress are what the worker
			// observed off-host and persisted, empty until that first probe.
			HostKeyFingerprint: remoteexec.Fingerprint(row.HostKey.String),
			Platform:           row.Platform.String,
			Egress:             row.Egress.String,
			By:                 row.CreatedByUsername,
		}
		if row.CreatedAt.Valid {
			v.At = row.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, v)
	}
	return out
}

func endpointString(host string, port int32) string {
	return host + ":" + strconv.Itoa(int(port))
}

// vantageLatencyLabel formats a vantage's measured connect round-trip for the
// Dashboard Vantages card (P0.5, SPEC-CHANGE.md collision #7): the spec's mono
// "34ms" reading once a first measurement exists, and the empty string when the
// datum is still NULL — the template renders the pending em dash for that case.
// The latency is measured on the prober connect that pins the host key and stored
// nullable on the vantage, so an unmeasured prober reads NULL, never a fabricated
// number.
func vantageLatencyLabel(ms pgtype.Int4) string {
	if !ms.Valid {
		return ""
	}
	return strconv.Itoa(int(ms.Int32)) + "ms"
}
