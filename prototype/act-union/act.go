package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.

// Act is the closed union an Act corpus stores: one variant per act class, each
// naming its own typed subject (map #1786 Settled #12). A subject_type TEXT plus
// a nullable column per kind is the shape ADR-0126 rejected by name.
//
// The alternative weighed and rejected here was the product — an Action enum of
// 60 beside a Subject union of 26. It halves the code and makes an illegal pair
// expressible: seed.withdrawn carrying an SSOProvider subject would compile.
type Act interface {
	Class() string
	Subject() string
	isAct()
}

// 62 variants. 60 are #1791's corrected class count; observation.currency.set
// and dispatch.cadence.set are the Q4 split of POST /coverage/retention, which
// #1789 left open; migration.applied is conditional on #1805.

// ---- auth and onboarding ----

type SetupCompleted struct{ AccountAtRole }
type PasswordResetApplied struct{ AccountRef }
type InviteAccepted struct{ AccountAtRole }
type OnboardingFinished struct{ ScanKindRef }

func (SetupCompleted) Class() string       { return "setup.completed" }
func (PasswordResetApplied) Class() string { return "password.reset" }
func (InviteAccepted) Class() string       { return "invite.accepted" }
func (OnboardingFinished) Class() string   { return "onboarding.finished" }

// ---- scope and declaration ----

type SeedDeclared struct{ SeedScope }
type SeedWithdrawn struct{ SeedScope }
type SeedCustodyMoved struct{ ScopeMove }
type ZoneDeclared struct{ ZoneName }
type ZoneCadenceSet struct{ DialMove }
type DNSCadenceSet struct{ DialMove }
type ExclusionDeclared struct{ ExclusionTerm }
type ExclusionLifted struct{ ExclusionTerm }
type ColdMoved struct{ ScopeMove }
type VantageDeclared struct{ VantageRef }
type VantageResolverSet struct{ VantageRef }

func (SeedDeclared) Class() string       { return "seed.declared" }
func (SeedWithdrawn) Class() string      { return "seed.withdrawn" }
func (SeedCustodyMoved) Class() string   { return "seed.custody.moved" }
func (ZoneDeclared) Class() string       { return "zone.declared" }
func (ZoneCadenceSet) Class() string     { return "zone.cadence.set" }
func (DNSCadenceSet) Class() string      { return "dns.cadence.set" }
func (ExclusionDeclared) Class() string  { return "exclusion.declared" }
func (ExclusionLifted) Class() string    { return "exclusion.lifted" }
func (ColdMoved) Class() string          { return "cold.moved" }
func (VantageDeclared) Class() string    { return "vantage.declared" }
func (VantageResolverSet) Class() string { return "vantage.resolver.set" }

// ---- reports ----

type ScheduleDeclared struct{ ScheduleRef }
type ScheduleEdited struct{ ScheduleRef }
type ScheduleWithdrawn struct{ ScheduleRef }

func (ScheduleDeclared) Class() string  { return "schedule.declared" }
func (ScheduleEdited) Class() string    { return "schedule.edited" }
func (ScheduleWithdrawn) Class() string { return "schedule.withdrawn" }

// ---- annotations ----

// The Act carries the actor; the Annotation object still carries none, because
// that field would sit inside a Declared term the probing gate reads (ADR-0073).

type AnnotationDeclared struct{ AnnotationRef }
type AnnotationWithdrawn struct{ AnnotationRef }

func (AnnotationDeclared) Class() string  { return "annotation.declared" }
func (AnnotationWithdrawn) Class() string { return "annotation.withdrawn" }

// ---- proposals ----

type ProposalQueried struct{ ProposalQuery }
type ProposalConfirmed struct{ SeedScope }
type ProposalDeclined struct{ ExclusionTerm }
type ProposalDeclineUndone struct{ ExclusionTerm }

func (ProposalQueried) Class() string       { return "proposal.queried" }
func (ProposalConfirmed) Class() string     { return "proposal.confirmed" }
func (ProposalDeclined) Class() string      { return "proposal.declined" }
func (ProposalDeclineUndone) Class() string { return "proposal.decline.undone" }

// ---- scans ----

type ScanTriggered struct{ ScanKindRef }
type ScanStopped struct{ DispatchRef }
type ScanTerminated struct{ DispatchRef }

func (ScanTriggered) Class() string  { return "scan.triggered" }
func (ScanStopped) Class() string    { return "scan.stopped" }
func (ScanTerminated) Class() string { return "scan.terminated" }

// ---- verge-core and sources ----

type FrequencyMoved struct{ PortMove }
type SourceMoved struct{ SourceMove }

func (FrequencyMoved) Class() string { return "frequency.moved" }
func (SourceMoved) Class() string    { return "source.moved" }

// ---- profile ----

type PasswordChanged struct{ AccountRef }
type TokenMinted struct{ TokenRef }
type TokenRevoked struct{ TokenRef }
type SSOUnlinked struct{ ProviderRef }

func (PasswordChanged) Class() string { return "password.changed" }
func (TokenMinted) Class() string     { return "token.minted" }
func (TokenRevoked) Class() string    { return "token.revoked" }
func (SSOUnlinked) Class() string     { return "sso.unlinked" }

// ---- account ----

type AccountCreated struct{ AccountAtRole }
type TOTPEnrolled struct{ AccountRef }

func (AccountCreated) Class() string { return "account.created" }
func (TOTPEnrolled) Class() string   { return "totp.enrolled" }

// ---- settings, team ----

type InviteMinted struct{ RoleRef }
type AccountRoleMoved struct{ AccountAtRole }
type TOTPStripped struct{ AccountRef }
type AccountRemoved struct{ AccountRef }

func (InviteMinted) Class() string     { return "invite.minted" }
func (AccountRoleMoved) Class() string { return "account.role.moved" }
func (TOTPStripped) Class() string     { return "totp.stripped" }
func (AccountRemoved) Class() string   { return "account.removed" }

// ---- settings, channels ----

type ChannelDeclared struct{ ChannelRef }
type ChannelUpdated struct{ ChannelRef }
type ChannelWithdrawn struct{ ChannelRef }
type ChannelTested struct{ ChannelRef }

func (ChannelDeclared) Class() string  { return "channel.declared" }
func (ChannelUpdated) Class() string   { return "channel.updated" }
func (ChannelWithdrawn) Class() string { return "channel.withdrawn" }
func (ChannelTested) Class() string    { return "channel.tested" }

// ---- settings, instance dials ----

// ObservationCurrencySet and DispatchCadenceSet are one form submit and two
// rows: folding them puts a list in the Subject cell, the shape #1788 barred.

type TranscriptCurrencySet struct{ DialMove }
type ObservationCurrencySet struct{ DialMove }
type DispatchCadenceSet struct{ DialMove }
type AddressCapSet struct{ DialMove }
type UpdateCheckMoved struct{ ToggleMove }
type RestoreApplied struct{ ArchiveRef }
type APIAccessMoved struct{ ToggleMove }

func (TranscriptCurrencySet) Class() string  { return "transcript.currency.set" }
func (ObservationCurrencySet) Class() string { return "observation.currency.set" }
func (DispatchCadenceSet) Class() string     { return "dispatch.cadence.set" }
func (AddressCapSet) Class() string          { return "address.cap.set" }
func (UpdateCheckMoved) Class() string       { return "update.check.moved" }
func (RestoreApplied) Class() string         { return "restore.applied" }
func (APIAccessMoved) Class() string         { return "api.access.moved" }

// ---- settings, SSO ----

// SSOProviderSecretSet has one field. ADR-0053's split — the act is auditable
// and its value is not — is inexpressible here rather than discouraged.

type SSOProviderDeclared struct{ ProviderRef }
type SSOProviderUpdated struct{ ProviderRef }
type SSOProviderSecretSet struct{ ProviderRef }
type SSOProviderWithdrawn struct{ ProviderRef }
type SSOBindingRemoved struct{ BindingRef }

func (SSOProviderDeclared) Class() string  { return "sso.provider.declared" }
func (SSOProviderUpdated) Class() string   { return "sso.provider.updated" }
func (SSOProviderSecretSet) Class() string { return "sso.provider.secret.set" }
func (SSOProviderWithdrawn) Class() string { return "sso.provider.withdrawn" }
func (SSOBindingRemoved) Class() string    { return "sso.binding.removed" }

// ---- settings, integrations ----

type IntegrationInstalled struct{ IntegrationRef }
type IntegrationRemoved struct{ IntegrationRef }
type IntegrationChannelBound struct{ IntegrationChannelRef }
type IntegrationTested struct{ IntegrationChannelRef }

func (IntegrationInstalled) Class() string    { return "integration.installed" }
func (IntegrationRemoved) Class() string      { return "integration.removed" }
func (IntegrationChannelBound) Class() string { return "integration.channel.bound" }
func (IntegrationTested) Class() string       { return "integration.tested" }

// ---- acts that are not POST routes ----

// SSOBindingCreated and TranscriptDisclosed both arrive on a GET, so a sweep of
// the POST registrations misses them (#1789, #1791).

type SSOBindingCreated struct{ BindingRef }
type TranscriptDisclosed struct{ JobRef }

// MigrationApplied is conditional on #1805, which rules whether a migration may
// use the System variant at all.
type MigrationApplied struct{ MigrationRef }

func (SSOBindingCreated) Class() string   { return "sso.binding.created" }
func (TranscriptDisclosed) Class() string { return "transcript.disclosed" }
func (MigrationApplied) Class() string    { return "migration.applied" }

func (SetupCompleted) isAct()          {}
func (PasswordResetApplied) isAct()    {}
func (InviteAccepted) isAct()          {}
func (OnboardingFinished) isAct()      {}
func (SeedDeclared) isAct()            {}
func (SeedWithdrawn) isAct()           {}
func (SeedCustodyMoved) isAct()        {}
func (ZoneDeclared) isAct()            {}
func (ZoneCadenceSet) isAct()          {}
func (DNSCadenceSet) isAct()           {}
func (ExclusionDeclared) isAct()       {}
func (ExclusionLifted) isAct()         {}
func (ColdMoved) isAct()               {}
func (VantageDeclared) isAct()         {}
func (VantageResolverSet) isAct()      {}
func (ScheduleDeclared) isAct()        {}
func (ScheduleEdited) isAct()          {}
func (ScheduleWithdrawn) isAct()       {}
func (AnnotationDeclared) isAct()      {}
func (AnnotationWithdrawn) isAct()     {}
func (ProposalQueried) isAct()         {}
func (ProposalConfirmed) isAct()       {}
func (ProposalDeclined) isAct()        {}
func (ProposalDeclineUndone) isAct()   {}
func (ScanTriggered) isAct()           {}
func (ScanStopped) isAct()             {}
func (ScanTerminated) isAct()          {}
func (FrequencyMoved) isAct()          {}
func (SourceMoved) isAct()             {}
func (PasswordChanged) isAct()         {}
func (TokenMinted) isAct()             {}
func (TokenRevoked) isAct()            {}
func (SSOUnlinked) isAct()             {}
func (AccountCreated) isAct()          {}
func (TOTPEnrolled) isAct()            {}
func (InviteMinted) isAct()            {}
func (AccountRoleMoved) isAct()        {}
func (TOTPStripped) isAct()            {}
func (AccountRemoved) isAct()          {}
func (ChannelDeclared) isAct()         {}
func (ChannelUpdated) isAct()          {}
func (ChannelWithdrawn) isAct()        {}
func (ChannelTested) isAct()           {}
func (TranscriptCurrencySet) isAct()   {}
func (ObservationCurrencySet) isAct()  {}
func (DispatchCadenceSet) isAct()      {}
func (AddressCapSet) isAct()           {}
func (UpdateCheckMoved) isAct()        {}
func (RestoreApplied) isAct()          {}
func (APIAccessMoved) isAct()          {}
func (SSOProviderDeclared) isAct()     {}
func (SSOProviderUpdated) isAct()      {}
func (SSOProviderSecretSet) isAct()    {}
func (SSOProviderWithdrawn) isAct()    {}
func (SSOBindingRemoved) isAct()       {}
func (IntegrationInstalled) isAct()    {}
func (IntegrationRemoved) isAct()      {}
func (IntegrationChannelBound) isAct() {}
func (IntegrationTested) isAct()       {}
func (SSOBindingCreated) isAct()       {}
func (TranscriptDisclosed) isAct()     {}
func (MigrationApplied) isAct()        {}
