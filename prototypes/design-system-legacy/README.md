# design-system-legacy (frozen)

This is the frozen **"engineered paper"** stylesheet — the original Verge ASM visual language (flat, sharp 0px corners, near-monochrome plus one blue, Helvetica / IBM Plex Mono, hard `6px 6px 0` offset shadow, 2px ink rules). It has been superseded by the redesigned system now at [`../../design-system/`](../../design-system/) (see wayfinder map #263).

It is kept here for one reason only: the dated prototypes alongside it in [`../`](../) link this stylesheet so they keep **rendering exactly as drawn**. Per [ADR-0075](../../docs/adr/0075-a-prototype-is-a-dated-record-of-a-reading-never-of-a-rule.md), a prototype is a dated record of a reading and is never redrawn — the wrong screen is the evidence — so its stylesheet must not move out from under it. It lives here, under `prototypes/`, to sit next to its only consumers.

Only `styles.css` and `tokens/` are retained (the components, guidelines, and UI kits were removed in the pivot). **Do not edit or extend this directory.** New visual work — production UI, prototypes, mocks, slides — uses [`../../design-system/`](../../design-system/).
