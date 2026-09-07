# The wall-clock of a hot scan at the default cap, and the rate it emits at N workers

- **Status:** Measured — 2026-09-07
- **Ticket:** [#1115 Measure the wall-clock of a hot scan at the default address-scope cap, and the emitted rate at N workers](https://github.com/winniel123/verge-asm/issues/1115)
- **Blocks:** [#1116](https://github.com/winniel123/verge-asm/issues/1116), the budget grant
- **Answers:** [ADR-0137](../adr/0137-the-safety-budget-promises-a-targets-rate-and-enforces-a-vantages.md) §6, which blocks the grant on this measurement
- **Repo commit under test:** `c068bb9`

Nobody measured the active-scan path before this note.
[#1092](https://github.com/winniel123/verge-asm/issues/1092) said so.
[#1105](https://github.com/winniel123/verge-asm/issues/1105) said so again.
ADR-0137 recorded it a third time, and named it the first thing owed.
This note is that measurement.

**The measurement can kill work, and it partly does.** ADR-0137 §6 states the falsifier. If the
hot scan finishes comfortably inside its cadence at one worker, nobody scales, and the grant
guards a case that does not occur. §5 gives that verdict. §7 gives the number that supports the
grant instead. Both are here, and neither run was tuned.

Every number below comes from a run on the rig in §2. §10 lists what this note did not measure.

---

## 1. The numbers this note reports

| Question | Answer | Section |
| --- | --- | --- |
| Wall-clock, one hot scan, default cap, one worker, refusing estate | 56 min 32 s, measured | §4.2 |
| Wall-clock, same scan, silently dropping estate | 20 h 39 min, extrapolated | §4.3 |
| Does one worker fit the daily cadence at the default cap? | Yes, in both cases | §5.3 |
| Scope size where a refusing estate stops fitting | 26,000 addresses, or 11,000 with the drift | §5.2 |
| Scope size where a dropping estate stops fitting | about 1,190 addresses | §5.2 |
| Rate one target receives at one worker | 50 conn/s, as declared | §7.2 |
| Rate one target receives at four workers | 200 conn/s, four times declared | §7.2 |
| Rate one vantage emits at eight workers | 400 conn/s, twice declared | §7.3 |
| Scope size no worker count rescues | about 124,000 addresses per vantage | §8 |

---

## 2. The rig

### 2.1 Host

| Item | Value |
| --- | --- |
| OS | Ubuntu 24.04.4 LTS |
| Kernel | `6.8.0-139-generic`, `x86_64` |
| CPU | 6 cores |
| RAM | 7 GiB |
| Docker | 29.1.3 |
| Go | 1.26.8 |
| Postgres | `postgres:16-bookworm@sha256:60f4761b…` from `compose.yaml` |

The untracked `docker-compose.override.yml` in the working checkout sets `VERGE_DEV=1` on `web`
and `worker`. **It did not affect any run here.** No run used `docker compose`. Every run below
started its own containers by name, and no run set `VERGE_DEV`.

### 2.2 The target, and why it never leaves the host

Every scan here points at a target the operator controls. **No packet reached a third party.**

`custody.EgressGuard` refuses to dial any address the special-purpose table marks
non-globally-reachable (`internal/custody/egressguard.go:22`). Loopback, RFC 1918, and every
documentation prefix fall under that refusal. So a target on `127.0.0.0/8` or `192.0.2.0/24` cannot
be probed at all, and the usual local-listener rig is unavailable.

The rig captures a public prefix on a local bridge instead. Docker holds the whole prefix on a
host bridge interface. The route is local, so a packet to any address inside it stops at the
bridge. Removing the network removes the route.

| Item | Value |
| --- | --- |
| Docker network `vergemeas` | `192.88.96.0/21`, gateway `192.88.96.1` |
| Declared address scope | `192.88.100.0/22`, 1024 addresses |
| Target container `vergetarget` | `192.88.96.2`, holds `192.88.100.0/22` on `lo` |
| Prober container `vergeprobe` | `192.88.96.3` |
| Worker container `vergeworker` | `192.88.96.4`, and `172.28.44.3` for Postgres |

The prober and worker containers route `192.88.100.0/22` **via** `192.88.96.2`, so the host
neighbour table holds one entry and not 1024. The worker container has **no default route**, so
it can reach Postgres and the target and nothing else.

`192.88.99.0/24` inside this range is the deprecated 6to4 relay anycast prefix (RFC 7526).
`192.88.100.0/22` is ordinary public space, held locally for the length of the run. A reader who
holds a public prefix should substitute it.

### 2.3 The scan under test

| Item | Value | Source |
| --- | --- | --- |
| Default address cap | 1024 addresses | `internal/seed/seed.go:17`, `instance_config.seed_address_cap` |
| Scope declared | `192.88.100.0/22` = 1024 addresses, exactly the default cap | `seed.address_cidr` |
| Addresses the custody gate admitted | 1024 of 1024 | §4.2 |
| TCP ports probed per address | 131 | `vergecore.Default().TCPProbed()` |
| UDP ports recorded per address | 5, never probed | `vergecore.Default().UDPRecorded()` |
| Blanket-discriminator control ports per address | 8 | `blanketdiscrim.ControlPortCount` |
| Connects per address against a refusing target | 139 | measured, §7.2 |
| Vantages | 1, the shipped `local` row, `host` NULL, so the prober execs locally | `vantage` |
| Workers | 1 | `docker exec` of one `worker -trigger hot` |
| Hot cadence | 86,400 s | `scan.cadence_seconds` |

---

## 3. How to re-run it

Build the binaries on the host.

```sh
CGO_ENABLED=0 go build -tags netgo -o bin/prober ./cmd/prober
CGO_ENABLED=0 go build -tags netgo -o bin/worker ./cmd/worker
CGO_ENABLED=0 go build -tags netgo -o bin/web    ./cmd/web
```

Create the two networks and the three containers.

```sh
docker network create --subnet 192.88.96.0/21 --gateway 192.88.96.1 vergemeas
docker network create --subnet 172.28.44.0/24 vergemeasctl

docker run -d --name vergetarget --network vergemeas --ip 192.88.96.2 \
  --cap-add NET_ADMIN --cap-add NET_RAW alpine:3.20 sh -c \
  'apk add --no-cache iproute2 tcpdump iptables >/dev/null 2>&1
   ip route add local 192.88.100.0/22 dev lo
   sleep infinity'

docker run -d --name vergeprobe --network vergemeas --ip 192.88.96.3 \
  --cap-add NET_ADMIN alpine:3.20 sh -c \
  'apk add --no-cache iproute2 >/dev/null 2>&1
   ip route add 192.88.100.0/22 via 192.88.96.2
   sleep infinity'

docker run -d --name vergepg --network vergemeasctl \
  -e POSTGRES_PASSWORD=measure1115 -e POSTGRES_USER=verge -e POSTGRES_DB=verge \
  -p 127.0.0.1:55432:5432 \
  postgres:16-bookworm@sha256:60f4761b9035e0b8d5218f701a8c3382f641bf12b1604822574cf5be3baeb537

docker run -d --name vergeworker --network vergemeasctl --cap-add NET_ADMIN \
  alpine:3.20 sh -c 'apk add --no-cache iproute2 >/dev/null 2>&1; sleep infinity'
docker network connect --ip 192.88.96.4 vergemeas vergeworker
docker exec vergeworker sh -c 'ip route add 192.88.100.0/22 via 192.88.96.2; ip route del default'
docker cp bin/worker vergeworker:/worker
docker cp bin/prober vergeworker:/prober
docker cp bin/prober vergeprobe:/prober
docker exec vergeworker mkdir -p /state /tkey
```

Apply the schema. `cmd/web` runs `goose.Up`. `cmd/worker` never does.

```sh
DATABASE_URL='postgres://verge:measure1115@127.0.0.1:55432/verge?sslmode=disable' \
VERGE_STATE_DIR=./state VERGE_TRANSCRIPT_KEY_DIR=./tkey \
VERGE_LISTEN_ADDR=127.0.0.1:18080 VERGE_SETUP_TOKEN=measure1115setuptoken \
./bin/web
```

Stop `web` after the migration log line. Then declare the scope. `seed.created_by` needs an
`account` row.

```sh
docker exec vergepg psql -U verge -d verge \
 -c "INSERT INTO account (username, role, password_hash) VALUES ('measure','admin','x');" \
 -c "INSERT INTO seed (kind, address_cidr, created_by)
     SELECT 'address','192.88.100.0/22', id FROM account WHERE username='measure';"
```

Run the scan. The flag enqueues, drains, and exits.

```sh
docker exec \
 -e DATABASE_URL='postgres://verge:measure1115@vergepg:5432/verge?sslmode=disable' \
 -e VERGE_PROBER_PATH=/prober -e VERGE_STATE_DIR=/state -e VERGE_TRANSCRIPT_KEY_DIR=/tkey \
 vergeworker sh -c 'time /worker -trigger hot'
```

Remove the rig afterwards.

```sh
docker rm -f vergetarget vergeprobe vergeworker vergepg
docker network rm vergemeas vergemeasctl
```

---

## 4. Measurement A — wall-clock

### 4.1 Per-address cost by response class

Each run below fed one `connect-outcome` job spec to the real `cmd/prober` binary inside
`vergeprobe`. The spec carried one address, the 131 verge-core TCP ports, the 5 UDP ports, and
`connectoutcome.DefaultProfile()`. It is the same spec the dispatcher builds.

| Target behaviour | Wall-clock | Connects | Observations emitted |
| --- | --- | --- | --- |
| Refuses every port with RST | 2.76 s | 139 | 131 reachability |
| Answers 4 ports of 131, refuses the rest | 2.76 s | 143 | 131 reachability, 4 certificate |
| Drops every packet silently | 72.05 s | 24 | 131 gap |

The dropping row ran three times. All three took 72.05 s. The capture counted 216 SYNs across the
three runs, or 72 SYNs for 24 connect attempts. The kernel retransmits a SYN that draws no answer,
so one paced connect can put three packets on the wire. §9 records that.

Three facts sit inside those three rows.

**The refusing case is exactly the pacer.** The declared 50 conn/s is a 20 ms interval. The first
connect starts at once, so 139 connects span 138 intervals, or 2.76 seconds. That is the measured
figure. Nothing else in the leaf costs measurable time.

**The dropping case never reaches the port sweep.** The blanket discriminator probes 8 control
ports first. Each attempt times out at 3000 ms. The leaf retries a timeout twice, because
`Retries` is 2. Eight ports at three attempts of three seconds is 72 seconds. All 8 control
results come back incomplete.
`blanketdiscrim.Decide` then returns a gap verdict. `RunExchange` emits 131 gaps **without
probing a single service port** (ADR-0104). The short-circuit saves the sweep. The control probe
still costs 72 seconds.

**The answering case emits 4 connects the pacer never saw.** 143 SYNs arrived, not 139. The
certificate handshake in `RunExchange` uses `NetHandshaker` directly. It does not ride the paced
connector. One reached port is one unpaced connection. §9 files this.

### 4.2 The full scan at the default cap, end to end

One `worker -trigger hot`, one worker process, one vantage, 1024 addresses, refusing target.

```
2026/09/07 00:49:32 dispatcher: hot fanned out 1024 job(s) at 2026-09-07T00:49:31Z
2026/09/07 00:49:32 worker: triggered hot, 1024 job(s) enqueued
```

| Raw count | Value |
| --- | --- |
| Addresses in the declared scope | 1024 |
| Jobs enqueued | 1024 |
| Addresses the custody gate refused | 0 |
| Batches recorded, all `completed` | 1024 |
| Observations recorded, all `reachability` | 134,144 |
| Connects issued | 142,336 |
| **Wall-clock, enqueue to drained** | **56 min 31.99 s** = 3392 s |
| First batch to last batch | 3387.8 s |
| Mean per job | 3.31 s |
| Cadence consumed | 3.93% of 86,400 s |

The per-job mean minus the 2.76 s leaf cost is the worker's own overhead. It covers the claim
query, the prober exec, the NDJSON read, and the batch and observation writes.

**The per-job cost rose steadily through the tick.** Each row below is 128 consecutive jobs of the
same run.

| Jobs | Seconds per job |
| --- | --- |
| 1 to 128 | 2.930 |
| 129 to 256 | 3.043 |
| 257 to 384 | 3.160 |
| 385 to 512 | 3.290 |
| 513 to 640 | 3.351 |
| 641 to 768 | 3.466 |
| 769 to 896 | 3.579 |
| 897 to 1024 | 3.674 |

The rise is 25% from the first block to the last, and it is monotonic. The leaf cost is fixed at
2.76 s, so the whole rise sits in the worker's overhead. That overhead runs 0.17 s at the start of
the tick and 0.91 s at the end. The `observation` table grew to 134,144 rows during the run, and
the growing table is the obvious candidate. **This note did not confirm the cause.** §9 records it.

A smaller control run at 8 addresses (`192.88.100.0/29`) took 23.02 s, a mean of 2.878 s per job.
That run held a listener on 4 ports, so each address also ran 4 certificate handshakes. Its mean
matches the first block of the full run.

Three probes of the dropping range ran in a separate container between job 100 and job 220 of this
run. They cost about 15 s of the 3392 s total, which is 0.4%.

### 4.3 The same scan against a silently dropping estate

This case ran at one address and **not at 1024**. At 72.05 s per address the full scan needs 20.6
hours. This session could not hold the rig that long. The table below extrapolates from the
measured per-address cost, under the assumptions in §5.1.

| Estate | Per address | 1024 addresses | Share of the cadence |
| --- | --- | --- | --- |
| Refuses every port | 3.31 s | **56 min 32 s, measured** | 3.9% |
| Drops every packet | 72.60 s | 20 h 39 min, extrapolated | 86% |
| Half refuses, half drops | 37.96 s | 10 h 48 min, extrapolated | 45% |

The dropping row takes the measured leaf cost of 72.05 s and adds the measured mean worker
overhead of 0.55 s.

The dropping row is the case that matters. It is also the row nobody costed before now.

---

## 5. Extrapolation, and the verdict on ADR-0137 §6

### 5.1 The assumptions

Four assumptions carry every extrapolation in this note. Each one is checkable.

1. **The drain is serial.** `Worker.drain` loops `RunOnce`, and `RunOnce` claims one job and runs
   it to completion. There is no goroutine and no batching in the loop.
2. **A job costs the same whatever the scope size.** This one **fails**, and §4.2 measures the
   failure. The per-job cost rose 25% inside one tick. Every flat-rate figure below is therefore
   an optimistic bound.
3. **Job count is addresses times vantages.** `BuildHotJobs` emits one job per `(address, vantage)`
   pair, with one address in each.
4. **The target's response class sets the cost.** The three classes in §4.1 differ by a factor of
   26. A real estate is a mixture, and this note does not know the mixture.

### 5.2 Where the scan stops fitting the cadence

The hot cadence is 86,400 s. Divide it by the per-address cost. The quotient is the scope size at
which one worker stops draining inside one tick.

| Estate | Per address | Scope size that fills the cadence |
| --- | --- | --- |
| Refuses every port | 3.31 s | about 26,100 addresses |
| Half refuses, half drops | 37.96 s | about 2,280 addresses |
| Drops every packet | 72.60 s | about 1,190 addresses |

**The refusing row is optimistic, and §4.2 says by how much.** The per-job cost rose 0.107
seconds for each 128 jobs. Continue that slope. The refusing estate then fills the cadence at
about 11,300 addresses, not 26,100. **Nothing here measures the slope beyond 1024 jobs.** A reader
should treat 26,100 as the ceiling and 11,300 as the warning.

**A silently dropping estate stops fitting just above the default cap.** 1024 addresses take 86%
of the daily cadence. 1,190 addresses take all of it. The operator never has to raise the cap to
reach that cliff. They only have to point the scan at a network that drops.

### 5.3 The verdict

> **Does one worker fit the cadence at the default cap? Yes.**

Both cases fit, and the answer holds across the span of target behaviour this note covers. A
refusing estate uses 3.9% of the daily cadence. A dropping estate uses 86% of it.

**ADR-0137 §6's falsifier fires.** The ADR set the condition. If the scan finishes comfortably
inside its cadence at one worker, then nobody scales. No second prober then runs on a vantage, and
the grant guards a case that does not occur. At the default cap the scan finishes inside the
cadence.

Two things qualify that, and both are measurements, not readings.

- **"Comfortably" holds for the fast case only.** 86% of a cadence is not comfort. The margin
  against a dropping estate is 14%, at the default cap, with one vantage. A second vantage doubles
  the job count and removes the margin.
- **The grant guards a real case at a raised cap.** §5.2 puts the cliff at about 1,190 addresses
  for a dropping estate. ADR-0127 admits any cap the operator sets. So an operator who raises the
  cap reaches the case the grant guards. §7 shows what happens there today.

The recommendation is in §9.

---

## 6. What the `Batch` recorded

Every batch in every run recorded the same offers document.

```json
{"retries": 2, "technique": "tcp-connect", "host_discovery": "skipped",
 "adaptive_backoff": {"halve_on_429": true, "halve_on_503": true, "halve_on_timeout": true,
                      "touches_deadline": false, "halve_on_rst_spike": true},
 "round_robin_by_host": true, "per_host_concurrency": 20, "per_host_conn_per_sec": 50,
 "connect_timeout_millis": 3000, "per_vantage_packets_per_sec": 200}
```

Three of those values are the claim §7 checks. `per_host_conn_per_sec` is 50.
`per_vantage_packets_per_sec` is 200. `per_host_concurrency` is 20.

---

## 7. Measurement B — the rate at N workers

### 7.1 Method

`tcpdump` ran inside `vergetarget` on `eth0` and captured every inbound SYN. The filter was
`tcp[tcpflags] & tcp-syn != 0 and tcp[tcpflags] & tcp-ack == 0`, narrowed by destination. N prober
processes started together inside `vergeprobe`, each with one full 131-port job spec. The counts
below are SYNs per one-second bucket, taken from the capture.

Each prober process is one job. One worker runs one prober at a time, so N concurrent probers is
what N workers produce on one vantage. The target refused every port during these runs, and no
listener was live, so no unpaced certificate handshake entered the counts.

### 7.2 The rate one target receives

All N probers pointed at `192.88.100.10`.

| N | SYNs at the target | Peak SYN/s | Wall-clock | `Batch` declared |
| --- | --- | --- | --- | --- |
| 1 | 139 | **50** | 2.81 s | 50 conn/s |
| 2 | 278 | **100** | 2.80 s | 50 conn/s |
| 4 | 556 | **200** | 2.80 s | 50 conn/s |

**One worker emits exactly the declared rate.** The 50/s buckets at N=1 match
`per_host_conn_per_sec` to the packet.

**N workers emit N times the declared rate, and the `Batch` still records 50.** This is the claim
#1092 made and could not support. This rig measures it. Four workers put 200 conn/s on one host,
and every batch of that scan asserted 50 conn/s.

The wall-clock does not move. Each prober paces itself to the full budget and finishes in the same
2.8 s. Nothing serialises them, nothing warns, and nothing refuses.

### 7.3 The rate one vantage emits

The N probers pointed at N distinct addresses, one each. The capture counted every SYN into
`192.88.100.0/22`.

| N | SYNs at the vantage | Peak SYN/s | `Batch` declared |
| --- | --- | --- | --- |
| 4 | 556 | **200** | 200 pkt/s |
| 8 | 1,112 | **400** | 200 pkt/s |

**Four workers reach the declared per-vantage aggregate exactly. Eight workers double it.**

This also shows the thing [#1105](https://github.com/winniel123/verge-asm/issues/1105) recorded in
prose. Inside one prober the aggregate ceiling never binds. One address per batch makes the 20 ms
per-host interval later than the 5 ms aggregate one. So a single prober emits 50/s and not 200/s.
The aggregate ceiling binds across processes only, and across processes nothing enforces it.

---

## 8. The ceiling that binds the estate, not the process

The ticket asked for the uncomfortable case. If the scan stops fitting at a size no worker count
rescues, then scaling is the wrong fix. That size exists, and it follows from §7.3.

A refusing address costs 139 connects, and each connect is one packet. The declared per-vantage
budget is 200 packets per second. One vantage may therefore emit 17,280,000 packets in one
cadence. That is

> **about 124,000 addresses per vantage per day.**

Above that scope size, **no worker count fits the cadence without breaking the declared budget.**
The binding constraint is the safety promise, not the process count.

Below that size, more workers do drain faster. They do it by emitting more than the declared rate,
which is what §7.2 measured. So scaling buys cadence by spending the undertaking the `Batch`
records. That is the trade the grant exists to price.

One further case is worse than the arithmetic suggests. A dropping estate spends 72 s per address
on 24 connect attempts and 72 packets. That is one packet per second. Its cost is latency, not
rate. The declared aggregate would allow about 200 such addresses at once. The scan runs them one
at a time.

---

## 9. Findings

This note fixes nothing. The list below is what a later session should file. **No issue was filed
by this ticket.**

1. **Every `Batch` records `per_host_concurrency: 20`, and no code reads it.**
   `grep PerHostConcurrency` finds the struct field, the default value, and one test assertion.
   `RunExchange` probes targets in a serial `for` loop. In-flight is always 1. The declared value
   is 20. This is the largest single finding here. It is why §4.3 costs 20.6 hours. A dropping
   address spends 72 seconds on 24 sequential timeouts. At the declared concurrency of 20 that
   address costs about 3.6 seconds. That figure sits near the per-host rate floor of 2.76 seconds.
   **A declared parameter that nobody implemented carries the whole dropping-estate cadence
   problem.** File this before the grant.
2. **The certificate handshake bypasses the pacer.** `RunExchange` calls `h.Handshake` with the
   bare `NetHandshaker`. Only the connect path is paced. One reached port adds one unpaced
   connection to the target. §4.1 measured 143 SYNs where the pacer accounted for 139. A host with
   many open TLS ports emits proportionally more.
3. **Keep the grant unbuilt for now. Fix the concurrency defect first.** §5.3 says one worker
   fits the cadence at the default cap. Finding 1 says the one thin-margin case has a cheaper fix
   than scaling. A grant built before finding 1 lands prices a scaling decision the operator
   should not need to make.
4. **Retries multiply a timeout by three, and the control probe pays that eight times.**
   A `Retries` of 2 and a 3000 ms connect timeout give 9 s per undecided control port. The
   blanket discriminator needs 8 of them before it can gap. That is the entire 72 s. This note
   asks whether a gap verdict needs three attempts per control port. It does not answer that.
5. **`EgressGuard` makes the leaf untestable against a normal local target.** Every loopback,
   RFC 1918, and documentation prefix is refused at the socket. §2.2 shows the workaround. A
   supported test affordance would remove the need to capture a public prefix.
6. **The drain slows as the tick proceeds, by 25% inside one tick.** §4.2 has the segment table.
   The leaf cost is fixed, so the rise sits in the worker's own overhead. The `observation` table
   reached 134,144 rows during the run. Row growth is the obvious candidate and this note did not
   confirm it. The consequence is in §5.2. A flat-rate extrapolation over-states the scope size
   that fits the cadence, and by a factor of about two at 26,000 addresses.
7. **The budget names packets and the pacer counts connects.** `per_vantage_packets_per_sec` is
   200. The pacer spaces connect attempts, not packets. Against a refusing target the two agree,
   because one connect is one SYN. Against a dropping target the kernel retransmits. The capture
   in §4.1 counted 72 SYNs for 24 connect attempts. The absolute rate stays low in that case.
   Nothing here breaches the declared ceiling. The unit on the recorded field is still not the
   unit the code enforces.

---

## 10. What this note did not measure

Recorded as not measured. Not inferred from the code path.

1. **The 1024-address drain against a dropping estate.** §4.3 is extrapolated from one measured
   address, not run. The procedure is in §3. Add
   `iptables -A INPUT -d 192.88.100.0/22 -p tcp -j DROP` inside `vergetarget`, then run the same
   trigger. Budget 21 hours.
2. **A real estate's mixture of response classes.** §4.1 measures three pure classes. Which
   mixture a real deployment sees is unknown, and this note does not guess.
3. **A remote SSH vantage.** Every run used the shipped `local` vantage, whose `host` is NULL, so
   the prober ran on the worker host. `remoteexec.Probe` pushes a binary over SSH per invocation.
   That per-job cost is not in any number here. It can only raise the per-job overhead.
4. **More than one vantage.** All runs used one. Job count scales with the vantage count, so the
   wall-clock figures multiply. The measurement does not confirm that.
5. **N real worker containers.** §7 ran N prober processes on one host, which is what N workers
   produce on one vantage. It did not run N `worker` binaries against one queue. The queue claim
   uses `FOR UPDATE SKIP LOCKED`, so N workers would take N distinct jobs. This note did not
   confirm that under load.
6. **The cadence-lag gate.** [#1114](https://github.com/winniel123/verge-asm/issues/1114)'s gate
   was live in the build under test. No run made it fire, because no run overlapped two dispatches.
7. **IPv6.** Every address here is IPv4. The leaf treats the families alike, and this note did not
   check that.
8. **Any load on the target beyond SYN counting.** The target refused or dropped. It did not model
   a host under stress, so the adaptive backoff halved only in the dropping case.
