package queue

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// Verification is the CT-logs-direct point-check (spec §5, map #854, ADR-0027): it confirms
// that ONE specific certificate is logged in CT. It complements the drift tail — the tail
// catches certificates in CT the operator did not expect, verification catches certificates
// the operator observed that are NOT in CT (an internal CA, or evasion). It is NOT a Scan
// (§5.2): it has no schedule and no scope fan-out, mints no subject and stores no durable
// result. It runs two ways — auto, once per newly-captured certificate the moment a
// measurement lands it (autoVerifyCerts), and on demand for one fingerprint
// (VerifyByFingerprint) — and both reduce to verifyMaterial: recompute the RFC 6962 leaf hash
// from the captured bytes and ask the correct log or shard whether that leaf is included.
//
// The design invariant is that verification only POINT-CHECKS and never enumerates: it always
// starts from an SCT or the certificate bytes, never a name (§5.1). The pure half — SCT
// parsing, the precert reconstruction, the leaf hash, the inclusion recompute and the log
// lookup — is internal/scan/ctverify.go; this file holds the impure half: the log fetches and
// the result. No log SIGNATURE is checked (deferred with the tail's, §4.4) — inclusion is
// recomputed to the head's stated root.

// maxAutoVerifyPerJob bounds how many freshly-captured certificates one completed measurement
// job auto-verifies, so a job that presents many distinct certificates does a bounded amount
// of out-of-band log I/O. On-demand verification (VerifyByFingerprint) is unbounded — the
// operator asked for exactly one.
const maxAutoVerifyPerJob = 8

// WithCTVerify wires the verification fetcher onto the Worker. It is separate from NewWorker so
// the measurement-only construction stays unchanged; a worker with no verify fetcher does not
// auto-verify and refuses an on-demand re-check, so opting out costs nothing. It reuses the
// CTFetcher seam — the same distinctive User-Agent and unfollowed-redirect (SSRF) guard as the
// crt.sh and tail fetchers — pointed at the RFC 6962 and static-ct-api read endpoints.
func (w *Worker) WithCTVerify(fetcher CTFetcher) *Worker {
	w.ctVerifyFetcher = fetcher
	return w
}

// VerifyOutcome is the ephemeral result of a point-check (spec §5.4): logged, not-logged, or
// unverifiable. NOT-logged is the notable signal — a certificate the operator observed that no
// reachable CT log holds. Unverifiable is neither logged nor absent: no usable SCT, an unknown
// log, or a log we could not reach — never a false NOT-logged.
type VerifyOutcome int

const (
	// VerifyUnverifiable is the fail-closed default: the check could not resolve to logged or
	// not-logged, so it asserts neither.
	VerifyUnverifiable VerifyOutcome = iota
	// VerifyLogged means a reachable CT log proved the certificate's leaf is included.
	VerifyLogged
	// VerifyNotLogged means a reachable CT log definitively does not hold the certificate's
	// leaf, and none holds it — the certificate is not in CT.
	VerifyNotLogged
)

// VerifyResult is one certificate's point-check outcome and a redacted, human phrase. It is
// ephemeral: it is surfaced on the live stream and returned to an on-demand caller, and never
// persisted (§5.1).
type VerifyResult struct {
	Outcome VerifyOutcome
	Reason  string
}

// logCheck is the result of asking one log about one leaf hash.
type logCheck int

const (
	checkErrored  logCheck = iota // transient: unreachable, malformed, or an unverifiable proof
	checkNotFound                 // the log does not hold this leaf
	checkIncluded                 // the log proved this leaf is included
)

// VerifyByFingerprint runs an on-demand point-check for one captured certificate (spec §5.4).
// It reads the certificate's material back from the side store — the leaf DER, the out-of-cert
// SCTs and the issuer SPKI — and verifies it. A fingerprint the store never captured is
// unverifiable, not an error. It is the operator-triggered entry point the Sources-UI re-check
// (#880/#881) calls; #878 provides the capability, not its UI trigger.
func (w *Worker) VerifyByFingerprint(ctx context.Context, fingerprint string) (VerifyResult, error) {
	if w.ctVerifyFetcher == nil {
		return VerifyResult{}, fmt.Errorf("queue: verification not configured")
	}
	m, err := w.q.GetCertificateMaterial(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return VerifyResult{Outcome: VerifyUnverifiable, Reason: "certificate not captured"}, nil
		}
		return VerifyResult{}, fmt.Errorf("queue: read certificate material: %w", err)
	}
	logs, err := scan.AllLogs()
	if err != nil {
		return VerifyResult{}, err
	}
	return w.verifyMaterial(ctx, logs, m.Der, m.Scts, m.IssuerSpki), nil
}

// autoVerifyCerts verifies each certificate a completed job captured, out of band and
// best-effort (spec §5.4). It is a no-op unless the worker was built WithCTVerify. It runs
// AFTER the terminal tx (its caller commits first), so its log fetches never ride a database
// transaction, and it bounds itself to maxAutoVerifyPerJob so a cert-heavy job does bounded
// I/O. Each result rides the originating job's live-stream line (emitVerifyEvent); a verify
// failure never fails the job.
func (w *Worker) autoVerifyCerts(ctx context.Context, job db.ClaimJobRow, obs []wire.Observation) {
	if w.ctVerifyFetcher == nil {
		return
	}
	logs, err := scan.AllLogs()
	if err != nil {
		w.log.Printf("worker: ct-verify job %d log list: %v", job.ID, err)
		return
	}
	verified := 0
	for _, o := range obs {
		if o.CertMaterial == nil {
			continue
		}
		if verified >= maxAutoVerifyPerJob {
			break
		}
		verified++
		res := w.verifyMaterial(ctx, logs, o.CertMaterial.DER, o.CertMaterial.SCTs, o.CertMaterial.IssuerSPKI)
		w.emitVerifyEvent(ctx, job, res)
	}
}

// verifyMaterial is the shared point-check for both trigger paths (spec §5.1). It gathers every
// SCT the certificate carries — embedded (signed over the precertificate), and out-of-cert from
// the TLS extension and the OCSP staple (signed over the final certificate) — computes the RFC
// 6962 leaf hash for each, and asks the log the SCT names whether that leaf is included. The
// first inclusion is a LOGGED result; if some log was unreachable the result is unverifiable
// (never a false not-logged); a definitive absence with no error is NOT-logged; no usable SCT
// is unverifiable.
func (w *Worker) verifyMaterial(ctx context.Context, logs []scan.CTLog, leafDER, sctsBlob, issuerSPKI []byte) VerifyResult {
	capture, _ := wire.DecodeSCTCapture(sctsBlob)
	type taggedSCT struct {
		raw     []byte
		precert bool
	}
	var scts []taggedSCT
	if embedded, err := scan.EmbeddedSCTs(leafDER); err == nil {
		for _, e := range embedded {
			scts = append(scts, taggedSCT{raw: e, precert: true})
		}
	}
	for _, e := range capture.TLSExt {
		scts = append(scts, taggedSCT{raw: e, precert: false})
	}
	if ocsp, err := scan.OCSPSCTs(capture.OCSP); err == nil {
		for _, e := range ocsp {
			scts = append(scts, taggedSCT{raw: e, precert: false})
		}
	}
	if len(scts) == 0 {
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "no SCTs presented"}
	}

	// The precert TBS is reconstructed at most once and only when an embedded SCT needs it.
	var precertTBS []byte
	precertReady := false
	precertTBSOK := true

	usable, notFound, errored := false, false, false
	for _, tagged := range scts {
		sct, err := scan.ParseSCT(tagged.raw)
		if err != nil {
			continue
		}
		lg, ok := scan.FindLogByLogID(logs, sct.LogID)
		if !ok {
			continue // an SCT from a log the embedded list does not carry
		}
		var leafHash []byte
		if tagged.precert {
			if len(issuerSPKI) == 0 {
				continue // an embedded SCT needs the issuer key, which was not captured
			}
			if !precertReady {
				precertTBS, err = scan.PrecertTBS(leafDER)
				precertReady = true
				precertTBSOK = err == nil
			}
			if !precertTBSOK {
				continue
			}
			leafHash = scan.LeafHashPrecert(scan.IssuerKeyHash(issuerSPKI), precertTBS, sct.Extensions, sct.Timestamp)
		} else {
			leafHash = scan.LeafHashX509(leafDER, sct.Extensions, sct.Timestamp)
		}
		usable = true
		switch w.checkOneLog(ctx, lg, leafHash, sct) {
		case checkIncluded:
			return VerifyResult{Outcome: VerifyLogged, Reason: "logged in CT"}
		case checkNotFound:
			notFound = true
		case checkErrored:
			errored = true
		}
	}

	switch {
	case errored:
		// A log we could not reach or whose proof did not verify: the certificate may be
		// logged where we could not check, so this is never a not-logged (§5.4).
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "CT log unreachable"}
	case notFound:
		return VerifyResult{Outcome: VerifyNotLogged, Reason: "not logged in CT"}
	case !usable:
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "no usable SCT"}
	default:
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "no CT log matched"}
	}
}

// checkOneLog asks one log whether the leaf is included, with the client the log's shape names:
// the RFC 6962 get-proof-by-hash for a dynamic log, the static-ct-api hash tile for a tiled
// one (spec §5.1).
func (w *Worker) checkOneLog(ctx context.Context, lg scan.CTLog, leafHash []byte, sct scan.SCT) logCheck {
	if lg.Tiled {
		return w.checkTiled(ctx, lg, leafHash, sct)
	}
	return w.checkRFC(ctx, lg, leafHash)
}

// checkRFC verifies inclusion against an RFC 6962 log (§5.1): fetch get-sth for the tree size
// and root, ask get-proof-by-hash for the audit path, and recompute the root from the leaf hash
// and path. A 400/404 is the log saying it does not hold the hash (not-logged); a proof that
// does not recompute to the head's root is treated as inconclusive, never as inclusion. No log
// signature is checked — the root compared against is the one the log itself served (§4.4).
func (w *Worker) checkRFC(ctx context.Context, lg scan.CTLog, leafHash []byte) logCheck {
	base := ensureTrailingSlash(lg.URL)
	status, body, err := w.ctVerifyFetcher.Fetch(ctx, base+"ct/v1/get-sth")
	if err != nil || status != 200 {
		return checkErrored
	}
	sth, err := scan.ParseSTH(body)
	if err != nil {
		return checkErrored
	}
	root, err := scan.STHRoot(body)
	if err != nil {
		return checkErrored
	}
	hash := url.QueryEscape(base64.StdEncoding.EncodeToString(leafHash))
	proofURL := fmt.Sprintf("%sct/v1/get-proof-by-hash?hash=%s&tree_size=%d", base, hash, sth.TreeSize)
	status, body, err = w.ctVerifyFetcher.Fetch(ctx, proofURL)
	if err != nil {
		return checkErrored
	}
	if status == 400 || status == 404 {
		return checkNotFound // the log does not hold this leaf hash
	}
	if status != 200 {
		return checkErrored
	}
	index, auditPath, err := scan.ParseProofByHash(body)
	if err != nil {
		return checkErrored
	}
	if scan.VerifyInclusion(leafHash, index, sth.TreeSize, auditPath, root) {
		return checkIncluded
	}
	return checkErrored // a proof that does not verify is inconclusive, not proof of absence
}

// checkTiled confirms inclusion against a static-ct-api (tiled) log (§5.1). A tiled log serves
// no get-proof-by-hash; its SCT carries the leaf's index, so the check reads the leaf's slot in
// the level-0 hash tile and confirms it holds this leaf hash. An index beyond the checkpoint's
// tree is not-logged; a matching slot is inclusion; a differing slot is not-logged (the log's
// leaf at the SCT-named index is a different certificate). The full audit-path recompute to the
// checkpoint root is deferred to fog with #877's inclusion proof (§4.4).
func (w *Worker) checkTiled(ctx context.Context, lg scan.CTLog, leafHash []byte, sct scan.SCT) logCheck {
	base := ensureTrailingSlash(lg.URL)
	index, ok := scan.SCTLeafIndex(sct.Extensions)
	if !ok {
		return checkErrored // a tiled SCT with no leaf_index cannot be located
	}
	status, body, err := w.ctVerifyFetcher.Fetch(ctx, base+"checkpoint")
	if err != nil || status != 200 {
		return checkErrored
	}
	sth, err := scan.ParseCheckpoint(body)
	if err != nil {
		return checkErrored
	}
	if index >= sth.TreeSize {
		return checkNotFound // the SCT names an index the checkpoint's tree does not yet reach
	}
	tileBase := (index / scan.CTTileWidth) * scan.CTTileWidth
	width := int64(scan.CTTileWidth)
	if tileBase+scan.CTTileWidth > sth.TreeSize {
		width = sth.TreeSize - tileBase
	}
	tileURL := base + "tile/" + scan.HashTilePath(index)
	if width < scan.CTTileWidth {
		tileURL += fmt.Sprintf(".p/%d", width)
	}
	status, tileBody, err := w.ctVerifyFetcher.Fetch(ctx, tileURL)
	if err != nil || status != 200 {
		return checkErrored
	}
	hashes, err := scan.ParseHashTile(tileBody)
	if err != nil {
		return checkErrored
	}
	matches, present := scan.LeafHashInTile(leafHash, index, hashes)
	if !present {
		return checkErrored // the head tile has not grown to this slot yet
	}
	if matches {
		return checkIncluded
	}
	return checkNotFound
}

// emitVerifyEvent surfaces one point-check result on the originating job's live-stream line
// (spec §5.4), so the operator sees a certificate's CT status beside the measurement that
// captured it. NOT-logged rides at `warn` — the notable signal — and logged at the plain level.
// An unverifiable result is not surfaced: it is a non-result, and surfacing it would only add
// noise. The text is a fixed redacted phrase — never the certificate, its names or its
// fingerprint (#780). It emits on the pool, not a job tx, because verification runs after the
// job's terminal transaction has committed.
func (w *Worker) emitVerifyEvent(ctx context.Context, job db.ClaimJobRow, res VerifyResult) {
	var level string
	switch res.Outcome {
	case VerifyLogged:
		level = ""
	case VerifyNotLogged:
		level = "warn"
	default:
		return // unverifiable: nothing to surface
	}
	w.emitProgress(ctx, w.q, jobProgress{
		Dispatch: job.DispatchID.Int64,
		Job:      job.ID,
		Level:    level,
		Text:     "CT verification · " + res.Reason,
	})
}
