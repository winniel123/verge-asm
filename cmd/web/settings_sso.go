package main

import (
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

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
	rows, err := s.store.ListSSOProviders(r.Context())
	if err != nil {
		return err
	}
	out := make([]ssoProviderView, 0, len(rows))
	for _, p := range rows {
		out = append(out, ssoProviderView{
			ID: p.ID, Slug: p.Slug, Name: p.Name, Issuer: p.Issuer, ClientID: p.ClientID,
			Enabled: p.Enabled, HasSecret: p.HasSecret,
			CreatedBy: p.CreatedByUsername, CreatedAt: p.CreatedAt.Time.UTC().Format(spanTimeFmt),
		})
	}
	data["SSOProviders"] = out

	bindings, err := s.store.ListSSOBindings(r.Context())
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
	if _, err := s.store.InsertSSOProvider(r.Context(), db.InsertSSOProviderParams{
		Slug: v.slug, Name: v.name, Issuer: v.issuer, ClientID: v.clientID,
		ClientSecret: optionalSecret(r.FormValue("client_secret")),
		Enabled:      true, CreatedBy: acct.ID,
	}); err != nil {
		if isUniqueViolation(err) {
			fail("A provider with that slug already exists. Choose another slug.")
			return
		}
		s.serverError(w, "create sso provider", err)
		return
	}
	s.backToSection(w, r, "sso")
}

func (s *server) updateSSOProvider(w http.ResponseWriter, r *http.Request, _ db.Account) {
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
	rows, err := s.store.UpdateSSOProvider(r.Context(), db.UpdateSSOProviderParams{
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
	s.backToSection(w, r, "sso")
}

func (s *server) setSSOProviderSecret(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
	// A blank field with the box unchecked must keep the stored secret, so no default arm clears it.
	switch {
	case r.FormValue("clear_secret") != "":
		if err := s.store.SetSSOProviderSecret(r.Context(), db.SetSSOProviderSecretParams{ID: id}); err != nil {
			s.serverError(w, "clear sso provider secret", err)
			return
		}
	case strings.TrimSpace(r.FormValue("client_secret")) != "":
		if err := s.store.SetSSOProviderSecret(r.Context(), db.SetSSOProviderSecretParams{
			ID: id, ClientSecret: optionalSecret(r.FormValue("client_secret")),
		}); err != nil {
			s.serverError(w, "set sso provider secret", err)
			return
		}
	}
	s.backToSection(w, r, "sso")
}

func (s *server) deleteSSOProvider(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
	if err := s.store.DeleteSSOProvider(r.Context(), id); err != nil {
		s.serverError(w, "delete sso provider", err)
		return
	}
	s.backToSection(w, r, "sso")
}

func (s *server) removeSSOBinding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That identity could not be found."})
		return
	}
	// A departed or recycled identity must stop authenticating as the account it bound (ADR-0113).
	if err := s.store.DeleteSSOIdentity(r.Context(), id); err != nil {
		s.serverError(w, "remove sso binding", err)
		return
	}
	log.Printf("web: sso: admin %d removed identity binding %d", acct.ID, id)
	s.backToSection(w, r, "sso")
}
