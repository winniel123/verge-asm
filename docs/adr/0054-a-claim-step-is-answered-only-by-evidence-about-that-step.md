# ADR-0054: A claim step is answered only by evidence about that step, and reachability is read at the internet-reached frame

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#92 Claim 1's Class A walk cites a non-owner in two Step 1 cells, and the Step 2 operations were never walked against the bytes](https://github.com/winniel123/verge-asm/issues/92)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §10.1 restated Claim 1 as a **two-step test**:
Step 1 asks whether *publication is the purpose* and refuses outright if it is; Step 2 asks what
**authority** the anonymous caller gets. Both steps are answered *"from the specification or the
owner's own documentation"*, and §10.1 carries a twelve-row table applying the test to every Class A
row — one row per cell pair.

[#84](https://github.com/winniel123/verge-asm/issues/84) §20.6 established that **those cells are
attestations too**, so §10.5's owner rule governs them and a corroborator may not answer a step. It
found two Step 1 cells resting on a non-owner, repaired `4369/tcp`'s, and ticketed
`2379`/`2380`'s. It also found that `4369`'s **Step 2** cell named an operation an internet caller
cannot reach — registration is gated on `s->local_peer` in the shipped `epmd_srv.c` — and checked no
other row's. §20.9 declined an ADR on §16.6's test, on the ground that both limbs state what §10.1
and §10.5 already require.

[#92](https://github.com/winniel123/verge-asm/issues/92) discharged the ticket and walked every
remaining Step 2 cell against shipped source at named tags. Two questions fell out that §20.6's
rule does not reach, and each of them decides cells.

**First**, the available repair for `2379`/`2380` was etcd's own prohibition — *"It **must not** be
exposed to untrusted networks or the public internet"*, `THREAT_MODEL.md` at `etcd-io/etcd`
`v3.7.1`. That is a **Claim 3-shaped boundary** answering Claim 1's Step 1, which is the exact shape
§10.1's own strike-through on `10250/tcp` flags as *"the tell nobody read"*. #84 expressly declined
to decide whether a prohibition may answer Step 1 implicitly.

**Second**, §20.6 asked *can an unauthenticated remote caller reach this operation?* without saying
**in which configuration**. On `4369` the question never bit, because epmd listens on all interfaces
by default. Walked across the whole table it bites hard: **[measured]** seven of §10.1's eleven Class
A rows do not serve an anonymous remote caller on their shipped default — `2375/tcp` (unix socket
only), `2379` and `2380` (`http://localhost:…`), `10255/tcp` (`readOnlyPort: 0`), `11211/udp`
(`settings.udpport = 0`), `9042/tcp` (`rpc_address: localhost`) and `6379/tcp` (loopback `bind`
**and** an accept-time refusal of non-local peers). Read literally against the shipped default,
Step 2 answers *no* for seven of eleven rows.

`6379/tcp` forced the question. **[measured]** at `redis/redis` `8.10.0`, `clientAcceptHandler()`
rejects any non-local peer with `-DENIED` and calls `freeClientAsync` whenever protected mode is on
and the `default` user is `nopass` — which is the shipped `redis.conf`. The condition no longer
consults `server.bindaddr_count`; that clause is present at `6.2.19` and absent from `7.0.0` onward,
so a public `bind` does not lift it. §10.1's cell for the row — *"`FLUSHALL`, and arbitrary writes"*
— therefore names operations the shipped configuration serves to no remote caller at all, and unlike
`4369` there is no ungated operation to fall back to, because the gate is on the **connection**.

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) §25.

## Decision

| Concern | Decision |
|---|---|
| May a Claim 3-shaped prohibition answer Claim 1's Step 1 | **No, not on its own.** A prohibition is evidence about **where the owner supports the deployment**; Step 1 asks **whom the protocol is specified to answer** |
| In which configuration is Step 2's reachability question asked | **The configuration in which the port is reached from the internet**, holding the **authentication** configuration at its shipped default |
| Does a shipped default that withholds the port defeat Claim 1 | **No** — it is the network act the frame already assumes |
| Does a shipped default that authenticates the caller defeat Claim 1 | **Yes**, and Step 2 answers in the negative with it |
| Does a gate on particular operations defeat Claim 1 | **Only for those operations.** The row survives on whichever limb still has one |

### Limb 1 — a claim step is answered only by evidence about that step

A sentence admissible for one claim is not thereby admissible for another claim's step. In
particular, an owner's **boundary or prohibition** — *do not expose this to an untrusted network* —
does not answer Claim 1 Step 1's *is publication the purpose?*.

The ground is measurable rather than stylistic. §18.6's **explicit prohibition** tier is 15 pairs and
it straddles the classes: `6379`, `11211/tcp`, `11211/udp`, `9042`, `2379` and `2380` are **Class
A**, while `3306`, ~~`1433`~~, ~~`9200`~~, ~~`9300`~~, `873`, `445`, `623`, `10250` and `10255` are **Class C**.
*(`9200` and `9300` are struck by [#114](https://github.com/winniel123/verge-asm/issues/114) — both
rows are removed and the prohibition tier is 13. The argument is unaffected: it needs the tier to
straddle the classes, and it still does.)*
A sentence-shape compatible with six rows in one class and nine in another cannot discriminate
between them, so it cannot carry the step whose whole job is to discriminate.

> **`1433/tcp` is struck: the row is REMOVED from the list by
> [#109](https://github.com/winniel123/verge-asm/issues/109)** and is in `sensitive-ports.md` §4.6.
> **The measurement above is unaffected** — it is a count over the tier as it stood, and a
> sentence-shape compatible with six Class A rows and **eight** Class C rows discriminates no better
> than with nine. **This ADR is confirmed by use** by that ruling: §10.3's failure condition and
> ADR-0050 limb 3's defeat test are **different steps**, answered by evidence about different
> propositions, which is this ADR's own discipline moved from steps-within-a-gate to the two gates —
> [ADR-0067](./0067-a-claim-fails-on-the-owners-affirmative-naming-not-on-the-reach-of-its-own-prohibition.md).

What *does* answer Step 1 is an owner's statement about its **intended caller population** or about
whether its callers are **identified** — which is what the corrected cells rest on:

> "Any client request must prove its identity at the transport layer using client certificates."
> — `THREAT_MODEL.md`, `etcd-io/etcd` `v3.7.1`, *The Client-to-Server Boundary*, the section naming
> Port 2379

> "This boundary must be strictly limited to authorized cluster members using dedicated, private peer
> certificates (mTLS)."
> — same document, *The Peer-to-Peer Boundary*, the section naming Port 2380

Such a sentence may be **prescriptive** and may describe a deployment the owner does not ship. That
is not a defect: Step 1 reads the owner's statement of whom the protocol is for, and Step 2 reads the
shipped bytes for what an unidentified caller actually gets. **The gap between the two is the Claim 1
row.** A reading that collapses them leaves one of the steps with no work to do.

### Limb 2 — Step 2's reachability is read at the internet-reached frame

§20.6's *can an unauthenticated remote caller reach it?* is answered against the configuration in
which the port **is reached from the internet** — the state `sensitive-port-reached-from-internet`
fires on and the only state this list is about — with the **authentication** configuration held at
its shipped default. Three consequences, and each is an existing finding read at this frame:

1. A shipped default that **withholds the port** — does not listen, listens on a unix socket, binds
   loopback, or refuses non-loopback peers **by address** — is the network act the frame assumes. It
   neither answers Step 2 nor defeats Claim 1. This is §19.6, with its *"turns the port off"* read
   for what it does rather than for how it does it.
2. A shipped default that **authenticates the caller** answers Step 2 in the negative and defeats
   Claim 1. This is §19.5's `10250/tcp`, which left Class A on exactly this.
3. A gate on **particular operations** is read per operation, and the row survives on whichever limb
   retains one. This is §20.6's `4369/tcp`, which survived on the read limb.

The discriminator between (1) and (2) is whether the operator can reach anonymous remote service
**without supplying a credential**. Redis's `protected-mode no` demands none of anyone, so `6379/tcp`
falls under (1); the kubelet's `anonymous-auth: false` demands one, so `10250/tcp` fell under (2).

## Consequences

- **No row moves, and no figure moves.** §1 stays 37 pairs, §3's class totals stay `11 / 7 / 19`,
  §2.2's footing table stays 26 of 37 at tiers `15 / 9 / 2 / 11`, §6.1's containment arithmetic is
  untouched and [ADR-0009](0009-verge-core-is-a-union.md)'s union gains and loses no member. Three
  §10.1 cells are corrected in §25.
- **§10.1's Class A walk is closed.** Every cell has been checked against §10.5 and every Step 2 cell
  read off the dispatch. All eleven rows survive.
- **A future Step 1 cell may not be repaired with a prohibition.** Where no owner sentence answers
  Step 1 directly, the honest outcomes are to retrieve one or to report that the row's Step 1 has no
  footing — not to substitute the row's Claim 3 evidence.
- **The cost of limb 2, stated.** It admits rows whose shipped default is genuinely safe against the
  anonymous internet caller, `6379/tcp` most sharply. That is the intended direction: this list
  answers *what is never correct to expose*, and a protection the operator must switch off — with no
  credential and one config line — is not authentication. Narrowing limb 2 later takes `6379/tcp`
  first, and takes it to **Class C** on Redis's own *"trusted clients inside trusted environments"*,
  not off the list.
- **Where it is thin.** Limb 1 is decided on argument: no cell in #92 turns on it, because the etcd
  retrieval found a sentence answering Step 1 directly. It is minted anyway because the ruling is
  what forced that retrieval — under the other ruling the cell would today rest on the shape §10.1
  flags as a tell. Limb 2 is forced by measurement and is not thin.
- **Why this goes the other way from §16.6's and §20.9's ADR refusals.** Those applied §16.6's test —
  *does the rule state what an existing section already requires?* — and answered yes. Here the
  answer is no for both limbs: nothing states limb 1, and limb 2's extension of §19.6 from row
  admission to Step 2 is a choice whose alternative empties seven of eleven rows out of Class A.
  §20.6's own rule is untouched and remains an ADR-less corollary.

### Note — [#95](https://github.com/winniel123/verge-asm/issues/95): limb 1 cuts both ways, and its *thin ground* disclosure is discharged

**A cell turns on limb 1 now.** [#95](https://github.com/winniel123/verge-asm/issues/95) admitted
`10249/tcp` kube-proxy metrics to Class A, and its **Step 1** rests entirely on limb 1's admissible
shape — Kubernetes' own metrics documentation stating that *"reading metrics requires authorization
via a user, group or ServiceAccount with a ClusterRole that allows accessing `/metrics`"*, in a
document that names kube-proxy in its own list of components exposing the endpoint. **[measured]**
that sentence is conditioned on RBAC and RBAC is not in the request path for `10249`, whose metrics
mux carries no authorization filter at all — which is limb 1's *"may be prescriptive and may describe
a deployment the owner does not ship"* met exactly, and *"the gap between the two is the Claim 1 row"*
paying out for the first time. The *Where it is thin* bullet above — *"no cell in #92 turns on it"* —
is **discharged**.

**And the limb runs in both directions, which #92 had no occasion to state.** The same owner ships a
second sentence about `10249`: `source-ip.md` tells the reader to run
`curl http://localhost:10249/proxyMode` *"in a shell on the node you want to query"*. Read as Step 1
evidence that would **refuse** the step — the owner instructing an unauthenticated fetch. It may not
be used that way. **It is a statement about where the caller stands, which is Claim 3's subject, and
limb 1 bars a placement sentence from Step 1 without regard to which way it points.** A limb that
excluded boundary evidence only when it helped the row would be a thumb on the scale rather than a
rule; #95 used the sentence at its footing, where it belongs, and rested Step 1 on the authorization
sentence alone. [`sensitive-ports.md`](../research/sensitive-ports.md) §27.3.

**Limb 2's arithmetic moves and its point is unchanged.** *Seven of eleven* Class A rows whose shipped
default withholds the port becomes **eight of twelve**: `metricsBindAddress` defaults to
`127.0.0.1:10249`, and the discriminator resolves to case (1) because reaching the port remotely takes
one config line and demands a credential of nobody.

**One thing #95 had to keep apart, recorded because the confusion is easy.** The same loopback default
is *silent for limb 2* — the network act the frame assumes — and *attesting under §10.4's one-way
rule*, where it founds the row's footing. The two answer different questions: limb 2 asks **in which
configuration reachability is read**, §10.4 asks **whether a default carries evidence**. A session
that reads limb 2 as making a restricting default silent for all purposes would take `5432/tcp` off
the list.


## Annotation — [#96](https://github.com/winniel123/verge-asm/issues/96)

**The Decision and both limbs are unchanged.** What is recorded is a gap limb 2 left, found the first
time the frame was applied to a row outside Class A.

Limb 2 answers Step 2's reachability *"holding the **authentication** configuration at its shipped
default"*. **All eleven Class A rows have a shipped default to hold. `623/udp` IPMI has none**, and
the owner says so: **[measured]** IPMI v2.0 rev 1.1 §22.22, page 300 — *"The choice of factory default
setting for the non-volatile parameters is **left to the implementer or system integrator**."* The
specification is a multi-vendor consortium's and ships no implementation; a BMC's configuration is
firmware ([`sensitive-ports.md`](../research/sensitive-ports.md) §13.1).

> **Where the owner ships no configuration, the frame reads the specification's own dispatch.** This
> is §10.1 as written — both steps are answered *"from the specification or the owner's own
> documentation"* — and for IPMI the dispatch is **Table G-1's `p` notation**, which marks the
> commands executable before a session is established. A **vendor's** firmware default is admissible
> under §2.2's third form **over that vendor's own product** and, for the protocol, falls under
> §10.5's distributor fence as amended by §12: never sole grounds.

**It did not decide the row, which is why it is an annotation and not an ADR.** Two of the four
co-authors turned out to document a **restricting** default — Dell's `iDRAC.IPMILan.Enable — 0 -
Disabled`, HPE's *"This setting is disabled by default"* — and limb 2's own first bullet already rules
that a default withholding the port neither answers Step 2 nor defeats Claim 1. `623/udp` stays
**Class C**: Claim 1 is available at Step 1 and fails at Step 2, because nothing in the pre-session set
writes, deletes, registers, deregisters, executes, controls, or reads content carried for another
party. §28.4, §28.5, §28.6.

**Two arguments were refused that a later session will re-derive if this is not written down.**
**Cipher Suite 0** — the specification marks RAKP-none `M` for BMC *support* (Table 13-17) and states
it *"provides a way to enable access to the BMC without requiring a username and password"* (§13.28.2)
— loses because mandatory **support** is not a shipped **configuration**, and because the reading that
collapses them moves `5432/tcp` PostgreSQL to Class A on its `trust` method, along with `3306`, ~~`1433`~~
(**removed from the list by [#109](https://github.com/winniel123/verge-asm/issues/109)**; the argument
is unchanged and loses one of its four examples)
and `2049`. And **RAKP Message 2's password-keyed `HMAC_K[UID]`**, disclosed before the caller proves
anything (§13.31), loses because a handshake message is not an **operation**, because the value is
computed rather than carried, and because it is a **Claim 2**-shaped fact on a protocol where §10.8
already makes Claim 2 structurally unavailable — IPMI negotiates encryption in-band on `26Fh` rather
than on a successor port, which is RFC 6335 §9's pattern this instrument is blind to by construction.
**§10.2's closed claim set met its sharpest candidate fourth to date and held**: a leaky authenticator
is a property of the protocol, not a fourth property of an internet vantage.

**No `(port, transport)` pair moves, no row moves, no class total moves, no tier moves.** The list
stays at **39 pairs**, classes **`11 / 7 / 21`**, coverage **28 of 39**. **No ADR is minted and 0057 is
left unused**, on the precedent of #76, #84, #87, #90, #91 and #93.

> **Merge reconciliation.** *"Nothing moves"* is true of **#96's own act** and stays on the record; the
> absolutes are as of its own pass. [#95](https://github.com/winniel123/verge-asm/issues/95) admitted
> `10249/tcp` and `10248/tcp` and [#98](https://github.com/winniel123/verge-asm/issues/98) moved
> `873/tcp` between footing tiers in the same merge, so composed the list is **41 pairs**, classes
> **`12 / 7 / 22`**, coverage **30 of 41** at tiers **14 · 13 · 3 · 11**. **`623/udp` is untouched by
> both.** [`sensitive-ports.md`](../research/sensitive-ports.md) §1 carries the current absolutes.
> **Composed again after [#107](https://github.com/winniel123/verge-asm/issues/107) and
> [#109](https://github.com/winniel123/verge-asm/issues/109): the list is 40 pairs, classes
> `12 / 7 / 21`, coverage 29 of 40 at tiers 15 · 11 · 3 · 11** — `1433/tcp`'s cell left the graded
> table and then its row left the list. **`623/udp` is untouched by both of those too.**
> **Composed again after [#114](https://github.com/winniel123/verge-asm/issues/114): the list is 38
> pairs, classes `12 / 7 / 19`, coverage 27 of 38 at tiers 13 · 11 · 3 · 11** — `9200/tcp` and
> `9300/tcp`'s rows left the list and their cells left with them. **`623/udp` is untouched by that
> too**, and this ADR is **confirmed by use**: §38.2's admission of Elastic's HTTP-layer sentence turns
> on its being evidence about **the claim**, at the claim.
