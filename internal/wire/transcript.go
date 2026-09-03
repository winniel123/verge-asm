package wire

import "time"

// Nothing derives from a transcript: the worker writes it, one debug handler reads it (#838).

type Transcript interface { // a closed union, never a record with optional fields (CONTEXT.md)
	Frame() TranscriptFrame
	isTranscript()
}

type TranscriptFrame struct {
	QueueJobID int64 // the queue_job grain: one row per attempt, so a retry keeps its own
	Kind       string
	Duration   time.Duration
	CapturedAt time.Time // stamped at capture, so the §4 duration dial retires it by age
}

func (f TranscriptFrame) Frame() TranscriptFrame { return f }

// The transcript rides every outcome: raw output matters most when the job failed (§2.2).

type ProbeResult struct {
	Observations []Observation
	Transcript   Transcript // nil is absent — no capture — never a captured-but-empty one
}

type ProberTranscript struct {
	TranscriptFrame
	SentScope      []byte // the verbatim stdin bytes, never a re-encoded struct (§2.3)
	Stdout         []byte // verbatim, captured before the #773 scope re-gate
	Stderr         []byte
	Outcome        ProberOutcome
	StdoutOverflow bool // a capture signal the persist step reads, never a stored field (§3.2)
}

// The marker sits per variant, never on the promoted frame, or an outside type could join.

func (ProberTranscript) isTranscript() {}

// A ctx-killed prober reads as cancelled, never a fake exit code 0 (§1.2).

type ProberOutcome interface{ isProberOutcome() }

type ProberExited struct{ Code int }

type ProberSignalled struct{ Signal string }

type ProberContextCancelled struct{}

func (ProberExited) isProberOutcome()           {}
func (ProberSignalled) isProberOutcome()        {}
func (ProberContextCancelled) isProberOutcome() {}

type CTTranscript struct {
	TranscriptFrame
	RequestURL   string
	ResponseBody []byte
	Outcome      CTOutcome
}

func (CTTranscript) isTranscript() {}

type CTOutcome interface{ isCTOutcome() }

type CTHTTP struct{ Status int }

type CTTransportError struct{ Text string }

type CTContextCancelled struct{}

func (CTHTTP) isCTOutcome()             {}
func (CTTransportError) isCTOutcome()   {}
func (CTContextCancelled) isCTOutcome() {}

// The zone file is not stored here: the operator's supplied zone-file row holds it (§1.3).

type ZoneTranscript struct {
	TranscriptFrame
	Restated int
	Skipped  []string
	Outcome  ZoneOutcome
}

func (ZoneTranscript) isTranscript() {}

type ZoneOutcome interface{ isZoneOutcome() }

type ZoneParsed struct{}

type ZoneDecodeError struct{ Text string }

func (ZoneParsed) isZoneOutcome()      {}
func (ZoneDecodeError) isZoneOutcome() {}
