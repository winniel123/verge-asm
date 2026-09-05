# ADR-0189: a CT source's Go constant is the one slug literal, and the source catalogue names that constant rather than repeating the string

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1308 ADR gaps: internal/scan (CT and zone Scans)](https://github.com/winniel123/verge-asm/issues/1308), gap 1
- **PR that deleted the comment:** [#1307](https://github.com/winniel123/verge-asm/pull/1307)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Rests on:** [ADR-0003](./0003-third-party-source-consent-bar.md), which makes the source toggle an act of operator consent and puts the catalogue itself in the release. It rules what the toggle means. It does not rule how the catalogue's key reaches the code that reads it
- **Rests on:** [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md), whose Decision table states *"`fanOutCT` gates on the **`crtsh` source** being enabled"* and *"`source = crtsh`"* on the `admitted_name` row. It names the gate and the column value. It does not state that the two literals must be equal, or what happens when they are not
- **Withdrawal convention:** [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

## Context

`internal/scan/crtsh.go:31` carried this, until #1307 rewrote it:

```go
// CrtshSource is the source every CT admission is attributed to: crt.sh, the
// keyless CT front door (ADR-0003), `authority: inferred`, `completeness:
// corroborative` (CONTEXT.md `Source`). It matches the source-catalogue slug so
// the enablement state keys line up.
```

The sweep kept the last clause as a one-line reason at `internal/scan/crtsh.go:21`:

```go
// It must equal the source-catalogue slug, or the enablement state keys stop lining up.
```

**The tail's half of the same rule is not stated anywhere.** `internal/scan/cttail.go:23` reads
`const CTTailSource = "ct-tail"` and carries no comment at all. The issue body named `CTTailSource`
as a pre-sweep symbol. It survives, uncommented.

### The string exists three times in production, and Go holds only one of them

| Site | Form | Who reads it |
| --- | --- | --- |
| `internal/scan/crtsh.go:23` | `const CrtshSource = "crtsh"` | `internal/queue`, `cmd/web` |
| `cmd/web/sources.go:43` | `Slug: "crtsh"` — a bare literal in `sourceCatalog` | the toggle handler, the Sources page |
| `db/migrations/23600_ct_throttle_per_source.sql:17` | `VALUES ('crtsh', …)` | PostgreSQL, once, at install |

`ct-tail` repeats the shape at `internal/scan/cttail.go:23` and `cmd/web/sources.go:49`.
`certspotter` repeats it at `internal/scan/certspotter.go:16` and `cmd/web/sources.go:99`.

**`cmd/web` already imports `internal/scan` and already uses the constants.** `sources.go:209`,
`:211`, `:219` and `:314` all read `scan.CrtshSource`, `scan.CertSpotterSource` and
`scan.CTTailSource`. The bare literals at `:43`, `:49` and `:99` are a second spelling of a string
the same file resolves through the constant fifty lines further down.

### The writer and the reader are on opposite sides of that split

`cmd/web/sources.go:448` `toggleSource` takes `r.FormValue("slug")`, validates it with
`catalogBySlug`, and writes `UpsertSourceState{Slug: slug}`. **The row's key is therefore the
catalogue literal.** `cmd/web/sources.go:476` `settingsSources` does the same on the settings form.

`internal/queue/crtsh.go:360` `sourceEnabled` reads it back:

```go
for _, s := range states {
	if s.Slug == slug {
		return s.Enabled, nil
	}
}
return shipDefault, nil
```

and is called with the **Go constant**, at `internal/queue/crtsh.go:339`
(`d.selectedCTSource()`) and `internal/queue/cttail.go:285` (`scan.CTTailSource`).

### The failure is not one failure, and *"silently disables"* names only one third of it

Three consumers read the same constant, and drift breaks each of them differently.

| Consumer | What a drifted constant does | Loud or silent |
| --- | --- | --- |
| `sourceEnabled` (`internal/queue/crtsh.go:360`) | No row matches, so the function returns `shipDefault` and the operator's override is discarded | **Silent** |
| `ReserveCTSlot` (`db/queries/crtsh.sql:29`) | `UPDATE ct_throttle … WHERE source = $1` matches no row. The CTE is empty, the `:one` query returns no row, and `internal/queue/crtsh.go:149` wraps it as `ct throttle: no rows in result set` | **Loud** |
| `InsertAdmittedName`'s `source` column (`internal/queue/crtsh.go:279`, `internal/queue/cttail.go:215`) | Rows land under a source string the catalogue does not know, so the Sources page counts them nowhere | **Silent** |

**The silent half does not always disable.** `sourceEnabled` takes a `shipDefault` argument and the
two CT sources pass opposite values:

- `crtsh` passes `true` (`internal/queue/crtsh.go:339`). An operator who toggles crt.sh **off** gets
  a `source_state` row the dispatcher never finds, so the fallback returns `true` and **the Scan
  keeps querying crt.sh.** The failure is an ignored **off**, not an ignored on. Under ADR-0003 the
  toggle is consent, so this direction spends a consent the operator withdrew.
- `ct-tail` passes `false` (`internal/queue/cttail.go:285`). An operator who toggles the tail **on**
  gets nothing. That direction is the silent disable the deleted comment described.

So the deleted comment was right that the coupling exists and wrong about its cost. The cost is
whichever way the ship default points, and for `crtsh` that is the worse way.

### Nothing checks the equality, and nothing in `internal/scan` can

`internal/scan` cannot import `cmd/web`, which is a `main` package. No test anywhere compares a
constant against `sourceCatalog`. `cmd/web/sources_test.go:328` ranges over `sourceCatalog` and
asserts that each row's display `Name` renders in the Sources modal, which touches no slug at all.
The one direction that compiles is `cmd/web` reading `internal/scan`, and that direction is the fix
rather than the check.

## Decision

> **A CT source's slug has exactly one literal in Go, and it is the exported constant in
> `internal/scan`. Every other Go site names that constant. The source catalogue's `Slug` field
> names it too, rather than repeating the string, so the two cannot drift. A SQL literal in a landed
> migration is not a second Go site and is not a live coupling.**

### 1. The constant is the literal, and the catalogue names it

`scan.CrtshSource`, `scan.CTTailSource` and `scan.CertSpotterSource` are where the string is
spelled. `cmd/web`'s `sourceCatalog` rows read `Slug: scan.CrtshSource`, `Slug: scan.CTTailSource`
and `Slug: scan.CertSpotterSource`.

The constants sit in `internal/scan` and not in `cmd/web` because the readers are in
`internal/queue`, which cannot import a `main` package either. `internal/scan` is the one package
both consumers already depend on.

### 2. The invariant is a definition, not a test

**This is the whole reason the rule takes this form.** A test that asserts
`sourceCatalog[i].Slug == scan.CrtshSource` is a mechanism that can be absent, skipped or deleted.
Naming the constant makes the violating state **inexpressible**: there is no second string to
disagree with.

That is the move [ADR-0009](./0009-verge-core-is-a-union.md) made for `sensitive ⊆ verge-core`, and
[ADR-0144](./0144-the-verge-core-body-is-compiled-in-and-an-operator-edit-layers-over-it.md) §2 made
again for the `verge-core` body, on the same ground: *a definition cannot fail, because the violating
state is not expressible.*

**A test is refused here for a second, specific reason.** A test would have to enumerate which
catalogue rows are CT sources, which is a fourth place the same knowledge lives. The proposer rows,
the barred row and the `NoRunner` rows have no Go constant at all and must not gain one, so the test
needs an allow-list that drifts on its own.

### 3. A catalogue row with no runner keeps its bare literal

`arin`, `afrinic`, `apnic-caida`, `ripestat`, `ripe-db`, `apnic-registry`, `lacnic-registry` and
`hackertarget` have no Go constant and gain none. They are catalogue-only rows: a proposer, a barred
source, or a source whose runner does not ship (`NoRunner`, #241). Their slug is read by
`catalogBySlug` and by nothing else, so there is one literal already and this rule has no work to do.

**The rule reaches a slug that a second Go package keys on.** That is what makes the drift possible
and what makes the fix available.

### 4. The migration literal stays a literal, and it is not a live coupling

`db/migrations/23600_ct_throttle_per_source.sql:17` seeds `ct_throttle` with `'crtsh'`, and
`db/migrations/24100_certspotter_throttle.sql:11` seeds `'certspotter'`. SQL cannot read a Go
constant, so those two sites cannot be brought under §1.

They are not a live coupling and they do not need to be. A migration runs **once**, at install, and
is not re-read. What it leaves behind is a row, and from that instant the row's key is data rather
than source. A later change to the Go constant does not silently disagree with the migration — it
disagrees with the **row**, and §Context's table shows `ReserveCTSlot` raises that disagreement as
an error on the first poll.

**This is a real bound and not a dismissal.** Adding a new CT source that needs a throttle row still
means writing its slug twice, once in Go and once in the migration that seeds it. That second
writing is a one-time act under review, and it fails loudly rather than silently.

### 5. A Scan kind and a source slug are two constants, and `ct-tail` being both is a coincidence

`scan.CTTailKind` and `scan.CTTailSource` are both `"ct-tail"` (`internal/scan/cttail.go:21` and
`:23`). They are not the same fact. The kind is the `scan.kind` column, constrained by
`db/migrations/23900_ct_log_cursor.sql:15`. The slug is the `source_state.slug` and
`admitted_name.source` key.

`ct` and `crtsh` show that the two namespaces are genuinely separate: ADR-0106 named the Scan for the
exchange and left the instrument's name on the source. **A caller must not read
enablement with `CTTailKind`, or dispatch with `CTTailSource`.** Either compiles and either works
today, by coincidence, and stops working the moment one of the two moves.

### 6. What this rule does not reach

- **Whether a source ships on or off.** That is ADR-0003 and the catalogue's `DefaultOn`.
- **The `shipDefault` argument `sourceEnabled` takes.** Its two values are correct and are
  ADR-0003's ruling reaching code. This ADR rules the key, never the default.
- **The display name.** `catalogSource.Name` and `CTSource.DisplayName()` are renderings, and #780
  already rules that a rendering is never the key.
- **Non-CT catalogue rows**, per §3.

## Consequences

- **This ADR changes no Go code.** The catalogue's three bare literals at `cmd/web/sources.go:43`,
  `:49` and `:99` are a known violation of §1 and are **not** fixed here. That ships as its own
  ticket: replace the three literals with `scan.CrtshSource`, `scan.CTTailSource` and
  `scan.CertSpotterSource`. It is a three-line change with no behaviour change, because the strings
  are equal today.
- **`internal/scan/cttail.go:23` gains a comment.** `CTTailSource` carries none, and the rule it is
  bound by is not recoverable from `const CTTailSource = "ct-tail"`. Recorded in this issue's
  manifest.
- **`internal/scan/crtsh.go:21` gains this ADR's citation** on the surviving line that states the
  rule. Recorded in this issue's manifest.
- **The deleted comment's failure claim is corrected rather than carried forward.** It said a
  drifted constant *"silently disables the source instead of erroring"*. §Context measures three
  consumers: one is silent and ignores the operator's override in whichever direction the ship
  default points, one errors loudly, and one writes orphaned rows. Any document that repeats the
  single-failure reading is repeating a claim this ADR measured false.
- **An ignored **off** on `crtsh` is a consent defect, not only a correctness defect.** ADR-0003
  makes the toggle the operator's consent. Discarding it runs a third-party query the operator
  withdrew consent for. This is the reason the rule is worth a definition rather than a convention.
- **Nothing enforces §4.** A new CT source that seeds a throttle row writes its slug in two
  languages. Review carries that, and the first poll fails loudly if it is wrong.
- **`CONTEXT.md` gains nothing.** `Source` is already defined there and this rule is about how a
  source's key is spelled in two packages, which is code structure rather than a domain term.
- **A neighbouring rule of the same shape is not merged into this one.** [#1319](https://github.com/winniel123/verge-asm/issues/1319)
  gap 3 states that a resolver fallback constant must equal a migration's shipped default. It is the
  same *two literals must stay equal, and drift is silent* shape over different code, and it is
  ruled on its own ticket. This ADR does not bind it and does not claim to.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **A test asserting each catalogue row's `Slug` equals its `internal/scan` constant** | A test is a mechanism that can be absent, and it does not remove the second literal it guards. It also needs an allow-list of which catalogue rows are CT sources, because eight rows have no constant and must not gain one, so the test grows a fourth copy of the knowledge and drifts on its own |
| **Move the catalogue into `internal/scan` so the slug has one home** | The catalogue holds consent tier, ship note, `MayResolve` and `Unresolvable` prose for eight non-CT rows, none of which `internal/scan` has any business holding. It would drag the whole ADR-0003 catalogue into a package about Scan mechanics to fix three strings |
| **Move the constants into `cmd/web` beside the catalogue** | `internal/queue` reads them and cannot import a `main` package. This direction does not compile |
| **Key `source_state` on the catalogue row's index or on a numeric id** | `db/migrations/18500_source_state.sql` states the reason it keys on the slug: *"Keying on the source's stable slug (never a display name, which is a rendering) keeps the catalogue free to move a label without stranding a row."* An index or an id is not stable across a catalogue reorder, so it trades a visible drift for an invisible one |
| **Have `sourceEnabled` return an error where no row matches** | It is not an error. A source with no row is the normal state — the shipped default, which is what `source_state` exists to override. The absent row means *the operator has not spoken*, and turning that into an error would break every install on its first tick |
| **Have `ReserveCTSlot` upsert instead of update, so a drifted constant self-heals** | It would convert this ADR's one loud failure into a third silent one. The new row starts with a fresh reservation, so the instance-wide 5 req/min ceiling ADR-0005 and ADR-0106 both rest on is briefly doubled. The loud failure is the property worth keeping |
| **State the rule on [ADR-0106](./0106-the-ct-poll-is-a-scan-that-schedules-and-a-ct-admission-is-a-name-citing-its-batch.md)** | ADR-0106 rules the `ct` Scan. This rule reaches `ct-tail` and `certspotter`, neither of which existed when it was written, and it reaches the catalogue in `cmd/web`, which ADR-0106 does not touch |
| **State the rule on [ADR-0003](./0003-third-party-source-consent-bar.md)** | ADR-0003 rules whether a source may ship enabled. It is about terms and consent, and a rule about which Go file spells a key would sit inside it as an unrelated paragraph |
| **Merge with [#1319](https://github.com/winniel123/verge-asm/issues/1319) gap 3 into one *"two literals must stay equal"* ADR** | The two rules bind different code and resolve differently. This one is fixable by a definition, because both Go sites are in the same build. A Go constant against a shipped SQL default is not, because SQL cannot name the constant — which is exactly the case §4 carves out here. One document stating both would have to state the fix twice and the exception twice |
