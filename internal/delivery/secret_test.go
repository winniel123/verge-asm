package delivery

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/secretseal"
)

var testTranscriptKey = []byte("transcriptkey0123456789abcdef012")

func sealedChannel(t *testing.T, secret string) (db.GetChannelForDeliveryRow, []byte) {
	t.Helper()
	key, err := secretseal.DeriveKey(testTranscriptKey, secretseal.LabelChannelSecret)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := secretseal.SealText(key, secret)
	if err != nil {
		t.Fatal(err)
	}
	return db.GetChannelForDeliveryRow{Url: "https://hooks.example/hook", Secret: stored}, key
}

func TestRunnerSignsWithTheOpenedChannelSecret(t *testing.T) {
	const secret = "sign-me"
	ch, key := sealedChannel(t, secret)
	if ch.Secret.String == secret || strings.Contains(ch.Secret.String, secret) {
		t.Fatalf("the row holds the secret in cleartext: %q", ch.Secret.String)
	}
	fake := &captureDoer{}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	r := &Runner{
		doer:      fake,
		now:       func() time.Time { return now },
		resolver:  fakeResolver{"hooks.example": {netip.MustParseAddr("93.184.216.34")}},
		secretKey: key,
	}
	body := []byte(`{"kind":"test"}`)

	status, err := r.sendToChannel(context.Background(), ch, body)
	if err != nil || !Delivered(status) {
		t.Fatalf("sendToChannel = %d, %v; want delivered", status, err)
	}
	if want := "sha256=" + Sign([]byte(secret), body, now); fake.sig != want {
		t.Fatalf("signature = %q, want one computed over the original secret %q", fake.sig, want)
	}
}

func TestRunnerRefusesALegacyCleartextChannelSecret(t *testing.T) {
	_, key := sealedChannel(t, "sign-me")
	fake := &captureDoer{}
	r := &Runner{
		doer:      fake,
		now:       time.Now,
		resolver:  fakeResolver{"hooks.example": {netip.MustParseAddr("93.184.216.34")}},
		secretKey: key,
	}
	legacy := db.GetChannelForDeliveryRow{
		Url:    "https://hooks.example/hook",
		Secret: pgtype.Text{String: "legacy-cleartext-secret", Valid: true},
	}

	_, err := r.sendToChannel(context.Background(), legacy, []byte("{}"))
	if !errors.Is(err, errOpenSecret) {
		t.Fatalf("a legacy cleartext row was tolerated: err=%v", err)
	}
	if fake.called {
		t.Fatal("the body was POSTed despite an unopenable secret")
	}
	if strings.Contains(err.Error(), "legacy-cleartext-secret") {
		t.Fatalf("the error carries the cleartext: %v", err)
	}
}

func TestRunnerSendsUnsignedWhenNoSecretIsSet(t *testing.T) {
	_, key := sealedChannel(t, "unused")
	fake := &captureDoer{}
	r := &Runner{
		doer:      fake,
		now:       time.Now,
		resolver:  fakeResolver{"hooks.example": {netip.MustParseAddr("93.184.216.34")}},
		secretKey: key,
	}
	ch := db.GetChannelForDeliveryRow{Url: "https://hooks.example/hook"}

	if _, err := r.sendToChannel(context.Background(), ch, []byte("{}")); err != nil {
		t.Fatalf("sendToChannel with no secret: %v", err)
	}
	if !fake.called || fake.sig != "" {
		t.Fatalf("an unsigned channel sent called=%v sig=%q; want a call with no signature", fake.called, fake.sig)
	}
}
