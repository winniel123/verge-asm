package main

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

func declare(t *testing.T, c *http.Client, base, kind, scope string) *http.Response {
	t.Helper()
	return postForm(t, c, base+"/seeds", url.Values{"kind": {kind}, "scope": {scope}})
}

func seedsBody(t *testing.T, c *http.Client, base string) string {
	t.Helper()
	resp, err := c.Get(base + "/scope")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /scope status = %d, want 200", resp.StatusCode)
	}
	return body(t, resp)
}

func TestDeclareNameAndAddressSeeds(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	resp := declare(t, ac, base, "name", "Example.com")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/scope") {
		t.Fatalf("declare name: status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	resp.Body.Close()

	resp = declare(t, ac, base, "address", "203.0.113.0/24")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("declare address: status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "example.com") {
		t.Errorf("name scope not listed; body: %s", page)
	}
	if !strings.Contains(page, "203.0.113.0/24") {
		t.Errorf("address scope not listed; body: %s", page)
	}
}

func TestAddressScopeOverCapRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/21"))
	if !strings.Contains(got, "over the 1,024-address cap") {
		t.Fatalf("over-cap scope not rejected clearly; body=%s", got)
	}
	if !strings.Contains(got, `value="10.0.0.0/21"`) {
		t.Errorf("rejected scope not retained in the form; body: %s", got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds after rejected declaration = %d, want 0", len(f.seeds))
	}

	resp := declare(t, ac, base, "address", "10.0.0.0/22")
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("/22 at the cap should be accepted: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
	if resp = declare(t, ac, base, "address", "2001:db8::/118"); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("IPv6 /118 at the cap should be accepted: status=%d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestOverCapIPv6NamesTheRouteAndShutsTheKnob(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "address", "2001:db8::/48"))
	if !strings.Contains(got, "Do not raise the cap for this") {
		t.Fatalf("IPv6 over-cap refusal does not shut the knob; body=%s", got)
	}
	if strings.Contains(got, "Raise your cap") {
		t.Errorf("IPv6 over-cap refusal offers the cap setting; body=%s", got)
	}
	if !strings.Contains(got, "custody") {
		t.Errorf("IPv6 over-cap refusal does not name the custody extension as the route; body=%s", got)
	}
	if !strings.Contains(got, `value="2001:db8::/48"`) {
		t.Errorf("rejected scope not retained in the form; body: %s", got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds = %d, want 0", len(f.seeds))
	}

	got = refusalPage(t, ac, base, declare(t, ac, base, "address", "10.0.0.0/21"))
	if !strings.Contains(got, "Raise your cap") {
		t.Fatalf("IPv4 over-cap refusal must still offer the cap setting; body=%s", got)
	}
}

func TestULabelSeedNamesWhereToObtainTheALabel(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "name", "café.example"))
	if !strings.Contains(got, "DNS provider") {
		t.Fatalf("U-label refusal does not name where to obtain the A-label; body=%s", got)
	}
	if strings.Contains(got, "xn--caf") {
		t.Errorf("U-label refusal rendered the computed A-label; body=%s", got)
	}
	if strings.Contains(got, "not a bare domain") {
		t.Errorf("U-label fell to the generic refusal; body=%s", got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds = %d, want 0", len(f.seeds))
	}
}

func TestWildcardSeedNamesTheSubtreeExclusion(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "name", "*.example.com"))
	if !strings.Contains(got, "subtree exclusion") {
		t.Fatalf("wildcard refusal does not name the subtree exclusion; body=%s", got)
	}
	if !strings.Contains(got, "never matches") {
		t.Errorf("wildcard refusal does not state the apex difference; body=%s", got)
	}
	if strings.Contains(got, "not a bare domain") {
		t.Errorf("wildcard fell to the generic refusal; body=%s", got)
	}
	if len(f.seeds) != 0 || len(f.exclusions) != 0 {
		t.Fatalf("seeds=%d exclusions=%d, want none written", len(f.seeds), len(f.exclusions))
	}
}

func TestNameScopeMustBeRegistrable(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	got := refusalPage(t, ac, base, declare(t, ac, base, "name", "www.example.com"))
	if !strings.Contains(got, "registrable domain example.com") {
		t.Fatalf("subdomain not rejected toward its registrable domain; body=%s", got)
	}
	if len(f.seeds) != 0 {
		t.Fatalf("seeds = %d, want 0", len(f.seeds))
	}
}

func TestDuplicateSeedRejected(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")

	declare(t, ac, base, "name", "example.com").Body.Close()
	got := refusalPage(t, ac, base, declare(t, ac, base, "name", "example.com"))
	if !strings.Contains(got, "already declared") {
		t.Fatalf("duplicate not reported; body: %s", got)
	}
}

func TestViewerCannotDeclareButCanView(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")

	ac := login(t, base, "admin", "hunter2hunter2")
	declare(t, ac, base, "name", "example.com").Body.Close()

	vc := login(t, base, "viewer", "hunter2hunter2")

	resp := declare(t, vc, base, "name", "example.org")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("viewer declare: status=%d, want 403", resp.StatusCode)
	}
	if len(f.seeds) != 1 {
		t.Fatalf("seeds after denied declare = %d, want 1", len(f.seeds))
	}

	page := seedsBody(t, vc, base)
	if !strings.Contains(page, "example.com") {
		t.Errorf("viewer cannot see the seeds list; body: %s", page)
	}
	if strings.Contains(page, `action="/seeds"`) {
		t.Errorf("declare form shown to a viewer; body: %s", page)
	}
}

func scopeMain(body string) string {
	i := strings.Index(body, "<main")
	j := strings.LastIndex(body, "</main>")
	if i < 0 || j < 0 || j < i {
		return body
	}
	return body[i:j]
}

func TestScopeDeclaredNameTree(t *testing.T) {
	f := newFakeStore()
	admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	addNameSeed(t, f, admin.ID, "example.com")
	// An internal address resolved from the internet class fires a medium-severity Name rule.
	f.addClassResolution(t, "leak.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["10.0.0.5"]}`)
	f.addClassResolution(t, "www.example.com", "internet", obsClock, `{"outcome":"Resolved","addresses":["93.184.216.34"]}`)

	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	main := scopeMain(seedsBody(t, ac, base))

	for _, want := range []string{"Declared name tree", `class="sc-tree"`} {
		if !strings.Contains(main, want) {
			t.Errorf("scope missing name-tree marker %q", want)
		}
	}
	for _, want := range []string{
		`class="tl">example.com<`,
		`class="tc">2<`,
		`class="tl">leak<`,
		`class="tl">www<`,
	} {
		if !strings.Contains(main, want) {
			t.Errorf("name tree missing %q; body: %s", want, main)
		}
	}
	if !strings.Contains(main, "var(--sev-medium-dot)") {
		t.Errorf("name tree leaf lost its severity dot; body: %s", main)
	}
}

func TestDeclareRequiresLogin(t *testing.T) {
	base := start(t, newFakeStore(), "")
	c := newClient(t)
	resp := declare(t, c, base, "name", "example.com")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("anon declare: status=%d location=%q, want redirect to /login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestScopeServesWhenOneRegionReadFails(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail func(*fakeStore)
	}{
		{"exclusions", func(f *fakeStore) { f.exclusionsErr = errors.New("list exclusions failed") }},
		{"vantages", func(f *fakeStore) { f.vantagesErr = errors.New("list vantages failed") }},
		{"proposals", func(f *fakeStore) { f.pendingPropsErr = errors.New("list proposals failed") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			admin := seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			addNameSeed(t, f, admin.ID, "example.com")
			tc.fail(f)

			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")

			page := getBody(t, ac, base+"/scope", http.StatusOK)
			if !strings.Contains(page, "example.com") {
				t.Errorf("a failed %s read cost the screen its declared scopes; body: %s", tc.name, page)
			}
		})
	}
}

func (f *fakeStore) CreateNameSeed(_ context.Context, arg db.CreateNameSeedParams) (db.Seed, error) {
	for _, s := range f.seeds {
		if s.Kind == "name" && s.NameDomain.String == arg.NameDomain.String {
			return db.Seed{}, &pgconn.PgError{Code: "23505", Message: "duplicate seed"}
		}
	}
	sd := db.Seed{
		ID: f.seedNextID, Kind: "name", NameDomain: arg.NameDomain, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.seeds = append(f.seeds, sd)
	f.seedNextID++
	f.actTrail = append(f.actTrail, "CreateNameSeed")
	return sd, nil
}

func (f *fakeStore) WithdrawSeed(_ context.Context, arg db.WithdrawSeedParams) (db.WithdrawSeedRow, error) {
	for i, s := range f.seeds {
		if s.ID != arg.SeedID {
			continue
		}
		f.seeds = append(f.seeds[:i], f.seeds[i+1:]...)
		f.actTrail = append(f.actTrail, "WithdrawSeed")
		w := db.SeedWithdrawal{
			ID:        int64(len(f.seedWithdrawals) + 1),
			Kind:      s.Kind,
			CreatedBy: arg.CreatedBy,
		}
		// The scope rides the DELETE's own RETURNING, so it comes back even with no tombstone.
		out := db.WithdrawSeedRow{SeedsRemoved: 1, AddressCidr: s.AddressCidr, NameDomain: s.NameDomain}
		switch {
		case s.Kind == "address" && s.AddressCidr != nil:
			w.AddressCidr = s.AddressCidr
		case s.Kind == "name" && s.NameDomain.Valid:
			w.NameDomain = s.NameDomain
		default:
			return out, nil
		}
		f.seedWithdrawals = append(f.seedWithdrawals, w)
		out.TombstonesWritten = 1
		return out, nil
	}
	return db.WithdrawSeedRow{}, nil
}

func (f *fakeStore) ListSeedWithdrawalCandidates(_ context.Context, cidrs []string) ([]db.ListSeedWithdrawalCandidatesRow, error) {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, p)
	}
	out := []db.ListSeedWithdrawalCandidatesRow{}
	for _, row := range f.withdrawalCandidates {
		key := row.SubjectKey
		if i := strings.IndexByte(key, ':'); i >= 0 {
			key = key[:i]
		}
		addr, err := netip.ParseAddr(key)
		if err != nil {
			continue
		}
		for _, p := range prefixes {
			if p.Contains(addr) {
				out = append(out, row)
				break
			}
		}
	}
	return out, nil
}

func (f *fakeStore) ListNameSeedWithdrawalCandidates(_ context.Context, domains []string) ([]db.ListNameSeedWithdrawalCandidatesRow, error) {
	out := []db.ListNameSeedWithdrawalCandidatesRow{}
	for _, row := range f.nameWithdrawalCandidates {
		key := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(row.SubjectKey)), ".")
		for _, d := range domains {
			if key == d || strings.HasSuffix(key, "."+d) {
				out = append(out, row)
				break
			}
		}
	}
	return out, nil
}

func (f *fakeStore) ListAdmittedNamesOutsideSeed(_ context.Context, seedID int64) ([]string, error) {
	seen := map[string]bool{}
	out := []string{}
	for _, a := range f.admitted {
		if a.SeedID == seedID || seen[a.Name] {
			continue
		}
		seen[a.Name] = true
		out = append(out, a.Name)
	}
	sort.Strings(out)
	return out, nil
}

func (f *fakeStore) ListExclusions(context.Context) ([]db.Exclusion, error) {
	if f.exclusionsErr != nil {
		return nil, f.exclusionsErr
	}
	rows := make([]db.Exclusion, 0, len(f.exclusions))
	for i := len(f.exclusions) - 1; i >= 0; i-- {
		e := f.exclusions[i]
		rows = append(rows, db.Exclusion{
			ID: e.ID, Kind: e.Kind, Name: e.Name, AddressCidr: e.AddressCidr,
			CreatedBy: e.CreatedBy, CreatedAt: e.CreatedAt,
		})
	}
	return rows, nil
}

func (f *fakeStore) CreateZoneFile(_ context.Context, arg db.CreateZoneFileParams) (db.CreateZoneFileRow, error) {
	f.zoneFiles = append(f.zoneFiles, fakeZoneFile{
		seedID: arg.SeedID, suppliedAt: arg.SuppliedAt.Time, content: arg.Content, uploadedBy: arg.UploadedBy,
	})
	f.zoneNextID++
	return db.CreateZoneFileRow{ID: f.zoneNextID, SuppliedAt: arg.SuppliedAt}, nil
}

func (f *fakeStore) SetZoneCadenceSeconds(_ context.Context, cadenceSeconds int64) error {
	f.zoneCadence = cadenceSeconds
	return nil
}

func (f *fakeStore) SetDnsCadenceSeconds(_ context.Context, cadenceSeconds int64) error {
	f.dnsCadence = cadenceSeconds
	return nil
}
