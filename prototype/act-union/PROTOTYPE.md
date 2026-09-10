# Prototype — the `Act` closed union in four columns

Throwaway. Answers [#1790](https://github.com/winniel123/verge-asm/issues/1790) for map
[#1786](https://github.com/winniel123/verge-asm/issues/1786). Nothing here is production code and
nothing here ships. No PR.

## Run it

```sh
go test ./prototype/act-union      # the three hazards, the count, the collision
go run  ./prototype/act-union      # prints the table, writes act-union.html
```

Open `act-union.html` in a browser. Five tabs: all 62 variants, the three hazards, and the Actor
collision. Click any row to see the stored row. Two free-play controls: **Stress the widths** appends
a long value to every Subject cell, **Dark** flips the theme.

## What it is

- `act.go` — the closed union. 62 variants, one per act class, each naming its own typed subject.
- `subject.go` — the 26 payload structs the variants share.
- `actor.go` — #1795's `Account | GrantHolder(Grant) | System`.
- `row.go` — the stored row shape, plus encode and decode.
- `catalogue.go` — one real sample per variant, and the Action label table.
- `hazard_test.go` — the ticket's three hazards, plus the count and the collision.

The catalogue is the single source for the label table, the decoder table and the rendered
walkthrough, so a variant added without a sample fails the count test rather than rendering blank.

## The count

62 = **60** (#1791's corrected class count) + **1** (the split of `POST /coverage/retention` into two
rows) + **1** (`migration.applied`, conditional on [#1805](https://github.com/winniel123/verge-asm/issues/1805)).

By limb, in classes rather than acts: limb 1 · 33, limb 2 · 20, limb 3 · 7, limb 1+2 · 1, limb 4 · 1.
#1789's per-limb tallies count **acts**; three route pairs collapse, so limb 1 · 34 acts is 32
classes and limb 3 · 8 acts is 7 classes.

## What it decided

Settled #12 holds. The four shipped columns render all 62 variants with no empty cell and no fifth
column. The design decisions and the findings are on the ticket's resolution comment.
