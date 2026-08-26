package resolutionwalk

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/winniel123/verge-asm/internal/custody"
)

// NetPeer is the production Peer: it puts the leaf's offers on the wire against
// a real recursive resolver (the declared path) and against delegated
// authorities (the walk), using golang.org/x/net/dns/dnsmessage so the EDNS and
// transport offers are passed explicitly rather than taken from a library
// default (ADR-0025).
//
// It is not exercised by the hermetic golden corpus, which scripts an in-process
// Peer; it is the adapter that runs at a live Vantage.
type NetPeer struct {
	// Resolver is the Vantage's recursive resolver, "host:port". It is part of
	// the Vantage's identity (ADR-0070), so it is supplied per batch and never
	// defaulted here.
	Resolver string
	// Timeout bounds one exchange. Zero uses a conservative default.
	Timeout time.Duration
}

// netResolver backs both the pre-flight custody vetting (walkServerReachable)
// and the dialer's own resolution (custodyDialer). In production both are the
// system resolver, so the dialer re-resolves the NS name independently of the
// vet — exactly the TOCTOU the Control hook exists to close. Tests substitute it
// to force a vetting/dial disagreement (DNS rebinding) deterministically.
var netResolver = net.DefaultResolver

func (p NetPeer) exchangeTimeout() time.Duration {
	if p.Timeout > 0 {
		return p.Timeout
	}
	return 3 * time.Second
}

// Exchange implements Peer against the network. A dial or read error against the
// server is reported as Unreachable — we could not look at all (ADR-0108) — which
// aborts the batch on the declared path; it is never folded into a resolution
// value. A build error, by contrast, is our own bug and stays a silent Msg{}.
// dialsDeclaredResolver reports whether an exchange targets the Vantage's own
// operator-declared recursive resolver rather than a discovered authority. It is
// THE trust boundary for the SSRF/rebinding egress guard (ADR-0121): a true
// result exempts the dial from the custody guard, a false result gates it, so it
// is the one place to audit when the guard's scope is in question. Every
// declared-path query, and the delegation walk's initial NS query (which names no
// Server), is asked of the resolver.
//
// CONTRACT: a discovered walk authority is named verbatim in NS RDATA and MUST
// carry that authority in a non-empty Server — leaf.walk sets Server = rr.Data,
// and dnsmessage never renders an empty name (a root NS is ".", not ""). Any
// future PathWalk query that dials a discovered target must uphold this; a
// discovered target reaching here with an empty Server would be silently
// exempted from the guard.
func (q Query) dialsDeclaredResolver() bool {
	return q.Path == PathDeclared || q.Server == ""
}

func (p NetPeer) Exchange(q Query) Msg {
	dialingResolver := q.dialsDeclaredResolver()
	server := q.Server
	if dialingResolver {
		server = p.Resolver
	}
	server = withDefaultPort(server)

	msgBytes, err := buildQuery(q)
	if err != nil {
		return Msg{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.exchangeTimeout())
	defer cancel()

	// SSRF gate (#324/#335). The delegation walk dials authorities named verbatim
	// in attacker-controlled NS RDATA (leaf.go walk() sets Query.Server = rr.Data).
	// A DNS query is exempt from the custody probing gate — "a query is not a
	// connect" (custody/gate.go) — so nothing else stops an in-scope delegation
	// from pointing its NS RDATA at 169.254.169.254, 127.0.0.1, an RFC1918/ULA
	// host or an internal name and having us send packets there. Refuse to dial a
	// discovered-authority target that is, or resolves to, a non-globally-reachable
	// address, and report it unreached exactly as a dial failure would — the walk
	// then records the authority as a silent (unreached) one, never a value.
	//
	// The Vantage's own recursive resolver is exempt (#612): it is operator-
	// declared configuration (ADR-0070) supplied out of band, not attacker-
	// influenced, and a legitimate deployment points it at a non-globally-reachable
	// address — Docker's embedded DNS 127.0.0.11 on the docs' compose deployment
	// (ADR-0036), or a private-LAN resolver on bare metal. Gating it refused every
	// default install's dns scan (a regression of #239). Only discovered
	// authorities are gated, so dialingResolver skips both the pre-flight vet and
	// the Control-hooked dialer.
	if !dialingResolver && !walkServerReachable(ctx, server) {
		return Msg{Unreachable: true}
	}

	dialer := trustedDialer()
	if !dialingResolver {
		dialer = custodyDialer()
	}

	var resp []byte
	if q.Transport == TCP {
		resp, err = exchangeTCP(ctx, dialer, server, msgBytes)
	} else {
		resp, err = exchangeUDP(ctx, dialer, server, msgBytes)
	}
	if err != nil {
		return Msg{Unreachable: true}
	}
	return parseResponse(resp)
}

func withDefaultPort(server string) string {
	if _, _, err := net.SplitHostPort(server); err == nil {
		return server
	}
	return net.JoinHostPort(server, "53")
}

// walkServerReachable reports whether a delegation-walk authority (host:port)
// may be dialed: its dial target must be globally reachable (#324). An IP
// literal is classified directly against the special-purpose registries; a
// hostname is resolved and barred if ANY address it would dial is non-globally-
// reachable — a conservative reading, since the dialer, not us, picks which of
// a name's addresses to connect to, so a name mixing public and private
// addresses is refused rather than gambled on. A name that does not resolve is
// left to the dial to fail on its own: it cannot reach an internal address, so
// it is not an SSRF concern. custody.IsNonGloballyReachable is the sole IP-range
// authority (it Unmaps, so an IPv4-mapped literal is caught in IPv4 space).
func walkServerReachable(ctx context.Context, server string) bool {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		host = server
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return !custody.IsNonGloballyReachable(addr)
	}
	addrs, err := netResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return true
	}
	for _, a := range addrs {
		if custody.IsNonGloballyReachable(a) {
			return false
		}
	}
	return true
}

func buildQuery(q Query) ([]byte, error) {
	name, err := dnsmessage.NewName(fqdn(q.Name))
	if err != nil {
		return nil, err
	}
	qt, ok := wireType(q.Qtype)
	if !ok {
		return nil, errors.New("resolutionwalk: unsupported qtype " + string(q.Qtype))
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:               uint16(time.Now().UnixNano()),
		RecursionDesired: q.Path == PathDeclared,
	})
	b.EnableCompression()
	if err := b.StartQuestions(); err != nil {
		return nil, err
	}
	if err := b.Question(dnsmessage.Question{
		Name:  name,
		Type:  qt,
		Class: dnsmessage.ClassINET,
	}); err != nil {
		return nil, err
	}
	if q.EDNS {
		if err := b.StartAdditionals(); err != nil {
			return nil, err
		}
		var rh dnsmessage.ResourceHeader
		// OPT: advertise the declared UDP buffer size in the class field.
		if err := rh.SetEDNS0(1232, dnsmessage.RCodeSuccess, false); err != nil {
			return nil, err
		}
		if err := b.OPTResource(rh, dnsmessage.OPTResource{}); err != nil {
			return nil, err
		}
	}
	return b.Finish()
}

// trustedDialer dials the Vantage's operator-declared recursive resolver with no
// custody Control hook. That resolver is trusted configuration (ADR-0070),
// supplied by the operator out of band rather than derived from attacker-
// controlled RDATA, and a legitimate deployment may point it at a non-globally-
// reachable address — Docker's embedded DNS 127.0.0.11 on the docs' compose
// deployment (ADR-0036), or a private-LAN resolver on bare metal. The #335
// backstop exists to stop DISCOVERED walk authorities reaching internal
// addresses; applying it to the declared resolver refused every default install
// (#612, a regression of #239). It keeps custodyDialer's Resolver so a
// name-based resolver still resolves through netResolver.
//
// PRECONDITION: dropping the socket-level backstop here is sound ONLY because the
// resolver is trusted config — vantage.resolver is populated solely from operator
// input (run.go and wildcarddiscrim/run.go source it from the JobSpec Scope,
// itself the vantage row), never from scan, proposer, or any request-surface
// data. If a future feature ever lets untrusted data flow into a Vantage's
// resolver, this dial regains an SSRF vector and must be re-gated (ADR-0121).
func trustedDialer() net.Dialer {
	return net.Dialer{Resolver: netResolver}
}

// custodyDialer returns a net.Dialer whose Control hook inspects the ACTUAL
// resolved socket address of every connection and refuses to open the socket
// when it lands in a non-globally-reachable range (#335). It dials only
// discovered walk authorities — the declared resolver goes through trustedDialer.
// walkServerReachable vets the NS hostname's addresses before the dial, but the
// dialer re-resolves the name independently, so a name that flips from a public
// to a private answer between the pre-flight check and the dial (DNS rebinding, a
// TOCTOU) would otherwise slip a packet to an internal address. Control runs
// after DNS resolution, on the very address the kernel is about to connect to, so
// the vetted address is the one dialed — the rebinding-proof backstop, mirroring
// the delivery runner's hook (NewHTTPDoer, #325). An IP-literal target resolves
// to itself, so the safe literal branch is re-affirmed here rather than regressed.
func custodyDialer() net.Dialer {
	return net.Dialer{
		Resolver: netResolver,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil {
				return err
			}
			if custody.IsNonGloballyReachable(ip.Unmap()) {
				return fmt.Errorf("resolutionwalk: refusing to dial non-globally-reachable address %s", host)
			}
			return nil
		},
	}
}

func exchangeUDP(ctx context.Context, d net.Dialer, server string, msg []byte) ([]byte, error) {
	conn, err := d.DialContext(ctx, "udp", server)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	if _, err := conn.Write(msg); err != nil {
		return nil, err
	}
	buf := make([]byte, 1232)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func exchangeTCP(ctx context.Context, d net.Dialer, server string, msg []byte) ([]byte, error) {
	conn, err := d.DialContext(ctx, "tcp", server)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	framed := make([]byte, 2+len(msg))
	binary.BigEndian.PutUint16(framed, uint16(len(msg)))
	copy(framed[2:], msg)
	if _, err := conn.Write(framed); err != nil {
		return nil, err
	}
	var lenBuf [2]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return nil, err
	}
	respLen := binary.BigEndian.Uint16(lenBuf[:])
	resp := make([]byte, respLen)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func parseResponse(raw []byte) Msg {
	var parser dnsmessage.Parser
	hdr, err := parser.Start(raw)
	if err != nil {
		return Msg{}
	}
	msg := Msg{Reached: true, Truncated: hdr.Truncated, Rcode: rcodeName(hdr.RCode)}
	if err := parser.SkipAllQuestions(); err != nil {
		return msg
	}
	for {
		rh, err := parser.AnswerHeader()
		if errors.Is(err, dnsmessage.ErrSectionDone) {
			break
		}
		if err != nil {
			break
		}
		rr, ok := decodeAnswer(&parser, rh)
		if ok {
			msg.Answer = append(msg.Answer, rr)
		} else {
			_ = parser.SkipAnswer()
		}
	}
	return msg
}

func decodeAnswer(parser *dnsmessage.Parser, rh dnsmessage.ResourceHeader) (RR, bool) {
	owner := rh.Name.String()
	switch rh.Type {
	case dnsmessage.TypeA:
		body, err := parser.AResource()
		if err != nil {
			return RR{}, false
		}
		return RR{Name: owner, Type: QtypeA, Data: netip.AddrFrom4(body.A).String()}, true
	case dnsmessage.TypeAAAA:
		body, err := parser.AAAAResource()
		if err != nil {
			return RR{}, false
		}
		return RR{Name: owner, Type: QtypeAAAA, Data: netip.AddrFrom16(body.AAAA).String()}, true
	case dnsmessage.TypeCNAME:
		body, err := parser.CNAMEResource()
		if err != nil {
			return RR{}, false
		}
		return RR{Name: owner, Type: QtypeCNAME, Data: body.CNAME.String()}, true
	case dnsmessage.TypeNS:
		body, err := parser.NSResource()
		if err != nil {
			return RR{}, false
		}
		return RR{Name: owner, Type: QtypeNS, Data: body.NS.String()}, true
	case dnsmessage.TypeMX:
		body, err := parser.MXResource()
		if err != nil {
			return RR{}, false
		}
		return RR{Name: owner, Type: QtypeMX, Data: body.MX.String()}, true
	case dnsmessage.TypeTXT:
		body, err := parser.TXTResource()
		if err != nil {
			return RR{}, false
		}
		data := ""
		for i, s := range body.TXT {
			if i > 0 {
				data += " "
			}
			data += strconv.Quote(s)
		}
		return RR{Name: owner, Type: QtypeTXT, Data: data}, true
	case dnsmessage.TypeSOA:
		body, err := parser.SOAResource()
		if err != nil {
			return RR{}, false
		}
		return RR{Name: owner, Type: QtypeSOA, Data: body.NS.String()}, true
	default:
		return RR{}, false
	}
}

func wireType(qt Qtype) (dnsmessage.Type, bool) {
	switch qt {
	case QtypeA:
		return dnsmessage.TypeA, true
	case QtypeAAAA:
		return dnsmessage.TypeAAAA, true
	case QtypeCNAME:
		return dnsmessage.TypeCNAME, true
	case QtypeNS:
		return dnsmessage.TypeNS, true
	case QtypeSOA:
		return dnsmessage.TypeSOA, true
	case QtypeMX:
		return dnsmessage.TypeMX, true
	case QtypeTXT:
		return dnsmessage.TypeTXT, true
	default:
		return 0, false
	}
}

func rcodeName(rc dnsmessage.RCode) Rcode {
	switch rc {
	case dnsmessage.RCodeSuccess:
		return NOERROR
	case dnsmessage.RCodeNameError:
		return NXDOMAIN
	case dnsmessage.RCodeFormatError:
		return FORMERR
	case dnsmessage.RCodeRefused:
		return REFUSED
	case dnsmessage.RCodeServerFailure:
		return SERVFAIL
	default:
		return Rcode(rc.String())
	}
}

func fqdn(name string) string {
	if len(name) == 0 {
		return "."
	}
	if name[len(name)-1] == '.' {
		return name
	}
	return name + "."
}
