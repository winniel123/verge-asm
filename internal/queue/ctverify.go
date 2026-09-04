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

// Each auto-verify is an out-of-band fetch from a third-party log, so the count is capped.

const maxAutoVerifyPerJob = 8

func (w *Worker) WithCTVerify(fetcher CTFetcher) *Worker {
	w.ctVerifyFetcher = fetcher
	return w
}

type VerifyOutcome int

const (
	VerifyUnverifiable VerifyOutcome = iota
	VerifyLogged
	VerifyNotLogged
)

// Nothing is minted and nothing stored, so verification is not a Scan (ct-source-replacement §5.2).

type VerifyResult struct {
	Outcome VerifyOutcome
	Reason  string
}

type logCheck int

const (
	checkErrored logCheck = iota
	checkNotFound
	checkIncluded
)

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

func (w *Worker) autoVerifyCerts(ctx context.Context, job db.ClaimJobRow, obs []wire.Observation) {
	// A log fetch must never ride the job transaction, so the caller commits before calling this.
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

func (w *Worker) verifyMaterial(ctx context.Context, logs []scan.CTLog, leafDER, sctsBlob, issuerSPKI []byte) VerifyResult {
	// This can only point-check: no CT protocol answers a query by name (ct-source-replacement §5.1).
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
			continue
		}
		var leafHash []byte
		if tagged.precert {
			if len(issuerSPKI) == 0 {
				continue
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
		// An unreachable log may hold the leaf, so this is never a not-logged verdict.
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "CT log unreachable"}
	case notFound:
		return VerifyResult{Outcome: VerifyNotLogged, Reason: "not logged in CT"}
	case !usable:
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "no usable SCT"}
	default:
		return VerifyResult{Outcome: VerifyUnverifiable, Reason: "no CT log matched"}
	}
}

func (w *Worker) checkOneLog(ctx context.Context, lg scan.CTLog, leafHash []byte, sct scan.SCT) logCheck {
	if lg.Tiled {
		return w.checkTiled(ctx, lg, leafHash, sct)
	}
	return w.checkRFC(ctx, lg, leafHash)
}

func (w *Worker) checkRFC(ctx context.Context, lg scan.CTLog, leafHash []byte) logCheck {
	// The root is the one the log served, so no signature is checked (ct-source-replacement §4.4).
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
		return checkNotFound
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
	return checkErrored
}

func (w *Worker) checkTiled(ctx context.Context, lg scan.CTLog, leafHash []byte, sct scan.SCT) logCheck {
	// A tiled log serves no proof endpoint, so inclusion is a slot compare in its hash tile.
	base := ensureTrailingSlash(lg.URL)
	index, ok := scan.SCTLeafIndex(sct.Extensions)
	if !ok {
		return checkErrored
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
		return checkNotFound
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
	// A slot compare is not the audit-path recompute, which stays deferred.
	matches, present := scan.LeafHashInTile(leafHash, index, hashes)
	if !present {
		return checkErrored
	}
	if matches {
		return checkIncluded
	}
	return checkNotFound
}

func (w *Worker) emitVerifyEvent(ctx context.Context, job db.ClaimJobRow, res VerifyResult) {
	var level string
	switch res.Outcome {
	case VerifyLogged:
		level = ""
	case VerifyNotLogged:
		level = "warn"
	default:
		return
	}
	w.emitProgress(ctx, w.q, jobProgress{
		Dispatch: job.DispatchID.Int64,
		Job:      job.ID,
		Level:    level,
		Text:     "CT verification · " + res.Reason,
	})
}
