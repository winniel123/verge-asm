package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
)

func seedClassFixtureVantages(t *testing.T, f *fakeStore) {
	t.Helper()
	txt := func(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }
	f.vantages = append(f.vantages,
		db.Vantage{
			ID: 1, Name: "inside", Class: "unverified", Resolver: "9.9.9.9:53",
			Host: txt("inside.example.net"), Port: pgtype.Int4{Int32: 22, Valid: true},
			Username: txt("scanner"), Availability: txt("available"),
			DialledAddr: txt("10.0.0.7"), Egress: txt("10.0.0.7"),
		},
		db.Vantage{
			ID: 2, Name: "outside", Class: "internal", Resolver: "9.9.9.9:53",
			Host: txt("outside.example.net"), Port: pgtype.Int4{Int32: 22, Valid: true},
			Username: txt("scanner"), Availability: txt("available"),
			DialledAddr: txt("203.0.113.9"), Egress: txt("203.0.113.9"),
		},
	)
	f.vantageNextID = 3
}

func vantagesSectionRows(t *testing.T, srv *server, r *http.Request) []vantageRow {
	t.Helper()
	data := map[string]any{}
	if err := srv.fillVantagesSection(r, settingsForms{section: "vantages"}, data); err != nil {
		t.Fatalf("fillVantagesSection: %v", err)
	}
	rows, ok := data["Vantages"].([]vantageRow)
	if !ok {
		t.Fatalf("Vantages = %T, want []vantageRow", data["Vantages"])
	}
	return rows
}

func TestVantagesTabTagsTheDerivedClassNotTheColumn(t *testing.T) {
	f := newFakeStore()
	seedClassFixtureVantages(t, f)
	srv := newServer(f, testKey, "", fixedClock())

	rows := vantagesSectionRows(t, srv, httptest.NewRequest(http.MethodGet, "/settings?tab=vantages", nil))

	want := map[string]string{
		"inside":  string(custody.ClassInternal),
		"outside": string(custody.ClassInternet),
	}
	if len(rows) != len(want) {
		t.Fatalf("vantage rows = %d, want %d", len(rows), len(want))
	}
	for _, row := range rows {
		if row.Class != want[row.Name] {
			t.Errorf("vantage %q tag = %q, want %q: the tab read the vestigial class column, not the derivation (#709)",
				row.Name, row.Class, want[row.Name])
		}
	}
}

func TestVantagesTabRendersTheDerivedTag(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedClassFixtureVantages(t, f)

	base := start(t, f, "")
	page := vantagesBody(t, login(t, base, "admin", "hunter2hunter2"), base)

	for _, want := range []string{
		`<span class="st-tag">` + string(custody.ClassInternal) + `</span>`,
		`<span class="st-tag">` + string(custody.ClassInternet) + `</span>`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the vantages tab renders no %q tag; the page tags the column, not the derivation", want)
		}
	}
	if strings.Contains(page, `<span class="st-tag">`+string(custody.ClassUnverified)+`</span>`) {
		t.Error("a vantage that presents a covered address still tags unverified: the tab read `vantage.class`")
	}
}

func TestVantagesTabAndDashboardChipAgree(t *testing.T) {
	f := newFakeStore()
	seedClassFixtureVantages(t, f)
	srv := newServer(f, testKey, "", fixedClock())
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	tab := map[string]string{}
	for _, row := range vantagesSectionRows(t, srv, r) {
		tab[row.Name] = row.Class
	}

	chips, ok := srv.dashboardData(r, db.Account{})["Vantages"].([]dashVantageView)
	if !ok {
		t.Fatalf("dashboard Vantages = %T, want []dashVantageView", srv.dashboardData(r, db.Account{})["Vantages"])
	}
	if len(chips) != len(tab) {
		t.Fatalf("dashboard chips = %d, tab tags = %d", len(chips), len(tab))
	}
	for _, chip := range chips {
		if tab[chip.Name] != chip.Class {
			t.Errorf("vantage %q: the tab tags %q and the dashboard chip derives %q; one install, two answers",
				chip.Name, tab[chip.Name], chip.Class)
		}
	}
}
