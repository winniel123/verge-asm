# Deriving a port set from `nmap-services` does not make verge-asm a derivative work — but the top-1000 tier is the one place where that stops being clear

Research ticket [#78](https://github.com/winniel123/verge-asm/issues/78) — wayfinder research for
the verge-asm v1 spec.

**Question.** Does deriving a port set from `nmap-services` make verge-asm a derivative work under
the **NPSL** — and if so, is that compatible with **AGPL-3.0**? And if it is not, what does
`verge-core`'s frequency half become?

**Framing.** [ADR-0009](../adr/0009-verge-core-is-a-union.md) fixes the shape:
`verge-core = frequency-set ∪ sensitive-list`. **The sensitive half is not in question** — it is
attested per row under [#21](https://github.com/winniel123/verge-asm/issues/21) and owes nothing to
nmap. Only the frequency half and
[`safe-active-probing.md`](./safe-active-probing.md) §2.4's **weekly top-1000 tier** are.

Three constraints bind before any evidence is gathered, and they are the ticket's own.

1. **The project licence is AGPL-3.0 and it is not moving.** The map's standing instruction is to
   *flag anything that would force a licence change*. This note is that flag being checked, not
   raised.
2. **This is not a re-litigation of the port tiers.**
   [`project-authored-constants.md`](./project-authored-constants.md) §3.5 already ruled that the
   staleness of `nmap-services` costs **aperture only** and is not grounds to change anything. §11
   below moves the warm tier on a **licence** ground. #71's value finding is cited only to price
   that move, never as an independent reason to make it.
3. **This is a project decision on retrieved text, not legal advice.** §13 states plainly which
   question turns on a merits call only a court could make, and rules on the conservative reading
   there.

[#71](https://github.com/winniel123/verge-asm/issues/71) §5.4 opened this **as a question rather
than a finding**, and said why: the NPSL text was never retrieved and quoted against bytes.
[#37](https://github.com/winniel123/verge-asm/issues/37)'s precedent is why that mattered — *a row
may not move on a re-reading of text already held. A verdict changes on retrieval*. This note is
that retrieval.

---

## 1. Summary

| Decision | Answer |
|---|---|
| Which NPSL is in force, and was it read off the default branch | **Version 0.95**, `nmap/nmap` **`master`** (the default branch), HEAD `b403ddee` of 2026-08-13. The `LICENSE` blob is unchanged since `47919b8d`, **2023-01-11** — §2, §3 |
| Does the ticket's reported §3 quote survive contact with the bytes | **Both halves are verbatim-accurate — and the report drops the parenthetical that is the entire point of v0.95** — §3.2 |
| Is `nmap-services` a "Covered Software data file" | **Yes.** §3's `such as` list names `nmap-os-db` and `nmap-service-probes` and not `nmap-services`, but the list is illustrative and the file's own header asserts the NPSL over itself — §4 |
| Is there any carve-out for data files | **No — negative retrieval, extent stated in §5** |
| **Limb 1 — the ~~~140-port~~ hot set** (**123 TCP** measured, §6 note) | **Not a derivative work, on two independent grounds either of which suffices.** The licence does not reach it by its own terms (§6.1), and there is no protectable subject matter to reach (§8) — §6 |
| **Limb 2 — the weekly top-1000 tier** | **Depends on how it is built, and one of the three builds is squarely inside §3.** A build step that parses the file is the case §3's `Reads` bullet was written for. Shipping the 1,000 integers is not, but it is the one place we would reproduce **nmap's** selection rather than making our own — §7 |
| Is the NPSL compatible with AGPL-3.0 | **No, and there is no upgrade path.** Nmap says so first-party; the FSF says so; and GPLv2 §6 — quoted from the NPSL's own Exhibit A — is the mechanism — §9 |
| Is a list of port numbers copyrightable | **The integers are not, ever. A selection of them can be — and neither nmap's top-N nor ours is the kind that is** — §8 |
| Is this [#27](https://github.com/winniel123/verge-asm/issues/27)'s shape | **Limb 1, no — there is no redistribution permission we need. Limb 2 built as a shipped list, yes in shape but it survives on different ground, and that ground is thinner** — §10 |
| Does the waiver route work | **No, and #27 is why.** Nmap offers one in writing. A waiver granted to the project **does not travel downstream**, and AGPL-3.0 requires that it does — §10.3 |
| **What the frequency half becomes** | **The hot set is unchanged. The weekly top-1000 tier is retired and not replaced** — the opt-in full-range tier already covers what it covered, at 30-day latency instead of 7 — §11, §12 |
| The losing option | **Self-generating a ranking.** It loses on **capability**, not cost: verge-asm has no internet vantage and acquiring one contradicts #4 §1's entire safety posture — §11.1 |
| Is IANA's registry the replacement | **No — and its licence is not the reason.** It is **CC0 1.0** and unambiguously AGPL-clean. It loses on fitness: it is a **name** registry with no frequency column, so it answers a different question — §11.3 |

**The headline is that NPSL v0.95 was rewritten in January 2023 to answer exactly the question this
ticket asks, and the answer is in the licensor's own commit message.** Commit `d0a8fb0f`
(2023-01-11) says v0.95's *"only changes (besides version number) are clarifications that
derivative works definition and all other license clauses only apply to parties who choose to
accept the license in return for the special rights granted … If a party can do everything they
need to using copyright provisions outside of this license such as fair use, we support that and
aren't trying to claim any control over their work."* #71 §5.4's reported reading was of a clause
that had already been narrowed three years before it was reported.

**Two measured facts frame the rest.** verge-asm's frequency half is **not** nmap's top-100:
measured against [`safe-active-probing.md`](./safe-active-probing.md) §2.3's own text, it retains
**81** of nmap's 100 after **19** project-chosen deletions and adds **44** ports net-new that
nmap's ranking does not support at any size — a set of **125** as #4 specified it, **123** after
ADR-0009's two transport removals (§6.2). And the weekly tier is the opposite: **1,000 of 1,000**
of nmap's, with no project judgement in it at all. The two limbs are not the same object and the
ticket was right to refuse to conflate them.

---

## 2. What was retrieved

Everything below was fetched on **2026-08-14** unless stated. Sizes and hashes are recorded because
[#67](https://github.com/winniel123/verge-asm/issues/67)'s `lego` hazard is a *successful fetch of
superseded bytes*, not a failed fetch.

| Source | What it is | How |
|---|---|---|
| **NPSL v0.95** | The licence, from nmap's own repository | `raw.githubusercontent.com/nmap/nmap/master/LICENSE` — HTTP 200, **29,575 bytes**, 583 lines, `sha256 9d9a9a76…dae6d3` |
| Repository metadata | Establishes `master` **is** the default branch | `api.github.com/repos/nmap/nmap` → `"default_branch":"master"`; HEAD `b403ddee`, 2026-08-13 |
| `LICENSE` commit history | Establishes the licence has not moved since v0.95 | `api.github.com/repos/nmap/nmap/commits?path=LICENSE` — last touch `47919b8d`, **2023-01-11** |
| **nmap.org's copy** | The divergence check | `svn.nmap.org/nmap/LICENSE` — HTTP 200, 29,575 bytes, **byte-identical** to the GitHub blob (`diff` clean) |
| Annotated NPSL v0.95 | Licensor's own gloss, explicitly non-binding | `nmap.org/npsl/npsl-annotated.html` — HTTP 200, 45,902 bytes |
| **`nmap-services`** | The file itself, for its header | `raw.githubusercontent.com/nmap/nmap/master/nmap-services` — HTTP 200, **998,268 bytes**, 27,483 lines, 8,390 TCP rows |
| `nmap-service-probes`, `nmap-os-db`, `nmap-protocols` | Comparators for the header question | same host, same branch |
| `docs/DATAFILES.md`, `docs/licenses/` | The data-file carve-out search | `api.github.com/repos/nmap/nmap/contents/docs` |
| **nmap.org Legal Notices** | Nmap's own compatibility statement | `nmap.org/book/man-legal.html` — HTTP 200 |
| **nmap/nmap#2199** | Licensor's own reading of the `Reads` bullet, on the record | `api.github.com/repos/nmap/nmap/issues/2199` + 30 comments, **open** since 2020-12-06 |
| **FSF licence list** | GPLv2 ↔ GPLv3/AGPL compatibility, primary | `gnu.org/licenses/license-list.en.html` — HTTP 200, 145,965 bytes |
| **17 U.S.C. §101, §102** | Compilation definition; the idea/expression bar | Cornell LII |
| ***Feist v. Rural Telephone***, 499 U.S. 340 (1991) | The controlling authority on factual compilations | Cornell LII. *(Justia returned **HTTP 403**; recorded per §14)* |
| **Compendium of U.S. Copyright Office Practices, 3rd ed.** | The registrar's own practice on facts and compilations | `copyright.gov/comp3/chap300/…pdf`, 641,671 bytes, ed. **01/28/2021**, extracted locally |
| **37 C.F.R. §202.1** | The regulation the Compendium cites | Cornell LII |
| **Directive 96/9/EC** | The EU *sui generis* database right | EUR-Lex, CELEX `31996L0009` |
| **IANA port registry + terms** | The replacement candidate, and its licence | `iana.org/assignments/service-names-port-numbers/…csv`, **15,399 rows**, last updated **2026-08-11**; `iana.org/help/licensing-terms` |

**No divergence between nmap's two published copies.** The ticket asked for this specifically. The
GitHub blob on `master` and the copy served at `svn.nmap.org/nmap/LICENSE` — which is the URL
`nmap-services`' own header points readers at — are byte-for-byte the same file, both declaring
**Version 0.95**. The `lego` hazard does not fire here.

One dangling pointer, noted and **not load-bearing**: `nmap-service-probes` directs readers to
`https://nmap.org/data/LICENSE`, which **404s**. `nmap-services` points at `svn.nmap.org`, which
resolves. Our file's pointer is good.

---

## 3. The NPSL as retrieved

### 3.1 What it is

The file opens:

> Nmap Public Source License Version 0.95
> For more information on this license, see https://nmap.org/npsl/

The structure is a **Main License Body** of 13 numbered articles plus **Exhibit A**, which is the
GNU GPL version 2 reproduced in full (minus its "How to Apply These Terms" appendix). §1 defines:

> * **"GPL"** means the GNU General Public License Version 2, as published by the Free Software
>   Foundation and provided in Exhibit A.

**There is no `or any later version`.** The GPL incorporated by the NPSL is pinned at 2. This
matters in §9.

§2 states the precedence rule and the incompatibility in the same breath:

> Covered Software is licensed to you under the terms of the GPL (Exhibit A), with all the
> exceptions, clarifications, and additions noted in this Main License Body. Where the terms in this
> Main License Body conflict in any way with the GPL, the Main License Body terms shall take
> precedence. **These additional terms mean that You may not distribute Covered Software or
> Derivative Works under plain GPL terms without special permission from Licensor.**

### 3.2 §3 verbatim — the derivative-work definition and the distribution clause

Quoted from the retrieved bytes, lines 111–164, with the article number as it appears:

> **3. Derivative Works**
>
> This License (including the GPL portion) places important restrictions on derived works. Licensor
> interprets that term quite broadly. To avoid any misunderstandings, we consider software to
> constitute a "derivative work" of Covered Software for the purposes of this license if it does any
> of the following:
>
> * Integrates source code from Covered Software
>
> * **Reads or includes Covered Software data files, such as nmap-os-db or nmap-service-probes.**
>
> * Is designed specifically to execute Covered Software and parse the results (as opposed to
>   typical shell or execution-menu apps, which will execute anything you tell them to).
>
> * Includes Covered Software in a proprietary executable installer. […]
>
> * Links (statically or dynamically) to a library which does any of the above
>
> * Executes a helper program, module, or script to do any of the above.
>
> This list is not exclusive, but is meant to clarify Licensor's intentions with some common
> examples. **Distribution of any works which meet these criteria (and that also choose to accept
> this license to benefit from the rights granted herein) must be under the terms of this license
> (including this Main License Body and GPL), with no additional conditions or restrictions.** They
> must abide by all restrictions that the GPL places on derivative or collective works, including
> the requirements for distributing their source code and allowing royalty-free redistribution.
>
> **Licensor does not purport to control through this license any software which does not require
> the rights granted herein** (such as rights to redistribute and/or incorporate Covered Software
> executables and source code). In particular, many software packages include the ability to parse
> Covered Software results provided by an end user or to execute Covered Software that end user may
> have already installed on their system. **To the extent that copyright doctrines such as fair use
> allow their practices without the need to exercise any rights granted by this license, vendors and
> distributors of such software are not bound by our definition of derivative works or any other
> clauses in this license.**

**#71 §5.4's two reported quotes are verbatim-accurate.** *"Reads or includes Covered Software data
files"* and *"under the terms of this license (including this Main License Body and GPL), with no
additional conditions or restrictions"* are both in the bytes, in §3, exactly as reported.

**And the report drops the eleven words that decide this ticket.** The distribution clause is not
*"distribution of any works which meet these criteria must be under the terms of this license."* It
is *"distribution of any works which meet these criteria **(and that also choose to accept this
license to benefit from the rights granted herein)** must be…"*. The obligation is **conditional on
acceptance**, and acceptance is conditional on **needing something the licence grants**. That is
not an inference from the parenthetical. The paragraph immediately following says it outright, and
so does the commit that added both.

### 3.3 The version history says the parenthetical was added on purpose, for this exact case

`LICENSE`'s commit history on `master`:

| Date | Commit | Message (first line) |
|---|---|---|
| 2023-01-11 | `47919b8d` | Add paragraph break for easier reading |
| **2023-01-11** | **`d0a8fb0f`** | **Update Nmap Public Source License to Version 0.95.** *"The only changes (besides version number) are clarifications that derivative works definition and all other license clauses only apply to parties who choose to accept the license in return for the special rights granted (such as Nmap redistribution rights). If a party can do everything they need to using copyright provisions outside of this license such as fair use, we support that and aren't trying to claim any control over their work"* |
| 2021-11-23 | `158c2e49` | Change Insecure.Com LLC to Nmap Software LLC |
| 2021-01-12 | `a3c846c3` | …bump the version number to 0.93 |
| 2020-10-05 | `ef8213a3` | Reintegrate Nmap 7.90 release branch |

**Nothing has touched the licence in three and a half years.** The retrieval is of a settled file,
not a moving one.

### 3.4 The licensor's stated reading of the `Reads` bullet, on the record

`nmap/nmap` issue **#2199**, *"NPSL License Improvements"*, opened **2020-12-06** by Gentoo's
licences team, **still open**, 30 comments. Gordon Lyon (`fyodor`) answering the `Reads` objection
directly, **2020-12-07**:

> Section 3 -> **That is not how we meant or interpret the clause** (which existed in previous Nmap
> versions too). **The intent is to stop companies from trying to avoid complying with the Nmap
> license by reading and parsing and interpreting the Nmap data files directly rather than through
> Nmap execution.** I wouldn't say that "cat" itself reads Nmap data files, even if it can be used
> to display them if a user specifically requests that.

And announcing v0.95, **2023-01-11**, in the same thread:

> After further review of the NPSL (Version 0.94), I can see how it could be construed as burdening
> Nmap-related software … which doesn't ship with Nmap and doesn't need the extra rights granted by
> the NPSL (such as redistribution rights). **That was never our intention**, but we just released
> NPSL Version 0.95 … to further clarify that the NPSL **is not meant to contaminate other software
> packages which are separate works that don't incorporate Nmap source code or executables**.

**Weight, stated honestly.** These are the licensor's statements of *intent*, not licence text, and
§0 of the licence says its own explanatory page *"does not and can not modify its governing terms in
any way."* They are cited here for what they are: contemporaneous evidence of how the party who
would have to enforce §3 reads §3 — which is the only party whose reading is at risk of being
tested against us. They corroborate the text. They do not carry it. The text carries itself.

---

## 4. Does `nmap-services` carry terms of its own? No — it asserts the repository licence over itself

§3's `such as` list names **`nmap-os-db`** and **`nmap-service-probes`**. It does **not** name
`nmap-services`. That is not a carve-out: the sentence introducing the bullets says *"To avoid any
misunderstandings, we consider software to constitute a 'derivative work' … if it does any of the
following"*, and the sentence closing them says *"This list is not exclusive."* Reading the omission
as an exemption would be reading an illustrative list as exhaustive.

The file settles it anyway. `nmap-services` lines 1–21, verbatim from the retrieved bytes:

> ```
> # THIS FILE IS GENERATED AUTOMATICALLY FROM A MASTER - DO NOT EDIT.
> # EDIT /nmap-private-dev/nmap-services-all IN SVN INSTEAD.
> # Well known service port numbers -*- mode: fundamental; -*-
> # From the Nmap Security Scanner ( https://nmap.org/ )
> #
> # $Id: nmap-services 9746 2008-08-26 18:45:24Z fyodor $
> #
> # Derived from IANA data and our own research
> #
> # This collection of service data is (C) 1996-2025 by Insecure.Com
> # LLC.  It is distributed under the Nmap Public Source license as
> # provided in the LICENSE file of the source distribution or at
> # https://svn.nmap.org/nmap/LICENSE .  Note that this license
> # requires you to license your own work under a compatable open source
> # license.  If you wish to embed Nmap technology into proprietary
> # software, we sell alternative licenses (contact sales@insecure.com).
> # Dozens of software vendors already license Nmap technology such as
> # host discovery, port scanning, OS detection, and version detection.
> # For more details, see https://nmap.org/book/man-legal.html
> #
> # Fields in this file are: Service name, portnum/protocol, open-frequency, optional comments
> ```

**Three findings, all against bytes.**

1. **It carries no terms of its own.** It asserts a copyright and then points at the repository
   licence. There is no additional grant and no additional restriction. The parenthetical sentence
   — *"this license requires you to license your own work under a compatable open source license"*
   [*sic*] — is a paraphrase of §3, not a separate term, and it is subject to §3's own conditional.
2. **It is the same boilerplate as a file §3 names by name.** `nmap-service-probes` carries the
   identical block, differing only in dates and the (broken) LICENSE URL. So there is no textual
   basis for treating `nmap-services` as a lesser-protected data file than `nmap-service-probes`,
   and none is claimed here.
3. **It says where the data came from, and that provenance is CC0.** *"Derived from IANA data and
   our own research."* IANA's protocol registries are dedicated to the public domain under CC0 1.0
   (§11.3). #71 §5.1 measured how much of the file is the IANA half rather than the research half:
   **1,969 TCP lines** carry the filler value `0.000076` imported from IANA's *name* registry on
   2016-09-14. That does not make the file unprotected — but it means a large part of *nmap's
   selection* is **IANA's** selection, and nobody's exclusive right attaches to that.

**Two footnotes on chain of title, neither load-bearing.** The header says the data is
`(C) 1996-2025 by Insecure.Com LLC`. The licence says `"Licensor" means Nmap Software LLC`, and
commit `158c2e49` (2021-11-23) renamed the entity in `LICENSE` and did not update this header. And
the `$Id` still reads **2008-08-26** — #71 §5.1's finding, confirmed again here on a fresh fetch.

---

## 5. Is there any separate nmap statement about `nmap-services`? No — negative retrieval, with its extent

[#66](https://github.com/winniel123/verge-asm/issues/66) makes a negative retrieval a verdict.
[#25](https://github.com/winniel123/verge-asm/issues/25) requires it to state its own reach. What
was searched:

| Searched | Result |
|---|---|
| Repository root of `nmap/nmap` on `master`, for `LICENSE*` / `COPYING*` / `NPSL*` | **One file: `LICENSE`.** No `COPYING`, no per-file licence, no `LICENSE-DATA` |
| `docs/licenses/` | 13 files, **all third-party** (BSD, LGPL-2, Lua, MIT, MPL-1.1, OpenSSL, PCRE, WinPcap, zlib, Libdnet, LIBLINEAR). **No nmap data-file licence** |
| `docs/DATAFILES.md` | Exists, and is about **update provenance**, not terms. Its entire `nmap-services` line: *"nmap-services - no current automated process for updating these from IANA"* |
| `nmap.org/npsl/` | Links only to the annotated and plain licence. **No data-file discussion at all** |
| `nmap.org/npsl/npsl-annotated.html` | 641 lines. Grep for `nmap-services`: **zero hits.** `data file` appears only in the §3 bullet already quoted and its annotation |
| `nmap.org/book/man-legal.html` (Legal Notices) | Discusses the NPSL, Npcap, the reference guide's CC licence and third-party libraries. **Says nothing about data files as a class** |
| `nmap.org/book/nmap-services.html` | The file's own manual chapter. **The only occurrence of the string `licen` is the site-footer navigation link.** It documents the format and the frequency column and states no terms |
| `nmap.org/npsl/faq.html`, `nmap.org/oem/faq.html` | **HTTP 404 both.** There is no NPSL FAQ and no OEM FAQ at those paths |
| GitHub code search, `repo:nmap/nmap nmap-services license` | 38 hits, all the boilerplate header or `Makefile`/spec install rules |

**Verdict: nmap has published no carve-out, no clarification and no separate grant for
`nmap-services` or for data files as a class.** The file inherits the NPSL and nothing modifies it.

**Extent, stated.** This searched nmap's repository default branch, nmap.org's licensing pages, the
book chapter for the file itself, and the licence-improvement issue thread. It did **not** search
the `nmap-dev` mailing list archives, and did not attempt private correspondence. A statement made
only on a mailing list would have been missed. That is the whole of the gap.

---

## 6. Limb 1 — the ~140-port hot set is not a derivative work

> **`~140` is a legacy label for limb 1's object and is not its size.** **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97), independently reproducing **this note's own
> §6.2 count**: `verge-core`'s frequency half — the thing limb 1 is about — is **123, all TCP**
> (`81 + 44 = 125`, less ADR-0009's removal of `161/tcp` and `623/tcp`), and `verge-core` itself is
> **136 pairs** under ADR-0009's union ([`sensitive-ports.md`](./sensitive-ports.md) §29, composed with
> [#95](https://github.com/winniel123/verge-asm/issues/95)). §6.2 parked the reconciliation as out of
> scope for a licence question and it is now discharged. **The name is left standing per the
> name-and-withdraw convention — it appears in §1's summary table and in §12 ruling 2 as well — and no
> part of the ruling turns on the count**: §6.1 and §8 both fail the trigger for reasons that hold at
> any size.

`safe-active-probing.md` §2.3 specifies `verge-core`'s frequency half as an **editable list file**
of hand-selected integers. It is **not** `nmap-services` and **not** a copy of it. Nothing of
nmap's ships. Nothing of nmap's is read at build time. Nothing of nmap's is read at runtime. The
question is whether *"Reads or includes Covered Software data files"* reaches a **design-time
selection** that ships neither the file nor its contents.

It does not, on two grounds. Either is sufficient. They fail independently, which is why both are
argued.

### 6.1 Ground one — the licence does not reach it, by the licence's own terms

Three steps, each on retrieved text.

**Step 1. The obligation is conditional, and the condition is not met.** §3's distribution clause
binds works which meet the criteria *"and that also choose to accept this license to benefit from
the rights granted herein"*. §3's closing paragraph names the escape in the affirmative:
*"Licensor does not purport to control through this license any software which does not require the
rights granted herein (such as rights to redistribute and/or incorporate Covered Software
executables and source code)."*

verge-asm requires **none** of those rights. It does not redistribute Covered Software. It does not
incorporate Covered Software executables or source. It does not link to nmap, does not execute
nmap, and does not parse nmap output. There is no NPSL-granted right in its dependency graph, so
there is nothing to accept in exchange for.

**Step 2. The bullet's subject is `software`, and no software of ours reads anything.** §3's
operative sentence is *"we consider **software** to constitute a 'derivative work' … if it does any
of the following: … Reads or includes Covered Software data files."* The actor is a program. In
verge-asm's case the reading was done by a person at design time and produced a written argument in
a research note. There is no build step that opens the file, no vendored copy, no fetch. The
shipped artefact is a list file whose contents were **decided**, not **extracted**.

**Step 3. The licensor's stated reading is the same one, and it names the distinction.**
`#2199`, 2020-12-07: the intent is *"to stop companies from trying to avoid complying with the Nmap
license by reading and parsing and interpreting the Nmap data files directly rather than through
Nmap execution."* Reading, parsing, interpreting — the thing being described is a *program
substituting file access for nmap execution*. Nothing in verge-asm does that. §3.4's caveat on
weight applies: this corroborates the text, and the text stands alone.

**The one clause that cuts the other way, and why it does not bite.** §2's final sentence: *"you
agree to the terms of this License by clicking the Accept button or downloading the software."*
Downloading `nmap-services` in order to read it is, on that sentence's face, acceptance. Two
answers. First, even granting acceptance, acceptance does not manufacture a derivative work — it
would bind us to a definition that our software still does not meet, per steps 1 and 2. Second, the
clause is contested by the licensor's own reviewers: Gentoo's objection in #2199 is precisely that
*"If I download the software from a third party (like Github or a distro mirror), I won't enter into
any contract with either Insecure.Com LLC or the third party"*, and Lyon's reply was *"Maybe we
should just remove it. We will think about this more."* It has not been removed. **We do not rely on
that dispute** — step 1 and step 2 hold whether or not §2 binds on download.

### 6.2 Ground two — the hot set is verge-asm's selection, not nmap's, and this is measurable

`safe-active-probing.md` §2.3 describes the construction. Computed from that section's own text:

| Component | Count |
|---|---|
| nmap top-100 TCP, as reproduced in §2.1 | 100 |
| Project-chosen deletions (`1025–1029`, `49152–49157`, `2717`, `5101`, `5190`, `6646`, `3986`, `5051`, `5009`, `1755`) | **−19** |
| Retained from nmap's ranking | **81** |
| §2.3 supplement, as listed (49 entries, 5 already retained: `10000`, `9100`, `3389`, `5900`, `5800`) | **+44 net-new** |
| Frequency half as #4 specified it | **125** |
| ADR-0009's removal of `161/tcp` and `623/tcp` | **123** |

*(#4 §2.3 describes the result as "roughly 140". That figure is not reconciled here and is out of
scope under the Framing's second constraint — nothing in this note turns on which number is right,
because the ratio is what matters and it is the same either way.)*

**Roughly a third of the set is ports nmap's ranking does not support at any size, and the rest is
what survived nineteen deliberate rejections.** #71 §5.1 measured the supplement's standing in the
file exactly: the 140th-ranked TCP port sits at `0.002129`, about **28×** the filler value that
`6379` (Redis) and `2375` (Docker) carry, and `10250` (kubelet) and `9042` (Cassandra) are **absent
from the file entirely**. Those ports are in `verge-core` because #4 §2.3 chose them on a
*signal-mapping* rule — each maps to a named v1 signal — and for no other reason.

So the selection is not nmap's with edits. It is a different selection, made on a different rule,
that used nmap's ranking as one input among several and then overrode it 63 times (19 deletions,
44 additions). Under §8's authority that matters, and §8 shows it would matter even if the overlap
were total.

### 6.3 What the repo actually holds today, named rather than glossed

Honesty requires naming this: `safe-active-probing.md` §2.1 **reproduces nmap's `-F` top-100 list
verbatim as a code block**, together with 20 open-frequency values to three decimal places. That is
the one place a portion of `nmap-services`' content sits inside an AGPL-3.0 tree.

**It is fine, on §8's reasoning, and it is worth stating why rather than assuming it.** The
integers are facts. The frequencies are measurements of facts. The arrangement is descending
numerical order. All three are outside copyright under *Feist* and 17 U.S.C. §102(b). And the
quotation is in a research note documenting a retrieval — the canonical fair-use posture, which §3's
own closing paragraph expressly declines to override (*"To the extent that copyright doctrines such
as fair use allow their practices …"*). No change is required.

**Verdict on limb 1: not a derivative work. No NPSL obligation attaches. Nothing changes.**

---

## 7. Limb 2 — the weekly top-1000 tier is materially different, and it resolves differently

`safe-active-probing.md` §2.4 specifies the warm tier as **nmap top-1000 TCP**. This is a set
defined **by reference to nmap's own ranking**, and it cannot be evaluated without the file. The
ticket was right that this is a stronger dependence. It is stronger in a specific way that the
licence analysis is sensitive to, and the answer depends on which of three builds is chosen.

### 7.1 Build (i) — ship the 1,000 integers as a list file

This ships no nmap file and reads no nmap file, so §3's `Reads or includes` bullet does not fire, and
§6.1's steps 1 and 2 apply unchanged. **The licence does not reach it.**

But §6.2's ground is **gone**. There is no project judgement in this set. It is 1,000 of nmap's
1,000, in nmap's order, by nmap's rule. If any part of `nmap-services` is protectable, it is
*nmap's selection* — and this build reproduces that selection whole. *Feist* is explicit that this
is the one thing a subsequent compiler may not do:

> a subsequent compiler remains free to use the facts contained in another's publication to aid in
> preparing a competing work, **so long as the competing work does not feature the same selection and
> arrangement**.

So limb 2 build (i) rests on **one** ground where limb 1 rests on two, and that ground is §8's — the
merits question of whether *take the top 1,000 by a measured value* is an original selection. §8
answers no, with authority. But §8's answer is a prediction about how a court would characterise
someone else's work, where §6.1's answer is a reading of a sentence the licensor wrote. **The second
is a much better place to stand than the first**, and that asymmetry is the whole of this ticket's
finding.

### 7.2 Build (ii) — a build step that parses `nmap-services` and emits the top 1,000

**This is squarely inside §3, and it is the case §3's bullet was written for.**

A build script that opens `nmap-services`, parses the `open-frequency` column, sorts and takes the
first 1,000 is *software* that *reads Covered Software data files*. That is the bullet's plainest
reading. It is also precisely Lyon's stated intent — *"reading and parsing and interpreting the
Nmap data files directly rather than through Nmap execution"* — this time pointing **at** us rather
than away.

And v0.95's carve-out does not save it. The carve-out exempts *"software which does not require the
rights granted herein."* A build that reproduces the file into our build environment in order to
derive an output from it is exercising a reproduction right. We would be an accepting party, and
then the distribution clause binds by its own terms: distribution *"must be under the terms of this
license (including this Main License Body and GPL), with no additional conditions or
restrictions."* §9 shows what that costs.

### 7.3 Build (iii) — fetch the file at install or first run

Worse than (ii) on three independent counts, and not otherwise analysed because it is refused
anyway:

- It is (ii) with the reading moved into the operator's install, so the NPSL analysis is unchanged
  and the acting party is now our software running on their machine.
- It puts a third party in the default path, which
  [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)'s release-coupling test and
  [#27](https://github.com/winniel123/verge-asm/issues/27)'s framing both weigh against.
- A refreshed ranking newly admits ports to probe scope, which is an **aperture widening** under
  [ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md) with no release boundary to hang it on —
  exactly the machinery #27 declined to build for CAIDA.

### 7.4 What the dependence is *worth*, stated separately from whether it is legal

The ticket required these be kept apart. They are.

**#71 §5.1's filler finding does not reach inside the top 1,000, and saying so is the honest
reading.** The `0.000076` plateau spans ranks **1442–3410**. The top-1000 boundary sits *above* it,
so the warm tier's 1,000 members are drawn from the genuinely measured 2008 head, not from the 2016
IANA name import. The tier is **stale**, not **fabricated**, and the distinction is worth making
because it is the strongest form of the case *for* keeping it.

**And it does not survive the next step.** The frequency half's job, per the ticket, is *catching a
listener no signal names*. Measured by #71 §5.1 and #4 §2.2, the listeners that job is about are
**all outside the top 1,000**: Redis, Docker, MongoDB, memcached, ZooKeeper, etcd, the Kubernetes
API, kubelet, Cassandra, ActiveMQ, CouchDB, Kibana — every one of them ranks below 1,441 or is
absent from the file. So the ~900 ports the warm tier adds beyond the hot set are **2008's long
tail**: real measurement, of a distribution that predates every service the product was built to
notice.

The dependence is on eighteen-year-old data for a set whose contribution to the frequency half's
stated job is, on the evidence retrieved, **not measurable and plausibly zero**. That is a value
statement, made independently of the licence, and §11 prices it.

**Verdict on limb 2: build (ii) is inside §3 and is refused. Build (iii) is refused on three
grounds. Build (i) is probably clean and stands on one ground rather than two — and §11 shows we do
not need to find out.**

---

## 8. Is a list of port numbers copyrightable at all? The integers never. The selection, only if it is original — and a top-N is not

This is where the answer mostly lives, and it is reasoned from primary authority rather than
assumed in either direction.

### 8.1 The statute puts the facts outside, twice

17 U.S.C. §102(b), verbatim:

> **In no case does copyright protection for an original work of authorship extend to any idea,
> procedure, process, system, method of operation, concept, principle, or discovery, regardless of
> the form in which it is described, explained, illustrated, or embodied in such work.**

17 U.S.C. §101's definition, which is the only route by which a list of facts gets in at all:

> A **"compilation"** is a work formed by the collection and assembling of preexisting materials or
> of data that are **selected, coordinated, or arranged in such a way that the resulting work as a
> whole constitutes an original work of authorship**.

### 8.2 *Feist* is the controlling reading and it is directly on this shape

*Feist Publications, Inc. v. Rural Telephone Service Co.*, 499 U.S. 340 (1991). The facts are the
closest available analogue: a defendant took 1,309 entries out of a plaintiff's directory of facts.
Retrieved from Cornell LII:

> The most fundamental axiom of copyright law is that "[n]o author may copyright his ideas or the
> facts he narrates."

> **Facts are never original**, so the compilation author can claim originality, if at all, only in
> the way the facts are presented. To that end, the statute dictates that the principal focus should
> be on whether the selection, coordination, and arrangement are sufficiently original to merit
> protection.

> **Not every selection, coordination, or arrangement will pass muster.** … the facts must be
> selected, coordinated, or arranged "in such a way" as to render the work as a whole original. This
> implies that some "ways" will trigger copyright, but that others will not.

> This inevitably means that **the copyright in a factual compilation is thin**. Notwithstanding a
> valid copyright, a subsequent compiler remains free to use the facts contained in another's
> publication to aid in preparing a competing work, so long as the competing work does not feature
> the same selection and arrangement.

And the doctrine the Court killed is the one nmap's position would need:

> Known alternatively as **"sweat of the brow" or "industrious collection,"** the underlying notion
> was that copyright was a reward for the hard work that went into compiling facts. … The "sweat of
> the brow" doctrine had numerous flaws, the most glaring being that it extended copyright
> protection in a compilation beyond selection and arrangement — the compiler's original
> contributions — **to the facts themselves.**

**This is exactly nmap's claim's weakest joint.** What is valuable in `nmap-services` is the
industrious collection: *"scanning tens of millions of Internet IP addresses"* (nmap book,
`performance-port-selection.html`, cited at `safe-active-probing.md` §2.1). *Feist* holds that the
effort is precisely what copyright does **not** reward.

### 8.3 The Copyright Office's own practice says a mechanical top-N is not registrable

Compendium of U.S. Copyright Office Practices, 3rd ed. (01/28/2021), **§312.2**:

> For example, the Office **generally will not register a compilation consisting of all the elements
> from a particular set of data, because the selection is standard or obvious.**

and among its enumerated factors:

> * Is the selection exhaustive (e.g., a parts catalog containing standard information for all of the
>   parts sold by a particular company)?
> * **Is the coordination or arrangement obvious (e.g., is the information listed in alphabetical,
>   numerical, or chronological order)?**

**§313.3(C), *Facts*:**

> Facts are not copyrightable and cannot be registered with the U.S. Copyright Office. "No one may
> claim originality as to facts … because facts do not owe their origin to an act of authorship." …
> **A person who finds and records a particular fact does not create that fact; he or she merely
> discovers its existence.** … "[This] is true of all facts — scientific, historical, biographical,
> and news of the day."

And **§313.4(D)**, quoting 37 C.F.R. §202.1(d) — retrieved separately and confirmed verbatim:

> Works consisting entirely of information that is common property containing no original
> authorship, such as, for example: Standard calendars, height and weight charts, tape measures and
> rulers, schedules of sporting events, and **lists or tables taken from public documents or other
> common sources.**

### 8.4 Applying the distinction to what verge-asm actually does

The distinction the ticket asked for is **the integers** versus **the selection**. Held apart:

| Object | Status | Why |
|---|---|---|
| The port numbers themselves (`80`, `443`, `6379`) | **Never protectable** | Facts. §102(b); Compendium §313.3(C) |
| The open-frequency values (`0.484`, `0.000076`) | **Never protectable** | Measurements of facts. §313.3(C) extends to *"theories, predictions, or conclusions that are asserted to be facts … even if the assertion of fact is erroneous"* |
| The *format* of `nmap-services` — four whitespace-separated columns | Arguably thin protection, **irrelevant** | Nothing of ours reproduces the format |
| **nmap's top-N-by-frequency selection** | **Very likely unprotectable, and the Office would not register it** | It is exhaustive over a threshold, arranged in numerical order, and generated by a rule. §102(b) puts *"procedure, process, system, method of operation"* outside; *take the N largest values in a column* is a method |
| **nmap's selection of which 27,483 rows exist at all** | The strongest candidate on nmap's side, and it is **not what we take** | And per §4, a large part of it is IANA's CC0 selection rather than nmap's |
| **verge-asm's hot set** | **Its own selection of unprotectable facts** | 19 deletions and 44 additions against nmap's ranking, on a signal-mapping rule of the project's own (§6.2). *Feist*'s test — *"does not feature the same selection and arrangement"* — is met with room to spare |
| **verge-asm's warm tier as specified** | **Identical to nmap's selection** | The one place *Feist*'s safe harbour is not available, and where we would be relying on nmap's selection being unoriginal rather than on ours being different (§7.1) |

### 8.5 The EU *sui generis* right — retrieved, and it does not reach `nmap-services`

Directive 96/9/EC, from EUR-Lex. **Article 7(1)** is the right that would otherwise be the real
hazard, because it protects investment where copyright refuses to:

> Member States shall provide for a right for the maker of a database which shows that there has
> been qualitatively and/or quantitatively **a substantial investment in either the obtaining,
> verification or presentation of the contents** to prevent extraction and/or re-utilization of the
> whole or of a substantial part …

and **Article 7(4)** makes it independent of copyrightability:

> The right … shall apply **irrespective of the eligibility of that database for protection by
> copyright** or by other rights.

Scanning tens of millions of addresses is exactly the substantial investment Article 7(1) was
written to reward, and *Feist*'s rejection of sweat-of-the-brow does not travel to the EU. **But
Article 11 closes it:**

> **Beneficiaries of protection under the sui generis right**
>
> 1. The right provided for in Article 7 shall apply to database **whose makers or rightholders are
>    nationals of a Member State** or who have their habitual residence in the territory of the
>    Community.
> 2. Paragraph 1 shall also apply to companies and firms **formed in accordance with the law of a
>    Member State** and having their registered office, central administration or principal place of
>    business within the Community …
> 3. Agreements extending the right … to databases made in **third countries** … shall be concluded
>    by the Council acting on a proposal from the Commission.

**Nmap Software LLC / Insecure.Com LLC is a US entity**, and NPSL §9 confirms it: *"This License is
governed by the laws of the State of Washington."* No Council agreement extending the right to US
databases has ever been concluded. The *sui generis* right therefore **does not subsist** in
`nmap-services`. Article 3(2) disposes of the copyright half in the same direction: *"The copyright
protection of databases … shall not extend to their contents."*

*(Article 10(3)'s rolling-term rule — a substantial change from accumulated additions restarts the
15-year clock — would have been the interesting question for a continuously-updated file. It is not
reached, because Article 11 fails first.)*

---

## 9. AGPL-3.0 compatibility — no, and there is no upgrade path

This matters only if a derivative-work trigger fires. §6 and §7 say it need not. It is settled
anyway, because the ticket's real question is *what would it cost if we were wrong*, and the answer
is *the project licence*, which is the one thing that may not move.

### 9.1 Nmap's own first-party statement

`nmap.org/book/man-legal.html`, *Legal Notices*, retrieved 2026-08-14:

> **Even though the NPSL is based on GPLv2, it contains different provisions and is not directly
> compatible. It is incompatible with some other open source licenses as well.** In some cases we
> can relicense portions of Nmap or grant special permissions to use it in other open source
> software. Please contact fyodor@nmap.org with any such requests.

And Lyon in `#2199`, 2020-12-07:

> **And NPSL code cannot be integrated into GPLv2 programs. They aren't compatible licenses.** … The
> license also clearly states that it is not GPLv2 compatible.

### 9.2 The FSF's own statements, both directions

`gnu.org/licenses/license-list.en.html`, the **AGPLv3** entry:

> **Please note that the GNU AGPL is not compatible with GPLv2.** It is also technically not
> compatible with GPLv3 in a strict sense: you cannot take code released under the GNU AGPL and
> convey or modify it however you like under the terms of GPLv3, or vice versa. However, you are
> allowed to combine separate modules or source files released under both of those licenses in a
> single project …

and the **GPLv2** entry:

> **Please note that GPLv2 is, by itself, not compatible with GPLv3.** However, most software
> released under GPLv2 allows you to use the terms of later versions of the GPL as well. When this
> is the case, you can use the code under GPLv3 to make the desired combination.

**The escape hatch in that second quotation is closed here.** The FSF's route out of GPLv2 is the
*"or any later version"* permission. NPSL §1 defines `"GPL"` as *"the GNU General Public License
Version **2**, as published by the Free Software Foundation and provided in Exhibit A"*, full stop.
There is no later-version option anywhere in the retrieved 29,575 bytes. **The path from NPSL to
AGPL-3.0 does not exist.**

### 9.3 The mechanism, quoted rather than asserted

GPLv2 **§6**, quoted from **the NPSL's own Exhibit A** (retrieved `LICENSE`, lines 486–492) — which
is the strongest form of the citation, because it is the licensor's own copy:

> 6. Each time you redistribute the Program (or any work based on the Program), the recipient
> automatically receives a license from the original licensor to copy, distribute or modify the
> Program subject to these terms and conditions. **You may not impose any further restrictions on
> the recipients' exercise of the rights granted herein.**

And NPSL §2, which imposes exactly such further restrictions and says so:

> Where the terms in this Main License Body conflict in any way with the GPL, **the Main License
> Body terms shall take precedence. These additional terms mean that You may not distribute Covered
> Software or Derivative Works under plain GPL terms without special permission from Licensor.**

That is the whole mechanism. NPSL = GPLv2 + added terms. GPLv2 §6 forbids added terms. Therefore
NPSL is not GPLv2 and cannot be combined with it. AGPL-3.0 is a *third* licence, incompatible with
GPLv2 in the FSF's own words and reachable from NPSL by no route at all.

### 9.4 What it would cost if a trigger fired — so the flag has a price on it

§3's requirement is distribution *"under the terms of this license (including this Main License Body
and GPL), **with no additional conditions or restrictions**."* AGPL-3.0 is, relative to the NPSL, a
set of additional conditions — §13's network-use term most obviously. So a triggered derivative
work could not ship under AGPL-3.0.

The available responses would be:

- **relicense** (refused by the map, and this note does not reopen it),
- **obtain a waiver** (§10.3 — refused, and #27 is why), or
- **remove the dependence** (§11).

Only the third is open. That is why §7's build (ii) is refused rather than argued about.

**Two things are also true and are worth recording, because they bear on how much benefit of the
doubt this licence earns.** NPSL 0.95 is not recognised as free or open source by the parties who
review such things: Fedora concluded it *"not-allowed"* (`#2199`, 2024-07-05, Richard Fontana,
citing the Fedora legal tracker), Gentoo licence-masks it, and Debian, Guix, NixOS and Parabola all
have open objections in the same thread. The two clauses Fontana names as the problems are
*"the 'External Deployment' badgeware-ish provision and the section redefining 'Derivative
Works'"* — the second being §3. **A clause that four distributions have refused to accept as
free is not a clause to build a dependency behind**, independently of who is right about it.

---

## 10. Is this #27's shape? Argued, because the ticket says it points the same way

[#27](https://github.com/winniel123/verge-asm/issues/27) killed bundling CAIDA on a rule this repo
now applies by name: **redistribution is a separate permission from use, and the party who needs it
is the project.** #71 §5.1 applied it a second time to Shodan and Censys. The ticket says
`nmap-services` may be the same shape with a stricter licence. It is not, on limb 1 — and the
reason is specific enough to be worth writing down rather than asserting.

### 10.1 Limb 1 is a different shape, and the difference is what is in the tree

#27's structure was: *third party's data, inside our image, conveyed to our users, under terms that
do not let us convey it.* Every element was present. CAIDA's terms bar it. Censys's bar it in the
words *"Under no circumstances may any Customer … incorporate any Censys Data into its own software
products or services that are distributed"*. Shodan's grant no data licence at all and demand a
third-party ownership assertion that cannot ride inside an AGPL-3.0 tree.

Limb 1 has **no** element present. There is no third party's data in the tree. There is nothing to
convey. The artefact is 123 integers that verge-asm chose, on verge-asm's rule, using nmap's
published ranking as one input the way any author uses a source. **#27's rule needs a permission we
require. Here we require none.** Applying #27 to limb 1 would be applying a rule about
redistribution to a case with no redistribution in it.

### 10.2 Limb 2 build (i) *is* the shape — and survives on different, thinner ground

Shipping nmap's 1,000-port selection as a list file is nmap's selection travelling inside our tree,
under terms that (if the selection is protectable) do not let it. That is #27's shape.

It survives — but note **why**, because the difference is the finding. CAIDA, Shodan and Censys were
decided on **terms we could read**: their contracts say no, in sentences, and the analysis stops.
Limb 2 build (i) is decided on **subject matter**: nmap's terms say no too, and the answer is that
there is nothing for the terms to attach to. §8 makes that call with authority behind it, and it is
still a call about someone else's work that only a court finally makes.

**A verdict that rests on the other side's rights not existing is weaker than one that rests on our
not needing them.** Limb 1 has the second. Limb 2 build (i) has only the first. §12 rules on that
gap rather than around it.

### 10.3 The waiver route — offered in writing, and refused, and #27 is exactly why

The retrieval turned up an option the ticket did not name. NPSL §0, **Preamble**:

> **Open source developers who wish to incorporate parts of Covered Software into free software with
> conflicting licenses may write Licensor to request a waiver of terms.**

and the annotated version, more specifically:

> **Open source authors: If your license is not compatible with this one, we are sometimes willing
> to grant permission to use portions of Nmap under a different license than this one (such as plain
> GPL or BSD).** Contact Gordon "Fyodor" Lyon for permission.

and *Legal Notices*: *"In some cases we can relicense portions of Nmap or grant special permissions
to use it in other open source software."*

**Refused, and on #27's own rule.** A waiver is granted **to the project**. AGPL-3.0 requires that
every recipient of verge-asm receives the same rights the project has, and that they may pass them
on. A permission granted to one licensee **does not travel**, so an AGPL-3.0 tree resting on it
would be conveying rights it does not have — which is #27's holding word for word: *redistribution
is a separate permission from use, and the party who needs it is the project.* Here the party who
needs it is **every downstream recipient**, and no waiver reaches them.

This is #27's rule arriving a third time, and it is worth logging as such: it now disposes of a
bundled dataset (#27), a live-readable ranking (#71 §5.1), and a first-party offer of individual
permission. **The rule generalises past data licensing into permissions of any kind.**

---

## 11. The options, priced

The frequency half's job is **catching a listener no signal names**. Every option is priced against
that, not against coverage in the abstract.

Given, from #71 §5.1 and not re-derived: **there is no free, redistributable replacement with
frequency data.** Shodan is readable and unvendorable. Censys bars incorporation in terms. Rapid7
Sonar bars bulk redistribution. scans.io is non-commercial. Shadowserver publishes no per-port
breakdown *(that last marked unconfirmed there and carried forward unconfirmed here)*.

### 11.1 Self-generate a ranking — **the losing option**

Our scan, our copyright, no licence question at all. It is the only option that would restore a
*current* frequency ranking, and #71 §5.1 already named it as the only AGPL-clean source that could
exist.

**It loses on capability, not on cost, and the distinction matters.** An open-port frequency
ranking is a measurement **of the internet**. verge-asm measures **the operator's own estate** —
that is the premise of the whole product, and `safe-active-probing.md` §1 states it as the framing
sentence: *"Authorisation is not the constraint here — the operator owns the targets."* To produce a
ranking, the project would have to scan tens of millions of addresses it does not own, continuously,
which:

- contradicts #4 §1's second named risk (*"being a permanent, self-inflicted source of noise"*)
  at a scale the section never contemplated.
- requires a vantage, an abuse-handling function and a legal posture the project does not have and
  has never proposed acquiring.
- would make verge-asm a **different product** that happens to also do ASM.

This is not a scoping problem to be deferred to v2. It is a refusal. Named here as the loser
because it is the option that looks best on the licence axis and is unavailable on every other one.

### 11.2 Collapse to the hot set plus the sensitive list — **the recommendation**

Retire the weekly top-1000 tier. Keep the hot tier (daily, `verge-core`) and the **cold** tier
(opt-in full-range 1–65535, monthly ceiling, plus the onboarding baseline) exactly as
`safe-active-probing.md` §2.4 specifies them.

**What it costs, priced against the three ADRs that price it:**

| Cost | Amount |
|---|---|
| Licence exposure removed | **All of it.** No build reads nmap's file; no nmap selection ships |
| Drift manufactured | **None.** ADR-0007 and ADR-0009: revising the frequency half manufactures no drift — a batch whose recorded scope excludes a port does not touch that timeline |
| Version bump | **None.** ADR-0009: the version-bump-and-aperture-widening cost belongs to the **sensitive** half. This is the cheap half by construction |
| Aperture | A **narrowing** on the frequency half. Pre-release this is **vacuous** — ADR-0009 is explicit that it is vacuous rather than waived, because `revealed` is a property of timelines and before the first install there are none |
| Coverage | The ~900 ports beyond the hot set move from **7-day** latency to the cold tier's **30-day** — **and the cold tier is opt-in**, so on default settings they are covered only by the onboarding baseline sweep. Stated plainly rather than as *"they do not stop being seen"*, which would not be true for an operator who never opts in |
| What is actually lost | Detection latency on 2008's long tail, and on default settings the recurring half of it — for which §7.4 found no measured contribution to the frequency half's stated job |

**One ambiguity in #4 that this ruling inherits and does not resolve.** `safe-active-probing.md`
§2.4's table marks the cold tier **opt-in**, while its prose says the full-range sweep *"should also
run **once at target onboarding**, so the operator gets a complete baseline immediately"* without
repeating the qualifier. Whether the onboarding sweep is itself opt-in decides whether a
default-settings install ever sees those ~900 ports at all. **It is #4's question, not this note's**,
and it does not change the ruling: the tier being retired is the *weekly* one, and its licence
exposure is the same either way.

**Deciding this before v1 rather than after is the entire reason it is cheap.** ADR-0009 made the
same argument in the same words and this note is a second instance of it.

### 11.3 IANA's registry — licence-clean, and the wrong instrument

Retrieved, because the ticket named it the load-bearing fact if it became the replacement.

**What it is.** The *Service Name and Transport Protocol Port Number Registry*, `iana.org`, **last
updated 2026-08-11**, retrieved as CSV: **15,399 data rows**, **6,262** distinct numeric ports,
**6,121** distinct TCP ports. Columns: `Service Name, Port Number, Transport Protocol, Description,
Assignee, Contact, Registration Date, Modification Date, Reference, Service Code, Unauthorized Use
Reported, Assignment Notes`. **There is no frequency column and no ranking of any kind.**

**Its licence, stated explicitly.** `iana.org/help/licensing-terms` serves the *Joint Statement of
IANA and IETF Concerning Copyright Rights in the Protocol Registries*, dated **10 November 2021**:

> IANA and IETF intend that the Protocol Registries **may be freely used by any party for any
> purpose**. Both IANA and IETF believe that the Protocol Registries **consist primarily of factual
> information that is unlikely to be protectable as a matter of copyright law**. However, for
> additional clarity … both IANA and IETF affirm that **any applicable rights that they may have in
> the Protocol Registries are subject to the Creative Commons CC0 1.0 dedication** found at
> https://creativecommons.org/publicdomain/zero/1.0/legalcode.

**CC0 1.0. Unambiguously AGPL-3.0-clean, redistributable, bundlable, with no attribution
requirement and no downstream condition.** It is the cleanest third-party source the project has
retrieved to date, and it is worth recording as such: the licence answer for IANA is *yes* without
qualification, where CAIDA, Shodan, Censys, Rapid7 and now nmap have all been *no* or *only just*.
Note also that IANA's own statement makes §8's point for us in its own words.

**And it does not do the job.** A registration is a **claim about intent** — somebody asked for a
number and an expert reviewer granted it. It is not an observation that anything is listening. The
registry's own header note says so more bluntly than this note could:

> ASSIGNMENT OF A PORT NUMBER DOES NOT IN ANY WAY IMPLY AN ENDORSEMENT OF AN APPLICATION OR PRODUCT,
> AND **THE FACT THAT NETWORK TRAFFIC IS FLOWING TO OR FROM A REGISTERED PORT DOES NOT MEAN THAT IT
> IS "GOOD" TRAFFIC, NOR THAT IT NECESSARILY CORRESPONDS TO THE ASSIGNED SERVICE.**

So substituting it changes **what the tier is for**: from *the 1,000 ports most often found open* to
*the 6,121 ports anyone ever registered*. That is 6× the probe cost of the tier it replaces, ordered
by nothing, and it cannot be truncated because there is no column to truncate on. As a warm tier it
is unaffordable. As a filter it is uninformative.

**Kept, in the role it actually fits.** IANA's registry is the right source for **service naming** —
rendering *"port 5432"* as *"PostgreSQL"* — which is a job the product has independently, which
`nmap-services`' own header says it derives its names from, and which is CC0. **Adopt it for names.
Do not adopt it as a ranking.** That is a smaller conclusion than the ticket's option list
anticipated and it is the one the evidence supports.

### 11.4 The option nobody listed — keep the tier and ship the 1,000 integers

§7.1 says this is probably clean. It is priced here for completeness because a note that only
prices the options it likes is not pricing anything.

**In its favour:** it is the status quo, it costs nothing to keep, and §8's analysis says the
selection is unprotectable with the Copyright Office's own registration practice behind it.

**Against:**

- it is the **only** remaining place where verge-asm would rely on nmap's selection being
  unoriginal rather than on verge-asm's being different (§7.1),
- it is #27's shape surviving on subject-matter grounds rather than on terms (§10.2),
- §3 is a clause four Linux distributions have refused to accept as free (§9.4), and
- what it buys is **2008's long tail, at 7-day instead of 30-day latency, for a job the tail was
  measured not to serve** (§7.4).

**It loses on the ratio.** The uncertainty is small but real and permanent. The benefit is small and
measured. **When the uncertain option buys almost nothing, you do not buy it** — and here the
project's own cost model prices the alternative at exactly zero.

---

## 12. Ruling

1. **`nmap-services` is Covered Software under NPSL v0.95.** Its header asserts the licence over
   itself, §3's `such as` list is illustrative and not exhaustive, and no carve-out exists (§4, §5).
   The ticket's reported §3 quotes are verbatim-accurate.

2. **The ~~~140-port~~ hot set is not a derivative work** — *the set is 123 TCP ports, measured. See
   §6's note.* It fails §3's trigger on two independent
   grounds: NPSL v0.95 §3 by its own terms does not reach software that requires none of the rights
   it grants, and nothing verge-asm ships or builds reads any nmap file (§6.1). There is no
   protectable subject matter in a selection of port numbers to reach, least of all one that
   overrides its source 63 times (§6.2, §8). **`verge-core`'s frequency half, as ADR-0009 defines
   it, does not change.**

3. **The weekly top-1000 tier is retired and not replaced.** Its ports fall to the cold tier and the
   onboarding baseline — at 30-day rather than 7-day latency, and on default settings to the
   onboarding sweep alone, since the cold tier is opt-in (§11.2). The change is a frequency-half
   revision, which manufactures **no drift**, requires **no version bump**, and is an aperture
   narrowing that is **vacuous pre-release**. It is retired because it is the one place the project
   would depend on nmap's own selection, and it is affordable to retire because §7.4 measured what
   it contributes.

4. **A build step that parses `nmap-services` is refused outright** and should be recorded as
   refused rather than merely not-built, because it is the one construction that is squarely inside
   §3's plainest reading, inside the licensor's stated intent, and outside v0.95's carve-out
   (§7.2). A fetch at install or first run is refused on three grounds (§7.3).

5. **The NPSL is incompatible with AGPL-3.0 and there is no upgrade path** (§9). This does not bite,
   because rulings 2–4 mean no trigger fires. **The flag the map asks for is checked and lowered,
   not raised.**

6. **The waiver route is refused on #27's rule** (§10.3), even though nmap offers it in writing —
   a permission granted to the project cannot travel to AGPL-3.0 recipients.

7. **IANA's registry is adopted for service *naming*, on CC0 1.0, and rejected as a ranking**
   (§11.3). Self-generating a ranking is **the losing option**, refused on capability (§11.1).

8. **One amendment falls out and is not made here.** `safe-active-probing.md` §2.3 describes the hot
   set as *"nmap top-100 minus the ephemeral/obsolete tail … plus a modern-services supplement."*
   That sentence describes a **derivation from** nmap where §6.2 measures an **independent selection
   informed by** nmap. The set does not change. Only the sentence that says where it came from
   does, and getting it right is what makes ruling 2 legible to a reader who arrives at the file
   cold. **That file is not edited here** — it is #4's, and this is a footing repair on someone
   else's note.

---

## 13. Thin ground, and what only a lawyer settles

Stated rather than smoothed over, per the ticket's instruction.

**Thin — ruled anyway, on the conservative reading.**

- **Whether nmap's top-1000 selection is protectable.** §8 says no, with *Feist*, §102(b), the
  Compendium and 37 C.F.R. §202.1(d) behind it, and the Copyright Office's stated practice is
  directly against registering a numerically-ordered exhaustive top-N. But it is a **merits
  characterisation of somebody else's work**, and no retrieved authority decides *this* compilation.
  A court could find creative judgement in which 27,483 rows exist and in how the scans were
  designed. **Ruled conservatively: retire the tier rather than find out**, which costs nothing
  (§11.2). Had it cost something, this ruling would have gone the other way and said so.
- **Whether NPSL §2's download-is-acceptance sentence binds.** Contested by Gentoo in `#2199`, not
  removed by the licensor, not resolved. §6.1 is constructed **not to depend on it** in either
  direction. Flagged so a later session does not discover it and think the analysis missed it.
- **Whether §3's `Reads` bullet is enforceable at all.** Several distributions argue it is not, on
  GPLv2 §6 and on the DFSG's anti-contamination clause. This note **does not rely on it being
  unenforceable** — §6 wins on the clause as written. Flagged for the same reason.

**Only a lawyer settles these, and none of them is load-bearing after the rulings above.**

- Whether a design-time human reading of a data file, producing a written selection rule that a
  program later implements, is "software that reads" the file. §6.1's steps 1 and 2 are both
  needed only because this is genuinely open. §7.2 shows where the same question has an obvious
  answer, which is itself evidence that the boundary is real.
- Whether §3's added terms are severable from Exhibit A's GPLv2 under NPSL §13, and what NPSL
  §12's anti-*contra proferentem* clause does to the reading of a licence its own author calls
  *"rather terse"*.
- The `(C) Insecure.Com LLC` / `Licensor means Nmap Software LLC` mismatch in a file that has not
  been touched since 2025 (§4). Nothing here turns on it.

**What was not established, and is recorded as unestablished rather than absent (#25).**

- **No nmap mailing-list archive was searched** (§5). A data-file carve-out stated only on
  `nmap-dev` would have been missed.
- **`supreme.justia.com` returned HTTP 403.** *Feist* was retrieved from Cornell LII instead. The
  quoted passages match the syllabus and Part II–III structure of 499 U.S. 340. The LoC PDF of the
  U.S. Reports was fetched as a second copy (730,036 bytes) and **not diffed line-by-line against
  the LII text**. One notch below a direct first-party retrieval, flagged.
- **No EU or UK case law on the *sui generis* right was retrieved.** §8.5 rests on Articles 3, 7 and
  11 of the Directive as enacted, and Article 11 is dispositive on its face, so no case was needed
  — but the reading is unbacked by any tribunal.
- **No lawyer reviewed this.** It is a project decision on retrieved text, as the ticket required.

---

## 14. What this hands to other tickets

- **To [#4](https://github.com/winniel123/verge-asm/issues/4) / `safe-active-probing.md`.** Two
  edits and one open question, none of them made or answered here. §2.4's tier table loses its
  **Warm** row per ruling 3. §2.3's provenance sentence needs the ruling-8 correction. The **ports
  themselves do not move** and no aperture question arises from either edit. The open question is
  §11.2's: **is the onboarding full-range sweep opt-in, like the cold tier's monthly cadence, or
  unconditional?** Retiring the warm tier makes that the difference between a default install seeing
  the ~900 tail ports once and never, so it is now load-bearing where before it was not.
- **To [ADR-0009](../adr/0009-verge-core-is-a-union.md).** Nothing in the Decision moves.
  `verge-core = frequency-set ∪ sensitive-list` is untouched, the sensitive half is untouched, and
  the frequency half's *membership* is untouched. What moves is the **weekly tier**, which is a
  scheduling construct in #4 and not part of ADR-0009's union at all. Recorded here so a later
  reader does not go looking for an ADR amendment that is not needed.
- **To [#27](https://github.com/winniel123/verge-asm/issues/27)'s rule.** It now has a **third**
  application and a wider statement: *a permission granted to the project does not travel to
  AGPL-3.0 recipients* — which covers bundled datasets (#27), live-readable rankings (#71 §5.1) and
  **individually granted waivers** (§10.3). Whoever next writes down the project's third-party
  source bar should carry the general form, not the data-specific one.
- **To whatever ticket owns service naming.** IANA's Service Name and Transport Protocol Port Number
  Registry is **CC0 1.0** (§11.3), retrieved and quoted, and is the right source for rendering a
  service name against a port. It is the first third-party source this project has retrieved whose
  terms are clean without qualification. That is a finding worth spending.
- **To [#71](https://github.com/winniel123/verge-asm/issues/71) §5.4.** Closed. The reading it
  recorded as a question is answered: the quotes were accurate, the clause was already narrowed in
  2023, and the exposure it feared does not exist on the hot set. Its practical finding — *the only
  AGPL-clean source of open-port frequency data is one we generate ourselves* — is **confirmed and
  sharpened**: it is one we generate ourselves, and §11.1 rules that we will not, so there is no
  frequency ranking in verge-asm's future and the frequency half's job passes entirely to the
  hand-authored hot set.
- **The map's licence flag is checked and lowered.** No dependency, design choice or shipped
  artefact identified by this note forces a change to AGPL-3.0.

---

## Sources

**The licence, and nmap's own statements**
- [NPSL Version 0.95](https://raw.githubusercontent.com/nmap/nmap/master/LICENSE) — `nmap/nmap`, branch **`master`** (the default branch), HEAD `b403ddee` of 2026-08-13. `LICENSE` blob unchanged since `47919b8d`, 2023-01-11. 29,575 bytes, `sha256 9d9a9a763c0e6145172cfe7d8483e23b38ce60b6c79a82e4894242917bdae6d3`. Retrieved 2026-08-14
- [The same file served by nmap.org](https://svn.nmap.org/nmap/LICENSE) — byte-identical, verified by `diff`
- [Nmap Public Source License — Annotated Text, v0.95](https://nmap.org/npsl/npsl-annotated.html) · [NPSL landing page](https://nmap.org/npsl/)
- [Legal Notices, *Nmap Network Scanning*](https://nmap.org/book/man-legal.html) — *"not directly compatible"*
- [`nmap/nmap` issue #2199, *NPSL License Improvements*](https://github.com/nmap/nmap/issues/2199) — opened 2020-12-06, **still open**, 30 comments. Lyon on the `Reads` bullet (2020-12-07), on GPLv2 incompatibility (2020-12-07), and announcing v0.95 (2023-01-11). Fontana on Fedora's review (2024-07-05)
- [`LICENSE` commit history](https://github.com/nmap/nmap/commits/master/LICENSE) — commit `d0a8fb0f`, 2023-01-11, whose message states v0.95's purpose

**The data file**
- [`nmap-services`](https://raw.githubusercontent.com/nmap/nmap/master/nmap-services) — 998,268 bytes, 27,483 lines, 8,390 TCP rows, `$Id … 2008-08-26`. Retrieved 2026-08-14
- [`nmap-service-probes`](https://raw.githubusercontent.com/nmap/nmap/master/nmap-service-probes) · [`nmap-os-db`](https://raw.githubusercontent.com/nmap/nmap/master/nmap-os-db) · [`nmap-protocols`](https://raw.githubusercontent.com/nmap/nmap/master/nmap-protocols) — header comparators
- [`docs/DATAFILES.md`](https://raw.githubusercontent.com/nmap/nmap/master/docs/DATAFILES.md) · [`docs/licenses/`](https://github.com/nmap/nmap/tree/master/docs/licenses) · [`nmap.org/book/nmap-services.html`](https://nmap.org/book/nmap-services.html) — the §5 negative retrieval

**Licence compatibility**
- [FSF, *Various Licenses and Comments about Them*](https://www.gnu.org/licenses/license-list.en.html) — AGPLv3 and GPLv2 entries. Retrieved 2026-08-14
- GNU GPL version 2 §6 — quoted from **Exhibit A of the retrieved NPSL**, lines 486–492

**Copyright**
- [17 U.S.C. §102](https://www.law.cornell.edu/uscode/text/17/102) · [17 U.S.C. §101](https://www.law.cornell.edu/uscode/text/17/101) — Cornell LII
- [*Feist Publications, Inc. v. Rural Telephone Service Co.*, 499 U.S. 340 (1991)](https://www.law.cornell.edu/supremecourt/text/499/340) — Cornell LII (Justia HTTP 403, see §13)
- [Compendium of U.S. Copyright Office Practices, 3rd ed., Chapter 300](https://www.copyright.gov/comp3/chap300/ch300-copyrightable-authorship.pdf) — ed. 01/28/2021, §§312.2, 313.3(C), 313.4(D)
- [37 C.F.R. §202.1](https://www.law.cornell.edu/cfr/text/37/202.1) — Cornell LII
- [Directive 96/9/EC on the legal protection of databases](https://eur-lex.europa.eu/legal-content/EN/TXT/HTML/?uri=CELEX:31996L0009) — EUR-Lex, Articles 3, 7, 9, 10, 11

**The replacement candidate**
- [IANA, *Service Name and Transport Protocol Port Number Registry*](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml) · [CSV](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.csv) — last updated 2026-08-11. 15,399 rows, 6,121 distinct TCP ports. Retrieved 2026-08-14
- [IANA, *Licensing Terms*](https://www.iana.org/help/licensing-terms) — *Joint Statement of IANA and IETF Concerning Copyright Rights in the Protocol Registries*, 10 November 2021, **CC0 1.0**

**Held in this repository, cited not re-derived**
- [`project-authored-constants.md`](./project-authored-constants.md) §5.1 (the replacement landscape, the 2008 scan, the `0.000076` plateau across 1,969 TCP lines, ranks 1442–3410) and §5.4 (the question this note answers)
- [`safe-active-probing.md`](./safe-active-probing.md) §1, §2.1–§2.5 (the tiers, the hot set's construction, the top-100 list)
- [ADR-0009](../adr/0009-verge-core-is-a-union.md) (the union, the frequency half's zero drift cost, pre-release vacuity) · [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md) · [ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md)
- [#27](https://github.com/winniel123/verge-asm/issues/27) (redistribution is a separate permission from use) · [#25](https://github.com/winniel123/verge-asm/issues/25) · [#37](https://github.com/winniel123/verge-asm/issues/37) · [#66](https://github.com/winniel123/verge-asm/issues/66) · [#67](https://github.com/winniel123/verge-asm/issues/67)
