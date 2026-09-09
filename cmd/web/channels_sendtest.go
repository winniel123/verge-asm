package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/delivery"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/secretseal"
)

type channelSendTestStore interface {
	GetChannelForDelivery(ctx context.Context, id int64) (db.GetChannelForDeliveryRow, error)
}

func (s *server) testChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	dest := "/settings?tab=channels"

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.toastRedirectBack(w, r, dest, "danger", "Test message not sent",
			"That channel could not be found.")
		return
	}

	ch, err := s.channelSendTestStore.GetChannelForDelivery(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.toastRedirectBack(w, r, dest, "danger", "Test message not sent",
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

	secret, err := secretseal.OpenText(s.channelSecretKey, ch.Secret)
	if err != nil {
		s.serverError(w, "test channel: open secret", err)
		return
	}

	statusCode, sendErr := s.channelSender.Send(r.Context(), ch.Url, body, secret)
	if sendErr != nil || !delivery.Delivered(statusCode) {
		s.toastRedirectBack(w, r, dest, "danger", "Test message not sent",
			"Delivery to the channel failed — check the channel URL and try again.")
		return
	}
	s.toastRedirectBack(w, r, dest, "ok", "Test message sent",
		"Check the channel's destination for the delivery.")
}
