package main

import (
	"html/template"

	designfs "github.com/winniel123/verge-asm/design-system"
)

// Inbox — the V3 primary message surface at `/inbox`, the destination the shell
// bell deep-links to. As of batch 5 · screen 18 this screen is byte-served from the
// frozen design-owned template design-system/templates/inbox.tmpl (package v3.11.1,
// WORKFLOW v4, #590): the old repo-authored markup/CSS const is gone and the tmpl is
// parsed into the shared `tmpl` set here. Its page define "inbox" wires the holes
// .Unread .Filter .AllHref .UnreadHref .Messages[{ID,Read,Selected,Class,Instant,Rel,
// Headline}] and the nullable .Selected{ID,Class,Headline,Rel,Instant,Census[{Kind,
// Key,Href}],Deliveries[{State,ChannelHost,Failed,LastError}],Href,JumpLabel}.
//
// SPEC-CHANGE #24 (ruled, supersedes #23i's .Body): the detail card carries NO prose
// body — internal/message is valence-free by design (ADR-0064) and no prose producer
// is built; the detail form IS the census + delivery receipts, with depth one jump
// away. The census + delivery regions are now design-owned in the tmpl (its own
// .ib-census / .ib-del classes), so the repo's global msgcensus/msgdelivery CSS
// (templates_shell.go) retires for THIS route — it stays only for the /messages
// Settings fold (templates_settings.go) that still renders against it.
//
// The handler (inboxPage / inboxFixtureData, messages.go + devfixtures.go) shapes the
// data: the PRG skeleton is kept (open = ?id link marking read, filter = query param,
// mark-all-read / mark-unread POSTs), and under VERGE_DEV the pinned fixtures.json
// inbox slice is served so the seeded instance renders byte-for-byte what the golden
// composes. inbox.tmpl auto-embeds through designfs's existing templates/*.tmpl glob,
// so no designfs.go change is needed; its "inbox" define calls no cross-tmpl define.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/inbox.tmpl"))
