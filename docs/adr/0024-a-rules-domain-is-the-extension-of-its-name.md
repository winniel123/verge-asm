# A rule is four parts, and its domain is the extension of its name

- **Status:** Accepted
- **Date:** 2026-08-13
- **Ticket:** [#53 What is a Signal's predicate domain, and is it a seventh thing a rule must declare?](https://github.com/winniel123/verge-asm/issues/53)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[#44](https://github.com/winniel123/verge-asm/issues/44) made a `Signal` render a census of three
members over one population — fired, did not fire, `not-evaluable` — and made the denominator the
rule's **predicate domain over subjects currently in the estate**, never the set of timelines the
rule happens to hold. Counted the other way a rule that has never been evaluable anywhere reports
`0 / 0 / 0` and the never-evaluable population is invisible by construction, which is
[ADR-0004](./0004-signals-are-release-coupled-rules.md)'s bill of health arriving as a number
instead of a word.

Nothing said where that domain is written down, what it may read, or what happens when it moves.

The three easiest domains to state are easy for three different reasons and they do not compose.
`sensitive-port-reached-from-internet` joins against **release-coupled reference data**.
`plaintext-http-no-https` was carrying a **literal** — *`Endpoint`s on 80/tcp* — sitting outside
the rule's versioned reference data with nothing versioning it. And `certificate-expiring`'s domain
— *`Endpoint`s presenting a certificate* — is itself a **measured fact**, because
[ADR-0011](./0011-a-facet-is-six-parts.md) made `NoTLS` a value rather than an absence. So the
denominator of a rule nobody thinks of as reading the world twice moves when the world moves.

The volume makes it decisive rather than academic. ADR-0011 attempts TLS on **every open
`Service`**, so — as the map's retention patch already records — most `certificate` timelines in
any estate hold `NoTLS` forever. Whether those endpoints are counted in *did not fire* or are
outside the domain is the difference between a census over a handful of endpoints and a census
whose *did not fire* column is the whole estate.

## Decision

| Concern | Decision |
| --- | --- |
| Where the domain is written | **In the rule.** A rule is **four parts** — domain, predicate, `not-evaluable` case, version vector — plus one **cost**, the new measurement it requires |
| Is it a seventh thing | **No.** ADR-0011's six are the parts of a **facet**; a rule had no such list. This makes [#35](https://github.com/winniel123/verge-asm/issues/35)'s informal three structural, and the domain is the fourth |
| What fixes the domain | **The rule's name.** The domain is the set of subjects of which the fact the rule is named for could be asserted |
| May the domain be measured | **Yes**, and for the certificate rules it **must** be |
| What the domain may read | Only evidence **the rule itself declares**; domain and predicate are read from one evidence set |
| Does it compose into the version | **Yes — inside the rule's own leaf**, and it adds no new leaf, because it cites nothing the rule does not already compose |
| Editing a domain | An **output-affecting change**: the leaf moves, and the rule `Break`s uniformly. One cadence, one rule |
| Is *outside the domain* a third case beside #44's two | **No.** It is not rendered as a member, a row, a state or a transition |
| Census members | **Still three.** #44 is confirmed, not amended |
| A census over a window | **Forbidden.** No delta, no series, no trend on a rule's census |
| The three registers | *Outside the domain* = the question does not arise · `not-evaluable` = we cannot see the answer · `Gap` = we did not look |
| `NoTLS` under a certificate rule | **Outside the domain** |
| `Shadowed` under the DNS rules | **`not-evaluable`**, unchanged — and #44's thin ground is now firm |
| `plaintext-http-no-https`'s `80/tcp` literal | **Withdrawn.** Its domain is *`Endpoint`s that answered HTTP* |
| An empty domain | [#44](https://github.com/winniel123/verge-asm/issues/44) decision 9's **no-population panel**, whose copy must now name the measured reason |

## Rationale

### The domain is authored because it is not recoverable from the firing set

The tempting answer is that the domain needs no declaration at all: write the predicate, and the
domain is wherever the predicate has a truth value. That answer dies on one observation.

Take `plaintext-http-no-https`. Its evidence is two facet values — `http-identity` and
`certificate` — and there are two total decompositions of the same rule:

- domain *the endpoint answered HTTP*, predicate *it presented no TLS*; or
- domain *the endpoint presented no TLS*, predicate *it answered HTTP*.

**Both are total. Both fire on exactly the same subjects. Their censuses are different objects** —
the first counts plaintext endpoints out of endpoints serving HTTP, the second counts HTTP speakers
out of endpoints with no TLS. Nothing downstream can recover the choice, because the extension of
the rule is identical either way and the census is the only place the difference shows. A thing
that changes what the operator is told and cannot be derived from anything else is a thing somebody
has to write down.

That is also why it cannot be a filter on the `Signals` screen. Two independently-authored
populations over one rule is [#28](https://github.com/winniel123/verge-asm/issues/28)'s
two-numbers-on-one-screen hazard, and a screen-side domain is unversioned by construction — the
denominator moves with nothing recorded, and a census that dropped 400 subjects looks identical to
one where 400 subjects were fixed.

### The name fixes the domain, so it is a review question and not a taste question

An authored domain with no constraint is a dial. [ADR-0007](./0007-drift-is-a-timeline-of-spans.md)
refused model-layer damping outright, [ADR-0015](./0015-the-value-space-is-the-commitment.md)
settled that a signal being true of most of the estate is not a defect, and
[ADR-0010](./0010-exposure-composes-two-reaches.md) had already refused to gate the certificate,
TLS and HTTP signals on an internet vantage because it "would change nothing except to make
internally-observed defects `not-evaluable` — reporting less than we measured, in order to express
a severity the model refuses to carry." A domain is the cheapest possible place to rebuild all
three refusals at once: narrow the population and the rule goes quiet with nothing recorded.

The constraint is already in the model and needed only to be pointed at this. ADR-0010 and
ADR-0015 fixed that **a signal is named for the fact it reads**. So:

> **A rule's domain is the extension of its name** — the subjects of which the fact the rule is
> named for could be asserted. A domain may exclude a subject only where that fact could not be
> true of it. Excluding a subject on which the rule would fire is damping, whatever it is called.

The test is mechanical enough to run in review, and it cuts every case put to it:

- `NoTLS` under `certificate-expiring` — *this certificate is expiring* cannot be true of an
  endpoint with no certificate. **Outside.**
- 8080/tcp under `sensitive-port-reached-from-internet` — *this sensitive port is reached from the
  internet* cannot be true of a port that is not on the list. **Outside.**
- An internally-observed expiring certificate — *this certificate is expiring* **is** true.
  **Inside**, and excluding it is ADR-0010's refusal under a new name.
- An `Annotation` recording accepted risk — the fact is still true. **Inside.** Operator opinion
  may not do the job measurement does ([ADR-0006](./0006-subjects-leave-by-measurement.md)), and
  this closes the last door into [#22](https://github.com/winniel123/verge-asm/issues/22)'s refused
  suppression.
- 8080/tcp under `plaintext-http-no-https` — *this endpoint serves plaintext HTTP and no HTTPS* is
  exactly what is true of a plaintext app on 8080. **Inside**, and the literal is damping. See
  below.

The second guard is narrower and stops the first being evaded: **a domain may cite only evidence
the rule declares**, and everything a rule declares composes into its version vector. A domain
therefore cannot reach for a new fact without the vector growing and the corpus having to cover it.
There is no free read.

### The `80/tcp` literal was not an unversioned domain, it was a curated port list

The ticket asked what versions the literal. The answer is that nothing should, because it does not
survive the test above: an HTTP application on 8080 with no TLS is precisely the thing the rule is
named for, and the literal excludes it.

It is also the exact table this map has now refused twice.
[ADR-0009](./0009-verge-core-is-a-union.md) found four pairs missing from a hand-maintained port
set that nobody derived, and ADR-0011 **deleted** the curated implicit-TLS port list on the
grounds that opportunistic TLS on every open `Service` "makes `NoTLS` honest everywhere instead of
a `Gap` on every unlisted port, and it finds implicit TLS on odd ports, which is exactly what a
small org accidentally leaves listening." A port literal inside `plaintext-http-no-https` puts that
list back, one rule at a time, where no one would look for it.

So the literal is withdrawn. The rule's domain is *`Endpoint`s whose `http-identity` is
`Responded`*, and its predicate is *`certificate` is `NoTLS`*. The rule fires on more subjects than
it did, which ADR-0015 already settled is not a defect — commonness is not disqualifying, the
transitions are the subject, and narrowing the rule to quieten it is the damping ADR-0007 refused.
The name does not move: it never named the port.

### A measured domain is exactly as checkable as a measured predicate

The ticket's fourth question assumed a fork: a static structural domain is checkable in CI against
the golden corpus, a measured one is not, so perhaps a domain must be a static key.

That premise is false, and [ADR-0021](./0021-a-version-leaf-is-a-decision-not-a-binary.md) is what
falsifies it. **Corpus rows are authored, never captured**, and the reviewable content of a row is
its claim in prose. A row asserting *this endpoint presented no certificate, so the expiry rule
does not ask about it* is exactly as authorable and exactly as human-judgeable as a row asserting
*this leaf expires in 20 days, so the rule fires.* Nothing about the domain being read from a
measurement puts it beyond the gate.

The fork also has no principled edge. A static domain is available only where the discriminating
fact happens to sit in the subject's **natural key** — which is ADR-0011's own observation that
`reachability` never needed a discriminator "only because its subject `Service` already carries
port and transport." Port and transport are measured facts too; they are merely measured earlier
and stored in a key. Drawing the legality of a domain along that line makes the rule set a
consequence of where the model happened to put things.

And the static option is unbuyable for the case that forced the ticket. A static domain for the
certificate rules would have to be a port list, and the previous section is what happens to port
lists here.

### Three registers, cut on what the evidence is about

`NoTLS` and `Shadowed` are both values, both stop a rule producing fired-or-did-not-fire, and #44
put `Shadowed` in `not-evaluable` while flagging it as the piece most likely to be wrong. They are
not the same thing, and the difference is in what the value is *about*:

| The evidence says | Register | Rendered as |
| --- | --- | --- |
| A fact about the world under which the rule's question does not arise (`NoTLS`, `NotHTTP`, no CNAME, a non-sensitive port) | **outside the domain** | nothing — the subject is not in this rule's population |
| A fact about **our own sight** (`Shadowed` — ADR-0007's *a value meaning we cannot see*) | **`not-evaluable`** | a census member and a row, cause *we measured; this rule cannot read the answer* |
| No value at all | **`Gap`** | a census member and a row, cause *we never looked* / *we stopped looking* / *you stopped answering* |

So #44's fourth cause survives and stops being thin ground. It is not a wording judgement about one
sentence of ADR-0004; `Shadowed` is the only value in the model whose content is our own blindness,
which is why it earns the coverage register while every other measured negative does not.

The corollary matters more than the cut: **`not-evaluable` is a coverage word, and a rule whose
inapplicable population lands there destroys it.** Route `NoTLS` to `not-evaluable` and the
majority of every `certificate` timeline in the estate renders in #44's sunken not-comparable band,
permanently, with no action available and nothing failed — which trains the operator to ignore the
band that exists to catch a real outage. That is the same failure ADR-0011 caught from the other
side when it refused to route `NoTLS` to a `Gap` and send `plaintext-http-no-https` to
`not-evaluable` on exactly the estates where it is true.

### Outside the domain is not a third case, because it is already a `Transition` one level down

#44 split on whether a subject exists: a per-subject row on `Signals`, or a standing aperture
statement on `Coverage` for the half that can never have a row. A `NoTLS` endpoint is neither — it
is a subject in the estate that a certificate rule says nothing about — so the question is whether
it is a third.

It is not, and the reason is the one [#42](https://github.com/winniel123/verge-asm/issues/42) gave
for refusing a fourth transition name. **A subject leaving a rule's domain is not a new fact.** It
is a `Transition` on the facet timeline underneath — `Presented` → `NoTLS` on `certificate` — which
the product already stores, already renders and already alerts on. Rendering it a second time as a
fourth census member, an *outside the domain* state or an *entered/left the domain* transition is a
second representation of one fact, which [ADR-0007](./0007-drift-is-a-timeline-of-spans.md) refuses
outright.

So the domain is exactly the line. Three members stand, the denominator is the domain, and a
subject outside it is not rendered by the rule at all.

What that leaves is one presentation obligation, and it is the honest answer to *a census that
dropped 400 subjects looks identical to one where 400 were fixed*: **do not subtract two censuses.**
A census is current state and not a comparison ([ADR-0008](./0008-derivation-versions-move-on-content.md)),
so it asserts nothing about last cadence and nothing false is said. It becomes a lie the moment it
is rendered as a delta, a trend or a series, because that number conflates a change in the domain
with a change in the predicate. Subtracting censuses is a comparison the drift model already
performs correctly, one level down, per subject.

### The denominator has exactly two ways to move, and neither is silent

The remaining worry is that the denominator moves with nothing recorded. It has exactly two causes
and both are already instrumented:

- **The world moved** — a subject entered or left the domain. That is a `Transition` on the facet
  timeline the domain reads, in the ordinary drift class, alerted at the cause.
- **We moved** — the domain was edited. It is part of the rule, so it is an **output-affecting
  change** under ADR-0008: corpus rows move, the rule's leaf must move or the build fails, and the
  rule `Break`s uniformly. One cadence, one rule, because versions are per rule.

There is no third way, so a domain cannot move quietly. The price is the one the ticket named and it
is accepted: **editing a domain costs a `Break` on the whole rule.** Under ADR-0008 that costs one
cadence and clamps one rule's horizon, and withdrawing the `80/tcp` literal removes the only domain
in the v1 set that anybody was likely to want to edit — the rest are partitions of closed unions,
which move only when the union moves, which ADR-0011 already prices.

### The corpus row gains a fourth outcome

ADR-0008 defined a signal's corpus row as *these observations → fires / does not fire /
`not-evaluable`*. That shape cannot express a domain, so it gains a fourth outcome, **outside the
domain**, and the gate becomes bidirectional over all four exactly as ADR-0021 made it for every
other derivation: a row whose outcome moves without the leaf moving fails the build, and a leaf that
moves for no moved row fails it too.

This is what makes the domain a mechanism rather than a discipline, which is the standard this
project has held everywhere in the comparison path. It also gives the failure mode a name, in
ADR-0011's style, because it is specific: **a session adds a rule, writes the predicate, and lets
the domain default to every subject the predicate typechecks over** — which is how
`plaintext-http-no-https` acquired a port literal nobody versioned, and how `certificate-expiring`
would have put the majority of the estate into *did not fire*.

## The v1 domains

Recorded here because the ticket's question was *where is this written down*, and because writing
all sixteen out is what tested the rule.

| Rule | Domain | Outside it |
| --- | --- | --- |
| `certificate-expired` · `-not-yet-valid` · `-expiring` · `-self-signed` · `-weak-key-or-signature` | `certificate` is `Presented` | `NoTLS` |
| `certificate-hostname-san-mismatch` | `certificate` is `Presented` **and** the `Endpoint` has a `Name` | `NoTLS`; a nameless `Endpoint` (ADR-0011) |
| `tls-1.0-accepted` | the `Service` completed at least one handshake in the batch's candidate set | a `Service` that accepted no TLS at all |
| `plaintext-http-no-https` | `http-identity` is `Responded` | `NotHTTP` — **not** a port |
| `redirect-does-not-upgrade-to-tls` · `redirect-to-host-outside-estate` | `http-identity` is `Responded` with a 3xx and a `Location` | `NotHTTP`; any non-redirect response |
| `unauthenticated-request-answered` | `http-identity` is `Responded` with a 2xx or a 401/403 | `NotHTTP`; a **3xx**, which ADR-0015 called "outside the predicate" and meant this |
| `sensitive-port-reached-from-internet` | `Service`s whose `(port, transport)` is on the sensitive list | every other `(port, transport)` |
| `lame-delegation` | every `Name` in the estate — **a total domain is legal** | nothing; `Shadowed` is `not-evaluable`, inside |
| `cname-target-name-error` | `Name`s whose `dns-record` holds a CNAME | a `Name` with no CNAME; `Shadowed` is `not-evaluable`, inside |
| `zone-declared-name-returns-name-error` | `Name`s the operator's zone file declares | every `Name` the file does not declare |
| `resolved-name-absent-from-zone` | `Name`s our resolver resolved **within a declared zone** | a `Name` outside every declared zone ([#48](https://github.com/winniel123/verge-asm/issues/48)'s zone-not-registrable-domain rule) |

Two things fell out of writing it. `hostname-SAN mismatch` has a domain nobody had noticed, because
ADR-0011 made an `Endpoint`'s `Name` optional and a nameless endpoint has no hostname to mismatch.
And the same value sits on both sides of the line in one facet: a **3xx** is outside
`unauthenticated-request-answered`'s domain and inside the two redirect rules', which is the
clearest available demonstration that a domain is a property of the **rule**, not of the facet.

## Consequences

- **[`CONTEXT.md`](../../CONTEXT.md) gains `Predicate domain`** and `Signal` records the four parts
  and the three registers.
- **[ADR-0004](./0004-signals-are-release-coupled-rules.md) is amended.**
  [#35](https://github.com/winniel123/verge-asm/issues/35)'s *what a proposal owes* — three items,
  explicitly informal — becomes the four-part structure, with *what new measurement it requires*
  restated as the cost it always was rather than a fourth part.
- **[ADR-0008](./0008-derivation-versions-move-on-content.md)'s signal corpus row gains a fourth
  outcome**, and the domain is inside the rule's leaf rather than beside it — the same treatment
  ADR-0008 gave the availability window and `k`.
- **[#44](https://github.com/winniel123/verge-asm/issues/44) is confirmed rather than amended**, and
  its one piece of stated thin ground is now firm. Its decision 9 no-population panel gains a
  second and far more common producer — a rule whose domain is empty because the estate measured
  that way, not because the operator decided anything — so its copy, currently written for the
  custody case (*"You are right that you do not control those listeners"*), must state the
  **measured** reason: *no endpoint in your estate presents a certificate.*
- **`plaintext-http-no-https` fires on more subjects**, and its census denominator changes from a
  port to a measurement. Nothing else in the v1 set moves.
- **A rule's census may not be rendered as a delta, a trend or a series.** This is a new binding
  presentation rule and it lands on `Signals`, alongside ADR-0015's existing obligation not to
  render a broad signal's census as a findings list.
- **`Annotation` loses its last route into the comparison path.** The map's open question about
  whether *accepted risk* is worth a Declared-layer term is untouched, but the domain is now
  explicitly not where it could have been spent.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| No domain — the predicate's own definedness is the domain | Two total decompositions of one rule fire identically and produce different censuses; nothing downstream can recover the choice |
| The domain is a filter on the `Signals` screen | Unversioned by construction, so the denominator moves with nothing recorded — and two independently-authored populations over one rule is #28's two-numbers hazard |
| The domain must be a static structural key | The CI objection is false (ADR-0021: corpus rows are authored), the static/measured line tracks only where a fact happens to sit in a natural key, and the certificate rules could buy it only with a curated port list |
| Keep `plaintext-http-no-https`'s `80/tcp` literal and version it | Versions the wrong thing. The literal excludes plaintext HTTP on 8080, which is exactly the fact the rule is named for — ADR-0011's deleted implicit-TLS port list rebuilt one rule at a time |
| `NoTLS` counts as *did not fire* | Puts the majority of the estate in a column reading *this endpoint's certificate is not expiring*, which is true and is a bill of health for a population that has no certificate |
| `NoTLS` yields `not-evaluable` | Fills the coverage band permanently with rows where nothing failed and no action exists, destroying the word #44 built the band for — ADR-0011's refusal seen from the other side |
| *Outside the domain* as a fourth census member, state or transition name | A second representation of one fact: the subject left the domain because a facet `Transition` already recorded and alerted. #42 refused a fourth name on this exact ground |
| A domain free to read any evidence | The cheapest available rebuild of ADR-0007's damping, ADR-0010's refused vantage gate and #22's refused suppression, all at once and all unrecorded |
| Render the census over a window so a moved denominator is visible | The number conflates a moved domain with a moved predicate; the drift model already makes that comparison correctly, per subject, one level down |
