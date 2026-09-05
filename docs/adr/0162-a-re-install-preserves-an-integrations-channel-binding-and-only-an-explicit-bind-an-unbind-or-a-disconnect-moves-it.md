# ADR-0162: A re-install preserves an integration's Channel binding, and only an explicit bind, an unbind, or a disconnect moves it

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1389 ADR gaps: db/queries (3/4)](https://github.com/winniel123/verge-asm/issues/1389), gap 1
- **PR that deleted the comment:** [#1388](https://github.com/winniel123/verge-asm/pull/1388)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [`docs/guides/integrations.md`](../guides/integrations.md)'s *"Re-installing is an upsert of the one current state"*, at that clause's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [ADR-0073](./0073-an-operator-dial-carries-no-author-however-specific-its-target.md) and [ADR-0093](./0093-an-instant-on-a-declared-term-is-earned-by-an-act-nothing-else-dates.md). They rule that a Declared act carries no timeline, no actor and no instant. That is why the install is an upsert at all. Neither rules the Channel binding

## Context

[`db/queries/integrations.sql`](../../db/queries/integrations.sql) carried this above
`UpsertIntegrationState`, until #1388 deleted it:

```sql
-- holds only the current install state. The channel binding is NOT touched here:
-- a re-install keeps whatever delivery Channel the integration was bound to (the
-- ON CONFLICT omits channel_id, leaving the existing value in place), and a first
-- install lands unbound (channel_id defaults NULL).
```

The block cited ADR-0073 and ADR-0093, and it cited them for the Declared-act half alone. Neither
ADR names a Channel, a binding or a column. The citation therefore suppresses nothing here, under
`comment-policy.md` §8.3's rule that a source suppresses only where it states the rule.

**Two migrations authored the table, and neither states the preservation.**

[`db/migrations/21200_integration_state.sql`](../../db/migrations/21200_integration_state.sql)
creates `integration_state` as `(slug, state)` and states the Declared-act half in the same words the
deleted comment used.
[`db/migrations/23200_integration_channel.sql`](../../db/migrations/23200_integration_channel.sql)
adds `channel_id BIGINT REFERENCES channel (id) ON DELETE SET NULL`. Its comment states two of the
four facts this ADR rules: `NULL` means unbound and is *"the freshly-installed default"*, and
deleting a bound Channel *"degrades that integration to unbound"*. It says nothing about what a
re-install does to an existing binding. A migration is also not one of §4.7's five documented
places, so it could not have suppressed the record on its own.

**The upsert's shape is the whole mechanism, and it is invisible from the outside.**

```sql
INSERT INTO integration_state (slug, state)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
    SET state = EXCLUDED.state
RETURNING slug, state, channel_id;
```

`channel_id` appears in `RETURNING` and nowhere else. It is absent from the `INSERT` column list, so
a first install leaves the column at its `NULL` default. It is absent from the `DO UPDATE SET` list,
so a re-install over an existing row leaves the stored value where it was. Adding one column name to
that `SET` list would silently unbind every integration on its next install, and nothing in the
query, the handler or the tests would say that the omission was a choice.

**The rule binds two paths outside the query file.**

- [`cmd/web/integrations.go`](../../cmd/web/integrations.go)'s `installIntegration` calls the upsert.
- The same file's `testIntegration` reads `GetIntegrationChannel` and refuses to send when the
  binding is absent or `NULL`.

**`docs/guides/integrations.md` is the on-topic source that rules something else.** Its line 51
states *"Re-installing is an upsert of the one current state. Disconnecting deletes the row."* It
never names `channel_id`. Its own `CREATE TABLE integration_state` block shows `(slug, state)`, which
is migration 21200 as authored and is stale against the live schema. The guide's longest section
argues that an integration is not a delivery channel, which is a neighbouring rule and not this one.

## Decision

> **An install writes `state` alone. The upsert never writes `channel_id`, so a re-install over an
> existing row preserves the delivery Channel the integration was already bound to. Three operator
> acts move the binding and no others: an explicit bind, an explicit unbind, and a disconnect, which
> deletes the row and the binding with it. A row created by a first install lands unbound.**

Four limbs.

### 1. The omission from `ON CONFLICT DO UPDATE SET` is the decision

`channel_id` is deliberately absent from the `INSERT` column list and from the `DO UPDATE SET` list
of `UpsertIntegrationState`. An author who adds it to either list changes the rule this ADR states
and must change this ADR first.

The `RETURNING` clause names `channel_id` because the caller reads the row back. Reading it is not
writing it.

### 2. Three acts move the binding, and one event outside the integration moves it too

| Act | Query | Effect on the binding |
| --- | --- | --- |
| Bind a Channel | `SetIntegrationChannel` with a channel id | Sets `channel_id` |
| Unbind | `SetIntegrationChannel` with an empty form value | Sets `channel_id` to `NULL` |
| Disconnect | `DeleteIntegrationState` | Removes the row, and the binding with it |
| Install or re-install | `UpsertIntegrationState` | **None** |

`SetIntegrationChannel` is the only statement in the tree that writes `channel_id`.
`bindIntegrationChannel` in `cmd/web/integrations.go` is its only caller, and an empty `channel`
form value there is an explicit unbind rather than a no-op.

One event moves the binding from outside the integration. Deleting the bound Channel sets
`channel_id` to `NULL` through `ON DELETE SET NULL`. Migration 23200 states that rule and this ADR
does not restate it.

### 3. A disconnect and then an install is a first install, not a re-install

A disconnect deletes the row. The binding is a column of that row, so it goes with it. The next
install is an `INSERT` that finds no conflict, and it lands unbound.

The distinction is the reason limb 2 lists disconnect separately from re-install. Both reach the
integration through the same drawer, and only one of them keeps the Channel.

### 4. An unbound integration is rendered and refused, never guessed

Two readers see the unbound state and neither picks a Channel for the operator.

- `fillIntegrationsSection` copies `channel_id` into the drawer only when the value is valid. An
  unbound integration leaves `BoundChannel` empty, and the select shows the `"Not connected"` option.
- `testIntegration` treats `pgx.ErrNoRows` and an invalid binding as the same case. It toasts *"No
  delivery channel"* and sends nothing.

## Consequences

- **[`docs/guides/integrations.md`](../guides/integrations.md) gains a bounding clause** at the
  sentence that states the upsert, and one line marking its `CREATE TABLE` block as migration 21200
  as authored rather than as the live schema. ADR-0058 requires the edit at that site. A reader who
  finds the clause and the live table must not have to find this ADR first.
- **[`cmd/web/integrations.go`](../../cmd/web/integrations.go)'s install path gains one comment**
  carrying this citation, beside the upsert call. That is the site an author reaches when they change
  what an install writes.
- **[`db/queries/integrations.sql`](../../db/queries/integrations.sql) gains nothing.** `sqlc` copies
  a comment above a query into `internal/db` as a doc comment on the generated method and on
  `Querier`. `comment-policy.md` §2.2 keeps Go declaration position empty, and #1388 removed 1,360
  generated lines for that reason. Re-adding the comment would put it back and cost a regeneration of
  a file no author edits.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** It avoids *integration* as a term at two sites
  and holds no entry for the install state or the binding.
- **No production behaviour changes.** The queries and the handlers already have the shape this ADR
  states. What changes is that the shape now has a record.
- **A second writer of `channel_id` needs its own decision.** This ADR names the one statement that
  writes the column today. A new writer is outside the rule and gets no cover from it.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Add `channel_id` to `ON CONFLICT DO UPDATE SET`, so a re-install resets the binding** | An install is a Declared act about install state, and the binding is a separate declaration the operator made separately. Resetting it makes a re-install a silent unbind, and the operator's only signal is a `"Send test"` button that stopped working. Migration 23200 already chose the softer direction for a Channel delete, where it degrades one integration rather than stranding the delete |
| **Make the install path bind a default Channel when the row lands unbound** | Fabricates a delivery target the operator never chose, and the guide's own banner states that Channels need no integration. It also contradicts limb 4, where both readers refuse rather than guess |
| **Restate the rule in a comment on `db/queries/integrations.sql`** | The comment becomes a doc comment on the generated `UpsertIntegrationState` method and on `Querier`. §2.2 keeps declaration position empty, and #1388 deleted the generated copies for that reason. The citation goes on the Go call site, which is where an author who changes the install path is reading |
| **Write the rule in [`docs/guides/integrations.md`](../guides/integrations.md) and file no ADR** | The guide is operator-facing and states what the screen does. This is a rule about which statement may write a column, and it binds a query file and two handlers. The guide still gains the bounding clause, because ADR-0058 requires it at the stale site |
| **Fold the rule into [ADR-0093](./0093-an-instant-on-a-declared-term-is-earned-by-an-act-nothing-else-dates.md)** | That ADR rules what dates a Declared term. The deleted comment cited it for exactly that half and for nothing else. A rule about a nullable foreign key's write set is not a rule about instants |
| **Fix the guide's stale `CREATE TABLE` block by deleting it** | ADR-0058's name-and-withdraw convention refuses a silent deletion. The block is a true quotation of migration 21200 and it is only stale as a claim about the live schema, so it is marked rather than removed |
