package main

// PROTOTYPE — throwaway. Answers #1790 for map #1786. Not production code.

import "fmt"

// The subject payloads. Every field is a value captured at write time, never an
// id to join on: a join renders blank exactly on the acts most worth reading
// (map #1786 Settled #8).
//
// Every field is a string or an int64 on purpose. A variant with no field for a
// secret cannot carry one, which is how ADR-0053's split becomes inexpressible
// rather than discouraged. hazard_test.go holds that line.

type SeedScope struct {
	Scope string `json:"scope"`
}

type ScopeMove struct {
	Scope string `json:"scope"`
	Move  string `json:"move"`
}

type ZoneName struct {
	Zone string `json:"zone"`
}

type ExclusionTerm struct {
	ExclusionKind string `json:"exclusion_kind"`
	Value         string `json:"value"`
}

type DialMove struct {
	Dial  string `json:"dial"`
	Value string `json:"value"`
}

type VantageRef struct {
	Vantage string `json:"vantage"`
}

type ScheduleRef struct {
	Schedule string `json:"schedule"`
}

type AnnotationRef struct {
	SubjectKey string `json:"subject_key"`
	Signal     string `json:"signal"`
}

type ProposalQuery struct {
	Org string `json:"org"`
}

type ScanKindRef struct {
	ScanKind string `json:"scan_kind"`
}

type DispatchRef struct {
	Dispatch string `json:"dispatch"`
	ScanKind string `json:"scan_kind"`
}

type PortMove struct {
	Port string `json:"port"`
	Move string `json:"move"`
}

type SourceMove struct {
	Slug  string `json:"slug"`
	State string `json:"state"`
}

type AccountRef struct {
	AccountID int64  `json:"account_id"`
	Username  string `json:"username"`
}

type AccountAtRole struct {
	AccountID int64  `json:"account_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
}

type TokenRef struct {
	TokenName string `json:"token_name"`
	Prefix    string `json:"prefix"`
}

type ProviderRef struct {
	Slug string `json:"slug"`
}

type BindingRef struct {
	Slug          string `json:"slug"`
	BoundUsername string `json:"bound_username"`
}

type ChannelRef struct {
	Endpoint string `json:"endpoint"`
}

type IntegrationRef struct {
	Slug string `json:"slug"`
}

type IntegrationChannelRef struct {
	Slug     string `json:"slug"`
	Endpoint string `json:"endpoint"`
}

type RoleRef struct {
	Role string `json:"role"`
}

type ArchiveRef struct {
	Filename string `json:"filename"`
	TakenAt  string `json:"taken_at"`
}

type ToggleMove struct {
	Toggle string `json:"toggle"`
	State  string `json:"state"`
}

// JobRef carries the resolved job, run and vantage, because the Transcript it
// names self-destructs on ADR-0126's dial while the Act never does (#1794).
type JobRef struct {
	Job     string `json:"job"`
	Run     string `json:"run"`
	Vantage string `json:"vantage"`
}

type MigrationRef struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

func (s SeedScope) Subject() string      { return s.Scope }
func (s ScopeMove) Subject() string      { return s.Scope + " · " + s.Move }
func (s ZoneName) Subject() string       { return s.Zone }
func (s ExclusionTerm) Subject() string  { return s.ExclusionKind + " " + s.Value }
func (s DialMove) Subject() string       { return s.Dial + " · " + s.Value }
func (s VantageRef) Subject() string     { return s.Vantage }
func (s ScheduleRef) Subject() string    { return s.Schedule }
func (s AnnotationRef) Subject() string  { return s.SubjectKey + " · " + s.Signal }
func (s ProposalQuery) Subject() string  { return s.Org }
func (s ScanKindRef) Subject() string    { return s.ScanKind }
func (s DispatchRef) Subject() string    { return s.Dispatch + " · " + s.ScanKind }
func (s PortMove) Subject() string       { return "port " + s.Port + " · " + s.Move }
func (s SourceMove) Subject() string     { return s.Slug + " · " + s.State }
func (s AccountRef) Subject() string     { return s.Username }
func (s AccountAtRole) Subject() string  { return s.Username + " · " + s.Role }
func (s TokenRef) Subject() string       { return fmt.Sprintf("%s (%s)", s.TokenName, s.Prefix) }
func (s ProviderRef) Subject() string    { return s.Slug }
func (s BindingRef) Subject() string     { return s.Slug + " · " + s.BoundUsername }
func (s ChannelRef) Subject() string     { return s.Endpoint }
func (s IntegrationRef) Subject() string { return s.Slug }
func (s IntegrationChannelRef) Subject() string {
	if s.Endpoint == "" {
		return s.Slug + " · unbound"
	}
	return s.Slug + " · " + s.Endpoint
}
func (s RoleRef) Subject() string    { return "invite · " + s.Role }
func (s ArchiveRef) Subject() string { return s.Filename + " · taken " + s.TakenAt }
func (s ToggleMove) Subject() string { return s.Toggle + " · " + s.State }
func (s JobRef) Subject() string {
	return fmt.Sprintf("job %s · run %s · %s", s.Job, s.Run, s.Vantage)
}
func (s MigrationRef) Subject() string { return s.Version + " " + s.Name }
