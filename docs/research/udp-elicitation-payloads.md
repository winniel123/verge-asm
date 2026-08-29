# UDP elicitation payloads for the five sensitive pairs — corroborated against implementations

Research ticket #174 — wayfinder research for the verge-asm v1 spec.

**Question.** [ADR-0083](../adr/0083-silence-decides-only-on-a-connection-oriented-transport.md)
specified an honest UDP reachability instrument (`datagram-outcome`, a sixth leaf, not shipped) and
ruled it worthless without a **per-pair elicitation payload** — a datagram that actually provokes a
reply from each of `verge-core`'s five sensitive UDP pairs. That payload table is an **eighth
aperture input** and must be authored from the protocols' own specifications, never lifted from
nmap's NPSL-licensed `nmap-payloads`/`nmap-service-probes`. ADR-0083's Consequences state: *"A
retrieval per pair against implementations, not only specifications, is owed before any payload
ships."*

[`safe-active-probing.md`](./safe-active-probing.md) §13.6 already did the spec-only half of that
walk — RFCs and DSP0136 alone, no implementation source consulted — and §13.7 named exactly where it
was thin: the `138/udp` DATAGRAM ERROR row (no owner documentation found either way) and the
`623/udp` row (the IPMI 2.0 specification itself could not be retrieved, Intel returns 403, so the
row rested on the ASF companion spec alone). **This document is the retrieval against
implementations.** It does not rule on cost or recommend opening the knob — that is a downstream
decision. It only prices the evidence.

Three marks are used, matching [`measurement-offers.md`](../spec/measurement-offers.md)'s standard:

| Mark | Means |
| --- | --- |
| `[spec]` | Attested by the protocol's own specification |
| `[owner]` | Attested by the implementation's own source code, or the owning party's first-party documentation |
| `[thin]` | Chosen or unresolved. No attestation, and the gap is stated rather than papered over |

---

## 1. `69/udp` TFTP

**Wire bytes sent** — an RRQ for a filename guaranteed not to exist:

```
2 bytes     string    1 byte   string   1 byte
opcode=0001 filename    0x00    "octet"   0x00
```

RFC 1350 §5: opcode `0x0001` (RRQ), a filename unlikely to collide with anything real (a random
token is sufficient — no exploit or path-traversal content is needed), mode `octet`, both strings
NUL-terminated.

**Expected reply** — `[spec]` RFC 1350 §2, §4, §5, Error Codes appendix:

```
2 bytes     2 bytes         string    1 byte
opcode=0005 errorcode=0001  message    0x00
```

ERROR, opcode `0x0005`, error code `0x0001` ("File not found"). §2: the ERROR packet is neither
acknowledged nor retransmitted — one shot, one listen window.

**Implementation corroboration — `[owner]`, confirmed.** Read tftp-hpa's `tftpd/tftpd.c` (the
`tftpd-hpa` package, the TFTP server shipped by Debian, Ubuntu, RHEL and most Linux distributions).
`validate_access()` maps a missing file to `ENOTFOUND` on `ENOENT`/`ENOTDIR`, and `nak()` (the
function that emits the ERROR packet) sends it with error code 1, matching RFC 1350 exactly.

Critically, the reply-port behavior is also confirmed in the same source, and it is **not** the
well-known port. Before sending, `tftpd.c` does:

```c
peer = socket(myaddr.sa.sa_family, SOCK_DGRAM, 0);
if (pick_port_bind(peer, &myaddr, portrange_from, portrange_to) < 0) {
    syslog(LOG_ERR, "bind: %m");
    exit(EX_IOERR);
}
if (connect(peer, &from.sa, SOCKLEN(&from)) < 0)
```

A fresh datagram socket is bound to a new (optionally range-restricted) ephemeral port per transfer,
`connect()`ed back to the requester, and the ERROR packet goes out on *that* socket — never on the
listening port-69 socket. This is RFC 1350 §4's TID rule ("a fresh server-chosen TID, not port 69")
observed in a real, widely-deployed server, not only asserted by the spec.

**Caveat for the prober/listener.** The reply arrives from an ephemeral source port, not port 69.
`answered` must therefore mean *a datagram came back to the socket we sent from*, never *from the
port we probed* — which forces an **unconnected** UDP socket on the receive side, because a connected
socket filters on peer port and would silently drop this exact reply (ADR-0083's own finding, now
doubly confirmed: it is both RFC 1350's rule and tftp-hpa's actual behavior).

---

## 2. `137/udp` NetBIOS Name Service

**Wire bytes sent** — a NODE STATUS REQUEST with the wildcard `*` NBSTAT query, RFC 1002 §4.2.1,
§4.2.17–18:

```
Header (12 bytes): NAME_TRN_ID (prober-chosen), OPCODE=0 (query), NM_FLAGS: broadcast=0
  (unicast, direct to the target), RD=0; RCODE=0; QDCOUNT=1, ANCOUNT=0, NSCOUNT=0, ARCOUNT=0
Question:
  QUESTION_NAME  = first-level-encoded "*"           — the 16-byte name "*" + 15 spaces (0x20),
                                                         suffix byte 0x00, each of the 16 bytes
                                                         split into two nibbles and mapped to
                                                         'A'..'P', giving a 32-byte encoded label
                                                         prefixed with length 0x20 and NUL-terminated
  QUESTION_TYPE  = 0x0021 (NBSTAT)
  QUESTION_CLASS = 0x0001 (IN)
```

**Expected reply** — `[spec]` RFC 1002 §4.2.18, §5.1: a NODE STATUS RESPONSE (RR_TYPE NBSTAT,
`0x0021`) listing every name the node owns, statistics, and MAC address, sent to *the source UDP
port and source IP address of the request packet* (§5.1, stated as an unconditional rule).

**Implementation corroboration — `[owner]`, confirmed, and the confirmation is more specific than the
spec walk anticipated.** Read Samba's `nmbd` source (github.com/samba-team/samba, `source3/nmbd/`).
Two findings:

1. `process_node_status_request()` in `nmbd_incomingrequests.c` only answers a NODE STATUS REQUEST
   if the queried name is found among the node's **own** registered names:
   `find_name_on_subnet(subrec, &nmb->question.question_name, FIND_SELF_NAME)`, an exact `memcmp`
   against the subnet's name list (`nmbd_namelistdb.c`, `find_name_on_subnet()`) — no wildcard
   matching logic exists in the comparison itself.
2. That would make a literal `*` query fail — **except** `nmbd` deliberately registers `*` as one of
   its own names. `add_samba_names_to_subnet()` in `nmbd_namelistdb.c` runs at subnet setup and calls:

   ```c
   add_name_to_subnet(subrec,"*",0x0,samba_nb_type, PERMANENT_TTL, PERMANENT_NAME, num_ips, iplist);
   add_name_to_subnet(subrec,"*",0x20,samba_nb_type,PERMANENT_TTL, PERMANENT_NAME, num_ips, iplist);
   ```

   registering `*` (types `0x00` and `0x20`) as a `PERMANENT_NAME` on every subnet specifically so
   that `find_name_on_subnet(subrec, "*", FIND_SELF_NAME)` succeeds. `process_node_status_request()`
   then filters `*` and `__SAMBA__` back out of the name list it *returns* (`!strequal(name,"*") &&
   !strequal(name,"__SAMBA__")`) so the pseudo-name doesn't appear as one of the node's real services.
   This is purpose-built support for exactly the wildcard convention the spec walk inferred was
   necessary — not an accident of loose matching.
3. `reply_netbios_packet()` in `nmbd_packets.c` builds the response by `packet = *orig_packet;` (a
   full structure copy, which carries the request's source IP and port), so the reply is addressed
   back to the requester's address and port — RFC 1002 §5.1 confirmed in the implementation, not only
   in the spec.

Also checked Microsoft's MS-NBTE Open Specification (Microsoft Learn) as the Windows-side source,
since Windows also runs this service. Its overview states it "modifies the syntax of allowable
NetBIOS names and the behavior of timers, and add[s] support for multihomed hosts" over the RFC
1001/1002 base — it does not claim to change name-query/NBSTAT response behavior. Its Appendix A
"Product Behavior" page (19 numbered deviation notes, current through Windows Server 2025) contains
no exception for NBSTAT/wildcard query handling, meaning Windows is attested to follow the RFC 1002
base here by the absence of a documented deviation, though MS-NBTE's body text was not found to
explicitly restate the NBSTAT response rule itself. This is a first-party doc corroboration weaker
than the Samba source read, and is marked accordingly below.

**Caveat.** None beyond the shared unconnected-socket requirement in §1 — the reply legitimately
comes from port 137, so this pair does not force the unconnected-socket design on its own, but must
not regress it for the other four.

---

## 3. `138/udp` NetBIOS Datagram Service — still the weakest row, now with more specific negative evidence

**Wire bytes sent** — a DIRECT_UNIQUE datagram (RFC 1002 §4.4.1) addressed to a destination NetBIOS
name the target is not expected to own:

```
MSG_TYPE       = 0x10 (DIRECT_UNIQUE DATAGRAM)                     1 byte
FLAGS          = 0x02 (B-node, first/only fragment, more=0)        1 byte
DGM_ID         (prober-chosen)                                     2 bytes
SOURCE_IP                                                          4 bytes
SOURCE_PORT                                                        2 bytes
DGM_LENGTH     (computed)                                          2 bytes
PACKET_OFFSET  = 0x0000                                            2 bytes
SOURCE_NAME      (encoded NetBIOS name, any valid name)            34 bytes
DESTINATION_NAME (encoded NetBIOS name not owned by the target)    34 bytes
USER_DATA        (minimal — RFC 1002 §5.3.3's pseudocode resolves the destination-name lookup
                   before any mailslot/SMB dispatch, so content is not load-bearing for elicitation)
```

**Expected reply** — `[spec]` RFC 1002 §4.4.3, §5.3.3's own pseudocode: a DATAGRAM ERROR
(`MSG_TYPE = 0x13`), `ERROR_CODE = 0x82` ("DESTINATION NAME NOT PRESENT"), sent to the source IP and
source UDP port of the original datagram.

**Implementation corroboration — attempted hard, and the honest result is still no confirmed sender,
with one new and specific piece of evidence in each direction.** This was the row flagged as weakest
in §13.7 of the spec walk, and the retrieval effort here was the largest of the five.

*What was found, in Samba's favor:* Samba's own protocol model matches RFC 1002 §4.4.3 exactly.
`librpc/idl/nbt.idl` — the IDL that generates Samba's NDR parsers — defines:

```
DGRAM_ERROR                  = 0x13,
DGRAM_ERROR_NAME_NOT_PRESENT = 0x82,
DGRAM_ERROR_INVALID_SOURCE   = 0x83,
```

and `source3/include/nameserv.h` / `smb_macros.h` independently define `DGRAM_ERROR 0x13` and
`DGRAM_DIRECT_UNIQUE 0x10` for the classic `nmbd` codebase. Samba's developers modeled this exact
wire structure in two independent places in the tree, which is meaningful — a project does not
usually encode a message shape it has no reason to parse or emit.

*What was searched and not found:* actual code that **constructs and sends** a `DGRAM_ERROR` packet
for an unrecognized `DIRECT_UNIQUE` destination, in either of Samba's two independent NBT
implementations:

- **`source3/nmbd/` (the classic `nmbd` daemon)** — `nmbd_incomingdgrams.c` is the file that
  processes incoming datagram-service traffic. Its only functions are
  `process_host_announce()`, `process_workgroup_announce()`, `process_local_master_announce()`,
  `process_master_browser_announce()`, `process_lm_host_announce()`,
  `send_backup_list_response()`, `process_get_backup_list_request()`, `process_reset_browser()`,
  `process_announce_request()`, `process_lm_announce_request()` — all browse-list/announcement
  handling, none of them a name-ownership check on a `DIRECT_UNIQUE` datagram, none of them emitting
  `DGRAM_ERROR`. `nmbd_incomingrequests.c` (the file handling name-service requests) has no datagram
  code at all — it is scoped to port 137, not 138.
- **`source4/nbt_server/dgram/` (the AD-DC-embedded NBT server, a second, independent
  implementation used by the `samba` AD DC process)** — `request.c`'s `dgram_request_handler()`
  explicitly filters to `DGRAM_DIRECT_UNIQUE` (`if (packet->msg_type != DGRAM_DIRECT_UNIQUE) {...}`)
  and then dispatches to `nb_packet_dispatch(nbtsrv->unexpected_server, pstruct)` with no destination-
  name lookup and no error-reply path visible in the handler.

So: **two** independent, current Samba NBT implementations were checked, both define the wire
constant, and neither shows a visible path that emits it. That is stronger negative evidence than
"not verified" — it is "looked in both places a Samba emission would have to live, and it isn't
there" — though it is still not proof of absence (the send path could exist behind indirection not
reached by this search, e.g. inside `nb_packet_dispatch()` itself, which was not further traced).

On the Microsoft side: MS-NBTE's Appendix A Product Behavior page was read in full (19 numbered
notes). One note (`<11>`, §3.2.5.1) documents a Windows-specific deviation in datagram
**distribution** for group names — proving MS-NBTE's authors do track datagram-service deviations
when they exist — but no note anywhere addresses the `DIRECT_UNIQUE`-to-unowned-name error case.
Under MS-NBTE's own convention (deviations are called out — silence means the RFC 1001/1002 base
applies unmodified), this is *consistent with* Windows following RFC 1002 §5.3.3, but MS-NBTE's body
text was not found to affirmatively restate or test that specific behavior, so this remains an
absence of a documented exception, not a positive statement.

**Verdict for this row: still `[thin]`, unresolved in the direction of confirmation.** No
implementation — Samba or Windows — was found to affirmatively send the DATAGRAM ERROR RFC 1002
describes. The new information is that the two Samba trees checked define the constant but show no
call site, which is a more specific and slightly more troubling gap than "wasn't checked." Whether
Windows's `nbt.sys` emits it, and whether some other path in Samba's `nb_packet_dispatch()` chain
does, was not resolved. This is exactly the retrieval-against-implementations ADR-0083 asked for, and
it came back negative rather than uncorroborated.

**Caveat.** If a real deployment target does not emit this error, `138/udp`'s only payload produces
`unanswered` universally, which is indistinguishable at the wire from *nobody there* — the same
concern ADR-0083 raised about a payload-free instrument, now possibly true of this one specific pair
even with a payload authored in good faith from the spec.

---

## 4. `623/udp` IPMI / ASF-RMCP

**Wire bytes sent** — an RMCP Presence Ping, DMTF DSP0136 §3.2.4.8, §3.2.1:

```
RMCP header:
  Version          = 0x06 (RMCP version 1.0)     1 byte
  Reserved         = 0x00                        1 byte
  Sequence Number  = 0xFF (no ACK requested)      1 byte
  Class of Message = 0x06 (ASF)                   1 byte
ASF message:
  Enterprise Number (IANA) = 0x000011BE (4542)    4 bytes, network byte order
  Message Type             = 0x80 (Presence Ping) 1 byte
  Message Tag              (vary across retries)  1 byte
  Reserved                 = 0x00                 1 byte
  Data Length              = 0x00 (no payload)    1 byte
```

12 bytes total. §3.2.1: the prober's source port becomes the reply's destination port. §3.2.3 places
discovery before session creation, so no credential is needed. §3.2.2 permits a device to answer only
unique Message Tags, hence varying the tag across retries.

**Expected reply** — `[spec]` DSP0136 §3.2.4.3: Presence Pong, `Message Type = 0x40`, same Message
Tag echoed, followed by a 16-byte payload (OEM IANA, OEM-defined field, Supported Entities byte with
bit 7 = IPMI support, Supported Interactions byte, 6 reserved bytes).

**Implementation corroboration — `[owner]`, new and direct.** The IPMI 2.0 specification itself is
**still not retrievable**: `intel.com`'s spec page returns HTTP 403 again today, and a PDF found via
web search at a third-party host (`img1.wsimg.com`) was fetched and inspected directly — it is a
2-page, `wkhtmltopdf`-rendered document, not the ~600-page IPMI 2.0 spec, so it was discarded as a
false hit rather than cited. `scribd.com` and `studylib.net` mirrors were located but returned
`429`/rate-limited on fetch and were not read. **So the corroborating read of the IPMI 2.0 spec
itself, called out as missing in §13.7, is still missing** — this is an honest non-result, not a
silent drop.

What *was* newly obtained is a first-party client implementation read: `ipmitool`
(github.com/ipmitool/ipmitool, `src/plugins/lanplus/lanplus.c`, function `ipmiv2_lan_ping()`), the
de facto standard open-source IPMI client. It builds exactly the structure above:

```c
struct asf_hdr asf_ping = {
    .iana = htonl(ASF_RMCP_IANA),   /* 0x000011be */
    .type = ASF_TYPE_PING,          /* 0x80 */
};
struct rmcp_hdr rmcp_ping = {
    .ver   = RMCP_VERSION_1,        /* 0x06 */
    .class = RMCP_CLASS_ASF,        /* 0x06 */
    .seq   = 0xff,
};
```

A second, independent confirmation of the same 12-byte structure came from a real captured packet
quoted in `ipmitool` issue #268 (github.com/ipmitool/ipmitool/issues/268): `06 00 ff 06 00 00 11 be
80 00 00 00` — byte-for-byte the structure above (version, reserved, seq, class, IANA, type, tag,
reserved, data length). Reply validation is handled by `ipmi_handle_pong()`, which checks
`pong->sup_entities & 0x80` for IPMI-capability, matching DSP0136 §3.2.4.3's Supported Entities byte.
This corroborates DSP0136's description from a real, widely-run client — the corroboration ADR-0083
asked for — even though the base IPMI 2.0 spec remains unread.

**Discrepancy worth recording.** `ipmitool`'s `ipmiv2_lan_ping()` does **not** vary the Message Tag
across calls — it is left at its `memset`-zeroed default every time, contradicting the spec walk's
own recommendation (echoed above) to vary the tag to avoid DSP0136 §3.2.2's dedup. The reference
client evidently doesn't need to, likely because it does not send concurrent/duplicate pings to the
same target. A prober sending retries in flight would still need to vary the tag per
DSP0136's own text. `ipmitool`'s choice not to is a single-shot client's convenience, not a
counter-example to the spec rule.

**Caveat.** Reply routing depends on the socket being connected (or otherwise scoped) to the specific
target — `ipmitool` calls `recv()` on the same socket it sent from, relying on the OS/connect-state
rather than inspecting the reply's source port itself. Unlike TFTP, this pair does not force an
unconnected socket. It is compatible with either, since the reply's source is the probed host on the
port we sent to.

---

## 5. `11211/udp` memcached

**Wire bytes sent** — the 8-byte UDP frame header (`doc/protocol.txt`, UDP section) prepended to a
command that always answers regardless of cache contents:

```
Request ID       (prober-chosen, echoed back)   2 bytes
Sequence number  = 0x0000                       2 bytes
Total datagrams  = 0x0001                       2 bytes
Reserved         = 0x0000                       2 bytes
ASCII command: "version\r\n"                    9 bytes
```

`version` is preferred over a `get` of a nonexistent key because it elicits a reply unconditionally —
no dependency on what happens to be cached.

**Expected reply** — `[spec]`/`[owner]` `doc/protocol.txt`: the same 8-byte header (Request ID
echoed) followed by `VERSION <version>\r\n`.

**Implementation corroboration — `[owner]`, confirmed directly against source, not the man page.**
Read `github.com/memcached/memcached`, `memcached.c`. Two independent confirmations:

1. **The 8-byte header matches the doc exactly.** `build_udp_header()`:

   ```c
   static void build_udp_header(unsigned char *hdr, mc_resp *resp) {
       ...
       *hdr++ = resp->request_id / 256;
       *hdr++ = resp->request_id % 256;
       *hdr++ = resp->udp_sequence / 256;
       *hdr++ = resp->udp_sequence % 256;
       *hdr++ = resp->udp_total / 256;
       *hdr++ = resp->udp_total % 256;
       *hdr++ = 0;
       *hdr++ = 0;
       resp->udp_sequence++;
   }
   ```

   Request ID, sequence, total-datagrams, two reserved zero bytes — the same four fields
   `doc/protocol.txt` describes, written by the server itself, not merely documented.

2. **`-U 0` ("off") is the default in the argument parser, not only the man page.** `settings_init()`
   sets `settings.udpport = 0;`, and the `-U` flag's handler in `main()`'s `getopt` loop is
   `case 'U': settings.udpport = atoi(optarg); udp_specified = true; break;` — confirming the
   `doc/memcached.1` claim and the 1.5.6 release-note claim ("primarily disables the UDP protocol by
   default") directly in the code path that decides whether the listener exists at all, rather than
   relying on documentation that could have drifted from the binary.

**Caveat.** This is the one pair whose *own shipped default* removes it from the population before
the payload is ever relevant. The payload is correct and fully corroborated. What it can reach is
bounded to pre-2018 builds, distros that re-enable UDP, or an operator's deliberate `-U 11211` — the
same population ADR-0083's walk already named, now confirmed at the source rather than the man page.

---

## 6. What the table costs, and what carries it

This section rules. It does not reopen whether UDP ships — ADR-0083 already settled that and it is
not this ticket's to revisit — it prices the table that document deferred.

### 6.1 Per-row engineering cost

| Pair | Wire shape | State | Auth | Send-path confirmed on a real target? |
| --- | --- | --- | --- | --- |
| `69/udp` | One packet out (RRQ, ~10–70 B), one back (ERROR, ~10–20 B) | None | None | **Yes** — tftp-hpa |
| `137/udp` | One packet out (50 B fixed), one back (variable NBSTAT) | None | None | **Yes** — Samba `nmbd` |
| `138/udp` | One packet out (~80 B), one back (DATAGRAM ERROR, ~14 B) *if it arrives* | None | None | **No** — two independent Samba trees checked, no send path found in either |
| `623/udp` | One packet out (12 B), one back (28 B) | None | None | **Yes**, client-side (`ipmitool`) — spec itself unread |
| `11211/udp` | One packet out (~17 B), one back (variable) | None | None | **Yes** — memcached source, but the port is off by the project's own default since 1.5.6 |

Every row is a single unauthenticated request/reply datagram with no session state — there is no row
here that costs more than a small encoder and a reply matcher. **The dominant cost is not the wire
format, which is now fully authored and mostly corroborated. It is the receive-side plumbing already
named by ADR-0083 and confirmed twice over in this retrieval (§1, TFTP's ephemeral reply port): the
listener must bind one **unconnected** socket per probe and match replies by *content*, never by
source port, because a connected socket silently drops the TFTP row's reply. That is one shared
design constraint, paid once, not five times.

### 6.2 The one row that is a genuinely close call — `138/udp`

Ship the row as authored, or drop it from the table until a live Samba/Windows box confirms it?
Arguing both sides:

**For dropping it.** The retrieval was not a shallow miss — it checked both of Samba's independent
NBT implementations (the classic `nmbd` and the AD-DC `source4` server), found the wire constants
defined in two places but no call site that emits them, and found no Microsoft first-party text
affirmatively describing the behavior either (only the absence of a documented deviation, which is
weaker than a positive statement). A payload with no confirmed sender anywhere is close to
authoring a probe for a mechanism that may not exist in the deployed estate at all, and the whole
point of this ticket's evidence marks is to not paper over exactly this kind of gap.

**For shipping it.** Three reasons outweigh the gap. First, cost: the row is authored to the letter
of RFC 1002 §5.3.3's own pseudocode, so nothing is *invented* — if some implementation (a NAS
appliance's embedded NBT stack, an older Windows build, a third NBT implementation neither Samba tree
represents) does emit DATAGRAM ERROR, the row is already correct for it, at zero marginal engineering
cost over the other four. Second, the failure mode is not silent or misleading: if the real
population never emits this reply, the row degenerates to exactly what a payload-free probe already
does — `unanswered` → `not-evaluable`, ADR-0083's own documented default for these five pairs today.
Nothing gets *worse*. The row simply may not earn its keep, which is a fact this document states
rather than hides. Third, blocking it on a lab-verified live Samba/Windows box is a measurement this
ticket was not resourced to make and the map's own convention (measurement-offers.md's `[thin]`
mark, ADR-0032's per-row evidence standard) exists precisely so a thin row can ship *labelled* rather
than stall the table waiting for evidence that may cost more to obtain than the row is worth.

**Ruling: ship it, marked `[thin]` for the send-path, not `[owner]` or `[spec]`.** The cost of
carrying a possibly-inert row is zero. The cost of blocking the whole table on one unverifiable
protocol behavior is not. The revision price is stated rather than hidden, matching this repo's
standing convention: the day a live retrieval (or an operator report) confirms or refutes the
138/udp row, only that row's mark changes — nothing else in the table moves.

### 6.3 ADR-0015's unclosed question is not closed here, and two rows are where it will bite

The ticket's brief correctly flags that ADR-0015's *"a wrong dispatch guess fails safe for the data,
and never asks whether it is safe for the listener"* is still open, and this pricing exercise does
not close it — that is `listener-negotiation`'s obligation the day a wire prober is built, not this
table's. But authoring the five rows surfaces exactly where that question will have teeth:

- **`69/udp` is the row ADR-0083 already named as different in kind** — the payload is a TFTP **read
  request**, not a hello. This document's chosen filename is a random token designed never to
  collide with a real file, so the modal case is an immediate ERROR with no transfer. The residual
  case — the random name coincides with a file that exists — is not zero, and if it happens the
  listener would begin serving that file's contents to the prober. That is a real, if small, "is it
  safe for the listener" exposure this table does not eliminate, only makes unlikely.
- **`138/udp` carries a second, structural version of the same question.** RFC 1002's own model routes
  a correctly-addressed `DIRECT_UNIQUE` datagram into mailslot/browse dispatch on the receiving
  service. This table's destination name is chosen to be unowned specifically to avoid that path and
  land on the error case instead. Whether every real implementation's name-ownership check happens
  strictly *before* any dispatch side effect — rather than a wrong guess being partially processed —
  was not established by this retrieval and is exactly the shape of question ADR-0015 left open.

Neither observation changes this table's rows. Both are recorded so the successor that eventually
builds `datagram-outcome`'s sender does not have to re-derive that these two rows are where the
still-open dispatch-safety question concentrates.

### 6.4 What carries the table

**`datagram-outcome`** — the sixth leaf ADR-0083 specified and left unshipped. This table is that
leaf's **declared parameter**, in exactly the sense §1–§5 of
[`measurement-offers.md`](../spec/measurement-offers.md) use the term for the TLS candidate set, the
qtype set, the ALPN list, the EDNS option set and the DNS transport policy: a list carried by one
leaf, revisable independently of the ADR that ruled the leaf exists, with its own per-row evidence
marks rather than one blanket attestation. It stays in this research document rather than migrating
into `measurement-offers.md` itself, for the same reason ADR-0083 gave for not shipping the leaf: the
table only becomes a **live** declared parameter — carried by a `Batch`, subject to `measurement-offers.md`'s
own build-time offerability gate — the day `datagram-outcome` ships. Until then this document is
where the successor reads it from, and moving it into `measurement-offers.md` early would misstate an
unshipped leaf's parameter as one already carried on the wire.

### 6.5 The verdict, stated once

**The table costs almost nothing to build and nothing new to ship.** Four of five rows are authored
to specification and corroborated against a real, current, widely-deployed implementation
(`[owner]`). The fifth (`138/udp`) is authored to specification but its send-path is unconfirmed after
a genuine two-implementation search and ships marked `[thin]` rather than blocking the table. No row
needs session state, credentials, or more than a small encoder/matcher pair. The one shared
engineering cost — an unconnected receive socket matching by content, not source port — was already
named by ADR-0083 and is paid once. **It is carried by `datagram-outcome`, the already-specified,
still-unshipped sixth leaf, as that leaf's declared parameter**, alongside the operator-invisible
per-host rate/retry parameters ADR-0083 §13.5 already assigned it. Building this table does not turn
UDP on — that remains ADR-0083's aperture decision, untouched — and does not close ADR-0015's
listener-safety question, which stays open and now has two named rows where it will matter most,
for whoever builds the sender.

---

## Sources

**Specifications:**
- RFC 1350 — The TFTP Protocol (Revision 2)
- RFC 1002 — Protocol Standard for a NetBIOS Service on a TCP/UDP Transport: Detailed Specifications
- DMTF DSP0136 — Alert Standard Format (ASF) Specification v2.0, 23 April 2003
  (https://www.dmtf.org/sites/default/files/standards/documents/DSP0136.pdf)
- memcached `doc/protocol.txt` and `doc/memcached.1` (github.com/memcached/memcached)

**Implementation source read:**
- tftp-hpa `tftpd/tftpd.c` — `nak()`, `validate_access()` — https://sources.debian.org/src/tftp-hpa/5.2-4/tftpd/tftpd.c/
- Samba `source3/nmbd/nmbd_incomingrequests.c` — `process_node_status_request()` — https://github.com/samba-team/samba/blob/master/source3/nmbd/nmbd_incomingrequests.c
- Samba `source3/nmbd/nmbd_namelistdb.c` — `find_name_on_subnet()`, `add_samba_names_to_subnet()` — https://raw.githubusercontent.com/samba-team/samba/master/source3/nmbd/nmbd_namelistdb.c
- Samba `source3/nmbd/nmbd_packets.c` — `reply_netbios_packet()` — https://raw.githubusercontent.com/samba-team/samba/master/source3/nmbd/nmbd_packets.c
- Samba `source3/nmbd/nmbd_incomingdgrams.c` — checked for a DATAGRAM ERROR send path (none found) — https://github.com/samba-team/samba/blob/master/source3/nmbd/nmbd_incomingdgrams.c
- Samba `source4/nbt_server/dgram/request.c` — `dgram_request_handler()`, checked for a DATAGRAM ERROR send path (none found) — https://raw.githubusercontent.com/samba-team/samba/master/source4/nbt_server/dgram/request.c
- Samba `librpc/idl/nbt.idl` — `DGRAM_ERROR`, `DGRAM_ERROR_NAME_NOT_PRESENT`, `DGRAM_DIRECT_UNIQUE` constants
- Samba `source3/include/nameserv.h` and `source3/include/smb_macros.h` — `DGRAM_ERROR`, `DGRAM_DIRECT_UNIQUE` constants
- `ipmitool` `src/plugins/lanplus/lanplus.c` — `ipmiv2_lan_ping()`, `ipmi_handle_pong()` — https://raw.githubusercontent.com/ipmitool/ipmitool/master/src/plugins/lanplus/lanplus.c
- `ipmitool` GitHub issue #268 — captured Presence Ping packet bytes — https://github.com/ipmitool/ipmitool/issues/268
- memcached `memcached.c` — `build_udp_header()`, `settings_init()`, `main()` argument parsing (`case 'U'`) — https://github.com/memcached/memcached/blob/master/memcached.c

**First-party vendor documentation:**
- MS-NBTE: NetBIOS over TCP (NBT) Extensions, Microsoft Open Specifications overview — https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-nbte/3461cfa8-3d28-4fa3-8163-131bf1046fa3
- MS-NBTE Appendix A: Product Behavior — https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-nbte/6dbf0972-bb15-4f29-afeb-baaae98416ed

**Attempted and not usable (recorded so the gap is not silently repeated):**
- `intel.com`'s IPMI 2.0 specification page — HTTP 403, confirmed still blocked
- `img1.wsimg.com/.../ipmi_specification_v2.0.pdf` — fetched and inspected. A 2-page `wkhtmltopdf`
  rendering, not the IPMI 2.0 spec. Discarded
- `scribd.com` and `studylib.net` IPMI 2.0 spec mirrors — located, not read (429/rate-limited on fetch)
- GitHub code search on samba-team/samba — blocked without authentication. Substituted with direct
  `curl` of raw file contents and `gh search code`, both of which succeeded and are cited above
