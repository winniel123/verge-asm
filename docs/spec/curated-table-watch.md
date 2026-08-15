# The release's account of its curated tables

- **Status:** Accepted — spec content for [#12](https://github.com/winniel123/verge-asm/issues/12)
- **Ruling:** [ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md), on
  [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
- **Ticket:** [#136 Where does the curated-table queue's bounded-residue disclosure live?](https://github.com/winniel123/verge-asm/issues/136)

[ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) ruled that a curated
table is revised by **the release**, and that the watch over the four curated tables is **two
instruments**: a **gate** over what is *closed*, which terminates and so runs to completion over the
table as edited, and a **queue** over what is *open*, which cannot terminate and so can only be
**sampled**. It gave the release one obligation towards the queue — a **bounded residue** disclosure
— and sited nothing. [ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)
sites it here.

**This document is the release's account, and it has two halves that are read together.** The gate's
result is *closed, complete and terminating*. The residue is *open, sampled and bounded*. Publishing
either alone invites the reader to take one for the whole assurance, and a green gate read alone is
**completeness arriving labelled as coverage**.

**It is a curator's document and reaches no interface.**
[ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §7 keeps this
material off every v1 screen on four grounds, its condition is unmet
([ADR-0016](../adr/0016-an-annotation-moves-a-message-never-a-number.md)), and ADR-0078 does not
reopen it. Nor does this content belong in an operator-facing release note: §7's fourth ground is
that the consumer is **not the operator**, and moving it to a different operator surface would defeat
§7 rather than honour it.

**No length is quoted anywhere in this document** — not the register's, not a head's, not a residue's.
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.2 bars the queue's count as an indicator,
its membership having changed five consecutive times without the count carrying the change.

---

## 1. The register — the queue, as it currently stands

**The register is a live absolute: the queue's current membership, in one place.** Its members are
`(cell, artefact, revision act)` triples over the cells of the four curated tables, ordered by the
five rungs of ADR-0057, tie-broken by how far the owner has moved past the tag the cell was read at,
in the owner's own release line.

> **The register is not yet transcribed here, and that is a state rather than an omission.**
> [`sensitive-ports.md`](../research/sensitive-ports.md) **§39.4** holds the queue, as built by
> [#125](https://github.com/winniel123/verge-asm/issues/125) and as tightened by
> [#135](https://github.com/winniel123/verge-asm/issues/135), and §39.9 marks it **provisional**: the
> per-cell independence test has been run only over cells already measured. The first **live**
> register is the output of [#134](https://github.com/winniel123/verge-asm/issues/134)'s per-cell
> walk over all 38 pairs plus the frequency half. **§39.4 is the register until then, and is cited
> rather than copied**: a second copy of a provisional list is exactly the shape gate check **G4**
> exists to catch, and a transcription made before #134 would be superseded on arrival.
>
> > **The *until then* has arrived — [#134](https://github.com/winniel123/verge-asm/issues/134) has
> > run**, and this clause is marked here per
> > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) rather
> > than at the site that supersedes it. **The live register is
> > [`sensitive-ports.md`](../research/sensitive-ports.md) §43.3.** §39.4 and §41.4 are no longer the
> > register; their items all stand on it with their grounds and rungs unchanged, and §39.9's
> > provisionality is discharged. **The register is still CITED and not COPIED here**, for this box's own
> > reason and for one more: it now spans cell kinds §39.4 never held — claim cells, *why* cells and
> > cells whose artefact coordinate holds a **set** of carriers
> > ([ADR-0076](../adr/0076-a-conjunctively-carried-cell-is-one-item-entered-at-the-rung-of-its-most-volatile-carrier.md)),
> > and a second copy would fork on the next walk exactly as G4 predicts. **No length of it appears here,
> > and §43 quotes none either.**
> >
> > **Two things the register's new shape owes an entry, and neither changes the form at §2.1.** An item
> > whose proposition is carried by a **set** states its **intensive bound per carrier opened**, which is
> > strictly more falsifiable than one bound per item. And **one reading can discharge several items** —
> > **[measured]** §43.2's shape 2, where one owner artefact carries both a row's claim cell and its
> > footing cell at `3306/tcp`, `25672/tcp`, `2049/tcp` and the `445`-group — so a head that names both
> > items and one opening is correct and is not double-counting. **[measured]** §43.6 also records that
> > **G11 is vacuous for the register's largest class**, its carriers having no retrievable tag, which is
> > what §3 below asks the gate to supply and cannot yet get.
> >
> > > **The *still CITED and not COPIED* clause above is superseded here, by**
> > > [#155](https://github.com/winniel123/verge-asm/issues/155), **per
> > > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).**
> > > ADR-0078's own Decision row (*"Where the register lives today"*) and its Consequences already
> > > commit to this step — *"the first live register is #134's per-cell walk, and it transcribes into
> > > the watch document at that point"* — and the box above did the citing half of that but deferred the
> > > copying half to its own ticket, which is #155. **§1.1 now holds the transcription.** The two reasons
> > > this box gave for not copying are answered rather than dropped: the new cell kinds (claim cells,
> > > *why* cells, set-carried cells) are carried into §1.1's triples exactly as §43.3 states them, and
> > > the G4 fork risk is why §1.1 is explicit that it is a snapshot *as transcribed*, cited to the
> > > sensitive-ports.md rows it copies, rather than a second live authority — a future growth of the
> > > register (§39.2's *movement is one-way*) still lands in `sensitive-ports.md` first and this snapshot
> > > goes stale until a later ticket re-transcribes it, exactly as a residue entry goes stale one release
> > > later. **This document remains the CITED pointer's owner; §1.1 is a dated transcription of it, not a
> > > second live register.**

**No figure for the register's size appears in this document, here or anywhere below.** #135 has
already moved it once and #134 is expected to move it again — and **the movement is one-way**, #135's
tightened filter narrowing what counts as a ground so that cells can only be **added**. Every
statement in this document is over the register's **members**, which is what §2.2 requires of an
entry and what §39.2 requires of anyone quoting the queue at all.

**The register spans four curated tables and is one order, not four.** **[measured]** §39.4 carries
two cells that are not port cells — `verge-core`'s frequency half at `nmap-services`, and
`certificate-expiring`'s fraction — and they are **interleaved by rung** among the port cells rather
than appended after them: the frequency half sits at **rung 1**, above every rung-2 and rung-3 port
item. That is why the residue disclosure below is one statement and not one per table: *how far down
it read* is a fact about the order, and no single table's document owns the order.

### 1.1 The register, transcribed — every member, by rung

**[Ticket #155](https://github.com/winniel123/verge-asm/issues/155).** This is a transcription, not a
re-derivation: every member below stands on the measurement cited in its **Ground** column, and this
section restates none of that measurement's reasoning. The **rung** and the **ordering within a rung**
are copied as `sensitive-ports.md` §43.3 states them — tie-broken, per [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
and §39.3, on how far the owner has moved past the tag the cell was read at, in the owner's own release
line — and that tie-break is **not re-run here**. §43.3 already carries [#151](https://github.com/winniel123/verge-asm/issues/151)'s
ruling ([ADR-0077](../adr/0077-a-second-ground-counts-only-where-it-would-have-carried-the-cells-proposition-alone.md)'s
Amendment, worked at §48) folded in at the cell, marked **RULED §151** below where it changed a cell's
disposition; nothing here is walked independently of that fold. A conjunctively-carried cell
([ADR-0076](../adr/0076-a-conjunctively-carried-cell-is-one-item-entered-at-the-rung-of-its-most-volatile-carrier.md))
is entered once, at its most volatile carrier's rung, and names every carrier. **No length appears
anywhere below** — not of a rung, not of the whole register — per §39.2 and ADR-0078's own bar.

**Rung 1 — one contributor, one commit, nothing rendered changes.**

| Item — `(cell, artefact, revision act)` | Pairs | Ground |
| --- | --- | --- |
| `verge-core`'s **frequency half** @ `nmap-services` — a third party publishing replacement frequency data, never announced to us | *not a port cell* | ADR-0038; §39.4 item 2 |
| `10248/tcp`'s **footing** @ the config-API doc comment `healthzBindAddress: "127.0.0.1"` | `10248/tcp` | §27.5, §27.12; §39.4 item 1 |
| `10248/tcp`'s **claim** cell @ the same doc comment | `10248/tcp` | §27.6, §31.6 — the port is a label; the comment's prose adds no second ground (Shape 2, §43.2) |
| `10255/tcp`'s **claim** cell @ `readOnlyPort`'s *"no authentication/authorization"* doc comment, `staging/src/k8s.io/kubelet/config/v1beta1/types.go` | `10255/tcp` | §41.7's flagged first cell; discharged sole-ground, and **undetermined at Step 1** — §43.3, §43.5 |
| The **rexec / rlogin / rsh claim cell** @ the IANA Service Name and Transport Protocol Port Number Registry's own service descriptions | `512/tcp`, `513/tcp`, `514/tcp` | §3.2; the attestation question over a registry description is routed, not decided — §43.6 |

**Rung 2 — a continuously-published page with no version pin, or a documentation branch tracking a
release line.**

| Item — `(cell, artefact, revision act)` | Pairs | Ground |
| --- | --- | --- |
| The **SMB footing cell** @ one Microsoft page | `445/tcp`, `139/tcp`, `137/udp`, `138/udp` | §16.7, §13.1 — no configuration artefact behind any of the four; §39.4 item 3 |
| The **SMB claim cell** @ the same page's perimeter directive, which numbers all four pairs | `445/tcp`, `139/tcp`, `137/udp`, `138/udp` | §33.3 (Shape 2) |
| `623/udp`'s **footing** @ Dell's and HPE's default-value documentation | `623/udp` | §13.1 as qualified by §28; §36.7 already de-attested once; §39.4 item 4 |
| `623/udp`'s **claim** cell @ the same convergent owner corpus | `623/udp` | §28.6 (Shape 2) |
| etcd's **prohibition (footing) cell** @ `THREAT_MODEL.md` | `2379/tcp`, `2380/tcp` | §16.3, §15.9; §39.4 item 5 |
| `10250/tcp`'s **claim** cell @ `ports-and-protocols.md`'s `Used By: Self, Control plane` | `10250/tcp` | §19.12; §41.4 item 9 |
| `10250/tcp`'s **footing** cell @ `security-checklist.md` | `10250/tcp` | **RULED §151** — one `kubernetes/website` `release-1.34` act reaches both this and the claim cell above; §48.3 |
| `2375/tcp`'s and `2376/tcp`'s **footing** cell @ docs.docker.com *Docker Engine security* | `2375/tcp`, `2376/tcp` | **RULED §151** — three pages of one continuously-published documentation set; §48.3 |
| `2376/tcp`'s **claim** cell @ the same page | `2376/tcp` | **RULED §151** — same act as the footing cell above; §48.3 |
| The **memcached footing cell** @ the project wiki's `ConfiguringServer` | `11211/tcp`, `11211/udp` | §13.2 — upstream ships permissive and silent; strike the wiki and nothing remains |
| The **memcached claim cell** @ the wiki sentence + `memcached/memcached` `1.6.45` dispatch | `11211/tcp`, `11211/udp` | Conjunctive — §10.1, §25.2; ADR-0076, entered at the wiki's rung |
| `25672/tcp`'s **footing** @ `rabbitmq.com/docs/networking` | `25672/tcp` | §32.6 (Shape 3) |
| `25672/tcp`'s **claim** cell @ the same page's *Port Access* bullet | `25672/tcp` | §18.5 (Shape 2) |
| `2375/tcp`'s **claim** cell @ Docker Engine security + `moby/moby` `docker-v29.7.2` dispatch | `2375/tcp` | Conjunctive — §10.1, §25.2; ADR-0076 |
| `2375/tcp`'s ***why*** cell @ docs.docker.com's *deprecated features* page | `2375/tcp` | The one *why* cell in §3 carrying a proposition its claim cell does not — §43.6 |
| `6379/tcp`'s **claim** cell @ redis.io *Security* + `redis` `8.10.0` shipped bytes | `6379/tcp` | Conjunctive — §10.1, §25.4/§25.5; ADR-0076 |
| `9042/tcp`'s **claim** cell @ cassandra.apache.org *security* + `apache/cassandra` `cassandra-5.0.9` dispatch | `9042/tcp` | Conjunctive — §10.1, §25.2; ADR-0076 |
| `10249/tcp`'s **claim** cell @ Kubernetes' metrics documentation + `serveMetrics` at `v1.34.0` | `10249/tcp` | Conjunctive — §27.2–§27.4; ADR-0076 |
| The **etcd claim cell** @ `THREAT_MODEL.md` + `etcd-io/etcd` `v3.7.1` dispatch | `2379/tcp`, `2380/tcp` | Conjunctive — §25.1–§25.3; ADR-0076 |
| The **kube-scheduler / kube-controller-manager claim cell** @ `ports-and-protocols.md`'s `Used By: Self` + kubeadm's `--bind-address=127.0.0.1` | `10259/tcp`, `10257/tcp` | §24.6; ADR-0076 |
| `6000/tcp`'s **claim** cell @ `Xsecurity(7)` + `xhost(1)` | `6000/tcp` | §3.4 — two X11 artefacts, neither yielding Claim 2 alone; both served at moving locations, rung 2 |

**Rung 3 — a shipped configuration default, announced by a version we can pin and diff.**

| Item — `(cell, artefact, revision act)` | Pairs | Ground |
| --- | --- | --- |
| `5432/tcp`'s **footing** @ `postgresql.conf.sample` | `5432/tcp` | §4.5; §39.4 item 6. **RULED §151 — unmoved**: already an item under ADR-0077's removal bar, confirmed independently by the act reading; §48.3 |
| `5984/tcp`'s **footing** @ CouchDB's `default.ini` | `5984/tcp` | §13.2; §39.4 item 7 (Shape 3) |
| `5432/tcp`'s **claim** cell @ `postgresql.conf.sample` | `5432/tcp` | **RULED §151** — one PostgreSQL-release act moves the manual and the sample together; §48.3 |
| `5984/tcp`'s **claim** cell @ `default.ini` | `5984/tcp` | **RULED §151** — CouchDB's documentation ships in the project's own tree; §48.3 |
| `9042/tcp`'s **footing** @ shipped `conf/cassandra.yaml` | `9042/tcp` | §12.7, §13.2 — the owner sentence is the configuration artefact |
| `2181/tcp`'s **claim** cell @ the Administrator's Guide + `apache/zookeeper` `release-3.9.5` dispatch | `2181/tcp` | Conjunctive — §10.1, §25.2; ADR-0076, shipped bytes the weakest link |
| `4369/tcp`'s **claim** cell @ Erlang/OTP's epmd documentation + shipped `epmd_srv.c` | `4369/tcp` | Conjunctive — §10.1 (read limb), §20.6; ADR-0076 |

**Rung 4 — issued prose in a versioned documentation set.**

| Item — `(cell, artefact, revision act)` | Pairs | Ground |
| --- | --- | --- |
| `3306/tcp`'s **footing** @ Oracle's *Security Guidelines*, MySQL Reference Manual §8.1.1 | `3306/tcp` | §13.2 — four Oracle packaging files, permissive and silent |
| `3306/tcp`'s **claim** cell @ the same page | `3306/tcp` | §30.3 (Shape 2) |
| `2049/tcp`'s **footing** @ `nfs(5)` (`utils/mount/nfs.man`, `nfs-utils-2.9.2`) | `2049/tcp` | §13.2 — `nfs.conf` ships every setting commented, permissive and silent |
| `2049/tcp`'s **claim** cell @ the same man page's SECURITY CONSIDERATIONS | `2049/tcp` | §3.3, §26.2 row 19 (Shape 2) |
| `4369/tcp`'s **footing** @ Erlang/OTP `secure_coding.md` rule `DEP-001` | `4369/tcp` | §20.7 — RabbitMQ's and CouchDB's sentences corroborate only; sole-ground on `DEP-001` |
| `873/tcp`'s **footing** cell @ `rsyncd.conf.5.md` | `873/tcp` | **RULED §151** — `rsyncd.conf.5.md` and `rsync.1.md` are shipped man pages of `RsyncProject/rsync` `v3.5.0`; one release moves both; §48.3 |
| `873/tcp`'s **claim** cell @ the same man page | `873/tcp` | **RULED §151** — same act as the footing cell above; §48.3 |
| `2181/tcp`'s **footing** cell @ the Administrator's Guide | `2181/tcp` | **RULED §151 — undetermined, step 4.** Whether one act reaches the version-pinned Guide and the unpinned `security.html` is not decidable from bytes this note holds; §48.3, §48.5 |

**Rung 5 — a specification: a new document with a new number, announced and never silently.**

| Item — `(cell, artefact, revision act)` | Pairs | Ground |
| --- | --- | --- |
| `certificate-expiring`'s **fraction** @ RFC 9773 §1 on form and the issuer's published lifetime schedule on value | *not a port cell* | ADR-0038; §39.4 item 8 |
| `23/tcp`'s **claim** cell @ RFC 4248 §3 | `23/tcp` | Sole-ground on the cleartext conjunct; **undetermined on Claim 2's successor conjunct** — §43.5 |
| `21/tcp`'s **claim** cell @ RFC 2577 §§5–6 | `21/tcp` | Same shape (Shape 3) |
| `5900/tcp`'s **claim** cell @ RFC 6143 §9 (with §7.2.1 and §7.2.2) | `5900/tcp` | Same shape; the vendor position points the other way, so no vendor artefact is available as a fallback |

**What is deliberately absent from this transcription.** Cells §41.2 disposed as *not an item* —
`sensitive-ports.md` §43.4's whole table, and §43.5's `10255/tcp` and `certificate-weak-key-or-signature`
rows — carry no ground here because they are not register members; naming them is that table's job, not
this one's. The *why*-cell and by-catch routing notes at §43.6 are cited above only where they changed a
member's ground, per the same rule.

---

## 2. The residue ledger — one dated entry per release, appended

**Every release writes one entry. A release that read nothing writes one too**, with an empty head
and the whole register as its residue, saying why the head is empty
([#47](https://github.com/winniel123/verge-asm/issues/47): a group that renders when populated
renders when empty and says why). **Entries are appended and never rewritten**, and they are ordered
by release tag, so a release that skips its entry leaves a **visible hole** rather than a previous
entry standing and reading true.

### 2.1 The entry form — five parts

| Part | What it carries | Falsified by |
| --- | --- | --- |
| **1 — the order** | The register state this release read against, cited at that state. **Never re-enumerated inside the entry** | Naming a register state the entry could not have read |
| **2 — the head** | Every item **read**, in the order read, each by its `(cell, artefact, revision act)` triple, with what the reading found: **unchanged** · **a question raised**, and the ticket it became · **the cell moved**, and the release act that moved it | Naming an item claimed read whose artefact was not fetched |
| **3 — the intensive bound** | For each item in part 2: the artefacts **opened**, and the **class boundary** of that opening — *this owner's issued documentation set at tag X and its release notes, and no other class* | Naming one artefact **inside** a stated class boundary that the reading did not open |
| **4 — the extensive residue** | Every item **not read**, **named** by its triple, each carrying its rung and the **G11** mark that would have promoted it | Naming one register item appearing in neither part 2 nor part 4 |
| **5 — the gate's record** | The gate run this release's edits were completed against, cited to §3 | — |

### 2.2 The two bounds, and why they are stated differently

**The extensive residue is named, member by member.** The register is **our own list** and is
enumerable, so a described boundary would be a count with the number filed off — unfalsifiable
without first guessing what we thought the register held. **Where the population is the project's own
and enumerable, a residue is named rather than described**
([ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md), sharpening
[ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md)).

**The intensive bound is described, per item, with a class boundary.** Reading **one** item means
opening **somebody else's** corpus, which does not terminate. *We read item 3* is unfalsifiable;
*we opened this owner's issued documentation at tag `v1.37.0` and its release notes, and no other
class* is falsified by naming one artefact inside that class. This is ADR-0040's form unchanged,
applied at the item rather than at the list — and **[measured]** ADR-0040's founding failure was a
class boundary drawn too narrow, [#68](https://github.com/winniel123/verge-asm/issues/68) having read
the **specification** class and missed the **deployment BCP** that carried the number.

### 2.3 What the entry may not contain

| Barred | Why |
| --- | --- |
| **Any length** — of the register, of the head, of the residue | §39.2. A count over the queue moved five times without carrying the change, and ranking a release by how far it got restores exactly the meaning that bar removes |
| **A fraction** — *read k of the register* | A manufactured figure nobody can attest ([ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md)), and a length in another coat |
| **A quantum, or a target** | ADR-0057 refused it. An entry with a one-item head is as compliant as one with a seven-item head; the discipline is that the head is **visible and dated**, not that it is large |
| **A verdict on a row** | A machine may **prepare** an entry from G11's marks; only the release may **sign** it, and only a release may move a row. The watch's output is a **question**, and the answer is a retrieval, which is a ticket |
| **A permanent caveat** | ADR-0032 §7. Every part of the entry is falsifiable by the test in its own row above, which is what *bounded* means and *permanent* does not |

### 2.4 The ledger

**Empty, and the reason is stated rather than left blank.** v1 has not shipped, so no release has
spent a reading budget and no entry is owed yet. The first entry is owed by the first release that
ships a curated table.

| Release | Entry |
| --- | --- |
| — | **None yet.** No release has occurred |

---

## 3. The gate ledger — reserved

**The gate is thirteen checks, G1–G13, run over the table as edited; an edit is complete only when the
gate is green over the post-edit state, and a red gate blocks the *edit*, never the release**
([ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md);
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.6). G12 and G13 were added in the 2026-08-15
merge of [#149](https://github.com/winniel123/verge-asm/issues/149) and
[#152](https://github.com/winniel123/verge-asm/issues/152)/[ADR-0099](../adr/0099-a-stated-horizon-is-a-second-comparand-a-tag-match-does-not-discharge.md);
ADR-0057's own table is the canonical list.

**This section's shape is [#133](https://github.com/winniel123/verge-asm/issues/133)'s and is
deliberately unspecified here.** The gate has never been run whole; #133 is running G1–G11 to
completion over the composed table for the first time and establishes the baseline every later edit
is measured against. ADR-0078 reserves the gate's record a place in this document — because the two
halves of a release's account are read together — and specifies nothing about its form.

**One thing the gate owes the residue ledger, whatever shape #133 gives it.** **G11** marks every
footing cell whose owner has moved past the tag the cell was read at. Those marks are the reason an
item is read or passed over, so **entry part 4 cites the G11 mark for every item it leaves unread**.
The gate **supplies inputs to the residue disclosure and never signs it**: an instrument defined by
termination cannot certify a statement about what does not terminate, and what it would emit if
forced is a **length**.

---

## 4. What this document is not

- **It is not the queue's reasoning.** That is
  [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) and
  [`sensitive-ports.md`](../research/sensitive-ports.md) §39.
- **It is not a curated table**, and nothing in it is asserted about the world. It records what the
  project **did**, which is why it needs no evidence standard under
  [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md).
- **It is not read by the product**, and the curator is not a subject in the model —
  [`CONTEXT.md`](../../CONTEXT.md) is not amended, per ADR-0057 and ADR-0078.
- **It is not a screen, and it is not a changelog.** ADR-0032 §7, on both counts.
