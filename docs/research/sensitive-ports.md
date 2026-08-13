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
| The list | **36 `(port, transport)` pairs** in three classes — §3 |
| Evidence standard | A **named claim** from three permitted claims, **attested** by the source that owns it, plus a **determinacy** gate — §2 |
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
| 1521/tcp | `ncube-lm` — "nCube License Manager", annotated **"Unauthorized Use Known on port 1521"** | Oracle DB listener |
| 10250/tcp, 10255/tcp, 9042/tcp, 6000/tcp, 15672/tcp | **not registered at all** | kubelet, Cassandra, X11, RabbitMQ mgmt |

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

### 2.6 The category route, stated explicitly

Two rows on the list are carried by a syllogism rather than a single quote, and it is better to name
the pattern than to let it hide:

1. The protocol's **own specification** attests that it *is* a network-management or out-of-band
   management interface (category membership).
2. **CISA CPG 3.S** attests that interfaces in that category should never be exposed
   (category verdict).

This is a legitimate second attestation route because CISA is not being asked to know anything about
ports — only about the category, which is the thing it actually has a position on. It carries
`161/udp` (SNMP) and `623/udp` (IPMI/BMC), and nothing else.

---

## 3. The list

**36 `(port, transport)` pairs.** The `reg.` column records whether IANA registers the port to the
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

### 3.2 Class B — credentials in cleartext, encrypted successor on another port (Claim 2)

| Port/transport | Service implied | reg. | Why internet exposure is never correct |
|---|---|---|---|
| 23/tcp | Telnet | yes | Username, password and the entire session travel unprotected; SSH on 22 is the standardised replacement |
| 21/tcp | FTP control | yes | `PASS` sends the password in clear text; SFTP and FTPS are the standardised replacements |
| 512/tcp | rexec | yes | IANA's own registry describes it as "remote process execution; authentication performed using passwords and UNIX login names" |
| 513/tcp | rlogin | yes | Cleartext credentials over an untrusted path; superseded by SSH |
| 514/tcp | rsh | yes | Cleartext, and trust is host-based rather than credential-based; superseded by SSH |
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
| 161/udp | SNMP | yes | A network-management interface by its own specification, and SNMPv1/v2c authenticate on cleartext community strings (category route, §2.6) |
| 623/udp | IPMI / ASF-RMCP (BMC) | yes | An out-of-band server management interface — the exact class CISA says should never be directly accessible via the public internet (category route, §2.6) |

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

**Class B.**

> "The Telnet protocol normally uses passwords in the clear for authentication, and normally offers
> no privacy. In normal telnet, both the user's identity and their password are exposed without any
> protection; after that, the contents of the entire Telnet session is exposed without any
> protection."
> — [RFC 4248, The telnet URI Scheme](https://www.rfc-editor.org/rfc/rfc4248.txt), §3

> "Standard FTP [PR85] sends passwords in clear text using the "PASS" command."
> — [RFC 2577, FTP Security Considerations](https://www.rfc-editor.org/rfc/rfc2577.txt), §5

> "This type of authentication is known to be cryptographically weak and is not intended for use on
> untrusted networks. Many implementations will want to use stronger security, such as running the
> session over an encrypted channel provided by IPsec [RFC4301] or SSH [RFC4254]."
> — [RFC 6143, The Remote Framebuffer Protocol](https://www.rfc-editor.org/rfc/rfc6143.txt), §7.2.2. §7.2.1 also defines a "None" security type: "No authentication is needed."

> "The cookie is transmitted on the network without encryption, so there is nothing to prevent a
> network snooper from obtaining the data and using it to gain access to the X server."
> — [Xsecurity(7)](https://man.openbsd.org/Xsecurity.7), on MIT-MAGIC-COOKIE-1, which the same page lists as "Shared plain-text "cookies""

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

For the two category-route rows:

> "Protocol operations via SNMPv1 and SNMPv2c message wrappers support only trivial authentication
> based on plain-text community strings and, as a result, are fundamentally insecure. When the
> SNMPv3 specifications for security and administration, which include strong security, reached full
> Standard status, the full Standard SNMPv1 … and the experimental SNMPv2c specifications … were
> declared Historic due to their weaknesses with respect to security"
> — [RFC 3410, Introduction and Applicability Statements for Internet-Standard Management Framework](https://www.rfc-editor.org/rfc/rfc3410.txt), §8.2

> "These out of band interfaces should never be directly accessible via the public internet."
> — [CISA BOD 23-02 implementation guidance](https://www.cisa.gov/news-events/directives/bod-23-02-implementation-guidance-mitigating-risk-internet-exposed-management-interfaces), on the iLo/iDRAC class that 623/udp serves

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
| **3389/tcp RDP** | Same category. CISA names it "high-risk" and Microsoft recommends JIT access — both are risk positions, not legitimacy positions. GCP ships it open by default |
| **5985, 5986/tcp WinRM** | Same category. Azure's JIT list names them, but that is the sole evidence, and §2.3 forbids a cloud-provider list carrying a row alone |
| **6443/tcp kube-apiserver** | §4.4 |
| **5601/tcp Kibana** | Elastic states no prohibition; Kibana is routinely and legitimately fronted on the internet behind auth. Its evidence is a secure default only, and it squats on `esmagent` |
| **8500/tcp Consul HTTP API** | Consul "is not secure-by-default" and ships `acl.default_policy` = `"allow"`, but its stated position is only that external access "should be considered", and 8500 is registered to `fmtp` |
| **9092/tcp Kafka** | Upstream declines to take any network posture. Its only relevant sentence is neutral: "security is optional - non-secured clusters are supported" |
| **5672, 15672/tcp RabbitMQ AMQP + management UI** | Upstream's "should not be publicly exposed" sentence covers 4369 and 25672 specifically, not these. AMQP brokers are sometimes legitimately public |
| **8080/tcp Jenkins** | Upstream explicitly acknowledges public-internet deployment as supported: "Jenkins is used everywhere from workstations on corporate intranets, to high-powered servers connected to the public internet", and responds with auth-by-default rather than a network demand |
| **1099/tcp Java RMI registry** | Contrary to reputation, modern JDK defaults are secure — `jmxremote.ssl` and `jmxremote.authenticate` both default to `true` |
| **Hadoop NameNode / YARN UIs** | Hadoop's default `simple` authentication would qualify under Claim 1, but the web UI port moved from 50070 to 9870 between major versions, so port-to-service inference is version-dependent. Fails determinacy |
| **110/tcp POP3, 143/tcp IMAP, 25/tcp SMTP** | Mail protocols whose intended audience genuinely is the internet. Cleartext variants are deprecated, but a server on 143 offering STARTTLS is correct, so the *port* is not the discriminator |
| **111/tcp rpcbind, 2049/tcp NFS, 873/tcp rsync, 389/tcp LDAP, 79/tcp finger** | Plausible members for which no verbatim primary attestation was obtained in this pass. Deliberately excluded rather than admitted on reputation — see §8 |

The last row is the discipline the standard exists to enforce. Every one of those five is a port a
practitioner would casually call "never internet-facing", and four of them sit in the nmap top-100.
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

**In the hot set already — 26 of 36:** 21, 23, 139, 445, 513, 514, 2181, 2375, 2376, 2379, 2380,
3306, 5432, 1433, 5900, 5984, 6000, 6379, 9042, 9200, 9300, 10250, 10255, 11211/tcp, 27017, 27018.

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
4. **Do the five ports excluded for want of attestation deserve a second pass?** 111/tcp rpcbind,
   2049/tcp NFS, 873/tcp rsync, 389/tcp LDAP, 79/tcp finger. Four are in the nmap top-100 and all
   five are plausible. They were excluded on evidence discipline, not on judgement, and a targeted
   pass for primary attestations could admit them — but that pass must happen **before** v1 ships,
   because adding them later costs a comparability cycle (§7.2).
5. **Should the `reg.` disclosure surface in the product?** Nine rows rest on convention rather than
   IANA registration. When the signal fires on 9200 it is asserting Elasticsearch on a port
   registered to `wap-wsp`. The evidence the signal cites should probably say so.
6. **Does this note's evidence standard generalise to the other v1 signals?** The claim/attestation/
   determinacy structure was built for the one signal with curated reference data, but "state the
   claim, cite the source that owns it" is not specific to ports.

---

## Sources

Government and standards bodies
- [CISA BOD 23-02, Mitigating the Risk from Internet-Exposed Management Interfaces](https://www.cisa.gov/news-events/directives/binding-operational-directive-23-02) (13 June 2023) · [implementation guidance / FAQ](https://www.cisa.gov/news-events/directives/bod-23-02-implementation-guidance-mitigating-risk-internet-exposed-management-interfaces)
- [CISA Cross-Sector Cybersecurity Performance Goals v2.0](https://www.cisa.gov/sites/default/files/2025-12/CPG_Report_2.0_508c.pdf) (December 2025), goal 3.S — verified against the PDF's own text
- [CISA BOD 22-01 (Revoked)](https://www.cisa.gov/news-events/directives/bod-22-01-reducing-significant-risk-known-exploited-vulnerabilities) — revoked 10 June 2026, superseded by BOD 26-04
- [CISA/US-CERT, SMB Security Best Practices](https://www.cisa.gov/news-events/alerts/2017/01/16/smb-security-best-practices)
- [CISA AA22-137A, Weak Security Controls and Practices Routinely Exploited for Initial Access](https://www.cisa.gov/news-events/cybersecurity-advisories/aa22-137a)
- [CISA #StopRansomware Guide v3.0](https://www.cisa.gov/sites/default/files/2025-03/StopRansomware-Guide%20508.pdf)
- [NSA/CISA Kubernetes Hardening Guidance v1.2](https://media.defense.gov/2022/Aug/29/2003066362/-1/-1/0/CTR_KUBERNETES_HARDENING_GUIDANCE_1.2_20220829.PDF) (August 2022)
- [IANA Service Name and Transport Protocol Port Number Registry](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml) — registry data retrieved as [CSV](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.csv), last updated 2026-08-11
- [RFC 1350 (TFTP)](https://www.rfc-editor.org/rfc/rfc1350.txt) · [RFC 2577 (FTP Security Considerations)](https://www.rfc-editor.org/rfc/rfc2577.txt) · [RFC 3410 (SNMP framework)](https://www.rfc-editor.org/rfc/rfc3410.txt) · [RFC 4248 (telnet URI scheme)](https://www.rfc-editor.org/rfc/rfc4248.txt) · [RFC 6143 (RFB / VNC)](https://www.rfc-editor.org/rfc/rfc6143.txt) · [RFC 6335 (port registry procedures)](https://www.rfc-editor.org/rfc/rfc6335.txt)
- [CIS Amazon Web Services Foundations Benchmark v1.2.0](https://d1.awsstatic.com/whitepapers/compliance/AWS_CIS_Foundations_Benchmark.pdf) (05-23-2018) — the CIS-authored PDF, ungated. Current versions are behind a registration form at [learn.cisecurity.org/benchmarks](https://learn.cisecurity.org/benchmarks); the "remote server administration ports" phrasing in v3.0.0+ was **not** verified against a CIS document and is second-hand via AWS's mapping page

Cloud providers
- AWS: [default security group](https://docs.aws.amazon.com/vpc/latest/userguide/default-security-group.html) · [Trusted Advisor security checks](https://docs.aws.amazon.com/awssupport/latest/user/security-checks.html) (check `HCP4007jGY`) · [Security Hub EC2 controls](https://docs.aws.amazon.com/securityhub/latest/userguide/ec2-controls.html) (EC2.13, EC2.14, EC2.19, EC2.21) · [CIS mapping](https://docs.aws.amazon.com/securityhub/latest/userguide/cis-aws-foundations-benchmark.html)
- Azure: [NSG default security rules](https://learn.microsoft.com/en-us/azure/virtual-network/network-security-groups-overview) · [Defender for Cloud networking recommendations](https://learn.microsoft.com/en-us/azure/defender-for-cloud/recommendations-reference-networking) · [just-in-time VM access](https://learn.microsoft.com/en-us/azure/defender-for-cloud/enable-just-in-time-access)
- Google Cloud: [VPC firewall rules, including the default network's pre-populated rules](https://docs.cloud.google.com/firewall/docs/firewalls)
- Microsoft (non-cloud): [Preventing SMB traffic from lateral connections](https://support.microsoft.com/en-us/topic/preventing-smb-traffic-from-lateral-connections-and-entering-or-leaving-the-network-c0541db7-2244-0dce-18fd-14a3ddeb282a) · [Secure SMB traffic in Windows Server](https://learn.microsoft.com/en-us/windows-server/storage/file-server/smb-secure-traffic) · [SQL Server installation security considerations](https://learn.microsoft.com/en-us/sql/sql-server/install/security-considerations-for-a-sql-server-installation)

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
- [Prometheus security model](https://prometheus.io/docs/operating/security/)
- [Jenkins security](https://www.jenkins.io/doc/book/security/) · [network services](https://www.jenkins.io/doc/book/security/services/)
- [Oracle Net Listener security](https://docs.oracle.com/en/database/oracle/oracle-database/26/netag/managing-oracle-net-listener-security.html)
- [Java JMX monitoring and management](https://docs.oracle.com/en/java/javase/21/management/monitoring-and-management-using-jmx-technology.html)
- [Neo4j security checklist](https://neo4j.com/docs/operations-manual/current/security/checklist/)
- [Xsecurity(7)](https://man.openbsd.org/Xsecurity.7)

Consulted and deliberately not used as evidence
- `nmap-services` open-frequency data — frequency, and 2008-vintage. Used in §6.1 only to state where a port ranks, never to justify a verdict
- Shodan / Censys internet-exposure studies — frequency. Named as candidates by the ticket; not used anywhere in this note
