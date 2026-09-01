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

// vergeCorePage is the folded read surface for the aperture tab: /verge-core renders
// the same section /settings?tab=aperture does, so it is a legitimate submitting URL
// for the frequency forms and therefore a legitimate LANDING for a refused one. It
// reads the session form flash for that reason (ADR-0130 §1, flash.go).
//
// The tab it claims is written as tabForSection("vergecore"), not as the literal
// "aperture". This is the one place in the surface where the section and its tab carry
// DIFFERENT names, so a literal here would be a second spelling of a mapping
// settings.go already owns. failSettings stamps the claim from the same call, and a
// claim that drifted from it would drop every callout on this page in silence — the
// exact failure this ticket closes.
func (s *server) vergeCorePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, s.takeSettingsFlash(r, tabForSection("vergecore")))
}

// editVergeCoreFrequency applies one add/remove/reset to the frequency half. It
// is admin-only. It refuses a non-numeric or out-of-range port; a valid edit is
// an upsert (add/remove) or a delete (reset) of the port's delta row. A rejected
// edit comes back to the aperture tab with its error and the typed port still in the
// input.
//
// Both outcomes are a post-redirect-get back to the URL the form was submitted from
// (ADR-0130 §1 and §3, map #969 ticket #975), so an operator who edits from the folded
// /verge-core surface lands there and one who edits from /settings?tab=aperture lands
// there. The refusal's message and typed port ride the session form flash, not the
// query.
//
// The no-field FALLBACK moved from /verge-core to /settings?tab=aperture, because
// backToSection derives it from the section. Only a POST that carries no usable
// `return` reaches it — a stale cached page, or a hand-crafted submit. Both surfaces
// render this same section, so such a caller still lands on the control it acted on.
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
