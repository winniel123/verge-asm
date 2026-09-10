package act

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// It is the decoder's registry and the test's denominator, so a variant with no sample fails.

var Classes = []Act{
	SetupCompleted{},
	PasswordResetCompleted{},
	InviteAccepted{},
	OnboardingFinished{},
	SeedDeclared{},
	SeedWithdrawn{},
	SeedCustodyMoved{},
	ZoneDeclared{},
	ZoneCadenceSet{},
	DNSCadenceSet{},
	ExclusionDeclared{},
	ExclusionLifted{},
	ColdMoved{},
	VantageDeclared{},
	VantageResolverSet{},
	ScheduleDeclared{},
	ScheduleEdited{},
	ScheduleWithdrawn{},
	AnnotationDeclared{},
	AnnotationWithdrawn{},
	ProposalQueried{},
	ProposalConfirmed{},
	ProposalDeclined{},
	ProposalDeclineUndone{},
	ScanTriggered{},
	ScanStopped{},
	ScanTerminated{},
	FrequencyMoved{},
	SourceMoved{},
	PasswordChanged{},
	TokenMinted{},
	TokenRevoked{},
	SSOUnlinked{},
	AccountCreated{},
	TOTPEnrolled{},
	InviteMinted{},
	AccountRoleMoved{},
	TOTPStripped{},
	AccountRemoved{},
	ChannelDeclared{},
	ChannelUpdated{},
	ChannelWithdrawn{},
	ChannelTested{},
	TranscriptCurrencySet{},
	ObservationCurrencySet{},
	DispatchCadenceSet{},
	AddressCapSet{},
	UpdateCheckMoved{},
	RestoreApplied{},
	APIAccessMoved{},
	SSOProviderDeclared{},
	SSOProviderUpdated{},
	SSOProviderSecretSet{},
	SSOProviderWithdrawn{},
	SSOBindingRemoved{},
	IntegrationInstalled{},
	IntegrationRemoved{},
	IntegrationChannelBound{},
	IntegrationTested{},
	SSOBindingCreated{},
	TranscriptDisclosed{},
}

var registry = func() map[string]reflect.Type {
	m := make(map[string]reflect.Type, len(Classes))
	for _, a := range Classes {
		class := a.Class()
		if prior, dup := m[class]; dup {
			// A duplicate token makes two variants indistinguishable on read.
			panic(fmt.Sprintf("act: class %q is claimed by both %s and %T", class, prior, a))
		}
		m[class] = reflect.TypeOf(a)
	}
	return m
}()

func EncodeSubject(a Act) (string, []byte, error) {
	if a == nil {
		return "", nil, fmt.Errorf("act: a nil Act carries no class")
	}
	class := a.Class()
	if _, known := registry[class]; !known {
		// A variant outside Classes decodes to nothing, so it never round-trips.
		return "", nil, fmt.Errorf("act: class %q of %T is not in Classes", class, a)
	}
	payload, err := json.Marshal(a)
	if err != nil {
		return "", nil, fmt.Errorf("act: marshal subject of %q: %w", class, err)
	}
	return class, payload, nil
}

func DecodeSubject(action string, subject []byte) (Act, error) {
	t, ok := registry[action]
	if !ok {
		// A closed union we author errors on an unknown token, never a mislabelled row (ADR-0209).
		return nil, fmt.Errorf("act: unknown action %q", action)
	}
	p := reflect.New(t)
	if err := json.Unmarshal(subject, p.Interface()); err != nil {
		return nil, fmt.Errorf("act: decode subject of %q: %w", action, err)
	}
	return p.Elem().Interface().(Act), nil
}
