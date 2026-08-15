# ADR-0039: A channel carries the message, never the estate — and a delivery is an operational record

- **Status:** Accepted
- **Date:** 2026-08-15
- **Ticket:** [#119 Which notification channels ship in v1, and what are their delivery semantics?](https://github.com/winniel123/verge-asm/issues/119)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

The notification layer has been named in twelve decisions and specified in none. It holds all
damping ([ADR-0007](./0007-drift-is-a-timeline-of-spans.md)), it holds
[#22](https://github.com/winniel123/verge-asm/issues/22)'s coverage threshold and all routing
([ADR-0008](./0008-derivation-versions-move-on-content.md)), it is where a wrong direction cut gets
corrected cheaply ([ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md),
[ADR-0029](./0029-an-alert-fires-on-a-leg.md)), and
[ADR-0001](./0001-stack-and-runtime.md) deferred the JSON API into it in terms — *"the integration
need here is push, not pull ... served by the map's separate notification-channels question, so
deferring the API costs no integration story"*.

**What fires is fully settled and none of it is reopened here.** Four causes, three classes
partitioning **messages** rather than events; the flagship is the internet `Reach` leg
`not-reached` → `reached` with a census (ADR-0029 §2, §7); the drift class's membership is at the
`Reach`, `Signal` and membership layers and never at the facet layer (ADR-0026); a census is
computed once **at the cause** and is a description, never a `Transition`. This ADR is downstream
of every one of those and moves none of them.

Two constraints bound the answer, and both come from the map rather than from this ticket.

**The instance is a high-value target — its database is a complete, current map of the operator's
attack surface.** A channel that carries message bodies off the instance is carrying that map with
it, to a destination the project does not run and cannot see.

**Whatever authenticates an outbound channel must not re-open the bearer-token bypass
[#6](https://github.com/winniel123/verge-asm/issues/6) closed** — an API token is a bearer
credential over the whole inventory that walks past
[#11](https://github.com/winniel123/verge-asm/issues/11)'s TOTP.

The second constraint is usually cited as though it decides the transport, and it does not. It
kills exactly one option — a **pull** feed, an RSS/Atom/JSON endpoint a reader polls — because a
feed reader cannot hold a session and cannot do TOTP, so a feed needs a bearer credential that
reads the estate, which is #6's refusal verbatim. It says nothing about an **outbound** credential,
which authenticates us to them and grants no read of anything. Pretending otherwise would let the
constraint do work it cannot do.

## Decision

**A `Channel` carries a `Message` and never the estate. There is one outbound transport, it is
one-way, and the store the message already sits in is not a channel. A `Delivery` is an Operational
record: a failed one loses the escalation, never the message, and it never becomes a message of its
own.**

### 1. The store is not a channel, and it cannot fail

Every `Message` is written to the instance and rendered in the interface **unconditionally** — no
configuration, no enable, no routing, no `Delivery`, and no way to turn it off. ADR-0007 already
required this without naming it: *"if operators need to mark a transition seen, that is
notification read-state, not a property of history"*, and read-state needs something to be state
**of**.

This is what makes every other ruling here affordable. The channel is the **escalation**, not the
product: a channel that is misconfigured, disabled, dead or never created loses no message and
hides no fact. It is also what answers the monitoring-the-monitor problem in §5 — there is exactly
one surface that cannot fail, and it is the one the operator logs into.

### 2. One outbound transport: a signed HTTPS POST of the message

A `Channel` is an absolute `https://` URL, an optional secret, and the subset of the three classes
it receives. Delivering a message is one HTTP POST of one JSON document.

- **`https` is required.** Plaintext `http` is refused, with **one** exception: a **loopback**
  destination, where nothing transits. The test is over the **resolved address**, family-matched,
  never over the string `localhost` — the same discipline that keeps the `Custody` gate and
  `Vantage class` from turning on a rendering
  ([ADR-0051](./0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)),
  and for the same reason: a name resolving somewhere else would otherwise buy a plaintext
  exemption for a body that is the operator's attack surface.
- **Redirects are not followed.** [#4](https://github.com/winniel123/verge-asm/issues/4)'s probing
  profile already refuses to follow them; here the reason is sharper, because following one moves
  the destination of the operator's attack surface to a host they never declared. A redirect is a
  delivery failure.
- **The secret signs; it is never sent.** Where a secret is set, the request carries an HMAC-SHA256
  signature over the body and a timestamp. No bearer header, because a bearer sits in the
  receiver's access log and a signature does not.
- **The channel grants no read of the instance.** One-way is a property, not a default: there is no
  callback, no ack channel, no fetch, and nothing the receiver can ask us for. #6's bypass class is
  untouched because no inbound surface is created.

### 3. The body is the message and nothing more

The POST body carries exactly what the in-app message carries — its class, its cause, the key of
the subject or scope it fired at, the instant of the cause, its census counts where it has one, a
stable message identifier, and a link back into the instance. **It carries no rows.** Not the 37
`exposed` services behind a census count, not the address set behind a `resolution` move, not the
evidence behind a `Signal`.

This is the high-value-target constraint discharged in the design rather than deprecated in prose.
An operator's chat history, their receiver's disk and their SaaS log pipeline accumulate a list of
*what happened*, never a reconstructable copy of *what they have*. The operator who wants the rows
follows the link and authenticates.

It is also [#50](https://github.com/winniel123/verge-asm/issues/50)'s rule arriving across a screen
and a channel: the payload is **one computation at the cause, two renderings**. The message is
written once and never recomputed — recomputing it would reach back across a `Break`, which is the
licence ADR-0008 spends nowhere.

### 4. What the operator configures, and what they do not

Zero or more channels, each with a URL, an optional secret, the classes it receives (defaulting to
all three), and an enable. **No channel ships configured.** Admin only
([#11](https://github.com/winniel123/verge-asm/issues/11): a permission check on every mutating
endpoint), and the secret is write-only in the interface — never rendered back — on the same
footing as #11's recovery codes.

**Routing is by class and by nothing finer.** The three classes are the partition the model itself
owns; a per-rule or per-subject filter is an operator-authored predicate over a versioned rule set,
which is [#16](https://github.com/winniel123/verge-asm/issues/16)'s refusal one layer across, and
it fails silently the first time a rule is renamed. Class routing is also the v1 answer to volume
(§6), which is why it earns its place rather than being one dial too many.

What the operator does not get: the message set, any predicate, the payload, the wording, per-rule
routing, the retry budget, and the delivery retention. The retry budget is ours because it governs
how hard we hammer somebody else's server — the same class of safety property as #4's probing
limits, which are the project's and not the operator's.

### 5. A failed delivery is an Operational record, and never a message

Delivery retries on a bounded budget and is then **dead-lettered**, on the queue machinery
[ADR-0001](./0001-stack-and-runtime.md) and [#9](https://github.com/winniel123/verge-asm/issues/9)
already built rather than a second one beside it. The semantics are stated as they are rather than
as one would like them:

- **At-least-once at the receiver.** A POST that succeeds and whose record is lost is retried, so
  the receiver may see a duplicate. The stable identifier in §3 is what lets it de-duplicate. There
  is no exactly-once claim, because there is no exactly-once mechanism.
- **No ordering.** Causes fire across concurrent batches and retries reorder by construction, so
  every message carries the instant of its cause and the channel promises nothing about sequence.
- **No back-pressure.** A dead channel never blocks measurement, folding, or the writing of a
  `Span`. Delivery is downstream of everything and holds nothing up.
- **A dead-lettered `Delivery` licenses no silence**, which is ADR-0005's *a dead-lettered `Batch`
  licenses no absence* one layer across. The operator must never be able to read a quiet channel as
  a quiet estate.

**And it is a record, not a message.** *Your channel is broken* is the most tempting fifth cause in
the product and it is refused. It cannot go over the broken channel; sending it over another makes
channels aware of each other and turns one dead receiver into a cross-channel storm; and in the
in-app store it would be a message belonging to none of the four causes — the world did not move,
our looking did not change, and no clock crossed. ADR-0026 refused a fifth cause on the ground that
*a fifth cause needs a reason and not a slot*, and this one has a slot and no reason.

It surfaces instead in the two places it is **about**: as the delivery state of the message it
failed to carry, in the store, and as the state of the channel, on the channel's own surface. That
is #22's shape exactly — a cadence misconfiguration is *"a configuration statement next to the
thing they would change, not an entry in a log they would have to go read"* — and #22's *raw errors
appear as drill-down, never as a top-level log* governs the error string.

**A delivery failure never touches `Coverage`.** Coverage answers *is what I am looking at
complete?*; delivery answers *were you told?*. Folding one into the other would make a broken
webhook render as a blind prober, which is #127's *the audit trail records what operators did, the
operational record records what the system did* defect one step across.

### 6. v1 ships no coalescing and no flap suppression

The licence stays exactly where ADR-0007 put it — damping belongs in notification and nowhere else
— and v1 exercises none of it. Three reasons, in order of weight:

1. **Every proposed suppression rests on an unmeasured base rate.** ADR-0026 flagged three unmeasured
   volumes and ADR-0029 flagged two more; the map has warned about this shape four times. A window
   chosen now is a threshold deciding whether the operator is told, set from nothing.
2. **Coalescing delays the flagship.** A digest window is a promise to hold the one message the
   product exists to send, in order to solve a volume problem produced by a different class.
3. **Class routing already separates them, at zero latency and zero threshold**, and the model
   already answers burst its own way: messages fire **at the cause** with a **census**, so a
   thousand openings are one message with counts rather than a thousand messages.

The residue is stated: routing is per class, so an operator drowned by **one rule inside** a class
must silence the whole class on that channel — and the messages are still in the store either way.
Reopening condition: a **measured** volume in the class routing cannot rescue.

### 7. A message is sent when its cause folds, and never held for a schedule

[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) and
[ADR-0033](./0033-a-move-carries-the-rule-that-opens-at-fired.md) both recorded one thing as the
notification patch's and refused to guess it: *whether the census is emitted incrementally per
completed `Batch`, or the membership message waits for a defined set of tiers.* It is a delivery
semantic and it is answered here.

**Neither. A message is written and sent when the fold that caused it completes, and its census is
what that fold could see.** Later openings on a slower tier are silent, which is the cost ADR-0031
already stated rather than a new one.

Waiting for a defined set of tiers loses three ways: it makes the message's **content** a function
of *when* it fired rather than of what happened, so two installs on one release send different
messages for one event; it holds the flagship for up to a week on `tls-acceptance`'s own `Scan`;
and it is a coalescing window under another name, refused in §6. Emitting incrementally loses once
and fatally — it turns one cause into a stream of messages, which is ADR-0007's *alert on the cause,
never per consequence* given away at the layer that was supposed to enforce it.

The existing law already decides this and the fog was only whether **delivery** could rescue it. It
cannot: a census is computed **once at the cause** (ADR-0029 §7, ADR-0026 §3), and *a schedule
arriving is not the world moving* (ADR-0033). A channel that holds a message until a schedule
completes is a schedule arriving, dressed as a message.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) gains three terms** — `Channel` in Declared, on `Proposal`'s
  own layer test (*it is input and does not drift*); `Message` and `Delivery` in Operational, on
  `Dispatch`'s terms (the comparison path may never read them). `Message` is Operational because it
  records that the operator was **told**, not what is true of the estate — the fact is in the
  timelines, and if the two ever disagree the timeline wins and the message is still a true record
  of what we said. That is what stops a stored message being ADR-0007's second representation of
  one fact, and what keeps `Finding` from returning as a diffed message log.
- **ADR-0001's redirect is discharged.** *"Deferring the API costs no integration story"* is now
  true rather than promissory: the push story is §2, and it costs no inbound credential, which was
  the whole reason the API was deferred. What is **not** served is the operator who wanted pull —
  ADR-0001's session-authed export remains their only route, and §2's refusal of a feed closes the
  cheap alternative deliberately.
- **Four documents say notification-layer suppression already exists, and none of it did.**
  Withdrawn at each site per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md):
  `CONTEXT.md`'s `Derivation` entry, ADR-0004 twice, ADR-0007's summary table and ADR-0008's
  what-the-operator-still-has list. ADR-0004 is the sharp case — it says *"the remedy is the
  notification layer's existing suppression"* and then, two sentences later, *"if one is ever
  wanted it is legal ... but it is not wanted now"*. The contradiction was internal to one
  paragraph and this ADR resolves it rather than creating it.
- **The ACME flap has no remedy in v1 and that is now explicit.** ADR-0004 routed it to a
  suppression that did not exist. Its actual v1 treatment is class routing, and whether that
  reaches it **depends on an unsettled class assignment** — see the next consequence.
- **A class-assignment conflict is surfaced rather than papered over.** ADR-0026 §5 puts
  `not-fired` → `fired` in the **drift** class *"for all sixteen"* rules;
  [#60](https://github.com/winniel123/verge-asm/issues/60) and ADR-0004 put the three
  certificate-lifetime rules in the **clock** class, because they become true with no new
  observation. Both cannot be right, and routing is by class, so an operator routing the clock
  class away either does or does not escape the ACME flap depending on the answer. Flagged, not
  ruled: it is a question about causes, and causes are
  [#120](https://github.com/winniel123/verge-asm/issues/120)'s.
- **One fog patch is discharged rather than passed on.** ADR-0031 and ADR-0033 both recorded
  *incrementally per `Batch`, or wait for a defined set of tiers?* as the notification patch's and
  refused to guess it. §7 answers it with neither, and the answer is the existing law rather than a
  new rule — the fog was only whether delivery could rescue an opening the model had already ruled
  silent, and delivery cannot.
- **Three conditional remedies elsewhere are confirmed, not withdrawn.** ADR-0026, ADR-0031 and
  ADR-0033 each say that *if* a measured volume drowns the channel, the remedy is coalescing here
  and never a predicate change. That is precisely §6's reopening condition, so those sentences
  survive intact — unlike the four that asserted the mechanism already existed.
- **The interface gains a message surface and no nav destination.** The store renders as a global
  element with an unread count, present on every screen, whose rows link to the object each message
  is about. `Coverage` was refused — #22 built it to answer completeness and explicitly turned down
  the operations framing — and a seventh nav item was refused because #10 and #22 both spent real
  capital keeping the nav short. Recorded here rather than promoted, on #10's and #22's precedent
  that IA decisions live in the ticket.
- **The measurement path gains nothing and the worker gains one job kind.** Delivery is worker-side
  ([ADR-0001](./0001-stack-and-runtime.md) split the tiers by blast radius and this is outbound
  network work), which means the **channel secret is a worker secret** — a fourth entry in
  [#124](https://github.com/winniel123/verge-asm/issues/124)'s split, and the first one whose
  compromise is useful **outside** the instance.
- **Decided on thin ground in two places, and neither is dressed as a derivation.** The **retry
  budget** — five attempts over roughly an hour, exponential — is chosen on shape (survive a
  receiver restart, do not survive a decommissioned receiver) and measures nothing. And the
  judgement that **a JSON-only body is affordable** rests on the claim that a small org can stand up
  a ten-line receiver or already runs something that accepts one; that is an argument about the
  modal operator, not a measurement, and it is the ruling here most likely to be overturned.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **SMTP / email as a second transport** | The obvious *yes*, and it loses on the credential rather than on cost. An SMTP credential on the instance is usually a **send-as-the-organisation** credential: its blast radius when the high-value target falls is phishing in the operator's name, while a webhook URL's is *the attacker can post to your alerts channel*. Two more losses beside it — mail is store-and-forward through relays the operator mostly does not run, so the body lands archived in a third party's mail store by construction; and SMTP yields *the relay accepted it*, never *it was delivered*, so a `Delivery` reading `succeeded` would be reassurance the record cannot support, which is #14's false-reassurance failure re-entering through the notification layer. **Reopens** for v1.1 with a delivery vocabulary honest enough to say *handed to a relay* |
| **Vendor integrations — Slack, Discord, Telegram, Teams, Matrix, PagerDuty** | Each is an API client, an auth flow, a rate-limit regime and a body format we do not own, moving at a cadence we do not control — ADR-0004's release-coupling test applied outside the signal set. It is also precisely what reNgine already ships (Discord/Slack/Telegram), and [#2](https://github.com/winniel123/verge-asm/issues/2) is explicit that the differentiation is narrow and must not be spent competing where the prior art is strong |
| **A per-vendor body shape as an enum — `json` \| `slack` \| `discord`** | The cheap middle, and it is a **curated table asserting about the world** with a vendor's cadence, no owner's attestation and no watch — handing [#125](https://github.com/winniel123/verge-asm/issues/125) a table nobody can revise on evidence. It also converts *we POST this document* into an ongoing compatibility **claim** about an artefact we do not own, which [#57](https://github.com/winniel123/verge-asm/issues/57)'s *the project may only claim an act it can produce* refuses in its own register. **Reopens** if #125's watch acquires an owner for it |
| **An operator-editable payload template** | Operator-authored code in the alerting path — #16's refusal one layer across — and its failure mode is a malformed body that 400s on the night the flagship fires. It also hands the operator the ability to put rows in the body, which is the whole of §3 given away in a text area |
| **A pull feed — RSS, Atom or a JSON endpoint** | The one option the #6 constraint actually kills. A feed reader holds no session and does no TOTP, so the feed needs a bearer credential that **reads the estate** — #6's bypass exactly, and worse than the API it refused because a feed URL ends up in a third-party reader |
| **Structured log lines to stdout as a channel** | Free and genuinely tempting: `docker compose logs` is already the documented path for #11's setup token. It loses on the same clause as everything else here — it duplicates complete message bodies into a corpus with no retention rule ([#121](https://github.com/winniel123/verge-asm/issues/121) has three and does not want a fourth), readable by anyone in the `docker` group and shipped to a third-party log SaaS by default in most deployments. An operator who wants it points a channel at a loopback receiver that logs, which is the same outcome with a delivery record attached |
| **In-app only, no outbound transport at all** | Maximally safe and it makes ADR-0001's *deferring the API costs no integration story* false retroactively. Nobody logs into an ASM tool to find out that a port opened |
| **Make a delivery failure a message** | Needs a fifth cause for a fact about us rather than about the estate, cannot use the channel it is about, and storms across channels when a shared receiver dies. ADR-0026 refused a fifth cause for a **world** event that the four already carried; refusing one for a **product** event is the easier case |
| **Route delivery health into `Coverage`** | Reads *we could not tell you* as *we could not see*, so a dead webhook renders in the same band as a blind prober — #22 cut that band's four routes apart precisely because collapsing them trains the operator to dismiss all of them |
| **A coalescing or digest window as an operator dial** | Not damping, and still refused: it delays the flagship to solve a volume problem in another class, and class routing solves that one at zero latency. **Reopens on** a measured volume in the class routing cannot rescue |
| **Fixed retry numbers as operator dials** | They govern request rate against somebody else's server, which puts them on #4's side of the line — a safety property the project owns — rather than on #22's side, where an operator's threshold changes only who gets woken |
| **A seventh nav destination for messages** | #10 kept the nav to five and demoted the estate listing to do it; #22 spent the sixth slot and argued for it at length. A feed is not a destination you go to — its whole job is to be visible where you already are |
