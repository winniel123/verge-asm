package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
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
	By           string
	At           string
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
		s.renderSeeds(w, r, acct, seedsForms{
			proberError: msg, proberHost: host, proberPort: port, proberUser: username,
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
		Name:     fmt.Sprintf("%s@%s:%d", ep.Username, ep.Host, ep.Port),
		Host:     ep.Host, Port: int32(ep.Port), Username: ep.Username, CreatedBy: acct.ID,
	}); err != nil {
		if isUniqueViolation(err) {
			fail("That prober endpoint is already provisioned.")
			return
		}
		fail("Could not provision the prober.")
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

func toProberViews(rows []db.ListVantagesRow) []proberView {
	out := make([]proberView, 0, len(rows))
	for _, row := range rows {
		v := proberView{
			Endpoint:     endpointString(row.Host.String, row.Port.Int32),
			Username:     row.Username.String,
			Availability: row.Availability.String,
			KeySet:       row.PublicKey.Valid && row.PublicKey.String != "",
			PublicKey:    row.PublicKey.String,
			By:           row.CreatedByUsername,
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
