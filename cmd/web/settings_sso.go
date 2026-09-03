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

// Settings → single sign-on (#293, ADR-0112, ADR-0113): declare, edit, re-key and remove
// the OIDC providers the SignIn screen offers, and manage the verified-identity bindings
// authentication keys on. Every act here is admin-gated (the routes carry requireAdmin),
// and the client secret is write-only — an edit that leaves the secret field blank keeps
// the stored one, a value replaces it, the clear box removes it — exactly the
// channel-secret pattern.

// ssoSlugPattern keeps a provider slug URL-safe: it rides the flow routes
// (/login/sso/<slug>), so it is lowercase alphanumeric with internal hyphens.
var ssoSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ssoProviderView is one configured provider shaped for the Settings table: its
// display fields, whether a secret is set (never the value), and who declared it.
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

// ssoBindingView is one verified-identity binding for the admin table: which account an
// external identity (via which provider) authenticates as, so an admin can remove it on
// offboarding or a seat reassignment (ADR-0113).
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
	// Echo a rejected add form so the operator does not retype it; defaults otherwise.
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

// validateSSOForm checks the shared provider fields, returning a message on the first
// failure. The issuer must be an https URL (an OIDC issuer is always https, and the
// discovery fetch would otherwise be plaintext).
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

// createSSOProvider declares one OIDC provider. The secret is optional (a public
// PKCE-only client sets none) and write-only.
//
// Both outcomes are a post-redirect-get back to the URL the form was submitted from
// (ADR-0130 §1 and §3, map #969 ticket #975). A refusal carries its message and the
// operator's typed values to that landing GET through the session form flash, so the
// typed client secret never enters the URL and the operator keeps their scroll offset.
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
		// A duplicate slug is the one expected user error the DB refuses (unique); tell
		// the operator plainly rather than 500ing.
		if isUniqueViolation(err) {
			fail("A provider with that slug already exists. Choose another slug.")
			return
		}
		s.serverError(w, "create sso provider", err)
		return
	}
	s.backToSection(w, r, "sso")
}

// updateSSOProvider edits a provider's fields and enabled state, applying a secret
// change only if one was asked for (blank leaves the stored secret untouched).
//
// The edit form is one row's disclosure (settings.tmpl st-disc), not a query-opened
// dialog, so a refusal has no modal to re-open: it lands on the tab and the callout
// renders above the provider table. The typed values are not echoed, because the
// disclosure renders each field from the stored row. The migration does not change
// that.
func (s *server) updateSSOProvider(w http.ResponseWriter, r *http.Request, _ db.Account) {
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
		// The id parsed but matched nothing (deleted in another tab): say so rather than
		// redirecting as if the edit applied.
		fail("That provider could not be found.")
		return
	}
	s.backToSection(w, r, "sso")
}

// setSSOProviderSecret writes, replaces or clears a provider's client secret through
// its own path, so a general edit never has to carry the secret. A blank field with
// the clear box unchecked is a no-op — the stored secret is kept, honouring the form's
// "leave blank to keep" — exactly the channel-secret pattern (settings.go). Clearing a
// stored secret requires the explicit clear box, which wins over any typed value.
func (s *server) setSSOProviderSecret(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
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

// removeSSOBinding lets an admin revoke any verified-identity binding — the offboarding
// / seat-reassignment case ADR-0113 is about, where a departed user's linked identity
// (or a recycled one) must stop authenticating as an account. Idempotent: removing a row
// already gone satisfies the intent either way.
func (s *server) removeSSOBinding(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "sso", ssoError: "That identity could not be found."})
		return
	}
	if err := s.store.DeleteSSOIdentity(r.Context(), id); err != nil {
		s.serverError(w, "remove sso binding", err)
		return
	}
	log.Printf("web: sso: admin %d removed identity binding %d", acct.ID, id)
	s.backToSection(w, r, "sso")
}
