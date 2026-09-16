package main

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestKeyedSubjectReadFailureRendersTheUnresolvedPage2170(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		key  string
		set  func(*fakeStore)
	}{
		{
			name: "endpoint",
			path: "/subjects/endpoint?key=host.example.com%40198.51.100.1%3A443%2Ftcp",
			key:  "host.example.com@198.51.100.1:443/tcp",
			set:  func(f *fakeStore) { f.getEndpointSubjectErr = errors.New("endpoint subject read failed") },
		},
		{
			name: "service",
			path: "/subjects/service?key=198.51.100.1%3A5900%2Ftcp",
			key:  "198.51.100.1:5900/tcp",
			set:  func(f *fakeStore) { f.getServiceSubjectErr = errors.New("service subject read failed") },
		},
		{
			name: "name",
			path: "/asset/gone.example.com",
			key:  "gone.example.com",
			set:  func(f *fakeStore) { f.getNameSubjectErr = errors.New("name subject read failed") },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeStore()
			seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
			tc.set(f)
			base := start(t, f, "")
			ac := login(t, base, "admin", "hunter2hunter2")

			page := getBody(t, ac, base+tc.path, http.StatusServiceUnavailable)

			if strings.Contains(page, "internal error") {
				t.Errorf("a failed keyed read answered with the bare internal-error body; body: %s", page)
			}
			if strings.Contains(page, "No such subject") || strings.Contains(page, subjectKeyedNowhere) {
				t.Errorf("a failed read rendered as a fact about the subject; body: %s", page)
			}
			for _, want := range []string{withdrawalDidNotResolve, "did not resolve on this load", tc.key, "Back to inventory"} {
				if !strings.Contains(page, want) {
					t.Errorf("unresolved-subject page missing %q; body: %s", want, page)
				}
			}
		})
	}
}
