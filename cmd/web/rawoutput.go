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

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/rundetail-raw.tmpl"))

// Attacker-influenceable stdout builds with textContent, never innerHTML (raw-job-output.md §6.4).

type rawOutputBytes struct {
	Stdout    []string `json:"stdout"`
	Stderr    []string `json:"stderr"`
	SentScope string   `json:"sentScope"`
}

type rawOutputView struct {
	RunID   int64
	RunHref string
	JobID   int64
	Kind    string
	Vantage string

	Captured bool
	Variant  string

	ExitCode     string
	Signal       string
	CtxCancelled string
	OutcomeOK    bool
	Duration     string
	CapturedAt   string

	Restated    string
	ZoneOutcome string

	RequestURL string
	CTOutcome  string

	StdoutSize   string
	StdoutCapped bool
	StdoutTrunc  string
	StderrTrunc  string
	SentTrunc    string

	Bytes rawOutputBytes
}

// The Transcript carries secrets the state log cannot, so admin only (raw-job-output.md §5.2).

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

	runBase := strings.TrimSuffix(r.URL.Path, "/raw")
	view := rawOutputView{
		RunID:   runID,
		RunHref: runBase + "?job=" + jobParam,
		JobID:   jobID,
	}

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

	// Written in the job's terminal tx, so no raw stream is tailable (raw-job-output.md §6.2).
	row, err := s.store.GetTranscriptByJob(r.Context(), jobID)
	if errors.Is(err, pgx.ErrNoRows) {
		s.render(w, r, "runraw", s.rawOutputData(acct, view))
		return
	}
	if err != nil {
		s.serverError(w, "raw output: get transcript", err)
		return
	}

	if err := s.fillRawOutputView(&view, row); err != nil {
		s.serverError(w, "raw output: open transcript", err)
		return
	}
	view.Captured = true
	if view.Kind == "" {
		view.Kind = row.Kind
	}
	s.render(w, r, "runraw", s.rawOutputData(acct, view))
}

func (s *server) rawOutputData(acct db.Account, view rawOutputView) map[string]any {
	return map[string]any{
		"Title":     "Raw output · job #" + strconv.FormatInt(view.JobID, 10),
		"Account":   acct,
		"IsAdmin":   acct.Role == roleAdmin,
		"NavActive": "drift",
		"Raw":       view,
	}
}

func (s *server) fillRawOutputView(view *rawOutputView, row db.Transcript) error {
	view.Variant = row.Variant
	view.Duration = time.Duration(row.DurationNs).String()
	if row.CapturedAt.Valid {
		view.CapturedAt = row.CapturedAt.Time.UTC().Format(time.RFC3339)
	}
	notes := rawDecodeTruncation(row.Truncation)

	stdout, err := transcript.Open(s.transcriptKey, row.Stdout)
	if err != nil {
		return fmt.Errorf("open stdout: %w", err)
	}
	view.Bytes.Stdout = rawSplitLines(stdout)
	view.StdoutSize = rawHumanBytes(len(stdout))
	view.StdoutTrunc = notes["stdout"]
	view.StdoutCapped = view.StdoutTrunc != ""

	if row.Variant == "zone" {
		view.Restated, view.ZoneOutcome = rawDecodeZoneOutcome(row.Outcome)
		return nil
	}

	if row.Variant == "ct" {
		view.RequestURL, view.CTOutcome = rawDecodeCTOutcome(row.Outcome)
		return nil
	}

	stderr, err := transcript.Open(s.transcriptKey, row.Stderr)
	if err != nil {
		return fmt.Errorf("open stderr: %w", err)
	}
	sent, err := transcript.Open(s.transcriptKey, row.SentScope)
	if err != nil {
		return fmt.Errorf("open sent scope: %w", err)
	}
	view.Bytes.Stderr = rawSplitLines(stderr)
	view.Bytes.SentScope = string(sent)
	view.ExitCode, view.Signal, view.CtxCancelled, view.OutcomeOK = rawDecodeOutcome(row.Outcome)
	view.StderrTrunc = notes["stderr"]
	view.SentTrunc = notes["sent_scope"]
	return nil
}

func rawSplitLines(b []byte) []string {
	if len(b) == 0 {
		return []string{}
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

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

func rawDecodeZoneOutcome(b []byte) (restated, outcome string) {
	restated, outcome = "—", "—"
	var o struct {
		Kind     string `json:"kind"`
		Restated *int   `json:"restated"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(b, &o); err != nil {
		return restated, outcome
	}
	if o.Restated != nil {
		restated = strconv.Itoa(*o.Restated)
	}
	switch o.Kind {
	case "parsed":
		outcome = "parsed"
	case "decode-error":
		outcome = "decode-error"
		if o.Text != "" {
			outcome += ": " + o.Text
		}
	}
	return restated, outcome
}

func rawDecodeCTOutcome(b []byte) (requestURL, outcome string) {
	requestURL, outcome = "—", "—"
	var o struct {
		Kind    string `json:"kind"`
		Status  *int   `json:"status"`
		Text    string `json:"text"`
		Request string `json:"request_url"`
	}
	if err := json.Unmarshal(b, &o); err != nil {
		return requestURL, outcome
	}
	if o.Request != "" {
		requestURL = o.Request
	}
	switch o.Kind {
	case "http":
		if o.Status != nil {
			outcome = "HTTP " + strconv.Itoa(*o.Status)
		}
	case "transport-error":
		outcome = "transport error"
		if o.Text != "" {
			outcome += ": " + o.Text
		}
	case "context-cancelled":
		outcome = "context-cancelled"
	}
	return requestURL, outcome
}

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
