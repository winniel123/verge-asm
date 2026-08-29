# ADR-0056: A port constant in a library is not a shipped listener

- **Status:** Accepted
- **Date:** 2026-08-14
- **Ticket:** [#95 `10249/tcp` kube-proxy metrics: a bare mux with no authn or authz, and two weaker numbers beside it](https://github.com/winniel123/verge-asm/issues/95)
- **Map:** [#1 Map: verge-asm v1 spec](https://github.com/winniel123/verge-asm/issues/1)

## Context

[`sensitive-ports.md`](../research/sensitive-ports.md) §2.2 admits a row on an owner's
**specification**, its **issued documentation**, or its **shipped default**.
[ADR-0036](./0036-a-shipped-default-is-the-configuration-that-takes-effect.md) says which artefacts
count as that third form — the configuration that *takes effect* and that the project *documents as
its default* — and §12(a) rules that an **example** configuration file satisfies neither and attests
nothing in either direction. [ADR-0035](./0035-a-cryptographic-primitives-owner-is-its-specifier.md)
and §10.5 define **owner** by **authorship**: the party that designed the protocol or authors the
reference implementation.

None of them asks whether the owner actually **runs** anything on the number.

[#95](https://github.com/winniel123/verge-asm/issues/95) met the case where that matters. Kubernetes'
`pkg/cluster/ports/ports.go` at `v1.34.0` is a single file, written by a single party, opening with
*"In this file, we can see all default port of cluster. It's also an important documentation for us."*
It names eight constants in the same comment style. Two of them —
`ProxyStatusPort = 10249` and `KubeletHealthzPort = 10248` — belong to binaries Kubernetes builds and
ships. One — `CloudControllerManagerPort = 10258`, re-exported from
`staging/src/k8s.io/cloud-provider/ports.go` — belongs to a binary Kubernetes does **not** build.

**[measured]** at `v1.34.0`:

> ```go
> // This file should be written by each cloud provider.
> // For a minimal working example, please refer to k8s.io/cloud-provider/sample/basic_main.go
> // The current file demonstrate how other cloud provider should leverage CCM and it uses fake
> // parameters. Please modify for your own use.
> ```
> — `cmd/cloud-controller-manager/main.go`, its opening comment

and `hack/lib/golang.sh` lists `cmd/kube-proxy`, `cmd/kube-apiserver`, `cmd/kube-controller-manager`,
`cmd/kubelet`, `cmd/kubeadm`, `cmd/kube-scheduler`, `kube-log-runner`, `kube-aggregator`,
`apiextensions-apiserver` and `gce/gci/mounter` as `server_targets`, five of those as
`server_image_targets`, and four as `node_targets`. **`cmd/cloud-controller-manager` appears in none
of them.**

So on the rules as written, `10249` and `10258` are indistinguishable: same file, same author, same
comment shape, same registry position inside an *Unassigned* range. Nothing written down separates a
number its owner listens on from a number its owner publishes for other people to listen on.

The full working is [`sensitive-ports.md`](../research/sensitive-ports.md) §27.7.

## Decision

> **A default port constant published in a library is a number, not a listener. §2.2's third form and
> [ADR-0048](./0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md)'s *established*
> limb are both satisfied only where the party stating the default **ships software that listens on
> the pair**.**

Three limbs.

1. **The shipping test, and it is read off the build rather than judged.** A party places a
   `(port, transport)` pair only where it distributes an artefact that binds it — a binary in its
   release, an image it publishes, a package it ships. The evidence is the owner's own build
   definition, release manifest or published artefact list, in the same way ADR-0036 reads the
   configuration that *takes effect*. Authorship of the constant is not enough, and neither is
   authorship of the library that consumes it.
2. **An example program is §12(a)'s example file, one artefact class over.** Where the owner's only
   entry point for the number self-declares as a demonstration — *"this file should be written by
   each cloud provider"*, *"it uses fake parameters"* — it satisfies neither of ADR-0036's limbs and
   **attests nothing in either direction**. §12.2's finding that *the file always says which it is*
   holds for a `main.go` exactly as it held for nine configuration files. The numbering constant does
   not supply what the example withholds: a constant states a value, an example states that nobody
   runs it.
3. **The party that does ship the listener is the party whose default counts, and it is tested per
   build.** Where third parties build their own binaries from the library — every cloud provider's
   cloud-controller-manager — each such build has a shipped default its own vendor owns, admissible
   under §2.2's third form **about that build**, and subject to §10.5 like any other owner's. This is
   [#76](https://github.com/winniel123/verge-asm/issues/76)'s *ownership is tested per port, not per
   sentence* read one level over: a **library** is not a unit of evidence either. A claim about a
   thing that runs is.

## Rationale

**The asymmetry the rule fixes is the one §2.2 exists to prevent.** §2.2's opening sentence is *the
claim may not be asserted by us*. A row founded on `CloudControllerManagerPort = 10258` would be
founded on a claim analysis inherited from a **sibling port** — §24.3's delegating-auth stack,
transferred whole because the same options constructor is called — attached to a number nobody's
shipped software binds. That is the reader closing the gap, which is exactly what §2.3 refuses when a
corroborator does it. It is not better because the party is the owner. It is the right party
speaking about the wrong object, which is the shape [ADR-0045](./0045-an-owners-documentation-is-what-it-has-issued.md)
named for the wrong *time*.

**The alternative reading collapses determinacy as well.** ADR-0048's unit is the **protocol, not the
vendor**, and *live* means **current, never numerous**. If a library constant places a pair, then the
service `10258` implies is *some vendor's cloud-controller-manager* — a population whose members are
built by different parties, released on different schedules, and free to choose otherwise. A fired
signal must name a service (§2.4) — *whichever CCM this operator happens to run* is not one. The rule
therefore keeps §2.4's gate honest without needing to invent a contest, which ADR-0048 would demand a
defeating artefact for and which does not exist.

**It is cheap and it is falsifiable.** The test is a build definition, which is a shorter read than
any of the artefacts §12 enumerates, and its answer is a list rather than a judgement. And limb 3
states its own reopening condition: name a vendor that ships a CCM on `10258` and documents a
boundary, and the row is admissible on that vendor's attestation.

**Why it is not an owner rule.** §10.5 defines owner by authorship and stays untouched. Kubernetes
still owns the cloud-controller-manager interface and would still be the party to attest a claim
about it. What this ADR adds is that **attesting a placement requires a listener**, and placement is a
different question from ownership — the distinction §10.4's *rules whether a shipped default attests
and never what about* already draws for direction, drawn here for subject.

## Consequences

- **`10258/tcp` is refused** and enters `sensitive-ports.md` §4.6, taking the exclusion count to
  **20**. The refusal's operative ground is the claim gate — Claim 3's boundary limb has nothing to
  answer it — and this ADR is the **second, independent** ground, so the negative is overdetermined
  and bounded on arrival under [ADR-0046](./0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
  limb 1.
- **No listed row is disturbed.** Every pair on the list is bound by software its attesting owner
  ships. `10249/tcp` and `10248/tcp` are admitted in the same pass on binaries that are in
  Kubernetes' build sets.
- **`CloudControllerManagerWebhookPort` is disposed in advance.** §24.11 recorded its value as
  unretrieved. Whatever it is, it is a library constant for a binary Kubernetes does not ship.
  `sensitive-ports.md` §27.14 records that as **disposed by rule rather than by retrieval** and names
  it as the weaker disposal it is.
- **ADR-0036 is extended, not amended.** Its subject stays configuration. This ADR carries the
  example rule into executables and cites it.
- **It travels to every curated table under [ADR-0032](./0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md).**
  The weak-key table's analogue is a body that specifies an algorithm identifier nobody implements —
  ADR-0035 already refuses a *distributor's* acceptance floor as evidence about the primitive, and
  this is the same distinction on the *existence* axis rather than the *authority* axis.
- **Where it is thin.** It is minted on **one row**, which is the cost §25.6 priced. Its second
  instance is already visible (`CloudControllerManagerWebhookPort`) and its reopening condition is
  named, but the rule is correct-and-untested rather than measured across a population.
  `sensitive-ports.md` §27.13 records that.

## Alternatives rejected

**Decline the ADR — §12(a) and ADR-0048 already require it.** §16.6's test, applied seriously, and
the test that produced *no ADR* at §20.9, §23 and §24.9. **It loses on both halves.** §12(a) and
ADR-0036 are about **configuration files** — all ten of §12.2's artefacts are — and say nothing about
a program entry point. Carrying them to an executable is an extension. And ADR-0048's *its own
software listens* clause governs **determinacy**, which §27.7 expressly declines to rest the refusal
on, while §2.2's attestation gate and §10.5's owner definition key on authorship and never ask
whether the author ships a listener. The gap is demonstrable rather than theoretical: on the existing
rules, `10249` and `10258` are the same case.

**Refuse `10258` on determinacy instead, and skip the ADR.** Cleaner-sounding. **It loses on
ADR-0048's own refusal-artefact rule** — *every determinacy refusal must name the artefact that
defeated the convention* — and there is no such artefact. Nobody contests `10258`. The range is
Unassigned. The defect is that the convention was never **established**, not that it was defeated.
Routing the refusal through determinacy would put a finding with no defeating artefact into the one
table ADR-0048 built to require one.

**State the rule as *the owner must run it in production*.** Narrower and it would be wrong. The list
holds `2375/tcp` Docker, whose shipped default listens on a unix socket, and `10255/tcp`, whose
shipped default is off — neither of which the owner runs on the pair as shipped. The test is whether
the owner **distributes something that binds the pair when configured to**, not whether anybody has
it switched on, which is [ADR-0054](./0054-a-claim-step-is-answered-only-by-evidence-about-that-step.md)
limb 2's frame kept intact.

**Treat the library as the shipped artefact, on the ground that Kubernetes publishes
`k8s.io/cloud-provider` as a release module.** True and beside the point. A Go module is a
distribution of **source**, and what it distributes here is a default a caller may adopt, ignore or
override. §12's *costly act* test decides it the same way it decided the example config: the cost
§10.4 prices is friction at first run, and a module nobody has built produces no first run.
