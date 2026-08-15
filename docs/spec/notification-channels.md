# Notification channels and delivery

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ruling:** [ADR-0039](../adr/0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
- **Ticket:** [#119 Which notification channels ship in v1, and what are their delivery semantics?](https://github.com/winniel123/verge-asm/issues/119)

ADR-0039 rules that a `Channel` carries a `Message` and never the estate, that the store is not a
channel, and that a `Delivery` is an Operational record which never becomes a message. This
document is the enumeration behind it. It is a separate file from the ADR because it is **a list
that will be revised** and an ADR is a decision that will not.

**This document does not decide what fires.** The message set is closed by
[ADR-0026](../adr/0026-the-facet-layer-is-evidence-not-a-channel.md),
[ADR-0029](../adr/0029-an-alert-fires-on-a-leg.md),
[ADR-0031](../adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md),
[ADR-0033](../adr/0033-a-move-carries-the-rule-that-opens-at-fired.md) and
[#22](https://github.com/winniel123/verge-asm/issues/22). It does not decide the **wording** of
anything, which is [#120](https://github.com/winniel123/verge-asm/issues/120)'s.

Two marks are used, on `measurement-offers.md`'s convention:

| Mark | Means |
| --- | --- |
| `[derived]` | Follows from an accepted decision named in the row |
| `[thin]` | Chosen. No attestation, no measurement. Revisable, and the revision price is stated |

---

## 1. The surfaces

| Surface | Configured? | Can it fail? | Produces a `Delivery`? | May a message skip it? |
| --- | --- | --- | --- | --- |
| **The store**, rendered in the interface | No — it is not a channel | No | No | **Never** |
| A **`Channel`** — outbound `https` POST | Yes, zero or more, none shipped | Yes | One per attempt | Yes, by class routing or by there being no channel |

The store is complete by construction. Every message is written and rendered whatever happens to
every channel, which is why a misconfigured, disabled or dead channel loses no fact — and why the
one surface that reports a delivery failure is the one surface that cannot have one.

### 1.1 Where the store renders

A **global element carrying an unread count**, present on every screen, opening a list whose rows
each link to the object the message is about. Not a nav destination: `Exposure · Subjects · Signals
· Seeds · Coverage · Settings` is unchanged. `[derived]` — [#10](https://github.com/winniel123/verge-asm/issues/10)
kept the nav to five and demoted the estate listing to do it; #22 spent the sixth slot and argued
for it.

Read-state is a property of the message and never of history —
[ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md), verbatim.

---

## 2. What a `Channel` holds

| Field | Value | Note |
| --- | --- | --- |
| URL | An absolute `https://` URL | `http://` refused except to a **loopback** destination — §4.1 |
| Secret | Optional | Signs the body; never transmitted — §3.2 |
| Classes | A subset of `drift` · `coverage` · `clock`, defaulting to all three | The only routing axis — §5 |
| Enabled | Boolean | Disabling is not a predicate change |

**No channel ships configured**, and creating or editing one is an **admin** act
([#11](https://github.com/winniel123/verge-asm/issues/11): a permission check on every mutating
endpoint). The secret is **write-only** in the interface — set, replaced, cleared, never rendered
back — on the footing #11 set for recovery codes.

There is no cap on the number of channels and no reason for one: each is one POST per message it
subscribes to.

---

## 3. The request

### 3.1 The body

One JSON document per message. It carries exactly what the in-app message carries and **no rows**.

| Field | Carries |
| --- | --- |
| Message identifier | Stable, unique, unchanged across retries — the receiver's de-duplication key |
| Class | `drift` · `coverage` · `clock` |
| Cause | Which of the four causes, at the granularity the store holds |
| Subject or scope | The **key** of the thing the message fired at — never a rendering of it |
| Instant of the cause | Not the instant of the attempt |
| Census | Counts only, where the message has one |
| Link | An absolute URL into this instance, at the object the message is about |

**No rows, in any field.** Not the services behind a census count, not the address set behind a
`resolution` move, not the evidence behind a `Signal`. `[derived]` — the map's
high-value-target constraint, discharged in the contract rather than in prose. The chat history,
the receiver's disk and the log pipeline accumulate *what happened*; the operator who wants *what
they have* follows the link and authenticates.

The payload is **one computation at the cause, two renderings** — the same figure in the store and
in the body, read from one computation, which is
[#50](https://github.com/winniel123/verge-asm/issues/50)'s rule arriving across a screen and a
channel. A message is written once and **never recomputed**: recomputing reaches back across a
`Break`.

### 3.2 Authentication

| Where a secret is | What the request carries |
| --- | --- |
| Set | `HMAC-SHA256` over the body and a timestamp, in a header |
| Not set | Nothing; the URL is the only credential, as it is for every incoming-webhook receiver |

**No bearer header, ever.** `[derived]` — a bearer sits in the receiver's access log and a
signature does not. The signature authenticates **us to them**; nothing authenticates them to us,
because there is nothing for them to ask.

The channel is **one-way**: no callback, no ack channel, no fetch, no inbound surface.
[#6](https://github.com/winniel123/verge-asm/issues/6)'s bearer-token bypass is untouched, because
no credential here reads the estate.

---

## 4. What counts as a failure

| Outcome | Verdict |
| --- | --- |
| `2xx` | Delivered |
| Any `3xx` | **Failed.** The redirect is not followed — [#4](https://github.com/winniel123/verge-asm/issues/4)'s rule, and here it would move the operator's attack surface to a host they never declared |
| `4xx`, `5xx` | Failed |
| Timeout, connection refused, DNS failure, TLS failure | Failed |
| A non-loopback `http://` target | Refused at configuration time, not at delivery time |

### 4.1 The loopback exception, and why it is over the address

`http://` is permitted only where the destination **resolves to a loopback address**, tested
family-matched over the address and never over the string `localhost`. `[derived]` — the same
discipline that keeps the `Custody` gate and `Vantage class` from turning on a rendering
([ADR-0051](../adr/0051-a-subject-key-is-the-thing-denoted-and-its-normalisation-may-never-move.md)).
A name resolving elsewhere would otherwise buy a plaintext exemption for a body that is the
operator's attack surface.

**Stated cost.** A receiver in a sibling compose container is not loopback, so it needs TLS or must
be reached over a loopback-published port.

### 4.2 The retry budget

**Five attempts over roughly one hour, exponential, then dead-lettered.** `[thin]` — chosen on
shape: it must survive a receiver restart or deploy (minutes) and must not survive a decommissioned
receiver (hours). Nothing is measured. The revision price is one constant and no `Break`, because
nothing here is inside a derivation.

It is **fixed and project-authored, not a dial**: it governs request rate against somebody else's
server, which is #4's class of safety property rather than #22's class of operator threshold.

Retries run on the queue's existing retry, backoff and dead-lettering
([ADR-0001](../adr/0001-stack-and-runtime.md),
[#9](https://github.com/winniel123/verge-asm/issues/9)) rather than a second mechanism beside it.

### 4.3 The semantics, stated as they are

- **At-least-once at the receiver.** A POST that succeeds and whose record is lost is retried. The
  message identifier in §3.1 is the de-duplication key. There is no exactly-once claim because
  there is no exactly-once mechanism.
- **No ordering.** Causes fire across concurrent batches and retries reorder. Every message carries
  the instant of its cause; the channel promises nothing about sequence.
- **No back-pressure.** A dead channel never blocks measurement, folding, or the writing of a
  `Span`.
- **A dead-lettered `Delivery` licenses no silence** — ADR-0005's *a dead-lettered `Batch` licenses
  no absence*, one layer across.

---

## 5. Routing

| Axis | Available? | Why |
| --- | --- | --- |
| **Class** — `drift` · `coverage` · `clock` | **Yes** | The partition the model itself owns, over messages rather than events |
| Cause | No | The four causes are a wording distinction and belong to #120; the classes are the partition |
| Per rule, per signal | No | An operator-authored predicate over a versioned rule set — #16's refusal one layer across — and it fails silently the first time a rule is renamed |
| Per subject, per `Seed`, per scope | No | Same shape, and it would let an operator route the flagship away from a scope without saying so |
| Severity | No | There is no severity — [#16](https://github.com/winniel123/verge-asm/issues/16) |

**Class routing is also v1's whole answer to volume**, which is why it earns its slot rather than
being one dial too many. See §6.

---

## 6. What v1 does not do

- **No coalescing, no digest window, no flap suppression.** The licence stays where
  [ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md) put it — damping belongs in notification
  and nowhere else — and v1 exercises none of it. Every candidate rests on an unmeasured base rate;
  a digest window delays the flagship to solve another class's volume; and the model already
  answers burst its own way, since a message fires **at the cause** with a **census**, so a
  thousand openings are one message with counts.
- **Stated cost, and it is real.** An operator drowned by **one rule inside** a class must silence
  the whole class on that channel. The messages are in the store either way. **Reopens on** a
  measured volume in the class routing cannot rescue.
- **The ACME flap has no remedy in v1.** ADR-0004 routed it to a notification-layer suppression
  that did not exist; its actual v1 treatment is class routing, and whether that reaches it depends
  on an unsettled class assignment — ADR-0026 §5 puts `not-fired` → `fired` in the **drift** class
  for all sixteen rules, while [#60](https://github.com/winniel123/verge-asm/issues/60) and ADR-0004
  put the three certificate-lifetime rules in the **clock** class. Flagged, not ruled here: it is a
  question about causes, and causes are #120's.
- **No pull surface.** No feed, no JSON API, no polling endpoint — #6, unchanged, and the reason
  is the credential rather than the format.
- **No email, no vendor integrations, no log-line channel.** ADR-0039's rejected alternatives, each
  with its reopening condition.
- **Stated cost of the JSON-only body**: a chat platform's incoming-webhook URL is **not** a valid
  channel target in v1 without a small receiver in front of it. `[thin]` — that this is affordable
  for the modal small-org operator is an argument, not a measurement, and it is the ruling here
  most likely to be overturned.

---

## 7. When a message is sent

**When the fold that caused it completes, with the census that fold could see.** Not held for a
defined set of tiers, and not emitted incrementally per completed `Batch` — the two options
[ADR-0031](../adr/0031-membership-alerts-at-the-root-of-the-entering-subtree.md) and
[ADR-0033](../adr/0033-a-move-carries-the-rule-that-opens-at-fired.md) recorded as fog and refused
to guess. `[derived]` — a census is computed **once at the cause**, and *a schedule arriving is not
the world moving*.

| Option | Why not |
| --- | --- |
| Wait for a defined set of tiers | The message's **content** becomes a function of when it fired; it holds the flagship for up to a week on `tls-acceptance`'s weekly `Scan`; and it is §6's coalescing window under another name |
| Emit the census incrementally per `Batch` | Turns one cause into a stream — ADR-0007's *alert on the cause, never per consequence*, given away at the layer meant to enforce it |

**Stated cost, unchanged from ADR-0031**: a sensitive port that first answers on a slower tier days
after its `Address` entered is silent. It opened, so no `Transition` exists, and the membership
message has already fired.

---

## 8. What a `Delivery` records, and where it is seen

| Where | What it shows |
| --- | --- |
| On the **message**, in the store | Whether it was delivered, to which channels, and whether any is dead-lettered |
| On the **channel**, on its own surface | Current state, consecutive failures, and the last error string as **drill-down** |
| Nowhere else | It is **never a message**, and it **never touches `Coverage`** |

A delivery failure is not the world moving, our looking changing, or a clock crossing, so it has no
cause and gets no fifth one — ADR-0026's *a fifth cause needs a reason and not a slot*. It is not a
`Signal` either: it is not about a subject and carries no evidence
([#22](https://github.com/winniel123/verge-asm/issues/22), verbatim).

The channel surface follows #22's shape for a cadence misconfiguration — a configuration statement
next to the thing the operator would change, never an entry in a log they would have to go read —
and #22's *raw errors appear as drill-down, never as a top-level log* governs the error string.

**Retention of the delivery record is settled by
[#139](https://github.com/winniel123/verge-asm/issues/139) ·
[ADR-0081](../adr/0081-a-floor-is-territory-and-an-unbounded-default-is-a-position.md).** A `Delivery`
**travels with its `Message`** and holds no retention rule of its own, so neither corpus gets a dial:
the message corpus is what the operator reads back, and a message renders *its* delivery outcomes.
~~Nothing is lost by compacting it aggressively~~ is **superseded here, at the site that specifies
it** ([ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)):
*nothing in the comparison path reads it* is not *nothing reads it*, and discarding a dead-lettered
delivery converts *we could not reach you* into *we told you* — **a dead-lettered `Delivery` licenses
no silence**, defeated by storage.
