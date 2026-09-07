# ADR-0222: a bar is authored in the release and a health record is per-install, so the two never share a badge

- **Status:** Accepted
- **Date:** 2026-09-07
- **Tickets:** [#1519 Two shipped proposers point at api.caida.org, which is NXDOMAIN](https://github.com/winniel123/verge-asm/issues/1519), [#1583 A proposer source with a dead endpoint has no health surface](https://github.com/winniel123/verge-asm/issues/1583), [#1604 A source that ships off for a broken endpoint cannot say so](https://github.com/winniel123/verge-asm/issues/1604)
- **Rests on:** [ADR-0003](./0003-third-party-source-consent-bar.md), which rules the consent bar and owns the phrase *excluded on terms*. This ADR takes no ground from it and narrows the badge that was speaking for it
- **Rests on:** [ADR-0012](./0012-a-proposer-is-not-a-source.md), which rules that a proposer carries `consent` alone. A proposer's reachability is not `authority`, not `completeness`, and this ADR does not reintroduce either
- **Rests on:** [ADR-0023](./0023-consent-names-the-door.md), whose structure this ADR reuses: a project reading, taken once and written into the release, is the same for every install
- **Continues:** [#1553](https://github.com/winniel123/verge-asm/pull/1553), which ruled *a source-health surface, not a build-time check and not nowhere*. That ruling is not relitigated here. This ADR settles the four points it left open

## Context

Three tickets ask one question in three places.

[#1519](https://github.com/winniel123/verge-asm/issues/1519) found that two shipped proposers
point at `https://api.caida.org/as2org/v1`. That host is NXDOMAIN. Both shipped `DefaultOn: true`
from first commit until an unrelated audit tripped over them.
[#1583](https://github.com/winniel123/verge-asm/issues/1583) asked why nothing showed an operator
that a source was failing. [#1604](https://github.com/winniel123/verge-asm/issues/1604) asked why
the obvious guard was refused: `settings.tmpl` renders every barred entry under a hardcoded
`barred — excluded on terms` badge, and the two CAIDA sources clear the consent bar.

The three share one defect. The interface has one word for several different facts, which is the
`Host` defect ADR-0012 refused. Below, the facts are separated first, and the surfaces follow.

### The endpoint was measured, and the result is worse than a dead host

Taken 2026-09-07 from the dev machine, against every candidate
[#1519](https://github.com/winniel123/verge-asm/issues/1519) named:

| Host and path | Result |
| --- | --- |
| `api.caida.org` (any path) | NXDOMAIN |
| `api.data.caida.org/as2org/v1/org2ids?org=…` | `500` |
| `api.data.caida.org/as2org/v1/` | `200`, redirects to a Swagger document |
| `data.caida.org/as2org/v1/…` | `401` |
| `publicdata.caida.org/as2org/…` | `404` |
| `caida.org/as2org/…` | `404` |

`api.data.caida.org` resolves and serves an AS2org API. It publishes its own Swagger document at
`/as2org/v1/doc`. That document enumerates every path the API serves:

```
/as2org/v1/asns/            /as2org/v1/asns/{asns}
/as2org/v1/datasets/        /as2org/v1/datasets/{date}
/as2org/v1/orgs/            /as2org/v1/orgs/{org}
/as2org/v1/search/
```

**No `org2ids` path is published on any candidate host.** The `500` is a routing failure. The path
this code calls does not exist, and the measurement gives no date at which it did.

**`opaque_ids` does not survive.** The live envelope is
`{"totalCount":…,"pageInfo":{…},"errors":null,"data":[…]}`, and the identifier on each record is
`opaqueId` — camelCase, singular, one per record. `caidaOrgIDs` decodes
`{"opaque_ids": []string}`.

The consequence is the reason this ADR refuses a constant swap outright. Decoding the live envelope
into `caidaOrgIDs` **succeeds**. `encoding/json` ignores unknown keys, so `OpaqueIDs` is `nil` and
no error is returned. `CAIDA.Propose` then reaches its own second branch:

```go
if len(ids) == 0 {
    return nil, nil // no holder matched — no proposal, not a proposal of absence
}
```

A swap of the constant alone therefore converts a **loud transport failure into a silent empty
result**. #1519 states, correctly, that today's failure is loud and is not a false absence. A naive
repair removes that property. This is [#50](https://github.com/winniel123/verge-asm/issues/50)'s
shape of hazard, reached by a one-line change that looks like a fix.

`data.caida.org` answers `401`. That host is not keyless. A proposer moved to it would leave
`unencumbered` for `operator-credentialed` under ADR-0003, which is a consent change and not a URL
change.

The Public-AUA was re-read on 2026-09-07 and is unchanged. It still grants *"a limited,
non-exclusive, non-transferable, non-assignable, and terminable license to copy, modify, and use"*
and still states *"there are no implied licenses"*. ADR-0003's CAIDA row therefore stands as
written. The AUA governs **redistribution**, which is why it bars bundling; a runtime fetch by the
operator's own install is use, not redistribution. So the AUA applies unchanged to any replacement
CAIDA endpoint and bars none of them. It is not the obstacle here. The absent path is.

### The badge already states something false, today

#1604 describes a falsehood the badge *would* state. It already states one.

`fillSourcesSection` routes two different populations into one `Barred` bucket:

```go
case v.NoRunner:
    barred = append(barred, row)
…
default:
    barred = append(barred, row)
```

The `default` arm holds `hackertarget`, which really is excluded on terms. The `NoRunner` arm holds
`ripestat`, `ripe-db`, `apnic-registry` and `lacnic-registry`. Every one of those is
`consentAccepted`, and every one carries a `ShipNote` that says *"no proposer runner ships for this
path yet (#241)"*. Their terms are unresolved. Their terms did not fail.

The badge and the note contradict each other on the same row, in the one area whose whole
discipline is stating consent truthfully. Four shipped sources are affected before CAIDA is
considered at all.

## Decision

### 1. There are three facts, on two layers, and the third is the one that was missing

#1604 asks whether *endpoint does not answer* is a Declared fact or a measured one. The question
has been asked of one fact that is really two, which is why it resisted an answer.

1. **Excluded on terms.** A consent judgement. **Declared.** ADR-0003 rules it. A session reads the
   terms once and writes the verdict into the release.
2. **The release cannot reach this path.** A capability finding — no runner ships (#241), or no
   published endpoint serves the path (#1519). **Declared.** A session measures it once and writes
   the finding into the release. It is the same for every install, it does not vary with the
   operator, and it changes only when a release changes it. This is exactly ADR-0023's structure,
   applied to reachability instead of to consent.
3. **This install's last attempt to this source failed.** **Operational.** That layer records what
   the system *did*, and an outbound request is something the system did. It varies per install, it
   changes without a release, and no session authors it.

Facts 1 and 2 are Declared and may be badged. Fact 3 is Operational and may not.

Fact 3 is **not Observed**. ADR-0012 is the ground: an Observation has a subject, a facet and a
vantage. A third party's endpoint is no subject of the estate, so its reachability observes nothing.
It is **not Derived**, because it concludes nothing about the estate. Naming it Operational is what
lets the rest of this ADR be consistent, and it is what #1604 was missing.

### 2. A bar is authored, and its reason is data

The `Barred` bucket carries the project's authored judgement. It gains a reason, and the badge
renders the reason instead of asserting one:

| Reason | Ground | Entries |
| --- | --- | --- |
| `excluded on terms` | ADR-0003 | `hackertarget` |
| `no runner ships` | [#241](https://github.com/winniel123/verge-asm/issues/241) | `ripestat`, `ripe-db`, `apnic-registry`, `lacnic-registry` |
| `endpoint does not answer` | #1519, measured above | `afrinic`, `apnic-caida` |

No new colour and no new component. Both reasons are quiet facts about why nothing runs. Neither is
a severity and neither is a drift, so the existing `st-badge neutral` treatment is correct for all
three, and only the words change.

**A barred source is forced off.** `sourceViews` forced `enabled = false` for `NoRunner` and not for
`Barred`. That was latent while the one barred entry had no runner. It stops being latent the moment
a proposer is barred, because a stale `source_state` override would then still run it. A bar
overrides an override.

### 3. The two CAIDA proposers are barred, not repaired

They are barred on reason 2. The finding is authored: no published CAIDA endpoint serves
`/as2org/v1/org2ids`, and the response shape the decoder expects is not the shape any candidate
returns. `DefaultOn: false` (#1553) was not enough, because an admin could still enable a path that
cannot work.

**The `caidaBase` constant is not changed.** #1519's instruction is followed literally: no candidate
serves the path, so the negative result is recorded and the constant stays. Changing it would trade
a loud failure for a silent one, per the Context.

### 4. The health surface: the four open points

#1553's ruling stands and is not reopened. These four are settled.

**Where the state lives — a new table, never `source_state`.** `source_state` holds the operator's
Declared enablement override, and its own migration says so at length. Health is Operational. Two
layers in one row is the defect this ADR exists to remove. In-memory is refused on the ticket's own
ground: it loses the signal on restart, which is the disappearing-signal defect one level up.

**What writes it — the query path, and nothing else.** A periodic probe is refused. Its cost is a
scheduled outbound request per source, and the stronger objection is a consent objection: a probe
sends requests to a third party for a source **the operator has not enabled**, which is an act
ADR-0003 governs and no operator has authorised. A source nobody enables therefore stays at *never
attempted*, and that is rendered as *never attempted* and never as healthy. This does not reopen
#1519. Those two sources shipped `DefaultOn: true` and were queried, so a query-time record would
have carried the failure from the first lookup. What was missing was persistence and placement, which
is what #1553 said.

**What it shows — the last outcome, its instant, and the consecutive-failure count.** A derived
`healthy / degraded / dead` state is refused. It needs a threshold nobody has picked, and picking one
here would invent a judgement rather than record one. The deeper reason is that a derived word states
a fact about the world, and the project measured only its own requests. *Last attempt failed, 37 in a
row* is what is known. The operator draws the conclusion.

**Whether it may change enablement — no, and this is an explicit refusal, not an omission.** An
Operational record may never move a Declared value. Enablement is the operator's consent state under
ADR-0003, and letting a per-install runtime record overwrite it inverts the layers. The second ground
is #50's: a third party's outage would silently narrow an operator's coverage, and a narrowing nobody
chose reads as absence. Auto-disable would have contained #1519 without a human. It is still refused.

Note the two answers do not conflict. Section 2 lets a Declared capability finding bar a source,
because a session authored it into a release and an operator can read why. Section 4 forbids an
Operational record from doing the same thing, because nobody authored it and nobody can read why.
The layer is what separates them.

## Consequences

- **`NoRunner` keeps its meaning and loses its monopoly.** It states that no runner ships. It no
  longer stands in for every reason a source does not run.
- **A new barred entry must carry a reason.** A test asserts that every barred and every `NoRunner`
  catalogue entry has one, so an entry cannot silently inherit somebody else's reason.
- **#1519's first box stays open.** The two proposers still reach no host that answers. They are now
  barred rather than quietly toggleable, and the finding is recorded, but the org→prefix coverage
  for AFRINIC and APNIC is not restored by this ADR.
- **The health surface is specified here and built separately, as
  [#1615](https://github.com/winniel123/verge-asm/issues/1615).** No smaller true increment of it
  exists: this ADR refuses in-memory state, so any surface needs the table, the query, the write path
  and the render together. Landing the ruling and the badge without it is a split along the seam this
  ADR draws, not a partial build.
- **The delegated-stats half is untouched.** `ftp.afrinic.net` and `ftp.apnic.net` both answer. Only
  the CAIDA join key is missing, which is why the ADR bars the joined proposer and claims nothing
  about the RIR files.

## Reopening condition

Reopen this ADR when any one of these holds.

1. **A CAIDA endpoint is found that serves an org-name-to-opaque-id lookup, and its response is
   decoded against a real body.** The bar in §3 is a finding about a measured API surface, and a new
   measurement retires it. `api.data.caida.org/as2org/v1/search/?name=` is a published, keyless
   org-name search and is the first candidate; it returns `orgName` and `opaqueId` and it is
   **not** a drop-in, because the path, the envelope and the field name all differ. Filed as
   [#1616](https://github.com/winniel123/verge-asm/issues/1616). Any such repair must also confirm
   the replacement host is keyless, since `data.caida.org` answers `401`.
2. **A source is enabled and never queried for long enough that *never attempted* misleads.** §4
   refuses the periodic probe on a consent ground, and that ground assumes an enabled source gets
   queried. If an operator enables a source and reads *never attempted* for weeks, the refusal has
   stopped paying and the probe is worth re-pricing.
3. **A threshold for a derived state is picked somewhere else in the product on a stated ground.**
   §4 refuses `healthy / degraded / dead` because nobody has picked a threshold. A threshold picked
   elsewhere, with a ground, would remove that objection.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Swap `caidaBase` to `api.data.caida.org` | The path is not published there. The `500` is a routing failure, and the live envelope decodes into `caidaOrgIDs` without error, so the swap turns a loud failure into a silent empty result — #50's hazard, reached by a line that looks like a fix |
| Swap to `data.caida.org` | Answers `401`. Not keyless, so it is an ADR-0003 consent change and not a URL change |
| Leave the two proposers toggleable with a truthful `ShipNote` | #1553 already did that, and it is what #1604 filed against. An admin can still enable a path that cannot answer, and the note is read after the toggle is offered |
| Give the two CAIDA proposers `NoRunner: true` | Their runner ships. The flag would be false, and the badge under it was the original complaint |
| One badge, reworded to cover every reason | The generic wording that covers consent, a missing runner and a dead endpoint states none of them. #1604 asked for a badge that says the second reason |
| Put health on `source_state` | Mixes an Operational record into the Declared override row, which is the layer confusion this ADR removes. That table also holds a row only where the operator has toggled, so an untouched source has nowhere to write |
| Hold health in memory | Loses the signal on restart. That is the same disappearing-signal defect, one level up, and the ticket names it |
| Probe every source on a schedule | Sends requests to a third party for sources the operator never enabled, which ADR-0003 governs and nobody authorised. The per-source request cost is the smaller objection |
| Derive `healthy / degraded / dead` | Needs a threshold nobody has picked, and states a conclusion about the world from a record of our own requests |
| Auto-disable a persistently dead source | **Refused explicitly.** An Operational record may not overwrite the Declared consent state ADR-0003 governs, and a third party's outage would silently narrow coverage nobody chose to narrow — #50's shape |
| A build-time or CI check that resolves the host | Already refused by #1553 and #1519. A build cannot see DNS, and the check converts a third party's bad minute into a block on our own merges |
