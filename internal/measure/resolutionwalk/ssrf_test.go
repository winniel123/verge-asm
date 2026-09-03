package resolutionwalk

import (
	"context"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// gatedWalkPeer scripts the initial NS answer of a delegation walk but routes
// every per-authority SOA sub-query through a real NetPeer, so walk() sees
// exactly what a live Vantage would when an in-scope delegation points its NS
// RDATA at a non-globally-reachable address. The initial NS query carries no
// Server (walk() leaves it to the resolver); the per-authority sub-queries carry
// Server = the NS RDATA, and those are the ones the SSRF gate must catch.
type gatedWalkPeer struct {
	ns   []RR
	real NetPeer
}

func (p gatedWalkPeer) Exchange(q Query) Msg {
	if q.Server == "" {
		return Msg{Reached: true, Answer: p.ns}
	}
	return p.real.Exchange(q)
}

// TestWalkRefusesCloudMetadataAuthority is finding #324: an in-scope delegation
// whose NS RDATA is the cloud-metadata address must not be dialed. The walk
// records the authority as unreached (a Gap, our own blindness), never a value.
func TestWalkRefusesCloudMetadataAuthority(t *testing.T) {
	peer := gatedWalkPeer{
		ns: []RR{{Name: "victim.example.com", Type: QtypeNS, Data: "169.254.169.254"}},
		// A short timeout would still expire if the gate failed and we dialed
		// the (unroutable) metadata address; the gate returns before any dial.
		real: NetPeer{Resolver: "127.0.0.1:53"},
	}

	del := walk(peer, DefaultOffers(), "victim.example.com")

	if !del.Gap {
		t.Errorf("walk did not record a Gap for the barred authority: %+v", del)
	}
	if len(del.Nameservers) != 1 {
		t.Fatalf("want exactly one recorded authority, got %+v", del.Nameservers)
	}
	if got := del.Nameservers[0]; got.Server != "169.254.169.254" || got.Serves {
		t.Errorf("barred authority recorded as %+v, want {169.254.169.254 false} (unreached)", got)
	}
}

func TestWalkServerReachableGate(t *testing.T) {
	ctx := context.Background()
	barred := []string{
		"169.254.169.254:53",   // cloud metadata (link-local)
		"127.0.0.1:53",         // loopback
		"10.0.0.5:53",          // RFC1918
		"192.168.1.1:53",       // RFC1918
		"172.16.0.1:53",        // RFC1918
		"[fd00::1]:53",         // ULA
		"[::1]:53",             // IPv6 loopback
		"[::ffff:10.0.0.1]:53", // IPv4-mapped RFC1918
	}
	for _, s := range barred {
		if walkServerReachable(ctx, s) {
			t.Errorf("walkServerReachable(%q) = true, want false (non-globally-reachable)", s)
		}
	}
	allowed := []string{"8.8.8.8:53", "1.1.1.1:53", "[2001:4860:4860::8888]:53"}
	for _, s := range allowed {
		if !walkServerReachable(ctx, s) {
			t.Errorf("walkServerReachable(%q) = false, want true (globally reachable)", s)
		}
	}
}

// TestCustodyDialerControlRefusesNonGlobal pins the socket-level backstop
// directly (#335): the dialer's Control hook, given the ACTUAL resolved socket
// address the kernel is about to connect to, refuses every non-globally-reachable
// range and passes ordinary global unicast. This is the check that stands even
// when the pre-flight vet was satisfied by a different (public) address.
func TestCustodyDialerControlRefusesNonGlobal(t *testing.T) {
	control := custodyDialer().Control
	if control == nil {
		t.Fatal("custodyDialer has no Control hook; the rebinding backstop is absent")
	}
	refused := []string{
		"169.254.169.254:53",      // cloud metadata (link-local)
		"127.0.0.1:53",            // loopback
		"10.0.0.5:53",             // RFC1918
		"192.168.1.1:53",          // RFC1918
		"[fd00::1]:53",            // ULA
		"[::1]:53",                // IPv6 loopback
		"[::ffff:169.254.0.1]:53", // IPv4-mapped link-local
	}
	for _, addr := range refused {
		if err := control("udp", addr, nil); err == nil {
			t.Errorf("Control(%q) = nil, want refusal (non-globally-reachable dial target)", addr)
		}
	}
	allowed := []string{"8.8.8.8:53", "1.1.1.1:53", "[2001:4860:4860::8888]:53"}
	for _, addr := range allowed {
		if err := control("udp", addr, nil); err != nil {
			t.Errorf("Control(%q) = %v, want nil (globally reachable dial target)", addr, err)
		}
	}
}

// rebindResolver starts an in-process UDP DNS server whose A answer flips from
// pub (first query) to priv (every later query), and returns a *net.Resolver
// wired to it. It models DNS rebinding: the pre-flight vet resolves the NS name
// and sees the public address, then the dial re-resolves the same name and gets
// the private one. AAAA queries answer empty, so only the A record decides.
func rebindResolver(t *testing.T, pub, priv netip.Addr) *net.Resolver {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start rebind DNS server: %v", err)
	}
	t.Cleanup(func() { _ = pc.Close() })

	var aCount atomic.Int32
	go func() {
		buf := make([]byte, 1500)
		for {
			n, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			resp, ok := buildRebindResponse(buf[:n], pub, priv, &aCount)
			if ok {
				_, _ = pc.WriteTo(resp, from)
			}
		}
	}()

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "udp", pc.LocalAddr().String())
		},
	}
}

func buildRebindResponse(query []byte, pub, priv netip.Addr, aCount *atomic.Int32) ([]byte, bool) {
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
	if q.Type == dnsmessage.TypeA {
		ip := pub
		if aCount.Add(1) > 1 {
			ip = priv
		}
		_ = b.AResource(
			dnsmessage.ResourceHeader{Name: q.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET},
			dnsmessage.AResource{A: ip.As4()},
		)
	}
	out, err := b.Finish()
	if err != nil {
		return nil, false
	}
	return out, true
}

// TestExchangeRefusesRebindToPrivate is finding #335 end to end: the walk-path
// vet resolves the NS name and sees a public address (so the gate admits it),
// but the dialer re-resolves the same name to a loopback address (rebinding).
// The Control-hooked dialer must refuse that socket, so Exchange reports
// Unreachable and NOT ONE packet reaches the private address.
func TestExchangeRefusesRebindToPrivate(t *testing.T) {
	pub := netip.MustParseAddr("8.8.8.8")    // global — the vet admits it
	priv := netip.MustParseAddr("127.0.0.1") // loopback — the dial must be refused

	orig := netResolver
	netResolver = rebindResolver(t, pub, priv)
	t.Cleanup(func() { netResolver = orig })

	// A sink at the exact private socket the rebind points the dial at. If the
	// Control hook let the dial through, this would receive the DNS query.
	sink, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start private sink: %v", err)
	}
	defer func() { _ = sink.Close() }()
	var sinkGot atomic.Int32
	go func() {
		buf := make([]byte, 512)
		for {
			if _, _, err := sink.ReadFrom(buf); err != nil {
				return
			}
			sinkGot.Add(1)
		}
	}()
	_, sinkPort, _ := net.SplitHostPort(sink.LocalAddr().String())

	p := NetPeer{Timeout: 500 * time.Millisecond}
	msg := p.Exchange(Query{
		Path:      PathWalk,
		Server:    "rebind.test:" + sinkPort,
		Name:      "victim.example.com",
		Qtype:     QtypeA,
		Transport: UDP,
	})

	if !msg.Unreachable {
		t.Errorf("Exchange = %+v, want Unreachable (dial to rebound private address refused)", msg)
	}
	// Exchange returned synchronously; any packet would already have been sent.
	time.Sleep(50 * time.Millisecond)
	if n := sinkGot.Load(); n != 0 {
		t.Errorf("private sink received %d packet(s); the rebound dial was NOT refused", n)
	}
}
