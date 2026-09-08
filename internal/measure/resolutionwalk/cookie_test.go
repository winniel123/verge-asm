package resolutionwalk

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

type cookieServer struct {
	addr         string
	serverCookie []byte
	alwaysBad    bool

	mu      sync.Mutex
	queries [][]byte
}

func startCookieServer(t *testing.T, alwaysBad bool) *cookieServer {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start cookie DNS server: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close() })
	s := &cookieServer{addr: pc.LocalAddr().String(), serverCookie: []byte("srvcooki"), alwaysBad: alwaysBad}
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			query := append([]byte(nil), buf[:n]...)
			s.mu.Lock()
			s.queries = append(s.queries, query)
			s.mu.Unlock()
			if resp, ok := s.respond(query); ok {
				_, _ = pc.WriteTo(resp, from)
			}
		}
	}()
	return s
}

func (s *cookieServer) seen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.queries)
}

func queryCookie(query []byte) ([]byte, bool) {
	var p dnsmessage.Parser
	if _, err := p.Start(query); err != nil {
		return nil, false
	}
	if err := p.SkipAllQuestions(); err != nil {
		return nil, false
	}
	if err := p.SkipAllAnswers(); err != nil {
		return nil, false
	}
	if err := p.SkipAllAuthorities(); err != nil {
		return nil, false
	}
	for {
		rh, err := p.AdditionalHeader()
		if err != nil {
			return nil, false
		}
		if rh.Type != dnsmessage.TypeOPT {
			if err := p.SkipAdditional(); err != nil {
				return nil, false
			}
			continue
		}
		opt, err := p.OPTResource()
		if err != nil {
			return nil, false
		}
		for _, o := range opt.Options {
			if o.Code == 10 {
				return o.Data, true
			}
		}
		return nil, false
	}
}

func (s *cookieServer) respond(query []byte) ([]byte, bool) {
	var p dnsmessage.Parser
	hdr, err := p.Start(query)
	if err != nil {
		return nil, false
	}
	q, err := p.Question()
	if err != nil {
		return nil, false
	}
	cookie, hasCookie := queryCookie(query)

	rcode := dnsmessage.RCodeSuccess
	answer := false
	switch {
	case !hasCookie || len(cookie) < 8:
		rcode = dnsmessage.RCodeRefused
	case s.alwaysBad || !bytes.Equal(cookie[8:], s.serverCookie):
		rcode = dnsmessage.RCode(23)
	default:
		answer = true
	}

	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: hdr.ID, Response: true, RecursionAvailable: true, RCode: rcode & 0xF})
	if err := b.StartQuestions(); err != nil {
		return nil, false
	}
	if err := b.Question(q); err != nil {
		return nil, false
	}
	if err := b.StartAnswers(); err != nil {
		return nil, false
	}
	if answer && q.Type == dnsmessage.TypeA {
		_ = b.AResource(
			dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET},
			dnsmessage.AResource{A: [4]byte{203, 0, 113, 5}},
		)
	}
	if hasCookie {
		if err := b.StartAdditionals(); err != nil {
			return nil, false
		}
		var rh dnsmessage.ResourceHeader
		if err := rh.SetEDNS0(1232, rcode, false); err != nil {
			return nil, false
		}
		full := append(append([]byte(nil), cookie[:8]...), s.serverCookie...)
		if err := b.OPTResource(rh, dnsmessage.OPTResource{Options: []dnsmessage.Option{{Code: 10, Data: full}}}); err != nil {
			return nil, false
		}
	}
	out, err := b.Finish()
	if err != nil {
		return nil, false
	}
	return out, true
}

func TestExchangeSendsCookieAndRetriesOnceOnBadCookie(t *testing.T) {
	// ADR-0030 §7: the cookie is sent and BADCOOKIE is retried once,
	// so it never reaches the walk (#1660).
	s := startCookieServer(t, false)
	p := NetPeer{Resolver: s.addr, Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathDeclared, Name: "example.com", Qtype: QtypeA, Transport: UDP, EDNS: true, Cookie: true})

	if !msg.Reached || msg.Rcode != NOERROR || len(msg.Answer) != 1 {
		t.Fatalf("cookie-requiring server was not answered through the RFC 7873 retry: %+v", msg)
	}
	if got := s.seen(); got != 2 {
		t.Fatalf("server saw %d queries, want 2 (client-only cookie, then the full cookie)", got)
	}
}

func TestExchangeRetriesBadCookieOnlyOnce(t *testing.T) {
	s := startCookieServer(t, true)
	p := NetPeer{Resolver: s.addr, Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathDeclared, Name: "example.com", Qtype: QtypeA, Transport: UDP, EDNS: true, Cookie: true})

	if !msg.Reached {
		t.Fatalf("a persistent BADCOOKIE is a reached answer, got %+v", msg)
	}
	if got := s.seen(); got != 2 {
		t.Fatalf("server saw %d queries, want exactly 2: one BADCOOKIE retry, never a loop", got)
	}
}

func TestExchangeSendsNoCookieWhenOfferIsOff(t *testing.T) {
	s := startCookieServer(t, false)
	p := NetPeer{Resolver: s.addr, Timeout: 500 * time.Millisecond}

	msg := p.Exchange(Query{Path: PathDeclared, Name: "example.com", Qtype: QtypeA, Transport: UDP, EDNS: true, Cookie: false})

	if !msg.Reached || msg.Rcode != REFUSED {
		t.Fatalf("a cookieless query should be REFUSED by this server, got %+v", msg)
	}
	if got := s.seen(); got != 1 {
		t.Fatalf("server saw %d queries, want 1: REFUSED is not BADCOOKIE and earns no retry", got)
	}
}
