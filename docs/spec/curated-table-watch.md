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
> > >
> > > > **What keeps §1.1 from going stale as the register grows is ruled at [§5](#5-keeping-11-honest-as-the-register-grows)
> > > > below** ([#179](https://github.com/winniel123/verge-asm/issues/179)). Written here per
> > > > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s own
> > > > lesson — *a clause that names no successor is re-derived by the next session that needs one* — so a
> > > > future growth ticket landing on this box does not have to rediscover the obligation.

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
| ~~The **rexec / rlogin / rsh claim cell** @ the IANA Service Name and Transport Protocol Port Number Registry's own service descriptions~~ | ~~`512/tcp`, `513/tcp`, `514/tcp`~~ | **DISCHARGED — [#178](https://github.com/winniel123/verge-asm/issues/178), `sensitive-ports.md` §43.10.** The registry never clears §2.2's attestation gate alone; **not a register item.** `513/tcp` was already carried by RFC 1282 (rung 5). `512/tcp` and `514/tcp` are re-founded on NetBSD's `rexecd(8)` and FreeBSD's `rshd(8)` respectively (rung 4). ADR-0098 minted |

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

**No longer empty — the entry below is the form's first execution.** §2.1's five-part form has been
specified since this document's founding and nothing had run it:
[`sensitive-ports.md`](../research/sensitive-ports.md) §43.9 said so in terms — *"the entry form at its
§2.1 has never been executed against a real register"* — and that is the gap the entry below closes, on
the same footing [#133](https://github.com/winniel123/verge-asm/issues/133) closed the matching gap for
the gate: **a form whose first run has not happened is a design.**

**A note on who signs it, argued rather than assumed.**
[ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)'s *"who writes it"* row ties
authorship to **the release** — *"the same act that revises a curated table"*
([ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)). Read strictly, no such
act has occurred since the obligation took effect: v1 has not shipped, and every session since #125 that
has touched `sensitive-ports.md`'s register — #134, #151, #153 — is queue-walk or gate-walk work the
corpus is careful to mark as moving **no row, class, tier or coverage figure**, the curated table's own
content untouched. On that reading this entry is signed ahead of the event ADR-0078 names. It is signed
anyway, for the reason #133 ran the gate to completion before v1 shipped: an obligation that has never
run is unfalsifiable by construction, which is the worse failure. The signing act below is this ticket's
own resolution (#156, 2026-08-15) — a human-judgement reading of cited artefacts, never a mechanical
sweep, which is the distinction ADR-0078 actually polices (*"a machine may raise; only a release may
rule"*). **The stricter reading — that no entry may be signed before an actual curated-table-shipping
release — is not dismissed here; it is left as live, thin ground**, and is this entry's own successor
question (closing note, below).

#### Entry — 2026-08-15 (ticket #156)

**Part 1 — the order.** The register read against is
[`sensitive-ports.md`](../research/sensitive-ports.md) **§43.3**, as it stands after
[#151](https://github.com/winniel123/verge-asm/issues/151)'s RULED amendment (§48) — the live register
per this document's own §1 box, confirmed unmoved since [#155](https://github.com/winniel123/verge-asm/issues/155)'s
transcription by [#179](https://github.com/winniel123/verge-asm/issues/179)'s independent check. Not
re-enumerated here; §43.3 and this document's own §1.1 are the citation.

**Part 2 — the head.** Every cell below was read this release, in pairs under Shape 2 (§43.2 of
`sensitive-ports.md`): one artefact-opening discharges both cells of each pair.

| Item — `(cell, artefact, revision act)` | Rung | Found |
| --- | --- | --- |
| `10248/tcp`'s **footing** cell @ the config-API doc comment `healthzBindAddress: "127.0.0.1"` | 1 | Unchanged |
| `10248/tcp`'s **claim** cell @ the same doc comment | 1 | Unchanged |
| `623/udp`'s **footing** cell @ Dell's and HPE's default-value documentation | 2 | Unchanged |
| `623/udp`'s **claim** cell @ the same convergent owner corpus | 2 | Unchanged |

**Part 3 — the intensive bound.** One row per reading — a class boundary is a property of the reading,
not of the cell, so a Shape-2 pair states it once.

| Reading | Artefacts opened | Class boundary | Found |
| --- | --- | --- | --- |
| `10248/tcp`'s footing + claim | The `kubernetes/kubernetes` config-API source file carrying the `healthzBindAddress` doc comment, at the commit §27.5/§27.12 cite | This one doc comment, in this one source file, at this one commit — no other comment, file, or commit in the `kubernetes/kubernetes` tree | Unchanged. The comment still states the restricting default; §31.6's reading of the claim cell as a label, with no second ground, still holds |
| `623/udp`'s footing + claim | HPE's *iLO 7 Security Technology Brief* (PN 30-869C87FF-011, July 2026), Dell's *iDRAC10 Security Configuration Guide* (December 2024), and NEC's advisory NV21-002 — the three co-owner documents §28.8 cites | These three named documents, cited by §28.8, and no other BMC vendor's documentation | Unchanged, and the tension already on record is restated rather than closed — see the falsification note below |

**Part 4 — the extensive residue.** Every register item not read, named by rung. G11's mark is cited per
item, per §3's own rule.

*Rung 1*

| Item | Pairs | G11 |
| --- | --- | --- |
| `verge-core`'s frequency half @ `nmap-services` | not a port cell | Vacuous — no retrievable tag (§43.6 item 5) |
| `10255/tcp`'s claim cell @ `readOnlyPort`'s doc comment | `10255/tcp` | Vacuous — no retrievable tag (§43.6 item 5) |
| The rexec/rlogin/rsh claim cell @ the IANA registry's service descriptions | `512/tcp`, `513/tcp`, `514/tcp` | Vacuous — no retrievable tag (§43.6 item 5) |

*Rung 2*

| Item | Pairs | G11 |
| --- | --- | --- |
| The SMB footing cell @ one Microsoft page | `445/tcp`, `139/tcp`, `137/udp`, `138/udp` | Vacuous (§43.6 item 5) |
| The SMB claim cell @ the same page's perimeter directive | `445/tcp`, `139/tcp`, `137/udp`, `138/udp` | Vacuous (§43.6 item 5) |
| etcd's prohibition (footing) cell @ `THREAT_MODEL.md` | `2379/tcp`, `2380/tcp` | Vacuous (§43.6 item 5) |
| `10250/tcp`'s claim cell @ `ports-and-protocols.md`'s `Used By` | `10250/tcp` | Vacuous (§43.6 item 5) |
| `10250/tcp`'s footing cell @ `security-checklist.md` (RULED §151) | `10250/tcp` | Vacuous (§43.6 item 5) |
| `2375/tcp`'s and `2376/tcp`'s footing cell @ docs.docker.com (RULED §151) | `2375/tcp`, `2376/tcp` | Vacuous (§43.6 item 5) |
| `2376/tcp`'s claim cell @ the same page (RULED §151) | `2376/tcp` | Vacuous (§43.6 item 5) |
| The memcached footing cell @ the project wiki | `11211/tcp`, `11211/udp` | Vacuous (§43.6 item 5) |
| The memcached claim cell @ the wiki + shipped dispatch | `11211/tcp`, `11211/udp` | Vacuous (§43.6 item 5) |
| `25672/tcp`'s footing @ `rabbitmq.com/docs/networking` | `25672/tcp` | Vacuous (§43.6 item 5) |
| `25672/tcp`'s claim cell @ the same page's *Port Access* bullet | `25672/tcp` | Vacuous (§43.6 item 5) |
| `2375/tcp`'s claim cell @ Docker Engine security + shipped dispatch | `2375/tcp` | Vacuous (§43.6 item 5) |
| `2375/tcp`'s *why* cell @ the deprecated-features page | `2375/tcp` | Vacuous (§43.6 item 5) |
| `6379/tcp`'s claim cell @ redis.io + shipped bytes | `6379/tcp` | Vacuous (§43.6 item 5) |
| `9042/tcp`'s claim cell @ cassandra.apache.org + shipped dispatch | `9042/tcp` | Vacuous (§43.6 item 5) |
| `10249/tcp`'s claim cell @ Kubernetes' metrics documentation + `serveMetrics` | `10249/tcp` | Vacuous (§43.6 item 5) |
| The etcd claim cell @ `THREAT_MODEL.md` + shipped dispatch | `2379/tcp`, `2380/tcp` | Vacuous (§43.6 item 5) |
| The kube-scheduler / kube-controller-manager claim cell @ `Used By: Self` + kubeadm | `10259/tcp`, `10257/tcp` | Vacuous (§43.6 item 5) |
| `6000/tcp`'s claim cell @ `Xsecurity(7)` + `xhost(1)` | `6000/tcp` | Vacuous (§43.6 item 5) |

*Rung 3*

| Item | Pairs | G11 |
| --- | --- | --- |
| `5432/tcp`'s footing @ `postgresql.conf.sample` | `5432/tcp` | Not localized this entry — the composed-table run is RED overall (§40.6); this entry does not resolve which rung-3-through-5 cell(s) it names |
| `5984/tcp`'s footing @ CouchDB's `default.ini` | `5984/tcp` | Same as above |
| `5432/tcp`'s claim cell @ `postgresql.conf.sample` (RULED §151) | `5432/tcp` | Same as above |
| `5984/tcp`'s claim cell @ `default.ini` (RULED §151) | `5984/tcp` | Same as above |
| `9042/tcp`'s footing @ shipped `conf/cassandra.yaml` | `9042/tcp` | Same as above |
| `2181/tcp`'s claim cell @ the Administrator's Guide + shipped dispatch | `2181/tcp` | Same as above |
| `4369/tcp`'s claim cell @ epmd documentation + shipped `epmd_srv.c` | `4369/tcp` | Same as above |

*Rung 4*

| Item | Pairs | G11 |
| --- | --- | --- |
| `3306/tcp`'s footing @ Oracle's *Security Guidelines* | `3306/tcp` | Not localized this entry (as above) |
| `3306/tcp`'s claim cell @ the same page | `3306/tcp` | Same |
| `2049/tcp`'s footing @ `nfs(5)` | `2049/tcp` | Same |
| `2049/tcp`'s claim cell @ the same man page's SECURITY CONSIDERATIONS | `2049/tcp` | Same |
| `4369/tcp`'s footing @ `secure_coding.md` rule `DEP-001` | `4369/tcp` | Same |
| `873/tcp`'s footing cell @ `rsyncd.conf.5.md` (RULED §151) | `873/tcp` | Same |
| `873/tcp`'s claim cell @ the same man page (RULED §151) | `873/tcp` | Same |
| `2181/tcp`'s footing cell @ the Administrator's Guide (RULED §151, undetermined step 4) | `2181/tcp` | Same |

*Rung 5*

| Item | Pairs | G11 |
| --- | --- | --- |
| `certificate-expiring`'s fraction @ RFC 9773 §1 + issuer schedule | not a port cell | Not localized this entry (as above) |
| `23/tcp`'s claim cell @ RFC 4248 §3 | `23/tcp` | Same |
| `21/tcp`'s claim cell @ RFC 2577 §§5–6 | `21/tcp` | Same |
| `5900/tcp`'s claim cell @ RFC 6143 §9 | `5900/tcp` | Same |

**Part 5 — the gate's record.** The gate this release's reading was completed against:
[`sensitive-ports.md`](../research/sensitive-ports.md) **§40**'s baseline run (G1–G11 over the composed
table), extended by **G12** (§47) and **G13** (§49) — thirteen checks, per
[ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)'s table. State as last
confirmed at §43.7's closing table: **G5 RED→GREEN**, **G11 RED** (`2375/tcp` itself listed Current),
**G8's external half UNRUN**; **G12** and **G13** each population-complete and clean (no cell yet marked
`demoted-untagged` or `horizon-passed-unverified`). This release opened no artefact the gate had not
already fetched and edited no curated-table row, so it required no fresh gate run of its own.

**Falsification, run.**

- *Extensive test* (name one register item in neither part 2 nor part 4): **survives.** Every item at
  §43.3, across all five rungs, appears in exactly one of the two parts above — checked rung by rung
  against §43.3 and against this document's own §1.1 transcription, which #179 confirmed is not stale.
- *Intensive test* (name one artefact inside a stated class boundary the reading did not open):
  **survives, on the narrow reading of the class — and nearly did not.** The `10248/tcp` class (one
  file, one comment, one commit) has nothing else inside it. The `623/udp` class was first drafted
  broader — *"HPE's, Dell's and NEC's BMC-security documentation"* — and that draft **fails**: HPE's own
  *iLO 6 Security Technology Brief* (PN 30-3CC3279C-024), cited in this note's own Sources list, sits
  inside that class and was not opened by this reading. The class actually stated above is narrower —
  the three documents §28.8 names, and no other — which the iLO 6 Brief sits outside of, so it survives.
  **This is ADR-0040's own founding failure shape, caught before publication rather than after**: #68's
  defect was a class drawn too wide (the *specification* class, missing the deployment BCP); this entry
  repeats that near-miss in miniature, and the fix is the same in both places — narrow the class to what
  was actually opened, not to what sounds like the right authority.

**The predicted defect fired.** `sensitive-ports.md` §43.8 named it before this entry existed: *"the
register is now large enough that §42.9's predicted defect — the intensive bound being expensive enough
per item to depress the head — is the one to watch for."* It is confirmed rather than merely predicted
now. Writing two honest, narrow, falsifiable class boundaries — including catching and correcting the
`623/udp` over-broad draft above — was the majority of this entry's effort; doing the same for the
register's remaining rung-3-through-5 items, each resting on a versioned artefact whose class boundary
has never been drawn this precisely before, was not attempted this release. **The extensive residue is
cheap** — every unread item was already named and grounded by the queue walk, so naming it again costs a
citation. **The intensive bound is not cheap** — each one requires opening artefacts fresh and checking
the drawn class against the rest of the corpus for a document that would sit inside a looser version of
it. That asymmetry, not any reluctance to read, is why this entry's head is small against a register
much larger than it. Per the ticket that commissioned this entry: **the answer is not to drop the
bound.** ADR-0078 argued that trade in full and ruled against it, and the near-miss above is itself the
argument restated — a form without the intensive bound would have let the `623/udp` reading stand on the
wrong class, and nobody but a second reader guessing at what *"BMC documentation"* was supposed to mean
could have caught it.

**What this entry does not do.** It does not move a row, a class, a tier, or a coverage figure — no part
above authors a claim, and no cell here changes disposition. It does not quote a length anywhere above —
the residue is stated over members, per rung, exactly as §2.2 and §39.2 require. It does not resolve the
*"who signs it"* tension recorded above; that is named as fog for a successor ticket, not settled here by
the act of writing the entry.

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

---

## 5. Keeping §1.1 honest as the register grows

**[Ticket #179](https://github.com/winniel123/verge-asm/issues/179).** §1.1 is a transcription, not a
live register, and its own history box says so twice — ending *"§1.1 is a dated transcription of it,
not a second live register."* This section is not a re-litigation of whether that transcription was
correct to make; [#155](https://github.com/winniel123/verge-asm/issues/155) already ruled that, and its
two reasons for finally copying (the new cell kinds, and the G4 risk being answered by making the
snapshot explicit) stand. What #155 left open is what keeps §1.1 honest **afterwards**, since
[`sensitive-ports.md`](../research/sensitive-ports.md) §39.2 fixes the one property that decides how
expensive that upkeep is: **the register only ever grows.** [#135](https://github.com/winniel123/verge-asm/issues/135)'s
and [#134](https://github.com/winniel123/verge-asm/issues/134)'s filter tightenings are one-way — they
narrow what counts as a ground, so cells are **added** and never removed or reordered out. A copy of a
monotonic, append-only source has exactly one way to go stale: a growth lands in the source and nothing
lands in the copy. Three instruments were on the table.

**Option 1 — every register-growth ticket (in [#125](https://github.com/winniel123/verge-asm/issues/125)'s,
[#135](https://github.com/winniel123/verge-asm/issues/135)'s, [#151](https://github.com/winniel123/verge-asm/issues/151)'s
shape) carries an explicit re-sync step.** The ticket that adds a cell to
`sensitive-ports.md`'s register has, by construction, already done the harder half of the work: it has
read the new cell's ground, fixed its rung, and placed it in the tie-break order. Appending the same
triple to §1.1's matching rung table is a few minutes of transcription by a session that is already
holding both states — the register before the growth and the register after it. This is
[ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s own
argument, applied to a sibling failure shape: *"the pass that supersedes holds both states in hand … every
later reader holds only one. Deferring the edit does not avoid it; it relocates it to a session that must
first discover the discrepancy."* A growth ticket is exactly that pass, for the growth shape rather than
the supersession shape ADR-0058 was written for.

**Option 2 — a periodic or gated staleness check**, run on a cadence or as a standing item in the
gate's own checklist. This is the instrument the map already has a name and a cost history for, and it
loses on both. [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md) and
ADR-0058 both refuse a **standing watch** for a failure that fires when **we** move rather than when the
world does — ADR-0058's own classification test, applied here: `sensitive-ports.md` growing is an edit
this project makes, not an owner's artefact changing silently, so it is a **detectable defect**'s shape
and not a **curation trigger**'s. A trigger implies a cadence with an expected yield of zero between
growth tickets and a spike exactly when one lands — the same shape ADR-0058's Alternatives section
priced and declined: *"the watch here would be re-read every superseded document forever … the
discharge point is a change that is already in flight, so it costs nothing to attach it there and
everything to attach it to a clock."* A gated check fares no better: none of the gate's thirteen checks
(ADR-0057's own table) reaches this shape — G4 catches a superseded sentence standing unmarked, and
§1.1's rows are never superseded by growth, only **incomplete** relative to it — so a gated instrument
would mean widening ADR-0057's own gate table with a fourteenth check. That is a heavier intervention
than the failure warrants, given that the register's one-way growth (§39.2) already makes the manual
step in Option 1 as cheap as an automated one: there is never a removal or a reorder to reconcile, only
an append.

**Option 3 — revert [#155](https://github.com/winniel123/verge-asm/issues/155) and return §1.1 to a
cite-only pointer.** The cleanest instrument in one sense: a pointer cannot go stale, because it asserts
nothing about membership. It loses because the failure this ticket found is not the failure #155's
transcription was weighed against. #155's G4 risk was about copying a **provisional** register that
could fork on the very next walk ([#134](https://github.com/winniel123/verge-asm/issues/134)); that risk
was real and #155 answered it by making the copy an explicit, attributed snapshot rather than a second
live authority. The risk this ticket measures is different: not a fork on a still-moving filter, but an
**omission** on a since-stabilized, monotonically-growing one — and reverting to a pointer would trade a
copy that needs a cheap, bounded, append-only maintenance step for the exact cost #78 and
[ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md) already measured for
`sensitive-ports.md`'s own citation-only period: every reader who needs the register's cell kinds,
rungs and tie-break order has to resolve the pointer themselves, at `sensitive-ports.md`'s much larger
scale. Nothing about Option 3's failure mode is cheaper than Option 1's fix, so reverting a working
instrument to avoid a maintenance step that costs less than the revert itself is not a trade worth
making.

**Ruling: Option 1.** Every ticket that adds a cell to the register — in
[#125](https://github.com/winniel123/verge-asm/issues/125)'s, [#135](https://github.com/winniel123/verge-asm/issues/135)'s
or [#151](https://github.com/winniel123/verge-asm/issues/151)'s shape — appends the new member(s) to
§1.1's matching rung table, in the same commit, citing the ground exactly as it does in
`sensitive-ports.md`, and touching no existing row. Where the growth ticket only re-founds or
re-disposes a cell already in §1.1 (as [#151](https://github.com/winniel123/verge-asm/issues/151) did),
the same commit updates that row's Ground column rather than appending a new one. **This is not a gate
check and not a curation trigger** — no obligation is created to re-read §1.1 on any cadence, and its
absence is discovered the ordinary way any other unmarked growth would be: by the next session that
reads both documents together, per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
own detectable-defect classification. **This does not amend [ADR-0057](../adr/0057-a-watch-keys-on-the-act-that-would-falsify-a-cell.md)
or [ADR-0078](../adr/0078-a-residue-is-disclosed-by-the-act-that-leaves-it.md)** — neither rules on a
downstream copy's own upkeep, so nothing here reopens the queue's design or the residue's siting. See
[ADR-0100](../adr/0100-a-copy-of-a-monotonic-source-is-kept-honest-by-the-act-that-grows-it.md) for the
ruling stated as a standing rule, and this map's own Notes section for the standing-process consequence:
**a register-growth ticket is not complete until §1.1 carries its cells.**
