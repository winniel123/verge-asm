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

// The Scope screen (screen 10, batch 3 · #574) is served byte-for-byte from the
// frozen design-owned design-system/templates/scope.tmpl (package v3.9.0, WORKFLOW
// v4), which replaces BOTH the repo-authored "scope" define AND the "proposals"
// define (templates_scope.go + proposals.go proposalTemplates, deleted). The tmpl
// renders inside the full app chrome ({{template "chrome" .}}) and declares the holes
// renderSeeds shapes below: .Notice .IsAdmin .AddressCap .Seeds[{ID,Anchor,Scope,
// IsAddress}] .FormScope .FormError .Refusals[{Input,Reason,Reachable(nullable)}] (DF-F1,
// one per refused paste token; replaces the single .Refusal)
// .CustodyScopes[{ID,Scope,CustodyExtension,Census}] .CustodyError .ZoneScopes[{ID,
// Domain,HasFile,SuppliedAt,IntervalLabel,AgingLabel}] .ZoneErrors[{File,Reason}] (DF-F2,
// one per refused file; replaces .ZoneError) .ZoneIntervalDays
// .ZoneIntervalError .NameTree[{Label,Count,Sev,Children[{Label,Sev}]}] .CoverageMsgs[
// {Kind,Badge,Bound,Subject,Text,When,ISO}] .Proposals[{ID,Value,Kind,Source}] .OrgQuery
// .Exclusions[{ID,Kind,Value}] .ExclError .ExclKind .ExclValue .ExclPreview{Fires,
// Headline,Loss} .SeedConfirm{ID,Scope,Fires,Headline,Loss,Failed} (#1046, the chip
// under confirmation). It styles against the design token vocabulary, so the render opts in
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
//
// It carries only what /scope actually renders. Four fields went in ticket #978 because
// nothing on this surface read them any more: seedKind (the seed form's kind select,
// dropped in #21a when declareSeed began inferring the shape from the value), and the
// coldError and prober echoes (the cold-tier and provisioning regions relocated to
// Settings in #21d, where settingsForms carries the same names). A field with a setter
// and no hole is a refusal the operator never sees, so this struct holds none.
type seedsForms struct {
	seedError, seedScope           string
	exclError, exclKind, exclValue string
	custodyError                   string
	zoneIntervalError              string
	// zoneErrors are the per-file zone-upload refusals (DF-F2): one row per rejected
	// file, in upload order. It replaces the single zoneError string — a bulk upload
	// refuses each file independently.
	zoneErrors []zoneErrorView
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
	// seedConfirm is the chip the operator clicked remove on, held for the confirm
	// step (#1046). Nil when no chip is under confirmation, which is every render but
	// the landing of a preview.
	seedConfirm *seedConfirmView
	// refusals are the per-token declaration refusals (DF-F1): one RefusalCallout per
	// refused token in a paste, in declaration order. It replaces the single .Refusal
	// hole — a paste declares many scopes at once, each validated independently.
	refusals []refusalView
}

// zoneErrorView is one per-file zone-upload refusal (DF-F2): the file's name and the
// reason it was refused (apex outside the name scopes, or not a zone file). It replaces
// the single .ZoneError hole so a bulk upload lists a row per rejected file.
type zoneErrorView struct {
	File   string
	Reason string
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

// backToScope answers a mutating scope act with the 303 ADR-0130 §3 asks for: back to
// the URL the form was submitted from, falling back to bare /scope when the form
// carried no `return` that passed the guard (backurl.go).
//
// A success and a refusal share it. The destination rule is the same for both — that is
// the point of the contract, since a refusal the operator cannot tell apart from a
// success is a refusal that keeps their scroll offset — and only the flash differs.
//
// The scope surface needs no dialogParams twin of the settings helper. Every form on
// this screen renders in the page itself. No query parameter on /scope opens a modal,
// so there is nothing to drop from the destination.
func (s *server) backToScope(w http.ResponseWriter, r *http.Request) {
	s.redirectBack(w, r, "/scope")
}

// flashScopeBack stashes f as this session's pending scope form and 303s back to the
// submitting URL. It is the whole of the migration for a caller: an inline
// `renderSeeds(w, r, acct, seedsForms{...})` becomes `flashScopeBack(w, r,
// seedsForms{...})` with the same struct, and seedsPage renders it on the landing.
//
// It carries a REFUSAL and it also carries the narrowing preview, which is not one. The
// exclusion preview is a receipt the operator reads before they commit (#205 AC8), and
// it rides the same carrier because it needs the same thing a refusal does: to survive
// the redirect and render on the landing GET. Only renderSeeds tells the two apart, and
// it does so off the error fields, not off the carrier.
//
// /scope is the surface's ONLY landing GET, so this needs no claim check of the kind
// takeFormFlashIf exists for. The flash is typed, so a rejected form belonging to
// another surface is left in place rather than consumed here.
func (s *server) flashScopeBack(w http.ResponseWriter, r *http.Request, f seedsForms) {
	stashFormFlash(s, r, f)
	s.backToScope(w, r)
}

// flashScopeToastBack is flashScopeBack with a success toast: it stashes f AND fires one
// toast on the landing. It is the MIXED bulk result — a paste or an upload where some
// items committed and others were refused.
//
// That case is why seedsForms used to carry an inline toast. A mixed result must show
// the success receipt and every per-item callout in ONE response, and before ADR-0130 a
// redirect would have dropped the callouts, so the toast rode the inline render instead.
// Under §1 the callouts ride the session flash through the redirect, so both arrive on
// the landing GET and the workaround is gone: the toast takes the ordinary `toast=`
// query carrier every other success uses.
//
// The receipt is no part of the page's identity, on either side. backURL and resolveBack
// strip `toast` from the submitting URL, and the §2 scroll key ignores it (shell.tmpl),
// so appending one here does not move the operator's stash.
func (s *server) flashScopeToastBack(w http.ResponseWriter, r *http.Request, f seedsForms, tone, title, desc string) {
	stashFormFlash(s, r, f)
	s.toastRedirectBack(w, r, "/scope", tone, title, desc)
}

// takeScopeFlash reads this session's pending scope form off the flash carrier. It is
// the GET half of the ADR-0130 §1 post-redirect-get, and /scope is the one landing that
// calls it.
//
// A GET that is nobody's landing takes nothing and renders a zero seedsForms, which is
// the ordinary read. The render needs no flag distinguishing the two, because it answers
// 200 either way — see renderSeeds.
func (s *server) takeScopeFlash(r *http.Request) seedsForms {
	f, _ := takeFormFlash[seedsForms](s, r)
	return f
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
		s.render(w, r, "scope", s.scopeFixtureData(acct, scopeOverlay{}))
		return
	}
	// A refused scope act redirected here and left its callouts, the operator's typed
	// values, and any narrowing preview in the session-keyed form flash (ADR-0130 §1,
	// flash.go). Read it once, here, and hand it to the render. The read is
	// single-consume, so a reload of the same URL shows a clean page.
	//
	// The partial-lookup caveat rides the same carrier (#251, proposals.go runLookup).
	// It used to travel as a `?notice=` query flag, which kept the search idempotent on
	// refresh but changed the URL — and a landing URL that differs from the submitting
	// one is exactly the §2 scroll-key miss this map exists to close. The flash keeps
	// the idempotence and the destination both.
	//
	// The VERGE_DEV branch above returns before this read, and declareSeed's own dev
	// branch still renders its refusal in place. Both are the pixel-parity harness, which
	// posts fixed states to compare bytes and never rides the contract; a flash stashed
	// under VERGE_DEV is simply retired by the TTL. A real deployment takes neither path.
	s.renderSeeds(w, r, acct, s.takeScopeFlash(r))
}

// declareSeed handles a scope declaration. It is reached only through
// requireAdmin, so a viewer can list seeds but never declare one.
//
// #21a: the seed form no longer carries a kind select — the handler infers name
// vs. address from the value's SHAPE (a slash or a bare address literal is an
// address scope; anything else is a name). An address block wider than the cap is
// REFUSED, never auto-corrected: the RefusalCallout names the reachable in-cap set
// (the /22 that fits the base) for the operator to declare themselves.
//
// DF-F1 (paste-split): the `scope` field is a RAW string the operator may paste
// several scopes into. It is tokenized on commas, whitespace and newlines by the
// SAME parseSeedTokens onboarding's seedsadd uses (cmd/web/onboarding.go) — the one
// tokenizer, not a fork — and each token is validated and committed independently as
// its own dated act. Successes fire a flash; each failure fills a .Refusals[] callout
// in declaration order. A token that duplicates one already declared in the same paste
// (or a pre-existing seed) is refused `already declared`.
func (s *server) declareSeed(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := r.FormValue("scope")

	// VERGE_DEV pixel-parity: the scope "refusal" golden posts 203.0.113.0/20 through
	// the seed form (states.json). Serve the pinned fixture + the RefusalCallout so the
	// candidate renders byte-for-byte what the golden composes, without touching the DB.
	if s.devMode {
		s.render(w, r, "scope", s.scopeFixtureDataRefusal(acct, strings.TrimSpace(raw)))
		return
	}

	// The shared tokenizer (DF-F1): commas / whitespace / newlines split, empty tokens
	// drop. This is parseSeedTokens verbatim — the exact commit boundary the onboarding
	// TagInput uses — so the two entry points can never diverge.
	tokens := parseSeedTokens(raw)
	if len(tokens) == 0 {
		// Nothing to declare — an empty submit is a no-op redirect, no flash, no error.
		s.backToScope(w, r)
		return
	}

	// The operator address-scope cap, read once per paste off the instance_config
	// singleton (#888, ADR-0127) rather than a compiled constant, so a raise on the
	// Settings control takes effect on the next declaration. Read here, not per token,
	// so one paste is one store read; every token in the paste checks the same cap.
	addrCap := s.addressCap(r.Context())

	// declared tracks the normalized keys committed in THIS paste so a duplicate token
	// within one paste is refused `already declared` even before the DB unique constraint
	// would catch it — the first token declares, the second refuses (DF-F1 edge).
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
			// Pure success: a plain post-redirect-get back to the submitting URL fires the
			// toast (PARITY-CHART P1.7).
			s.toastRedirectBack(w, r, "/scope", "neutral", title, desc)
			return
		}
		// Mixed: the callouts must render AND the toast must fire, and under ADR-0130 §1
		// both survive one redirect — the callouts in the session flash, the toast in the
		// `toast` query. The operator lands back on their own URL with the whole result.
		s.flashScopeToastBack(w, r, seedsForms{
			refusals:  refusals,
			seedScope: joinRefusedInputs(refusals),
		}, "neutral", title, desc)
		return
	}

	// All-refused: no toast, callouts only. The field-level line reddens the input and
	// replaces the hint; a single over-cap token keeps its terse cap line (and its
	// TestAddressScopeOverCapRejected contract), any other single refusal shows its
	// reason, and a multi-token paste summarizes the count.
	s.flashScopeBack(w, r, seedsForms{
		refusals:  refusals,
		seedScope: joinRefusedInputs(refusals),
		seedError: allRefusedFormError(refusals, addrCap),
	})
}

// declareOneScope validates and commits ONE pasted scope token (DF-F1). It returns nil
// on a committed declaration, or a *refusalView describing why the token was refused —
// an over-cap block (with the reachable in-cap set named), an unparseable value, or a
// duplicate (`already declared`). declared holds the normalized keys already committed
// in this paste so a within-paste duplicate refuses before the DB is touched. Each
// success is its own dated act (a distinct CreateSeed call).
func (s *server) declareOneScope(r *http.Request, acct db.Account, value string, declared map[string]bool, addrCap int) *refusalView {
	value = strings.TrimSpace(value)
	if isAddressValue(value) {
		if _, err := seed.ParseCIDR(cidrForm(value)); err != nil {
			return &refusalView{Input: value, Reason: err.Error()}
		}
		// seed.ParseCIDR validated the block, so the raw (unmasked) re-parse cannot fail.
		// The raw form is kept for the callout so its Input/Reachable echo the operator's
		// own base address rather than the masked network address.
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

// alreadyDeclaredReason is the refusal reason for a token that duplicates a scope
// already declared — within the same paste or a pre-existing seed (DF-F1).
const alreadyDeclaredReason = "already declared"

// createRefusal maps a CreateSeed error to a refusal: a unique-constraint violation
// means the scope is already declared; any other error is an opaque failure.
func createRefusal(value string, err error) *refusalView {
	if isUniqueViolation(err) {
		return &refusalView{Input: value, Reason: alreadyDeclaredReason}
	}
	return &refusalView{Input: value, Reason: "could not be declared"}
}

// joinRefusedInputs joins the refused tokens back into the input field so the operator
// can edit and resubmit just the ones that failed (the successes are already committed).
// Tokens never contain a comma (comma is a split boundary), so ", " re-joins cleanly.
func joinRefusedInputs(refusals []refusalView) string {
	parts := make([]string, 0, len(refusals))
	for _, rv := range refusals {
		parts = append(parts, rv.Input)
	}
	return strings.Join(parts, ", ")
}

// allRefusedFormError builds the field-level error line for an all-refused paste. A
// single over-cap token keeps its terse cap line (Reachable is set only for over-cap
// refusals); any other single refusal shows its reason; a multi-token paste names the
// count and points at the callouts.
func allRefusedFormError(refusals []refusalView, cap int) string {
	if len(refusals) == 1 {
		if refusals[0].Reachable != "" {
			return overCapFormError(cap)
		}
		return refusals[0].Reason
	}
	return fmt.Sprintf("%d refused — see the callouts.", len(refusals))
}

// seedConfirmView is the one chip under confirmation and the receipt its withdrawal
// would produce (#1046). ID is the chip's id as a STRING because the dev fixture's
// seed rows carry string ids and the live rows carry int64, and the template compares
// the two through one printf.
//
// The copy is the stored message's own: Headline and Loss are the strings
// message.PreviewSeedWithdrawal renders, so the confirm step and the coverage message
// the fold writes read as one sentence. Fires is false where the count is zero, and
// the template renders no receipt block at all there — the exclusion preview's
// non-firing sentence states a model rule that is false of this act.
// Failed is the honest degrade, after .CustodyCensusFailed on the same screen: the
// count read did not resolve, so the block says so rather than rendering a zero the
// estate does not hold. The withdrawal stays available — see previewSeedWithdrawal.
type seedConfirmView struct {
	ID       string
	Scope    string
	Fires    bool
	Headline string
	Loss     string
	Failed   bool
}

// previewSeedWithdrawal is the FIRST of the chip-remove act's two steps (#1046). It
// withdraws nothing. It computes the narrowing receipt the withdrawal would produce
// and re-renders the chip in a confirm state, so the act that closes timelines shows
// its count before the operator commits — as declaring an exclusion already does.
//
// It rides the same one-shot scope flash the exclusion preview rides, so the landing
// GET consumes the confirm state: reloading or navigating away abandons the
// withdrawal and leaves the Seed declared. The carrier holds one chip, so at most one
// is under confirmation.
//
// A stale chip whose row is already gone redirects cleanly rather than erroring,
// matching the withdrawal's own idempotency.
//
// BOTH LIMBS CARRY A COUNT (ADR-0135). Each reads its own receipt — the address
// scope counts what falls under its CIDR, the name scope what falls under its domain
// — and each applies its own survivor rules, so the confirm step states the act the
// fold will actually perform. The name limb stated none until #1045, when withdrawing
// a name Seed still closed nothing.
func (s *server) previewSeedWithdrawal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// VERGE_DEV pixel-parity: the fixture corpus is not an estate, so there is no
	// honest count to state over it. The confirm state renders with no receipt block,
	// which is what a zero count renders anyway, and no database is touched.
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
	// A FAILED COUNT DEGRADES THE BLOCK, IT DOES NOT REFUSE THE ACT. The receipt
	// reads the candidate spans and the corpora the survivors are decided from, and
	// this step is now the ONLY route to the withdrawal — the chip's control reaches
	// /seeds/delete through here. A 500 on any of those reads would leave the
	// operator with no way to withdraw the scope at all, over a count that is
	// advisory by construction (ADR-0134 §5). So the confirm step renders, says the
	// count did not resolve, and still offers the act.
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

// deleteSeed withdraws a declared Seed by id — the SECOND of the chip-remove act's
// two steps (#21a, #1046). It is admin-only (requireAdmin) and idempotent: removing a
// row already gone satisfies the operator's intent either way, so a stale chip submit
// redirects back cleanly rather than erroring.
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
	// Resolve the scope's display string BEFORE the withdrawal so the removal flash can
	// name it (WORK-ORDER-DOGFOOD-R1 item 2). A stale chip whose row is already gone
	// leaves scope empty and simply redirects — the act stays idempotent.
	scope, _ := s.seedScopeByID(r, id)
	// The delete and the tombstone the withdrawal owes commit together (ADR-0134 §2,
	// ADR-0135 §2), so no path can leave a withdrawn scope of either kind with no
	// mover for the membership fold to name.
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

// removalFlash is the sentence the removal toast states about what the act does to
// the subjects already in the estate.
//
// BOTH limbs are enforcing now (ADR-0134 for the address scope, ADR-0135 for the
// name scope), so the two say one thing. The name limb said the opposite until
// #1045 — "existing subjects keep their citations" — which was a plain statement of
// the bug, and the address limb said it until #1040.
func removalFlash(scope string) string {
	return scope + " — nothing new is admitted under it; the subjects it alone held " +
		"leave the estate on the next completed job."
}

// seedScopeByID returns the display scope for a declared seed id — the address CIDR for
// an address scope, the domain for a name scope — and whether it is an address scope,
// or "" when no such seed exists. It reuses toSeedViews so the string matches the chip
// the operator clicked.
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
	// The custody-extension census (#987): the declines and the held candidates the
	// `edge-fanout` veto produced. Best-effort and additive, exactly like the name
	// tree above — a failed read degrades this one section to an honest note rather
	// than 500ing the Scope screen, and it NEVER falls back to a fabricated row.
	census, censusErr := s.custodyCensus(r.Context())
	data := map[string]any{
		"Title": "Scope", "NavActive": "scope",
		"Account": acct, "IsAdmin": acct.Role == roleAdmin,
		// scope.tmpl styles against the design token vocabulary; the "head" block
		// inlines tokens/*.css only when this datum is set (as the batch-2 screens do).
		"DesignTokens": true,
		"Seeds":        seeds, "AddressCap": s.addressCap(r.Context()),
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
		// The RefusalCallouts (DF-F1): one per refused token in a paste, declaration
		// order. Replaces the single .Refusal hole.
		"Refusals":   f.refusals,
		"Exclusions": toExclusionViews(excl),
		"ExclError":  f.exclError, "ExclKind": f.exclKind, "ExclValue": f.exclValue,
		// The custody-extension section reads name scopes alone — an address scope can
		// never carry one — each with its per-name census meter (#21b).
		"CustodyScopes": toCustodyViews(nameSeeds), "CustodyError": f.custodyError,
		// The custody-extension census (#987, ADR-0129 §5): the in-zone names whose
		// direct-A edge the extension DECLINED, each with its citing name and the
		// address-scope remedy. .CustodyCensusFailed is the honest degrade — the
		// section says the read did not resolve rather than showing a row nothing
		// measured.
		"CustodyCensus": census.Rows, "CustodyCensusFailed": censusErr != nil,
		// The held candidates collapse to this one count (#1015). A pending candidate
		// carries no remedy and clears within one Scan cadence, so a row each would
		// make the section's worst render its first one.
		"CustodyCensusPending": census.Pending,
		// The zone-file section (#21c): the status rows show which name scopes hold a
		// supplied file, and the interval dial is the declared re-supply cadence. The
		// FileDrop infers the apex from the uploaded file, so no per-scope select.
		"ZoneScopes": toZoneViews(nameSeeds, zoneStatus, cadence, s.now().UTC()),
		// The per-file zone-upload refusals (DF-F2): one row per rejected file. Replaces
		// the single .ZoneError hole.
		"ZoneErrors":        f.zoneErrors,
		"ZoneIntervalError": f.zoneIntervalError,
		"ZoneIntervalDays":  intervalDays,
		// Pending Proposals flattened to the spec rows + the org-name search echo (#21).
		// ProposalError is the refusal beside that search box. The field had two setters
		// and no hole between the parity conversion (#574, which dropped both the hole and
		// this line) and ticket #978's sweep, so an empty search and an already-declared
		// scope each refused in silence — the operator saw a clean page and no reason.
		"Proposals": flattenProposals(lookups), "OrgQuery": f.proposalQuery,
		"ProposalError": f.proposalError,
		// The narrowing receipt (#205 AC8): shown before an exclusion commits, only
		// where a withdrawal message would fire.
		"ExclPreview": f.exclPreview,
		// The chip under confirmation and the receipt its withdrawal would produce
		// (#1046). The withdrawal is the same class of act as an exclusion, so it
		// states the same count before it commits.
		"SeedConfirm": f.seedConfirm,
	}
	if f.proposalNotice != "" {
		data["Notice"] = f.proposalNotice
	}
	// Always 200, callouts or none. This render has one caller, seedsPage, and it is a
	// GET: a refusal reaches it only as the landing of a post-redirect-get (ADR-0130 §1),
	// which is an ordinary navigation. It used to answer 400 when it rendered a refusal
	// in place at the POST URL, and that branch went with the last handler that did.
	// A refusal now answers exactly as a success does, which is what lets the shell
	// restore the operator's scroll offset on both.
	s.render(w, r, "scope", data)
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

// maxTotalZoneUpload bounds the WHOLE multipart request body (DF-F2 allows N
// zonefile parts in one POST). It caps total bytes read off the wire before any
// parse, so a hostile or accidental oversize body cannot exhaust memory; the
// per-file 8 MiB cap still applies to each accepted part.
const maxTotalZoneUpload = 64 << 20 // 64 MiB

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
// DF-F2 (bulk zone-file upload): the drop/picker is `multiple`, so one multipart POST
// carries N `zonefile` parts. Each file is processed independently — its apex inferred
// from the content, checked against the declared name scopes — and each accepted file
// records its own dated act at the shared upload instant (the observation instant; the
// zone status query tie-breaks equal instants by insertion order, so two files for one
// apex leave the later as the current supply). Per-file refusals fill .ZoneErrors[]; a
// flash fires on ≥1 accepted file.
func (s *server) uploadZoneFile(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.backToScope(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTotalZoneUpload)
	if err := r.ParseMultipartForm(maxZoneUpload); err != nil { // #nosec G120 (request body bounded by the MaxBytesReader immediately above; per-part 8 MiB cap enforced on read)
		// A body this handler could not parse carries no `return` field either, so the
		// redirect falls back to bare /scope. That is the honest destination: the
		// submitting URL is exactly what was lost with the rest of the body.
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

	// A single upload instant for the whole drop — every accepted file's observation
	// instant. Equal instants tie-break by insertion order (zone.sql ORDER BY id DESC),
	// so files pasted for the same apex settle with the last as the current supply.
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
		// Mixed: refusal rows AND the success toast, both carried through one redirect
		// back to the submitting URL (see flashScopeToastBack).
		s.flashScopeToastBack(w, r, seedsForms{zoneErrors: zoneErrors}, "neutral", title, desc)
		return
	}
	// Zero accepted: no toast, refusal rows only.
	s.flashScopeBack(w, r, seedsForms{zoneErrors: zoneErrors})
}

// uploadOneZoneFile stores ONE uploaded zone file (DF-F2). It returns nil on a recorded
// supply act, or a *zoneErrorView naming the file and why it was refused: unreadable or
// empty content, an unparseable file (`not a zone file`), or an apex outside every
// declared name scope. now is the shared upload instant — this file's observation
// instant (v1 spec §3.4).
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
	// #21c/DF-F2: the handler infers the scope from the file's apex ($ORIGIN, or the SOA
	// owner). An unparseable file has no apex; an apex outside every declared name scope
	// is refused, never silently attached to a scope the operator did not name.
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
