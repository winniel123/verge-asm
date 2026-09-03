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

func TestWalkRefusesCloudMetadataAuthority(t *testing.T) {
	// #324: an NS RDATA pointing at cloud metadata must never be dialed.
	peer := gatedWalkPeer{
		ns: []RR{{Name: "victim.example.com", Type: QtypeNS, Data: "169.254.169.254"}},
		// The gate returns before any dial, so no timeout is needed here.
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
		"169.254.169.254:53",
		"127.0.0.1:53",
		"10.0.0.5:53",
		"192.168.1.1:53",
		"172.16.0.1:53",
		"[fd00::1]:53",
		"[::1]:53",
		"[::ffff:10.0.0.1]:53",
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

func TestCustodyDialerControlRefusesNonGlobal(t *testing.T) {
	// #335: Control refuses on the actual socket address, even when the vet passed.
	control := custodyDialer().Control
	if control == nil {
		t.Fatal("custodyDialer has no Control hook; the rebinding backstop is absent")
	}
	refused := []string{
		"169.254.169.254:53",
		"127.0.0.1:53",
		"10.0.0.5:53",
		"192.168.1.1:53",
		"[fd00::1]:53",
		"[::1]:53",
		"[::ffff:169.254.0.1]:53",
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

func TestExchangeRefusesRebindToPrivate(t *testing.T) {
	// #335 end to end: a name that flips to loopback between vet and dial must send no packet.
	pub := netip.MustParseAddr("8.8.8.8")
	priv := netip.MustParseAddr("127.0.0.1")

	orig := netResolver
	netResolver = rebindResolver(t, pub, priv)
	t.Cleanup(func() { netResolver = orig })

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
	// The dial is synchronous, so a leaked packet would already have arrived.
	time.Sleep(50 * time.Millisecond)
	if n := sinkGot.Load(); n != 0 {
		t.Errorf("private sink received %d packet(s); the rebound dial was NOT refused", n)
	}
}
