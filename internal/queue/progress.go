package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/winniel123/verge-asm/internal/db"
)

// --- per-job live progress producer (#780, SPEC-CHANGE collision #40 producer half) -------
//
// #771 shipped the live TRANSPORT (the RunDetail long-poll endpoint + client) but not the
// PRODUCER: the worker discarded every per-job event, so the stream only ever re-derived a
// job's bare `state`. This is the producer collision #40 mandated — the worker emits
// EPHEMERAL, redacted per-job progress events over a liveness channel (the queue's own
// LISTEN/NOTIFY), and the RunDetail stream enriches its state-derived log line for the job
// with them. The ruling scopes this to the live transport ONLY: NOTHING is persisted at rest
// (there is no raw-stdout column or table — ADR-0041's corpus separation and the instance-
// privacy posture stand), so on a job's conclusion the persisted state-derived `.Log` stands
// exactly as today. The events are REDACTED exactly as that `.Log` is: a state line carries
// only the job's own state, never prober output, and redactCause holds the same discipline
// for the failure reason it adds — a whitelisted, already-safe cause verbatim, everything
// else a generic phrase.

// ProgressChannel is the LISTEN/NOTIFY channel the worker emits per-job progress on, distinct
// from the queue's own wake channel ("queue_job", notifyChannel): a listener that only wants
// the live log tail (cmd/web) subscribes here without being woken by every dispatch. It is
// exported so the web consumer's LISTEN stays in lock-step with the producer's NOTIFY; the
// NotifyJobProgress query hardcodes the same literal (sqlc cannot bind a channel name).
const ProgressChannel = "queue_job_progress"

type jobProgress struct {
	Dispatch int64  `json:"dispatch"`
	Job      int64  `json:"job"`
	Level    string `json:"level"` // "" | "warn" | "error", the runLogLine levels
	Text     string `json:"text"`
}

// safeCause marks a failure cause whose message is ALREADY redacted and safe to surface in
// the live stream — the CT non-200 ("crt.sh returned HTTP 502") the ticket cites. Every other
// cause is treated as unsafe: a prober exec error carries the prober's stderr, a decode or raw
// net error carries internal detail, and none of it may reach the UI (ADR-0041 / privacy). By
// marking safety at the emit site — where the safety of each string is actually known — rather
// than scrubbing an opaque error after the fact, the default is fail-closed to generic.
type safeCause struct{ msg string }

func (e safeCause) Error() string { return e.msg }

func safeProgress(msg string) error { return safeCause{msg} }

func redactCause(err error) string {
	var s safeCause
	if errors.As(err, &s) {
		return s.msg
	}
	return "measurement failed"
}

func retryLabel(failedAttempt int32, cause error) string {
	return fmt.Sprintf("attempt %d failed · %s · retrying", failedAttempt, redactCause(cause))
}

func deadLetterLabel(attempts int32, cause error) string {
	return fmt.Sprintf("dead-lettered after %d attempts · %s", attempts, redactCause(cause))
}

// countLabel is the redacted text a completion rides: a bare count of what the job produced —
// "12 observations", "3 names admitted" — never the produced data itself.
func countLabel(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// emitJobEvent publishes one ephemeral progress event for a claimed job, filling the run/job
// identity from the row so each call site states only the redacted level and text. The event is
// keyed to the job that experienced the outcome; the stream appends it as a line beside that
// job's state (cmd/web).
func (w *Worker) emitJobEvent(ctx context.Context, qtx *db.Queries, job db.ClaimJobRow, level, text string) {
	w.emitProgress(ctx, qtx, jobProgress{
		Dispatch: job.DispatchID.Int64,
		Job:      job.ID,
		Level:    level,
		Text:     text,
	})
}

// emitProgress publishes one ephemeral progress event inside the job's terminal transaction
// (via qtx, so a job cancelled mid-flight rolls its event back with the rest of its work). It
// is best-effort by contract — a marshal failure is logged and swallowed so it never fails a
// job — but the NOTIFY itself runs on the transaction: pg_notify is a builtin that does not
// fail in practice, and a connection broken enough to fail it would fail the commit anyway.
func (w *Worker) emitProgress(ctx context.Context, qtx *db.Queries, ev jobProgress) {
	payload, err := json.Marshal(ev)
	if err != nil {
		w.log.Printf("worker: progress marshal job %d: %v", ev.Job, err)
		return
	}
	if err := qtx.NotifyJobProgress(ctx, string(payload)); err != nil {
		w.log.Printf("worker: progress notify job %d: %v", ev.Job, err)
	}
}
