# Safe active probing against your own production

Research ticket #4 — wayfinder research for the verge-asm v1 spec.

**Question.** What does a safe default active-probing profile look like when the target is the
operator's own production infrastructure, and the probe runs continuously and unattended?

**Framing.** Authorisation is not the constraint here — the operator owns the targets. The two
risks that actually bite are:

1. **Degrading or destabilising live production.** Nmap's own documentation is candid about this:
   "Reports of systems being crashed by Nmap are rare, but they do happen," and "Many NAT/firewall
   devices keep a state entry for every port probe … occasional (pathetic) implementations crash
   instead" ([nmap.org/book/legal-issues.html](https://nmap.org/book/legal-issues.html)). Masscan is
   blunter: it warns it can "melt most networks" and that "networks often crash under the load that
   masscan can generate"
   ([masscan README](https://github.com/robertdavidgraham/masscan/blob/master/README.md)).
2. **Being a permanent, self-inflicted source of noise.** A one-shot scan that trips the WAF once is
   an annoyance. A scheduled scan that trips it every hour trains the operator to ignore their own
   alerting. Everything below optimises for *repeatability at low cost* over *single-pass
   completeness*.

A third constraint runs through the whole design: verge-asm ships as `docker compose up`. Whatever
the scanner needs from the kernel becomes something the compose file must ask the operator for. That
budget should be **zero added capabilities**, and the technique choices below are made to fit inside
it.

---

## 1. Recommended default profile (summary)

| Knob | Default | Rationale (detail below) |
|---|---|---|
| Scan technique | **TCP connect (`connect()`)**, non-root, no added capabilities | §3 — SYN requires raw sockets *and* root; nmap gates on euid |
| Host discovery | **Skipped** (`-Pn` equivalent) | §3.4 — targets are operator-seeded, not discovered by sweep |
| Continuous port set | **`verge-core`: ~140 curated TCP ports** (nmap top-100 minus ephemeral noise, plus a modern-services supplement) | §2 |
| Weekly port set | **Retired** — there is no warm tier | §2.4, and [`nmap-services-licence.md`](./nmap-services-licence.md) §12 ruling 3 |
| Full-range 1–65535 | **Off by default**; opt-in **per `Seed` scope**, rate-capped, monthly at most. It ships as a configured-and-**disabled** `Scan` with an empty scope list, and it never runs unasked — including at onboarding | §2.4, and [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md) |
| UDP | **Off by default** | §2.5 |
| Port-scan rate | **≤ 50 packets/sec per target host**, ≤ 20 concurrent connections per host | §6 |
| Connect timeout | 3 s, 2 retries | §6 |
| HTTP concurrency | **≤ 10 req/s per host**, 5 concurrent | §6 |
| HTTP request | `GET /` , capped body read (64 KB), 10 s timeout | §4 |
| Redirects | **Not followed** by default; same-host-only when enabled, max 5 | §4.3 |
| Admin-panel paths | Small curated path list, **response-matching only** | §4.4 |
| Default-credential login attempts | **Never** (not even opt-in for v1) | §4.4, §9 |
| TLS | Cert fetch every run; version/cipher enumeration **weekly** | §5 |
| Cert expiry alert thresholds | 30 / 14 / 7 days | §5.1 |
| Scheduling | Default **daily**, ±20 % jitter, operator-set quiet hours | §6.4 |
| Vantage point | Recorded on **every** observation; exposure findings require an external vantage | §7 |
| Concurrency across targets | Round-robin by host, never by port | §6.3 |

---

## 2. Port selection

### 2.1 Where the "top ports" lists actually come from

Nmap's default port set is not IANA's registry and not a guess; it is empirical frequency data
shipped in the `nmap-services` file. The file's own header states the fields:

> `# Fields in this file are: Service name, portnum/protocol, open-frequency, optional comments`
> — [`nmap-services`](https://github.com/nmap/nmap/blob/master/nmap-services)

and describes itself as "Derived from IANA data and our own research". The third column is
"a measure of how often the port was found open during research scans of the Internet"
([nmap.org/book/nmap-services.html](https://nmap.org/book/nmap-services.html)).

The provenance of that frequency column is stated in the Nmap book:

> "Nmap's port registration file (`nmap-services`) contains empirical data about how frequently each
> TCP or UDP port is found to be open. This data was collected by scanning tens of millions of
> Internet addresses, then combining those results with internal scan data contributed by large
> enterprises."
> — [nmap.org/book/port-scanning-options.html](https://nmap.org/book/port-scanning-options.html)

and elaborated as "data from scanning tens of millions of Internet IP addresses as well as
enterprise networks scanned from within"
([nmap.org/book/performance-port-selection.html](https://nmap.org/book/performance-port-selection.html)).

The defaults derived from it:

- Default scan: "Nmap scans the 1,000 most popular ports of each protocol it is asked to scan."
- `-F`: "scan only the 100 most common ports in each protocol."
- `--top-ports N`: "specify an arbitrary number of ports to scan."
  — all from [port-scanning-options.html](https://nmap.org/book/port-scanning-options.html)

Nmap's published coverage claims: top 10 ports ≈ half of open ports per protocol; top 100 covers
**78 % of TCP** and 39 % of UDP; top 1,000 catches **~93 % of TCP** and 49 % of UDP
([performance-port-selection.html](https://nmap.org/book/performance-port-selection.html)).

**I verified this against the shipped data.** Summing the `open-frequency` column for all TCP
entries in the current `nmap-services` (27,483 lines, fetched from
[raw.githubusercontent.com/nmap/nmap/master/nmap-services](https://raw.githubusercontent.com/nmap/nmap/master/nmap-services))
and taking each prefix's share of the total mass reproduces the book's figures:

```
top10   = 50.3 %
top100  = 78.2 %
top1000 = 92.9 %
top2000 = 95.9 %
```

The top-100 TCP set derived from that file is exactly nmap's `-F` list:

```
7,9,13,21,22,23,25,26,37,53,79,80,81,88,106,110,111,113,119,135,139,143,144,179,199,
389,427,443,444,445,465,513,514,515,543,544,548,554,587,631,646,873,990,993,995,
1025,1026,1027,1028,1029,1110,1433,1720,1723,1755,1900,2000,2001,2049,2121,2717,
3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,
5800,5900,6000,6001,6646,7070,8000,8008,8009,8080,8081,8443,8888,9100,9999,10000,
32768,49152,49153,49154,49155,49156,49157
```

Top 20 by frequency, for scale: 80 (0.484), 23 (0.221), 443 (0.209), 21 (0.198), 22 (0.182),
25 (0.131), 3389 (0.084), 110 (0.077), 445 (0.057), 139 (0.051), 143 (0.050), 53 (0.048),
135 (0.048), 3306 (0.045), 8080 (0.042), 1723 (0.032), 111 (0.030), 995 (0.030), 993 (0.027),
5900 (0.024).

### 2.2 The vintage problem — the load-bearing finding

The frequency data is old. The `nmap-services` header carries
`$Id: nmap-services 9746 2008-08-26 18:45:24Z fyodor $`
([source](https://raw.githubusercontent.com/nmap/nmap/master/nmap-services)), and the book describes
the underlying scans in the past tense as a discrete research effort. The consequence, measured
directly against the current file, is that **the modern services a small org is most likely to
accidentally expose rank far outside the top 1,000, or are absent entirely**:

| Port | Service | Rank by open-frequency | In top-1000? |
|---|---|---|---|
| 8443 | https-alt | 37 | yes |
| 5432 | PostgreSQL | 84 | yes |
| 9090 | (listed as `zeus-admin`; today Prometheus/Cockpit) | 107 | yes |
| 9200 | (listed as `wap-wsp`; today Elasticsearch) | 732 | yes, barely |
| 623 | IPMI/BMC | 1,267 | **no** |
| 6379 | Redis | 1,683 | **no** |
| 5672 | AMQP/RabbitMQ | 1,906 | **no** |
| 27017 | MongoDB | 2,841 | **no** |
| 2375 / 2376 | Docker daemon API | 2,903 / 2,902 | **no** |
| 8161 | ActiveMQ console | 4,481 | **no** |
| 5984 | CouchDB | 5,010 | **no** |
| 5601 | (listed as `esmagent`; today Kibana) | 5,112 | **no** |
| 2181 | ZooKeeper | 7,574 | **no** |
| 11211 | memcached | 8,317 | **no** |
| 2379 / 2380 | etcd | frequency 0.000000 | **no** |
| 6443 | Kubernetes API | frequency 0.000000 | **no** |
| **10250** | **kubelet** | **not present in the file at all** | **no** |
| 9042 | Cassandra | not present in the file at all | **no** |
| 50070 | HDFS NameNode | not present in the file at all | **no** |

(Ranks computed from the current `nmap-services` TCP entries sorted by open-frequency descending.)

Conversely, ~15 % of the top-100 is dead weight for an ASM tool: `1025–1029` and `49152–49157` are
Windows RPC ephemeral ranges, and `2717`, `5101`, `5190`, `6646`, `3986` are artefacts of 2008-era
consumer software. They cost probe budget and produce findings the operator cannot act on.

**Conclusion: neither top-100 nor top-1000 is the right continuous set.** Top-100 is too small
*and* misdirected; top-1000 is 10× the cost for coverage of a distribution that no longer matches
the estate being monitored.

### 2.3 Recommended continuous set — `verge-core`

Roughly 140 TCP ports. **This is the project's own selection, informed by nmap's published ranking
rather than derived from it** — the wording matters and is
[`nmap-services-licence.md`](./nmap-services-licence.md) §12 ruling 8's correction to this section.
Measured there (§6.2): the set retains **81** of nmap's top-100 after **19** project-chosen
deletions and adds **44** ports net-new that nmap's ranking does not support at any size — 63
overrides against the source, on a signal-mapping rule of the project's own. The two limbs below
are how that selection was made, not a transformation applied to somebody else's list:

- **Nmap's top-100 as one input, minus the ephemeral/obsolete tail** (drop 1025–1029, 49152–49157,
  2717, 5101, 5190, 6646, 3986, 5051, 5009, 1755) — keeps the high-frequency mass that genuinely
  correlates with real listeners.
- **Plus a modern-services supplement**, chosen because each maps to a *named v1 risk signal*
  (exposed admin panel, unexpected open port, plaintext service):
  - HTTP-ish alternates: 3001, 4000, 5601, 7001, 8006, 8069, 8086, 8090, 8161, 8500, 8834, 9000,
    9090, 9200, 9300, 9443, 10000
  - Data stores: 1521, 5984, 6379, 9042, 11211, 27017, 27018
  - Messaging/coordination: 1883, 2181, 5672, 15672, 61616
  - Orchestration/control planes: 2375, 2376, 2379, 2380, 6443, 10250, 10255
  - Management/OOB: 161 (TCP), 623, 5985, 5986, 9100
  - Remote access: 3389, 5900–5905, 5800

Rationale: an ASM tool is not trying to characterise the internet, it is trying to notice
*this estate's* drift. The correct prior is "what does a small org accidentally leave listening,"
which is a different distribution from "what is open across tens of millions of 2008 internet
hosts". `verge-core` should be shipped as an editable list file, not compiled in (§8).

### 2.4 Continuous vs periodic — the schedule is the real answer

The question "top-100 or top-1000 or full range" is the wrong shape for a continuously running tool.
Continuous operation changes the economics in three ways:

1. **Amortisation.** A full 65,535-port sweep is unaffordable *per run* but entirely affordable
   *once per month*. The information it yields (a service on a weird high port) changes slowly.
2. **Diffing beats coverage.** The value of run N is `run N − run N−1`. A narrow set scanned daily
   detects a newly opened port within 24 h. A wide set scanned monthly detects it within 30 days.
   For the named v1 signals (drift, unexpected open ports), *latency* dominates *breadth*.
3. **Cumulative load.** Cost is `ports × hosts × runs/period`. Top-1000 hourly is 240× top-1000
   daily. Any port-count argument must be made jointly with a frequency argument.

Recommended tiering:

| Tier | Port set | Default cadence | Purpose |
|---|---|---|---|
| Hot | `verge-core` — the union, whose membership [ADR-0009](../adr/0009-verge-core-is-a-union.md) owns | daily | drift detection, low latency |
| Cold | full 1–65535 | **opt-in per `Seed` scope**, monthly ceiling | rare discovery |

**The warm tier is retired**, by [`nmap-services-licence.md`](./nmap-services-licence.md) §12
ruling 3. It was the one place the project defined a set *by reference to nmap's own ranking* — a
set that cannot be evaluated without `nmap-services`, and so the only place we would have reproduced
nmap's selection whole rather than made our own. It is affordable to retire because §7.4 of that
note measured what it contributed: its ~900 ports beyond the hot set are 2008's long tail, and every
modern service the product exists to notice (Redis, Docker, MongoDB, etcd, the Kubernetes API,
kubelet, Cassandra, CouchDB) ranks below 1,441 or is absent from the file entirely. Those ports fall
to the cold tier, at 30-day rather than 7-day latency.

**There is no unconditional onboarding sweep.** This paragraph used to say the full-range sweep
*"should also run once at target onboarding"*, on the argument that onboarding is the one moment the
operator is present and watching. [#80](https://github.com/winniel123/verge-asm/issues/80) settled it
against that reading and [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md) carries the
reasoning. Four things decided it:

- **§6.4 forbids the mechanism that would make the operator present.** *"Never scan on config save.
  Adding a target should queue a scan, not fire one."* A queued sweep runs at the next tick, under
  jitter, inside quiet hours — overnight and unattended. The premise was false under this note's own
  scheduling rules.
- **A one-off has no cadence, so it has no currency.** [ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md)
  sets currency at `k` cadences of the covering `Scan` and publishes the full-range figure as two
  months; a sweep that runs once leaves `k × cadence` undefined for every timeline it opens.
- **No configured object accounts for its scope**, which [ADR-0005](../adr/0005-scan-execution-model.md)
  refuses outright for ad-hoc runs.
- **It would make [#44](https://github.com/winniel123/verge-asm/issues/44)'s standing aperture
  statement non-constant** — *1–65535* for one night and `verge-core` ever after — falsifying the
  premise that discharged its three-densities obligation.

The onboarding baseline is real and it already exists as an operator act:
[#51](https://github.com/winniel123/verge-asm/issues/51)'s first-run step 4, *Run the first batch*,
which dispatches whichever `Scan`s the operator has enabled. If they enabled the cold tier, the
baseline is full-range; if they did not, it is `verge-core`. Either way it is a button, not a default.

**So a default-settings install measures `verge-core` and nothing else, permanently** — including the
~900 tail ports the retired warm tier used to cover. That is the honest statement of v1's aperture,
and it is stated on `Coverage` rather than left to be discovered: the port-tier line names the tier,
its cadence and its off state, and carries `0 of 37 sensitive pairs unread` and `0 of 16 rules
unevaluable`. Both are true by construction — [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union
puts every sensitive pair inside the hot set, and of the sixteen rules one names a port (fully
covered), four read `Name`s, and eleven read a facet on a subject. **The tier bounds which subjects
exist, never which rules can speak**, so what the cold tier buys is drift breadth rather than signal
correctness. A count of unmeasured ports is deliberately absent: it is knowable, which is what makes
it tempting, and it is [#28](https://github.com/winniel123/verge-asm/issues/28)'s refused
estate-completeness score in port clothing.

**No middle tier replaces the retired warm one**, and the refusal does not rest on
[`nmap-services-licence.md`](./nmap-services-licence.md) §3. Under ADR-0009's union, any set authored
on the project's own signal-mapping rule is already inside `verge-core` or is the cold tier's
population at the cold tier's cadence. There is no middle to occupy.

### 2.5 UDP

Off by default. Nmap's own data puts top-100 UDP coverage at 39 % and top-1000 at 49 %
([performance-port-selection.html](https://nmap.org/book/performance-port-selection.html)) — i.e.
even a 1,000-port UDP scan misses half of what is open, while costing far more time because
open|filtered states must be resolved by timeout. The signal-to-cost ratio does not justify making
it a default for a tool that runs unattended. Offer it as an opt-in for a hand-picked list
(53, 123, 161, 500, 623, 1900, 5353) where a finding is genuinely actionable.

---

## 3. Scan technique, and what it costs the compose file

### 3.1 SYN vs connect

- **SYN scan (`-sS`)** never completes the handshake — a "half-open reset". It "requires root access
  on Unix systems to send raw packets" and is nmap's default *when privileges are available*
  ([nmap.org/book/man-port-scanning-techniques.html](https://nmap.org/book/man-port-scanning-techniques.html)).
- **Connect scan (`-sT`)** "asks the underlying operating system to establish a connection … by
  issuing the `connect` system call". It is "usable by unprivileged users" and becomes the default
  "when a user does not have raw packet privileges" (same page).

The documented downsides of connect scan are real but modest at our rates:

> "completes connections to open target ports rather than performing the half-open reset" — requiring
> "more packets to obtain the same information", and "target machines are more likely to log the
> connection. … many services on your average Unix system will add a note to syslog, and sometimes a
> cryptic error message, when Nmap connects and then closes the connection without sending data."
> — [man-port-scanning-techniques.html](https://nmap.org/book/man-port-scanning-techniques.html)

For verge-asm this trade-off is **inverted in favour of connect**:

- The "more packets" penalty is ~1.5× at a rate deliberately capped two orders of magnitude below
  what either technique can sustain (§6). It is not the binding constraint.
- The "target logs the connection" penalty is *desirable*. Scanning your own production and having
  it appear in your own logs is auditability, not stealth failure. Log noise is managed by scanning
  from a stable, known source address the operator can annotate — not by hiding.
- Half-open SYN scanning against your own stateful firewall is the exact pattern that fills
  connection-tracking tables with half-open entries. Nmap notes NAT/firewall devices "keep a state
  entry for every port probe" ([legal-issues.html](https://nmap.org/book/legal-issues.html));
  completed-and-closed connections clear state promptly, dangling SYNs age out on a timer.

### 3.2 What SYN would demand of the Docker container

This is the decisive constraint. Raw-socket sending needs `CAP_NET_RAW`, defined as:

> "Use RAW and PACKET sockets; bind to any address for transparent proxying."
> — [capabilities(7)](https://man7.org/linux/man-pages/man7/capabilities.7.html)

`CAP_NET_ADMIN` (interface configuration, routing tables, promiscuous mode, firewall administration
— same page) is **not** needed for SYN scanning and should never appear in the compose file.

Docker's default capability set *already includes* `NET_RAW` (alongside AUDIT_WRITE, CHOWN,
DAC_OVERRIDE, FOWNER, FSETID, KILL, MKNOD, NET_BIND_SERVICE, SETFCAP, SETGID, SETPCAP, SETUID,
SYS_CHROOT), while `NET_ADMIN` is not and must be added explicitly
([docs.docker.com/engine/containers/run](https://docs.docker.com/engine/containers/run/)).

**But having the capability is not sufficient for nmap.** Nmap gates on effective UID, not on the
capability bit. The `--privileged` flag exists precisely because of this:

> "Tells Nmap to simply assume that it is privileged enough to perform raw socket sends, packet
> sniffing, and similar operations that usually require root privileges on Unix systems." … by
> default Nmap exits if such operations are requested but `geteuid` is not zero … this "works well
> with Linux kernel capabilities and similar systems that may be configured to allow unprivileged
> users to perform raw-packet scans" (`NMAP_PRIVILEGED` env var is equivalent).
> — [nmap.org/book/man-misc-options.html](https://nmap.org/book/man-misc-options.html)

So the SYN path costs the compose file **either `user: root` or an `NMAP_PRIVILEGED` override plus a
capability the operator must reason about** — and root-in-container is the thing every hardening
guide tells a small-org operator not to do. Kubernetes and hardened runtimes commonly drop `NET_RAW`
by default, so a SYN-dependent design also breaks on the deployment targets a self-hoster is most
likely to graduate to.

Masscan raises the cost further: it "uses its own ad hoc network stack", which conflicts with the
host stack and requires either a dedicated `--src-ip`, a reserved `--src-port`, or firewall rules to
keep the OS from RSTing its own connections
([masscan README](https://github.com/robertdavidgraham/masscan/blob/master/README.md)). That is
several paragraphs of deployment documentation for a tool whose entire promise is `docker compose up`.

### 3.3 Recommendation

```yaml
services:
  scanner:
    user: "10001:10001"     # non-root
    cap_drop: [ALL]         # including the default NET_RAW
    # no cap_add
    # no privileged: true
    # no network_mode: host
```

Compose syntax per
[docs.docker.com/reference/compose-file/services](https://docs.docker.com/reference/compose-file/services/),
which documents `cap_add`, `cap_drop`, `privileged: true`, and that `network_mode: host` "gives the
container raw access to the host's network interface" (and that "Port mapping must not be used with
`network_mode: host`").

`network_mode: host` deserves an explicit "no": it is the difference between the scanner having its
own network identity (attributable in the operator's logs, firewallable, throttleable) and it being
indistinguishable from the host. That identity is worth more to an ASM tool than the raw access.

This is the same posture ProjectDiscovery's naabu degrades to. Naabu's current default scan type is
`-s c` (CONNECT) — `-scan-type, -s string  type of port scan (SYN/CONNECT) (default "c")` — and it
logs `Running CONNECT scan with non root privileges` vs `Running SYN scan with root privileges`
depending on what it has
([naabu README](https://github.com/projectdiscovery/naabu/blob/main/README.md)). It also fails safe
in other constrained cases: "Syn Scan can't be used with socks proxy: falling back to connect scan"
and "Routing could not be determined (are you using a VPN?). falling back to connect scan"
([pkg/runner/validate.go](https://github.com/projectdiscovery/naabu/blob/main/pkg/runner/validate.go)).

**verge-asm should treat SYN as an unsupported optimisation, not a degraded mode.** If a future
version wants it, it belongs behind an explicitly documented `privileged-scanner` compose profile
that the operator opts into, never in the default file.

### 3.4 What we give up, and why it does not matter here

Running unprivileged loses:

- **ICMP echo host discovery.** Nmap's default discovery sends "an ICMP echo request, a TCP SYN
  packet to port 443, a TCP ACK packet to port 80, and an ICMP timestamp request"; the TCP and ICMP
  raw probes "require root access on Unix systems to send raw packets"
  ([man-host-discovery.html](https://nmap.org/book/man-host-discovery.html)). Unprivileged ICMP is
  possible via `SOCK_DGRAM` ping sockets, but the kernel default is off: `ping_group_range` defaults
  to `"1 0"`, meaning nobody, including root, may create ping sockets
  ([docs.kernel.org/networking/ip-sysctl.html](https://docs.kernel.org/networking/ip-sysctl.html)).
  Enabling it needs a `sysctls:` entry in compose — another ask.
- **ARP discovery on the local segment.** Nmap uses ARP/Neighbor Discovery by default for local
  targets because it is faster and more reliable (same page). Not available unprivileged.
- **OS fingerprinting**, which needs raw packets.

None of these are load-bearing for verge-asm, because **targets are seeded by the operator, not
discovered by sweeping**. The operator types in domains and IP ranges they own; the tool does not
need to establish liveness before probing, it needs to record "no ports responded" as a legitimate,
diffable observation. So the default should be the `-Pn` posture — "skips the host discovery stage
altogether … attempt the requested scanning functions against *every* target IP address specified"
([man-host-discovery.html](https://nmap.org/book/man-host-discovery.html)) — with a per-target
sanity cap on range size so a fat-fingered `/8` cannot be entered.

Losing OS fingerprinting is not a loss at all: it is one of the more intrusive things a scanner
does, and an ASM tool for owned infrastructure gets better OS data from the operator's own inventory.

---

## 4. HTTP probing

### 4.1 What to request

**`GET /`, not `HEAD`.** RFC 9110 §9.3.2 defines HEAD as identical to GET except "the server MUST
NOT send message content in the response", and requires the response to include "the same header
fields that would have been sent in the response to a GET request"
([RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html)). In principle HEAD is the polite choice.
In practice the *content* is where the signal lives — page title, framework strings, error text,
default-install pages, and the takeover fingerprints in §6 are all body matches. A HEAD-only probe
cannot detect an exposed admin panel or a dangling-DNS 404 body. Reverse proxies and app frameworks
also routinely mishandle HEAD (returning 405, or a different status than GET), which manufactures
false drift.

Mitigate the cost instead of avoiding the request: **cap the body read**. 64 KB is enough for
`<title>`, meta generator tags, and every takeover fingerprint string, and bounds memory per probe.
httpx does the same thing with `-response-size-to-read` and previews only the first 100 characters
of body by default
([httpx usage](https://docs.projectdiscovery.io/tools/httpx/usage)).

GET is also the safe choice by HTTP's own definition. §9.2.1 defines a method as safe when "the
client does not request, and does not expect, any state change on the origin server as a result of
applying a safe method to a target resource", and the method registry records GET as Safe = yes
([RFC 9110 §9.2.1](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2.1),
[§16.1.1 Table 4](https://www.rfc-editor.org/rfc/rfc9110.html#section-16.1.1)). No
default probe should ever use POST/PUT/DELETE (§9).

### 4.2 Fingerprinting a service

Collect, per probe, and store as a versioned observation:

| Signal | Source / mechanism |
|---|---|
| Status code | httpx `-sc` |
| Page title | httpx `-title` |
| `Server` header | RFC 9110 §10.2.4: "information about the origin server software" ([RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html)) |
| `X-Powered-By`, other framework headers | same class of voluntary disclosure |
| Content length + body hash | drift primitive — hash a *normalised* body (strip CSRF tokens, timestamps, nonces) or you get a diff every run |
| Favicon hash (MMH3) | httpx `-favicon` — "display MMH3 hash for `/favicon.ico`"; stable across cosmetic changes, strong product identifier |
| Technology detect | httpx `-td` "identifies technologies using Wappalyzer dataset"; `-cff` for custom fingerprints |
| JARM | httpx `-jarm` — TLS-stack fingerprint, useful when HTTP tells you nothing |

Flags per [httpx README](https://github.com/projectdiscovery/httpx/blob/main/README.md) and
[httpx usage](https://docs.projectdiscovery.io/tools/httpx/usage).

Scheme handling: httpx's default is HTTPS-first with "smart auto fallback from https to http"
([httpx README](https://github.com/projectdiscovery/httpx/blob/main/README.md)). verge-asm should do
the same but **record the fallback as a finding, not just a mechanism** — an HTTP-only endpoint on
80 with no redirect to 443 is precisely the "plaintext HTTP" v1 risk signal.

Related: RFC 6797 requires that an HSTS host "MUST NOT include the STS header field in HTTP responses
conveyed over non-secure transport" ([RFC 6797](https://www.rfc-editor.org/rfc/rfc6797.html)). So
"port 443 serves a valid cert but 80 neither redirects nor is closed, and no HSTS policy is in
effect" is a well-defined, citable finding rather than a matter of taste.

### 4.3 Redirect handling

**Do not follow redirects by default.** httpx makes the same call: `-follow-redirects` defaults to
`false` ([runner/options.go](https://github.com/projectdiscovery/httpx/blob/main/runner/options.go)).

Reasons specific to an ASM tool:

- A 301/302 to a *different host* means you stop measuring the asset you are tracking and start
  measuring something else. The redirect target may not even be operator-owned (a CDN, a SaaS login,
  a marketing site). Silently following it corrupts the inventory.
- The redirect itself is the finding. RFC 9110 §15.4 distinguishes 301 "the target resource has been
  assigned a new permanent URI", 302 (temporary), 307 (temporary, method-preserving) and 308
  (permanent, method-preserving) ([RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html)). A
  host that starts 302-ing to an unexpected destination *is* exposure drift.
- Following multiplies request volume by the chain length against a live service.

When the operator enables following, use httpx's safer variant: `-follow-host-redirects` ("follow
redirects on the same host only"), cap at 5 (httpx's default is 10 —
`-max-redirects int ... (default 10)`), and honour `-respect-hsts`
([httpx usage](https://docs.projectdiscovery.io/tools/httpx/usage)). Always record the full redirect
chain, so an off-host hop is visible even when it was not followed.

### 4.4 Identifying an admin panel or default install

The reliable, low-risk method is **path probe + response matcher**, exactly the shape of nuclei's
`exposed-panels` templates. Concrete example
([grafana-detect.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/exposed-panels/grafana-detect.yaml)):

```yaml
id: grafana-detect
info:
  name: Grafana Login Panel - Detect
  severity: info
  classification: { cwe-id: CWE-200 }
  metadata: { max-request: 2 }
  tags: panel,grafana,detect,discovery
http:
  - method: GET
    path:
      - "{{BaseURL}}/login"
      - "{{BaseURL}}/graph/login"
    stop-at-first-match: true
    matchers:
      - type: word
        part: body
        words: ["<title>Grafana</title>"]
    extractors:
      - type: regex
        part: body
        regex: ['\"subTitle\":\"Grafana v([0-9.]+)']
```

Note the properties worth copying:

- **`method: GET` only**, unauthenticated, no state change.
- **`stop-at-first-match`** and **`max-request: 2`** — a hard, declared budget per fingerprint. A
  continuously-running tool needs this to be a first-class concept, not an emergent property.
- **Severity `info`** for detection; the risk judgement is layered on separately.
- **Version extraction from the same response** — no extra request.

Strong panel signals, in descending order of reliability: exact `<title>` match; favicon MMH3 hash;
a product-specific header or cookie name (e.g. `grafana_session`); a unique static asset path; a
generic-but-corroborated body string. Weak signals that produce false positives on their own:
status-code-only matches, and the word "login" or "admin" appearing anywhere.

**Where to draw the line: detection yes, authentication never.** The contrast is stark inside
nuclei's own template set. `grafana-default-login.yaml` does a `POST /login` with
`{"user":"admin","password":"admin"}` and `{"user":"admin","password":"prom-operator"}`, matching on
a `grafana_session` cookie plus a `"Logged in"` body and HTTP 200 — severity `high`, tags
`grafana,default-login,vuln`
([source](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/default-logins/grafana/grafana-default-login.yaml)).

That is a genuine authentication attempt against production. Unattended, on a schedule, it will:
create real sessions, populate audit logs with successful admin logins, trip brute-force lockouts
against a real admin account, and — if the credentials work — leave the tool holding a live
privileged session it has no plan for. **verge-asm must not perform credential submission of any
kind in v1, default or opt-in.** Report "a Grafana login panel is reachable from the internet"; that
is the actionable exposure. Whether the password is `admin` is the operator's to check, once,
by hand.

The panel path list should stay small (10–20 well-chosen paths, not thousands) and be a shipped,
editable data file (§8). A path list is the thing most likely to trip a WAF, because directory-ish
request bursts are the canonical scanner signature.

---

## 5. TLS inspection

A single TLS handshake is the highest-yield probe in the whole tool: one connection, no
authentication, no state change, and it answers several v1 risk signals at once.

### 5.1 Expiry

`Validity` carries `notBefore` and `notAfter`; RFC 5280 §4.1.2.5 requires dates through 2049 to be
encoded as `UTCTime` and 2050+ as `GeneralizedTime`
([RFC 5280](https://www.rfc-editor.org/rfc/rfc5280.html)) — worth honouring in the parser rather
than assuming one encoding.

Alert thresholds should account for the fact that certificate lifetimes are collapsing. The CA/Browser
Forum adopted ballot SC-081v3 in April 2025, introducing "eventual reduction of maximum validity
period from 398 days to 47 days … proposed to occur starting in March 2026 and concluding in
March 2029"
([cabforum.org ballot SC081v3](https://cabforum.org/2025/04/11/ballot-sc081v3-introduce-schedule-of-reducing-validity-and-data-reuse-periods/)).
The Baseline Requirements compliance-date table gives the steps: **2026-03-15 → 200 days,
2027-03-15 → 100 days, 2029-03-15 → 47 days**
([cabforum/servercert `docs/BR.md`](https://github.com/cabforum/servercert/blob/main/docs/BR.md)).

Practical implication: **30 / 14 / 7 day thresholds are right today and will still be right at
47-day lifetimes** (a 30-day warning on a 47-day cert fires at 36 % of life remaining — noisy but
not absurd), but the thresholds must be operator-configurable (§8), because an operator on
90-day ACME certs with working automation wants 7/3/1, and an operator with a manual annual EV cert
wants 60/30/14. Also alert on **expiry-window shrinkage** — a cert whose lifetime drops from 398 to
47 days without the renewal automation changing is a future outage.

Additionally: alert on `notBefore` in the future (clock/issuance error) and on a cert that is
*already* expired but still being served — the latter is an active outage, not a warning.

### 5.2 Issuer

RFC 5280 §4.1.2.4 requires the issuer field to "contain a non-empty distinguished name (DN)"
([RFC 5280](https://www.rfc-editor.org/rfc/rfc5280.html)). Store the issuer DN as a tracked
attribute: an unexplained issuer change (Let's Encrypt → self-signed, or → an unfamiliar CA) is a
first-class drift signal, often the visible symptom of a misrouted host, a resurrected staging box,
or a failed ACME renewal that fell back to a default self-signed cert.

### 5.3 SAN entries as a discovery source

This is the highest-value feedback loop in the tool. RFC 5280 §4.2.1.6 defines `subjectAltName`,
including `dNSName`, and notes the subject field may be an empty sequence when naming information
lives only in the extension ([RFC 5280](https://www.rfc-editor.org/rfc/rfc5280.html)). RFC 6125
makes SAN the authoritative place to look: clients should "move toward including and checking DNS
domain names via the subjectAlternativeName extension designed for that purpose: dNSName", and
"A client MUST NOT seek a match for a reference identifier of CN-ID if the presented identifiers
include a DNS-ID …" ([RFC 6125](https://www.rfc-editor.org/rfc/rfc6125.html)).

So: **parse SAN `dNSName` entries and feed them back into the discovery seed set.** A single
handshake against one known host routinely yields a dozen sibling hostnames the operator forgot
about — including internal-looking names on a public cert, which is itself a finding.

Certificate Transparency is the passive twin of this. RFC 6962's design is to "publicly log the
existence of TLS certificates as they are issued or observed", with the expectation that "public CAs
will contribute all their newly issued certificates to one or more logs", and domain owners can
"monitor the logs, asking them regularly for all new entries, and can thus check whether domains
they are responsible for have had certificates issued that they did not expect"
([RFC 6962](https://www.rfc-editor.org/rfc/rfc6962.html)). Active SAN harvesting finds names on
certs *currently served*; CT monitoring finds names on certs *ever issued*, including for hosts that
are down or were never publicly reachable. v1 should do both and reconcile them — a name in CT with
no live host is a strong dangling-DNS lead (§6).

Parse CN too, but only as a fallback and marked as low-confidence, per RFC 6125's deprecation.

### 5.4 Protocol and cipher support

RFC 8996 is unambiguous: "TLS 1.0 MUST NOT be used. Negotiation of TLS 1.0 from any version of TLS
MUST NOT be permitted", and the same for TLS 1.1
([RFC 8996](https://www.rfc-editor.org/rfc/rfc8996.html)). Reasons cited include reliance on SHA-1
for handshake integrity and peer authentication, absence of AEAD ciphers, and mandatory support for
weaker algorithms such as 3DES. So "this host negotiates TLS 1.0" is a *standards-backed* finding,
not an opinion — which matters for a tool whose credibility rests on not crying wolf.

**But version and cipher enumeration is expensive and must not run every cycle.** Determining
supported versions and ciphers means one handshake per candidate — tlsx exposes this as `-version-enum`
and `-cipher-enum`, separate from the default single-handshake collection which yields IPs, ports,
connection status, validity dates, subject DN, issuer, and cert hashes
([tlsx README](https://github.com/projectdiscovery/tlsx/blob/main/README.md)). Recommendation:

- **Every run:** one handshake — cert (expiry, issuer, SAN, CN, serial, chain), negotiated version
  and cipher, plus the `-expired` / `-self-signed` / `-mismatched` / `-untrusted` class of checks
  (tlsx flags, same source).
- **Weekly:** full `-version-enum` / `-cipher-enum`. Cipher support changes when someone changes
  config; it does not change hourly.

Note tlsx's default concurrency is 300 with a 5 s timeout and 3 retries
([tlsx README](https://github.com/projectdiscovery/tlsx/blob/main/README.md)). That is tuned for
scanning many *different* hosts; against a handful of operator-owned hosts it is far too aggressive
per-host and must be re-derived (§6).

---

## 6. Rate limiting and concurrency

### 6.1 What established tools default to, and why they differ

| Tool | Default rate | Default concurrency | Source |
|---|---|---|---|
| nmap `-T3` (default) | adaptive; 0 ms initial scan delay, dynamic parallelism | dynamic, "from one if the network proves unreliable" to "several hundred in perfect conditions" | [timing-templates](https://nmap.org/book/performance-timing-templates.html), [man-performance](https://nmap.org/book/man-performance.html) |
| nmap `-T2` (polite) | 400 ms initial scan delay, 1 s max TCP scan delay | serialised | [timing-templates](https://nmap.org/book/performance-timing-templates.html) |
| nmap `-T4` | max TCP scan delay 10 ms, max 6 retries | dynamic | same |
| masscan | **100 pkt/s** | own TCP/IP stack | [README](https://github.com/robertdavidgraham/masscan/blob/master/README.md) |
| naabu (SYN) | 1000 pkt/s, 1 s port timeout, 3 retries | 25 worker threads | [pkg/runner/default.go](https://github.com/projectdiscovery/naabu/blob/main/pkg/runner/default.go) |
| naabu (CONNECT) | **1500** pkt/s, **3 s** port timeout, 3 retries | 25 threads | same |
| httpx | 150 req/s | 50 threads, 10 s timeout, 0 retries | [runner/options.go](https://github.com/projectdiscovery/httpx/blob/main/runner/options.go), [usage](https://docs.projectdiscovery.io/tools/httpx/usage) |
| tlsx | — | 300 concurrency, 5 s timeout, 3 retries | [README](https://github.com/projectdiscovery/tlsx/blob/main/README.md) |

Three things to read out of that table:

1. **The ProjectDiscovery defaults are not for our workload.** Naabu's README states the assumption
   directly: "As default naabu is configured with a assumption that you are running it from VPS. We
   suggest tuning the flags / rate if running naabu from local system"
   ([naabu README](https://github.com/projectdiscovery/naabu/blob/main/README.md)). These tools are
   built to spray *many hosts shallowly*, where 1000 pkt/s spread over 10,000 hosts is 0.1 pkt/s
   each. verge-asm does the opposite: *few hosts, repeatedly, forever.* The same aggregate rate
   concentrated on three production hosts is a different event entirely. **Rate must be defined
   per-target-host, not globally.**
2. **Masscan's 100 pkt/s default is the honest one.** The fastest scanner in the list ships the
   slowest default, precisely because its authors know what happens otherwise ("melt most
   networks", "networks often crash under the load")
   ([README](https://github.com/robertdavidgraham/masscan/blob/master/README.md)). Capability and
   default should be decoupled.
3. **Nmap's adaptivity is the model to copy, not its numbers.** Nmap adjusts dynamically: when it
   "detects poor network reliability, it may try many more times before giving up on a port", with
   ideal parallelism ranging from one to several hundred based on observed conditions
   ([man-performance](https://nmap.org/book/man-performance.html)). It also exposes
   `--defeat-rst-ratelimit` and `--defeat-icmp-ratelimit` to *ignore* target rate limiting (same
   page) — which tells you the default is to *respect* it. verge-asm should never expose an
   equivalent: if your own infrastructure is rate-limiting the scanner, the correct response is to
   slow down.

### 6.2 On IDS/WAF

Nmap's own framing is instructive: T0 (Paranoid) and T1 (Sneaky) exist because "The first two are
for IDS evasion", and T2 "slows down the scan to use less bandwidth and target machine resources"
([timing-templates](https://nmap.org/book/performance-timing-templates.html)). The inverse is the
point for us: **default-speed scanning is what IDS is built to catch.** A tool that trips the
operator's own IDS every night is worse than useless — it manufactures the alert fatigue that lets a
real detection slide past.

The docs also warn about the cost of politeness: a T2 scan "may take ten times longer than a default
scan" (same page). That is an acceptable trade when the scan runs overnight, unattended, on a
schedule, and nobody is waiting for it. It is exactly the trade verge-asm should make. Our budget is
wall-clock hours, and we have plenty.

### 6.3 Recommended defaults

Per target host:

- **Port scanning:** ≤ 50 connection attempts/sec, ≤ 20 concurrent. 3 s connect timeout (matching
  naabu's `DefaultPortTimeoutConnectScan = 3 * time.Second`
  ([default.go](https://github.com/projectdiscovery/naabu/blob/main/pkg/runner/default.go))),
  2 retries. At 50/s, `verge-core` completes in under 3 s per host; even a full 65,535-port sweep
  finishes in ~22 minutes — entirely acceptable for a monthly job.

  > **Corrected by [#80](https://github.com/winniel123/verge-asm/issues/80) — "~22 minutes" is the
  > best case, not the figure.** It assumes the host answers closed ports with an RST, so each
  > attempt resolves in one RTT and the 50/s **rate** cap binds: 65,535 ÷ 50 = **21 min 51 s**, with
  > about one of the twenty concurrent slots in use and no retries, because an RST is an answer.
  >
  > On a host that **drops** — the default cloud security-group posture, and the common case on an
  > internet-facing address — every attempt runs to the 3 s timeout and the **concurrency** cap binds
  > instead: 20 ÷ 3 s = **6.67 attempts/s**, one seventh of the rate cap. One pass is **2 h 44 min**
  > and the 2 retries take it to **8 h 11 min**. The dominant term is timeout × concurrency, which
  > this bullet never multiplied out.
  >
  > Two consequences, and they point in opposite directions. **Peak load is unchanged**: both tiers
  > run under identical caps, so §1's middlebox state-table hazard — bounded by concurrency, 20
  > either way — is no worse for a full-range sweep than for the daily one. **Duration is not**: at
  > the 200 pkt/s global ceiling the estate figure is 5.5 min/address answering and 16.4 min/address
  > dropping-with-retries, so at §9's own `/22` range cap a single pass is **3.9 to 11.6 days**. The
  > cap exists to stop a typo becoming a multi-day scan; it does not stop this one. That arithmetic
  > is why the full-range tier stays opt-in — [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md).
- **HTTP:** ≤ 10 req/s, ≤ 5 concurrent, 10 s timeout (httpx's default —
  `"timeout in seconds (default 10)"`
  ([usage](https://docs.projectdiscovery.io/tools/httpx/usage))), 1 retry.
- **TLS:** ≤ 5 handshakes/s, ≤ 3 concurrent.

Global ceiling across all targets: 200 pkt/s, so that adding targets does not multiply load without
bound — the scanner's own egress link and the operator's edge firewall are shared resources.

**Schedule round-robin by host, not by port.** Iterating ports within a host produces a dense burst
against one destination — the canonical port-scan signature and the canonical way to fill a state
table. Cycling hosts spreads load in both time and destination. Masscan does the same thing for the
same reason: it "randomizes target IP addresses to spread our traffic evenly over the target"
([README](https://github.com/robertdavidgraham/masscan/blob/master/README.md)). Nmap likewise
"randomizes the port scan order by default"
([port-scanning-options](https://nmap.org/book/port-scanning-options.html)) — keep that, do not use
the `-r` sequential equivalent.

**Adaptive back-off is mandatory, not optional.** On connection timeouts, RSTs at an elevated rate,
HTTP 429/503, or a rising latency trend, halve the rate and record a `throttled` event on the scan
run. Never back off silently — an operator seeing "scan completed at 12 % of configured rate"
learns something real about their infrastructure.

**Port-count sanity check.** Naabu ships `-port-threshold` for this reason. If a host reports an
implausible number of open ports, a firewall is SYN/ACK-ing everything and every "finding" is
fictional. Default: if > 100 ports respond open on `verge-core`, mark the whole host result
`suspect-firewall` and suppress the individual findings rather than emitting 140 false positives
into the operator's inbox.

### 6.4 Scheduling

- **Default cadence: daily**, not hourly. Hourly buys ~23 h of detection latency for 24× the load
  and 24× the log noise. The v1 signals (cert expiry, dangling DNS, new open port) do not move on an
  hourly timescale.
- **Jitter ±20 %.** A scan that starts at exactly 03:00:00 every night is trivially correlated in
  logs — but more importantly, exact periodicity means a transient failure recurs at the same phase
  as whatever else runs at 03:00 (backups, log rotation, cert renewal).
- **Operator-set quiet hours / maintenance windows.** Never probe during the operator's declared
  change window; a scan concurrent with a deploy produces drift findings that are pure noise.
- **Never scan on config save.** Adding a target should queue a scan, not fire one — otherwise
  editing 20 targets fires 20 simultaneous scans.

---

## 7. Dangling DNS / subdomain takeover

### 7.1 The mechanism

Microsoft's official description is the clearest primary statement:

> "A subdomain takeover can occur when you have a DNS record that points to a deprovisioned Azure
> resource. Such DNS records are also known as 'dangling DNS' entries. CNAME records are especially
> vulnerable to this threat."

with the lifecycle: provision a resource with an FQDN → point a CNAME at it → deprovision the
resource without removing the CNAME → "it's advertised as an active domain but doesn't route traffic
to an active Azure resource" → an attacker "provisions an Azure resource with the same FQDN of the
resource you previously controlled"
([learn.microsoft.com — Prevent dangling DNS entries and avoid subdomain takeover](https://learn.microsoft.com/en-us/azure/security/fundamentals/subdomain-takeover)).

Microsoft's stated impact is worth quoting because it justifies the severity rating: loss of content
control, cookie harvesting ("It's common for web apps to expose session cookies to subdomains
(\*.contoso.com). Any subdomain can access them"), phishing, and — explicitly — that TLS is not a
defence: "a threat actor can use the hijacked subdomain to apply for and receive a valid SSL
certificate" (same page).

Their recommended discovery procedure is also directly transferable, and notably it is *not*
primarily an active-probing task: "Query your DNS zones for resources pointing to Azure subdomains
such as \*.azurewebsites.net or \*.cloudapp.azure.com" and "Confirm that you own all resources that
your DNS subdomains are targeting" (same page).

### 7.2 How it is detected in practice — the ladder

Detection is a staged pipeline, and the first three stages are DNS-only (free, zero load on the
target):

**Stage 1 — resolve the full chain.** For every known name, record the CNAME chain and terminal
A/AAAA records. Store the chain, not just the endpoint; the chain is what identifies the provider.

**Stage 2 — classify the DNS answer.** RFC 2308 draws the critical distinction: NXDOMAIN (Name
Error) means "the domain referred to by the QNAME does not exist", whereas NODATA is "a pseudo RCODE
which indicates that the name is valid, for the given class, but are no records of the given type"
([RFC 2308](https://www.rfc-editor.org/rfc/rfc2308.html)). RFC 8499 confirms NXDOMAIN as the synonym
for Name Error ([RFC 8499](https://www.rfc-editor.org/rfc/rfc8499.html)).

**A CNAME that resolves to a target returning NXDOMAIN is the highest-confidence dangling signal
available, and it requires zero packets to the operator's infrastructure.** The
can-i-take-over-xyz project carries this as a per-service flag, distinguishing "NXDOMAIN-based"
detection (the DNS lookup returns NXDOMAIN — e.g. AWS Elastic Beanstalk, Microsoft Azure services)
from "HTTP-response-based" detection which "requires actually accessing the subdomain and parsing
the response body"
([can-i-take-over-xyz](https://github.com/EdOverflow/can-i-take-over-xyz)).

Also check the **NS delegation** case: a subdomain delegated to nameservers that no longer serve the
zone typically yields SERVFAIL or REFUSED rather than NXDOMAIN. Different failure mode, same class
of finding.

**Stage 3 — match the CNAME target against a provider fingerprint list.** `foo.example.com` →
`bucket.s3.amazonaws.com` tells you which provider's error page to expect before you send a request.

**Stage 4 — one HTTP GET, matched against the provider's known error body.** Canonical strings from
[can-i-take-over-xyz](https://github.com/EdOverflow/can-i-take-over-xyz):

| Service | CNAME pattern | Fingerprint string |
|---|---|---|
| AWS S3 | `s3.amazonaws.com` | `The specified bucket does not exist` |
| GitHub Pages | `github.io` | `There isn't a GitHub Pages site here` |
| Heroku | `heroku.com` | `No such app` |
| WordPress.com | `wordpress.com` | `Do you want to register .*.wordpress.com?` |
| Help Scout | `helpscoutdocs.com` | `No settings were found for this company:` |
| Netlify | `netlify.com` | `Not Found - Request ID:` |

The project's entry structure — engine, status (Vulnerable / Not vulnerable / Edge case),
fingerprint regex, test domains, NXDOMAIN flag — is a good schema to mirror directly.

### 7.3 False-positive control — copy the negative matchers

Naive body matching produces false positives, and nuclei's takeover templates show exactly how to
suppress them. From
[aws-bucket-takeover.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/takeovers/aws-bucket-takeover.yaml)
(severity `high`, tags `takeover,aws,bucket,vuln`), a single `GET {{BaseURL}}` gated by
`matchers-condition: and`:

- `dsl: Host != ip` — only fire for name-based hosts, never bare IPs
- body must contain **both** `The specified bucket does not exist` **and** `BucketName`
- **negative** matchers excluding headers `x-guploader-uploadid` and `aliyunoss` (other providers
  echoing similar text)
- **negative** regex on the host excluding legitimate AWS-owned S3 endpoint patterns
- **negative** word matcher on host excluding `amazonaws.com`, `ks3.ksyun.com`, and other cloud
  storage endpoints — i.e. do not report the provider's own domain as taken-over

[github-takeover.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/takeovers/github-takeover.yaml)
uses the same pattern: four alternative GitHub Pages error strings OR'd, then AND'ed with
`!contains(host,"githubapp.com")`, `!contains(host,"github.com")`, `!contains(host,"github.io")`,
plus a DSL extractor that pulls the CNAME records into the finding.

Design rules that fall out of this:

- Every takeover fingerprint needs **negative matchers**, not just positive ones.
- **Corroborate DNS and HTTP.** Fire only when the CNAME chain matches the provider *and* the body
  matches. Either alone is a lead, not a finding.
- **Require two consecutive runs.** A provider having a bad five minutes should not page anyone.
  This is cheap for a daily scanner and eliminates the dominant false-positive source.
- **Extract and display the CNAME chain in the finding.** The operator's first question is "pointing
  at what?" — answer it in the alert.
- **Feed CT-log names in.** A name that appears in CT (§5.3) but does not resolve, or resolves to a
  provider-error page, is a takeover lead the operator's DNS zone export alone would not surface.

### 7.4 Hard boundary: never claim the resource

The only *conclusive* proof of takeover is registering the orphaned resource yourself. verge-asm must
never do this. It costs money, creates real cloud resources under the operator's account without
their knowledge, and in a shared-provider namespace it is indistinguishable from the attack. Report
high-confidence-suspected and let the operator confirm.

Microsoft's remediation guidance is the right thing to surface alongside each finding: remove CNAME
records pointing to FQDNs of resources no longer provisioned; use Azure DNS alias records, which
couple "the lifecycle of a DNS record with an Azure resource"; and use App Service custom domain
verification via an `asuid.{subdomain}` TXT record, since "When such a TXT record exists, no other
Azure subscription can validate the custom domain or take it over"
([learn.microsoft.com](https://learn.microsoft.com/en-us/azure/security/fundamentals/subdomain-takeover)).

---

## 8. Inside the network vs outside it

A self-hosted deployment may sit on either side of the perimeter, and **the two vantage points do not
produce the same picture**. Getting this wrong produces the single most damaging failure mode for the
tool: confidently reporting "exposed to the internet" about something that is not.

### 8.1 Concrete divergences, with mechanisms

**DNS answers differ.** RFC 8499 §6 defines "Split DNS" / "split-horizon DNS" as "situations where
DNS servers provide different answers depending on query source", such that "a domain name that is
notionally globally unique nevertheless has different meanings for different network users", and
defines Views as "A configuration for a DNS server that allows it to provide different responses
depending on attributes of the query" — typically by source IP
([RFC 8499](https://www.rfc-editor.org/rfc/rfc8499.html)). So an internal scanner resolving
`app.example.com` may get an RFC 1918 address while the world gets a public one — and the two are
genuinely different hosts running different code.

**Private addresses are not reachable from outside, by design.** RFC 1918 defines 10.0.0.0/8,
172.16.0.0/12 and 192.168.0.0/16, and specifies that "routing information about private networks
shall not be propagated on inter-enterprise links, and packets with private source or destination
addresses should not be forwarded across such links"
([RFC 1918](https://www.rfc-editor.org/rfc/rfc1918.html)). An internal scanner enumerating
`10.0.0.0/24` finds services that are *by construction* not exposed. Reporting them as exposure is
categorically wrong.

**Scanning your own public IP from inside may not do what you think.** RFC 4787 REQ-9 requires
"A NAT MUST support 'Hairpinning'" with hairpinning behaviour "External source IP address and port",
enabling "communications between two endpoints behind the same NAT when they are trying each other's
external IP addresses" ([RFC 4787](https://www.rfc-editor.org/rfc/rfc4787.html)). The requirement
exists because the behaviour is not universal; where hairpinning is absent or handled differently,
probing your own public IP from inside either fails outright (false "closed" — worse, a false
*negative*) or is short-circuited by the NAT without traversing the inbound firewall policy that
would gate a real external connection (false "open" that does not reflect internet reachability).

**The local segment behaves differently at layer 2.** Nmap uses ARP (IPv4) and Neighbor Discovery
(IPv6) by default for local-segment targets because it is faster and more effective
([man-host-discovery](https://nmap.org/book/man-host-discovery.html)). An internal scanner therefore
sees hosts that are simply invisible from outside — and, since verge-asm runs unprivileged (§3),
does *not* get this capability, meaning internal results will also be inconsistent with what a
privileged internal tool would report.

**Source-IP allowlists invert the result.** Management interfaces, admin panels and databases are
routinely firewalled to internal source ranges. From inside: open, and an "exposed admin panel"
finding fires. From outside: filtered, correctly. The internal result is not merely
less accurate — it is inverted, and it is the *alarming* direction.

**Egress and inspection paths differ.** From outside, probes traverse the CDN/WAF/reverse proxy, so
what is fingerprinted is the *edge*: the CDN's `Server` header, the CDN's cert, the WAF's error
pages. From inside, they hit the origin directly, so the tool fingerprints origin software the
internet never sees. Both are useful; conflating them is not.

### 8.2 Design consequences

1. **Vantage point is a first-class field on every observation**, not a global config setting.
   Persist `vantage_id` (with its egress IP and a `network_position` of `internal` | `external` |
   `unknown`) alongside every port, HTTP and TLS result. A finding without a vantage is
   uninterpretable.
2. **Never emit an exposure-class finding from an internal vantage.** "Exposed admin panel",
   "plaintext HTTP", "unexpected open port" all carry an implicit "…to the internet". From inside,
   downgrade these to `internal-reachable` observations with distinct wording and no alerting.
   Findings that are vantage-independent — cert expiry, weak TLS version, dangling DNS — can fire
   from either side.
3. **Ask the operator, then verify.** A first-run setup step should ask where the deployment sits,
   then check it: compare the scanner's egress IP against RFC 1918 ranges, and compare the resolved
   address of a seed domain from the local resolver against a public resolver. A mismatch is
   positive evidence of split DNS and should be surfaced, not swallowed.
4. **Support multiple vantages in the data model from v1, even if v1 ships one.** The genuinely
   correct answer for a small org is a small external prober plus the internal instance, and
   diffing the two. Internal-open + external-filtered = correctly firewalled (reassuring, worth
   showing). Internal-open + external-open on a management port = a real finding, now with
   corroboration. Retrofitting a multi-vantage schema later is far more expensive than allowing for
   it now.
5. **An internal-only deployment should say so, prominently and permanently.** "This instance is
   internal; exposure findings reflect reachability from your network, not from the internet"
   belongs in the UI, not in a README paragraph the operator read once.

---

## 9. Knobs that must be operator-configurable — and why

Each of these is *not* a preference; there is a specific way the default is wrong for some real
operator.

| Knob | Default | Why it must be configurable |
|---|---|---|
| **Rate limit (per host)** | 50 pkt/s | The single most likely thing to hurt production. Some operators run fragile embedded/OT/legacy devices — nmap notes crash reports are "usually older legacy devices" ([legal-issues](https://nmap.org/book/legal-issues.html)). Must be settable to near-zero, and **per-target**, not just globally: one fragile appliance should not force the whole estate to crawl. |
| **Concurrency (per host / global)** | 20 / 200 | Bounded by the scanner's own link, the target's connection-tracking table, and any shared-tenancy limits. Naabu itself says to tune when not on a VPS ([README](https://github.com/projectdiscovery/naabu/blob/main/README.md)). |
| **Port set (per tier)** | `verge-core` / full range | Estates differ wildly; the shipped list is a prior, not a truth. Must be an editable file, and per-target-group (DMZ web hosts and a management VLAN want different sets). **The top-1000 tier is retired** ([#78](https://github.com/winniel123/verge-asm/issues/78)), and only the **frequency half** of `verge-core` is editable ([ADR-0009](../adr/0009-verge-core-is-a-union.md)). |
| **Full-range sweep** | off | Genuinely risky against stateful middleboxes; genuinely necessary for some estates. Explicit opt-in **per `Seed` scope**, with a rate cap that cannot be disabled, and it never runs unasked — including at onboarding ([ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)). |
| **UDP scanning** | off | Low yield (49 % even at top-1000, [performance-port-selection](https://nmap.org/book/performance-port-selection.html)), high cost, but essential for operators exposing DNS/SNMP/IPMI. |
| **Scan cadence per tier** | daily / monthly | Compliance regimes and change-velocity vary. Must allow *slower*, not just faster. (The weekly tier is retired — [#78](https://github.com/winniel123/verge-asm/issues/78).) |
| **Quiet hours / maintenance windows** | none set | Scanning during a deploy produces pure-noise drift findings. Must be per-target. |
| **Follow redirects** | off | Some estates redirect everything at the edge; without following, every finding is "301". Sub-knob: same-host-only (default on when following is enabled). |
| **HTTP probe paths** | small curated list | The most likely thing to trip a WAF. Operators must be able to shrink it to `/` only — or extend it for their own products. |
| **Cert expiry thresholds** | 30 / 14 / 7 days | 90-day ACME with automation wants 7/3/1; manual annual certs want 60/30/14; and the CA/B schedule takes the maximum to 47 days by 2029-03-15 ([BR.md](https://github.com/cabforum/servercert/blob/main/docs/BR.md)). A hardcoded threshold ages badly. |
| **TLS version/cipher enumeration cadence** | weekly | It is N handshakes per host (tlsx `-version-enum` / `-cipher-enum`, [README](https://github.com/projectdiscovery/tlsx/blob/main/README.md)). Some operators want it daily; some want it never. |
| **Vantage / network position** | prompted at setup | Determines whether exposure findings are meaningful at all (§8). |
| **Source address / interface** | container default | Operators who allowlist the scanner at the edge, or who need it to egress a specific path, must be able to pin it. |
| **User-Agent** | identifying, e.g. `verge-asm/1.0 (+https://…; self-hosted ASM)` | Must be identifiable **by default** so the operator recognises their own traffic in their own logs; must be *changeable* because some WAFs block unknown agents outright, and then no probe works at all. |
| **Per-target enable/disable + pause-all** | enabled | An incident is exactly when the operator wants the scanner to stop immediately. A global kill switch must exist and take effect mid-run. |
| **Adaptive back-off aggressiveness** | halve on error | Operators with lossy links need it gentler; operators with headroom find it too timid. |
| **`suspect-firewall` port threshold** | 100 open ports | Depends on the estate; some hosts legitimately listen on many ports. |
| **Target range size cap** | /22 per target | Prevents a typo'd `/8` from becoming a multi-day scan. |

---

## 10. Explicitly not defaults (and mostly not options)

- **No credential submission of any kind.** Not default-login checks, not "just testing admin:admin".
  nuclei's `default-logins` templates POST real credentials and match on a real session cookie
  ([grafana-default-login.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/default-logins/grafana/grafana-default-login.yaml)).
  Unattended, that is repeated authentication against production. Out of scope for v1 entirely.
- **No vulnerability exploitation or version-specific exploit probes.** Stated project scope: not a
  vulnerability scanner.
- **No state-changing HTTP methods** in any default probe (POST/PUT/DELETE/PATCH). GET is defined as
  safe ([RFC 9110 §9.2.1](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2.1)); stay inside
  that.
- **No `--privileged`, no `cap_add`, no `network_mode: host`** in the shipped compose file (§3).
- **No claiming/registering of suspected-dangling resources** (§7.4).
- **No rate-limit-defeating behaviour.** Nmap's `--defeat-rst-ratelimit` and
  `--defeat-icmp-ratelimit` ([man-performance](https://nmap.org/book/man-performance.html)) have no
  legitimate analogue when the thing rate-limiting you is your own firewall.
- **No IDS-evasion features** (fragmentation, decoys, source spoofing, timing designed to slip under
  thresholds). Nmap's T0/T1 exist for evasion
  ([timing-templates](https://nmap.org/book/performance-timing-templates.html)); a defensive tool
  scanning owned assets wants to be *seen* by its owner's controls, and wants that traffic to be
  attributable.
- **No brute-force enumeration** of directories, subdomains by wordlist against live hosts, or
  parameters. Passive discovery (CT logs, §5.3) and cert SANs cover the subdomain case at a fraction
  of the cost.

---

## 11. Open questions for the spec

1. Should the external-vantage prober ship as a second, tiny compose service (or a documented
   deploy-this-on-a-VPS artefact)? §8 argues the two-vantage diff is where most of the real signal
   is, and a single internal deployment structurally cannot produce it.
2. Where does the `verge-core` port list live, how is it versioned, and how does a list update
   interact with historical diffs? (A port added to the list will look like a newly opened port
   unless the diff is list-version-aware — a guaranteed false-positive burst on upgrade.)
3. Should CT-log monitoring (§5.3) be v1 or v2? It is the cheapest discovery source available, costs
   the operator's infrastructure nothing, and directly feeds both SAN discovery and dangling-DNS
   detection.
4. What is the body-normalisation function for drift hashing? Without it, any page with a CSRF token
   or a timestamp diffs on every run. This is small but load-bearing for the entire drift feature.
5. Is `verge-core` shipped as one list or as per-profile lists (`web-edge`, `internal-mgmt`,
   `data-tier`)? Per-profile is more useful and more work.

---

## Sources

Nmap
- [nmap-services (raw)](https://raw.githubusercontent.com/nmap/nmap/master/nmap-services) — `$Id: nmap-services 9746 2008-08-26 …`, `Fields in this file are: Service name, portnum/protocol, open-frequency`
- [Well Known Port List: nmap-services](https://nmap.org/book/nmap-services.html)
- [Port Specification and Scan Order / port scanning options](https://nmap.org/book/port-scanning-options.html)
- [Port Selection Data and Strategies](https://nmap.org/book/performance-port-selection.html)
- [Port Scanning Techniques (-sS, -sT)](https://nmap.org/book/man-port-scanning-techniques.html)
- [Timing Templates (-T0..-T5)](https://nmap.org/book/performance-timing-templates.html)
- [Timing and Performance options](https://nmap.org/book/man-performance.html)
- [Host Discovery](https://nmap.org/book/man-host-discovery.html)
- [Miscellaneous Options (--privileged / --unprivileged)](https://nmap.org/book/man-misc-options.html)
- [Legal Issues (crash risk)](https://nmap.org/book/legal-issues.html)

ProjectDiscovery
- [naabu README](https://github.com/projectdiscovery/naabu/blob/main/README.md) · [naabu usage](https://docs.projectdiscovery.io/tools/naabu/usage) · [pkg/runner/default.go](https://github.com/projectdiscovery/naabu/blob/main/pkg/runner/default.go) · [pkg/runner/validate.go](https://github.com/projectdiscovery/naabu/blob/main/pkg/runner/validate.go)
- [httpx README](https://github.com/projectdiscovery/httpx/blob/main/README.md) · [httpx usage](https://docs.projectdiscovery.io/tools/httpx/usage) · [runner/options.go](https://github.com/projectdiscovery/httpx/blob/main/runner/options.go)
- [tlsx README](https://github.com/projectdiscovery/tlsx/blob/main/README.md)
- [nuclei-templates: grafana-detect.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/exposed-panels/grafana-detect.yaml) · [grafana-default-login.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/default-logins/grafana/grafana-default-login.yaml) · [aws-bucket-takeover.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/takeovers/aws-bucket-takeover.yaml) · [github-takeover.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/takeovers/github-takeover.yaml)

Other tools
- [masscan README](https://github.com/robertdavidgraham/masscan/blob/master/README.md)
- [can-i-take-over-xyz](https://github.com/EdOverflow/can-i-take-over-xyz)

Containers / kernel
- [Docker: runtime privilege and Linux capabilities](https://docs.docker.com/engine/containers/run/)
- [Docker Compose services reference](https://docs.docker.com/reference/compose-file/services/)
- [capabilities(7)](https://man7.org/linux/man-pages/man7/capabilities.7.html)
- [Linux ip-sysctl (`ping_group_range`)](https://docs.kernel.org/networking/ip-sysctl.html)

RFCs and standards
- [RFC 1918 — Address Allocation for Private Internets](https://www.rfc-editor.org/rfc/rfc1918.html)
- [RFC 2308 — Negative Caching of DNS Queries (NXDOMAIN vs NODATA)](https://www.rfc-editor.org/rfc/rfc2308.html)
- [RFC 4787 — NAT Behavioral Requirements for UDP (REQ-9 hairpinning)](https://www.rfc-editor.org/rfc/rfc4787.html)
- [RFC 5280 — X.509 PKI Certificate and CRL Profile](https://www.rfc-editor.org/rfc/rfc5280.html)
- [RFC 6125 — Service Identity in TLS (SAN vs CN-ID)](https://www.rfc-editor.org/rfc/rfc6125.html)
- [RFC 6797 — HTTP Strict Transport Security](https://www.rfc-editor.org/rfc/rfc6797.html)
- [RFC 6962 — Certificate Transparency](https://www.rfc-editor.org/rfc/rfc6962.html)
- [RFC 8499 — DNS Terminology (split DNS, views, NXDOMAIN)](https://www.rfc-editor.org/rfc/rfc8499.html)
- [RFC 8996 — Deprecating TLS 1.0 and TLS 1.1](https://www.rfc-editor.org/rfc/rfc8996.html)
- [RFC 9110 — HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.html)
- [CA/Browser Forum Ballot SC-081v3](https://cabforum.org/2025/04/11/ballot-sc081v3-introduce-schedule-of-reducing-validity-and-data-reuse-periods/) · [CA/B Baseline Requirements (BR.md)](https://github.com/cabforum/servercert/blob/main/docs/BR.md)

Vendor guidance
- [Microsoft Learn — Prevent dangling DNS entries and avoid subdomain takeover](https://learn.microsoft.com/en-us/azure/security/fundamentals/subdomain-takeover)
