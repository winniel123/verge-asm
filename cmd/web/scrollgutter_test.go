package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// The viewport gutter is reserved from the root alone, never from body (CSS Overflow 3, #1086).

func cssRules(page, selector string) []string {
	// The head also inlines the design tokens, so a selector can match more than the shell's rule.
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s\s*\{[^}]*\}`, regexp.QuoteMeta(selector)))
	return re.FindAllString(page, -1)
}

func shellPage(t *testing.T) string {
	t.Helper()
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	return settingsBody(t, ac, base)
}

func TestShellReservesTheScrollbarGutterOnTheRoot(t *testing.T) {
	rules := cssRules(shellPage(t), "html")
	if len(rules) == 0 {
		t.Fatal("the shell declares no html rule. The viewport gutter stays unreserved")
	}
	for _, rule := range rules {
		if strings.Contains(rule, "scrollbar-gutter: stable") {
			return
		}
	}
	t.Fatalf("no html rule reserves the gutter. Got %q", rules)
}

func TestShellDoesNotReserveTheGutterOnBody(t *testing.T) {
	for _, rule := range cssRules(shellPage(t), "body") {
		if strings.Contains(rule, "scrollbar-gutter") {
			t.Fatalf("body still declares scrollbar-gutter. It does nothing there and it hides the real rule: %q", rule)
		}
	}
}
