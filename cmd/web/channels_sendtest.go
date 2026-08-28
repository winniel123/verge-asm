package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/delivery"
	"github.com/winniel123/verge-asm/internal/message"
)

// testChannel sends a real, signed test payload through a Channel's own transport and
// toasts the outcome (#757, R4-D2). Unlike an integration's Send test (#38/#39), a
// channel always holds its own URL, so it is always testable — there is no binding
// gate to defend. The payload is a plainly-marked test Message built through the
// delivery package's own BuildBody path and POSTed via the shared, SSRF-guarded
// SendSigned transport (the channelTestSender seam #39b) — the exact path the delivery
// and report-notify runners take, so the test rides the same guard and no second HTTP
// client exists. A 2xx toasts the spec's "Test message sent"; a transport error or any
// non-2xx toasts an honest non-ok degrade — never a fabricated delivery (honesty
// discipline #37). A missing or unknown channel id flashes-and-redirects rather than
// crashing, mirroring the sibling channel handlers. It is an admin act.
func (s *server) testChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	dest := "/settings?tab=channels"

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.toastRedirect(w, r, dest, "danger", "Test message not sent",
			"That channel could not be found.")
		return
	}

	ch, err := s.store.GetChannelForDelivery(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		// The channel is gone (a race with a delete, or a stale form) — degrade
		// honestly rather than send nothing into the void or crash.
		s.toastRedirect(w, r, dest, "danger", "Test message not sent",
			"That channel could not be found.")
		return
	}
	if err != nil {
		s.serverError(w, "test channel: read channel", err)
		return
	}

	body, err := delivery.MarshalBody(delivery.BuildBody(delivery.Firing{
		Class:       message.ClassDrift,
		Cause:       message.CauseDrift,
		SubjectKind: "channel-test",
		FiredAt:     strconv.FormatInt(id, 10),
		Instant:     s.now().UTC(),
		Headline:    "Test message from Verge ASM — delivery check for this channel.",
	}, s.externalURL))
	if err != nil {
		s.serverError(w, "test channel: marshal body", err)
		return
	}

	var secret []byte
	if ch.Secret.Valid {
		secret = []byte(ch.Secret.String)
	}

	statusCode, sendErr := s.channelSender.Send(r.Context(), ch.Url, body, secret)
	if sendErr != nil || !delivery.Delivered(statusCode) {
		s.toastRedirect(w, r, dest, "danger", "Test message not sent",
			"Delivery to the channel failed — check the channel URL and try again.")
		return
	}
	s.toastRedirect(w, r, dest, "ok", "Test message sent",
		"Check the channel's destination for the delivery.")
}
