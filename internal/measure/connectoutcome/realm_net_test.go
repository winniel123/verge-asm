package connectoutcome

import (
	"context"
	"encoding/json"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/wire"
)

func loopbackListener(t *testing.T) netip.AddrPort {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("no loopback listener available: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	ap, err := netip.ParseAddrPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("parse listener addr: %v", err)
	}
	return ap
}

func specFor(t *testing.T, target netip.AddrPort, realm []string) wire.JobSpec {
	t.Helper()
	raw, err := json.Marshal(Scope{
		Vantage:      "local",
		VantageClass: string(custody.ClassUnverified),
		Addresses:    []string{target.Addr().String()},
		TCPPorts:     []uint16{target.Port()},
		Profile:      DefaultProfile(),
	})
	if err != nil {
		t.Fatalf("marshal scope: %v", err)
	}
	return wire.JobSpec{Batch: "batch", Kind: Kind, Scope: raw, Realm: realm}
}

// The end-to-end rig ADR-0222 called for, reached by a production configuration (ADR-0225 §5).

func TestConnectorReachesAnAddressTheJobsDeclaredRealmCovers(t *testing.T) {
	target := loopbackListener(t)
	spec := specFor(t, target, []string{"127.0.0.0/22"})

	c := NetConnector{Timeout: time.Second, realm: custody.ParseRealm(spec.Realm)}
	if got := c.Connect(context.Background(), target); got != ConnOpen {
		t.Fatalf("Connect(%s) = %s, want open", target, got)
	}
}

func TestConnectorRefusesTheSameAddressWithNoDeclaredRealm(t *testing.T) {
	target := loopbackListener(t)

	c := NetConnector{Timeout: time.Second}
	if got := c.Connect(context.Background(), target); got != ConnError {
		t.Fatalf("Connect(%s) = %s, want error (the guard refuses an undeclared address)", target, got)
	}
}

func TestConnectorRefusesAnAddressOutsideTheJobsDeclaredRealm(t *testing.T) {
	target := loopbackListener(t)
	// The realm names private space the listener is not in, so the guard still refuses it.
	spec := specFor(t, target, []string{"10.0.0.0/24"})

	c := NetConnector{Timeout: time.Second, realm: custody.ParseRealm(spec.Realm)}
	if got := c.Connect(context.Background(), target); got != ConnError {
		t.Fatalf("Connect(%s) = %s, want error (outside the declared realm)", target, got)
	}
}
