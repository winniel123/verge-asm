# Prototype — the copy a landed `Act` corpus makes true

Throwaway. Answers [#1792](https://github.com/winniel123/verge-asm/issues/1792) for map
[#1786](https://github.com/winniel123/verge-asm/issues/1786). Nothing here is production code and
nothing here ships. No PR.

## Run it

```sh
bash prototype/act-copy/build.sh
```

Open `act-copy.html` in a browser. Five sites, each switching between the shipped string and its
candidates. Two free-play controls in the bottom bar: **Show every "today"** puts all five wrong
strings on screen in one pass, **Dark** flips the theme.

## What it is

- `body.html` — the five drawn sites and the verdict notes.
- `prototype.css` — chrome only. The variant switcher, the verdict pills, the ADR-0075 mark.
- `prototype.js` — the switcher and the theme toggle.
- `build.sh` — assembles one self-contained file.

`build.sh` reads the seven token files and the `st-` block **out of
`design-system/templates/settings.tmpl` itself**, so the drawing cannot drift from the shipped
styles and the copy is judged at the size and colour it will really have. `act-copy.html` is
generated and gitignored.

## The five sites

| # | Site | Verdict |
| --- | --- | --- |
| 1 | Remove-member dialog — `settings.tmpl:743`, `Settings.jsx:456`, `DocsPage.jsx:113` | Say what is refused |
| 2 | Audit tab lede — `settings.tmpl:866` | "Who did what, when." alone |
| 3 | Empty state — `settings.tmpl:882-883` | Dated from when the record began |
| 4 | Sources callout — `settings.tmpl:898` | Both instants, kept apart |
| 5 | Removal refusal — `cmd/web/settings.go:485` | A fifth site the ticket did not name |

## ADR-0075

Each site draws its shipped string so the loss is visible. A drawn line of product copy is the
operative voice under ADR-0075 limb 2, so every `today` variant carries the mark on the rendered
surface, as a replacement rather than a strike — it names the ruling and states what the surface
would draw instead.

## What it decided

The five replacement strings, and one amendment to #1787's row for site 4. The rulings and the
findings are on the ticket's resolution comment.
