package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/act"
	"github.com/winniel123/verge-asm/internal/db"
)

type ssoAdminStore interface {
	DeleteSSOIdentity(ctx context.Context, id int64) (db.DeleteSSOIdentityRow, error)
	DeleteSSOProvider(ctx context.Context, id int64) (string, error)
	InsertSSOProvider(ctx context.Context, arg db.InsertSSOProviderParams) (int64, error)
	ListSSOBindings(ctx context.Context) ([]db.ListSSOBindingsRow, error)

	// A secret is read only where its act is performed, so no listing read selects it (ADR-0053).

	ListSSOProviders(ctx context.Context) ([]db.ListSSOProvidersRow, error)
	SetSSOProviderSecret(ctx context.Context, arg db.SetSSOProviderSecretParams) (string, error)
	UpdateSSOProvider(ctx context.Context, arg db.UpdateSSOProviderParams) (int64, error)
}

var ssoSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type ssoProviderView struct {
	ID        int64
	Slug      string
	Name      string
	Issuer    string
	ClientID  string
	Enabled   bool
	HasSecret bool
	CreatedBy string
	CreatedAt string
}

type ssoBindingView struct {
	ID           int64
	ProviderName string
	Account      string
	DisplayName  string
	LinkedAt     string
}

func (s *server) fillSSOSection(r *http.Request, f settingsForms, data map[string]any) error {
	rows, err := s.ssoAdminStore.ListSSOProviders(r.Context())
	if err != nil {
		return err
	}
	out := make([]ssoProviderView, 0, len(rows))
	for _, p := range rows {
		out = append(out, ssoProviderView{
			ID: p.ID, Slug: p.Slug, Name: p.Name, Issuer: p.Issuer, ClientID: p.ClientID,
			Enabled: p.Enabled, HasSecret: p.HasSecret,
			CreatedBy: p.CreatedByUsername.String, CreatedAt: p.CreatedAt.Time.UTC().Format(spanTimeFmt),
		})
	}
	data["SSOProviders"] = out

	bindings, err := s.ssoAdminStore.ListSSOBindings(r.Context())
	if err != nil {
		return err
	}
	bviews := make([]ssoBindingView, 0, len(bindings))
	for _, b := range bindings {
		bviews = append(bviews, ssoBindingView{
			ID: b.ID, ProviderName: b.ProviderName, Account: b.AccountUsername,
			DisplayName: b.DisplayName, LinkedAt: b.CreatedAt.Time.UTC().Format(spanTimeFmt),
		})
	}
	data["SSOBindings"] = bviews

	data["SSOError"] = f.ssoError
	data["SSOSlug"] = f.ssoSlug
	data["SSOName"] = f.ssoName
	data["SSOIssuer"] = f.ssoIssuer
	data["SSOClientID"] = f.ssoClientID
	return nil
}

type ssoFormValues struct {
	slug, name, issuer, clientID string
}

func readSSOForm(r *http.Request) ssoFormValues {
	return ssoFormValues{
		slug:     strings.TrimSpace(r.FormValue("slug")),
		name:     strings.TrimSpace(r.FormValue("name")),
		issuer:   strings.TrimSpace(r.FormValue("issuer")),
		clientID: strings.TrimSpace(r.FormValue("client_id")),
	}
}

func validateSSOForm(v ssoFormValues) string {
	switch {
	case v.slug == "":
		return "A short slug is required (it appears in the sign-on URL)."
	case !ssoSlugPattern.MatchString(v.slug):
		return "The slug must be lowercase letters, digits and hyphens only."
	case v.name == "":
		return "A display name is required."
	case v.issuer == "":
		return "The issuer URL is required."
	case v.clientID == "":
		return "The client ID is required."
	}
	u, err := url.Parse(v.issuer)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "The issuer must be an https URL (e.g. https://issuer.example.com)."
	}
	return ""
}

func (s *server) createSSOProvider(w http.ResponseWriter, r *http.Request, acct db.Account) {
	v := readSSOForm(r)
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{
			section: "sso", ssoError: msg,
			ssoSlug: v.slug, ssoName: v.name, ssoIssuer: v.issuer, ssoClientID: v.clientID,
		})
	}
	if msg := validateSSOForm(v); msg != "" {
		fail(msg)
		return
	}
	secret, ok := s.sealFormSecret(w, s.ssoSecretKey, r.FormValue("client_secret"), "seal sso provider secret")
	if !ok {
		return
	}
	if _, err := s.ssoAdminStore.InsertSSOProvider(r.Context(), db.InsertSSOProviderParams{
		Slug: v.slug, Name: v.name, Issuer: v.issuer, ClientID: v.clientID,
		ClientSecret: secret,
		Enabled:      true, CreatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		if isUniqueViolation(err) {
			fail("A provider with that slug already exists. Choose another slug.")
			return
		}
		s.serverError(w, "create sso provider", err)
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.SSOProviderDeclared{
		ProviderRef: act.ProviderRef{Slug: v.slug},
	})
	s.backToSection(w, r, "sso")
}

func (s *server) updateSSOProvider(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// The row's disclosure re-renders each field from the stored row, so a refusal echoes nothing.
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: msg})
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		fail("That provider could not be found.")
		return
	}
	v := readSSOForm(r)
	if msg := validateSSOForm(v); msg != "" {
		fail(msg)
		return
	}
	rows, err := s.ssoAdminStore.UpdateSSOProvider(r.Context(), db.UpdateSSOProviderParams{
		ID: id, Slug: v.slug, Name: v.name, Issuer: v.issuer, ClientID: v.clientID,
		Enabled: r.FormValue("enabled") != "",
	})
	if err != nil {
		if isUniqueViolation(err) {
			fail("A provider with that slug already exists. Choose another slug.")
			return
		}
		s.serverError(w, "update sso provider", err)
		return
	}
	if rows == 0 {
		fail("That provider could not be found.")
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.SSOProviderUpdated{
		ProviderRef: act.ProviderRef{Slug: v.slug},
	})
	s.backToSection(w, r, "sso")
}

func (s *server) setSSOProviderSecret(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
	// A default arm would clear the stored secret when the box is unchecked and the field blank.
	slug, set := "", false
	switch {
	case r.FormValue("clear_secret") != "":
		cleared, err := s.ssoAdminStore.SetSSOProviderSecret(r.Context(), db.SetSSOProviderSecretParams{ID: id})
		if errors.Is(err, pgx.ErrNoRows) {
			s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
			return
		}
		if err != nil {
			s.serverError(w, "clear sso provider secret", err)
			return
		}
		slug, set = cleared, true
	case strings.TrimSpace(r.FormValue("client_secret")) != "":
		sealed, ok := s.sealFormSecret(w, s.ssoSecretKey, r.FormValue("client_secret"), "seal sso provider secret")
		if !ok {
			return
		}
		stored, err := s.ssoAdminStore.SetSSOProviderSecret(r.Context(), db.SetSSOProviderSecretParams{
			ID: id, ClientSecret: sealed,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
			return
		}
		if err != nil {
			s.serverError(w, "set sso provider secret", err)
			return
		}
		slug, set = stored, true
	}
	if set {
		// The slug is the whole subject, so no field fits the secret value (spec §4.2).
		s.recorder().Record(r.Context(), actingAccount(acct), act.SSOProviderSecretSet{
			ProviderRef: act.ProviderRef{Slug: slug},
		})
	}
	s.backToSection(w, r, "sso")
}

func (s *server) deleteSSOProvider(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
	slug, err := s.ssoAdminStore.DeleteSSOProvider(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
	if err != nil {
		s.serverError(w, "delete sso provider", err)
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.SSOProviderWithdrawn{
		ProviderRef: act.ProviderRef{Slug: slug},
	})
	s.backToSection(w, r, "sso")
}

func (s *server) removeSSOBinding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That identity could not be found."})
		return
	}
	// A departed or recycled identity must stop authenticating as the account it bound (ADR-0113).
	gone, err := s.ssoAdminStore.DeleteSSOIdentity(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That identity could not be found."})
		return
	}
	if err != nil {
		s.serverError(w, "remove sso binding", err)
		return
	}
	s.recorder().Record(r.Context(), actingAccount(acct), act.SSOBindingRemoved{
		BindingRef: act.BindingRef{Slug: gone.Slug, Username: gone.Username},
	})
	log.Printf("web: sso: admin %d removed identity binding %d", acct.ID, id)
	s.backToSection(w, r, "sso")
}
