package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// Annotation management on the Signals screen (CONTEXT.md `Annotation`, v1 spec
// §6.5, ADR-0016/ADR-0073/ADR-0092/ADR-0093). An operator declares an acceptance
// on one `(subject, signal-name)` pair: that a fired rule is an accepted risk on
// a thing we are still measuring. Its whole effect is on the message — a
// `not-fired` → `fired` `Transition` on an annotated pair is recorded and is not
// a message. It moves no number: the pair is still measured, still inside the
// rule's `Predicate domain`, still counted under `fired`.
//
// Both endpoints are reached only through requireAdmin — a declaration is an
// operator act — and neither mints a `Message`: declaring and withdrawing are
// plain state changes, since a `Message` is one firing of one cause and an
// operator dial's movement is none of the four causes (ADR-0016). There is no
// author recorded on either act (ADR-0073).

// normalizeSubjectKey folds a typed subject key to the form the census renders it
// in, so an annotation keys on the thing denoted rather than the text that named it
// (CONTEXT.md `Subject`). Census subject keys — Names, Addresses, Service and
// Endpoint keys — are all rendered in lower case with no trailing dot, and none of
// their parts is case-sensitive, so an ASCII-case fold and a trailing-dot trim
// align a typed `Host.Example.COM` with the census's `host.example.com`. The fold
// is ASCII-only (ADR-0055): the model never folds an octet the protocol does not.
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

// declareAnnotation declares an Annotation on one `(subject, signal-name)` pair.
// It carries the operator's reason and the instant declared, and nothing else —
// no status, no expiry, no author. Re-declaring an existing pair is rejected as a
// duplicate rather than editing it: an Annotation cannot be edited, so changing a
// reason is a withdraw-then-declare (ADR-0093).
//
// The success path answers 303 to the URL the form was submitted from (backurl.go,
// ADR-0130 §3), not to a bare `/signals`, so an operator who accepts a risk from a
// filtered, sorted, paginated list lands back on that same list. The error path still
// re-renders in place; ticket #972 turns it into a redirect of its own.
func (s *server) declareAnnotation(w http.ResponseWriter, r *http.Request, acct db.Account) {
	subject := normalizeSubjectKey(r.FormValue("subject"))
	sigName := strings.TrimSpace(r.FormValue("signal"))
	reason := strings.TrimSpace(r.FormValue("reason"))

	fail := func(msg string) {
		s.renderSignals(w, r, acct, signalsForms{
			annoError: msg, annoSubject: subject, annoSignal: sigName, annoReason: reason,
		})
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

	if _, err := s.store.CreateAnnotation(r.Context(), db.CreateAnnotationParams{
		SubjectKey: subject, SignalName: sigName, Reason: reason,
	}); err != nil {
		if isUniqueViolation(err) {
			fail("That subject already carries an annotation on this signal. Withdraw it first to change the reason.")
			return
		}
		s.serverError(w, "create annotation", err)
		return
	}
	s.redirectBack(w, r, "/signals")
}

// withdrawAnnotation withdraws an Annotation. Withdrawing is a plain state change
// that produces no `Message`: its carrier is the message it releases, the pair's
// own next firing. It is admin-only and idempotent — deleting a row already gone
// is not an error, since the operator's intent that the acceptance no longer
// stand is satisfied either way. Like the declare above, the success path 303s back
// to the submitting URL (backurl.go, ADR-0130 §3) rather than to a bare `/signals`.
func (s *server) withdrawAnnotation(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSignals(w, r, acct, signalsForms{annoError: "That annotation could not be found."})
		return
	}
	if err := s.store.DeleteAnnotation(r.Context(), id); err != nil {
		s.serverError(w, "delete annotation", err)
		return
	}
	s.redirectBack(w, r, "/signals")
}

// knownRule reports whether name is a shipped rule. An Annotation may only name a
// rule that exists — accepting a firing on a signal that can never fire would be
// an acceptance with no reader.
func knownRule(name string) bool {
	for _, n := range signal.RuleNames() {
		if n == name {
			return true
		}
	}
	return false
}
