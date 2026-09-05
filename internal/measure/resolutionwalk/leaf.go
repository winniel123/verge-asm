package resolutionwalk

import (
	"net/netip"
	"sort"
	"strings"
)

type Path string

const (
	PathDeclared Path = "declared"
	PathWalk     Path = "walk"
)

type Transport string

const (
	UDP Transport = "udp"
	TCP Transport = "tcp"
)

type Rcode string

const (
	NOERROR  Rcode = "NOERROR"
	NXDOMAIN Rcode = "NXDOMAIN"
	FORMERR  Rcode = "FORMERR"
	REFUSED  Rcode = "REFUSED"
	SERVFAIL Rcode = "SERVFAIL"
	OTHER    Rcode = "OTHER"
)

type RR struct {
	Name string `json:"name"`
	Type Qtype  `json:"type"`
	Data string `json:"data"` // RFC 5952 for A/AAAA, a canonical name for CNAME/NS
}

type Query struct {
	Path      Path
	Server    string
	Name      string
	Qtype     Qtype
	Transport Transport
	EDNS      bool
	Cookie    bool
}

type Msg struct {
	Reached     bool
	Rcode       Rcode
	Truncated   bool
	Unreachable bool
	Answer      []RR
}

type Peer interface {
	Exchange(q Query) Msg
}

type Outcome string

const (
	OutcomeResolved  Outcome = "Resolved"
	OutcomeNoData    Outcome = "NoData"
	OutcomeNameError Outcome = "NameError"
	OutcomeGap       Outcome = "Gap"
)

type Resolution struct {
	Outcome   Outcome  `json:"outcome"`
	Addresses []string `json:"addresses,omitempty"`
}

type NSStatus struct {
	Server string `json:"server"`
	Serves bool   `json:"serves"`
}

type Delegation struct {
	Lame        bool       `json:"lame"`
	Gap         bool       `json:"gap"`
	Nameservers []NSStatus `json:"nameservers,omitempty"`
}

type Record struct {
	Qtype Qtype `json:"qtype"`
	RRs   []RR  `json:"rrs"`
}

type Result struct {
	Name       string     `json:"name"`
	Resolution Resolution `json:"resolution"`
	Delegation Delegation `json:"delegation"`
	Records    []Record   `json:"records"`

	Unreachable bool `json:"-"`
}

func Resolve(peer Peer, offers Offers, name string) Result {
	key := CanonicalName(name)
	res := Result{Name: key}

	var addrs addrSet
	nxAll := true
	anyReached := false
	anyNoError := false
	sawCNAME := false

	for _, qt := range offers.Qtypes {
		msg, ok, unreachable := exchangeDeclared(peer, offers, key, qt)
		if unreachable {
			// The resolver is one position for the whole batch, so an all-Gap fold is wrong (ADR-0108).
			res.Unreachable = true
			return res
		}
		rec := Record{Qtype: qt, RRs: normaliseRRs(msg.Answer)}
		res.Records = append(res.Records, rec)

		if !ok {
			// Absent coverage must not set nxAll, or an unanswered name reads as withdrawn.
			continue
		}
		anyReached = true
		if msg.Rcode != NXDOMAIN {
			nxAll = false
		}
		if msg.Rcode == NOERROR {
			anyNoError = true
		}
		for _, rr := range msg.Answer {
			switch rr.Type {
			case QtypeA, QtypeAAAA:
				addrs.add(rr.Data)
			case QtypeCNAME:
				sawCNAME = true
			}
		}
	}

	res.Resolution = decideResolution(addrs, nxAll, anyNoError, anyReached, sawCNAME)
	res.Delegation = walk(peer, offers, key)
	return res
}

func decideResolution(addrs addrSet, nxAll, anyNoError, anyReached, sawCNAME bool) Resolution {
	// Shadowed is wildcard-discrimination's outcome and never this leaf's (golden-corpus.md §1).
	if addrs.len() > 0 {
		return Resolution{Outcome: OutcomeResolved, Addresses: addrs.sorted()}
	}
	if !anyReached {
		return Resolution{Outcome: OutcomeGap}
	}
	if nxAll {
		// NXDOMAIN reflects the final name, so the alias itself survives (RFC 6604, M2.b).
		if sawCNAME {
			return Resolution{Outcome: OutcomeNoData}
		}
		return Resolution{Outcome: OutcomeNameError}
	}
	if anyNoError {
		return Resolution{Outcome: OutcomeNoData}
	}
	return Resolution{Outcome: OutcomeGap}
}

func exchangeDeclared(peer Peer, offers Offers, name string, qt Qtype) (msg Msg, ok, unreachable bool) {
	q := Query{
		Path:      PathDeclared,
		Name:      name,
		Qtype:     qt,
		Transport: UDP,
		EDNS:      true,
		Cookie:    offers.EDNS.Cookie,
	}
	msg = peer.Exchange(q)
	if msg.Unreachable {
		return Msg{}, false, true
	}

	if msg.Reached && msg.Rcode == FORMERR && q.EDNS && offers.Transport.EDNSlessRetry {
		q.EDNS = false
		msg = peer.Exchange(q)
		if msg.Unreachable {
			return Msg{}, false, true
		}
	}

	// A truncated answer is never a value; the Gap → value edge is not revealed (ADR-0014).
	if msg.Reached && msg.Truncated && offers.Transport.FallbackOnTC && offers.Transport.TCPAttempts > 0 {
		tq := q
		tq.Transport = TCP
		msg = peer.Exchange(tq)
		if msg.Unreachable {
			return Msg{}, false, true
		}
		if msg.Truncated {
			return Msg{}, false, false
		}
	}

	if !msg.Reached {
		return Msg{}, false, false
	}
	// A transport-level failure is coverage we did not obtain, never a value.
	if msg.Rcode == SERVFAIL && len(msg.Answer) == 0 {
		return Msg{}, false, false
	}
	return msg, true, false
}

func walk(peer Peer, offers Offers, name string) Delegation {
	msg := peer.Exchange(Query{
		Path:      PathWalk,
		Name:      name,
		Qtype:     QtypeNS,
		Transport: UDP,
		EDNS:      true,
		Cookie:    offers.EDNS.Cookie,
	})
	if len(msg.Answer) == 0 && !msg.Reached {
		return Delegation{}
	}

	var statuses []NSStatus
	anyReached := false
	anyServes := false
	allRefused := true
	for _, rr := range msg.Answer {
		if rr.Type != QtypeNS {
			continue
		}
		// A walk authority must reach the guard with a non-empty Server, or it is exempted (#324).
		ns := peer.Exchange(Query{
			Path:      PathWalk,
			Server:    rr.Data,
			Name:      name,
			Qtype:     QtypeSOA,
			Transport: UDP,
			EDNS:      true,
			Cookie:    offers.EDNS.Cookie,
		})
		serves := ns.Reached && ns.Rcode != REFUSED && ns.Rcode != SERVFAIL
		if ns.Reached {
			anyReached = true
			if serves {
				allRefused = false
				anyServes = true
			}
		}
		statuses = append(statuses, NSStatus{Server: rr.Data, Serves: serves})
	}

	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Server < statuses[j].Server })

	switch {
	case len(statuses) == 0:
		return Delegation{}
	case !anyReached:
		// Silence is our own blindness, never a zone-level value (M2.e).
		return Delegation{Gap: true, Nameservers: statuses}
	case allRefused && !anyServes:
		return Delegation{Lame: true, Nameservers: statuses}
	default:
		return Delegation{Nameservers: statuses}
	}
}

func CanonicalName(name string) string {
	// ASCII-only folding is exactly what DNS folds, so 0x20 randomisation is a no-op (ADR-0055).
	trimmed := strings.TrimSuffix(name, ".")
	return asciiLower(trimmed)
}

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

type addrSet struct {
	seen map[netip.Addr]struct{}
}

func (s *addrSet) add(text string) {
	addr, err := netip.ParseAddr(text)
	if err != nil {
		return
	}
	addr = addr.Unmap()
	if s.seen == nil {
		s.seen = make(map[netip.Addr]struct{})
	}
	s.seen[addr] = struct{}{}
}

func (s *addrSet) len() int { return len(s.seen) }

func (s *addrSet) sorted() []string {
	out := make([]netip.Addr, 0, len(s.seen))
	for a := range s.seen {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	rendered := make([]string, len(out))
	for i, a := range out {
		rendered[i] = a.String()
	}
	return rendered
}

func normaliseRRs(in []RR) []RR {
	if len(in) == 0 {
		return nil
	}
	out := make([]RR, 0, len(in))
	for _, rr := range in {
		norm := RR{Name: CanonicalName(rr.Name), Type: rr.Type, Data: rr.Data}
		if rr.Type == QtypeA || rr.Type == QtypeAAAA {
			if addr, err := netip.ParseAddr(rr.Data); err == nil {
				norm.Data = addr.Unmap().String()
			}
		} else if rr.Type == QtypeCNAME || rr.Type == QtypeNS {
			norm.Data = CanonicalName(rr.Data)
		}
		out = append(out, norm)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		return out[i].Data < out[j].Data
	})
	return out
}
