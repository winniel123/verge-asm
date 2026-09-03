// Package corpus is the golden corpus for the resolution-walk leaf
// (docs/spec/golden-corpus.md §2). Every row runs hermetically against an
// in-process scripted peer — no network, no containers, no fixture images.
package corpus

import "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"

type scriptRule struct {
	Path      resolutionwalk.Path
	Server    string
	Name      string
	Qtype     resolutionwalk.Qtype
	Transport resolutionwalk.Transport
	EDNS      *bool // a pointer so the M2.i FORMERR boundary can say "only with an OPT"
	Reply     resolutionwalk.Msg
}

type ScriptPeer struct {
	Rules []scriptRule
}

func (s ScriptPeer) Exchange(q resolutionwalk.Query) resolutionwalk.Msg {
	for _, r := range s.Rules {
		if r.matches(q) {
			return r.Reply
		}
	}
	// A missing rule must fail as our own blindness, never as a wrong value.
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
