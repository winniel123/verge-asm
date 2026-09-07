# ADR-0224: The vantage ceiling counts the connections the pacer spaces, and a control port spends one attempt

- **Status:** Accepted
- **Date:** 2026-09-07
- **Tickets:** [#1589 `per_vantage_packets_per_sec` names packets and the pacer counts connects](https://github.com/winniel123/verge-asm/issues/1589), [#1586 Retries multiply a control-port timeout by three, and the blanket discriminator pays it eight times](https://github.com/winniel123/verge-asm/issues/1586)
- **Follows:** [ADR-0137](./0137-the-safety-budget-promises-a-targets-rate-and-enforces-a-vantages.md) §5, which weighed the same rename trade on this same struct and ruled it
- **Constrained by:** [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) (a changed declared parameter moves a leaf's `Version`), [ADR-0104](./0104-an-undiscriminated-reach-is-a-gap-and-a-blanket-responder-is-measured-not-listed.md) (the control probe rides `SafetyProfile`, and a `Gap` is the safe direction of error)
- **Measured by:** [`hot-scan-wall-clock-and-emitted-rate.md`](../research/hot-scan-wall-clock-and-emitted-rate.md) §4.1, findings 4 and 7 of §9

## Context

`SafetyProfile` is the offers document. Every `Batch` records it by content. Two of its statements
did not match the code that reads them.

**Finding 7.** The field `PerVantagePacketsPerSec` carried the JSON tag
`per_vantage_packets_per_sec`, valued at 200. That field becomes `aggregateInterval` inside
`NewPacer`. `Pacer.Next` runs once per `Connect` and once per `Handshake`. The pacer therefore
spaces **connection attempts**. Against a refusing target the two units agree, because one connect
is one SYN. Against a dropping target the kernel retransmits an unanswered SYN below that seam. The
capture in §4.1 counted **72 SYNs for 24 connect attempts**.

The absolute rate stays low in that case. **Nothing measured breaches the declared ceiling.** The
defect is that the unit on the recorded field is not the unit the code enforces.

**Finding 4.** `Retries` is 2, and `ConnectTimeoutMillis` is 3000. An undecided port therefore costs
nine seconds. `discriminateBlanket` calls `Probe` once per control port, with the same profile.
`blanketdiscrim.ControlPortCount` is eight. Eight ports at three attempts of three seconds is 72
seconds. That is the whole measured cost of a dropping address. The research note asks whether a gap
verdict needs three attempts per control port. It does not answer that question. This ADR answers
it.

## Decision

### 1. The per-vantage ceiling **counts connections**, and the field is renamed on the wire

`PerVantagePacketsPerSec` becomes `PerVantageConnPerSec`. The JSON tag
`per_vantage_packets_per_sec` becomes `per_vantage_conn_per_sec`. The value stays 200. The ceiling
is neither raised nor lowered.

The new name states the enforced unit. It also puts the two arms of one pacer in one unit. The
`Pacer.Next` call consults a per-host arm and an aggregate arm together. Both arms gate the same
event. The tag `per_host_conn_per_sec` already named that event correctly. Now the aggregate arm
does too.

**ADR-0137 §5 binds this case and it is not distinguished.** That section renamed
`GlobalPacketsPerSec` on this same struct, in the Go field **and** in the JSON tag. Its rejected
alternative is holding the tag to protect the corpus lock. The ground given there was that the
alternative keeps a name known to be wrong on every `Batch` we will ever record. That ground reaches
this field unchanged.

The other repair is a **re-definition**: make the ceiling count packets and enforce it. That is
refused, because the leaf cannot observe the packets. A kernel SYN retransmission happens below
`net.Dialer.DialContext`, and no userspace count of it exists at that seam. A ceiling the leaf
cannot measure is not a ceiling the leaf can enforce.

**The disclosure that follows is stated rather than smoothed.** After this rename the project
declares **no packet ceiling at all**. It never enforced one. The packets one vantage puts on the
wire can exceed the declared connection rate. §4.1 measured that factor at about three. It came from
one dropping target on one day. That factor is not declared, and this ADR does not declare it.

### 2. A control port spends **one attempt**, and the count is a **declared parameter**

`SafetyProfile` gains `ControlPortRetries`, valued **0**. The control probe spends
`ControlPortRetries + 1` attempts per port. A service port keeps `Retries + 1`, which stays 3.

Three grounds carry this.

**The set already holds the redundancy.** `blanketdiscrim.Decide` reads a disjunction over eight
draws. One `ControlClosed` returns `NotBlanket` on its own. A retry buys a second draw at one port.
The control set buys eight draws at eight ports. ADR-0069's sizing rule put them there. The constant
`ControlPortCount = 8` already carries the comment *eight falsifies with margin*. A per-port retry
adds a ninth through twenty-fourth draw for three times the wall clock.

**A control port is an input to a verdict, not a value on a timeline.** The `Retries` budget exists
for one reason. One dropped SYN must not manufacture a false `not-reached` on a `Service`. A
reachability value is durable, and the corpus row `C3/transient-timeout-recovers` holds that
behaviour. A control port produces no value. It produces one of eight `ControlResult`s that `Decide`
folds into one verdict.

**Fewer attempts can only move the verdict toward `Gap`.** A lost attempt turns `ControlAnswered` or
`ControlClosed` into `ControlIncomplete`. `VerdictBlanket` needs every result answered, so an added
incomplete can never create a `Blanket` the full budget would not have produced. The two reachable
moves are `NotBlanket` to `Gap` and `Blanket` to `Gap`. **ADR-0104 declares that direction the safe
one**: *a false `Gap` withholds one reach, and a false reach fabricates surface.*

The cost of the shorter probe is real, and this ADR names it. One pass can lose a live origin's
refusal in transit on **all eight** control ports. The address then reads undiscriminated. Its
`Service`s then hold a `Gap` that a retry would have made a value. **Nobody has measured that
probability.** The eight-port disjunction bounds it, and the retry budget does not. That is the
argument above rather than a measurement.

### 3. The parameter lives on `SafetyProfile`, and **one `Version` bump pays for both repairs**

`ControlPortRetries` is recorded on the `Batch`, beside `Retries`. ADR-0104's consequence already
rules that *the control-port probe spends packets and rides the existing safety budget*. So the
budget the control probe spends is `connect-outcome`'s declared budget, and the count it spends it at
belongs in the same document.

`connect-outcome` moves from `connect-outcome/v3` to `connect-outcome/v4`. ADR-0021's gate justifies
one bump for a changed declared parameter, and both repairs change one. `SafetyProfile.Digest()`
moves once, and `internal/measure/connectoutcome/corpus/corpus.lock.json` re-blesses once.

**`blanket-discrimination` does not move.** Its own declared parameters are unchanged, and no row of
its corpus moved. ADR-0021's second gate direction refuses a bump that nothing justifies.

## Consequences

- **`connect-outcome/v4` `Break`s every `reachability` timeline once.** The facet vector in
  `internal/queue/spanfold.go` composes this leaf, so the version move breaks the reach half of the
  estate. That is the model's own price for a correction, and ADR-0104 already paid it once for this
  facet.
- **No golden `.ndjson` moved, and `corpus_digest` did not move.** That is the evidence that no
  measured output moved. The connect-outcome corpus renders through `RunWithConnector`, which runs
  no control probe. The `blanketdiscrim` corpus scripts a constant result per target, so its rows
  render the same at one attempt as at three.
- **A dropping address costs about a third of what it cost.** The figures below separate what was
  measured from what is arithmetic over it.

  | Figure | Value | Status |
  | --- | --- | --- |
  | One dropping address, before | 72.05 s | measured, §4.1 |
  | One dropping address, after | about 24 s | arithmetic over the measured 3 s timeout and 8 ports |
  | 1024 dropping addresses, before | 20 h 39 min, 86% of the cadence | extrapolated, §4.3 |
  | 1024 dropping addresses, after | about 7 h, 29% of the cadence | extrapolated over a derived per-address figure |
  | The scope size a dropping estate fills the cadence at | about 1,190 → about 3,520 addresses | extrapolated, §5.2 |

- **The dropping-estate cadence margin widens, and no scan was re-run to confirm it.** §4.3's row
  was extrapolated from one measured address before this change, and the row after it inherits every
  assumption in §5.1. A later session that re-runs §3's procedure supersedes both rows.
- **The reopening triggers for ADR-0137 §6's grant are unchanged in kind and one is cheaper.**
  Trigger 2 is a raised address cap over a dropping estate. Its measured cliff moves out, so the
  operator reaches it later. Trigger 1, a second `Vantage`, is untouched.
- **The `Safety budget` glossary term states the unit and the split retry budget.** ADR-0058 requires
  the withdrawal at the site that specifies the superseded sentence. Three documents carried the
  packet unit in the present tense. `CONTEXT.md`, `docs/spec/v1-spec.md` §3.3 and
  `docs/guides/running.md` are each corrected. The research note and the ADRs before this one record
  their own date, so this ADR leaves them alone.
- **`effectiveCadenceSeconds` in `cmd/web/settings.go` is untouched.** It projects a service-port
  sweep and reads `Retries`, which does not move.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Rename the Go field only and hold the JSON tag** | ADR-0137 §5 rejected exactly this on this same struct. It keeps a name known to be wrong on every `Batch` we record, to protect one lock file and two test assertions |
| **Re-define the ceiling to count packets and enforce it** | The leaf cannot observe a kernel SYN retransmission at the `net.Dialer` seam, so it cannot enforce a packet count. It would also be a new safety-budget decision rather than a repair of a wrong record |
| **Lower `Retries` from 2 for every port** | It pays for the control probe out of the value path. A service port's retry budget stops one dropped SYN manufacturing a false `not-reached` on a durable timeline. The cost sits at the control probe, so the repair belongs there |
| **Put the control-port attempt count on `blanketdiscrim.Params`** | `blanketdiscrim.DefaultParams()` reaches no `Batch`. It is read by that leaf's corpus lock alone. A count recorded nowhere is the wrong-record shape both these tickets are instances of, and the `Batch` would still read `retries: 2` over a probe that spends one |
| **Change nothing on either ticket** | For #1589 the record stays wrong on every `Batch`, which is the failure #1092 named. For #1586 the whole measured cost of a dropping address stays, and no ground was found for the second and third attempt |
| **Stop probing control ports once one refuses** | It saves one RST round trip per remaining port on an origin, which is about 20 ms each, and it saves nothing at all in the dropping case this ADR is about. It is out of both tickets' scope |

## Thin ground, flagged rather than smoothed

- **Nothing here is re-measured.** The 72.05 s is measured and the 24 s is arithmetic over the
  measured 3 s timeout and the shipped port count. No scan ran against a dropping estate after this
  change.
- **The false-`Gap` probability is unmeasured.** §2 argues that the eight-port disjunction carries
  the redundancy a retry would carry. That is an argument from the shape of `Decide`, not a measured
  loss rate on a real path.
- **A slow origin is not a modelled case.** A control port that answers after more than 3000 ms
  reads incomplete at one attempt and at three, because each attempt carries the same timeout. The
  retry budget was never the defence against slowness. `ConnectTimeoutMillis` is, and this ADR does
  not move it.
- **The SYN multiplication factor of about three is one capture.** It came from one dropping target,
  from one vantage, on one day. This ADR declares no factor and states only that the packet count is
  larger than the connection count.
