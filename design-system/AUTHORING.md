# Authoring boundary — read this before adding a component

If you opened `design-system/` intending to add a component, stop here first.

## The rule

Verge ASM does not author design-system components. All components are created in Claude Design and imported into `design-system/`. When a screen needs a component the system does not have, do not build it here: write a component-request markdown file from `design-system/COMPONENT-REQUEST.md`, and hand it to the user to give to Claude Design. Restyling within existing tokens/components is fine; creating a new component file in this repo is not.

## Why

This is the canonical decision recorded in [ADR-0109](../docs/adr/0109-design-system-components-are-authored-in-claude-design-and-imported.md) — design-system components are authored in Claude Design and imported. See also `../docs/agents/design-system.md` for how visual work uses the system.

## What to do instead

- **Need a new component?** Copy [`COMPONENT-REQUEST.md`](./COMPONENT-REQUEST.md), fill it in, save it under `design-system/requests/<name>.md`, and hand it to the user for Claude Design. Do not build the component in this repo.
- **Just need a different look?** Restyling within existing tokens and composing existing components is allowed and expected — that is not authoring a new component.
