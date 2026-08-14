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
| The list | **38 `(port, transport)` pairs** in three classes — §3. **Superseded by §11 — the list is 37 pairs; `161/udp` is removed** |
| Evidence standard | A **named claim** from three permitted claims, **attested** by the source that owns it, plus a **determinacy** gate — §2. **Amended by §12 — an example config attests nothing, and a distributor's shipped default corroborates and never carries a row** |
| Cloud-provider and government port lists | **Corroboration only, never sole grounds.** They are risk lists, not never-lists, and they contradict each other — §2.3 |
| Management planes inside a VPC | **Not a problem for the list.** `Exposure` is defined from an internet vantage, so the vantage does the relativising and the list can be absolute — §4.1 |
| Does TLS change a verdict | **No.** TLS bears on one of the three claims and never on the other two — §4.2 |
| High ports that are conventionally anything | **Excluded by the determinacy gate**, which is a gate on the *port*, not on the service — §4.3 |
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
9. **Is `161/udp`'s Claim 3 boundary attested by an owner?** Opened by §10.6. The row's boundary limb
   — *SNMPv1/v2c assumes a management network* — is carried in this note by CISA TA17-156A, a
   corroborator. RFC 3410 §8.2 attests the *insecurity* and not the *boundary*. Routed to
   [#66](https://github.com/winniel123/verge-asm/issues/66), because it is a retrieval question and a
   row may not move on a re-reading.

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
  rows** whose shipped configuration bytes have never been read.
- **PostgreSQL was checked three ways because its footing is the note's weakest row** — the sample
  file, the sample file's own statement about what a commented line means, and the compiled-in default
  in `guc_tables.c`. All three agree on `localhost`.

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

Checked and found to contain **no** position, which is itself the finding
- `rpcbind(8)` — no security section, no exposure statement
- [Apache Kafka security overview](https://kafka.apache.org/43/security/security-overview/) — "security is optional - non-secured clusters are supported"
- PostgreSQL `ssl-tcp.html` and `client-authentication.html` — no statement on network placement (§4.5)

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
