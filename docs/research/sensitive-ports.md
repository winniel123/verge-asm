# Ports that are never legitimately internet-facing

Research ticket #21 — wayfinder research for the verge-asm v1 spec.

**Question.** Which ports are never legitimately internet-facing, and on what evidence — given that
this list is load-bearing for the shipped `sensitive-port-exposed` signal rather than a scanning
convenience?

**Framing.** This is a **normative** question, and it is not the question
[#4](https://github.com/winniel123/verge-asm/issues/4) answered. That ticket asked *how likely is
this port to be found open* and built the ~140-port `verge-core` hot set from it. This one asks
*given the port is open to the internet, is that ever correct?* A port can be common and always
wrong (3306), or rare and perfectly fine (8443). Frequency data cannot answer it, and the standing
temptation throughout this note is to let a frequency source stand in for a position. §2 exists to
stop that.

Three constraints from decisions already made shape the answer before any evidence is gathered:

1. **No severity.** [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md) settled that a
   signal is a named fact with evidence. Whatever this list is, it is not a ranking, and a "middle"
   cannot be smuggled in as a severity level (§5).
2. **The signal reads derived `Exposure`, not raw reachability.** So the list never has to reason
   about *where* the observer stood — [#14](https://github.com/winniel123/verge-asm/issues/14)
   already did that. This turns out to dissolve the hardest-looking boundary case (§4.1).
3. **Changing the list is aperture-adjacent.** Adding *or removing* a port bumps the rule version
   and makes two evaluations non-comparable. The cost is symmetric, which is the argument for a
   **tight** list over a generous one (§2.4).

---

## 1. Summary

| Decision | Answer |
|---|---|
| The list | **38 `(port, transport)` pairs** in three classes — §3. **Superseded by §11 — the list is 37 pairs; `161/udp` is removed. Confirmed at 37 by §14, which refused `7000/tcp` and `7001/tcp` on determinacy** |
| Evidence standard | A **named claim** from three permitted claims, **attested** by the source that owns it, plus a **determinacy** gate — §2. **Amended by §12 — an example config attests nothing, and a distributor's shipped default corroborates and never carries a row.** **§2.2's footing table re-derived from shipped bytes by §13 — every cell confirmed, no row moves, and an attestation is retrieved over the artefact rather than over the row** |
| Cloud-provider and government port lists | **Corroboration only, never sole grounds.** They are risk lists, not never-lists, and they contradict each other — §2.3 |
| Management planes inside a VPC | **Not a problem for the list.** `Exposure` is defined from an internet vantage, so the vantage does the relativising and the list can be absolute — §4.1 |
| Does TLS change a verdict | **No.** TLS bears on one of the three claims and never on the other two — §4.2 |
| High ports that are conventionally anything | **Excluded by the determinacy gate**, which is a gate on the *port*, not on the service — §4.3. **Amended by §14 — a squat is contested where the other convention is *live*, which is why 9200 is listed and 9100, 7000 and 7001 are not ([ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md))** |
| Does the list have a middle | **No middle, one signal, binary** — and not because the middle is empty, but because it is not a property of the port — §5 |
| Hot-set containment | **Independent lists, one-directional build-time invariant `sensitive ⊆ hot`** — §6 |
| Closest call | **6443 kube-apiserver, excluded** — §4.4 |

The headline result is the one that would not have come out of a frequency instrument:

> **The list contains no remote-administration ports.** No 22, no 3389, no 5985/5986. Every
> cloud-provider and government list surveyed here leads with exactly those ports. The divergence is
> not an oversight — it is the whole difference between "commonly attacked" and "never correct", and
> §2.3 shows the evidence forcing it.

---

## 2. The evidence standard

> **Amendment — [#33](https://github.com/winniel123/verge-asm/issues/33).** This standard
> attaches to **this table**, not to the rule that reads it. Per
> [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md), only §2.2's
> **attestation** gate (with §2.3's corroborator rule) travels to other curated tables; §2.1's
> closed claim set is a *theorem*
> about `sensitive-port-reached-from-internet` specifically — it enumerates the mismatches an
> internet vantage can supply — and §2.4's **determinacy** gate applies only where a port stands
> **surrogate** for a service, which is the one surrogate v1 has. Neither generalises, and
> elsewhere they are *outside the domain* rather than passed. Do not cite §2.1 or §2.4 as a
> general evidence bar.

### 2.1 The claim must be one of three

A port earns a place only if a **specific, checkable claim** can be made about the service it
implies. Three claims are permitted, and nothing else is:

- **Claim 1 — No authentication in the protocol as shipped.** The service, in the configuration its
  maintainers ship, admits anonymous commands. This is a fact about released software, not a
  judgement.
- **Claim 2 — Credentials or session content in cleartext, with a standardised encrypted successor
  reachable on a different port.** The successor clause matters: it is what makes the *plaintext*
  port wrong rather than the *protocol family* wrong, and it is why the list is keyed on
  `(port, transport)` pairs rather than services.
- **Claim 3 — The protocol's intended clients are other components of the same system, not internet
  users.** Database wire protocols, cache protocols, cluster coordination and inter-node transports.
  Exposing these enables no intended use whatsoever; it only enlarges the attack surface.

Claim 3 is the one that carries the databases, and it is also the one that does the most work
excluding things. Applied honestly it removes SSH, RDP, WinRM, Kibana, Grafana, Jenkins and the
Kubernetes API server from consideration in a single stroke — because for each of those, a human or
a remote client *is* the intended audience, and remote administration over an untrusted network is
the express purpose the protocol was designed to serve.

> **Amended by §10** ([#37](https://github.com/winniel123/verge-asm/issues/37)). Claim 1 carries a
> **qualifier** it has always relied on and never stated (§10.1); Claim 3's boundary is wider than
> the words "same system" (§10.3); and the set of three is **closed by construction** rather than by
> enumeration (§10.2). No row moved. Read §10.1-§10.3 before applying this section.

### 2.2 The claim must be attested by the source that owns it

The claim may not be asserted by us. It must be quotable, verbatim, from:

- the protocol's **specification** (an RFC or equivalent), or
- the project's or vendor's **own documentation**, or
- the project's **shipped default**, as documented by the project.

That third form is deliberate and it is not a weaker substitute. A default listen address is a
maintainer position expressed in code rather than prose, and it is what actually ships. PostgreSQL
never writes a "do not expose" sentence, but it does document
`listen_addresses` as defaulting to `localhost`, "which allows only local TCP/IP 'loopback'
connections"
([postgresql.org/docs/current/runtime-config-connection.html](https://www.postgresql.org/docs/current/runtime-config-connection.html)).
That is an attestation. Without allowing it, the list would flag MySQL and not PostgreSQL — an
asymmetry driven by a documentation accident rather than by any difference in the two services'
deployment models, which is exactly the kind of arbitrariness that destroys a curated list's
credibility.

**But the two forms are not equally strong, and the list must not hide the difference.** Several
ports everyone "knows" are sensitive turn out to have **no vendor sentence behind them at all** —
only a documented default. Disclosed here rather than smoothed over:

| Row | Footing |
|---|---|
| 6379 Redis, 11211 memcached, 3306 MySQL, 1433 MS SQL, 9200 Elasticsearch, 873 rsync, 445 SMB, 623 IPMI, 161 SNMP | **Explicit prohibition** in the owner's own words |
| 27017/27018/27019 MongoDB, 2049 NFS, 2181 ZooKeeper, 25672 RabbitMQ, 2376 Docker | **Explicit trusted-network scoping**, slightly weaker than a prohibition |
| **5432 PostgreSQL, 5984 CouchDB, 9042 Cassandra** | **Shipped default only** — no prohibition exists upstream |

The last row is the weak footing. PostgreSQL is treated at length in §4.5; Cassandra's strongest
upstream sentence is an attack-surface observation rather than a prohibition; and CouchDB's actual
"do not expose" warning covers the **Erlang distribution port, not 5984**, so it must not be
transposed onto the HTTP API. All three are on the list, and all three are labelled.

> **Amended by §10.4** ([#37](https://github.com/winniel123/verge-asm/issues/37)). A shipped default
> attests in **one direction only**. A default that *restricts* is a maintainer position; a default
> that is *permissive* is the absence of one, and may neither admit a row nor exclude one. The
> separate route [#30](https://github.com/winniel123/verge-asm/issues/30) actually found — a
> **remedy** aimed at the exposure hazard that stops short of the port — is a different object and is
> specified in §10.4.3. No row moved.

> **Amended by §12** ([#69](https://github.com/winniel123/verge-asm/issues/69)). **The footing table
> above is wrong in one cell and the paragraph beneath it in one clause.** `9042 Cassandra` is **not**
> shipped-default-only: **[measured]** the shipped `conf/cassandra.yaml` carries *"For security
> reasons, you should not expose this port to the internet. Firewall it if needed."* immediately above
> both `native_transport_port: 9042` and `rpc_address: localhost`. The row moves to the **explicit
> prohibition** tier and *"Cassandra's strongest upstream sentence is an attack-surface observation
> rather than a prohibition"* is **withdrawn**. **The weak tier is two rows — 5432 PostgreSQL and 5984
> CouchDB — not three.** No `(port, transport)` pair moves.
>
> §12 also states what the **third form** reads: the configuration that **takes effect** and that the
> project **documents as its default**, both limbs. An **example** file — `EXAMPLE.conf`,
> `*.conf.sample`, `*.conf.example` — satisfies neither and attests nothing in either direction; a
> **distributor** that installs one gains operativeness and not ownership, so its packaging
> corroborates under §2.3 and is **never sole grounds for a row**. Read §12 before applying this
> section to a configuration file.

> **Amended by §13** ([#70](https://github.com/winniel123/verge-asm/issues/70)). **The footing table
> above has now been re-derived from the shipped configuration bytes of every row that has any, and
> every surviving cell is confirmed.** No footing moves and no `(port, transport)` pair moves. The
> prohibition tier is `6379`, `11211`, `3306`, `1433`, `9200`, `873`, `445`, `623` and `9042`; the
> scoping tier is `27017`/`27018`/`27019`, `2049`, `2181`, `25672` and `2376`; **the weak tier is
> `5432` and `5984`**. Two rows strengthen inside their tier — rsync's shipped `rsyncd.conf.5` puts the
> supported public listener on **874** with `873` on loopback behind it, and `nfs(5)` says *"NFS was
> developed to allow file sharing between systems residing on a local area network"* — and three
> (`27017`/`27018`/`27019`, `5432`, `5984`) have their footing re-founded on the owner's bytes rather
> than on a documentation page. **[measured]** Across fourteen artefacts from twelve projects, **no
> shipped configuration file names the public internet as a supported deployment environment**, so
> §10.3's failure condition is not met anywhere; three near-misses are examined and refused in §13.3.
>
> **Two things this table does not yet say, both routed rather than fixed here.** It places **19 of the
> 37 pairs**, and seven listed rows in its own subject matter have no cell —
> [#76](https://github.com/winniel123/verge-asm/issues/76), §13.7; the `2379`/`2380` etcd pair may
> belong in the weak tier, which would make it four rows rather than two. And **`1433`, `445` and `623`
> have no shipped configuration artefact at all**, so §13 adds no evidence about them in either
> direction. Read §13.9 before quoting this table as measured.

> **Amended by §16** ([#76](https://github.com/winniel123/verge-asm/issues/76)). **The seven listed
> rows §13.7 counted as having no cell are placed, and the table now states its own coverage.** The
> table below supersedes the one above. **[measured]** `2379`/`2380` etcd do **not** join the weak
> tier: `THREAT_MODEL.md` at `etcd-io/etcd` `v3.7.1` states *"It **must not** be exposed to untrusted
> networks or the public internet"* and names both ports by number, so §13.7's prediction is refuted
> (§16.3). **The weak tier grows to three on a different row** — `10255/tcp` kubelet, whose owner
> states no network position and whose shipped config API carries `readOnlyPort` *"Default: 0
> (disabled)"* (§16.5). **No `(port, transport)` pair moves and no row moves.**
>
> | Footing | Pairs |
> |---|---|
> | **Explicit prohibition** in the owner's own words | 6379 Redis · 11211/tcp + 11211/udp memcached · 3306 MySQL · 1433 MS SQL · 9200 **and 9300** Elasticsearch · 873 rsync · 445 SMB · 623 IPMI · 9042 Cassandra · **2379 and 2380 etcd** — **13 pairs** |
> | **Explicit trusted-network scoping**, slightly weaker than a prohibition | 27017/27018/27019 MongoDB · 2049 NFS · 2181 ZooKeeper · 25672 RabbitMQ · **4369 epmd** · 2376 **and 2375** Docker · **10250 kubelet** — **10 pairs** |
> | **Shipped default only** — no prohibition exists upstream | **5432 PostgreSQL · 5984 CouchDB · 10255 kubelet** — **3 pairs** |
> | *Outside this table's subject, and correctly absent* | *Class B's seven (23, 21, 512, 513, 514, 5900, 6000) and 69/udp rest on §2.2's **first** form — a specification, IANA's registry, or OpenBSD's deletions. 139/tcp, 137/udp and 138/udp are carried by the same Microsoft sentence as 445 and sit inside that row's cell — **11 pairs*** |
>
> **This table places 26 of the list's 37 pairs; the remaining 11 are the fourth row, and 26 + 11 =
> 37.** The coverage line is stated here so that a reader never has to count — §13.7 had to, and
> §13.10 recorded the hand count as *"the kind of thing that goes wrong"*.
>
> **Two cells are thin and are flagged rather than smoothed** (§16.9). `10250`'s scoping rests on a
> **table cell** — kubernetes.io's `Used By: Self, Control plane` — rather than on a sentence naming a
> network, which makes it the weakest member of its tier. And `4369`'s rests on an Erlang/OTP sentence
> about **distributed nodes** that does not name epmd or the port; RabbitMQ's sentence, which §3.4
> cites for it, is a **non-owner's** under §10.5 and corroborates only (§16.6). Read §16.9 before
> quoting either cell.

> **Amended by §18** ([#88](https://github.com/winniel123/verge-asm/issues/88)). **The table above is
> superseded in three cells.** An owner's **category** statement reaches the members the owner's own
> artefacts place inside it
> ([ADR-0049](../adr/0049-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)),
> so `10250/tcp` and `10255/tcp` kubelet are carried by Kubernetes' own *"The Kubernetes API, kubelet
> API and etcd are not exposed publicly on Internet"* and both join the **explicit prohibition**
> tier. **The tiers are prohibition 15 pairs · scoping 9 pairs · weak 2 rows**, coverage unchanged at
> **26 of 37**, and **the weak tier is `5432` and `5984` again**. **No `(port, transport)` pair
> moves.** Both kubelet cells are **conditional on
> [#83](https://github.com/winniel123/verge-asm/issues/83)**, which may remove either row; a cell
> cannot outlive its row. §18.6 carries the restated table and §18.7 flags `10255` as the tier's
> thinnest member. **One cell in the tier above is newly exposed and is ticketed rather than moved**
> — `623/udp`, whose owner sentence names a **product line** and whose port number is supplied by a
> corroborator ([#90](https://github.com/winniel123/verge-asm/issues/90), §18.5).

### 2.3 Cloud-provider and government lists corroborate; they never carry a port alone

This is the load-bearing methodological finding, and it took the most work to establish.

The instinct is to reach for CISA and the cloud providers first, because they are authoritative and
they publish enumerated port lists. Read closely, **not one of them is a "never internet-facing"
list**, and treating them as one produces a demonstrably wrong answer.

**They contradict each other, and AWS contradicts itself.** AWS Security Hub control **EC2.19**
enumerates ports that "should not allow unrestricted access", and the list includes `25 (SMTP)`,
`22 (SSH)`, `3000 (Go, Node.js, and Ruby web development frameworks)`, `5000 (Python web development
frameworks)`, `8080 (proxy)` and `8888 (alternative HTTP port)`
([docs.aws.amazon.com/securityhub/latest/userguide/ec2-controls.html](https://docs.aws.amazon.com/securityhub/latest/userguide/ec2-controls.html)).
Meanwhile AWS's own Trusted Advisor check `HCP4007jGY` puts port 25 in the *green* band, describing
green as ports "typically used by applications that require unrestricted access, such as HTTP and
SMTP"
([docs.aws.amazon.com/awssupport/latest/user/security-checks.html](https://docs.aws.amazon.com/awssupport/latest/user/security-checks.html)).
The same vendor calls port 25 high-risk in one product and expected-to-be-open in another. Both
statements are defensible as *risk* guidance. Neither can support a claim that exposure is never
correct.

**A major provider ships two of these ports open to the world by default.** Google Cloud's default
VPC network is pre-populated with `default-allow-ssh` (`tcp:22`) and `default-allow-rdp`
(`tcp:3389`), both with source range `0.0.0.0/0`
([docs.cloud.google.com/firewall/docs/firewalls](https://docs.cloud.google.com/firewall/docs/firewalls)).
Those are precisely the two ports CIS AWS Foundations v1.2.0 §4.1 and §4.2 tell you to close. Note
the asymmetry: the sibling rule `default-allow-internal` *is* source-scoped, to `10.128.0.0/9`, so
Google plainly understood source-scoping and chose not to apply it to SSH and RDP. A port that one
hyperscaler opens to the internet by design cannot be described as never legitimately
internet-facing.

**The federal directive does not do what it is usually cited as doing.** CISA **BOD 23-02** is the
obvious authority, and it enumerates protocols — but it enumerates *protocols, never port numbers*,
its list is explicitly non-exhaustive, and it includes HTTPS, SSH and RDP:

> "Devices for which the management interfaces are using network protocols for remote management
> over public internet, including, but not limited to: Hypertext Transfer Protocol (HTTP),
> Hypertext Transfer Protocol Secure (HTTPS), File Transfer Protocol (FTP), Simple Network
> Management Protocol (SNMP), Teletype Network (Telnet), Trivial File Transfer Protocol (TFTP),
> Remote Desktop Protocol (RDP), Remote Login (rlogin), Remote Shell (RSH), Secure Shell (SSH),
> Server Message Block (SMB), Virtual Network Computing (VNC), and X11 (X Window System)."
> — [BOD 23-02](https://www.cisa.gov/news-events/directives/binding-operational-directive-23-02), 13 June 2023

More decisively, BOD 23-02 **does not prohibit internet exposure at all**. It permits it, subject to
mediation:

> "networked management interfaces are allowed to remain accessible from the internet on networks
> where agencies employ capabilities to mediate all access to the interface in alignment with OMB
> M-22-09, NIST 800-207, the TIC 3.0 Capability Catalog, and CISA's Zero Trust Maturity Model."
> — [BOD 23-02](https://www.cisa.gov/news-events/directives/binding-operational-directive-23-02)

Any note citing BOD 23-02 for "these must never face the internet" is misreading it. The mapping
from its protocol list to port numbers is the reader's inference, not CISA's.

**Two CISA sources do state an unqualified position, and they are worth having.** The BOD 23-02
implementation guidance says of out-of-band interfaces:

> "These out of band interfaces should never be directly accessible via the public internet."
> — [BOD 23-02 implementation guidance](https://www.cisa.gov/news-events/directives/bod-23-02-implementation-guidance-mitigating-risk-internet-exposed-management-interfaces)

And the current Cross-Sector Cybersecurity Performance Goals, goal 3.S, is broader in audience than
BOD 23-02 and unqualified in wording:

> "Network management interfaces (NMIs) should never be exposed to the public internet and should
> only be accessible from within enterprise networks."
> — [CPG v2.0, December 2025, goal 3.S](https://www.cisa.gov/sites/default/files/2025-12/CPG_Report_2.0_508c.pdf)
> (verified against the PDF text; the CPG applies to all critical infrastructure, whereas BOD 23-02
> binds only FCEB agencies)

Both attest a **category** — "management interface", "out-of-band interface". Neither attests a
port. They are therefore excellent corroboration for the Class C entries in §3.3 and useless as the
sole grounds for any individual row.

**The rule that follows.** A cloud-provider control, a CIS recommendation, or a government directive
may corroborate a row. It may never be the only evidence for one. The permitted claims in §2.1 must
be attested per §2.2, by a source that owns the protocol.

> **Amended by §10.5** ([#37](https://github.com/winniel123/verge-asm/issues/37)). *Owns* is defined:
> the party that **designed the protocol**, or that **authors the reference implementation**, speaking
> about the thing it designed or wrote. A **distributor owns its own shipped configuration and owns
> nothing about the protocol**, so its packaging is admissible under §2.2's third form while its
> security-guide prose is corroboration under this section — which is why §9.1 could read Debian's
> `rpcbind.default` and decline Red Hat's Security Guide in the same breath. The self-contradiction
> is evidence the line is right, never the ground for it. No row moved.

> **Amended by §18** ([#88](https://github.com/winniel123/verge-asm/issues/88)). **This section's
> refusal is about *standing*, never about *grammar*, and the two had never been made to disagree**
> because every category statement in the corpus when it was written came from a non-owner. **An
> owner's category statement reaches the members the owner's own artefacts place inside it**, on
> three limbs — standing under §10.5, **membership established by the owner's own artefact and never
> by a corroborator's port number**, and **defeat per member** wherever the owner elsewhere names
> that member's internet-facing deployment as supported
> ([ADR-0049](../adr/0049-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md),
> §18.1). The category's unit is a **protocol or interface, never a vendor's product line**
> ([ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md)).
>
> **Nothing in this section weakens.** CISA, the CPG, NSA, AWS, Google Cloud and CIS still carry no
> row, for exactly the reason given above: they cannot close the category-to-port gap, so the reader
> closes it. What is **withdrawn** is the implication in *"Neither attests a port"* that the
> grammatical shape of a category statement is the defect — §18.2 measures that reading against the
> corpus and finds it would leave the list resting on which maintainers happened to type a number.
> **The hardening-instruction objection of §9.1 and §4.4 is untouched against a non-owner and adds
> nothing against an owner** that §10.3's failure condition does not already test (§18.3).

### 2.4 Determinacy: the port must imply the service

A fired signal names a service. If the `(port, transport)` pair does not determine one, the signal
is unactionable however sound the claim is. Two failure modes:

- **Conventionally generic ports.** 8080, 8000, 8888, 8443, 3000, 5000, 9000, 9090, 8088, 10000 are
  conventionally *anything*. Excluded regardless of what runs there (§4.3).
- **Version-dependent ports.** Hadoop's NameNode web UI moved from 50070 to 9870 between major
  versions, so the inference from port to service depends on which version is running. Excluded.

There is a third, subtler issue the IANA registry surfaces, and it cuts *against* over-trusting port
numbers generally. Many of the best-known sensitive ports are **squatted, not registered**. From
[IANA's Service Name and Transport Protocol Port Number Registry](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml)
(retrieved 2026-08-13, registry last updated 2026-08-11):

| Port | Registered as | Commonly assumed to be |
|---|---|---|
| 9200/tcp | `wap-wsp` — "WAP connectionless session service" | Elasticsearch |
| 9300/tcp | `vrace` — "Virtual Racing Service" | Elasticsearch transport |
| 6443/tcp | `sun-sr-https` — "Service Registry Default HTTPS Domain" | Kubernetes API server |
| 2181/tcp | `eforward` | ZooKeeper |
| 5601/tcp | `esmagent` — "Enterprise Security Agent" | Kibana |
| 8500/tcp | `fmtp` — "Flight Message Transfer Protocol" | Consul HTTP API |
| 9092/tcp | `XmlIpcRegSvc` — "Xml-Ipc Server Reg" | Kafka |
| 9100/tcp | `hp-pdl-datastr` — "PDL Data Streaming Port" | Prometheus node_exporter |
| 1521/tcp | `ncube-lm` — "nCube License Manager", annotated "Unauthorized Use Known on port 1521". Oracle's registered SQL\*NET is port **66** | Oracle DB listener |
| 5985, 5986/tcp | `wsman` / `wsmans` (DMTF WS-Management). The name `winrm` **is** registered — at port **47001** | WinRM |
| 623/udp | `asf-rmcp`. The string "IPMI" appears **nowhere** in the registry | IPMI |
| 10250/tcp, 10255/tcp, 9042/tcp, 15672/tcp | **inside explicit "Unassigned" ranges** | kubelet, Cassandra, RabbitMQ mgmt |

The strings `elasticsearch`, `cassandra`, `kubelet`, `kubernetes`, `ipmi` and `jetdirect` return
**zero matches** across all 14,531 rows of the registry — name, description and assignee. 6000/tcp
*is* registered, as `x11`, across the 6000-6063 range.

One field must not be over-read. 112 rows carry a non-empty `Unauthorized Use Reported` value — the
`ncube-lm` annotation above is one — but the field marks **squatting on a number without
registration**, a registry-hygiene matter. It is not a security judgement and is not used as one
here.

Properly registered to the service everyone assumes: `6379 redis`, `2375 docker` / `2376 docker-s`,
`2379 etcd-client` / `2380 etcd-server`, `27017 mongodb`, `11211 memcache`, `5984 couchdb`,
`3306 mysql`, `5432 postgresql`, `873 rsync`, `23 telnet`, `445 microsoft-ds`, `5900 rfb`.

And the registry itself disclaims the inference in the strongest terms available:

> "ASSIGNMENT OF A PORT NUMBER DOES NOT IN ANY WAY IMPLY AN ENDORSEMENT OF AN APPLICATION OR
> PRODUCT, AND THE FACT THAT NETWORK TRAFFIC IS FLOWING TO OR FROM A REGISTERED PORT DOES NOT MEAN
> THAT IT IS "GOOD" TRAFFIC, NOR THAT IT NECESSARILY CORRESPONDS TO THE ASSIGNED SERVICE. FIREWALL
> AND SYSTEM ADMINISTRATORS SHOULD CHOOSE HOW TO CONFIGURE THEIR SYSTEMS BASED ON THEIR KNOWLEDGE
> OF THE TRAFFIC IN QUESTION, NOT WHETHER THERE IS A PORT NUMBER REGISTERED OR NOT."
> — IANA registry header note

**Consequence for the product, not just for the list.** The signal's name and evidence must be
written to survive this. `sensitive-port-exposed` claims *a port associated with a sensitive service
is reachable from an internet vantage* — it does not claim the service is running. Squatting means
registration cannot be the determinacy test; uncontested convention has to be, and the note records
which rows rest on convention rather than registration (§3, "reg." column).

> **Amended by §14** ([#75](https://github.com/winniel123/verge-asm/issues/75)). *"Uncontested
> convention"* is the right test and this section never said what makes a convention **contested**,
> which left `9200`'s squat (listed) and `9100`'s squat (excluded) separated by nothing written down.
> **A squat is contested where the other convention is *live*** — where the competing service's own
> owner currently documents that service on that number — and it is uncontested where the competing
> registration has no deployed population. Liveness is read off the competing owner's documentation
> and **never** off a frequency source, which is what keeps §1's exclusion of frequency intact. See
> [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md). §14 applies it
> to refuse `7000/tcp` and `7001/tcp`; **no existing row moves**, and the rule is a statement of what
> §4.3 and §4.6 have been doing since §21. This section still carries **no stated evidence standard**
> for determinacy the way §2.2 carries one for attestation — §8 question 10.

> **Amended by §15** ([#82](https://github.com/winniel123/verge-asm/issues/82)). **This section now has
> an evidence standard, and the sentence above about *"no stated evidence standard"* is withdrawn.** A
> determinacy finding is made on **placement statements** — a party's statement, in its own current
> documentation or its own shipped bytes, that **its own software listens on a given
> `(port, transport)` pair by default** — and on nothing else. A convention is **established** by the
> candidate's own owner; **contested** where another party places a *different protocol* on the same
> pair, or **displaced** where the candidate's own owner puts its service elsewhere or makes the pair
> version-dependent; **one statement suffices** and a survey is never claimed. The unit is the
> **protocol, not the vendor** — ScyllaDB on `9042` and OpenSearch on `9200` declare the row's own
> protocol and are one convention with it, not a contest. *Live* means **current**, never **numerous**.
> Everything else corroborates and never carries: IANA rows and the `Unauthorized Use Reported` field,
> `nmap-services`' name column, cloud-provider and government port tables, and **this project's own
> frequency half**. And **every determinacy refusal must name the artefact that defeated the
> convention** — a refusal citing no document is not a finding. See
> [ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md). **No row
> moves**; two exclusion grounds are restated and both strengthen (§4.3's `9090`, §9.3.4's `79`). Read
> §15 before applying this section.

### 2.5 What was excluded from evidence entirely

- **`nmap-services` open-frequency data.** Frequency, and 2008-vintage —
  [#4](https://github.com/winniel123/verge-asm/issues/4) §2.2 already established it is unusable
  even for its own question.
- **Shodan/Censys exposure studies.** The ticket named them as candidates. They report frequency.
  Every one of them answers "how many are exposed", which is evidence that a problem is widespread,
  not evidence that exposure is wrong. Not used, anywhere in this note.
- **Redis's own prevalence rhetoric.** The upstream security page says "many users fail to protect
  Redis instances from being accessed from external networks. Many instances are simply left exposed
  on the internet with public IPs"
  ([redis.io](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)). This is a
  frequency assertion embedded in a primary source. It accompanies a position and it is *not* the
  position; Redis is on the list for the "trusted clients inside trusted environments" sentence, and
  this sentence is cited nowhere.
- **CISA BOD 22-01 (KEV).** Checked and discarded on two grounds: it is keyed to CVE IDs rather than
  ports, and it was **revoked on 10 June 2026**, superseded by BOD 26-04
  ([cisa.gov](https://www.cisa.gov/news-events/directives/bod-22-01-reducing-significant-risk-known-exploited-vulnerabilities)).
  Worth recording that even the government normative sources move under this list — see §7.

### 2.6 An escape hatch that proved unnecessary

An earlier draft of this standard allowed a second attestation route for cases where no
protocol-owner sentence could be found: the protocol's own specification attests that it *is* a
management interface (category membership), and CISA CPG 3.S attests that interfaces in that
category should never be exposed (category verdict). It was drafted to carry `161/udp` and
`623/udp`.

**It carries nothing, because direct first-party attestations were found for both** — CISA's own
IPMI alert and Dell's statement via CERT/CC for 623, and CISA's SNMP alert for 161 (§3.4). The
escape hatch is recorded because its disuse is the strongest available statement about the
standard's tightness:

> **No row on this list rests on a government or cloud-provider source alone.** Every one of the 38
> is attested by the specification, the project, or the vendor that owns the protocol.

> **Amended by §10.6** ([#37](https://github.com/winniel123/verge-asm/issues/37)). That sentence is
> **established for 37 rows and not for `161/udp`** — the row this escape hatch was drafted to carry,
> and the one whose first-party attestation this section names as "CISA's SNMP alert", which is a
> corroborator rather than an owner. `161/udp` **stays on the list** and is now the note's disclosed
> **second-weakest row** beside 5432 (§4.5). The retrieval that would settle it is
> [#66](https://github.com/winniel123/verge-asm/issues/66).

### 2.7 The most tempting laundering instance in the corpus, and why it was refused

`111/tcp` (rpcbind/portmapper) is the case that shows the §2.3 rule earning its keep, so it is worth
recording rather than leaving as a silent exclusion.

There is an obvious CISA citation available — Alert TA14-017A, *UDP-Based Amplification Attacks* —
which names "Portmap (RPCbind)" with a bandwidth amplification factor of "7 to 28". It is
authoritative, it is CISA, and it is about internet-reachable rpcbind. **It is also a frequency and
magnitude source, not a position.** It says how hard an exposed rpcbind can hit someone; it does not
say that exposing it is never correct. Citing it for the latter is precisely the substitution this
ticket was written to prevent, and it is tempting exactly because the source is so reputable.

Checking the protocol's actual owners closes the door rather than opening it:

- `rpcbind(8)` contains **no security section and no exposure statement** at all.
- RFC 1833's Security Considerations section reads, in its entirety: *"Security issues are not
  discussed in this memo."* That is an explicit non-statement, citable as such and as nothing else.

So 111 is excluded — not because it is defensible on the internet, but because **we could not find
anyone entitled to say it isn't.** That is an uncomfortable result and it is the correct one; the
alternative is a list whose rows are backed by whichever authoritative-looking document was nearest.

> **Amended by §17** ([#79](https://github.com/winniel123/verge-asm/issues/79)). *"We could not find
> anyone entitled to say it isn't"* is this note's model sentence for a negative, and
> [ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md) requires that such a
> sentence name the **document classes** searched rather than the documents. §17.1 does that for every
> negative the note stands on, and finds that **`111/tcp`'s carries no weight**: §10.1 Step 1 refuses
> Claim 1 independently and §10.4.3's remedy route is a second independent ground, so no document
> retrieved from any class could re-admit the row. The negative is **bounded on arrival** — it is
> preserved here as the standard's clearest statement of its own tightness, not as a live ground.
> **Two negatives elsewhere in the note were exposed and swept, and one of them found a sentence**;
> read §17 before quoting this paragraph as the model.

---

## 3. The list

> **Amended by §11 ([#66](https://github.com/winniel123/verge-asm/issues/66)): the list is 37 pairs.**
> `161/udp` is **removed**. Its row is left standing in §3.3 and its quotes in §3.4, marked here rather
> than deleted, per the name-and-withdraw convention — where §11 and this section disagree, §11 governs.

**38 `(port, transport)` pairs.** The `reg.` column records whether IANA registers the port to the
service named — `yes` means registered, `sq.` means the service squats on a registration belonging
to something else, `--` means the port is unregistered. Per §2.4 this is disclosure, not
qualification: convention, not registration, is the determinacy test.

### 3.1 Class A — the protocol as shipped admits anonymous commands (Claim 1)

| Port/transport | Service implied | reg. | Why internet exposure is never correct |
|---|---|---|---|
| 2375/tcp | Docker daemon REST API (plaintext) | yes | Unauthenticated control of the container runtime; Docker deprecated unauthenticated TCP in v26.0 and the daemon now refuses to start with TLS disabled on a TCP address |
| 2379/tcp | etcd client API | yes | Ships with neither RBAC nor transport authentication; read access to it is read access to every cluster Secret |
| 2380/tcp | etcd peer API | yes | Same defaults, and it is a node-to-node channel that has no internet client under any topology |
| 10250/tcp | kubelet API | -- | `--anonymous-auth` defaults to `true` and `--authorization-mode` defaults to `AlwaysAllow`; upstream scopes its users to "Self, Control plane" |
| 10255/tcp | kubelet read-only port | -- | Upstream describes it as serving "with no authentication/authorization"; deprecated and slated for removal |
| 6379/tcp | Redis | yes | Designed for trusted clients only; a single unauthenticated `FLUSHALL` destroys the dataset |
| 11211/tcp | memcached | yes | Upstream states outright that it must not be exposed to the internet or any untrusted user |
| 11211/udp | memcached (UDP) | yes | Spoofed-source amplification vector; upstream disabled UDP by default in 1.5.6 for exactly this reason |
| 2181/tcp | ZooKeeper client port | sq. | A fresh ensemble ships with no transport encryption, no peer authentication and world-readable/writable znodes |
| 4369/tcp | Erlang port mapper (epmd) | yes | Cluster-discovery channel whose only protection is a shared Erlang cookie |
| 9042/tcp | Cassandra native protocol | -- | Authentication and authorization are disabled by default so that nodes can find each other |
| 69/udp | TFTP | yes | The protocol has no authentication mechanism of any kind, by specification |

> **Amended by §10.7** ([#37](https://github.com/winniel123/verge-asm/issues/37)). Two "why" cells
> lead with something that is not the row's grounds. **11211/udp** leads with spoofed-source
> amplification, which is a **magnitude** and is the exact source class §2.7 refuses; the row rests
> on Claim 1 and the amplification sentence is colour. **161/udp** (§3.3) leads with a CISA
> directive, which §2.3 permits only as corroboration. Both rows stand; the wording is corrected in
> §10.7 rather than in place.

### 3.2 Class B — credentials in cleartext, encrypted successor on another port (Claim 2)

| Port/transport | Service implied | reg. | Why internet exposure is never correct |
|---|---|---|---|
| 23/tcp | Telnet | yes | Username, password and the entire session travel unprotected; SSH on 22 is the standardised replacement |
| 21/tcp | FTP control | yes | `PASS` sends the password in clear text; SFTP and FTPS are the standardised replacements |
| 512/tcp | rexec | yes | IANA's own registry describes it as "remote process execution; authentication performed using passwords and UNIX login names" |
| 513/tcp | rlogin | yes | Authentication is delegated to host-based trust, which a compromised DNS or network can forge; superseded by SSH |
| 514/tcp | rsh | yes | Same trust model as 513, applied to arbitrary command execution; superseded by SSH |
| 5900/tcp | VNC / RFB | yes | The RFB specification states its authentication is cryptographically weak and not intended for untrusted networks |
| 6000/tcp | X11 display :0 | -- | The magic-cookie authenticator is transmitted without encryption, so observing it is sufficient to seize the display |

Only display `:0` is listed. 6001-6063 are additional displays whose presence does not follow from
the same convention with the same confidence, and adding 63 rows to catch a rare case is a poor
trade against the determinacy gate.

### 3.3 Class C — the intended clients are same-system components (Claim 3)

| Port/transport | Service implied | reg. | Why internet exposure is never correct |
|---|---|---|---|
| 2376/tcp | Docker daemon REST API (TLS) | yes | TLS authenticates the client but does not change the audience; whoever holds the keys has root on the host, and Docker still directs that it be reachable only from a trusted network or VPN |
| 3306/tcp | MySQL / MariaDB | yes | A database wire protocol whose clients are application tiers; upstream states the port should not be reachable from untrusted hosts |
| 5432/tcp | PostgreSQL | yes | Same role; upstream ships `listen_addresses` defaulting to loopback (see §4.5 — this is the list's weakest row and it is labelled as such) |
| 1433/tcp | Microsoft SQL Server | yes | Vendor directs that instances not be connected directly to the internet |
| 27017/tcp | MongoDB | yes | Binds to localhost by default; upstream directs that instances be reachable only on trusted networks |
| 27018/tcp | MongoDB shard member | yes | An intra-cluster port with no external client |
| 27019/tcp | MongoDB config server | yes | Holds cluster metadata; intra-cluster only |
| 9200/tcp | Elasticsearch HTTP API | sq. | Upstream's instruction is to never expose an unprotected node to the public internet |
| 9300/tcp | Elasticsearch transport | sq. | Node-to-node binary protocol; a transport connection can reach system-internal APIs |
| 5984/tcp | CouchDB HTTP API | yes | Ships bound to `127.0.0.1`; its clients are application tiers |
| 25672/tcp | RabbitMQ inter-node (Erlang distribution) | -- | Upstream states these ports should not be publicly exposed |
| 445/tcp | SMB | yes | Microsoft: it is unlikely that any internet-originated or internet-bound SMB traffic is legitimate |
| 139/tcp | NetBIOS session service | yes | Named in the same vendor perimeter-blocking directive as 445 |
| 137/udp | NetBIOS name service | yes | Same |
| 138/udp | NetBIOS datagram service | yes | Same |
| 2049/tcp | NFS | yes | The default `sys` auth flavour trusts a client-asserted user ID, which upstream calls adequate only "on a trusted physical network between trusted hosts" |
| 873/tcp | rsync daemon | yes | Upstream directs outright that a cleartext daemon not be exposed to an untrusted network; a module without `auth users` is readable by anyone who reaches the port |
| 161/udp | SNMP | yes | CISA directs that SNMP traffic be segregated onto a separate management network; SNMPv1/v2c authenticate on cleartext community strings |
| 623/udp | IPMI / ASF-RMCP (BMC) | yes | CISA directs that IPMI be restricted to trusted internal networks, and Dell states BMCs are "not designed nor intended to be placed on or connected to the internet" |

### 3.4 The quotes behind the rows

**Class A.**

> "Configuring the Docker daemon to listen on a TCP address will require mandatory TLS verification.
> … In version 27.0 and later, specifying `--tls=false` or `--tlsverify=false` CLI flags causes the
> daemon to fail to start if it's also configured to accept remote connections over TCP."
> — [Docker: deprecated features, "Unauthenticated TCP connections"](https://docs.docker.com/engine/deprecated/) (deprecated v26.0, target removal v28.0; verified against the page's own text)

> "etcd doesn't enable RBAC based authentication or the authentication feature in the transport
> layer by default to reduce friction for users getting started with the database." … "An etcd
> cluster which doesn't enable security features can expose its data to any clients."
> — [etcd.io security guide](https://etcd.io/docs/v3.5/op-guide/security/)

> "Access to etcd is equivalent to root permission in the cluster so ideally only the API server
> should have access to it."
> — [kubernetes.io, Operating etcd clusters for Kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/)

> "By default, requests to the kubelet's HTTPS endpoint that are not rejected by other configured
> authentication methods are treated as anonymous requests, and given a username of
> `system:anonymous` and a group of `system:unauthenticated`."
> — [kubernetes.io, Kubelet authentication/authorization](https://kubernetes.io/docs/reference/access-authn-authz/kubelet-authn-authz/)

> "`--anonymous-auth`  Default: true" … "`--authorization-mode string`  Default: "AlwaysAllow"" …
> "`--read-only-port int32`  Default: 10255"
> — [kubernetes.io, kubelet CLI reference](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) (the 10255 default verified directly against the rendered page)

> "readOnlyPort is the read-only port for the Kubelet to serve on with no authentication/authorization."
> — [kubernetes.io, kubelet config API v1beta1](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/)

> "Redis is designed to be accessed by trusted clients inside trusted environments. This means that
> usually it is not a good idea to expose the Redis instance directly to the internet or, in
> general, to an environment where untrusted clients can directly access the Redis TCP port or UNIX
> socket." … "Failing to protect the Redis port from the outside can have a big security impact
> because of the nature of Redis. For instance, a single FLUSHALL command can be used by an external
> attacker to delete the whole data set."
> — [redis.io security](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)

> "Memcached does not spend much, if any, effort in ensuring its defensibility from random internet
> connections. So you _must not_ expose memcached directly to the internet, or otherwise any
> untrusted users. Using SASL authentication here helps, but should not be totally trusted."
> — [memcached wiki, ConfiguringServer](https://github.com/memcached/memcached/wiki/ConfiguringServer)

> "A ZooKeeper ensemble is expected to operate in a trusted computing environment. It is thus
> recommended deploying ZooKeeper behind a firewall."
> — [ZooKeeper Administrator's Guide](https://zookeeper.apache.org/doc/r3.9.3/zookeeperAdmin.html). [zookeeper.apache.org/security.html](https://zookeeper.apache.org/security.html) quotes that line and adds: "ZooKeeper is a coordination service intended for use inside a trusted network, not exposed directly to the Internet." (both verified by direct retrieval of the pages, not via a summarising layer)

> "By default, these features are disabled as Cassandra is configured to easily find and be found by
> other members of a cluster. In other words, an out-of-the-box Cassandra installation presents a
> large attack surface for a bad actor." … "Enabling authentication for clients using the binary
> protocol is not sufficient to protect a cluster."
> — [cassandra.apache.org security](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/security.html)

> "It cannot list directories, and currently has no provisions for user authentication." … "Since
> TFTP includes no login or access control mechanisms, care must be taken in the rights granted to a
> TFTP server process so as not to violate the security of the server hosts file system."
> — [RFC 1350, The TFTP Protocol (Revision 2)](https://www.rfc-editor.org/rfc/rfc1350.txt), §1 and Security Considerations

> "TFTP has no mechanism for access control within the protocol, and there is no protection from a
> man in the middle attack." … "In summary, use of TFTP is strongly discouraged except in the most
> limited of circumstances where memory and CPU are at the highest premium."
> — [RFC 3617](https://www.rfc-editor.org/rfc/rfc3617.txt), §5, a section titled "Security Considerations and Concerns about TFTP's use"

**Class B.**

> "The Telnet protocol normally uses passwords in the clear for authentication, and normally offers
> no privacy. In normal telnet, both the user's identity and their password are exposed without any
> protection; after that, the contents of the entire Telnet session is exposed without any
> protection."
> — [RFC 4248, The telnet URI Scheme](https://www.rfc-editor.org/rfc/rfc4248.txt), §3

> "Standard FTP [PR85] sends passwords in clear text using the "PASS" command."
> — [RFC 2577, FTP Security Considerations](https://www.rfc-editor.org/rfc/rfc2577.txt), §5. §6 adds: "All data and control information (including passwords) is sent across the network in unencrypted form by standard FTP [PR85]."

For 512, 513 and 514 the attestation is IANA's own registry descriptions, which self-document the
trust model — 512/tcp `exec` is "remote process execution; authentication performed using passwords
and UNIX login names", and 513/tcp `login` is "remote login a la telnet; automatic authentication
performed based on priviledged port numbers and distributed data bases which identify
"authentication domains"" (IANA's typo preserved). RFC 1282's entire Security Considerations section
defers to its "A Cautionary Tale":

> "The rlogin protocol (as commonly implemented) allows a user to set up a class of trusted users
> and/or hosts which will be allowed to log on as himself without the entry of a password. While
> extremely convenient, this represents a weakening of security that has been successfully exploited
> in previous attacks on the internet." … "Bypassing password authentication from trusted hosts
> opens ALL the systems so configured when just one is compromised."
> — [RFC 1282, BSD Rlogin](https://www.rfc-editor.org/rfc/rfc1282.txt)

Note carefully what that does **not** say: RFC 1282's objection is to **host-based trust
delegation**, not to cleartext credentials. The rows for 513 and 514 are worded accordingly. The
vendor position is OpenBSD's, expressed by deletion rather than prose — `rlogin(1)`, `rlogind(8)`
and `rexecd(8)` were removed in [OpenBSD 3.2](https://www.openbsd.org/plus32.html) and `rsh(1)` in
[OpenBSD 5.6](https://www.openbsd.org/plus56.html) ("Removed rsh(1)"), and all three man pages
return 404 on man.openbsd.org today.

> "The RFB protocol as defined here provides no security beyond the optional and cryptographically
> weak password check described in Section 7.2.2. In particular, it provides no protection against
> observation of or tampering with the data stream. It has typically been used on secure physical or
> virtual networks."
> — [RFC 6143, The Remote Framebuffer Protocol](https://www.rfc-editor.org/rfc/rfc6143.txt), §9 (the whole section). §7.2.2 adds that VNC authentication "is known to be cryptographically weak and is not intended for use on untrusted networks", and §7.2.1 defines a "None" security type: "No authentication is needed."

The claim is scoped to the RFB protocol's security type 2, not to any particular product. Vendors
have since moved on — RealVNC scopes the DES/8-character weakness to a non-default "Legacy" mode —
so a row asserting "VNC passwords are limited to 8 characters" would be contradicted by the vendor.
RFC 6143 §9 carries the row instead, because it is a statement about the protocol on the wire.

> "The cookie is transmitted on the network without encryption, so there is nothing to prevent a
> network snooper from obtaining the data and using it to gain access to the X server."
> — [Xsecurity(7)](https://man.openbsd.org/Xsecurity.7), on MIT-MAGIC-COOKIE-1, which the same page lists as "Shared plain-text "cookies"". [xhost(1)](https://www.x.org/releases/current/doc/man/man1/xhost.1.xhtml) describes host-based access control as "a rudimentary form of privacy control and security … only sufficient for a workstation (single user) environment".

Two things frequently attributed to the X11 documentation are **not in it**, were checked, and are
not used here: there is no sentence calling `xhost` "dangerous", and `-nolisten tcp` being the
default is distribution and build-time behaviour that X.Org does not document as a default.

**Class C.**

> "It is also recommended to ensure that it is reachable only from a trusted network or VPN." …
> "That means anyone with the keys can give any instructions to your Docker daemon, giving them root
> access to the machine hosting the daemon."
> — [Docker Engine security](https://docs.docker.com/engine/security/) and [Protect the Docker daemon socket](https://docs.docker.com/engine/security/protect-access/). From the [dockerd reference](https://docs.docker.com/reference/cli/dockerd/): "If you are binding to a TCP port, anyone with access to that port has full Docker access; so it's not advisable on an open network."

> "MySQL uses port 3306 by default. This port should not be accessible from untrusted hosts." … "Put
> MySQL behind the firewall or in a demilitarized zone (DMZ)."
> — [MySQL 8.4 Security Guidelines](https://dev.mysql.com/doc/refman/8.4/en/security-guidelines.html)

> "Install databases in the secure zone of the corporate intranet and don't connect your SQL Server
> instances directly to the Internet."
> — [Microsoft Learn, Security considerations for a SQL Server installation](https://learn.microsoft.com/en-us/sql/sql-server/install/security-considerations-for-a-sql-server-installation)

> "MongoDB binaries, `mongod` and `mongos`, bind to `localhost` by default." … "Make sure that your
> `mongod` and `mongos` instances are only accessible on trusted networks."
> — [MongoDB security hardening](https://www.mongodb.com/docs/manual/core/security-hardening/). The 27017 / 27018 shard member / 27019 config server assignments are from [configuration options](https://www.mongodb.com/docs/manual/reference/configuration-options/)

> "Never expose an unprotected node to the public internet. If you do, you are permitting anyone in
> the world to download, modify, or delete any of the data in your cluster."
> — [Elasticsearch networking](https://www.elastic.co/guide/en/elasticsearch/reference/8.19/modules-network.html)

> "Transport connections between Elasticsearch nodes are security-critical and you must protect them
> carefully." … "A malicious actor who can establish a transport connection might be able to invoke
> system-internal APIs, including APIs that read or modify cluster data."
> — [Elastic, Secure cluster communications](https://www.elastic.co/docs/deploy-manage/security/secure-cluster-communications)

> "If you expose the distribution port to the Internet or any other untrusted network, then the only
> thing protecting you is the Erlang cookie."
> — [CouchDB cluster setup](https://docs.couchdb.org/en/stable/setup/cluster.html). CouchDB's `[chttpd] bind_address` defaults to `127.0.0.1` per [config/http](https://docs.couchdb.org/en/stable/config/http.html)

> "It is important to only expose these ports to the hosts and subnets that run other cluster nodes,
> or where CLI tools are used, and not exposed to the public Internet." … "Unless external
> connections on these ports are really necessary … these ports should not be publicly exposed."
> — [RabbitMQ networking](https://www.rabbitmq.com/docs/networking), covering 4369 (epmd) and 25672 (inter-node)

> "It is unlikely that any SMB communication originating from the internet or destined for the
> internet is legitimate."
> — [Microsoft, Preventing SMB traffic from lateral connections and entering or leaving the network](https://support.microsoft.com/en-us/topic/preventing-smb-traffic-from-lateral-connections-and-entering-or-leaving-the-network-c0541db7-2244-0dce-18fd-14a3ddeb282a), which tabulates SMB TCP 445, NetBIOS Name Resolution UDP 137, NetBIOS Datagram Service UDP 138 and NetBIOS Session Service TCP 139

This is the strongest single sentence in the entire corpus, and it is worth noticing *why*: it is a
statement about legitimacy, not about risk. Every other vendor sentence gathered here is about
consequences. Microsoft's is about whether the traffic has any business existing — which is exactly
the question this list asks.

Microsoft names a carve-out in the next breath, and it is worth quoting because an informed reader
will raise it:

> "The primary case might be for a cloud-based server or service such as Azure Files. You should
> create IP address-based restrictions in your perimeter firewall to allow only those specific
> endpoints." … "Organizations can allow port 445 access to specific Azure Datacenter and O365 IP
> ranges to enable hybrid scenarios in which on-premises clients (behind an enterprise firewall) use
> the SMB port to talk to Azure file storage."

The carve-out does not weaken the row, because it is about **outbound** access to named IP ranges —
an on-premises client reaching Azure Files. This signal is about an **inbound listener** reachable
from an internet vantage, which is the case Microsoft's "unlikely … legitimate" sentence covers
without qualification. Worth recording because the distinction between the two directions is exactly
the kind of thing a hasty reading collapses.

> "Block TCP port 445 inbound from the internet at your corporate hardware firewalls."
> — [Microsoft, Secure SMB traffic in Windows Server](https://learn.microsoft.com/en-us/windows-server/storage/file-server/smb-secure-traffic)

CISA corroborates those four rows, and attaches an honest caveat worth carrying:

> "US-CERT recommends that users and administrators consider: disabling SMBv1 and blocking all
> versions of SMB at the network boundary by blocking TCP port 445 with related protocols on UDP
> ports 137-138 and TCP port 139, for all boundary devices." … "US-CERT cautions users and
> administrators that disabling or blocking SMB may create problems by obstructing access to shared
> files, data, or devices. The benefits of mitigation should be weighed against potential
> disruptions to users."
> — [CISA/US-CERT, SMB Security Best Practices](https://www.cisa.gov/news-events/alerts/2017/01/16/smb-security-best-practices)

> "Typically, file data and user ID values appear unencrypted (i.e. "in the clear") on the network.
> Moreover, NFS versions 2 and 3 use separate sideband protocols for mounting, locking and unlocking
> files, and reporting system status of clients and servers. These auxiliary protocols use no
> authentication." … "This is an easy system to spoof, but on a trusted physical network between
> trusted hosts, it is entirely adequate."
> — [nfs(5)](https://man7.org/linux/man-pages/man5/nfs.5.html), SECURITY CONSIDERATIONS, on the default `sys` auth flavour

That second sentence is the cleanest "trusted-network protocol" statement in the whole corpus: it
concedes the spoofing weakness and then names the exact deployment condition under which it is
acceptable. The condition is the negation of an internet vantage.

> "Do not expose a cleartext daemon to an untrusted network: front it with a TLS proxy (see the
> SSL/TLS Daemon Setup section below) or run it over ssh." … "A module without "auth users" is
> reachable by anyone who can reach the port."
> — [rsyncd.conf(5), upstream](https://download.samba.org/pub/rsync/rsyncd.conf.5)

Cited to the upstream Samba copy rather than a distribution rendering, because the two have
diverged: the older "128 bit MD4 based challenge response system … fairly weak protection" wording
still appears in some renderings but has been replaced upstream by SHA-512 negotiation text.

For SNMP and IPMI:

> "Protocol operations via SNMPv1 and SNMPv2c message wrappers support only trivial authentication
> based on plain-text community strings and, as a result, are fundamentally insecure."
> — [RFC 3410](https://www.rfc-editor.org/rfc/rfc3410.txt), §8.2. The same section supplies the counterweight that stops this being overclaimed: "the IETF standards process does not control actions of vendors or users who may choose to promote or deploy historic protocols, such as SNMPv1 and SNMPv2c, in spite of known short-comings." (RFC 3414 is the wrong citation here — it defines the threat model and never mentions community strings.)

> "Segregate SNMP traffic onto a separate management network. Management network traffic should be
> out-of-band"
> — [CISA TA17-156A, Reducing the Risk of SNMP Abuse](https://www.cisa.gov/news-events/alerts/2017/06/05/reducing-risk-snmp-abuse)

> "Restrict IPMI traffic to trusted internal networks. Traffic from IPMI (usually UDP port 623)
> should be restricted to a management VLAN segment with strong network controls."
> — [CISA TA13-207A, Risks of Using the Intelligent Platform Management Interface (IPMI)](https://www.cisa.gov/news-events/alerts/2013/07/26/risks-using-intelligent-platform-management-interface-ipmi), which also lists among the risks: "Passwords for IPMI authentication are saved in clear text."

And the vendor statement, which is what actually carries the row:

> "DRAC's are intended to be on a separate management network; they are not designed nor intended to
> be placed on or connected to the internet."
> — Dell, published in [CERT/CC VU#843044](https://www.kb.cert.org/vuls/id/843044)

> "These out of band interfaces should never be directly accessible via the public internet."
> — [CISA BOD 23-02 implementation guidance](https://www.cisa.gov/news-events/directives/bod-23-02-implementation-guidance-mitigating-risk-internet-exposed-management-interfaces), corroborating

One thing deliberately **not** claimed: there is no CERT/CC note for the RAKP pre-authentication
password-hash disclosure. VU#163057 covers a different RAKP flaw (session-ID predictability), and no
VU# should be attributed to the hash-disclosure class.

IANA registers 623/**udp** as `asf-rmcp` — "ASF Remote Management and Control Protocol" — and
623/**tcp** as `oob-ws-http`, the DMTF out-of-band web services management protocol. They are
different protocols on the same number, and §6 shows this has already caused a concrete error.

> **Amended by §16** ([#76](https://github.com/winniel123/verge-asm/issues/76)). **Four rows in this
> section are attested by a document it does not cite, and one is attested by a party that does not
> own the port.** Added rather than substituted, per the name-and-withdraw convention; the quotes
> above stand and §16 says what each is now doing.
>
> **etcd — the owner does state a position, and it is not on the website.** The two sentences above
> are a **consequence** and a **third party's** (kubernetes.io, §10.5). etcd's position is in the bytes
> it ships:
>
> > "etcd Server assumes it is deployed within a strictly isolated, private network segment. It
> > **must not** be exposed to untrusted networks or the public internet." … "etcd clients communicate
> > with etcd Servers over Port 2379." … "etcd Server members communicate with other cluster members
> > over Port 2380 to run Raft consensus."
> > — `THREAT_MODEL.md`, [etcd-io/etcd](https://github.com/etcd-io/etcd/blob/v3.7.1/THREAT_MODEL.md) `v3.7.1` (present at `v3.7.0`; **absent at `v3.6.14` and `v3.5.33`** — §16.9)
>
> **Elasticsearch — *"Never expose an unprotected node"* reaches 9300, and the owner's own page makes
> the link.** The warning sits in the file that configures both interfaces, which defines the node as
> its two interfaces and whose `network.host` *"Sets the address of this node for **both HTTP and
> transport traffic**"*, with `transport.port` *"Defaults to `9300-9400`"*:
>
> > "Each Elasticsearch node has two different network interfaces. Clients send requests to
> > Elasticsearch's REST APIs using its HTTP interface, but nodes communicate with other nodes using
> > the transport interface." … "By default Elasticsearch binds only to `localhost` which means it
> > cannot be accessed remotely." … "**Never expose an unprotected node to the public internet.**"
> > — `docs/reference/elasticsearch/configuration-reference/networking-settings.md`, [elastic/elasticsearch](https://github.com/elastic/elasticsearch/blob/v9.5.1/docs/reference/elasticsearch/configuration-reference/networking-settings.md) `v9.5.1`
>
> **kubelet — the CLI-reference quotes above are from a generated page the shipped source
> contradicts.** **[measured]** `--anonymous-auth` defaults to `false`, `--authorization-mode` to
> `Webhook` and `--read-only-port` to `0`, in `kubernetes/kubernetes` `v1.34.1`. That bears on **Claim
> 1** and is routed to [#83](https://github.com/winniel123/verge-asm/issues/83), which **blocks
> [#12](https://github.com/winniel123/verge-asm/issues/12)**; §3.4's parenthetical *"the 10255 default
> verified directly against the rendered page"* should be read as the hazard §11.9 records rather than
> as a verification. What carries the two footings is:
>
> > "`| TCP | Inbound | 10250 | Kubelet API | Self, Control plane |`"
> > — `content/en/docs/reference/networking/ports-and-protocols.md`, [kubernetes/website](https://github.com/kubernetes/website/blob/release-1.34/content/en/docs/reference/networking/ports-and-protocols.md) `release-1.34`. **`10255` does not appear in this file at all.**
>
> > "readOnlyPort is the read-only port for the Kubelet to serve on with no authentication/authorization." … "Setting this field to 0 disables the read-only service." … "**Default: 0 (disabled)**"
> > — `staging/src/k8s.io/kubelet/config/v1beta1/types.go`, [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes/blob/v1.34.1/staging/src/k8s.io/kubelet/config/v1beta1/types.go) `v1.34.1`
>
> **epmd — RabbitMQ does not own it, and Erlang/OTP does.** The RabbitMQ sentence above carries
> `25672`, which is RabbitMQ's own port, and **corroborates only** for `4369` (§10.5, §12(c), §16.6).
> Erlang/OTP's own words:
>
> > "Starting a distributed node without also specifying `-proto_dist inet_tls` will expose the node to
> > attacks that may give the attacker complete access to the node and by extension the cluster. **When
> > using insecure distributed nodes, make sure that the network is configured to keep potential
> > attackers out.**"
> > — `system/doc/reference_manual/distributed.md`, [erlang/otp](https://github.com/erlang/otp/blob/OTP-29.0.5/system/doc/reference_manual/distributed.md) `OTP-29.0.5`
>
> > "The `epmd` daemon accepts messages from both the local host and remote hosts. However, only the
> > query commands are answered (and acted upon) if the query comes from a remote host." … "**To
> > restrict access further, firewall software must be used.**"
> > — `erts/doc/references/epmd_cmd.md`, [erlang/otp](https://github.com/erlang/otp/blob/OTP-29.0.5/erts/doc/references/epmd_cmd.md) `OTP-29.0.5`, section *Access Restrictions*
>
> Neither Erlang/OTP sentence names `4369`, which is why §16.9 flags this as the table's thinnest cell.
> **[measured]** No cookie is involved in epmd's remote exchange, so §3.1's *"why"* cell for the row
> describes the distribution transport's protection rather than epmd's — routed to
> [#84](https://github.com/winniel123/verge-asm/issues/84).

---

## 4. The boundary cases

### 4.1 Management planes that are routine inside a VPC

The ticket names this as where the list earns or loses trust: kubelet 10250, Docker 2375/2376 and
etcd 2379 are unambiguously wrong on the public internet and completely routine inside a VPC, and
[#14](https://github.com/winniel123/verge-asm/issues/14) established that "internal" and "external"
are vantage-relative.

**This is not a problem for the list, and the reason is worth stating precisely: the list never sees
a vantage-relative question, because `Exposure` has already absorbed it.** #14 defined `Exposure` as
a derived conclusion whose definition *contains its own precondition* — reachable from an internet
vantage, where an internet vantage is one verified to be outside every range the operator owns. Its
states are:

| state | meaning |
|---|---|
| `exposed` | reachable from an internet vantage |
| `firewalled` | reachable internally, **checked** from an internet vantage, not reachable there |
| `internal-only` | reachable internally, **never checked** from an internet vantage |
| `edge-only` | reachable from the internet but not internally |
| `unreachable` | no vantage reaches it |

A kubelet reachable only from inside the operator's VPC derives `firewalled` or `internal-only`. It
never derives `exposed`, so the signal never fires on it, and the list did not have to know anything
about VPCs to get that right. **The vantage does the relativising; the list is free to be absolute.**
That division is what lets §3 state "never correct" without hedging.

Two consequences fall out, and both are load-bearing:

**The signal must fire on `edge-only` as well as `exposed`.** Both are "reachable from an internet
vantage". A naive implementation testing `state == exposed` would silently miss `edge-only` — which
is the *more* alarming case, because an etcd reachable from the internet but not from inside the
operator's own network is an asset that is not where they think it is. This is an easy bug to write
and it fails in the quiet direction.

**`internal-only` must never be reported as clean.** It means *reachable internally, never checked
from an internet vantage* — absence of evidence, not evidence of absence, which #14 was explicit
about. For this signal that maps directly onto ADR-0004's `not-evaluable`. An operator running
verge-asm inside their own VPC with no external prober has a deployment that structurally cannot
evaluate this signal, and must be told so rather than shown a clean board.

### 4.2 Ports where TLS appears to change the verdict

The ticket's example is 5432 plain versus 5432 behind a TLS proxy. The general answer:

**TLS never changes a verdict, because two of the three claims are not about confidentiality.** TLS
bears on Claim 2 and on nothing else. It has no bearing on whether the protocol authenticates
(Claim 1) or on who its intended clients are (Claim 3). So:

- **A Claim 2 row is on the list precisely because an encrypted successor exists on a different
  port.** That is why the list is keyed on `(port, transport)` and not on services: 23 is listed and
  22 is not; 21 is listed and 990 is not. The encrypted sibling is not an exception to the rule, it
  *is* the rule — its existence is what makes the plaintext port wrong.
- **A Claim 3 row is unaffected by TLS.** 5432 wrapped in TLS is still a database wire protocol
  whose clients are application tiers. The confidentiality of the channel is not the objection; the
  audience is. A TLS-terminating proxy in front of 5432 that still speaks the PostgreSQL frontend
  protocol to the internet has changed nothing that the claim depends on.
- **2375 and 2376 are both listed, and the pair is the clean demonstration.** They differ only in
  TLS. 2375 is listed under Claim 1 (Docker's own deprecation of unauthenticated TCP). 2376 is
  listed under Claim 3, and Docker's own documentation says so directly even for the TLS case: it is
  "recommended to ensure that it is reachable only from a trusted network or VPN", and "anyone with
  the keys can give any instructions to your Docker daemon, giving them root access to the machine
  hosting the daemon". TLS moved 2376 from one class to another. It did not move it off the list.

The residual case — an operator who has genuinely put an authenticating gateway in front of a listed
port — is an `Annotation`, not a list change. That mechanism already exists and is the right place
for operator opinion about a specific subject.

> **Amended by §14** ([#75](https://github.com/winniel123/verge-asm/issues/75)). This section is
> **unchanged** and gains a third instance of the pattern §9.2 and §11.5 record, which is the one
> that limits it. A split pair only exists to be judged where the owner keeps the split: Apache
> Cassandra **collapsed** `7000`/`7001` at 4.0 — *"a single port can be used for either/both secure
> and insecure connections"*, in the shipped `conf/cassandra.yaml` — so `7001` is a **withdrawn
> sibling** rather than an encrypted successor, and §4.2 is not reached for that pair at all. Stated
> because the ticket asked for the answer rather than the assumption; had it been reached, the answer
> would be the one above. §14.5.

**An objection worth surfacing, because it comes from the body that assigns the numbers.** Claim 2
depends on the split-port pattern — plaintext on one number, TLS on another — and RFC 6335, the
document governing the registry itself, says that pattern should not exist:

> "Services are expected to include support for security, either as default or dynamically
> negotiated in-band. The use of separate service name or port number assignments for secure and
> insecure variants of the same service is to be avoided in order to discourage the deployment of
> insecure services."
> — [RFC 6335](https://www.rfc-editor.org/rfc/rfc6335.txt), §9

This does not undermine the list; it explains it. The IETF's objection is to *creating* new split
pairs, and the pairs this list relies on (23/22, 21/990, 2375/2376, 5985/5986, 110/995, 143/993) all
predate that guidance. But it does mean the pattern will not extend to newer protocols, so Claim 2
is a closing category rather than a growing one — which is a useful thing to know about a list whose
revision cost is the subject of §7.2.

### 4.3 High ports that are conventionally anything

Handled entirely by the determinacy gate (§2.4), and it is worth being explicit that the gate is on
the **port**, not on the service. **Excluded, regardless of what may be running there: 8080, 8000,
8888, 8443, 3000, 5000, 9000, 9090, 8088, 10000.**

Two of these deserve their reasoning on the record because a strong upstream quote exists and is
*still* not enough:

- **9090.** Prometheus's own security model page says "the HTTP endpoints provided by Prometheus
  components should not be exposed to publicly accessible networks like the internet (unless you
  know what you are doing and have taken appropriate measures)"
  ([prometheus.io](https://prometheus.io/docs/operating/security/)). That is a genuine Claim 3
  attestation. 9090 is still excluded, because 9090 is conventionally Cockpit, Openfire's admin
  console, and a generic HTTP alternate, and `nmap-services` still lists it as `zeus-admin`. A signal
  firing "Prometheus exposed" on a Cockpit host is worse than no signal. Note also the hedge in that
  sentence — "unless you know what you are doing" — which is a conditional position, not an absolute
  one.
- **9100.** node_exporter squats on a registration belonging to `hp-pdl-datastr`, "PDL Data Streaming
  Port" — i.e. JetDirect printers. One port, two completely different services, opposite
  populations. Excluded.

The ticket's own example holds: **8443 is rare and perfectly fine**, and its rarity is exactly why a
frequency instrument would have got it wrong in both directions.

> **Amended by §14** ([#75](https://github.com/winniel123/verge-asm/issues/75)). **Also excluded:
> `7000/tcp` and `7001/tcp`**, and they are not on the enumerated generic list above — they are the
> `9100` case rather than the `8080` case. `7000` squats on `afs3-fileserver` while Apple documents
> `AirPlay · 7000 · TCP`; `7001` squats on `afs3-callback`, carries Oracle's WebLogic AdminServer as
> a documented HTTP admin console, and its Cassandra usage is **version-dependent** — deprecated at
> 4.0 with `legacy_ssl_storage_port_enabled: false` shipped — which is §2.4's Hadoop limb. The rule
> separating this paragraph's `9100` from §3.3's `9200` is now written down as
> [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md). §14.3, §14.4.

> **Amended by §15** ([#82](https://github.com/winniel123/verge-asm/issues/82)). **`9090`'s ground
> moves and its verdict does not.** *"`nmap-services` still lists it as `zeus-admin`"* is a catalogue,
> and [ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md) limb 5
> makes a catalogue **corroboration and never grounds**. The exclusion does not need it: Red Hat's own
> current RHEL 8 and RHEL 9 web-console documentation states that the console *communicates through
> TCP port 9090*, which is a first-party **placement statement** for Cockpit — a different protocol
> with no compatibility declared either way. This paragraph's own sentence *"9090 is conventionally
> Cockpit"* was right and is now footed. **The ten generic ports above are grandfathered as a class**,
> and each owes its artefact the first time it is individually relied on, per the refusal artefact
> rule. §15.5.

### 4.4 The closest call: 6443, kube-apiserver — excluded

This is the single hardest exclusion and the one most likely to be challenged, so the working is
shown in full.

**The evidence for listing it is strong and comes from two independent normative sources.** NSA and
CISA:

> "The Kubernetes API server runs on port 6443, which should be protected by a firewall to accept
> only expected traffic. The Kubernetes API server should not be exposed to the Internet or an
> untrusted network."
> — [NSA/CISA Kubernetes Hardening Guidance v1.2, August 2022](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF)

And Kubernetes upstream, more sweepingly:

> "The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet."
> — [kubernetes.io security checklist](https://kubernetes.io/docs/concepts/security/security-checklist/)

**It is excluded anyway, on three grounds.**

1. **Claim 3 fails on the facts.** The API server's intended clients include human operators and CI
   systems running `kubectl` from arbitrary networks, authenticated with client certificates or
   OIDC. That is not a misconfiguration; it is the default topology of every major managed
   Kubernetes offering. Upstream itself concedes the point in the sentence immediately following the
   checklist item above: "Be careful, as **many managed Kubernetes distributions are publicly
   exposing the API server by default**." A protocol whose dominant vendor-supported deployment is
   internet-reachable does not satisfy "the intended clients are same-system components".
2. **Determinacy fails.** IANA registers 6443/tcp as `sun-sr-https`. More awkwardly, Kubernetes'
   own documentation notes that "In a typical production Kubernetes cluster, the API serves on port
   443" ([controlling access](https://kubernetes.io/docs/concepts/security/controlling-access/)), so
   the port most associated with the service is not reliably the port the service is on.
3. **The product cost of being wrong here is asymmetric and severe.** If the signal fires on every
   managed Kubernetes control plane in the operator's estate and the correct answer is "yes, that is
   how EKS works", it has cried wolf on its most visible firing. The list's entire value is that a
   firing is never arguable.

**What the two quotes actually are is a hardening preference expressed against a real, supported
architecture.** That is precisely the "rarely correct, worth mentioning" shape that §5 declines to
build. 6443 is therefore the concrete case that makes the middle-band question real rather than
theoretical — and the case that shows why answering it with a band would not have helped, since what
6443 needs is not a softer verdict but a *measurement* (is anonymous access permitted?), which is
§5's actual answer.

### 4.5 The list's weakest row, named

**5432/tcp.** PostgreSQL is the one service surveyed whose upstream documentation states **no
position at all** on network placement. `ssl-tcp.html` is purely mechanical and contains no sentence
about untrusted networks; `client-authentication.html` treats accepting remote connections as an
ordinary configuration; and `server-start.html` frames failure to listen on the network as a
*mistake*: "A common mistake is to forget to configure listen_addresses so that the server accepts
remote TCP connections."

5432 is on the list on the strength of the shipped default (`listen_addresses` = `localhost`) plus
Claim 3, corroborated by AWS Trusted Advisor's red band and AWS FSBP EC2.19. Under §2.3 that
corroboration cannot carry the row alone, and it does not — the default carries it.

This is disclosed rather than buried because the alternative was worse. Excluding 5432 would have
produced a list that flags MySQL and not PostgreSQL for reasons that exist in documentation rather
than in the world, and an operator who noticed would be right to distrust the whole list. **The
criterion that would change the verdict:** if PostgreSQL upstream ever documents a supported
direct-to-internet deployment pattern, the row should be reconsidered.

### 4.6 Excluded, with reasons — the negative space

The exclusions are as much of the deliverable as the list, because a curated list is judged on what
it refuses.

| Excluded | Why |
|---|---|
| **22/tcp SSH** | Remote administration over an untrusted network is the protocol's express purpose. GCP ships it open to `0.0.0.0/0` by default |
| **3389/tcp RDP** | Same category. Microsoft's position is **Azure-scoped** ("Disable direct RDP and SSH access to your Azure virtual machines from the internet") and no first-party non-Azure prohibition was found; CISA calls it "high-risk". Both are risk positions, not legitimacy positions. GCP ships it open by default |
| **5985, 5986/tcp WinRM** | See below — the obvious argument for listing 5985 is factually wrong |
| **9100/tcp** | Squats on `hp-pdl-datastr`, and HP's own best-practices document says the opposite of what one would assume: "9100 Printing should always be enabled. It is the standard printing protocol used by MFP print drivers." The print path HP tells you to disable is IPP, not 9100 |
| **111/tcp rpcbind** | §2.7 — the only available authority is a frequency source, and the protocol's owners state nothing |
| **389/tcp LDAP** | RFC 4513 §6.3.3 discourages cleartext credentials "unless the data on the session is protected using TLS" — so 389 with StartTLS is correct, and the port is not the discriminator |
| **79/tcp finger** | RFC 1288's position is conditional ("should not run Finger without an explicit understanding of how much information it is giving away"), and finger was designed as an internet-facing service, so Claim 3 fails |
| **6443/tcp kube-apiserver** | §4.4 |
| **5601/tcp Kibana** | Elastic states no prohibition; Kibana is routinely and legitimately fronted on the internet behind auth. Its evidence is a secure default only, and it squats on `esmagent` |
| **8500/tcp Consul HTTP API** | Consul "is not secure-by-default" and ships `acl.default_policy` = `"allow"`, but its stated position is only that external access "should be considered", and 8500 is registered to `fmtp` |
| **9092/tcp Kafka** | Upstream declines to take any network posture. Its only relevant sentence is neutral: "security is optional - non-secured clusters are supported" |
| **5672, 15672/tcp RabbitMQ AMQP + management UI** | Upstream's "should not be publicly exposed" sentence covers 4369 and 25672 specifically, not these. AMQP brokers are sometimes legitimately public |
| **8080/tcp Jenkins** | Upstream explicitly acknowledges public-internet deployment as supported: "Jenkins is used everywhere from workstations on corporate intranets, to high-powered servers connected to the public internet", and responds with auth-by-default rather than a network demand |
| **1099/tcp Java RMI registry** | Contrary to reputation, modern JDK defaults are secure — `jmxremote.ssl` and `jmxremote.authenticate` both default to `true` |
| **Hadoop NameNode / YARN UIs** | Hadoop's default `simple` authentication would qualify under Claim 1, but the web UI port moved from 50070 to 9870 between major versions, so port-to-service inference is version-dependent. Fails determinacy |
| **110/tcp POP3, 143/tcp IMAP, 25/tcp SMTP** | Mail protocols whose intended audience genuinely is the internet. Cleartext variants are deprecated, but a server on 143 offering STARTTLS is correct, so the *port* is not the discriminator |

> **Amended by §14** ([#75](https://github.com/winniel123/verge-asm/issues/75)). **Two rows are added
> to this table and it is now eighteen: `7000/tcp` and `7001/tcp`, Cassandra's internode storage port
> and its deprecated TLS twin.** They are unlike every other entry above, and the difference is the
> point: for `7000` the **claim and the attestation both pass** — Claim 1 on both of §10.1's steps,
> Claim 3 with §10.3's boundary limb named by the owner, and a first-party prohibition naming the
> port by number in operative shipped bytes at three release tags. The refusal is **determinacy
> alone**. Every other exclusion here fails a claim, fails to find an owner, or finds an owner
> pointing the other way; these two are the first to fail only §2.4. Grounds and the criteria that
> would change either verdict are in §14.8.

**The WinRM case deserves its own paragraph, because the tempting argument is factually false.**
5985 is the WinRM *HTTP* listener, and the natural inference — HTTP transport, therefore cleartext
credentials, therefore Claim 2 — is contradicted by Microsoft directly:

> "Regardless of the transport protocol used (HTTP or HTTPS), WinRM always encrypts all PowerShell
> remoting communication after initial authentication."
> — [Security considerations for PowerShell Remoting using WinRM](https://learn.microsoft.com/en-us/powershell/scripting/security/remoting/winrm-security)

The defensible narrower statement is that 5985 carries no *transport-layer* TLS, so confidentiality
rests on the negotiated authentication protocol — AES-256 under modern Kerberos, RC4-128 under NTLM,
and none at all under Basic, which the same page notes "provides no encryption". That is a real
weakness and it is **not** the same claim, so 5985 does not qualify under Claim 2. Nor is there any
first-party Microsoft sentence prohibiting internet exposure of WinRM; the strongest that exists is
a firewall warning and the fact that no listener is configured by default. Recording this because it
is exactly the reading-laundered-into-a-position failure §2 exists to catch, and it nearly landed a
row on the list.

**Mail ports were checked properly and the nuance holds.** RFC 8314 is the document that would be
cited to condemn 110 and 143, and it cannot be: it governs MUA-to-server submission and access only,
it excludes MTA relay explicitly ("This memo does not address the use of TLS with SMTP for message
relay"), it never names 110 or 143 by number, and it rules that 587-with-STARTTLS and 465-with-
implicit-TLS have "no significant difference" in security properties. Its position is *deprecate
cleartext as soon as practicable* — not *these ports must not be internet-facing*. 25, 110, 143 and
587 all remain legitimately internet-facing.

The exclusions above are the discipline the standard exists to enforce. Several are ports a
practitioner would casually call "never internet-facing", and most sit in the nmap top-100.
Admitting them on the strength of that feeling is exactly how a curated list becomes unfalsifiable.

---

## 5. Does the list have a middle? No — and the reason is not that the middle is empty

The ticket asks whether this is binary or whether it needs a "rarely correct, worth telling the
operator" band, noting that [#16](https://github.com/winniel123/verge-asm/issues/16) rejected
severity, so a band would have to be a second separately-named signal — a real cost.

**Verdict: binary. One signal. No second signal, no band.** Three arguments, in increasing order of
force.

**1. A band is severity with different spelling.** ADR-0004 settled that a signal is a *named fact
with evidence*, and that urgency belongs to the transition that surfaced it. "Rarely correct" is not
a fact about the observed world; it is a prior about a population. Shipping
`sensitive-port-exposed` alongside `rarely-correct-port-exposed` gives the operator two names whose
only difference is our confidence — which is a two-level severity scale wearing a hat. The cost
ADR-0004 accepted (an operator facing many signals has no intrinsic ranking) was accepted knowingly;
this would quietly refund it.

**2. Every candidate for the middle turned out to fail a gate, not to sit between gates.** This is
the empirical finding, and it is the one that actually settles the question. Going through them:
6443, 5985/5986, 8500, 9090, 5601, 5672, 8080, 3389 — each is excluded because in some real,
vendor-supported architecture its intended audience genuinely includes internet clients, not because
it is *sort of* wrong. There was no port in the survey for which the honest verdict was "usually
wrong". The middle band, asked to name its members, could not.

**3. Where a genuine gradation exists, it is measurable rather than curatable — and that is the
whole point.** The cases that create pressure for a middle are ones where the verdict depends on
something we can *observe*: does this Postgres listener require TLS? does this kubelet permit
anonymous requests? does this admin interface demand authentication? Those are facts about the
world, discoverable from an observation, and a signal built on them would be **more** honest than
the curated list — because its reference data would not be curated at all. The v1 set already
contains rules of exactly this shape: *plaintext HTTP with no HTTPS*, *TLS 1.0/1.1 negotiated*.

So the pressure toward a middle is real, and it discharges in a different direction than a band. The
right destination is a future observation-driven rule, not a softer list. That is recorded as a
follow-up in §8 rather than smuggled into v1, because it needs protocol-level probing that
[#4](https://github.com/winniel123/verge-asm/issues/4)'s safety profile does not currently authorise.

**One thing the binary verdict costs, stated plainly.** An operator with an exposed kube-apiserver
gets nothing from this signal. That is a real gap, it is the direct price of refusing the band, and
it should be visible rather than argued away.

---

## 6. Containment: is the list derived from the hot set, or independent?

**Neither. The lists are maintained independently, and the relationship between them is a
one-directional invariant enforced at build time:**

```
every (port, transport) on the sensitive list MUST be a member of the hot set
```

with a test that fails the build otherwise.

**Why not derive it from the hot set.** They answer different questions with incompatible evidence
standards. The hot set's membership rule is *likely to be found open in a small org's estate* —
frequency. The sensitive list's is *never legitimately internet-facing* — position. Selecting the
sensitive list as a subset of the hot set would make frequency a precondition of normativity, which
is precisely the laundering this ticket exists to prevent.

**Why not leave them fully independent.** A sensitive port absent from the hot set **can never fire
the signal at all**. That is the worst available failure mode: the operator believes the signal
covers kubelet, and it is simply never probed. Worse, under a naive implementation it would present
as *did not fire* — a clean bill of health the estate never earned, which is the exact error
ADR-0004's `not-evaluable` rule exists to prevent.

**Why the invariant points this way.** Adding a port to the sensitive list *forces* an addition to
the hot set, never the reverse. That is the correct coupling direction because the two lists have
asymmetric constraints: the hot set is constrained by probe cost, the sensitive list by correctness.
Probing one more port per host per day is cheap; being unable to evaluate the product's best signal
is not. Cost yields to correctness, and the invariant encodes which is which.

### 6.1 The containment gaps that exist right now

Computing the current `verge-core` hot set from
[#4](https://github.com/winniel123/verge-asm/issues/4) §2.3 — nmap top-100 TCP, minus the named
ephemeral/obsolete tail, plus the named modern-services supplement — against §3's list:

**In the hot set already — 28 of 38:** 21, 23, 139, 445, 513, 514, 873, 2049, 2181, 2375, 2376,
2379, 2380, 3306, 5432, 1433, 5900, 5984, 6000, 6379, 9042, 9200, 9300, 10250, 10255, 11211/tcp,
27017, 27018.

**Missing from the hot set — 4 TCP rows that would silently never fire:**

| Missing | Service | Why it slipped |
|---|---|---|
| 512/tcp | rexec | Ranks 239th by nmap open-frequency, so it falls outside the top-100 the hot set was built from — while 513 and 514 are inside it. A frequency-derived base set splits a family the normative question treats as one |
| 4369/tcp | Erlang port mapper (epmd) | Not in top-100, not in the supplement |
| 25672/tcp | RabbitMQ inter-node | The supplement took 5672 and 15672 — the two ports §4.6 *excludes* — and omitted the one the upstream prohibition actually names |
| 27019/tcp | MongoDB config server | The supplement took 27017 and 27018 and stopped one short |

Note what the 25672 row shows: the hot set selected RabbitMQ's *popular* ports and the sensitive
list selects its *indefensible* ones, and they are disjoint. That is the containment argument in a
single example — deriving either list from the other would have produced the wrong answer.

**Missing because the whole transport is off by default — 6 UDP rows:** 69/udp, 137/udp, 138/udp,
161/udp, 623/udp, 11211/udp. [#4](https://github.com/winniel123/verge-asm/issues/4) §2.5 put UDP
off by default on signal-to-cost grounds, and the hot set is TCP-only.

**These must be `not-evaluable`, not clean.** An operator on default settings has six UDP rows on
the sensitive list that are never measured. Reporting those as not-firing would be exactly the
"bill of health it never earned" ADR-0004 forbids. There are two honest options — drop the UDP rows,
or keep them and report `not-evaluable` — and **keeping them is right**, because it makes the
coverage gap visible in the product instead of invisible in a list file. The failure mode to avoid
is shipping them silently.

### 6.2 A concrete error the invariant would already have caught

`verge-core`'s management/OOB group is specified as "161 (TCP), 623". Per IANA:

- **623/udp** is `asf-rmcp`, "ASF Remote Management and Control Protocol" — this is IPMI-over-LAN,
  the BMC interface the row is meant to catch.
- **623/tcp** is `oob-ws-http`, the DMTF out-of-band web services management protocol — a different
  protocol.
- **161/udp** is `snmp`. **161/tcp** is registered but is not where SNMP agents listen in practice.

So the hot set currently probes the TCP siblings of two UDP services and would evaluate neither.
This is not a hypothetical: it is a live defect in the shipped hot set, and it is the argument for
keying **both** lists on `(port, transport)` pairs rather than bare port numbers. A containment check
over bare integers would have passed and the bug would have survived.

### 6.3 Who owns the invariant

The invariant is mechanical, so it wants a test, not a person. But the two lists still need
different owners' *judgement* on revision, because their evidence standards differ — and that is the
governance question the map still carries as fog. This note settles what the sensitive list is and
what admits a row to it; it does not settle who revises it or on what trigger. §8 records that.

---

## 7. `not-evaluable`, and what a revision costs

### 7.1 The four routes to `not-evaluable`

ADR-0004 requires that absent evidence yield `not-evaluable` rather than "did not fire". For this
signal there are four distinct routes, and an implementation that collapses them will under-report:

1. **Ownership.** On `third-party` or `unknown` addresses,
   [ADR-0002](../adr/0002-ownership-gates-probing.md) limits probing to the ports the `Name` implies
   (443, 80). No sensitive port is ever probed there, so **every** row is `not-evaluable` on those
   addresses. For a SaaS-fronted estate that is most of the estate.
2. **No internet vantage.** `Exposure` cannot be constructed at all without one — its definition
   contains that precondition — so the signal composes `Exposure`'s outcome and inherits it. A
   default deployment with no external prober can never evaluate this signal.
3. **Tier cadence.** A sensitive port sitting in the warm (weekly) or cold (monthly, opt-in) tier
   rather than the hot set is unmeasured between probes. This is why §6's invariant targets the
   **hot** set specifically: it is the only tier with a guaranteed cadence.
4. **Transport not probed.** The six UDP rows, per §6.1.

### 7.2 What a revision costs, and the case for stopping here

Per ADR-0004 the rule's version bumps on any list edit and the two evaluations become
non-comparable. **The cost is symmetric between adding and removing a port**, which is the argument
that shaped §2's conservatism: a generous list that later sheds a bad row costs the operator exactly
as much as a tight list that later gains a good one. Given symmetric cost, the tight list is
strictly better, because its errors are the kind an operator can discover (a missing signal) rather
than the kind that teaches them to ignore the product (a wrong one).

Worth recording that **the normative sources themselves move**. Over the period this note covers:
CISA BOD 22-01 was revoked and superseded; CPG 1.0's goal "2.W – No Exploitable Services on the
Internet" was reorganised into CPG 2.0's 3.S; CIS AWS Foundations moved from naming ports 22 and
3389 explicitly (v1.2.0 §4.1/§4.2) to the abstraction "remote server administration ports" (v3.0.0
§5.2 onward); Docker deprecated unauthenticated TCP entirely; and Prometheus *softened* its position
once it gained native TLS and basic auth. A revision policy that assumes the sources are static will
be wrong within a year.

---

## 8. Open questions for the spec

1. **Who revises the sensitive list, on what trigger, and how often?** This note settles membership
   and the admission standard; it does not settle governance. It is the surviving half of the map's
   port-set fog, and it is now sharper: the trigger is not "a new service became popular" (frequency)
   but "a primary source changed its position, or a protocol changed its shipped defaults" — both of
   which are watchable events rather than a continuous obligation.
2. **Should the middle band's real content ship as an observation-driven rule in v1.1?** §5 argues
   the pressure toward a band discharges into rules like *listener offered no TLS* or *anonymous
   access permitted*. Those need protocol-level probing beyond the TLS handshake, which
   [#4](https://github.com/winniel123/verge-asm/issues/4)'s safety profile does not authorise, and
   credentials are never submitted. What can be established without crossing that line is unresolved.
3. **Does the signal fire on `edge-only` as well as `exposed`?** §4.1 argues it must. This should be
   stated in the rule definition rather than left to the implementer, because the failure is silent.
4. ~~**Do the three ports still excluded for want of attestation deserve another pass?**~~
   **Closed by §9** ([#30](https://github.com/winniel123/verge-asm/issues/30)). 111/tcp rpcbind,
   389/tcp LDAP and 79/tcp finger were all re-searched against their specifications, reference
   implementations and shipped defaults. **None is admitted; the list stays at 38.** The pass was
   right to run — the precedent of 2049 and 873 was sound — and it was not a null result, because
   111 and 389 moved from *no evidence found* to *evidence found pointing the other way*.
5. **Should the `reg.` disclosure surface in the product?** Several rows rest on convention rather
   than IANA registration. When the signal fires on 9200 it is asserting Elasticsearch on a port
   registered to `wap-wsp`, and on 623/udp it is asserting IPMI on a number where the string "IPMI"
   appears nowhere in the registry. The evidence the signal cites should probably say so.
6. **Does this note's evidence standard generalise to the other v1 signals?** The claim/attestation/
   determinacy structure was built for the one signal with curated reference data, but "state the
   claim, cite the source that owns it" is not specific to ports.
7. ~~**Should Claim 1's unstated qualifier be written into §2.1?**~~ **Closed by §10.1**
   ([#37](https://github.com/winniel123/verge-asm/issues/37)). The qualifier is written down as a
   two-step test a reviewer applies to the specification, the whole list was re-checked against the
   stated criterion, and **all 38 pairs survive it unchanged**.
8. ~~**Class B has a permanent blind spot — should §2.1 say so rather than leaving it in §9.2?**~~
   **Closed by §10.8** ([#37](https://github.com/winniel123/verge-asm/issues/37)): yes, §2.1 says so
   in one sentence, and the pressure is already discharged **out of scope** — the wire-protocol
   prober and the `listener-negotiation` facet were ruled out of v1 in
   [#41](https://github.com/winniel123/verge-asm/issues/41), which priced this boundary at six ports
   and zero net new firings across all 38 rows.
9. ~~**Is `161/udp`'s Claim 3 boundary attested by an owner?**~~ **Closed by §11**
   ([#66](https://github.com/winniel123/verge-asm/issues/66)). The retrieval was performed over ten
   RFCs and five net-snmp artefacts, **no first-party placement sentence exists**, and the row was
   **removed**. The list is 37 pairs. (Opened by §10.6 on the ground that the boundary limb was
   carried by CISA TA17-156A, a corroborator, while RFC 3410 §8.2 attests the *insecurity* and not
   the *boundary*.)
10. ~~**What evidence establishes that a convention is contested?**~~ **Closed by §15**
    ([#82](https://github.com/winniel123/verge-asm/issues/82)). A **placement statement** does, and
    nothing else: a party's statement, in its own current documentation or its own shipped bytes, that
    its own software listens on a `(port, transport)` pair by default. The unit is the **protocol, not
    the vendor**; *live* means **current**, never numerous; every other class — IANA rows, the
    `Unauthorized Use Reported` field, `nmap-services`, cloud and government port tables, and this
    project's own frequency half — corroborates and never carries; and **every determinacy refusal must
    name the artefact that defeated the convention**.
    [ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md). All nine
    convention-resting rows were walked (§15.4) and **no `(port, transport)` pair moves**; two
    exclusion grounds are restated and both strengthen. The original text follows.

    **What evidence establishes that a convention is contested?** Opened by §14.6. §2.2 has three
    attestation forms and an owner definition (§10.5); §2.1 has a claim set closed by construction
    (§10.2). **§2.4 has *"uncontested convention"* and no account of what establishes it** — and §14
    is the first ruling to rest entirely on that gate, weighing a vendor's product-port table, a
    vendor's container-image README and an IANA annotation against each other with nothing in the
    standard to rank them. [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md)
    settles the *liveness* half and does not supply an evidence standard for the rest. The live
    hazard is symmetrical to §2.3's: a determinacy gate with no source rule can be cleared by
    whichever authoritative-looking document is nearest, or blocked by one. **No row moves on the
    answer** — §14's sources are first-party about their own products, which is the strongest class
    on any plausible standard — so it does not block
    [#12](https://github.com/winniel123/verge-asm/issues/12). Routed to
    [#82](https://github.com/winniel123/verge-asm/issues/82).
11. **Has [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md)'s
    liveness test ever been run on `esmagent` and `fmtp`?** Opened by §17.8. **[measured] No.** Two of
    §4.6's exclusions — `5601/tcp` Kibana and `8500/tcp` Consul — rest on **owner-silence plus a
    squat**, and §17.1 finds they are the only two rows in the note whose exclusion is not
    overdetermined. ADR-0042 made *contested* testable by keying it to the competing owner's current
    documentation, and neither `Enterprise Security Agent` nor `Flight Message Transfer Protocol` has
    been checked for a living owner. If either registration is dead, that row's determinacy gate
    passes and its exclusion rests on owner-silence alone — a sole-ground negative that
    [ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md) then obliges someone
    to sweep by document class. **No row moves on the answer today** and both are currently excluded,
    so it does not block [#12](https://github.com/winniel123/verge-asm/issues/12). Routed to
    [#87](https://github.com/winniel123/verge-asm/issues/87), after
    [#82](https://github.com/winniel123/verge-asm/issues/82).
12. **Is an owner's unreleased document the owner's documentation?** Opened by §17.5. §2.2's second
    form reads *"the project's or vendor's own documentation"* and does not say whether that means
    what the project **publishes** or what sits in its **tree**; §12 answered the equivalent question
    for **configuration** and nothing answers it for **prose**. The first case to turn on it is
    `9092/tcp`, where a first-party sentence exists on `trunk`, is absent from release tag `4.3.1` and
    404s on the published site. **The list is definitively 37 under the standard as written**, so it
    does not block [#12](https://github.com/winniel123/verge-asm/issues/12). Routed to
    [#86](https://github.com/winniel123/verge-asm/issues/86).

---

## 9. Second attestation pass: 111/tcp rpcbind, 389/tcp LDAP, 79/tcp finger

Research ticket [#30](https://github.com/winniel123/verge-asm/issues/30), resolving §8 question 4
before v1 ships. The pass was worth running on precedent: 2049/tcp and 873/tcp were admitted late on
`nfs(5)` and upstream `rsyncd.conf(5)`, both found only on a targeted second look, so absence of a
citation had already proven to be weak evidence of absence of a position.

**Result: none of the three is admitted. The list stays at 38 `(port, transport)` pairs and §1 is
unchanged.** But the pass is not a null result, because for two of the three the footing moved from
*no evidence found* to *evidence found, pointing the other way*. That is a materially stronger place
to leave an exclusion than §2.7 left 111, and it is the difference between a row we could not
justify and a row we can say is wrong.

The pass also surfaced two defects in the standard itself, which are recorded here and carried into
§8 rather than acted on: **Claim 1 is missing a qualifier it has always relied on** (§9.3.3), and
**Class B's successor-on-another-port clause turns out to be the load-bearing half of the claim
rather than a scoping detail** (§9.2).

Every quote below was retrieved with `curl` and verified against the retrieved bytes. The retrieval
hazards, and the one widely-repeated claim about RFC 4513 that turns out to be simply false, are
disclosed in §9.5.

### 9.1 111/tcp rpcbind — not admitted, and now on positive grounds

§2.7 excluded 111 because *"we could not find anyone entitled to say it isn't"* defensible. That
survives re-checking, and it understates the result. The protocol's owners do speak; what they say
is that the exposure problem was answered **inside the software**, and that the port answers from
every interface by design.

**The two confirmed non-statements hold.** RFC 1833's Security Considerations section reads, in its
entirety, *"Security issues are not discussed in this memo."* (re-verified against
[the retrieved text](https://www.rfc-editor.org/rfc/rfc1833.txt)). The upstream `rpcbind(8)` roff
source has `NAME`, `SYNOPSIS`, `DESCRIPTION`, `OPTIONS`, `FILES`, `NOTES`, `SEE ALSO` and
`LINUX PORT` — and no `SECURITY` section.

**But the man page does contain a security sentence, and it cuts against the row.** From the
`-i` option:

> "'Insecure' mode. Allow calls to SET and UNSET from any host. Normally rpcbind accepts these
> requests only from the loopback interface for security reasons."
> — [`rpcbind(8)`, upstream `man/rpcbind.8`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=man/rpcbind.8;hb=HEAD)
> (quoted from the rendered form; cross-checked against
> [Debian's rendering](https://manpages.debian.org/unstable/rpcbind/rpcbind.8.en.html), which agrees verbatim)

Read carefully, that is the maintainers drawing the line themselves and putting it in a different
place than a row would need. The **mutating** operations — the ones that let a caller register or
unregister an RPC service — are loopback-only in the shipped default, "for security reasons", in
the project's own words. What remains answerable to a remote client is the lookup path.

**Upstream then removed the remaining internet-facing hazard by disabling it at build time.** The
indirect-call machinery (`RPCBPROC_CALLIT` / `RPCBPROC_INDIRECT`, RFC 1833 §§511-573) is the
proxying and amplification surface that CISA TA14-017A measured. It is off in a default build:

> `AC_ARG_ENABLE([rmtcalls],`
> `  AS_HELP_STRING([--enable-rmtcalls], [Enables Remote Calls @<:@default=no@:>@]))`
> — [upstream `configure.ac`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=configure.ac;hb=HEAD)
> (`@<:@` and `@:>@` are autoconf quadrigraphs for `[` and `]`; the help string renders as
> "Enables Remote Calls [default=no]")

Debian says the same thing in prose and dates it:

> "Since version 1.2.5 due to security concerns upstream has turned off the remote calls
> functionality by default and added a configuration flag at build time to enable it. This
> functionality caused rpcbind to open up random listening ports."
> — [`debian/README.debian`, rpcbind 1.2.7-1](https://sources.debian.org/data/main/r/rpcbind/1.2.7-1/debian/README.debian)

This is the single most important finding for 111, and it is easy to read backwards. It is **not**
an upstream admission that rpcbind is indefensible on the internet. It is upstream identifying the
specific feature that made exposure dangerous and taking it out of the default build — the same
shape as memcached disabling UDP in 1.5.6, which §3.1 cites as *supporting* a row, but with the
opposite consequence, because memcached pairs it with a "you *must not* expose" sentence and rpcbind
pairs it with nothing.

**Every shipped default binds to the world.** Upstream's own socket unit:

> `ListenStream=0.0.0.0:111`
> `ListenDatagram=0.0.0.0:111`
> `ListenStream=[::]:111`
> `ListenDatagram=[::]:111`
> — [upstream `systemd/rpcbind.socket`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=systemd/rpcbind.socket;hb=HEAD)

And Debian ships the loopback restriction **present but commented out**, which is as explicit a
statement of intended default as a config file can make:

> `OPTIONS="-w"`
> `# Uncomment the following line to restrict rpcbind to localhost only for UDP requests`
> `# OPTIONS="${OPTIONS} -h 127.0.0.1 -h ::1"`
> — [`debian/rpcbind.default`, rpcbind 1.2.7-1](https://sources.debian.org/data/main/r/rpcbind/1.2.7-1/debian/rpcbind.default)

**This closes the last available route.** §2.2 admits three forms of attestation, and the third —
the documented shipped default — is the weak one that carries 5432, 5984 and 9042. For 111 it does
not merely fail to help; it points the other way. PostgreSQL's `listen_addresses = localhost` is on
the list because it is a maintainer position expressed in code. `ListenStream=0.0.0.0:111` is the
same kind of statement with the opposite content, and a standard that reads one as evidence must
read the other as evidence too.

**Claim 3 fails on the facts, and the man page is the one that says so.** rpcbind's whole function
is answering remote lookups:

> "When a client wishes to make an RPC call to a given program number, it first contacts rpcbind on
> the server machine to determine the address where RPC requests should be sent."
> — [`rpcbind(8)`, DESCRIPTION](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=man/rpcbind.8;hb=HEAD)

A protocol whose defined client is a remote client cannot satisfy "the intended clients are other
components of the same system".

**The source that comes closest is a distribution hardening guide, and §2.3 bars it.** Red Hat's
RHEL 7 Security Guide §4.3.4 is the only document in the corpus that characterises rpcbind's
security posture in its own voice:

> "It has weak authentication mechanisms and has the ability to assign a wide range of ports for the
> services it controls. For these reasons, it is difficult to secure." … "It is important to use TCP
> Wrappers to limit which networks or hosts have access to the rpcbind service since it has no
> built-in form of authentication."
> — [RHEL 7 Security Guide §4.3.4, "Securing rpcbind"](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/security_guide/sec-securing_services)
> (read via an [Internet Archive snapshot dated 2024-02-22](http://web.archive.org/web/20240222061242/https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/security_guide/sec-securing_services) — see §9.5)

Three reasons it cannot carry the row, in increasing order of force:

1. **Red Hat does not own the protocol.** §2.3's rule is that the claim must be attested "by a
   source that owns the protocol". Microsoft qualifies for SMB and Dell for DRAC because they
   designed the thing; Red Hat packages rpcbind, and its Security Guide is a Red Hat documentation
   product rather than an rpcbind project document.
2. **It is a hardening instruction, not a legitimacy statement.** "Limit which networks or hosts
   have access" is the exact shape of the NSA/CISA sentence §4.4 refused for 6443 — a preference
   expressed against a real supported architecture. Note also that Red Hat's *own* enumeration in
   the neighbouring §4.3.3, "Services that should be carefully implemented and behind a firewall",
   lists `auth`, `nfs-server`, `smb and nbm (Samba)`, `yppasswdd`, `ypserv` and `ypxfrd` — and
   **omits rpcbind**.
3. **Red Hat contradicts itself across products, in the AWS pattern of §2.3.** The Storage
   Administration Guide's firewall procedure directs the reader to "Allow TCP and UDP port 111
   (rpcbind/sunrpc)."
   ([RHEL 6 Storage Administration Guide §9.7.3](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/6/html/storage_administration_guide/s2-nfs-nfs-firewall-config)).
   One vendor, one port, opposite instructions in two manuals — which is precisely why §2.3 exists.

The same guide adds the observation that is the honest practical advice and is *not* a position on
exposure: "Securing rpcbind only affects NFSv2 and NFSv3 implementations, since NFSv4 no longer
requires it." The correct answer for most estates is that rpcbind should not be *running*, which is
a different claim from the one this list makes.

**Determinacy is not the problem.** IANA registers both `sunrpc,111,tcp,SUN Remote Procedure Call`
and `sunrpc,111,udp` to Chuck McManis, with no competing usage and an empty Unauthorized Use
Reported field (registry CSV retrieved 2026-08-13).

**Verdict: 111/tcp and 111/udp remain excluded**, and §2.7's reasoning is upgraded rather than
replaced — not "nobody will say it", but "the people entitled to say it shipped the opposite
default and fixed the actual hazard in the code". **The criterion that would change the verdict:**
an upstream, Sun/Oracle or nfs-utils statement on network placement, or a change to
`systemd/rpcbind.socket` binding loopback. Neither is a watchable-event stretch; both are exactly
the trigger §8 question 1 describes.

### 9.2 389/tcp LDAP — not admitted, and the case tests Class B rather than LDAP

§4.6 excluded 389 in one line: RFC 4513 §6.3.3 discourages cleartext credentials "unless the data on
the session is protected using TLS", so the port is not the discriminator. That verdict is correct
and it now rests on three independent attestations instead of one. The interesting part is what the
case does to **Class B**.

**Falsified first, because it is the premise everyone brings to this port.** RFC 4513 does *not*
deprecate `ldaps://` on 636. It does not mention them. The strings `636` and `ldaps://` occur
**zero times** across RFC 4510, RFC 4511, RFC 4513, RFC 4516 and the obsoleted RFC 2830 — every
document in the LDAP technical specification that could plausibly carry such a statement, retrieved
and counted. (The trap: a case-insensitive search for `ldaps` matches `LDAPString`, which appears
throughout RFC 4511's ASN.1. That false positive is almost certainly the origin of the belief.)

The IETF's position on the split-port pattern for LDAP is expressed **by omission**, and the
omission is total: the standards-track LDAP suite defines TLS via StartTLS *on the LDAP port* and
never defines, registers or references a TLS-on-a-separate-port variant. IANA's 636 row is a bare
legacy registration — `ldaps,636,tcp,ldap protocol over TLS/SSL (was sldap)`, assigned to Pat
Richard, with no RFC reference — sitting beside `ldap,389,tcp,Lightweight Directory Access
Protocol`, assigned to Tim Howes.

**The protocol's reference implementation states the preference directly.** OpenLDAP:

> "OpenLDAP supports negotiation of TLS (SSL) via both StartTLS and ldaps://. See the Using TLS
> chapter for more information. StartTLS is the standard track mechanism."
> — [OpenLDAP 2.6 Administrator's Guide §14.2, Data Integrity and Confidentiality Protection](https://www.openldap.org/doc/admin26/security.html)

**So Class B's central clause is not satisfiable for LDAP, and that is the whole finding.** Claim 2
requires "credentials or session content in cleartext, **with a standardised encrypted successor
reachable on a different port**". §4.2 already explained that the successor clause is not a
scoping detail — "the encrypted sibling is not an exception to the rule, it *is* the rule — its
existence is what makes the plaintext port wrong". LDAP is the case that proves the clause is
doing real work: the standardised encrypted form of LDAP is reachable on **the same port**, so
there is no plaintext port to condemn. 23 is wrong because 22 exists; 389 is not wrong, because
389 is also where the fix lives.

**RFC 4513 §6.3.3 says this in its own construction, and it is worth quoting at the length that
shows it.** The prohibition is conditional, and the mandate is *per session*:

> "The use of clear text passwords and other unprotected authentication credentials is strongly
> discouraged over open networks when the underlying transport service cannot guarantee
> confidentiality. LDAP implementations SHOULD NOT by default support authentication methods using
> clear text passwords and other unprotected authentication credentials unless the data on the
> session is protected using TLS or other data confidentiality and data integrity protection."
> — [RFC 4513](https://www.rfc-editor.org/rfc/rfc4513.txt), §6.3.3

> "To mitigate the security risks associated with the transfer of passwords, a server implementation
> that supports any password-based authentication mechanism that transmits passwords in the clear
> MUST support a policy mechanism that at the time of authentication or password modification,
> requires that: A TLS layer has been successfully installed. OR Some other data confidentiality
> mechanism that protects the password value from eavesdropping has been provided. OR The server
> returns a resultCode of confidentialityRequired for the operation"
> — [RFC 4513](https://www.rfc-editor.org/rfc/rfc4513.txt), §6.3.3

The MUST is the sharpest available statement of why a port-keyed list cannot answer this question.
RFC 4513 mandates a mechanism that decides, **at the time of the bind, for that session**, whether
the credential is protected. Two connections to the same `(389, tcp)` pair can land on opposite
sides of that decision. A signal keyed on the pair sees neither.

RFC 4513 §2 is the same shape one level up: implementations "MUST be capable of protecting this
name/password authentication using TLS as established by the StartTLS operation" and "SHOULD
disallow the use of the name/password authentication mechanism by default when suitable data
security services are not in place" — a requirement on the *session state*, never on the port.

**The dominant vendor's remedy is also on-port, and its current shipped default enforces it.**
Microsoft's answer to cleartext credentials against Active Directory is LDAP signing and channel
binding, negotiated over the existing connection:

> "When you enforce LDAP signing on a domain controller, it rejects SASL LDAP binds that don't
> request signing and rejects simple binds performed over nonencrypted connections."
> — [Microsoft, LDAP signing for Active Directory Domain Services](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/ldap-signing)

> "**LDAP signing**: All new Active Directory deployments require LDAP signing by default through
> the "Domain controller: LDAP server signing requirements enforcement" policy."
> — same page, "Windows Server 2025 and later"

Under §2.2 a shipped default is an attestation, and this one attests a **hardened 389**, not a
migration to 636. The honest counterweight, recorded because it is the strongest thing available to
the other side: on "Windows Server 2019 and earlier" the same page records "**LDAP signing**:
Optional by default" and "**LDAP channel binding**: Set to "Never" by default". That is a permissive
default, and it is evidence the vendor tolerated cleartext binds — not evidence that exposure is
illegitimate — and the current release reverses it.

**Claim 3 fails explicitly, in the reference implementation's first sentence on the subject.**

> "OpenLDAP Software is designed to run in a wide variety of computing environments from
> tightly-controlled closed networks to the global Internet."
> — [OpenLDAP 2.6 Administrator's Guide §14, Security Considerations](https://www.openldap.org/doc/admin26/security.html)

That is a protocol owner naming the public internet as an intended deployment environment, which is
the same disqualification §4.4 applied to 6443 and §4.6 applied to Jenkins. The shipped default
agrees:

> "slapd will by default serve ldap:/// (LDAP over TCP on all interfaces on default LDAP port).
> That is, it will bind using INADDR_ANY and port 389."
> — [`slapd(8)`, `-h URLlist`](https://www.openldap.org/software/man.cgi?query=slapd&sektion=8)

OpenLDAP's own guidance on simple binds is, once again, conditional and session-scoped rather than
port-scoped: the mechanism "offers no eavesdropping protection (e.g., the password is set in the
clear)", so "it is recommended that it be used only in tightly controlled systems or when the LDAP
session is protected by other means (e.g., TLS, IPsec)" (§14.3.1). And OpenLDAP's network-security
advice is generic firewalling — §14.1.2 observes only that "slapd(8) listens on port 389/tcp for
ldap:// sessions and port 636/tcp for ldaps://) sessions" (upstream's stray parenthesis preserved).

**Determinacy is not the problem.** `ldap,389,tcp` and `ldap,389,udp` are registered to Tim Howes
with an empty Unauthorized Use Reported field.

**Verdict: 389/tcp remains excluded**, and the reason is upgraded from "the port is not the
discriminator" to "the encrypted successor is on the same port, and the protocol's owner calls that
the standards-track mechanism." **The criterion that would change the verdict:** none that is
reachable by re-reading sources. 389 becomes listable only if the IETF or OpenLDAP moves the
standards-track encrypted form off 389, which is the opposite of the direction RFC 6335 §9 pushes.

**What this does to the standard.** §4.2 noted that RFC 6335 §9 makes Class B a *closing* category
because the IETF discourages new split-port pairs. LDAP is the concrete instance of the pattern the
IETF prefers instead — in-band negotiation on one port — and it is **unlistable by construction**.
So Class B is not merely closing; the alternative that replaced it is structurally invisible to a
port-keyed list. That is a real and permanent coverage boundary of `sensitive-port-exposed`, it is
not a defect in the claim's wording, and it belongs in §8.

### 9.3 79/tcp finger — not admitted, on two independent grounds

The ticket framed this as the sharpest question of the three: is disclosure-by-design one of the
three admitted claims, or a fourth claim class? **Answer: neither. It is not a fourth class, and the
reason is that it does not discriminate.** 79 also fails determinacy independently, which is the
more mechanical of the two grounds and the more decisive.

#### 9.3.1 The RFC's position is conditional, and the two famous sentences are about different things

RFC 1288 §3.2 is unusually vivid for an RFC, and it is routinely quoted in a way that merges two
separate passages. The full paragraph:

> "Warning!! Finger discloses information about users; moreover, such information may be considered
> sensitive. Security administrators should make explicit decisions about whether to run Finger and
> what information should be provided in responses. One existing implementation provides the time
> the user last logged in, the time he last read mail, whether unread mail was waiting for him, and
> who the most recent unread mail was from! This makes it possible to track conversations in
> progress and see where someone's attention was focused. Sites that are information-security
> conscious should not run Finger without an explicit understanding of how much information it is
> giving away."
> — [RFC 1288](https://www.rfc-editor.org/rfc/rfc1288.txt), §3.2, RUIP security

The "sleep of system administrators" line is **not** in that paragraph and is not about the protocol
at large. It is in §3.2.4, and it is scoped to one optional feature:

> "Allowing an RUIP to return information out of a user-modifiable file should be seen as equivalent
> to allowing any information about your system to be freely distributed. That is, it is potentially
> the same as turning on all specifiable options. This information security breach can be done in a
> number of ways, some cleverly, others straightforwardly. This should disturb the sleep of system
> administrators who wish to control the returned information."
> — [RFC 1288](https://www.rfc-editor.org/rfc/rfc1288.txt), §3.2.4, User information files

Precision matters here because the merged version reads as a prohibition and neither half is one.
§3.2's operative sentence is an **informed-consent condition** — do not run it *without
understanding what it gives away* — which is the same grammatical shape as Prometheus's "unless you
know what you are doing and have taken appropriate measures", refused in §4.3. Compare the sentences
that actually carry rows: memcached's "you _must not_ expose memcached directly to the internet",
Elastic's "Never expose an unprotected node to the public internet", Microsoft's "It is unlikely
that any SMB communication originating from the internet or destined for the internet is
legitimate". Those are verdicts. §3.2 is a disclosure requirement placed on the administrator.

**And RFC 1288 places finger on the internet by design, in the immediately preceding section.**

> "Finger is one of the avenues for direct penetration, as the Morris worm pointed out quite
> vividly. Like Telnet, FTP and SMTP, Finger is one of the protocols at the security perimeter of a
> host."
> — [RFC 1288](https://www.rfc-editor.org/rfc/rfc1288.txt), §3.1, Implementation security

"At the security perimeter of a host" is the specification describing an internet-facing service and
demanding it be implemented to that standard. The rest of the document assumes the same: §2.5.5
specifies how network-reachable vending machines should answer queries, and the `{Q2}` cross-host
forwarding feature is *recommended off by default* (§3.2.1) rather than forbidden, which is a
statement that forwarding is a supported mode. RFC 1288 is still a **Draft Standard** on the IETF
datatracker; it was never reclassified Historic.

#### 9.3.2 Disclosure-by-design cannot be a fourth claim class

A new claim class is a change to the standard, not a row, so it has to be tested by naming what else
it would admit. Drawn honestly — *the protocol's purpose is to disclose information about the system
to whoever asks* — it admits:

| Also admitted | Why that is fatal |
|---|---|
| 43/tcp WHOIS | Disclosure to anonymous internet clients is the entire specification. Universally, correctly internet-facing |
| 53/tcp+udp DNS | Same, at planetary scale |
| 389/tcp LDAP anonymous bind | RFC 4513 §2: servers **MUST** support anonymous simple bind. §9.2 just excluded it |
| 11/tcp `systat`, 15/tcp `netstat` | "Active Users" and a netstat dump, by registration. Nobody would argue, and nobody probes them either |
| 111/tcp rpcbind `DUMP` | Discloses the host's RPC service inventory to any caller — §9.1 just excluded it |

A class that admits WHOIS and DNS is not a class. The failure is structural, and naming it is the
useful output of this ticket:

> **All three existing claims name a *mismatch* between what the protocol assumes and what an
> internet vantage supplies.** Claim 1: no authentication, where the operations need authority.
> Claim 2: cleartext credentials, where a standardised encrypted pair already exists elsewhere.
> Claim 3: a same-system audience, where the internet is not one. **Disclosure by design is not a
> mismatch — it is the protocol working as specified.** There is nothing for the claim to be
> *against*.

The intuition that finger is different from WHOIS is real, but it is about *what* is disclosed —
usernames, login times, mail activity — not about *whether* disclosure is the design. That is a
judgement about data sensitivity, and it is exactly the kind of judgement §2.1 exists to keep out of
this list, because it has no owner entitled to attest it and no bright line once admitted.

#### 9.3.3 Claim 1 fits finger literally, and refusing it exposes a missing qualifier

This is the pass's most useful by-product and it should not be buried. Claim 1 as written is "**No
authentication in the protocol as shipped.** The service, in the configuration its maintainers ship,
admits anonymous commands." Finger satisfies that sentence completely — it has no authentication
mechanism of any kind, precisely as 69/udp TFTP does, and 69/udp is on the list under Claim 1 with
the reason "The protocol has no authentication mechanism of any kind, by specification."

Finger is refused anyway, and the refusal only works because Claim 1 has always carried an unstated
qualifier. Read literally it admits every public web server, every WHOIS server and every
authoritative nameserver. The qualifier the Class A rows all rely on but the text never states:

> The anonymous commands must be ones that would otherwise require **authority** — writing data,
> executing code, reading data not intended for the caller, controlling a runtime — not the
> operations the protocol exists to answer.

Every Class A row satisfies that once it is written down: `FLUSHALL` on Redis, container control on
2375, cluster Secrets on 2379, arbitrary file read on TFTP. Finger does not: its anonymous
operations are the ones RFC 1288 specifies it to answer. **This is a wording defect in §2.1, not a
list change**, and it is recorded in §8 rather than fixed here, because editing the claim text after
the list was built against the unstated version would need the whole list re-checked against the
stated one.

#### 9.3.4 79/tcp fails determinacy, and it fails on the registry's own text

The more mechanical ground, and the one that would exclude 79 even if a claim had fitted. IANA's
79/tcp row is:

> `finger,79,tcp,Finger,[David_Zimmerman],[David_Zimmerman],,,,,Unauthorized use by some mail users (see [RFC4146] for details),`
> — [IANA Service Name and Transport Protocol Port Number Registry](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.csv) (retrieved 2026-08-13)

Following that reference to the RFC it names:

> "With this technique, the server sends the string "nm_notifyuser" immediately followed by CRLF to
> the finger port on the IP address for the user who has received new mail. The finger port is 79.
> Note that only the port for finger is used; the finger protocol itself is not used." … "On the
> client system, a process must be listening to the finger port"
> — [RFC 4146, Simple New Mail Notification](https://www.rfc-editor.org/rfc/rfc4146.txt), §3

> "The notify mail hack (and this document) should be included as an additional usage for port 79."
> — [RFC 4146](https://www.rfc-editor.org/rfc/rfc4146.txt), §6, IANA Considerations

So a listener on 79/tcp may be a **mail client waiting for a new-mail notification**, running a
protocol that is not finger, on a usage IANA was asked to record as additional. This is the §2.4
version-dependence failure mode with the registry on record instead of an inference of ours: the
`(port, transport)` pair does not determine one service, and a signal firing "finger exposed" on an
email client's notification listener is exactly the unactionable firing the gate exists to prevent.

Worth stating explicitly that §2.4's caution about the `Unauthorized Use Reported` field is
respected. That field "marks squatting on a number without registration, a registry-hygiene matter.
It is not a security judgement and is not used as one here." It is used here only as evidence about
**what else listens on the port** — determinacy — which is the one thing it is actually competent
to say.

> **Amended by §15** ([#82](https://github.com/winniel123/verge-asm/issues/82)). **The care above is
> vindicated and the footing is strengthened one step further.** Under
> [ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md) an IANA row is
> a **record of a registrant's placement declaration** rather than an authority, so the annotation
> **corroborates** and the carrying artefact is the one it points at: **RFC 4146 itself**, in force,
> in which the IETF specifies that *"a process must be listening to the finger port"* for a protocol
> that is not finger. That is the specifying party's own statement about its own protocol — the
> defeating placement statement, quoted above. **The verdict is unchanged.** §15.5.

#### 9.3.5 Implementations and shipped defaults: no position anywhere

Exhausted per the ticket, and every one is a negative:

- **OpenBSD `fingerd(8)`** — `NAME`, `SYNOPSIS`, `DESCRIPTION`, options, `SEE ALSO`, `STANDARDS`,
  `HISTORY`. No `SECURITY` section, no exposure statement. Its `-s` flag ("Enable secure mode.
  Forwarding of queries to other remote hosts is denied") is opt-in, so OpenBSD does not even apply
  RFC 1288 §3.2.1's RECOMMENDED default ([man.openbsd.org/fingerd.8](https://man.openbsd.org/fingerd.8),
  page dated March 31, 2022).
- **The OpenBSD data point that cuts hardest.** §3.4 carries the rows for 512, 513 and 514 on
  OpenBSD's position "expressed by deletion rather than prose" — `rlogin(1)`, `rlogind(8)` and
  `rexecd(8)` removed in 3.2, `rsh(1)` in 5.6, all four man pages now 404. **`fingerd(8)` was not
  deleted.** It is in the current tree with a current man page. A project that states positions by
  removal has declined to state this one, and that is a stronger negative than silence.
- **FreeBSD `fingerd(8)`** — likewise no `SECURITY` section and no exposure statement
  ([man.freebsd.org](https://man.freebsd.org/cgi/man.cgi?query=fingerd&sektion=8)).
- **GNU inetutils ships no finger daemon at all.** The string "finger" occurs **zero** times in the
  [inetutils manual](https://www.gnu.org/software/inetutils/manual/inetutils.html), which documents
  `telnetd`, `tftpd` and `whois` (38, 28 and 40 occurrences respectively — counted to confirm the
  extraction was working rather than the file empty). An absence, not an attestation.
- **Fedora** still ships a finger daemon, as an optional `finger-server` subpackage built from
  `bsd-finger`, carrying `in.fingerd`, `finger.socket` and `finger@.service`, with the description
  "The server daemon (fingerd) must be started using systemctl to receive finger requests"
  ([`finger.spec`, rawhide](https://src.fedoraproject.org/rpms/finger/raw/rawhide/f/finger.spec)).
- **Debian** has no plain `fingerd` source package — only `cfingerd`, `efingerd`, `ffingerd` and
  `xfingerd`, none of them the reference implementation
  ([sources.debian.org](https://sources.debian.org/api/search/fingerd/)).

Absent-by-default is where the distribution evidence lands, and it does not reach §2.2's third form.
That form needs a documented **listen** default, the way PostgreSQL documents `listen_addresses`. An
inetd- or socket-activated service that is simply not installed documents nothing about where it
listens when it is.

**Verdict: 79/tcp remains excluded**, on two independent grounds either of which suffices: no
admitted claim fits and disclosure-by-design cannot become one, and the pair fails determinacy
against a usage IANA itself records. **The criterion that would change the verdict:** a fourth claim
class that survives the naming test in §9.3.2 — which, on the analysis above, does not exist — plus
a determinacy answer for RFC 4146. Neither is close.

### 9.4 What this pass changed

**Nothing on the list.** Still 38 `(port, transport)` pairs; §1's summary table, §3's tables and
§6.1's containment arithmetic are all untouched. §8 question 4 is closed.

**Two things about the evidence standard**, both carried into §8 rather than acted on here:

1. **Claim 1 needs its unstated qualifier written down** (§9.3.3). Read literally it admits every
   anonymous public service, and only an unwritten "the anonymous operations must be ones requiring
   authority" keeps finger, WHOIS and HTTP out. Every existing Class A row satisfies the stated
   version, so this is a documentation fix — but an unstated criterion is exactly what makes a
   curated list unfalsifiable, which is the failure §2 exists to prevent.
2. **Class B has a permanent blind spot, and it is not a wording problem** (§9.2). In-band upgrade
   on a single port — StartTLS, LDAP signing — is the pattern RFC 6335 §9 tells implementers to
   prefer over split ports, and it is invisible to a list keyed on `(port, transport)`. The pressure
   this creates discharges the same way §5 said the middle band's did: into an observation-driven
   rule (*did this listener require confidentiality before accepting a credential?*), not into a
   softer list.

**And one thing about the method.** §8 question 4 was written on the belief that the remaining three
were "a search problem rather than a settled verdict", on the precedent of 2049 and 873. The search
was run properly and the belief was wrong for these three — but the pass was still worth its cost,
because it converted two exclusions from *no evidence found* to *evidence found pointing the other
way*, which is the difference between a gap and a finding. Done before v1, it cost nothing; after
v1 it would have cost a comparability cycle on the product's best signal whichever way it came out
(§7.2).

### 9.5 Retrieval hazards and errors caught

Recorded in the spirit of §3.4's `xhost` note and §4.6's WinRM paragraph — the near-misses are part
of the deliverable.

- **"RFC 4513 deprecates `ldaps://` on 636" is false.** The document does not mention either. Nor
  do RFC 4510, 4511, 4516 or 2830 — `636` occurs zero times in all five, `ldaps://` zero times in
  all five. A case-insensitive search for `ldaps` returns many hits in RFC 4511, all of them
  `LDAPString`, which is very likely how the belief propagated. This claim was in the framing of
  the ticket that commissioned this pass and it did not survive the bytes.
- **`docs.redhat.com` and `access.redhat.com` refuse `curl`** — both return an edge "Access Denied"
  page regardless of user-agent. The RHEL 7 Security Guide and RHEL 6 Storage Administration Guide
  text quoted in §9.1 was read from Internet Archive snapshots (the Security Guide snapshot carries
  `x-archive-orig-date: Thu, 22 Feb 2024`). That is one step weaker than direct retrieval and it is
  flagged rather than smoothed over — though it does not affect the verdict, since §9.1 declines the
  source on ownership grounds anyway.
- **`git.linux-nfs.org` serves an expired TLS certificate** (`SEC_E_CERT_EXPIRED`). The rpcbind
  `man/rpcbind.8`, `README`, `configure.ac` and `systemd/rpcbind.socket` were retrieved with
  certificate verification disabled. The man page text was independently cross-checked against
  [Debian's rendering of the shipped page](https://manpages.debian.org/unstable/rpcbind/rpcbind.8.en.html),
  which agrees verbatim on the `-i` paragraph, so the content is corroborated even though the
  channel was not authenticated.
- **`learn.microsoft.com` returns a JavaScript shell to `curl`.** The LDAP signing quotes in §9.2
  were taken from the article's source Markdown in Microsoft's own public docs repository
  ([`MicrosoftDocs/windowsserverdocs`, `WindowsServerDocs/identity/ad-ds/ldap-signing.md`](https://raw.githubusercontent.com/MicrosoftDocs/windowsserverdocs/main/WindowsServerDocs/identity/ad-ds/ldap-signing.md),
  `ms.date: 01/15/2026`), which is the same first-party content before rendering.
- **Not exhausted, and recorded as such:** the non-reference finger daemons in Debian (`cfingerd`,
  `efingerd`, `ffingerd`, `xfingerd`) were not read. None is the protocol's reference
  implementation, so none could attest under §2.2, but the claim "no finger implementation states a
  position" is scoped to the reference implementations and GNU inetutils, not to all of them.

---

## 10. Repair of the evidence standard

Wayfinder ticket [#37](https://github.com/winniel123/verge-asm/issues/37), acting on the four defects
[#30](https://github.com/winniel123/verge-asm/issues/30) surfaced in §9 and recorded rather than
fixed. This section **amends §2 and §3 in place by reference**: earlier text is left standing and
marked, per the repo's name-and-withdraw convention, and where §10 and an earlier section disagree,
§10 governs.

**Headline result, stated first because it is the strongest form the repair can take.**

> **No `(port, transport)` pair is added, and none is removed. The list stays at 38 and §1's summary
> table, §3's tables and §6.1's containment arithmetic are all untouched.** Every repair below was
> tested by walking the rows it could plausibly move — all 12 Class A rows for §10.1, all 38 for
> §10.4, all 16 named exclusions in §4.6 for §10.3 — rather than by asserting that none moves.

Two rows change their **stated reason** and neither changes its grounds (§10.7), and one row's
footing is **disclosed as thinner than §2.6 claimed** and routed to a retrieval ticket (§10.6).

### 10.1 Defect 1 — Claim 1's qualifier, written down

§9.3.3 found Claim 1 read literally admits finger, WHOIS, DNS and every public web server, and that
only an unwritten *"the anonymous operations must be ones that would otherwise require authority"*
keeps them out. That sentence is true and it is **not applicable**: *would otherwise require* asks a
reviewer to imagine a counterfactual, and *authority* is a judgement with no owner — which is the
severity-shaped reasoning §2 exists to keep out. The qualifier is therefore restated as a **two-step
test read off the specification**, in order, the first step dispositive.

> **Claim 1 — No authentication in the protocol as shipped.** The service, in the configuration its
> maintainers ship, admits anonymous commands. This is a fact about released software, not a
> judgement. **A row is admitted under Claim 1 only if both steps pass.**
>
> **Step 1 — the publication test, and it refuses outright.** If the operations answerable
> anonymously are the ones the protocol's own specification defines it to **perform for callers it
> does not identify** — if the specification's statement of purpose is *answer whoever asks* — then
> Claim 1 is **unavailable**, whatever else is true. Anonymous access is the protocol working as
> specified, and there is nothing for the claim to be against.
>
> **Step 2 — the authority test.** Otherwise, the anonymous operations must **write or delete data,
> register or deregister a service, execute code, control a runtime, or read content the protocol
> carries on behalf of a party other than the caller**. If they do none of these, Claim 1 is not
> satisfied.
>
> Both steps are answered from the specification or the owner's own documentation. Neither reads a
> judgement about how sensitive the disclosed data is, and neither reads how often the port is
> attacked.

**Why the order is load-bearing and must not be reversed.** Step 2 alone admits a public web server,
because a web server's corpus is operator-supplied content served to a stranger. Step 1 refuses it
first, and Step 1's ground is a sentence the protocol's owner wrote. Reversing the steps rebuilds the
defect.

**Worked refusals** — the test applied to everything §9.3 showed the unqualified claim admitting:

| Refused | Step | The owner's own words |
|---|---|---|
| 79/tcp finger | 1 (and 2) | RFC 1288 §3.1 places finger *"at the security perimeter of a host"* alongside Telnet, FTP and SMTP, and §2.5 specifies exactly what a RUIP returns — the corpus is the specification's own. Read-only, so Step 2 fails independently |
| 43/tcp WHOIS | 1 | Publishing registration data to anonymous queriers is the entire specification |
| 53/tcp+udp DNS | 1 | A public distributed database answering queries from any resolver. Note also that RFC 2136 UPDATE is *not* anonymous in the shipped default (`allow-update { none; }`), so Step 2 fails too |
| 80/443 HTTP | 1 | Representations identified by globally-scoped URIs and served to any client; RFC 9110's authentication framework is **opt-in** precisely because publication is the default case. Excluded on determinacy in any event |
| 11/tcp `systat`, 15/tcp `netstat` | 1 | "Active Users" and a netstat dump, answerable to any caller by registration |
| 111/tcp rpcbind lookups | 1 | `rpcbind(8)`: *"When a client wishes to make an RPC call … it first contacts rpcbind on the server machine to determine the address"* — the specification names a remote client as the intended recipient |

That last row is worth pausing on. **The repaired Claim 1 refuses 111/tcp on claim grounds, independently of §9.1's attestation grounds.** §9.1's verdict is strengthened rather than disturbed: the mutating path (`SET`/`UNSET`) is loopback-only in the shipped default, so it is not anonymous from the network at all, and the path that *is* anonymous is the lookup the specification exists to answer.

**Every Class A row walked, in full.** The test can only ever *remove*, so this is the whole check:

| Row | Step 1 — is publication the purpose? | Step 2 — what authority does the anonymous caller get? |
|---|---|---|
| 2375/tcp Docker | No — *"anyone with access to that port has full Docker access"* | Controls a runtime |
| 2379/tcp etcd client | No — *"ideally only the API server should have access to it"* | Writes cluster state; reads every Secret |
| 2380/tcp etcd peer | No — node-to-node | Writes cluster state |
| 10250/tcp kubelet | No — upstream scopes users to *"Self, Control plane"* | Controls a runtime |
| 10255/tcp kubelet read-only | No — same scoping | Reads node and pod status the kubelet carries **on behalf of the control plane** |
| 6379/tcp Redis | No — *"trusted clients inside trusted environments"* | `FLUSHALL`, and arbitrary writes |
| 11211/tcp memcached | No — *"you must not expose memcached … to any untrusted users"* | `set` / `delete` |
| 11211/udp memcached | No — same | Same |
| 2181/tcp ZooKeeper | No — *"a trusted computing environment"* | World-**writable** znodes |
| 4369/tcp epmd | No — *"only expose these ports to the hosts and subnets that run other cluster nodes"* | Registers and deregisters node names |
| 9042/tcp Cassandra | No — the docs frame authentication as the thing that is off | Writes |
| 69/udp TFTP | No — RFC 1350 warns about *"the rights granted to a TFTP server process so as not to violate the security of the server hosts file system"*, i.e. the corpus is the operator's filesystem, not a designated public one | `WRQ` writes; `RRQ` reads the host's filesystem |

**All twelve survive. Class A is unchanged, and no row changes class.** The one row that needs Step
2's read limb rather than its mutation limb is 10255/tcp, and it needs no reclassification: Claim 1
and Claim 3 are adjacent on that row and Claim 1 still fits.

**What the qualifier costs, stated plainly.** It makes Claim 1 slightly harder to reach than the
sentence a reader would infer from its title, and a future candidate that is unauthenticated,
read-only and carries no third party's content will now be refused where the old text would have
admitted it. That is the intended direction: an unstated criterion cannot be argued against, and a
list nobody can argue against is not evidence.

### 10.2 Defect 2 — the three claims are closed, and closed by construction

§9.3.2 named the unifying property — *all three claims name a mismatch between what the protocol
assumes and what an internet vantage supplies* — and left it as reasoning rather than a rule.
**Promoted, and the enumeration is derived rather than asserted.**

> **The claim set is closed.** A permitted claim must name a **mismatch between an assumption the
> protocol makes and something an internet vantage supplies**. An internet vantage — per
> [ADR-0010](../adr/0010-exposure-composes-two-reaches.md) and
> [ADR-0017](../adr/0017-exposure-needs-both-legs.md), a `Reach` from a vantage verified outside every
> scope the operator holds — supplies exactly three things, and there is nothing else it *is*:
>
> | The vantage supplies | The protocol assumed | Claim |
> |---|---|---|
> | An **unknown principal** | its callers are authorised | **Claim 1** |
> | An **untrusted path** | its channel is private | **Claim 2** |
> | A **caller outside the boundary** | its callers are inside one | **Claim 3** |
>
> Three properties, three claims. **The set reopens only if someone names a fourth property of an
> internet vantage**, which is a falsifiable condition rather than a failure of imagination.

**Three candidate fourths were tested and all three fail.**

- **Integrity.** An untrusted path is not only observable, it is modifiable — RFC 6143 §9 names both
  in one sentence. It is not a fourth claim, because Claim 2's discriminator is the **standardised
  encrypted successor on another port** (§4.2, §9.2), and the successor fixes observation and
  tampering together. Integrity never admits a row that confidentiality does not, so it does not
  split the claim.
- **Availability and amplification.** An internet vantage supplies arbitrary volume, and three rows
  sit on UDP where that matters. It is not a fourth claim, because **§2.7 already refused it**: CISA
  TA14-017A's amplification factor for portmap is a *magnitude*, not a position, and admitting
  magnitude is exactly how frequency re-enters. Where an amplification sentence appears in a "why"
  cell it is colour, not grounds — corrected in §10.7.
- **Disclosure by design.** Refused in §9.3.2 and the refusal now has a home: it is not a mismatch at
  all. The construction above is why.

**What closure buys.** An open set makes the standard's tightness contingent — the next port with an
odd shape reopens the question, and a list whose admission rules can be extended on demand is
unfalsifiable in the slower way. This is
[ADR-0009](../adr/0009-verge-core-is-a-union.md)'s move applied one level up: a **definition that
cannot fail** rather than a list that can be added to.

**Thin ground, flagged.** The derivation rests on the claim that an internet vantage supplies exactly
three things, which is read off `Exposure`'s own definition rather than measured. It is the tightest
available footing and it is not a proof; if a fourth property is named, the set reopens and this
paragraph is the record of what would do it.

### 10.3 Claim 3's boundary is wider than "same system" — a wording correction closure forced

Closing the set obliges every row to name one of the three, and that immediately exposes a defect in
Claim 3's **wording** rather than its content. Two of Claim 3's own rows are management planes whose
assumed locality is *a network the operator controls*, not *the same host*: `161/udp` SNMP and
`623/udp` IPMI. Dell's sentence for 623 — *"DRACs are intended to be on a separate management
network"* — is a locality claim that the words "other components of the same system" do not fit.

> **Claim 3 — The protocol's intended clients are inside a boundary the operator controls**, not
> internet users: the same system, the same cluster, or a management network the owner names.
> Database wire protocols, cache protocols, cluster coordination, inter-node transports and
> out-of-band management interfaces. Exposing these enables no intended use whatsoever; it only
> enlarges the attack surface.
>
> **The boundary must be named by the owner.** Where the owner names the public internet as a
> supported deployment environment, Claim 3 fails however strongly a third party disapproves — which
> is the sentence doing all the exclusion work, and it is unchanged.

**This widens the words and not the list, and the check is a walk of every exclusion in §4.6.** Not
one of the sixteen was excluded because "same system" was too narrow:

| Excluded | The ground, unchanged by the wider wording |
|---|---|
| 22/tcp SSH | Remote administration over an untrusted network is the express purpose; GCP ships it open to `0.0.0.0/0` |
| 3389/tcp RDP | Risk position only; GCP ships it open |
| 5985/5986 WinRM | No first-party prohibition, and the Claim 2 argument is factually false (§4.6) |
| 9100/tcp | HP says printing on 9100 *"should always be enabled"* |
| 111/tcp rpcbind | No owner statement; and now refused at §10.1 Step 1 as well |
| 389/tcp LDAP | OpenLDAP names *"the global Internet"* as an intended environment |
| 79/tcp finger | RFC 1288 places it at the security perimeter; fails determinacy independently |
| 6443/tcp kube-apiserver | Upstream concedes managed distributions publicly expose it by default |
| 5601/tcp Kibana | No prohibition; routinely fronted on the internet behind auth |
| 8500/tcp Consul | Position is only that external access *"should be considered"* |
| 9092/tcp Kafka | *"security is optional - non-secured clusters are supported"* |
| 5672/15672 RabbitMQ | The prohibition names 4369 and 25672, not these |
| 8080/tcp Jenkins | Upstream names public-internet deployment as supported |
| 1099/tcp Java RMI | Modern JDK defaults are secure |
| Hadoop UIs | Fails determinacy — the port moved between major versions |
| 110/143/25 mail | The intended audience genuinely is the internet |

**Nothing is admitted by the wider wording. It is a correction to what Claim 3 has always been
doing.** And it does not let the government lists back in through the side door: §2.3 is untouched,
so CISA's category verdict still cannot carry a row, and the *boundary* limb still needs an owner —
which is what §10.6 finds is missing for one row.

> **Amended by §17** ([#79](https://github.com/winniel123/verge-asm/issues/79)). The walk above tests
> each exclusion's ground against a **wording** change and correctly finds none moves. It does not test
> whether the ground itself was established over the right **document classes**, which is
> [ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md)'s question and did not
> exist when this table was written. §17.1 runs that test on every negative in it. **Two rows' stated
> grounds change and neither row moves:** `5672`/`15672` — not in the table above, and excluded on *the
> prohibition names 4369 and 25672, not these* — becomes an owner **naming public networks as
> supported** for exactly those two ports, which is this section's own failure condition met from the
> other side (§17.4); and `9092/tcp` Kafka's *"security is optional"* is joined by a prohibition-shaped
> sentence in an unreleased first-party document (§17.5). **The `111/tcp`, `389/tcp` and `79/tcp` rows
> above are unchanged and are un-exposed**, each carrying a second ground no document can reach.

### 10.4 Defect 3 — a shipped default attests in one direction only

[#30](https://github.com/winniel123/verge-asm/issues/30) found *"a shipped default is evidence in
both directions"*, on rpcbind's `ListenStream=0.0.0.0:111` and Debian's commented-out loopback line.
**That framing is too strong, and the measurement that kills it is a walk of the list itself.**

#### 10.4.1 The option that lost, and what it would have cost

Read symmetrically — *a documented default that binds to the world excludes the row* — the rule
deletes roughly **half the list**, because binding to all interfaces is the ordinary default of most
of the services on it:

> **Listed rows whose shipped default binds widely:** 23/tcp Telnet · 21/tcp FTP · 512, 513, 514/tcp ·
> 5900/tcp VNC · 6000/tcp X11 (`-nolisten tcp` is distribution behaviour X.Org does not document as a
> default — §3.4) · 3306/tcp MySQL (`bind_address` has defaulted to `*` since 8.0) · 445, 139/tcp and
> 137, 138/udp SMB and NetBIOS · 2049/tcp NFS · 873/tcp rsync · 161/udp SNMP · 623/udp IPMI ·
> 25672/tcp RabbitMQ · 4369/tcp epmd · 2181/tcp ZooKeeper · 11211/tcp+udp memcached · 10250, 10255/tcp
> kubelet (`--address` defaults to `0.0.0.0`) · 69/udp TFTP.

A rule that removes **23/tcp Telnet** and **445/tcp SMB** from a list of ports that are never
legitimately internet-facing has failed. 445 carries the strongest sentence in the entire corpus —
Microsoft's *"it is unlikely that any SMB communication originating from the internet or destined for
the internet is legitimate"* — and it would be deleted by a bind address. The symmetric reading is
refused on that measurement, not on taste.

#### 10.4.2 The rule

> **§2.2's third form is an admission route only.** A shipped default is an attestation **only where
> it restricts**: a loopback bind, a feature off by default, a daemon that refuses to start. A
> restriction is a **costly act** — it buys friction at first run and the maintainer paid for it
> anyway, which is why etcd can say in its own words that the *absence* of security features exists
> *"to reduce friction for users getting started with the database"*. A **permissive** default is the
> absence of that act, and the absence of an act is not a position. It **neither admits a row nor
> excludes one**; it is silent.

This is why the weak tier holds. 5432 PostgreSQL, 5984 CouchDB and 9042 Cassandra all rest on
**restricting** defaults — `listen_addresses = localhost`, `bind_address = 127.0.0.1`,
`rpc_address: localhost` — so the tier rests on a costly signal rather than on an accident, and §2.2's
disclosure of it is unchanged.

> **Amended by §12** ([#69](https://github.com/winniel123/verge-asm/issues/69)). All three defaults
> above are confirmed against the shipped bytes, so this paragraph's point stands — but **9042 no
> longer rests on its default alone** and has left the weak tier (§12.7). §12 also settles what
> *shipped default* means when the bytes have two owners: the third form reads the configuration that
> **takes effect** and that the project **documents as its default**, so an **example** file is silent
> in both directions, and installing one gives the installer **operativeness** rather than ownership.
> §10.4's *costly act* is the test that decides it, and it decides it **against** the example: the cost
> §10.4 prices is friction at first run, and a file no daemon reads produces no first run.

#### 10.4.3 What #30 actually found, stated precisely — the remedy route

The valuable half of #30's finding is not the bind address. It is that rpcbind's maintainers **took
two restricting acts aimed at the exposure hazard and neither of them reached the port**: `SET`/
`UNSET` confined to loopback *"for security reasons"*, and `--enable-rmtcalls [default=no]` removing
the amplification surface from the default build. That is a different object from a default listen
address, and it does carry evidence against a row.

> **The remedy route (exclusion).** Where the owner has taken a **restricting act directed at the
> internet-exposure hazard itself**, and that act **stops short of the port** — the hazardous feature
> disabled, the mutating path confined, while the port goes on answering — the owner has placed the
> line somewhere other than the port, and Claim 1 or Claim 3 may not be asserted over their heads.
>
> **Rider, and it is what keeps the route honest: this applies only where the same owner states no
> prohibition.** Where a prohibition exists, the prohibition governs and the remedy is a second,
> compatible act by the same party.

**The route is tested against its three closest analogues and moves no row.**

| Case | Did the remedy reach the port? | Outcome |
|---|---|---|
| **11211/udp memcached** — UDP disabled by default in 1.5.6 for the amplification hazard | **Yes** — disabling UDP takes the port out of the default listener set entirely. And upstream pairs it with *"you must not expose memcached directly to the internet"* | Row stands, on both limbs |
| **2375/tcp Docker** — the daemon refuses to start with TLS disabled on a TCP address | **Yes** — the remedy is the port | Row stands, and is strengthened |
| **10255/tcp kubelet** — deprecated and slated for removal | **Yes** | Row stands |
| **111/tcp rpcbind** — mutating path confined, remote calls off, port still answering `0.0.0.0` | **No**, and no prohibition exists anywhere | **Excluded** — and now on a rule rather than on a residue |
| **389/tcp LDAP** — StartTLS and LDAP signing harden 389 *on 389* | Reaches the port and keeps it | Already excluded (§9.2); consistent |

So §9.1's *"the people entitled to say it shipped the opposite default and fixed the actual hazard in
the code"* survives with its two halves separated: the opposite default attests nothing, and the fix
in the code attests everything.

**These are materially different instruments and the ticket was right to ask which.** A permissive
default is **silent**. A remedy that stops short of the port **excludes**. #30 read them as one
because rpcbind is the case where both are present.

### 10.5 Defect 4 — what "owns" means, and where a distributor stands

§2.2 says *"the source that owns it"* and §2.3 says *"a source that owns the protocol"*. One idea,
two phrasings, and the gap is where the Red Hat question lives.

> **Owner.** The party that **designed the protocol**, or that **authors the reference
> implementation**, speaking about the thing it designed or wrote. Microsoft owns SMB, Dell owns the
> DRAC, the IETF owns what its RFCs specify, and OpenBSD owns `fingerd(8)` because OpenBSD writes it.
>
> **A distributor owns its own shipped configuration and owns nothing about the protocol.** Its
> packaging — a unit file, a `default` file, a documented listen address — is admissible under §2.2's
> third form on exactly the same terms as any other shipped default, including §10.4's one-way rule.
> Its **security-guide prose** about a protocol it did not design is a third-party hardening opinion,
> and §2.3 governs it: corroboration, never sole grounds.

> **Amended by [#68](https://github.com/winniel123/verge-asm/issues/68) — the artefact class this
> definition did not enumerate.** A **cryptographic primitive's owner is the body whose standard
> specifies the primitive**, speaking about the primitive it specified: NIST owns SHA-1, the SHA-2
> family, ECDSA and the P-curves because FIPS 180-4, FIPS 186-5 and SP 800-186 are what those
> algorithms **are**, exactly as the IETF owns what its RFCs specify. **A body that specifies which
> primitives a population may *accept* — the CA/Browser Forum over the WebPKI, a root programme over
> its own trust store — is not thereby an owner of the primitive.** It is in the distributor position
> above: its restriction is admissible under §2.2's third form **over its own artefact**, and §2.3
> governs its prose. §10.4 rules *whether* a shipped default attests and never *what about*, so a
> floor over a relying party's acceptance attests about that acceptance and not about the algorithm.
> See [ADR-0035](../adr/0035-a-cryptographic-primitives-owner-is-its-specifier.md). No row moves.

**This is not a new line; it is the line §9.1 already walked and could not explain.** §9.1 read
Debian's `rpcbind.default` as evidence in the same breath as declining Red Hat's Security Guide, and
"a distributor is never the vendor" cannot account for both. The **artefact**, not the party, is what
the rule keys on.

**The rider the ticket asked for, because it decides the next case.** Would this be the line for a
distributor that does **not** contradict itself? **Yes, and the contradiction must not be the ground.**
Red Hat's self-contradiction — *"difficult to secure"* in one manual, *"Allow TCP and UDP port 111"*
in another — is *evidence that the line is right*, in the AWS pattern §2.3 was built on. It is not the
reason. Had Red Hat been consistent, its Security Guide would still be a hardening document addressed
to its own users, and *"use TCP Wrappers to limit which networks or hosts have access"* would still be
the shape §4.4 refused for 6443: a preference expressed against a real supported architecture. A
session that finds a consistent distributor and reads the door as open has read the wrong sentence.

**The cost, stated rather than smoothed.** The distributor is frequently the only party to say
anything at all about a daemon whose upstream is silent, and refusing that prose loses coverage on
exactly those daemons. The list accepts the loss, because the alternative admits one organisation's
two manuals with opposite instructions — which is the failure §2.3 exists to prevent, demonstrated on
the very first instance the standard met.

**No row moves.** 2049/tcp is carried by `nfs(5)` from nfs-utils (the reference implementation, not a
distributor product); 873/tcp by upstream `rsyncd.conf(5)` cited to the Samba copy precisely to avoid
a distribution rendering; 512/513/514 by OpenBSD's own deletions, which are OpenBSD speaking as the
author of the code it removed.

> **Amended by §12** ([#69](https://github.com/winniel123/verge-asm/issues/69)). *"On exactly the same
> terms as any other shipped default"* is unchanged and had a **latent condition**, now stated: the
> same terms include §10.3's requirement that the boundary be **named by the owner**, and a distributor
> cannot meet it for a protocol it did not design — nor §10.1's, which answers Claim 1 from the
> specification or the owner. **A distributor's shipped default is therefore never sole grounds for a
> row on this table**; it corroborates under §2.3 and stops there. *Artefact, not party* is untouched
> and is what keeps a distributor's packaging distinct from its security-guide prose. The distributor
> owns the **choice to install**; the owner owns the **claim**.

### 10.6 A fifth defect, found by closing the set — `161/udp`'s boundary rests on a corroborator

Closing the claim set (§10.2) obliges every row to name one of the three, and one row cannot.

**`161/udp` SNMP sits in Class C, and Claim 3's boundary limb — *SNMPv1/v2c assumes a management
network* — is carried in this note by CISA TA17-156A.** CISA is a corroborator under §2.3, not an
owner under §10.5. RFC 3410 §8.2 is the owner statement and it attests something else: *"SNMPv1 and
SNMPv2c … support only trivial authentication based on plain-text community strings and, as a result,
are fundamentally insecure."* That is insecurity, not placement. Claim 2 is unavailable on the §9.2
shape — SNMPv3 hardens **161 on 161**, exactly as StartTLS hardens 389 on 389 — and Claim 1 is
unavailable because a community string is a credential, however weak.

**§2.6's boast is therefore established for 37 rows and not for the 38th**, and the coincidence is
exact: `161/udp` is one of the two rows the abandoned escape hatch was *drafted to carry*, and §2.6
declares the hatch unnecessary by naming *"CISA's SNMP alert for 161"* as a first-party attestation.
It is not first-party. `623/udp` is genuinely fine — Dell's statement via CERT/CC is the owner
speaking about the DRAC it designed — which is why the defect shows on one row and not two.

**Ruling: `161/udp` stays on the list, disclosed as the note's second-weakest row beside 5432/tcp.**

Three reasons for disclosure rather than removal, in increasing order of force. Removing a row costs
a `Break` on every `sensitive-port-reached-from-internet` evaluation
([ADR-0008](../adr/0008-derivation-versions-move-on-content.md)) and a `verge-core` narrowing
([ADR-0009](../adr/0009-verge-core-is-a-union.md)). §4.5 has already set the house style for a row
whose footing is thin — name it, do not bury it. And decisively, **a row may not move on a re-reading
of text already in this note**; §9's own precedent is that a verdict changes on **retrieval**, and no
retrieval was performed here. The IETF's SNMP management-architecture documents (RFC 3411 and its
family) were **not** read, and the honest position is that a first-party placement sentence may well
exist and nobody has looked.

**Routed to [#66](https://github.com/winniel123/verge-asm/issues/66)** as a research ticket, on §8's
new question 9. **The criterion that would change the verdict:** a first-party IETF or SNMP-owner
sentence placing SNMPv1/v2c inside a management domain admits the row cleanly on Claim 3; its absence
means the row rests on a corroborator and must be removed before v1 ships, which is free now and
costs a comparability cycle later (§7.2).

### 10.7 Two "why" cells corrected — the row's grounds, not its colour

Corrected here rather than in §3's tables, per the name-and-withdraw convention. **Neither row's
grounds change and neither row moves.**

| Row | §3 reads | It should read |
|---|---|---|
| **11211/udp** memcached (Class A) | "Spoofed-source amplification vector; upstream disabled UDP by default in 1.5.6 for exactly this reason" | "Admits anonymous `set`/`delete` over UDP; upstream states outright that it must not be exposed to the internet or any untrusted user, and disabled UDP by default in 1.5.6" |
| **161/udp** SNMP (Class C) | "CISA directs that SNMP traffic be segregated onto a separate management network; SNMPv1/v2c authenticate on cleartext community strings" | "SNMPv1/v2c authenticate on cleartext community strings, which RFC 3410 §8.2 calls *fundamentally insecure*; CISA corroborates that SNMP traffic belongs on a separate management network. **See §10.6 — the boundary limb rests on the corroborator**" |

The 11211/udp correction matters beyond tidiness: as written, the cell leads with a **magnitude**,
which is the precise source class §2.7 refuses for 111/tcp. A note that refuses amplification as
grounds in §2.7 and leads with it in §3.1 is arguing against itself, and a reader who noticed would
be right to.

### 10.8 Class B's coverage boundary, stated in §2.1 rather than left in §9.2

Closes §8 question 8. The ticket that commissioned this repair deliberately kept the boundary out of
scope, and it is recorded here as the reason Class B is **not** being widened.

> **Claim 2 carries a permanent coverage boundary, and it is not a wording defect.** RFC 6335 §9
> tells implementers to prefer in-band security negotiation over split ports, and LDAP is the working
> instance: StartTLS and Microsoft's LDAP signing both harden 389 **on 389**, so there is no plaintext
> port to condemn and no Class B row is constructible. A `(port, transport)`-keyed signal is
> structurally blind to in-band upgrade, and RFC 6335 §9 means the pattern it is blind to is the one
> the IETF is steering implementers toward. Class B is therefore not merely a closing category — its
> replacement is invisible to this instrument by construction.

**The pressure discharges out of scope, not into a wider claim.** *Did this listener require
confidentiality before accepting a credential?* is an **observation**, not a curation — the line
[#31](https://github.com/winniel123/verge-asm/issues/31) drew and
[ADR-0025](../adr/0025-an-offer-is-scope-only-where-the-value-enumerates-it.md)'s siblings priced —
and the wire-protocol prober that would answer it was **ruled out of v1** in
[#41](https://github.com/winniel123/verge-asm/issues/41), at a measured cost of six ports and **zero
net new firings across all 38 rows**. Two notes for whoever reopens it: `insecure-listener-rules.md`
§5 preserves the per-protocol costing, and
[ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) has since **overturned** §9.2's rule that
a single-protocol signal is illegitimate — a signal is named for the fact it reads, so a rule covering
exactly one protocol is legitimate.

### 10.9 What this repair changed

**Nothing on the list.** 38 `(port, transport)` pairs. §1's summary table, §3's tables and §6.1's
containment arithmetic are untouched, and the arithmetic in
[ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union is untouched with them. No `Break`, no
version bump, no aperture change.

**Five things about the standard**, all of them narrowings:

1. Claim 1 has a **stated qualifier**, applied in two ordered steps against the specification (§10.1).
2. The claim set is **closed by construction**, over the three things an internet vantage supplies,
   with three candidate fourths tested and refused (§10.2).
3. Claim 3's boundary is **a boundary the owner names**, not literally "the same system" — a wording
   correction that admits nothing (§10.3).
4. A shipped default attests **one way**; the exclusion #30 found belongs to a different object, a
   **remedy that stops short of the port** (§10.4).
5. **Owner** is defined, and a distributor owns its **configuration** and not the **protocol** —
   which is what §9.1 was already doing (§10.5).

**And one thing about the list**: closing the claim set found `161/udp`'s boundary limb resting on a
corroborator, disclosed rather than smoothed and routed to retrieval (§10.6).

**The test every repair was weighed against.** The standard exists to keep *commonly attacked* out of
a list of *never correct*. All five changes are restrictive: Claim 1's qualifier refuses the most
attacked services on the internet by construction; closure shuts the door a fourth class would open;
Claim 3's rewording admits nothing; the one-way default rule tightens what may admit a row; and the
owner definition is unchanged in effect. **No repair here lets frequency back in**, and the one place
frequency tried to re-enter — amplification, in an unguarded "why" cell — is caught and corrected in
§10.7.

---

## 11. `161/udp` comes off the list — the retrieval §10.6 called for

Wayfinder ticket [#66](https://github.com/winniel123/verge-asm/issues/66), performing the retrieval
§10.6 named as the only thing that could move the row. This section **amends §1, §2, §3, §6, §7, §8
and §10 by reference**: earlier text is left standing and marked, per the name-and-withdraw
convention, and where §11 and an earlier section disagree, **§11 governs**.

**Headline result, stated first.**

> **`161/udp` SNMP is removed. The list is 37 `(port, transport)` pairs, Class C is 18 rows, and no
> other row moves.** The IETF's SNMP management-architecture documents were retrieved and read, and
> **no first-party placement sentence exists anywhere in them**. The reference implementation's
> position, retrieved, points the other way: net-snmp states that SNMP's secure form accepts or
> rejects a request *"irrespective of where it originated"*. Claim 3's boundary limb is therefore
> unattested by an owner, Claims 1 and 2 remain unavailable, and after §10.2 the claim set is closed —
> so a row that can name none of the three is not admissible.

**And §2.6's headline sentence is restored to every row on the list — by the row leaving, not by the
row being rescued.** That is the weaker of the two outcomes §10.6 contemplated and it is the honest
one.

### 11.1 What was retrieved

Ten RFCs and five net-snmp artefacts, all fetched as bytes and searched directly rather than through
a summarising layer, per §9.5 and [#46](https://github.com/winniel123/verge-asm/issues/46).

| Retrieved | What it is | Did it place SNMP? |
|---|---|---|
| RFC 3411 | *An Architecture for Describing SNMP Management Frameworks* (STD 62) | **No** |
| RFC 3412 | Message Processing and Dispatching (STD 62) | **No** |
| RFC 3413 | SNMP Applications (STD 62) | **No** |
| RFC 3414 | User-based Security Model (STD 62) | **No** |
| RFC 3417 | Transport Mappings (STD 62) | **No** — and it puts SNMPv3 on 161 (§11.5) |
| RFC 3584 | Coexistence between SNMP versions (BCP 74) | **No** |
| RFC 3410 | Applicability Statements for SNMP | **No** — §8.2 delegates to the operator (§11.2) |
| RFC 2570 | RFC 3410's predecessor | **No** |
| RFC 1157 | SNMPv1 itself (STD 15, Historic) | **No** — and §3.2.5 refuses locality outright (§11.2) |
| RFC 6353 | TLS Transport Model (Standards Track), the 10161 registration | **No** (§11.5) |
| net-snmp `snmpd(8)`, `snmpd.conf(5)`, upstream FAQ, wiki `TUT:Security`, `EXAMPLE.conf.def` | the reference implementation | **No** (§11.3, §11.4) |

**The measurement that carries the section.** Across all ten RFCs the strings `management network`,
`segregat`, `firewall`, `separate network`, `isolated network`, `private network`, `untrusted
network`, `public network` and `out of band` occur **twice in total**, both in RFC 6353 and both
about *certificate distribution* — *"a trusted out-of-band mechanism"* for exchanging fingerprints,
and *"an out-of-band transfer"* of trust anchors. **Zero occurrences of any of them describe where an
SNMP agent belongs on a network.** This is the §9.2 method applied to SNMP: the documents are citable
for what they do not contain, and the absence is total rather than partial.

### 11.2 The IETF documents, and the two near-misses that are the whole trap

**`administrative domain` is a naming scope, not a network.** The phrase occurs 8 times in RFC 3411,
once each in RFC 3412, RFC 3414 and RFC 6353, and **every occurrence bounds an identifier's
uniqueness**:

> "Within an administrative domain, an snmpEngineID is the unique and unambiguous identifier of an
> SNMP engine. … Note that it is possible for SNMP entities in different administrative domains to
> have the same value for snmpEngineID."
> — [RFC 3411](https://www.rfc-editor.org/rfc/rfc3411.txt), §3.1.1.1

The same for `contextEngineID` (§3.3.2), for the `contextEngineID`/`contextName` pair (§3.3.1) and for
the `snmpEngineID` textual convention's guidance that a generated value be *"unique in the agent's
administrative domain"*. `management domain` (§3.3.1) is the same idea for contexts. **Reading either
phrase as a network boundary is §10.6's error one level down** — a term lifted out of the context that
gives it its only operative meaning, which is precisely
[#46](https://github.com/winniel123/verge-asm/issues/46)'s finding.

**RFC 1157 defines SNMP's own membership concept with no locality in it at all.** This is the
strongest single sentence retrieved, and it points against the row:

> "A pairing of an SNMP agent with some arbitrary set of SNMP application entities is called an SNMP
> community."
> — [RFC 1157](https://www.rfc-editor.org/rfc/rfc1157.txt), §3.2.5, *Definition of Administrative Relationships*

*Arbitrary set.* The specification's word for who may talk to an agent is a **credential-scoped
relationship**, chosen in the section explicitly titled for administrative relationships, and it
declines to say anything about where those entities sit. RFC 1157's Security Considerations reads in
full: *"Security issues are not discussed in this memo."*

**RFC 3410 §8.2's deployment sentence is a delegation, not a placement claim.** §10.6 quoted the
insecurity sentence; the section's *operative* deployment sentence is the next one, and it hands the
boundary to the operator rather than naming one:

> "Of course, it is important that users deploying multi-lingual systems with insecure protocols
> exercise sufficient due diligence to insure that configurations limit access via SNMPv1 and SNMPv2c
> appropriately, **in keeping with the organization's security policy**, just as they should carefully
> limit access granted via SNMPv3 with a security level of no authentication and no privacy"
> — [RFC 3410](https://www.rfc-editor.org/rfc/rfc3410.txt), §8.2 (emphasis added)

Two things about it. It governs **access control**, not network position — *limit access*, not *place
it here*. And it is **conditional on the organisation's own policy**, which is exactly the shape §9.3.1
refused for finger and §4.4 refused for 6443: a preference expressed against whatever architecture the
operator actually has. A sentence that defers to the reader's policy cannot be the owner naming a
boundary.

**RFC 3411 §9 does the same one level up**, and it is the architecture document — the single most
likely home for a placement sentence:

> "It is the responsibility of the purchaser of an implementation to ensure that: … 2) the Security and
> Access Control Models utilized satisfy the security and access control needs of the organization"
> — [RFC 3411](https://www.rfc-editor.org/rfc/rfc3411.txt), §9

**The threat model is retrieved and confirms the shape.** RFC 3411 §1.4 and RFC 3414 §1.1 enumerate
modification of information and masquerade as principal threats, disclosure and message-stream
modification as secondary, and expressly exclude denial of service and traffic analysis. Every one is
a property of the *message*; not one is a property of the *network the agent sits on*. RFC 3414 §1.2's
four goals are integrity, identity, timeliness and confidentiality — again, no placement.

**The one sentence that reads like locality, and does not survive its context.** RFC 3411 §1.4 says of
traffic analysis that *"entities may be managed on a regular basis by a relatively small number of
management stations - and therefore there is no significant advantage afforded by protecting against
traffic analysis."* It is a **reason for declining to defend against a threat**, and quoting it as a
statement about SNMP's intended audience is the truncated-conditional shape verbatim.

### 11.3 net-snmp, the reference implementation — and it points the other way

Under §10.5 net-snmp is an **owner**: it authors the reference implementation. Four retrieved facts,
in increasing order of force.

**Its shipped default is permissive, so under §10.4 it is silent.** The ticket anticipated this:

> "By default, **snmpd** listens for incoming SNMP requests on UDP port 161 on all IPv4 interfaces."
> — [`snmpd(8)`](http://www.net-snmp.org/docs/man/snmpd.html), LISTENING ADDRESSES

`snmpd.conf(5)` documents `agentaddress` as *"a list of listening addresses, on which to receive
incoming SNMP requests"* and carries **no exposure statement of any kind**. The only loopback default
anywhere in the manual belongs to a different port: SMUX on 199, where `--enable-local-smux` *"causes
it to only listen on 127.0.0.1 by default"*.

**Its access-control default restricts by credential, and confirms Claim 1's unavailability by
measurement rather than by assertion.** §10.6 refused Claim 1 on the reasoning that a community string
is a credential; the reference implementation says it outright:

> "By default, the Net-SNMP agent starts up with a completely empty access control configuration. This
> means that *no* SNMP request would be successful."
> — [net-snmp FAQ](http://www.net-snmp.org/FAQ.html), *Why does the agent complain about 'no access control information'?*

That is a **restricting** default, so §10.4 admits it — but it restricts on the **principal**, not on
the network, so it attests against Claim 1 and says nothing about Claim 3.

**Its one sentence about port 161 and a boundary is an option in a menu, not a position.** Under *How
can I stop other people getting at my agent?* — a question the answer opens by asking *"Firstly, are
you concerned with read access or write access?"* — the FAQ offers three external mechanisms:

> "Other options include:
>   - Blocking access to port 161 from outside your organisation (using filters on network routers)
>   - Using kernel-level network filtering on the system itself (such as IPTables)
>   - Configuring TCP wrapper support ("--with-libwrap")"
> — [net-snmp FAQ](http://www.net-snmp.org/FAQ.html)

This is the closest thing to a placement sentence in the entire corpus and it is not one. It is a
hardening option offered conditionally, on the same footing as `iptables` and `hosts.deny`, and §9.1
already refused Red Hat's *"use TCP Wrappers to limit which networks or hosts have access"* as exactly
this shape. It names no boundary SNMP *assumes*; it names a boundary an operator *may impose*.

**And the decisive sentence points the other way.** In the same answer, the owner states that the
protocol's secure form is location-independent **by design**:

> "For strict security you should use only SNMPv3, which is the secure form of the protocol. However,
> note that the agent access control mechanisms does not restrict SNMPv3 traffic by location - an
> SNMPv3 request will be accepted or rejected based purely on the user authentication, **irrespective
> of where it originated**."
> — [net-snmp FAQ](http://www.net-snmp.org/FAQ.html) (upstream's *"mechanisms does"* preserved; emphasis added)

**This is the [#30](https://github.com/winniel123/verge-asm/issues/30) shape**: the row moves from
*no evidence found* to *evidence found pointing the other way*. SNMP's own reference implementation
says the secured protocol deliberately does not read network position — which is the opposite of a
protocol that assumes its callers are inside a boundary.

### 11.4 The strongest counter-argument, retrieved and refused

There is a real case for keeping the row and it is not the one §10.6 was worried about. It is a
**restricting shipped default**, which §10.4 admits and §2.2's weak tier already relies on for
5432/tcp, 5984/tcp and 9042/tcp.

**[measured]** Upstream net-snmp's `EXAMPLE.conf.def` opens its AGENT BEHAVIOUR block with the
loopback line **active** and the all-interfaces line **commented out**:

```
#  Listen for connections from the local system only
agentAddress  udp:127.0.0.1:161
#  Listen for connections on all interfaces (both IPv4 *and* IPv6)
#agentAddress udp:161,udp6:[::]:161
```

— [`EXAMPLE.conf.def`](https://raw.githubusercontent.com/net-snmp/net-snmp/master/EXAMPLE.conf.def), net-snmp upstream

And Debian **installs that file verbatim as the operative `/etc/snmp/snmpd.conf`** — retrieved from
`net-snmp_5.9.5.2+dfsg-2.1.debian.tar.xz`, `debian/snmpd.conf`, byte-identical in this block except
for the commented IPv6 literal. So the modal Debian install of SNMP genuinely binds loopback only.

**It loses, on three independent grounds, and each is worth stating because the next case will look
like this one.**

1. **Upstream it is an example, not a default.** §2.2's third form is *the project's shipped default*.
   net-snmp's own FAQ says of `snmpd.conf`: *"It doesn't exist in the distribution as shipped. You need
   to create it to reflect your local requirement."* The file's own header says *"Some entries are
   deliberately commented out, and will need to be explicitly activated"*, and its network example is
   `#rocommunity secret 10.0.0.0/16` under *"Adjust this network address to match your local
   settings"* — a template to be edited, not a boundary. The agent's **actual** default, documented in
   `snmpd(8)`, is all IPv4 interfaces.
2. **Debian's copy is a shipped default, and Debian is not the owner.** §10.5 admits a distributor's
   packaging under §2.2's third form — so §10.4 does not silence it, because it restricts. But §10.3
   requires that **the boundary be named by the owner**, and §10.5's other half says a distributor
   *"owns nothing about the protocol"*. Debian's file attests about Debian's package. It cannot attest
   about SNMP's intended audience, and admitting it would let a packaging choice carry a normative row
   — the door §2.3 exists to hold shut, arriving through a `.deb` instead of a government PDF.
3. **The restriction is incoherent as a statement about SNMP's audience.** An SNMP agent listening only
   on loopback can be polled by **no management station at all**, which is the protocol's entire
   purpose. The line is a secure-out-of-the-box posture — configure before use — not a claim about
   where SNMP belongs. This is the load-bearing difference from 5432/tcp: `listen_addresses =
   localhost` describes a **real and common** PostgreSQL deployment, an application tier on the same
   host, so the default and the claim say the same thing. Loopback-only SNMP describes nothing anybody
   runs.

**Recorded as a general defect in the repaired standard, because it will recur.** The same bytes are
*silent upstream documentation* under reading 1 and an *admitting shipped default* under reading 2, and
§2.2/§10.4/§10.5 do not say which governs. It did not decide this row — both readings lose on §10.3 —
but it decides the next daemon whose upstream ships an `EXAMPLE.conf` a distribution installs. Opened
as [#69](https://github.com/winniel123/verge-asm/issues/69).

> **Closed by §12** ([#69](https://github.com/winniel123/verge-asm/issues/69)). **Reading 1 governs**,
> so grounds 1 and 2 above are now applications of a rule rather than case reasoning: an example
> attests nothing in either direction (§12(a), §12(b)), and installing one transfers **operativeness**
> and not **ownership**, so a distributor's shipped default corroborates and never carries a row
> (§12(c)). **Ground 3 is not promoted** — *the restriction describes nothing anybody runs* was
> considered as a general gate and refused, because *does anybody run this?* is the ownerless
> counterfactual §10.1 deleted (§12.4). It stays a sanity check on this row and nothing more. `161/udp`
> does **not** re-open: §11.6 rests on §10.3's owner requirement and §10.2's closed claim set, neither
> of which §12 touches.

### 11.5 Claim 2 tested against RFC 6353's 10161 — refused, and not on deployment share

The ticket asked whether `snmptls`/`snmpdtls` on 10161 satisfies Claim 2's successor clause, which
would admit the row without any placement sentence. **It does not.**

**The registration is real, and materially stronger than LDAP's.** RFC 6353 is Standards Track and
registers four ports itself, with an RFC reference — unlike §9.2's `ldaps,636`, a bare legacy IANA row
assigned to an individual with no RFC behind it:

> "snmptls 10161/tcp SNMP-TLS [RFC6353] · snmpdtls 10161/udp SNMP-DTLS [RFC6353] · snmptls-trap
> 10162/tcp SNMP-Trap-TLS [RFC6353] · snmpdtls-trap 10162/udp SNMP-Trap-DTLS [RFC6353]" … "These are
> the default ports for receipt of SNMP command messages (snmptls and snmpdtls) and SNMP notification
> messages (snmptls-trap and snmpdtls-trap) over a TLS Transport Model as defined in this document."
> — [RFC 6353](https://www.rfc-editor.org/rfc/rfc6353.txt), §8 IANA Considerations

**And it still fails, on Claim 2's own discriminator rather than on how much of the internet runs it.**
§4.2 and §9.2 state the clause's job: *"the encrypted sibling is not an exception to the rule, it is the
rule — its existence is what makes the plaintext port wrong"*. 23 is wrong **because** 22 exists and
the fix is not on 23. For SNMP the fix is on 161:

> "It is suggested that administrators configure their SNMP entities supporting command responder
> applications to listen on UDP port 161."
> — [RFC 3417](https://www.rfc-editor.org/rfc/rfc3417.txt), §3.2, *Well-known Values* (STD 62)

RFC 3417 is the SNMPv3 framework's own transport mapping, and RFC 3414's fourth goal is to *"Provide,
when necessary, that the contents of each received SNMP message are protected from disclosure."* So a
fully authenticated **and encrypted** standards-track SNMP — v3/USM at `authPriv` — is reached on
`161/udp` itself. **SNMP hardens 161 on 161**, exactly as StartTLS hardens 389 on 389, and §9.2's
verdict transfers without modification.

**The IETF's own stated remedy was the version, not the port.** RFC 3410 §8.2 declares SNMPv1 and
SNMPv2c Historic *"to send a clear message that the third version of the Internet Standard Management
Framework is the framework of choice"* — nine years before RFC 6353 existed. **[measured] the string
`161` does not occur in RFC 6353 at all** (only `10161` and `10162`). RFC 6353 does not deprecate 161,
does not reference it, and does not describe itself as its successor; it adds a transport model
alongside the existing one.

**The scepticism the ticket asked for is not needed, and the note refuses to lean on it.** *A registered
but undeployed successor* would have been the easy refusal, and it is unavailable on the evidence:
`snmpd(8)` lists `dtlsudp` among its transport specifiers, so TLSTM is implemented, not vapour. The
refusal rests on structure — the fix lives on the port — which is the ground that would survive TLSTM
becoming universal tomorrow.

### 11.6 Ruling, and every number that moves

> **`161/udp` is removed from the list.** No permitted claim is available to it: Claim 1 fails on a
> credential the reference implementation confirms it demands, Claim 2 fails on §9.2's shape, and
> Claim 3's boundary limb has **no owner attestation of any kind** after retrieval. §10.2 closed the
> claim set, so *no claim* means *no row*.

**§10.6's own criterion is met.** It wrote: *"a first-party IETF or SNMP-owner sentence placing
SNMPv1/v2c inside a management domain admits the row cleanly on Claim 3; its absence means the row rests
on a corroborator and must be removed before v1 ships."* The retrieval was performed and the sentence
does not exist. §10.6's *"stays on the list, disclosed"* is **withdrawn**.

**Every dependent figure, walked rather than asserted:**

| Where | Was | Is |
|---|---|---|
| §1 summary table | 38 pairs | **37 pairs** |
| §2.2 footing table, "explicit prohibition" row | includes `161 SNMP` | **`161 SNMP` struck** — §10.7 had already found the cell wrong, and there is now no row to place |
| §3 preamble, §3.3 Class C | 38 total, 19 Class C rows | **37 total, 18 Class C rows.** Class A 12 + Class B 7 + Class C 18 = 37 |
| §3.4 SNMP quote block | RFC 3410 §8.2 + CISA TA17-156A | **Retained, marked** — citable now only as the record of what the row rested on |
| §6.1 "in the hot set already" | 28 of 38 | **28 of 37** |
| §6.1 "missing, whole transport off by default" | **6** UDP rows: 69, 137, 138, **161**, 623, 11211 | **5** UDP rows: 69/udp, 137/udp, 138/udp, 623/udp, 11211/udp. 28 + 4 + 5 = 37 ✓ |
| §6.2 | names `161/tcp` vs `161/udp` as a live hot-set defect | **Half discharged.** The 623/tcp-vs-623/udp defect stands; the 161 half dissolves, because containment no longer requires `161/udp` at all |
| §7.1 route 4 | "The six UDP rows" | **five** |
| §8 question 9 | open | **closed by §11** |
| §10.6 | ruling: stays, disclosed | **withdrawn** (above) |
| §10.9 | "Nothing on the list. 38 pairs" | **Superseded**: one row leaves, on retrieval §10.9 correctly said had not happened |

**Outside this note.** [ADR-0009](../adr/0009-verge-core-is-a-union.md) defines
`verge-core = frequency-set ∪ sensitive-list`, so the union **narrows by one member**. Its §*UDP is a
transport capability, not a list* says *"the sensitive half contributes six"* UDP pairs — **now five** —
and its §*"Correcting a mis-aimed port"* worked example says *"161/udp and 623/udp enter. Plain
`revealed`"*; **`161/udp` no longer enters**, because the frequency half is TCP-only
([#4](https://github.com/winniel123/verge-asm/issues/4) §2.5) and ADR-0009 attributes 161 to the
sensitive half. `161/udp` leaves `verge-core` entirely. Amendment recorded on the ADR.
[ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)'s
[#44](https://github.com/winniel123/verge-asm/issues/44) amendment uses *"no `Service` is observed on
161/udp"* as its worked example of a rule with no subject; the example is now stale in its subject and
unchanged in its point — any of the five surviving UDP pairs substitutes for it.

**What it costs, per [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) and
[ADR-0009](../adr/0009-verge-core-is-a-union.md), and why now.** The edit bumps
`sensitive-port-reached-from-internet`'s rule version and `Break`s every evaluation, and it narrows
`verge-core`, which is an aperture change. **Both are vacuous before the first install** — ADR-0009
already ruled that vacuity is not a grace period — so the price today is zero and the price after v1 is
a comparability cycle on the product's best signal. §7.2's symmetric-cost argument is what makes this
the right moment: the tight list is better precisely because its errors are discoverable, and this one
was discovered.

**The honest loss, stated rather than smoothed.** SNMP is among the most exposed management protocols
on the internet, and after this edit `sensitive-port-reached-from-internet` says **nothing about it**.
The note accepts that, for the reason §2.7 accepted it for `111/tcp`: we could not find anyone entitled
to say that exposing it is never correct, and a row backed by whichever authoritative-looking document
was nearest is the failure this standard exists to prevent. The row that had the most obviously
"correct" verdict attached to it is the row with no owner behind it, and that is the finding.

### 11.7 What §2.6 now says, and why it is a weaker sentence than it looks

§2.6's boast is **restored, and its second sentence needs one word of care**:

> **No row on this list rests on a government or cloud-provider source alone.** Every one of the **37**
> is attested by the specification, the project, or the vendor that owns the protocol.

It is now true of every row, because the row that falsified it is gone. Two riders bind whoever quotes
it next. It was **not** restored by finding an attestation, so it is evidence about the standard's
tightness and **not** evidence that the standard's coverage is complete — §2.7's `111/tcp` and this
row are the two measured cases of a port that is indefensible in practice and unlistable on the
evidence. And §2.6's abandoned escape hatch was drafted to carry **two** rows; one of them is now off
the list and the other, `623/udp`, is carried by Dell speaking about the DRAC it designed. **The hatch
would have carried exactly the row that turned out not to belong** — which is the strongest available
statement that abandoning it was right, and it is a stronger statement than the one §2.6 makes.

### 11.8 Thin ground, and the criterion that would change the verdict

**Flagged per the effort's standing rule.** This is a **negative** finding over a corpus chosen by us.
The claim is *no first-party placement sentence exists in the SNMP standards family or in the reference
implementation's documentation*, and it is established over ten RFCs and five net-snmp artefacts —
which is the family §10.6 named plus RFC 1157, RFC 3417 and the upstream example config. It is not
established over every SNMP implementation on earth. A vendor SNMP stack — Cisco's, Juniper's — might
carry a placement sentence about its own agent, and under §10.5 that vendor would own **its** agent and
not the protocol, which is why those were not pursued: a Cisco sentence cannot attest a `(port,
transport)` row about SNMP in general, only about Cisco.

**What would change the verdict, in one line.** A first-party sentence from the IETF or from net-snmp
placing SNMPv1/v2c inside a network boundary — not limiting *access*, not recommending a *filter*, but
stating that the protocol assumes its callers are inside one. Re-adding the row on such a sentence
would be a new admission priced at a version bump and an aperture widening, never a reversal of this —
ADR-0009's rule that a correction is a removal plus an addition, each priced separately.

### 11.9 Retrieval hazards met, recorded per §9.5

- **`sources.debian.org` and `salsa.debian.org` both serve a proof-of-work JavaScript challenge** to a
  plain client, byte-identical in shape to the LACNIC shell §9.5 recorded. Debian's packaging was
  obtained instead from `deb.debian.org`'s pool as `net-snmp_5.9.5.2+dfsg-2.1.debian.tar.xz` and
  extracted locally, which is a retrieval of the shipped bytes rather than of a rendering of them.
- **The net-snmp wiki page `TUT:Security` contains no security prose** — the retrieved document is a
  MediaWiki shell in which none of `firewall`, `Internet`, `expose`, `localhost`, `untrusted` or
  `public network` occurs. Recorded so that a later session does not read its title as a position.
- **`snmpd.conf(5)` and `snmpd(8)` on net-snmp.org are `man2html` renderings dated 2002 and 2005**, and
  the pages carry internal `http://localhost/cgi-bin/man/man2html` links from the generator. The
  substantive text was cross-checked against the current upstream `EXAMPLE.conf.def` on GitHub, which
  agrees.
- **The `ldaps`-matches-`LDAPString` trap has an SNMP analogue and it was avoided**: a case-insensitive
  search for `161` matches page numbers, RFC numbers and `10161`. The counts in §11.5 use a word
  boundary, which is why RFC 6353 reads zero rather than four.

## 12. An example config attests nothing, and installing one transfers operativeness rather than ownership

Wayfinder ticket [#69](https://github.com/winniel123/verge-asm/issues/69), on the defect §11.4
recorded as general rather than fixed. This section **amends §1, §2.2, §10.4 and §10.5 by
reference**: earlier text is left standing and marked, per the name-and-withdraw convention, and
where §12 and an earlier section disagree, **§12 governs**.

**Headline result, stated first.**

> **Upstream's reading governs, and it governs because §2.2's third form already has two limbs.** It
> reads *"the project's **shipped default**, as documented by the project"* — the configuration that
> **takes effect without the operator acting**, and that the **project documents as its default**. An
> example file satisfies neither: nothing starts from it, and where the project documents a default
> at all it documents a different one. **An example config attests nothing, in either direction.**
>
> **Installing one transfers operativeness, never ownership.** A distributor that installs upstream's
> example as its package's operative configuration has made that file a shipped default **of its own
> package**, admissible under §10.5 on exactly the terms §10.5 states. But every one of §2.1's three
> claims is answered from the specification or from the owner — §10.1 says so in terms, §10.3 says so
> for the boundary limb, and Claim 2 is a fact about the wire and the registry — so **a distributor's
> shipped default is never sole grounds for a row on this table**. It corroborates, on §2.3's terms,
> and does nothing more.
>
> **No row is added and none is removed. The list stays at 37 `(port, transport)` pairs.** §1's count,
> §3's tables and class totals, §6.1's containment arithmetic and
> [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union are all untouched, and each was checked
> rather than assumed (§12.7). **One footing changes, and it changes because the shipped bytes were
> read for the first time:** `9042/tcp` Cassandra carries an owner prohibition §2.2 says does not
> exist, so it leaves the weak tier. **The weak tier is two rows, not three.**

### 12.1 The rule

> **§12 — an example is not a default, and installation is not ownership.**
>
> **(a) The third form reads what takes effect, in the hand of the party being quoted.** §2.2's
> *shipped default* is the configuration a user gets **without acting**. A file the party in question
> does not install — an `EXAMPLE.conf`, a `*.conf.sample`, a `*.conf.example`, a `*.default` its
> packaging copies nowhere — is not that party's shipped default, whatever it contains and whoever
> wrote it. Where the file's own text disclaims operativeness, that disclaimer is the author's own
> statement and it settles the question against the file. **And where the party documents a default
> elsewhere and the two disagree, the documented default governs**, because the third form asks for
> the default *as documented by the project* and only one of the two is.
>
> **(b) A directive is not a position; prose in a config file may be.** §2.2's **second** form takes a
> quotable position wherever the owner wrote it, and a comment in a shipped configuration file is the
> owner's prose exactly as a manual page is. What the second form does **not** take is a **directive**
> — an instruction to software attests only through the third form — nor a **label** describing what
> the directive beneath it does. *"For security reasons, you should not expose this port to the
> internet"* is a position. *"# Listen for connections from the local system only"* is a label. The
> position-versus-preference discrimination is §2.3's and §4.4's, unchanged.
>
> **(c) Installation transfers operativeness, not ownership.** Where a distributor installs another
> party's bytes as its package's operative configuration, the artefact is a shipped default **of that
> package** and attests **about that package**. Every claim in §2.1 is a claim about the protocol, or
> about the service as its owner ships it, and every one of them is answered from the specification or
> the owner (§10.1's two steps, §10.3's boundary limb). **A distributor's shipped default is therefore
> never sole grounds for a row on this table.** §10.5's *"on exactly the same terms as any other
> shipped default"* is unchanged and had a latent condition, now stated: the same terms include
> §10.3's requirement that the boundary be **named by the owner**, and a distributor cannot meet it
> for a protocol it did not design. The distributor owns the **choice to install**; the owner owns the
> **claim**.
>
> **(d) Silent in both directions.** §10.4 makes a permissive default silent. An example is weaker
> than a default and is silent **whichever way it points** — it cannot admit a row and it cannot
> exclude one. In particular it is **not** a §10.4.3 remedy: that route needs a restricting act the
> owner **took**, and an act that takes effect nowhere was not taken.

**Why §10.4's own test decides this, against the reading it appears to support.** The ticket's
argument for admitting the example is that a maintainer who comments out one line and leaves the
other active has made a choice and paid for it — §10.4's *costly act*. Applied rather than invoked,
that test **refuses** the example. §10.4.2's cost is named exactly: a restriction *"buys friction at
first run and the maintainer paid for it anyway"*. The friction is borne by **users at first run**,
and the maintainer pays for it in support. A file nobody's daemon reads produces no first run, so it
costs its author nothing and it is cheap talk, which is precisely the thing §10.4 distinguishes a
default from. Where the friction **is** paid — because a distributor installed the file — the party
who paid is the distributor, which is limb (c).

### 12.2 The test is read off the file rather than judged — nine artefacts, nine self-declarations

The objection that would sink limb (a) is that *"takes effect"* sounds like something a reviewer must
adjudicate. It is not. **[measured]** Every configuration artefact retrieved for this ticket says what
it is, in its own bytes, and the two categories never blur:

| Artefact | What the file says about itself | Status |
|---|---|---|
| net-snmp `EXAMPLE.conf.def` | *"An example configuration file for configuring the Net-SNMP agent ('snmpd')"* · *"Some entries are deliberately commented out, and will need to be explicitly activated"* · *"See the 'snmpd.conf(5)' man page for details"* | **Example** |
| RabbitMQ `rabbitmq.conf.example` | *"This file is AN EXAMPLE. It is NOT MEANT TO BE USED IN PRODUCTION."* — and every directive in it is commented, including both `listeners.tcp.default = 5672` and `listeners.tcp.local = 127.0.0.1:5672` | **Example** |
| Debian `debian/rpcbind.default` | *"Uncomment the following line to restrict rpcbind to localhost only for UDP requests"* — and the line is commented | **Menu; the restriction was not taken** |
| PostgreSQL `postgresql.conf.sample` | *"The commented-out settings shown in this file represent the default values."* | **Record of defaults** |
| Kibana `config/kibana.yml` | *"The default is 'localhost', which usually means remote machines will not be able to connect."* | **Record of defaults** |
| CouchDB `rel/overlay/etc/default.ini` | *"Upgrading CouchDB will overwrite this file."*, beside `local.ini`: *"Custom settings should be made in this file … unlike changes made to default.ini, this file won't be overwritten on server upgrade"* | **Operative** |
| Redis `redis.conf` | *"So by default we uncomment the following bind directive"* · *"IF YOU ARE SURE YOU WANT YOUR INSTANCE TO LISTEN TO ALL THE INTERFACES COMMENT OUT THE FOLLOWING LINE."* | **Operative** |
| Cassandra `conf/cassandra.yaml` | no template disclaimer anywhere; `listen_address: localhost` and `rpc_address: localhost` both active | **Operative** |
| Debian `debian/memcached.conf` | *"memcached default config file … This configuration file is read by the start-memcached script provided as part of the Debian GNU/Linux distribution."*, `-l 127.0.0.1` active | **Operative (distributor's)** |

> **Amended by §13.6** ([#70](https://github.com/winniel123/verge-asm/issues/70)). **Ten artefacts,
> nine self-declarations.** **[measured]** etcd's `etcd.conf.yml.sample` opens *"This is the
> configuration file for the etcd server."* — an operative self-description on a `.sample` file nothing
> installs. Limb (a)'s second sentence resolves it and the outcome is unchanged, but *nine for nine* is
> a sample rather than a law, exactly as §12.8 anticipated. The `.sample` **suffix** is not a
> self-declaration either — that is surface syntax arriving through the filename.

**Nine for nine.** The distinction limb (a) turns on is not a judgement about deployment reality; it
is a sentence in the artefact, retrievable by the same method §9.5 and §11.9 already require. That is
what makes §12 a test read off the document rather than the counterfactual §10.1 deleted.

**And the surface syntax is worthless as a signal, which is why the rule cannot key on it.** A
commented-out `listen_addresses` line in PostgreSQL's sample **is** the default; a commented-out
`agentAddress` line in net-snmp's example is the branch not taken; a commented-out `-h 127.0.0.1` in
Debian's `rpcbind.default` is an offer to the operator. Same syntax, three meanings, and only the
file's own prose separates them.

### 12.3 The option that lost, stated at length because it is a good argument

**Reading 2: the artefact governs, so the distributor's install decides.** §10.5 says in terms that
the rule keys on the **artefact**, not the party. The artefact a Debian operator's daemon actually
reads is `/etc/snmp/snmpd.conf`; what upstream calls those bytes is irrelevant to what runs. The list
is about the world, and in the world the modal Debian SNMP agent binds loopback only. On that reading
the file is a restricting shipped default, §10.4 admits restricting defaults, and the row is admitted.

**It loses on three independent grounds.**

1. **It answers a different question.** §2.2's three forms are routes to one of §2.1's **claims about
   the protocol**, not to a description of the modal install. Deployment share is exactly what §2.3
   refuses — a risk list is a fact about the world, a never-list is a claim about legitimacy — and
   §11.5 refused a deployment-share argument by name on this very port. *Most installs bind loopback*
   is a frequency statement, and §1's framing rules frequency out at the top of the note.
2. **It is not stable under repackaging, and the proof is in one archive.** **[measured]** Debian
   ships net-snmp bound to loopback and rpcbind bound to `0.0.0.0:111` — `debian/snmpd.conf` versus
   `debian/rpcbind.socket`'s `ListenStream=0.0.0.0:111` — from the same archive under the same policy.
   A rule whose verdict depends on which distributor a reader checks, and on which of that
   distributor's files, is not a rule. §2.3 was built on exactly this failure: one organisation, two
   documents, opposite instructions.
3. **It re-opens the door §2.3 holds shut, one level down.** A packaging choice would carry a
   normative row — §11.4's *"the door §2.3 exists to hold shut, arriving through a `.deb` instead of a
   government PDF"*. §12 generalises that sentence into limb (c) rather than leaving it as an
   observation about one file.

**What reading 2 was right about, and what §12 keeps.** §10.5's *artefact, not party* is correct and
untouched: a distributor's packaging is a different kind of object from a distributor's security-guide
prose, and §12 does not collapse them. What §12 adds is that the artefact route terminates at §10.3's
owner gate, so it reaches corroboration and stops there.

### 12.4 The second option that lost — a coherence gate, refused

§11.4's third ground was that a loopback-only SNMP agent *"can be polled by no management station at
all"*, so the restriction describes nothing anybody runs, and it named the contrast with
`listen_addresses = localhost`, which describes a real and common PostgreSQL deployment. It is
tempting to promote that into a general gate: **a restricting default attests only where the
restriction describes a deployment somebody actually runs.**

**Refused, and refused on §10.1's own ground.** *Does anybody run this?* is a judgement about
deployment reality with no owner, and it is the shape §10.1 deleted when it removed *"would otherwise
require authority"* for asking a reviewer to imagine a counterfactual. It would also need frequency
evidence, which §1 excludes as a matter of framing. §11.4's third ground stays where it is — a sanity
check on one row, explicitly not load-bearing there either, since that row was decided by §10.3.

**The criterion that would reopen it:** a candidate row that passes §12 on the **owner's own**
operative default, where the same owner's documentation elsewhere describes that configuration as
unusable. The incoherence would then be attested rather than reasoned, and it would be an owner
statement like any other.

### 12.5 The walk, direction one — refusing examples: does any listed row lose its footing?

Only a row footed on a shipped default can be touched, and §2.2's own footing table names them: the
two upper tiers are prose (an explicit prohibition, or explicit trusted-network scoping), and the
bottom tier is the weak one. **Each was re-verified against shipped bytes rather than against the
documentation page the note originally cited.**

| Row | Footing, verified against shipped bytes | §12 verdict |
|---|---|---|
| **5432/tcp PostgreSQL** | `src/backend/utils/misc/postgresql.conf.sample` line 60 is `#listen_addresses = 'localhost'` — **commented** — and the file's header says *"The commented-out settings shown in this file represent the default values."* The binary agrees: `guc_tables.c` carries `{"listen_addresses", …}, &ListenAddresses, "localhost"`. The manual documents it independently | **Stands.** A record of a documented default, not an example; both limbs of (a) pass, three ways over |
| **5984/tcp CouchDB** | `rel/overlay/etc/default.ini`, `[chttpd]` → `bind_address = 127.0.0.1`, **active and uncommented**, in the file CouchDB says it overwrites on upgrade, with `local.ini` as the place operators are told to edit | **Stands**, and it is the cleanest instance of §2.2's third form in the note: the project separates its default from its template and labels both |
| **9042/tcp Cassandra** | `conf/cassandra.yaml`, `rpc_address: localhost`, **active** — in a file with no template disclaimer — and, immediately above it, a prohibition (§12.7) | **Stands, and its footing improves** |

**Two more rows are named because a reader will otherwise wonder.**

- **6000/tcp X11.** §3.4 already refused `-nolisten tcp`, on the ground that it is *"distribution and
  build-time behaviour that X.Org does not document as a default"*. That refusal is limbs (a) and (c)
  arrived at case by case, before either existed; §12 supplies the rule behind it. The row is
  unaffected either way — it rests on `Xsecurity(7)`, which is prose from the party that wrote the
  code.
- **11211/tcp memcached.** **[measured]** Debian's `debian/memcached.conf` ships `-l 127.0.0.1` and
  `-l ::1` **active**, under *"This parameter is one of the only security measures that memcached
  has, so make sure it's listening on a firewalled interface."* This is limb (c) working in the benign
  direction: a distributor's restricting default that agrees with the owner's prohibition, corroborates
  it under §2.3, and carries nothing. The row never rested on it.
- **2181/tcp ZooKeeper.** Upstream ships `conf/zoo_sample.cfg`, which is an example by name — and
  **[measured]** it contains no address or bind directive of any kind, only `clientPort=2181`. There
  is nothing in it to admit or refuse. The row rests on the Administrator's Guide prose.

**No listed row moves.**

### 12.6 The walk, direction two — admitting examples: does any §4.6 exclusion re-open?

A rule admitting examples could only **add**, since §10.4 makes the third form an admission route
only, so the whole check is §4.6's sixteen exclusions. Six could conceivably be touched by a
configuration file; the other ten are excluded on express purpose, on determinacy, or on an owner
sentence naming the internet as a supported environment, and no configuration file bears on any of
those grounds.

| Excluded | What a config file could have supplied | Why it does not re-open |
|---|---|---|
| **111/tcp rpcbind** | [#30](https://github.com/winniel123/verge-asm/issues/30)'s *"Debian's commented-out loopback line"* — the artefact this whole question was built on | **[measured]** In `rpcbind_1.2.7-1.debian.tar.xz`: `debian/rpcbind.default` carries `# OPTIONS="${OPTIONS} -h 127.0.0.1 -h ::1"` **commented**, under *"Uncomment the following line to restrict rpcbind to localhost only for UDP requests"*, while `debian/rpcbind.socket` ships `ListenStream=0.0.0.0:111` and `ListenDatagram=0.0.0.0:111` **active**. The restriction was **offered to the operator and not taken**, so there is no restricting default to admit under *either* reading — before reaching §10.1 Step 1 and §10.4.3, each of which excludes the row independently |
| **5601/tcp Kibana** | its secure default, which §4.6 already counts | `#server.host: "localhost"` with *"The default is 'localhost'"* — a documented default, already weighed and already insufficient. The row is out because Elastic states no prohibition, internet fronting behind auth is supported, and 5601 squats on `esmagent` |
| **9092/tcp Kafka** | `config/server.properties` | **[measured]** `#listeners=PLAINTEXT://:9092`, commented, and the effective default binds every interface — **permissive, therefore silent** under §10.4 in either reading |
| **5672, 15672/tcp RabbitMQ** | `rabbitmq.conf.example` | **[measured]** the file declares itself *"AN EXAMPLE … NOT MEANT TO BE USED IN PRODUCTION"* and every directive in it is commented, `listeners.tcp.local = 127.0.0.1:5672` included. Under the admitting reading there is still nothing active to admit; under §12 the file attests nothing at all |
| **8500/tcp Consul** | — | Consul ships no configuration file, so there is no artefact on either reading |
| **1099/tcp Java RMI** | `management.properties` | The row is out because Claim 1's **fact** is absent, not because a default excluded it — §10.4 makes no default capable of excluding. Weakening the secure-default evidence supplies no evidence that the shipped configuration *admits anonymous commands*, so the row stays out whichever way §12 had gone |

**Nothing re-opens.**

**`161/udp` does not re-open, and the reason is independent of everything §12 decides.** Under §12:
upstream's `EXAMPLE.conf.def` is an example and attests nothing (limbs (a) and (b) — its
`# Listen for connections from the local system only` is a label, and its `agentAddress` line is a
directive); net-snmp's actual documented default in `snmpd(8)` is all IPv4 interfaces, which is
permissive and therefore silent under §10.4; and Debian's installed copy is a **distributor's**
operative default, which attests about Debian's package (limb (c)). But **§11.6 did not rest on any of
that.** It removed the row because §10.2 closed the claim set and §10.3's boundary limb had no owner
attestation of any kind after retrieval — and **§10.3's owner requirement is untouched by §12**. Had
§12 gone the other way and admitted the example outright, the row would still fail §10.3, exactly as
§11.4 said. §12 replaces §11.4's grounds 1 and 2 with a rule and leaves its ground 3 as colour.

### 12.7 The one thing that moves — Cassandra's footing, and every number checked

**[measured]** Cassandra's shipped `conf/cassandra.yaml` — read from
`apache-cassandra-5.0.2-bin.tar.gz`, extracted locally — carries this sentence **twice**, at line 996
immediately above `native_transport_port: 9042` and again at line 1060 immediately above
`rpc_address: localhost`:

```
# For security reasons, you should not expose this port to the internet.  Firewall it if needed.
native_transport_port: 9042
```

```
# For security reasons, you should not expose this port to the internet.  Firewall it if needed.
rpc_address: localhost
```

— `conf/cassandra.yaml`, Apache Cassandra 5.0.2 (the `cassandra-5.0.2` tag agrees byte-for-byte on
both lines)

That is a **prohibition**, first-party, naming the listed port by number, in bytes the project ships.
It is limb (b)'s prose limb, not limb (a)'s directive limb, and it is exactly the kind of sentence
§2.2's footing table asserts does not exist for this row.

**§2.2 is wrong in two places and both are withdrawn rather than rewritten:** its footing table puts
`9042 Cassandra` in the *shipped default only — no prohibition exists upstream* row, and its prose
says *"Cassandra's strongest upstream sentence is an attack-surface observation rather than a
prohibition"*. Both were derived from the project's web documentation, which is where the note looked;
neither survives the shipped bytes.

| Where | Was | Is |
|---|---|---|
| §2.2 footing table, weak row | 5432 PostgreSQL, 5984 CouchDB, **9042 Cassandra** | **5432 PostgreSQL, 5984 CouchDB.** `9042` moves up to the **explicit prohibition** row |
| §2.2 prose | *"Cassandra's strongest upstream sentence is an attack-surface observation rather than a prohibition"* | **Withdrawn** — measured false against `conf/cassandra.yaml` |
| §10.4.2 | *"5432 PostgreSQL, 5984 CouchDB and 9042 Cassandra all rest on restricting defaults"* | **Stands as to the defaults** — all three verified active or documented — but 9042 no longer rests on its default **alone** |
| §1 pair count | 37 | **37, unchanged** |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 = 37 | **unchanged.** 9042 stays Class A: a footing is evidence for a claim, not a claim, and the row's claim is unchanged |
| §6.1 containment arithmetic | 28 in the hot set + 4 + 5 = 37 | **unchanged** |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | `verge-core = frequency-set ∪ sensitive-list` | **unchanged** — no member enters or leaves. 9042/tcp was already in the union and still is |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version, and the `Break` | — | **not triggered.** No `(port, transport)` pair moves, so `sensitive-port-reached-from-internet`'s content is byte-identical and no evaluation is made non-comparable. §12 is free in the strong sense, not merely the vacuous-before-v1 sense §11.6 relied on |
| §4.5 *the list's weakest row* | 5432/tcp | **unchanged.** 5432's footing is untouched, and with 9042 promoted it is if anything more clearly the weakest |
| §2.6 | boast restored to all 37 by §11.7 | **unchanged, and marginally stronger** — one more row now has an owner prohibition behind it rather than a default alone |

**What this costs and what it buys.** It costs nothing — no row, no version, no aperture. It buys one
thing worth having: §2.2's disclosure of its own weak tier is the note's most-quoted honesty
mechanism, and it was overstating the weakness by a third.

### 12.8 Thin ground, flagged per the standing rule

**The ruling is chosen, not derived, and the ticket exists because the text underdetermines it.**
§2.2's third form, §10.4's one-way rule and §10.5's distributor rule each speak to one hand and none
says which reading governs. Limb (a) is the best-grounded — it is a reading of the words *as documented
by the project* that were already there, and it decides the upstream half on the text. Limb (c) is
composed from §10.1 and §10.3 rather than stated anywhere. **Limb (b)'s directive-versus-label line is
the thinnest**: *"# Listen for connections from the local system only"* is clearly a label and
*"you should not expose this port to the internet"* is clearly a position, but a comment that argues
for its own directive sits between them, and §12 offers only §2.3's and §4.4's existing
position-versus-preference discrimination to place it. A case in that gap should be ticketed, not
decided by whoever meets it.

**The nine-artefact corpus is small and chosen by us.** The self-declaration finding is 9 for 9 and
that is what makes limb (a) mechanical, but a file that is neither installed nor self-disclaiming
would fall through to limb (a)'s second sentence — does the project document a default elsewhere? — and
if that is also silent the artefact is simply silent. That is the safe direction: the third form is an
**admission route only**, so an unresolvable file admits nothing and no row can enter on one.

**Cassandra's prohibition is measured on 5.0.2 alone.** The rows in this note are version-agnostic and
this measurement is not. What is established is that the sentence is in the current shipped bytes,
which is precisely what §2.2's footing table asserts the absence of.

> **Discharged by §13** ([#70](https://github.com/winniel123/verge-asm/issues/70)). The sentence is
> confirmed at `cassandra-5.0.2`, `cassandra-5.0.6` and `cassandra-5.0.9`. §13 also found it appears
> **four times rather than twice** in each of them — the two occurrences §12.7 did not record name
> `7000` and `7001`, neither of which is on the list ([#75](https://github.com/winniel123/verge-asm/issues/75),
> §13.5). That is not a defect in this section's ruling; it is a defect in the **extent** of its
> retrieval, and it is the measurement behind
> [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md).

### 12.9 Retrieval method and hazards, recorded per §9.5

**Every default quoted in §12 was read from shipped bytes, never from a rendered documentation page.**
Debian packaging came from `deb.debian.org`'s pool as `rpcbind_1.2.7-1.debian.tar.xz` and
`memcached_1.6.45-1.debian.tar.xz`, extracted locally, per §11.9 — `sources.debian.org` and
`salsa.debian.org` were not used and still serve a proof-of-work JavaScript challenge to a plain
client. Cassandra came from `apache-cassandra-5.0.2-bin.tar.gz` off `archive.apache.org`. The
remaining upstream files were fetched as raw bytes at release tags: `postgres/postgres`
`REL_17_STABLE`, `apache/couchdb` `3.4.2`, `apache/cassandra` `cassandra-5.0.2`, `redis/redis` `7.4`,
`apache/zookeeper` `release-3.9.3`, `apache/kafka` `3.8.0`, `elastic/kibana` `v8.15.0`,
`rabbitmq/rabbitmq-server` `v3.13.7`, `net-snmp/net-snmp` `master`.

- **A 404 on a guessed path looks exactly like a project shipping no default.** CouchDB's configuration
  lives at `rel/overlay/etc/default.ini`; the plausible guess `default.ini.tpl` (the name the file
  carried in earlier releases) returns 404 at every tag. A session that read that 404 as absence would
  have concluded CouchDB has no shipped default and moved the weakest row on the list. The path was
  resolved from the repository tree listing rather than guessed a second time.
- **The measurement that changed a cell was in a file no earlier pass had opened.** §2.2's footing
  table was built from the projects' web documentation. Cassandra's prohibition is in the shipped
  configuration file, and it took reading the bytes to find it. This is the general hazard rather than
  a fact about Cassandra, and it is routed to [#70](https://github.com/winniel123/verge-asm/issues/70)
  rather than fixed by inspection here: the footing table is a published claim about **nine other
  rows** whose shipped configuration bytes have never been read. **Discharged by §13** — they have now
  been read, every cell held, and the hazard recurred one level in: the two ports in §13.5 were in a
  file this ticket *had* opened.
- **PostgreSQL was checked three ways because its footing is the note's weakest row** — the sample
  file, the sample file's own statement about what a commented line means, and the compiled-in default
  in `guc_tables.c`. All three agree on `localhost`.

## 13. The footing table re-derived from shipped configuration bytes

Wayfinder ticket [#70](https://github.com/winniel123/verge-asm/issues/70), on the defect §12.9 recorded
and routed rather than fixed: §2.2's footing table was built from the projects' **web documentation**,
one file's bytes falsified one of its cells, and the other rows were assessed the same way. This
section **amends §2.2, §12.2 and §12.8 by reference**; earlier text stands and is marked, per the
name-and-withdraw convention, and where §13 and an earlier section disagree, **§13 governs**.

**Headline result, stated first.**

> **The footing table is right, and it is now right for a better reason.** Every row in it was
> re-derived from the owner's shipped configuration artefact, retrieved as bytes and with its path
> resolved from the repository tree. **No footing moves. No `(port, transport)` pair moves. The list
> stays at 37 pairs and the weak tier stays at two rows — 5432 PostgreSQL and 5984 CouchDB.**
>
> **Claim 3's failure condition was tested and is not met anywhere.** §10.3 fails a Class C row where
> *"the owner names the public internet as a supported deployment environment"*, and §12 makes a
> configuration file's prose an owner statement, so a config file is a place that sentence could live.
> **[measured]** Across fourteen artefacts from twelve projects, no shipped configuration file carries
> such a sentence. Three near-misses were examined at length and all three were refused on the rule
> rather than on taste (§13.3).
>
> **Two things fall out that a row-scoped read could not have found**, and both are routed rather than
> decided. Cassandra's prohibition names **`7000` and `7001` as well as `9042`** — two ports that are
> not on the list ([#75](https://github.com/winniel123/verge-asm/issues/75), §13.5). And the footing
> table places **19 of the 37 pairs**, with seven listed rows in its own subject matter having no cell
> at all ([#76](https://github.com/winniel123/verge-asm/issues/76), §13.7). The instrument behind both
> is [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md): **an
> attestation is retrieved over the artefact, not over the row.**

### 13.1 What was retrieved

Fourteen artefacts, from twelve projects, plus the four §12 had already read. Every path was resolved
from the project's own repository tree or release tarball listing, never guessed — §12.9's CouchDB
`default.ini.tpl` hazard is why, and it was met again in a smaller form (§13.10).

| Project | Artefact, at the tag or release named | Kind, per §12.2 |
|---|---|---|
| Redis | `redis.conf` — `redis/redis` `8.0` (§12 read `7.4`) | Operative |
| Apache Cassandra | `conf/cassandra.yaml` — `apache/cassandra` `cassandra-5.0.2`, `cassandra-5.0.6`, `cassandra-5.0.9` | Operative |
| rsync | `rsyncd.conf.5.md`, `packaging/systemd/rsync.service`, `rsync@.service`, `rsync.socket`, `packaging/lsb/rsync.xinetd`, `README.md` — `RsyncProject/rsync` `v3.5.0` | Operative (units); owner's prose (man page source) |
| memcached | `scripts/memcached.sysconfig` — `memcached/memcached` `1.6.45` | Operative (upstream's own RPM `Source1`) |
| MySQL | `packaging/deb-in/extra/mysqld.cnf`, `my.cnf.fallback`, `mysql.cnf`, `packaging/rpm-common/my.cnf.in` — `mysql/mysql-server` `mysql-9.7.2` | Operative (Oracle's own packaging) |
| Elasticsearch | `distribution/src/config/elasticsearch.yml` **and** `distribution/docker/src/docker/config/elasticsearch.yml` — `elastic/elasticsearch` `v9.5.1` | Operative (both) |
| MongoDB | `rpm/mongod.conf` **and** `debian/mongod.conf` — `mongodb/mongo` `r8.3.8` | Operative (MongoDB Inc's own packaging) |
| nfs-utils | `nfs.conf`, `utils/mount/nfs.man`, `utils/exportfs/exports.man` — `nfs-utils-2.9.2.tar.xz` from `kernel.org`, extracted locally | Operative (`nfs.conf`); owner's prose (man pages) |
| ZooKeeper | `conf/` in full — `apache/zookeeper` `release-3.9.5` (§12 read `release-3.9.3`) | Example (`zoo_sample.cfg`); no operative file exists |
| RabbitMQ | `deps/rabbit/docs/rabbitmq.conf.example` — `rabbitmq/rabbitmq-server` `v4.3.4` (§12 read `v3.13.7`) | Example, self-declared |
| Docker | `contrib/init/systemd/docker.service` + `docker.socket`, `contrib/init/sysvinit-debian/docker.default`, `contrib/init/openrc/docker.confd` — `moby/moby` `docker-v29.7.2` | Operative |
| PostgreSQL | `src/backend/utils/misc/postgresql.conf.sample` **and `src/backend/libpq/pg_hba.conf.sample`** — `postgres/postgres` `REL_18_STABLE` (§12 read `REL_17_STABLE`, and only the first file) | Record of defaults |
| CouchDB | `rel/overlay/etc/default.ini` — `apache/couchdb` `3.5.0` (§12 read `3.4.2`) | Operative |
| etcd | `etcd.conf.yml.sample`, `contrib/systemd/etcd.service` — `etcd-io/etcd` `v3.7.1` | Neither, and §13.6 is about that |

**Three rows have no artefact to read, and that is a finding rather than a gap.** `1433/tcp` MS SQL
Server, `445/tcp` SMB with `139/tcp` and `137`, `138/udp`, and `623/udp` IPMI. Microsoft configures
SMB and SQL Server through setup and the registry rather than through a file it ships, and a BMC's
configuration is firmware. **There is nothing to open, so nothing in these rows can move on this
ticket** — recorded per §11.8's rule that a negative retrieval is a verdict, and per
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 3 that
it is a verdict about what was read.

### 13.2 The re-derivation, row by row

| Row and current tier | What the shipped bytes carry | Verdict |
|---|---|---|
| **6379/tcp Redis** — prohibition | *"binding to all the interfaces is dangerous and will expose the instance to everybody on the internet"*, with `bind 127.0.0.1 -::1` and `protected-mode yes` **active** | **Stands.** A §12(b) position, in the bytes as well as on the web page |
| **9042/tcp Cassandra** — prohibition (promoted by §12.7) | the prohibition, verbatim, above `native_transport_port: 9042` and `rpc_address: localhost` | **Stands**, and now confirmed at three releases rather than one — §12.8's version flag discharged |
| **873/tcp rsync** — prohibition | *"Do not expose a cleartext daemon to an untrusted network: front it with a TLS proxy … or run it over ssh"* in shipped `rsyncd.conf.5.md`; and the same file's **SSL/TLS Daemon Setup** puts the public listener on **874** with *"server local-rsync 127.0.0.1:873"* behind it, adding *"You should limit the access to the backend-rsyncd port to only allow the proxy to connect. If it is on the same host as the proxy, then configuring it to only listen on localhost is a good idea."* | **Stands, and strengthens.** The owner does not merely prohibit exposure of 873 — it publishes the supported alternative and puts 873 on loopback inside it |
| **11211/tcp+udp memcached** — prohibition | upstream ships **no** `memcached.conf`; the only file of that name in the tree is `t/sasl/memcached.conf`, a test fixture. Its own RPM's `Source1` is `scripts/memcached.sysconfig`, which carries `OPTIONS=""` and no `-l` | **Stands, unchanged.** Permissive, therefore **silent** under §10.4; the row rests on the wiki sentence, as §3.4 says. Debian's `memcached.conf` corroborates and carries nothing (§12(c), §12.5) |
| **3306/tcp MySQL** — prohibition | four Oracle-authored packaging files, none carrying a `bind-address` and none carrying a sentence about networks | **Stands, unchanged.** Permissive, therefore silent; the row rests on the Security Guidelines |
| **9200/tcp Elasticsearch** — prohibition | the tarball config records the default — *"By default Elasticsearch is only accessible on localhost"* — with `#network.host` commented. **[measured]** Elastic's **official Docker image** config ships `network.host: 0.0.0.0` **active** | **Stands, unchanged**, and see §13.3: the same owner ships two operative defaults pointing opposite ways, and §10.4's one-way rule disposes of it without adjudication |
| **1433 MS SQL · 445 SMB · 623 IPMI** — prohibition | no artefact exists (§13.1) | **Stand, untouched.** Nothing to read |
| **27017/27018/27019 MongoDB** — scoping | `bindIp: 127.0.0.1` **active** in both `rpm/mongod.conf` and `debian/mongod.conf`, which are **MongoDB Inc's own** packaging and therefore an owner's shipped default, not a distributor's. The rpm copy's trailing *"# Enter 0.0.0.0,:: to bind to all IPv4 and IPv6 addresses or, alternatively, use the net.bindIpAll setting."* is a **label** under §12(b) | **Stands.** The tier is unchanged; what changes is that the *default* limb is now read off the owner's bytes rather than off a documentation page |
| **2049/tcp NFS** — scoping | `nfs.conf` ships every setting commented, `# host=` included → permissive → silent. The scoping sentence §3.4 quotes is in the shipped source verbatim (`utils/mount/nfs.man`), **and** the same page's DESCRIPTION carries one nobody had quoted: *"NFS was developed to allow file sharing between systems residing on a local area network."* | **Stands, and strengthens inside its tier** — that is a locality claim in the owner's own words, which is exactly what §10.3's boundary limb asks for |
| **2181/tcp ZooKeeper** — scoping | `conf/` holds `zoo_sample.cfg`, `configuration.xsl` and `logback.xml` and nothing else. The sample carries `clientPort=2181` and **no address or bind directive of any kind** | **Stands, unchanged.** #69's finding confirmed at a later release; the row rests on the Administrator's Guide |
| **25672/tcp RabbitMQ** — scoping | `rabbitmq.conf.example` still self-declares *"This file is AN EXAMPLE. It is NOT MEANT TO BE USED IN PRODUCTION."*, and `distribution.listener.port_range.min = 25672` is **commented** | **Stands, unchanged.** The file attests nothing (§12(a), §12(d)); the row rests on the networking guide |
| **2376/tcp Docker** — scoping | `docker.socket` ships `ListenStream=/run/docker.sock` and `docker.service` `ExecStart=… -H fd://`. **The operative default has no TCP listener at all**; `docker.default` and `docker.confd` offer `DOCKER_OPTS` empty or commented | **Stands, unchanged.** A restricting default, admissible under §10.4 and corroborating; no prose position anywhere in the bytes |
| **5432/tcp PostgreSQL** — **shipped default only** | `postgresql.conf.sample` is as §12.5 found it. **`pg_hba.conf.sample` had never been opened**, and it is a **second, independent** restricting default: the only `host` records shipped are `127.0.0.1/32` and `::1/128`, under *"If you want to allow non-local connections, you need to add more \"host\" records."* Neither file contains a sentence about untrusted networks, exposure, or the internet | **Stays in the weak tier**, and stays §4.5's weakest row. Two restricting defaults are still two defaults |
| **5984/tcp CouchDB** — **shipped default only** | `bind_address = 127.0.0.1` **active** under `[chttpd]`, labelled *"These settings affect the main, clustered port (5984 by default)"*, and again under `[httpd]` and `[prometheus]`. No prose position anywhere in the file | **Stays in the weak tier.** The label names the listed port, which is worth having and is not a position |

**No cell moves.** Two rows are stronger inside their tier (`873`, `2049`), three have their footing
re-founded on the owner's bytes rather than a rendered page (`27017`/`27018`/`27019`, `5432`, `5984`),
and one has its version-specificity flag discharged (`9042`).

### 13.3 Claim 3's failure condition, tested — three near-misses, all refused

§10.3 is the only route by which this ticket could have moved a **row** rather than a footing: *"Where
the owner names the public internet as a supported deployment environment, Claim 3 fails however
strongly a third party disapproves."* §12(b) makes config-file prose an owner statement, so a shipped
config is somewhere that sentence could live. It was searched for and it is not there. Three artefacts
came close enough to argue about.

**1. rsync's systemd units quote rsync's own README, and the quoted sentence says *public*.**
**[measured]** `packaging/systemd/rsync.service` and `rsync@.service` both carry:

```
# Citing README.md:
#
#   [...] Using ssh is recommended for its security features.
#
#   Alternatively, rsync can run in `daemon' mode, listening on a socket.
#   This is generally used for public file distribution, [...]
```

That is upstream's prose about upstream's daemon, in bytes upstream ships, and it names a public
deployment. **It is refused, on the same document that carries the prohibition.** The `rsyncd.conf.5`
SSL/TLS Daemon Setup section shows the supported public deployment in full, and in it the public
listener is on **874** and `873` is a loopback backend the owner says *"should limit … to only allow
the proxy to connect"*. So the owner names a supported public deployment **of the service** and
explicitly does not name one **of `873` cleartext**, which is the listed pair. §10.3's failure
condition is about the row, not about the product.

**Two riders, both #46's shape.** The unit's own `[...]` **truncates** the README, which continues
*"although authentication and access control are available"* — the truncated-conditional hazard §9.5
recorded, met here in a first-party file. And the comment's stated purpose is the opposite of an
endorsement: it cites the sentence to justify `ProtectSystem=full`, `PrivateDevices=on` and
`NoNewPrivileges=on`, i.e. *"let's assume some extra security is more than welcome here"*. A session
that quotes the fragment without the two lines under it has read it backwards.

**2. nfs-utils' `exports(5)` describes an example export as reaching *every host in the world*.**
**[measured]** Against the sample line `/pub  *(ro,insecure,all_squash)`: *"Line 5 exports the public
FTP directory to every host in the world, executing all requests under the nobody account."* **Refused
on §12(b): that is a label**, and the clearest instance of one in the corpus — it describes what the
directive above it does, sentence by sentence, for eight numbered lines. It is also about the **export
access list** rather than about a network, and it is in a man page's worked example rather than in a
shipped configuration. The same project's `nfs(5)` points the other way in its own DESCRIPTION
(§13.2).

**3. Elastic's official Docker image binds every interface.** **[measured]**
`distribution/docker/src/docker/config/elasticsearch.yml` is two lines, and one of them is
`network.host: 0.0.0.0` — **active**, in an operative shipped default, from the owner. Two facts make
it silent rather than damaging. It is a **directive**, and §12(b) says a directive attests only through
the third form. And through the third form it is **permissive**, which §10.4 makes silent in both
directions. **The case is worth recording because it is the cleanest demonstration that §10.4's
one-way rule does real work**: one owner ships two operative defaults that contradict each other on
the same setting — `localhost` in the tarball, `0.0.0.0` in the image — and the rule resolves it with
no adjudication at all, because the restricting one attests and the permissive one is silent. The
alternative reading, in which a bind address is a position, would have the owner of `9200` naming the
world as its supported deployment and would take the row off the list.

**Nothing else came within reach.** A sweep of all fourteen artefacts for `internet`, `untrusted`,
`expose`, `public network`, `firewall` and `insecure` returns four hits in total: Elasticsearch's
*"address here to expose this node on the network"* (a label), `docker.service`'s
`After=… firewalld.service` (an ordering dependency), and two CouchDB comments about an unrelated
option and about replication logging.

### 13.4 The ruling

> **§2.2's footing table is confirmed as it stands after §11.6 and §12.7. No row moves and no footing
> moves.** The prohibition tier is `6379`, `11211`, `3306`, `1433`, `9200`, `873`, `445`, `623` and
> `9042`; the trusted-network-scoping tier is `27017`/`27018`/`27019`, `2049`, `2181`, `25672` and
> `2376`; **the weak tier is `5432` and `5984`, and it is two rows.**
>
> **What changes is the table's warrant rather than its content.** It was a claim derived from web
> documentation about artefacts nobody had opened; it is now a claim derived from the artefacts. §12.9
> called that *"a published claim … known to have been built the wrong way"*. It has now been built the
> right way, and it came out the same.

**The result is deflating and that is most of its value.** #69 opened one file and found one cell
wrong, which is the strongest possible reason to expect more. The measured answer is that the error
rate was one cell in a table of nineteen pairs, and that the surviving eighteen are correct — so
§2.2's disclosure can be quoted as measured rather than as derived, which is what §4.5 and the map's
curation patch both lean on when they call the weak tier a watch list.

### 13.5 By-catch: the prohibition names two more ports, and neither is on the list

**[measured]** `conf/cassandra.yaml` carries *"For security reasons, you should not expose this port to
the internet. Firewall it if needed."* **four times**, not twice, and identically at `cassandra-5.0.2`
— the tag §12.7 read — `cassandra-5.0.6` and `cassandra-5.0.9`:

```
# For security reasons, you should not expose this port to the internet.  Firewall it if needed.
storage_port: 7000
```

```
# For security reasons, you should not expose this port to the internet. Firewall it if needed.
ssl_storage_port: 7001
```

`7000/tcp` is Cassandra's inter-node storage and gossip port and `7001/tcp` its TLS twin — Claim 3's
paradigm case, an inter-node transport with no internet client under any topology — carrying the same
first-party sentence, naming each port by number, in the same shipped file.

**They are not admitted here, on two grounds.** A footing ticket may not add a row: an addition bumps
the rule version, `Break`s every evaluation and widens `verge-core`, which is the pricing §11.6 paid to
*remove* `161/udp`, and it belongs to a ticket scoped to pay it. And **§2.4's determinacy gate is live
and may sink both**: IANA registers `7000/tcp` to `afs3-fileserver` and `7001/tcp` to `afs3-callback`,
so Cassandra squats on each, and §4.3 excludes high ports that are conventionally anything. The
strongest sentence in the corpus does not clear a gate it does not speak to. Routed to
[#75](https://github.com/winniel123/verge-asm/issues/75), which **blocks
[#12](https://github.com/winniel123/verge-asm/issues/12)**, because unlike #70 it is a candidate row
rather than a footing.

> **Discharged by §14** ([#75](https://github.com/winniel123/verge-asm/issues/75)). **Neither port is
> admitted, and this paragraph's guess about which gate would decide it was right.** Determinacy sank
> both: `7000/tcp` because Apple documents `AirPlay · 7000 · TCP` as a live competing service beside
> the `afs3-fileserver` squat, and `7001/tcp` on that limb **and** on version-dependence — Cassandra
> deprecated `ssl_storage_port` at 4.0 and ships `legacy_ssl_storage_port_enabled: false`. **The list
> stays at 37 pairs**, no rule version bumps and `verge-core` does not move, so the addition pricing
> this paragraph reserved was never spent. `7000`'s claim and attestation both **passed**, which
> makes it the note's first row refused on determinacy alone (§14.2). The sentence quoted above is
> the one the ruling turned on.

**The instrument is the finding.** #69 opened the right file, at the right tag, and read the two lines
it had gone looking for. It recorded the sentence as appearing twice; it appears four times, and the
two it did not record are the interesting ones. A retrieval scoped to the rows a table already holds
can confirm those rows and can discover nothing. That is
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md).

### 13.6 A tenth artefact, and the first that does not declare itself

§12.2's *"nine artefacts, nine self-declarations"* is what makes §12 limb (a) mechanical rather than a
judgement. A tenth was retrieved here and it is the first near-miss.

**[measured]** `etcd.conf.yml.sample`, `etcd-io/etcd` `v3.7.1`, opens:

```
# This is the configuration file for the etcd server.
```

That is an **operative** self-description, on a file whose name ends `.sample` and which nothing
installs — `contrib/systemd/etcd.service` runs `/usr/bin/etcd` with no `--config-file`. Under a
syntax-reading it would be an example; under its own first sentence it would be the configuration.

**§12 handles it without strain, through the limb it was built with.** Limb (a)'s second sentence —
*"where the party documents a default elsewhere and the two disagree, the documented default
governs"* — resolves it, because etcd documents `--listen-client-urls` and `--listen-peer-urls` and
their defaults independently of this file, and the file agrees with them (`http://localhost:2379`,
`http://localhost:2380`). **So the outcome is the same whichever way the file is read**, which is why
this is recorded rather than ticketed.

**What it costs is the strength of the 9-for-9 count, and §12.8 predicted exactly this**: *"a file that
is neither installed nor self-disclaiming would fall through to limb (a)'s second sentence."* It did,
and the fallback worked. The count is now **ten artefacts, nine self-declarations and one that
describes itself as the thing it is not** — still enough for limb (a) to be a test read off the
document, and no longer quotable as a law.

### 13.7 The table's coverage gap — seven listed rows have no footing cell

Counting the table's cells against the list, which nobody had done: **§2.2's footing table places 19 of
the 37 `(port, transport)` pairs**, after §11.6 struck `161 SNMP` and §12.7 promoted `9042`.

Eleven of the eighteen uncovered pairs are outside the table's subject and are correctly absent. The
table discriminates §2.2's **second** and **third** forms, documentation against shipped default, and
says so in its own lead-in. Class B's seven rows (`23`, `21`, `512`, `513`, `514`, `5900`, `6000`) and
`69/udp` rest on the **first** form — a specification, IANA's registry, or OpenBSD's deletions — and
`139/tcp`, `137/udp` and `138/udp` are carried by the same Microsoft sentence as `445` and sit inside
that row's ambit.

**Seven do not have that excuse**, and each rests on the owner's documentation or on a shipped default:
`2375/tcp` Docker, `4369/tcp` epmd, `9300/tcp` Elasticsearch, `10250/tcp` and `10255/tcp` kubelet, and
`2379/tcp` and `2380/tcp` etcd. Two are clerical — `2375` shares Docker's sentences with `2376`, and
RabbitMQ's *"these ports should not be publicly exposed"* names `4369` alongside `25672` — and the
other five need a ruling this ticket is not scoped to make.

**The etcd pair is why this is filed rather than noted.** **[measured]** `etcd.conf.yml.sample` ships
`listen-client-urls: http://localhost:2379` and `listen-peer-urls: http://localhost:2380`, a
**restricting** default. etcd's own sentence in §3.4 is a **consequence** (*"can expose its data to any
clients"*) rather than a position, and the *"ideally only the API server should have access to it"*
sentence is **kubernetes.io** — a different party speaking about its own use of etcd, which §10.5 makes
corroboration. If etcd's own prose states no position, `2379` and `2380` rest on **a shipped default
and nothing else**, and **the weak tier goes from two rows to four**. That is not a row move — a
footing is not a claim — but the map's curation patch reads the weak tier as the **curator's watch
list** for ADR-0032 §8's silent de-attestation, and a watch list missing half its members is worse than
one that is honestly short. Routed to [#76](https://github.com/winniel123/verge-asm/issues/76), which
does **not** block [#12](https://github.com/winniel123/verge-asm/issues/12).

> **Discharged by §16** ([#76](https://github.com/winniel123/verge-asm/issues/76)). **All seven are
> placed, the table now states its own coverage — 26 of 37, with the other 11 named as out of subject
> — and this paragraph's prediction is refuted.** **[measured]** etcd ships `THREAT_MODEL.md`
> (`etcd-io/etcd` `v3.7.1`) saying *"It **must not** be exposed to untrusted networks or the public
> internet"* and naming **Port 2379** and **Port 2380** by number, so both go to the **explicit
> prohibition** tier rather than the weak tier. The paragraph was right that etcd's website states no
> position and that kubernetes.io is corroboration; it was wrong that the project states none, and the
> document is at the repository root rather than where §3.4's citation points — ADR-0037 again.
> **The weak tier does grow, to three rather than four, and on a row this paragraph did not name:**
> `10255/tcp` kubelet, whose owner states no network position and whose shipped config API ships
> `readOnlyPort` *"Default: 0 (disabled)"* (§16.5). `9300` joins the prohibition tier and `2375`,
> `4369` and `10250` the scoping tier. **No `(port, transport)` pair moves**, so #12 stays unblocked by
> this line — but §16.5's by-catch, that §3.4's kubelet defaults are quoted from a generated page the
> shipped source contradicts, is routed to [#83](https://github.com/winniel123/verge-asm/issues/83),
> which **does** block #12.

### 13.8 Every dependent figure, checked rather than asserted

| Where | Was | Is |
|---|---|---|
| §1 pair count | 37 | **37, unchanged** |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 = 37 | **unchanged.** No row changes class, because no row's claim changes |
| §2.2 footing table | prohibition 9 labels · scoping 5 · weak 2 | **unchanged in every cell**, and now derived from the artefacts rather than from web pages |
| §2.6's boast | true of all 37 | **unchanged**, and marginally better founded: the rows it covers were re-checked against the owners' own bytes |
| §4.5 *the list's weakest row* | 5432/tcp | **unchanged.** `pg_hba.conf.sample` adds a second restricting default and no sentence, which is the same footing twice |
| §6.1 containment arithmetic | 28 in the hot set + 4 + 5 = 37 | **unchanged** |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | `verge-core = frequency-set ∪ sensitive-list` | **unchanged** — no member enters or leaves |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version, and the `Break` | — | **not triggered.** `sensitive-port-reached-from-internet`'s content is byte-identical, so no evaluation is made non-comparable. Free in the strong sense |
| §12.7's *"one footing changes"* | `9042` promoted | **unchanged and now confirmed at three releases** — §12.8's *"measured on 5.0.2 alone"* is discharged |
| §12.2's nine-for-nine | nine artefacts, nine self-declarations | **ten artefacts, nine self-declarations** — §13.6 |
| §12.9's *"a published claim … built the wrong way"* | open, routed to #70 | **discharged.** The table was rebuilt from the artefacts and came out the same |

**Outside this note.** Nothing. ADR-0009, ADR-0008 and ADR-0032 are untouched;
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) is added and
is a method rather than a table edit.

### 13.9 Thin ground, flagged per the standing rule

**The negative result is the thin part, and it is thin in a stated way.** The claim *no shipped
configuration file names the public internet as a supported deployment environment* is established
over **fourteen artefacts from twelve projects**, chosen because §2.2's table names their rows. It is
not established over every file every one of those projects ships, and by
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 3 it may
be relied on only inside that extent. Each artefact was read end to end, which is the whole of the
improvement over §12's pass.

**Three rows were verified by finding nothing to verify.** `1433`, `445` and `623` have no shipped
configuration artefact, so this ticket's instrument cannot reach them at all. Their footings are
exactly as strong as they were before — carried by Microsoft's and Dell's prose — and this pass adds
no evidence in either direction. A reader who takes §13 as *the whole table has now been measured
against bytes* has over-read it by three rows.

**The rsync refusal in §13.3 is the thinnest single judgement.** It turns on the listed pair being
`873` cleartext rather than the rsync daemon in general, and on the owner's own SSL/TLS section putting
the public listener on `874`. That is a good argument and it is not an airtight one: a reader could say
that a project whose README calls daemon mode *"generally used for public file distribution"* has named
a public deployment and that §10.3 does not ask which port. **The criterion that would reopen it:** an
rsync sentence endorsing a **cleartext** daemon on an untrusted network, or the removal of the
`rsyncd.conf.5` guidance that puts `873` behind a proxy on loopback. Either would put the row in
genuine jeopardy; neither is in the current bytes.

### 13.10 Retrieval method and hazards, recorded per §9.5

**Every artefact quoted in §13 was read as shipped bytes, never as a rendered page**, at a named tag or
release, with paths resolved from the repository tree or tarball listing. `sources.debian.org` and
`salsa.debian.org` were not used and still serve a proof-of-work JavaScript challenge to a plain
client (§11.9); no distributor packaging was needed, because every project in this pass ships its own.

- **`nfs-utils` has no upstream forge mirror that serves bytes, and the release tarball does.**
  `nfs-utils-2.9.2.tar.xz` was taken from `kernel.org/pub/linux/utils/nfs-utils/`, verified by
  extracting it locally, and `nfs.conf`, `nfs(5)` and `exports(5)` read from the extracted tree. Going
  to a distribution's rendering of `nfs(5)` would have reproduced §10.5's error class — §3.4 already
  cites `nfs(5)` to `man7.org`, which is a rendering, and the shipped source agrees with it word for
  word, which is worth having on the record.
- **The `.sample` suffix is not a self-declaration and one project proves it.** etcd's
  `etcd.conf.yml.sample` calls itself *"the configuration file for the etcd server"* (§13.6). Reading
  the suffix as the answer is the surface-syntax error §12.2 already refused for commented lines,
  arriving through the filename instead.
- **An owner can ship two operative defaults that contradict each other**, and Elastic does: `localhost`
  in the tarball config, `0.0.0.0` in the official Docker image config (§13.3). A session that opens one
  of the two and stops has measured a coin flip. Both paths were resolved from the same tree listing in
  the same query.
- **The hazard §12.9 named as general is now measured twice.** §12.9 recorded that the cell which moved
  was in a file no earlier pass had opened. This pass found the same shape one level in: **the two
  Cassandra ports in §13.5 were in a file an earlier pass *had* opened**, sixty lines from the quoted
  line. *Opened* is not *read*, and only the second is a retrieval.
- **The count in §13.7 was done by hand against §3's tables and is the kind of thing that goes wrong.**
  It is stated as 19 covered and 18 uncovered, summing to 37, with the eighteen enumerated by name so
  the arithmetic is checkable rather than assertable.

## 14. `7000/tcp` and `7001/tcp` are refused — the first rows where every gate but determinacy passes

Wayfinder ticket [#75](https://github.com/winniel123/verge-asm/issues/75), on the two candidate rows
§13.5 found and routed. This section **amends §1, §2.4, §4.2, §4.3, §4.6, §8 and §13.5 by
reference**; earlier text stands and is marked, per the name-and-withdraw convention, and where §14
and an earlier section disagree, **§14 governs**.

**Headline result, stated first.**

> **Neither `7000/tcp` nor `7001/tcp` is admitted. The list stays at 37 `(port, transport)` pairs,
> the class totals stay 12 / 7 / 18, and the weak tier stays at two rows.** Both fail §2.4's
> **determinacy** gate, on different limbs of it, and nothing else about either row is wrong.
>
> **`7000/tcp` passes every other gate on this list, and that is the finding.** Claim 1 passes both
> of §10.1's steps and Claim 3 passes with its §10.3 boundary limb **named by the owner**; §2.2's
> attestation is an explicit first-party prohibition naming the port by number in operative shipped
> bytes, confirmed at three release tags, with a restricting shipped default active beside it. It is
> better attested than `5432/tcp`, better attested than `9042/tcp` was before §12.7, and it is
> refused anyway. **This is the first row in the note refused with its claim and its attestation both
> established** — the cleanest available demonstration that §2.4 is an independent gate rather than a
> tiebreaker, which is what §13.5 predicted when it wrote that *"the strongest sentence in the corpus
> does not clear a gate it does not speak to."*
>
> **The rule that decides `7000` is new and is written down**, because the note has carried an
> unexplained tension since §21: `9200/tcp` squats on `wap-wsp` and is **on** the list, while
> `9100/tcp` squats on `hp-pdl-datastr` and is **off** it, and no section says what separates them.
> **A squat is contested where the other convention is live** —
> [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md).
>
> **It costs nothing.** No pair moves, so `sensitive-port-reached-from-internet`'s content is
> byte-identical, no rule version bumps under
> [ADR-0008](../adr/0008-derivation-versions-move-on-content.md), no evaluation is made
> non-comparable, and [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union is untouched. Free in
> the strong sense §12.7 and §13.8 used, not the vacuous-before-v1 sense §11.6 relied on. **The pair
> count [#12](https://github.com/winniel123/verge-asm/issues/12) assembles is 37 and is now
> definite.**

### 14.1 What was retrieved

Every artefact was fetched as bytes at a named tag or release, with paths resolved from the
repository tree rather than guessed, per §12.9, §13.10 and
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md). Each
configuration file was read end to end, per that ADR's limb 1.

| Retrieved | What it is | What it settled |
|---|---|---|
| `conf/cassandra.yaml` — `apache/cassandra` `cassandra-5.0.2`, `cassandra-5.0.6`, `cassandra-5.0.9` | the operative shipped default (§12.2) | The prohibition, §10.3's boundary limb, `listen_address`, and `ssl_storage_port`'s **deprecation** |
| `conf/cassandra_latest.yaml` — `cassandra-5.0.9` | the **second** operative default the file's own header announces | Agrees line for line — §13.10's Elastic hazard tested and not met (§14.6) |
| `src/java/org/apache/cassandra/config/Config.java`, `config/DatabaseDescriptor.java`, `auth/AuthConfig.java` — `cassandra-5.0.9` | the compiled defaults | `storage_port = 7000`, `ssl_storage_port = 7001`, and the internode authenticator fallback |
| `cassandra.apache.org` security page and 5.0 FAQ | the owner's prose | Claim 1's authority limb, and *"7000 for cluster communication (7001 if SSL is enabled)"* |
| IANA registry CSV, retrieved 2026-08-14, registry last updated 2026-08-11 | the registration | `afs3-fileserver` / `afs3-callback`, and a `Known Unauthorized Use` annotation on 7001 |
| `support.apple.com/en-us/103229`, *TCP and UDP ports used by Apple software products* | the competing vendor, about its own product | **`AirPlay · 7000 · TCP`** — the fact that decides `7000` |
| `oracle/docker-images` `OracleWebLogic/dockerfiles/14.1.2.0/README.md`; Oracle WLS administration guide | the competing vendor, about its own product | **`ADMIN_LISTEN_PORT` (default: `7001`)**; *"one port for HTTP communication (7001 by default)"* |
| [`safe-active-probing.md`](./safe-active-probing.md) §2.3 | **this project's own** frequency half | `7001` is already in it, classified as an **HTTP-ish alternate** (§14.4) |
| `nmap-services`, `nmap/nmap` `master` | rank only, per §2.5 and §6.1's existing limit | `7000/tcp` ranks **146th**, `7001/tcp` **232nd** — neither is in the top-100 |

### 14.2 The claim, and it is not the problem — `7000/tcp` passes on both Claim 1 and Claim 3

Recorded at length precisely because the row is refused. A reader who sees only the verdict will
assume the evidence was thin. It is not, and the note is worth more if the refusal is visible as a
gate rather than as a shortage.

**§2.2's attestation — the prose limb, §12(b).** **[measured]** `conf/cassandra.yaml` carries the
prohibition immediately above `storage_port: 7000`, identically at all three tags (lines 933-934 at
`cassandra-5.0.2`, 959-960 at `cassandra-5.0.6` and `cassandra-5.0.9`):

```
# TCP port, for commands and data
# For security reasons, you should not expose this port to the internet.  Firewall it if needed.
storage_port: 7000
```

That is a **position**, not a label and not a directive — the same sentence, in the same file, that
§12.7 used to move `9042/tcp` into the prohibition tier. Under §12(b) it attests exactly as a manual
page would.

**§2.2's attestation — the third form, §12(a).** `listen_address: localhost` is **active** in the
same file, in an artefact §12.2 already classified as **operative**. It is a *restricting* default,
so §10.4 admits it. The owner labels it in terms that name the audience: *"Address or interface to
bind to and tell other Cassandra nodes to connect to. You \_must\_ change this if you want multiple
nodes to be able to communicate!"* and, four lines down, *"Setting listen_address to 0.0.0.0 is
always wrong."*

**Claim 3, including §10.3's boundary limb — the limb that removed `161/udp`.** §11.6 removed SNMP
because no owner sentence placed it inside a boundary. Cassandra's owner places `7000` inside one
three times over: the yaml comment above, the FAQ's *"By default, Cassandra uses 7000 for cluster
communication (7001 if SSL is enabled), 9042 for native protocol clients, and 7199 for JMX"*, and the
security page's *"Cassandra is configured to easily find and be found by other members of a
cluster."* The named boundary is **the cluster**, which is §10.3's *"the same cluster"* verbatim.
There is no counterweight: **[measured]** neither `cassandra.yaml` nor `cassandra_latest.yaml`
contains the strings `internet` (outside the four prohibition sentences), `public`, or `untrusted`,
so §10.3's failure condition — the owner naming the public internet as a supported environment — is
not met, consistent with §13.3's finding across fourteen artefacts.

**Claim 1, checked as the ticket asked, and it also passes.** §10.1 Step 1 (publication): answering
strangers is not the purpose — the directive's own label is *"TCP port, for commands and data"*.
Step 2 (authority): the owner enumerates what an anonymous caller on this port gets, and every item
is on §10.1's list:

> "Enabling authentication for clients using the binary protocol is not sufficient to protect a
> cluster. Malicious users able to access internode communication and JMX ports can still: Craft
> internode messages to insert users into authentication schema · Craft internode messages to
> truncate or drop schema · Use tools such as sstableloader to overwrite system_auth tables · Attach
> to the cluster directly to capture write traffic"
> — [cassandra.apache.org security](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/security.html)

That the caller is anonymous is measured from bytes three ways rather than assumed. The shipped yaml
leaves the directive commented, naming the class it would otherwise set:

```
# Internode authentication backend, implementing IInternodeAuthenticator;
# used to allow/disallow connections from peer nodes.
#internode_authenticator:
#  class_name: org.apache.cassandra.auth.AllowAllInternodeAuthenticator
```

`server_encryption_options.internode_encryption: none` is **active** beside it. And the compiled
fallback agrees: `AuthConfig.java` reads
`authInstantiate(conf.internode_authenticator, IInternodeAuthenticator.class, AllowAllInternodeAuthenticator.class)`,
with `DatabaseDescriptor.java` initialising the field to `new AllowAllInternodeAuthenticator()`.

**So the row is over-determined on claim and on attestation, and it is refused anyway.** Note what
that costs the note in a good way: it removes the last reading under which §2.2's attestation tiers
could be mistaken for a ranking of *how likely a row is to survive*. A prohibition-tier attestation
is not a pass.

### 14.3 `7000/tcp` fails determinacy — the competing convention is live, and its own vendor documents it

§2.4's gate is on the **port**, not the service (§4.3), and the note has already ruled that
registration cannot be the test, because *"many of the best-known sensitive ports are squatted, not
registered"* and *"uncontested convention has to be"* the test instead. So `afs3-fileserver` alone
does not sink `7000` — `9200`, `9300`, `2181`, `9042`, `10250`, `10255` and `623/udp` are all on the
list over a squat.

**What sinks it is a second live service, documented by the vendor that ships it.** **[measured]**
Apple's *TCP and UDP ports used by Apple software products* tabulates:

> `AirPlay · 7000 · TCP`
> — [support.apple.com/en-us/103229](https://support.apple.com/en-us/103229)

That is the strongest class of determinacy evidence available: not a third party's guess about what
runs on a port, but **the owner of the competing service saying where its service listens**. And it
is not a legacy registration — AirPlay ships on every current Mac, Apple TV and HomePod.

**The controlling precedent is `9100/tcp`, and it is exact.** §4.6 excludes `9100` in one line:
*"node_exporter squats on a registration belonging to `hp-pdl-datastr`, 'PDL Data Streaming Port' —
i.e. JetDirect printers. One port, two completely different services, opposite populations.
Excluded."* Substitute the names and it is this row: Cassandra squats on a registration belonging to
`afs3-fileserver`, and Apple ships AirPlay on the same number. One port, two live services, opposite
populations — and the populations are opposite in the way that matters most to this product, because
Cassandra internode is an enterprise deployment and AirPlay is a consumer one.

**Why `9200` is not the precedent, which is the question a reader will raise.** `9200` squats on
`wap-wsp`, *"WAP connectionless session service"* — a registration for a protocol that has no
deployed population at all in 2026. Nothing else answers on 9200, so Elasticsearch's convention there
is uncontested in fact even though it is unregistered. That distinction has been doing silent work in
this note since §21 and is now written down as
[ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md): **a squat is
contested where the other convention is live**, and liveness is read off the competing owner's own
current documentation, never off a frequency source.

**The argument for admitting it anyway, stated and refused.** It is a good one: a 7000/tcp listener
*reachable from an internet vantage* is far more likely to be Cassandra than AirPlay, because AirPlay
receivers sit behind NAT. **That is a frequency argument, and this note rules frequency out at the
top of §1**; §11.5 refused a deployment-share argument by name, and §12.3 ground 1 refused another.
Determinacy asks whether the **pair determines a service**, not whether one of two services is more
often exposed. It also is not safely true: `Exposure` fires on `edge-only` as well as `exposed`
(§4.1), IPv6 estates give every device a globally routable address, and UPnP forwards ports without
being asked. A signal firing *"Cassandra internode transport exposed"* on somebody's Mac is §4.3's
Cockpit-on-9090 failure and §4.4's *"the list's entire value is that a firing is never arguable"*,
arriving on a different number.

**A weaker reading of the signal does not rescue it either.** §2.4 concedes that
`sensitive-port-exposed` *"claims a port associated with a sensitive service is reachable from an
internet vantage — it does not claim the service is running"*. `9090` has a genuine Claim 3
attestation from Prometheus and is excluded under that same framing, because *"a signal firing
'Prometheus exposed' on a Cockpit host is worse than no signal."* The weak framing was already priced
as insufficient and it is not re-priced here.

### 14.4 `7001/tcp` fails determinacy twice over, and either limb alone suffices

**Limb one — version-dependence, the §2.4 failure mode Hadoop failed, on the owner's own bytes.**
**[measured]** The comment block above the directive is not the same as `7000`'s. It carries three
extra lines, identically at all three tags:

```
# SSL port, for legacy encrypted communication. This property is unused unless enabled in
# server_encryption_options (see below). As of cassandra 4.0, this property is deprecated
# as a single port can be used for either/both secure and insecure connections.
# For security reasons, you should not expose this port to the internet. Firewall it if needed.
ssl_storage_port: 7001
```

And the switch that would bind it ships **off**, with the owner saying when to use it:

```
  # If enabled, will open up an encrypted listening socket on ssl_storage_port. Should only be used
  # during upgrade to 4.0; otherwise, set to false.
  legacy_ssl_storage_port_enabled: false
```

§2.4's second named failure mode is *"Version-dependent ports. Hadoop's NameNode web UI moved from
50070 to 9870 between major versions, so the inference from port to service depends on which version
is running. Excluded."* **Cassandra's encrypted internode transport moved from 7001 to 7000 at 4.0**,
by the owner's own sentence — *"a single port can be used for either/both secure and insecure
connections"* — so the inference from `7001` to *Cassandra encrypted internode* depends on which
major version is running, and on no supported release is it the default. This is the Hadoop case with
better evidence than Hadoop's, because it is stated in the shipped configuration rather than inferred
from two documentation sets.

**Limb two — the convention is contested, and one of the contestants is this product's own reference
data.** Three facts, in increasing order of force.

1. **IANA registers `afs3-callback,7001,tcp` and annotates it `Known Unauthorized Use on port
   7001`.** §2.4 is right that this field is a registry-hygiene matter and not a security judgement,
   and §9.3.4 already fixed the one use it is competent for: *"evidence about what else listens on
   the port — determinacy — which is the one thing it is actually competent to say."* Used here on
   exactly those terms and no others. Note that `7000`'s row carries **no** such annotation, so the
   two rows are not refused on the same evidence.
2. **Oracle documents WebLogic Server's AdminServer on 7001.** **[measured]** Oracle's own current
   container image README ships `ADMIN_LISTEN_PORT` `(default: 7001)` and its worked commands are
   `docker run -d -p 7001:7001 …`
   ([`oracle/docker-images`, `OracleWebLogic/dockerfiles/14.1.2.0/README.md`](https://github.com/oracle/docker-images/blob/main/OracleWebLogic/dockerfiles/14.1.2.0/README.md));
   the WebLogic administration guide states it in prose — *"It provides a single Listen Address, one
   port for HTTP communication (7001 by default), and one port for HTTPS communication (7002 by
   default)"*. That is the competing vendor about its own product, as Apple is for `7000`. It is also
   the worse shape: WebLogic's 7001 is an **HTTP admin console**, so `7001` is conventionally an HTTP
   alternate — the §4.3 category, in the same class as `8080` and `8443`.
3. **verge-asm's own frequency half already calls `7001` an HTTP alternate.** **[measured]**
   [`safe-active-probing.md`](./safe-active-probing.md) §2.3's modern-services supplement lists
   `7001` under **"HTTP-ish alternates: 3001, 4000, 5601, 7001, 8006, 8069, 8086, 8090, 8161, 8500,
   8834, 9000, 9090, 9200, 9300, 9443, 10000"**. So `7001` is *already* in `verge-core`, entered on
   the frequency half's own terms, labelled as the thing §4.3 excludes. Admitting the row would ship
   a product whose two port lists disagree about what `7001` is, and whose signal names Cassandra on
   a port its own probe schedule selected as a generic web port.

**Fact 3 corroborates and does not carry.** §6 forbids deriving the sensitive list from the hot set,
because that would make frequency a precondition of normativity; letting #4's service *labels* decide
a §2.4 question would be the same laundering arriving through the determinacy gate instead of through
membership. The ground for limb two is IANA plus Oracle. #4's label is recorded because a
**product-coherence** defect is worth seeing, and because it is the sharpest available illustration
that the convention is contested — not because #4 is entitled to attest anything.

### 14.5 §4.2 answered for this pair rather than assumed — and §4.2 is not even reached

The ticket asked whether TLS changes the verdict for `7001`, noting that §4.2 answers *no* in general
but that the pair deserves the answer rather than the assumption. **It is not reached, and saying why
is worth more than the answer.**

§4.2's rule governs a **Claim 2** pair: a plaintext port is wrong *because* a standardised encrypted
successor sits on a different port, which is why 23 is listed and 22 is not. `7000`/`7001` is not that
shape in either direction. `7000` is not a Claim 2 row — it is Claim 1 and Claim 3 — and `7001` is
not its successor but its **withdrawn** sibling: the owner collapsed the split, and the encrypted form
now lives on `7000` itself.

**So this pair is the §9.2 and §11.5 shape, and it is the third instance.** StartTLS hardens 389 on
389; SNMPv3 hardens 161 on 161; and **Cassandra hardens 7000 on 7000**, in the owner's own words —
*"a single port can be used for either/both secure and insecure connections"*, and, in
`server_encryption_options`, *"When set to true, encrypted and unencrypted connections are allowed on
the storage_port."* §9.2 recorded that RFC 6335 §9 steers implementers away from split ports and that
Class B's replacement is structurally invisible to a `(port, transport)`-keyed list; this is a project
executing that migration inside a single major version, with the before and after both in the file.

**And had §4.2 been reached, its answer would be unchanged.** TLS bears on Claim 2 and on nothing
else, so it could not have moved a row resting on Claim 3. `2375`/`2376` remains the demonstration.

### 14.6 Hazards met, recorded per §9.5, §11.9, §12.9 and §13.10

- **`cassandra.yaml`'s own header announces a second operative default, and §13.10's Elastic hazard
  was tested rather than assumed away.** The file opens by saying it *"is provided in two versions"*,
  `cassandra.yaml` and `cassandra_latest.yaml`, the second *"for new users of Cassandra who want to
  get the most out of their cluster"*. Both were retrieved at `cassandra-5.0.9`. **[measured]** They
  agree on every line this section relies on — the same prohibition above `storage_port: 7000`, the
  same deprecation block above `ssl_storage_port: 7001`, `listen_address: localhost`,
  `internode_encryption: none`, `legacy_ssl_storage_port_enabled: false` and a commented
  `#internode_authenticator:`. Elastic ships `localhost` in one operative default and `0.0.0.0` in
  another; Cassandra does not, and it took opening the second file to know that. A session that reads
  §12.2's *"Cassandra `conf/cassandra.yaml` … Operative"* as naming the only operative file has
  under-read the artefact by one.
- **`docs.oracle.com` serves a JavaScript shell to a plain client on its current documentation
  pages**, in the shape §9.5 recorded for `learn.microsoft.com` — the Fusion Middleware port-numbers
  appendix retrieves as 24 KB of loader script with no port table in it. The WebLogic default was
  taken instead from Oracle's own bytes on GitHub (`oracle/docker-images`) and cross-checked against
  an older Oracle documentation page that does serve text. Bytes over renderings, per §12.9.
- **`nmap-services` is used for rank and for its name column only, on §2.5's and §6.1's existing
  terms.** `7000/tcp` ranks 146th and `7001/tcp` 232nd of 8,387 TCP rows, against a top-100 boundary
  at open-frequency `0.003149` — used in §14.7 to state where the ports sit relative to the frequency
  half, and nowhere as grounds for a verdict, exactly as §4.3 uses it to note that 9090 is still
  `zeus-admin`.
- **The determinacy gate has no stated evidence standard, and this ticket is the first to lean its
  whole ruling on it.** §2.2 has three attestation forms and an owner definition (§10.5); §2.1 has a
  closed claim set (§10.2); §2.4 has *"uncontested convention"* and no account of what establishes
  it. This section had to weigh a vendor's product-port table, a vendor's container image README, an
  IANA annotation and a rendered manual against each other with nothing in the standard to rank them.
  The ruling does not turn on the gap — every source used is first-party about its own product, which
  is the strongest class on any plausible standard — but the gap is real and is opened as §8
  question 10, routed to [#82](https://github.com/winniel123/verge-asm/issues/82).

### 14.7 Every dependent figure, walked rather than asserted

| Where | Was | Is |
|---|---|---|
| §1 pair count | 37 | **37, unchanged** — and now definite for [#12](https://github.com/winniel123/verge-asm/issues/12) |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 = 37 | **unchanged.** No row enters, so no class total moves |
| §2.2 footing table | prohibition 9 · scoping 5 · weak 2, placing 19 of 37 pairs | **unchanged in every cell.** The two refused ports are not rows, so they take no cell, and the denominator [#76](https://github.com/winniel123/verge-asm/issues/76) works against stays **37** |
| §2.6's boast | true of all 37 | **unchanged** |
| §4.5 *the list's weakest row* | 5432/tcp | **unchanged** |
| §4.6 exclusions | 16 named | **18 named** — `7000/tcp` and `7001/tcp` join the negative space (§14.8) |
| §6.1 containment arithmetic | 28 in the hot set + 4 missing TCP + 5 UDP = 37 | **unchanged**, and checked for both ports rather than assumed: neither becomes a row, so neither enters the sensitive half and neither can create a containment gap |
| `verge-core` membership, for the record | — | **`7001/tcp` is already in the frequency half** ([`safe-active-probing.md`](./safe-active-probing.md) §2.3's supplement), so it is probed daily and fires no sensitive-port signal, which is what this refusal means. **`7000/tcp` is in neither half** — 146th by open-frequency, outside the top-100, and absent from the supplement — so it is reached only by the opt-in cold full-range sweep. Both are correct outcomes of decisions already made and neither is a defect (§14.9) |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | `verge-core = frequency-set ∪ sensitive-list` | **unchanged** — no member enters or leaves. The ADR needs no amendment |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version, and the `Break` | — | **not triggered.** `sensitive-port-reached-from-internet`'s content is byte-identical. No `Break`, no version bump, no aperture change, no comparability cycle |
| §13.5's routing | *"Routed to #75, which blocks #12"* | **discharged.** #75 is answered; #12's blocker clears with the count it was waiting on |
| §12.8's version flag | *"Cassandra's prohibition is measured on 5.0.2 alone"* — discharged by §13 at three tags | **unchanged, and extended**: the same three tags plus `cassandra_latest.yaml` at `cassandra-5.0.9` |
| §8 | 9 questions, all closed or routed | **10** — question 10 opened (§14.6) |

**Outside this note.** [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md)
is added and is a rule about the determinacy gate rather than a table edit. ADR-0008, ADR-0009,
ADR-0032, ADR-0036 and ADR-0037 are untouched. **ADR-0037's limb 2 is confirmed by outcome rather
than by argument** — it predicted that a subject found beside a listed one supplies the attestation
limb only and that *"§2.4's determinacy gate is live and may sink the row despite the strongest kind
of sentence the corpus has."* It did, on both ports.

### 14.8 The two rows added to §4.6's negative space

Per §4.6's own principle that *"the exclusions are as much of the deliverable as the list, because a
curated list is judged on what it refuses"*.

| Excluded | Why |
|---|---|
| **7000/tcp Cassandra internode storage/gossip** | Claim 1, Claim 3 and §2.2's attestation all pass — an owner prohibition naming the port by number in operative shipped bytes at three tags, with `listen_address: localhost` beside it. **Fails determinacy**: it squats on `afs3-fileserver`, and Apple documents `AirPlay · 7000 · TCP` as a live competing service. The `9100` rule, now [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md) — §14.2, §14.3 |
| **7001/tcp Cassandra `ssl_storage_port`** | Same prohibition, and it still fails determinacy on **two** independent limbs. Version-dependence: the owner deprecated it at 4.0 and moved encrypted internode onto `7000`, shipping `legacy_ssl_storage_port_enabled: false`. Contested convention: IANA `afs3-callback` with a `Known Unauthorized Use` annotation, Oracle's WebLogic AdminServer, and this project's own frequency half already classing `7001` as an HTTP-ish alternate — §14.4 |

**The criterion that would change either verdict.** For `7000`: Apple retiring AirPlay from 7000, or
some other event that leaves Cassandra's convention uncontested — at which point the row is a clean
admission on evidence that already exists, priced as an addition. For `7001`: nothing reachable. The
owner has withdrawn the port, so the direction of travel is away from admissibility, and re-admitting
it would require Cassandra to un-deprecate `ssl_storage_port` **and** Oracle to move WebLogic.

### 14.9 Thin ground, flagged per the standing rule

**The `7000` refusal is the thinnest single judgement in this section**, and it is thin in a stated
way. It turns on AirPlay being a *live* competing convention, and liveness is a property of the world
rather than of a document — the very kind of judgement §10.1 deleted when it removed *"would
otherwise require authority"* for asking a reviewer to imagine a counterfactual. ADR-0042 answers
that by keying liveness to **the competing owner's own current documentation** rather than to
deployment share, which is the tightest available footing and is not a proof. A reader who says that
Apple's ports table is a specification of *Apple's* products and cannot bear on what `7000` means
generally has a real argument; the reply is that the same reader must then explain `9100`, which is
excluded on nothing stronger.

**The claim that no other live service occupies `7000` or `7001` is not made.** What is established
is that at least one does, on each, from that service's own vendor. That is sufficient to fail a gate
that requires convention to be **un**contested, and it is not a survey.

**`7001`'s version-dependence limb rests on a comment, not on a code path.** The deprecation sentence
and `legacy_ssl_storage_port_enabled: false` were read from the shipped configuration and from the
`Config.java` field; the listener-binding code that consumes the flag was not read. The comment is the
owner's own prose under §12(b) and the flag is an active directive under §12(a), so the finding stands
on the standard's own terms — but a session that needs the binding behaviour itself has not got it
here.

**Neither port's absence from `verge-core`'s frequency half is treated as a defect, and that is a
ruling rather than an omission.** #4's supplement admits a port because it *"maps to a named v1 risk
signal"*. `7000` maps to none once this section refuses it, so its absence is the frequency half
working correctly rather than a gap; and `7001`'s presence is #4's own HTTP-alternate judgement, which
this note has no standing to revise. **The criterion that would reopen it:** a v1 signal other than
`sensitive-port-reached-from-internet` that `7000` maps to.

---

## 15. §2.4 gets an evidence standard — a convention is evidenced by placement, never by catalogue

Wayfinder ticket [#82](https://github.com/winniel123/verge-asm/issues/82), the gap
[ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md) named in its own
Consequences and §14.6 recorded as a hazard met. This section **amends §2.4, §4.3, §8 and §9.3.4 by
reference**; earlier text stands and is marked, per the name-and-withdraw convention, and where §15
and an earlier section disagree, **§15 governs**.

**Headline result, stated first.**

> **A determinacy finding is made on *placement statements* and on nothing else** — a party's
> statement, in its own current documentation or its own shipped bytes, that **its own software
> listens on a given `(port, transport)` pair by default**. Everything else — IANA rows, the
> `Unauthorized Use Reported` field, `nmap-services`' name column, cloud-provider and government port
> tables, and **this project's own frequency half** — corroborates and never carries, which is §2.3's
> rule one gate across with a descriptive population instead of a normative one.
> [ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md).
>
> **No `(port, transport)` pair moves. The list stays at 37.** Nine rulings were re-run against the
> stated criterion rather than assumed — ADR-0042's six, plus `7001`, `9090` and the convention-resting
> rows walked individually in §15.4 — and every verdict reproduces.
>
> **Two grounds move, both on exclusions, and both get stronger.** `9090/tcp` was excluded on
> `nmap-services`' name column, which limb 5 makes inadmissible as grounds; it is re-founded on Red
> Hat's own current documentation putting **Cockpit** on 9090. `79/tcp`'s determinacy limb was carried
> by IANA's `Unauthorized Use` annotation; it is re-founded on **RFC 4146** itself, with the annotation
> corroborating.
>
> **One limb was forced by the walk and is the section's real finding.** `9200/tcp` and `9042/tcp`
> both have current first-party placement statements from parties other than the row's own owner —
> OpenSearch and the Wazuh indexer on 9200, ScyllaDB on 9042. **The unit is the protocol, not the
> vendor**: parties that declare they speak the row's own protocol are one convention, not two.
> Without that limb this standard would have deleted two rows nobody thinks are wrong.
>
> **And the phrase that would have let frequency back in is withdrawn.** ADR-0042 explains `9200` with
> *"WAP has no deployed population"*. **Liveness is the currency of a declaration, not the size of a
> population**, and §15.3 re-founds `9200` on that footing.

### 15.1 The standard

> **Placement statement.** A party's statement, in its **own current documentation** or its **own
> shipped or compiled bytes**, that **its own software listens on a given `(port, transport)` pair by
> default**.

| Limb | Rule |
|---|---|
| **1 — establishing** | A row's convention is established by the **candidate's own owner's** placement statement, in §10.5's sense of owner. Not by us, not by a catalogue |
| **2 — defeating** | **Contested** where another party has a current placement statement on the same pair for a **different protocol**; **displaced** where the candidate's own owner puts its service on a different pair or makes the pair version-dependent. **One statement suffices**; a survey is never required and never claimed (§14.9) |
| **3 — the unit** | The unit is the **protocol**, not the vendor. Two parties placing the same protocol on a pair are **one** convention. Compatibility is read off the second party's own declaration, never judged by us |
| **4 — currency** | **Current** means the party still presents the statement as applicable: bytes at a supported release, docs for a supported product, or a specification **in force** — not obsoleted, withdrawn, or reclassified Historic. *Live* means current, **never** numerous |
| **5 — everything else** | Corroborates and never carries. IANA rows and the `Unauthorized Use Reported` field, `nmap-services`' name column, cloud and government port tables, third-party port references, and **this project's own frequency half** |

> **The refusal artefact rule.** Determinacy is a **defeasible presumption**: once limb 1 is met it
> holds until a document defeats it. **Every refusal on determinacy names the artefact that defeated
> it**, quoted and dated, in §4.6's negative space. A determinacy refusal citing no document is not a
> finding. This is the operative change for the next session, and it is the direct answer to the
> asymmetry §14 exposed — an admission leaves a row to argue with and a refusal, until now, left
> nothing.

**Two riders.** *First-party is a property of the row, not of the document* — Apple's ports table is
first-party about `AirPlay · 7000 · TCP` and third-party about anything else in it, which is
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)'s shape on
the other gate. And *a registration reaches the limbs only through its registrant* — an IANA row is a
**record of a placement declaration**, not an authority, so it is followed to the registrant's current
documents and stands or falls with them.

**Where it comes from, in one paragraph.** [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md)
§6 cut three instruments on where a table sits relative to the wire — what we **ask**, what a byte
**means**, what a value means **normatively**. §2.1 and §2.2 are *after* the wire; **§2.4 is at it**,
which is what ADR-0032 meant by calling determinacy the **surrogate** gate. So determinacy's evidence
standard is [#31](https://github.com/winniel123/verge-asm/issues/31)'s and not
[#21](https://github.com/winniel123/verge-asm/issues/21)'s: *a table deciding what an answer means is
a signature database*, and a port-to-service mapping asserted by a third party is exactly that —
`nmap-services` **is** one. A placement statement is the port-number analogue of #31's spec-defined
field: the field is defined by the party that defines the software. **No fourth instrument is minted
and the table does not move between them.**

**And there is no second sense of *owner*.** The ticket asks whether Apple, owning AirPlay but not
port 7000, needs naming as an **owner of a convention** the way
[ADR-0035](../adr/0035-a-cryptographic-primitives-owner-is-its-specifier.md) named the primitives
case. **No.** §10.5's *owner* is a normative entitlement — *who may say exposure is wrong* — and
determinacy asks a factual question nobody is entitled to answer, which the registry disclaims in
capitals. Minting the second sense would make Apple's prose about `7000` admissible under §2.2. What
Apple's table is instead is a **self-declaration**, falsifiable by running the software and
self-interested in the direction that makes it reliable: a vendor that misstates its own default port
ships a product that does not work. That is §12.2's *"nine artefacts, nine self-declarations"*
applied one gate across.

### 15.2 This project may not supply its own determinacy evidence — §14.4's passing refusal, stated as a rule

§14.4 fact 3 declined to let [`safe-active-probing.md`](./safe-active-probing.md) §2.3's *HTTP-ish
alternates* label decide `7001`, and did it in a subordinate clause. **Promoted to limb 5, on three
grounds where one was given.**

- **§6's invariant is one-directional.** Deriving the sensitive list from the hot set would make
  frequency a precondition of normativity. Letting the hot set's **labels refuse** a sensitive row is
  the same laundering with the sign flipped.
- **The labels are frequency artefacts.** *HTTP-ish alternate* is a probe-scheduling grouping produced
  by [#4](https://github.com/winniel123/verge-asm/issues/4)'s frequency question. It is not a claim
  about what listens on a number and it was never retrieved from anybody.
- **§2.2's first sentence already binds.** *"The claim may not be asserted by us."* Determinacy
  inherits it **in both directions**: we may not establish a convention either, so no row may be
  admitted because our own list says a number means something.

The **product-coherence** use §14.4 made of it is untouched and stays: recording that this product's
two port lists disagree about `7001` is worth seeing, and is not evidence.

### 15.3 `9200/tcp` re-founded — the phrase that would have let frequency back in

ADR-0042's reconstruction table explains the note's oldest determinacy asymmetry with the words
**"No — WAP has no deployed population"**. That is a population sentence inside a rule that forbids
population sentences, and left alone it is the crack the next session widens: if *no deployed
population* establishes uncontestedness, *a large deployed population* is one step from establishing
contestedness, and §1's exclusion of frequency is gone.

**Re-founded on two grounds, in order of force, with the weaker one marked.**

1. **No current placement statement for WSP on `9200/tcp` was found.** The registrant is the WAP
   Forum, absorbed into the Open Mobile Alliance in 2002; the surviving specification —
   [*Wireless Session Protocol* `OMA-WAP-TS-WSP-V1_0-20110315-A`](https://www.openmobilealliance.org/release/Browser_Protocol_Stack/V2_1-20110315-A/OMA-WAP-TS-WSP-V1_0-20110315-A.pdf) —
   sits in an archived release of a suite no party currently ships or supports. Under limb 4 that is
   **not current**, and it places nothing.
2. **The WAP bearer ports are UDP, so they are not this key.** WSP runs over WDP, and the four
   standard WDP ports are `9200`-`9203` **UDP**. This table is keyed on `(port, transport)` pairs;
   `9200/tcp` and `9200/udp` are different keys, and a competing convention on the wrong transport
   cannot contest a pair it is not on. **Marked as corroborated rather than measured** — the transport
   fact is taken from a third-party protocol reference and from the IANA row, both limb-5 sources, and
   the primary that would settle it is OMA's own WDP specification, which was not retrieved (§15.7).

**And a third party now places a live service on the number, which is why limb 3 exists.** OpenSearch
defaults `http.port` to the `9200-9300` range in its own network-settings documentation, and the Wazuh
indexer is distributed on the same number. Both declare the Elasticsearch REST API, so under limb 3
they are **the same convention**, not a contest — and a firing that names Elasticsearch and finds
OpenSearch has named the service at the granularity the row asserts: same protocol, same Claim 3, same
remediation.

**What the re-founding costs, stated plainly.** The rule is deliberately non-monotone in deployment
size, in both directions. Had WAP a hundred million handsets in the field and no current placement
document, `9200/tcp` would still be listed; had AirPlay a thousand installs and a current Apple page,
`7000/tcp` would still be refused. That is uncomfortable and it is the price of a gate a reviewer can
run as a retrieval instead of a judgement — the same trade §10.1 made when it deleted *"would
otherwise require authority"* and §12.4 made when it refused a coherence gate.

### 15.4 The walk — every row resting on convention rather than registration

#37's discipline: the standard is not repaired against a list nobody re-tested. §3's `reg.` column
marks the rows that rest on convention with `sq.` or `--`. **All nine are walked, and each names what
was looked for.**

| Row | Registration | Competing placement statement found? | Verdict |
|---|---|---|---|
| `9200/tcp` Elasticsearch | `sq.` `wap-wsp` | **No.** WSP: no current statement, and the WAP bearer is UDP (§15.3). OpenSearch and Wazuh indexer: **same protocol**, limb 3 | **Listed, unchanged** |
| `9300/tcp` Elasticsearch transport | `sq.` `vrace` | **No.** No current placement document for *Virtual Racing Service* found under that name or its registrant. OpenSearch's transport range is the same protocol | **Listed, unchanged** |
| `2181/tcp` ZooKeeper | `sq.` `eforward` | **No.** Every current first-party document found on 2181 is ZooKeeper's own or a ZooKeeper **client's** connection string, which places nothing of its own | **Listed, unchanged** |
| `9042/tcp` Cassandra | `--` unassigned | **No contest.** ScyllaDB's configuration reference defaults `native_transport_port` to `9042` — a different vendor and a different product, declaring **CQL**, so limb 3 makes it one convention | **Listed, unchanged** |
| `10250/tcp` kubelet | `--` unassigned | **No.** Kubernetes' own ports-and-protocols reference is the only current placement statement found; an upstream proposal to reuse the number inside the same project places nothing | **Listed, unchanged** |
| `10255/tcp` kubelet read-only | `--` unassigned | **No.** Same corpus, same result | **Listed, unchanged** |
| `25672/tcp` RabbitMQ inter-node | `--` unassigned | **No.** RabbitMQ's own networking guide derives it as `NODE_PORT + 20000`; nothing else found | **Listed, unchanged** |
| `6000/tcp` X11 display :0 | `--` per §3.2 | **No.** Registered to `x11` across 6000-6063 per §2.4, and the registrant is the placer | **Listed, unchanged** |
| `623/udp` IPMI | `yes` per §3.3, `sq.` per ADR-0042 | **No contest.** `asf-rmcp` is the **transport IPMI rides**, not a rival service, so limb 3 disposes of it whichever way the cell reads | **Listed, unchanged** |

**Two `reg.` cells are inconsistent with §2.4 and neither changes a verdict.** §3.2 marks `6000/tcp`
`--` while §2.4 states that *"6000/tcp is registered, as `x11`, across the 6000-6063 range"*; and
§3.3 marks `623/udp` `yes` while §2.4's own table and ADR-0042 treat it as a squat on `asf-rmcp`.
Both rows survive on either reading, so the cells are **recorded here rather than corrected** — the
tables are in another session's lane this week, and a `reg.` cell is disclosure rather than
qualification by §3's own sentence.

### 15.5 The exclusion side — the two grounds that move, and §4.3's grandfathering

**`9090/tcp` — ground replaced, verdict unchanged.** §4.3 excludes it partly because *"`nmap-services`
still lists it as `zeus-admin`"*, and limb 5 makes that inadmissible **as grounds**. The exclusion does
not need it: Red Hat's current RHEL 8 and RHEL 9 web-console documentation states that the console
*communicates through TCP port 9090*, which is a first-party placement statement for **Cockpit** — a
completely different protocol from Prometheus exposition, with no compatibility declared in either
direction. §4.3's own sentence — *"9090 is conventionally Cockpit"* — was right and is now footed.

**`79/tcp` — ground strengthened, verdict unchanged.** §9.3.4 leans its determinacy limb on IANA's
`Unauthorized use by some mail users` annotation, carefully limited to *"evidence about what else
listens on the port"*. Under this standard the annotation **corroborates**, and the carrying artefact
is the one it points at: **RFC 4146**, in force, in which the IETF specifies that *"a process must be
listening to the finger port"* for a protocol that is not finger. That is the specifying party's own
statement about its own protocol — limb 2, cleanly — and §9.3.4's care about the field is vindicated
rather than reversed.

**§4.3's ten generic ports are grandfathered as a class, and the refusal artefact rule is prospective
for them.** `8080`, `8000`, `8888`, `8443`, `3000`, `5000`, `9000`, `9090`, `8088` and `10000` are
excluded as a category, and re-founding all ten on cited artefacts is not this ticket's work. **Each
owes its artefact the first time it is individually relied on** — the first row-level ruling that
turns on one of these numbers names the document, as `9090` now does. Grandfathering a category and
pricing the next individual use is §10.4.3's shape and not an exemption.

### 15.6 Every dependent figure, walked rather than asserted

| Where | Was | Is |
|---|---|---|
| §1 pair count | 37 | **37, unchanged** — no row is admitted or removed |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 | **unchanged.** No row enters or leaves a class |
| §2.2 footing table | prohibition 9 · scoping 5 · weak 2, placing 19 of 37 | **unchanged in every cell**, and its denominator stays **37** — this section touches determinacy only, which takes no footing cell ([#76](https://github.com/winniel123/verge-asm/issues/76) works against the same 37) |
| §2.6's boast | true of all 37 | **unchanged** — no row's *attestation* is touched |
| §4.5 *the list's weakest row* | 5432/tcp | **unchanged.** §4.5's weakness is an attestation weakness and this section adds nothing to it in either direction |
| §4.6 exclusions | 18 named | **18, unchanged.** Two entries (`9100`, and `79` via §9.3.4) have their determinacy ground restated; none is added or removed |
| §6.1 containment arithmetic | 28 in the hot set + 4 missing TCP + 5 UDP = 37 | **unchanged** — no member enters or leaves either half |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | `verge-core = frequency-set ∪ sensitive-list` | **unchanged.** No amendment needed |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version, and the `Break` | — | **not triggered.** `sensitive-port-reached-from-internet`'s content is byte-identical. No version bump, no aperture change, no comparability cycle |
| §8 | 10 questions, question 10 open | **10 — question 10 closed** |
| ADR-0042's reconstruction table | six verdicts, one criterion | **six verdicts unchanged**, two grounds restated (`9200`, and `79` in this note); limb 3 added |

**Outside this note.**
[ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md) is added.
[ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md) is amended in one
phrase and keeps its criterion. ADR-0008, ADR-0009, ADR-0032, ADR-0035, ADR-0036 and ADR-0037 are
untouched — and **ADR-0032's §4 prediction is confirmed by use rather than by argument**: it said any
future rule keying on a surrogate owes a determinacy argument, and the surrogate gate has now acquired
the source rule that makes such an argument checkable.

### 15.7 Retrieval method and hazards, recorded per §9.5, §11.9, §12.9, §13.10 and §14.6

- **The walk's negatives are searched, not read as bytes, and that is the section's main limitation.**
  §15.4's *no competing placement statement found* cells rest on searches over vendor and project
  documentation for each number, not on the artefact-by-artefact reading
  [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) requires
  of an **attestation**. That is defensible under the standard's own burden — a negative here is a
  presumption not yet defeated, never a proven absence (§14.9's rule, generalised) — but a session that
  needs *proven absence* has not got it here, and per
  [ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md) the corpus is named
  rather than implied: vendor and project documentation reachable by search, plus the IANA row for each
  registered competitor. **The smallest extension that could change an answer** is a first-party
  document from any party shipping software on one of these nine numbers.
- **The positives are stronger than the negatives, and deliberately so.** Every *defeating* artefact
  cited in §15.5 and in ADR-0048's reproduction table is a named, current, first-party document. The
  asymmetry is the standard's, not the retrieval's: one document defeats, and no number of documents
  proves a negative.
- **`9200`'s transport limb rests on a corroborator.** The claim that the WAP bearer ports are UDP is
  taken from a third-party protocol reference and the IANA registration, both limb-5 sources. It is
  offered as the **second** ground behind currency for exactly that reason, and the primary that would
  settle it is OMA's own WDP specification, unretrieved.
- **The `9090` re-founding was checked against Red Hat's current documentation rather than against
  memory**, in both the RHEL 8 and RHEL 9 web-console guides, because a single-version citation is the
  §12.8 hazard.
- **The temptation this section had to refuse by name.** The obvious way to write a liveness standard
  is *the competing service must be widely deployed*, and it is available in every source. It is
  frequency, §1 excludes it, ADR-0042 limb 3 excludes it, and limb 4 is the fence: **currency, not
  size.** Recording it because the word *live* invites the substitution and the next session will feel
  the same pull.

### 15.8 Thin ground, flagged per the standing rule

**Limb 2's *one statement suffices* is the thinnest structural choice here.** Somewhere on the
internet a party can probably be found documenting some product on almost any number, and one such
find would defeat a row. Two things hold it: the placement must be a **documented default of a
generally available product**, which excludes *our software can be configured on 3306*; and the
error direction is the one this table has always chosen — §4.4's *"the list's entire value is that a
firing is never arguable"*. **The criterion that would falsify it** is the walk itself: if a later
pass finds defeaters for most convention-resting rows, the gate is too easy to trip and this standard
is wrong. This pass found none for nine of nine.

**Limb 3 is the newest sentence in the section and had no prior instance until this walk.** It is
read off §4.6's *"completely different services"* and confirmed on two rows (`9200`, `9042`) plus a
third by construction (`623/udp`). Its test — *does the second party declare it speaks the first's
protocol?* — is mechanical where the declaration exists and has no answer where a second party is
simply silent about compatibility. **The criterion that would sharpen it:** a candidate row where a
second party places a partially compatible protocol on the number and declares neither.

**Nothing here is measured in the `[measured]` sense the note reserves for bytes.** §14's Cassandra
findings were bytes at named tags; this section's retrievals are documents read for a sentence. The
distinction matters because the standard this section writes is about **documents**, so the artefacts
are the right kind — but no claim in §15 should be quoted as measured.
## 16. The footing table's seven uncovered rows are placed, and it now states its own coverage

Wayfinder ticket [#76](https://github.com/winniel123/verge-asm/issues/76), on the coverage gap §13.7
counted and routed rather than filled. This section **amends §2.2 and §3.4 by reference**; earlier
text stands and is marked, per the name-and-withdraw convention, and where §16 and an earlier section
disagree, **§16 governs**.

**Headline result, stated first.**

> **All seven are placed, the table now says what it covers, and the ticket's central prediction was
> wrong in the direction that matters.** §13.7 expected `2379`/`2380` etcd to fall into the **weak
> tier** on a shipped default alone, taking it from two rows to four. **[measured]** etcd ships
> `THREAT_MODEL.md` at `etcd-io/etcd` `v3.7.0` and `v3.7.1`, and it says *"It **must not** be exposed
> to untrusted networks or the public internet"* — naming **Port 2379** and **Port 2380** by number in
> the same document. Both go to the **explicit prohibition** tier.
>
> **The weak tier does grow, by one, and not from etcd.** `10255/tcp` kubelet joins it: the owner
> states no network position anywhere, and what exists is a **restricting** shipped default —
> `readOnlyPort` *"Default: 0 (disabled)"* in the shipped config API. **The weak tier is three rows —
> `5432` PostgreSQL, `5984` CouchDB and `10255` kubelet.** The map's curation patch reads the weak
> tier as the curator's watch list, so the watch list gains a member.
>
> **The table now places 26 of the 37 pairs, and the remaining eleven are named in the table itself
> as deliberately out of its subject** — §13.7's count had to be done by hand, and #70 had to do it
> twice. The coverage line is now part of the artefact rather than a fact about it.
>
> **No `(port, transport)` pair moves and no row moves.** A footing is evidence for a claim and not a
> claim ([ADR-0036](../adr/0036-a-shipped-default-is-the-configuration-that-takes-effect.md), §12.7),
> so this changes §2.2's **disclosure** and nothing the rule reads: the list stays at **37 pairs**,
> class totals stay **12 / 7 / 18**, no rule version bumps and `verge-core` does not move.
>
> **Two by-catch findings are routed rather than decided**, both of them row grounds rather than
> footings, and both found by [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)'s
> instrument: §3.4's kubelet defaults are quoted from a **generated page the shipped source
> contradicts** (§16.5), and `4369/tcp`'s attestation is a **non-owner's** (§16.6).

### 16.1 What was retrieved

Every artefact was read as shipped bytes at a named tag or release branch, with paths resolved from
the project's own repository tree rather than guessed — §11.9's and §13.10's discipline, and the
reason `THREAT_MODEL.md` was found at all.

| Project | Artefact, at the tag or branch named | What it settles |
|---|---|---|
| etcd | `THREAT_MODEL.md` — `etcd-io/etcd` `v3.7.1`, `v3.7.0` and `main` | `2379`, `2380` |
| etcd | `etcd.conf.yml.sample` — `v3.7.1` (re-read; §13.6's artefact) | corroborating default |
| etcd | `content/en/docs/v3.6/op-guide/security.md` — `etcd-io/website` `main` | the website states **no** position |
| Elasticsearch | `docs/reference/elasticsearch/configuration-reference/networking-settings.md` — `elastic/elasticsearch` `v9.5.1` | `9300` |
| Elasticsearch | `distribution/src/config/elasticsearch.yml` — `v9.5.1` (re-read; §13.1's artefact) | no transport prose anywhere in it |
| Kubernetes | `staging/src/k8s.io/kubelet/config/v1beta1/types.go`, `pkg/kubelet/apis/config/v1beta1/defaults.go` — `kubernetes/kubernetes` `v1.34.1` | `10255`, and §16.5's by-catch |
| Kubernetes | `content/en/docs/reference/networking/ports-and-protocols.md` — `kubernetes/website` `release-1.34` | `10250` |
| Kubernetes | `content/en/docs/reference/command-line-tools-reference/kubelet.md` — `release-1.34` | the stale generated page |
| Erlang/OTP | `erts/doc/references/epmd_cmd.md`, `system/doc/reference_manual/distributed.md` — `erlang/otp` `OTP-29.0.5` | `4369`, and §16.6's by-catch |

`2375/tcp` Docker needed no retrieval: the sentence is already in §3.4, cited for its cell-mate.

### 16.2 The two clerical placements

**`2375/tcp` Docker → trusted-network scoping, with `2376`.** §3.4 already carries *"It is also
recommended to ensure that it is reachable only from a trusted network or VPN"* from Docker Engine
security, and the sentence is about the daemon's remote API rather than about either port
individually. `2376` sits in the scoping tier on it; `2375` is the same sentence's other port. The
placement is clerical in the strict sense — no new evidence, and refusing it would put two ports of
one API in two tiers on one sentence.

Two things deliberately do **not** promote it. Docker's *"anyone with access to that port has full
Docker access; so it's not advisable on an open network"* is a **preference** in §2.3's and §4.4's
sense, not a prohibition. And §13.2 found Docker's operative default has **no TCP listener at all**,
which is a restricting default under §10.4 — that corroborates, and the tier records the strongest
form available, which here is the second form's scoping sentence.

**`4369/tcp` epmd → trusted-network scoping, with `25672`. The placement is right and the ticket's
reason for it is not** — see §16.6, which is why this row took a retrieval after all.

### 16.3 `2379` and `2380` etcd — the ticket's crux, and it resolves the other way

§13.7's hypothesis was precise and testable: etcd's own sentence in §3.4 is a **consequence**
(*"An etcd cluster which doesn't enable security features can expose its data to any clients"*), the
*"only the API server should have access to it"* sentence is **kubernetes.io** and therefore
corroboration under §10.5, and if etcd's own prose states no position then both ports rest on a
shipped default alone.

**The first two limbs are confirmed and the third is false.** **[measured]** `etcd-io/website`'s
current `v3.6/op-guide/security.md` carries the consequence sentence verbatim and nothing else: a
sweep for `internet`, `untrusted`, `expose`, `firewall`, `trusted network` and `public` returns the
consequence sentence, two TLS certificate mechanics, and no position. **So a session that looked
where §3.4 points would have confirmed the hypothesis.**

**The position is in the repository, not on the website.** **[measured]** `THREAT_MODEL.md`,
`etcd-io/etcd` `v3.7.1`:

```
### The Network Boundary

etcd Server assumes it is deployed within a strictly isolated, private network segment.
It **must not** be exposed to untrusted networks or the public internet.
Both the **etcd Client** and the **etcd Server** reside inside this protected perimeter.
```

And the same document names both listed ports, by number, in its own section headings:

```
### The Client-to-Server Boundary

etcd clients communicate with etcd Servers over Port 2379.
```

```
### The Peer-to-Peer Boundary

etcd Server members communicate with other cluster members over Port 2380 to run Raft consensus.
This boundary must be strictly limited to authorized cluster members using dedicated, private peer
certificates (mTLS).
```

That is the owner — etcd, the project that designed the protocol and authors the reference
implementation — writing *must not be exposed to … the public internet* in bytes it ships, and
binding it to each of the two listed ports. It is a **position** under §12(b), not a directive and
not a label: it has no configuration setting beneath it to describe. **Both ports go to the explicit
prohibition tier.**

**The strongest objection, and it is a good one.** The commit that created the file
(2026-05-19) is titled *"Define THREAT_MODEL for etcd to decentivize agents reporting CVEs outside
it"*, and the document's second line reads *"Automated vulnerability scanners and security
researchers MUST evaluate any security concern against these baseline boundaries."* A reader can
say the file's purpose is **triage scope** — what etcd will accept as a vulnerability report — rather
than deployment guidance, and that reading it as a prohibition is reading a disclaimer as an
instruction.

**It is refused, on three grounds, and the third is the one that decides it.**

1. **The sentence is prescriptive, not descriptive.** The paragraph has two sentences and they do
   different work: *"etcd Server **assumes** it is deployed within…"* is the assumption, and *"It
   **must not** be exposed to untrusted networks or the public internet"* is the requirement, in the
   project's own bolded RFC-2119 register. §12(b) takes *"a quotable position wherever the owner
   wrote it"*, and does not ask what the surrounding document is for.
2. **A threat model is a stronger commitment than a hardening tip, not a weaker one.** Declaring
   internet exposure outside the trust boundary is the project stating it **will not defend** that
   deployment. Every other prohibition in the corpus is a recommendation the owner could quietly walk
   back; this one has a cost attached, in the §10.4 sense — etcd pays for it by declining reports.
3. **The purpose reading proves too much, and §13.3 already refused its mirror image.** rsync's
   systemd unit quotes a README sentence naming *public* file distribution, and §13.3 refused to read
   it as an endorsement partly because *"the comment's stated purpose is the opposite"* — it cites the
   sentence to justify hardening. Purpose was allowed to inform the reading there and it is allowed
   to here, and it points the same way both times: a document written to bound what the project
   defends is a document about where the project expects to be deployed.

**Version scope, stated rather than smoothed.** **[measured]** `THREAT_MODEL.md` is present at
`v3.7.0` and `v3.7.1` and **absent at `v3.6.14` and `v3.5.33`** — it is a 3.7-era document, and §3.4
still cites `etcd.io/docs/**v3.5**/op-guide/security/`. The Network Boundary paragraph is
**byte-identical at `v3.7.0`, `v3.7.1` and `main`**, across the one revision the file has had
(2026-07-31, which added triage disposition rules and left this paragraph untouched). §13.1 read etcd
at `v3.7.1`; this section reads the same tag, which is the consistent choice. §16.9 flags what that
costs.

### 16.4 `9300` Elasticsearch — the *node* sentence does reach the transport port, and the owner says so

§13.7 framed this as the open question: `9200`'s *"Never expose an unprotected node to the public
internet"* is about a **node**, so it may reach `9300` too. **[measured]** It does, and the link is
made by the owner's own document rather than by inference.

The sentence lives in `networking-settings.md`, which is the page that configures **both** interfaces,
and it appears immediately after the two paragraphs that establish that:

> "Each Elasticsearch node has two different network interfaces. Clients send requests to
> Elasticsearch's REST APIs using its **HTTP interface**, but nodes communicate with other nodes using
> the **transport interface**." … "By default Elasticsearch binds only to `localhost` which means it
> cannot be accessed remotely."
> — `docs/reference/elasticsearch/configuration-reference/networking-settings.md`, `elastic/elasticsearch` `v9.5.1`

> "**Never expose an unprotected node to the public internet.** If you do, you are permitting anyone
> in the world to download, modify, or delete any of the data in your cluster."
> — same file, in a `warning` admonition three lines below

And the setting the warning is about governs both ports explicitly:

> "`network.host` … Sets the address of this node for **both HTTP and transport traffic**." …
> "`transport.port` … The port to bind for communication between nodes. … Defaults to `9300-9400`."

So *node* is the owner's own unit of exposure here, the page defines the node as its two interfaces,
and the transport interface is the listed pair. **`9300` goes to the explicit prohibition tier, in the
same cell as `9200`.** §3.4's transport-specific sentence — *"Transport connections between
Elasticsearch nodes are security-critical and you must protect them carefully"* — stands underneath it
and is not what carries the row: on its own it is an instruction to protect rather than a position on
placement, which is precisely why the row needed this retrieval.

**[measured]** The shipped `elasticsearch.yml` at `v9.5.1` contains **no** prose about the transport
interface or about `9300` at all; its Network section is about `network.host` and `http.port` only. A
session that had gone to the config file — the §13 instrument — would have found nothing. The
attestation is in the documentation source, retrieved from the same tree at the same tag.

### 16.5 `10250` and `10255` kubelet — one to scoping, one to the weak tier

**`10250/tcp` → trusted-network scoping.** Kubernetes owns the kubelet: it authors the reference
implementation, so kubernetes.io speaking about the kubelet is the owner speaking, which is exactly
what it is **not** doing for etcd in §16.3. **[measured]** `ports-and-protocols.md` at
`kubernetes/website` `release-1.34` places the port in both the control-plane and the worker-node
table, with the same scope in both:

```
| Protocol | Direction | Port Range | Purpose      | Used By             |
| TCP      | Inbound   | 10250      | Kubelet API  | Self, Control plane |
```

That names the boundary — the node itself and the cluster's control plane — which is what §10.3's
boundary limb asks for. The shipped default is `Address = "0.0.0.0"`
(`pkg/kubelet/apis/config/v1beta1/defaults.go`, `v1.34.1`), permissive and therefore **silent** under
§10.4, so the second form is all there is. **Scoping tier**, and it is the weakest member of that tier
— §16.9.

**`10255/tcp` → the weak tier.** **[measured]** The port does not appear in `ports-and-protocols.md`
at all, and `kubelet-authn-authz.md` says nothing about network placement. The owner states **no
position** on `10255` anywhere that was retrieved. What exists is a restricting shipped default, in
the config API's own bytes:

```go
// readOnlyPort is the read-only port for the Kubelet to serve on with
// no authentication/authorization.
// The port number must be between 1 and 65535, inclusive.
// Setting this field to 0 disables the read-only service.
// Default: 0 (disabled)
ReadOnlyPort int32 `json:"readOnlyPort,omitempty"`
```
— `staging/src/k8s.io/kubelet/config/v1beta1/types.go`, `kubernetes/kubernetes` `v1.34.1`

*"Default: 0 (disabled)"* is a **restricting** default and admissible under §10.4; the sentence above
it is the §3.4 quote and is a **description of what the port serves**, not a position on where it may
be reached from. So `10255` rests on a shipped default and nothing else. **It joins `5432` and `5984`
in the weak tier, which is now three rows.**

**The generated CLI page says the opposite, and §10.4's one-way rule disposes of it without
adjudication.** **[measured]** `kubernetes/website` `release-1.34`'s
`command-line-tools-reference/kubelet.md` — the page §3.4 cites, and cites as *"verified directly
against the rendered page"* — still carries `--read-only-port int32  Default: 10255`. Two of the
owner's own artefacts disagree, which is **exactly** §13.3's Elastic case (`localhost` in the tarball,
`0.0.0.0` in the Docker image) in a second instance: the **restricting** one attests and the
**permissive** one is silent, and no judgement about which page is authoritative is required. §13.3
called that case *"the cleanest demonstration that §10.4's one-way rule does real work"*; it now has
company, and the company arrived from a different project and a different artefact class.

> **By-catch, routed rather than decided — §3.4's kubelet defaults are quoted from a page the shipped
> source contradicts, and it bears on Claim 1 rather than on a footing.** §3.4 quotes the same
> generated reference for `--anonymous-auth  Default: true` and `--authorization-mode string
> Default: "AlwaysAllow"`, and §3.1's *"why"* cell for `10250` rests on both. **[measured]**
> `pkg/kubelet/apis/config/v1beta1/defaults.go` at `v1.34.1` sets
> `obj.Authentication.Anonymous.Enabled = ptr.To(false)` and
> `obj.Authorization.Mode = …KubeletAuthorizationModeWebhook`, and `cmd/kubelet/app/options/options.go`
> registers both flags with the **already-defaulted struct value** as the flag default
> (`fs.BoolVar(&c.Authentication.Anonymous.Enabled, "anonymous-auth", c.Authentication.Anonymous.Enabled, …)`),
> so the flags inherit the config defaults rather than contradicting them. If the kubelet as shipped
> does not admit anonymous commands, `10250`'s **Claim 1** grounds are in question — which is a row
> question, priced as a removal, and not this ticket's to answer. Routed to
> [#83](https://github.com/winniel123/verge-asm/issues/83), which **blocks
> [#12](https://github.com/winniel123/verge-asm/issues/12)**.

> **Amended by §18** ([#88](https://github.com/winniel123/verge-asm/issues/88)). **Both placements in
> this subsection move up, and the negative above is discharged rather than withdrawn.** *"The owner
> states **no position** on `10255` anywhere that was retrieved"* was already narrowed by §17.6 to a
> statement about the two documents it names; §18 rules that the owner's position, retrieved by
> [#79](https://github.com/winniel123/verge-asm/issues/79) from Kubernetes' security documentation,
> **reaches both ports** — so `10255/tcp` goes to the **explicit prohibition** tier and `10250/tcp`
> joins it from scoping. The `Used By: Self, Control plane` table cell above stops carrying `10250`'s
> **position** and carries its **membership**, which is the job it fits (§18.6). `readOnlyPort`'s
> restricting default is unchanged and is no longer the row's only footing. **No `(port, transport)`
> pair moves**, and both cells are conditional on
> [#83](https://github.com/winniel123/verge-asm/issues/83).

### 16.6 `4369` epmd — the placement §13.7 called clerical, and the reason it is not

§13.7 read this row as the easiest of the seven: RabbitMQ's *"these ports should not be publicly
exposed"* names `4369` alongside `25672`, `25672` is already in the scoping tier, so `4369` follows
its cell-mate. **The tier is right and the warrant is wrong, and the defect is the one §10.5 exists to
catch.**

**RabbitMQ does not own epmd.** §10.5 defines the owner as *"the party that designed the protocol, or
that authors the reference implementation, speaking about the thing it designed or wrote."* The
Erlang Port Mapper Daemon is Ericsson's: it is specified and implemented in Erlang/OTP
(`erts/epmd/src/epmd.c`), and RabbitMQ is a **consumer** of it that ships it. RabbitMQ's sentence
about `4369` is therefore a different party speaking about its own use of somebody else's daemon —
**the same shape as kubernetes.io speaking about etcd**, which §13.7 spotted and this row's entry in
the same paragraph did not. Under §10.5 and §12(c) it corroborates under §2.3 and is **never sole
grounds**.

**The distinction is per-port, not per-sentence, and that is the general point.** One RabbitMQ
sentence names two ports; RabbitMQ **owns** `25672`, its own inter-node port, and **does not own**
`4369`. So the same sentence carries one of its two ports and cannot carry the other. This is
[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)'s finding
seen from the ownership side, and it is worth stating as a rule:

> **Ownership is tested per port, not per sentence.** Where one attestation names several ports, each
> port must be tested against §10.5 separately. A sentence that is an owner's for one of the ports it
> names may be a third party's for another, and the corroborator rule then applies to that port alone.

**What actually carries `4369` is Erlang/OTP's own warning, which this note had never cited.**
**[measured]** `system/doc/reference_manual/distributed.md`, `erlang/otp` `OTP-29.0.5`:

> "Starting a distributed node without also specifying `-proto_dist inet_tls` will expose the node to
> attacks that may give the attacker complete access to the node and by extension the cluster. **When
> using insecure distributed nodes, make sure that the network is configured to keep potential
> attackers out.**"

That is the owner directing that the distribution mechanism sit inside a network the operator
controls — trusted-network scoping in §2.2's second form. And epmd's own reference page carries a
consistent, weaker sentence in a section built for the question:

> "The `epmd` daemon accepts messages from both the local host and remote hosts. However, only the
> query commands are answered (and acted upon) if the query comes from a remote host." … "**To
> restrict access further, firewall software must be used.**"
> — `erts/doc/references/epmd_cmd.md`, `OTP-29.0.5`, section *Access Restrictions*

**`4369` goes to the trusted-network scoping tier**, carried by Erlang/OTP with RabbitMQ demoted to
corroboration. Its shipped default does not help in either direction: **[measured]** epmd listens on
all interfaces unless `-address` or `ERL_EPMD_ADDRESS` is given, both opt-in, so the default is
permissive and **silent** under §10.4. §16.9 flags how thin this is.

> **Second by-catch, routed rather than decided.** §3.1's *"why"* cell for `4369` reads *"Cluster-discovery
> channel whose only protection is a shared Erlang cookie."* **[measured]** The magic cookie protects
> the **distribution handshake**, not epmd: `epmd_cmd.md`'s *Access Restrictions* section describes
> epmd answering port queries and name listings to any remote host with no cookie in the exchange at
> all. The cell describes the wrong port's protection, which is §10.7's *"why"*-cell shape — the row's
> grounds, not its colour — and §3.1 is not this section's to edit. Routed to
> [#84](https://github.com/winniel123/verge-asm/issues/84), which does **not** block #12.

### 16.7 The ruling — the table restated, with its coverage in the table

> **§2.2's footing table now places 26 of the 37 `(port, transport)` pairs.** The **explicit
> prohibition** tier is `6379`, `11211` (tcp and udp), `3306`, `1433`, `9200`, `873`, `445`, `623`,
> `9042`, **`9300`**, **`2379`** and **`2380`**. The **trusted-network scoping** tier is
> `27017`/`27018`/`27019`, `2049`, `2181`, `25672`, `2376`, **`2375`**, **`4369`** and **`10250`**.
> **The weak tier is `5432`, `5984` and `10255`, and it is three rows.**
>
> **The eleven uncovered pairs are uncovered by design, and the table now says so rather than leaving
> the next reader to count.** Class B's seven (`23`, `21`, `512`, `513`, `514`, `5900`, `6000`) and
> `69/udp` rest on §2.2's **first** form — a specification, IANA's registry, or OpenBSD's deletions —
> which is not what this table discriminates. `139/tcp`, `137/udp` and `138/udp` are carried by the
> same Microsoft sentence as `445` and sit inside that row's cell. 26 + 11 = 37.

**What changed and what did not.** Two placements are clerical (§16.2), one confirms a reading §13.7
proposed (`9300`), one refutes the reading §13.7 proposed (`2379`/`2380`), one is new evidence nobody
had looked for (`10255`), and one is right for a reason nobody had checked (`4369`). **The weak tier
grows by one and shrinks by nothing**, which is the result the map's curation patch needs to hear:
the watch list was missing a member, and it was not either of the two §13.7 predicted.

**The coverage sentence is the deliverable §13.7 actually asked for.** §13.10 recorded that the
count *"was done by hand against §3's tables and is the kind of thing that goes wrong"*. A table that
carries its own extent cannot go wrong that way: the arithmetic is in the artefact, and a reader who
disagrees is disagreeing with a claim rather than reconstructing one.

> **Amended by §18** ([#88](https://github.com/winniel123/verge-asm/issues/88)). **The blockquoted
> table above is superseded in three cells and stands in every other**, including its coverage
> sentence. `10250` and `10255` are both in the **explicit prohibition** tier, the scoping tier is
> **9 pairs**, and **the weak tier is two rows** — so *"The weak tier grows by one and shrinks by
> nothing"* is superseded: it grew by one and has now shrunk by that one, on evidence rather than on
> a correction. §18.6 carries the restated table. The coverage arithmetic is untouched at **26 of
> 37**, because a re-tiering moves a pair between rows of this table and never in or out of it.

### 16.8 Every dependent figure, checked rather than asserted

| Where | Was | Is |
|---|---|---|
| §1 pair count | 37 | **37, unchanged.** No row is added or removed |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 = 37 | **unchanged.** No row changes class, because no row's claim changes |
| §2.2 footing table — coverage | 19 of 37 pairs (§13.7) | **26 of 37**, with the remaining 11 named in the table as out of subject |
| §2.2 footing table — prohibition tier | 10 pairs (9 labels) | **13 pairs.** `+9300`, `+2379`, `+2380` |
| §2.2 footing table — scoping tier | 7 pairs | **10 pairs.** `+2375`, `+4369`, `+10250` |
| §2.2 footing table — **weak tier** | **2 rows** (`5432`, `5984`) | **3 rows.** `+10255` |
| §13.4's *"the weak tier is two rows"* | 2 | **superseded — three.** §13.4 was correct over the 19 pairs it had placed |
| §4.5 *the list's weakest row* | `5432/tcp` | **unchanged.** `10255` joins the tier and does not displace §4.5's row: `5432`'s upstream states *no position at all*, while `10255`'s owner is silent only on network placement |
| §12.2's artefact count | ten artefacts, nine self-declarations (§13.6) | **unchanged.** Nothing retrieved here is a configuration artefact in §12.2's sense — `THREAT_MODEL.md` and the man-page sources are prose, and `types.go` is the config API's definition rather than a config file |
| §13.1's *"three rows have no artefact to read"* | `1433`, `445`, `623` | **unchanged**, and untouched by this pass |
| §6.1 containment arithmetic | 28 in the hot set + 4 + 5 = 37 | **unchanged** |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | `verge-core = frequency-set ∪ sensitive-list` | **unchanged** — no member enters or leaves |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version, and the `Break` | — | **not triggered.** `sensitive-port-reached-from-internet`'s content is byte-identical, so no evaluation is made non-comparable |
| §13.7's *"seven listed rows have no footing cell"* | open, routed to #76 | **discharged.** All seven are placed |
| §13.7's *"the weak tier goes from two rows to four"* | predicted | **refuted.** etcd goes to the prohibition tier; the weak tier goes to **three**, on `10255` |

**Outside this note.** The map's *how the tiered port sets are curated* patch names the weak tier as
the curator's watch list and must record **three** rows rather than two.

**[ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8 is amended,
and it was already stale before this ticket touched it.** Its silent-de-attestation blockquote
enumerates the watch list as *"5432/tcp, 5984/tcp and 9042/tcp"* — but `9042` left the weak tier at
§12.7 (#69) and **nobody propagated the change to the ADR**, which is the exact failure
`docs/agents/domain.md` records for #27's `Vantage class` sentence: a clause that goes on saying
something a later decision withdrew, because nobody searched for the other place it lived. The
amendment re-enumerates the list as **`5432/tcp`, `5984/tcp` and `10255/tcp`** and records that the
count is coincidentally three again, so a reader comparing counts rather than members would see
nothing move.

**No ADR is added.** Every ruling here applies §10.4, §10.5, §12(b) and ADR-0037 as they stand, and
§16.6's per-port ownership rule is a statement of what §10.5 already requires rather than a new
decision.

### 16.9 Thin ground, flagged per the standing rule

**`10250`'s scoping cell is the thinnest placement in the table, and it is thinner than every other
member of its tier.** Every other scoping row has a **sentence** naming a network — MongoDB's *"only
accessible on trusted networks"*, ZooKeeper's *"behind a firewall"*, NFS's *"on a trusted physical
network between trusted hosts"*, RabbitMQ's *"not exposed to the public Internet"*, Docker's *"a
trusted network or VPN"*. `10250`'s is a **table cell**, `Used By: Self, Control plane`, which names
the port's clients rather than its permitted network, and sits under a page framed as *"useful to be
aware of"* for firewall planning. It is admitted because §10.3's boundary limb asks the owner to name
the boundary and this does name one. **The criterion that would change the verdict:** a kubernetes.io
sentence placing the kubelet API on an untrusted network as a supported deployment would remove it
from the tier outright; the absence of any stronger sentence than the table cell would justify moving
it to the weak tier, on the argument that `Used By` is a §12(b) **label**. That argument was
considered and lost only narrowly.

**`4369`'s cell rests on a sentence that does not name the port, and that is a real gap.**
Erlang/OTP's *"make sure that the network is configured to keep potential attackers out"* is about
distributed nodes and the distribution transport; epmd is the mechanism's registry rather than the
transport, and the page that **is** about epmd declines to prohibit anything, saying only that a
firewall is what further restriction takes. A reader who says Erlang/OTP has stated a position about
the distribution port and no position about `4369` has a live argument, and on that reading `4369`
has **no admissible owner attestation at all** — RabbitMQ's and CouchDB's sentences are both
non-owners' — which is §10.6's finding for `161/udp` in a second instance, and would be priced as a
row removal rather than as a footing. **The criterion that would settle it:** an Erlang/OTP sentence
naming `4369` or epmd specifically, in either direction. None is in the current bytes.

**`2379`/`2380` rest on a document that is one major version old and has existed for three months.**
`THREAT_MODEL.md` appeared 2026-05-19 and is absent from the `3.5` and `3.6` lines, which are what
most deployments run and what §3.4 still cites. The finding is honest at `v3.7.x` and it is exposed
to exactly [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's
silent de-attestation from the other direction: a young document can be withdrawn as easily as a
default can be flipped, and this one was written to bound CVE triage rather than to guide operators.
**The criterion that would change the verdict:** removal or material weakening of the Network
Boundary paragraph, at which point both ports fall to the **weak tier** on
`listen-client-urls: http://localhost:2379` alone — which is precisely §13.7's prediction, deferred
rather than refuted. Both ports are therefore worth watching even though they are not in the weak
tier.

**The `9300` placement is the most confident of the seven and is still an inference of one step.**
The owner writes *node*, the owner defines the node as its two interfaces on the same page, and the
setting the warning concerns governs both — but the owner does not write `9300`. The step is smaller
than `4369`'s and is not zero.

> **Amended by §18** ([#88](https://github.com/winniel123/verge-asm/issues/88)). **The first flag
> above is discharged.** `10250`'s cell no longer rests on a table cell for its network position — it
> rests on the owner's *"The Kubernetes API, kubelet API and etcd are not exposed publicly on
> Internet"*, and the `Used By` cell now supplies **membership** instead, which is what a table of
> ports and their clients is fit for. The alternative this subsection recorded as *"considered and
> lost only narrowly"* — demoting `10250` to the weak tier on the argument that `Used By` is a §12(b)
> label — is now moot in the other direction. **A new flag opens in its place at §18.7**: `10255` is
> the thinnest member of the **prohibition** tier, and both kubelet cells rest on a **checklist
> item** in a documentation release branch, which is this subsection's `2379`/`2380` volatility
> argument arriving on two more ports.

### 16.10 Retrieval method and hazards, recorded per §9.5, §11.9, §12.9 and §13.10

**Every artefact was read as shipped bytes at a named tag or release branch, with paths resolved from
the repository tree.** No rendered documentation site was used as a source of record; where a rendered
page is quoted (`kubelet.md`, `ports-and-protocols.md`) it is quoted **from its source file in the
project's own docs repository at a release branch**, which is the shipped byte of the page.

- **The document that decided this ticket is not where the row's citation points, and only a tree
  listing found it.** §3.4 cites `etcd.io/docs/v3.5/op-guide/security/` for etcd; the position is in
  `THREAT_MODEL.md` at the repository root. It was found by listing the tree at `v3.7.1` and reading
  what was there, which is [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)
  working exactly as §13.5 described it — *"a retrieval scoped to the rows a table already holds can
  confirm those rows and can discover nothing."* A session that fetched the cited URL would have
  confirmed §13.7's hypothesis and been wrong.
- **A generated reference page can be stale against the source that generates it, and the corpus now
  has an instance.** `kubelet.md` at `release-1.34` says `--read-only-port … Default: 10255`; the
  shipped `types.go` at `v1.34.1` says `Default: 0 (disabled)`. §3.4's parenthetical *"the 10255
  default verified directly against the rendered page"* names the hazard it walked into: **verifying
  against a rendering is not verifying**, and §11.9 had already recorded the same class for net-snmp's
  `man2html` pages.
- **The `.sample` trap from §13.6 has a sibling in file *purpose*, and it was met.** etcd's
  `THREAT_MODEL.md` was written to scope CVE triage, and reading the purpose as the answer would have
  discarded a prohibition — the mirror of reading `.sample` as the answer and discarding a default.
  §16.3 rules on the sentence and records the purpose argument in full rather than suppressing it.
- **Ownership was checked per port rather than inherited from a cell-mate**, which is what turned
  `4369` from clerical into a retrieval. The check that catches it is cheap and was not previously
  part of the method: for each port a sentence names, ask who wrote the sentence and who wrote the
  daemon.
- **Erlang/OTP tags are not semantic-version tags** — the repository uses `OTP-29.0.5`, and a guessed
  `OTP-28.1.2` returns 404. The tag list was read before any path was resolved, per §13.10's rule that
  paths come from the tree rather than from expectation.
- **`10255`'s absence was verified positively rather than assumed.** A search of `kubernetes/website`
  for `10255` returns eight files, of which the English-language substantive ones are the CLI
  reference, `kubelet-config-file.md` and a standalone-kubelet tutorial; none states a network
  position, and `ports-and-protocols.md` — the page that would carry one — does not list the port.
  Per §11.8, a negative retrieval is a verdict, and per ADR-0037 limb 3 it is a verdict about what was
  read.
## 17. The class audit — which negatives rest on specifications alone

Wayfinder ticket [#79](https://github.com/winniel123/verge-asm/issues/79), applying
[ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md)'s rule that **a
specification's silence is not the owner's silence** to every negative this note is standing on. Each
was established before that rule existed. This section **amends §2.7, §4.5, §4.6, §10.3, §11.8 and
§12.6 by reference**: earlier text is left standing and marked, per the name-and-withdraw convention,
and where §17 and an earlier section disagree, **§17 governs**. The general rule it establishes is
[ADR-0046](../adr/0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md).

**Two sibling sections landed while this one was being written, and both touch it.**
[#82](https://github.com/winniel123/verge-asm/issues/82)'s §15 gave §2.4's determinacy gate a source
rule — [ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md), *a
convention is evidenced by placement, never by catalogue* — and walked the nine **listed** rows resting
on convention. It did **not** walk the **excluded** rows' squats, which is where §17.8's residue lives.
And [#76](https://github.com/winniel123/verge-asm/issues/76)'s §16 placed the footing table's seven
uncovered rows and **moved `10255/tcp` into the weak tier**, which created a fourteenth sole-ground
negative after this audit's population had been fixed. It is swept in §17.6 rather than left out, and
what that costs the *sweep terminates* claim is stated in §17.8 rather than smoothed.

**Headline result, stated first.**

> **No row moves. The list stays at 37 `(port, transport)` pairs and no footing cell is edited.**
> Fourteen load-bearing negatives are enumerated; **six are overdetermined** — a gate no document can
> reach already refuses the row — **five were swept**, **one had its class list already exhausted**, and
> **two are the named residue**. `161/udp` does **not** re-open: the IETF's operational class, which #66
> never opened, contains no placement sentence and contains the opposite. `5672`/`15672`'s negative
> **strengthens into a positive**. Two previously-unopened owner documents do carry the sentence the
> note says does not exist — for `9092/tcp`, **which is in no release and on no published page**, and
> for `10255/tcp`, **which names the category and not the port**. Both are questions about the standard
> rather than about the row, and both are handed on rather than answered here.

### 17.1 The population, enumerated rather than asserted

ADR-0040's disclosure rule binds this section's own output, so the corpus is listed. A negative is
**load-bearing** where this note cites an absence as a reason. *Classes* are the three ADR-0040 §1
names — the defining **specification**, the **operational or deployment** recommendation, and the
**implementation guidance** — plus §2.2's third form, the **shipped default**.

| # | The negative, and where it lives | Classes searched | Classes that exist and were **not** searched | Sole ground? | Verdict |
|---|---|---|---|---|---|
| 1 | **`161/udp`** — no first-party sentence places SNMPv1/v2c inside a boundary (§11, the negative that **removed a row**) | specification ×9 (RFC 3411/3412/3413/3414/3417/3584/2570/1157/6353) · applicability statement (RFC 3410) · implementation ×5 (net-snmp `snmpd(8)`, `snmpd.conf(5)`, FAQ, wiki, `EXAMPLE.conf.def`) · distributor default (Debian) | **operational** — the IETF's OPS-area documents | **Yes** | **Swept (§17.3). Survives, bounded** |
| 2 | **`111/tcp`** — *"we could not find anyone entitled to say it isn't"* (§2.7, §9.1) | specification (RFC 1833, RFC 5531) · implementation (`rpcbind(8)`, `configure.ac`, `rpcbind.socket`) · distributor (Debian) · third-party operational (Red Hat, declined on §2.3) | none material — ONC RPC has no IETF deployment document | **No** — §10.1 Step 1 refuses Claim 1 independently, and §10.4.3's remedy route is a second | **Bounded on arrival** |
| 3 | **`389/tcp`** — RFC 4513 does not mention `ldaps://` or `636` (§9.2) | specification ×5 (RFC 4510/4511/4513/4516/2830) · implementation (OpenLDAP Administrator's Guide, `slapd(8)`) · vendor (Microsoft LDAP signing) | LDAP has no IETF deployment BCP | **No** — OpenLDAP names *"the global Internet"*; §10.8 makes Class B structurally unavailable | **Bounded on arrival** |
| 4 | **`79/tcp`** — no implementation states a position (§9.3.5) | specification (RFC 1288) · implementation ×5 (OpenBSD, FreeBSD, GNU inetutils, Fedora, Debian) | — | **No** — fails determinacy against RFC 4146, and §10.1 Step 1 | **Bounded on arrival** |
| 5 | **`9092/tcp`** — *"upstream declines to take any network posture"* (§4.6, §10.3, §12.6) | project documentation (`security-overview`) · shipped default (`config/server.properties`) | **the project's own security-model document** | **Yes** | **Swept (§17.5). A sentence exists and is not shipped** |
| 6 | **`5672`/`15672`** — the prohibition names 4369 and 25672, not these (§4.6, §10.3, §12.6) | project documentation ×3 (networking, access control, clustering) · example config (`rabbitmq.conf.example`) | **the deployment class — the production checklist** | **Yes** | **Swept (§17.4). Survives, and now points the other way** |
| 7 | **`5432/tcp`** — PostgreSQL states **no position at all** on network placement (§4.5, the note's weakest row) | manual ×4 (`runtime-config-connection`, `ssl-tcp`, `client-authentication`, `server-start`) · shipped bytes ×3 (`postgresql.conf.sample`, `guc_tables.c`, `pg_hba.conf.sample`) | **none — PostgreSQL ships one manual, and #70 read both classes** | **Yes** | **Bounded — the class list is exhausted (§17.6)** |
| 8 | **`5984/tcp`** — the *"do not expose"* warning covers the Erlang distribution port, not 5984 (§2.2) | project documentation ×2 (cluster setup, HTTP config) · shipped bytes (`default.ini`, `local.ini`) | **the security introduction** | **Yes** | **Swept (§17.6). Survives; near-miss recorded** |
| 9 | **`5985`/`5986`** — no first-party Microsoft sentence prohibits internet exposure of WinRM (§4.6) | vendor guidance (`winrm-security`) · vendor deployment, cloud-scoped (Azure) | the DMTF WS-Management specification | **No** — remote administration is the express purpose, and Microsoft's own sentence refutes the Claim 2 argument | **Bounded on arrival** |
| 10 | **`3389/tcp`** — no first-party non-Azure prohibition was found (§4.6) | vendor deployment, cloud-scoped (Azure) · third-party (CISA) | Microsoft's on-premises hardening guidance | **No** — express purpose; GCP ships it open to `0.0.0.0/0` | **Bounded on arrival** |
| 11 | **`5601/tcp`** — Elastic states no prohibition (§4.6, §12.6) | project documentation (Kibana settings) · shipped default (`kibana.yml`) | Elastic's deployment guidance for Kibana | **Partly** — the second ground is a squat on `esmagent`, and ADR-0042's liveness test has **never been run on it** | **Exposed, not swept — §17.8** |
| 12 | **`8500/tcp`** — the position is only that external access *"should be considered"* (§4.6, §12.6) | project documentation ×3 (security model, ACL config, ports) · no configuration file ships | HashiCorp's deployment guidance | **Partly** — same shape: the second ground is a squat on `fmtp`, untested under ADR-0042 | **Exposed, not swept — §17.8** |
| 13 | **`1433`, `445`, `623`** — *"no shipped configuration artefact exists"* (§13.1) | — | — | **No** — it is a negative about an **artefact's existence**, not about an owner's silence, and all three rows rest on prose in the prohibition tier | **Not a class negative** |
| 14 | **`10255/tcp`** — *"The owner states **no position** on `10255` anywhere that was retrieved"* (§16.5, [#76](https://github.com/winniel123/verge-asm/issues/76)) — **created after this population was fixed** | project documentation ×2 (`ports-and-protocols.md`, `kubelet-authn-authz.md`) · shipped bytes (`types.go`, `defaults.go`) · generated CLI reference | **the project's security documentation** | **Yes** — it is the weak tier's third row and rests on a restricting default alone | **Swept (§17.6). A category prohibition exists in the owner's voice, and whether it reaches the port is a footing question** |

### 17.2 The filter, and it is what makes the sweep affordable

Six of the fourteen are **overdetermined**, and that is a property of an evidence standard with more
than one gate rather than good luck.

> **A negative is exposed to ADR-0040's class sweep only where it is the row's *sole* ground.** Where
> the row is also refused by **determinacy**, by the **closed claim set** (§10.2), or by an **owner
> sentence naming the internet as supported** (§10.3), a document retrieved from an unopened class
> changes the note's footing and cannot change its verdict. None of those three gates is a silence, so
> none of them has a class list to enumerate.

**The arithmetic.** Fourteen negatives. **Six are un-exposed** — rows 2, 3, 4, 9, 10 and 13. **Eight
are exposed**, and they divide three ways: **five were searched** (rows 1, 5, 6, 8 and 14), **one had
its class list already exhausted before this ticket** (row 7, PostgreSQL — §17.6), and **two are the
residue** (rows 11 and 12, whose second ground is an untested determinacy call — §17.8).

The classes are still **enumerated** for all fourteen, because that is what ADR-0040 actually requires
and it costs a table. Only the five were **searched**.

### 17.3 `161/udp` — the operational class, opened for the first time, and it points the other way

#66 read ten RFCs and five net-snmp artefacts. **[measured]** Every one of the ten is a
**specification**, a **coexistence transition document** (RFC 3584) or an **applicability statement**
(RFC 3410); the net-snmp five are **implementation guidance** and a **shipped example**. The IETF's
**operational** class — the OPS-area documents that tell an operator how to run a managed network —
was never opened. #66's own re-admission criterion is *"a first-party sentence from the IETF or from
net-snmp placing SNMPv1/v2c inside a network boundary"*, and if one exists this is where it is.

Three documents were retrieved as bytes and searched directly.

| Retrieved | What it is | Did it place SNMP? |
|---|---|---|
| **RFC 3512** | *Configuring Networks and Devices with SNMP* (Informational, April 2003) — the SNMP family's own deployment guide | **No** |
| **RFC 3871** | *Operational Security Requirements for Large ISP IP Network Infrastructure* (Informational, September 2004) | **No — and §2.2 says the opposite** |
| **RFC 4778** | *Operational Security Current Practices in Internet Service Provider Environments* (Informational, January 2007) | **No — and it is a survey, not a position** |

**RFC 3512 is the closest thing the SNMP family has to a deployment BCP, and its security sections are
about everything except placement.** §6.2 *Secure Agent Considerations* is about shipped community
strings — *"Vendors should not ship a device with a community string 'public' or 'private'"* — and
about SNMPv3 being *"recommended for all network management applications"*. §6.3 is about
authentication-notification defaults, §6.4 about MIB object sensitivity. **[measured]** the strings
`management network`, `segregat`, `firewall`, `separate network`, `isolated network`, `private
network`, `untrusted network`, `public network` and `out of band` occur **five times** in the whole
document, and not one describes where an SNMP agent belongs: two are about trap delivery, one is the
title of an ITU-T reference in the bibliography, and two are about VPN forwarding as a MIB example.

**RFC 3871's SNMP paragraph is the boilerplate, and the boilerplate is a version statement.**

> "Furthermore, deployment of SNMP versions prior to SNMPv3 is NOT RECOMMENDED. Instead, it is
> RECOMMENDED to deploy SNMPv3 and to enable cryptographic security. It is then a customer/operator
> responsibility to ensure that the SNMP entity giving access to MIB objects is properly configured"
> — [RFC 3871](https://www.rfc-editor.org/rfc/rfc3871.txt), §9, *Security Considerations*

This is the strongest unscoped IETF sentence about SNMPv1/v2c anywhere in the corpus and **it is not a
placement sentence**. It is §11.5's structure exactly: the IETF's stated remedy is the **version**, and
the version is reached on `161/udp` itself (RFC 3417 §3.2). RFC 3410 §8.2's *"framework of choice"* is
the same body saying the same thing nine years earlier, and #66 held it and correctly declined it. Its
second clause hands the boundary to the operator, which is the delegation §11.2 refused.

**And the decisive sentence points the other way.** RFC 3871 §2.2, introducing the in-band management
requirements:

> "There are many situations where in-band management makes sense, is used, and/or is the only option.
> The following requirements are meant to provide means of securing in-band management traffic."
> — [RFC 3871](https://www.rfc-editor.org/rfc/rfc3871.txt), §2.2, *In-Band Management Requirements*

The IETF's operational-security document enumerates in-band management's disadvantages — including
*"Since public interfaces/channels are used, it is possible for attackers to directly address and reach
the device"* — and then specifies requirements for **doing it anyway**. Its out-of-band section (§2.3)
is a parallel set of requirements for a different topology, introduced by *"trade-offs that must be
weighed in considering which is appropriate to a given situation"*. That is a conditional deferring to
the reader's architecture, which is the shape §9.3.1 refused for finger, §4.4 refused for 6443 and
§11.2 refused for RFC 3410.

**RFC 4778 is the frequency trap in the class most likely to contain it.** Its title is the most
placement-shaped in the corpus and it is a **survey of what operators already do**:

> "In all large ISPs that were interviewed, HTTP management was never used"
> — [RFC 4778](https://www.rfc-editor.org/rfc/rfc4778.txt), §2.3

> "In instances where SNMP is used, some legacy devices only support SNMPv1, which then requires the
> provider to mandate its use across all infrastructure devices for operational simplicity. SNMPv2 is
> primarily deployed since it is easier to set up than v3."
> — [RFC 4778](https://www.rfc-editor.org/rfc/rfc4778.txt), §2.4

*Primarily deployed.* This is *frequency is not a position* in the source's own words — the shape #73
met in RFC 8422 §5.1.1 and §14 refused again on `7000` — and it arrives in the **deployment class**,
which is where the ticket predicted it would. **A deployment document is where frequency arguments
live, and the operational-practices genre is a measurement by construction.** Recorded so that the next
session reaching for an operational document knows to check its genre before its content.

**One scope note that would have mattered had anything been found.** RFC 3871's own abstract addresses
*"vendors"* on behalf of *"large Internet Service Provider (ISP) IP networks (routers and switches)"* —
a body addressing a constituency, which is ADR-0035 §7's scope-weakness shape. A placement sentence
found there would have been scoped rather than unscoped and would have needed that argued. None was
found, so it does not arise.

> **Ruling: `161/udp` does not re-open.** #66's negative is **bounded** rather than merely unsearched:
> the class it never opened has now been opened, and it contains no placement sentence and one sentence
> pointing the other way. §11.8's criterion is unchanged and now has a shorter boundary — a placement
> sentence would have to come from a class outside {specification, applicability statement, operational
> recommendation, reference implementation}, and the IETF has no fifth. **Re-admission, if it ever
> came, would be [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s removal plus addition, priced
> separately, never a reversal of §11.6.**

### 17.4 `5672`/`15672` RabbitMQ — the negative strengthens into a positive

§4.6 excludes both on *"Upstream's 'should not be publicly exposed' sentence covers 4369 and 25672
specifically, not these"* — a **scoping** negative about a document already held, which §10.3 and §12.6
each re-confirmed without adding evidence. The class never opened is RabbitMQ's **deployment**
documentation: its production checklist.

> "Ports used by RabbitMQ can be broadly put into one of two categories: Ports used by client libraries
> (AMQP 0-9-1, AMQP 1.0, MQTT, STOMP, HTTP API); All other ports (inter node communication, CLI tools
> and so on). Access to ports from the latter category generally should be restricted to hosts running
> RabbitMQ nodes or CLI tools. Ports in the former category should be accessible to hosts that run
> applications, **which in some cases can mean public networks, for example, behind a load balancer.**"
> — [RabbitMQ production checklist](https://www.rabbitmq.com/docs/production-checklist), *Firewall Configuration*

Two findings, and the second is the one that matters.

1. **The scoping negative is confirmed in a second class, in the owner's own taxonomy.** *"All other
   ports (inter node communication, CLI tools and so on)"* is exactly 4369 and 25672, and it is exactly
   those that *"should be restricted to hosts running RabbitMQ nodes"*. §4.6's reading of the
   prohibition's scope was correct and now rests on two documents rather than one.
2. **The negative becomes a positive.** `5672` is AMQP 0-9-1 and 1.0; `15672` is the HTTP API. Both are
   in the **first** category, and the owner says that category *"in some cases can mean public
   networks"*. That is **§10.3's failure condition met from the other side** — *"Where the owner names
   the public internet as a supported deployment environment, Claim 3 fails however strongly a third
   party disapproves"* — and it is the [#30](https://github.com/winniel123/verge-asm/issues/30) shape a
   third time. §4.6's *"AMQP brokers are sometimes legitimately public"*, which was an assertion of ours
   and the weakest clause in that table, is **withdrawn and replaced by the owner's own sentence**.

**No row moves**, because both were already excluded. What changes is that two exclusions move from
*the prohibition does not reach them* to *the owner names their audience as possibly public*, which is
the difference §9.4 called *the difference between a gap and a finding*.

### 17.5 `9092/tcp` Kafka — the sentence exists, and it is not shipped

§4.6 excludes Kafka on *"Upstream declines to take any network posture. Its only relevant sentence is
neutral: 'security is optional - non-secured clusters are supported'"*. **[measured]** That sentence is
still exactly where §4.6 found it: `docs/security/security-overview.md` at release tag `4.3.1`, 2,068
bytes, verified against the retrieved file.

The class never opened is the project's own **security-model** document, which is a different file:

> "**Security is off by default.** A freshly-installed Apache Kafka cluster accepts unauthenticated
> `PLAINTEXT` connections on every listener and applies no authorization. This is appropriate only for
> closed test environments. Production deployments **must** explicitly configure authentication,
> authorization, and transport encryption before being exposed to any untrusted network."
> — [`apache/kafka`, `docs/security/security-model.md`](https://raw.githubusercontent.com/apache/kafka/trunk/docs/security/security-model.md), `trunk`

That is not neutral, and finding it is a **retrieval** on
[ADR-0040](../adr/0040-a-specifications-silence-is-not-the-owners-silence.md) §4's test — the claim is
about a **different document's content**, not about the **import** of one already held.

**The row does not move, on two independent grounds.**

1. **[ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 2
   governs.** *A subject the artefact names and the table lacks is a finding, and it is ticketed rather
   than admitted* — the artefact supplies the **attestation limb only**, and the claim and determinacy
   gates were not what this retrieval was scoped to answer. `7000/tcp` is the measured precedent for
   what happens when a session sweeps them in together: the strongest sentence in the corpus, sunk by a
   different gate. `9092/tcp` squats on `XmlIpcRegSvc` and §2.4's liveness test under ADR-0042 has not
   been run on that registration either.
2. **[measured] The sentence is not shipped.** `docs/security/security-model.md` is present on
   `apache/kafka` `trunk`; it is **absent from release tag `4.3.1`** (HTTP 404 on the raw path, against
   HTTP 200 for `security-overview.md` at the same tag); and
   `kafka.apache.org/documentation/security/security-model/` returns **HTTP 404**, while
   `kafka.apache.org/43/security/security-overview/` returns 200.

**The second ground is a gap in this standard rather than an application of it.** §12 ruled what an
**example configuration** attests — nothing, in either direction — and §2.2's second form reads *"the
project's or vendor's own documentation"* without saying whether that means what the project
**publishes** or what sits in its **tree**. The two readings give opposite verdicts here, and the
question has never been asked because until now no case turned on it. **The parallel is exact and it is
why the answer is probably the same:** §12's rule is that operativeness is read off what **takes
effect**, and a document no release contains and no page serves takes effect on nobody.

> **Ruling: `9092/tcp` remains excluded, and the list remains definitively 37.** §4.6's stated reason
> is **narrowed**: it is no longer *"upstream declines to take any network posture"* — upstream has
> taken one, in its repository — but *the document taking it is in no release and on no published page,
> so it does not reach §2.2's second form on the reading §12 committed this note to for configuration.*
> **The criterion that would change the verdict:** the file appearing in a release tag or on
> `kafka.apache.org`, at which point the row is a candidate whose claim and determinacy gates are
> untested. Routed to [#86](https://github.com/winniel123/verge-asm/issues/86), with the standards
> question, rather than settled here — §8 question 12.

**This does not block [#12](https://github.com/winniel123/verge-asm/issues/12)**, and the difference
from ADR-0037's routing of `7000` is worth stating. There, two subjects had passed the attestation gate
**on shipped bytes** and only determinacy stood between them and the list, so the count was genuinely in
doubt. Here the artefact fails §2.2 before any gate is reached, so the count is not in doubt under the
standard as written. If the standard changes, the row is a **new admission** under ADR-0009, priced
then.

### 17.6 The weak tier, all three rows, and one class list that was already exhausted

**[#76](https://github.com/winniel123/verge-asm/issues/76)'s §16.5 made the weak tier three rows —
`5432/tcp`, `5984/tcp` and `10255/tcp` — and the map's curation patch names the tier as the watch
list.** Every row in it rests on a **negative**, the absence of an owner prohibition, so every one is
sole-ground and every one is exposed. Do not quote a weak-tier membership from an earlier section: it
was `5432`/`5984`/`9042` before §12, `5432`/`5984` after it, and it is three again for a different
reason.

**`5432/tcp` PostgreSQL — nothing left to open, and that is the finding.** §4.5's negative is the
strongest-sounding in this note: *"PostgreSQL is the one service surveyed whose upstream documentation
states no position at all on network placement."* Its owner has **two** document classes — one manual
and its shipped bytes — and #70 read both, at two release branches. There is no deployment BCP because
there is no standards body, and no separate implementation guidance because the implementation and the
specification are the same project. **§4.5's disclosure was already complete before ADR-0040 existed; it
simply could not say so.** It can now: the corpus is the manual and the shipped configuration, the
result is empty, and the smallest extension that could change it is a PostgreSQL manual page that does
not exist today.

**`5984/tcp` CouchDB — the class was opened, and the near-miss is recorded rather than smoothed.**
§2.2's negative is that CouchDB's *"do not expose"* warning covers the **Erlang distribution port, not
5984**. CouchDB's security introduction was never read, and it contains the corpus's nearest miss:

> "If you are in a production environment, however, you need to reconsider. Will your CouchDB instance
> communicate over a **public network**? Even a LAN shared with other collocation customers is public.
> There are multiple ways to secure communication between you or your application and CouchDB that
> exceed the scope of this documentation. CouchDB as of version 1.1.0 comes with SSL built in."
> — [CouchDB documentation, *Security*](https://docs.couchdb.org/en/stable/intro/security.html), §1.5.1.2

Read carelessly this is §10.3's failure condition — the owner contemplating a public network. It is
not, on the sentence's own next clause: **CouchDB defines what it means by *public* in the following
sentence, and it means a shared LAN.** *"Even a LAN shared with other collocation customers is public"*
makes *public* the opposite of *exclusively yours*, not a synonym for *the internet*. This is
[#46](https://github.com/winniel123/verge-asm/issues/46)'s truncated-conditional finding in the smallest
possible space — a term lifted out of the sentence that gives it its only operative meaning — and it is
the same error §11.2 caught on `administrative domain`.

And the remedy the passage offers is **SSL**, which is a §10.4.3 remedy that stops short of the port and
bears on Claim 2 rather than on Claim 3. The row rests on Claim 3 plus the restricting default, and
neither is touched.

**`10255/tcp` kubelet read-only — the class was opened and it is not empty, and the finding is reported
rather than applied.** §16.5's negative is fresh and it is precise: **[measured]** the port appears
nowhere in `ports-and-protocols.md`, `kubelet-authn-authz.md` says nothing about network placement, and
the row rests on `readOnlyPort`'s *"Default: 0 (disabled)"*. The class those two documents do not
belong to is Kubernetes' **security documentation**, and it contains this:

> "- [ ] The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet."
> — [`kubernetes/website`, `content/en/docs/concepts/security/security-checklist.md`](https://raw.githubusercontent.com/kubernetes/website/release-1.34/content/en/docs/concepts/security/security-checklist.md), `release-1.34`, *Network security*

> "The kubelet API access should be restricted and not exposed publicly, the default authentication and
> authorization settings, when no configuration file specified with the `--config` flag, are permissive."
> — same document, *Authentication & Authorization*

**This is the owner naming a boundary**, in the owner's own voice, about the owner's own component —
§10.3's boundary limb and, in the first quote, a prohibition rather than a scoping. `10255/tcp` serves
the kubelet API read-only and without authentication, which §3.4's own quote for the row states.

**Two reasons it is reported here and not applied.**

1. **It would move a footing cell, and the footing table is
   [#76](https://github.com/winniel123/verge-asm/issues/76)'s.** Promoting `10255` out of the weak tier
   changes §2.2's tier counts and §16.7's restated table, both written this week by another pass. A
   negative swept in one lane may not silently re-tier a row placed in another.
2. **The honest counter-argument is real and it is §2.3's.** Neither sentence names `10255` by number;
   both name *"the kubelet API"* as a **category**, and §2.3's rule is that a source attesting a
   category and not a port *"is therefore excellent corroboration and useless as the sole grounds for
   any individual row"*. What is different here — and it is the whole of the argument for promotion —
   is that §2.3's refusal was aimed at **CISA and the cloud providers**, parties that do not own the
   protocol, whereas Kubernetes both owns the kubelet **and** defines what *the kubelet API* contains.
   Whether an **owner's** category statement reaches a port the owner has not numbered is a question
   §2.3 has never had to answer, because until now every category statement in the corpus came from a
   non-owner.

> **`10255/tcp` stays in the weak tier today, and its footing is disclosed as *under-stated rather than
> absent*.** §16.5's *"the owner states no position on `10255` anywhere that was retrieved"* is
> **withdrawn as a statement about the owner** and stands as a statement about the two documents it
> names. **The criterion that would move the cell:** a ruling that an owner's category statement reaches
> its own unnumbered members — which is a question about §2.2 and §2.3, not about kubelet. Reported to
> the curator rather than ticketed, because it belongs to a pass that is running now.

> **All three weak-tier rows survive their class sweep as *rows*, and no row moves.** What changes is
> that the tier's disclosure can now name a **searched corpus with a stated boundary** rather than an
> unexamined silence, which is what ADR-0040 requires of a surviving weakness — and that one of the
> three has a candidate footing on the record.

> **Amended by §18** ([#88](https://github.com/winniel123/verge-asm/issues/88)). **The candidate is
> applied and the question is answered: an owner's category statement *does* reach its own unnumbered
> members**, on §18.1's three limbs. `10255/tcp` leaves the weak tier for **explicit prohibition**,
> and `10250/tcp` comes with it. Two consequences for this section. *"All three weak-tier rows
> survive their class sweep as rows"* stands — the row survives; it is the **footing** that moves.
> And **the fourteenth sole-ground negative this ticket recorded retires rather than being answered**:
> `10255` no longer rests on an absence, so under
> [ADR-0046](../adr/0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
> limb 1 it leaves the exposed population, and §17.8's table-state qualifier fires in the
> **disarming** direction for the first time. The weak tier, and therefore the curator's watch list,
> is **`5432/tcp` and `5984/tcp`**.

### 17.7 Every dependent figure, walked rather than asserted

| Where | Was | Is |
|---|---|---|
| §1 pair count | 37 | **37, unchanged** |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 = 37 | **unchanged.** No row enters or leaves |
| §2.2 footing table, as restated by §16.7 | [#76](https://github.com/winniel123/verge-asm/issues/76)'s tiers and coverage | **unchanged in every cell by this section**, and the denominator stays **37**. §17.6 records a **candidate** move for `10255/tcp` and does not make it — the cell is #76's |
| §2.2 weak tier, as §16.5 left it | `5432`, `5984`, `10255` | **three, unchanged.** All three survive §17.6; `10255`'s footing is disclosed as **under-stated rather than absent** |
| §2.6's boast | true of all 37 | **unchanged** |
| §4.5 *the list's weakest row* | 5432/tcp | **unchanged**, and its corpus is now stated as exhausted rather than open |
| §4.6 exclusions | 18 named | **18 named.** No entry is added or removed; three have their **stated reason** corrected (`5672`/`15672` in §17.4, `9092` in §17.5) without any grounds moving |
| §6.1 containment arithmetic | 28 + 4 + 5 = 37 | **unchanged** — no row enters the sensitive half |
| §11.6's removal of `161/udp` | performed | **confirmed on a wider corpus.** §11.8's criterion is unmet in the one class it had not covered |
| §8 | 10 questions | **12** — question 11 opened (§17.8, routed to [#87](https://github.com/winniel123/verge-asm/issues/87)) and question 12 opened (§17.5, routed to [#86](https://github.com/winniel123/verge-asm/issues/86)) |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | unchanged | **unchanged** — no member enters or leaves |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version and the `Break` | — | **not triggered.** `sensitive-port-reached-from-internet`'s content is byte-identical |

**Outside this note.**
[ADR-0046](../adr/0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)
is added and is a rule about **which negatives ADR-0040's obligation applies to and how the resulting
backlog terminates**, not a table edit. ADR-0008, ADR-0009, ADR-0032, ADR-0035, ADR-0036, ADR-0037,
ADR-0040, ADR-0042 and ADR-0048 are untouched.
[`weak-key-and-signature.md`](./weak-key-and-signature.md) §14 carries the same audit for the other
curated table. **One finding is reported to the curator rather than applied**: §17.6's candidate
footing for `10255/tcp`, which would move a cell in §16.7's restated table.

### 17.8 Does the sweep have an end? Yes, and the residue is named

The map's curation patch has carried, since [#66](https://github.com/winniel123/verge-asm/issues/66),
*a row's footing was never checked against the standard it is now held to* as **a backlog with an end
rather than a watch**, and asks whoever discharges the patch to say whether that backlog is finished.
**For this table, it is — as of a stated table state, and that qualifier is not a hedge.**

The reason it terminates is not diligence. It is that **a class list is a property of the owner, not of
the subject**, and no owner has an unbounded one: a standards body has three, a single-project owner
usually two, a project with a reference implementation three, a vendor two. The list is fixed before the
search begins, and an owner cannot invent a fourth class to defeat a sweep. That is the whole difference
between this and a general *somebody may have said this somewhere*, and it is
[ADR-0046](../adr/0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)'s
second limb.

**And the qualifier earned itself during this ticket, which is the most useful thing in this section.**
The population was fixed at thirteen negatives, and while the sweep was running
[#76](https://github.com/winniel123/verge-asm/issues/76)'s §16.5 moved `10255/tcp` into the weak tier
on a **restricting default alone** — manufacturing a fourteenth sole-ground negative out of a row that
had not had one. It was swept (§17.6) and it was the one that returned a sentence.

> **A class sweep is complete as of a table state, and a *footing* move re-arms it for exactly the row
> that moved.** A row whose footing changes tier acquires or loses a sole-ground negative, so the
> backlog's end is a **fixed point** rather than a date: it is finished when no row's footing has moved
> since the last sweep. That is cheap to check — the footing table names the tiers — and it is a
> different obligation from the *watch* the map's curation patch already prices, which fires when the
> **world** moves. **This one fires when *we* move**, and it had never been named.

Its first application is already owed: `10255/tcp` entered the weak tier this week, and §17.6 swept it
the same week. Had the two passes run in the other order, the sweep would have reported *finished* over
a population that was about to grow by one.

**The residue is two rows, and it is a different gate's backlog rather than this one's.** `5601/tcp`
Kibana and `8500/tcp` Consul are each excluded on owner-silence **plus** a squat — `esmagent` and
`fmtp` — and [ADR-0042](../adr/0042-a-squat-is-contested-where-the-other-convention-is-live.md) made
*contested* testable by keying it to the competing owner's **current documentation**. **[measured]
Nobody has run that test on either registration**, so their second ground is asserted rather than
established, and until it is, neither row's owner-silence negative can be called overdetermined with
confidence. That is **§2.4's backlog**, not §2.2's, it is the smallest extension of this section's
corpus that could change an answer, and it is opened as §8 question 11 and routed to
[#87](https://github.com/winniel123/verge-asm/issues/87) — after
[#82](https://github.com/winniel123/verge-asm/issues/82), which supplies the evidence standard the
test needs — rather than swept in here. Same routing §14.6 used, and for the same reason.

**One thing this section does not establish**, stated because a reader will otherwise assume it. The
audit covers negatives **this note is standing on**. It does not certify that every *positive* footing
was gathered under ADR-0040's rule; §13 did that work for the shipped-configuration half, and the prose
half of §3.4 has never been re-derived class by class. That is a different question, it moves no row on
any answer, and it is out of scope here.

### 17.9 Thin ground, flagged per the standing rule

**The exposure filter is a rule about this note's own gates, and it inherits their soundness.** Calling
eleven negatives overdetermined depends on §10.2's closed claim set, §2.4's determinacy gate and §10.3's
owner requirement each being right. §10.2's own thin-ground paragraph already flags that the closure
rests on *an internet vantage supplies exactly three things*, read off `Exposure`'s definition rather
than measured. If that derivation fails, several rows in §17.1's *Sole ground?* column change value and
the sweep is larger than four. The dependency is stated rather than buried.

**The `10255` finding is a checklist item, and a checklist is a genre with its own hazard.**
*"- [ ] The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet"* is a box for an
operator to tick, which is grammatically a **hardening instruction** — the shape §9.1 refused for Red
Hat's *"use TCP Wrappers to limit which networks or hosts have access"* and §4.4 refused for the
NSA/CISA Kubernetes guidance. The reply is that both of those were refused on **ownership**, and
Kubernetes owns the kubelet; but a reader who holds that a checklist is a preference however
well-owned has a real argument, and it is a second reason the cell is reported rather than moved.

**The Kafka finding rests on the absence of a file at one tag and one URL.** `4.3.1` is the current
release and `trunk` is upstream's own branch; intermediate release branches were not enumerated, and a
`4.4.x` that ships the file would change the verdict tomorrow. The claim made is narrow — *not at
`4.3.1`, not on the published site on 2026-08-14* — and it is not *never shipped*.

**`5601` and `8500` were classified without being searched, and that is a decision rather than an
omission.** Their exclusions rest partly on a determinacy call whose test has not been run, so a class
sweep of Elastic's and HashiCorp's deployment documentation could not have settled either row on its own
— it would have produced an attestation answer for a row whose other gate is open, which is the sequencing
mistake ADR-0037 limb 2 forbids. They are named as the residue instead.

**No claim is made that these owners have no further documents.** What is established is that each
owner's **class list** was enumerated and each existing class was either searched or shown not to be
sole-ground. A reader who names a document of a class not in an owner's list falsifies this section,
which is the property ADR-0040 requires of a bounded weakness and a permanent caveat lacks.

### 17.10 Retrieval method and hazards, recorded per §9.5, §11.9, §12.9, §13.10 and §14.6

Every RFC was fetched with `curl` as plain text from `rfc-editor.org` and searched over the retrieved
bytes, never through a rendering or a summarising layer, per
[#46](https://github.com/winniel123/verge-asm/issues/46). Every artefact opened was read end to end for
every subject in this table's domain that it names, per ADR-0037 limb 1.

- **A document's *genre* has to be read before its content, and RFC 4778 is the case.** *Operational
  Security Current Practices* is a **survey**, and a session that greps it for `management network` and
  quotes the hit has founded a row on interview data. The tell is in the document's own sentences —
  *"In all large ISPs that were interviewed"* — not in its title or its category. This is the operational
  class's characteristic hazard, and it is the reason ADR-0040's *deployment recommendation* class needs
  reading with §2.5's rule in hand.
- **`kafka.apache.org/documentation/` and `/43/documentation.html` both serve a JavaScript shell** to a
  plain client — 19,879 and 19,985 bytes of navigation with no prose — in the `learn.microsoft.com` shape
  §9.5 recorded and the `docs.oracle.com` shape §14.6 recorded. The Kafka documents were read from
  `raw.githubusercontent.com` at `trunk` and at tag `4.3.1`, which is the shipped source rather than a
  rendering of it. **The 404 that carries §17.5's second ground was taken against the rendered site**,
  where a JavaScript shell would have returned 200 — so it is a real absence and not a retrieval failure.
- **`www.rabbitmq.com` returns HTTP 403 to a plain client** and 200 with a browser user-agent. The
  production checklist was read from the second response, with markup stripped locally. That is one step
  weaker than a raw artefact and it is flagged; the quoted passage was cross-checked against the page's
  own section anchor (`Firewall Configuration`).
- **The `LDAPString` trap has a Kafka analogue and it was avoided.** A search for `expose` in Kafka's
  documentation matches *"the broker does not expose the controller listener"* and *"brokers expose one
  or more listeners"* — the verb in its ordinary technical sense — far more often than it matches an
  exposure position. The §17.5 quote was located by reading the file, not by grepping for the word.
- **Not exhausted, and recorded as such.** Kafka's `docs/security/` holds ten files and
  `docs/operations/` thirteen; four were read (`security-model.md`, `security-overview.md`,
  `listener-configuration.md`, `multi-tenancy.md`). **[measured]** the string `internet` occurs **zero**
  times in all four. The remaining files are about SASL mechanisms, SSL configuration, ACL syntax and
  Connect/Streams, and none is a network-posture document — but that is a judgement from their titles,
  not a measurement of their contents.
- **RFC 3512's five hits were each read in context rather than counted.** Two are about trap delivery
  during a denial-of-service flood, one is an ITU-T bibliography entry (*Recommendation M.3010*), and two
  are VPN forwarding used as a MIB-design example. A count alone would have read as five near-misses.

---

## 18. An owner's category statement reaches the members its own artefacts place inside it

Wayfinder ticket [#88](https://github.com/winniel123/verge-asm/issues/88), applying
[ADR-0049](../adr/0049-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md).

§17.6 retrieved the first **owner** category statement in the corpus and reported it rather than
applying it, naming the question it turns on: **§2.3 refuses a source that attests a category rather
than a port — is that refusal about the *grammar* of a category statement, or about the *standing*
of the party making it?** Every category statement in the corpus when §2.3 was written came from
CISA or a cloud provider, so the two readings had never been made to disagree.

**They disagree now, and the answer is standing.** This section states the rule, walks it across
every cell in §16.7's table and every exclusion it could reach, moves two footing cells and no rows,
and names the one row the rule newly exposes.

**This ticket performed no retrieval.** A row moves on a retrieval and never on a re-reading
([#37](https://github.com/winniel123/verge-asm/issues/37)); §79's retrieval is the evidence and this
is the rule over it. Every quote below is already in this note or in §17.

### 18.1 The rule, in three limbs

> **An owner's statement about a category reaches every member of that category that the owner's own
> artefacts place inside it.** Three limbs, all required:
>
> 1. **Standing.** The speaker **owns the protocol** under §10.5 — it designed the protocol or
>    authors the reference implementation, speaking about the thing it designed or wrote.
> 2. **Membership.** **The owner's own artefact places the `(port, transport)` pair inside the
>    category.** The mapping may not be the reader's inference and may not be supplied by a
>    corroborator. The row is the concatenation of two owner statements.
> 3. **Defeat, per member.** Reach **fails for any member whose internet-facing deployment the owner
>    elsewhere names as supported** — §10.3's failure condition, tested per member rather than per
>    sentence.
>
> **The category's unit is a protocol or an interface, never a vendor's product line**
> ([ADR-0048](../adr/0048-a-convention-is-evidenced-by-placement-never-by-catalogue.md)). *The
> kubelet API* is an interface of one component and qualifies. *Kubernetes*, *our appliances* or
> *Dell's DRACs* are product categories and do not.
>
> **§2.3 is narrowed, not weakened.** A non-owner's category statement still carries nothing, for
> exactly the reason §2.3 gives. What is withdrawn is the implication that the **grammar** was the
> defect.

### 18.2 Why the grammatical reading lost, and it is a measurement rather than a preference

§2.3 states its reason twice and the two sentences come apart here. *"Both attest a category …
neither attests a port"* is grammatical. *"The mapping from its protocol list to port numbers is
**the reader's inference**, not CISA's"* is not — it says the defect is that **nobody with standing
closed the gap**. The two agree on every case §2.3 was written about, because a non-owner cannot
close the gap: CISA does not get to say which ports are management interfaces.

**[measured] Under the grammatical reading, most of the footing table's prohibition tier falls.** Of
the sentences carrying §16.7's 26 placed pairs, five write the port number:

| The sentence that carries the row (§3.4) | Owner writes the number? |
|---|---|
| Redis — *"expose the Redis instance directly to the internet … the Redis TCP port"* | **No** — a definite description; `redis.conf` ships `port 6379` |
| memcached — *"you must not expose memcached directly to the internet"* | **No** — the software |
| MS SQL Server — *"don't connect your SQL Server instances directly to the Internet"* | **No** — instances |
| Elasticsearch — *"Never expose an unprotected node to the public internet"* | **No** — and §16.4 already ruled it reaches `9300` |
| rsync — *"Do not expose a cleartext daemon to an untrusted network"* | **No** — the daemon |
| SMB — *"unlikely that any SMB communication … is legitimate"* | **No** in the sentence; the same document tabulates 445 / 139 / 137 / 138 |
| MongoDB — *"your `mongod` and `mongos` instances are only accessible on trusted networks"* | **No** — binaries |
| NFS — *"on a trusted physical network between trusted hosts"* | **No** |
| ZooKeeper — *"a ZooKeeper ensemble … behind a firewall"* | **No** — the ensemble |
| Docker — *"reachable only from a trusted network or VPN"* | **No** |
| IPMI — Dell, *"DRAC's are intended to be on a separate management network"* | **No**, and the number comes from CISA — §18.5 |
| MySQL `3306` · etcd `2379`/`2380` · Cassandra `9042` · RabbitMQ `25672` · kubelet `10250` | **Yes** — five cells |

That reading leaves the sensitive list as *the set of ports whose maintainers happened to type a
number*. §2.2's founding paragraph refuses the outcome in advance: a list resting on *"an asymmetry
driven by a documentation accident rather than by any difference in the two services' deployment
models"* is *"exactly the kind of arbitrariness that destroys a curated list's credibility."*

And it is not a clean-slate option. **§16.4 is already a decided instance of reach** — it admitted
`9300` on a sentence about *nodes*, quoted the owner's own definition of a node, and called the
result *"an inference of one step"*, which is limb 2 without a name. Adopting the grammatical
reading would remove `9300` from the prohibition tier, unseat `139`/`137`/`138` from the SMB cell
that carries them, and leave `623` with no footing at all.

### 18.3 The counter-argument, argued rather than dismissed

The retrieved sentence is a **checklist item** — grammatically an instruction to a deployer, which
is §9.1's Red Hat shape (*"a hardening instruction, not a legitimacy statement"*) and §4.4's
NSA/CISA shape (*"a hardening preference expressed against a real, supported architecture"*). Both
were refused. A checklist states a floor for a hardened deployment; it does not say every other
deployment is illegitimate, and this list's claim is the stronger one.

**Three answers, in increasing order of force.**

1. **§9.1's three grounds were never equal.** Red Hat lost on standing, on shape and on
   self-contradiction across its own products. Standing is cured here. Self-contradiction is absent
   for this member and **present for another** — §18.4. Shape is what remains, and it never stood
   alone.
2. **The standard already admits weaker modality.** §2.2's third form asserts nothing about
   legitimacy at all: PostgreSQL's `listen_addresses = localhost` says only what the software does,
   and it is the **sole** footing for two rows. A sentence in the owner's voice saying the interface
   *"is not exposed publicly on Internet"* is not weaker than a config default; it is the same
   position stated out loud.
3. **[measured] Applied consistently, the objection removes rows nobody proposes removing.** MySQL's
   *Security Guidelines*, MongoDB's *security hardening*, Microsoft's *Security considerations for a
   SQL Server installation* and ZooKeeper's *Administrator's Guide* are hardening documents in the
   owner's voice, and all four carry rows in §3.4 today. **The genre is not the defect; the party
   is.**

> **Against a non-owner, *this is a hardening instruction* is fatal because limb 1 has already
> refused the party. Against an owner it adds nothing that §10.3's failure condition does not
> already test** — an instruction is a *preference* exactly where the architecture it advises
> against is one the owner supports.

### 18.4 One sentence, three members, and it is defeated for one of them

The retrieved checklist item names **the Kubernetes API, the kubelet API and etcd**. §4.4 excludes
`6443/tcp` and quotes this very sentence — then quotes what upstream says immediately after it:

> "Be careful, as **many managed Kubernetes distributions are publicly exposing the API server by
> default**."

So the owner states the category prohibition and, in the next breath, concedes that its own
ecosystem's dominant deployment violates it **for one member**. For `6443` the checklist item is a
preference expressed against a real supported architecture — §4.4's words, now demonstrably about
the owner's own architecture. For the kubelet API the owner concedes nothing of the kind anywhere
retrieved: `10250` is *Used By: Self, Control plane* and `10255` ships disabled.

> **A category sentence is tested per member, not per sentence** — the sibling of
> [#76](https://github.com/winniel123/verge-asm/issues/76)'s *ownership is tested per port, not per
> sentence* (§16.6), and produced by the same defect: a sentence is not a unit of evidence, a claim
> about a subject is.

**`6443/tcp` does not re-open.** Its exclusion rests on three independent grounds and the
attestation limb was never one of them — Claim 3 fails on the facts, determinacy fails against
`sun-sr-https` and against Kubernetes' own *"the API serves on port 443"*, and §4.4's third ground
is a product judgement about false firings. What changes is §4.4's **disclosure**: the upstream
quote is an **owner prohibition that this row defeats on other grounds**, not merely *a hardening
preference*. §4.4's wording is left standing per the name-and-withdraw convention and is corrected
here.

### 18.5 The walk — every row the rule reaches, and the one it exposes

**Ratified, cell unchanged (limbs 1-3 satisfied on artefacts this note already holds):** `6379`,
`11211/tcp`+`11211/udp`, `1433`, `9200`+`9300`, `873`, `445`+`139`+`137`+`138`, `27017`/`27018`/`27019`,
`2049`, `2181`, `2375`+`2376`. In each the owner states the category and the owner's own
documentation or shipped configuration numbers the port — for several (`6379`, `2181`, `27017`) the
two are the same file. **Nothing about these cells moves**; what changes is that their warrant is
now named rather than assumed.

**Numbered by the owner, so the rule is not needed:** `3306`, `2379`, `2380`, `9042`, `25672`,
`10250` (whose `ports-and-protocols.md` row numbers it).

**Moved — two cells, both up, both conditional on [#83](https://github.com/winniel123/verge-asm/issues/83):**
`10250/tcp` and `10255/tcp`, §18.6.

**Not rescued — `4369/tcp` epmd.** Erlang/OTP's sentence is about *distributed nodes* and the
distribution transport; epmd is the mechanism's **registry** rather than the transport, and the
owner's own epmd page distinguishes them and declines to prohibit anything (§3.4, §16.6). The owner
does not place `4369` inside the category it spoke about, so **limb 2 fails** and §16.9's *"a reader
who says Erlang/OTP has stated a position about the distribution port and no position about `4369`
has a live argument"* survives this section completely intact. This is the load-bearing negative
result: the rule is not a device for laundering an inference into a citation.

**Newly exposed — `623/udp` IPMI, ticketed and not moved.** Dell's *"DRAC's are intended to be on a
separate management network"* is the sentence §3.4 says *"actually carries the row"*. It names
**DRACs**, a product line, which fails the ADR-0048 unit check on its face; and the number that
connects it to the row is **CISA's** *"usually UDP port 623"*, a corroborator, which limb 2 forbids.
That is §10.6's shape — *a corroborator standing where an owner should* — in a second instance, in
the **prohibition** tier, and §10.6 is what took `161/udp` off the list. **It is not decided here:**
this ticket performed no retrieval, ADR-0037 limb 2 requires a finding of this shape to be ticketed
rather than acted on, and the likely resolution is that the IPMI or ASF specification numbers the
port and re-founds the cell on §2.2's **first** form. Routed to
[#90](https://github.com/winniel123/verge-asm/issues/90), which **blocks
[#12](https://github.com/winniel123/verge-asm/issues/12)** because it can reach a row removal.

**Exclusions checked and unmoved:** `6443` (§18.4). `5601` Kibana, `8500` Consul, `1099` Java RMI,
`8080` Jenkins, `9100`, `22`, `3389`, `5985`/`5986`, `111`, `389`, `79`, the Hadoop UIs and the mail
ports are excluded on grounds that contain no category statement by an owner, so the rule does not
reach them. `5672`/`15672` RabbitMQ is untouched: §17.4 found the owner naming **public networks as
supported** for exactly those two ports, which is limb 3's defeat condition met directly.

**One exclusion the rule composes with rather than decides — `9092/tcp` Kafka.** §17.5's sentence
(*"accepts unauthenticated `PLAINTEXT` connections on **every listener**"*, *"before being exposed to
any untrusted network"*) names **no port**, so even if
[#86](https://github.com/winniel123/verge-asm/issues/86) rules the unreleased document admissible as
the owner's documentation, the sentence needs **this** rule as well to reach `9092`. Limb 2 looks
satisfiable there on the owner's own shipped `server.properties`, and that is a statement about
where the evidence would come from, **not** a ruling — the row stays excluded and #86 owns it.

### 18.6 The two cells that move, and the tier table restated

**`10255/tcp` → explicit prohibition.** Limb 1: Kubernetes authors the kubelet's reference
implementation and §16.5 already records that kubernetes.io speaking about the kubelet is the owner
speaking. Limb 2: the owner's own shipped bytes place the port inside *the kubelet API* —
`readOnlyPort` is *"the read-only port for the Kubelet to serve on with no
authentication/authorization"* (`staging/src/k8s.io/kubelet/config/v1beta1/types.go`, `v1.34.1`).
Limb 3: nothing retrieved has the owner supporting an internet-facing kubelet. The first §17.6 quote
is an unqualified negative about internet exposure and names **no trusted network**, which is what
makes it a prohibition rather than a scoping.

**`10250/tcp` → explicit prohibition**, from the scoping tier. The same two sentences, and the
membership evidence is the **stronger** of the two: `ports-and-protocols.md` at `release-1.34`
numbers `10250` as *Kubelet API* in the owner's own table. §16.9 flagged this cell as *"the thinnest
placement in the table"* precisely because `Used By: Self, Control plane` *"names the port's clients
rather than its permitted network"* — under this rule the table cell stops carrying the **position**
and carries the **membership**, which is the job it fits, and the position comes from a sentence.
That flag is discharged.

> **§2.2's footing table, restated. It places 26 of the 37 pairs; the eleven uncovered are uncovered
> by design and unchanged from §16.7.**
>
> | Footing | Pairs |
> |---|---|
> | **Explicit prohibition** in the owner's own words | 6379 Redis · 11211/tcp + 11211/udp memcached · 3306 MySQL · 1433 MS SQL · 9200 and 9300 Elasticsearch · 873 rsync · 445 SMB · 623 IPMI · 9042 Cassandra · 2379 and 2380 etcd · **10250 and 10255 kubelet** — **15 pairs** |
> | **Explicit trusted-network scoping**, slightly weaker than a prohibition | 27017/27018/27019 MongoDB · 2049 NFS · 2181 ZooKeeper · 25672 RabbitMQ · 4369 epmd · 2376 and 2375 Docker — **9 pairs** |
> | **Shipped default only** — no prohibition exists upstream | **5432 PostgreSQL · 5984 CouchDB** — **2 rows** |
> | *Outside this table's subject, and correctly absent* | *Class B's seven (23, 21, 512, 513, 514, 5900, 6000) and 69/udp rest on §2.2's **first** form; 139/tcp, 137/udp and 138/udp sit inside 445's cell — **11 pairs*** |
>
> 15 + 9 + 2 + 11 = 37. **No `(port, transport)` pair moves.**
>
> **Both kubelet cells are conditional on [#83](https://github.com/winniel123/verge-asm/issues/83)**,
> which is deciding whether either row survives Class A on the finding that the shipped source
> contradicts the generated reference page. A footing is evidence for a claim and not a claim
> ([ADR-0036](../adr/0036-a-shipped-default-is-the-configuration-that-takes-effect.md), §12.7), so a
> cell cannot outlive its row: if #83 removes a kubelet row, its cell leaves with it and the
> prohibition tier reads 14 or 13 pairs with the coverage denominator falling to 25 or 24. **If #83
> moves `10250` from Class A to Class C, nothing here moves at all** — the footing is about network
> position and is indifferent to which claim the row rests on.

### 18.7 Thin ground, flagged per the standing rule

**`10255`'s cell is the thinnest member of the prohibition tier, and it is thinner than `10250`'s.**
Its membership evidence is a **doc comment on a config-API field** rather than an entry in the
owner's ports table; the port appears nowhere in `ports-and-protocols.md` at all (§16.5, verified
positively). And both cells rest on a **checklist item** — the only cells in the tier that do. The
tier's other members are prose sentences in reference documentation or shipped configuration
comments, genres that change slowly. A checklist line in a documentation release branch can be
edited in one commit by one contributor, which is
[ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's silent
de-attestation arriving from the direction §16.9 named for `2379`/`2380` — a young or easily-edited
**positive** rather than a flippable default. **The criterion that would change the verdict:**
removal or material weakening of the *Network security* checklist item, at which point `10255` falls
back to the weak tier on `readOnlyPort` alone and `10250` falls back to the scoping tier on the
ports table.

**The rule was ruled without a live human, and one limb is thinner than the other two.** Limbs 1 and
2 are read off existing text — §10.5 defines ownership, and membership is a documentary fact.
**Limb 3 is the constructed one**: nothing in §2.3, §9.1 or §4.4 says in so many words that the
hardening-instruction objection reduces to §10.3's failure condition. It is inferred from §9.1's
three grounds not being equal and from §4.4's own phrase *against a real, supported architecture*,
and it is confirmed by the one case in the corpus where an owner's category sentence is defeated for
one member and not another. That is a strong instance and it is **one** instance.

**What was not done.** No artefact was retrieved for this ruling, by design. So the walk in §18.5 is
a walk over **quotes this note holds**, and where it says an owner numbers a port it means the note
records that it does. `1433` is the softest of those: Microsoft numbers `1433` widely in its own
documentation, but no quote in §3.4 does, and §13.1 records that `1433` has **no shipped
configuration artefact at all**. It is ratified on the strength of the number being uncontroversially
the owner's, which is an inference of the kind limb 2 exists to discipline. If `623` (§18.5) comes
back badly, `1433` is the next cell to check.

### 18.8 Every dependent figure, walked rather than asserted

| Where | Was | Is |
|---|---|---|
| §1 pair count | 37 | **37, unchanged.** No row is added or removed |
| §3.1 / §3.2 / §3.3 class totals | 12 / 7 / 18 = 37 | **unchanged.** No row changes class; no claim moves |
| §2.2 footing table — coverage | 26 of 37 (§16.7) | **26 of 37, unchanged.** No pair enters or leaves the table's subject |
| §2.2 footing table — **prohibition tier** | 13 pairs | **15 pairs.** `+10250`, `+10255` |
| §2.2 footing table — **scoping tier** | 10 pairs | **9 pairs.** `−10250` |
| §2.2 footing table — **weak tier** | 3 rows (`5432`, `5984`, `10255`) | **2 rows** (`5432`, `5984`). `−10255` |
| §16.7's restated table | #76's tiers | **superseded by §18.6 in three cells**; every other cell and the coverage sentence stand |
| §16.9's *`10250` is the thinnest placement in the table* | flagged | **discharged.** The table cell now carries membership, not position. §18.7 opens a new flag on `10255` |
| §17.6's candidate footing for `10255` | reported, not applied | **applied.** §17.6's *"the criterion that would move the cell"* is met by ADR-0049 |
| §17.7's *§2.2 weak tier: three, unchanged* | 3 | **superseded — two.** §17.7 was correct on the day it was written |
| §4.4's characterisation of the upstream quote | *a hardening preference* | **corrected** — an owner prohibition the row defeats on other grounds (§18.4). The exclusion is unchanged |
| §4.5 *the list's weakest row* | `5432/tcp` | **unchanged**, and now the weak tier's senior member of two rather than three |
| §4.6 exclusions | 18 named | **18 named.** None is added or removed |
| §6.1 containment arithmetic | 28 + 4 + 5 = 37 | **unchanged** |
| §8 | 12 questions | **12.** No question opens here; §18.5's `623` finding is a **row** question and is ticketed rather than parked |
| [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union | unchanged | **unchanged** — no member enters or leaves |
| [ADR-0008](../adr/0008-derivation-versions-move-on-content.md) rule version and the `Break` | — | **not triggered.** `sensitive-port-reached-from-internet`'s content is byte-identical |

**Outside this note.**
[ADR-0049](../adr/0049-an-owners-category-statement-reaches-the-members-its-own-artefacts-place-inside-it.md)
is added. **ADR-0032 §8's watch-list enumeration is amended for the third time** and reads `5432/tcp`
and `5984/tcp`; the sequence is now 3 → 2 → 3 → 2 with the membership changing at every step, which
is §8's own *compare members, not counts* lesson in a second instance. The map's *how the tiered port
sets are curated* patch must record the watch list as **two rows** again.
[ADR-0046](../adr/0046-a-negatives-corpus-is-its-owners-class-list-and-only-a-sole-ground-negative-is-exposed.md)'s
fourteenth sole-ground negative **retires**: `10255` no longer rests on an absence, so it leaves the
exposed population, and ADR-0046's table-state qualifier fires in the **disarming** direction for the
first time. ADR-0036, ADR-0037, ADR-0040, ADR-0042, ADR-0048 and
[`weak-key-and-signature.md`](./weak-key-and-signature.md) are untouched — the rule travels to that
table under ADR-0032 and **moves nothing there**, checked against its §13.5 restated tier table,
where every surviving footing names its own subject.

### 18.9 Retrieval method, recorded per §9.5, §11.9, §12.9, §13.10, §14.6 and §16.10

**No artefact was retrieved.** This is the first numbered section of this note that performs none,
and it is deliberate: [#37](https://github.com/winniel123/verge-asm/issues/37) rules that a row moves
on a retrieval and never on a re-reading, and every quote this section relies on was retrieved by
[#76](https://github.com/winniel123/verge-asm/issues/76) or
[#79](https://github.com/winniel123/verge-asm/issues/79) and is reproduced from this note rather than
re-fetched. **Two consequences are worth recording.** No cell here can be stronger than the retrieval
that produced its quote — `10250`/`10255` inherit §16.10's discipline and its caveats whole. And the
`623` finding is stated as a **gap in the artefacts this note holds**, which is not the same claim as
*no owner artefact numbers the port*; that stronger claim needs the search
[#90](https://github.com/winniel123/verge-asm/issues/90) will run, and ADR-0040 binds its negative.

---

## Sources

Government and standards bodies
- [CISA BOD 23-02, Mitigating the Risk from Internet-Exposed Management Interfaces](https://www.cisa.gov/news-events/directives/binding-operational-directive-23-02) (13 June 2023) · [implementation guidance / FAQ](https://www.cisa.gov/news-events/directives/bod-23-02-implementation-guidance-mitigating-risk-internet-exposed-management-interfaces)
- [CISA Cross-Sector Cybersecurity Performance Goals v2.0](https://www.cisa.gov/sites/default/files/2025-12/CPG_Report_2.0_508c.pdf) (December 2025), goal 3.S — verified against the PDF's own text
- [CISA BOD 22-01 (Revoked)](https://www.cisa.gov/news-events/directives/bod-22-01-reducing-significant-risk-known-exploited-vulnerabilities) — revoked 10 June 2026, superseded by BOD 26-04
- [CISA/US-CERT, SMB Security Best Practices](https://www.cisa.gov/news-events/alerts/2017/01/16/smb-security-best-practices)
- [CISA TA13-207A, Risks of Using the Intelligent Platform Management Interface (IPMI)](https://www.cisa.gov/news-events/alerts/2013/07/26/risks-using-intelligent-platform-management-interface-ipmi) · [CISA TA17-156A, Reducing the Risk of SNMP Abuse](https://www.cisa.gov/news-events/alerts/2017/06/05/reducing-risk-snmp-abuse)
- [CERT/CC VU#843044](https://www.kb.cert.org/vuls/id/843044) — carries Dell's statement that DRACs are "not designed nor intended to be placed on or connected to the internet"
- [NIST SP 800-123, Guide to General Server Security](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-123.pdf) §6.5 — corroborating, not load-bearing
- [CISA AA22-137A, Weak Security Controls and Practices Routinely Exploited for Initial Access](https://www.cisa.gov/news-events/cybersecurity-advisories/aa22-137a)
- [CISA #StopRansomware Guide v3.0](https://www.cisa.gov/sites/default/files/2025-03/StopRansomware-Guide%20508.pdf)
- [NSA/CISA Kubernetes Hardening Guidance v1.2](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF) (August 2022)
- [IANA Service Name and Transport Protocol Port Number Registry](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml) — registry data retrieved as [CSV](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.csv), last updated 2026-08-11
- [RFC 1282 (BSD Rlogin)](https://www.rfc-editor.org/rfc/rfc1282.txt) · [RFC 1288 (Finger)](https://www.rfc-editor.org/rfc/rfc1288.txt) · [RFC 1350 (TFTP)](https://www.rfc-editor.org/rfc/rfc1350.txt) · [RFC 2577 (FTP Security Considerations)](https://www.rfc-editor.org/rfc/rfc2577.txt) · [RFC 3410 (SNMP framework)](https://www.rfc-editor.org/rfc/rfc3410.txt) · [RFC 3617 (TFTP URI scheme + security concerns)](https://www.rfc-editor.org/rfc/rfc3617.txt) · [RFC 4146 (Simple New Mail Notification — the second registered usage of port 79)](https://www.rfc-editor.org/rfc/rfc4146.txt) · [RFC 4248 (telnet URI scheme)](https://www.rfc-editor.org/rfc/rfc4248.txt) · [RFC 4513 (LDAP authentication methods)](https://www.rfc-editor.org/rfc/rfc4513.txt) · [RFC 6143 (RFB / VNC)](https://www.rfc-editor.org/rfc/rfc6143.txt) · [RFC 6335 (port registry procedures)](https://www.rfc-editor.org/rfc/rfc6335.txt) · [RFC 8314 (cleartext mail considered obsolete)](https://www.rfc-editor.org/rfc/rfc8314.txt)
- Retrieved for §9.2 and citable only for what they **do not** contain — `636` and `ldaps://` occur zero times in each: [RFC 4510 (LDAP roadmap)](https://www.rfc-editor.org/rfc/rfc4510.txt) · [RFC 4511 (LDAP protocol)](https://www.rfc-editor.org/rfc/rfc4511.txt) · [RFC 4516 (LDAP URL)](https://www.rfc-editor.org/rfc/rfc4516.txt) · [RFC 2830 (obsoleted LDAP StartTLS extension)](https://www.rfc-editor.org/rfc/rfc2830.txt)
- Retrieved for §11 ([#66](https://github.com/winniel123/verge-asm/issues/66)) — the SNMP standards family, read for a **placement** sentence and citable chiefly for the absence of one. `management network`, `segregat`, `firewall`, `separate network`, `isolated network`, `private network`, `untrusted network`, `public network` and `out of band` occur **twice across all ten**, both in RFC 6353 and both about certificate distribution: [RFC 3411 (SNMP architecture, STD 62)](https://www.rfc-editor.org/rfc/rfc3411.txt) — `administrative domain` ×8, every one an identifier scope · [RFC 3412 (message processing, STD 62)](https://www.rfc-editor.org/rfc/rfc3412.txt) · [RFC 3413 (SNMP applications, STD 62)](https://www.rfc-editor.org/rfc/rfc3413.txt) · [RFC 3414 (USM, STD 62)](https://www.rfc-editor.org/rfc/rfc3414.txt) §1.1–1.2, the threat model, every threat a property of the message · [RFC 3417 (transport mappings, STD 62)](https://www.rfc-editor.org/rfc/rfc3417.txt) §3.2, which puts SNMPv3 command responders on UDP 161 · [RFC 3584 (coexistence, BCP 74)](https://www.rfc-editor.org/rfc/rfc3584.txt) · [RFC 2570 (RFC 3410's predecessor)](https://www.rfc-editor.org/rfc/rfc2570.txt) · [RFC 1157 (SNMPv1, STD 15, Historic)](https://www.rfc-editor.org/rfc/rfc1157.txt) §3.2.5, which defines an SNMP community as a pairing with *"some arbitrary set of SNMP application entities"* and whose Security Considerations reads in full *"Security issues are not discussed in this memo."* · [RFC 6353 (TLS Transport Model)](https://www.rfc-editor.org/rfc/rfc6353.txt) §8, the `snmptls`/`snmpdtls` 10161 and 10162 registrations — the string `161` occurs in it **zero** times
- [RFC 5531 (ONC RPC v2)](https://www.rfc-editor.org/rfc/rfc5531.txt) §14 — checked for §9.1. Its Security Considerations govern RPC *auth flavours* (AUTH_SYS, AUTH_DH, RPCSEC_GSS), never rpcbind or port 111, so it attests nothing about exposure
- Checked and citable only as **non-statements**: RFC 854 (Telnet) has no Security Considerations section and no occurrence of "security", "password", "encrypt" or "authentic"; [RFC 1833](https://www.rfc-editor.org/rfc/rfc1833.txt)'s Security Considerations reads in full: "Security issues are not discussed in this memo."; [RFC 1288](https://www.rfc-editor.org/rfc/rfc1288.txt) §6 reads in full: "Security issues are discussed in Section 3." — and RFC 1288 remains a **Draft Standard** on the [IETF datatracker](https://datatracker.ietf.org/api/v1/doc/document/rfc1288/), never reclassified Historic
- [CIS Amazon Web Services Foundations Benchmark v1.2.0](https://d1.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf) (05-23-2018) — the CIS-authored PDF, ungated. Current versions are behind a registration form at [learn.cisecurity.org/benchmarks](https://learn.cisecurity.org/benchmarks); the "remote server administration ports" phrasing in v3.0.0+ was **not** verified against a CIS document and is second-hand via AWS's mapping page

Cloud providers
- AWS: [default security group](https://docs.aws.amazon.com/vpc/latest/userguide/default-security-group.html) · [Trusted Advisor security checks](https://docs.aws.amazon.com/awssupport/latest/user/security-checks.html) (check `HCP4007jGY`) · [Security Hub EC2 controls](https://docs.aws.amazon.com/securityhub/latest/userguide/ec2-controls.html) (EC2.13, EC2.14, EC2.19, EC2.21) · [CIS mapping](https://docs.aws.amazon.com/securityhub/latest/userguide/cis-aws-foundations-benchmark.html)
- Azure: [NSG default security rules](https://learn.microsoft.com/en-us/azure/virtual-network/network-security-groups-overview) · [Defender for Cloud networking recommendations](https://learn.microsoft.com/en-us/azure/defender-for-cloud/recommendations-reference-networking) · [just-in-time VM access](https://learn.microsoft.com/en-us/azure/defender-for-cloud/enable-just-in-time-access)
- Google Cloud: [VPC firewall rules, including the default network's pre-populated rules](https://docs.cloud.google.com/firewall/docs/firewalls)
- Microsoft (non-cloud): [Preventing SMB traffic from lateral connections](https://support.microsoft.com/en-us/topic/preventing-smb-traffic-from-lateral-connections-and-entering-or-leaving-the-network-c0541db7-2244-0dce-18fd-14a3ddeb282a) · [Secure SMB traffic in Windows Server](https://learn.microsoft.com/en-us/windows-server/storage/file-server/smb-secure-traffic) · [SQL Server installation security considerations](https://learn.microsoft.com/en-us/sql/sql-server/install/security-considerations-for-a-sql-server-installation) · [LDAP signing for Active Directory Domain Services](https://learn.microsoft.com/en-us/windows-server/identity/ad-ds/ldap-signing) — quoted from [its source Markdown](https://raw.githubusercontent.com/MicrosoftDocs/windowsserverdocs/main/WindowsServerDocs/identity/ad-ds/ldap-signing.md) because the rendered page is a JavaScript shell (§9.5)
- Red Hat (§9.1) — **corroboration only; declined as sole grounds on §2.3 ownership**, and self-contradictory across products. Both pages refuse direct retrieval and were read via Internet Archive snapshots: [RHEL 7 Security Guide §4.3, Securing Services](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/security_guide/sec-securing_services) ("difficult to secure", "no built-in form of authentication") · [RHEL 6 Storage Administration Guide §9.7.3, Running NFS Behind a Firewall](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/6/html/storage_administration_guide/s2-nfs-nfs-firewall-config) ("Allow TCP and UDP port 111 (rpcbind/sunrpc).")

Upstream projects
- [Redis security](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)
- [memcached ConfiguringServer](https://github.com/memcached/memcached/wiki/ConfiguringServer) · [DDoS advisory](https://docs.memcached.org/advisories/ddos/)
- [MongoDB security hardening](https://www.mongodb.com/docs/manual/core/security-hardening/) · [security checklist](https://www.mongodb.com/docs/manual/administration/security-checklist/) · [configuration options](https://www.mongodb.com/docs/manual/reference/configuration-options/)
- [Elasticsearch networking](https://www.elastic.co/guide/en/elasticsearch/reference/8.19/modules-network.html) · [secure cluster communications](https://www.elastic.co/docs/deploy-manage/security/secure-cluster-communications) · [Kibana settings](https://www.elastic.co/docs/reference/kibana/configuration-reference/general-settings)
- [PostgreSQL connection settings](https://www.postgresql.org/docs/current/runtime-config-connection.html) · [SSL/TCP](https://www.postgresql.org/docs/current/ssl-tcp.html) · [server start](https://www.postgresql.org/docs/current/server-start.html)
- [MySQL security guidelines](https://dev.mysql.com/doc/refman/8.4/en/security-guidelines.html)
- [CouchDB cluster setup](https://docs.couchdb.org/en/stable/setup/cluster.html) · [HTTP config](https://docs.couchdb.org/en/stable/config/http.html)
- [Cassandra security](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/security.html) · [ports FAQ](https://cassandra.apache.org/doc/5.0/cassandra/overview/faq/index.html)
- [Docker Engine security](https://docs.docker.com/engine/security/) · [protect the daemon socket](https://docs.docker.com/engine/security/protect-access/) · [dockerd reference](https://docs.docker.com/reference/cli/dockerd/) · [deprecated features](https://docs.docker.com/engine/deprecated/)
- [Kubernetes ports and protocols](https://kubernetes.io/docs/reference/networking/ports-and-protocols/) · [kubelet authn/authz](https://kubernetes.io/docs/reference/access-authn-authz/kubelet-authn-authz/) · [kubelet CLI reference](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/) · [kubelet config API](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/) · [security checklist](https://kubernetes.io/docs/concepts/security/security-checklist/) · [controlling access](https://kubernetes.io/docs/concepts/security/controlling-access/) · [operating etcd](https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/)
- [etcd security guide](https://etcd.io/docs/v3.5/op-guide/security/)
- [ZooKeeper security](https://zookeeper.apache.org/security.html) · [Administrator's Guide](https://zookeeper.apache.org/doc/r3.9.3/zookeeperAdmin.html)
- [RabbitMQ networking](https://www.rabbitmq.com/docs/networking) · [access control](https://www.rabbitmq.com/docs/access-control) · [clustering](https://www.rabbitmq.com/docs/clustering)
- [Consul security model](https://developer.hashicorp.com/consul/docs/secure/security-model/core) · [ACL configuration](https://developer.hashicorp.com/consul/docs/reference/agent/configuration-file/acl) · [ports reference](https://developer.hashicorp.com/consul/docs/reference/architecture/ports)
- [Kafka security overview](https://kafka.apache.org/43/security/security-overview/)
- [Hadoop secure mode](https://hadoop.apache.org/docs/current/hadoop-project-dist/hadoop-common/SecureMode.html) · [cluster setup](https://hadoop.apache.org/docs/current/hadoop-project-dist/hadoop-common/ClusterSetup.html)
- net-snmp (§11), the SNMP reference implementation: [`snmpd(8)`](http://www.net-snmp.org/docs/man/snmpd.html) (*"By default, snmpd listens for incoming SNMP requests on UDP port 161 on all IPv4 interfaces"*) · [`snmpd.conf(5)`](http://www.net-snmp.org/docs/man/snmpd.conf.html) (no exposure statement anywhere) · [upstream FAQ](http://www.net-snmp.org/FAQ.html) (*"irrespective of where it originated"*; *"a completely empty access control configuration"*) · [`EXAMPLE.conf.def`](https://raw.githubusercontent.com/net-snmp/net-snmp/master/EXAMPLE.conf.def) (`agentAddress udp:127.0.0.1:161` active) · [wiki `TUT:Security`](http://www.net-snmp.org/wiki/index.php/TUT:Security) — retrieved and found to contain **no** security prose (§11.9). Debian's packaging read from the shipped bytes, `net-snmp_5.9.5.2+dfsg-2.1.debian.tar.xz` `debian/snmpd.conf`, via [`deb.debian.org`](https://deb.debian.org/debian/pool/main/n/net-snmp/), because `sources.debian.org` and `salsa.debian.org` both serve a proof-of-work challenge (§11.9)
- [Prometheus security model](https://prometheus.io/docs/operating/security/)
- [Jenkins security](https://www.jenkins.io/doc/book/security/) · [network services](https://www.jenkins.io/doc/book/security/services/)
- [Oracle Net Listener security](https://docs.oracle.com/en/database/oracle/oracle-database/26/netag/managing-oracle-net-listener-security.html)
- [Java JMX monitoring and management](https://docs.oracle.com/en/java/javase/21/management/monitoring-and-management-using-jmx-technology.html)
- [Neo4j security checklist](https://neo4j.com/docs/operations-manual/current/security/checklist/)
- [Xsecurity(7)](https://man.openbsd.org/Xsecurity.7) · [xhost(1), X.Org](https://www.x.org/releases/current/doc/man/man1/xhost.1.xhtml)
- [nfs(5)](https://man7.org/linux/man-pages/man5/nfs.5.html) · [rsyncd.conf(5), upstream Samba copy](https://download.samba.org/pub/rsync/rsyncd.conf.5)
- rpcbind upstream (§9.1), all retrieved from `git.linux-nfs.org` with certificate verification disabled — see §9.5: [`man/rpcbind.8`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=man/rpcbind.8;hb=HEAD) · [`configure.ac`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=configure.ac;hb=HEAD) (`--enable-rmtcalls [default=no]`) · [`systemd/rpcbind.socket`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=systemd/rpcbind.socket;hb=HEAD) (`ListenStream=0.0.0.0:111`) · [`README`](https://git.linux-nfs.org/?p=steved/rpcbind.git;a=blob_plain;f=README;hb=HEAD) (contains no security text). Cross-checked against [Debian's rendering of `rpcbind(8)`](https://manpages.debian.org/unstable/rpcbind/rpcbind.8.en.html) and Debian's packaging: [`debian/README.debian`](https://sources.debian.org/data/main/r/rpcbind/1.2.7-1/debian/README.debian) · [`debian/rpcbind.default`](https://sources.debian.org/data/main/r/rpcbind/1.2.7-1/debian/rpcbind.default)
- OpenLDAP (§9.2): [Administrator's Guide §14, Security Considerations](https://www.openldap.org/doc/admin26/security.html) · [§16, Using TLS](https://www.openldap.org/doc/admin26/tls.html) · [`slapd(8)`](https://www.openldap.org/software/man.cgi?query=slapd&sektion=8) (default `-h` is `ldap:///`, i.e. `INADDR_ANY:389`)
- Finger implementations (§9.3.5), all checked and none carrying a SECURITY section or exposure statement: [OpenBSD `fingerd(8)`](https://man.openbsd.org/fingerd.8) — notably **not** removed, unlike `rlogind(8)`/`rexecd(8)`/`rsh(1)` · [FreeBSD `fingerd(8)`](https://man.freebsd.org/cgi/man.cgi?query=fingerd&sektion=8) · [GNU inetutils manual](https://www.gnu.org/software/inetutils/manual/inetutils.html), which ships no finger daemon at all · [Fedora `finger.spec`](https://src.fedoraproject.org/rpms/finger/raw/rawhide/f/finger.spec) (optional `finger-server` subpackage) · [Debian source search for `fingerd`](https://sources.debian.org/api/search/fingerd/) (no reference implementation packaged)
- [OpenBSD 3.2 changelog](https://www.openbsd.org/plus32.html) (removal of rlogin/rlogind/rexecd) · [OpenBSD 5.6 changelog](https://www.openbsd.org/plus56.html) (removal of rsh)
- [Microsoft, Security considerations for PowerShell Remoting using WinRM](https://learn.microsoft.com/en-us/powershell/scripting/security/remoting/winrm-security) · [Azure network security best practices](https://learn.microsoft.com/en-us/azure/security/fundamentals/network-best-practices)
- [HP Printing Security Best Practices](https://h10032.www1.hp.com/ctg/Manual/c05318850.pdf) — cited for the position that 9100 "should always be enabled"

Shipped configuration bytes (§12) — read as bytes at a release, never as a rendered documentation page
- Apache Cassandra `conf/cassandra.yaml`, from `apache-cassandra-5.0.2-bin.tar.gz` on [`archive.apache.org`](https://archive.apache.org/dist/cassandra/5.0.2/), extracted locally — *"For security reasons, you should not expose this port to the internet. Firewall it if needed."* above both `native_transport_port: 9042` and `rpc_address: localhost`. The [`cassandra-5.0.2`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.2/conf/cassandra.yaml) tag agrees byte-for-byte on both lines. **This is the measurement that moved 9042 out of §2.2's weak tier**
- PostgreSQL [`src/backend/utils/misc/postgresql.conf.sample`](https://raw.githubusercontent.com/postgres/postgres/REL_17_STABLE/src/backend/utils/misc/postgresql.conf.sample) (*"The commented-out settings shown in this file represent the default values."*; `#listen_addresses = 'localhost'`) and [`guc_tables.c`](https://raw.githubusercontent.com/postgres/postgres/REL_17_STABLE/src/backend/utils/misc/guc_tables.c) (`&ListenAddresses, "localhost"`)
- CouchDB [`rel/overlay/etc/default.ini`](https://raw.githubusercontent.com/apache/couchdb/3.4.2/rel/overlay/etc/default.ini) (`[chttpd]` → `bind_address = 127.0.0.1`, active; *"Upgrading CouchDB will overwrite this file."*) and [`local.ini`](https://raw.githubusercontent.com/apache/couchdb/3.4.2/rel/overlay/etc/local.ini). **Path hazard (§12.9):** `default.ini.tpl`, the name used in earlier releases, 404s at every current tag
- Redis [`redis.conf`](https://raw.githubusercontent.com/redis/redis/7.4/redis.conf) — `bind 127.0.0.1 -::1` active, *"So by default we uncomment the following bind directive"*, and a position: *"binding to all the interfaces is dangerous and will expose the instance to everybody on the internet"*
- RabbitMQ [`rabbitmq.conf.example`](https://raw.githubusercontent.com/rabbitmq/rabbitmq-server/v3.13.7/deps/rabbit/docs/rabbitmq.conf.example) — *"This file is AN EXAMPLE. It is NOT MEANT TO BE USED IN PRODUCTION."*, every directive commented
- ZooKeeper [`conf/zoo_sample.cfg`](https://raw.githubusercontent.com/apache/zookeeper/release-3.9.3/conf/zoo_sample.cfg) — no address or bind directive of any kind
- Kafka [`config/server.properties`](https://raw.githubusercontent.com/apache/kafka/3.8.0/config/server.properties) (`#listeners=PLAINTEXT://:9092`, commented) · Kibana [`config/kibana.yml`](https://raw.githubusercontent.com/elastic/kibana/v8.15.0/config/kibana.yml) (*"The default is 'localhost'"*)
- net-snmp [`EXAMPLE.conf.def`](https://raw.githubusercontent.com/net-snmp/net-snmp/master/EXAMPLE.conf.def) — *"An example configuration file"*, *"Some entries are deliberately commented out, and will need to be explicitly activated"*
- Debian packaging from [`deb.debian.org`](https://deb.debian.org/debian/pool/main/)'s pool as tarballs, extracted locally, because `sources.debian.org` and `salsa.debian.org` serve a proof-of-work challenge (§11.9): `rpcbind_1.2.7-1.debian.tar.xz` → `debian/rpcbind.default` (*"Uncomment the following line to restrict rpcbind to localhost only"*, **commented**) and `debian/rpcbind.socket` (`ListenStream=0.0.0.0:111`, **active**) · `memcached_1.6.45-1.debian.tar.xz` → `debian/memcached.conf` (`-l 127.0.0.1` and `-l ::1`, **active**)

Shipped configuration bytes (§13) — the footing-table re-derivation; every path resolved from the repository tree or tarball listing, never guessed
- Apache Cassandra `conf/cassandra.yaml` at [`cassandra-5.0.2`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.2/conf/cassandra.yaml), [`cassandra-5.0.6`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.6/conf/cassandra.yaml) and [`cassandra-5.0.9`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.9/conf/cassandra.yaml) — the prohibition appears **four times** in each, above `storage_port: 7000`, `ssl_storage_port: 7001`, `native_transport_port: 9042` and `rpc_address: localhost` (§13.5, [#75](https://github.com/winniel123/verge-asm/issues/75))
- rsync [`rsyncd.conf.5.md`](https://raw.githubusercontent.com/RsyncProject/rsync/v3.5.0/rsyncd.conf.5.md) — *"Do not expose a cleartext daemon to an untrusted network: front it with a TLS proxy … or run it over ssh"*, and the SSL/TLS Daemon Setup section putting the public listener on **874** with `127.0.0.1:873` behind it · [`packaging/systemd/rsync.service`](https://raw.githubusercontent.com/RsyncProject/rsync/v3.5.0/packaging/systemd/rsync.service), [`rsync@.service`](https://raw.githubusercontent.com/RsyncProject/rsync/v3.5.0/packaging/systemd/rsync%40.service), [`rsync.socket`](https://raw.githubusercontent.com/RsyncProject/rsync/v3.5.0/packaging/systemd/rsync.socket) (`ListenStream=873`) · [`README.md`](https://raw.githubusercontent.com/RsyncProject/rsync/v3.5.0/README.md) (*"generally used for public file distribution, although authentication and access control are available"* — §13.3)
- nfs-utils `nfs-utils-2.9.2.tar.xz` from [`kernel.org`](https://www.kernel.org/pub/linux/utils/nfs-utils/2.9.2/), extracted locally — `nfs.conf` (every setting commented, `# host=` included) · `utils/mount/nfs.man` (*"on a trusted physical network between trusted hosts, it is entirely adequate"*; *"NFS was developed to allow file sharing between systems residing on a local area network"*) · `utils/exportfs/exports.man` (*"exports the public FTP directory to every host in the world"* — a §12(b) label, §13.3)
- MongoDB [`rpm/mongod.conf`](https://raw.githubusercontent.com/mongodb/mongo/r8.3.8/rpm/mongod.conf) and [`debian/mongod.conf`](https://raw.githubusercontent.com/mongodb/mongo/r8.3.8/debian/mongod.conf) — `bindIp: 127.0.0.1` **active** in both, MongoDB Inc's own packaging
- Elasticsearch [`distribution/src/config/elasticsearch.yml`](https://raw.githubusercontent.com/elastic/elasticsearch/v9.5.1/distribution/src/config/elasticsearch.yml) (*"By default Elasticsearch is only accessible on localhost"*, `#network.host` commented) **and** [`distribution/docker/src/docker/config/elasticsearch.yml`](https://raw.githubusercontent.com/elastic/elasticsearch/v9.5.1/distribution/docker/src/docker/config/elasticsearch.yml) (`network.host: 0.0.0.0`, **active**) — one owner, two operative defaults, opposite directions (§13.3)
- MySQL [`packaging/deb-in/extra/mysqld.cnf`](https://raw.githubusercontent.com/mysql/mysql-server/mysql-9.7.2/packaging/deb-in/extra/mysqld.cnf), [`my.cnf.fallback`](https://raw.githubusercontent.com/mysql/mysql-server/mysql-9.7.2/packaging/deb-in/extra/my.cnf.fallback), [`mysql.cnf`](https://raw.githubusercontent.com/mysql/mysql-server/mysql-9.7.2/packaging/deb-in/extra/mysql.cnf), [`packaging/rpm-common/my.cnf.in`](https://raw.githubusercontent.com/mysql/mysql-server/mysql-9.7.2/packaging/rpm-common/my.cnf.in) — no `bind-address`, no security prose in any of the four
- memcached [`scripts/memcached.sysconfig`](https://raw.githubusercontent.com/memcached/memcached/1.6.45/scripts/memcached.sysconfig) (`OPTIONS=""`, upstream's own RPM `Source1`) — upstream ships no `memcached.conf`; the only file of that name in the tree is `t/sasl/memcached.conf`, a test fixture
- Docker [`contrib/init/systemd/docker.socket`](https://raw.githubusercontent.com/moby/moby/docker-v29.7.2/contrib/init/systemd/docker.socket) (`ListenStream=/run/docker.sock`) and [`docker.service`](https://raw.githubusercontent.com/moby/moby/docker-v29.7.2/contrib/init/systemd/docker.service) (`-H fd://`) — no TCP listener in the operative default
- PostgreSQL [`pg_hba.conf.sample`](https://raw.githubusercontent.com/postgres/postgres/REL_18_STABLE/src/backend/libpq/pg_hba.conf.sample) — shipped `host` records are `127.0.0.1/32` and `::1/128` only, under *"If you want to allow non-local connections, you need to add more \"host\" records."* A second restricting default, never opened before §13 · [`postgresql.conf.sample`](https://raw.githubusercontent.com/postgres/postgres/REL_18_STABLE/src/backend/utils/misc/postgresql.conf.sample) at `REL_18_STABLE`, agreeing with §12's `REL_17_STABLE` reading
- CouchDB [`rel/overlay/etc/default.ini`](https://raw.githubusercontent.com/apache/couchdb/3.5.0/rel/overlay/etc/default.ini) at `3.5.0` — `bind_address = 127.0.0.1` **active** under `[chttpd]`, `[httpd]` and `[prometheus]`
- ZooKeeper [`conf/zoo_sample.cfg`](https://raw.githubusercontent.com/apache/zookeeper/release-3.9.5/conf/zoo_sample.cfg) at `release-3.9.5` · RabbitMQ [`rabbitmq.conf.example`](https://raw.githubusercontent.com/rabbitmq/rabbitmq-server/v4.3.4/deps/rabbit/docs/rabbitmq.conf.example) at `v4.3.4` — both confirm §12's readings at a later release
- etcd [`etcd.conf.yml.sample`](https://raw.githubusercontent.com/etcd-io/etcd/v3.7.1/etcd.conf.yml.sample) (*"This is the configuration file for the etcd server."* — the tenth artefact and the first that does not declare itself, §13.6) and [`contrib/systemd/etcd.service`](https://raw.githubusercontent.com/etcd-io/etcd/v3.7.1/contrib/systemd/etcd.service) (runs `/usr/bin/etcd` with no `--config-file`)

Shipped configuration and compiled bytes (§14) — the `7000`/`7001` ruling; every path resolved from the repository tree, every configuration file read end to end per [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md)
- Apache Cassandra `conf/cassandra.yaml` at [`cassandra-5.0.2`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.2/conf/cassandra.yaml), [`cassandra-5.0.6`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.6/conf/cassandra.yaml), [`cassandra-5.0.9`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.9/conf/cassandra.yaml) — the prohibition above `storage_port: 7000` and above `ssl_storage_port: 7001`; `listen_address: localhost` **active** under *"You \_must\_ change this if you want multiple nodes to be able to communicate!"* and *"Setting listen_address to 0.0.0.0 is always wrong"*; `#internode_authenticator:` / `# class_name: org.apache.cassandra.auth.AllowAllInternodeAuthenticator` **commented**; `internode_encryption: none` and `legacy_ssl_storage_port_enabled: false` **active**. **The `7001` deprecation is the decisive line:** *"As of cassandra 4.0, this property is deprecated as a single port can be used for either/both secure and insecure connections."*
- Apache Cassandra [`conf/cassandra_latest.yaml`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.9/conf/cassandra_latest.yaml) at `cassandra-5.0.9` — **the second operative default the file's own header announces**, retrieved because §13.10 measured that an owner can ship two that contradict each other. It agrees on every line §14 relies on (§14.6)
- Apache Cassandra compiled defaults at `cassandra-5.0.9`: [`config/Config.java`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.9/src/java/org/apache/cassandra/config/Config.java) (`public int storage_port = 7000;`, `public int ssl_storage_port = 7001;`) · [`auth/AuthConfig.java`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.9/src/java/org/apache/cassandra/auth/AuthConfig.java) (`authInstantiate(conf.internode_authenticator, IInternodeAuthenticator.class, AllowAllInternodeAuthenticator.class)`) · [`config/DatabaseDescriptor.java`](https://raw.githubusercontent.com/apache/cassandra/cassandra-5.0.9/src/java/org/apache/cassandra/config/DatabaseDescriptor.java) (`internodeAuthenticator = new AllowAllInternodeAuthenticator()`)
- [Cassandra security](https://cassandra.apache.org/doc/latest/cassandra/managing/operating/security.html) — *"Malicious users able to access internode communication and JMX ports can still: Craft internode messages to insert users into authentication schema · Craft internode messages to truncate or drop schema · Use tools such as sstableloader to overwrite system_auth tables · Attach to the cluster directly to capture write traffic"* — Claim 1's §10.1 Step 2 limb · [Cassandra 5.0 FAQ](https://cassandra.apache.org/doc/5.0/cassandra/overview/faq/index.html) — *"By default, Cassandra uses 7000 for cluster communication (7001 if SSL is enabled), 9042 for native protocol clients, and 7199 for JMX."*
- **[Apple, *TCP and UDP ports used by Apple software products*](https://support.apple.com/en-us/103229)** — tabulates `AirPlay · 7000 · TCP`. **The fact that refuses `7000/tcp`**: the competing service's own vendor, documenting its own product on the number
- **Oracle WebLogic Server on `7001`** — [`oracle/docker-images`, `OracleWebLogic/dockerfiles/14.1.2.0/README.md`](https://github.com/oracle/docker-images/blob/main/OracleWebLogic/dockerfiles/14.1.2.0/README.md), Oracle's own bytes: `ADMIN_LISTEN_PORT` *(default: `7001`)* and `docker run -d -p 7001:7001 …`. Cross-checked against [Oracle's WebLogic administration guide](https://docs.oracle.com/cd/E13222_01/wls/docs81/adminguide/network.html) — *"one port for HTTP communication (7001 by default), and one port for HTTPS communication (7002 by default)"*. **Retrieval hazard (§14.6):** `docs.oracle.com`'s current Fusion Middleware port-numbers appendix serves a JavaScript shell to a plain client, in the `learn.microsoft.com` shape §9.5 recorded
- IANA registry CSV, retrieved 2026-08-14 — `afs3-fileserver,7000,tcp,file server itself` with an **empty** Unauthorized Use field, and `afs3-callback,7001,tcp,callbacks to cache managers,…,Known Unauthorized Use on port 7001`. The annotation is used only as evidence about **what else listens on the port**, per §9.3.4
- `nmap-services` at `nmap/nmap` `master` — **rank and name column only**, on §2.5's and §6.1's existing terms: `afs3-fileserver 7000/tcp 0.001995` (**146th** of 8,387 TCP rows) and `afs3-callback 7001/tcp 0.000891` (**232nd**), against a top-100 boundary at `0.003149`. Neither is in the top-100, so neither entered `verge-core`'s frequency half that way
- [`safe-active-probing.md`](./safe-active-probing.md) §2.3 — **this project's own** modern-services supplement, which lists `7001` under *"HTTP-ish alternates"*. Recorded in §14.4 as corroboration and as a product-coherence defect, **never as grounds**, because §6 forbids the frequency half deciding a normative question

Placement statements (§15) — the determinacy source rule and the walk. **Read as documents for a sentence, not as bytes at a tag**, and the negatives are searched rather than exhausted (§15.7)
- **Red Hat, RHEL web console (Cockpit) on `9090`** — [Managing systems using the RHEL 9 web console, ch. 1](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/managing_systems_using_the_rhel_9_web_console/getting-started-with-the-rhel-9-web-console_system-management-using-the-rhel-9-web-console) and the [RHEL 8 equivalent](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/8/html/managing_systems_using_the_rhel_8_web_console/getting-started-with-the-rhel-8-web-console_system-management-using-the-rhel-8-web-console) — the console *communicates through TCP port 9090*. **The artefact that re-founds §4.3's `9090` exclusion**, replacing an `nmap-services` name-column citation that limb 5 makes inadmissible as grounds
- **OpenSearch on `9200`** — [Network settings](https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/network-settings/), `http.port` defaulting to the `9200-9300` range. A **different vendor** on a listed row's number, declaring the row's own protocol: limb 3's first instance, and the reason limb 3 exists
- **ScyllaDB on `9042`** — [Configuration parameters](https://docs.scylladb.com/manual/stable/reference/configuration-parameters.html), `native_transport_port` defaulting to `9042`, declaring **CQL**. Limb 3's second instance
- **Open Mobile Alliance, *Wireless Session Protocol* `OMA-WAP-TS-WSP-V1_0-20110315-A`** — [openmobilealliance.org](https://www.openmobilealliance.org/release/Browser_Protocol_Stack/V2_1-20110315-A/OMA-WAP-TS-WSP-V1_0-20110315-A.pdf). Cited for what it **is not**: an archived release of a suite no party currently ships, so under limb 4 it places nothing and `9200/tcp` stays uncontested (§15.3). **Not retrieved:** OMA's own WDP specification, which is the primary for the WDP/UDP bearer-port claim §15.3 rests its second ground on
- **RFC 4146** — already in the standards list above, and **re-cited here as the carrying artefact for `79/tcp`'s determinacy limb**, with IANA's `Unauthorized use by some mail users` annotation demoted to corroboration (§15.5, amending §9.3.4's footing rather than its verdict)
- Walked and found to carry **no competing placement statement** (§15.4): [Kubernetes ports and protocols](https://kubernetes.io/docs/reference/networking/ports-and-protocols/) for `10250` and `10255` · [RabbitMQ networking](https://www.rabbitmq.com/docs/networking) for `25672`, derived as `NODE_PORT + 20000` · [ZooKeeper Administrator's Guide](https://zookeeper.apache.org/doc/r3.9.3/zookeeperAdmin.html) for `2181` · [xhost(1), X.Org](https://www.x.org/releases/current/doc/man/man1/xhost.1.xhtml) for `6000`
Shipped bytes (§16) — the footing table's seven uncovered rows; every path resolved from the repository tree at a named tag or release branch, and where a rendered page is quoted it is quoted from its **source file** in the project's own docs repository
- **etcd [`THREAT_MODEL.md`](https://github.com/etcd-io/etcd/blob/v3.7.1/THREAT_MODEL.md) at `v3.7.1`** — **the document that decided this ticket, and it is not where §3.4's citation points.** *"etcd Server assumes it is deployed within a strictly isolated, private network segment. It **must not** be exposed to untrusted networks or the public internet."* · *"etcd clients communicate with etcd Servers over Port 2379."* · *"etcd Server members communicate with other cluster members over Port 2380 to run Raft consensus."* Present at [`v3.7.0`](https://github.com/etcd-io/etcd/blob/v3.7.0/THREAT_MODEL.md), **absent at `v3.6.14` and `v3.5.33`**; the Network Boundary paragraph is byte-identical at `v3.7.0`, `v3.7.1` and `main`. Created 2026-05-19 as *"Define THREAT_MODEL for etcd to decentivize agents reporting CVEs outside it"* — the purpose argument §16.3 states in full and refuses
- etcd [`etcd.conf.yml.sample`](https://raw.githubusercontent.com/etcd-io/etcd/v3.7.1/etcd.conf.yml.sample) at `v3.7.1` — re-read (§13.6's artefact); `listen-peer-urls: http://localhost:2380` and `listen-client-urls: http://localhost:2379`, a restricting default that now **corroborates** a prohibition rather than standing alone
- **Elasticsearch [`docs/reference/elasticsearch/configuration-reference/networking-settings.md`](https://github.com/elastic/elasticsearch/blob/v9.5.1/docs/reference/elasticsearch/configuration-reference/networking-settings.md) at `v9.5.1`** — *"Never expose an unprotected node to the public internet"* in a `warning` admonition, three lines below *"Each Elasticsearch node has two different network interfaces"* and *"By default Elasticsearch binds only to `localhost`"*; `network.host` *"Sets the address of this node for **both HTTP and transport traffic**"*; `transport.port` *"Defaults to `9300-9400`"*. **The owner's own page is what links the *node* sentence to `9300`**
- Elasticsearch [`distribution/src/config/elasticsearch.yml`](https://github.com/elastic/elasticsearch/blob/v9.5.1/distribution/src/config/elasticsearch.yml) at `v9.5.1` — re-read (§13.1's artefact); **contains no prose about the transport interface or `9300` at all**, which is why this row needed the docs source
- **Kubernetes [`staging/src/k8s.io/kubelet/config/v1beta1/types.go`](https://github.com/kubernetes/kubernetes/blob/v1.34.1/staging/src/k8s.io/kubelet/config/v1beta1/types.go) at `v1.34.1`** — `readOnlyPort` *"Setting this field to 0 disables the read-only service. **Default: 0 (disabled)**"*. The restricting default that puts `10255` in the weak tier
- Kubernetes [`pkg/kubelet/apis/config/v1beta1/defaults.go`](https://github.com/kubernetes/kubernetes/blob/v1.34.1/pkg/kubelet/apis/config/v1beta1/defaults.go) and [`cmd/kubelet/app/options/options.go`](https://github.com/kubernetes/kubernetes/blob/v1.34.1/cmd/kubelet/app/options/options.go) at `v1.34.1` — `obj.Address = "0.0.0.0"` (permissive, therefore silent under §10.4) · `obj.Authentication.Anonymous.Enabled = ptr.To(false)` · `obj.Authorization.Mode = …KubeletAuthorizationModeWebhook`, with each flag registered against the already-defaulted struct value. **Routed to [#83](https://github.com/winniel123/verge-asm/issues/83)** as a Claim 1 question
- Kubernetes [`content/en/docs/reference/networking/ports-and-protocols.md`](https://github.com/kubernetes/website/blob/release-1.34/content/en/docs/reference/networking/ports-and-protocols.md) at `release-1.34` — `10250` `Kubelet API` `Used By: Self, Control plane`, in both the control-plane and worker-node tables. **`10255` does not appear.** §16.9 flags this as the thinnest cell in the tier, because it is a table cell rather than a sentence naming a network
- Kubernetes [`content/en/docs/reference/command-line-tools-reference/kubelet.md`](https://github.com/kubernetes/website/blob/release-1.34/content/en/docs/reference/command-line-tools-reference/kubelet.md) at `release-1.34` — the **generated** page §3.4 cites, still carrying `--read-only-port int32  Default: 10255`, `--anonymous-auth  Default: true` and `--authorization-mode string  Default: "AlwaysAllow"`. Retrieved as the *contradicted* artefact, not as evidence; §10.4's one-way rule makes the permissive one silent without adjudication (§16.5)
- **Erlang/OTP [`system/doc/reference_manual/distributed.md`](https://github.com/erlang/otp/blob/OTP-29.0.5/system/doc/reference_manual/distributed.md) at `OTP-29.0.5`** — *"When using insecure distributed nodes, make sure that the network is configured to keep potential attackers out."* What actually carries `4369`, in place of RabbitMQ's non-owner sentence (§10.5, §16.6)
- Erlang/OTP [`erts/doc/references/epmd_cmd.md`](https://github.com/erlang/otp/blob/OTP-29.0.5/erts/doc/references/epmd_cmd.md) at `OTP-29.0.5`, section *Access Restrictions* — *"only the query commands are answered … if the query comes from a remote host"* · *"To restrict access further, firewall software must be used."* · `-address` / `ERL_EPMD_ADDRESS` are **opt-in**, so epmd's default is permissive and silent. **No cookie appears in epmd's remote exchange**, which is the §3.1 *"why"*-cell defect routed to [#84](https://github.com/winniel123/verge-asm/issues/84)
The class audit (§17) — the document classes nobody had opened, retrieved as bytes and searched directly
- **The IETF's operational class for SNMP**, opened for the first time by [#79](https://github.com/winniel123/verge-asm/issues/79) and citable chiefly for the absence of a placement sentence: [RFC 3512, *Configuring Networks and Devices with SNMP*](https://www.rfc-editor.org/rfc/rfc3512.txt) (Informational, April 2003) — §6.2–§6.4, the SNMP family's own deployment guide; the placement strings §11.1 counted occur **five** times and not one describes where an agent belongs · [RFC 3871, *Operational Security Requirements for Large ISP IP Network Infrastructure*](https://www.rfc-editor.org/rfc/rfc3871.txt) (Informational, September 2004) — §2.2, *"There are many situations where in-band management makes sense, is used, and/or is the only option"*, and §9's boilerplate *"deployment of SNMP versions prior to SNMPv3 is NOT RECOMMENDED"*, a **version** statement whose remedy is on 161 · [RFC 4778, *Operational Security Current Practices in Internet Service Provider Environments*](https://www.rfc-editor.org/rfc/rfc4778.txt) (Informational, January 2007) — §2.3, §2.4. **A survey, not a position**: *"In all large ISPs that were interviewed"*, *"SNMPv2 is primarily deployed since it is easier to set up than v3"*. The most placement-shaped title in the corpus and the clearest instance of frequency in a deployment document
- **[RabbitMQ production checklist](https://www.rabbitmq.com/docs/production-checklist)**, *Firewall Configuration* — the deployment class, never opened. Divides RabbitMQ's ports into client-library ports and everything else, restricts the second category to *"hosts running RabbitMQ nodes or CLI tools"*, and says the first *"should be accessible to hosts that run applications, which in some cases can mean **public networks**, for example, behind a load balancer."* **Turns §4.6's `5672`/`15672` negative into a positive** (§17.4). Retrieval hazard: `www.rabbitmq.com` returns HTTP 403 to a plain client and 200 with a browser user-agent
- **Apache Kafka**, read from `raw.githubusercontent.com` at `trunk` and at release tag `4.3.1` because `kafka.apache.org` serves a JavaScript shell (§17.10): [`docs/security/security-model.md`](https://raw.githubusercontent.com/apache/kafka/trunk/docs/security/security-model.md) — *"Security is off by default … This is appropriate only for closed test environments. Production deployments **must** explicitly configure authentication, authorization, and transport encryption before being exposed to any untrusted network."* **[measured] Present on `trunk`; HTTP 404 at tag `4.3.1`; HTTP 404 at `kafka.apache.org/documentation/security/security-model/`** · [`docs/security/security-overview.md`](https://raw.githubusercontent.com/apache/kafka/4.3.1/docs/security/security-overview.md) at tag `4.3.1`, 2,068 B — **the shipped document**, still carrying *"security is optional - non-secured clusters are supported"*, the sentence §4.6 quotes · `docs/security/listener-configuration.md` and `docs/operations/multi-tenancy.md` — read end to end per [ADR-0037](../adr/0037-an-attestation-is-retrieved-over-the-artefact-not-over-the-row.md) limb 1; **[measured]** the string `internet` occurs **zero** times across all four files
- **[CouchDB documentation, *Security*](https://docs.couchdb.org/en/stable/intro/security.html)** §1.5.1.2 — the class never opened for `5984/tcp`, and the corpus's nearest miss: *"Will your CouchDB instance communicate over a public network? Even a LAN shared with other collocation customers is public."* The second sentence defines what the first means by *public*, and it is not the internet (§17.6)
- **Kubernetes' security documentation**, the class §16.5 did not open for `10255/tcp`: [`kubernetes/website`, `content/en/docs/concepts/security/security-checklist.md`](https://raw.githubusercontent.com/kubernetes/website/release-1.34/content/en/docs/concepts/security/security-checklist.md) at `release-1.34` — *"The Kubernetes API, kubelet API and etcd are not exposed publicly on Internet"*, and *"The kubelet API access should be restricted and not exposed publicly"*. **The owner naming a boundary about its own component, and naming it by category rather than by port** (§17.6) · [`controlling-access.md`](https://raw.githubusercontent.com/kubernetes/website/release-1.34/content/en/docs/concepts/security/controlling-access.md) at the same tag — read end to end, and it contains no port and no placement statement
- **Named as the boundary and not searched, because the row's other gate is untested**: Elastic's deployment guidance for Kibana and HashiCorp's for Consul. §17.8 and §8 question 11 — routed to [#87](https://github.com/winniel123/verge-asm/issues/87)

No shipped configuration artefact exists (§13.1) — nothing to read, so §13 adds no evidence about these rows in either direction
- **1433/tcp** Microsoft SQL Server and **445/tcp** SMB with **139/tcp**, **137**, **138/udp** — configured through setup and the registry rather than through a file Microsoft ships
- **623/udp** IPMI — a BMC's configuration is firmware

Checked and found to contain **no** position, which is itself the finding
- `rpcbind(8)` — no security section, no exposure statement
- [Apache Kafka security overview](https://kafka.apache.org/43/security/security-overview/) — "security is optional - non-secured clusters are supported"
- PostgreSQL `ssl-tcp.html` and `client-authentication.html` — no statement on network placement (§4.5)
- **[etcd.io `v3.6/op-guide/security`](https://github.com/etcd-io/website/blob/main/content/en/docs/v3.6/op-guide/security.md)**, `etcd-io/website` `main` — the page §3.4 cites for etcd. A sweep for `internet`, `untrusted`, `expose`, `firewall`, `trusted network` and `public` returns the **consequence** sentence *"can expose its data to any clients"*, TLS certificate mechanics, and **no position**. §13.7's hypothesis was correct about this page and wrong about the project (§15.3)
- Kubernetes [`content/en/docs/reference/access-authn-authz/kubelet-authn-authz.md`](https://github.com/kubernetes/website/blob/main/content/en/docs/reference/access-authn-authz/kubelet-authn-authz.md) — nothing about network placement for either kubelet port; the `10255` search across `kubernetes/website` returns eight files and no position (§15.10)

Consulted and deliberately not used as evidence
- `nmap-services` open-frequency data — frequency, and 2008-vintage. Used in §6.1 only to state where a port ranks, never to justify a verdict
- Shodan / Censys internet-exposure studies — frequency. Named as candidates by the ticket; not used anywhere in this note
- [CISA TA14-017A, UDP-Based Amplification Attacks](https://www.cisa.gov/news-events/alerts/2014/01/17/udp-based-amplification-attacks) — a magnitude source (portmap's bandwidth amplification factor is given as "7 to 28"), not a position. It is the most tempting laundering candidate in the corpus and §2.7 records why it was refused
- Redis's protected-mode prevalence rhetoric — §2.5

Named in §10 and **not retrieved** — recorded so the gap is visible rather than assumed closed
- **RFC 3411 and the SNMP management-architecture family.** §10.6 finds `161/udp`'s Claim 3 boundary
  limb resting on CISA TA17-156A, a corroborator. A first-party IETF sentence placing SNMPv1/v2c
  inside a management domain may well exist; **nobody has looked**, and §10.6 declines to move the row
  on a re-reading of text already in this note. The retrieval is
  [#66](https://github.com/winniel123/verge-asm/issues/66)
- **RFC 3912 (WHOIS) and RFC 9110 (HTTP semantics).** Cited in §10.1's refusal table for what their
  statements of purpose are, not quoted. Neither is load-bearing for a row: 43/tcp is refused by the
  publication test on any reading, and 80/443 fail determinacy independently
- **Shipped bind-address defaults for the eighteen rows tabulated in §10.4.1.** Compiled to size the
  option that lost, not to carry any row. The rule the walk establishes — a permissive default is
  silent — means **no verdict depends on any entry in that list being exact**, which is why it is
  presented as a measurement of the losing option's blast radius rather than as evidence
