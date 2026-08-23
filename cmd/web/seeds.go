package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/seed"
)

// seedView is a declared Seed shaped for rendering: the scope collapsed to one
// display string, with the kind kept so name and address scopes stay visually
// distinct.
type seedView struct {
	ID        int64
	IsAddress bool
	Scope     string
	// Anchor is the row's in-page id — the seed-scoped fragment an
	// aperture-widening message links to so it lands on the exact Seed whose
	// scope moved, not merely the Seeds list (v1 spec §5.3). Built from Scope by
	// seedAnchor, which the message renderer uses for the same key so the two
	// agree.
	Anchor string
	By     string
	At     string
	// CustodyExtension is the name scope's declared custody extension — the
	// operator's assertion that the addresses its names resolve to are under
	// their Custody. Off by default and meaningful on name scopes alone.
	CustodyExtension bool
}

// seedsForms carries the echo state of the two forms the Seeds screen hosts —
// the scope declaration and the exclusion — so a rejected submission on one
// leaves its own error and typed value in place without disturbing the other.
type seedsForms struct {
	seedError, seedKind, seedScope                  string
	exclError, exclKind, exclValue                  string
	custodyError                                    string
	proberError, proberHost, proberPort, proberUser string
	zoneError, zoneIntervalError                    string
	coldError                                       string
	// zoneIntervalDays echoes a rejected interval so the admin need not retype
	// it; empty means render the stored dial.
	zoneIntervalDays string
	// The org-name lookup echo: an error keeps the search box populated on a
	// rejected submit, a notice reports a lookup that returned no candidates.
	proposalError, proposalNotice, proposalQuery string
	// exclPreview is the narrowing receipt shown before the operator commits an
	// exclusion (#205 AC8, ADR-0074): the count of what it would withdraw, and the
	// loss named — but only where a withdrawal message would actually fire. Nil
	// when no preview was requested.
	exclPreview *message.NarrowingReceipt
}

// nameScopes returns the name-scope subset of a seed listing, in the same order.
// The custody-extension section is over name scopes alone: an address scope is
// its own complete enumeration and carries no extension.
func nameScopes(views []seedView) []seedView {
	out := make([]seedView, 0, len(views))
	for _, v := range views {
		if !v.IsAddress {
			out = append(out, v)
		}
	}
	return out
}

func (s *server) seedsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	var f seedsForms
	// A partial-failure lookup redirects here with a notice flag rather than
	// rendering inline off its POST, so the caveat survives the redirect without
	// making a refresh re-file the proposals (#251).
	if r.URL.Query().Get("notice") == noticePartialProposals {
		f.proposalNotice = partialProposalNotice
	}
	s.renderSeeds(w, r, acct, f)
}

// declareSeed handles a scope declaration. It is reached only through
// requireAdmin, so a viewer can list seeds but never declare one.
func (s *server) declareSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	kind := r.FormValue("kind")
	value := strings.TrimSpace(r.FormValue("scope"))
	fail := func(msg string) {
		s.renderSeeds(w, r, acct, seedsForms{seedError: msg, seedKind: kind, seedScope: value})
	}

	switch kind {
	case "name":
		domain, err := seed.NormalizeDomain(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if _, err := s.store.CreateNameSeed(r.Context(), db.CreateNameSeedParams{
			NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: acct.ID,
		}); err != nil {
			fail(seedCreateError(err, "domain"))
			return
		}
	case "address":
		p, err := seed.ParseCIDR(value)
		if err != nil {
			fail(err.Error())
			return
		}
		if !seed.WithinCap(p, s.seedAddressCap) {
			fail(fmt.Sprintf(
				"%s covers %s addresses, over the cap of %d — declare a smaller block.",
				p, seed.AddressCount(p), s.seedAddressCap))
			return
		}
		if _, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
			AddressCidr: &p, CreatedBy: acct.ID,
		}); err != nil {
			fail(seedCreateError(err, "block"))
			return
		}
	default:
		fail("Choose a scope type.")
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

func (s *server) renderSeeds(w http.ResponseWriter, r *http.Request, acct db.Account, f seedsForms) {
	rows, err := s.store.ListSeeds(r.Context())
	if err != nil {
		s.serverError(w, "list seeds", err)
		return
	}
	excl, err := s.store.ListExclusions(r.Context())
	if err != nil {
		s.serverError(w, "list exclusions", err)
		return
	}
	probers, err := s.store.ListVantages(r.Context())
	if err != nil {
		s.serverError(w, "list vantages", err)
		return
	}
	zoneStatus, err := s.store.ListZoneFileStatus(r.Context())
	if err != nil {
		s.serverError(w, "list zone files", err)
		return
	}
	cadence, err := s.store.GetZoneCadenceSeconds(r.Context())
	if err != nil {
		s.serverError(w, "get zone cadence", err)
		return
	}
	lookups, err := s.proposalLookups(r.Context())
	if err != nil {
		s.serverError(w, "list proposals", err)
		return
	}
	coldOptedIn, err := s.store.ListColdScopeSeedIds(r.Context())
	if err != nil {
		s.serverError(w, "list cold scan scope", err)
		return
	}
	status := http.StatusOK
	if f.seedError != "" || f.exclError != "" || f.custodyError != "" || f.proberError != "" ||
		f.zoneError != "" || f.zoneIntervalError != "" || f.proposalError != "" || f.coldError != "" {
		status = http.StatusBadRequest
	}
	seeds := toSeedViews(rows)
	nameSeeds := nameScopes(seeds)
	intervalDays := f.zoneIntervalDays
	if intervalDays == "" {
		intervalDays = strconv.FormatInt(cadence/86400, 10)
	}
	s.renderStatus(w, status, "scope", map[string]any{
		"Title": "Scope", "NavActive": "scope",
		"Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Seeds": seeds, "AddressCap": s.seedAddressCap,
		// Coverage messages folded onto Scope (#278): the honest coverage-fact read
		// this screen can make from data it already holds — a provisioned vantage we
		// currently cannot look from is a silence, exactly what the design system's
		// CoverageMessageList carries. The full aperture statement lives on /coverage
		// (owned elsewhere); nothing here is fabricated.
		"CoverageMsgs": coverageMessages(probers),
		"FormError": f.seedError, "FormKind": f.seedKind, "FormScope": f.seedScope,
		"Exclusions": toExclusionViews(excl),
		"ExclError":  f.exclError, "ExclKind": f.exclKind, "ExclValue": f.exclValue,
		// The custody-extension section reads name scopes alone — an address
		// scope can never carry one.
		"CustodyScopes": nameSeeds, "CustodyError": f.custodyError,
		"Probers":     toProberViews(probers),
		"ProberError": f.proberError, "ProberHost": f.proberHost,
		"ProberPort": f.proberPort, "ProberUser": f.proberUser,
		// The zone-file section: the upload dropdown lists name scopes, the
		// status rows show which hold a supplied file, and the interval dial is
		// the operator's declared re-supply cadence.
		"ZoneScopes": toZoneViews(nameSeeds, zoneStatus, cadence, s.now().UTC()), "NameScopes": nameSeeds,
		"ZoneError": f.zoneError, "ZoneIntervalError": f.zoneIntervalError,
		"ZoneIntervalDays": intervalDays,
		// The cold Scan opt-in (#200): every declared scope with its full-range
		// opt-in state. The tier ships disabled with an empty scope list.
		"ColdScopes": toColdScopeViews(seeds, coldOptedIn), "ColdError": f.coldError,
		"ColdEnabled": len(coldOptedIn) > 0,
		// Pending Proposals and the org-name lookup echo (#210).
		"ProposalLookups": lookups,
		"ProposalError":   f.proposalError, "ProposalNotice": f.proposalNotice,
		"ProposalQuery": f.proposalQuery,
		// The narrowing receipt (#205 AC8): shown before an exclusion commits,
		// only where a withdrawal message would fire.
		"ExclPreview": f.exclPreview,
	})
}

// coverageMsgView is one coverage fact shaped for the Scope screen's coverage
// card, after design-system CoverageMessageList: a badge, the subject it is about,
// and the fact in the operator's words. It is never a severity — coverage is its
// own language (gap / staleness / silence), and this screen only ever fills it
// from real reads, never a fabricated example.
type coverageMsgView struct {
	Badge   string
	Subject string
	Text    string
	When    string
}

// coverageMessages derives the coverage-message list from the provisioned
// vantages the Scope render already reads. A vantage whose availability is
// `unavailable` is a position we currently cannot look from — its batches covered
// nothing, so the reach it would have measured is a Gap, not a clean empty result.
// That is a silence in coverage terms, and the honest thing to surface here. When
// every vantage is reporting, the list is empty and the card shows its empty state.
func coverageMessages(vantages []db.ListVantagesRow) []coverageMsgView {
	var out []coverageMsgView
	for _, v := range vantages {
		if v.Availability.String != "unavailable" {
			continue
		}
		out = append(out, coverageMsgView{
			Badge:   "silent",
			Subject: v.Name,
			Text: "Vantage is unreachable, so its most recent batches covered nothing. " +
				"What it would have measured is recorded as a Gap, not a clean empty result.",
		})
	}
	return out
}

func toSeedViews(rows []db.ListSeedsRow) []seedView {
	out := make([]seedView, 0, len(rows))
	for _, row := range rows {
		v := seedView{ID: row.ID, By: row.CreatedByUsername, CustodyExtension: row.CustodyExtension}
		if row.Kind == "address" && row.AddressCidr != nil {
			v.IsAddress = true
			v.Scope = row.AddressCidr.String()
		} else {
			v.Scope = row.NameDomain.String
		}
		v.Anchor = seedAnchor(v.Scope)
		if row.CreatedAt.Valid {
			v.At = row.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, v)
	}
	return out
}

// seedAnchor slugs a Seed's scope into a stable in-page anchor id. A scope key
// carries dots and (for a CIDR) a slash, so every run of non-alphanumeric octets
// collapses to a single '-': "198.51.100.0/24" -> "198-51-100-0-24",
// "example.com" -> "example-com". The message renderer builds the same slug from
// an aperture message's fired-at Seed key, so its `/scope#seed-<anchor>` link
// (#286) resolves to the row this stamps. A withdrawn Seed leaves no row and the link
// falls back to the list head, which is acceptable.
func seedAnchor(scope string) string {
	var b strings.Builder
	dash := false
	for _, r := range scope {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

func seedCreateError(err error, noun string) string {
	if isUniqueViolation(err) {
		return "That " + noun + " is already declared."
	}
	return "Could not declare the scope."
}

// maxZoneUpload bounds a single zone-file upload. A zone file is text and the
// modal operator's is small; the cap is a defence against an accidental huge
// upload, not a product limit.
const maxZoneUpload = 8 << 20 // 8 MiB

// zoneView is a name scope shaped for the zone-file section: the scope, and
// whether it currently holds a supplied file with its supply instant, uploader
// and size.
type zoneView struct {
	SeedID     int64
	Domain     string
	HasFile    bool
	SuppliedAt string
	By         string
	Bytes      int64
	// AgingStale reports that the supplied file has aged past the re-supply
	// interval into a coverage gap; AgingLabel is the warn-tone badge's copy
	// ("ages into a gap in 7d" while current, "aged into a gap 5d ago" once
	// stale). AgingLabel is empty when there is nothing to surface — no file, or
	// no cadence to age against.
	AgingStale bool
	AgingLabel string
	// IntervalLabel renders the operator's declared re-supply cadence for the
	// file line ("monthly", "weekly", or "every N days").
	IntervalLabel string
}

// toZoneViews decorates each name scope with its latest supplied zone file, if
// any, and computes the file's staleness → gap read against the operator's
// re-supply cadence. A scope with no file is shown too, as an empty state
// inviting an upload. now is the render instant, threaded from s.now().
func toZoneViews(nameSeeds []seedView, status []db.ListZoneFileStatusRow, cadenceSeconds int64, now time.Time) []zoneView {
	interval := time.Duration(cadenceSeconds) * time.Second
	intervalLabel := zoneIntervalLabel(cadenceSeconds)
	bySeed := make(map[int64]db.ListZoneFileStatusRow, len(status))
	for _, st := range status {
		bySeed[st.SeedID] = st
	}
	out := make([]zoneView, 0, len(nameSeeds))
	for _, s := range nameSeeds {
		v := zoneView{SeedID: s.ID, Domain: s.Scope, IntervalLabel: intervalLabel}
		if st, ok := bySeed[s.ID]; ok {
			v.HasFile = true
			v.By = st.UploadedByUsername
			v.Bytes = st.ContentBytes
			if st.SuppliedAt.Valid {
				v.SuppliedAt = st.SuppliedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
				if interval > 0 {
					a := scan.ZoneAgingAt(st.SuppliedAt.Time, now, interval)
					v.AgingStale = a.Stale
					v.AgingLabel = zoneAgingLabel(a)
				}
			}
		}
		out = append(out, v)
	}
	return out
}

// zoneAgingLabel renders a supplied file's staleness → gap read in the
// operator's words. A current file counts down to the gap; a stale file names
// the gap it has aged into. It never fabricates: the read is derived from the
// dated supply and the declared cadence alone.
func zoneAgingLabel(a scan.ZoneAging) string {
	if !a.Supplied {
		return ""
	}
	if !a.Stale {
		if a.Days == 0 {
			return "ages into a gap today"
		}
		return fmt.Sprintf("ages into a gap in %dd", a.Days)
	}
	if a.Days == 0 {
		return "aged into a gap today"
	}
	return fmt.Sprintf("aged into a gap %dd ago", a.Days)
}

// zoneIntervalLabel renders the re-supply cadence for the file line: the common
// cadences by name, anything else as "every N days".
func zoneIntervalLabel(cadenceSeconds int64) string {
	switch days := cadenceSeconds / 86400; days {
	case 0:
		return ""
	case 1:
		return "daily"
	case 7:
		return "weekly"
	case 30:
		return "monthly"
	default:
		return fmt.Sprintf("every %d days", days)
	}
}

// uploadZoneFile stores an operator's zone file for a name scope. The upload is
// the supply act, so its instant is recorded now — the zone Scan restates the
// file's observations at this instant, never at the worker's later read (v1 spec
// §3.4). The file is stored in the shared database so both web and worker read
// it; it is evidence, not a secret (§4.2).
func (s *server) uploadZoneFile(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.renderSeeds(w, r, acct, seedsForms{zoneError: msg})
	}
	if err := r.ParseMultipartForm(maxZoneUpload); err != nil {
		fail("The upload was too large or malformed. A zone file is text, up to 8 MB.")
		return
	}
	seedID, err := strconv.ParseInt(r.FormValue("seed_id"), 10, 64)
	if err != nil {
		fail("Choose a name scope to attach the zone file to.")
		return
	}
	if !s.isNameSeed(r, seedID) {
		fail("That name scope no longer exists.")
		return
	}
	file, _, err := r.FormFile("zonefile")
	if err != nil {
		fail("Choose a zone file to upload.")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxZoneUpload+1))
	if err != nil {
		fail("Could not read the uploaded file.")
		return
	}
	if len(content) == 0 {
		fail("The uploaded file is empty.")
		return
	}
	if len(content) > maxZoneUpload {
		fail("The zone file is over the 8 MB cap.")
		return
	}
	if _, err := s.store.CreateZoneFile(r.Context(), db.CreateZoneFileParams{
		SeedID:     seedID,
		SuppliedAt: pgtype.Timestamptz{Time: s.now().UTC(), Valid: true},
		Content:    string(content),
		UploadedBy: acct.ID,
	}); err != nil {
		s.serverError(w, "create zone file", err)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// setZoneInterval moves the re-supply interval dial: the operator's promise
// about how often they will re-export, held as the zone Scan's cadence and
// shipped at monthly (v1 spec §3.4).
func (s *server) setZoneInterval(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := strings.TrimSpace(r.FormValue("interval_days"))
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 {
		s.renderSeeds(w, r, acct, seedsForms{
			zoneIntervalError: "Enter a re-supply interval of at least one day.",
			zoneIntervalDays:  raw,
		})
		return
	}
	if err := s.store.SetZoneCadenceSeconds(r.Context(), int64(days)*86400); err != nil {
		s.serverError(w, "set zone cadence", err)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// isNameSeed reports whether id is a currently declared name-scope Seed.
func (s *server) isNameSeed(r *http.Request, id int64) bool {
	rows, err := s.store.ListSeeds(r.Context())
	if err != nil {
		return false
	}
	for _, row := range rows {
		if row.ID == id && row.Kind == "name" {
			return true
		}
	}
	return false
}
