package wire

import "time"

// Transcript is the verbatim record of one job's exchange with its producer — the
// operator-debugging corpus (#838, ADR-0126 draft). It is a CLOSED UNION, one variant
// per producer kind, never a record with optional fields (CONTEXT.md: "every value
// space is a closed union, never a record with optional fields"). Each variant names
// the one exchange it made and carries its OWN typed outcome; there is no shared
// outcome union across variants.
//
// A nil Transcript is ABSENT — no capture — a legible state distinct from a captured
// variant whose byte streams are empty. This ticket (#863) defines the shape only: the
// producer paths return an absent Transcript until #840 (local capture) and #841
// (remote transport) fill the bytes. Nothing derives from a Transcript: only the worker
// writes it, only the §6 handler reads it.
//
// The union is sealed by the unexported isTranscript marker, defined once per variant
// (not on the promoted frame), so no type outside this package can join it.
type Transcript interface {
	// Frame returns the common frame every variant carries.
	Frame() TranscriptFrame
	isTranscript()
}

// TranscriptFrame is the frame every Transcript variant shares: which job produced it
// (queue_job grain — one row per attempt, so a retried job keeps its own transcript),
// the job kind, how long the exchange took, and when it was captured. CapturedAt is
// stamped w.now() at capture, so the §4 duration dial can retire a transcript by age.
//
// A variant embeds TranscriptFrame for the frame fields and its Frame accessor; the
// variant supplies isTranscript itself, so embedding this frame alone does not make an
// outside type a Transcript.
type TranscriptFrame struct {
	QueueJobID int64
	Kind       string
	Duration   time.Duration
	CapturedAt time.Time
}

// Frame returns the frame, so a variant reads its common fields through the Transcript
// interface without a type switch.
func (f TranscriptFrame) Frame() TranscriptFrame { return f }

// ProbeResult is what a producer returns for one job: the observations it decoded and
// the verbatim Transcript of the exchange that produced them. The Transcript rides the
// result on EVERY outcome — success, decode failure, and the error path — because the
// raw output is highest-value exactly when the job failed (§2.2). An absent (nil)
// Transcript means no capture, distinct from a captured-but-empty one.
type ProbeResult struct {
	Observations []Observation
	Transcript   Transcript
}

// --- Prober variant -------------------------------------------------------------

// ProberTranscript is the exchange with an exec'd measurement prober — local now,
// remote across the vantage wire via #841. It carries the exact bytes sent to the
// prober's stdin, and the stdout and stderr it wrote back, each verbatim.
type ProberTranscript struct {
	TranscriptFrame
	// SentScope is the exact bytes written to the prober's stdin (wire.EncodeJobSpec
	// output), verbatim — not a re-encoded struct (§2.3).
	SentScope []byte
	// Stdout is the prober's verbatim stdout: the NDJSON observation stream, captured
	// before the #773 scope re-gate.
	Stdout []byte
	// Stderr is the prober's verbatim stderr: normally empty, non-empty only on a
	// crash (a panic or log.Fatalf).
	Stderr []byte
	// Outcome is how the prober process ended.
	Outcome ProberOutcome
	// StdoutOverflow reports that Stdout tripped the 64 MiB MaxProberStdout memory
	// guard: Stdout then holds only the head bytes the guard retained, and an unknown
	// further amount was never captured. It is an in-process capture signal (producer
	// to persist), not a stored field — the persist step (§3.2) reads it to mark the
	// stored stdout stream memory-guard-tripped rather than plain head+tail truncated.
	StdoutOverflow bool
}

func (ProberTranscript) isTranscript() {}

// ProberOutcome is how an exec'd prober ended: a closed union of three. A ctx-killed
// prober (job timeout or mid-flight cancel) reads as ProberContextCancelled, never a
// fake ProberExited{Code: 0} (§1.2).
type ProberOutcome interface{ isProberOutcome() }

// ProberExited is a normal process exit carrying the exit code.
type ProberExited struct{ Code int }

// ProberSignalled is a process killed by a signal, carrying the signal name.
type ProberSignalled struct{ Signal string }

// ProberContextCancelled is a prober killed by ctx — a job timeout or a mid-flight
// cancel — carrying no code.
type ProberContextCancelled struct{}

func (ProberExited) isProberOutcome()           {}
func (ProberSignalled) isProberOutcome()        {}
func (ProberContextCancelled) isProberOutcome() {}

// --- CT variant -----------------------------------------------------------------

// CTTranscript is the crt.sh HTTP producer exchange: the request URL dialled and the
// verbatim response body. The HTTP status and any transport-error text ride the typed
// outcome, not a separate field.
type CTTranscript struct {
	TranscriptFrame
	// RequestURL is the crt.sh URL dialled (scan.CrtshURL), verbatim.
	RequestURL string
	// ResponseBody is the verbatim HTTP response body.
	ResponseBody []byte
	// Outcome is how the HTTP exchange ended.
	Outcome CTOutcome
}

func (CTTranscript) isTranscript() {}

// CTOutcome is how the crt.sh exchange ended: a closed union of three. The
// transport-error text is the stderr analog for this producer (§1.2).
type CTOutcome interface{ isCTOutcome() }

// CTHTTP is a completed HTTP exchange carrying its status code.
type CTHTTP struct{ Status int }

// CTTransportError is a failure before a status (DNS, dial, TLS) carrying the error
// text.
type CTTransportError struct{ Text string }

// CTContextCancelled is a fetch killed by ctx, carrying no status.
type CTContextCancelled struct{}

func (CTHTTP) isCTOutcome()             {}
func (CTTransportError) isCTOutcome()   {}
func (CTContextCancelled) isCTOutcome() {}

// --- Zone variant ---------------------------------------------------------------

// ZoneTranscript is the zone-restate debug artifact. Zone sends nothing to a prober,
// so the artifact is the restate RESULT: the count restated and, above all, the records
// RestateZone SKIPPED because it could not marshal them ("why is this DNS record missing
// from the estate?" → "we skipped it"). It does NOT store the zone-file bytes — the file
// already sits in the operator's supplied zone-file row (§1.3).
type ZoneTranscript struct {
	TranscriptFrame
	// Restated is the count of records restated.
	Restated int
	// Skipped names the records RestateZone could not marshal and dropped.
	Skipped []string
	// Outcome is whether the restate parsed or hit a decode error.
	Outcome ZoneOutcome
}

func (ZoneTranscript) isTranscript() {}

// ZoneOutcome is how the zone restate ended: a closed union of two.
type ZoneOutcome interface{ isZoneOutcome() }

// ZoneParsed is a restate that parsed the zone file.
type ZoneParsed struct{}

// ZoneDecodeError is a restate that hit a decode error, carrying the error text.
type ZoneDecodeError struct{ Text string }

func (ZoneParsed) isZoneOutcome()      {}
func (ZoneDecodeError) isZoneOutcome() {}
