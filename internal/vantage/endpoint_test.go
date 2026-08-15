package vantage

import "testing"

func TestParseEndpointAcceptsGood(t *testing.T) {
	got, err := ParseEndpoint(" prober.example.com ", "2222", " scanner ")
	if err != nil {
		t.Fatalf("ParseEndpoint: %v", err)
	}
	if got.Host != "prober.example.com" || got.Port != 2222 || got.Username != "scanner" {
		t.Errorf("ParseEndpoint = %+v, want trimmed host/port/username", got)
	}
}

func TestParseEndpointDefaultsPort(t *testing.T) {
	got, err := ParseEndpoint("198.51.100.7", "", "scanner")
	if err != nil {
		t.Fatalf("ParseEndpoint: %v", err)
	}
	if got.Port != DefaultPort {
		t.Errorf("blank port = %d, want default %d", got.Port, DefaultPort)
	}
}

func TestParseEndpointRejects(t *testing.T) {
	cases := []struct {
		name                 string
		host, port, username string
	}{
		{"empty host", "", "22", "scanner"},
		{"host with scheme", "ssh://host", "22", "scanner"},
		{"host with path", "host/x", "22", "scanner"},
		{"port zero", "host", "0", "scanner"},
		{"port too big", "host", "70000", "scanner"},
		{"port not a number", "host", "ssh", "scanner"},
		{"empty username", "host", "22", ""},
		{"root username", "host", "22", "root"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseEndpoint(c.host, c.port, c.username); err == nil {
				t.Errorf("ParseEndpoint(%q,%q,%q) accepted, want error", c.host, c.port, c.username)
			}
		})
	}
}
