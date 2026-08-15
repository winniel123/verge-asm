package env

import "testing"

func TestOrDefault(t *testing.T) {
	t.Setenv("VERGE_TEST_SET", "value")

	if got := OrDefault("VERGE_TEST_SET", "fallback"); got != "value" {
		t.Fatalf("OrDefault = %q, want %q", got, "value")
	}
	if got := OrDefault("VERGE_TEST_UNSET", "fallback"); got != "fallback" {
		t.Fatalf("OrDefault = %q, want %q", got, "fallback")
	}
}

func TestRequire(t *testing.T) {
	t.Setenv("VERGE_TEST_REQUIRED", "value")

	got, err := Require("VERGE_TEST_REQUIRED")
	if err != nil || got != "value" {
		t.Fatalf("Require = (%q, %v), want (%q, nil)", got, err, "value")
	}

	if _, err := Require("VERGE_TEST_MISSING"); err == nil {
		t.Fatal("expected an error for a missing key")
	}

	t.Setenv("VERGE_TEST_EMPTY", "")
	if _, err := Require("VERGE_TEST_EMPTY"); err == nil {
		t.Fatal("expected an error for an empty value")
	}
}
