# What an observation-driven insecure-listener rule can establish

Research ticket #31 — wayfinder research for the verge-asm v1 spec.

**Question.** What can an **observation-driven** "insecure listener" rule establish, inside
[#4](https://github.com/winniel123/verge-asm/issues/4)'s safety profile?

**Framing.** [#21](https://github.com/winniel123/verge-asm/issues/21) found that where a real
gradation exists on the sensitive-port question, **it is measurable rather than curatable**. A
curated list can only say *this port usually means this service*. An observation can say *this
listener offered no TLS*, which is a fact about the estate rather than about our list — and whose
reference data is not a list at all, so
[ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)'s release-coupling test applies to it
very differently.

That promise has a clause hidden in it, and finding the clause is most of this note. *This listener
offered no TLS* is only a signal if we also know the listener **should** have offered TLS — and the
natural form of that second half is a port-to-protocol map, which is the same reference data #21
already carries. Worse: for most of the protocols in question the **client speaks first**, so the
prober cannot even ask without first deciding, from the port number, which protocol to speak. The
determinacy problem #21 solved by exclusion does not vanish. It moves from the verdict to the
dispatch.

What rescues the idea is a distinction that turns out to be this note's methodological finding, and
it is stated once here because everything downstream leans on it:

> **A table that decides where to look is aperture. A table that decides what an answer means is a
> signature database.** #5 rejected the second. #4's ~~~140-port~~ **136-pair** `verge-core` hot set is
> already reference data of the first kind and nobody calls it fingerprinting. §2.1 shows the two kinds
> measured side by side in the canonical corpus, where they differ by a factor of 65.

Three decisions already made shape the answer before any evidence is gathered:

1. **Only observed values enter drift** ([#5](https://github.com/winniel123/verge-asm/issues/5)). A
   verdict computed from a signature-like corpus is fingerprinting under another name. This is the
   sharpest constraint and §2.1 and §7 are where it is discharged.
2. **Absent evidence yields `not-evaluable`**, never a clean bill
   ([ADR-0004](../adr/0004-signals-are-release-coupled-rules.md)). The `not-evaluable` surface for
   these rules is larger than for any signal in the v1 set — §11.
3. **Credentials are never submitted** (#4 §10, unconditional: "not default-login checks, not 'just
   testing admin:admin'"). Determining *anonymous access permitted* without crossing that line is
   the crux, and §6 draws the line by a test rather than case by case.

---

## 1. Summary

| Decision | Answer |
|---|---|
| Reachable from a TCP connect plus a TLS handshake alone | **One fact, and it is too weak to ship**: implicit TLS present or absent. Table-free, and fires on every plaintext-by-design listener in the estate — §3 |
| Free by-product of the handshake #4 already performs | **Server-speaks-first vs silent**, distinguishable from the TLS failure mode alone, zero application bytes. Measured, §3.3 |
| Reachable from the first server flight (zero client bytes) | MySQL capability flags; the four line-oriented greetings (SMTP, FTP, POP3, IMAP). **IMAP `* PREAUTH` is the only *anonymous-access* fact free at connect** — §4 |
| Reachable from one protocol-mandated hello, no credential | PostgreSQL `SSLRequest`; LDAP StartTLS and a bind-free root-DSE read; AMQP `Connection.Start`; RFB security-type list; RDP negotiation; SMB2 `NEGOTIATE`; SMTP `EHLO`; IMAP `CAPABILITY`; POP3 `CAPA`; FTP `FEAT`; Redis `PING`; any HTTP `GET /` — §5 |
| Out of scope — the only test is the authentication message | FTP `USER anonymous`; LDAP anonymous simple bind; MQTT `CONNECT`; PostgreSQL `StartupMessage`; MySQL/MongoDB login; SMB `SESSION_SETUP` — §6 |
| Where the hard line falls | **The auth-message test**: the prober may send any message the protocol defines *except* its authentication message. An empty credential is still a credential — §6.1 |
| Redis `PING` before `AUTH` | **In.** `AUTH` is a separate command and `PING` is not it — §6.3 |
| Anonymous LDAP bind | **Out** — RFC 4511 calls Bind "the 'authenticate' operation". And **the exclusion costs nothing**, because RFC 4513 §4 gives a bind-free path to the same fact — §6.4 |
| FTP `USER anonymous` | **Out.** RFC 959 files USER under access control and its replies under "Authentication and accounting"; RFC 1635 describes an account with a password — §6.5 |
| Unauthenticated Elasticsearch `GET /` | **In, and table-free — but it establishes less than it looks.** The fact is `401`-vs-`200`, which is RFC 9110, not Elasticsearch — §6.6 |
| How much table each rule needs | A **dispatch** table (~14 rows) that is aperture, and a **verdict** table of **zero** rows, provided the canonicaliser projects onto a closed value space — §7 |
| Is the dispatch table release-coupled | **Yes, and more strictly than the sensitive list.** A new protocol needs new prober code, so it *cannot* ship out of band — §7.3 |
| New `Facet`(s) | **One**, `listener-negotiation`, on `Service`, holding a closed two-field tuple. The HTTP-shaped members need **none** — `http-identity` already carries the status code, on `Endpoint` — §8 |
| One signal or several | **Two**, split on the fact measured and never on the protocol — §9 |
| Does it read `Exposure` | **Yes, both**, and that is precisely what separates them from `tls-1.0-negotiated`, which does not — §9.3 |
| Closest call | ~~**SMB signing not required** — a real, spec-defined, credential-free fact, excluded because a rule covering one protocol is a per-protocol signal by another name~~ — **restated by [#104](https://github.com/winniel123/verge-asm/issues/104): still the closest call, still excluded, and on neither of the two grounds this row gave.** Both are withdrawn; what excludes it is the **aperture** — reading `SMB2_NEGOTIATE_SIGNING_REQUIRED` needs the wire prober and `listener-negotiation`, which ADR-0015 put out of scope for this map — §9.2 |

Two headline results, and the second is the one that changes what v1 should build.

> **The good half of the idea survives and the headline half does not.** *Offered no TLS on a port
> whose protocol has a TLS variant* is not reachable without the port-to-protocol table, and that
> table is #21's. What survives table-free is narrower and better: **the listener's own
> unauthenticated self-report said it offers no encrypted mode** — read out of a field the protocol
> itself defines, in the server's own bytes. The port number is then only the address of the door,
> never the evidence.

> **Almost all the genuinely new coverage over the curated list is the HTTP case, and v1 already
> collects it.** #21 excluded 8080, 8000, 8888, 8443, 3000, 5000, 9000, 9090, 8088, 10000 and 6443
> on determinacy, and every one of them is HTTP. On HTTP the whole measurement is `401`-vs-`200` and
> plaintext-vs-TLS, both of which live inside the `http-identity` facet v1 already has. The
> wire-protocol prober — the expensive part, the new facet, the new safety surface — buys mail
> STARTTLS, LDAP StartTLS, AMQP mechanisms, RDP NLA, and TLS-absence on database ports **where
> `sensitive-port-exposed` already fires**. §10 does the count.

---

## 2. The two lines this note has to hold

### 2.1 The fingerprinting line: aperture tables and verdict tables

#5 rejected technology fingerprinting, and ADR-0004 sharpened the harm to *reference data mutating
underneath a comparison*, with the proxy test being whether the reference set is "closed and
enumerable" or "an open corpus that grows without bound". Every protocol-level probe needs *some*
reference data, so the question is not whether there is a table but **which job the table does**.

The canonical corpus splits along exactly this seam, and it is worth measuring rather than
asserting. `nmap-service-probes` is the service-detection database, and its two directive types are
described by nmap's own file-format documentation:

> "The Probe directive tells Nmap what string to send to recognize various services."
> — [nmap.org/book/vscan-fileformat.html](https://nmap.org/book/vscan-fileformat.html)

> "The match directive tells Nmap how to recognize services based on responses to the string sent by
> the previous Probe directive. A single Probe line may be followed by dozens or hundreds of match
> statements."
> — [same page](https://nmap.org/book/vscan-fileformat.html)

**Measured** against the current file
([raw.githubusercontent.com/nmap/nmap/master/nmap-service-probes](https://raw.githubusercontent.com/nmap/nmap/master/nmap-service-probes),
17,154 lines), counting directives at line start:

| Directive | Job | Count |
|---|---|---|
| `Probe` | what to send — **aperture** | **187** |
| `match` + `softmatch` | what the answer means — **verdict** | **12,171** |
| `fallback` | — | 12 |

```
grep -c '^Probe '     nmap-service-probes   ->    187
grep -c '^match '     nmap-service-probes   ->  11968
grep -c '^softmatch ' nmap-service-probes   ->    203
```

A ratio of about **65 to 1**. The aperture half is the size of a shipped port list. The verdict half
is 12,171 rows, it emits product names and CPEs —

> `match ftp m/^220.*Welcome to .*Pure-?FTPd (\d\S+\s*)/ p/Pure-FTPd/ v/$1/ cpe:/a:pureftpd:pure-ftpd:$1/`
> — [vscan-fileformat.html](https://nmap.org/book/vscan-fileformat.html), the documentation's own example

— and the file's header solicits growth in exactly that half:

> "This is a database of custom probes and expected responses that the Nmap Security Scanner uses to
> identify what services (eg http, smtp, dns, etc.) are listening on open ports. **Contributions to
> this database are welcome.**"
> — [nmap-service-probes](https://raw.githubusercontent.com/nmap/nmap/master/nmap-service-probes) header

That is ADR-0004's "open corpus that grows without bound", named by its maintainers as such. The
187-row half is not, and its most-used entry is literally the empty probe:

```
Probe TCP NULL q||
```

**The rule that follows, and it is the design constraint for everything below.** A candidate rule is
legal only if its reference data does the `Probe` job and never the `match` job:

- The table may decide **what to send and where**. It may not decide **what the answer means**.
- The verdict must be read from a field the protocol's own specification defines — an EHLO keyword,
  a capability bit, a resultCode, a security-type number, an HTTP status class.
- Nothing free-text in a response may enter a value. Banners, version strings and product names are
  the `match` half, and they are out even when they are sitting right there in bytes we already
  received.

The third bullet is not hypothetical. §4.2 shows the two protocols where the fingerprint arrives
*unavoidably*, in the same packet as the legal fact.

### 2.2 The credential line: the auth-message test

#4 settled that credentials are never submitted. The naive reading of that — *do not send a secret* —
does not survive contact with the protocols, because the three sharpest cases (FTP `USER anonymous`,
LDAP anonymous bind, MQTT `CONNECT` with no username) all send **no secret at all** and are all
plainly logins.

So the line is drawn one level up:

> **The prober may send any message the protocol defines, except the message that carries
> authentication. Whether that message's credential fields are populated is irrelevant — the message
> *is* the login.**

Three reasons this is the right line rather than a convenient one.

**It is checkable from the specification rather than from intent.** RFC 4511 §4.2 says the Bind
operation "should be thought of as the 'authenticate' operation". RFC 959 files USER under "ACCESS
CONTROL COMMANDS" and its reply codes under "Authentication and accounting". MQTT's CONNECT packet
is the packet with the User Name and Password fields in it. Nobody has to adjudicate.

**It is symmetric with what #4 already banned.** #4 refused `POST /login` with `admin:admin`. A
`POST /login` with the fields left empty is the same request to the same endpoint, hitting the same
audit log and the same lockout counter. An unattended tool doing that nightly is exactly the failure
#4 named: "repeated authentication against production".

**The alternative line degrades.** Once *empty credentials are not credentials* is accepted, the
next case is a username with an empty password (which RFC 4513 §5.1.2 defines as a distinct
mechanism and warns about), and the case after that is a documented default. There is no stable
resting place between "no authentication message" and "some authentication messages".

**What it costs, stated plainly.** It puts *anonymous access permitted* out of reach for every
protocol whose authorization state is established by a bind, a connect packet, or a session setup —
which is most of them. §6.7 lists the casualties. The one case where the cost looked severe (LDAP)
turns out to be refunded by the protocol itself (§6.4).

---

## 3. What a TCP connect plus a TLS handshake alone establishes

This is where the ticket hoped the answer was free, and it is the first deflating result.

### 3.1 The facts that are genuinely free today

#4 already performs, every run, a TCP connect and — where it can — one TLS handshake (§5: "one
handshake — cert (expiry, issuer, SAN, CN, serial, chain), negotiated version and cipher"). From
that alone, with no new probing whatsoever:

| Fact | Where it already lands |
|---|---|
| The `Service` accepted a TCP connection | `reachability` facet → `Exposure` |
| A TLS handshake completed | prerequisite for the whole certificate half of the v1 signal set |
| The negotiated TLS version and cipher | `tls-1.0/1.1-negotiated`, already in the v1 set |
| Certificate expiry / self-signed / SAN mismatch / weak key | already in the v1 set |

**Implicit TLS present or absent is therefore free.** It costs nothing new. The problem is what it
means.

### 3.2 Why "offered no TLS" is not free, decomposed

The ticket's candidate — *offered no TLS on a port whose protocol has a TLS variant* — is three
claims wearing one sentence:

1. **A TLS handshake to this `Service` did not complete.** An observation. No table.
2. **…and this `Service` speaks a protocol that has a standardised TLS mode.** A port-to-protocol
   table. This is #21's table, with #21's determinacy problem.
3. **…and that mode was not in use.** Requires (2) *plus* an application-level read, because a
   protocol with an in-band upgrade looks identical, at the TLS layer, to a protocol with no TLS at
   all.

Claim (1) on its own fires on: every plaintext-by-design listener in the estate; every service whose
TLS is terminated at an edge address; every port in the ~~~140-port~~ **131-port probed** hot set where
something answers a connect and nothing answers a ClientHello. On a real estate that is most of what
responds. **A
signal that fires on most of what responds is not a signal**, and no amount of care in phrasing
changes that.

> **`~140` was never `verge-core`'s size.** **[measured]** by
> [#97](https://github.com/winniel123/verge-asm/issues/97): the frequency half is **123, all TCP**, the
> union is **136 pairs**, and **131** are probed on default settings
> ([`sensitive-ports.md`](./sensitive-ports.md) §29, composed with
> [#95](https://github.com/winniel123/verge-asm/issues/95)). **This note's ruling does not move** — it
> turns on the signal firing on *most of what responds*, which is a proportion and not a count. Both
> occurrences of the figure in this note are marked; neither is load-bearing.

So: **a connect plus a handshake yields an input, not an insecure-listener signal.** That is the
headline the ticket hoped for, and it does not survive.

### 3.3 One thing the handshake does buy, free, and it is structural — measured

The *failure mode* of a failed TLS handshake distinguishes a server that spoke first from a server
that said nothing, with zero application bytes sent. Measured locally against two Node listeners on
loopback, one emitting a line-oriented greeting and one silent:

```
$ openssl s_client -connect 127.0.0.1:59001 -brief </dev/null     # greeting-first listener
Connecting to 127.0.0.1
error:0A00010B:SSL routines:tls_validate_record_header:wrong version number

$ openssl s_client -connect 127.0.0.1:59002 -brief </dev/null     # silent listener
Connecting to 127.0.0.1
error:0A000126:SSL routines::unexpected eof while reading

$ openssl s_client -connect www.rfc-editor.org:443 -brief </dev/null    # control
CONNECTION ESTABLISHED
Protocol version: TLSv1.3
```

(OpenSSL 3.5.7. Loopback listeners are my own; the control is an ordinary HTTPS client connection.)

`wrong version number` is the TLS record layer rejecting the greeting's first byte as a record
header. So the handshake #4 already performs partitions every responding `Service` three ways —
**implicit TLS**, **spoke first in something that is not TLS**, **said nothing** — before any
protocol decision is made. That partition is the aperture input for §4 and §5, it is free, and it is
table-free.

It is worth noticing that this is the same partition nmap encodes as `Probe TCP NULL q||` with a
6-second wait (§2.1). The distinction is old and structural; what is new here is that we get it as a
by-product rather than as a deliberate probing step.

### 3.4 The one place implicit TLS alone is a verdict

A `Service` on a port whose **only** standardised mode is implicit TLS — 993, 995, 465, 636, 5671,
8883, 2376 — that completes a TCP connect and refuses a ClientHello is a listener in a mode its
port does not have. RFC 8314 is explicit about what those ports mean:

> "The term 'Implicit TLS' refers to the automatic negotiation of TLS whenever a TCP connection is
> made on a particular TCP port that is used exclusively by that server for TLS connections."
> — [RFC 8314 §2](https://www.rfc-editor.org/rfc/rfc8314.txt)

> "When a TCP connection is established for the 'imaps' service (default port 993), a TLS handshake
> begins immediately."
> — [RFC 8314 §3.2](https://www.rfc-editor.org/rfc/rfc8314.txt) (and §3.1 for 995, §3.3 for 465)

This is real, and it still needs the table — it is claim (2) above with a shorter list. It is
recorded because it is the cheapest row in the whole dispatch table: no bytes sent beyond the
ClientHello #4 already sends.

---

## 4. Protocols where the server speaks first

Zero client bytes. The prober connects, reads, and closes. This is the smallest possible aperture and
the only class where nothing at all is sent.

### 4.1 The five, and what their first flight contains

| Protocol | Port | First server flight | Establishes | Source |
|---|---|---|---|---|
| SMTP | 25, 587 | `220 <text>` | listener is line-oriented and greeting-first; nothing about TLS | [RFC 5321 §3.1, §4.3.1](https://www.rfc-editor.org/rfc/rfc5321.txt) |
| FTP | 21 | `220 <text>` | same | [RFC 959 §4.2](https://www.rfc-editor.org/rfc/rfc959.txt) |
| POP3 | 110 | `+OK <text>` | same | [RFC 1939 §4](https://www.rfc-editor.org/rfc/rfc1939.txt) |
| IMAP | 143 | `* OK` / `* PREAUTH` / `* BYE`, optionally with a `[CAPABILITY …]` response code | **the state the connection starts in**, and possibly the full capability list including `STARTTLS` and `LOGINDISABLED` | [RFC 9051 §3, §7.1.1, §7.1.4, §7.2.2](https://www.rfc-editor.org/rfc/rfc9051.txt) |
| MySQL / MariaDB | 3306 | `Protocol::HandshakeV10` | **capability flags, including `CLIENT_SSL`** | §4.3 |

The three quotable server-speaks-first statements:

> "One important reply is the connection greeting. Normally, a receiver will send a 220 'Service
> ready' reply when the connection is completed. The sender SHOULD wait for this greeting message
> before sending any commands."
> — [RFC 5321 §4.3.1](https://www.rfc-editor.org/rfc/rfc5321.txt)

> "Once the TCP connection has been opened by a POP3 client, the POP3 server issues a one line
> greeting."
> — [RFC 1939 §4](https://www.rfc-editor.org/rfc/rfc1939.txt)

> "One important group of informational replies is the connection greetings. Under normal
> circumstances, a server will send a 220 reply, 'awaiting input', when the connection is completed."
> — [RFC 959 §4.2](https://www.rfc-editor.org/rfc/rfc959.txt)

**Note what the three line-oriented greetings do *not* establish.** SMTP, FTP and POP3 greetings say
nothing about TLS or authentication. The reply code is spec-defined; everything after it is free
text and is the `match` half of §2.1. RFC 5321 even documents the free text as a fingerprinting
affordance:

> "SMTP server implementations MAY include identification of their software and version information
> in the connection greeting reply after the 220 code, a practice that permits more efficient
> isolation and repair of any problems."
> — [RFC 5321 §3.1](https://www.rfc-editor.org/rfc/rfc5321.txt)

So for three of the five, the free flight buys dispatch and nothing else.

### 4.2 IMAP `* PREAUTH` — the only anonymous-access fact that is free at connect

> "The PREAUTH response is always untagged and is one of three possible greetings at connection
> startup. It indicates that the connection has already been authenticated by external means; thus,
> no LOGIN/AUTHENTICATE command is needed."
> — [RFC 9051 §7.1.4](https://www.rfc-editor.org/rfc/rfc9051.txt) (RFC 3501 §7.1.4 is materially identical)

Zero client bytes; the greeting itself states that the connection is in the authenticated state. It
is the single cleanest instance in this entire survey of the shape the ticket was looking for: the
listener's own unauthenticated self-report, answering the question, in a spec-defined token.

Two honest qualifications:

- The RFC says "authenticated by **external means**", not "with no credentials". For a prober that
  submitted none, the two coincide — but the note should not overclaim, and the value the
  canonicaliser records is *the greeting was PREAUTH*, not *anonymous access is permitted*.
- RFC 9051 makes PREAUTH on a cleartext port a documented anomaly in its own right: "the PREAUTH
  response SHOULD only be returned by servers on connections that are protected by TLS … Clients
  that require mandatory TLS MUST close the connection after receiving the PREAUTH response on a
  non-protected port." That is a standards-backed reading, in the same class as RFC 8996's TLS 1.0
  MUST NOT.

The IMAP greeting may also carry the whole capability list, making even the STARTTLS fact free:

> "CAPABILITY — Followed by a list of capabilities. This can appear in the initial OK or PREAUTH
> response to transmit an initial capabilities list. … This makes it unnecessary for a client to
> send a separate CAPABILITY command."
> — [RFC 9051 §7.1](https://www.rfc-editor.org/rfc/rfc9051.txt)

> `S: * OK [CAPABILITY STARTTLS AUTH=SCRAM-SHA-256 LOGINDISABLED IMAP4rev2] IMAP4rev2 Service Ready`
> — [RFC 9051 §8](https://www.rfc-editor.org/rfc/rfc9051.txt), the specification's own sample connection

It is a `MAY`, so the prober must be prepared to fall through to §5's `CAPABILITY` command. But where
a server does it, IMAP's entire transport-security and authentication posture is readable **without
the prober transmitting a single byte**.

### 4.3 MySQL — the whole TLS answer in the first packet, with the fingerprint attached

MySQL's server sends `Protocol::HandshakeV10` before the client says anything:

> "The initial handshake starts with the server sending the `Protocol::Handshake` packet. After this,
> optionally, the client can request an SSL connection to be established with the
> `Protocol::SSLRequest:` packet and then the client sends the `Protocol::HandshakeResponse:` packet."
> — [dev.mysql.com, Connection Phase](https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_connection_phase.html)

That packet carries the server's capability flags, and one of them is exactly the fact we want:

> `#define CLIENT_SSL   2048` — "Use SSL encryption for the session." … "Server — Supports SSL"
> — [dev.mysql.com, capability flags](https://dev.mysql.com/doc/dev/mysql-server/latest/group__group__cs__capabilities__flags.html)

MariaDB's independently written documentation of the same wire format agrees, naming the flag `SSL`
at `1 << 11` (= 2048) and listing the same field order
([mariadb.com/kb/en/connection](https://mariadb.com/kb/en/connection/)).

**So MySQL's transport-security posture is readable with zero client bytes.** It is the cheapest row
in the whole survey, and it is also the cleanest illustration of §2.1's third bullet, because the
same packet's second field is:

> `string<NUL>` — server version — "human readable status information"
> — [dev.mysql.com, Protocol::HandshakeV10](https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_connection_phase_packets_protocol_handshake_v10.html)

and MariaDB documents its exact content ("MariaDB Server 10.X versions are by default prefixed
`5.5.5-`"). That is a version fingerprint of precisely the kind #5 rejected, arriving unavoidably in
bytes we are entitled to read. **The canonicaliser must discard it**, and the reason is not only
policy: a facet value that included it would `Break` on every point release of the database (§7.4).

---

## 5. Protocols where one protocol-mandated hello suffices

For everything else the client speaks first. The question per protocol is then exactly the ticket's:
*what is the smallest message the specification defines that does not enter the authentication
exchange, and does its answer carry the fact?*

An independent check on the shape of this set, before the per-protocol working. OpenSSL ships
`s_client -starttls`, which drives a protocol's in-band TLS upgrade with no credentials. **Measured**
on OpenSSL 3.5.7 by passing an invalid value:

```
$ openssl s_client -starttls bogus -connect 127.0.0.1:59001
s_client: Value must be one of:
        smtp   pop3   imap   ftp   xmpp   xmpp-server   telnet
        irc    mysql  postgres   lmtp   nntp   sieve   ldap
```

Fourteen protocols. That is a shipped, maintained implementation's answer to *which protocols can a
client negotiate TLS on without logging in*, arrived at independently of this note. It includes
`mysql` and `postgres` — corroborating §4.3 and §5.2 from code rather than prose — and it **excludes**
MongoDB, Redis, MQTT, AMQP, RDP, RFB and SMB, which is also the finding below.

### 5.1 The per-protocol table

`prober sends` is what must be transmitted beyond the TCP connect. The last two columns are whether
the exchange establishes the transport-security and anonymous-access halves.

| Protocol / port | Speaks first | Prober sends | TLS fact | Anonymous-access fact | Verdict |
|---|---|---|---|---|---|
| SMTP 25/587 | server | `EHLO` | **yes** — `STARTTLS` keyword present or absent | partial — the `AUTH` mechanism list | **in** |
| IMAP 143 | server | nothing, or `CAPABILITY` | **yes** — `STARTTLS` / `LOGINDISABLED` | **yes** — a `PREAUTH` greeting | **in** |
| POP3 110 | server | `CAPA` | **yes** — the `STLS` capability | no | **in** |
| FTP 21 | server | `FEAT` | **yes** — `AUTH TLS` advertised | no | **in** |
| MySQL 3306 | **server** | **nothing** | **yes** — the `CLIENT_SSL` flag | no | **in** |
| PostgreSQL 5432 | client | 8 bytes (`SSLRequest`) | **yes, one-sided** — `N` proves absence, `S` proves nothing about requirement | no | **in** |
| LDAP 389 | client | StartTLS ExtendedRequest, or a bind-free root-DSE read | **yes** — the `resultCode`, or `supportedExtension` | **yes** — the bind-free read succeeding *is* the fact | **in** |
| AMQP 0-9-1 5672 | client | 8-byte protocol header | no — TLS is implicit on 5671 | **yes** — `mechanisms` contains `ANONYMOUS` | **in** |
| AMQP 1.0 5672 | client | 8-byte SASL header | no | **yes** — the `sasl-mechanisms` frame | **in** |
| Redis 6379 | client (server, in protected mode) | `PING`, or nothing | no — TLS is a separate `tls-port` | **yes** — a RESP error versus `+PONG` | **in**, §6.3 |
| RFB / VNC 5900 | server | 12-byte `ProtocolVersion` reply | no | **yes** — security type `1` (`None`) offered | **in** |
| RDP 3389 | client | X.224 Connection Request + negotiation request | **yes** — `selectedProtocol` or `failureCode` | partial — whether CredSSP is required | **in**, §5.6 |
| SMB 445 | client | SMB2 `NEGOTIATE` | partial — the encryption capability | no — needs `SESSION_SETUP` | **in** for negotiate facts only |
| MongoDB 27017 | client | `hello` | **partial** — `requireTLS` detectable; the other three modes are not distinguishable | **not established** — §5.8 | **partial** |
| HTTP, any port | client | `GET /` | **yes** — which scheme answered | **yes** — `401`/`403` versus `200` | **in**, and needs no new facet |
| MQTT 1883 | client | — | **no** | **no** | **out** — §6.7 |
| Docker 2375 | client | — | — | — | **moot** — §5.9 |

### 5.2 PostgreSQL — the smallest exchange, and a one-sided answer

> "To initiate an SSL-encrypted connection, the frontend initially sends an SSLRequest message rather
> than a StartupMessage. The server then responds with a single byte containing `S` or `N`,
> indicating that it is willing or unwilling to perform SSL, respectively."
> — [PostgreSQL §54.2.10, SSL Session Encryption](https://www.postgresql.org/docs/current/protocol-flow.html)

The message is eight bytes and contains no identity of any kind:

> `Int32(8)` — "Length of message contents in bytes, including self." · `Int32(80877103)` — "The SSL
> request code."
> — [PostgreSQL §54.7, SSLRequest (F)](https://www.postgresql.org/docs/current/protocol-message-formats.html)

No username, no database name, no startup packet, and the answer is one byte. This is the smallest
credential-free TLS interrogation in the survey, and it comes with the sharpest limitation in the
survey, stated by PostgreSQL itself:

> "While the protocol itself does not provide a way for the server to force SSL encryption, the
> administrator can configure the server to reject unencrypted sessions **as a byproduct of
> authentication checking**."
> — [PostgreSQL §54.2.10](https://www.postgresql.org/docs/current/protocol-flow.html)

**So `N` establishes that TLS is unavailable, and `S` establishes nothing about whether it is
required** — because the requirement is enforced at authentication time, which is exactly where §6's
line stops us. The facet value must therefore be able to say *encryption offered* without ever
implying *encryption required*, and §7.4's value space is built so the distinction is expressible
rather than collapsed.

The `StartupMessage` is out of reach for a blunt reason:

> "`user` — The database user name to connect as. **Required; there is no default.**"
> — [PostgreSQL §54.7, StartupMessage (F)](https://www.postgresql.org/docs/current/protocol-message-formats.html)

A prober sending one must invent an account name, and the server's next message is an
`Authentication*` message. That is the authentication exchange, entered (§6.1).

### 5.3 The mail and file-transfer trio — and the one standards-backed verdict in the note

All four advertise their TLS upgrade in the answer to a command outside the authentication exchange.

**SMTP.** RFC 3207 defines both the advertisement and its meaning:

> "the EHLO keyword value associated with the extension is STARTTLS"
> — [RFC 3207 §2](https://www.rfc-editor.org/rfc/rfc3207.txt)

> "The STARTTLS keyword is used to tell the SMTP client that the SMTP server is currently able to
> negotiate the use of TLS. It takes no parameters."
> — [RFC 3207 §3](https://www.rfc-editor.org/rfc/rfc3207.txt)

The same EHLO response carries the AUTH mechanism list — "The AUTH EHLO keyword contains as a
parameter a space-separated list of the names of available [SASL] mechanisms"
([RFC 4954 §3](https://www.rfc-editor.org/rfc/rfc4954.txt)) — and RFC 4954 makes advertising a
plaintext mechanism on an unprotected channel a documented error:

> "Modern implementations SHOULD NOT advertise mechanisms that are not permitted due to lack of
> encryption, unless an encryption layer of sufficient strength is currently being employed."
> — [RFC 4954 §6](https://www.rfc-editor.org/rfc/rfc4954.txt)

**Recorded as a limitation, because it bounds the SMTP row.** RFC 3207 contains **no** rule about
what a server's *omission* of STARTTLS means. Its only treatment of a missing advertisement is §7,
where deleting it is described as a man-in-the-middle attack. So *STARTTLS absent* is a legal
observation and not a spec violation.

**POP3.** `CAPA` is explicitly valid before login, and `STLS` is the capability:

> "The POP3 CAPA command returns a list of capabilities supported by the POP3 server. It is available
> in both the AUTHORIZATION and TRANSACTION states."
> — [RFC 2449 §5](https://www.rfc-editor.org/rfc/rfc2449.txt)

> "The capability name 'STLS' indicates this command is present and permitted in the current state."
> — [RFC 2595 §4](https://www.rfc-editor.org/rfc/rfc2595.txt)

**IMAP is the one row carrying a normative verdict rather than a bare observation.** RFC 9051 makes
LOGINDISABLED a conditional MUST:

> "Unless the client is accessing IMAP service on an Implicit TLS port [RFC8314], the STARTTLS
> command has been negotiated, or some other mechanism that protects the session from password
> snooping has been provided, a server implementation MUST implement a configuration in which it
> advertises the LOGINDISABLED capability and does NOT permit the LOGIN command."
> — [RFC 9051 §6.2.3](https://www.rfc-editor.org/rfc/rfc9051.txt) (RFC 3501 §6.2.3 is materially identical)

> "Client and server implementations MUST implement the STARTTLS (Section 6.2.1) and LOGINDISABLED
> capabilities on cleartext ports."
> — [RFC 9051 §5](https://www.rfc-editor.org/rfc/rfc9051.txt)

So an IMAP listener on a cleartext port whose capability list contains `AUTH=PLAIN` and lacks
`LOGINDISABLED` is in violation of a MUST — the same footing RFC 8996 gives `tls-1.0-negotiated`,
which is the strongest footing any signal in the v1 set has. **This is the best-founded row in the
note**, and it is worth noticing that it is founded on a *capability list*, not on a port number.

**FTP.** RFC 4217 requires the advertisement:

> "If a server supports the FEAT command, then it MUST advertise supported AUTH, PBSZ, and PROT
> commands in the reply … Additionally, the AUTH command should have a reply that identifies 'TLS' as
> one of the possible parameters to AUTH."
> — [RFC 4217 §6](https://www.rfc-editor.org/rfc/rfc4217.txt)

and its own canonical session diagram puts `AUTH TLS` immediately after the 220 and before `USER`
([RFC 4217 §12.1](https://www.rfc-editor.org/rfc/rfc4217.txt)).

**Recorded as a gap, because it is load-bearing and it is an absence.** RFC 2389 says **nothing**
about whether `FEAT` may be issued before login — the strings "login", "logged" and "authenticat" do
not occur anywhere in the document, and its reply-code list omits 530. So *`FEAT` is a pre-login
command* is an inference from the absence of a restriction plus RFC 4217's diagram, not a quotable
rule. It is the weakest footing of the four and should be labelled in the implementation rather than
smoothed over.

### 5.4 LDAP — where the hard line turns out to cost nothing

StartTLS is an Extended operation carrying no identity at all:

> "A client requests TLS establishment by transmitting a StartTLS request message to the server. The
> StartTLS request is defined in terms of an ExtendedRequest. The requestName is
> '1.3.6.1.4.1.1466.20037', and the requestValue field is always absent."
> — [RFC 4511 §4.14.1](https://www.rfc-editor.org/rfc/rfc4511.txt)

and its answer is directly the fact:

> "If the server does not support TLS (whether by design or by current configuration), it returns
> with the resultCode set to protocolError"
> — [RFC 4511 §4.14.1](https://www.rfc-editor.org/rfc/rfc4511.txt)

It may precede a Bind, explicitly:

> "There is no general requirement that the client have or have not already performed a Bind
> operation (Section 5) before sending a StartTLS operation request; however, where a client intends
> to perform both a Bind operation and a StartTLS operation, it SHOULD first perform the StartTLS
> operation"
> — [RFC 4513 §3.1.1](https://www.rfc-editor.org/rfc/rfc4513.txt)

The alternative route is the root DSE's `supportedExtension` attribute, which lists the OIDs of
extended operations the server recognises ([RFC 4511 §4.12](https://www.rfc-editor.org/rfc/rfc4511.txt),
[RFC 4512 §5.1.4](https://www.rfc-editor.org/rfc/rfc4512.txt)). §6.4 shows why that second route
matters far more than it looks.

**Recorded as a non-finding.** RFC 4512 says nothing whatever about anonymous access to the root
DSE — the word "anonymous" does not occur in the document — and qualifies root-DSE reads only as
"subject to access control and other restrictions". The nearest normative sentence in the LDAP suite
covers one attribute: "LDAP servers SHOULD allow all clients — even those with an anonymous
authorization — to retrieve the 'supportedSASLMechanisms' attribute of the root DSE"
([RFC 4513 §5.2.1.5](https://www.rfc-editor.org/rfc/rfc4513.txt)).

### 5.5 AMQP — the strongest anonymous-access measurement in the survey

AMQP 0-9-1's entire pre-authentication exchange is one 8-octet constant:

> "The client MUST start a new connection by sending a protocol header. This is an 8-octet sequence:
> `'A','M','Q','P',0,0,9,1`"
> — [AMQP 0-9-1 specification §4.2.2](https://www.rabbitmq.com/resources/specs/amqp0-9-1.pdf)

> "The client opens a TCP/IP connection to the server and sends a protocol header. **This is the only
> data the client sends that is not formatted as a method.** The server responds with its protocol
> version and other properties, **including a list of the security mechanisms that it supports** (the
> Start method). The client selects a security mechanism (Start-Ok). The server starts the
> authentication process, which uses the SASL challenge-response model."
> — [AMQP 0-9-1 specification §2.2.4](https://www.rabbitmq.com/resources/specs/amqp0-9-1.pdf)

Note the sequencing in the vendor's own words: `Connection.Start` arrives, with the mechanism list,
**before** the authentication process begins. The field is mandatory —

> "`mechanisms` — A list of the security mechanisms that the server supports, delimited by spaces."
> — [amqp0-9-1.xml](https://www.rabbitmq.com/resources/specs/amqp0-9-1.xml), `connection.start`, carrying `<assert check="notnull"/>`

AMQP 1.0 makes the semantics of that list explicit rather than conventional:

> "A list of the sasl security mechanisms supported by the sending peer. It is invalid for this list
> to be null or empty. **If the sending peer does not require its partner to authenticate with it,
> then it SHOULD send a list of one element with its value as the SASL mechanism ANONYMOUS.**"
> — [OASIS AMQP 1.0 §5.3.3.1](http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-security-v1.0-os.html)

and RFC 4505 makes the announcement itself the fact:

> "the purpose of this SASL mechanism is to allow the user to gain access to services or resources
> without requiring the user to establish or otherwise disclose their identity to the server. That
> is, this mechanism provides an anonymous login method." … "**A server that permits anonymous access
> will announce support for the ANONYMOUS mechanism** and allow anyone to log in using that
> mechanism, usually with restricted access."
> — [RFC 4505 §1, §2](https://www.rfc-editor.org/rfc/rfc4505.txt)

This matters because the dominant implementation ships it enabled:

> "ANONYMOUS — This mechanism is enabled by default allowing anonymous clients to connect without
> providing any credentials. … In other words, any unauthenticated client will be able to connect and
> act as the configured `anonymous_login_user`. **For production environments, remove this
> mechanism.**"
> — [RabbitMQ, Authentication Mechanisms](https://www.rabbitmq.com/docs/access-control)

with `auth_mechanisms` documented as defaulting to `PLAIN`, `AMQPLAIN`, `ANONYMOUS` (same page).

**So for AMQP the anonymous-access question is answered by the server, unprompted, after eight
constant bytes, with the specification stating what the answer means.** It is the ticket's hoped-for
shape in its purest form — and it lands on 5672, which #21 §4.6 excluded from the sensitive list.
This is the clearest single case of the observation-driven rule reaching where the curated list
cannot.

### 5.6 RFB / VNC — "no authentication" as a wire constant

The server speaks first with a 12-byte version string, the client echoes one, and the server then
lists what it will accept:

> "Handshaking begins by the server sending the client a ProtocolVersion message. … The
> ProtocolVersion message consists of 12 bytes interpreted as a string of ASCII characters in the
> format `RFB xxx.yyy\n`"
> — [RFC 6143 §7.1.1](https://www.rfc-editor.org/rfc/rfc6143.txt)

> "Once the protocol version has been decided, the server and client must agree on the type of
> security to be used on the connection. **The server lists the security types that it supports**"
> — [RFC 6143 §7.1.2](https://www.rfc-editor.org/rfc/rfc6143.txt)

> "**7.2.1. None** — No authentication is needed. The protocol continues with the SecurityResult
> message."
> — [RFC 6143 §7.2.1](https://www.rfc-editor.org/rfc/rfc6143.txt)

Security type `1` appearing in that list *is* "anonymous access permitted", as a number defined by
the specification. The prober sends 12 bytes — its own ProtocolVersion — which is not the
authentication message; RFB's authentication message is the 16-byte challenge response of §7.2.2,
and the prober never reaches it.

RFC 6143 also supplies the reason 5900 is on #21's list at all —

> "This type of authentication is known to be cryptographically weak and is not intended for use on
> untrusted networks."
> — [RFC 6143 §7.2.2](https://www.rfc-editor.org/rfc/rfc6143.txt)

— so on 5900 the measurement adds a distinction (`None` versus VNC Authentication) to a port where
`sensitive-port-exposed` already fires. That is the redundancy pattern §10 quantifies.

### 5.7 RDP — the negotiation is in the clear, and Microsoft says so

RDP's first exchange advertises and selects a security protocol before anything else happens:

> "The RDP Negotiation Request structure is used by a client to advertise the security protocols
> which it supports." … "`requestedProtocols` (4 bytes): A 32-bit, unsigned integer that contains
> flags indicating the supported security protocols."
> — [MS-RDPBCGR §2.2.1.1.1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/902b090b-9cb3-4efc-92bf-ee13373371e3)

with `PROTOCOL_RDP` = `0x00000000` ("Standard RDP Security"), `PROTOCOL_SSL` = `0x00000001`
("TLS 1.0, 1.1, or 1.2") and `PROTOCOL_HYBRID` = `0x00000002` ("Credential Security Support Provider
protocol (CredSSP)") — CredSSP being what Microsoft markets as Network Level Authentication. The
server answers with its choice, or with a refusal that names the reason:

> "`SSL_REQUIRED_BY_SERVER` — 0x00000001 — The server requires that the client support Enhanced RDP
> Security (section 5.4) with either TLS 1.0, 1.1 or 1.2 … or CredSSP" · "`HYBRID_REQUIRED_BY_SERVER`
> — 0x00000005 — The server requires that the client support Enhanced RDP Security (section 5.4) with
> CredSSP" · "`SSL_NOT_ALLOWED_BY_SERVER` — 0x00000002 — The server is configured to only use Standard
> RDP Security mechanisms (section 5.3) and does not support any External Security Protocols"
> — [MS-RDPBCGR §2.2.1.2.2](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/1b3920e7-0116-4345-bc45-f2c4ad012761)

And the specification states outright that this happens unprotected:

> "Because both the RDP Negotiation Request and RDP Negotiation Response are **initially exchanged in
> the clear**, they are re-exchanged in the reverse direction after the External Security Protocol
> handshake … This step ensures that no tampering has taken place."
> — [MS-RDPBCGR §5.4.2.1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/db98be23-733a-4fd2-b086-002cd2ba02e5)

That is as close as the document comes; **it never uses the phrase "before authentication"**, and the
inference is from "in the clear" plus the ordering. Microsoft's product documentation supplies the
verdict half:

> "Network Level Authentication (NLA) adds an extra layer of security to Remote Desktop connections.
> With NLA enabled, users must authenticate themselves before a remote session is established,
> reducing the risk of unauthorized access … Enabling NLA is recommended for most environments."
> — [learn.microsoft.com, Allow access to your PC from outside your network](https://learn.microsoft.com/en-us/windows-server/remote/remote-desktop-services/remotepc/remote-desktop-allow-access)

**RDP is therefore the one genuinely new, high-value row.** #21 excluded 3389 deliberately — remote
administration over an untrusted network is what the protocol is *for*, so no claim on the port could
be made — and this measurement asks a different question the port never could: *does this particular
listener require CredSSP?* No credential is submitted; the answer is a spec-defined integer.

Two honest deflations. The measurement is *NLA required or not*, which is not the same as *anonymous
access permitted* — an RDP server without NLA still presents a login screen. And "recommended for
most environments" is a hardening preference, not a MUST, so this row is `sensitive-port-exposed`'s
§4.4 problem in miniature: a strong fact whose interpretation is contested.

### 5.8 SMB and MongoDB — partial rows, recorded as partial

**SMB.** `NEGOTIATE` is the first message and the authentication is somewhere else:

> "The SMB2 SESSION_SETUP Request packet is sent by the client to request a new **authenticated**
> session"
> — [MS-SMB2 §2.2.5](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/5a3c2c28-d6b0-48ed-b917-a86b2ca4575f)

so the NEGOTIATE response's fields are readable without entering it:

> "`SecurityMode` (2 bytes): The security mode field specifies whether SMB signing is enabled,
> required at the server, or both." — `SMB2_NEGOTIATE_SIGNING_ENABLED` 0x0001, "security signatures
> are enabled on the server"; `SMB2_NEGOTIATE_SIGNING_REQUIRED` 0x0002, "security signatures are
> required by the server". And `SMB2_GLOBAL_CAP_ENCRYPTION` 0x00000040, "the server supports
> encryption".
> — [MS-SMB2 §2.2.4](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/63abf97c-0d09-47e2-88d6-6bfa552949a5)

Real, credential-free, spec-defined facts. But **signing is integrity, not confidentiality, and
guest/null-session access needs `SESSION_SETUP`**, so SMB contributes to neither of the two rules
cleanly. §9.2 records it as the closest call in the note.

**MongoDB.** TLS is negotiated on the same port rather than a sibling — `net.tls.mode` takes
`disabled`, `allowTLS`, `preferTLS`, `requireTLS`, where the middle two mean "the server accepts both
TLS and non-TLS"
([mongodb.com, configuration options](https://www.mongodb.com/docs/manual/reference/configuration-options/)).
So the free handshake distinguishes `requireTLS` from everything else and cannot separate the other
three, which is a weaker answer than any other row.

On anonymous access MongoDB is a **documented non-finding**, and it is recorded as such rather than
filled in by inference. The `hello` reference page says nothing about authentication; MongoDB's own
driver specification establishes only that the handshake precedes it ("It MUST be the first command
sent over the respective socket", with `speculativeAuthenticate` as an optional rider —
[mongodb/specifications, handshake.md](https://raw.githubusercontent.com/mongodb/specifications/master/source/mongodb-handshake/handshake.md));
and no MongoDB page quotes what an unauthenticated client receives. Under #21's evidence standard
that is not enough to build a rule on. **MongoDB is excluded for want of an attested answer**, in the
same way and for the same reason #21 excluded 111/tcp.

### 5.9 Docker and Kubernetes — one row is moot and the other is HTTP

**Docker 2375 is moot.** Docker no longer permits the configuration:

> "Exposing the daemon API over HTTP without TLS is not permitted, and such a configuration causes the
> daemon to fail early on startup"
> — [docs.docker.com, Docker daemon attack surface](https://docs.docker.com/engine/security/)

> "In version 27.0 and later, specifying `--tls=false` or `--tlsverify=false` CLI flags causes the
> daemon to fail to start if it's also configured to accept remote connections over TCP."
> — [docs.docker.com, deprecated features](https://docs.docker.com/engine/deprecated/) (deprecated v26.0, removal targeted v28.0)

A plaintext Docker API on a current daemon cannot exist, so there is nothing for an observation-driven
rule to measure that `sensitive-port-exposed` does not already cover on the port. Recorded because it
was on the ticket's list.

**Kubernetes 6443 is HTTP and the interesting part is not measurable.** Anonymous access is on by
default —

> "`--anonymous-auth`  Default: true — Enables anonymous requests to the secure port of the API
> server. … Anonymous requests have a username of `system:anonymous`, and a group name of
> `system:unauthenticated`."
> — [kubernetes.io, kube-apiserver reference](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/)

— and it is *supposed* to be:

> "Default cluster role bindings authorize unauthenticated and authenticated users to read API
> information that is deemed safe to be publicly accessible"
> — [kubernetes.io, RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)

while discovery is not: `system:discovery` is bound to `system:authenticated` only, and the docs note
"Prior to v1.14, this role was also bound to `system:unauthenticated` by default" (same page).

So the meaningful measurement would be *which* endpoints answer anonymously — and that needs a list
of Kubernetes-specific paths and an expectation of what each should return. That is a **verdict
table** (§2.1), and it is out. **Recorded as a non-finding:** kubernetes.io does not anywhere
enumerate the non-resource URLs `system:public-info-viewer` grants, so even building that table
correctly would mean reading the API server's source rather than its documentation — which is a good
independent signal that we should not be building it.

What is left on 6443 is the table-free HTTP fact of §6.6, and that is genuinely useful: #21 §4.4 made
6443 its closest call and had to exclude it, and *an unauthenticated request to this listener's root
was answered rather than refused* is precisely the measurement #21 said the port could not supply.

---

## 6. Where the hard line sits

### 6.1 The test, stated precisely

§2.2 gave the shape. The precise form, refined by the cases below:

> **The prober may send any message the protocol defines, except a message that enters the protocol's
> authentication exchange** — the message after which the server's next move is an authentication
> challenge, an authentication result, or the establishment of an authorization state on our behalf.
> Whether that message's credential fields are populated is irrelevant.

The operational test is a question about the *server's* next move, not about our payload, and it is
answerable from the specification in every case in this note:

| Message | Server's next move | Verdict |
|---|---|---|
| PostgreSQL `SSLRequest` | one byte, `S` or `N` | not the exchange — **in** |
| PostgreSQL `StartupMessage` | an `Authentication*` message | **out** |
| SMTP `EHLO` | the extension list | **in** |
| SMTP `AUTH` | a SASL challenge | **out** |
| IMAP `CAPABILITY` | the capability list | **in** |
| IMAP `LOGIN` / `AUTHENTICATE` | a tagged result or a challenge | **out** |
| FTP `FEAT` | the feature list | **in** |
| FTP `USER` | `331`/`230`/`530` — an authentication reply | **out** |
| LDAP StartTLS ExtendedRequest | an ExtendedResponse `resultCode` | **in** |
| LDAP Bind (any form) | a `BindResponse` | **out** |
| AMQP protocol header | `Connection.Start` | **in** |
| AMQP `Connection.Start-Ok` | `Connection.Secure` (a SASL challenge) | **out** |
| MQTT `CONNECT` | `CONNACK` with an authorization result | **out** |
| SMB2 `NEGOTIATE` | the negotiate response | **in** |
| SMB2 `SESSION_SETUP` | "a new **authenticated** session" | **out** |
| Redis `PING` | `+PONG` or a RESP error | **in** |
| Redis `AUTH` | `OK` or an error | **out** |
| HTTP `GET /` with no `Authorization` | a representation or a challenge | **in** |
| HTTP `POST /login` | an authentication result | **out** |

The four edge cases the ticket named are argued individually below, because each of them is where an
implementer would be tempted to slide.

### 6.2 Why not the softer line

The obvious alternative — *do not transmit a secret* — is what an implementer reaches for, and all
three of the hardest cases pass it while being unambiguously logins. FTP `USER anonymous` transmits
a documented public username. An LDAP anonymous simple bind transmits two zero-length strings. An
MQTT `CONNECT` with both flag bits clear transmits neither field. None of the three carries a secret;
all three are the protocol's login.

The concrete harm is the one #4 named. Unattended and nightly, these produce successful or failed
authentication events in the operator's own audit log, they increment whatever lockout counter the
service keeps, and — where they succeed — they leave the tool holding an authorized session it has
no plan for. #4 refused `POST /login` with `admin:admin` for exactly these reasons, and none of them
turn on whether the password field was populated.

There is also no stable resting place on the softer line. Once *empty credentials are not
credentials* is accepted, the next case is a non-empty name with an empty password — which RFC 4513
§5.1.2 defines as its own distinct mechanism and warns about — and the case after that is a
documented default username. The line has to sit at the exchange or it does not sit anywhere.

### 6.3 Redis `PING` before `AUTH` — in

**Verdict: in, and it is not a close call.** Redis has no connect-time handshake and no login packet.
`AUTH` is one command among hundreds; `PING` is another; sending `PING` does not enter any exchange
that `AUTH` participates in, and the server's answer is not an authentication result but an ordinary
command reply.

The behaviour is documented on both sides:

> "When the `requirepass` setting is enabled, Redis will refuse any query by unauthenticated clients."
> — [redis.io, Security](https://redis.io/docs/latest/operate/oss_and_stack/management/security/)

> "In this configuration Redis will deny any command executed by the just connected clients, unless
> the connection gets authenticated via AUTH."
> — [redis.io, AUTH](https://redis.io/docs/latest/commands/auth/)

> "This command returns the bulk string PONG if no argument is provided" … "Simple string reply: PONG
> when no argument is provided."
> — [redis.io, PING](https://redis.io/docs/latest/commands/ping/)

**But there is a fingerprinting trap here that must be named, because it is easy to fall into.** The
error string every practitioner knows — `-NOAUTH Authentication required.` — is **not documented
anywhere on redis.io**. Six pages were checked (AUTH, PING, Security, Encryption, ACL, RESP protocol
spec) and the token `NOAUTH` appears in none of them. A rule matching that literal would be matching
an undocumented implementation string, which is a `match` line in the sense of §2.1 and is out.

What *is* legal is the RESP type discrimination, which the protocol specification defines: a reply
beginning `-` is an error and a reply beginning `+` is a simple string
([redis.io, RESP protocol spec](https://redis.io/docs/latest/develop/reference/protocol-spec/)). So
the observation is **"`PING` was answered with a RESP error" versus "`PING` was answered with
`+PONG`"** — a distinction in the wire format, not in a string table. That is the whole of what may
be recorded, and it is sufficient.

**One free bonus, and it is the safest measurement of the lot.** Redis's protected mode makes the
server speak first, which the protocol specification documents as an explicit exception to
request-response:

> "Connections opened from a non-loopback address to a Redis while in protected mode are denied and
> terminated by the server. Before terminating the connection, Redis **unconditionally sends a
> `-DENIED` reply, regardless of whether the client writes to the socket**."
> — [redis.io, RESP protocol spec](https://redis.io/docs/latest/develop/reference/protocol-spec/)

So *protected mode is active* is measurable with **zero client bytes**, and unlike `-NOAUTH` the
`-DENIED` token is documented by the project. A `Service` on 6379 that sends bytes before we do is
telling us it is protecting itself.

### 6.4 An anonymous LDAP bind — out, and the exclusion is refunded

**Verdict: out.** The specification supplies the argument in its own words:

> "The function of the Bind operation is to allow authentication information to be exchanged between
> the client and server. **The Bind operation should be thought of as the 'authenticate' operation.**"
> — [RFC 4511 §4.2](https://www.rfc-editor.org/rfc/rfc4511.txt)

and RFC 4513 files the anonymous form under authentication mechanisms without qualification:

> "The simple authentication method of the Bind Operation provides three authentication mechanisms:
> An anonymous authentication mechanism (Section 5.1.1). An unauthenticated authentication mechanism
> (Section 5.1.2). A name/password authentication mechanism…"
> — [RFC 4513 §5.1](https://www.rfc-editor.org/rfc/rfc4513.txt)

> "An LDAP client may use the **anonymous authentication mechanism of the simple Bind method** to
> explicitly establish an anonymous authorization state by sending a Bind request with a name value
> of zero length and specifying the simple authentication choice containing a password value of zero
> length."
> — [RFC 4513 §5.1.1](https://www.rfc-editor.org/rfc/rfc4513.txt)

An empty credential inside the authenticate operation is still the authenticate operation. Out.

**And it costs nothing, which is the finding worth carrying.** LDAP gives the same fact by a route
that involves no Bind at all:

> "Upon initial establishment of the LDAP session, the session has an anonymous authorization
> identity. Among other things this implies that **the client need not send a BindRequest in the
> first PDU of the LDAP message layer. The client may send any operation request prior to performing
> a Bind operation**, and the server MUST treat it as if it had been performed after an anonymous
> Bind operation (Section 5.1.1)."
> — [RFC 4513 §4](https://www.rfc-editor.org/rfc/rfc4513.txt)

So the prober issues a Search for the root DSE requesting `supportedExtension` by name, sends no Bind,
and reads two facts at once: whether an unauthenticated read is permitted (the search succeeded or
returned `insufficientAccessRights`), and whether StartTLS is supported (the OID is present or
absent). **The hard line and the measurement do not conflict here — the protocol has already
separated them**, and an implementation that reached for the anonymous bind would be choosing the
forbidden route over an equivalent permitted one.

This is the single most useful structural result in §6, because it generalises: where a protocol
distinguishes *being unauthenticated* from *authenticating as nobody*, the first is always available
and the second is never needed.

### 6.5 FTP `USER anonymous` — out

**Verdict: out**, on three independent grounds, and it is not rescued the way LDAP was.

RFC 959 places USER under access control and describes it as opening the login sequence:

> "**USER NAME (USER)** — The argument field is a Telnet string identifying the user. The user
> identification is that which is required by the server for access to its file system. … Servers may
> allow a new USER command to be entered at any point in order to change the access control and/or
> accounting information. This has the effect of flushing any user, password, and account information
> already supplied and **beginning the login sequence again**."
> — [RFC 959 §4.1.1](https://www.rfc-editor.org/rfc/rfc959.txt)

Its replies are classified as authentication replies:

> "`x3z` Authentication and accounting — Replies for the login process and accounting procedures."
> — [RFC 959 §4.2.1](https://www.rfc-editor.org/rfc/rfc959.txt)

and §5.4's command-reply table lists USER under the heading `Login`, with `230 User logged in`,
`530 Not logged in` and `331 User name okay, need password` among its replies.

And anonymous FTP is an account with a password, not an absence of authentication:

> "These sites create a special account called 'anonymous'. … Traditionally, this special anonymous
> user account accepts any string as a password, although it is common to use either the password
> 'guest' or one's electronic mail (e-mail) address."
> — [RFC 1635](https://www.rfc-editor.org/rfc/rfc1635.txt)

RFC 2577 characterises it the same way in the one sentence it devotes to the subject: "Anonymous FTP
refers to the ability of a client to connect to an FTP server **with minimal authentication**"
([RFC 2577 §9](https://www.rfc-editor.org/rfc/rfc2577.txt)) — reduced authentication, not none.

**What FTP loses.** Unlike LDAP there is no bind-free path: RFC 959 makes USER the gateway to
everything, so *anonymous FTP is enabled on this listener* is simply not measurable inside the
profile. The `FEAT`-based TLS fact (§5.3) survives; the anonymous-access fact does not. This is the
clearest case of the line costing us something real.

### 6.6 An unauthenticated Elasticsearch `GET /` — in, and it establishes less than it looks

**Verdict: in.** No `Authorization` header is sent, so no credential is submitted, and the server's
answer is either a representation or a challenge — never an authentication result. GET is defined as
safe:

> "Request methods are considered 'safe' if their defined semantics are essentially read-only; i.e.,
> the client does not request, and does not expect, any state change on the origin server as a result
> of applying a safe method to a target resource." … "Of the request methods defined by this
> specification, the GET, HEAD, OPTIONS, and TRACE methods are defined to be safe."
> — [RFC 9110 §9.2.1](https://www.rfc-editor.org/rfc/rfc9110.txt)

*(A small correction while we are here: #4 §4.1 and §10 both cite **§9.3.1** for the safety of GET.
§9.3.1 defines the GET method and never uses the word "safe"; safety is §9.2.1. The claim is right
and the section number is wrong in both places.)*

**But the fact established is HTTP's, not Elasticsearch's**, and the distinction is the whole point
of §2.1:

> "The 401 (Unauthorized) status code indicates that the request has not been applied because it
> lacks valid authentication credentials for the target resource."
> — [RFC 9110 §15.5.2](https://www.rfc-editor.org/rfc/rfc9110.txt)

> "The 403 (Forbidden) status code indicates that the server understood the request but refuses to
> fulfill it."
> — [RFC 9110 §15.5.4](https://www.rfc-editor.org/rfc/rfc9110.txt)

*An unauthenticated GET of this endpoint's root was answered with a representation rather than
refused* is a fact about the estate, computed from a status class the standard defines, needing no
per-product knowledge whatsoever. It applies to 9200, 6443, 8080, 8443, 5601, 9090, 15672, 2375 and
every other HTTP-shaped listener identically.

**What must not be added on top.** *Therefore this Elasticsearch node is unprotected* requires
knowing the listener is Elasticsearch (determinacy, which #21 excluded 9200-class ports from at the
`(port, transport)` level) and knowing that a 200 on `/` means API access rather than a login page
(a verdict table). Both are out. And the vendor evidence does not support the stronger claim anyway:
Elastic documents `GET /` as *requiring* the `monitor` cluster privilege
([elastic.co, Get cluster info](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-info)),
and states only that "Unless you enable anonymous access (not recommended), all requests that don't
include credentials are rejected"
([elastic.co, minimal security](https://www.elastic.co/docs/deploy-manage/security/set-up-minimal-security))
without ever documenting what an unsecured cluster returns to an anonymous caller. **Recorded as a
non-finding:** there is no Elastic sentence to cite for the behaviour everyone knows.

**One measured wrinkle in the table-free rule**, because it bounds how strong the HTTP row can be.
RFC 9110 §15.5.2 says the server "MUST send a WWW-Authenticate header field". Measured:

```
$ curl -sI https://api.github.com/user       -> HTTP/1.1 401 Unauthorized, no WWW-Authenticate
$ curl -sI https://httpbin.org/basic-auth/user/passwd
                                             -> HTTP/1.1 401 UNAUTHORIZED
                                                WWW-Authenticate: Basic realm="Fake Realm"
```

A widely-used production API returns a 401 without the mandatory header. So the rule must key on the
**status class** and treat `WWW-Authenticate` as corroboration rather than a requirement — otherwise
it will report a well-protected endpoint as unprotected because its 401 was non-conformant.

### 6.7 MQTT `CONNECT` — out, and the protocol leaves nothing behind

**Verdict: out, totally.** MQTT's first packet is mandatory and it is the login:

> "After a Network Connection is established by a Client to a Server, **the first Packet sent from
> the Client to the Server MUST be a CONNECT Packet** [MQTT-3.1.0-1]."
> — [OASIS MQTT 3.1.1 §3.1](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)

and that packet is where the credentials live:

> "**3.1.2.8 User Name Flag** — Position: bit 7 of the Connect Flags. If the User Name Flag is set to
> 0, a user name MUST NOT be present in the payload [MQTT-3.1.2-18]." · "**3.1.3.4 User Name** — … It
> can be used by the Server for authentication and authorization."
> — [OASIS MQTT 3.1.1 §3.1.2.8, §3.1.3.4](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html)

with the response carrying an authorization verdict — `0x04` "Connection Refused, bad user name or
password" and `0x05` "Connection Refused, not authorized" (§3.2.2.3, Table 3.1; MQTT 5.0's
equivalents are reason codes `0x86` and `0x87`).

So there is no MQTT message that is not the authentication exchange. **MQTT contributes nothing to
either rule** — no TLS fact (8883 is implicit TLS, so §3.4's port-level check is all there is), no
anonymous-access fact. It is the clean example of a protocol the profile simply cannot reach, and it
is worth having on the record so that a later implementer does not spend a sprint discovering it.

### 6.8 What the line costs, counted

Anonymous access is measurable for **six** protocols: IMAP (`PREAUTH`), RFB (security type `None`),
AMQP (`ANONYMOUS` in the mechanism list), LDAP (a bind-free root-DSE read), Redis (a RESP error
versus `+PONG`), and any HTTP listener (`401`/`403` versus `200`).

It is **not** measurable for: FTP, MQTT, MySQL, PostgreSQL, MongoDB, SMB, Telnet, SSH, and every
other protocol whose authorization state is established by a bind, a connect packet, or a session
setup. That is the larger set, and it contains the services an operator is most likely to have
exposed.

The honest summary is that **the credential line does not merely make anonymous-access expensive; it
makes it unavailable for most of the estate.** #21's §5 hoped this rule would carry the middle band
it declined to build. It carries about a third of it.

---

## 7. How much table, and is it release-coupled

### 7.1 Three tables, named and sized

| Table | Job | Rows | Verdict |
|---|---|---|---|
| **Dispatch** — port → which protocol to speak | what to send, and where | ~14 (§5's `s_client` list is the natural size) | **aperture — legal** |
| **Grammar** — first bytes → which protocol | the same job, dispatched on measurement rather than on the port | ~6 (the greeting-first protocols of §4) | **aperture — legal, and better** |
| **Verdict** — product → what its response means | what the answer means | unbounded | **out** |

The verdict table is out by §2.1, and there is no version of these rules that needs one, which is
the load-bearing claim of the whole note. Every fact in §4 and §5 is read from a field the protocol's
own specification defines: an EHLO keyword, a capability token, a capability bit, a `resultCode`, a
SASL mechanism name, a security-type integer, a `failureCode`, a RESP type prefix, an HTTP status
class. Not one of them requires knowing what product is running.

### 7.2 Why the grammar table is better than the dispatch table

The two do the same job and one of them dispatches on evidence. §3.3's free partition tells us, at no
cost, whether the listener spoke first; where it did, §4's greetings identify the protocol from bytes
the estate sent us rather than from a number we assigned meaning to. **That is a determinacy
improvement over `sensitive-port-exposed`, and it is the ticket's claim made good** — but only for
the greeting-first protocols, which are five of the seventeen rows in §5.1.

For the rest the prober must choose from the port, and the determinacy problem returns
undiminished: speaking PostgreSQL at 5432 assumes PostgreSQL is there. **The mitigation is that the
cost of being wrong is asymmetric and small.** A wrong guess produces a protocol error or silence,
which canonicalises to `not-evaluable`, not to a firing. `sensitive-port-exposed` guessing wrong
produces a *signal*; this rule guessing wrong produces a `Gap`. That asymmetry is the real answer to
the ticket's "an observation-driven rule has no determinacy problem" — it has one, and it fails in
the safe direction.

### 7.3 Is the dispatch table release-coupled? Yes, and more strictly than the list it supplements

ADR-0004's test is a question about our own intentions: *would we ever want to push updates to this
list out of band?*

For the dispatch table the answer is that **we could not, even if we wanted to.** Adding a protocol
means shipping code that speaks it — a message encoder, a response parser, a canonicalisation of the
result into the closed value space. There is no file an operator or a maintainer could edit at 3am
to add MQTT support. The sensitive-port list *could* be hot-loaded from a data file and is
release-coupled only because we have decided it is; the dispatch table is release-coupled by
construction.

That is a stronger property than ADR-0004 asks for, and it is worth stating because it is what makes
the aperture framing safe rather than merely convenient.

**And ADR-0008 has already priced the growth.** A `Break`'s two causes are a `Derivation` vector
change or **the aperture widening**, and a widened aperture surfaces newly covered subjects as
`revealed` rather than `appeared` — *we started looking, the world did not move*. So adding a
protocol to the dispatch table:

- does **not** move the rule's version, because the rule reads a canonicalised facet value and does
  not know which protocols exist;
- does **not** re-baseline the estate, because the aperture widened only over `Service`s the prober
  could not previously address;
- costs exactly a `revealed` transition on the newly covered subjects, which is the treatment
  ADR-0007 already defined for this shape of change.

**This is the payoff of putting the protocol set in the aperture rather than in the rule**, and it is
what the curated sensitive list cannot do: there, the list *is* the verdict, so every edit bumps the
rule version and makes two evaluations non-comparable estate-wide (#21 §7.2). Here, the reference
data and the conclusion have been separated, and only the reference data grows.

### 7.4 The canonicaliser, and its churn budget

Adding a facet means adding a canonicaliser, and a canonicaliser is a versioned `Derivation` whose
version moves on any output-affecting change. ADR-0008 names the certificate canonicaliser as
"genuinely churny" and treats that as a cost to be avoided rather than accepted. This canonicaliser
must therefore project onto a **closed value space**, and everything outside it is discarded:

```
transport      ∈ { implicit-tls
                 , upgrade-offered          # STARTTLS / STLS / AUTH TLS / StartTLS / CLIENT_SSL / S
                 , upgrade-absent           # the protocol has one and this listener did not offer it
                 , plaintext-only           # the protocol defines no in-band upgrade
                 , not-evaluable }

authentication ∈ { required                 # the listener refused an unauthenticated read
                 , anonymous-permitted      # PREAUTH / security-type None / ANONYMOUS / 200 / +PONG
                 , not-evaluable }
```

Five values and three values. Compare `reachability`, which ADR-0008 singles out as the
*non*-churny counterexample precisely because it "maps a small closed value space".

Three design consequences fall out, and each is a bug avoided:

- **No free text, ever.** MySQL's `server version` string, SMTP's post-220 greeting text, Redis's
  `-NOAUTH` literal, RFB's `RFB 003.008` minor version — all discarded. A value carrying any of them
  would move on a point release of the software, and under ADR-0008 a moved output with an unmoved
  version fails the build, while a moved version `Break`s every listener timeline in the estate. The
  fingerprinting rule and the churn budget point the same way here, which is a good sign.
- **`upgrade-offered` is not `required`.** PostgreSQL's `S` (§5.2) and MySQL's `CLIENT_SSL` say the
  server *can*, never that it *insists*. Collapsing the two would make the value a claim we cannot
  measure, and the collapse is invisible in the data.
- **`not-evaluable` is a first-class value in both fields, not a null.** Most `Service`s will carry
  `not-evaluable` in the `authentication` field permanently (§6.8), and that has to be a stated fact
  rather than a gap in the record.

### 7.5 The one place a table would creep back in, and the guard against it

The temptation is `upgrade-absent`. Distinguishing *this protocol defines an in-band TLS upgrade and
this listener did not offer it* from *this protocol has no such upgrade* requires knowing which
protocols have one — a table of protocol capabilities.

**It is legal, and only because it is the same table.** The dispatch table already has to know it:
the prober cannot send `STARTTLS` at an FTP server or `AUTH TLS` at an IMAP server. The knowledge of
which upgrade a protocol has is inseparable from the ability to speak it, so `upgrade-absent`
consumes no reference data that the aperture does not already contain, and adds no row to it.

The guard is a rule an implementation can be tested against: **the canonicaliser may consult the
dispatch table only for the protocol it already chose to speak, and never to interpret a response it
did not solicit.** The moment it looks up "what should port 9200 have said", it has become a `match`
line.

---

## 8. What `Facet`s this implies

`CONTEXT.md` is explicit that "adding one means adding a way to compare its values, not a new way to
detect change", and ADR-0008 makes that comparison procedure a named, versioned `Derivation` whose
version moves on any output-affecting change. So a facet is not free and the question is how few we
can get away with.

### 8.1 The HTTP-shaped half needs no facet at all

The HTTP measurement is a status class on a `GET /` that v1 already performs (#4 §4.1: "`GET /`,
capped body read (64 KB), 10 s timeout"). Status code is already inside `http-identity`'s aperture.
**Adding nothing** is therefore the correct answer for the largest slice of coverage (§10), and it is
worth stating loudly because the instinct is to build the facet first.

The subject is also already right, and it matters. `http-identity` hangs off `Endpoint`, and `CONTEXT.md`
gives the reason: "two names on one address and port legitimately serve different content". A vhost
serving a public marketing site and a vhost serving an unprotected admin API on the same
`(Address, port, transport)` genuinely have different answers to *was an unauthenticated request
refused*. Keying this on `Service` would average two true facts into one false one.

### 8.2 The wire-protocol half needs exactly one facet, on `Service`

**Proposal: one new facet, `listener-negotiation`, on `Service`**, holding the two-field closed tuple
of §7.4.

> **Amended by [#104](https://github.com/winniel123/verge-asm/issues/104): whoever specifies this
> facet owes a decision on a *third* field — integrity — and it is the one part of the deferred work
> that is not free to defer twice.** §9.2's SMB signing fact is neither `transport` nor
> `authentication`, so it needs a field of its own; and
> [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) is explicit that a facet's value space
> is **decided once** and widened afterwards at the cost of a `Break` on every timeline it holds.
> While this facet does not exist the third field costs nothing to add and nothing to omit. The day
> it ships with two fields, adding the third stops being free. So the field question is owed **at
> specification time**, ahead of and independently of whether any rule reads it — which is ADR-0015's
> own finding about `http-identity`'s status class, one level down.

*Why on `Service` and not `Endpoint`.* None of the wire protocols in §5 is name-addressed. A
PostgreSQL listener's answer to `SSLRequest` does not vary by DNS name because nothing in the
exchange carries one. `(Address, port, transport)` is the key the fact is actually single-valued
under, which is the same test `CONTEXT.md` applies to justify `Endpoint` in the HTTP case.

*Why one facet rather than two.* The obvious alternative is `transport-security` and
`listener-authentication` as separate facets, on the grounds that they have wildly different
evaluability (§6.8) and so should gap independently. It is rejected:

- **They come from one exchange.** A `Span` is keyed by `(subject, facet, vantage, source)`, and two
  facets over one connection at one instant from one source produce two timelines that can never
  disagree. That is a second representation of one measurement — the shape ADR-0007 refuses for
  `Transition` and ADR-0004 refuses for re-derived reachability.
- **The tuple is the posture.** `plaintext-only` + `anonymous-permitted` is a materially different
  listener from `implicit-tls` + `anonymous-permitted`, and the pair is what an operator acts on.
- **Two facets is two canonicalisers, so two `Derivation` leaves**, each with its own golden corpus
  and its own ability to `Break`. Doubling the version surface to model one exchange is the wrong
  trade.

*The cost of one facet, named.* A span closes when **either** field changes, so a listener that
toggles its TLS mode truncates the duration on its authentication history too, and under ADR-0008
that renders as a labelled floor. This is a real loss and it is small: the two fields move for the
same reason (someone reconfigured the listener) far more often than independently.

### 8.3 The churn risk is in the canonicaliser, not in the facet

The facet is cheap; the canonicaliser is where a `Break` estate-wide comes from. §7.4's closed value
space is what keeps it non-churny, and the discipline is enforceable rather than aspirational:
ADR-0008's golden corpus takes fixed observations and expected outputs, and for this canonicaliser a
corpus row is *these bytes on the wire → this tuple*. Captured wire transcripts are exactly the kind
of fixture that is cheap to record once and never needs regenerating, which makes this one of the
easier corpora in the system to build.

---

## 9. One signal or several

### 9.1 Never per protocol

> **The principle *as stated in this heading* is WITHDRAWN — it is over-wide, and
> [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) says so in terms: *"The doubt is
> correct and **the principle as stated is wrong**."*** Written here by
> [#102](https://github.com/winniel123/verge-asm/issues/102) under
> [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md),
> because ADR-0015 recorded the correction in its own body and **this note contains no reference to
> ADR-0015 anywhere** — there is not even a pointer to fail.
>
> **What survives, unchanged, is the harm named below:** *growth coupled to a corpus*. ADR-0015:
> *"That harm is not caused by covering one protocol. It is caused by the signal being **named for**
> the protocol, so that admitting one commits us to admitting the next."*
>
> **The corrected test, which governs:**
> > **A signal is named for the fact it reads, and its scope is however many protocols happen to
> > express that fact.** One is fine when the fact is genuinely single-protocol. Three protocols
> > expressing one fact must be one signal, never three.
>
> So `smtp-starttls-absent` / `imap-starttls-absent` / `pop3-stls-absent` stay forbidden — one fact
> wearing three names — while **a single-member rule is not forbidden**. §12 q2 asked exactly this
> and ADR-0015 answered it; the question is closed. **§9.2's SMB exclusion rests on the withdrawn
> reading and is not re-decided here** — see the box at §9.2.

The tempting shape is `smtp-starttls-absent`, `mysql-tls-not-offered`, `amqp-anonymous-permitted`.
It is wrong for a reason that goes beyond taste: **a signal set that grows with the dispatch table is
a name per product, which is the signature database's silhouette even when every individual rule is
honest.** ADR-0004 excluded admin-panel titles and default-install pages on the grounds that a corpus
worth having is one updated continuously; a per-protocol signal set would recreate that pressure in
the signal namespace rather than in a data file.

It also forfeits §7.3's payoff. Per-protocol rules make the protocol set part of the rule, so adding
one bumps a version and costs comparability — precisely the cost the aperture framing exists to
avoid.

### 9.2 Two signals, split on the fact measured

- **`listener-offers-no-encryption`** — fires where `listener-negotiation.transport` is
  `upgrade-absent` or `plaintext-only`, or where the HTTP-shaped equivalent answered on a plaintext
  scheme with no TLS sibling.
- **`listener-permits-anonymous-access`** — fires where `listener-negotiation.authentication` is
  `anonymous-permitted`, or where an unauthenticated `GET /` was answered rather than refused.

**Why two rather than one.** Different evidence, different evaluability (five protocols versus
sixteen), and different remediations — one is "terminate TLS here", the other is "turn on
authentication". A combined `insecure-listener` would fire without telling the operator which thing
to do, and would fire roughly twice as often for the same information. ADR-0004's per-rule versioning
then does its job: an edit to the encryption rule leaves the anonymous-access rule comparable.

**Why not three.** The tempting third is a weak-authentication rule carrying RDP-without-NLA and
SMB-signing-not-required. It is refused:

- *Weak* is a ranking, and ranking one mode against another needs a per-protocol table of which mode
  counts as weak — a verdict table (§2.1). RDP's case actually dissolves: a server that accepts
  `PROTOCOL_RDP` has accepted a mode with no TLS, which is `transport = upgrade-absent` and belongs
  to the first signal already. Nothing is lost.
- **SMB signing is the closest call in the note.** `SMB2_NEGOTIATE_SIGNING_REQUIRED` is a real,
  spec-defined, credential-free bit, and its absence is a genuine and well-known weakness. But ~~it is
  integrity rather than confidentiality, so it fits neither rule~~, and ~~a rule built for it would cover
  exactly one protocol — which is §9.1's per-protocol signal wearing a general-sounding name~~.
  **Excluded from v1**, with the underlying question recorded in §12 rather than argued away: a
  single-protocol rule may be legitimate, and this note has not established that it is not.

  > **Both halves of this exclusion's ground are now struck, and the verdict survives on a third
  > ground this section never wrote down.** Ruled by
  > [#104](https://github.com/winniel123/verge-asm/issues/104) under
  > [ADR-0065](../adr/0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md),
  > completing the repair [#102](https://github.com/winniel123/verge-asm/issues/102) began under
  > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
  >
  > - **The per-protocol half** was withdrawn by
  >   [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md) — a single-protocol rule **is**
  >   legitimate where the fact is genuinely single-protocol — and struck here by #102.
  > - **The integrity-versus-confidentiality half is withdrawn too.** *"It fits neither rule"* is a
  >   claim about **the two rules this section admits**, never about SMB, and a fact fitting neither
  >   is a candidate for a **third** rule rather than a reason to drop the fact. ADR-0015 said as much
  >   while withdrawing the other half — *"it would be a third signal in any case — signing is
  >   integrity, which is neither of the two facts the other rules read"* — treating *third* as the
  >   shape of the answer and not as an objection to it.
  >
  > **What excludes it is the aperture, and that ground was true all along.** ADR-0015: *"SMB signing
  > is therefore admissible whenever the prober that can read `SMB2_NEGOTIATE_SIGNING_REQUIRED`
  > exists. It stays out of v1 because that prober does not."* Reading the field needs an SMB2
  > `NEGOTIATE` exchange — application bytes v1 does not send, under an `Offer` v1 has not declared —
  > and a facet to hold the result, which is §8.2's `listener-negotiation`, ruled **out of scope for
  > this map** by ADR-0015. `445/tcp` being a `verge-core` pair buys the **connect**, never the
  > exchange, so nothing here is free.
  >
  > **Three riders on the deferral.**
  > 1. **Where its coverage would actually be is not where this section assumed.** `445/tcp` is on the
  >    sensitive list in the explicit-prohibition tier, so `sensitive-port-reached-from-internet`
  >    already fires on it from the internet leg — Microsoft's own perimeter directive is the
  >    attestation. An integrity rule therefore adds **nothing** there, and its whole prize is the
  >    **internal** leg, which is the one place v1 is silent by design. That is why the row was the
  >    closest call, and it is also why the rule's relationship to `Exposure` is the open question
  >    below rather than a settled inheritance from §9.3.
  > 2. **It would not ship under the name `smb-signing-not-required`.** ADR-0015's corrected test
  >    forbids naming a rule for a protocol; the rule is named for the integrity fact, and its scope is
  >    however many protocols express that fact.
  > 3. **The deferral is free.** ADR-0014 and ADR-0015 price a new facet plus a new rule at `revealed`
  >    plus one coverage-class message with **no `Break`**, so there is no deadline and nothing bought
  >    by admitting it now. Admitting it early buys less than nothing: per ADR-0004's #44 amendment no
  >    subject would exist, so the rule would render **no row at all** — not even `not-evaluable` —
  >    while moving every *sixteen* in the corpus to *seventeen*.
  >
  > **Reopens** with the wire-protocol prober, and only with it. Two things travel to whoever picks
  > that up: the **third-field** obligation at §8.2's box, which is owed at specification time and
  > not at rule time; and the **generality check** ADR-0015's test requires — if another protocol
  > expresses the same integrity fact readably before authentication, it is **one** signal covering
  > both and never two.
  >
  > **One question is deliberately left open**, and #104 flags it as such rather than ruling it.
  > Whether an integrity rule reads `Exposure` — §9.2's two rules both do (§9.3) and this one
  > plausibly does not, which is exactly what would make the **internal** leg its coverage, since
  > `sensitive-port-reached-from-internet` already covers `445/tcp` on the internet leg and
  > [#58](https://github.com/winniel123/verge-asm/issues/58) refused an internal counterpart for want
  > of an attestation that the internal configuration is never correct. That is a question about a
  > rule's domain, and settling a domain before its facet exists is the wrong order.

### 9.3 Both read `Exposure`, and that is the difference from `tls-1.0-negotiated`

The ticket asks whether these read `Exposure` the way `sensitive-port-exposed` does, or apply
estate-wide regardless of vantage. **Both read `Exposure`**, and the reasoning is worth stating
because the v1 set already contains a rule that does *not*.

#4 §8.2 divides the v1 signals: exposure-class signals ("plaintext HTTP", "unexpected open port")
carry an implicit *…to the internet* and must not fire from an internal vantage, while
vantage-independent ones (cert expiry, weak TLS version, dangling DNS) may fire from either side.
`tls-1.0-negotiated` sits in the second group because RFC 8996 says "TLS 1.0 MUST NOT be used" with
no qualification about audience — there is no topology in which it is correct.

Neither of these two rules has that property:

- A plaintext PostgreSQL listener inside a VPC, behind a TLS-terminating proxy, is a normal and
  correct design. Its wrongness is entirely a function of who can reach it.
- A Redis with no password on a private network is the *documented default*, and RabbitMQ ships
  `ANONYMOUS` enabled deliberately (§5.5). Firing on those from an internal vantage is exactly #4
  §8.1's inverted result: the alarming direction, on evidence that means the opposite.

So both compose `Exposure`'s version, and per ADR-0008 the effective vector is the union of the
leaves:

```
listener-offers-no-encryption
  -> { the rule, listener-negotiation-canon } u Exposure's vector
  =  { the rule, listener-negotiation-canon, exposure, availability,
       reachability-canon, currency }
```

**Six leaves, and that is the price.** A move in any one of them `Break`s the signal, and four of the
six are outside the rule's control. ADR-0004 already accepted this shape for
`sensitive-port-exposed` — "the price is that bumping the `Exposure` derivation correctly makes the
signal non-comparable without anyone touching its rule" — and these rules inherit it with one extra
leaf of their own.

Whether they also fire on `edge-only` as well as `exposed` is #21 §8 q3's question, inherited
unchanged. This note does not settle it and should not: it is one question about every
`Exposure`-reading signal, and it should be answered once.

---

## 10. What this actually covers — the deflating count

The ticket's hope is that an observation-driven rule "could eventually cover ports the curated list
can never reach". It can. The question is how many, and the arithmetic is unflattering.

### 10.1 On the 38 rows of the sensitive list, the measurement adds no firing

Working through #21 §3's three classes: the measurement reaches Redis, MySQL, PostgreSQL, MongoDB,
Elasticsearch, VNC, FTP, SMB, CouchDB, kubelet, Docker and Telnet, and reaches none of etcd,
ZooKeeper, epmd, Cassandra, memcached, TFTP, rexec, rlogin, rsh, X11, NFS, rsync, SNMP, IPMI,
NetBIOS, MS SQL, the Elasticsearch transport port or the RabbitMQ inter-node port.

**On every one of the 38, `sensitive-port-exposed` already fires.** The measurement can refine what
is said — *and it offers no TLS*, *and it offered security type `None`* — but it produces no firing
the operator was not already going to see. **Net new signals on the sensitive list: zero.**

That is not a defect; it is the two rules doing different jobs on the same ports. But it means the
value has to come from somewhere else entirely.

### 10.2 Where the new coverage actually is

Ports **not** on the sensitive list where one of the two rules can fire:

| Port(s) | Why #21 excluded it | Reachable measurement | Shape |
|---|---|---|---|
| 8080, 8000, 8888, 8443, 3000, 5000, 9000, 9090, 8088, 10000 | determinacy (#21 §4.3) — conventionally anything | `401`/`403` vs `200`; plaintext vs TLS | **HTTP** |
| 6443 kube-apiserver | the closest call (#21 §4.4) | same | **HTTP** |
| 5985 / 5986 WinRM | remote administration is the purpose | same | **HTTP** |
| 5601, 8161, 8500, 15672 | determinacy / #21 §4.6 | same | **HTTP** |
| 3389 RDP | remote administration is the purpose | CredSSP required or not | **wire** |
| 25 / 587 SMTP | not on the list | `STARTTLS` present or absent | **wire** |
| 143 IMAP | not on the list | `STARTTLS` / `LOGINDISABLED` / `PREAUTH` | **wire** |
| 110 POP3 | not on the list | `STLS` present or absent | **wire** |
| 389 LDAP | want of attestation (#21 §8 q4) | StartTLS `resultCode`; bind-free root-DSE read | **wire** |
| 5672 AMQP | #21 §4.6 | `ANONYMOUS` in the mechanism list | **wire** |

**The wire-protocol prober's entire new coverage is six ports.** Everything else is HTTP.

### 10.3 The conclusion that should drive the spec

The HTTP-shaped rows need **no new facet, no new prober, no new dispatch table, no new safety
surface, and no new canonicaliser** — only a rule over a status class that `http-identity` already
records. They cover every port #21 excluded on determinacy grounds, which was the largest and most
frustrating category of that exclusion.

The wire-protocol rows need all of it — a new facet, a versioned canonicaliser with a golden corpus,
per-protocol encoders and parsers, a dispatch table, and a per-protocol expansion of what the prober
sends to production — and buy six ports, of which three (IMAP, LDAP, AMQP) yield an anonymous-access
fact and three (SMTP, POP3, RDP) yield only a transport fact.

**So the honest recommendation is a split.** The HTTP-shaped rule is v1-shaped and cheap. The
wire-protocol prober is not v1-shaped, and §5's per-protocol costing exists so that a later ticket
can take protocols one at a time — which §7.3 makes cheap, because each addition is an aperture
widening costing `revealed` on newly covered subjects rather than a rule-version bump costing
comparability estate-wide.

---

## 11. `not-evaluable`, and where the evidence is absent

ADR-0004 requires absent evidence to yield `not-evaluable` rather than "did not fire". These rules
inherit #21 §7.1's four routes and add three of their own, and an implementation that collapses them
will report a bill of health the estate never earned.

**Inherited from `sensitive-port-exposed`:**

1. **Ownership.** ADR-0002 limits probing on `third-party` and `unknown` addresses to the ports the
   `Name` implies. On a SaaS-fronted estate that is most of the estate, and only the HTTP-shaped rule
   has any evidence there at all.
2. **No internet vantage.** `Exposure` cannot be constructed without one, and both rules compose it
   (§9.3). A default single-vantage deployment can evaluate neither rule, ever.
3. **Tier cadence.** A port in the weekly or monthly tier is unmeasured between measurements.
4. **Transport not probed.** #4 §2.5 puts UDP off by default.

**New to these rules:**

5. **The protocol is not in the dispatch table.** MQTT, memcached, etcd, ZooKeeper, Cassandra, NFS,
   SNMP and the rest are `not-evaluable` permanently, not clean. This is the largest new route and it
   covers most of the sensitive list.
6. **The fact is unreachable inside the credential line.** The `authentication` field is
   `not-evaluable` for every protocol outside §6.8's six, and permanently so — no future probing
   effort inside #4's profile will change it.
7. **The listener did not answer the hello.** A wrong dispatch guess, a protocol error, a middlebox,
   or a server that closed on an unrecognised command. §7.2 argues this fails safe, and *safe* here
   means `not-evaluable`.

Route 6 deserves surfacing in the product rather than only in the data. An operator looking at a
listener whose authentication posture we structurally cannot measure should be told that, not shown
a blank — this is the same argument #21 §6.1 made for keeping its six unmeasured UDP rows visible
rather than dropping them.

---

## 12. Open questions for the spec

1. **Should the HTTP-shaped rule ship in v1 on its own, ahead of any wire-protocol prober?** §10.3
   argues yes and this note cannot settle it, because the answer depends on whether v1's signal set
   is closed. It is the cheapest signal in the whole survey and it covers #21's largest exclusion
   category.
2. ~~**Is a single-protocol signal ever legitimate?**~~ §9.2 excluded SMB signing on the grounds that a
   rule covering one protocol is a per-protocol signal by another name. That reasoning would also
   exclude any future rule for a protocol-specific weakness, which may be too strong. The question is
   general and the answer belongs somewhere other than an SMB row.
   > **ANSWERED — yes, and this question is closed.**
   > [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md): *"One is fine when the fact is
   > genuinely single-protocol."* The answer did indeed land somewhere other than an SMB row, exactly
   > as this question asked — and nothing carried it back here, which is why
   > [#102](https://github.com/winniel123/verge-asm/issues/102) writes it in under
   > [ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
   > See §9.1's box for the corrected test and §9.2's for what it does and does not move.
   >
   > **And the row it was raised from is settled too, so nothing is parked here any longer.**
   > [#104](https://github.com/winniel123/verge-asm/issues/104) ruled the SMB verdict: the fact is
   > **admissible** — single-protocol scope is legitimate, and it would be a **third** rule reading
   > **integrity**, which is neither fact §9.2's two rules read — and it is **excluded from v1 on the
   > aperture**, which is the only ground §9.2 never wrote down. See
   > [ADR-0065](../adr/0065-a-rule-is-excluded-by-its-fact-or-by-its-aperture-never-by-the-shape-of-the-set.md).
3. **Does the `revealed` treatment of a widened dispatch table actually hold in the presence of a
   `Gap`?** §7.3 leans on ADR-0008. But a `Service` already carrying `not-evaluable` under route 5
   has a timeline; widening the aperture over it looks like a value appearing where a `Gap` was, not
   like a new subject. Whether that is `revealed` or something else is not settled by ADR-0007's
   three membership transitions.
4. **Where does the golden corpus for the listener canonicaliser come from?** §8.3 proposes captured
   wire transcripts. That is a new *kind* of fixture for this project — every other corpus row is
   structured observations — and it needs a format and a capture procedure decided before the first
   protocol ships.
5. **Should `sensitive-port-exposed` and `listener-offers-no-encryption` be allowed to fire on the
   same `Service`?** §10.1 establishes that on all 38 sensitive rows the second adds no new firing,
   only refinement. Two signals on one service saying overlapping things is a presentation problem
   that #22 has not been asked about.
6. **Does #21's evidence standard apply to these rules, and does it change any verdict?** This note
   used a different standard — *is the fact read from a spec-defined field* — because the reference
   data is aperture rather than verdict. MongoDB (§5.8) was nonetheless excluded on #21's grounds
   (no attested answer), which suggests the two standards are complementary rather than alternative,
   but that has not been established.
7. **Is the `telnet` row worth anything?** OpenSSL ships `-starttls telnet` (§5), implying RFC 2941
   negotiation is drivable credential-free, and 23 is on the sensitive list. Nobody deploys it, so
   the measurement would almost always return `upgrade-absent` — which is the correct answer and also
   a useless one. Recorded because the `s_client` list makes it look reachable.

---

## Sources

**Specifications — mail, file transfer, directory**
- [RFC 5321 — SMTP](https://www.rfc-editor.org/rfc/rfc5321.txt) §3.1, §3.2, §4.3.1 · [RFC 3207 — SMTP over TLS](https://www.rfc-editor.org/rfc/rfc3207.txt) §2, §3, §4, §7 · [RFC 4954 — SMTP AUTH](https://www.rfc-editor.org/rfc/rfc4954.txt) §3, §4, §6, §9
- [RFC 8314 — Cleartext Considered Obsolete](https://www.rfc-editor.org/rfc/rfc8314.txt) §1, §2, §3.1–3.3, §4
- [RFC 9051 — IMAP4rev2](https://www.rfc-editor.org/rfc/rfc9051.txt) §3, §5, §6.2, §6.2.1, §6.2.3, §7.1, §7.1.1, §7.1.4, §7.2.2, §8, §11 · [RFC 3501 — IMAP4rev1](https://www.rfc-editor.org/rfc/rfc3501.txt) §6.2.1, §6.2.3, §7.1, §7.1.1, §7.1.4, §7.2.1
- [RFC 1939 — POP3](https://www.rfc-editor.org/rfc/rfc1939.txt) §3, §4 · [RFC 2449 — POP3 extension mechanism](https://www.rfc-editor.org/rfc/rfc2449.txt) §5 · [RFC 2595 — Using TLS with IMAP, POP3 and ACAP](https://www.rfc-editor.org/rfc/rfc2595.txt) §4, §8
- [RFC 959 — FTP](https://www.rfc-editor.org/rfc/rfc959.txt) §4.1.1, §4.2, §4.2.1, §5.4, §6 · [RFC 2389 — FTP feature negotiation](https://www.rfc-editor.org/rfc/rfc2389.txt) §3, §3.1, §3.2 · [RFC 4217 — Securing FTP with TLS](https://www.rfc-editor.org/rfc/rfc4217.txt) §4.1, §4.2, §6, §12.1 · [RFC 1635 — Anonymous FTP](https://www.rfc-editor.org/rfc/rfc1635.txt) · [RFC 2577 — FTP Security Considerations](https://www.rfc-editor.org/rfc/rfc2577.txt) §6, §9
- [RFC 4511 — LDAPv3](https://www.rfc-editor.org/rfc/rfc4511.txt) §4.2, §4.12, §4.14, §4.14.1 · [RFC 4512 — LDAP Directory Information Models](https://www.rfc-editor.org/rfc/rfc4512.txt) §5.1, §5.1.4 · [RFC 4513 — LDAP Authentication Methods](https://www.rfc-editor.org/rfc/rfc4513.txt) §3.1.1, §4, §5.1, §5.1.1, §5.1.2, §5.2.1.5, §6.2

**Specifications — transport, messaging, remote framebuffer, HTTP**
- [RFC 6143 — The Remote Framebuffer Protocol](https://www.rfc-editor.org/rfc/rfc6143.txt) §7.1.1, §7.1.2, §7.2.1, §7.2.2
- [RFC 4505 — SASL ANONYMOUS](https://www.rfc-editor.org/rfc/rfc4505.txt) §1, §2, §5
- [AMQP 0-9-1 specification](https://www.rabbitmq.com/resources/specs/amqp0-9-1.pdf) §2.2.4, §4.2.2 · [amqp0-9-1.xml](https://www.rabbitmq.com/resources/specs/amqp0-9-1.xml), `connection.start`
- [OASIS AMQP 1.0 — Security](http://docs.oasis-open.org/amqp/core/v1.0/os/amqp-core-security-v1.0-os.html) §5.2.1, §5.3, §5.3.2, §5.3.3.1
- [OASIS MQTT 3.1.1](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/os/mqtt-v3.1.1-os.html) §3.1, §3.1.2.8, §3.1.2.9, §3.1.3.4, §3.2.2.3 · [OASIS MQTT 5.0](https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html) §2.4, §3.2.2.2
- [RFC 9110 — HTTP Semantics](https://www.rfc-editor.org/rfc/rfc9110.txt) §9.2.1, §9.3.1, §11.1, §11.6.1, §15.5.2, §15.5.4
- [RFC 8996 — Deprecating TLS 1.0 and TLS 1.1](https://www.rfc-editor.org/rfc/rfc8996.html) — cited for the contrast in §9.3

**Vendor and project documentation**
- MySQL: [Connection Phase](https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_connection_phase.html) · [Protocol::HandshakeV10](https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_connection_phase_packets_protocol_handshake_v10.html) · [capability flags](https://dev.mysql.com/doc/dev/mysql-server/latest/group__group__cs__capabilities__flags.html) · MariaDB: [Connecting](https://mariadb.com/kb/en/connection/)
- PostgreSQL: [§54.2 Message Flow](https://www.postgresql.org/docs/current/protocol-flow.html) (§54.2.1, §54.2.10, §54.2.11) · [§54.7 Message Formats](https://www.postgresql.org/docs/current/protocol-message-formats.html)
- Redis: [Security](https://redis.io/docs/latest/operate/oss_and_stack/management/security/) · [Encryption](https://redis.io/docs/latest/operate/oss_and_stack/management/security/encryption/) · [AUTH](https://redis.io/docs/latest/commands/auth/) · [PING](https://redis.io/docs/latest/commands/ping/) · [RESP protocol spec](https://redis.io/docs/latest/develop/reference/protocol-spec/)
- MongoDB: [configuration options](https://www.mongodb.com/docs/manual/reference/configuration-options/) · [Configure mongod and mongos for TLS/SSL](https://www.mongodb.com/docs/manual/tutorial/configure-ssl/) · [hello](https://www.mongodb.com/docs/manual/reference/command/hello/) · [localhost exception](https://www.mongodb.com/docs/manual/core/localhost-exception/) · [mongodb/specifications, handshake.md](https://raw.githubusercontent.com/mongodb/specifications/master/source/mongodb-handshake/handshake.md)
- RabbitMQ: [Access Control / Authentication Mechanisms](https://www.rabbitmq.com/docs/access-control)
- Microsoft: [MS-RDPBCGR §2.2.1.1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/18a27ef9-6f9a-4501-b000-94b1fe3c2c10) · [§2.2.1.1.1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/902b090b-9cb3-4efc-92bf-ee13373371e3) · [§2.2.1.2.1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/b2975bdc-6d56-49ee-9c57-f2ff3a0b6817) · [§2.2.1.2.2](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/1b3920e7-0116-4345-bc45-f2c4ad012761) · [§5.4.2.1](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpbcgr/db98be23-733a-4fd2-b086-002cd2ba02e5) · [MS-SMB2 §2.2.4](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/63abf97c-0d09-47e2-88d6-6bfa552949a5) · [§2.2.5](https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-smb2/5a3c2c28-d6b0-48ed-b917-a86b2ca4575f) · [Allow access to your PC from outside your network](https://learn.microsoft.com/en-us/windows-server/remote/remote-desktop-services/remotepc/remote-desktop-allow-access) · [SMB signing](https://learn.microsoft.com/en-us/windows-server/storage/file-server/smb-signing)
- Kubernetes: [kube-apiserver reference](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/) · [Authenticating](https://kubernetes.io/docs/reference/access-authn-authz/authentication/) · [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- Docker: [Docker daemon attack surface](https://docs.docker.com/engine/security/) · [Protect the Docker daemon socket](https://docs.docker.com/engine/security/protect-access/) · [Deprecated features](https://docs.docker.com/engine/deprecated/)
- Elastic: [Self-managed security setup](https://www.elastic.co/docs/deploy-manage/security/self-setup) · [Minimal security](https://www.elastic.co/docs/deploy-manage/security/set-up-minimal-security) · [Get cluster info](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-info)
- Nmap: [nmap-service-probes](https://raw.githubusercontent.com/nmap/nmap/master/nmap-service-probes) · [Service and Version Detection file format](https://nmap.org/book/vscan-fileformat.html)

**Measured in the course of this note**
- `nmap-service-probes` directive counts (§2.1) — 187 `Probe`, 11,968 `match`, 203 `softmatch`, 12 `fallback`, over 17,154 lines of the current file
- TLS handshake failure modes against greeting-first and silent plaintext listeners on loopback, and against a real HTTPS endpoint (§3.3) — OpenSSL 3.5.7, listeners written for the purpose in Node 24
- `openssl s_client -starttls` supported-protocol enumeration (§5) — OpenSSL 3.5.7
- HTTP `401` with and without the RFC-mandated `WWW-Authenticate` header (§6.6) — ordinary unauthenticated `curl` requests to two public APIs

**Checked and found to contain no position, which is itself the finding**
- RFC 3207 — no rule on what a server's *omission* of the STARTTLS advertisement means (§5.3)
- RFC 2389 — entirely silent on whether `FEAT` may be issued before login; "login", "logged" and "authenticat" do not occur in the document (§5.3)
- RFC 4512 — silent on anonymous access to the root DSE; "anonymous" does not occur in the document (§5.4)
- redis.io — the `-NOAUTH` error string is documented nowhere across AUTH, PING, Security, Encryption, ACL and the RESP protocol spec; only `-DENIED` is documented (§6.3)
- mongodb.com — the `hello` reference page says nothing about authentication, and no page quotes what an unauthenticated client receives (§5.8)
- kubernetes.io — the non-resource URLs granted by `system:public-info-viewer` are not enumerated anywhere in the documentation (§5.9)
- elastic.co — neither the status returned to an unauthenticated request nor the behaviour of `GET /` on a security-disabled cluster is documented (§6.6)
- docs.docker.com — Docker never writes "no authentication"; it writes "not permitted" and "root access to the machine hosting the daemon" (§5.9)
- MS-RDPBCGR — never uses the phrase "before authentication"; "initially exchanged in the clear" is the closest verbatim support (§5.7)

**Correction to a sibling note**
- [`safe-active-probing.md`](https://github.com/winniel123/verge-asm/blob/research/safe-active-probing/docs/research/safe-active-probing.md) §4.1 and §10 cite RFC 9110 **§9.3.1** for the safety of GET. §9.3.1 defines the GET method and does not use the word "safe"; safe methods are **§9.2.1**. The claim is correct and the section number is wrong in both places (§6.6).
