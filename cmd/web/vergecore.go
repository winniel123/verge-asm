package main

import (
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

// verge-core is where the operator reads the composed hot-tier port set and edits
// its frequency half (v1 spec §3.5). Only the frequency half is operator-editable:
// the sensitive half is authored by the release, so it is rendered read-only and
// has no mutating endpoint. A frequency edit is stored as a delta over the shipped
// default and applied at hot fan-out, so the sensitive half is unreachable from
// every write path here by construction. The presentation folds into the Settings
// delivery sub-tab (#281); the composition read lives in fillDeliverySection.

// freqRow is one frequency port shaped for rendering, with whether it is also on
// the sensitive half (and so stays probed after a removal) and whether an
// operator edit currently moves it off its shipped state.
type freqRow struct {
	Port          int
	AlsoSensitive bool
	Edited        bool
	EditAction    string
}

// sensRow is one read-only sensitive pair. Service is the release-authored display
// label for the port (#26c) — the sensitive tier is a fixed set, so the label is a
// known-service lookup, empty for a pair with no authored label (the chip collapses).
type sensRow struct {
	Port      int
	Transport string
	Service   string
}

// sensitiveServiceLabels names the release-authored sensitive tier's ports for the
// aperture chips. The sensitive tier is fixed and moves only with the release, so
// these labels are static; a port with no entry renders no label.
var sensitiveServiceLabels = map[int]string{
	21: "ftp", 23: "telnet", 445: "smb", 1433: "mssql", 3389: "rdp", 5900: "vnc",
}

func (s *server) vergeCorePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: "aperture"})
}

// editVergeCoreFrequency applies one add/remove/reset to the frequency half. It
// is admin-only. It refuses a non-numeric or out-of-range port; a valid edit is
// an upsert (add/remove) or a delete (reset) of the port's delta row. A rejected
// edit re-renders the delivery sub-tab with its error and typed value.
func (s *server) editVergeCoreFrequency(w http.ResponseWriter, r *http.Request, acct db.Account) {
	action := r.FormValue("action")
	portRaw := r.FormValue("port")
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{section: "vergecore", vcError: msg, vcPort: portRaw})
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		fail("Enter a port between 1 and 65535.")
		return
	}

	switch action {
	case "add", "remove":
		if err := s.store.UpsertVergeCoreFrequencyEdit(r.Context(), db.UpsertVergeCoreFrequencyEditParams{
			Port: int32(port), Action: action, CreatedBy: acct.ID,
		}); err != nil {
			s.serverError(w, "upsert verge-core frequency edit", err)
			return
		}
	case "reset":
		if err := s.store.DeleteVergeCoreFrequencyEdit(r.Context(), int32(port)); err != nil {
			s.serverError(w, "delete verge-core frequency edit", err)
			return
		}
	default:
		fail("Choose add, remove or reset.")
		return
	}
	http.Redirect(w, r, "/verge-core", http.StatusSeeOther)
}
