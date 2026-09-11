package act

// An enum beside a subject union halves the code and makes an illegal pair expressible (§4).

type Act interface { // Class is the stored action token; Label and Subject are the rendered cells

	Class() string
	Label() string
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

// It is the rendered Action cell, so a copy change rewrites no row (§2.1, §4.1).

func (SetupCompleted) Label() string          { return "Instance set up" }
func (PasswordResetCompleted) Label() string  { return "Password reset" }
func (InviteAccepted) Label() string          { return "Invite accepted" }
func (OnboardingFinished) Label() string      { return "Scan dispatched" }
func (SeedDeclared) Label() string            { return "Seed declared" }
func (SeedWithdrawn) Label() string           { return "Seed withdrawn" }
func (SeedCustodyMoved) Label() string        { return "Custody moved" }
func (ZoneDeclared) Label() string            { return "Zone declared" }
func (ZoneCadenceSet) Label() string          { return "Dial moved" }
func (DNSCadenceSet) Label() string           { return "Dial moved" }
func (ExclusionDeclared) Label() string       { return "Exclusion declared" }
func (ExclusionLifted) Label() string         { return "Exclusion lifted" }
func (ColdMoved) Label() string               { return "Cold scan moved" }
func (VantageDeclared) Label() string         { return "Vantage declared" }
func (VantageResolverSet) Label() string      { return "Resolver set" }
func (ScheduleDeclared) Label() string        { return "Schedule declared" }
func (ScheduleEdited) Label() string          { return "Schedule edited" }
func (ScheduleWithdrawn) Label() string       { return "Schedule withdrawn" }
func (AnnotationDeclared) Label() string      { return "Annotation declared" }
func (AnnotationWithdrawn) Label() string     { return "Annotation withdrawn" }
func (ProposalQueried) Label() string         { return "Registries queried" }
func (ProposalConfirmed) Label() string       { return "Proposal confirmed" }
func (ProposalDeclined) Label() string        { return "Proposal declined" }
func (ProposalDeclineUndone) Label() string   { return "Decline lifted" }
func (ScanTriggered) Label() string           { return "Scan triggered" }
func (ScanStopped) Label() string             { return "Scan stopped" }
func (ScanTerminated) Label() string          { return "Scan terminated" }
func (FrequencyMoved) Label() string          { return "Frequency moved" }
func (SourceMoved) Label() string             { return "Source moved" }
func (PasswordChanged) Label() string         { return "Password changed" }
func (TokenMinted) Label() string             { return "Token minted" }
func (TokenRevoked) Label() string            { return "Token revoked" }
func (SSOUnlinked) Label() string             { return "SSO unlinked" }
func (AccountCreated) Label() string          { return "Account created" }
func (TOTPEnrolled) Label() string            { return "TOTP enrolled" }
func (InviteMinted) Label() string            { return "Invite minted" }
func (AccountRoleMoved) Label() string        { return "Role moved" }
func (TOTPStripped) Label() string            { return "TOTP stripped" }
func (AccountRemoved) Label() string          { return "Account removed" }
func (ChannelDeclared) Label() string         { return "Channel declared" }
func (ChannelUpdated) Label() string          { return "Channel updated" }
func (ChannelWithdrawn) Label() string        { return "Channel withdrawn" }
func (ChannelTested) Label() string           { return "Channel tested" }
func (TranscriptCurrencySet) Label() string   { return "Dial moved" }
func (ObservationCurrencySet) Label() string  { return "Dial moved" }
func (DispatchCadenceSet) Label() string      { return "Dial moved" }
func (AddressCapSet) Label() string           { return "Dial moved" }
func (UpdateCheckMoved) Label() string        { return "Update check moved" }
func (RestoreApplied) Label() string          { return "Restore applied" }
func (APIAccessMoved) Label() string          { return "API access moved" }
func (SSOProviderDeclared) Label() string     { return "SSO provider declared" }
func (SSOProviderUpdated) Label() string      { return "SSO provider updated" }
func (SSOProviderSecretSet) Label() string    { return "SSO secret set" }
func (SSOProviderWithdrawn) Label() string    { return "SSO provider withdrawn" }
func (SSOBindingRemoved) Label() string       { return "SSO binding removed" }
func (IntegrationInstalled) Label() string    { return "Integration installed" }
func (IntegrationRemoved) Label() string      { return "Integration removed" }
func (IntegrationChannelBound) Label() string { return "Channel bound" }
func (IntegrationTested) Label() string       { return "Integration tested" }
func (SSOBindingCreated) Label() string       { return "SSO linked" }
func (TranscriptDisclosed) Label() string     { return "Raw output disclosed" }

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
