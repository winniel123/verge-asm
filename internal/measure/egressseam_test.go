package measure_test

import (
	"context"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

var dialControlType = reflect.TypeOf(func(string, string, syscall.RawConn) error { return nil })

func guardedDialerTypes() []reflect.Type {
	// resolutionwalk is excluded: ADR-0121 gives it a second dialer on a different ground.
	return []reflect.Type{
		reflect.TypeOf(connectoutcome.NetConnector{}),
		reflect.TypeOf(connectoutcome.NetHandshaker{}),
		reflect.TypeOf(httpexchange.NetExchanger{}),
		reflect.TypeOf(tlsacceptance.NetEnumerator{}),
	}
}

func TestDialControlSeamStaysUnexported(t *testing.T) {
	// An exported seam lets a package outside the leaf replace the guard (ADR-0222 §1, #1598).
	for _, ty := range guardedDialerTypes() {
		for i := 0; i < ty.NumField(); i++ {
			f := ty.Field(i)
			if f.Type != dialControlType {
				continue
			}
			if f.IsExported() {
				t.Errorf("%s.%s is an exported dial-control field, want unexported", ty, f.Name)
			}
		}
	}
}

func TestOutsideCallerGetsTheGuard(t *testing.T) {
	p := httpexchange.DefaultParams()
	p.TimeoutMillis = 2000
	// This package cannot set the unexported seam, so it builds what a production caller builds.
	ex := httpexchange.NetExchanger{Params: p}

	start := time.Now()
	res := ex.Exchange(context.Background(), httpexchange.Target{
		Address: "127.0.0.1",
		Port:    80,
		Scheme:  "http",
	})
	if !res.Failed {
		t.Fatalf("Exchange(127.0.0.1) = %+v, want a refusal", res)
	}
	// A dial that reached the network would run to the deadline, so speed is the proof.
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Exchange(127.0.0.1) took %v, want a refusal before the dial", elapsed)
	}
}
