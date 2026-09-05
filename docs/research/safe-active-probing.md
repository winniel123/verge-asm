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
| Continuous port set | **`verge-core`: ~~~140~~ **123** curated TCP ports** (nmap top-100 minus ephemeral noise, plus a modern-services supplement). **[measured]** by [#97](https://github.com/winniel123/verge-asm/issues/97) — §2.3 | §2 |
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

Nmap's default port set is not IANA's registry and not a guess. It is empirical frequency data
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

Nmap's published coverage claims are these. The top 10 ports cover about half of open ports per
protocol. The top 100 covers **78 % of TCP** and 39 % of UDP. The top 1,000 catches **~93 % of TCP**
and 49 % of UDP
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
*and* misdirected. Top-1000 is 10× the cost for coverage of a distribution that no longer matches
the estate being monitored.

### 2.3 Recommended continuous set — `verge-core`

~~Roughly 140~~ **123** TCP ports (see the amendment below). **This is the project's own selection, informed by nmap's published ranking
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
  - Management/OOB: ~~161 (TCP)~~, ~~623~~, 5985, 5986, 9100
  - Remote access: 3389, 5900–5905, 5800

> **Amended by [#97](https://github.com/winniel123/verge-asm/issues/97) — two members left this limb
> and the section was never told, and *"roughly 140"* is not this set's size.**
>
> **`161/tcp` and `623/tcp` are struck.**
> [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s Decision table removed both: SNMP is **161/udp**,
> IPMI/ASF-RMCP is **623/udp**, and **623/tcp** is `oob-ws-http`, a different DMTF protocol on the same
> number — so as written this limb probed the TCP siblings of two UDP services and evaluated neither.
> Neither was ever selected on the frequency half's own terms; both entered through this limb, whose
> stated rule is that each member *maps to a named v1 risk signal*, and after the transport fix neither
> does. **`623/tcp` is re-addable later on a fresh frequency or signal argument, and that would be a new
> widening rather than a reversal.** The strike is recorded rather than deleted, per the
> name-and-withdraw convention, because the limb's original wording is the evidence for
> [`sensitive-ports.md`](./sensitive-ports.md) §6.2's defect.
>
> **The set is 123, not ~140.** **[measured]**, twice independently, and reproducing
> [`nmap-services-licence.md`](./nmap-services-licence.md) §6.2 exactly:
>
> | Component | Count |
> |---|---|
> | nmap top-100 TCP, reproduced at §2.1 | 100 |
> | The deletions named above | **−19** |
> | Retained from nmap's ranking | **81** |
> | The supplement above — 49 entries, of which 5 (`10000`, `9100`, `3389`, `5900`, `5800`) are already retained | **+44 net-new** |
> | The set as this section originally specified it | **125** |
> | ADR-0009's removal of `161/tcp` and `623/tcp` | **−2** |
> | **The frequency half today** | **123, all TCP** |
>
> *"Roughly 140"* is not reproducible from either limb and never was — a session adding `81 + 49`
> without removing the five duplicates gets 130, which is the nearest plausible route to it.
> §6.2 parked the reconciliation as out of scope for a licence question; it is discharged here.
> **`verge-core` itself is larger than this set and is not a list at all**: ADR-0009 makes it
> `frequency-set ∪ sensitive-list`, which is ~~**134 pairs** — these 123, plus the 11 the sensitive
> half contributes alone (6 TCP, 5 UDP). 129 are probed on default settings~~ — **composed, 136 pairs:
> these 123, plus the 13 the sensitive half contributes alone (8 TCP, 5 UDP), of which 131 are probed
> on default settings**, UDP being off (§2.5).
>
> > **Merge reconciliation.** [#95](https://github.com/winniel123/verge-asm/issues/95) admitted
> > `10249/tcp` and `10248/tcp` to the sensitive list in a pass concurrent with #97's, taking it to **41
> > pairs**. **This section's own measurement does not move**: `F` is **123, all TCP**, and **neither
> > new pair is in it** — the orchestration limb below still reads `2375, 2376, 2379, 2380, 6443, 10250,
> > 10255` and stops at the kubelet ([`sensitive-ports.md`](./sensitive-ports.md) §27.12). What moves is
> > the sensitive half's contribution, `|S \ F|`, from 11 to 13. The frequency half is editable and the
> > union is not ([ADR-0009](../adr/0009-verge-core-is-a-union.md)); a sensitive-list admission changes
> > `verge-core` without changing anything on this page.
> >
> > **And [#109](https://github.com/winniel123/verge-asm/issues/109) took the sensitive list to 40 by
> > REMOVING `1433/tcp`, and `verge-core` did not move either.** **[measured]** `1433/tcp` **is** in
> > `F` — it is in the retained top-100 — so `|S \ F|` stays at **13** and the union stays at **136
> > pairs, 131 probed**. This is the removal case of the sentence above: a sensitive-list edit changes
> > `verge-core` only where the pair is **not** in the frequency half, which is 13 of the list's 40
> > pairs and was never `1433/tcp`. [`sensitive-ports.md`](./sensitive-ports.md) §35.12.
>
> **This limb reaches no kube control-plane component beyond the kubelet.** The orchestration limb is
> `2375, 2376, 2379, 2380, 6443, 10250, 10255` and stops there, so `10259/tcp` kube-scheduler and
> `10257/tcp` kube-controller-manager are **not** in the frequency half and enter `verge-core` through
> the sensitive half alone — [`sensitive-ports.md`](./sensitive-ports.md) §29.

Rationale: an ASM tool is not trying to characterise the internet, it is trying to notice
*this estate's* drift. The correct prior is "what does a small org accidentally leave listening,"
which is a different distribution from "what is open across tens of millions of 2008 internet
hosts". ~~`verge-core` should be shipped as an editable list file, not compiled in (§8).~~

> **Withdrawn by
> [ADR-0144](../adr/0144-the-verge-core-body-is-compiled-in-and-an-operator-edit-layers-over-it.md)**,
> per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
> **The body is compiled in.** `internal/vergecore/vergecore.go:47` embeds `verge-core.tsv` and it is
> the only body a production path parses. The prior above — *this estate's drift, not the internet's
> distribution* — survives untouched. What changes is how the operator adjusts it: they **layer**
> `verge_core_frequency_edit` deltas over the shipped base (`internal/queue/hot.go:165`), and the
> frequency half alone is reachable that way ([ADR-0009](../adr/0009-verge-core-is-a-union.md)). A
> replaceable file would let the operator author the `half` column and so move the sensitive half,
> which ADR-0009 reserves to the release.

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
  months. A sweep that runs once leaves `k × cadence` undefined for every timeline it opens.
- **No configured object accounts for its scope**, which [ADR-0005](../adr/0005-scan-execution-model.md)
  refuses outright for ad-hoc runs.
- **It would make [#44](https://github.com/winniel123/verge-asm/issues/44)'s standing aperture
  statement non-constant** — *1–65535* for one night and `verge-core` ever after — falsifying the
  premise that discharged its three-densities obligation.

The onboarding baseline is real and it already exists as an operator act:
[#51](https://github.com/winniel123/verge-asm/issues/51)'s first-run step 4, *Run the first batch*,
which dispatches whichever `Scan`s the operator has enabled. If they enabled the cold tier, the
baseline is full-range. If they did not, it is `verge-core`. Either way it is a button, not a default.

**So a default-settings install measures ~~`verge-core`~~ `verge-core`'s TCP pairs and nothing else,
permanently** — including the
~900 tail ports the retired warm tier used to cover. That is the honest statement of v1's aperture,
and it is stated on `Coverage` rather than left to be discovered: the port-tier line names the tier,
its cadence and its off state, and carries ~~`0 of 39 sensitive pairs unread`~~ ~~`0 of 41 sensitive
pairs unread`~~ ~~`0 of 40 sensitive pairs unread`~~ ~~`0 of 38 sensitive pairs unread`~~
**`5 of 38 sensitive pairs unread`**, **`0 of 38 sensitive pairs the instrument cannot report as
`reached``** and `0 of` ~~`16`~~ **`17`** `rules unevaluable`. ~~Both are true by construction — [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s union
puts every sensitive pair inside the hot set, and~~ **The second is true by construction and the
first was not**, and of the ~~sixteen~~ **seventeen** rules one names a port (fully
covered), four read `Name`s, and ~~eleven~~ **twelve** read a facet on a subject.

> **DENOMINATOR moved to seventeen, 2026-08-15 by
> [#128](https://github.com/winniel123/verge-asm/issues/128)**
> ([ADR-0071](../adr/0071-a-vantage-scoped-claim-is-read-only-at-the-vantage-that-scopes-it.md)),
> which admits `non-globally-reachable-address-resolved-from-internet`. **The `0` is unchanged and
> the new rule does not disturb the aperture claim**: it reads `resolution`'s existing address set,
> which is a facet on a subject and names no port, so it joins the third bucket and the tier bounds
> it no more than the other eleven. *Recorded in the ten-ticket merge reconciliation; #128 and #124
> ran concurrently and neither could see the other's figure.* **The per-bucket split is arithmetic
> here, not a re-walk** — ADR-0032's own seventeen-rule walk is the authority, and running that gate
> whole is its own open item. **The tier bounds which subjects
exist, never which rules can speak**, so what the cold tier buys is drift breadth rather than signal
correctness. A count of unmeasured ports is deliberately absent: it is knowable, which is what makes
it tempting, and it is [#28](https://github.com/winniel123/verge-asm/issues/28)'s refused
estate-completeness score in port clothing.

> **The NUMERATOR is corrected here for the first time, 2026-08-15 by
> [#124](https://github.com/winniel123/verge-asm/issues/124). It is `5`, not `0`, and it always was.**
> Four passes re-checked the denominator and none re-checked the numerator, because the sentence
> struck above looked like a proof: *ADR-0009's union puts every sensitive pair inside the hot set*
> is **true**, and *so the rule reads all of them daily* does **not follow**. ADR-0009 says in the
> same breath that `verge-core` is **136 pairs of which 131 are probed on default settings, UDP being
> off**, and the five it leaves out — `69/udp`, `137/udp`, `138/udp`, `623/udp`, `11211/udp` — reach
> `verge-core` from the **sensitive list alone**. Every one is a sensitive pair a default install does
> not read.
>
> **Membership of `verge-core` is not measurement.** The aperture is `verge-core` ∩ the transports the
> shipped configuration probes, and the count is over the intersection. ~~**`0 of 16 rules unevaluable`
> is untouched**~~: `sensitive-port-reached-from-internet` reads a leg on a `Service` and its domain is
> populated by the 131 TCP pairs, so the rule speaks — what the five cost is **subjects**, not
> evaluability, there being no `Service` for a pair outside the recorded scope.
>
> **The denominator is superseded here, at the site that states it — the rule set is seventeen**
> ([#128](https://github.com/winniel123/verge-asm/issues/128) · ADR-0071, which discharged its other
> consequences by name and not this one, the figure being #124's and in another document). **The
> numerator is not re-filled**: the seventeenth rule reads `resolution` rather than a port, so the
> **A SECOND FIGURE joins this line, 2026-08-15 by
> [#173](https://github.com/winniel123/verge-asm/issues/173) ·
> [ADR-0095](../adr/0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)**,
> and **its value is `0` on every shipped configuration**, so nothing above moves.
> This section **restates** the specification; [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)
> is the site that **states** it, and both carry the figure now.
>
> `unread` counts pairs **outside the recorded scope**, which was the only way the instrument could
> fail to report a pair until [ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md).
> `unanswered` projects onto no `Reach`, so a **UDP** pair inside the scope is probed at cadence,
> writes an observation every run, and can still hold no `Reach` at all. **Open a UDP tier and this
> numerator reads `0 of 38 sensitive pairs unread` while five pairs return `not-evaluable` forever**
> — #124's defect on its third outing, arriving this time because *measured* was read as *a value
> produced*. So the line carries two figures and they never fuse: `unread` is an **invitation** the
> operator can act on, and `the instrument cannot report as reached` is a statement that no
> available action changes. Today `5` and `0`; on a payload-free UDP tier, `0` and `5`. Both are set
> arithmetic over our own list and our own tier config — a **per-cycle** count was refused on #44
> decision 10's constancy premise and #44 decision 7's estate-count refusal.
>
> **`0 of 17 rules unevaluable` is untouched and cannot detect this**, the rule's domain being
> populated by the 131 TCP pairs. The rules figure is over our rules; the pairs figure is over our
> list.
>
> aperture argument above does not reach it, but #137 · ADR-0079 and #138 · ADR-0080 together make it
> **permanently `not-evaluable` on an install with no internet-class vantage** — a second,
> install-shaped route to unevaluability this figure has never counted. `0 of 17` would assert a walk
> nobody has run. Found by [#141](https://github.com/winniel123/verge-asm/issues/141), recorded by the
> merging session, and ticketed.
>
> The line names the transport and, on [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)'s
> own allowance, may point at the tier config, because here an action genuinely exists. It must **not**
> be rounded to zero, which is the unearned clean bill of health #80 went out of its way to say this
> figure was not. Corrected at ADR-0044's site too;
> [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §6.

> **Denominator corrected by [#97](https://github.com/winniel123/verge-asm/issues/97): `0 of 37` reads
> `0 of 39`.** [#80](https://github.com/winniel123/verge-asm/issues/80) measured it when the sensitive
> list was 37 pairs; [#91](https://github.com/winniel123/verge-asm/issues/91) took it to 39.
>
> **And it reads `0 of 41` as composed.** [#95](https://github.com/winniel123/verge-asm/issues/95)
> admitted `10249/tcp` and `10248/tcp` in a pass concurrent with #97's, which is precisely the motion
> the paragraph below anticipates: **the denominator moved a third time and the numerator did not move
> at all.**
>
> **Read the identity rather than the numeral, because the list is still moving.** The **numerator is
> `0` for every possible `N`** — it is not a measurement that can come out otherwise, since ADR-0009's
> union puts every sensitive pair inside `verge-core` by construction, and the five UDP pairs are
> outside the count on [#44](https://github.com/winniel123/verge-asm/issues/44)'s ground that they
> hold no subject at all rather than on any aperture ground. **The denominator is simply `|S|`**, the
> sensitive list's own count. So a pass that adds or removes a sensitive row moves this figure's
> denominator and nothing else, and never needs to re-measure the numerator.
>
> Two ADRs quote this line at the stale `0 of 37` — [ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)
> and [ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md) — and are deliberately left
> for a single reconciliation pass rather than hand-patched while `|S|` is in motion
> ([`sensitive-ports.md`](./sensitive-ports.md) §29.9). **That pass has run and both now read ~~`0 of
> 41`~~ `0 of 40`.**
>
> **And it reads `0 of 40` after [#109](https://github.com/winniel123/verge-asm/issues/109)**, which
> **removed** `1433/tcp` from the sensitive list ([`sensitive-ports.md`](./sensitive-ports.md) §35).
> **The first time this denominator has moved *down*, and the identity above predicted it exactly** —
> the numerator stayed `0`, nothing was re-measured, and **[measured]** the **probed set did not move
> at all**, `1433/tcp` being in the frequency half so `verge-core` stays at 136 pairs / 131 probed.
> The aperture **narrows** by one pair's worth of rule coverage and reveals nothing, so
> [ADR-0014](../adr/0014-only-revealed-generalises.md) does not bite. ADR-0044 and ADR-0047 are
> updated with it.
>
> **And it reads `0 of 38` after [#114](https://github.com/winniel123/verge-asm/issues/114)**, which
> **removed** `9200/tcp` **and** `9300/tcp` from the sensitive list
> ([`sensitive-ports.md`](./sensitive-ports.md) §38). **The denominator's second downward move and its
> first by two**, and the identity predicted it again: the numerator stayed `0`, nothing was
> re-measured, and **[measured]** the **probed set did not move at all**, both pairs being in the
> frequency half so `verge-core` stays at 136 pairs / 131 probed. The aperture **narrows** and reveals
> nothing, so ADR-0014 does not bite. ADR-0044 and ADR-0047 are updated with it.

**No middle tier replaces the retired warm one**, and the refusal does not rest on
[`nmap-services-licence.md`](./nmap-services-licence.md) §3. Under ADR-0009's union, any set authored
on the project's own signal-mapping rule is already inside `verge-core` or is the cold tier's
population at the cold tier's cadence. There is no middle to occupy.

### 2.5 UDP

Off by default. Nmap's own data puts top-100 UDP coverage at 39 % and top-1000 at 49 %
([performance-port-selection.html](https://nmap.org/book/performance-port-selection.html)) — i.e.
even a 1,000-port UDP scan misses half of what is open, while costing far more time because
open|filtered states must be resolved by timeout. The signal-to-cost ratio does not justify making
it a default for a tool that runs unattended. ~~Offer it as an opt-in for a hand-picked list
(53, 123, 161, 500, 623, 1900, 5353) where a finding is genuinely actionable.~~

> **The hand-picked list is WITHDRAWN — superseded by
> [ADR-0009](../adr/0009-verge-core-is-a-union.md), and written here by
> [#102](https://github.com/winniel123/verge-asm/issues/102) under
> [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).**
>
> **The *off by default* ruling above stands and is not revisited** — ADR-0009 says so in terms.
> What is withdrawn is the **list**. ADR-0009: *"measured against the sensitive list that list is
> wrong in the same way `verge-core` was: it covers `161` and `623` and **misses `69/udp`,
> `137/udp`, `138/udp` and `11211/udp`**, four sensitive pairs. Under the union it is superseded
> rather than amended: the UDP leg is **`verge-core`'s UDP pairs** … **Nobody maintains a UDP list by
> hand again.**"* The two numbers the old list carried that are **not** `(port, transport)` sensitive
> pairs — `161` and `623` — are the TCP spellings ADR-0009 removed; the UDP ones are carried by the
> union.
>
> **This is the withdrawal that had to be written here.** §2.3 one section up was struck for ADR-0009
> by [#97](https://github.com/winniel123/verge-asm/issues/97) and §2.5 was not, so a session
> enumerating the UDP opt-in from this paragraph alone builds a seven-member hand list missing four
> sensitive pairs — and §1's summary row routes it straight here.

> **The cost argument above is no longer the whole reason, 2026-08-15 by
> [#141](https://github.com/winniel123/verge-asm/issues/141) /
> [ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md).** *Off by
> default* stands and is still not revisited. What is added is that a session reading this section
> alone would price UDP as a **signal-to-cost** decision it could reverse with a bigger budget, and it
> is not one: `connect-outcome` cannot produce an honest UDP value, the honest union is
> `answered │ refused │ unanswered` on a **sixth leaf**, and its one deciding positive — `answered` —
> is unreachable without a **per-pair elicitation payload**, which is the wire prober
> [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) deferred out of this map. **§13** holds
> the walk. The 39 %/49 % figures above are the same mechanism seen from nmap's end.

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
  entry for every port probe" ([legal-issues.html](https://nmap.org/book/legal-issues.html)).
  Completed-and-closed connections clear state promptly, dangling SYNs age out on a timer.

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

> **Two additions 2026-08-15 by [#124](https://github.com/winniel123/verge-asm/issues/124); nothing
> here is corrected.** The shipped file is `web`, `worker` and `postgres` from **one image**
> ([ADR-0001](../adr/0001-stack-and-runtime.md)) with **three named volumes** — `pgdata`,
> `web-state`, `worker-state` — the last two holding secrets each service **generates** rather than
> receives ([ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)),
> and `postgres` publishing no port. And the **prober binary inherits this posture on the host it is
> pushed to**: it is invoked as an ordinary unprivileged SSH user and needs no capability at all,
> which is a property of §3.1's connect-scan choice rather than of the compose file, and is why
> [#14](https://github.com/winniel123/verge-asm/issues/14)'s *"a host you can SSH into"* does not
> quietly become *a host you can SSH into as root*. Full shape:
> [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §2.

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
  packet to port 443, a TCP ACK packet to port 80, and an ICMP timestamp request". The TCP and ICMP
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
discovered by sweeping**. The operator types in domains and IP ranges they own. The tool does not
need to establish liveness before probing. It needs to record "no ports responded" as a legitimate,
diffable observation. So the default should be the `-Pn` posture — "skips the host discovery stage
altogether … attempt the requested scanning functions against *every* target IP address specified"
([man-host-discovery.html](https://nmap.org/book/man-host-discovery.html)) — with a per-target
sanity cap on range size so a fat-fingered `/8` cannot be entered.

Losing OS fingerprinting is not a loss at all: it is one of the more intrusive things a scanner
does, and an ASM tool for owned infrastructure gets better OS data from the operator's own inventory.

> **Confirmed 2026-08-14 by [#81](https://github.com/winniel123/verge-asm/issues/81) —
> [ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md).** This paragraph is the
> clearest statement in the repository of a question that was open until now, and it is **not
> corrected**. *"Targets are seeded by the operator, not discovered by sweeping"* refuses **host
> discovery**, not range enumeration: the operator's declared IP ranges are walked in full, under
> `-Pn`, and *"no ports responded"* is recorded as a diffable observation about a real subject. An
> address-scope `Seed` therefore **enumerates**; a name-scope `Seed` does not, and §10's refused
> wordlist brute-force is why. The *"cannot be entered"* clause is the operative half of §9's range
> size cap: it is checked at declaration, not applied as a filter while a batch runs.

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
[§16.1.1 Table 4](https://www.rfc-editor.org/rfc/rfc9110.html#section-16.1.1)). ~~No
default probe should ever use POST/PUT/DELETE (§9).~~

> **`No default probe` is too narrow, and the whole of §4.1 is an instance rather than a rule**, marked
> here 2026-09-05 by [#1279](https://github.com/winniel123/verge-asm/issues/1279) under [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
> [ADR-0148](../adr/0148-a-measurement-leaf-sends-an-authored-fixed-request-and-never-mutates-remote-state-or-follows-a-link.md) rules that **every** measurement leaf sends an authored, fixed request shape, mutates
> nothing, and follows no link — `connect-outcome`, `http-exchange`, `tls-acceptance`, `edge-fanout`,
> `resolution-walk`, `wildcard-discrimination` and `blanket-discrimination` alike, plus any leaf not yet
> built. So a mutating method is refused in **any** probe, not merely a default one, and it is refused
> opt-in as well: the method is a declared parameter of the leaf and no declared parameter is ever
> operator-configurable ([ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)). This
> section's GET-over-HEAD reasoning stands unchanged and is the ADR's `http-exchange` row.

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

~~When the operator enables following, use httpx's safer variant: `-follow-host-redirects` ("follow
redirects on the same host only"), cap at 5 (httpx's default is 10 —
`-max-redirects int ... (default 10)`), and honour `-respect-hsts`
([httpx usage](https://docs.projectdiscovery.io/tools/httpx/usage)).~~ Always record the full redirect
chain, so an off-host hop is visible even when it was not followed.

> **The operator never enables following, so this paragraph's premise is withdrawn**, 2026-08-15 by
> [#124](https://github.com/winniel123/verge-asm/issues/124) under
> [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). The
> redirect policy is a **declared parameter of `http-exchange`**
> ([ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)) — following a redirect moves
> the `status` and the `title` the leaf decides — and no declared parameter is ever
> operator-configurable. It ships valued at **not followed**, which is this section's own
> recommendation and its three reasons. §9's row is struck with it. **The last sentence stands and is
> the important one**: the `Location` is recorded, which is the half this section correctly called
> the finding.

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
- **Severity `info`** for detection. The risk judgement is applied separately.
- **Version extraction from the same response** — no extra request.

Strong panel signals, in descending order of reliability:

- an exact `<title>` match
- a favicon MMH3 hash
- a product-specific header or cookie name (e.g. `grafana_session`)
- a unique static asset path
- a generic-but-corroborated body string

Weak signals that produce false positives on their own: status-code-only matches, and the word
"login" or "admin" appearing anywhere.

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
kind in v1, default or opt-in.** Report "a Grafana login panel is reachable from the internet". That
is the actionable exposure. Whether the password is `admin` is the operator's to check, once,
by hand.

~~The panel path list should stay small (10–20 well-chosen paths, not thousands) and be a shipped,
editable data file (§8).~~ A path list is the thing most likely to trip a WAF, because directory-ish
request bursts are the canonical scanner signature.

> **There is no panel path list, and there will not be one**, marked here 2026-09-05 by
> [#1279](https://github.com/winniel123/verge-asm/issues/1279) under [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). Exposed-panel
> detection is [#5](https://github.com/winniel123/verge-asm/issues/5)'s refused fingerprinting and never
> entered the value space; `http-exchange` makes **one** exchange per `Endpoint`, `GET /`, and §9's
> *HTTP probe paths* row is struck. [ADR-0148](../adr/0148-a-measurement-leaf-sends-an-authored-fixed-request-and-never-mutates-remote-state-or-follows-a-link.md) now supplies the rule underneath both refusals: a leaf
> never expands its own target set, so guessing paths is refused on target-set grounds and not only on
> scope. **The last sentence stands and is the important one**, and it is a second reason. **The
> detection-yes-authentication-never paragraph above also stands**, and §10 generalises it.

---

## 5. TLS inspection

A single TLS handshake is the highest-yield probe in the whole tool: one connection, no
authentication, no state change, and it answers several v1 risk signals at once.

### 5.1 Expiry

`Validity` carries `notBefore` and `notAfter`. RFC 5280 §4.1.2.5 requires dates through 2049 to be
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
certs *currently served*. CT monitoring finds names on certs *ever issued*, including for hosts that
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
  config. It does not change hourly.

Note tlsx's default concurrency is 300 with a 5 s timeout and 3 retries
([tlsx README](https://github.com/projectdiscovery/tlsx/blob/main/README.md)). That is tuned for
scanning many *different* hosts. Against a handful of operator-owned hosts it is far too aggressive
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
  2 retries. At 50/s, `verge-core` completes in under 3 s per host. Even a full 65,535-port sweep
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
`suspect-firewall` and suppress the individual findings rather than emitting ~~140~~ ~~129~~ **131** false positives
into the operator's inbox — the probed-on-default-settings count, which
[#95](https://github.com/winniel123/verge-asm/issues/95) took from 129 to 131 by admitting two TCP
pairs to the sensitive half. The 100-port threshold is untouched and is nowhere near either figure.

### 6.4 Scheduling

- **Default cadence: daily**, not hourly. Hourly buys ~23 h of detection latency for 24× the load
  and 24× the log noise. The v1 signals (cert expiry, dangling DNS, new open port) do not move on an
  hourly timescale.
- **Jitter ±20 %.** A scan that starts at exactly 03:00:00 every night is trivially correlated in
  logs — but more importantly, exact periodicity means a transient failure recurs at the same phase
  as whatever else runs at 03:00 (backups, log rotation, cert renewal).
- **Operator-set quiet hours / maintenance windows.** Never probe during the operator's declared
  change window. A scan concurrent with a deploy produces drift findings that are pure noise.
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
A/AAAA records. Store the chain, not just the endpoint. The chain is what identifies the provider.

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

Microsoft's remediation guidance is the right thing to surface alongside each finding:

- remove CNAME records pointing to FQDNs of resources no longer provisioned
- use Azure DNS alias records, which couple "the lifecycle of a DNS record with an Azure resource"
- use App Service custom domain verification via an `asuid.{subdomain}` TXT record, since "When such
  a TXT record exists, no other Azure subscription can validate the custom domain or take it over"
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
exists because the behaviour is not universal. Where hairpinning is absent or handled differently,
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
internet never sees. Both are useful. Conflating them is not.

### 8.2 Design consequences

1. **Vantage point is a first-class field on every observation**, not a global config setting.
   Persist `vantage_id` (with its egress IP and ~~a `network_position` of `internal` | `external` |
   `unknown`~~) alongside every port, HTTP and TLS result. A finding without a vantage is
   uninterpretable.

   > **`network_position` is WITHDRAWN at the site that specifies it.** Rejected by
   > [#14](https://github.com/winniel123/verge-asm/issues/14) as **relative** — a prober in the
   > operator's own VPC is external to their office LAN and internal to their cloud assets, and one
   > enum value cannot hold both — and replaced by a **verified claim**, `Vantage class`, re-checked
   > every batch against the operator's declared address scopes. This paragraph was never amended and
   > is the earliest site of that ruling; written here 2026-08-15 by
   > [#124](https://github.com/winniel123/verge-asm/issues/124) under
   > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
   > The first half of the recommendation — *the vantage is a field on every observation* — is
   > **correct and load-bearing**; it is `vantage` in the timeline key.
   >
   > What replaces the enum: the class is verified over **the address the vantage is observed to
   > present**, which is the address the instance dialled for a prober and `SSH_CLIENT` for the
   > instance itself. An **interface address is not a presented address** — under the literal reading
   > a NATed instance could only verify `internal` by declaring its own LAN, which ADR-0049's
   > 1,024-address cap refuses outright. `CONTEXT.md`'s `Vantage class`,
   > [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §4.
2. **Never emit an exposure-class finding from an internal vantage.** "Exposed admin panel",
   "plaintext HTTP", "unexpected open port" all carry an implicit "…to the internet". From inside,
   downgrade these to `internal-reachable` observations with distinct wording and no alerting.
   Findings that are vantage-independent — cert expiry, weak TLS version, dangling DNS — can fire
   from either side.
3. ~~**Ask the operator, then verify.** A first-run setup step should ask where the deployment sits,
   then check it: compare the scanner's egress IP against RFC 1918 ranges, and compare the resolved
   address of a seed domain from the local resolver against a public resolver.~~ A mismatch is
   positive evidence of split DNS and should be surfaced, not swallowed.

   > **The *ask* half is WITHDRAWN at the site that specifies it**, 2026-08-15 by
   > [#124](https://github.com/winniel123/verge-asm/issues/124) under
   > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
   > *A first-run setup step should ask* is a **wizard**, and
   > [#22](https://github.com/winniel123/verge-asm/issues/22) into
   > [#28](https://github.com/winniel123/verge-asm/issues/28) made the day-one checklist the
   > **zero-coverage rendering of `Coverage` at two densities**, not a sequence of prompts. Read
   > alone and in the present tense this bullet would have a session build the one thing that
   > ticket refused.
   >
   > **Intent survives, and it is declared by an act rather than by an answer.** Provisioning a
   > prober is the declaration *this vantage is on the internet*; declaring an address scope over the
   > instance's presented address is the declaration *this one is inside my boundary*. Both are
   > Declared objects the model already has, both are audit-trailed, and neither can drift away from
   > the deployment it describes the way a stored answer can. The **verify** half stands unchanged
   > and is [#14](https://github.com/winniel123/verge-asm/issues/14) decision 7's three
   > self-contained checks, of which the split-DNS comparison named here is the third.
   >
   > **The RFC 1918 test named here is not the test that ships.** `Vantage class` is verified against
   > the **operator's own declared address scopes and nothing else** — no third party's file and no
   > protocol registry decides which side of their boundary a prober is on, for the same reason none
   > may open the probing gate.
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

Each of these is *not* a preference. There is a specific way the default is wrong for some real
operator.

> **Swept 2026-08-15 by [#124](https://github.com/winniel123/verge-asm/issues/124). Six of the
> eighteen rows are struck at their own cells and three are amended; the rest ship.** This section
> predates the model, and *there is a specific way the default is wrong for some real operator* is
> **necessary and not sufficient**. A knob must additionally clear three gates, and each of the six
> fails at least one:
>
> 1. **It sits outside every `Derivation`** — `CONTEXT.md`'s `Derivation` and
>    [#60](https://github.com/winniel123/verge-asm/issues/60) / [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md):
>    *a declared parameter is authored by the project and ships in the release; an operator's dial
>    may sit anywhere outside every derivation and nowhere inside one.*
> 2. **It cannot silence a finding by narrowing** —
>    [ADR-0009](../adr/0009-verge-core-is-a-union.md)'s *a port the operator can hide is a signal the
>    operator can silence*, generalised by [`measurement-offers.md`](../spec/measurement-offers.md)
>    §1.7 to *an offer the operator can narrow is a finding the operator can silence*.
> 3. **If it moves the aperture it moves a named dimension** — a `Scan` scope, a `Seed`, a `Seed`
>    exclusion; something a `Batch`'s recorded scope diffs, so the change carries its `revealed`, its
>    `Gap` and its coverage-class message.
>
> The full walk, verdict by verdict, is
> [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §5.3.

> **A fourth row is amended after this sweep, 2026-08-15 by
> [#175](https://github.com/winniel123/verge-asm/issues/175).** *Adaptive back-off aggressiveness*
> clears gate 1 today only because it composes `connect-outcome` alone; §13.5 found the gate fails it
> the day a knob-governed leaf's union can move with our own probe rate, which is exactly what a
> shipped `datagram-outcome` would do (spec'd, unshipped —
> [ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md)). The row is
> amended in place, at the site that specifies the knob, rather than left standing until whoever ships
> UDP has to rediscover the hazard — the same move already made for *HTTP probe paths* above, struck
> for a list that does not exist yet on the same gate-1 ground. No new ADR was minted: the rule applied
> is already [ADR-0004](../adr/0004-signals-are-release-coupled-rules.md) /
> [#60](https://github.com/winniel123/verge-asm/issues/60)'s declared-parameter gate, stated once at
> the top of this section, per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
> convention that a mechanism's specifying site carries its own caveat rather than leaving it findable
> only where the hazard was named.

| Knob | Default | Why it must be configurable |
|---|---|---|
| **Rate limit (per host)** | 50 pkt/s | The single most likely thing to hurt production. Some operators run fragile embedded/OT/legacy devices — nmap notes crash reports are "usually older legacy devices" ([legal-issues](https://nmap.org/book/legal-issues.html)). Must be settable to near-zero, and **per-target**, not just globally: one fragile appliance should not force the whole estate to crawl. |
| **Concurrency (per host / global)** | 20 / 200 | Bounded by the scanner's own link, the target's connection-tracking table, and any shared-tenancy limits. Naabu itself says to tune when not on a VPS ([README](https://github.com/projectdiscovery/naabu/blob/main/README.md)). |
| **Port set (per tier)** | `verge-core` / full range | Estates differ wildly; the shipped list is a prior, not a truth. ~~Must be an editable file~~, and per-target-group (DMZ web hosts and a management VLAN want different sets). **The top-1000 tier is retired** ([#78](https://github.com/winniel123/verge-asm/issues/78)), and only the **frequency half** of `verge-core` is editable ([ADR-0009](../adr/0009-verge-core-is-a-union.md)). **Not a file: withdrawn by [ADR-0144](../adr/0144-the-verge-core-body-is-compiled-in-and-an-operator-edit-layers-over-it.md) per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).** The body is compiled in (`internal/vergecore/vergecore.go:47`) and the knob is a **layer** of `verge_core_frequency_edit` deltas over it, which is what makes the frequency-half bound hold. |
| **Full-range sweep** | off | Genuinely risky against stateful middleboxes; genuinely necessary for some estates. Explicit opt-in **per `Seed` scope**, with a rate cap that cannot be disabled, and it never runs unasked — including at onboarding ([ADR-0044](../adr/0044-a-one-off-measurement-has-no-currency.md)). |
| **UDP scanning** — **AMENDED: it is not a shipped knob in v1** | off | Low yield (49 % even at top-1000, [performance-port-selection](https://nmap.org/book/performance-port-selection.html)), high cost, but essential for operators exposing DNS/SNMP/IPMI. **§2.5's *off by default* stands and the UDP leg is `verge-core`'s UDP pairs (ADR-0009). What [#124](https://github.com/winniel123/verge-asm/issues/124) found is that nobody has asked what the shipped instrument would *return*: `connect-outcome`'s union is `connected │ refused │ no-response`, and a connected UDP socket puts no packet on the wire, so `connected` would be a fact about our own kernel rather than about the world. ~~Until that is answered the knob has nothing honest to turn on.~~ **ANSWERED by [#141](https://github.com/winniel123/verge-asm/issues/141) / [ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md), and struck here per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): an honest instrument IS constructible — `answered │ refused │ unanswered` on a sixth leaf `datagram-outcome`, specified and not shipped. The row still does not ship, and the reason is now a different and better one. `unanswered` projects onto no `Reach`, so it returns `not-evaluable`; a payload-free datagram elicits nothing from almost anything; and therefore opening the knob payload-free moves the five pairs from `not-evaluable` because they are outside the recorded scope to `not-evaluable` because the exchange did not decide, buying zero net new firings. What would make the knob worth opening is a per-pair payload table, which is the wire prober [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) already deferred. See §13.** Two consequences that do not wait: the five UDP pairs are **outside v1's aperture**, so the aperture statement reads `5 of 38 sensitive pairs unread` rather than `0` (ADR-0044, corrected there), and **membership of `verge-core` is not measurement**.** |
| **Scan cadence per tier** | daily / monthly | Compliance regimes and change-velocity vary. Must allow *slower*, not just faster. (The weekly tier is retired — [#78](https://github.com/winniel123/verge-asm/issues/78).) |
| **Quiet hours / maintenance windows** | none set | Scanning during a deploy produces pure-noise drift findings. Must be per-target. |
| ~~**Follow redirects**~~ **STRUCK — gate 1** | **not followed, and not a knob** | ~~Some estates redirect everything at the edge; without following, every finding is "301". Sub-knob: same-host-only (default on when following is enabled).~~ **`http-exchange` decides `Responded(status, Location, WWW-Authenticate, Server, title)`; following a redirect moves the `status` and the `title`, so this is a declared parameter of that leaf and none is ever operator-configurable ([ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md), [#124](https://github.com/winniel123/verge-asm/issues/124)). It arrives valued at §4.3's own recommendation — *do not follow* — and §4.3's *"when the operator enables following"* paragraph is struck with it. The `Location` is recorded, which is the half §4.3 correctly called the finding.** |
| ~~**HTTP probe paths**~~ **STRUCK — stale, and gate 1 besides** | **`GET /`, one exchange per `Endpoint`** | ~~The most likely thing to trip a WAF. Operators must be able to shrink it to `/` only — or extend it for their own products.~~ **There is no path list in v1 to shrink or extend: §4.1 makes one request per `Endpoint` and `http-identity`'s value space holds one exchange's worth of fact. The *small curated list* is §4.4's exposed-panel detection, which is [#5](https://github.com/winniel123/verge-asm/issues/5)'s refused fingerprinting and never entered the model. Were there a list, editing it would move `http-identity` and fail gate 1 anyway ([#124](https://github.com/winniel123/verge-asm/issues/124)).** **Amended 2026-09-05 by [#1279](https://github.com/winniel123/verge-asm/issues/1279) under [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md): *there is no list* is an observation and *gate 1* is about dials, so read alone this cell refuses the **knob** and leaves the **list** buildable. [ADR-0148](../adr/0148-a-measurement-leaf-sends-an-authored-fixed-request-and-never-mutates-remote-state-or-follows-a-link.md) refuses the list itself — a measurement leaf never expands its own target set from what a response contains — and that holds whether or not anyone ever proposes a list to edit.** |
| ~~**Cert expiry thresholds**~~ **STRUCK — gate 1, and it was ruled twice before this sweep** | **⅓ of the certificate's validity period, ½ below a ten-day validity — a declared parameter** | ~~30 / 14 / 7 days~~ ~~90-day ACME with automation wants 7/3/1; manual annual certs want 60/30/14; and the CA/B schedule takes the maximum to 47 days by 2029-03-15. A hardcoded threshold ages badly.~~ **The complaint is right and the remedy was wrong, and both halves are already settled elsewhere. [#60](https://github.com/winniel123/verge-asm/issues/60): `N` is a declared parameter of `certificate-expiring`, *"and that is precisely why it may not be a dial"* — a settings field inside a leaf is the one actor that can `Break` the estate without a release. [#67](https://github.com/winniel123/verge-asm/issues/67) then fixed *ages badly* the legal way, by shipping the **fraction** rather than the product, which is [ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md)'s cure. This cell is the ADR-0058 site neither ticket struck; written here by [#124](https://github.com/winniel123/verge-asm/issues/124).** |
| **TLS version/cipher enumeration cadence** | weekly | It is N handshakes per host (tlsx `-version-enum` / `-cipher-enum`, [README](https://github.com/projectdiscovery/tlsx/blob/main/README.md)). Some operators want it daily; some want it never. |
| ~~**Vantage / network position**~~ **STRUCK — it is not a setting at all** | **declared by an act** | ~~prompted at setup~~ Determines whether exposure findings are meaningful at all (§8). **But *prompted at setup* is a wizard ([#22](https://github.com/winniel123/verge-asm/issues/22), [#28](https://github.com/winniel123/verge-asm/issues/28)) and *network position* is the enum [#14](https://github.com/winniel123/verge-asm/issues/14) rejected as relative. Intent is declared by **provisioning a prober** (*this vantage is on the internet*) and by **declaring an address scope covering the instance's presented address** (*this one is inside my boundary*) — see §8.2 rec 1 and rec 3 above, and [#124](https://github.com/winniel123/verge-asm/issues/124).** |
| **Source address / interface** — **AMENDED: it is a `Vantage` change, not a plain setting** | container default | Operators who allowlist the scanner at the edge, or who need it to egress a specific path, must be able to pin it. **Pinning egress moves the address the vantage *presents*, and the vantage is in the timeline key — so under [ADR-0070](../adr/0070-a-control-probe-is-asked-from-where-the-answer-it-discriminates-was-asked-from.md)'s generalisation this is a different `Vantage` and its timelines **open** (`revealed`) rather than a setting quietly relocating where answers were drawn from. It never `Break`s ([#124](https://github.com/winniel123/verge-asm/issues/124)).** |
| ~~**User-Agent**~~ **AMENDED — identifiable stays, changeable is STRUCK on gate 1** | identifying, e.g. `verge-asm/1.0 (+https://…; self-hosted ASM)` — **a declared parameter of `http-exchange`** | Must be identifiable **by default** so the operator recognises their own traffic in their own logs; ~~must be *changeable* because some WAFs block unknown agents outright, and then no probe works at all.~~ **The changeable half fails gate 1: a WAF that blocks unknown agents returns a **different response**, so the string moves `http-identity`, and a leaf's inputs are not the operator's ([#124](https://github.com/winniel123/verge-asm/issues/124)). The cost is stated rather than hidden — an estate whose WAF blocks us records the WAF's identity, which is a true answer to the question the facet asks, *what does a client that names nothing meet*; and the only remedy a knob offers is sending a browser's string, which is the impersonation §10 refuses. The operator's remedy is on their WAF, where the allowlist entry is the identifying string this row already requires.** |
| **Per-target enable/disable + pause-all** — **AMENDED: only in the form the model already has** | enabled | An incident is exactly when the operator wants the scanner to stop immediately. A global kill switch must exist and take effect mid-run. **Pause-all is operational and ships as written. A per-target *disable* is an aperture narrowing, so it is a `Seed` **exclusion** — a Declared claim about where the estate ends, which ~~opens a `Gap` and~~ says so — never a hidden mute, or it is gate 2 arriving one target at a time ([#124](https://github.com/winniel123/verge-asm/issues/124)).** **The `Gap` half was wrong and the *says so* half is now right — [#130](https://github.com/winniel123/verge-asm/issues/130) · [ADR-0074](../adr/0074-an-aperture-narrowing-that-takes-its-carrier-with-it-fires-at-the-scope.md).** A narrowing over ground nothing else cites opens **no `Gap`** ([ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)) — the subject leaves and there is nothing left to hold one — and that is precisely why it **does** say so: it fires **one coverage-class message at the scope**, carrying a count of subjects withdrawn. Marked at the sentence per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md), because read alone this cell would build a `Gap` the model does not open. |
| **Adaptive back-off aggressiveness** — **AMENDED: scoped to the leaf it composes** | halve on error | Operators with lossy links need it gentler; operators with headroom find it too timid. **It clears gate 1 today only because it composes `connect-outcome` alone, whose union — `connected │ refused │ no-response` — it cannot move: the knob changes *when* we retry, never what a completed exchange reports. That stops holding the moment a knob-governed leaf's union can move with our own probe rate, which [ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md) §13.5 found is exactly what ICMP rate limiting does on a connectionless leaf: `refused` vs `unanswered` for one target can flip depending on how fast we probed. Per-host rate and retry count are then declared parameters of *that* leaf, project-authored and fixed at the release — never a dial ([ADR-0004](../adr/0004-signals-are-release-coupled-rules.md) / [#60](https://github.com/winniel123/verge-asm/issues/60)). **This knob is therefore scoped per-leaf, not global**: it governs `connect-outcome` retry pacing today and does not extend by default to any future leaf whose union it could move — including `datagram-outcome` should it ship — merely because both are "retry" behaviour. A future connectionless leaf's back-off, if any, is that leaf's own declared parameter, gated by its own corpus. Ruled by [#175](https://github.com/winniel123/verge-asm/issues/175); the hazard was named without being closed by [#141](https://github.com/winniel123/verge-asm/issues/141) / ADR-0083 §13.5.** |
| ~~**`suspect-firewall` port threshold**~~ **STRUCK — there is no such rule** | ~~100 open ports~~ | ~~Depends on the estate; some hosts legitimately listen on many ports.~~ **The v1 rule set is **sixteen** ([ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md)) and `suspect-firewall` is not among them — §6.3's suppression heuristic was never carried into the model. A threshold for a rule that does not exist is not a knob, and were the rule ever admitted its threshold would be a **declared parameter** on [#60](https://github.com/winniel123/verge-asm/issues/60)'s ground rather than a dial ([#124](https://github.com/winniel123/verge-asm/issues/124)). The underlying observation — a host answering on every port — is still recorded; what is absent is a rule reading it.** |
| **Target range size cap** | 1,024 addresses per scope (`/22` in IPv4, `/118` in IPv6) | Prevents a typo'd `/8` from becoming a multi-day scan. **Checked when an address scope is *declared*** — §3.4's *"cannot be entered"* — and applied **per scope, never to a sum**, so four `/22`s are four deliberate acts. It bounds a Declared act's cost and asserts nothing about whether the claim is true; custody at a larger scale belongs to a `custody extension`, which the cap does not reach ([ADR-0047](../adr/0047-an-address-scope-is-its-own-enumeration.md)). **The unit is addresses and the knob is family-blind** — there is no separate IPv6 cap, which is why `/64` and every prefix an operator is assigned is refused: one `/64` is ≈ 4.1 × 10¹¹ years at the 200 pkt/s ceiling, so IPv6 space is not swept and an IPv6 estate is reached by a name scope with a `custody extension` ([ADR-0049](../adr/0049-an-address-scope-is-family-agnostic-and-the-cap-counts-addresses.md)). |

---

## 10. Explicitly not defaults (and mostly not options)

- **No credential submission of any kind.** Not default-login checks, not "just testing admin:admin".
  nuclei's `default-logins` templates POST real credentials and match on a real session cookie
  ([grafana-default-login.yaml](https://github.com/projectdiscovery/nuclei-templates/blob/main/http/default-logins/grafana/grafana-default-login.yaml)).
  Unattended, that is repeated authentication against production. Out of scope for v1 entirely.
- **No vulnerability exploitation or version-specific exploit probes.** Stated project scope: not a
  vulnerability scanner.
- **No state-changing request of any kind, in any probe** — ~~in any default probe~~
  (POST/PUT/DELETE/PATCH, and DNS `UPDATE`, `AXFR` and `IXFR` on the resolution leaves). GET is
  defined as safe ([RFC 9110 §9.2.1](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2.1)).
  Stay inside that.
  > **Widened from *default probe* to *every leaf*, and from HTTP to every protocol**, 2026-09-05 by
  > [#1279](https://github.com/winniel123/verge-asm/issues/1279) under [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
  > [ADR-0148](../adr/0148-a-measurement-leaf-sends-an-authored-fixed-request-and-never-mutates-remote-state-or-follows-a-link.md) makes this a property of a measurement leaf rather than of a default: it also binds
  > `connect-outcome`, `tls-acceptance`, `edge-fanout`, `resolution-walk`, `wildcard-discrimination`
  > and `blanket-discrimination`, and it binds an unshipped leaf such as `datagram-outcome`
  > ([ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md)). Read alone,
  > *in any default probe* left a mutating opt-in on the table; there is none, and the method is a
  > declared parameter besides.
- **No crawling, and no target a response handed us.** A probe never follows a link, never guesses a
  path, and never adds a target because a response mentioned one. Its targets arrive in the job spec
  and the job spec comes from custody. The one bounded exception is `resolution-walk`'s delegation
  walk — a single non-recursive `NS`-then-`SOA` hop whose discovered authorities pass the egress
  guard ([ADR-0121](../adr/0121-the-operator-declared-recursive-resolver-is-trusted-and-exempt-from-the-discovered-authority-egress-guard.md)) — and it is not precedent for a second one ([ADR-0148](../adr/0148-a-measurement-leaf-sends-an-authored-fixed-request-and-never-mutates-remote-state-or-follows-a-link.md) §3).
  This bullet generalises the *no brute-force enumeration* bullet below, which stated the same rule
  for one technique.
- **No `--privileged`, no `cap_add`, no `network_mode: host`** in the shipped compose file (§3).
- **No claiming/registering of suspected-dangling resources** (§7.4).
- **No rate-limit-defeating behaviour.** Nmap's `--defeat-rst-ratelimit` and
  `--defeat-icmp-ratelimit` ([man-performance](https://nmap.org/book/man-performance.html)) have no
  legitimate analogue when the thing rate-limiting you is your own firewall.
- **No IDS-evasion features** (fragmentation, decoys, source spoofing, timing designed to slip under
  thresholds). Nmap's T0/T1 exist for evasion
  ([timing-templates](https://nmap.org/book/performance-timing-templates.html)). A defensive tool
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

## 13. What a UDP leg would return — the honest union, and why an honest instrument is still not worth turning on

[#141](https://github.com/winniel123/verge-asm/issues/141) /
[ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md). §2.5 and §9's UDP
row are amended by this section. §2.4's aperture statement is **not** — nothing here moves a figure.

### 13.1 The question, and the two things it is not

[#124](https://github.com/winniel123/verge-asm/issues/124) found that four amendment passes had
priced UDP without anyone asking what the instrument would **return**. `connect-outcome` decides
`connected │ refused │ no-response`. `connect(2)` on a datagram socket puts no packet on the wire. It
only fixes the peer address in our own kernel, so `connected` would be a statement about us.

It is not a question about whether UDP ships — §2.5's *off by default* stands, ADR-0009 says so in
terms, and turning it on is an aperture change. It is not a question about `verge-core`'s membership
either: the five UDP pairs (`69`, `137`, `138`, `623`, `11211`) are in the union and stay there, and
**membership of `verge-core` is not measurement**. The question is whether the instrument could ever
report honestly, because until it is answered ADR-0009's UDP leg has nothing to turn on.

### 13.2 Walk all three members — the set does not fail the way the ticket's sentence implies

| Member | Honest for a connectionless exchange? | Why |
| --- | --- | --- |
| `connected` | **No** | There is no handshake to complete. The value would be a fact about our own kernel |
| `refused` | **Yes — reused unchanged** | An ICMP Destination Unreachable / Port Unreachable is the same fact in kind as an RST: the host is up, the datagram arrived, nothing is bound. The middlebox objection is symmetric with TCP and ADR-0011 accepted it there |
| `no-response` | **No — and not for `connected`'s reason** | It is honest about *what we measured* and dishonest about *what it projects* |

The third row is the finding. `no-response` projects to `not-reached`, which for TCP is sound — we
reached no listener that would answer. For a connectionless exchange the identical projection says a
live, internet-reachable, unauthenticated listener that ignores our datagram is **not reached**, and
`sensitive-port-reached-from-internet` then reports a clean bill of health on precisely the pairs the
sensitive list exists for. That is ADR-0010's founding defect — *it simply never fires on the more
alarming case* — reproduced by a projection rather than by a stale list, and it is #124's aperture
defect one layer down: a `0` standing where `5 unread` was true, this time inside a `Signal` instead
of inside a coverage line.

**The honest union is `answered │ refused │ unanswered`.** `answered` reads that a datagram came back
from that `(address, port)` and never which bytes, so it stays clear of
[#5](https://github.com/winniel123/verge-asm/issues/5)'s fingerprinting line. `unanswered` is a
**value** — recording nothing would make it indistinguishable from *we did not look*, which ADR-0011
refuses — and it is not a `Gap`, because we looked. It projects onto **neither** `Reach` value, and
ADR-0010 refuses a third by name while [#40](https://github.com/winniel123/verge-asm/issues/40)
deleted `unknown` outright, so the leg simply holds no value and the rule returns `not-evaluable`
through the mechanism that already exists.

### 13.3 It is a sixth leaf, not a widened one

`connect-outcome`'s stimulus is *"a socket event and a clock"*
([ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)). A UDP measurement's stimulus is a
datagram we compose and an ICMP message or datagram we receive. Widening the existing leaf would put
both decisions under one version, so **a UDP payload edit would `Break` every TCP `reachability`
timeline in the estate** — spending ADR-0021's *no leaf is composed by every timeline* property to
save a name. The honest UDP decision is `datagram-outcome`, a sixth leaf, **specified and not
shipped**. v1's leaf count stays five and `reachability`'s union stays three members: the UDP variants
are **strictly additive** by ADR-0011's CI-checkable test — every row whose output would move is a UDP
row and no UDP row has ever produced an observation — so there is no deadline and no reason to spend
them early.

### 13.4 Why it is still not worth turning on, which is a different reason from §2.5's

A datagram carrying no protocol-specific payload elicits nothing from almost anything. That is why
nmap ships a payload file at all, and it is the mechanism behind the coverage figures §2.5 already
quotes — **39 % at top-100 UDP and 49 % at top-1000**
([performance-port-selection](https://nmap.org/book/performance-port-selection.html)): the misses are
not ports nmap failed to try, they are ports that returned nothing to try against.

So a payload-free `datagram-outcome` returns `unanswered` on the five sensitive UDP pairs essentially
always, and `unanswered` returns `not-evaluable` — which is **what those five report today**, ADR-0009
recording them as *not-evaluable on default settings, by design and visibly*.

> **Opening the knob payload-free moves the five pairs from `not-evaluable` because they are outside
> the recorded scope to `not-evaluable` because the exchange did not decide, and changes nothing the
> operator sees.** It costs probe traffic, a sixth leaf, a golden corpus of a fourth medium and an
> eighth aperture input, and it buys **zero** net new firings — which is
> [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md)'s wire-prober refusal in a second
> costume.

**The elicitation payload is therefore the whole of the knob's value, and it is the instrument this
map already deferred.** A per-pair payload table with a per-protocol encoder is
`listener-negotiation`'s dispatch table under another name. Its first obligation is the one ADR-0015
named and nobody has closed — *§7.2 argues a wrong dispatch guess fails safe for the data, and never
asks whether it is safe for the listener* — now aimed at production over a transport with no handshake
to fail on. Classification, so the successor need not re-derive it: under
[#31](https://github.com/winniel123/verge-asm/issues/31) / ADR-0008 a table deciding *where to look*
is aperture and one deciding *what an answer means* is a signature database. A payload table decides
which pairs can produce a positive at all, so it is **aperture**, and an **eighth aperture input** the
day UDP ships.

### 13.5 The hazard nobody had named: ICMP rate limiting puts our own probe rate inside the value

Hosts rate-limit the emission of ICMP error messages. Probing many closed UDP pairs on one address
therefore yields `refused` for some and `unanswered` for others **in the same world**, split by **how
fast we probed**. ADR-0021's alternatives table refuses exactly this shape one layer up — *"had it
moved the deadline, a value would depend on how busy the run was"* — and for `connect-outcome` the
refusal holds, which is why §9's *adaptive back-off aggressiveness* knob clears gate 1 today.

It would not hold for `datagram-outcome`. Two consequences, both the successor's:

1. **Per-host rate and retry count are declared parameters of the UDP leaf, and adaptive back-off may
   not compose it** — or the leaf is non-deterministic and fails the golden-corpus gate, which is
   [`project-authored-constants.md`](./project-authored-constants.md) §6.1's existing argument arriving
   on a second leaf. §9's back-off row would fail gate 1 for that leg the day UDP ships.
2. **It is a second and worse unbounded `Span` generator.** The map already asks how many `Span`s an
   unstable network writes on `reachability`, noting `refused` ↔ `no-response` is the corpus's only
   unbounded generator and is silent by design. `refused` ↔ `unanswered` flaps on **our own rate**
   rather than on the network — unbounded for a reason the operator cannot fix and we can.

> **Resolved by [#175](https://github.com/winniel123/verge-asm/issues/175), rather than left for
> whoever ships UDP.** §9's back-off row is amended in place to state the per-leaf scoping this section
> found, so a session extending back-off to `datagram-outcome` meets the caveat at the row it would
> edit, not only here. No new ADR was minted: the rule is already ADR-0004 /
> [#60](https://github.com/winniel123/verge-asm/issues/60)'s declared-parameter gate, stated once at
> §9's own top and now applied to this row too, per
> [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)'s
> convention that a mechanism's specifying site carries the caveat rather than only the site that
> named the hazard.

### 13.6 The elicitation walk — all five are answerable in principle, and two carry caveats that move the price

Read against each protocol's own specification. **Spec-verified, not measured** — no probe was run,
and the claims inherit ADR-0021's rider that *a corpus row inherits the evidential status of the claim
it encodes*. A retrieval against **implementations** is owed before any payload ships.

| Pair | Elicits a reply? | The message, and its footing |
| --- | --- | --- |
| `69/udp` TFTP | **Yes**, with a caveat that reaches the value space | An RRQ for a nonexistent file returns **ERROR opcode 5, code 1 *File not found*** — RFC 1350 §2, §4, §5 and the Error Codes appendix. **But §4 fixes the reply's source port as a fresh server-chosen TID, not 69**, and §2 records the ERROR packet is neither acknowledged nor retransmitted |
| `137/udp` NetBIOS-NS | **Yes** | NAME QUERY REQUEST → POSITIVE/NEGATIVE NAME QUERY RESPONSE, NODE STATUS REQUEST → NODE STATUS RESPONSE — RFC 1002 §4.2.12–14, §4.2.17–18. **§5.1: *a RESPONSE packet is always sent to the source UDP port and source IP address of the request packet.*** The wildcard `*` NBSTAT query is what makes it unconditional |
| `138/udp` NetBIOS-DGM | **Yes — and this one inverts the expectation** | A DIRECT_UNIQUE datagram naming a destination the target does not own returns **DATAGRAM ERROR (§4.4.3), code 82h *DESTINATION NAME NOT PRESENT*, to the source IP and source UDP port** — RFC 1002 §5.3.3's own pseudocode, message types at §4.4.1. Correctly-addressed traffic is one-way and DATAGRAM QUERY REQUEST is NBDD-only (§5.3.4), which is what makes the service *look* unanswerable |
| `623/udp` IPMI / ASF-RMCP | **Yes, unauthenticated** | **Presence Ping (80h) → Presence Pong (40h)** — DMTF DSP0136 (ASF 2.0) §3.2.4.8 and §3.2.4.3, with §3.2.1's rule that our source port becomes the reply's destination port. §3.2.3 places discovery **before** session creation, so no credential is involved. §3.2.2 permits a device to answer only unique Message Tags, so vary the tag across retries |
| `11211/udp` memcached | **Yes by protocol, no by shipped default** | `doc/protocol.txt`'s UDP section defines the 8-byte frame header and states *the server's response will contain the same ID as the incoming request*. **But `doc/memcached.1` says `-U` defaults to port 0, *which is off*, and the 1.5.6 release notes say the release *primarily disables the UDP protocol by default***. A live `11211/udp` today means a pre-2018 build, a distro that re-enables it, or a deliberate `-U 11211` |

Three consequences, and the first two reach the ruling above.

**`answered` reads *a datagram came back to the socket we sent from*, not *from the port we
probed*.** TFTP forces it: bind the predicate to the target port and TFTP is silently unreadable.
Nmap's own Table 5.3 binds it — *any UDP response from target port* — and collides with RFC 1350 §4
exactly there. The **implementation consequence is sharp and ironic**: the leaf must use an
**unconnected** socket, because a connected datagram socket filters on peer address *and port* and
would drop TFTP's reply. The ticket opened on *a connected UDP socket puts no packet on the wire*. The
instrument needs an unconnected one for a second and independent reason.

**Nmap cannot supply the table, and for two separate reasons.** Mechanically, it has **no payload for
`138/udp`** — the one pair whose reply the walk above found least expected — so `nmap -sU -p138` sends
a zero-length datagram, which matches none of RFC 1002 §5.3.3's cases and can only ever report
`open|filtered`. And legally, the standalone `nmap-payloads` file is gone from nmap `master`. Payloads
are now built from **`nmap-service-probes`**, which is NPSL data.
[#78](https://github.com/winniel123/verge-asm/issues/78) cleared *deriving* `verge-core` from
`nmap-services` and clears nothing about selecting rows out of a second nmap data file, and
[#128](https://github.com/winniel123/verge-asm/issues/128)'s rule cuts against us here, since
*selecting from* a table **is** authoring. **The table must be authored against the protocols' own
specifications** — which the walk shows is possible for all five and unavoidable for one.

**The 39 %/49 % figures are the same mechanism, quantified from nmap's end.** Nmap's own words:
*"for most ports, this packet will be empty… open ports rarely respond to empty probes"*, and *"UDP
services generally define their own packet structure rather than adhering to some common general
format."* And its rate-limiting note is the §13.5 hazard confirmed at the source: *"many hosts rate
limit ICMP port unreachable messages by default. Linux and Solaris are particularly strict about
this"* — a one-per-second limit taking a full-range UDP scan past 18 hours.

### 13.7 Where this is thin

**Implementations were not read, only specifications.** RFC 1002 §5.3.3's DATAGRAM ERROR is the
weakest of the five: whether Samba's `nmbd` and the Windows datagram service actually emit it was not
verified and no owner documentation for it was found. It is also the row that would have to be
authored from scratch, so it is simultaneously the least-evidenced and the most expensive. And the
IPMI 2.0 specification itself could not be retrieved — Intel returns 403 — so `623/udp` rests on
DSP0136, which is the normative source IPMI references and is sufficient, but the corroborating read
was not made.

**`unanswered` would be v1's first facet value with no Derived projection.** ADR-0010's clean identity
— *absence of a `Reach` value means the pair was outside the recorded scope* — would stop holding, and
`Coverage`'s reading of a `Gap` changes with it. ~~Nobody has drawn what that renders.~~

> **ANSWERED 2026-08-15 by [#173](https://github.com/winniel123/verge-asm/issues/173) ·
> [ADR-0095](../adr/0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md),
> and struck here per [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).**
> `Coverage` renders it on the **aperture statement's port-tier line**, as a **second figure** beside
> `unread` — *`M of 38 sensitive pairs the instrument cannot report as reached`* — valued `0`
> today and `5` on a payload-free UDP tier. ADR-0010 takes a conditional at three sentences and a
> strike at none: its identity holds while every probed transport's outcome union projects totally
> onto `Reach`, which TCP's does. **The route splits by history** — a `Gap` where the leg had already
> opened, and *nothing at all* where it never did — and the second is why the aperture statement had
> to carry it, having no span and therefore no recorded cause. **Still undrawn is the surface**: no
> prototype contains an aperture statement at all, `prototypes/coverage/` predating #44. That is a
> successor ticket, not a defect in this section.

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
- [UDP Scan (`-sU`), Table 5.3, and Speeding Up UDP Scans](https://nmap.org/book/scan-methods-udp-scan.html) — §13
- [nmap-payloads](https://nmap.org/book/nmap-payloads.html) · [nmap-service-probes (raw)](https://raw.githubusercontent.com/nmap/nmap/master/nmap-service-probes) — §13.6. The standalone payload file is gone from `master` and payloads are built from the probe file, which is NPSL data

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
- [RFC 1350 — The TFTP Protocol (Revision 2)](https://www.rfc-editor.org/rfc/rfc1350.html) — §13.6, the ERROR packet and the server-chosen reply TID
- [RFC 1002 — NetBIOS over TCP/UDP: Detailed Specifications](https://www.rfc-editor.org/rfc/rfc1002.html) — §13.6, name-service responses (§4.2, §5.1) and the datagram service's DATAGRAM ERROR (§4.4, §5.3.3)
- [DMTF DSP0136 — Alert Standard Format (ASF) 2.0](https://www.dmtf.org/sites/default/files/standards/documents/DSP0136.pdf) — §13.6, Presence Ping / Presence Pong and the RMCP port rules
- [memcached: protocol.txt (UDP protocol)](https://github.com/memcached/memcached/blob/master/doc/protocol.txt) · [memcached.1 (`-U` defaults to off)](https://github.com/memcached/memcached/blob/master/doc/memcached.1) · [UDP DDoS advisory](https://docs.memcached.org/advisories/ddos/) — §13.6
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
