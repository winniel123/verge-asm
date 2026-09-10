package act

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// One inhabited value per act class, with the Subject render §2.1 draws for it.
// TestSamplesCoverEveryClass holds this table against Classes in both directions.

var samples = []struct {
	act     Act
	subject string
}{
	{SetupCompleted{AccountAtRole{Username: "root", Role: "admin"}}, "root · admin"},
	{PasswordResetCompleted{AccountRef{Username: "alice"}}, "alice"},
	{InviteAccepted{AccountAtRole{Username: "bob", Role: "viewer"}}, "bob · viewer"},
	{OnboardingFinished{ScanProfile{Profile: "hot"}}, "hot"},
	{SeedDeclared{SeedScope{Scope: "10.0.0.0/8"}}, "10.0.0.0/8"},
	{SeedWithdrawn{SeedScope{Scope: "10.0.0.0/8"}}, "10.0.0.0/8"},
	{SeedCustodyMoved{CustodyMove{Scope: "example.com", Disposition: "custody extended"}}, "example.com · custody extended"},
	{ZoneDeclared{ZoneRef{Zone: "example.com"}}, "example.com"},
	{ZoneCadenceSet{DialMove{Dial: "zone scan cadence", Value: "24h"}}, "zone scan cadence · 24h"},
	{DNSCadenceSet{DialMove{Dial: "dns scan cadence", Value: "6h"}}, "dns scan cadence · 6h"},
	{ExclusionDeclared{ExclusionRef{Kind: "address", Scope: "10.1.2.0/24"}}, "address 10.1.2.0/24"},
	{ExclusionLifted{ExclusionRef{Kind: "address", Scope: "10.1.2.0/24"}}, "address 10.1.2.0/24"},
	{ColdMoved{ColdMove{Scope: "10.0.0.0/8", Disposition: "cold opt-in"}}, "10.0.0.0/8 · cold opt-in"},
	{VantageDeclared{VantageRef{Endpoint: "probe@edge-01.example.com:22"}}, "probe@edge-01.example.com:22"},
	{VantageResolverSet{ResolverRef{Resolver: "local"}}, "local"},
	{ScheduleDeclared{ScheduleRef{Name: "weekly exposure"}}, "weekly exposure"},
	{ScheduleEdited{ScheduleRef{Name: "weekly exposure"}}, "weekly exposure"},
	{ScheduleWithdrawn{ScheduleRef{Name: "weekly exposure"}}, "weekly exposure"},
	{AnnotationDeclared{AnnotationRef{SubjectKey: "10.0.4.9:443", Signal: "expired certificate"}}, "10.0.4.9:443 · expired certificate"},
	{AnnotationWithdrawn{AnnotationRef{SubjectKey: "10.0.4.9:443", Signal: "expired certificate"}}, "10.0.4.9:443 · expired certificate"},
	{ProposalQueried{OrgQuery{Term: "Example Holdings Ltd"}}, "Example Holdings Ltd"},
	{ProposalConfirmed{SeedScope{Scope: "198.51.100.0/24"}}, "198.51.100.0/24"},
	{ProposalDeclined{ExclusionRef{Kind: "address", Scope: "203.0.113.0/24"}}, "address 203.0.113.0/24"},
	{ProposalDeclineUndone{ExclusionRef{Kind: "address", Scope: "203.0.113.0/24"}}, "address 203.0.113.0/24"},
	{ScanTriggered{ScanProfile{Profile: "hot"}}, "hot"},
	{ScanStopped{DispatchRef{DispatchID: 418, Profile: "hot"}}, "dispatch 418 · hot"},
	{ScanTerminated{DispatchRef{DispatchID: 418, Profile: "hot"}}, "dispatch 418 · hot"},
	{FrequencyMoved{FrequencyMove{Port: "8443", Disposition: "add"}}, "port 8443 · add"},
	{SourceMoved{SourceMove{Slug: "crtsh", Disposition: "enabled"}}, "crtsh · enabled"},
	{PasswordChanged{AccountRef{Username: "alice"}}, "alice"},
	{TokenMinted{TokenRef{Label: "ci reader", Prefix: "vga_7f21"}}, "ci reader (vga_7f21)"},
	{TokenRevoked{TokenRef{Label: "ci reader", Prefix: "vga_7f21"}}, "ci reader (vga_7f21)"},
	{SSOUnlinked{ProviderRef{Slug: "okta"}}, "okta"},
	{AccountCreated{AccountAtRole{Username: "bob", Role: "viewer"}}, "bob · viewer"},
	{TOTPEnrolled{AccountRef{Username: "alice"}}, "alice"},
	{InviteMinted{InviteMint{Role: "viewer"}}, "invite · viewer"},
	{AccountRoleMoved{AccountAtRole{Username: "bob", Role: "admin"}}, "bob · admin"},
	{TOTPStripped{AccountRef{Username: "bob"}}, "bob"},
	{AccountRemoved{AccountRef{Username: "bob"}}, "bob"},
	{ChannelDeclared{ChannelRef{Endpoint: "hooks.example.com/services/T0/B0"}}, "hooks.example.com/services/T0/B0"},
	{ChannelUpdated{ChannelRef{Endpoint: "hooks.example.com/services/T0/B0"}}, "hooks.example.com/services/T0/B0"},
	{ChannelWithdrawn{ChannelRef{Endpoint: "hooks.example.com/services/T0/B0"}}, "hooks.example.com/services/T0/B0"},
	{ChannelTested{ChannelRef{Endpoint: "hooks.example.com/services/T0/B0"}}, "hooks.example.com/services/T0/B0"},
	{TranscriptCurrencySet{DialMove{Dial: "transcript currency", Value: "14 days"}}, "transcript currency · 14 days"},
	{ObservationCurrencySet{DialMove{Dial: "observation currency", Value: "30 days"}}, "observation currency · 30 days"},
	{DispatchCadenceSet{DialMove{Dial: "dispatch cadence", Value: "4"}}, "dispatch cadence · 4"},
	{AddressCapSet{DialMove{Dial: "address-scope cap", Value: "1024"}}, "address-scope cap · 1024"},
	{UpdateCheckMoved{DialMove{Dial: "update check", Value: "off"}}, "update check · off"},
	{RestoreApplied{RestoreRef{Archive: "verge-2026-09-08.tar.zst", TakenAt: "2026-09-08T14:02Z"}}, "verge-2026-09-08.tar.zst · taken 2026-09-08T14:02Z"},
	{APIAccessMoved{DialMove{Dial: "API access", Value: "on"}}, "API access · on"},
	{SSOProviderDeclared{ProviderRef{Slug: "okta"}}, "okta"},
	{SSOProviderUpdated{ProviderRef{Slug: "okta"}}, "okta"},
	{SSOProviderSecretSet{ProviderRef{Slug: "okta"}}, "okta"},
	{SSOProviderWithdrawn{ProviderRef{Slug: "okta"}}, "okta"},
	{SSOBindingRemoved{BindingRef{Slug: "okta", Username: "bob"}}, "okta · bob"},
	{IntegrationInstalled{IntegrationRef{Slug: "pagerduty"}}, "pagerduty"},
	{IntegrationRemoved{IntegrationRef{Slug: "pagerduty"}}, "pagerduty"},
	{IntegrationChannelBound{IntegrationChannel{Slug: "pagerduty", Endpoint: "events.pagerduty.com"}}, "pagerduty · events.pagerduty.com"},
	{IntegrationTested{IntegrationChannel{Slug: "pagerduty", Endpoint: "events.pagerduty.com"}}, "pagerduty · events.pagerduty.com"},
	{SSOBindingCreated{BindingRef{Slug: "okta", Username: "alice"}}, "okta · alice"},
	{TranscriptDisclosed{TranscriptRef{JobID: 9182, RunID: 412, Vantage: "edge-01"}}, "job 9182 · run 412 · edge-01"},
}

// Asserted rather than derived, so a variant deleted with its sample still fails (§2.1).

const classCount = 61

func TestClassesHoldsEveryVariantOnce(t *testing.T) {
	if len(Classes) != classCount {
		t.Fatalf("Classes holds %d variants, spec §2.1 gives %d", len(Classes), classCount)
	}
	if len(registry) != classCount {
		t.Fatalf("the registry holds %d action tokens for %d variants — two share one token",
			len(registry), classCount)
	}
	seen := map[reflect.Type]bool{}
	for _, a := range Classes {
		typ := reflect.TypeOf(a)
		if seen[typ] {
			t.Errorf("%T appears twice in Classes", a)
		}
		seen[typ] = true
	}
}

func TestSamplesCoverEveryClass(t *testing.T) {
	covered := map[string]bool{}
	for _, s := range samples {
		if covered[s.act.Class()] {
			t.Errorf("two samples for class %q", s.act.Class())
		}
		covered[s.act.Class()] = true
	}
	for _, a := range Classes {
		if !covered[a.Class()] {
			t.Errorf("no sample for class %q (%T) — it round-trips unexercised", a.Class(), a)
		}
	}
	for class := range covered {
		if _, known := registry[class]; !known {
			t.Errorf("sample class %q is not in Classes, so it cannot decode", class)
		}
	}
}

func TestEveryVariantRoundTrips(t *testing.T) {
	for _, s := range samples {
		t.Run(s.act.Class(), func(t *testing.T) {
			action, subject, err := EncodeSubject(s.act)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if action != s.act.Class() {
				t.Errorf("stored action %q, Class() %q", action, s.act.Class())
			}
			back, err := DecodeSubject(action, subject)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !reflect.DeepEqual(back, s.act) {
				t.Errorf("round-trip gave %#v, want %#v", back, s.act)
			}
			if got := back.Subject(); got != s.subject {
				t.Errorf("Subject() after round-trip = %q, want %q", got, s.subject)
			}
		})
	}
}

// The render path reads no store, so the subject may be gone (spec §4.2 hazard 1).

func TestSubjectRendersWithNoStoreRead(t *testing.T) {
	withdrawn := []struct {
		act  Act
		want string
	}{
		{SeedWithdrawn{SeedScope{Scope: "10.0.0.0/8"}}, "10.0.0.0/8"},
		{AccountRemoved{AccountRef{Username: "bob"}}, "bob"},
		{AnnotationWithdrawn{AnnotationRef{SubjectKey: "10.0.4.9:443", Signal: "expired certificate"}}, "10.0.4.9:443 · expired certificate"},
	}
	for _, w := range withdrawn {
		_, subject, err := EncodeSubject(w.act)
		if err != nil {
			t.Fatalf("%T: encode: %v", w.act, err)
		}
		back, err := DecodeSubject(w.act.Class(), subject)
		if err != nil {
			t.Fatalf("%T: decode: %v", w.act, err)
		}
		if got := back.Subject(); got != w.want {
			t.Errorf("%T rendered %q, want %q", w.act, got, w.want)
		}
	}

	// The Actor renders after the account is gone, because §5.4 captured a value.
	kind, payload, err := EncodeActor(Account{AccountID: 7, UsernameSnapshot: "alice"})
	if err != nil {
		t.Fatalf("encode actor: %v", err)
	}
	actor, err := DecodeActor(kind, payload)
	if err != nil {
		t.Fatalf("decode actor: %v", err)
	}
	if got := actor.Name(); got != "alice" {
		t.Errorf("a removed account's Actor rendered %q, want %q", got, "alice")
	}
}

// It carries the slug and nothing else, so there is no field the value fits in (§4.2 hazard 2).

func TestSecretSettingVariantCarriesOnlyTheSlug(t *testing.T) {
	typ := reflect.TypeOf(SSOProviderSecretSet{})
	fields := flattenFields(typ)
	if len(fields) != 1 {
		t.Fatalf("SSOProviderSecretSet carries %d fields, want the slug alone: %v", len(fields), fields)
	}
	if fields[0].Name != "Slug" {
		t.Errorf("SSOProviderSecretSet's one field is %q, want %q", fields[0].Name, "Slug")
	}
	_, subject, err := EncodeSubject(SSOProviderSecretSet{ProviderRef{Slug: "okta"}})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(subject, &raw); err != nil {
		t.Fatalf("the stored payload is not an object: %v", err)
	}
	if len(raw) != 1 || raw["slug"] != "okta" {
		t.Errorf("stored payload is %v, want the slug alone", raw)
	}
}

// Its subject is the whole corpus at that instant, and its Actor survives the
// TRUNCATE because §5.4 captures a value and not a join (§4.2 hazard 3).

func TestRestoreVariantRoundTrips(t *testing.T) {
	want := RestoreApplied{RestoreRef{Archive: "verge-2026-09-08.tar.zst", TakenAt: "2026-09-08T14:02Z"}}
	action, subject, err := EncodeSubject(want)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	back, err := DecodeSubject(action, subject)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(back, want) {
		t.Fatalf("round-trip gave %#v, want %#v", back, want)
	}
	if got := back.Subject(); got != "verge-2026-09-08.tar.zst · taken 2026-09-08T14:02Z" {
		t.Errorf("Subject() = %q", got)
	}
}

// A []byte, a map or an any is the hole a secret fits through (spec §4.2 hazard 2).

func TestEveryPayloadFieldIsAStringOrAnInt64(t *testing.T) {
	for _, a := range Classes {
		typ := reflect.TypeOf(a)
		for _, f := range flattenFields(typ) {
			switch f.Type.Kind() {
			case reflect.String, reflect.Int64:
			default:
				t.Errorf("%s.%s is %s; a payload field must be a string or an int64",
					typ.Name(), f.Name, f.Type)
			}
		}
	}
	// A non-scalar asserted from outside the union, so the walk itself is held.

	type hole struct {
		Slug   string
		Secret []byte
	}
	got := flattenFields(reflect.TypeOf(hole{}))
	if len(got) != 2 || got[1].Type.Kind() != reflect.Slice {
		t.Fatalf("flattenFields missed a non-scalar field: %v", got)
	}
}

// The Actor union carries the same two field kinds, for the same reason (§4.2 hazard 2).

func TestActorFieldsAreStringsOrInt64s(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(Account{}),
		reflect.TypeOf(SetupToken{}),
		reflect.TypeOf(PasswordReset{}),
		reflect.TypeOf(Invite{}),
	} {
		for _, f := range flattenFields(typ) {
			switch f.Type.Kind() {
			case reflect.String, reflect.Int64:
			default:
				t.Errorf("%s.%s is %s; an actor field must be a string or an int64",
					typ.Name(), f.Name, f.Type)
			}
		}
	}
}

func TestEveryActorVariantRoundTrips(t *testing.T) {
	cases := []struct {
		actor Actor
		kind  string
		name  string
	}{
		{Account{AccountID: 7, UsernameSnapshot: "alice"}, KindAccount, "alice"},
		{GrantHolder{Grant: SetupToken{}}, KindGrantHolder, "setup token"},
		{GrantHolder{Grant: PasswordReset{}}, KindGrantHolder, "password-reset link"},
		{GrantHolder{Grant: Invite{InviteID: 12}}, KindGrantHolder, "invite 12"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kind, payload, err := EncodeActor(c.actor)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if kind != c.kind {
				t.Errorf("actor_kind = %q, want %q", kind, c.kind)
			}
			back, err := DecodeActor(kind, payload)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if !reflect.DeepEqual(back, c.actor) {
				t.Errorf("round-trip gave %#v, want %#v", back, c.actor)
			}
			if got := back.Name(); got != c.name {
				t.Errorf("Name() = %q, want %q", got, c.name)
			}
		})
	}
}

// There is no account inside the Actor to render by mistake (spec §3.3).

func TestGrantHolderCarriesNoAccount(t *testing.T) {
	for _, g := range []Grant{SetupToken{}, PasswordReset{}, Invite{InviteID: 12}} {
		_, payload, err := EncodeActor(GrantHolder{Grant: g})
		if err != nil {
			t.Fatalf("%T: encode: %v", g, err)
		}
		if strings.Contains(strings.ToLower(string(payload)), "account") {
			t.Errorf("%T's stored actor mentions an account: %s", g, payload)
		}
	}
}

// It makes the collision with an account name structural, never enumerated (spec §3.5).

func TestNoGrantLabelBeginsWithTheMark(t *testing.T) {
	for tag, typ := range grants {
		label := reflect.New(typ).Elem().Interface().(Grant).Label()
		if label == "" {
			t.Errorf("grant %q renders an empty label", tag)
		}
		if strings.HasPrefix(label, "@") {
			t.Errorf("grant %q renders %q, which collides with a marked account name", tag, label)
		}
	}
}

// The mark lives in a renderer and lands with the reader, never here (spec §3.5 fact 1).

func TestTheStoreKeepsTheBareUsername(t *testing.T) {
	_, payload, err := EncodeActor(Account{AccountID: 7, UsernameSnapshot: "alice"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var raw struct {
		UsernameSnapshot string `json:"username_snapshot"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if raw.UsernameSnapshot != "alice" {
		t.Errorf("stored username_snapshot = %q, want %q", raw.UsernameSnapshot, "alice")
	}
}

func TestDecoderRefusesAnUnknownToken(t *testing.T) {
	if _, err := DecodeSubject("seed.evaporated", []byte(`{}`)); err == nil {
		t.Error("DecodeSubject accepted an action token no variant claims")
	}
	if _, err := DecodeActor("system", []byte(`{}`)); err == nil {
		t.Error("DecodeActor accepted 'system', which §3.4 dropped")
	}
	if err := (&GrantHolder{}).UnmarshalJSON([]byte(`{"kind":"api_token"}`)); err == nil {
		t.Error("a GrantHolder decoded an unknown grant kind")
	}
}

// It walks the embedded payload, because a marker type declares no field of its own.

func flattenFields(t reflect.Type) []reflect.StructField {
	var out []reflect.StructField
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			out = append(out, flattenFields(f.Type)...)
			continue
		}
		out = append(out, f)
	}
	return out
}
