// Package corpus is the golden corpus for the resolution-walk leaf
// (docs/spec/golden-corpus.md §2). It holds a checked-in enumeration of the
// cells the leaf owes a pinning row, each row being a (job-spec fragment,
// authored peer script, expected NDJSON) triple run hermetically against an
// in-process scripted peer — no network, no containers, no fixture images.
//
// Every subsequent leaf ticket adds rows here rather than building its own
// harness. The bidirectional CI gate (A2/A5/A6) lives in harness_test.go and in
// the checked-in corpus.lock.json.
package corpus

import "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"

// scriptRule matches a subset of a Query's fields and returns a scripted reply.
// A zero field matches any value; EDNS is a *bool so "either" and "only with an
// OPT" are both expressible (the M2.i FORMERR boundary needs the distinction).
type scriptRule struct {
	Path      resolutionwalk.Path
	Server    string
	Name      string
	Qtype     resolutionwalk.Qtype
	Transport resolutionwalk.Transport
	EDNS      *bool
	Reply     resolutionwalk.Msg
}

// ScriptPeer is a deterministic, in-process Peer built from ordered rules. It is
// the authored peer script of a corpus row: the first matching rule answers, and
// an unmatched query returns a silent (not-reached) response so a missing rule
// fails loudly as our own blindness rather than as a wrong value.
type ScriptPeer struct {
	Rules []scriptRule
}

func (s ScriptPeer) Exchange(q resolutionwalk.Query) resolutionwalk.Msg {
	for _, r := range s.Rules {
		if r.matches(q) {
			return r.Reply
		}
	}
	return resolutionwalk.Msg{Reached: false}
}

func (r scriptRule) matches(q resolutionwalk.Query) bool {
	if r.Path != "" && r.Path != q.Path {
		return false
	}
	if r.Server != "" && r.Server != q.Server {
		return false
	}
	if r.Name != "" && resolutionwalk.CanonicalName(r.Name) != resolutionwalk.CanonicalName(q.Name) {
		return false
	}
	if r.Qtype != "" && r.Qtype != q.Qtype {
		return false
	}
	if r.Transport != "" && r.Transport != q.Transport {
		return false
	}
	if r.EDNS != nil && *r.EDNS != q.EDNS {
		return false
	}
	return true
}

// --- reply constructors, so a row's script reads like a zone answer ---

func boolPtr(b bool) *bool { return &b }

func noerror(answers ...resolutionwalk.RR) resolutionwalk.Msg {
	return resolutionwalk.Msg{Reached: true, Rcode: resolutionwalk.NOERROR, Answer: answers}
}

func nxdomain(answers ...resolutionwalk.RR) resolutionwalk.Msg {
	return resolutionwalk.Msg{Reached: true, Rcode: resolutionwalk.NXDOMAIN, Answer: answers}
}

func refused() resolutionwalk.Msg {
	return resolutionwalk.Msg{Reached: true, Rcode: resolutionwalk.REFUSED}
}

func formerr() resolutionwalk.Msg {
	return resolutionwalk.Msg{Reached: true, Rcode: resolutionwalk.FORMERR}
}

func silent() resolutionwalk.Msg { return resolutionwalk.Msg{Reached: false} }

func truncated() resolutionwalk.Msg {
	return resolutionwalk.Msg{Reached: true, Rcode: resolutionwalk.NOERROR, Truncated: true}
}

func rrA(name, ip string) resolutionwalk.RR {
	return resolutionwalk.RR{Name: name, Type: resolutionwalk.QtypeA, Data: ip}
}
func rrAAAA(name, ip string) resolutionwalk.RR {
	return resolutionwalk.RR{Name: name, Type: resolutionwalk.QtypeAAAA, Data: ip}
}
func rrCNAME(name, target string) resolutionwalk.RR {
	return resolutionwalk.RR{Name: name, Type: resolutionwalk.QtypeCNAME, Data: target}
}
func rrNS(name, ns string) resolutionwalk.RR {
	return resolutionwalk.RR{Name: name, Type: resolutionwalk.QtypeNS, Data: ns}
}
