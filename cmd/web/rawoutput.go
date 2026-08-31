package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/transcript"
)

// The dedicated admin raw-output view (#866, map #838, spec §6) is the design-owned
// design-system/templates/rundetail-raw.tmpl. It defines "runraw" and reuses the shared
// head/chrome/foot partials, so it parses into the same tmpl set the other design-owned
// templates do. It auto-embeds through designfs's existing templates/*.tmpl glob, so no
// designfs.go change is needed.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/rundetail-raw.tmpl"))

// rawOutputBytes carries the three verbatim streams to the browser as JSON. The view
// builds each stdout/stderr line and the sent scope into the DOM with textContent —
// never innerHTML — so attacker-influenceable pre-gate stdout cannot inject markup
// (spec §5.1, §6.4). html/template JS-escapes this value where it is embedded.
type rawOutputBytes struct {
	Stdout    []string `json:"stdout"`
	Stderr    []string `json:"stderr"`
	SentScope string   `json:"sentScope"`
}

// rawOutputView is one job's Transcript shaped for the dedicated admin view. Captured is
// false for a legible absence (no transcript row) — distinct from a captured-but-empty
// stream, which renders as an empty panel with Captured true. The exec-meta fields are
// meaningful only when Captured.
type rawOutputView struct {
	RunID   int64
	RunHref string // back to the filtered run page (?job={id})
	JobID   int64
	Kind    string
	Vantage string

	Captured bool

	// Exec-meta — the typed prober outcome unpacked into three display cells (exactly
	// one is active; the others read "—"/"false"), plus duration and captured-at.
	ExitCode     string
	Signal       string
	CtxCancelled string
	OutcomeOK    bool // exited(0): render the exit code in the ok colour
	Duration     string
	CapturedAt   string

	StdoutSize   string
	StdoutCapped bool
	StdoutTrunc  string // "" when the stream fit the cap
	StderrTrunc  string
	SentTrunc    string

	Bytes rawOutputBytes
}

// rawOutputPage renders the dedicated, admin-gated view of one job's verbatim Transcript
// (spec §6). The route is requireAdmin, so a viewer never reaches it — the Transcript can
// carry secrets the state-derived log cannot (§5.2). A job with no capture renders a legible
// "no transcript" absence rather than a 404. Post-hoc only: the Transcript is written in the
// job's terminal tx, so there is never a live stream to tail (§6.2).
func (s *server) rawOutputPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := r.PathValue("id")
	runID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		s.renderMissingRun(w, r, acct, raw)
		return
	}
	jobParam := r.URL.Query().Get("job")
	jobID, err := strconv.ParseInt(jobParam, 10, 64)
	if err != nil {
		s.renderMissingRun(w, r, acct, raw)
		return
	}

	// The back link returns to the filtered run page — the bare run route the request
	// hangs off (strip the trailing /raw), carrying the ?job filter through.
	runBase := strings.TrimSuffix(r.URL.Path, "/raw")
	view := rawOutputView{
		RunID:   runID,
		RunHref: runBase + "?job=" + jobParam,
		JobID:   jobID,
	}

	// Best-effort header identity (kind · vantage) from the dispatch's jobs. The Transcript
	// row alone carries the kind, so a miss here is not fatal — the row fills it below.
	if jobRows, jerr := s.store.ListJobsForDispatch(r.Context(), pgtype.Int8{Int64: runID, Valid: true}); jerr == nil {
		for _, j := range jobRows {
			if j.ID == jobID {
				view.Kind = j.Kind
				if j.VantageName.Valid {
					view.Vantage = j.VantageName.String
				}
				break
			}
		}
	}

	row, err := s.store.GetTranscriptByJob(r.Context(), jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		// A legible absence — no capture for this job. Render the empty state.
		s.render(w, r, "runraw", s.rawOutputData(acct, view))
		return
	}
	if err != nil {
		s.serverError(w, "raw output: get transcript", err)
		return
	}

	if err := s.fillRawOutputView(&view, row); err != nil {
		// A decrypt or decode failure is a hard fault — fail closed and loudly, never
		// render a partial or unauthenticated result (transcript.Open's contract, §5.3).
		s.serverError(w, "raw output: open transcript", err)
		return
	}
	view.Captured = true
	if view.Kind == "" {
		view.Kind = row.Kind
	}
	s.render(w, r, "runraw", s.rawOutputData(acct, view))
}

// rawOutputData wraps the view in the render map, with the chrome keys the shared
// head/chrome/foot partials read (Account/IsAdmin/NavActive) and the design-token opt-in
// the inverted log surfaces need.
func (s *server) rawOutputData(acct db.Account, view rawOutputView) map[string]any {
	return map[string]any{
		"Title":        "Raw output · job #" + strconv.FormatInt(view.JobID, 10),
		"Account":      acct,
		"IsAdmin":      acct.Role == roleAdmin,
		"NavActive":    "drift",
		"DesignTokens": true,
		"Raw":          view,
	}
}

// fillRawOutputView opens the three sealed streams with the instance key and decodes the
// outcome and truncation JSON the producer wrote. Any open failure is returned so the
// handler fails closed (§5.3).
func (s *server) fillRawOutputView(view *rawOutputView, row db.Transcript) error {
	stdout, err := transcript.Open(s.transcriptKey, row.Stdout)
	if err != nil {
		return fmt.Errorf("open stdout: %w", err)
	}
	stderr, err := transcript.Open(s.transcriptKey, row.Stderr)
	if err != nil {
		return fmt.Errorf("open stderr: %w", err)
	}
	sent, err := transcript.Open(s.transcriptKey, row.SentScope)
	if err != nil {
		return fmt.Errorf("open sent scope: %w", err)
	}

	view.Bytes = rawOutputBytes{
		Stdout:    rawSplitLines(stdout),
		Stderr:    rawSplitLines(stderr),
		SentScope: string(sent),
	}
	view.StdoutSize = rawHumanBytes(len(stdout))
	view.Duration = time.Duration(row.DurationNs).String()
	if row.CapturedAt.Valid {
		view.CapturedAt = row.CapturedAt.Time.UTC().Format(time.RFC3339)
	}

	view.ExitCode, view.Signal, view.CtxCancelled, view.OutcomeOK = rawDecodeOutcome(row.Outcome)

	notes := rawDecodeTruncation(row.Truncation)
	view.StdoutTrunc = notes["stdout"]
	view.StderrTrunc = notes["stderr"]
	view.SentTrunc = notes["sent_scope"]
	view.StdoutCapped = view.StdoutTrunc != ""
	return nil
}

// rawSplitLines splits a verbatim stream into display lines, dropping a single trailing
// newline so a well-formed NDJSON stream does not render a blank final line. Internal blank
// lines are kept. An empty stream (a captured-but-empty column) yields no lines.
func rawSplitLines(b []byte) []string {
	if len(b) == 0 {
		return []string{}
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// rawDecodeOutcome unpacks the prober outcome JSON the producer wrote — {"kind":"exited",
// "code":N} / {"kind":"signalled","signal":S} / {"kind":"context-cancelled"} (§1.2) — into
// three display cells. Exactly one is active; the others read the "—"/"false" placeholders.
// A ctx-cancelled prober reads as cancelled, never a fake exit 0.
func rawDecodeOutcome(b []byte) (exit, signal, ctx string, ok bool) {
	exit, signal, ctx = "—", "—", "false"
	var o struct {
		Kind   string `json:"kind"`
		Code   *int   `json:"code"`
		Signal string `json:"signal"`
	}
	if err := json.Unmarshal(b, &o); err != nil {
		return exit, signal, ctx, false
	}
	switch o.Kind {
	case "exited":
		if o.Code != nil {
			exit = strconv.Itoa(*o.Code)
			ok = *o.Code == 0
		}
	case "signalled":
		if o.Signal != "" {
			signal = o.Signal
		}
	case "context-cancelled":
		ctx = "true"
	}
	return exit, signal, ctx, ok
}

// rawDecodeTruncation turns the per-stream truncation markers ({"stdout":{"kept":…,
// "dropped":…,"memory_guard_tripped":…}}, §3.2) into human notes, keyed by stream name.
// A stream that fit its cap has no marker and no note.
func rawDecodeTruncation(b []byte) map[string]string {
	out := map[string]string{}
	if len(b) == 0 {
		return out
	}
	var m map[string]struct {
		Kept    int  `json:"kept"`
		Dropped int  `json:"dropped"`
		Mem     bool `json:"memory_guard_tripped"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return out
	}
	for k, v := range m {
		switch {
		case v.Mem:
			out[k] = fmt.Sprintf("memory guard tripped · head kept · %s dropped", rawHumanBytes(v.Dropped))
		case v.Dropped > 0:
			out[k] = fmt.Sprintf("head+tail truncated · %s elided", rawHumanBytes(v.Dropped))
		}
	}
	return out
}

// rawHumanBytes renders a byte count in binary units (KiB/MiB/…) for the exec-meta and
// truncation notes.
func rawHumanBytes(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := int64(n) / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
