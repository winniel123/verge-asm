package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

type annotationsStore interface {
	CreateAnnotation(ctx context.Context, arg db.CreateAnnotationParams) (db.Annotation, error)
	DeleteAnnotation(ctx context.Context, id int64) (db.DeleteAnnotationRow, error)
}

func normalizeSubjectKey(input string) string {
	s := strings.TrimSuffix(strings.TrimSpace(input), ".")
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (s *server) declareAnnotation(w http.ResponseWriter, r *http.Request, acct db.Account) {
	subject := normalizeSubjectKey(r.FormValue("subject"))
	sigName := strings.TrimSpace(r.FormValue("signal"))
	reason := strings.TrimSpace(r.FormValue("reason"))

	fail := func(msg string) {
		stashFormFlash(s, r, signalsForms{
			annoError: msg, annoSubject: subject, annoSignal: sigName, annoReason: reason,
		})
		s.redirectBack(w, r, "/signals")
	}

	if subject == "" {
		fail("Enter the subject the acceptance is keyed on.")
		return
	}
	if !knownRule(sigName) {
		fail("Choose the signal whose firing you are accepting.")
		return
	}
	if reason == "" {
		fail("State why this firing is an accepted risk. The reason is the reviewable artefact.")
		return
	}

	// The row carries no author; the act does (ADR-0073, #1786).
	if _, err := s.annotationsStore.CreateAnnotation(r.Context(), db.CreateAnnotationParams{
		SubjectKey: subject, SignalName: sigName, Reason: reason,
	}); err != nil {
		// A changed reason is withdraw-then-declare, never an edit (ADR-0093).
		if isUniqueViolation(err) {
			fail("That subject already carries an annotation on this signal. Withdraw it first to change the reason.")
			return
		}
		s.serverError(w, "create annotation", err)
		return
	}
	// The pair and no prose: prose is #127 §5's one reopening condition (spec §8 · D.6).
	s.recorder().Record(r.Context(), actingAccount(acct), act.AnnotationDeclared{
		AnnotationRef: act.AnnotationRef{SubjectKey: subject, Signal: sigName},
	})
	s.redirectBack(w, r, "/signals")
}

func (s *server) withdrawAnnotation(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		stashFormFlash(s, r, signalsForms{annoError: "That annotation could not be found."})
		s.redirectBack(w, r, "/signals")
		return
	}
	// A dial's movement is not one of the four causes, so neither act mints a Message (ADR-0092).
	row, err := s.annotationsStore.DeleteAnnotation(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		// An unknown id withdraws nothing, and a refused act directed nothing (spec §7.6).
		s.redirectBack(w, r, "/signals")
		return
	}
	if err != nil {
		s.serverError(w, "delete annotation", err)
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.AnnotationWithdrawn{
		AnnotationRef: act.AnnotationRef{SubjectKey: row.SubjectKey, Signal: row.SignalName},
	})
	s.redirectBack(w, r, "/signals")
}

func knownRule(name string) bool {
	for _, n := range signal.RuleNames() {
		if n == name {
			return true
		}
	}
	return false
}
