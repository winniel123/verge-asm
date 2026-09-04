package main

import (
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

// The sensitive half is release-authored, so no write path here may reach it (v1-spec §3.5).

type freqRow struct {
	Port          int
	AlsoSensitive bool
	Edited        bool
	EditAction    string
}

type sensRow struct {
	Port      int
	Transport string
	Service   string
}

var sensitiveServiceLabels = map[int]string{
	21: "ftp", 23: "telnet", 445: "smb", 1433: "mssql", 3389: "rdp", 5900: "vnc",
}

func (s *server) vergeCorePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A literal tab here would drift from the mapping failSettings stamps, dropping every callout.
	s.renderSettings(w, r, acct, s.takeSettingsFlash(r, tabForSection("vergecore")))
}

func (s *server) editVergeCoreFrequency(w http.ResponseWriter, r *http.Request, acct db.Account) {
	action := r.FormValue("action")
	portRaw := r.FormValue("port")
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{section: "vergecore", vcError: msg, vcPort: portRaw})
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		fail("Enter a port between 1 and 65535.")
		return
	}

	switch action {
	case "add", "remove":
		if err := s.store.UpsertVergeCoreFrequencyEdit(r.Context(), db.UpsertVergeCoreFrequencyEditParams{
			Port: int32(port), Action: action, CreatedBy: acct.ID, // #nosec G109 (port validated 1..65535 above)
		}); err != nil {
			s.serverError(w, "upsert verge-core frequency edit", err)
			return
		}
	case "reset":
		if err := s.store.DeleteVergeCoreFrequencyEdit(r.Context(), int32(port)); err != nil { // #nosec G109 (port validated 1..65535 above)
			s.serverError(w, "delete verge-core frequency edit", err)
			return
		}
	default:
		fail("Choose add, remove or reset.")
		return
	}
	s.backToSection(w, r, "vergecore")
}
