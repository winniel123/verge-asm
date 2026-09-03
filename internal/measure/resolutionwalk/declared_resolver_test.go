package resolutionwalk

import (
	"net"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

func loopbackDNSServer(t *testing.T) string {
	t.Helper()
	// A loopback address is exactly what the #335 backstop refuses (#612).
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

func TestExchangeDeclaredResolverOnLoopbackIsDialed(t *testing.T) {
	// The declared resolver is trusted operator config, never attacker RDATA (ADR-0070).
	// A default install points it at Docker's embedded DNS 127.0.0.11 (ADR-0036, #612).
	addr := loopbackDNSServer(t)
	p := NetPeer{Resolver: addr, Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathDeclared, Name: "example.com", Qtype: QtypeA, Transport: UDP})

	if msg.Unreachable {
		t.Fatalf("declared query to loopback resolver %s refused as Unreachable; the custody egress guard must not gate the trusted declared resolver (#612)", addr)
	}
	if !msg.Reached {
		t.Fatalf("declared query to loopback resolver %s did not reach: %+v", addr, msg)
	}
}

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

func TestExchangeWalkAuthorityOnLoopbackStillRefused(t *testing.T) {
	// The exemption is the declared resolver alone; a discovered authority stays gated (#324, #335).
	addr := loopbackDNSServer(t)
	p := NetPeer{Resolver: "1.1.1.1:53", Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathWalk, Server: addr, Name: "victim.example.com", Qtype: QtypeSOA, Transport: UDP})

	if !msg.Unreachable {
		t.Fatalf("walk authority at loopback %s was not refused (%+v); the SSRF gate must still bar a discovered private authority", addr, msg)
	}
}
