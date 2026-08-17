package resolutionwalk

import (
	"net/netip"
	"sort"
	"strings"
)

// Path names which of the leaf's two queries is being made (ADR-0070). The
// declared path is the Vantage's configured recursive resolver and decides
// Resolved/NoData/NameError; the delegation walk goes direct to the delegated
// authorities and decides Lame and the per-nameserver serves/does-not-serve
// RRset. The query-path parameter governs the declared path and never the walk.
type Path string

const (
	PathDeclared Path = "declared"
	PathWalk     Path = "walk"
)

// Transport is the wire transport for one exchange.
type Transport string

const (
	UDP Transport = "udp"
	TCP Transport = "tcp"
)

// Rcode is the DNS response code, by name, since the leaf reads only the
// spec-defined codes it discriminates on and never a numeric default.
type Rcode string

const (
	NOERROR  Rcode = "NOERROR"
	NXDOMAIN Rcode = "NXDOMAIN"
	FORMERR  Rcode = "FORMERR"
	REFUSED  Rcode = "REFUSED"
	SERVFAIL Rcode = "SERVFAIL"
)

// RR is one resource record in an answer. Data is the record's canonical
// payload: a dotted-quad or RFC 5952 address for A/AAAA, a name for CNAME/NS,
// opaque text otherwise. The leaf reads Data only for the record types it folds.
type RR struct {
	Name  string `json:"name"`
	Type  Qtype  `json:"type"`
	Data  string `json:"data"`
}

// Query is one exchange the leaf asks of a Peer. It carries the offers actually
// on the wire for this attempt (transport, whether an OPT is present, whether a
// cookie is sent) so a scripted peer can answer the truncation and FORMERR-on-OPT
// boundaries faithfully.
type Query struct {
	Path      Path
	Server    string // the delegated authority for a walk; the resolver otherwise
	Name      string
	Qtype     Qtype
	Transport Transport
	EDNS      bool
	Cookie    bool
}

// Msg is a decoded DNS response. Reached is false when nothing answered (the
// silent authority of boundary M2.e), which is distinct from a REFUSED answer.
// Unreachable is stronger still: the exchange failed at the socket — a dial or
// read error — so we could not look at all, which is "we could not look" rather
// than any value the resolver returned (ADR-0108). It aborts the batch only on
// the declared path; a walk-path authority's silence stays the Gap/Lame
// vocabulary.
type Msg struct {
	Reached     bool
	Rcode       Rcode
	Truncated   bool
	Unreachable bool
	Answer      []RR
}

// Peer answers the leaf's queries. The production adapter dials real
// authorities and resolvers; the golden corpus scripts an in-process peer, so
// the leaf's logic is exercised hermetically with no network and no containers.
type Peer interface {
	Exchange(q Query) Msg
}

// Outcome is the resolution value's tag. Shadowed is deliberately absent: it is
// wildcard-discrimination's outcome, decided by a different leaf, and this leaf
// never emits it (golden-corpus.md §1).
type Outcome string

const (
	OutcomeResolved  Outcome = "Resolved"
	OutcomeNoData    Outcome = "NoData"
	OutcomeNameError Outcome = "NameError"
	OutcomeGap       Outcome = "Gap"
)

// Resolution is the declared-path outcome for one Name.
type Resolution struct {
	Outcome   Outcome  `json:"outcome"`
	Addresses []string `json:"addresses,omitempty"`
}

// NSStatus is one authority's serves/does-not-serve verdict on the delegation
// walk. It is what a partly-lame delegation records instead of Lame (M1.5).
type NSStatus struct {
	Server string `json:"server"`
	Serves bool   `json:"serves"`
}

// Delegation is the walk's output. Lame is true only when every reached
// authority refused; Gap is true when no authority was reached at all (M2.e).
// Neither is a value the query-path parameter governs.
type Delegation struct {
	Lame        bool       `json:"lame"`
	Gap         bool       `json:"gap"`
	Nameservers []NSStatus `json:"nameservers,omitempty"`
}

// Record is one dns-record observation, per queried qtype, from the declared
// path. The qtype is the facet's discriminator (ADR-0011).
type Record struct {
	Qtype Qtype `json:"qtype"`
	RRs   []RR  `json:"rrs"`
}

// Result is the leaf's complete decision for one Name at one Vantage. It names
// no transition (golden-corpus.md R.1): the leaf emits outcomes, and whether a
// subject appeared, withdrew or returned is decided downstream from them.
type Result struct {
	Name       string     `json:"name"`
	Resolution Resolution `json:"resolution"`
	Delegation Delegation `json:"delegation"`
	Records    []Record   `json:"records"`

	// Unreachable reports that the declared-path resolver could not be reached
	// for this Name — a socket failure, not a value. It is a control signal read
	// by RunWithPeer to abort the whole batch (the resolver is one position for
	// every name), never an emitted observation, so it does not serialize and no
	// golden corpus moves (ADR-0108).
	Unreachable bool `json:"-"`
}

// Resolve runs the leaf for one Name against a Peer under the given Offers. It
// makes the declared-path queries over the qtype set, folds A/AAAA into an
// address set, and runs the delegation walk. It never converts a transport-level
// refusal into a zone-level value, and it never folds a partial RRset — a
// truncated answer that the fallback does not recover is a Gap (ADR-0025).
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
			// The resolver itself could not be reached. It is one position for
			// the whole batch, so there is nothing more to ask: mark the result
			// so RunWithPeer aborts the batch rather than folding an all-Gap
			// measurement (ADR-0108).
			res.Unreachable = true
			return res
		}
		rec := Record{Qtype: qt, RRs: normaliseRRs(msg.Answer)}
		res.Records = append(res.Records, rec)

		if !ok {
			// Neither an answer nor a recovery: coverage we could not obtain,
			// so the pair is absent from what we could say. It does not make
			// the name non-existent — nxAll stays honest by not being set here.
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

// decideResolution collapses the per-qtype declared-path answers into the one
// resolution outcome. The ordering is deliberate:
//   - a recovered address set is Resolved (M1.1; the CNAME-chain side of M2.c);
//   - a name nothing answered for is a Gap, never a withdrawal;
//   - an all-NXDOMAIN answer withdraws the Name (M1.3) — unless it carried a
//     CNAME, in which case the rcode reflects the final name and the alias
//     itself exists, so the queried name survives as NoData (M2.b non-withdraw
//     side, RFC 6604);
//   - an existing name (NOERROR) with no address is NoData (M1.2; empty
//     non-terminal M2.a; the no-address side of M2.c);
//   - reached but with no usable answer is a Gap (M2.i Gap side).
func decideResolution(addrs addrSet, nxAll, anyNoError, anyReached, sawCNAME bool) Resolution {
	if addrs.len() > 0 {
		return Resolution{Outcome: OutcomeResolved, Addresses: addrs.sorted()}
	}
	if !anyReached {
		return Resolution{Outcome: OutcomeGap}
	}
	if nxAll {
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

// exchangeDeclared makes one declared-path query, applying the transport and
// fallback policy: retry over TCP when the answer is truncated, and retry
// without an OPT when an authority FORMERRs an EDNS query. The bool is false
// when neither UDP, TCP fallback nor the EDNS-less retry produced an answer.
// The third bool reports that the resolver could not be reached at the socket
// (ADR-0108) — distinct from the second (ok), which is false for a reached-but-
// unusable answer (a Gap). Only the first governs whether the batch aborts.
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

	// FORMERR to an OPT-carrying query: retry once without EDNS (M2.i).
	if msg.Reached && msg.Rcode == FORMERR && q.EDNS && offers.Transport.EDNSlessRetry {
		q.EDNS = false
		msg = peer.Exchange(q)
		if msg.Unreachable {
			return Msg{}, false, true
		}
	}

	// Truncation: a truncated answer is never a value. Retry over TCP (M2.d).
	if msg.Reached && msg.Truncated && offers.Transport.FallbackOnTC && offers.Transport.TCPAttempts > 0 {
		tq := q
		tq.Transport = TCP
		msg = peer.Exchange(tq)
		if msg.Unreachable {
			return Msg{}, false, true
		}
		if msg.Truncated {
			// TCP could not recover the RRset: a Gap, not a partial fold.
			return Msg{}, false, false
		}
	}

	if !msg.Reached {
		return Msg{}, false, false
	}
	// A transport-level failure is not a value; treat SERVFAIL with no answer as
	// coverage we did not obtain rather than a resolution outcome.
	if msg.Rcode == SERVFAIL && len(msg.Answer) == 0 {
		return Msg{}, false, false
	}
	return msg, true, false
}

// walk runs the delegation walk direct to the delegated authorities, deciding
// Lame and the per-nameserver serves/does-not-serve RRset. A transport-level
// refusal is never converted into a zone-level value: an authority that was
// reached and REFUSED does-not-serve, but an authority that was not reached is
// our own blindness and yields a Gap where every authority is silent (M2.e/f).
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
		// No delegation information at all: nothing to walk.
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
		// Every delegated authority was silent: a Gap, our own blindness (M2.e).
		return Delegation{Gap: true, Nameservers: statuses}
	case allRefused && !anyServes:
		// Reached and refused everywhere: Lame (M1.4).
		return Delegation{Lame: true, Nameservers: statuses}
	default:
		// At least one serves: not Lame, carrying the per-nameserver RRset (M1.5/M2.f).
		return Delegation{Nameservers: statuses}
	}
}

// CanonicalName renders a Name key: labels lower-cased over ASCII only, joined
// by dots, no trailing dot (CONTEXT.md `Name`; ADR-0055). It folds exactly what
// the protocol folds — the 26 ASCII letters — which is what makes the 0x20
// case-randomisation boundary (M2.g) a no-op on the key.
func CanonicalName(name string) string {
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

// addrSet folds A and AAAA answers into Address keys — family and octets, never
// a spelling — so an IPv4-mapped AAAA and its A fold to one member (M2.h) and RR
// order does not move the set (M2.g).
type addrSet struct {
	seen map[netip.Addr]struct{}
}

func (s *addrSet) add(text string) {
	addr, err := netip.ParseAddr(text)
	if err != nil {
		return
	}
	addr = addr.Unmap() // IPv4-mapped IPv6 keys as the IPv4 address it represents
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

// normaliseRRs canonicalises the record names and folds address spellings so a
// dns-record observation is stable under RR order and case. Address RDATA is
// rendered from its Address key; other RDATA is passed through with a canonical
// owner name.
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
