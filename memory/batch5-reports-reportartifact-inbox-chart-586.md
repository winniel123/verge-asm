---
name: batch5-reports-reportartifact-inbox-chart-586
description: Batch-5 (screens 16-18) map #586 — 16/17 BUILT+held in PR #592 for sign-off; 18 Inbox BLOCKED on design collision #24.
metadata:
  type: project
---

Batch 5 of the design-parity screen conversion (WORKFLOW v4), package **v3.11.0**, Wayfinder map **#586** from `design-system/verify/WORK-ORDER-16-18-BATCH5.md`. Execution-mode map (overrides plan-don't-do, like all prior batches). Built AFK 2026-08-25 on one integration branch `batch5-integration`.

**Status 2026-08-25 (partial batch):**
- ✅ #587 Foundation — `a5ca1dd`: committed v3.11.0 export (3 tmpls + fixtures + states) embedded inert via designfs `templates/*.tmpl` glob; restored SPEC-CHANGE #20/#21/#22 + transcribed #23; G1SUMS regenerated (143 artifacts).
- ✅ #588 Reports (screen 16) — `1426b3e`: `/reports` byte-served; #23a–f wired incl. real `format=pdf`→RenderArtifactPDF. G1 OK, G2 all states ≤0.28%.
- ✅ #589 ReportArtifact (screen 17) — `cf43f39`: new `internal/message/artifactdoc.go` re-points RenderArtifact at the design-owned "artifactdoc" define (one HTML markup across on-screen/email/preview from one parsed bundle: reportartifact+reports+signals+drift). G1 OK, G2 both states 0.000%.
- ⛔ #590 Inbox (screen 18) — **BLOCKED on collision #24 (AWAITING DESIGN)**, no code written.
- #591 Serial-merge — rescoped to 16/17 (held in PR #592 for sign-off); 18's sign-off follows its later PR.
- **PR #592** (base main) holds Foundation+16+17, **awaiting operator visual sign-off** — do NOT merge without explicit go.

**Collision #24 (the block):** ruling #23i binds inbox `.Selected.Body` to "the same prose internal/message produces for delivery," but on the corpus it does NOT resolve — internal/message emits only a valence-free `Message.Headline` (no body producer), `internal/delivery.BuildBody` carries that headline + a census count (never prose, notification-channels.md §3.1), and the fixture body contains "critical" — a `message.ValenceWords` term `TestNoValenceInRenderedCopy` (ADR-0064) guarantees the vocabulary NEVER emits. So the same text is unproduceable from a real read. Operator chose to **rule it in the design workspace** (options: a=build a message-detail prose grammar + ADR-0064 carve-out; b=`.Body`=headline; c=defer/drop the hole) → needs a package re-export; screen 18 converts on the new version. This is a case where "honestly omit absent fields" (batch-3/4) did NOT apply — the ruling contradicted an ADR-guarded invariant, so escalation (not build-to-reality) was correct per CLAUDE.md §Design decisions.

**fpdf/ADR-0114 nuance (flagged for sign-off, not a collision):** the binary `/reports/delivery/pdf` + `format=pdf` stay on the pre-existing pure-Go `RenderArtifactPDF` (ADR-0114 — fpdf can't execute HTML), fed from the SAME `Artifact` data as artifactdoc. So "one markup three shells" holds for the HTML surfaces; the binary PDF is content-unified but not the literal HTML define.

**Orchestration lessons:** (1) subagents that run the Docker gate via Monitor/run_in_background emit a premature "detached/completed" task-notification, then SELF-RESUME when the gate finishes and complete the commit — always verify real git state (HEAD moved? gate green?) rather than trusting the notification either way. Instruct convert subagents to run gates FOREGROUND. (2) `GOLDENS=write` bleeds other screens' goldens every run — `git checkout` the non-target goldens before commit (recurs every screen). (3) The binding G2 verdict = pinned container `ADVISORY=0 bash run.sh` (exits non-zero over threshold); default ADVISORY=1 only prints diffs and always exits 0.

Predecessor: [[batch4-assetdetail-subjectdetail-graph-chart-579]]. Next after batch-5 completes = screen 19 (SearchResults). Playbook: [[workflow-v4-screen-conversion-playbook]].
