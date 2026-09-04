package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"mime/multipart"
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
	"github.com/winniel123/verge-asm/internal/queue"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/seed"
	"github.com/winniel123/verge-asm/internal/signal"
)

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/scope.tmpl"))

type seedView struct {
	ID               int64
	IsAddress        bool
	Scope            string
	Anchor           string
	By               string
	At               string
	CustodyExtension bool
}

// A field with a setter and no hole is a refusal the operator never sees.

type seedsForms struct {
	seedError, seedScope                         string
	exclError, exclKind, exclValue               string
	custodyError                                 string
	zoneIntervalError                            string
	zoneErrors                                   []zoneErrorView
	zoneIntervalDays                             string
	proposalError, proposalNotice, proposalQuery string
	exclPreview                                  *message.NarrowingReceipt
	seedConfirm                                  *seedConfirmView
	refusals                                     []refusalView
}

type zoneErrorView struct {
	File   string
	Reason string
}

func nameScopes(views []seedView) []seedView {
	out := make([]seedView, 0, len(views))
	for _, v := range views {
		if !v.IsAddress {
			out = append(out, v)
		}
	}
	return out
}

func (s *server) backToScope(w http.ResponseWriter, r *http.Request) {
	// A refusal returns where a success does, so the operator keeps their scroll offset (ADR-0130 §3).
	s.redirectBack(w, r, "/scope")
}

func (s *server) flashScopeBack(w http.ResponseWriter, r *http.Request, f seedsForms) {
	// The exclusion preview is no refusal, but it needs the same thing: to survive the redirect.
	stashFormFlash(s, r, f)
	s.backToScope(w, r)
}

func (s *server) flashScopeToastBack(w http.ResponseWriter, r *http.Request, f seedsForms, tone, title, desc string) {
	// The scroll key drops the receipt (backurl.go stripToastParam), so a toast never moves the stash.
	stashFormFlash(s, r, f)
	s.toastRedirectBack(w, r, "/scope", tone, title, desc)
}

func (s *server) takeScopeFlash(r *http.Request) seedsForms {
	// /scope is this surface's only landing GET, so no per-surface claim check is needed.
	f, _ := takeFormFlash[seedsForms](s, r)
	return f
}

func (s *server) seedsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// Deriving the design's curated figures from live reads would fabricate domain data.
	if s.devMode {
		s.render(w, r, "scope", s.scopeFixtureData(acct, scopeOverlay{}))
		return
	}
	s.renderSeeds(w, r, acct, s.takeScopeFlash(r))
}

func (s *server) declareSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := r.FormValue("scope")

	if s.devMode {
		s.render(w, r, "scope", s.scopeFixtureDataRefusal(acct, strings.TrimSpace(raw)))
		return
	}

	// One tokenizer, not a fork: onboarding's TagInput commits on the same boundary.
	tokens := parseSeedTokens(raw)
	if len(tokens) == 0 {
		s.backToScope(w, r)
		return
	}

	// Read off instance_config, so a raise on the Settings dial binds the next paste (ADR-0127).
	addrCap := s.addressCap(r.Context())

	declared := make(map[string]bool, len(tokens))
	var refusals []refusalView
	successes := 0
	for _, tok := range tokens {
		if ref := s.declareOneScope(r, acct, tok, declared, addrCap); ref != nil {
			refusals = append(refusals, *ref)
		} else {
			successes++
		}
	}

	if successes > 0 {
		title := fmt.Sprintf("%d %s declared", successes, plural(successes, "scope", "scopes"))
		desc := ""
		if len(refusals) > 0 {
			desc = fmt.Sprintf("%d refused — see the callouts", len(refusals))
		}
		if len(refusals) == 0 {
			s.toastRedirectBack(w, r, "/scope", "neutral", title, desc)
			return
		}
		s.flashScopeToastBack(w, r, seedsForms{
			refusals:  refusals,
			seedScope: joinRefusedInputs(refusals),
		}, "neutral", title, desc)
		return
	}

	s.flashScopeBack(w, r, seedsForms{
		refusals:  refusals,
		seedScope: joinRefusedInputs(refusals),
		seedError: allRefusedFormError(refusals, addrCap),
	})
}

func (s *server) declareOneScope(r *http.Request, acct db.Account, value string, declared map[string]bool, addrCap int) *refusalView {
	value = strings.TrimSpace(value)
	if isAddressValue(value) {
		if _, err := seed.ParseCIDR(cidrForm(value)); err != nil {
			return &refusalView{Input: value, Reason: err.Error()}
		}
		// The unmasked form is kept so the callout echoes the operator's own base, not the network.
		rawP, _ := netip.ParsePrefix(strings.TrimSpace(cidrForm(value)))
		if !seed.WithinCap(rawP, addrCap) {
			ref := refusalOverCap(value, rawP, addrCap)
			return &ref
		}
		p := rawP.Masked()
		key := "addr:" + p.String()
		if declared[key] {
			return &refusalView{Input: value, Reason: alreadyDeclaredReason}
		}
		if _, err := s.store.CreateAddressSeed(r.Context(), db.CreateAddressSeedParams{
			AddressCidr: &p, CreatedBy: acct.ID,
		}); err != nil {
			return createRefusal(value, err)
		}
		declared[key] = true
		return nil
	}
	domain, err := seed.NormalizeDomain(value)
	if err != nil {
		return &refusalView{Input: value, Reason: err.Error()}
	}
	key := "name:" + domain
	if declared[key] {
		return &refusalView{Input: value, Reason: alreadyDeclaredReason}
	}
	if _, err := s.store.CreateNameSeed(r.Context(), db.CreateNameSeedParams{
		NameDomain: pgtype.Text{String: domain, Valid: true}, CreatedBy: acct.ID,
	}); err != nil {
		return createRefusal(value, err)
	}
	declared[key] = true
	return nil
}

const alreadyDeclaredReason = "already declared"

func createRefusal(value string, err error) *refusalView {
	if isUniqueViolation(err) {
		return &refusalView{Input: value, Reason: alreadyDeclaredReason}
	}
	return &refusalView{Input: value, Reason: "could not be declared"}
}

func joinRefusedInputs(refusals []refusalView) string {
	// A token can hold no comma, since comma is a split boundary, so ", " re-joins unambiguously.
	parts := make([]string, 0, len(refusals))
	for _, rv := range refusals {
		parts = append(parts, rv.Input)
	}
	return strings.Join(parts, ", ")
}

func allRefusedFormError(refusals []refusalView, cap int) string {
	if len(refusals) == 1 {
		if refusals[0].Reachable != "" {
			return overCapFormError(cap)
		}
		return refusals[0].Reason
	}
	return fmt.Sprintf("%d refused — see the callouts.", len(refusals))
}

type seedConfirmView struct {
	// The dev fixture's ids are strings and the live rows' are int64; one printf compares both.

	ID       string
	Scope    string
	Fires    bool
	Headline string
	Loss     string
	Failed   bool
}

func (s *server) previewSeedWithdrawal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A reload consumes the confirm state and abandons the withdrawal, which is the safe default.
	if s.devMode {
		s.render(w, r, "scope", s.scopeFixtureDataConfirm(acct, r.FormValue("id")))
		return
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.flashScopeBack(w, r, seedsForms{seedError: "That scope could not be found."})
		return
	}
	scope, isAddress := s.seedScopeByID(r, id)
	if scope == "" {
		s.backToScope(w, r)
		return
	}
	// A failed count degrades the block; refusing the act would leave no route to the withdrawal.
	confirm := seedConfirmView{ID: strconv.FormatInt(id, 10), Scope: scope}
	var receipt message.NarrowingReceipt
	var rerr error
	if isAddress {
		p, perr := netip.ParsePrefix(scope)
		if perr != nil {
			s.serverError(w, "parse seed scope", perr)
			return
		}
		receipt, rerr = queue.SeedWithdrawalReceipt(r.Context(), s.store, s.now().UTC(), p)
	} else {
		receipt, rerr = queue.NameSeedWithdrawalReceipt(r.Context(), s.store, id, scope)
	}
	if rerr != nil {
		log.Printf("web: preview seed withdrawal %s: %v", scope, rerr)
		confirm.Failed = true
	} else {
		confirm.Fires = receipt.Fires
		confirm.Headline = receipt.Headline
		confirm.Loss = receipt.Loss
	}
	s.flashScopeBack(w, r, seedsForms{seedConfirm: &confirm})
}

func (s *server) deleteSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.backToScope(w, r)
		return
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.flashScopeBack(w, r, seedsForms{seedError: "That scope could not be found."})
		return
	}
	scope, _ := s.seedScopeByID(r, id)
	// The delete and the tombstone commit together, so no withdrawn scope lacks a mover (ADR-0135 §2).
	if _, err := s.store.WithdrawSeed(r.Context(), db.WithdrawSeedParams{
		SeedID: id, CreatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "withdraw seed", err)
		return
	}
	if scope == "" {
		s.backToScope(w, r)
		return
	}
	s.toastRedirectBack(w, r, "/scope", "neutral", "Scope removed", removalFlash(scope))
}

func removalFlash(scope string) string {
	// Both limbs enforce now, so one sentence is true of either scope (ADR-0134, ADR-0135).
	return scope + " — nothing new is admitted under it; the subjects it alone held " +
		"leave the estate on the next completed job."
}

func (s *server) seedScopeByID(r *http.Request, id int64) (string, bool) {
	rows, err := s.store.ListSeeds(r.Context())
	if err != nil {
		return "", false
	}
	for _, v := range toSeedViews(rows) {
		if v.ID == id {
			return v.Scope, v.IsAddress
		}
	}
	return "", false
}

type refusalView struct {
	Input     string
	Reason    string
	Reachable string
}

func isAddressValue(v string) bool {
	if strings.Contains(v, "/") {
		return true
	}
	_, err := netip.ParseAddr(v)
	return err == nil
}

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

func overCapFormError(cap int) string {
	return fmt.Sprintf("Refused — over the %s-address cap.", commaInt(cap))
}

func refusalOverCap(value string, raw netip.Prefix, cap int) refusalView {
	// The over-cap set is named, never applied: the operator declares the narrower block themselves.
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

func commaGroup(n string) string {
	// An address count can exceed a fixed-width int, so this twin groups the big.Int string.
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

type custodyView struct {
	ID               int64
	Scope            string
	CustodyExtension bool
	Census           int
}

func toCustodyViews(nameSeeds []seedView) []custodyView {
	// No measured resolution numerator exists yet, so this reads zero, never a fabricated figure.
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
	seeds := toSeedViews(rows)
	nameSeeds := nameScopes(seeds)
	intervalDays := f.zoneIntervalDays
	if intervalDays == "" {
		intervalDays = strconv.FormatInt(cadence/86400, 10)
	}
	var nameTree []nameTreeNode
	if corpus, cerr := s.buildSignalCorpus(r); cerr == nil {
		nameTree = declaredNameTree(nameSeeds, corpus.Names, signal.EvaluateCorpus(corpus))
	}
	// An additive card's failed read degrades that card, never the whole Scope screen.
	census, censusErr := s.custodyCensus(r.Context())
	data := map[string]any{
		"Title": "Scope", "NavActive": "scope",
		"Account": acct, "IsAdmin": acct.Role == roleAdmin,
		// The head block inlines tokens/*.css only when this datum is set.
		"DesignTokens": true,
		"Seeds":        seeds, "AddressCap": s.addressCap(r.Context()),
		"NameTree":     nameTree,
		"CoverageMsgs": coverageMessages(probers),
		"FormError":    f.seedError, "FormScope": f.seedScope,
		"Refusals":   f.refusals,
		"Exclusions": toExclusionViews(excl),
		"ExclError":  f.exclError, "ExclKind": f.exclKind, "ExclValue": f.exclValue,
		"CustodyScopes": toCustodyViews(nameSeeds), "CustodyError": f.custodyError,
		"CustodyCensus": census.Rows, "CustodyCensusFailed": censusErr != nil,
		// A pending candidate carries no remedy and clears within one cadence (#1015).
		"CustodyCensusPending": census.Pending,
		"ZoneScopes":           toZoneViews(nameSeeds, zoneStatus, cadence, s.now().UTC()),
		"ZoneErrors":           f.zoneErrors,
		"ZoneIntervalError":    f.zoneIntervalError,
		"ZoneIntervalDays":     intervalDays,
		"Proposals":            flattenProposals(lookups), "OrgQuery": f.proposalQuery,
		"ProposalError": f.proposalError,
		"ExclPreview":   f.exclPreview,
		"SeedConfirm":   f.seedConfirm,
	}
	if f.proposalNotice != "" {
		data["Notice"] = f.proposalNotice
	}
	// A refusal answers 200 exactly as a success does, so the shell restores the offset (ADR-0130 §1).
	s.render(w, r, "scope", data)
}

// Coverage is its own language — gap, staleness, silence — and never a severity (CONTEXT.md).

type coverageMsgView struct {
	Kind    string
	Badge   string
	Bound   string
	Subject string
	Text    string
	When    string
	ISO     string
}

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

type nameTreeNode struct {
	ID       string
	Label    string
	Sev      string
	HasCount bool
	Count    int
	Children []nameTreeNode
}

func declaredNameTree(nameSeeds []seedView, names []signal.NameFacts, censuses []signal.Census) []nameTreeNode {
	// The same subject-keyed rollup the AssetDetail header reads (assetHeaderSeverity, subjects.go).
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
				continue
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

func seedAnchor(scope string) string {
	var b strings.Builder
	// The message renderer slugs the same key, so a widening lands on the moved Seed (v1 spec §5.3).
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

const maxZoneUpload = 8 << 20

// Bounding the whole body before any parse is what stops an oversize upload exhausting memory.

const maxTotalZoneUpload = 64 << 20

type zoneView struct {
	SeedID        int64
	Domain        string
	HasFile       bool
	SuppliedAt    string
	By            string
	Bytes         int64
	AgingStale    bool
	AgingLabel    string
	IntervalLabel string
}

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

func (s *server) uploadZoneFile(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// The upload instant is the observation instant, never our later read (docs/spec/v1-spec.md §3.4).
	if s.devMode {
		s.backToScope(w, r)
		return
	}
	// A zone file is evidence, not a secret, so the shared database holds it (v1 spec §4.2).
	r.Body = http.MaxBytesReader(w, r.Body, maxTotalZoneUpload)
	if err := r.ParseMultipartForm(maxZoneUpload); err != nil { // #nosec G120 (request body bounded by the MaxBytesReader immediately above; per-part 8 MiB cap enforced on read)
		// The submitting URL went with the rest of the body, so bare /scope is the honest destination.
		s.flashScopeBack(w, r, seedsForms{zoneErrors: []zoneErrorView{{
			Reason: "The upload was too large or malformed. A zone file is text, up to 8 MB.",
		}}})
		return
	}
	var files []*multipart.FileHeader
	if r.MultipartForm != nil {
		files = r.MultipartForm.File["zonefile"]
	}
	if len(files) == 0 {
		s.flashScopeBack(w, r, seedsForms{zoneErrors: []zoneErrorView{{
			Reason: "Choose a zone file to upload.",
		}}})
		return
	}

	// Equal instants tie-break by insertion order (zone.sql), so the last file for an apex wins.
	now := s.now().UTC()
	var zoneErrors []zoneErrorView
	accepted := 0
	for _, fh := range files {
		if ref := s.uploadOneZoneFile(r, acct, fh, now); ref != nil {
			zoneErrors = append(zoneErrors, *ref)
		} else {
			accepted++
		}
	}

	if accepted > 0 {
		title := fmt.Sprintf("%d zone %s supplied", accepted, plural(accepted, "file", "files"))
		desc := ""
		if len(zoneErrors) > 0 {
			desc = fmt.Sprintf("%d refused", len(zoneErrors))
		}
		if len(zoneErrors) == 0 {
			s.toastRedirectBack(w, r, "/scope", "neutral", title, desc)
			return
		}
		s.flashScopeToastBack(w, r, seedsForms{zoneErrors: zoneErrors}, "neutral", title, desc)
		return
	}
	s.flashScopeBack(w, r, seedsForms{zoneErrors: zoneErrors})
}

func (s *server) uploadOneZoneFile(r *http.Request, acct db.Account, fh *multipart.FileHeader, now time.Time) *zoneErrorView {
	name := fh.Filename
	file, err := fh.Open()
	if err != nil {
		return &zoneErrorView{File: name, Reason: "could not be read"}
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxZoneUpload+1))
	if err != nil {
		return &zoneErrorView{File: name, Reason: "could not be read"}
	}
	if len(content) == 0 {
		return &zoneErrorView{File: name, Reason: "the file is empty"}
	}
	if len(content) > maxZoneUpload {
		return &zoneErrorView{File: name, Reason: "over the 8 MB cap"}
	}
	// An apex outside every declared name scope is refused, never attached to a scope nobody named.
	apex := zoneApex(string(content))
	if apex == "" {
		return &zoneErrorView{File: name, Reason: "not a zone file"}
	}
	seedID, ok := s.nameSeedForApex(r, apex)
	if !ok {
		return &zoneErrorView{File: name, Reason: fmt.Sprintf(
			"the zone's apex %s is outside every declared name scope — declare it as a name scope first, or upload the zone for a scope you hold.", apex)}
	}
	if _, err := s.store.CreateZoneFile(r.Context(), db.CreateZoneFileParams{
		SeedID:     seedID,
		SuppliedAt: pgtype.Timestamptz{Time: now, Valid: true},
		Content:    string(content),
		UploadedBy: acct.ID,
	}); err != nil {
		return &zoneErrorView{File: name, Reason: "could not be stored"}
	}
	return nil
}

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

func (s *server) setZoneInterval(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := strings.TrimSpace(r.FormValue("interval_days"))
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 {
		s.flashScopeBack(w, r, seedsForms{
			zoneIntervalError: "Enter a re-supply interval of at least one day.",
			zoneIntervalDays:  raw,
		})
		return
	}
	if err := s.store.SetZoneCadenceSeconds(r.Context(), int64(days)*86400); err != nil {
		s.serverError(w, "set zone cadence", err)
		return
	}
	s.backToScope(w, r)
}

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
