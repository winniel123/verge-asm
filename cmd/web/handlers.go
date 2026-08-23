package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
	"github.com/winniel123/verge-asm/internal/seed"
)

// store is the slice of the database the web handlers use. Narrowing it to an
// interface lets tests supply an in-memory fake instead of a live Postgres,
// the same seam the scaffold's healthz handler introduced.
type store interface {
	RecordHeartbeat(ctx context.Context) (db.Heartbeat, error)
	CountAccounts(ctx context.Context) (int64, error)
	CreateAccount(ctx context.Context, arg db.CreateAccountParams) (db.Account, error)
	GetAccountByUsername(ctx context.Context, username string) (db.Account, error)
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
	SetTOTPSecret(ctx context.Context, arg db.SetTOTPSecretParams) error
	ConfirmTOTP(ctx context.Context, id int64) error
	CreateNameSeed(ctx context.Context, arg db.CreateNameSeedParams) (db.Seed, error)
	CreateAddressSeed(ctx context.Context, arg db.CreateAddressSeedParams) (db.Seed, error)
	ListSeeds(ctx context.Context) ([]db.ListSeedsRow, error)
	SetCustodyExtension(ctx context.Context, arg db.SetCustodyExtensionParams) error
	CreateNameExclusion(ctx context.Context, arg db.CreateNameExclusionParams) (db.Exclusion, error)
	CreateAddressExclusion(ctx context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error)
	ListExclusions(ctx context.Context) ([]db.ListExclusionsRow, error)
	DeleteExclusion(ctx context.Context, id int64) error
	ListSourceStates(ctx context.Context) ([]db.SourceState, error)
	UpsertSourceState(ctx context.Context, arg db.UpsertSourceStateParams) (db.SourceState, error)
	CreateVantage(ctx context.Context, arg db.CreateVantageParams) (db.Vantage, error)
	ListVantages(ctx context.Context) ([]db.ListVantagesRow, error)
	ListUnavailableVantages(ctx context.Context) ([]db.ListUnavailableVantagesRow, error)
	GetScanByKind(ctx context.Context, kind string) (db.Scan, error)
	ListAccounts(ctx context.Context) ([]db.ListAccountsRow, error)
	CountAdmins(ctx context.Context) (int64, error)
	UpdateAccountRole(ctx context.Context, arg db.UpdateAccountRoleParams) error
	CreateChannel(ctx context.Context, arg db.CreateChannelParams) (int64, error)
	ListChannels(ctx context.Context) ([]db.ListChannelsRow, error)
	UpdateChannel(ctx context.Context, arg db.UpdateChannelParams) error
	SetChannelSecret(ctx context.Context, arg db.SetChannelSecretParams) error
	DeleteChannel(ctx context.Context, id int64) error
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	UpdateRetentionSettings(ctx context.Context, arg db.UpdateRetentionSettingsParams) error
	// TightestEnabledScanCadenceSeconds is the tightest bound in force, which the
	// observation dial floors at (#208, ADR-0094) — symmetric to the Dispatch
	// floor's SlowestEnabledScanCadenceSeconds.
	TightestEnabledScanCadenceSeconds(ctx context.Context) (int64, error)
	// The Subjects reads route through the live-tier gate (#237, ADR-0041), so each
	// carries the read instant (@as_of) and k (@floor_cadences) in its params: an
	// evidential observation is structurally unreadable here, not merely absent
	// after the Retirer sweeps. The handlers thread s.now() as the read instant.
	ListCurrentNameSubjects(ctx context.Context, arg db.ListCurrentNameSubjectsParams) ([]db.ListCurrentNameSubjectsRow, error)
	GetNameSubject(ctx context.Context, arg db.GetNameSubjectParams) (db.GetNameSubjectRow, error)
	GetNameCitation(ctx context.Context, arg db.GetNameCitationParams) (db.GetNameCitationRow, error)
	FindCoveringNameSeed(ctx context.Context, name string) (db.FindCoveringNameSeedRow, error)
	// FindNameSeedByID reads the terminating Seed of a CT admission's Citation chain
	// by the id the admitted_name row carries, rather than re-deriving it by suffix
	// (ADR-0027, #256).
	FindNameSeedByID(ctx context.Context, seedID int64) (db.FindNameSeedByIDRow, error)
	ListCurrentServiceSubjects(ctx context.Context, arg db.ListCurrentServiceSubjectsParams) ([]db.ListCurrentServiceSubjectsRow, error)
	GetServiceSubject(ctx context.Context, arg db.GetServiceSubjectParams) (db.GetServiceSubjectRow, error)
	// Endpoint subjects (#198): the (Name, Service) pair the http-exchange leaf's
	// http-identity facet is held on. Rendered on the Subjects page additively next
	// to Name and Service, with its own drill-down.
	ListCurrentEndpointSubjects(ctx context.Context, arg db.ListCurrentEndpointSubjectsParams) ([]db.ListCurrentEndpointSubjectsRow, error)
	GetEndpointSubject(ctx context.Context, arg db.GetEndpointSubjectParams) (db.GetEndpointSubjectRow, error)
	FindNameCitingAddress(ctx context.Context, arg db.FindNameCitingAddressParams) (db.FindNameCitingAddressRow, error)
	FindCoveringAddressSeed(ctx context.Context, address netip.Addr) (db.FindCoveringAddressSeedRow, error)
	ListSpansForSubject(ctx context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error)
	// Inventory axis (#243, ADR-0105): every open span across the estate, read
	// straight off the derived span corpus (not live-tier gated). Each open span is
	// the value one timeline currently holds, so this is the estate's "what do I
	// have right now" read, grouped by subject in the handler.
	ListAllOpenSpans(ctx context.Context) ([]db.ListAllOpenSpansRow, error)
	// Exposure landing view (#196): the two most recent reachability spans per
	// (Service, vantage), joined to the prober endpoint. The class is re-verified
	// per render from the presented address, so this read carries the host rather
	// than the static vantage class.
	ListReachabilitySpansForExposure(ctx context.Context) ([]db.ListReachabilitySpansForExposureRow, error)
	CreateZoneFile(ctx context.Context, arg db.CreateZoneFileParams) (db.CreateZoneFileRow, error)
	ListZoneFileStatus(ctx context.Context) ([]db.ListZoneFileStatusRow, error)
	GetZoneCadenceSeconds(ctx context.Context) (int64, error)
	SetZoneCadenceSeconds(ctx context.Context, cadenceSeconds int64) error
	// The cold Scan opt-in (#200): the full-range tier is enabled per-Seed, not
	// globally. Opting a scope in or out reconciles the Scan's enabled flag, which
	// is what puts it on — or takes it off — the dispatcher's cadence.
	ListColdScopeSeedIds(ctx context.Context) ([]int64, error)
	OptInColdScope(ctx context.Context, arg db.OptInColdScopeParams) error
	OptOutColdScope(ctx context.Context, seedID int64) error
	SyncColdScanEnabled(ctx context.Context) error
	CreateProposerLookup(ctx context.Context, arg db.CreateProposerLookupParams) (db.ProposerLookup, error)
	CreateProposal(ctx context.Context, arg db.CreateProposalParams) (db.Proposal, error)
	ListPendingProposals(ctx context.Context) ([]db.ListPendingProposalsRow, error)
	GetPendingProposal(ctx context.Context, id int64) (db.Proposal, error)
	ConfirmProposal(ctx context.Context, arg db.ConfirmProposalParams) (int64, error)
	DeclineLookup(ctx context.Context, lookupID int64) (int64, error)
	ListVergeCoreFrequencyEditsWithAuthor(ctx context.Context) ([]db.ListVergeCoreFrequencyEditsWithAuthorRow, error)
	UpsertVergeCoreFrequencyEdit(ctx context.Context, arg db.UpsertVergeCoreFrequencyEditParams) error
	DeleteVergeCoreFrequencyEdit(ctx context.Context, port int32) error
	// Signals reads (#202): the Derived corpus the Signal engine folds into its
	// per-Name snapshot — resolution per Vantage class, the dns-record CNAME/NS
	// records, and the operator's zone declarations.
	ListNameResolutionsByClass(ctx context.Context, arg db.ListNameResolutionsByClassParams) ([]db.ListNameResolutionsByClassRow, error)
	ListNameDNSRecords(ctx context.Context, arg db.ListNameDNSRecordsParams) ([]db.ListNameDNSRecordsRow, error)
	ListZoneDeclarations(ctx context.Context) ([]db.ListZoneDeclarationsRow, error)
	// The remaining twelve Signal rules (#203): the Service and Endpoint facets the
	// Signal engine folds into its per-subject snapshots — the internet-class
	// reachability leg (sensitive-port-reached-from-internet) and the per-Endpoint
	// certificate value (the six certificate rules and plaintext-http-no-https).
	// http-identity rides ListCurrentEndpointSubjects and the estate name set rides
	// ListCurrentNameSubjects.
	// buildServiceFacts reads the reachability SPAN (not the observation) so a
	// blanket responder's Gap leg reads as absent (ADR-0104); ListBlanketedReachServices
	// is the Coverage register's read of those Gap'd Services (#254).
	ListServiceReachabilitySpansByClass(ctx context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error)
	ListBlanketedReachServices(ctx context.Context) ([]string, error)
	ListEndpointCertificates(ctx context.Context, arg db.ListEndpointCertificatesParams) ([]db.ListEndpointCertificatesRow, error)
	// Annotation management (#204): an operator dial keyed on one
	// (subject, signal-name) pair, carrying the reason and the declared instant —
	// no status, no expiry, no author. Declaring and withdrawing mint no Message.
	CreateAnnotation(ctx context.Context, arg db.CreateAnnotationParams) (db.Annotation, error)
	ListAnnotations(ctx context.Context) ([]db.Annotation, error)
	DeleteAnnotation(ctx context.Context, id int64) error
	// The global message panel (#205): the Message store is unconditional — every
	// message is written and rendered, and the nav element carries the unread
	// count on every screen. There is no delete and no content update; a message
	// is computed once at the cause and read back verbatim.
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
	ListMessages(ctx context.Context) ([]db.Message, error)
	CountUnreadMessages(ctx context.Context) (int64, error)
	ListDeliveryOutcomes(ctx context.Context) ([]db.ListDeliveryOutcomesRow, error)
	MarkMessageRead(ctx context.Context, arg db.MarkMessageReadParams) error
	MarkAllMessagesRead(ctx context.Context, readAt pgtype.Timestamptz) error
	// PreviewExclusionWithdrawal counts the subjects a candidate exclusion would
	// withdraw and the timelines they hold — the honestly-computable narrowing
	// receipt (#205 AC8, ADR-0074). It reads only ground nothing else cites, so a
	// subject a current resolution still holds is not counted (its Gap carries it).
	PreviewExclusionWithdrawal(ctx context.Context, arg db.PreviewExclusionWithdrawalParams) (db.PreviewExclusionWithdrawalRow, error)
	// The Scans monitor (#245): a read over the Operational queue corpus — the
	// recent Dispatches with their per-state job counts, and the per-job detail for
	// one Dispatch. Both are barred from the comparison path by construction
	// (ADR-0041); the drift engine never reads Dispatch, queue_job or batch.
	ListDispatchProgress(ctx context.Context, limit int32) ([]db.ListDispatchProgressRow, error)
	ListJobsForDispatch(ctx context.Context, dispatchID pgtype.Int8) ([]db.ListJobsForDispatchRow, error)
	// The on-demand scan trigger (#252): the trigger panel lists every scan with
	// its enabled state so the disabled cold tier reads as not-triggerable rather
	// than vanishing, and GetScanByKind re-reads the live enabled flag at the
	// instant of a trigger (cold turns enabled once a scope opts in, ADR-0044).
	ListScans(ctx context.Context) ([]db.Scan, error)
}

// server holds everything the handlers need: the database, the session signing
// key (read from the web-only volume, never Postgres), the active single-use
// setup token (empty once an admin exists), and an injectable clock.
type server struct {
	store      store
	key        []byte
	setupToken string
	now        func() time.Time
	sessionTTL time.Duration
	pendingTTL time.Duration
	// seedAddressCap is the ceiling on addresses an address-scope Seed may
	// cover. It defaults to seed.DefaultAddressCap; the Settings screen (#206)
	// will make it operator-configurable.
	seedAddressCap int

	// proposer runs the enabled keyless registry proposer paths for one operator
	// lookup (ADR-0012). It defaults to the shipped registry over a real HTTP
	// client; tests inject a fake so no lookup touches the network.
	proposer proposerRunner

	// dispatcher is the on-demand scan-trigger seam (#252). main.go wires the real
	// queue Dispatcher over the pool; it enqueues the same fan-out the CLI -trigger
	// path uses and pg_notifies the running worker. Tests inject a fake so a trigger
	// asserts the enqueue without a live Postgres or a running worker. It is unset
	// on the read-only pages, which never reach the trigger handler.
	dispatcher scanTrigger

	// secureCookies forces the Secure attribute on auth cookies even when the
	// request did not itself arrive over TLS — set it when web is fronted by a
	// TLS-terminating proxy (VERGE_SECURE_COOKIES).
	secureCookies bool
	// setupMu serialises the first-boot setup so a concurrent pair of valid
	// POST /setup requests cannot both pass the no-accounts check and each
	// create an admin, which would break the token's single-use guarantee.
	setupMu sync.Mutex
}

func newServer(s store, key []byte, setupToken string, now func() time.Time) *server {
	return &server{
		store:          s,
		key:            key,
		setupToken:     setupToken,
		now:            now,
		sessionTTL:     12 * time.Hour,
		pendingTTL:     5 * time.Minute,
		seedAddressCap: seed.DefaultAddressCap,
		proposer:       proposer.DefaultRegistry(&http.Client{Timeout: 30 * time.Second}),
	}
}

// obsAsOf is the read instant every derivation read of the observation corpus is
// gated against (#237, ADR-0041): the same injectable server clock the handlers
// use everywhere, so a fixed-clock test reads its fixtures at the instant it
// seeded them rather than filtering them out against wall-clock now(). Paired with
// retention.FloorCadences (k), it bounds each read to the live tier — an
// evidential row is structurally unreadable, not merely awaiting the Retirer.
func (s *server) obsAsOf() pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: s.now().UTC(), Valid: true}
}

// redirectTo answers a deprecated GET route with a redirect to its canonical
// home (T10, IA reconciliation #286). It is an authedHandler so a deprecated
// route keeps the login gate it carried before the move — an unauthenticated
// hit still lands on /login rather than being bounced onward — and the account
// is otherwise unread. The original query string rides along so a bookmarked
// deep-link (e.g. /seeds?notice=…) survives the move. code is 301 for a pure
// move and 302 for a nuanced one.
func (s *server) redirectTo(target string, code int) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, _ db.Account) {
		dst := target
		if r.URL.RawQuery != "" {
			sep := "?"
			if strings.Contains(target, "?") {
				sep = "&"
			}
			dst += sep + r.URL.RawQuery
		}
		http.Redirect(w, r, dst, code)
	}
}

// handler wires every route. A permission check runs on every mutating
// endpoint from this commit forward (v1 spec §4.3): the only mutation that
// exists yet, POST /accounts, is gated behind requireAdmin.
func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /", s.requireLogin(s.home))

	mux.HandleFunc("GET /setup", s.setupForm)
	mux.HandleFunc("POST /setup", s.setupSubmit)

	mux.HandleFunc("GET /login", s.loginForm)
	mux.HandleFunc("POST /login", s.loginSubmit)
	mux.HandleFunc("POST /login/totp", s.loginTOTP)
	mux.HandleFunc("POST /logout", s.logout)

	// The onboarding wizard (#307, T12): first-run seeds -> cadence -> channel ->
	// review. Stepping is a viewer-safe re-render; completion enqueues the first
	// scan through the existing admin-only trigger (triggerScan), so /onboarding/finish
	// carries the same requireAdmin gate POST /scans/trigger uses.
	mux.HandleFunc("GET /onboarding", s.requireLogin(s.onboarding))
	mux.HandleFunc("POST /onboarding", s.requireLogin(s.onboardingStep))
	mux.HandleFunc("POST /onboarding/finish", s.requireAdmin(s.triggerScan))

	mux.HandleFunc("GET /scope", s.requireLogin(s.seedsPage))
	// /seeds moved to /scope (#286): the scope presentation is the canonical home
	// for scope declaration, so the old GET is a permanent redirect. Every POST
	// action below keeps its /seeds path (only the GET presentation moved) and now
	// answers 303 to /scope.
	mux.HandleFunc("GET /seeds", s.requireLogin(s.redirectTo("/scope", http.StatusMovedPermanently)))
	mux.HandleFunc("POST /seeds", s.requireAdmin(s.declareSeed))
	mux.HandleFunc("POST /seeds/custody", s.requireAdmin(s.setCustody))
	mux.HandleFunc("POST /seeds/zone", s.requireAdmin(s.uploadZoneFile))
	mux.HandleFunc("POST /seeds/zone/interval", s.requireAdmin(s.setZoneInterval))
	mux.HandleFunc("POST /seeds/cold", s.requireAdmin(s.setColdScope))
	mux.HandleFunc("POST /exclusions", s.requireAdmin(s.declareExclusion))
	mux.HandleFunc("POST /exclusions/preview", s.requireAdmin(s.previewExclusion))
	mux.HandleFunc("POST /exclusions/delete", s.requireAdmin(s.unexclude))
	mux.HandleFunc("POST /probers", s.requireAdmin(s.provisionProber))

	// The Exposure page (#300, T5, ADR-0110): `/exposure` is repurposed from the
	// #286 redirect-to-/reports into the first-class Exposure screen — the both-legs
	// table plus the WITHHELD state that names its cause when no internet vantage
	// exists. Reports still folds the period analytics; this is the dedicated
	// exposure board. Viewer-readable: a viewer reads the board, mutates nothing.
	mux.HandleFunc("GET /exposure", s.requireLogin(s.exposurePage))
	mux.HandleFunc("GET /reports", s.requireLogin(s.reportsPage))
	// The delivered-report artifact (#298, T3): the stable view of an already-
	// delivered report, reached from Reports' "view last delivery" (T17 links here,
	// so the route stays fixed). A viewer reads it — a delivered report is a record,
	// not a mutation — and its document body is the canonical render that doubles as
	// the PDF/email spec (internal/message.RenderArtifact).
	mux.HandleFunc("GET /reports/delivery", s.requireLogin(s.reportDeliveryPage))

	// The Subjects LIST folded into /inventory (#286): Inventory is the canonical
	// "what do I have right now" screen and lists every Name/Service/Endpoint, so
	// the old list GET is a permanent redirect. The detail drill-downs below are
	// NOT redirected — Inventory rows deep-link straight to them and they are the
	// subject detail pages, so they keep resolving. The Service and Endpoint
	// drill-downs read their key from a query parameter because those keys carry
	// `/` and `@`; their literal paths win over the {key} wildcard.
	mux.HandleFunc("GET /subjects", s.requireLogin(s.redirectTo("/inventory", http.StatusMovedPermanently)))
	mux.HandleFunc("GET /subjects/service", s.requireLogin(s.servicePage))
	mux.HandleFunc("GET /subjects/endpoint", s.requireLogin(s.endpointPage))
	mux.HandleFunc("GET /subjects/{key}", s.requireLogin(s.subjectPage))

	mux.HandleFunc("GET /inventory", s.requireLogin(s.inventoryPage))
	// The per-asset drill-in (#296, T1): the destination of an Inventory row-click.
	// The route is stable so T15's Inventory can link straight to it. A Name key
	// carries neither `/` nor `@`, so it rides a plain path segment.
	mux.HandleFunc("GET /asset/{key}", s.requireLogin(s.assetPage))
	mux.HandleFunc("GET /drift", s.requireLogin(s.driftPage))
	// The per-run drill-in (#297, T2): the destination of a Drift "Batch detail"
	// entry. A run is one Dispatch (a fan-out of one Scan); the route is stable so
	// T16's Drift can link straight to it. A viewer reads it, like the Scans monitor
	// — it is a read-only window onto the Operational queue corpus. A Dispatch id is
	// an integer carrying neither `/` nor `@`, so it rides a plain path segment.
	mux.HandleFunc("GET /run/{id}", s.requireLogin(s.runPage))

	mux.HandleFunc("GET /signals", s.requireLogin(s.signalsPage))
	mux.HandleFunc("POST /annotations", s.requireAdmin(s.declareAnnotation))
	mux.HandleFunc("POST /annotations/withdraw", s.requireAdmin(s.withdrawAnnotation))

	mux.HandleFunc("GET /graph", s.requireLogin(s.graphPage))

	// Search results (#303, T8, ADR-0110): the full-page search the shell's command
	// palette ("see everything") lands on. It reads only data a viewer already reads
	// elsewhere — current Name subjects, the fired signal corpus, recent Dispatches —
	// and every row links to that item's existing route (/asset/{key}, /signals,
	// /run/{id}). Viewer-readable, requireLogin: a search reads, it mutates nothing.
	mux.HandleFunc("GET /search", s.requireLogin(s.searchPage))

	// Registry proposer lookups and the confirm/decline of the Proposals they
	// yield are admin acts (v1 spec §4.3): confirming opens the probing gate on
	// address space, declining is a boundary claim. A viewer reads the pending
	// list on /scope but mutates nothing.
	mux.HandleFunc("POST /proposals", s.requireAdmin(s.runLookup))
	mux.HandleFunc("POST /proposals/confirm", s.requireAdmin(s.confirmProposal))
	mux.HandleFunc("POST /proposals/decline", s.requireAdmin(s.declineLookup))

	// /coverage KEPT RESOLVING (#286 judgement call): the full aperture statement —
	// one line per aperture input, its cadence, and whether it is on — is a distinct
	// viewer-readable artifact reached from Scope's "Aperture statement" / "Go to
	// Coverage" buttons. Scope surfaces coverage *messages* (gaps), not the aperture
	// statement itself, so redirecting /coverage → /scope would lose the artifact and
	// turn those Scope buttons into self-links. Viewer-readable; no downgrade.
	mux.HandleFunc("GET /coverage", s.requireLogin(s.coveragePage))

	// The Scans monitor (#245, v1 spec §4.1): a read-only window onto the queue.
	// A viewer reads it — it surfaces in-flight Dispatches and their job state.
	// The on-demand trigger (#252) is the one mutation on this page: dispatching a
	// scan enqueues a fan-out, so it is an admin act gated behind requireAdmin,
	// exactly as /sources toggling is. A viewer sees no trigger control.
	//
	// /scans KEPT RESOLVING (#286, #281 viewer-access caveat): this is the
	// viewer-readable live monitor (in-flight Dispatches + per-job progress).
	// Reports holds only the scans-per-day heatmap, not the live monitor, and the
	// full monitor otherwise lives behind admin /settings?tab=scans — so redirecting
	// /scans → /reports would silently downgrade a viewer's access to the live
	// monitor, which #281 forbids. It stays a viewer route; the admin trigger is the
	// {{if .IsAdmin}} panel here and mirrored at /settings?tab=scans.
	mux.HandleFunc("GET /scans", s.requireLogin(s.scansPage))
	mux.HandleFunc("POST /scans/trigger", s.requireAdmin(s.triggerScan))

	// The global message panel (#205, v1 spec §6.7): a viewer reads the unbounded
	// list and its unread count on every screen; marking read is a per-account
	// read-state change, so a viewer may do it. The store is unconditional and has
	// no admin surface — there is nothing here to gate behind requireAdmin.
	//
	// /messages KEPT RESOLVING (#286, #281 caveat; #295 V3 shell): the messages fold
	// stays viewer-readable for ALL users. The V3 shell bell now targets /inbox (the
	// T4 Inbox page), but /messages remains the viewer-readable messages fold — reached
	// directly and mirrored in the Settings messages tab — and is NOT redirected into
	// admin-gated /settings. It keeps resolving until T4's Inbox subsumes it.
	mux.HandleFunc("GET /messages", s.requireLogin(s.messagesPage))
	mux.HandleFunc("POST /messages/read", s.requireLogin(s.markMessageRead))
	mux.HandleFunc("POST /messages/read-all", s.requireLogin(s.markAllMessagesRead))

	// The Inbox (#299, T4, ADR-0110): the V3 primary message surface the shell bell
	// deep-links to (the /messages fold stays as the viewer-readable mirror). It reads
	// the same unconditional Message store as /messages, and its read/unread and
	// mark-all-read acts route through the shared POST /messages/read[-all] handlers
	// above (each inbox form carries a `return=/inbox` field). A viewer reads it and
	// may mark read — no admin surface, so nothing here is gated behind requireAdmin.
	mux.HandleFunc("GET /inbox", s.requireLogin(s.inboxPage))

	// verge-core: a viewer reads the composed set; editing the frequency half is
	// an admin act (v1 spec §3.5, §4.3). The sensitive half is authored by the
	// release and has no mutating endpoint at all.
	//
	// /verge-core KEPT RESOLVING (#286, #281 caveat): the composed sensitive-port
	// set is viewer-readable (requireLogin); only frequency editing is admin, and
	// that mirror lives at /settings?tab=integrations. Redirecting /verge-core into
	// admin Settings would 403 viewers, downgrading their read — so it stays a
	// viewer route.
	mux.HandleFunc("GET /verge-core", s.requireLogin(s.vergeCorePage))
	mux.HandleFunc("POST /verge-core/frequency", s.requireAdmin(s.editVergeCoreFrequency))

	// /sources KEPT RESOLVING (#286, #281 caveat): the source-enablement modal is
	// viewer-readable (requireLogin), reached from /coverage's "Manage source
	// enablement" entry point; only toggling is admin. Redirecting it into admin
	// Settings would 403 viewers, so it stays a viewer route.
	mux.HandleFunc("GET /sources", s.requireLogin(s.sourcesModal))
	mux.HandleFunc("POST /sources/toggle", s.requireAdmin(s.toggleSource))

	// The account surface folded into Settings → access (#281): GET /account now
	// redirects there (accountPage), so the merged SignIn's totp-enroll Cancel link
	// still lands somewhere real. The POST endpoints keep their paths and render the
	// Settings access sub-tab.
	mux.HandleFunc("GET /account", s.requireLogin(s.accountPage))
	mux.HandleFunc("POST /accounts", s.requireAdmin(s.createAccount))
	mux.HandleFunc("POST /account/totp/enable", s.requireLogin(s.totpEnable))
	mux.HandleFunc("POST /account/totp/confirm", s.requireLogin(s.totpConfirm))

	// The Settings destination and every mutation it hosts are admin acts
	// (v1 spec §4.3, §6.1): viewing the operator's dials and moving them are
	// both gated behind requireAdmin.
	mux.HandleFunc("GET /settings", s.requireAdmin(s.settingsPage))
	mux.HandleFunc("POST /settings/accounts", s.requireAdmin(s.inviteAccount))
	mux.HandleFunc("POST /settings/accounts/role", s.requireAdmin(s.setAccountRole))
	mux.HandleFunc("POST /settings/channels", s.requireAdmin(s.createChannel))
	mux.HandleFunc("POST /settings/channels/update", s.requireAdmin(s.updateChannel))
	mux.HandleFunc("POST /settings/channels/delete", s.requireAdmin(s.deleteChannel))
	mux.HandleFunc("POST /settings/retention", s.requireAdmin(s.updateRetention))
	return mux
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	hb, err := s.store.RecordHeartbeat(r.Context())
	if err != nil {
		log.Printf("web: healthz: record heartbeat: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Status    string    `json:"status"`
		CheckedAt time.Time `json:"checked_at"`
	}{Status: "ok", CheckedAt: hb.CheckedAt.Time})
}
