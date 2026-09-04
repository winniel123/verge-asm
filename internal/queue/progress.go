package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/winniel123/verge-asm/internal/db"
)

// A log-tail listener would otherwise be woken by every dispatch, so this is not the wake channel.
// sqlc cannot bind a channel name, so NotifyJobProgress repeats this literal and must move with it.

const ProgressChannel = "queue_job_progress"

type jobProgress struct {
	Dispatch int64  `json:"dispatch"`
	Job      int64  `json:"job"`
	Level    string `json:"level"`
	Text     string `json:"text"`
}

// Only the emit site knows a string is safe, so scrubbing an opaque error later would fail open.

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

func countLabel(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func (w *Worker) emitJobEvent(ctx context.Context, qtx *db.Queries, job db.ClaimJobRow, level, text string) {
	w.emitProgress(ctx, qtx, jobProgress{
		Dispatch: job.DispatchID.Int64,
		Job:      job.ID,
		Level:    level,
		Text:     text,
	})
}

func (w *Worker) emitProgress(ctx context.Context, qtx *db.Queries, ev jobProgress) {
	// An event persists nowhere at rest, so the job's state-derived log stands (raw-job-output §6.2).
	payload, err := json.Marshal(ev)
	if err != nil {
		// Progress is best-effort, so a failure here must never cost the job its outcome.
		w.log.Printf("worker: progress marshal job %d: %v", ev.Job, err)
		return
	}
	if err := qtx.NotifyJobProgress(ctx, string(payload)); err != nil {
		w.log.Printf("worker: progress notify job %d: %v", ev.Job, err)
	}
}
