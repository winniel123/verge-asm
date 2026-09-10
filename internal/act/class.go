package act

// An enum beside a subject union halves the code and makes an illegal pair expressible (§4).

type Act interface { // Class is the stored action token; Subject is the rendered cell

	Class() string
	Subject() string
	isAct()
}

// Limb 2 of the four-limb predicate — who may act on this instance (§1.2).

type SetupCompleted struct{ AccountAtRole }
type PasswordResetCompleted struct{ AccountRef }
type InviteAccepted struct{ AccountAtRole }
type PasswordChanged struct{ AccountRef }
type TokenMinted struct{ TokenRef }
type TokenRevoked struct{ TokenRef }
type SSOUnlinked struct{ ProviderRef }
type AccountCreated struct{ AccountAtRole }
type TOTPEnrolled struct{ AccountRef }
type InviteMinted struct{ InviteMint }
type AccountRoleMoved struct{ AccountAtRole }
type TOTPStripped struct{ AccountRef }
type AccountRemoved struct{ AccountRef }
type APIAccessMoved struct{ DialMove }
type SSOProviderDeclared struct{ ProviderRef }
type SSOProviderUpdated struct{ ProviderRef }
type SSOProviderSecretSet struct{ ProviderRef }
type SSOProviderWithdrawn struct{ ProviderRef }
type SSOBindingRemoved struct{ BindingRef }
type SSOBindingCreated struct{ BindingRef }

// Limb 1 of the four-limb predicate — the estate's declaration (§1.1).

type SeedDeclared struct{ SeedScope }
type SeedWithdrawn struct{ SeedScope }
type SeedCustodyMoved struct{ CustodyMove }
type ZoneDeclared struct{ ZoneRef }
type ZoneCadenceSet struct{ DialMove }
type DNSCadenceSet struct{ DialMove }
type ExclusionDeclared struct{ ExclusionRef }
type ExclusionLifted struct{ ExclusionRef }
type ColdMoved struct{ ColdMove }
type VantageDeclared struct{ VantageRef }
type VantageResolverSet struct{ ResolverRef }
type ScheduleDeclared struct{ ScheduleRef }
type ScheduleEdited struct{ ScheduleRef }
type ScheduleWithdrawn struct{ ScheduleRef }
type AnnotationDeclared struct{ AnnotationRef }
type AnnotationWithdrawn struct{ AnnotationRef }
type ProposalConfirmed struct{ SeedScope }
type ProposalDeclined struct{ ExclusionRef }
type ProposalDeclineUndone struct{ ExclusionRef }
type FrequencyMoved struct{ FrequencyMove }
type SourceMoved struct{ SourceMove }
type ChannelDeclared struct{ ChannelRef }
type ChannelUpdated struct{ ChannelRef }
type ChannelWithdrawn struct{ ChannelRef }
type TranscriptCurrencySet struct{ DialMove }
type ObservationCurrencySet struct{ DialMove }
type DispatchCadenceSet struct{ DialMove }
type AddressCapSet struct{ DialMove }
type UpdateCheckMoved struct{ DialMove }
type IntegrationInstalled struct{ IntegrationRef }
type IntegrationRemoved struct{ IntegrationRef }
type IntegrationChannelBound struct{ IntegrationChannel }

// Limb 3 of the four-limb predicate — directing the instance to act on the network (§1.3).

type OnboardingFinished struct{ ScanProfile }
type ProposalQueried struct{ OrgQuery }
type ScanTriggered struct{ ScanProfile }
type ScanStopped struct{ DispatchRef }
type ScanTerminated struct{ DispatchRef }
type ChannelTested struct{ ChannelRef }
type IntegrationTested struct{ IntegrationChannel }

// Limbs 1 and 2 together — a restore moves both the estate and who may act (§2.1).

type RestoreApplied struct{ RestoreRef }

// Limb 4 — a disclosure and never a read, which §1.4 refused by name (§1.4).

type TranscriptDisclosed struct{ TranscriptRef }

func (SetupCompleted) Class() string          { return "setup.completed" }
func (PasswordResetCompleted) Class() string  { return "password.reset" }
func (InviteAccepted) Class() string          { return "invite.accepted" }
func (OnboardingFinished) Class() string      { return "onboarding.finished" }
func (SeedDeclared) Class() string            { return "seed.declared" }
func (SeedWithdrawn) Class() string           { return "seed.withdrawn" }
func (SeedCustodyMoved) Class() string        { return "seed.custody.moved" }
func (ZoneDeclared) Class() string            { return "zone.declared" }
func (ZoneCadenceSet) Class() string          { return "zone.cadence.set" }
func (DNSCadenceSet) Class() string           { return "dns.cadence.set" }
func (ExclusionDeclared) Class() string       { return "exclusion.declared" }
func (ExclusionLifted) Class() string         { return "exclusion.lifted" }
func (ColdMoved) Class() string               { return "cold.moved" }
func (VantageDeclared) Class() string         { return "vantage.declared" }
func (VantageResolverSet) Class() string      { return "vantage.resolver.set" }
func (ScheduleDeclared) Class() string        { return "schedule.declared" }
func (ScheduleEdited) Class() string          { return "schedule.edited" }
func (ScheduleWithdrawn) Class() string       { return "schedule.withdrawn" }
func (AnnotationDeclared) Class() string      { return "annotation.declared" }
func (AnnotationWithdrawn) Class() string     { return "annotation.withdrawn" }
func (ProposalQueried) Class() string         { return "proposal.queried" }
func (ProposalConfirmed) Class() string       { return "proposal.confirmed" }
func (ProposalDeclined) Class() string        { return "proposal.declined" }
func (ProposalDeclineUndone) Class() string   { return "proposal.decline.undone" }
func (ScanTriggered) Class() string           { return "scan.triggered" }
func (ScanStopped) Class() string             { return "scan.stopped" }
func (ScanTerminated) Class() string          { return "scan.terminated" }
func (FrequencyMoved) Class() string          { return "frequency.moved" }
func (SourceMoved) Class() string             { return "source.moved" }
func (PasswordChanged) Class() string         { return "password.changed" }
func (TokenMinted) Class() string             { return "token.minted" }
func (TokenRevoked) Class() string            { return "token.revoked" }
func (SSOUnlinked) Class() string             { return "sso.unlinked" }
func (AccountCreated) Class() string          { return "account.created" }
func (TOTPEnrolled) Class() string            { return "totp.enrolled" }
func (InviteMinted) Class() string            { return "invite.minted" }
func (AccountRoleMoved) Class() string        { return "account.role.moved" }
func (TOTPStripped) Class() string            { return "totp.stripped" }
func (AccountRemoved) Class() string          { return "account.removed" }
func (ChannelDeclared) Class() string         { return "channel.declared" }
func (ChannelUpdated) Class() string          { return "channel.updated" }
func (ChannelWithdrawn) Class() string        { return "channel.withdrawn" }
func (ChannelTested) Class() string           { return "channel.tested" }
func (TranscriptCurrencySet) Class() string   { return "transcript.currency.set" }
func (ObservationCurrencySet) Class() string  { return "observation.currency.set" }
func (DispatchCadenceSet) Class() string      { return "dispatch.cadence.set" }
func (AddressCapSet) Class() string           { return "address.cap.set" }
func (UpdateCheckMoved) Class() string        { return "update.check.moved" }
func (RestoreApplied) Class() string          { return "restore.applied" }
func (APIAccessMoved) Class() string          { return "api.access.moved" }
func (SSOProviderDeclared) Class() string     { return "sso.provider.declared" }
func (SSOProviderUpdated) Class() string      { return "sso.provider.updated" }
func (SSOProviderSecretSet) Class() string    { return "sso.provider.secret.set" }
func (SSOProviderWithdrawn) Class() string    { return "sso.provider.withdrawn" }
func (SSOBindingRemoved) Class() string       { return "sso.binding.removed" }
func (IntegrationInstalled) Class() string    { return "integration.installed" }
func (IntegrationRemoved) Class() string      { return "integration.removed" }
func (IntegrationChannelBound) Class() string { return "integration.channel.bound" }
func (IntegrationTested) Class() string       { return "integration.tested" }
func (SSOBindingCreated) Class() string       { return "sso.binding.created" }
func (TranscriptDisclosed) Class() string     { return "transcript.disclosed" }

// The marker sits per variant, never on a promoted payload, or an outside type could join.

func (SetupCompleted) isAct()          {}
func (PasswordResetCompleted) isAct()  {}
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
