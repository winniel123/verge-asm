package report

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/delivery"
	"github.com/winniel123/verge-asm/internal/secretseal"
)

var testTranscriptKey = []byte("transcriptkey0123456789abcdef012")

func TestNotifySignsWithTheOpenedChannelSecret(t *testing.T) {
	const secret = "sign-me"
	key, err := secretseal.DeriveKey(testTranscriptKey, secretseal.LabelChannelSecret)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := secretseal.SealText(key, secret)
	if err != nil {
		t.Fatal(err)
	}
	if stored.String == secret || strings.Contains(stored.String, secret) {
		t.Fatalf("the row holds the secret in cleartext: %q", stored.String)
	}
	fake := &captureDoer{status: 200}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	n := &NotifyRunner{
		doer:      fake,
		now:       func() time.Time { return now },
		resolver:  fakeResolver{"hooks.example": {netip.MustParseAddr("93.184.216.34")}},
		secretKey: key,
	}
	ch := db.GetChannelForDeliveryRow{Url: "https://hooks.example/hook", Secret: stored}
	body := []byte(`{"kind":"report-ready"}`)

	status, err := n.sendToChannel(context.Background(), ch, body)
	if err != nil || !delivery.Delivered(status) {
		t.Fatalf("sendToChannel = %d, %v; want delivered", status, err)
	}
	if want := "sha256=" + delivery.Sign([]byte(secret), body, now); fake.sig != want {
		t.Fatalf("signature = %q, want one computed over the original secret %q", fake.sig, want)
	}

	legacy := db.GetChannelForDeliveryRow{
		Url:    "https://hooks.example/hook",
		Secret: pgtype.Text{String: "legacy-cleartext-secret", Valid: true},
	}
	fake.called = false
	_, err = n.sendToChannel(context.Background(), legacy, body)
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
