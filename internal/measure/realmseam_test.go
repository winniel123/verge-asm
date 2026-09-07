package measure_test

import (
	"reflect"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

// A repair reaching one site and not the others is the drift ADR-0079 refuses (ADR-0225 §4).

func TestEveryInstallSiteHoldsTheRealmSeamUnexported(t *testing.T) {
	want := reflect.TypeOf(custody.Realm{})
	dialers := []any{
		connectoutcome.NetConnector{},
		connectoutcome.NetHandshaker{},
		httpexchange.NetExchanger{},
		tlsacceptance.NetEnumerator{},
	}
	for _, d := range dialers {
		typ := reflect.TypeOf(d)
		field, ok := typ.FieldByName("realm")
		if !ok {
			t.Errorf("%s holds no realm field, so its guard cannot read a declared scope", typ)
			continue
		}
		if field.Type != want {
			t.Errorf("%s.realm is %s, want %s", typ, field.Type, want)
		}
		// An exported seam lets any caller widen the guard, which ADR-0222 refuses.
		if field.IsExported() {
			t.Errorf("%s.realm is exported, so a caller outside the package can set it", typ)
		}
	}
}

// A dialer built outside its own package carries the zero Realm, so nothing is exempt.

func TestAnOutsideCallerGetsNoRealm(t *testing.T) {
	built := []any{
		connectoutcome.NetConnector{Timeout: 0},
		connectoutcome.NetHandshaker{Timeout: 0},
		httpexchange.NetExchanger{},
		tlsacceptance.NetEnumerator{Timeout: 0},
	}
	for _, d := range built {
		typ := reflect.TypeOf(d)
		// IsZero reads an unexported field, which Interface refuses.
		if !reflect.ValueOf(d).FieldByName("realm").IsZero() {
			t.Errorf("%s built outside its package carries a realm, want none", typ)
		}
	}
}
