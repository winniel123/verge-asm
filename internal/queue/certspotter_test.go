package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The Cert Spotter fetcher carries the operator's API token as a Bearer credential
// (spec §2.4), and the keyless crt.sh fetcher sends none. The token rides only on
// the Authorization header — the credential the source authenticates against — so a
// server behind the fetcher sees "Bearer <token>" for Cert Spotter and no auth for
// crt.sh.
func TestCertSpotterFetcherSendsBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	keyed := NewCertSpotterFetcher("9.9.9", "s3cr3t-token")
	if _, _, err := keyed.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatalf("keyed Fetch: %v", err)
	}
	if gotAuth != "Bearer s3cr3t-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer s3cr3t-token")
	}

	// The keyless fetcher (crt.sh) sends no Authorization header at all.
	gotAuth = "unset"
	keyless := NewHTTPCTFetcher("9.9.9")
	if _, _, err := keyless.Fetch(context.Background(), srv.URL); err != nil {
		t.Fatalf("keyless Fetch: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("keyless Authorization = %q, want empty (crt.sh is keyless)", gotAuth)
	}
}
