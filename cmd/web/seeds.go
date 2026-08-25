package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/seed"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Scope screen (screen 10, batch 3 · #574) is served byte-for-byte from the
// frozen design-owned design-system/templates/scope.tmpl (package v3.9.0, WORKFLOW
// v4), which replaces BOTH the repo-authored "scope" define AND the "proposals"
// define (templates_scope.go + proposals.go proposalTemplates, deleted). The tmpl
// renders inside the full app chrome ({{template "chrome" .}}) and declares the holes
// renderSeeds shapes below: .Notice .IsAdmin .AddressCap .Seeds[{ID,Anchor,Scope,
// IsAddress}] .FormScope .FormError .Refusal{Input,Reason,Reachable}(nullable)
// .CustodyScopes[{ID,Scope,CustodyExtension,Census}] .CustodyError .ZoneScopes[{ID,
// Domain,HasFile,SuppliedAt,IntervalLabel,AgingLabel}] .ZoneError .ZoneIntervalDays
// .ZoneIntervalError .NameTree[{Label,Count,Sev,Children[{Label,Sev}]}] .CoverageMsgs[
// {Kind,Badge,Bound,Subject,Text,When,ISO}] .Proposals[{ID,Value,Kind,Source}] .OrgQuery
// .Exclusions[{ID,Kind,Value}] .ExclError .ExclKind .ExclValue .ExclPreview{Fires,
// Headline,Loss}. It styles against the design token vocabulary, so the render opts in
// with DesignTokens:true (the "head" block inlines tokens/*.css only then). scope.tmpl
// auto-embeds through designfs's existing templates/*.tmpl glob, so no designfs.go
// change is needed. Reconciliations (SPEC-CHANGE #21, ruled): the seed kind select drops
// (declareSeed infers name/address from the value shape, #21a); an over-cap block REFUSES
// with the reachable /22 named via .Refusal (never auto-corrects); custody renders the
// spec toggle + census once per name scope (#21b); zone upload is the spec FileDrop, the
// apex inferred from the uploaded file (#21c); the cold-tier + prober regions relocate to
// /settings (#21d). Forms keep their POST routes.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/scope.tmpl"))

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
	// refusal is the spec RefusalCallout (#21a): set alongside seedError when a scope
	// declaration is refused for being wider than the address cap. Nil otherwise.
	refusal *refusalView
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
	// VERGE_DEV pixel-parity path (#574). The frozen scope.tmpl renders a curated
	// corpus — the two seeds, the custody census, the seven-leaf name tree, the three
	// coverage messages, the proposals and exclusions — whose exact strings, ordering
	// and derived figures (the census, the aging label, the name-tree severities) are
	// the design's, not a live-estate read. Reproducing them from the live derivations
	// would mean fabricating domain data, which SPEC-CHANGE forbids — so, exactly as the
	// Coverage/Exposure screens pin their dev fixture and serve it under devMode with a
	// drift test (TestScopeFixtureMatchesPackage), seedsPage serves the pinned
	// fixtures.json → scope slice here so the seeded candidate renders byte-for-byte what
	// the golden composes. A real deployment (devMode == false) falls through to the
	// honest live reads below.
	if s.devMode {
		s.render(w, "scope", s.scopeFixtureData(acct, scopeOverlay{}))
		return
	}
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
//
// #21a: the seed form no longer carries a kind select — the handler infers name
// vs. address from the value's SHAPE (a slash or a bare address literal is an
// address scope; anything else is a name). An address block wider than the cap is
// REFUSED, never auto-corrected: the RefusalCallout (.Refusal) names the reachable
// in-cap set (the /22 that fits the base) for the operator to declare themselves.
func (s *server) declareSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	value := strings.TrimSpace(r.FormValue("scope"))

	// VERGE_DEV pixel-parity: the scope "refusal" golden posts 203.0.113.0/20 through
	// the seed form (states.json). Serve the pinned fixture + the RefusalCallout so the
	// candidate renders byte-for-byte what the golden composes, without touching the DB.
	if s.devMode {
		s.render(w, "scope", s.scopeFixtureDataRefusal(acct, value))
		return
	}

	fail := func(f seedsForms) {
		s.renderSeeds(w, r, acct, f)
	}

	if isAddressValue(value) {
		if _, err := seed.ParseCIDR(cidrForm(value)); err != nil {
			fail(seedsForms{seedError: err.Error(), seedScope: value})
			return
		}
		// seed.ParseCIDR validated the block, so the raw (unmasked) re-parse cannot fail.
		// The raw form is kept for the callout so its Input/Reachable echo the operator's
		// own base address rather than the masked network address.
		raw, _ := netip.ParsePrefix(strings.TrimSpace(cidrForm(value)))
		if !seed.WithinCap(raw, s.seedAddressCap) {
			ref := refusalOverCap(value, raw, s.seedAddressCap)
			fail(seedsForms{
				seedError: overCapFormError(s.seedAddressCap), seedScope: value, refusal: &ref,
			})
			return
		}
		p := raw.Masked()
		if _, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
			AddressCidr: &p, CreatedBy: acct.ID,
		}); err != nil {
			fail(seedsForms{seedError: seedCreateError(err, "block"), seedScope: value})
			return
		}
	} else {
		domain, err := seed.NormalizeDomain(value)
		if err != nil {
			fail(seedsForms{seedError: err.Error(), seedScope: value})
			return
		}
		if _, err := s.store.CreateNameSeed(r.Context(), db.CreateNameSeedParams{
			NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: acct.ID,
		}); err != nil {
			fail(seedsForms{seedError: seedCreateError(err, "domain"), seedScope: value})
			return
		}
	}
	// A save fires a toast across the post-redirect-get (PARITY-CHART P1.7).
	s.toastRedirect(w, r, "/scope", "neutral", "Seed added", value+" enters scope; it is scanned on cadence.")
}

// deleteSeed withdraws a declared Seed by id — the Scope chip-remove act (#21a). It
// is admin-only (requireAdmin) and idempotent: removing a row already gone satisfies
// the operator's intent either way, so a stale chip submit redirects back cleanly
// rather than erroring.
func (s *server) deleteSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		http.Redirect(w, r, "/scope", http.StatusSeeOther)
		return
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSeeds(w, r, acct, seedsForms{seedError: "That scope could not be found."})
		return
	}
	if _, err := s.store.DeleteSeed(r.Context(), id); err != nil {
		s.serverError(w, "delete seed", err)
		return
	}
	http.Redirect(w, r, "/scope", http.StatusSeeOther)
}

// refusalView is the spec RefusalCallout (#21a): a declaration the handler refused
// because it is wider than the address cap. It carries the rejected Input verbatim,
// the Reason in the operator's words, and the Reachable in-cap set it NAMES but never
// auto-applies — nothing is corrected for the operator. Nil unless a declaration was
// refused; set on the render map alongside .FormError via the seedsForms echo.
type refusalView struct {
	Input     string
	Reason    string
	Reachable string
}

// isAddressValue reports whether a declared scope value is an address scope by its
// shape (#21a): a CIDR block (carries a slash) or a bare address literal. Everything
// else is a name scope.
func isAddressValue(v string) bool {
	if strings.Contains(v, "/") {
		return true
	}
	_, err := netip.ParseAddr(v)
	return err == nil
}

// cidrForm turns a bare address literal into its single-host CIDR so a value declared
// without a prefix length still parses as an address scope. A value already carrying a
// slash is returned unchanged.
func cidrForm(v string) string {
	v = strings.TrimSpace(v)
	if strings.Contains(v, "/") {
		return v
	}
	if a, err := netip.ParseAddr(v); err == nil {
		if a.Is4() {
			return v + "/32"
		}
		return v + "/128"
	}
	return v
}

// overCapFormError is the inline error shown in the seed field when a block is refused
// over the cap — the terse line the tmpl renders in place of the hint.
func overCapFormError(cap int) string {
	return fmt.Sprintf("Refused — over the %s-address cap.", commaInt(cap))
}

// refusalOverCap builds the RefusalCallout for an over-cap block (#21a). Input echoes
// the operator's typed value; Reason states the span against the cap; Reachable is the
// largest prefix that fits the cap, anchored at the value's own base address (never
// re-masked) so the operator sees a set they can declare as-is. The reachable prefix
// length is derived: host bits = floor(log2(cap)), so the reachable length is the
// address width minus those bits (a /22 for the 1,024-address cap).
func refusalOverCap(value string, raw netip.Prefix, cap int) refusalView {
	bits := raw.Addr().BitLen()
	host := 0
	for host+1 <= bits && (1<<(host+1)) <= cap {
		host++
	}
	reachLen := bits - host
	if reachLen < 0 {
		reachLen = 0
	}
	return refusalView{
		Input:     value,
		Reason:    fmt.Sprintf("Spans %s addresses — the cap is %s per scope.", commaGroup(seed.AddressCount(raw).String()), commaInt(cap)),
		Reachable: netip.PrefixFrom(raw.Addr(), reachLen).String(),
	}
}

// commaGroup renders an integer STRING with thousands separators, so the refusal
// callout reads "4,096" over an address count that may exceed a fixed-width int (the
// same grouping humanCount and commaInt apply). commaInt (auth.go) does the same over
// an int; this twin takes the big.Int string AddressCount returns.
func commaGroup(n string) string {
	neg := strings.HasPrefix(n, "-")
	if neg {
		n = n[1:]
	}
	var b strings.Builder
	for i, c := range n {
		if i > 0 && (len(n)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// custodyView is a name scope shaped for the custody-extension section (#21b): the
// scope, whether its custody extension is declared, and the Census — the count of
// addresses the extension currently reaches, recomputed each batch. A live estate has
// no first-class census numerator yet, so the live render carries zero; the fixture-
// seeded instance the golden depicts pins the real figure (scopeFixtureData).
type custodyView struct {
	ID               int64
	Scope            string
	CustodyExtension bool
	Census           int
}

// toCustodyViews shapes the custody-extension section from the name scopes. The census
// is zero on a live estate (no measured resolution numerator yet); it is never
// fabricated.
func toCustodyViews(nameSeeds []seedView) []custodyView {
	out := make([]custodyView, 0, len(nameSeeds))
	for _, v := range nameSeeds {
		out = append(out, custodyView{
			ID: v.ID, Scope: v.Scope, CustodyExtension: v.CustodyExtension,
		})
	}
	return out
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
	status := http.StatusOK
	if f.seedError != "" || f.exclError != "" || f.custodyError != "" ||
		f.zoneError != "" || f.zoneIntervalError != "" || f.proposalError != "" {
		status = http.StatusBadRequest
	}
	seeds := toSeedViews(rows)
	nameSeeds := nameScopes(seeds)
	intervalDays := f.zoneIntervalDays
	if intervalDays == "" {
		intervalDays = strconv.FormatInt(cadence/86400, 10)
	}
	// The declared name tree (SPEC-CHANGE collision #12, ADR-0116): registrable
	// domains → leaf names with per-leaf severity, folded off the live signal corpus.
	// Best-effort and additive — a corpus read failure degrades the card to its empty
	// pattern rather than 500ing the whole Scope screen.
	var nameTree []nameTreeNode
	if corpus, cerr := s.buildSignalCorpus(r); cerr == nil {
		nameTree = declaredNameTree(nameSeeds, corpus.Names, signal.EvaluateCorpus(corpus))
	}
	data := map[string]any{
		"Title": "Scope", "NavActive": "scope",
		"Account": acct, "IsAdmin": acct.Role == roleAdmin,
		// scope.tmpl styles against the design token vocabulary; the "head" block
		// inlines tokens/*.css only when this datum is set (as the batch-2 screens do).
		"DesignTokens": true,
		"Seeds":        seeds, "AddressCap": s.seedAddressCap,
		// The declared name tree (Scope.jsx:86-98): registrable domains → leaf names,
		// each carrying its own max-of-firing-signals severity.
		"NameTree": nameTree,
		// Coverage messages folded onto Scope (#278): the honest coverage-fact read
		// this screen can make from data it already holds — a provisioned vantage we
		// currently cannot look from is a silence, exactly what the design system's
		// CoverageMessageList carries. The full aperture statement lives on /coverage
		// (owned elsewhere); nothing here is fabricated.
		"CoverageMsgs": coverageMessages(probers),
		"FormError":    f.seedError, "FormScope": f.seedScope,
		// The RefusalCallout (#21a): set alongside FormError when a block is over-cap.
		"Refusal":    f.refusal,
		"Exclusions": toExclusionViews(excl),
		"ExclError":  f.exclError, "ExclKind": f.exclKind, "ExclValue": f.exclValue,
		// The custody-extension section reads name scopes alone — an address scope can
		// never carry one — each with its per-name census meter (#21b).
		"CustodyScopes": toCustodyViews(nameSeeds), "CustodyError": f.custodyError,
		// The zone-file section (#21c): the status rows show which name scopes hold a
		// supplied file, and the interval dial is the declared re-supply cadence. The
		// FileDrop infers the apex from the uploaded file, so no per-scope select.
		"ZoneScopes": toZoneViews(nameSeeds, zoneStatus, cadence, s.now().UTC()),
		"ZoneError":  f.zoneError, "ZoneIntervalError": f.zoneIntervalError,
		"ZoneIntervalDays": intervalDays,
		// Pending Proposals flattened to the spec rows + the org-name search echo (#21).
		"Proposals": flattenProposals(lookups), "OrgQuery": f.proposalQuery,
		// The narrowing receipt (#205 AC8): shown before an exclusion commits, only
		// where a withdrawal message would fire.
		"ExclPreview": f.exclPreview,
	}
	if f.proposalNotice != "" {
		data["Notice"] = f.proposalNotice
	}
	s.renderStatus(w, status, "scope", data)
}

// coverageMsgView is one coverage fact shaped for the Scope screen's coverage
// card, after design-system CoverageMessageList: a badge, the subject it is about,
// and the fact in the operator's words. It is never a severity — coverage is its
// own language (gap / staleness / silence), and this screen only ever fills it
// from real reads, never a fabricated example.
type coverageMsgView struct {
	// Kind drives the badge (a dotted GapBadge for "gap", a bronze staleness chip
	// otherwise) — never the severity ramp (#21). Bound is the staleness chip's
	// trailing figure (e.g. "9d"), empty where the chip carries none; ISO is the
	// full RFC-3339 instant rendered as the When column's title tooltip.
	Kind    string
	Badge   string
	Bound   string
	Subject string
	Text    string
	When    string
	ISO     string
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
			Kind:    "silent",
			Badge:   "silent",
			Subject: v.Name,
			Text: "Vantage is unreachable, so its most recent batches covered nothing. " +
				"What it would have measured is recorded as a Gap, not a clean empty result.",
		})
	}
	return out
}

// nameTreeNode is one node of the Scope "Declared name tree" (Scope.jsx:86-98,
// SPEC-CHANGE collision #12): a registrable-domain root (a declared name-scope
// Seed) or a leaf Name under it. Sev is the Name's own max-of-firing-signals
// severity token (critical|high|medium|low|info), empty where the Name raises no
// signal — the leaf then renders no severity dot, the spec's per-leaf empty
// pattern. Count is the number of leaves and is set on roots only.
type nameTreeNode struct {
	ID       string
	Label    string
	Sev      string
	HasCount bool
	Count    int
	Children []nameTreeNode
}

// declaredNameTree builds the Scope "Declared name tree" from the current model
// (ADR-0116: build the datum the design renders, never re-skin it). Each declared
// name-scope Seed is a registrable-domain root; every current in-estate Name under
// it is a leaf, labelled by the sub-name left after the domain suffix. Each Name —
// root apex and leaf alike — carries its own max-of-firing-signals severity: the
// most urgent (lowest-rank) severity across the signals whose subject IS that Name,
// exactly the rollup the AssetDetail header reads (assetHeaderSeverity/assetSignals
// in subjects.go), keyed off the same signal corpus. A Name with no firing signal
// carries no severity, so its dot degrades away.
func declaredNameTree(nameSeeds []seedView, names []signal.NameFacts, censuses []signal.Census) []nameTreeNode {
	// Per-Name max severity: most urgent rule severity among fired members keyed on
	// the Name itself — the Name-rule population is keyed by the Name, so this is the
	// same subject==key filter assetSignals uses, rolled up like assetHeaderSeverity.
	sevByName := map[string]signal.Severity{}
	for _, c := range censuses {
		sev, ok := signal.SeverityFor(c.Rule)
		if !ok {
			continue
		}
		for _, m := range c.Fired {
			if cur, seen := sevByName[m.Subject]; !seen || sev.Rank() < cur.Rank() {
				sevByName[m.Subject] = sev
			}
		}
	}

	// The leaf universe: every current in-estate Name, sorted for a deterministic
	// tree. A Name measured out of the estate (a cross-class NameError) is not a leaf.
	estate := make([]string, 0, len(names))
	for _, n := range names {
		if n.InEstate {
			estate = append(estate, n.Name)
		}
	}
	sort.Strings(estate)

	roots := make([]nameTreeNode, 0, len(nameSeeds))
	for _, ns := range nameSeeds {
		domain := ns.Scope
		root := nameTreeNode{ID: domain, Label: domain, HasCount: true}
		if sev, ok := sevByName[domain]; ok {
			root.Sev = sev.String()
		}
		suffix := "." + domain
		for _, name := range estate {
			if name == domain || !strings.HasSuffix(name, suffix) {
				continue // the apex is the root itself; names outside the domain are other roots' leaves
			}
			leaf := nameTreeNode{ID: name, Label: strings.TrimSuffix(name, suffix)}
			if sev, ok := sevByName[name]; ok {
				leaf.Sev = sev.String()
			}
			root.Children = append(root.Children, leaf)
		}
		root.Count = len(root.Children)
		roots = append(roots, root)
	}
	return roots
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
	if s.devMode {
		http.Redirect(w, r, "/scope", http.StatusSeeOther)
		return
	}
	fail := func(msg string) {
		s.renderSeeds(w, r, acct, seedsForms{zoneError: msg})
	}
	if err := r.ParseMultipartForm(maxZoneUpload); err != nil {
		fail("The upload was too large or malformed. A zone file is text, up to 8 MB.")
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
	// #21c: the seed select drops — the handler infers the scope from the file's apex
	// ($ORIGIN, or the SOA owner). An apex outside every declared name scope is REFUSED
	// with the reason, never silently attached to a scope the operator did not name.
	apex := zoneApex(string(content))
	if apex == "" {
		fail("Could not read the zone's apex — the file has no $ORIGIN and no SOA record to infer it from.")
		return
	}
	seedID, ok := s.nameSeedForApex(r, apex)
	if !ok {
		fail(fmt.Sprintf("The zone's apex %s is outside every declared name scope — declare it as a name scope first, or upload the zone for a scope you hold.", apex))
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

// zoneApex extracts an uploaded zone file's apex — the $ORIGIN directive when present,
// otherwise the owner of the SOA record — as a bare (lowercased, trailing-dot-stripped)
// domain. It returns "" when neither can be read, so the caller refuses rather than
// guessing.
func zoneApex(content string) string {
	var origin, soaOwner string
	for _, raw := range strings.Split(content, "\n") {
		if i := strings.IndexByte(raw, ';'); i >= 0 {
			raw = raw[:i]
		}
		if strings.TrimSpace(raw) == "" {
			continue
		}
		fields := strings.Fields(raw)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "$ORIGIN") {
			origin = strings.TrimSuffix(strings.ToLower(fields[1]), ".")
			continue
		}
		if soaOwner == "" && raw[0] != ' ' && raw[0] != '\t' && fields[0] != "@" {
			for _, f := range fields[1:] {
				if strings.EqualFold(f, "SOA") {
					soaOwner = strings.TrimSuffix(strings.ToLower(fields[0]), ".")
					break
				}
			}
		}
	}
	if origin != "" {
		return origin
	}
	return soaOwner
}

// nameSeedForApex resolves a zone apex to the id of the name-scope Seed that holds it
// — an exact match on the registrable domain, or an apex that resolves under one. It
// reports ok=false when no name scope covers the apex, so the upload is refused (#21c).
func (s *server) nameSeedForApex(r *http.Request, apex string) (int64, bool) {
	rows, err := s.store.ListSeeds(r.Context())
	if err != nil {
		return 0, false
	}
	for _, row := range rows {
		if row.Kind != "name" || !row.NameDomain.Valid {
			continue
		}
		domain := strings.ToLower(row.NameDomain.String)
		if apex == domain || strings.HasSuffix(apex, "."+domain) {
			return row.ID, true
		}
	}
	return 0, false
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
