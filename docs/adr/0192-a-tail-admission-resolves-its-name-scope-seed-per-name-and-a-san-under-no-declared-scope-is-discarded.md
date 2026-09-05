# ADR-0192: a `ct-tail` admission resolves its name-scope Seed per name, and a SAN under no declared scope is discarded rather than attributed to the polling job

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1308 ADR gaps: internal/scan (CT and zone Scans)](https://github.com/winniel123/verge-asm/issues/1308), gap 5
- **PR that deleted the comment:** [#1307](https://github.com/winniel123/verge-asm/pull/1307)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0027](./0027-a-source-may-admit-without-observing.md), which rules that a CT admission's `Citation` is the `Batch` that returned it and that *"the chain still terminates at the `Seed`"*. It makes the Seed on an admission load-bearing rather than decorative. It does not say how a Seed is found where the Batch queried none
- **Rests on:** [ADR-0047](./0047-an-address-scope-is-its-own-enumeration.md), which rules that the `Seed` decides what is inside a scope, and that a name scope does not enumerate. It is the ground for the discard. It rules the declaration, not the per-name lookup
- **Bounded by:** [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md), whose Decision table fixes the **bulk** `ct` Scan at *"one `Batch` per name-scope `Seed`"* and rules names outside the queried scope **filtered**. It rules the path this ADR contrasts against. It predates `ct-tail` and does not reach it
- **Adds a site to, and does not close:** [ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md) §5, which names the declaration side's disagreement over which covering `Seed` wins and explicitly excludes it from its own ruling

## Context

`internal/scan/cttail.go:575` carried this, until #1307 shortened it:

```go
// CTAdmission is one Name the tail admits, with the name-scope Seed its Citation chain
// terminates at (ADR-0027). It is the tail's analogue of a crt.sh admitted name, but the
// Seed is resolved per-name (the tail reads the whole log firehose and keeps only the
// names under some declared scope), where crt.sh queries one Seed at a time.
```

One line survives, at `internal/scan/cttail.go:423`:

```go
// The tail reads the whole firehose, so each name's Seed is resolved per name, not per query.
```

The ADR-0027 citation the comment carried is real and on topic, and ADR-0027 does not state this
rule. It fixes where the chain terminates. It does not say how the terminus is found on a Scan that
queried no Seed.

### The two admitters, side by side

The repository holds two admission filters over CT names. They share four steps and differ in the
fifth.

| Step | `CTAdmitter` — bulk (`internal/scan/ctsource.go:18`) | `AdmitCTNames` — tail (`internal/scan/cttail.go:430`) |
| --- | --- | --- |
| Cap | `MaxAdmittedNames`, 100,000 | `MaxAdmittedNames`, 100,000 |
| Normalise | `normaliseName` | `normaliseName` |
| Wildcard | any `*` admits nothing (ADR-0060) | any `*` admits nothing (ADR-0060) |
| Dedupe | `seen` map | `seen` map |
| **Scope** | **`withinScope(n, a.domain)` — one domain, fixed at construction** | **`coveringSeed(n, seeds)` — every declared name scope, per name** |
| Returns | `[]string`. **No Seed** | `[]CTAdmission{Name, SeedID}`. **A Seed per name** |

### The tail's job scope names a log, and carries no Seed

This is the fact that forces the rule, and it is visible in the two job types.

`CTJob` (`internal/scan/crtsh.go:32`) carries `ScanID`, `SeedID` and `Domain`. Its wire scope,
`ctScope`, is `{"domain", "seed_id"}`. `internal/queue/crtsh.go:280` therefore writes
`SeedID: cs.SeedID` on **every** admitted row of that Batch, because the Batch asked about exactly
one Seed.

`CTTailJob` (`internal/scan/cttail.go:124`) carries `ScanID` and a `CTLog`. Its wire scope,
`ctTailScope`, is `{"log_id", "url", "description", "tiled"}`. **There is no Seed in it, and there is
no Seed to put in it.** §4.2 of the spec fixes the fan-out as *"per-log, not per-Seed"*, and
`internal/queue/queue.go:150` dispatches one job per followed log.

So the tail's runner reads the seed corpus at run time instead: `ctSeeds(ctx, w.q)` at
`internal/queue/cttail.go:102` and `:176` call `ListNameSeeds`, the whole declared name-scope set,
and hands it to `AdmitCTNames`.

### The read is the whole world, and that is why

`maxEntriesPerPoll` is 16,384 and `ctTailBatch` is 256 (`internal/queue/cttail.go:25` and `:21`).
The `ct-tail` Scan's shipped cadence is 300 seconds
(`db/migrations/23900_ct_log_cursor.sql:24`), across 39 followed logs on the shipped snapshot. §4.4
measures the firehose at about **9,500 entries per minute** on one usable current shard, and states
the shape plainly: *"the tail downloads every new entry to discard the non-matching majority."*

The bulk `ct` Scan asks `crt.sh` for `%.example.com` and gets back names under one domain. **The tail
asks the world and filters.** Nothing in what it fetches is scoped to the estate, so nothing in what
it fetches can tell it which Seed a name belongs to.

## Decision

> **The `ct-tail` Scan fans out per log, so its job scope names a log and carries no `Seed`. Each
> `Name` the tail admits therefore resolves its own name-scope `Seed`, per name, against every
> declared name scope, and takes the most specific covering one. A SAN under no declared scope is
> discarded — never admitted under the polling job's identity, under the log it was read from, or
> under a default `Seed`. The bulk `ct` Scan is unchanged: it queries one `Seed` at a time, so its
> admissions inherit the `Seed` that Batch asked about.**

### 1. There is no queried Seed to inherit, so the Seed is found per name

The bulk path's attribution is free, because the question and the answer share a scope: the Batch
asked about one Seed, so every name it returned is that Seed's. `internal/queue/crtsh.go:280` writes
`cs.SeedID` on every row and cannot be wrong.

The tail has no such fact. Its Batch asked a log for entries `N` to `N+k`, and the answer is every
certificate anyone logged in that window. **Per-name resolution is not a refinement of the bulk
rule. It is the only rule available**, because the alternative facts on hand — the job, the log, the
dispatch — are not Seeds and do not stand in for one.

### 2. Resolution runs against every declared name scope, read at job time

`AdmitCTNames` takes `[]CTSeed`, the full `ListNameSeeds` corpus, and `coveringSeed` walks all of it
per name. Not the scopes on some subset, and not a cached set from fan-out time.

Reading the corpus in the **job** rather than in the fan-out is what lets a Seed declared between the
fan-out and the poll be honoured on that poll. The fan-out has nothing to gain from the seed set,
because it fans out per log.

### 3. The most specific covering Seed wins, and this ADR adds a site to a live disagreement

`coveringSeed` (`internal/scan/cttail.go:459`) keeps the longest matching domain:

```go
if len(d) > bestLen {
	bestLen = len(d)
	bestID = s.SeedID
}
```

So `a.b.example.com` under both `example.com` and `b.example.com` is attributed to
`b.example.com`. That is ADR-0047's rule read forward: the `Citation` names the scope that actually
accounts for the name, and the broader scope accounts for it only incidentally.

**It agrees with the three reads [ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md)
names and disagrees with the fourth.** `FindCoveringAddressSeed` orders by `masklen(…) DESC`,
`FindCoveringNameSeed` by `length(s.name_domain) DESC`, and `narrowingScope` keeps the largest
`Bits()`. `coveringSeedKey` (`internal/queue/produce.go:370`) returns the **newest** covering Seed
instead, and ADR-0153 §5 names that disagreement and excludes it from its own ruling.

**This ADR rules the tail's own choice and does not close that disagreement.** It adds a fifth site
to the count, on the most-specific side. Closing the corpus-wide question needs a document whose
subject is *which covering Seed wins, everywhere*, and this is not that document.

### 4. A SAN under no declared scope is discarded, and that is the only available answer

`internal/scan/cttail.go:450` already cites the ground:

```go
seedID, ok := coveringSeed(n, seeds)
if !ok {
	continue // ADR-0047: the Seed decides what is inside, so a foreign SAN admits nothing
}
```

**The discard is not a filter for tidiness. It is forced by ADR-0027.** An `admitted_name` row's
`Citation` chain terminates at the Seed. A name under no declared scope has no terminus, so there is
no chain to write. Admitting it would mean inventing one.

**Three inventions are refused by name**, because each is a shape a later session might reach for:

- **The polling job's own identity.** A `ct-tail` job is scoped to a log. Terminating a name's chain
  there would say *this name is in the estate because we happened to be reading Argon2026h2*, which
  is not a scope the operator declared and not a fact about the name.
- **The log.** A log is not a `Seed` and has no `seed_id`. It also cannot be narrowed, withdrawn or
  reasoned about by ADR-0047's machinery.
- **A default or catch-all `Seed`.** It would make every certificate on earth a member of the estate,
  which is the enumeration ADR-0047 §10 refuses in its own domain.

**The discard costs nothing recoverable.** ADR-0027 rules CT `corroborative`, so a name the tail
drops is a name it did not admit and never a claim that the name does not exist. Declaring the scope
later admits it on the next poll that sees a certificate for it.

### 5. The bulk path is unchanged, and the contrast is structural rather than historical

Nothing here reaches `CTAdmitter`, `AdmittedNames`, `crtshCTSource` or `certSpotterSource`. They
query one Seed and attribute to it, exactly as ADR-0106 rules.

**The two rules are not two answers to one question.** They are the same answer — *a `Name` is
attributed to the declared scope that accounts for it* — under two query shapes. Bulk knows the scope
before it asks. The tail can only know it after.

### 6. What this rule does not reach

- **Which names are admitted.** The cap, the normalisation, the wildcard refusal (ADR-0060) and the
  dedupe are shared by both admitters and are ruled elsewhere.
- **Membership.** ADR-0106 already rules that an `admitted_name` row records *how a name entered* and
  that membership is measured, not admitted. The Seed on the row is the `Citation`'s terminus, not a
  claim of membership.
- **The drift signal.** `internal/queue/cttail.go:191` compares admissions against a known-name set
  and emits an ephemeral event. §4.1 rules that, and it is downstream of this one.
- **Duplicate admissions across logs.** The `seen` map is per job, so the same name in two logs is
  admitted by two jobs. `InsertAdmittedName` handles that and this ADR does not reach it.
- **The corpus-wide covering-Seed disagreement**, per §3.

## Consequences

- **This ADR changes no Go code.** `AdmitCTNames` and `coveringSeed` are correct as they stand.
- **`internal/scan/cttail.go:423` gains this ADR's citation** on the surviving line that states the
  rule. Recorded in this issue's manifest.
- **The per-name resolution is `O(names × seeds)` per job and nothing bounds the seed corpus.**
  `coveringSeed` walks every declared name scope for every candidate SAN, up to 16,384 entries per
  poll per log. It is fine at the estate sizes this product targets and it is not free. **This ADR
  does not rule it and no ticket is owed today**; it is named so a later session reading a slow
  `ct-tail` job knows where to look.
- **[ADR-0153](./0153-a-narrowing-mover-carries-no-precedence-so-the-first-covering-row-is-the-whole-attribution-rule.md)
  §5's excluded disagreement gains a fifth site.** That section gains a cross-reference so a reader
  enumerating the covering-Seed reads finds this one. Recorded in this issue's manifest.
- **A name the tail discards leaves no trace.** There is no counter, no log line and no event for
  *SANs seen and dropped*. That is correct under ADR-0027 — a discarded foreign name is not a fact
  about the estate — and it means the tail's job event reports only what it admitted. An operator
  cannot tell a poll that saw nothing relevant from a poll that saw a million irrelevant names.
- **`CONTEXT.md` gains nothing.** `Seed`, `Name`, `Citation` and `Source` are already defined there
  and this ADR uses them as defined. It adds no term.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Fan the tail out per Seed, so it inherits the bulk rule** | The logs carry no name index. §1 of the spec states it as a protocol fact: RFC 6962 and static-ct-api have **no query-by-domain**, and *"the drift tail and verification can **never** do bulk-by-name."* A per-Seed fan-out would read the same firehose once per Seed and multiply the work by the estate's size |
| **Attribute every admission to the polling job, the log, or a catch-all Seed** | None of the three is a `Seed`, so none is a terminus ADR-0027's `Citation` chain can end at. A catch-all makes every certificate on earth a member, which is the enumeration ADR-0047 refuses |
| **Admit the out-of-scope name with a null `seed_id` and resolve it later** | `admitted_name.seed_id` is the chain's terminus, and a null one is a `Citation` that ends nowhere. It also creates durable rows for names the operator never declared an interest in — a firehose-sized write amplification, on a source ADR-0027 already rules `corroborative` |
| **Take the first covering Seed rather than the most specific one** | `ListNameSeeds`' order would then decide the `Citation`, so the same certificate would attribute differently depending on declaration order. `FindCoveringNameSeed` already answers this question with `length(s.name_domain) DESC`, and disagreeing with it here would add a sixth reading rather than a fifth |
| **Take every covering Seed, and write one row per Seed** | It multiplies rows on the broadest scopes and makes the `Citation` chain branch. ADR-0027 says the chain terminates *at the Seed*, singular, and a branching terminus is a different model |
| **Read the seed corpus once at fan-out and put it in the job scope** | The scope would carry the whole estate's declared domains into every one of 39 job rows, per tick, and a Seed declared between fan-out and poll would be ignored for that poll. Reading it in the job costs one query and has neither problem |
| **Share one admitter between the two paths, parameterised by a scope function** | The two differ in their return type, not only their predicate: bulk returns names and the tail returns name-plus-Seed pairs. Unifying them means the bulk path carries a Seed field it already knows and the tail path carries a domain field it does not have. The four shared steps are 20 lines |
| **State the rule on [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)** | ADR-0106 rules the bulk `ct` Scan over crt.sh. §4.2 of the spec states the tail *"shares nothing with bulk `ct` except the CT theme"*, and ADR-0106's own *one `Batch` per name-scope `Seed`* row is the shape the tail does not have |
| **State it on [ADR-0027](./0027-a-source-may-admit-without-observing.md)** | ADR-0027 rules what a `Citation` for a CT-admitted `Name` is. That ruling is correct and unchanged for both paths. How the terminus is *found* on a Scan that queried no Seed is a mechanism, and filing it there would put a lookup rule inside a model ruling |
