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

// Only the public half reaches web; the private key stays on the worker volume (ADR-0053).

type proberView struct {
	Endpoint     string
	Username     string
	Availability string
	KeySet       bool
	PublicKey    string

	// A changed host key is a hard failure, never a prompt (packaging-and-configuration.md §3).

	HostKeyPinned      bool
	HostKeyFingerprint string
	Platform           string
	Egress             string
	By                 string
	At                 string
}

func (s *server) provisionProber(w http.ResponseWriter, r *http.Request, acct db.Account) {
	host := r.FormValue("host")
	port := r.FormValue("port")
	username := r.FormValue("username")
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{
			section: "vantages", proberError: msg, proberHost: host, proberPort: port, proberUser: username,
		})
	}

	ep, err := vantage.ParseEndpoint(host, port, username)
	if err != nil {
		fail(err.Error())
		return
	}
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
	s.backToSection(w, r, "vantages")
}

func toProberViews(rows []db.ListVantagesRow) []proberView {
	out := make([]proberView, 0, len(rows))
	for _, row := range rows {
		v := proberView{
			Endpoint:           endpointString(row.Host.String, row.Port.Int32),
			Username:           row.Username.String,
			Availability:       row.Availability.String,
			KeySet:             row.PublicKey.Valid && row.PublicKey.String != "",
			PublicKey:          row.PublicKey.String,
			HostKeyPinned:      row.HostKey.Valid && row.HostKey.String != "",
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

func vantageLatencyLabel(ms pgtype.Int4) string {
	if !ms.Valid {
		return ""
	}
	return strconv.Itoa(int(ms.Int32)) + "ms"
}
