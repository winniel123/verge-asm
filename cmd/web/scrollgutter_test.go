package main

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// #1086. `scrollbar-gutter` propagates to the viewport from the ROOT element, never
// from `body` (CSS Overflow 3). The shell's `body` is `min-height: 100vh` inside a
// column flex with no `overflow`, so the document scrolls the viewport and a `body`
// declaration reserves nothing. Every centered `*-main` block then re-centers by half
// the 10px scrollbar (tokens/base.css sets its width) when a view crosses the scroll
// threshold, and the whole layout slides ~5px between a tall tab and a short one.

// cssRules returns the shell's top-level rules for one element selector. The shell CSS
// is one declaration block per line, so a line-anchored match is enough here.
func cssRules(page, selector string) []string {
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
