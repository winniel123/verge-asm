package main

import (
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The verge-core screen is where the operator reads the composed hot-tier port
// set and edits its frequency half (v1 spec §3.5). Only the frequency half is
// operator-editable: the sensitive half is authored by the release, so it is
// rendered read-only and has no mutating endpoint. A frequency edit is stored as
// a delta over the shipped default and applied at hot fan-out, so the sensitive
// half is unreachable from every write path here by construction.

// freqRow is one frequency port shaped for rendering, with whether it is also on
// the sensitive half (and so stays probed after a removal) and whether an
// operator edit currently moves it off its shipped state.
type freqRow struct {
	Port          int
	AlsoSensitive bool
	Edited        bool
	EditAction    string
}

// sensRow is one read-only sensitive pair.
type sensRow struct {
	Port      int
	Transport string
}

// vergeCoreForms carries a rejected submission's echo state.
type vergeCoreForms struct {
	err    string
	notice string
	port   string
}

func (s *server) vergeCorePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderVergeCore(w, r, acct, vergeCoreForms{})
}

// editVergeCoreFrequency applies one add/remove/reset to the frequency half. It
// is admin-only. It refuses a non-numeric or out-of-range port; a valid edit is
// an upsert (add/remove) or a delete (reset) of the port's delta row.
func (s *server) editVergeCoreFrequency(w http.ResponseWriter, r *http.Request, acct db.Account) {
	action := r.FormValue("action")
	portRaw := r.FormValue("port")
	fail := func(msg string) {
		s.renderVergeCore(w, r, acct, vergeCoreForms{err: msg, port: portRaw})
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

func (s *server) renderVergeCore(w http.ResponseWriter, r *http.Request, acct db.Account, f vergeCoreForms) {
	editRows, err := s.store.ListVergeCoreFrequencyEditsWithAuthor(r.Context())
	if err != nil {
		s.serverError(w, "list verge-core edits", err)
		return
	}
	editByPort := make(map[uint16]string, len(editRows))
	edits := make([]vergecore.FrequencyEdit, 0, len(editRows))
	for _, e := range editRows {
		editByPort[uint16(e.Port)] = e.Action
		edits = append(edits, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action})
	}

	shipped := vergecore.Default()
	effective := shipped.WithFrequencyEdits(edits)

	freq := make([]freqRow, 0, len(effective.FrequencyPairs()))
	for _, p := range effective.FrequencyPairs() {
		action, edited := editByPort[p.Port]
		freq = append(freq, freqRow{
			Port:          int(p.Port),
			AlsoSensitive: shipped.IsSensitive(p),
			Edited:        edited,
			EditAction:    action,
		})
	}
	sens := make([]sensRow, 0, len(shipped.SensitivePairs()))
	for _, p := range shipped.SensitivePairs() {
		sens = append(sens, sensRow{Port: int(p.Port), Transport: string(p.Transport)})
	}

	c := effective.Count()
	status := http.StatusOK
	if f.err != "" {
		status = http.StatusBadRequest
	}
	s.renderStatus(w, status, "vergecore", map[string]any{
		"Title": "verge-core", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Counts":    c,
		"UDPCount":  c.UDP,
		"Frequency": freq,
		"Sensitive": sens,
		"Error":     f.err,
		"Notice":    f.notice,
		"FormPort":  f.port,
	})
}
