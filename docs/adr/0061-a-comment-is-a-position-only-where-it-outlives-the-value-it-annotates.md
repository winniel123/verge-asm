# ADR-0061: A comment is a position only where it outlives the value it annotates

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#100 Is a config-API doc comment a label or a position — and does it attest?](https://github.com/winniel123/verge-asm/issues/100)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md) limb 2 rules that *"a
directive is not a position; prose in a config file may be"*, and gives two poles:

> *"For security reasons, you should not expose this port to the internet"* is a **position**.
> *"# Listen for connections from the local system only"* is a **label**.

It then flags its own limb as the thinnest thing it says:

> "**Thin ground, flagged.** Limb 2's directive-versus-label line is the thinnest part. A comment that
> argues for its own directive sits between *"# Listen for connections from the local system only"* and
> *"you should not expose this port to the internet"*, and the only instrument offered for placing it is
> the existing position-versus-preference discrimination. **A case in that gap should be ticketed rather
> than decided by whoever meets it.**"

**Nothing has been written down since.** [#70](https://github.com/winniel123/verge-asm/issues/70)
applied the line three times across fourteen artefacts and reached no case in the gap.
[#76](https://github.com/winniel123/verge-asm/issues/76) §16.5 refused to promote `10255/tcp` on
*"the read-only port for the Kubelet to serve on with no authentication/authorization"*, calling it *"a
description of what the port serves, not a position on where it may be reached from"* — a verdict with
no test behind it. [#95](https://github.com/winniel123/verge-asm/issues/95) §27.6 refused to promote
`10248/tcp` on *"the port of the **localhost** healthz endpoint"*, citing #69's line and §16.5's
application of it, and §27.13 recorded what it had done:

> "**The label-versus-position line was applied and not argued.** … Whether *the localhost healthz
> endpoint* is a label or a position is a case in that gap, and this section has taken the conservative
> branch — the one that does not move a tier — rather than deciding it."

**Two sections have now reached the same conclusion by citation, and neither could state the rule it was
applying.** That is the position
[ADR-0042](./0042-a-squat-is-contested-where-the-other-convention-is-live.md) refused *"leave it
unwritten and decide case by case"* from, and the position
[ADR-0059](./0059-a-footing-tier-grades-evidential-distance-never-the-owners-conviction.md) was minted
from when §18.6 and §20.8 reconstructed a tier criterion in two different vocabularies.

**#95 also met the line in a third artefact class**, which neither ADR-0036 nor #70 contemplates: a
**published config-API doc comment** — a Go struct field's documentation in
`k8s.io/kubelet` `v0.34.0` `config/v1beta1/types.go` — rather than a comment in a shipped
configuration file. §16.5 had accepted that artefact for a **footing** on a row that was already
listed; §27.6 was the first time one was asked to carry **admission**. The class raises questions
ADR-0036 does not answer: whether the artefact has the same standing as shipped config bytes under the
*takes effect* test, which of the comment and the code it annotates the **costly act** test reads, and
whether the class changes the footing **tier**.

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) **§31**.

## Decision

> **A comment takes a position only where it outlives the value it annotates. Where its content is
> exhausted by that value, it is a label, and it attests only what the shipped-default form already
> attests.**

Four limbs.

1. **The survival test, and it is read off the artefact by substitution rather than judged.** Take the
   value the comment annotates — a directive in a configuration file, a documented default in a config
   API — and change it, within the value space the artefact itself enumerates. A comment that **remains
   true and operative** under that substitution is a **candidate position**: it constrains a
   configuration other than the one it sits above, so it says something the value does not. A comment
   that goes **false or moot** is a **label**: its whole content is entailed by the value, and reading it
   as a second attestation counts one act twice.
   **The test is *necessary* and not *sufficient*.** A surviving comment is not thereby a position; §2.2's
   second form still requires that what it says be a **position on the proposition the row asserts**, which
   is `sensitive-ports.md` §2.3's and §4.4's position-versus-preference discrimination, unchanged and now
   running second rather than alone.
2. **A published config-API doc comment is the same artefact class, one limb over — not a weaker
   class.** A type definition's doc comment annotates a **field and its documented default** rather than
   an operative directive, so limb 1 applies with the documented default substituted for the directive.
   Its **issuance** is not the question: a doc comment carried in a module release the owner publishes is
   issued under [ADR-0045](./0045-an-owners-documentation-is-what-it-has-issued.md) limb 1 exactly as a
   manual page is, and a comment fails, where it fails, on being a **label** and never on being unread.
3. **§2.2's third form has two limbs, they may be answered by two artefacts, and the costly act is paid
   only by the operative one.** A doc comment can satisfy *documented as its default*; it can **never**
   satisfy *takes effect*, because a type definition executes nothing. The defaulting code is the other
   limb and must be **retrieved**, not assumed from the comment. ADR-0036's costly-act reasoning reads
   the **code**: friction at first run is produced by the defaulting function, and a comment produces
   none — which is ADR-0036 limb 4's *"an act that takes effect nowhere was not taken"*, one artefact
   class over, in the direction ADR-0056 already carried it for executables. **Where the two disagree the
   third form is not satisfied at all:** the comment cannot supply the act and the code cannot supply the
   documentation.
4. **Artefact class does not grade a footing tier.** ADR-0059 limb 1 counts premises the reader supplies
   between the owner's **sentence** and the row's proposition; **the artefact carrying the sentence is not
   a premise.** A comment in a Go file, a comment in a YAML file and a paragraph on a documentation site
   sit at the same distance from the proposition or at different distances for reasons limb 1 already
   counts. Artefact class bears instead on **volatility** —
   [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's watch list, where a
   default changeable in one commit with no release note is a different exposure from one on a released
   documentation page — and that is a different column from the tier.

## Rationale

**The test reproduces every verdict the corpus already carries, which is the test ADR-0042 and ADR-0059
each set for themselves.** **[measured]** run over every comment in a configuration artefact this note
holds — nine of them, five projects, three artefact classes — limb 1 returns the recorded answer every
time, and **eight of the nine fail at step one**:

| Comment | Change the value | Limb 1 | The note's existing verdict |
|---|---|---|---|
| net-snmp `# Listen for connections from the local system only` | `agentAddress udp:0.0.0.0:161` → false | **Label** | ADR-0036 limb 2's own pole |
| Cassandra *"For security reasons, you should not expose this port to the internet. Firewall it if needed."* | `rpc_address: 0.0.0.0` → still true, still binding | **Candidate**, and §4.4 confirms it is a position on placement | ADR-0036 limb 2's other pole; `9042` prohibition tier |
| Redis *"it is not a good idea to expose the Redis instance directly to the internet"* | bind directive changed → still true | **Candidate**, position | `6379` prohibition tier |
| Redis *"IF YOU ARE SURE YOU WANT YOUR INSTANCE TO LISTEN TO ALL THE INTERFACES COMMENT OUT THE FOLLOWING LINE."* | line commented → moot | **Label** — a menu | never used as a footing |
| `exports(5)`'s `/pub *(ro,insecure,all_squash)` gloss — *"every host in the world"* | `*` narrowed → false | **Label** | §13.3's refused label |
| MongoDB *"Enter 0.0.0.0,::"* | value changed → moot | **Label** — a menu | #70's label |
| kube-proxy `metricsBindAddress`'s *"(Set to `0.0.0.0:10249` / `[::]:10249` to bind on all interfaces.)"* | value set → moot | **Label** — a menu, MongoDB's object exactly | §27.4 reads it as a **reachability frame** under [ADR-0054](./0054-a-claim-step-is-answered-only-by-evidence-about-that-step.md) limb 2, never as an attestation |
| kubelet `readOnlyPort`'s *"the read-only port for the Kubelet to serve on with no authentication/authorization"* | `readOnlyPort: 10255` → **still true** | **Candidate** — and refused at step two: it describes what the port serves, not where it may be reached from | §16.5's verdict, **re-founded at a different step** |
| kubelet `healthzPort`'s *"the port of the **localhost** healthz endpoint"* | `healthzBindAddress: "0.0.0.0"` → **false** | **Label** | §27.6's verdict, **re-founded at step one** |

**The two kubelet fields separate at two different steps, twelve lines apart in one struct in one
file.** That is the sharpest form of measurement this note recognises, and it is what makes limb 1's
*necessary and not sufficient* structure a finding rather than a hedge: a rule with one step would have
to give `readOnlyPort` and `healthzPort` the same answer, and the note has always given them different
ones.

**Limb 1 replaces a judgement with a substitution, which is ADR-0036's own move.** ADR-0036 limb 1
survives because *takes effect* is *"read off the file rather than judged"* — **[measured]** nine
artefacts, nine self-declarations. *Is this comment forceful enough to be a position?* is a judgement a
reviewer cannot be wrong about; *does this sentence still say something once the value beneath it
changes?* is answered by re-reading the sentence. `sensitive-ports.md` §15.7 recorded the standing
temptation — *"the word **live** invites the substitution and the next session will feel the same
pull"* — and ADR-0059 recorded it for the word *prohibition*. Here the word **position** invites reading
the line as a force ranking, which ADR-0059 limb 2 has already made inadmissible for the adjacent
column; limb 1 gives the label/position line the same kind of instrument that column now has.

**Limb 1 also explains the pole ADR-0036 asserted rather than restating it.** Cassandra's sentence sits
immediately above `rpc_address: localhost` and *argues for* it — the shape the map's fog patch names as
the hard middle. It is a position not because it is emphatic but because it **binds the operator who
overrides the value**: the sentence goes on being the owner's instruction after `localhost` becomes
`0.0.0.0`, where net-snmp's label simply stops describing anything. **The middle the patch feared is not
between the poles; it is the poles read at the wrong grain.**

**Limb 3 is the costly-act test applied where the artefact splits.** `sensitive-ports.md` §10.4.2 admits
a restricting default because a restriction *"buys friction at first run and the maintainer paid for it
anyway"*. In a shipped configuration file one artefact both states the value and enacts it, so the cost
and the documentation arrive together and nobody had to separate them. In a config API they are two
files: the comment is free — it compiles to nothing, no daemon reads it, no operator meets it at first
run — and the defaulting function is where the friction is bought. ADR-0036 refused an example config
because *"a file nobody's daemon reads produces no first run"* and ADR-0056 carried that into
executables; limb 3 carries it into the schema. **The asymmetry it forecloses is the dangerous one:** a
comment claiming a restriction the code does not perform would otherwise admit a row on an act nobody
took, which is §2.2's opening sentence — *the claim may not be asserted by us* — failing through the
owner's own typo.

**Limb 4 is ADR-0059 limb 2 read one column over.** *Mood, force, hedging and priority label are
inadmissible* because the tier measures **how much of the row's proposition the owner said**, not how the
owner said it. *Where the owner wrote it* is the same kind of fact as *how firmly the owner wrote it*: it
is about the utterance, not about the distance between the utterance and the proposition. **[measured]**
a class discount applied honestly does not stop at Go: `9042/tcp` Cassandra's prohibition-tier cell is a
comment in a YAML file inside a tarball and `5984/tcp` CouchDB's rests on a line in
`rel/overlay/etc/default.ini` — the artefact §12.5 calls *"the cleanest instance of §2.2's third form in
the note"*. A rule that demotes the cleanest instance of a form is measuring the wrong thing.

**Why this is an ADR and not only a note edit**, against a strong house norm of declining (§16.6, §20.9,
§23, §24.9 and #90 and #91 all declined one). Every declination passed the same test — *both general
rules this section applies were available*. Here the load-bearing rule was available **nowhere**:
ADR-0036 says so in its own *Thin ground*, the map has carried the gap as an open patch since #69, and
two sections have applied *label* as a conclusion with no criterion behind it. And it travels under
ADR-0032: any future curated table admitting an owner's prose inherits limb 1, and any table admitting a
published API surface inherits limbs 2 and 3.

## Consequences

- **No `(port, transport)` pair moves, no row moves, no class moves and no footing tier moves.**
  `sensitive-ports.md` stays at **41 pairs**, class totals `12 / 7 / 22`, tiers **prohibition 14 ·
  scoping 13 · weak 3 · outside-subject 11**, coverage **30 of 41**, §6.1's `28 + 8 + 5 = 41` and §4.6's
  20 exclusions untouched. [ADR-0009](./0009-verge-core-is-a-union.md)'s union is unchanged and
  [ADR-0008](./0008-derivation-versions-move-on-content.md) is **not** triggered — this ADR reads
  evidence and changes no reference data. *(Figures as of `main` at commit `c0881ae`, the composed
  post-merge state this ruling was walked against; three siblings were resolving concurrently.)*
- **`10248/tcp` stays in the weak footing tier**, and **[measured]** *"the port of the localhost healthz
  endpoint"* is a **label** under limb 1: set `healthzBindAddress: "0.0.0.0"` and the sentence is false.
  #95's conservative branch is **confirmed on a rule rather than on a citation**, and the
  `15 / 13 / 2` the ticket priced — `14 / 14 / 2` on the composed state — is not spent.
  [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's watch list stays at
  `5432/tcp`, `5984/tcp` and `10248/tcp`.
- **§16.5's `10255/tcp` ruling is re-founded rather than disturbed.** It fails at limb 1's **second**
  step, not its first: the sentence survives its value and is refused as a description of what the port
  serves. The row is in the prohibition tier on §18's category statement and its footing does not depend
  on the comment today.
- **ADR-0036's *Thin ground* flag is discharged**, and its limb 2 is unchanged: limb 1 here is the
  instrument limb 2 said it lacked, not a correction of it. ADR-0036 gains an amendment note pointing
  here.
- **ADR-0059 gains a *confirmed by use* note.** Limb 2's *the tier does not grade the utterance* is read
  one step over by limb 4, in a column limb 2 did not name.
- **A retrieval obligation is created and is already discharged for every row that has one.** Under limb
  3 a footing resting on a config-API doc comment must pair it with the defaulting code. **[measured]**
  the population is three — `10248/tcp` (§27.6), `10249/tcp` (§27.2) and `10255/tcp` (§16.5 with
  ADR-0036's #83 amendment) — and all three were paired at the time.
- **[#12](https://github.com/winniel123/verge-asm/issues/12) is not touched.** The spec carries the list,
  the claims and the containment arithmetic; none of them reads a footing or an artefact class.
- **`CONTEXT.md` is not edited**, on ADR-0036's, ADR-0045's, ADR-0048's, ADR-0056's and ADR-0059's
  precedent and for their reason: no term is minted, and the file is left alone while concurrent passes
  are running.
- **Where it is thin.** Limb 1's admitting half is **unexercised** — no comment in the corpus survives
  its value *and* takes a placement position except the two already-known positions, so
  *necessary-and-not-sufficient* has one instance on the sufficient side and none on the admitting one.
  Limb 3 is **derived rather than measured**: no config-API doc comment in the corpus disagrees with its
  defaulting code, so the disagreement rule is written for a case that has not happened.
  `sensitive-ports.md` §31.11 records both.

## Alternatives rejected

**Rule *"the port of the localhost healthz endpoint"* a position, and move `10248/tcp` to the scoping
tier.** The strongest losing option and the one the ticket priced. Its case is good: *localhost* is a
network locality and not a function; the owner did not have to write the word; §10.3's boundary limb asks
the owner to **name the boundary** and *localhost* is the narrowest boundary there is; and §27.6 conceded
the word does evidential work — *"the label tells you the default is deliberate"*. **It loses on limb 1**,
and it loses a second time and independently on **double-counting**: the comment's only truth-maker is
`healthzBindAddress: "127.0.0.1"` twelve lines below it, which is the very default the weak-tier cell
already records. Promoting on the comment counts one maintainer act twice and calls the second count a
second support — which is exactly what §24.12's *"more than a default"* criterion for the scoping tier
forbids. `10259/tcp` and `10257/tcp` are in that tier on **two artefacts and two acts** (the owner's
ports table plus kubeadm's `--bind-address=127.0.0.1`); `10248` has one act described twice.

**Rule the doc comment a weaker artefact class and discount the footing.** Its case: a Go struct comment
is further from a reader than a documentation page, §16.5 accepted the artefact only for a footing on an
already-listed row, and §27.13 named admission on it as the note's thinnest joint. **It loses on limb 4**
— the artefact is not a premise — and it loses on **measurement**: **[measured]** the same discount
applied honestly demotes `9042/tcp` Cassandra out of the prohibition tier (a YAML comment in a tarball)
and weakens `5984/tcp` CouchDB (a line in `default.ini`), moving two cells on a question nobody asked.

**Rule the doc comment not an owner artefact at all — a code comment is code, not documentation.**
Tempting on §19.7's provenance reasoning. **It loses on ADR-0045 limb 1**: `k8s.io/kubelet` `v0.34.0` is
a release artefact the owner publishes, so the comment is **issued** in exactly limb 1's sense, and
issuance is not weakened by the prose also being compiled past. It loses again on **cost**: it takes
`10255`'s footing and `10249`'s scoping cell with it, and it leaves `10248/tcp` — a row whose admission
#95 settled on Claim 3 and which this ticket may not re-open — with **no footing cell at all**, which is
the coverage gap §16.7's coverage line exists to prevent.

**Make the survival test sufficient as well as necessary.** It would make the whole question one
retrieval, which is the property limb 1 is otherwise built for. **It loses on `readOnlyPort`**:
*"the read-only port for the Kubelet to serve on with no authentication/authorization"* survives every
value flip and is plainly not a position on network placement, so sufficiency promotes §16.5's measured
**description** into a position and moves a prohibition-tier cell as a side effect. This is ADR-0059
limb 3's shape — *necessary for the top verdict, not sufficient* — met in a second column, and it is
recorded rather than invented.

**Decline the ADR and record #95's branch as settled.** The house default, and it is what §16.6's test
asks first. **It loses for ADR-0059's reason exactly**: every prior declination could name two general
rules that between them decided the case, and here the deciding rule was written down nowhere — which
is why #70 could apply the line three times without ever reaching its middle, why §16.5 and §27.6 each
concluded *label* by citation, and why ADR-0036 itself asked for the case to be ticketed rather than
decided.
