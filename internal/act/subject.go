package act

import "strconv"

// A field is a string or an int64: a []byte, a map or an any is the hole a secret fits (§4.2).

type AccountRef struct {
	Username string `json:"username"`
}

func (p AccountRef) Subject() string { return p.Username }

type AccountAtRole struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

func (p AccountAtRole) Subject() string { return p.Username + sep + p.Role }

type ScanProfile struct {
	Profile string `json:"profile"`
}

func (p ScanProfile) Subject() string { return p.Profile }

// A confirmed proposal shares it, because what confirmation declares is the scope (§2.1).

type SeedScope struct {
	Scope string `json:"scope"`
}

func (p SeedScope) Subject() string { return p.Scope }

type CustodyMove struct {
	Scope       string `json:"scope"`
	Disposition string `json:"disposition"`
}

func (p CustodyMove) Subject() string { return p.Scope + sep + p.Disposition }

type ZoneRef struct {
	Zone string `json:"zone"`
}

func (p ZoneRef) Subject() string { return p.Zone }

// The cell carries the dial name, so several classes share the Action label Dial moved (§2.1).

type DialMove struct {
	Dial  string `json:"dial"`
	Value string `json:"value"`
}

func (p DialMove) Subject() string { return p.Dial + sep + p.Value }

// A declined proposal shares it, because a decline is stored as an exclusion row (#1721).

type ExclusionRef struct {
	Kind  string `json:"kind"`
	Scope string `json:"scope"`
}

func (p ExclusionRef) Subject() string { return p.Kind + " " + p.Scope }

type ColdMove struct {
	Scope       string `json:"scope"`
	Disposition string `json:"disposition"`
}

func (p ColdMove) Subject() string { return p.Scope + sep + p.Disposition }

type VantageRef struct {
	Endpoint string `json:"endpoint"`
}

func (p VantageRef) Subject() string { return p.Endpoint }

type ResolverRef struct {
	Resolver string `json:"resolver"`
}

func (p ResolverRef) Subject() string { return p.Resolver }

type ScheduleRef struct {
	Name string `json:"name"`
}

func (p ScheduleRef) Subject() string { return p.Name }

// It carries no prose, because prose is the one reopening condition for #127 §5 (§8 · D.6).

type AnnotationRef struct {
	SubjectKey string `json:"subject_key"`
	Signal     string `json:"signal"`
}

func (p AnnotationRef) Subject() string { return p.SubjectKey + sep + p.Signal }

type OrgQuery struct {
	Term string `json:"term"`
}

func (p OrgQuery) Subject() string { return p.Term }

type DispatchRef struct {
	DispatchID int64  `json:"dispatch_id"`
	Profile    string `json:"profile"`
}

func (p DispatchRef) Subject() string {
	return "dispatch " + strconv.FormatInt(p.DispatchID, 10) + sep + p.Profile
}

type FrequencyMove struct {
	Port        string `json:"port"`
	Disposition string `json:"disposition"`
}

func (p FrequencyMove) Subject() string { return "port " + p.Port + sep + p.Disposition }

type SourceMove struct {
	Slug        string `json:"slug"`
	Disposition string `json:"disposition"`
}

func (p SourceMove) Subject() string { return p.Slug + sep + p.Disposition }

// The public prefix and never the token value: a secret being set is auditable, its value is not.

type TokenRef struct {
	Label  string `json:"label"`
	Prefix string `json:"prefix"`
}

func (p TokenRef) Subject() string { return p.Label + " (" + p.Prefix + ")" }

// The secret-setting variant carries this alone, so there is no field the value fits in (§4.2).

type ProviderRef struct {
	Slug string `json:"slug"`
}

func (p ProviderRef) Subject() string { return p.Slug }

// It names the role and no account, because the mint creates none (§2.1).

type InviteMint struct {
	Role string `json:"role"`
}

func (p InviteMint) Subject() string { return "invite" + sep + p.Role }

type ChannelRef struct {
	Endpoint string `json:"endpoint"`
}

func (p ChannelRef) Subject() string { return p.Endpoint }

// The subject is the whole corpus at that instant, written after the truncation (§5.3).

type RestoreRef struct {
	Archive string `json:"archive"`
	TakenAt string `json:"taken_at"`
}

func (p RestoreRef) Subject() string { return p.Archive + sep + "taken " + p.TakenAt }

type BindingRef struct {
	Slug     string `json:"slug"`
	Username string `json:"username"`
}

func (p BindingRef) Subject() string { return p.Slug + sep + p.Username }

type IntegrationRef struct {
	Slug string `json:"slug"`
}

func (p IntegrationRef) Subject() string { return p.Slug }

type IntegrationChannel struct {
	Slug     string `json:"slug"`
	Endpoint string `json:"endpoint"`
}

func (p IntegrationChannel) Subject() string { return p.Slug + sep + p.Endpoint }

// Three values and not a job id: the transcript expires and a bare id then dangles (§4.3).

type TranscriptRef struct {
	JobID   int64  `json:"job_id"`
	RunID   int64  `json:"run_id"`
	Vantage string `json:"vantage"`
}

func (p TranscriptRef) Subject() string {
	return "job " + strconv.FormatInt(p.JobID, 10) +
		sep + "run " + strconv.FormatInt(p.RunID, 10) +
		sep + p.Vantage
}
