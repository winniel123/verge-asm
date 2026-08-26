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
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
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
	// SetTOTPLastStep atomically advances an account's TOTP replay watermark to the
	// step just accepted at login (#323, #339). It updates only when the stored step is
	// still below the presented one and reports the rows affected, so of two concurrent
	// requests carrying the same code exactly one spends it — the single-use discipline
	// RFC 6238 §5.2 requires, without a read-then-write race.
	SetTOTPLastStep(ctx context.Context, arg db.SetTOTPLastStepParams) (int64, error)
	// Personal profile (#304, T9): an account's own credential and token surface.
	// UpdatePassword is the self-service password change; the personal-token trio is
	// the reveal-once API-token store — the plaintext is minted and shown once, only
	// its hash is kept, and a revoke is a hard delete scoped to the owner.
	UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error
	CreatePersonalToken(ctx context.Context, arg db.CreatePersonalTokenParams) (db.PersonalToken, error)
	ListPersonalTokens(ctx context.Context, accountID int64) ([]db.ListPersonalTokensRow, error)
	DeletePersonalToken(ctx context.Context, arg db.DeletePersonalTokenParams) error
	// SignIn delta (#314, T19): the pre-auth token stores behind forgot/reset,
	// TOTP recovery codes, and invite acceptance. Each keeps only a hash of its
	// secret — the plaintext is shown or handed out once and never persisted — and
	// each is single-use, spent by stamping the consumed/used instant the handler
	// threads from the server clock. Recovery codes are re-issued wholesale (delete
	// then insert) so a fresh enrollment never leaves a stale set redeemable. The
	// invite CREATION side lands in T18 (Settings -> Team); these are the reads and
	// spends the acceptance screen needs.
	CreatePasswordReset(ctx context.Context, arg db.CreatePasswordResetParams) (db.PasswordReset, error)
	GetPasswordResetByHash(ctx context.Context, tokenHash string) (db.PasswordReset, error)
	ConsumePasswordReset(ctx context.Context, arg db.ConsumePasswordResetParams) error
	CreateRecoveryCode(ctx context.Context, arg db.CreateRecoveryCodeParams) error
	DeleteRecoveryCodesForAccount(ctx context.Context, accountID int64) error
	ListUnusedRecoveryCodeHashes(ctx context.Context, accountID int64) ([]db.ListUnusedRecoveryCodeHashesRow, error)
	ConsumeRecoveryCode(ctx context.Context, arg db.ConsumeRecoveryCodeParams) error
	GetInviteByTokenHash(ctx context.Context, tokenHash string) (db.Invite, error)
	ConsumeInvite(ctx context.Context, arg db.ConsumeInviteParams) error
	// Server-side session registry (#405, ADR-0117): every login opens a session row
	// keyed by the hash of an opaque token the signed cookie carries, and every request
	// re-validates that row so a revocation takes effect on the next request rather than
	// at cookie expiry. CreateSession opens it; GetSessionByTokenHash is the per-request
	// validation lookup (live = unrevoked and unexpired); TouchSession refreshes
	// last_seen_at (throttled by the handler); RevokeSession ends one session scoped to
	// its owner (sign-out and the end-session action). The listing and bulk-revoke
	// queries back the personal Profile surface (#406), the admin Settings surface
	// (#407), and the credential-flow invalidation (#408): ListSessionsForAccount is an
	// account's own live sessions; RevokeOtherSessionsForAccount is "sign out other
	// devices" and the password-change invalidation (keeps the current session);
	// RevokeAllSessionsForAccount is the reset path and admin offboarding (no exception);
	// ListAllActiveSessions is every account's live sessions joined to username/role for
	// the admin view; RevokeSessionByIDForAdmin revokes any one session, admin-gated.
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error)
	GetSessionByTokenHash(ctx context.Context, arg db.GetSessionByTokenHashParams) (db.Session, error)
	TouchSession(ctx context.Context, arg db.TouchSessionParams) error
	RevokeSession(ctx context.Context, arg db.RevokeSessionParams) error
	ListSessionsForAccount(ctx context.Context, arg db.ListSessionsForAccountParams) ([]db.ListSessionsForAccountRow, error)
	RevokeOtherSessionsForAccount(ctx context.Context, arg db.RevokeOtherSessionsForAccountParams) error
	RevokeAllSessionsForAccount(ctx context.Context, arg db.RevokeAllSessionsForAccountParams) error
	ListAllActiveSessions(ctx context.Context, expiresAt pgtype.Timestamptz) ([]db.ListAllActiveSessionsRow, error)
	RevokeSessionByIDForAdmin(ctx context.Context, arg db.RevokeSessionByIDForAdminParams) error
	// CreateInvite is the invite CREATION side (Settings -> Team, T18): it mints a
	// single-use, time-boxed invite at a role against the same invite table T19's
	// acceptance screen spends. ResetAccountTOTP is Team's "require re-enrollment" —
	// it disarms an account's second factor so the next sign-in re-enrols it.
	CreateInvite(ctx context.Context, arg db.CreateInviteParams) (db.Invite, error)
	ResetAccountTOTP(ctx context.Context, id int64) error
	// DeleteAccount removes a member (Settings -> Team). It is gated behind a typed-
	// name confirmation and the not-self / last-admin guards; an account that authored
	// attributed acts (a NOT NULL created_by reference) is refused by the FK rather
	// than orphaning its work.
	DeleteAccount(ctx context.Context, id int64) error
	CreateNameSeed(ctx context.Context, arg db.CreateNameSeedParams) (db.Seed, error)
	CreateAddressSeed(ctx context.Context, arg db.CreateAddressSeedParams) (db.Seed, error)
	ListSeeds(ctx context.Context) ([]db.ListSeedsRow, error)
	// DeleteSeed withdraws a declared Seed by id — the Scope chip-remove act (#21a).
	DeleteSeed(ctx context.Context, id int64) (int64, error)
	SetCustodyExtension(ctx context.Context, arg db.SetCustodyExtensionParams) error
	CreateNameExclusion(ctx context.Context, arg db.CreateNameExclusionParams) (db.Exclusion, error)
	CreateAddressExclusion(ctx context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error)
	ListExclusions(ctx context.Context) ([]db.ListExclusionsRow, error)
	DeleteExclusion(ctx context.Context, id int64) error
	ListSourceStates(ctx context.Context) ([]db.SourceState, error)
	UpsertSourceState(ctx context.Context, arg db.UpsertSourceStateParams) (db.SourceState, error)
	ListIntegrationStates(ctx context.Context) ([]db.IntegrationState, error)
	UpsertIntegrationState(ctx context.Context, arg db.UpsertIntegrationStateParams) (db.IntegrationState, error)
	DeleteIntegrationState(ctx context.Context, slug string) error
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
	// GetInstanceHealth reads the instance-health tab's live database facts (#633): the
	// database size and Postgres server version, off the running server. Operational only.
	GetInstanceHealth(ctx context.Context) (db.GetInstanceHealthRow, error)
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
	// The estate-wide, batch-grouped drift feed (#288, ADR-0111): every span
	// open/close event a Batch caused within the period, joined to that Batch for the
	// group meta, so the handler classifies each into one of the six change kinds on
	// read (ADR-0007) and groups the transitions by batch. Reads span and batch only —
	// never dispatch — honoring the comparison-path separation (ADR-0041).
	ListRecentDriftEvents(ctx context.Context, arg db.ListRecentDriftEventsParams) ([]db.ListRecentDriftEventsRow, error)
	// Mean-time-to-withdrawal trend (#444, P0.3): every subject withdrawal since a
	// read instant, paired with the subject's first appearance, so the Reports trend
	// derives time-to-withdrawal (withdrawn_at − first_opened) per departure. A
	// withdrawal closes every open timeline a subject held at one instant (ADR-0082),
	// so the per-facet closures collapse to one departure per (subject, closed_at).
	// Reads FROM span only — the never-compacted derived corpus (ADR-0041) — not the
	// live-tier observation gate.
	ListWithdrawalLifespans(ctx context.Context, since pgtype.Timestamptz) ([]db.ListWithdrawalLifespansRow, error)
	// New assets discovered (#468, P2.4b): every Name/Service subject whose FIRST
	// appearance (MIN(opened_at), the `appeared` classification) is at or after a
	// read instant, so the Reports "New assets discovered" card folds a per-period
	// count, its vs-previous-period delta, and the daily-discovery bar series. Reads
	// FROM span only — the never-compacted derived corpus (ADR-0041).
	ListSubjectFirstAppearances(ctx context.Context, since pgtype.Timestamptz) ([]db.ListSubjectFirstAppearancesRow, error)
	// Exposure landing view (#196): the two most recent reachability spans per
	// (Service, vantage), joined to the prober endpoint. The class is re-verified
	// per render from the presented address, so this read carries the host rather
	// than the static vantage class.
	ListReachabilitySpansForExposure(ctx context.Context) ([]db.ListReachabilitySpansForExposureRow, error)
	// Vs-last-batch stat deltas (#443, P0.2, ADR-0116): the reads the Dashboard and
	// Exposure stat tiles derive their signed deltas from. PreviousBatchTime is the
	// boundary the "value a batch ago" is reconstructed at (NULL where no previous
	// batch exists); ListSpansOpenSince returns the span corpus still open now or
	// closed after that boundary, so the population open at it is reconstructable on
	// read (internal/drift.OpenAt); ListServiceReachabilitySpansByClassAt is the
	// as-of-@at twin of the current by-class read, for the exposure projection's
	// previous snapshot. All read the derived span/batch corpora, never dispatch (ADR-0041).
	PreviousBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	ListSpansOpenSince(ctx context.Context, since pgtype.Timestamptz) ([]db.ListSpansOpenSinceRow, error)
	ListServiceReachabilitySpansByClassAt(ctx context.Context, at pgtype.Timestamptz) ([]db.ListServiceReachabilitySpansByClassAtRow, error)
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
	// DeclineProposal declines one still-pending Proposal by id — the Scope
	// decline-many act declines each checked proposal (#574).
	DeclineProposal(ctx context.Context, id int64) (int64, error)
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
	// Per-instance signal identity (#442, P0.1): the mintable `SIG-####` id and
	// first-seen instant of each currently-fired (rule, subject) pair. MintSignalInstances
	// is the idempotent upsert on the Signals read path (a firing pair keeps its id and
	// first-seen); ListSignalInstances reads the identities back so the web layer attaches
	// each fired census member its stable id + first-seen. Severity and last-seen are
	// derived, not stored.
	MintSignalInstances(ctx context.Context, arg db.MintSignalInstancesParams) error
	ListSignalInstances(ctx context.Context) ([]db.SignalInstance, error)
	// The global message panel (#205): the Message store is unconditional — every
	// message is written and rendered, and the nav element carries the unread
	// count on every screen. There is no delete and no content update; a message
	// is computed once at the cause and read back verbatim.
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
	ListMessages(ctx context.Context) ([]db.Message, error)
	ListReadMessageIDs(ctx context.Context, accountID int64) ([]int64, error)
	CountUnreadMessages(ctx context.Context, accountID int64) (int64, error)
	ListDeliveryOutcomes(ctx context.Context) ([]db.ListDeliveryOutcomesRow, error)
	MarkMessageRead(ctx context.Context, arg db.MarkMessageReadParams) error
	MarkAllMessagesRead(ctx context.Context, arg db.MarkAllMessagesReadParams) error
	// MarkMessageUnread returns one message to unread for the caller (#473,
	// ADR-0116) by clearing this account's read-mark — the inverse of
	// MarkMessageRead, backing the Inbox "Mark unread" affordance (Inbox.jsx:59).
	MarkMessageUnread(ctx context.Context, arg db.MarkMessageUnreadParams) error
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
	// Stop-dispatch / terminate (DF-F4, #633): the admin acts that end a Dispatch in
	// flight. CancelReadyJobsForDispatch cancels the pending (ready) jobs of a stop and
	// returns the count; CancelActiveJobsForDispatch cancels ready AND running jobs for a
	// terminate; SetDispatchStatus records the operator-ended disposition ('stopped' /
	// 'terminated'), scoped to a still-'fanned-out' dispatch so it never overwrites a
	// recorded terminal status. All three read/write only the Operational queue corpus
	// (ADR-0041); the drift engine never sees them.
	CancelReadyJobsForDispatch(ctx context.Context, dispatchID pgtype.Int8) (int64, error)
	CancelActiveJobsForDispatch(ctx context.Context, dispatchID pgtype.Int8) (int64, error)
	SetDispatchStatus(ctx context.Context, arg db.SetDispatchStatusParams) error
	// The on-demand scan trigger (#252): the trigger panel lists every scan with
	// its enabled state so the disabled cold tier reads as not-triggerable rather
	// than vanishing, and GetScanByKind re-reads the live enabled flag at the
	// instant of a trigger (cold turns enabled once a scope opts in, ADR-0044).
	ListScans(ctx context.Context) ([]db.Scan, error)

	// Report scheduling (#290, live CRUD in P0.6/T4): the Reports screen's "New
	// schedule" wizard declares a recurring report, the "Recurring reports" table lists
	// them, and the row menu edits, deletes, and runs one now. A schedule is Declared —
	// Insert files one, List reads them newest-first, Get reads one (Edit prefill and
	// the Run-now dispatch), Update edits a declared schedule's contents in place (a
	// schedule carries no derived state, so this is not a recompute — migration 21700),
	// and Delete removes one. "Last delivery" resolves from the report_delivery receipts
	// store (#291/T2), not a stamp on the schedule itself.
	InsertReportSchedule(ctx context.Context, arg db.InsertReportScheduleParams) (db.ReportSchedule, error)
	ListReportSchedules(ctx context.Context) ([]db.ReportSchedule, error)
	GetReportSchedule(ctx context.Context, id int64) (db.ReportSchedule, error)
	UpdateReportSchedule(ctx context.Context, arg db.UpdateReportScheduleParams) (db.ReportSchedule, error)
	DeleteReportSchedule(ctx context.Context, id int64) error

	// Report-delivery receipts (#291/T2): the operational record of each run of a
	// schedule. GetLatestReportDelivery returns the newest non-failed run, backing the
	// "Recurring reports" table's "last sent" cell and the delivered-artifact view;
	// pgx.ErrNoRows is the genuine empty-state (never run, or only failed). The insert
	// and list paths round out the store the run path (T5) writes and reads.
	InsertReportDelivery(ctx context.Context, arg db.InsertReportDeliveryParams) (db.ReportDelivery, error)
	NextReportDeliveryNo(ctx context.Context, scheduleID int64) (int32, error)
	GetLatestReportDelivery(ctx context.Context, scheduleID int64) (db.ReportDelivery, error)
	ListReportDeliveries(ctx context.Context, scheduleID int64) ([]db.ReportDelivery, error)

	// SSO / OIDC providers (#293, ADR-0112): the config behind the SignIn buttons and
	// the Settings single-sign-on tab. The client secret is write-only — only
	// GetSSOProviderForAuth (the server-side token exchange) selects it; every other
	// read exposes has_secret alone, mirroring the channel secret (ADR-0053).
	InsertSSOProvider(ctx context.Context, arg db.InsertSSOProviderParams) (int64, error)
	ListSSOProviders(ctx context.Context) ([]db.ListSSOProvidersRow, error)
	ListEnabledSSOProviders(ctx context.Context) ([]db.ListEnabledSSOProvidersRow, error)
	GetSSOProvider(ctx context.Context, id int64) (db.GetSSOProviderRow, error)
	GetSSOProviderForAuth(ctx context.Context, slug string) (db.GetSSOProviderForAuthRow, error)
	UpdateSSOProvider(ctx context.Context, arg db.UpdateSSOProviderParams) (int64, error)
	SetSSOProviderSecret(ctx context.Context, arg db.SetSSOProviderSecretParams) error
	DeleteSSOProvider(ctx context.Context, id int64) error

	// SSO identity bindings (#319, ADR-0113): authentication keys on a stored
	// (provider, sub) → account binding, established by an authenticated Profile
	// self-link. The login match is GetAccountBySSOIdentity; GetSSOIdentityBySub gates
	// the link against an existing binding; the rest drive the Profile and admin lists.
	InsertSSOIdentity(ctx context.Context, arg db.InsertSSOIdentityParams) error
	GetAccountBySSOIdentity(ctx context.Context, arg db.GetAccountBySSOIdentityParams) (db.Account, error)
	GetSSOIdentityBySub(ctx context.Context, arg db.GetSSOIdentityBySubParams) (db.GetSSOIdentityBySubRow, error)
	ListSSOIdentitiesForAccount(ctx context.Context, accountID int64) ([]db.ListSSOIdentitiesForAccountRow, error)
	DeleteSSOIdentityForAccount(ctx context.Context, arg db.DeleteSSOIdentityForAccountParams) (int64, error)
	ListSSOBindings(ctx context.Context) ([]db.ListSSOBindingsRow, error)
	DeleteSSOIdentity(ctx context.Context, id int64) error
}

// server holds everything the handlers need: the database, the session signing
// key (read from the web-only volume, never Postgres), the active single-use
// setup token (empty once an admin exists), and an injectable clock.
type server struct {
	store store
	key   []byte
	// totpKey is the AEAD sub-key that encrypts account.totp_secret at rest (#337).
	// It is HKDF-derived from the file-backed session key with a domain-separation
	// label, so a database dump discloses no usable TOTP secret and the raw signing
	// key is never reused for two purposes (ADR-0053). Derived once in newServer.
	totpKey    []byte
	setupToken string
	now        func() time.Time
	// startedAt is the instant this process came up, read off the injectable clock
	// so the instance-health tab (#313) shows a real uptime rather than a fabricated
	// one. A fixed-clock test reads ~0, which humanizes honestly.
	startedAt  time.Time
	sessionTTL time.Duration
	pendingTTL time.Duration
	// resetTTL bounds a password-reset link's life (SignIn delta #314). A link
	// older than this is refused at /reset rather than setting a password, so a
	// leaked-then-stale link is inert. Kept short by default.
	resetTTL time.Duration
	// seedAddressCap is the ceiling on addresses an address-scope Seed may
	// cover. It defaults to seed.DefaultAddressCap; the Settings screen (#206)
	// will make it operator-configurable.
	seedAddressCap int

	// proposer runs the enabled keyless registry proposer paths for one operator
	// lookup (ADR-0012). It defaults to the shipped registry over a real HTTP
	// client; tests inject a fake so no lookup touches the network.
	proposer proposerRunner

	// sso is the OIDC single-sign-on seam (#293, ADR-0112). main.go wires the real
	// go-oidc/oauth2 flow over an HTTP client; tests inject a fake so a login flow
	// asserts its state/nonce/cookie handling and account mapping without a live
	// identity provider. newServer defaults it to the real flow.
	sso ssoFlow

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
	// externalURL is the trusted origin the deployment is reached at
	// (VERGE_EXTERNAL_URL, e.g. https://verge.example.com). When set it is the base
	// for the OIDC callback redirect_uri (#293), so that value never derives from the
	// attacker-influenceable Host header; empty falls back to the request host.
	externalURL string
	// setupMu serialises the first-boot setup so a concurrent pair of valid
	// POST /setup requests cannot both pass the no-accounts check and each
	// create an admin, which would break the token's single-use guarantee.
	setupMu sync.Mutex

	// flash is the in-process single-consume toast store (flash.go). The scan trigger
	// and stop/terminate acts stash one toast here and redirect to a clean URL, so the
	// in-flight auto-refresh does not re-show it (WORK-ORDER-DOGFOOD-R1 item 1); injectChrome
	// consumes it on the first chrome render. Best-effort, per-process.
	flash *flashStore

	// loginLimiter throttles failed credential attempts on /login and /login/totp
	// (#322): per-account and per-IP failed-attempt tracking with a temporary,
	// exponential lockout, so a 6-digit TOTP is no longer brute-forceable and an
	// online password guess has a bounded budget. It is in-process and clock-driven
	// (no DB, no new dependency), reset on a successful auth.
	loginLimiter *loginLimiter

	// pool is the raw pgx pool, wired ONLY for the VERGE_DEV pixel-parity harness
	// affordance that re-seeds the Profile fixture between capture states (#542) via
	// the /dev/profile/session route. A real deployment leaves it nil — that route is
	// registered only in devMode and guards on a nil pool — so no non-dev path touches it.
	pool *pgxpool.Pool

	// devMode is a VERGE_DEV build: it unlocks the dev-only pixel-parity affordances
	// (main.go sets it from VERGE_DEV). It gates the /dev/* harness routes (devfixtures.go)
	// and makes the 500 incident id deterministic — never set in a real deployment, so no
	// dev route is reachable and the incident id keeps its crypto/rand draw (errors.go).
	devMode bool

	// coverageMu / coverageEmptyOnce back the Coverage screen's empty-state capture
	// (#552). states.json coverage declares an "empty" state seeded "empty-authed":
	// GET /dev/seed/empty-authed (devMode only) sets coverageEmptyOnce so the NEXT
	// /coverage render serves the empty estate while the authed admin session is kept
	// (unlike /dev/seed/empty, which truncates accounts). coveragePage consumes the flag
	// (reads-and-clears) as it renders, so a later context's "default" state — which
	// applies no seed — renders the full fixture again. In-process, devMode-only.
	coverageMu        sync.Mutex
	coverageEmptyOnce bool
}

func newServer(s store, key []byte, setupToken string, now func() time.Time) *server {
	// Derive the TOTP-secret AEAD sub-key from the session key (#337). HKDF over a
	// present key of a sane length does not fail; a nil result would only fail-closed
	// on the encrypt path, never silently store cleartext.
	totpKey, _ := auth.DeriveTOTPKey(key)
	return &server{
		store:          s,
		key:            key,
		totpKey:        totpKey,
		setupToken:     setupToken,
		now:            now,
		startedAt:      now(),
		sessionTTL:     12 * time.Hour,
		pendingTTL:     5 * time.Minute,
		resetTTL:       30 * time.Minute,
		seedAddressCap: seed.DefaultAddressCap,
		proposer:       proposer.DefaultRegistry(&http.Client{Timeout: 30 * time.Second}),
		sso:            newOIDCFlow(&http.Client{Timeout: 30 * time.Second}),
		loginLimiter:   newLoginLimiter(now),
		flash:          newFlashStore(),
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
	// /signout is the design-owned shell's account-menu form action (shell.tmpl #27b):
	// a mechanical alias onto the existing /logout handler (a route rename, not a
	// redesign), so the frozen tmpl's sign-out posts to the same session-revoking path.
	mux.HandleFunc("POST /signout", s.logout)

	// Single sign-on (#293, ADR-0112): the OIDC authorization-code flow. Both hops are
	// unauthenticated by construction — a caller signing in has no session — so they
	// sit beside /login, not behind requireLogin. The state/nonce/PKCE ride a signed
	// short-lived cookie between them. Identity comes only from the verified id_token,
	// never a proxy header (§4.3, §7).
	mux.HandleFunc("GET /login/sso/{slug}", s.ssoStart)
	mux.HandleFunc("GET /login/sso/{slug}/callback", s.ssoCallback)

	// SignIn delta (#314, T19): the pre-auth self-service surfaces ported from
	// SignIn.jsx — forgot/reset password and invite acceptance. Like /login and
	// /setup these are chrome-less and unauthenticated by construction: a caller
	// who has lost their password or holds only an invite token has no session to
	// gate on. Each proves possession of a single-use, hashed, time-boxed token
	// (delivered out of band; on a host with no mail the reset link is written to
	// the web logs, as the setup token is). No account is enumerable: /forgot
	// answers the same way whether or not the username exists.
	mux.HandleFunc("GET /forgot", s.forgotForm)
	mux.HandleFunc("POST /forgot", s.forgotSubmit)
	mux.HandleFunc("GET /reset", s.resetForm)
	mux.HandleFunc("POST /reset", s.resetSubmit)
	mux.HandleFunc("GET /invite", s.inviteForm)
	mux.HandleFunc("POST /invite", s.inviteAccept)

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
	// The chip-remove act (#21a): scope.tmpl posts a seed's id to withdraw it.
	mux.HandleFunc("POST /seeds/delete", s.requireAdmin(s.deleteSeed))
	mux.HandleFunc("POST /seeds/custody", s.requireAdmin(s.setCustody))
	mux.HandleFunc("POST /seeds/zone", s.requireAdmin(s.uploadZoneFile))
	mux.HandleFunc("POST /seeds/zone/interval", s.requireAdmin(s.setZoneInterval))
	mux.HandleFunc("POST /exclusions", s.requireAdmin(s.declareExclusion))
	mux.HandleFunc("POST /exclusions/preview", s.requireAdmin(s.previewExclusion))
	mux.HandleFunc("POST /exclusions/delete", s.requireAdmin(s.unexclude))
	// #21d: the cold-tier opt-in and prober provisioning REGIONS + ROUTES relocate to
	// /settings (their design homes, shots 17/18). scope.tmpl no longer renders them; the
	// acts now redirect to /settings, where the read surfaces live. Settings' own design
	// parity lands at map #21 — batch 3 owns only the move-out.
	mux.HandleFunc("POST /settings/cold", s.requireAdmin(s.setColdScope))
	mux.HandleFunc("POST /settings/probers", s.requireAdmin(s.provisionProber))

	// The Exposure page (#300, T5, ADR-0110): `/exposure` is repurposed from the
	// #286 redirect-to-/reports into the first-class Exposure screen — the both-legs
	// table plus the WITHHELD state that names its cause when no internet vantage
	// exists. Reports still folds the period analytics; this is the dedicated
	// exposure board. Viewer-readable: a viewer reads the board, mutates nothing.
	mux.HandleFunc("GET /exposure", s.requireLogin(s.exposurePage))
	// The Exposure WITHHELD state's action links /settings/vantages (SPEC-CHANGE #20f,
	// ruled: provisioning a prober is a vantage act, not /scope). Prober provisioning now
	// lives under Settings → Vantages (#21d, batch 3), so this alias lands on that tab
	// rather than /scope. Viewer-readable, matching the exposure board it is reached from.
	mux.HandleFunc("GET /settings/vantages", s.requireLogin(s.redirectTo("/settings?tab=vantages", http.StatusSeeOther)))
	mux.HandleFunc("GET /reports", s.requireLogin(s.reportsPage))
	// The Reports export (#291): the KPI band + scans-per-day series for the active
	// ?weeks= range as a downloadable csv/json file. A viewer reads it — an export is
	// a read of the same operational figures the page shows, never a mutation.
	mux.HandleFunc("GET /reports/export", s.requireLogin(s.reportsExport))
	// The delivered-report artifact (#298, T3): the stable view of an already-
	// delivered report, reached from Reports' "view last delivery" (T17 links here,
	// so the route stays fixed). A viewer reads it — a delivered report is a record,
	// not a mutation — and its document body is the canonical render that doubles as
	// the PDF/email spec (internal/message.RenderArtifact).
	mux.HandleFunc("GET /reports/delivery", s.requireLogin(s.reportDeliveryPage))
	// The delivered-report PDF download (#345): the print form of the same artifact
	// /reports/delivery renders, produced by internal/message.RenderArtifactPDF (a
	// pure-Go render, no external engine — it runs inside the distroless-static web
	// image). A viewer reads it — a delivered report is a record, not a mutation.
	mux.HandleFunc("GET /reports/delivery/pdf", s.requireLogin(s.reportDeliveryPDF))
	// The Reports schedule CRUD (#290, P0.6/T4): report scheduling is live. The
	// "New schedule" wizard and the row menu declare, edit, delete, and run one report
	// now. Every route is an admin config act (declaring/changing a schedule), gated
	// behind requireAdmin, so a viewer is refused before the handler. GET /new and
	// GET /{id}/edit render the stepped wizard (server post-back, no client runtime,
	// the onboarding pattern); POST /reports/schedule and /reports/schedule/edit carry
	// both the step-advance and the finishing insert/update; run and delete are the
	// row-menu mutations. All redirect back to /reports on success.
	// The wizard is a PRG post-back (#23f): each step POSTs to its route and 303-redirects
	// to a GET at the same route carrying the accumulated values, so the flow is
	// bookmarkable and harness-addressable. Create posts to /reports/schedule/new, edit to
	// /reports/schedule/{id}/edit; run and delete are the row-menu mutations. All redirect
	// back to /reports on success.
	mux.HandleFunc("GET /reports/schedule/new", s.requireAdmin(s.newReportScheduleWizard))
	mux.HandleFunc("POST /reports/schedule/new", s.requireAdmin(s.createReportSchedule))
	mux.HandleFunc("GET /reports/schedule/{id}/edit", s.requireAdmin(s.editReportScheduleWizard))
	mux.HandleFunc("POST /reports/schedule/{id}/edit", s.requireAdmin(s.editReportSchedule))
	mux.HandleFunc("POST /reports/schedule/run", s.requireAdmin(s.runReportScheduleNow))
	mux.HandleFunc("POST /reports/schedule/delete", s.requireAdmin(s.deleteReportSchedule))

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
	// The Inventory CSV export (#347): the folded open-span corpus the page shows, as
	// a downloadable file. A viewer reads it — an export is a read of the current
	// values the page already renders, never a mutation — mirroring the Drift export.
	mux.HandleFunc("GET /inventory/export", s.requireLogin(s.inventoryExport))
	// The per-asset drill-in (#296, T1): the destination of an Inventory row-click.
	// The route is stable so T15's Inventory can link straight to it. A Name key
	// carries neither `/` nor `@`, so it rides a plain path segment.
	mux.HandleFunc("GET /asset/{key}", s.requireLogin(s.assetPage))
	mux.HandleFunc("GET /drift", s.requireLogin(s.driftPage))
	// The Drift CSV export (#288): the transition feed for the active ?period= as a
	// downloadable file. A viewer reads it — an export is a read of the change the page
	// already shows, never a mutation — mirroring the Reports export.
	mux.HandleFunc("GET /drift/export", s.requireLogin(s.driftExport))
	// The per-run drill-in (#297, T2): the destination of a Drift "Batch detail"
	// entry. A run is one Dispatch (a fan-out of one Scan); the route is stable so
	// T16's Drift can link straight to it. A viewer reads it, like the Scans monitor
	// — it is a read-only window onto the Operational queue corpus. A Dispatch id is
	// an integer carrying neither `/` nor `@`, so it rides a plain path segment.
	mux.HandleFunc("GET /run/{id}", s.requireLogin(s.runPage))
	// /runs/{id} is the design-canonical run-detail route (WAYFINDER-MAP.md screen 9;
	// verify/states.json's missing-run capture navigates /runs/1408). The repo has served
	// run detail at /run/{id} since T2; screen 9 (RunDetail) owns the eventual /run→/runs
	// migration. Until then this alias reuses the same handler so the design-declared route
	// resolves — a nonexistent id (1408) renders the missing-run ErrorPage, matching the
	// state states.json declares. Routes are repo-owned (WORKFLOW.md).
	mux.HandleFunc("GET /runs/{id}", s.requireLogin(s.runPage))

	mux.HandleFunc("GET /signals", s.requireLogin(s.signalsPage))
	// The Signals CSV export (#346): the current census set the page evaluates, as a
	// downloadable file. A viewer reads it — an export is a read of the same signal
	// facts the page already shows, never a mutation — mirroring the Drift export.
	mux.HandleFunc("GET /signals/export", s.requireLogin(s.signalsExport))
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
	// The frozen scope.tmpl posts the org-name search to /proposals/search (field `org`);
	// runLookup accepts either route (#574).
	mux.HandleFunc("POST /proposals/search", s.requireAdmin(s.runLookup))
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
	// Stop-dispatch / terminate (DF-F4, #633): ending a Dispatch in flight is the same
	// class of admin act as triggering one — it changes what the worker runs — so both
	// carry the same requireAdmin gate the trigger does (a non-admin POST is 403 before
	// the handler). A stop cancels pending jobs and lets running ones finish; a terminate
	// kills running jobs too. Kept adjacent to /scans/trigger so a parallel route add
	// union-merges cleanly.
	mux.HandleFunc("POST /scans/stop", s.requireAdmin(s.stopScan))
	mux.HandleFunc("POST /scans/terminate", s.requireAdmin(s.terminateScan))

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
	// Mark-unread returns a message to unread (#473, ADR-0116): the Inbox detail
	// renders a "Mark unread" ghost button (Inbox.jsx:59), so read is reversible.
	// It carries the same `return=/inbox` field as the read acts above.
	mux.HandleFunc("POST /messages/unread", s.requireLogin(s.markMessageUnread))

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
	// that mirror lives at /settings?tab=delivery. Redirecting /verge-core into
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
	// The spec sources tab (#26) posts the enable/disable act here; enabling an
	// operator-accepted source carries accept_terms=true from the consent dialog.
	mux.HandleFunc("POST /settings/sources", s.requireAdmin(s.settingsSources))

	// The account surface folded into Settings → access (#281): GET /account now
	// redirects there (accountPage), so the merged SignIn's totp-enroll Cancel link
	// still lands somewhere real. The POST endpoints keep their paths and render the
	// Settings access sub-tab.
	// Profile (#304, T9, ADR-0110): the account's own page — identity, credentials
	// (password + 2FA status linking the existing TOTP flow), the current session,
	// and personal API tokens. Every route is viewer-readable (requireLogin): a
	// Profile is personal, so an account manages its own credentials and tokens
	// regardless of role — org-wide access stays admin-gated in Settings. The
	// destructive acts (revoke token, end session) are reached through a
	// ConfirmDialog rendered from a query param, never fired on a menu click, and
	// the token revoke carries a typed-name gate.
	mux.HandleFunc("GET /profile", s.requireLogin(s.profilePage))
	mux.HandleFunc("POST /profile/password", s.requireLogin(s.changePassword))
	mux.HandleFunc("POST /profile/tokens", s.requireLogin(s.createPersonalToken))
	mux.HandleFunc("POST /profile/tokens/revoke", s.requireLogin(s.revokePersonalToken))
	mux.HandleFunc("POST /profile/session/revoke", s.requireLogin(s.revokeSession))
	// Personal sessions surface (#406, ADR-0117): an account revokes one of its own live
	// sessions or signs out all but the current one. Both are owner-scoped in SQL, so a
	// posted id can never reach another account's session.
	mux.HandleFunc("POST /profile/sessions/revoke", s.requireLogin(s.revokeOneSession))
	mux.HandleFunc("POST /profile/sessions/revoke-others", s.requireLogin(s.signOutOtherSessions))
	// SSO self-link (#319, ADR-0113): an authenticated user links their own verified
	// identity so it can sign them in, and unlinks it. The link runs the OIDC round-trip
	// inside the caller's session (requireLogin) and binds (provider, sub) → their
	// account — the same per-user security surface that hosts TOTP enrollment.
	mux.HandleFunc("GET /profile/sso/{slug}/link", s.requireLogin(s.ssoLinkStart))
	mux.HandleFunc("GET /profile/sso/{slug}/link/callback", s.requireLogin(s.ssoLinkCallback))
	mux.HandleFunc("POST /profile/sso/unlink", s.requireLogin(s.ssoUnlink))

	mux.HandleFunc("GET /account", s.requireLogin(s.accountPage))
	mux.HandleFunc("POST /accounts", s.requireAdmin(s.createAccount))
	// GET /account/totp/enroll renders the two-factor enrollment screen (v3.7.0, SignIn family):
	// the profile "Enable" button POSTs to /account/totp/enable, but the frozen SignIn "enroll"
	// capture state navigates here by GET (what page.goto can drive). Both open the same screen.
	mux.HandleFunc("GET /account/totp/enroll", s.requireLogin(s.totpEnrollForm))
	mux.HandleFunc("POST /account/totp/enable", s.requireLogin(s.totpEnable))
	mux.HandleFunc("POST /account/totp/confirm", s.requireLogin(s.totpConfirm))

	// The Settings destination and every mutation it hosts are admin acts
	// (v1 spec §4.3, §6.1): viewing the operator's dials and moving them are
	// both gated behind requireAdmin.
	mux.HandleFunc("GET /settings", s.requireSettingsAdmin(s.settingsPage))
	mux.HandleFunc("POST /settings/accounts", s.requireAdmin(s.inviteAccount))
	mux.HandleFunc("POST /settings/accounts/role", s.requireAdmin(s.setAccountRole))
	// Team (#313, T18): re-enrollment resets a member's second factor; remove passes
	// through a typed-name ConfirmDialog. Both are admin acts and mutate one account.
	mux.HandleFunc("POST /settings/accounts/reenroll", s.requireAdmin(s.reenrollAccount))
	mux.HandleFunc("POST /settings/accounts/remove", s.requireAdmin(s.removeAccount))
	// Admin-wide sessions (#407, ADR-0117): the Access-group Sessions tab lists every
	// account's live sessions. Revoking any one session (not owner-scoped) and revoking
	// every session for one account (offboarding, through a typed-name confirm) are both
	// admin acts, gated like the rest of the /settings block.
	mux.HandleFunc("POST /settings/sessions/revoke", s.requireAdmin(s.revokeSessionAdmin))
	mux.HandleFunc("POST /settings/sessions/revoke-account", s.requireAdmin(s.revokeAccountSessions))
	mux.HandleFunc("POST /settings/channels", s.requireAdmin(s.createChannel))
	mux.HandleFunc("POST /settings/channels/update", s.requireAdmin(s.updateChannel))
	mux.HandleFunc("POST /settings/channels/delete", s.requireAdmin(s.deleteChannel))
	mux.HandleFunc("POST /settings/retention", s.requireAdmin(s.updateRetention))

	// Single sign-on config (#293, ADR-0112): declaring, editing, re-keying and
	// removing an OIDC provider are admin config acts, gated like channel and seed
	// declaration. The secret has its own write path so an edit that leaves it blank
	// keeps the existing one (the channel-secret pattern). Removing an identity binding
	// (#319, ADR-0113) is the admin offboarding / seat-reassignment act.
	mux.HandleFunc("POST /settings/sso", s.requireAdmin(s.createSSOProvider))
	mux.HandleFunc("POST /settings/sso/update", s.requireAdmin(s.updateSSOProvider))
	mux.HandleFunc("POST /settings/sso/secret", s.requireAdmin(s.setSSOProviderSecret))
	mux.HandleFunc("POST /settings/sso/delete", s.requireAdmin(s.deleteSSOProvider))
	mux.HandleFunc("POST /settings/sso/identity/remove", s.requireAdmin(s.removeSSOBinding))

	// Integrations (#308): a third-party install tile is a Declared act. Installing
	// one records consent to the grants it would receive; disconnecting passes
	// through a confirm step (never fired on the tile click). Both mutations are
	// admin acts, and the Integrations tab itself is reached only through the
	// admin-gated /settings — an integration is distinct from a delivery channel
	// and from a discovery source, and keeps its own routes. These routes are only
	// registered when the Integrations surface is live (#388, integrationsEnabled):
	// with the surface hidden they stay unregistered, so no user-facing route can
	// write to integration_state.
	if integrationsEnabled {
		mux.HandleFunc("POST /settings/integrations/install", s.requireAdmin(s.installIntegration))
		// remove is the spec drawer's Remove act (#26j); disconnect is its pre-spec
		// alias, kept resolving. test acknowledges the spec drawer's "Send test".
		mux.HandleFunc("POST /settings/integrations/remove", s.requireAdmin(s.removeIntegration))
		mux.HandleFunc("POST /settings/integrations/disconnect", s.requireAdmin(s.removeIntegration))
		mux.HandleFunc("POST /settings/integrations/test", s.requireAdmin(s.testIntegration))
	}

	// Dev-only pixel-parity harness routes (devfixtures.go), registered only in a
	// VERGE_DEV build so no /dev surface exists in a real deployment. They let the
	// capture harness reach the error states states.json declares: /dev/403 renders the
	// plain 403, /dev/panic exercises the 500 recovery, and /dev/session/{role} mints a
	// session as the fixture admin/viewer so a state's per-state `session` is established
	// before capture. (404, missing-subject, missing-run, settings-forbidden reach their
	// kinds through real routes on fixture data.)
	if s.devMode {
		mux.HandleFunc("GET /dev/403", s.forbidden)
		mux.HandleFunc("GET /dev/panic", s.devPanic)
		mux.HandleFunc("GET /dev/session/{role}", s.devSessionMint)
		// Screen 3 (Profile, #542): prepare the capture context — re-seed the Profile fixture
		// (so a prior state's minted token never leaks) and hand back the seeded current-session
		// cookie (so the Firefox·macOS row wears the "this device" badge without minting a
		// fourth session). The literal path outranks the {role} wildcard above.
		mux.HandleFunc("GET /dev/profile/session", s.devProfileSessionPrepare)
		// Screen 5 (Setup, #550): realize states.json setup's seed:"empty" — empty the account
		// table and reopen the first-run window under the pinned fixture token, so GET /setup
		// renders the open bootstrap form. Captured last, so emptying the shared DB is safe.
		mux.HandleFunc("GET /dev/seed/empty", s.devSetupSeedEmpty)
		// Screen 6 (Coverage, #552): realize states.json coverage's "empty" state
		// (seed:"empty-authed"). Unlike /dev/seed/empty it keeps the account table (the
		// authed admin session Coverage needs), clearing only the coverage view for the
		// next render. A distinct literal path — it outranks nothing and collides with
		// nothing ("/dev/seed/empty" is a different literal).
		mux.HandleFunc("GET /dev/seed/empty-authed", s.devCoverageSeedEmpty)
	}

	// Recovered panics render the 500 error page with a real, logged incident id
	// (T11, #306). Wrapped once here at the mux-construction boundary; the render
	// and incident-id helpers live in errors.go.
	return s.recoverPanics(mux)
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
