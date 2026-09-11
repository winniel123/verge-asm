package main

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func (f *fakeStore) InsertSSOProvider(_ context.Context, arg db.InsertSSOProviderParams) (int64, error) {
	for _, p := range f.ssoProviders {
		if p.slug == arg.Slug {
			return 0, &pgconn.PgError{Code: "23505", Message: "duplicate sso slug"}
		}
	}
	f.ssoNextID++
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: f.ssoNextID, slug: arg.Slug, name: arg.Name, issuer: arg.Issuer,
		clientID: arg.ClientID, secret: arg.ClientSecret.String, hasSecret: arg.ClientSecret.Valid,
		enabled: arg.Enabled, createdBy: arg.CreatedBy,
		createdAt: obsClock,
	})
	return f.ssoNextID, nil
}

func (f *fakeStore) ListSSOProviders(context.Context) ([]db.ListSSOProvidersRow, error) {
	out := []db.ListSSOProvidersRow{}
	for i := len(f.ssoProviders) - 1; i >= 0; i-- {
		p := f.ssoProviders[i]
		out = append(out, db.ListSSOProvidersRow{
			ID: p.id, Slug: p.slug, Name: p.name, Issuer: p.issuer, ClientID: p.clientID,
			Enabled: p.enabled, HasSecret: p.hasSecret,
			CreatedBy: p.createdBy, CreatedAt: pgtype.Timestamptz{Time: p.createdAt, Valid: true},
			CreatedByUsername: f.usernameForID(p.createdBy),
		})
	}
	return out, nil
}

func (f *fakeStore) UpdateSSOProvider(_ context.Context, arg db.UpdateSSOProviderParams) (int64, error) {
	for i := range f.ssoProviders {
		if f.ssoProviders[i].id != arg.ID {
			continue
		}
		for _, p := range f.ssoProviders {
			if p.id != arg.ID && p.slug == arg.Slug {
				return 0, &pgconn.PgError{Code: "23505", Message: "duplicate sso slug"}
			}
		}
		f.ssoProviders[i].slug = arg.Slug
		f.ssoProviders[i].name = arg.Name
		f.ssoProviders[i].issuer = arg.Issuer
		f.ssoProviders[i].clientID = arg.ClientID
		f.ssoProviders[i].enabled = arg.Enabled
		return 1, nil
	}
	return 0, nil
}

func (f *fakeStore) SetSSOProviderSecret(_ context.Context, arg db.SetSSOProviderSecretParams) (string, error) {
	for i := range f.ssoProviders {
		if f.ssoProviders[i].id == arg.ID {
			f.ssoProviders[i].secret = arg.ClientSecret.String
			f.ssoProviders[i].hasSecret = arg.ClientSecret.Valid
			return f.ssoProviders[i].slug, nil
		}
	}
	return "", pgx.ErrNoRows
}

func (f *fakeStore) DeleteSSOProvider(_ context.Context, id int64) (string, error) {
	slug, found := "", false
	kept := f.ssoProviders[:0]
	for _, p := range f.ssoProviders {
		if p.id == id {
			slug, found = p.slug, true
			continue
		}
		kept = append(kept, p)
	}
	f.ssoProviders = kept
	var keptIdents []fakeSSOIdentity
	for _, i := range f.ssoIdentities {
		if i.providerID != id {
			keptIdents = append(keptIdents, i)
		}
	}
	f.ssoIdentities = keptIdents
	if !found {
		return "", pgx.ErrNoRows
	}
	return slug, nil
}

func (f *fakeStore) ListSSOBindings(_ context.Context) ([]db.ListSSOBindingsRow, error) {
	out := []db.ListSSOBindingsRow{}
	for k := len(f.ssoIdentities) - 1; k >= 0; k-- {
		i := f.ssoIdentities[k]
		out = append(out, db.ListSSOBindingsRow{
			ID: i.id, ProviderID: i.providerID,
			ProviderSlug: f.ssoSlugForID(i.providerID), ProviderName: f.ssoNameForID(i.providerID),
			AccountID: i.accountID, AccountUsername: f.usernameForID(i.accountID),
			DisplayName: i.displayName, CreatedAt: pgtype.Timestamptz{Time: i.createdAt, Valid: true},
		})
	}
	return out, nil
}

func (f *fakeStore) DeleteSSOIdentity(_ context.Context, id int64) (db.DeleteSSOIdentityRow, error) {
	var kept []fakeSSOIdentity
	var gone db.DeleteSSOIdentityRow
	found := false
	for _, i := range f.ssoIdentities {
		if i.id == id {
			gone = db.DeleteSSOIdentityRow{
				Slug: f.ssoProviderSlug(i.providerID), Username: f.usernameForID(i.accountID),
			}
			found = true
			continue
		}
		kept = append(kept, i)
	}
	f.ssoIdentities = kept
	if !found {
		return db.DeleteSSOIdentityRow{}, pgx.ErrNoRows
	}
	return gone, nil
}

func (f *fakeStore) ssoProviderSlug(id int64) string {
	for _, p := range f.ssoProviders {
		if p.id == id {
			return p.slug
		}
	}
	return ""
}
