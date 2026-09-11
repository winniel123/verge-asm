package db

import (
	"strings"
	"testing"
)

// TestOperatorReadsSkipAHeldMessage guards ADR-1806 §2: a message whose census
// is still pending reaches no operator surface. Every operator-facing read of
// the message table — the panel and inbox list, the unread count, and
// mark-all-read — must carry the pending test. One list query feeds the settings
// panel, the inbox and the bell, so one predicate holds all three. Both marks must
// pass over a held row too, or releasing it later shows it already read: the inbox
// takes its id from the URL, so no rendering test bounds what a mark can reach.
func TestOperatorReadsSkipAHeldMessage(t *testing.T) {
	for _, q := range []struct {
		name string
		sql  string
	}{
		{"listMessages", listMessages},
		{"countUnreadMessages", countUnreadMessages},
		{"markAllMessagesRead", markAllMessagesRead},
		{"markMessageRead", markMessageRead},
	} {
		if !strings.Contains(strings.ToLower(q.sql), "census_pending_after_batch is null") {
			t.Errorf("%s must exclude a held message with a census_pending_after_batch IS NULL test (ADR-1806 §2), got:\n%s", q.name, q.sql)
		}
	}
}

// TestMessageReadersCarryThePendingColumn guards the mark the release poll owns.
// A reader that drops the column cannot tell a held row from a released one, and
// sqlc gives such a query a row type of its own rather than Message.
func TestMessageReadersCarryThePendingColumn(t *testing.T) {
	for _, q := range []struct {
		name string
		sql  string
	}{
		{"listMessages", listMessages},
		{"insertMessage", insertMessage},
	} {
		if !strings.Contains(strings.ToLower(q.sql), "census_pending_after_batch") {
			t.Errorf("%s must carry census_pending_after_batch (ADR-1806 §2), got:\n%s", q.name, q.sql)
		}
	}
}

// TestInsertMessageWritesTheFrozenBasis guards ADR-1806 §4. A held row's census
// is computed by the release poll, and "beneath the root" must mean what the fold
// saw. The write path is the only place that value exists, so an insert that drops
// the column leaves the poll reading live resolution instead.
func TestInsertMessageWritesTheFrozenBasis(t *testing.T) {
	if !strings.Contains(strings.ToLower(insertMessage), "census_basis") {
		t.Errorf("insertMessage must carry census_basis (ADR-1806 §4), got:\n%s", insertMessage)
	}
}
