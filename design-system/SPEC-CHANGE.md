# Spec-change requests — the collision protocol

Written 2026-08-24, alongside PARITY-CHART.md. This file exists to prevent the class of drift the v3 port produced.

## What went wrong

The port ran a local doctrine — "the domain term wins and the visual convention gets re-skinned around it"; "fabricated mock data is re-skinned to honest current-state facts + empty-states" — and applied it as a port-side judgment call. Each individual call was well-reasoned and well-documented, and the sum was a console that no longer matches the design: severity gone from four screens, spec regions replaced by placeholder empty-states, the shell trimmed. Nobody upstream ever saw the collisions to rule on them.

## The protocol

**A domain–spec collision is a design question, never a port-side decision.**

1. When implementing a spec region and the domain lacks a datum it renders (or a domain rule appears to forbid its shape), STOP work on that region. Do not re-skin, reshape, empty-state, drop, or add anything.
2. File the collision here, one entry in the log below: spec file + region, the domain fact in the way (with its source — CONTEXT.md line, ADR number), and the smallest honest alternative you would propose.
3. **Notify the operator and hand off.** Print the banner and the filled hand-off prompt from §Stop and escalate below, then end the work item (park the worktree; other items may continue). The operator pastes the prompt into the design workspace.
4. Design answers one of three ways: **build the datum** (schema/derivation work gets specced and charted), **spec changes** (design updates the .jsx + screenshot; the new spec is binding), or **region deferred** (removed from the spec until the datum exists — also a spec edit, never a silent hole).
5. Until answered, the spec stands and the region waits. An unanswered collision never ships as an improvisation.

Vocabulary is the one standing exception — signal / seed / channel / vantage / annotation, withdrawn never resolved — enforced in copy without a request.

## Stop and escalate (for Claude Code / Wayfinder)

Add this to the repo's CLAUDE.md verbatim so it binds every session:

```
## Design decisions on the web UI
The design package (design-system/) is normative for look AND functionality.
If a work item needs ANY design decision — a domain–spec collision, an
unspecced state, a missing datum, an ambiguity between spec file and
screenshot — do NOT decide, work around, or approximate. Instead:
1. Stop that work item immediately (other items may continue).
2. Append the collision to design-system/SPEC-CHANGE.md §Collision log
   with Ruling = "AWAITING DESIGN".
3. Print, as the final output of the item, the DESIGN DECISION NEEDED
   banner and the filled hand-off prompt from SPEC-CHANGE.md, so the
   operator can paste it into the design workspace.
4. Treat the item as blocked until the operator lands the updated design
   package containing the ruling.
```

Banner + hand-off prompt template (fill every ⟨⟩; keep it under ~200 words so it pastes cleanly):

```
┏━━ DESIGN DECISION NEEDED — work item ⟨id⟩ stopped ━━┓
Paste the prompt below into the Verge ASM design workspace.
┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛

Design decision needed on ⟨screen · region⟩ (SPEC-CHANGE.md collision #⟨n⟩).
Spec: ⟨examples/console/File.jsx · lines/region⟩, screenshot ⟨NN-console.jpg⟩.
Blocked work item: ⟨PARITY-CHART/WORK-CHART id⟩.
The collision: ⟨one sentence — what the spec renders vs. what the domain
holds, citing the domain source (CONTEXT.md / ADR-nnnn)⟩.
Options we see: (a) build the datum: ⟨what data/derivation⟩; (b) change the
spec: ⟨smallest honest alternative⟩; (c) defer the region.
Constraint(s): ⟨anything binding — migration cost, privacy, perf⟩.
Please rule, update the design + screenshot if the spec changes, and
re-export the package; we resume from the new PARITY/SPEC-CHANGE entry.
```

## Why this direction

The design workspace is where look and functionality are decided, reviewed, and signed off as one artifact. A port that "corrects" the spec locally splits that authority: the code becomes a second source of truth that only its comments know about. The protocol keeps the authority in one place and turns every collision into a recorded decision instead of a buried judgment call.

## Collision log

| # | Spec region | Domain fact | Proposed | Ruling |
| --- | --- | --- | --- | --- |
| 1 | Severity across Signals, Dashboard, Graph, Search | CONTEXT.md: "a signal carries no severity" | Presence-only re-skin (shipped, unapproved) | Build the datum — PARITY-CHART P0.1 |
| 2 | Dashboard coverage + severity cards | Denominators live on Coverage; no ramp | Empty-state pointers (shipped, unapproved) | Restore per spec — P2.1 |
| 3 | Reports trend KPIs, MTTR, heatmap | No trend series carried | Honest scalars (shipped, unapproved) | Build the series — P0.3, P2.4 |
| 4 | Exposure stat deltas | Exposure is a current-state census | Drop deltas (shipped, unapproved) | Build vs-last-batch deltas — P0.2, P2.6 |
| 5 | Search Documentation group | No content store (#316) | Drop the group (shipped, unapproved) | Index docs/guides/ — P2.5 |
| 6 | Signals Export CSV | — (was never a collision; the button is in Signals.jsx) | — | Specced: exports the current tab's filtered rows; see Signals.jsx header |
| 7 | Dashboard Vantages card — per-vantage latency in ms (Dashboard.jsx VANTAGES "34ms"/"51ms"; PARITY-CHART P0.4/P2.1) | No latency is measured or stored anywhere: the `vantage` table has no latency column (db/migrations/18700), no observation/facet carries a round-trip, and the worker prober connect is itself unlanded — cmd/worker/vantages.go: "the host key is pinned on the first real connect a later ticket wires." So P0.4's cmd/web-only scope has no datum to read and no write path to populate. | Build the datum in a dedicated measurement ticket: capture round-trip latency on the prober connect and store it on the vantage (nullable), which the probers.go read then surfaces. Not derivable port-side from cmd/web. | AWAITING DESIGN |
| 8 | Dashboard Certs-expiring ≤30d stat cell (Dashboard.jsx) + its vs-last-batch delta (#443, P0.2) | The `certificate` facet value stores only `{outcome, chain[]fingerprints}` (connectoutcome/tls.go, InsecureSkipVerify — fingerprints, never the parsed leaf). No `notAfter` is captured, so `CertDetails.Expiring` is always nil and the `certificate-expiring` rule is structurally not-evaluable (signal/endpoint.go; ListEndpointCertificates comment). A ≤30d count needs each cert's expiry, which the TLS corpus does not hold. | Build the datum in a dedicated measurement ticket: capture the leaf's `notAfter` in the tls-handshake leaf (a CertVersion bump, re-locking certcorpus) or a new `certificate-validity` facet, then count endpoints expiring within 30d. Not derivable from cmd/web, and out of P0.4's cmd/web-only scope. | AWAITING DESIGN |
