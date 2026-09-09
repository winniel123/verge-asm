package resolutionwalk

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"github.com/winniel123/verge-asm/internal/custody"
)

type NetPeer struct {
	Resolver string
	Timeout  time.Duration
}

var netResolver = net.DefaultResolver

const (
	ednsOptionCookie uint16           = 10
	rcodeBadCookie   dnsmessage.RCode = 23
)

var cookieSecret = func() []byte {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("resolutionwalk: cookie secret: " + err.Error())
	}
	return b
}()

func clientCookie(server string) []byte {
	// RFC 7873 §4.1 keys the client cookie to the server so one server sees one cookie.
	h := sha256.New()
	h.Write(cookieSecret)
	h.Write([]byte(server))
	return h.Sum(nil)[:8]
}

func (p NetPeer) exchangeTimeout() time.Duration {
	if p.Timeout > 0 {
		return p.Timeout
	}
	return 3 * time.Second
}

func (q Query) dialsDeclaredResolver() bool {
	// The SSRF guard's whole trust boundary: true exempts the dial, false gates it (ADR-0121).
	return q.Path == PathDeclared || q.Server == ""
}

func (p NetPeer) Exchange(q Query) Msg {
	// The golden corpus scripts an in-process Peer, so nothing here is exercised by it.
	dialingResolver := q.dialsDeclaredResolver()
	server := q.Server
	if dialingResolver {
		server = p.Resolver
	}
	server = withDefaultPort(server)

	var cookie []byte
	if q.EDNS && q.Cookie {
		cookie = clientCookie(server)
	}
	// A build error is our own bug, not the network's, so it is never Unreachable (ADR-0108).
	msgBytes, err := buildQuery(q, cookie)
	if err != nil {
		return Msg{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.exchangeTimeout())
	defer cancel()

	// Nothing else stops a walk dial: custody's probing gate exempts a DNS query (#324, #335).
	if !dialingResolver && !walkServerReachable(ctx, server) {
		return Msg{Unreachable: true}
	}

	// Gating the operator-declared resolver refused every default install (#612, ADR-0070).
	dialer := trustedDialer()
	if !dialingResolver {
		dialer = custodyDialer()
	}

	resp, err := send(ctx, dialer, q.Transport, server, msgBytes)
	if err != nil {
		return Msg{Unreachable: true}
	}
	// One retry with the server cookie keeps BADCOOKIE out of the walk (ADR-0030 §7, #1660).
	if serverCookie, bad := badCookie(resp, cookie); bad {
		msgBytes, err = buildQuery(q, slices.Concat(cookie, serverCookie))
		if err != nil {
			return Msg{}
		}
		resp, err = send(ctx, dialer, q.Transport, server, msgBytes)
		if err != nil {
			return Msg{Unreachable: true}
		}
	}
	return parseResponse(resp)
}

func send(ctx context.Context, d net.Dialer, t Transport, server string, msg []byte) ([]byte, error) {
	if t == TCP {
		return exchangeTCP(ctx, d, server, msg)
	}
	return exchangeUDP(ctx, d, server, msg)
}

func badCookie(raw, clientCookie []byte) ([]byte, bool) {
	if len(clientCookie) == 0 {
		return nil, false
	}
	var p dnsmessage.Parser
	hdr, err := p.Start(raw)
	if err != nil {
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
		if rh.ExtendedRCode(hdr.RCode) != rcodeBadCookie {
			return nil, false
		}
		opt, err := p.OPTResource()
		if err != nil {
			return nil, false
		}
		for _, o := range opt.Options {
			if o.Code != ednsOptionCookie || len(o.Data) <= len(clientCookie) {
				continue
			}
			// Only a BADCOOKIE that echoes our own client cookie earns the retry (RFC 7873 §5.3).
			if !bytes.Equal(o.Data[:len(clientCookie)], clientCookie) {
				return nil, false
			}
			return o.Data[len(clientCookie):], true
		}
		return nil, false
	}
}

func withDefaultPort(server string) string {
	if _, _, err := net.SplitHostPort(server); err == nil {
		return server
	}
	return net.JoinHostPort(server, "53")
}

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
		// A name that does not resolve reaches no internal address, so it is no SSRF risk.
		return true
	}
	// The dialer picks the address, so a name mixing public and private is refused (#324).
	for _, a := range addrs {
		if custody.IsNonGloballyReachable(a) {
			return false
		}
	}
	return true
}

func buildQuery(q Query, cookie []byte) ([]byte, error) {
	name, err := dnsmessage.NewName(fqdn(q.Name))
	if err != nil {
		return nil, err
	}
	qt, ok := wireType(q.Qtype)
	if !ok {
		return nil, errors.New("resolutionwalk: unsupported qtype " + string(q.Qtype))
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{
		ID:               uint16(time.Now().UnixNano()), // #nosec G115 (intended 16-bit truncation for the DNS transaction ID field)
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
		if err := rh.SetEDNS0(1232, dnsmessage.RCodeSuccess, false); err != nil {
			return nil, err
		}
		var opt dnsmessage.OPTResource
		if len(cookie) > 0 {
			opt.Options = []dnsmessage.Option{{Code: ednsOptionCookie, Data: cookie}}
		}
		if err := b.OPTResource(rh, opt); err != nil {
			return nil, err
		}
	}
	return b.Finish()
}

func trustedDialer() net.Dialer {
	// Sound only while a Vantage resolver comes from operator input alone (ADR-0121).
	return net.Dialer{Resolver: netResolver}
}

func custodyDialer() net.Dialer {
	// The vet and this dial resolve separately, so only Control sees the dialed address (#335).
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
	binary.BigEndian.PutUint16(framed, uint16(len(msg))) // #nosec G115 (own small DNS query; DoT length prefix is a 16-bit field per RFC 1035)
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
		// The union is closed at the codes the leaf branches on (ADR-0143).
		return OTHER
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
