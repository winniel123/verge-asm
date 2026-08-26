package resolutionwalk

import (
	"net"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// loopbackDNSServer starts an in-process UDP DNS server bound to 127.0.0.1 and
// returns its "127.0.0.1:<port>" address. It answers any A query NOERROR with
// one A record and any NS query NOERROR with one NS record, so an exchange that
// reaches it parses a real response (Reached=true). Its whole point is that its
// address is a loopback one — the address the #335 custody backstop refuses —
// so a NetPeer that reaches it proves the declared-resolver dial was not gated.
func loopbackDNSServer(t *testing.T) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start loopback DNS server: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close() })

	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if resp, ok := buildLoopbackResponse(buf[:n]); ok {
				_, _ = pc.WriteTo(resp, from)
			}
		}
	}()
	return pc.LocalAddr().String()
}

func buildLoopbackResponse(query []byte) ([]byte, bool) {
	var p dnsmessage.Parser
	hdr, err := p.Start(query)
	if err != nil {
		return nil, false
	}
	q, err := p.Question()
	if err != nil {
		return nil, false
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: hdr.ID, Response: true, RecursionAvailable: true})
	if err := b.StartQuestions(); err != nil {
		return nil, false
	}
	if err := b.Question(q); err != nil {
		return nil, false
	}
	if err := b.StartAnswers(); err != nil {
		return nil, false
	}
	switch q.Type {
	case dnsmessage.TypeA:
		_ = b.AResource(
			dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET},
			dnsmessage.AResource{A: [4]byte{203, 0, 113, 5}},
		)
	case dnsmessage.TypeNS:
		_ = b.NSResource(
			dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeNS, Class: dnsmessage.ClassINET},
			dnsmessage.NSResource{NS: dnsmessage.MustNewName("ns1.example.com.")},
		)
	}
	out, err := b.Finish()
	if err != nil {
		return nil, false
	}
	return out, true
}

// TestExchangeDeclaredResolverOnLoopbackIsDialed is the #612 regression: the
// Vantage's operator-declared recursive resolver is trusted configuration
// (ADR-0070), supplied out of band and never derived from attacker-controlled
// RDATA, so a legitimate deployment may point it at a loopback address — Docker's
// embedded DNS 127.0.0.11 on the docs' compose deployment (ADR-0036). The #335
// rebinding backstop, which exists to stop DISCOVERED walk authorities reaching
// internal addresses, must not refuse the declared-resolver dial. Before the fix
// the Control hook refused the loopback socket and every default install's dns
// scan dead-lettered.
func TestExchangeDeclaredResolverOnLoopbackIsDialed(t *testing.T) {
	addr := loopbackDNSServer(t) // 127.0.0.1:<port> — a loopback the backstop refuses
	p := NetPeer{Resolver: addr, Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathDeclared, Name: "example.com", Qtype: QtypeA, Transport: UDP})

	if msg.Unreachable {
		t.Fatalf("declared query to loopback resolver %s refused as Unreachable; the custody egress guard must not gate the trusted declared resolver (#612)", addr)
	}
	if !msg.Reached {
		t.Fatalf("declared query to loopback resolver %s did not reach: %+v", addr, msg)
	}
}

// TestExchangeWalkInitialNSToLoopbackResolverIsDialed covers the second dial to
// the declared resolver: the delegation walk's initial NS query carries no
// Server and is therefore asked of the resolver (leaf.walk). On a loopback
// resolver, walkServerReachable would otherwise refuse it — but that pre-flight
// is for DISCOVERED authorities, so the resolver dial is exempt too.
func TestExchangeWalkInitialNSToLoopbackResolverIsDialed(t *testing.T) {
	addr := loopbackDNSServer(t)
	p := NetPeer{Resolver: addr, Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathWalk, Server: "", Name: "example.com", Qtype: QtypeNS, Transport: UDP})

	if msg.Unreachable {
		t.Fatalf("walk initial NS query to loopback resolver %s refused as Unreachable; the resolver dial must be exempt from the walk SSRF gate (#612)", addr)
	}
	if !msg.Reached {
		t.Fatalf("walk initial NS query to loopback resolver %s did not reach: %+v", addr, msg)
	}
}

// TestExchangeWalkAuthorityOnLoopbackStillRefused locks the exemption's scope:
// it applies ONLY to the declared resolver. A DISCOVERED walk authority named
// (in attacker-controlled NS RDATA) at a loopback/private address must still be
// refused — #324/#335 unchanged. This guards against a future over-broad
// exemption that would reopen the SSRF hole.
func TestExchangeWalkAuthorityOnLoopbackStillRefused(t *testing.T) {
	addr := loopbackDNSServer(t) // a real listener at 127.0.0.1:<port>
	p := NetPeer{Resolver: "1.1.1.1:53", Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathWalk, Server: addr, Name: "victim.example.com", Qtype: QtypeSOA, Transport: UDP})

	if !msg.Unreachable {
		t.Fatalf("walk authority at loopback %s was not refused (%+v); the SSRF gate must still bar a discovered private authority", addr, msg)
	}
}
