package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/secretseal"
)

func (s *server) useTranscriptKey(key []byte) {
	s.transcriptKey = key
	// A nil key fails every seal and open closed, so no cleartext is admitted (ADR-0172 §2).
	s.channelSecretKey, _ = secretseal.DeriveKey(key, secretseal.LabelChannelSecret)
	s.ssoSecretKey, _ = secretseal.DeriveKey(key, secretseal.LabelSSOClientSecret)
}

func (s *server) sealFormSecret(w http.ResponseWriter, key []byte, raw, what string) (pgtype.Text, bool) {
	sealed, err := secretseal.SealText(key, raw)
	if err != nil {
		s.serverError(w, what, err)
		return pgtype.Text{}, false
	}
	return sealed, true
}

func (s *server) ssoConfigFor(prov db.GetSSOProviderForAuthRow, redirectURL string) (ssoConfig, error) {
	secret, err := secretseal.OpenText(s.ssoSecretKey, prov.ClientSecret)
	if err != nil {
		return ssoConfig{}, err
	}
	return ssoConfig{
		Slug: prov.Slug, Issuer: prov.Issuer, ClientID: prov.ClientID,
		ClientSecret: string(secret),
		RedirectURL:  redirectURL,
	}, nil
}
