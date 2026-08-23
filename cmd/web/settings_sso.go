package main

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// Settings → single sign-on (#293, ADR-0112): declare, edit, re-key and remove the
// OIDC providers the SignIn screen offers. Every act here is admin-gated (the routes
// carry requireAdmin), and the client secret is write-only — an edit that leaves the
// secret field blank keeps the stored one, a value replaces it, the clear box removes
// it — exactly the channel-secret pattern.

const defaultUsernameClaim = "preferred_username"

// ssoSlugPattern keeps a provider slug URL-safe: it rides the flow routes
// (/login/sso/<slug>), so it is lowercase alphanumeric with internal hyphens.
var ssoSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ssoProviderView is one configured provider shaped for the Settings table: its
// display fields, whether a secret is set (never the value), and who declared it.
type ssoProviderView struct {
	ID            int64
	Slug          string
	Name          string
	Issuer        string
	ClientID      string
	UsernameClaim string
	Enabled       bool
	HasSecret     bool
	CreatedBy     string
	CreatedAt     string
}

// fillSSOSection reads the configured providers and the add-form echo. The section is
// the honest empty-state when none are configured (the SignIn "not configured" state
// mirrors it); once a provider exists, SignIn renders a button for it.
func (s *server) fillSSOSection(r *http.Request, f settingsForms, data map[string]any) error {
	rows, err := s.store.ListSSOProviders(r.Context())
	if err != nil {
		return err
	}
	out := make([]ssoProviderView, 0, len(rows))
	for _, p := range rows {
		out = append(out, ssoProviderView{
			ID: p.ID, Slug: p.Slug, Name: p.Name, Issuer: p.Issuer, ClientID: p.ClientID,
			UsernameClaim: p.UsernameClaim, Enabled: p.Enabled, HasSecret: p.HasSecret,
			CreatedBy: p.CreatedByUsername, CreatedAt: p.CreatedAt.Time.UTC().Format(spanTimeFmt),
		})
	}
	data["SSOProviders"] = out
	data["SSOError"] = f.ssoError
	// Echo a rejected add form so the operator does not retype it; defaults otherwise.
	data["SSOSlug"] = f.ssoSlug
	data["SSOName"] = f.ssoName
	data["SSOIssuer"] = f.ssoIssuer
	data["SSOClientID"] = f.ssoClientID
	claim := f.ssoClaim
	if claim == "" {
		claim = defaultUsernameClaim
	}
	data["SSOClaim"] = claim
	return nil
}

// ssoFormValues pulls and trims the shared provider fields from a submission.
type ssoFormValues struct {
	slug, name, issuer, clientID, claim string
}

func readSSOForm(r *http.Request) ssoFormValues {
	claim := strings.TrimSpace(r.FormValue("username_claim"))
	if claim == "" {
		claim = defaultUsernameClaim
	}
	return ssoFormValues{
		slug:     strings.TrimSpace(r.FormValue("slug")),
		name:     strings.TrimSpace(r.FormValue("name")),
		issuer:   strings.TrimSpace(r.FormValue("issuer")),
		clientID: strings.TrimSpace(r.FormValue("client_id")),
		claim:    claim,
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
func (s *server) createSSOProvider(w http.ResponseWriter, r *http.Request, acct db.Account) {
	v := readSSOForm(r)
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{
			section: "sso", ssoError: msg,
			ssoSlug: v.slug, ssoName: v.name, ssoIssuer: v.issuer, ssoClientID: v.clientID, ssoClaim: v.claim,
		})
	}
	if msg := validateSSOForm(v); msg != "" {
		fail(msg)
		return
	}
	if _, err := s.store.InsertSSOProvider(r.Context(), db.InsertSSOProviderParams{
		Slug: v.slug, Name: v.name, Issuer: v.issuer, ClientID: v.clientID,
		ClientSecret: optionalSecret(r.FormValue("client_secret")),
		UsernameClaim: v.claim, Enabled: true, CreatedBy: acct.ID,
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
	http.Redirect(w, r, "/settings?tab=sso", http.StatusSeeOther)
}

// updateSSOProvider edits a provider's fields and enabled state, applying a secret
// change only if one was asked for (blank leaves the stored secret untouched).
func (s *server) updateSSOProvider(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{section: "sso", ssoError: msg})
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
		UsernameClaim: v.claim, Enabled: r.FormValue("enabled") != "",
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
	http.Redirect(w, r, "/settings?tab=sso", http.StatusSeeOther)
}

// setSSOProviderSecret writes, replaces or clears a provider's client secret through
// its own path, so a general edit never has to carry the secret. A blank field with
// the clear box unchecked is a no-op — the stored secret is kept, honouring the form's
// "leave blank to keep" — exactly the channel-secret pattern (settings.go). Clearing a
// stored secret requires the explicit clear box, which wins over any typed value.
func (s *server) setSSOProviderSecret(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "sso", ssoError: "That provider could not be found."})
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
	http.Redirect(w, r, "/settings?tab=sso", http.StatusSeeOther)
}

// deleteSSOProvider removes a provider. Idempotent: deleting a row already gone
// satisfies the operator's intent either way.
func (s *server) deleteSSOProvider(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "sso", ssoError: "That provider could not be found."})
		return
	}
	if err := s.store.DeleteSSOProvider(r.Context(), id); err != nil {
		s.serverError(w, "delete sso provider", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=sso", http.StatusSeeOther)
}
