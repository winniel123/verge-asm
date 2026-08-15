# Project-authored constants: which are products of a moving world quantity?

- **Ticket:** [#71 Which project-authored constants are products of a moving world quantity rather than the quantity itself?](https://github.com/winniel123/verge-asm/issues/71)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)
- **Ruling:** [ADR-0038](../adr/0038-a-constant-is-a-product-only-where-the-quantity-is-readable.md)
- **Rule being applied:** [ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md) §4 — *where a
  project-authored constant is the product of a fraction and a moving world quantity, ship the fraction.*

---

## 0. The answer, up front

**The sweep found one instance, and it is the one already fixed.** Every other project-authored
constant in the repository either is not a product, or is a product of a quantity the rule cannot
read — and in the second case the cure is a **scheduled edit**, never a fraction, on grounds two
other tickets had already written down without knowing they were answering this one.

| Verdict | Constants |
| --- | --- |
| **Product, curable, cured** | `certificate-expiring`'s `N` — [#67](https://github.com/winniel123/verge-asm/issues/67), already done |
| **Product, *not* curable — the quantity is not on the subject** | `verge-core`'s frequency half; `certificate-weak-key-or-signature`'s five rows |
| **Not a product** | the connect timeout and retry budget; `k`; the availability window; #61's staleness bound; every rate and concurrency figure; the EDNS payload size; the coverage threshold (not project-authored at all) |

**Nothing is re-expressed, and nothing new is minted.** The two products that fail the reach test
were **already correctly disposed of** — one as aperture ([ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §5,
[#31](https://github.com/winniel123/verge-asm/issues/31)), one as a scheduled edit
([`weak-key-and-signature.md`](./weak-key-and-signature.md) §7.3) — so the sweep's output is a
**bound on the rule's reach**, which is what [#71](https://github.com/winniel123/verge-asm/issues/71)
asked for and expressly preferred to a longer list.

**And the sweep's most damaging finding is not an instance of the rule at all.** §7 measures a
place where the world moving has made an **accepted safety argument** wrong — `certificate`'s
currency bound, priced in [ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)
against 90-day certificates one day before [#67](https://github.com/winniel123/verge-asm/issues/67)
retrieved that six-day certificates are generally available. No constant there is a product and no
re-expression fixes it. That is the honest shape of the residual risk and it is recorded as such
rather than bent to fit.

---

## 1. The test, in four limbs

ADR-0034 §4 states the rule but not its scope. The sweep needed a test, and the test that fell out
of applying it to eighteen constants has **four** limbs, of which ADR-0034 names only the first two.

> A project-authored constant `C` is a **product** in this rule's sense exactly where:
>
> 1. **Form.** `C = f(Q)` for a project-authored rule `f` and a quantity `Q` the project did not choose.
> 2. **Motion.** `Q` moves, and moves with no document the constant cites being retracted.
> 3. **Reach.** `Q` is readable **from the subject, at evaluation time**, by machinery the rule already has.
> 4. **Silence.** When `Q` moves, the constant's wrongness produces no signal — no failing build, no
>    `not-evaluable`, no version move, no visible symptom.

**Limbs 1 and 2 make a constant *stale-able*. Limb 3 makes it *curable by construction*. Limb 4
makes it *worth curing*.**

Limb 3 is the one that does the work, and ADR-0034 has it implicitly: *"a row expressing the
fraction reads the moving quantity from the subject at evaluation time, so it cannot go stale."*
The *cannot* is the whole of the cure, and it is available only where the subject carries `Q`. A
certificate carries `not_before` and `not_after`; it does not carry NIST's strength table, the
internet's port-frequency distribution, or a calendar. **Where limb 3 fails, re-expressing the
constant as a fraction does not remove the staleness — it relocates it**, because the shipped
artefact must then carry a value for `Q` anyway, and that value is the stale thing wearing a
different name.

Limb 4 is the ticket's own price argument read as a test. ADR-0034 ranks silent staleness worse
than silent de-attestation *because* nobody has to do anything. Where something **does** happen —
a build fails, a rule goes `not-evaluable`, a version moves — the constant is stale-able but not
silently so, and the defence already exists.

`certificate-expiring`'s `N` passes all four: `30 = ⅓ × 90`; Let's Encrypt publishes the lifetime
schedule; `not_after − not_before` is on the certificate; and nothing signalled — indeed **nothing
signalled for the whole time `N` was wrong**, which is #67's measured point. It is the only
constant in the repository that passes all four.

### 1.1 The sweep's population

*Project-authored constant* is read as **a number or a set the project chose and ships in the
release** — [#60](https://github.com/winniel123/verge-asm/issues/60)'s declared parameters, plus the
curated tables under ADR-0032's three instruments, plus the operational budgets
[ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) puts outside every leaf. It
excludes an operator's dial, which is not project-authored, and it excludes a value we read from an
owner and pass through unmodified, which is not a product of anything of ours. Both exclusions
retire one of the ticket's five named candidates; see §6.3 and §6.6.

---

## 2. The instance, and why it is the only one — `certificate-expiring`'s `N`

Settled by [#67](https://github.com/winniel123/verge-asm/issues/67) and
[ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md); the retrieval is in
[`acme-renewal-timing.md`](./acme-renewal-timing.md). Recorded here only as the sweep's worked
example of all four limbs passing, and for one scope correction in §8.2.

| Limb | | Evidence |
| --- | --- | --- |
| Form | ✓ | `30 = ⅓ × 90`. The issuer's prose carries the arithmetic and its scope in one breath |
| Motion | ✓ | 90 → 64 on 2027-02-10 → 45 on 2028-02-16, published by the issuer; the CA/B ceiling 398 → 200 → 100 (2027) → 47 (2029) |
| Reach | ✓ | `not_after − not_before`, on the certificate, already read by `certificate-not-yet-valid` |
| Silence | ✓ | `N = 30` was **stale before it was written** and nothing anywhere said so |

---

## 3. Product, not curable — `verge-core`'s frequency half

`C = ` the ~~~140-port~~ **123-port** continuous set. `f = ` *the top hundred by open-frequency, minus
the ephemeral and obsolete tail, plus a modern-services supplement*. `Q = ` the distribution of open
ports across the internet.

> **The size is corrected and the finding is unchanged.** **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97): the frequency half — which is what `C` is,
> `f` being the frequency half's own selection rule — is **123, all TCP**, and *"~140"* was never
> reproducible from [#4](https://github.com/winniel123/verge-asm/issues/4) §2.3's two limbs
> ([`sensitive-ports.md`](./sensitive-ports.md) §29). `verge-core` as a whole is **136 pairs** under
> ADR-0009's union, but that is `C ∪ sensitive-list` and is not this section's constant. **§3.1's
> staleness measurement and §3.2's limb-3 failure are untouched** — neither reads the count.

### 3.1 It passes limbs 1, 2 and 4, and it is already stale — measured

[`safe-active-probing.md`](./safe-active-probing.md) §2.2 calls this **the load-bearing finding** of
that note and measures it directly against the shipped file: the header reads
`$Id: nmap-services 9746 2008-08-26 18:45:24Z fyodor $`, and the consequence is measured rank by
rank — **Redis at 1,683**, Docker's daemon API at 2,903, memcached at 8,317, etcd and the Kubernetes
API at frequency `0.000000`, and **kubelet not present in the file at all**.

So limbs 1, 2 and 4 are satisfied and the constant is **already stale, by eighteen years, and
measured rather than assumed**. This is the ticket's own observation — *"the same defect, already
measured, and never filed as one"* — and it is confirmed.

### 3.2 It fails limb 3, and there is no fraction to ship

No `Address` and no `Service` carries the internet's port-frequency distribution, and no rule can
read it at evaluation time. There is no fraction. The shape of the cure that worked for `N` is
simply unavailable here, and proposing one would mean shipping a *table* of frequencies instead of
a *set* of ports — the same staleness, one indirection out, plus a table.

### 3.3 It was already mitigated by construction, in the one half that could be

The list is not `top_100(D_2008)`. §2.3 drops the 2008 artefacts by name (`1025–1029`,
`49152–49157`, `2717`, `5101`, `5190`, `6646`, `3986`, `5051`, `5009`, `1755`) and adds a
supplement **chosen because each port maps to a named v1 risk signal**. That criterion is a
**rule**, and it is project-internal — it does not move when the world does.

> **The stale half of `verge-core` is exactly the half not expressed as a rule.** ADR-0009 already
> made the other half a rule: `verge-core = frequency-set ∪ sensitive-list`, and the sensitive list
> is defined by a claim and an attestation rather than by a frequency. The union was built to make
> an invariant unfalsifiable; it also, unremarked, converted half of a stale product into a
> rule-expressed set.

This generalises the cure and is §8.1's ruling: **ship the rule that generated the number wherever
the rule's inputs are available at evaluation time.** A fraction is one such rule — the one whose
input happens to sit on the subject.

### 3.4 Its residual cost is aperture only, which is why it stays

ADR-0032 §5 places the frequency half outside gate 2 as **aperture**, governed by #31's line. A
stale aperture makes us **not look**; it can never make us conclude wrongly. So the failure mode of
`verge-core`'s staleness is a missed listener recorded as *we did not look there*, and #44's
standing aperture statement on `Coverage` is where that already lands.

**It is not deleted, because the job is real.** The frequency half exists to catch a listener no
signal names — drift on an unexpected open port — and the rule-expressed half cannot do that by
construction, since a port is in it only where a signal already names it.

### 3.5 Disposal

**No change, and no new object.** It goes on the port-curation patch, which the map already routes
it to, under §8.3's widened watch criterion. §5 records what was retrieved about whether a live
replacement for the 2008 data exists.

---

## 4. Product, not curable, and *must not* be — the weak-key table

`C = {RSA nlen < 2048, ECDSA len(n) < 224, DSA below (2048, 224)}`. `f = ` *below 112 bits of
security strength*. `Q = ` NIST's strength-to-key-size mapping, and its dated transitions.

### 4.1 The form is explicit in the owner's own sentence

[`weak-key-and-signature.md`](./weak-key-and-signature.md) §3.3 quotes NIST SP 800-131A Rev 2 and
the primacy is unambiguous — the **strength** is the requirement and the **size** is the lookup:

> *"RSA: The length of the modulus n shall be 2048 bits or more **to meet the minimum
> security-strength requirement of 112 bits** for Federal Government use."*

> *"Therefore, the length of n shall be at least 224 bits **to meet the minimum security-strength
> requirement of 112 bits** for Federal Government use."*

So `2048` and `224` are **evaluations of one rule at one date** — precisely `30`'s shape, and the
file says so in as many words at §2.4: *"2048-bit RSA and a 224-bit curve order both deliver about
112 bits of security."* Limb 1 passes. Limb 2 passes on the owner's own published schedule: §7.1
tables *"the next scheduled move — the 112-bit level's end of life — ~2030."*

The retrieval sharpens it three ways, all against bytes:

- **The requirement is stated as a strength and never as a size.** SP 800-131A Rev 2 §1.2.1:
  *"a security strength of at least 112 bits is required at this time for applying cryptographic
  protection."* Its footnote concedes the other vocabulary is not NIST's — *"The term 'key size' is
  commonly used in other documents."*
- **All three key rows are one table row, transcribed.** SP 800-57 Part 1 Rev 5 **Table 2** at the
  112-bit level reads `FFC L = 2048, N = 224` · `IFC k = 2048` · `ECC f = 224-255`. Our RSA 2048,
  ECDSA 224 and DSA (2048, 224) are that row and nothing else.
- **`224` is `112 × 2` with the multiplication written out**, so the table has flattened a
  derivation twice over — once from strength to size, and once through an arithmetic the owner
  performs in the sentence itself.

**One contrast worth carrying.** The CA/Browser Forum states the same number as a **bare size** with
no rule behind it — BR v2.2.9 (6 Aug 2026) §6.1.5: *"Ensure that the modulus size, when encoded, is
at least 2048 bits"*, and for ECDSA it does not state a size at all but enumerates three named
curves. The phrase *security strength* does not appear. So whether our `2048` is a product **depends
on which owner it is routed through** — and [ADR-0035](../adr/0035-a-cryptographic-primitives-owner-is-its-specifier.md)
routed it through the **specifier**, which is NIST. Our table therefore has CA/B's *shape* and
claims NIST's *routing*, and it is the routing that decides: **product.**

### 4.2 Limb 3 fails, and #68 had already ruled the re-expression out on independent grounds

No certificate carries NIST's mapping. A certificate carries a modulus length; the table that turns
a modulus length into a security strength lives in a NIST publication and would have to ship with
us. So the fraction is unavailable, and **two separate rulings already on the books forbid reaching
for it anyway** — both written by [#68](https://github.com/winniel123/verge-asm/issues/68) without
knowing it was answering #71:

- **§2.4 — the key may not be a bare bit count.** *"A session tidying this table into a single bit-count
  threshold — keys under N bits are weak — would be keying on a **surrogate for security strength**,
  and gate 3 would come inside the domain and fail immediately, because the same integer means two
  incompatible things on a modulus and on a curve."* Re-expressing the table in strength is the
  same move with a better-sounding unit: it makes the shipped artefact a mapping we author, over
  which gate 3 has an opinion.
- **§7.3 — a scheduled transition is a scheduled edit.** *"Encoding any of these as a **date comparison
  in the predicate** would be a quiet violation: the rule's output would move at midnight on a date
  with no version bump and no release … That is release-coupling defeated from the inside."*

### 4.2a The retrieval found the reason the fraction would fail *even if reach were satisfied*

This is the sharpest thing the sweep retrieved and it is worth stating separately, because the
obvious objection to §4.2 is *fine, but ship `≥ 112 bits` anyway, it is more honest*.

**It is not, and the reason is measured.** The scheduled move is **not** `2048 → 3072`. Comparing
SP 800-57 Part 1 **Rev 4** (Jan 2016) with **Rev 5** (May 2020), Table 2's 112-bit row is
**numerically identical** — `L = 2048, N = 224 / k = 2048 / f = 224-255` in both. What Rev 5 changed
was the **floor beneath it**: *"a security strength of 80 bits is no longer considered adequate."*
And Rev 5's **Table 4, *Security strength time frames*** (FINAL) puts 112-bit strength at
**Disallowed for applying protection from 2031**, with the sizes still unchanged.

> **A table shipping `≥ 112 bits` would be exactly as stale as one shipping `≥ 2048`, because the
> quantity that moves is *whether 112 is enough*.** `f` is not project-authored either — it is
> NIST's floor, and NIST moves the floor. **There is no fixed point to ship.**

That is the general shape of a reach failure at its worst: the derivation can be unwound one level
and the next level up is moving too. Unwinding it further reaches *"as much security as is currently
adequate"*, which is not a predicate.

**Disclosed, per ADR-0034 §7's performed-retrieval limb.** The dates are thinner than they read.
**SP 800-131A Rev 3** and **NIST IR 8547**, which carry the *deprecated after 2030-12-31 /
disallowed after 2035* language, are **both Initial Public Drafts** and have sat in draft about
twenty-one months — `csrc.nist.gov/pubs/sp/800/131/a/r3/final` and `csrc.nist.gov/pubs/ir/8547/final`
both return **HTTP 404**, corroborated by each `/ipd` landing page showing a draft-only publication
history. The **only final normative date** is SP 800-57 Table 4's **2031**, which the drafts propose
to *soften* to a deprecation. **Established:** the thresholds are one transcribed table row and the
rule behind them is NIST's, retrieved. **Unestablished:** which date governs. **The condition that
would move it:** either draft going final. **It changes no row today**, because #68 refused to
encode dates at all — which is §7.3 earning its keep before the ambiguity arrived.

### 4.3 Limb 4 fails too, and that is the general point

The 112-bit transition is **published, dated and diarised**. Somebody can put ~2030 in a calendar,
and the map already has the entry — *re-read SP 800-131A when a revision goes final and otherwise do
nothing*, priced at **about one edit per row per decade** (§7.1: nine changes in twenty-two years
across five rows).

> **Where `Q` moves on a schedule its owner publishes but the rule cannot read `Q`, the answer is a
> scheduled edit, and the staleness rule is inapplicable rather than violated.**

That is §7.3 promoted from a local note to the general counter-rule, and it is the boundary the
staleness rule needed. The two rules do not compete: they partition on limb 3.

### 4.4 Disposal

**No change.** Retrieval confirming the mapping has not moved and the transition remains scheduled
is at §5.

---

## 5. What was retrieved

The ticket binds: *where a constant's staleness is claimed, measure it against a retrieved source.*
Three retrievals were run. Two tested a claimed staleness; one tested a claimed **non**-staleness,
because a sweep that only looks where it expects to find something is the failure ADR-0034 §1 is
about.

### 5.1 `nmap-services` — still 2008, never re-measured, and #4's ranks are artefacts

**Fetched 2026-08-14**, `https://raw.githubusercontent.com/nmap/nmap/master/nmap-services`, HTTP
200, 998,268 bytes, 27,483 lines. The header is **byte-identical** to what #4 measured:

```
# $Id: nmap-services 9746 2008-08-26 18:45:24Z fyodor $
```

**Corroborated against the changelog**, which is the stronger evidence, since a stale `$Id` alone
could mean the marker was dropped rather than the data frozen.
`https://raw.githubusercontent.com/nmap/nmap/master/CHANGELOG` mentions the frequency data **once**,
under `Nmap 4.75 [2008-9-7]`:

> *"Expanded nmap-services to include information on how frequently each port number is found open.
> The results were generated by scanning tens of millions of IPs on the Internet **this summer** …"*

Across the whole 7.x series — `7.00 [2015-11-19]` through `7.991 [2026-08-06]` — there is **no
second entry**. The file's commit history is IANA name syncs and one-off manual nudges
(e.g. *"Bump up wsman (winrm) port 5985 and 5986 frequency as these are commonly seen"*, 2023-07-20).
And nmap issue **#2399**, *"List of most common ports / open-frequency?"*, filed 2021-11-20, asking
this exact question, is **still open and unanswered**.

**Eighteen years, confirmed against bytes rather than remembered.**

#### The correction: Redis and Docker were never *ranked* low — they were never *measured*

The retrieval refines #4 §2.2 in a way that makes its argument stronger and its numbers wrong.

`safe-active-probing.md` §2.2 tables *"6379 Redis — rank by open-frequency **1,683**"* and
*"2375/2376 Docker daemon API — **2,903 / 2,902**"*. Measured now, both lines exist and carry:

```
docker  2375/tcp  0.000076  # docker.com | Docker REST API (plain text)
redis   6379/tcp  0.000076  # An advanced key-value cache and store
```

**`0.000076` is not a measurement.** It is a filler plateau: **1,969 TCP lines carry that exact
value**, against 57 at `0.000075` and 42 at `0.000088`. GitHub blame dates both lines to
**2016-09-14**, commit *"Merge latest IANA services. Includes 446 previously-unknown services"* —
so Redis and Docker entered the file from IANA's **name** registry **eight years after the only
scan**, and nothing was ever measured for either. 1,441 TCP ports rank strictly above `0.000076`
and the tie block spans ranks **1442–3410**, so any rank quoted inside that window is a tie-break
artefact of whatever sort was used.

> **The honest statement is stronger than the one in the note.** It is not that modern services rank
> low in the 2008 data; it is that **they are not in the 2008 data at all** — they are 2016 name
> imports wearing a placeholder. A frequency set built from this file does not under-weight modern
> exposure. It has **no information about it whatever**.

`10250/tcp` (kubelet) is confirmed **genuinely absent** — zero matches over the whole file. And the
140th-ranked TCP port sits at `0.002129` (`cslistener 9000/tcp`), about **28× the filler value**, so
none of Redis, Docker or kubelet is anywhere near a frequency-selected set of any plausible size.
**#4 §2.3's supplement is not a refinement of the ranking; it is the only reason these ports are in
`verge-core` at all** — which is §3.3's point, measured.

Exact amendment text for `safe-active-probing.md` is in the resolution comment. **That file is not
edited here.**

#### No free, redistributable replacement exists

Checked because §3.5's *no change* would be wrong if a live source were available.

| Source | Free port-frequency ranking? | Redistributable by an AGPL-3.0 project? |
| --- | --- | --- |
| **Shodan** | **Yes — public, no login** | **No** — no data licence granted at all |
| Censys | No — aggregation is paid-tier | **No, barred outright** |
| Rapid7 Project Sonar | Counts in reports only, not ranked | No — commercial; bulk redistribution barred |
| scans.io | No | Non-commercial only |
| ZMap | Tool only; publishes no data | n/a |
| Shadowserver | No public per-port breakdown *(unconfirmed — see below)* | No — non-profit research, no sublicensing |

**Premise correction carried from the retrieval: Project Sonar was never discontinued.** What ended
was free public access, announced 2022-02-10 — *"Free access (no account required) to a one-month
window of recent data from Project Sonar … **Beginning today, the latter will no longer be
available.**"* The project is alive and commercial, and `opendata.rapid7.com` now 301s to
`sonardata.rapid7.com`. Terms: *"The data cannot be redistributed in bulk."*

**Shodan is the one live, freely readable ranking**, and it is not vendorable. Its terms grant a
licence only over *"the software provided to you by SHODAN"* — personal and non-assignable, with no
data licence — and require that *"the materials referencing, including or otherwise based on Shodan
information or materials, **must clearly indicate Shodan's ownership and copyright**."* A mandatory
third-party ownership assertion cannot ride inside an AGPL-3.0 tree that must convey full
redistribution rights downstream. **Censys is barred in terms** (ToS §3.1): *"**Under no
circumstances** may any Customer … incorporate any Censys Data into its own software products or
services that are distributed or otherwise made available to a third party."*

This is [#27](https://github.com/winniel123/verge-asm/issues/27)'s CAIDA finding a second time —
*redistribution is a separate permission from use, and the party who needs it is the project* — and
it is why §3.5 rules **no change** rather than *re-source it*.

**Marked as partly unconfirmed:** the Shadowserver API documentation fetch failed, so *no public
per-port endpoint* there is **unestablished**, not established. Shodan's terms carry no revision
date and are quoted as served on 2026-08-14.

### 5.2 The prober's cited defaults — none moved, and the citation is wrong anyway

Run because §6.1 needed the *non*-staleness measured, not assumed. Eleven upstream defaults were
re-fetched.

**No cited number has moved.** naabu's `DefaultPortTimeoutConnectScan` is still `3 * time.Second`;
`DefaultRateSynScan` 1000; `DefaultRateConnectScan` 1500; httpx's `-timeout` still 10, `-rate-limit`
150, `-threads` 50, `-retries` 0; masscan still *"By default, the rate is set to 100
packets/second"*; tlsx still 300 / 5 s / 3. **So the 3 s timeout is not stale, measured.**

Three things the retrieval found that the numbers hide, all of which bear on §6.1's disposal:

- **A citation that never matched the bytes.** #4 §6.3 pairs *"3 s connect timeout, **2 retries**"*
  under one justification — *"matching naabu's `DefaultPortTimeoutConnectScan`"*. The timeout
  matches. **The retry count does not, and never did on any branch measured:** naabu is
  `DefaultRetriesConnectScan = 3`. **The 2 is our own choice wearing naabu's citation.** That is
  not staleness — it is #60's defect exactly, a rationale that was false when written — and it is
  the second measured instance of it in this effort.
- **The cited URLs point at a stale mirror, and this is the `lego` hazard live again.** Both
  ProjectDiscovery repos' default branch is **`dev`**, not `main`. naabu's `main` HEAD is
  **2026-05-05**; `dev` is **2026-08-10**, and the cited file **differs between them**. #67 logged
  this exact failure shape on lego's `master` — *"not a blocked fetch but a successful fetch of
  superseded bytes"* — and it is now live on a second citation.
- **A future de-attestation channel opened upstream.** naabu commit *"faster scanning (#1712)"*
  (2026-07-29, `dev`) moved the constants into an nmap-style timing template. `T3` reproduces the
  old numbers today, so nothing has changed — but *"naabu's default"* is now a **template
  selection**, and an upstream decision to ship a different default `-T` would move rate, retries,
  timeout and threads **at once, without touching a single constant we cite**. That is ADR-0032 §8's
  silent de-attestation, one level down into a corroborator.

**None of this changes a number**, because §6.1 rules the timeout is not a product and ADR-0032 §5
puts it outside gate 2 anyway. It changes a **footing**, and the footing is repaired in the
resolution comment's amendment text.

### 5.3 NIST's strength-to-size mapping — the derivation is in the owner's own sentence

Reported at §4.1 and §4.2a rather than repeated here. In summary: SP 800-131A Rev 2 (**FINAL**,
March 2019) and SP 800-57 Part 1 Rev 5 (**FINAL**, May 2020, Tables 2 and 4) were retrieved as
PDFs and extracted locally; the strength is normative and the size is the lookup; all three of our
key rows are one transcribed table row; and the scheduled move is a **tier status change** rather
than a size change. SP 800-131A **Rev 3** and **IR 8547** are both **Initial Public Drafts** —
`csrc.nist.gov/pubs/sp/800/131/a/r3/final` and `csrc.nist.gov/pubs/ir/8547/final` both **404** — so
their 2030/2035 dates are **not normative** and are not cited as settled anywhere in this note.

### 5.4 By-catch: a possible licence problem with `verge-core`'s derivation, which is not this ticket's

Surfaced by §5.1's retrieval while establishing whether `nmap-services` could be re-sourced, and
recorded here **as a question rather than a finding**, because it was not retrieved to the standard
this repository holds a verdict to.

Nmap ships under the **NPSL**, not a plain GPL. NPSL v0.95 §3 is reported to define a derivative
work so as to include software that *"Reads or includes Covered Software data files"*, and to
require distribution *"under the terms of this license (including this Main License Body and GPL),
with no additional conditions or restrictions."* NPSL is GPLv2-derived with added terms, and
**AGPL-3.0 is not compatible with GPLv2**.

`verge-core`'s frequency half is **derived from `nmap-services`**, and the weekly tier **is** nmap's
top-1000. If that reading holds, the exposure is to the map's own standing instruction —
*"flag anything that would force a licence change"* — and it sits squarely on
[#27](https://github.com/winniel123/verge-asm/issues/27)'s line, which killed bundling CAIDA on
exactly this ground.

**It is not ruled here and no verdict is implied.** The NPSL text was not retrieved and quoted
against bytes in this sweep, and whether a set of integers selected from a data file is a derivative
work of it is a question this note has no instrument for. Opened as
**[#78](https://github.com/winniel123/verge-asm/issues/78)**, which also carries the practical
consequence §5.1 measured: **the only AGPL-clean source of open-port frequency data is one we
generate ourselves.**

---

## 6. Not products — the sweep's negatives, walked

The ticket's list is deliberately not closed, so this section walks **every** remaining
project-authored constant, not only the named candidates. A negative is recorded with its reason,
because *the reason it is not a product* is what stops the next session re-opening it.

### 6.1 The connect timeout (3 s) and retry budget (2) — [#4](https://github.com/winniel123/verge-asm/issues/4), versioned per [#49](https://github.com/winniel123/verge-asm/issues/49)

The ticket asks the right question: *a timeout is a judgement about round-trip time, which is a
world quantity. Is 3 s a fraction of something, or is it genuinely about our own tolerance?*

**It is genuinely about our own tolerance, and the case is stronger than that — the re-expression
is already forbidden.**

- **Limb 1 fails.** 3 s is not `f(RTT)` for any `f`. It is a **classification boundary**: below it a
  verdict, above it `no-response`. Its stated footing is another tool's shipped default, which is a
  corroborator (ADR-0032 §2.3, and ADR-0034 §5 rules a required-parameter default a corroborator),
  never an arithmetic relation to a measured quantity.
- **Limb 3 is available, and taking it is prohibited.** RTT *is* observable per connect, so an
  adaptive deadline is technically reachable — and
  [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md) rejected it in terms:
  *"Adaptive back-off inside `connect-outcome` — **it halves the rate, never the deadline** — and had
  it moved the deadline, **a value would depend on how busy the run was**."* A deadline that reads a
  measured quantity makes `connect-outcome` non-deterministic, and the golden-corpus gate ADR-0021
  built is exactly the thing that would fail. **This is the ticket's *genuinely should not* case,
  and it was already ruled — the sweep only had to notice.**
- **Limb 4 fails in the direction that matters.** If round-trip times rose enough for 3 s to
  manufacture `no-response`, the symptom is visible: #4 §6.3 already mandates recording a
  `throttled` event and refuses silent back-off — *"an operator seeing 'scan completed at 12 % of
  configured rate' learns something real about their infrastructure."*

**But the retrieval found a real defect in its footing**, of a different and smaller kind. See §5.

### 6.2 `k`, the currency multiplier — the sweep's positive control

**`k` is the rule already satisfied, and the project got there four days before the rule was named.**

[ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md) sets currency at **`k` cadences** of the
covering Declared `Scan` — *not* at a number of hours. So:

- `k` = **the fraction**, project-authored, dimensionless, `2`.
- `k × cadence` = **the product**, evaluated per `Service` against the tightest covering cadence.
- The cadence is read from `Scan`, which is **Declared** — the moving quantity is read at evaluation
  time from the record, exactly as ADR-0034 prescribes.

Had ADR-0007 shipped *"an observation is current for 48 hours"*, it would have been `N = 30`'s twin,
and every cadence change would have silently mispriced it. It shipped the fraction. Recorded here
because a sweep that reports only failures cannot tell whether its instrument works.

**And `k` itself is not a product.** Its derivation is entirely about our own operation: *"It starts
at 2 rather than 1 because ADR-0005 treats skipped ticks as normal operation and #22 refuses to
surface a single skipped tick; at `k=1` every skip would open a `Gap`."* No world quantity appears
in it. **No change.**

### 6.3 The coverage threshold — not in the population

`CONTEXT.md` names the coverage alert threshold as one of the three places an **operator's dial**
legally sits, *outside* every derivation, alongside notification routing and flap suppression. It is
therefore **not a project-authored constant** and is out of the sweep's population by definition —
[#60](https://github.com/winniel123/verge-asm/issues/60)'s line, not a new one. The ticket named it
as a candidate; the correct answer is that it is not in the class.

### 6.4 The availability window — not a product, and **not yet a number**

[ADR-0005](../adr/0005-scan-execution-model.md) fixes it as *"Derived from batch outcomes; window is
**fixed and release-coupled**"* and rejects an operator-configurable one on `k`'s ground. Like `k`
it is about our own operation, and no world quantity enters.

**One handoff.** No value has ever been chosen for it. Whoever chooses it should express it in
**batches, not hours**, for `k`'s reason — a window in hours is `k`'s product and mis-sizes itself
the moment a cadence changes, while a window in batches rides the cadence. This costs nothing to
say now and a `Break` to fix later. → [#12](https://github.com/winniel123/verge-asm/issues/12).

### 6.5 [#61](https://github.com/winniel123/verge-asm/issues/61)'s staleness bound — *"two days modally and two months on the opt-in full-range tier"*

**A product, but of two project-authored quantities, so the rule is inapplicable.** It is
`k × cadence` — `2 × daily`, `2 × weekly`, `2 × monthly`. Neither factor is a world quantity, so it
cannot go wrong while nothing in the repository changes: it moves only when we move `k` or the
tiering, and both are releases.

**But the numerals are illustrations, and #60's exact failure mode is available at the
documentation level.** ADR-0028 writes them correctly — *"Modal `k`=2 on a daily tier is two days; a
top-1000-only port is two weeks; a full-range-only port is two months"* — the formula first, the
evaluation after. The **map's Notes do not**: they carry *"its stale-fact window is bounded by
`certificate`'s currency, two days modally"* with no formula beside it. #61 has already added a
fourth `Scan`, so the tiering demonstrably moves; when it moves again that numeral goes wrong while
every document it came from still reads true.

Costs nothing and no `Break` — nothing reads it. **Handed to the map as a wording fix**, in §9.

### 6.6 #4 §6's 5 handshakes/s per host, and every other rate and concurrency figure

**Not a product, and — the stronger point — not a declared parameter at all.**

ADR-0021's decision table puts *"the job-spec parser, the NDJSON writer, the SSH push, **concurrency,
rate-limiting and adaptive back-off**"* **outside every leaf**. So 5 handshakes/s, 50 conn/s,
20 concurrent, 10 req/s, 5 concurrent, 3 concurrent and the 200 pkt/s global ceiling sit outside the
comparison path entirely. **They can never move a value.** At worst they move wall-clock and, at the
extreme, the completed scope of a `Batch` — which is coverage, recorded, and rendered.

And on limb 2 they are **anti-fragile**: the world quantity they are budgeted against is a host's
capacity to absorb connections and handshakes, which moves **upward** with hardware. A stale budget
therefore becomes *more* conservative, never less — and #4 §6.2 already banks the asymmetry: *"Our
budget is wall-clock hours, and we have plenty."*

**No change.** The map's retention entry correctly notes these figures compose (*"an endpoint costs
a chain handshake plus an HTTP handshake plus the `tls-acceptance` enumeration, all sharing #4 §6's
5 handshakes/s per host"*) — that is a **throughput** question for the retention and packaging
patches, not a staleness one. *(The retention half closed as
[#121](https://github.com/winniel123/verge-asm/issues/121) ·
[ADR-0041](../adr/0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md), and it
does not read these figures: retention is set by what may still read a corpus rather than by what
generates it, so throughput remains packaging's question alone.)*

### 6.7 The EDNS UDP payload size, **1232** — the sweep's near-miss, and the reason limb 2 exists

This is the closest thing to a false positive in the repository and it is recorded as one.
[`measurement-offers.md`](../spec/measurement-offers.md) §4 states the form **outright**:

> UDP payload size **1232** — *"The DNS Flag Day 2020 coordinated position, **derived from IPv6's
> required 1280-byte MTU**, and the shipped default of BIND, Unbound and Knot."*

Limb 1 passes on the note's own words: `1232 = 1280 − 48`, a rule applied to a quantity we did not
choose. **Limb 2 fails, and decisively.** IPv6's minimum link MTU is fixed at 1280 by RFC 8200 §5
(and by RFC 2460 before it, since 1998). It is a **protocol constant, not a measurement** — nobody
publishes a schedule for it, and changing it would break IPv6 rather than update a number.

And 1232 is independently attested **as a value**, not only as an arithmetic result: it is the DNS
Flag Day 2020 coordinated position and the shipped default of three implementations, which is
ADR-0032's documented-shipped-default class from the parties that own the question.

> **A constant can be a product in form and a constant in fact.** Limb 2 is not a formality, and a
> sweep run on limb 1 alone would have "found" this one and re-expressed a correct, attested,
> frozen number into a formula over a quantity that never moves. This is the over-fitting the
> ticket warned about, caught in the act.

### 6.8 The remaining declared parameters

Walked for completeness, since ADR-0021's parameter set is the authoritative population.

| Leaf | Parameter | Product? | Note |
| --- | --- | --- | --- |
| `connect-outcome` | connect timeout, retry count | **No** | §6.1 |
| `tls-handshake` | handshake timeout | **No** | §6.1's reasoning, unchanged; the deadline may not read the run |
| `tls-handshake` | the TLS library; the candidate set | **No** | An offer under ADR-0030, not an assertion; widening is an aperture change, already priced |
| `http-exchange` | request timeout (10 s) | **No** | §6.1 |
| `http-exchange` | **capped body read** | **Open — no value has been chosen** | See below |
| `resolution-walk` | query timeout, retries, TCP fallback policy | **No** | §6.1 |
| `resolution-walk` | EDNS payload size | **No** | §6.7 |
| `resolution-walk` | the qtype set (seven) | **No** | An enumeration chosen against what v1 rules read; ADR-0015 gives it no deadline |
| `resolution-walk` **and** `wildcard-discrimination` | the **query path** | **No** | Added by [#116](https://github.com/winniel123/verge-asm/issues/116) / [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md), and **arriving with a value**, so the open count below does not move. **One parameter held jointly by two leaves**, taking **one value per `Batch`** — the shape ADR-0021's table already has for the EDNS option set and the DNS library — valued at **the `Vantage`'s configured recursive resolver**. No world quantity enters: it is a choice of instrument path, like the transport and fallback policy in §6.1's row above. Two things the cell must carry rather than imply. The jointness is **load-bearing** — **[measured]** a control probe direct to `s3.amazonaws.com`'s own authority reads `NoSynthesis` at A, a *determinate* reading, while a resolver answers every candidate beneath with eight addresses, so two settings that differ discriminate **every** fictional label and record it `Resolved` ([`passive-discovery-sources.md`](./passive-discovery-sources.md) §14.3). And **which resolver stands at the path is not this parameter** — it is part of the `Vantage`, because a declared parameter is never operator-configurable and §3.6 offers the operator that choice |
| `wildcard-discrimination` | control-label count and construction | **No** | A count of our own probes; no world quantity enters. ~~RFC 4592 fixes the mechanism~~ — **that clause is withdrawn**: RFC 4592 fixes the mechanism only for authorities that *implement* it, and **[measured]** three do not (`nip.io`, `sslip.io`, `traefik.me` compute the answer from the query name). **Valued** by [#113](https://github.com/winniel123/verge-asm/issues/113) / [ADR-0069](../adr/0069-a-control-label-is-one-label-and-the-set-must-falsify-label-independence.md): ~~**5 random labels + 1 structured label**~~ **9 random labels + 1 structured label** ([#115](https://github.com/winniel123/verge-asm/issues/115) raises the count; the construction is unchanged), each **exactly one label**, the structured one being `<a>-<b>-<c>-<d>` over a random RFC 5737 address. Still not a product — no world quantity enters — but the *count* had to stop being the range `3–5`, because [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s bidirectional gate diffs parameter **values**. **[measured]** the count now has a warrant of its own rather than *the top of the range*: the instability it samples is **per-label sharding** on two zones and **per-query rotation** on two more, and on a two-member pool the false-`Determinate` rate falls from 6.7% at six draws to 0 of 30 at eight ([`passive-discovery-sources.md`](./passive-discovery-sources.md) §13) |
| `wildcard-discrimination` | the match predicate | ~~**Open — no value has been chosen**~~ **No — closed, and not a product** | ~~ADR-0021 names it; nothing anywhere gives it one.~~ **Valued** by [#111](https://github.com/winniel123/verge-asm/issues/111) / [ADR-0068](../adr/0068-a-wildcard-is-discriminated-only-where-its-synthesis-is-determinate.md): **set equality on the RDATA set, per `(qtype, RR type)` component, at determinate components only**. No world quantity enters — it is a comparison rule over our own probe's answers, like the control-label count above. **[measured]** a synthesised answer set is not stable across control labels on two of four live wildcarded zones — **nor across repeated queries for one label at one authority**, which is why the value is a determinacy gate rather than a comparison |

**One thing ADR-0021's table does not contain, and this section's *authoritative population* claim
must not be read as covering it:** the **`Name`s the control labels are generated under**. That is
an **aperture input** recorded on the `Batch`, not a declared parameter of any leaf, because it is a
function of the batch's own scope where a parameter is authored data —
[ADR-0066](../adr/0066-a-control-probe-is-generated-under-a-names-parent-and-that-population-is-aperture.md).
It is therefore correctly absent from this walk rather than missing from it.

**The open-parameter count is back to two, and neither
[#113](https://github.com/winniel123/verge-asm/issues/113) nor
[#116](https://github.com/winniel123/verge-asm/issues/116) moves it** — #116 adds the **query path**
row above, which arrives valued, so the two remain the availability window (§6.4) and the capped body
read below. **The original sentence, unamended:**
[#108](https://github.com/winniel123/verge-asm/issues/108)
added the match-predicate row to this table without touching §9's and §10's *two declared parameters
have no value yet (§6.4, §6.8)*, which was briefly three;
[#111](https://github.com/winniel123/verge-asm/issues/111) closed the row, so both prose sites are
correct again and the two are the **availability window** (§6.4) and the **capped body read** below.

> **A caution for whoever next audits this table.** #113 gave *control-label count and construction*
> its value, and that row was **never in the open count** — this table filed it **No** on the ground
> *RFC 4592 fixes the mechanism*, which #113 **withdrew** (**[measured]** three authorities compute
> the answer from the query name and implement no RFC 4592 wildcard at all). So a parameter carrying
> a **range** rather than a value, and a construction nobody had specified, both sat inside a *No*
> cell for two rulings. The *Product?* column answers *is a world quantity in here* — it is **not**
> a has-a-value column, and reading it as one is what hid this. `wildcard-discrimination` now has no
> parameter without a value; the two open ones are elsewhere and unchanged.

**The capped body read is the one live item, and it is live because it has no value yet.**
ADR-0021 names the parameter; no ADR fixes a number, and #4 §4's *"64 KB"* was never carried
forward. Whoever sets it should know the shape matters: a **byte count** is a budget against where
`<title>` sits in a real document, which is a world quantity that moves upward as heads grow, and
whose failure is silent in the model's own currency — a page whose head outgrows the cap loses its
title, and `http-identity` records the loss as a `Transition` indistinguishable from a real one.
A **terminator** (read to `</title>` or `</head>`, whichever first, under a hard ceiling) is
deterministic on the bytes, so the corpus gate is unaffected, and it does not go stale.

This is **not** a fraction, and that is the point: it is a third cure shape, alongside the fraction
and the rule-expressed set of §3.3. → [#12](https://github.com/winniel123/verge-asm/issues/12).

---

## 7. The finding that is not an instance — `certificate`'s currency bound is now unsafe

**This is the sweep's most damaging result and it is not a product.** It is recorded in full because
ADR-0034 §4's argument — *the world parameter moves on a schedule its owner published, and our
number is wrong while every document it cites still says what it said* — applies here to an
**argument** rather than to a constant, and no re-expression of any constant repairs it.

### 7.1 What ADR-0028 ruled, and what it priced it against

[ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md) (**2026-08-13**) states
the hazard itself, precisely:

> The **clock class** — `certificate-expired`, `certificate-not-yet-valid` and `certificate-expiring`
> — is the only place in v1 where a rule reads an **always-current wall clock** against a **possibly
> stale observed value**. Everywhere else both sides age together, so staleness cannot make a
> comparison lie. Here it can, and the lie is a rule firing on a certificate that has already been
> replaced.

and then bounds it with `k`:

> **The guard is structural and already present**: past `k` cadences the value stops being current,
> so the rules go `not-evaluable` rather than firing on a stale `not_after`.

and prices the residue as acceptable:

> A `Service` only that tier reaches holds a `certificate` value current for **two months** … for up
> to two months they can speak about a certificate no longer served. **That is honest rather than
> fixable.**

### 7.2 The retrieved fact that re-prices it, one day later

[`acme-renewal-timing.md`](./acme-renewal-timing.md) §7.3 (**2026-08-14**) retrieves, from the
issuer:

> **"Short-lived and IP address certificates are now generally available from Let's Encrypt. These
> certificates are valid for 160 hours, just over six days."** — generally available since
> **2026-01-15**

and §7.2, the `tlsserver` profile at **45 days since 2026-05-13**, with the `classic` default going
64 days on **2027-02-10** and 45 days on **2028-02-16**.

**ADR-0028's guard is structural only where the window is short against the certificate's lifetime,
and it was checked against 90 days.** Against the lifetimes now shipping:

| Tier | Currency bound (`k`=2) | vs a 160-hour certificate | vs a 45-day certificate |
| --- | --- | --- | --- |
| daily (`verge-core`) | **2 days** | 0.32 of its life | 0.04 of its life |
| weekly (top-1000 only) | **14 days** | **2.3 × its entire life** | **0.31 — one whole `certificate-expiring` threshold width** |
| monthly (full-range only) | **60 days** | **≈ 10 generations** | 1.3 generations |

### 7.3 The consequence is a false firing, not silence

The guard delivers silence only *past* the bound. **Inside** it the observation is current and the
rules fire on it. So on a top-1000-only `Service` presenting a six-day certificate, the observed
`not_after` passes on day six and `certificate-expired` **fires true — on a current observation,
about an endpoint serving a valid certificate — and keeps firing for the remaining eight days of the
window.** It does not go quiet. It goes **loudly wrong**.

Three things make this worse than it looks, and each is already on the record:

- It lands on the **census**, which [#53](https://github.com/winniel123/verge-asm/issues/53) made the
  thing the operator reads — the identical argument #67 used to kill fixed `N`.
- It is in the **clock class**, which [#60](https://github.com/winniel123/verge-asm/issues/60) ruled
  is `certificate-expiring`'s **only carrier** and may never be folded into the drift class.
- The population is not exotic. The **weekly** tier, not the opt-in monthly one, is where this
  bites, and #67 records the 45-day `tlsserver` profile as **already the default for that profile**.

### 7.4 Why it is not an instance of #71's rule, stated plainly

Neither `k` (§6.2) nor the cadences is a product of a world quantity. What was computed against a
world quantity is the **safety argument** that `k × cadence` is short enough — and ADR-0028 performed
that comparison at a certificate lifetime with a **published expiry date**, one day before the
retrieval that falsified its premise.

> **The staleness rule catches stale numbers. It does not catch stale arguments, and the argument is
> where the residual risk in this repository actually sits.**

Calling this an instance would be the over-fitting #71 forbade. Not recording it because it fails
the test would be worse. It is recorded, and it is ticketed.

### 7.5 The repair exists and passes limb 3 — but it is a rule change, so it is a ticket

The cure shape *is* available, because the moving quantity **is** on the subject: the clock rules
already read `not_before` and `not_after` after ADR-0034. The repair is one extra evaluability
predicate — **a `certificate` observation feeds the clock class only while its age is small against
the observed certificate's own validity period; otherwise the three rules are `not-evaluable`** —
reading a value those rules already read, with no new measurement, no facet change and no ADR-0011
cost, at a `Break` on three rules for one cadence, vacuous before first install.

It is **not ruled here**, because it changes three rules' evaluability and interacts with
[#72](https://github.com/winniel123/verge-asm/issues/72) (what the census says when every endpoint
carries its own horizon) and with #44's `not-evaluable` rendering, and because the fraction's size
and the tier interaction want arguing both ways. Opened as **[#77](https://github.com/winniel123/verge-asm/issues/77)**.

---

## 8. Rulings

### 8.1 The cure is *ship the rule*, of which *ship the fraction* is one case

ADR-0034 states the cure as *ship the fraction*. The sweep found three cure shapes and one shared
form:

| Cure | Where it applies | Instance |
| --- | --- | --- |
| **A fraction of a quantity on the subject** | `Q` is a magnitude the subject carries | `certificate-expiring`'s `N` (#67) |
| **A rule-expressed membership criterion** | `Q` is a population and membership follows from something project-internal | `verge-core`'s sensitive half (ADR-0009's union); §2.3's supplement |
| **A terminator instead of a budget** | `Q` bounds *how much to read* and the stopping condition is in the bytes | the capped body read, §6.8 |

> **Ship the rule that generated the number, wherever the rule's inputs are available at evaluation
> time. A fraction is the case where the input is a magnitude carried by the subject.**

### 8.2 Amendment to ADR-0034 — *nothing to watch* is scoped to the mover it was cured against

ADR-0034 §4 and #67's resolution both say a row expressing the fraction *"has nothing to watch."*
That is true of the mover it was measured against and is over-broad as written.

> **Amendment.** Expressing a constant as a fraction removes the **quantity** from the watch, not the
> **attestation**. `⅓`, `½` and the 10-day threshold remain the issuer's published values and the
> issuer may revise any of them. That is an ordinary ADR-0032 §8 attestation move — loud, requiring
> a document to change — and the row is watched for it exactly as any attested row is.

The distinction is load-bearing and #67 already measured a live instance of the hazard: the CA/B
Forum moved its short-lived definition from ≤10 days to **≤7 days on 2026-03-15** while Let's
Encrypt, `boulder`, Certbot and lego all still use **10 days** for renewal halving. Our 10 is the
issuer's, correctly; but *the issuer's threshold is a value the issuer can move*, and #67's own
retrieval hazards section records that a session conflating the two would "correct" 10 to 7 and be
wrong. **The port-curation patch's discharge of `certificate-expiring`'s horizon stands** — nobody
revises a fraction when a lifetime changes — but the row is not attestation-immortal.

### 8.3 The class needs no new home — §8's watch is widened by shape, not by count

The ticket asks whether the class needs a home, since #67 established `N` belonged to neither of
ADR-0032 §8's two piles, and #68 has since added a third (**scope**).

**It does not, and adding a fourth pile would be the machinery the fix removes.**

ADR-0032 §8 defines its watch list by its **cause** — a maintainer flips a default — rather than by
its **shape**. Read by shape, the watch list is *rows where something must be noticed for the row to
stay right*, and that already covers a constant computed from a quantity the rule cannot read. So:

> **A weak row is *watched* wherever something must be noticed for it to stay right, whether what
> moves is an attestation or a quantity the row was computed from.** The pile names — **watched**,
> **chased**, **scope** — are causes of weakness, not kinds of watch, and a product that fails the
> reach test is **watched**. No fourth pile.

And the state ADR-0034 named as *"a third state … the one to aim for"* is now placed exactly:

> **It is not a pile. It is the absence of one.** `N` left both piles because its cure **removed the
> watch**, not because it was a new kind of weakness. A constant that passes limb 3 leaves the
> accounting entirely; a constant that fails it stays on the watch it was always on.

The map's port-curation patch already carries the routing and needs only the criterion. §9 has the
text.

> **CLOSED by [#125](https://github.com/winniel123/verge-asm/issues/125).** The criterion is the
> **revision act**, and this section's *watched by shape* survives it intact — what #125 adds is that
> the shape has an **order**, because the reading budget is finite. *No fourth pile* is respected and
> re-derived: the piles are causes, and the queue keys on the act rather than the cause.
> [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md).

### 8.4 The rule's reach, bounded

The sweep is the measurement, so the bound is stated as one:

**Eighteen project-authored constants and constant-sets were walked. One is a product that passes
all four limbs, and it was fixed before the sweep began. Two are products that fail the reach test,
and both were already correctly disposed of by tickets that did not know they were doing it.
Fifteen are not products.** One near-miss (§6.7) passes limb 1 and fails limb 2, and would have been
manufactured into an instance by a sweep run on form alone.

> **The rule's reach in this repository is one instance. That is the finding.** A rule that turned
> out to apply everywhere would have been evidence the test was wrong, not that the repository was
> rotten — and the one place the world's movement has actually done damage (§7) is a place the rule
> does not reach at all.

---

## 9. Handed on

- **To the map's port-curation patch** — the watch criterion widens per §8.3, and `verge-core`'s
  frequency half is a member of it under the widened criterion rather than a new object. Exact text
  in the resolution comment.

  > **DISCHARGED by [#125](https://github.com/winniel123/verge-asm/issues/125)**, which closed that
  > patch. §8.3's *watched by shape* is **kept and completed**: the three cause-piles stay causes, no
  > fourth pile is added, and all of them land on **one queue** keyed on the **revision act** — the
  > smallest act by the owner that would falsify the cell, and whether that act publishes a notice we
  > read. `verge-core`'s frequency half is **queue item 2**, at **rung 1**, and it is a
  > *cure-availability* item rather than a de-attestation one: nothing can de-attest data already
  > stale by eighteen years, and its act is a third party publishing a replacement dataset, which is
  > never announced to us. The queue holds both kinds and orders them on the same key, which is
  > exactly what §8.3 said a widened criterion should do.
  > [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md),
  > [`sensitive-ports.md`](./sensitive-ports.md) §39.4.
- **To the map's Notes** — #61's *"two days modally"* is an evaluation of `k × cadence` and should
  carry the formula beside it (§6.5). Exact text in the resolution comment.
- **To [#12](https://github.com/winniel123/verge-asm/issues/12)** — two parameters have no value yet
  and the **shape** of each is decided before the value: the availability window in **batches, not
  hours** (§6.4), and the capped body read as a **terminator, not a byte count** (§6.8).
- **To [#77](https://github.com/winniel123/verge-asm/issues/77)** — §7 in full. It is not this
  ticket's rule and it is this ticket's most damaging finding.
- **To [`safe-active-probing.md`](./safe-active-probing.md) §2.2** — its ranks are artefacts and its
  conclusion is stronger than it knew (§5.1). Exact amendment text in the resolution comment; the
  file is not edited here.
- **To [#78](https://github.com/winniel123/verge-asm/issues/78)** — §5.4, and §5.1's finding that no
  redistributable replacement exists. It **blocks [#12](https://github.com/winniel123/verge-asm/issues/12)**.
- **To whoever next cites an upstream default as a footing** — §5.2's naabu result. A cited default
  can be *unmoved* and the citation still wrong, and **the branch you fetch decides which bytes you
  get**: ProjectDiscovery's default branch is `dev`, and both cited URLs point at `main`, which is
  three months stale and serves different bytes. That is #67's `lego` hazard on a second citation.
- **Not handed to `sensitive-ports.md` or `weak-key-and-signature.md`** — both are owned by
  concurrent tickets this round ([#70](https://github.com/winniel123/verge-asm/issues/70),
  [#73](https://github.com/winniel123/verge-asm/issues/73)). §4 promotes
  `weak-key-and-signature.md` §7.3 to a general rule and **takes nothing away from it**; no
  amendment to either file is required.

---

## 10. Decided on thin ground

- **§8.1's three cure shapes are a generalisation from three instances**, one of which (the
  terminator) has no value chosen yet and so has never been tested against anything. If a fourth
  shape turns up that is none of these, the ruling is the four-limb test, not the taxonomy.
- **§7's arithmetic is certain; its population is not measured.** How many `Service`s sit on the
  weekly-only tier presenting short-lived certificates is unknown and unmeasurable before an
  install. The defect is real at any population above zero and its *severity* is unpriced, which is
  why #77 is a decision ticket rather than a correction.
- **§6.8's capped-body-read hazard is reasoned, not measured.** Nobody has measured how often a
  `<title>` sits past 64 KB, and the number is not chosen, so there is nothing yet to measure
  against. It is flagged as a shape decision, not asserted as a defect.
- **§5.4's licence reading is second-hand and is labelled so.** The NPSL was not retrieved and
  quoted against bytes in this sweep. It is opened as a question with no verdict implied, and the
  only thing this note asserts about it is that it is worth asking.
- **The sweep's population is *the constants this repository has written down*.** Two declared
  parameters have no value yet (§6.4, §6.8) and the spec is unassembled, so a constant that has not
  been chosen cannot have been swept. The four-limb test is the durable output; the tally of
  eighteen is a snapshot and [#12](https://github.com/winniel123/verge-asm/issues/12) will add to it.
</content>
