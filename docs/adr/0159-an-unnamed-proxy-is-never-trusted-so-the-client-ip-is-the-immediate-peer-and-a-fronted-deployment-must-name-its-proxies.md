# ADR-0159: An unnamed proxy is never trusted, so the client IP is the immediate peer, and a fronted deployment must name its proxies

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1371 ADR gaps: cmd/web production 17/17](https://github.com/winniel123/verge-asm/issues/1371), gap 1
- **PR that deleted the comment:** [#1373](https://github.com/winniel123/verge-asm/pull/1373). A second statement of the same rule left `cmd/web/main.go` in [#1372](https://github.com/winniel123/verge-asm/pull/1372). #1371 recorded that statement so the rule did not fall between the two sweep tickets
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Bounds:** [`running.md`](../guides/running.md)'s *Networking and security posture* proxy bullet and [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §5.1's environment enumeration, at their own sites, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [`v1-spec.md`](../spec/v1-spec.md) §4.3, which refuses reverse-proxy forward-auth. That clause rules **identity**. This ADR rules the **rate-limit key**, and the two never meet

## Context

[`cmd/web/clientip.go`](../../cmd/web/clientip.go):11 carried this, until #1373 deleted it:

```go
// trustedProxies is the set of proxy addresses whose X-Forwarded-For header web
// trusts when deriving a client IP for the rate-limit key ONLY (#738). It is
// parsed from VERGE_TRUSTED_PROXIES, a comma-separated list of IPs and/or CIDRs
// (e.g. "10.0.0.0/8, 192.0.2.7"). Empty (the default) means no proxy is trusted:
// the client IP is always the immediate RemoteAddr and no forwarding header is
// consulted — identical to the pre-#738 behaviour.
```

[`cmd/web/main.go`](../../cmd/web/main.go) stated the same rule a second time, in prose, until #1372
deleted it:

```go
// rate-limit client IP (#738). Behind a TLS-terminating proxy this keeps the
// per-IP limiter per-client instead of keying the whole console to the proxy
// address; unset (a direct-facing deployment) it is unused. A malformed spec is a
```

**The rule is unwritten everywhere else.** `VERGE_TRUSTED_PROXIES` appears in exactly two places in
the tree, both in code: `cmd/web/main.go:125` and `:127`. `docs/spec/`, `docs/adr/`,
`docs/research/`, `docs/guides/`, `CONTEXT.md` and `.env.example` hold zero occurrences.

### The citation is dead, so §8.3 does not suppress the record

The deleted block cited `#738` five times. `gh api repos/winniel123/verge-asm/issues/738` returns
HTTP 410, *"This issue was deleted"*. [`comment-policy.md`](../spec/comment-policy.md) §8.3 rules
that a deleted issue does not suppress a record. Two of the five uses were time markers
(*"identical to the pre-#738 behaviour"*), which date a change against a record nobody can read.

`v1-spec.md` §4.3 is the nearest live source and it does not state this rule. It refuses
header-trust for **identity**. That clause is already stated by survivors at
[`cmd/web/handlers.go`](../../cmd/web/handlers.go):234 and [`cmd/web/auth.go`](../../cmd/web/auth.go):47.
Under §8.3, a live and on-topic source suppresses only where it states the rule.

### What the code does today

[`cmd/web/clientip.go`](../../cmd/web/clientip.go):56 returns the peer address whenever any of three
conditions holds. The peer address does not parse, or the trusted set is empty, or the peer is not
in the trusted set. Only after all three fail does the function read `X-Forwarded-For`. It then
walks the chain from the right and stops at the first hop it does not vouch for.

`parseTrustedProxies` returns an error on a malformed IP or CIDR. `cmd/web/main.go:127` turns that
error into `log.Fatalf`, so a typo fails the boot instead of quietly trusting a smaller set.

### The derived IP reaches one consumer

`clientIP` has one caller, `loginIPKey`. `loginIPKey` has two callers, both in
[`cmd/web/auth.go`](../../cmd/web/auth.go) — the password submit at `:245` and the TOTP submit at
`:292`. Both use the value as one of two rate-limiter keys. No other code path reads it. No handler,
session check or audit write derives anything from a forwarding header.

### The measured cost of getting the set wrong

[`cmd/web/ratelimit.go`](../../cmd/web/ratelimit.go) locks on **either** key. Five failures inside
five minutes lock a key, and the lockout doubles from five minutes up to a ceiling of one hour.
The 15-minute release ceiling applies only to keys with the `acct:` prefix. An `ip:` key has no
release ceiling.

So a deployment behind an unnamed proxy gives every client the single key `ip:<proxy address>`. Five
failed logins by anyone then lock **every account on the instance**, and repeated failures hold the
lock open. The surviving comment at `cmd/web/clientip.go:83` states this consequence and does not
state the default that produces it.

The opposite error costs the other way. A trusted set wider than the real proxy fleet lets a client
inside that range choose its own `X-Forwarded-For` value. The per-IP half of the limiter then binds
that client to nothing.

## Decision

> **`web` trusts no proxy unless the operator names it in `VERGE_TRUSTED_PROXIES`. When that
> variable is unset or empty, the client IP is the immediate peer from `RemoteAddr` and `web` reads
> no forwarding header at all. An operator who puts a reverse proxy in front of `web` must name
> every hop between the client and `web` in that variable. The derived client IP keys the login rate
> limiter and reaches nothing else. It never reaches identity, authorization or an audit record.**

### 1. The default is deny, and an empty variable is a complete configuration

An unset `VERGE_TRUSTED_PROXIES` states a direct-facing deployment. It is not a missing value and
`web` never warns about it, because `web` cannot tell the two apart.

The refused default is the common one. Most reverse-proxied applications read the immediate peer's
`X-Forwarded-For` and take a value from it. On a direct-facing instance that default hands the
rate-limit key to the attacker, because the header is caller-supplied and free to forge. A forged
key defeats the per-IP half of the limiter completely. The deny default fails the other way instead,
and it fails loudly: the operator sees a shared lockout, not a silently absent control.

The party who knows the topology is the operator, and the operator is the only party who can know
it. So the burden sits on the deployment that is fronted, never on the deployment that is not.

### 2. What an operator must do to put a real proxy in front of `web`

Three steps, and all three are required.

1. **Set `VERGE_TRUSTED_PROXIES`** to a comma-separated list of the IP addresses or CIDRs of every
   hop between the client and `web`. For example, `10.0.0.0/8, 192.0.2.7`. Name the proxy's address
   as `web` sees it, which is the address the proxy connects **from**, not the address it listens on.
2. **Set `VERGE_SECURE_COOKIES=true`**, which [`running.md`](../guides/running.md) already requires
   for a TLS-terminating proxy.
3. **Make the proxy set or append `X-Forwarded-For`.** `web` reads that header and no other. A proxy
   that sends only `Forwarded` or only `X-Real-IP` leaves `web` with the peer address.

Name the set exactly. A set narrower than the real fleet keys every client to a proxy address, which
is §Context's whole-instance lockout lever. A set wider than the real fleet lets a client in the
extra range choose its own key. A malformed entry fails the boot rather than shrinking the set in
silence.

### 3. The chain is walked from the right, and the first unvouched hop wins

`web` reads the `X-Forwarded-For` entries right to left and returns the first entry it does not
trust. A hop that `web` does not trust cannot have vouched for anything to its left, so a prefix an
attacker prepends never wins. A malformed entry is not a trusted proxy, so the walk stops at it. If
every entry is trusted, `web` returns the peer address.

The code states this limb already, at `cmd/web/clientip.go:67` and `:71`. It is written here so the
Decision is applicable without reading the code.

### 4. The scope is the rate-limit key, and the scope does not grow by default

A forwarding header is caller-supplied. This ADR admits it for one purpose only. The worst outcome
of a forged value there is a rate-limit key. An attacker could already vary that key by changing
source address.

`v1-spec.md` §4.3 refuses header-trust for identity, and that refusal is untouched here. A future
consumer of `clientIP` is a new decision — an audit column, a per-IP allow list, an abuse counter.
Each needs a ruling against this ADR before it ships. A new caller is not a mechanical change.

### 5. What this rule does not reach

- **The behaviour of the proxy itself.** `web` cannot check that a named proxy strips an inbound
  `X-Forwarded-For`. Naming an address is a statement that the operator has configured it correctly.
- **Any header other than `X-Forwarded-For`.** `Forwarded`, `X-Real-IP` and the PROXY protocol are
  unimplemented, and this ADR neither admits nor refuses them.
- **`VERGE_SECURE_COOKIES` and `VERGE_EXTERNAL_URL`.** They are separate proxy-facing knobs with
  their own grounds. §2 lists the cookie flag as an operator step and rules nothing about it.

## Consequences

- **[`running.md`](../guides/running.md) gains a `VERGE_TRUSTED_PROXIES` row and one added
  instruction** on the *Networking and security posture* proxy bullet. Read alone, that bullet told
  an operator to front `web` with a proxy and to set one variable, which is the configuration §2
  rules incomplete. Marked at the bullet per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md).
- **[`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §5.1 gains a bound.**
  Its sentence enumerates the environment as the database URL and credential, the listen address and
  `VERGE_SETUP_TOKEN`. Read alone and in the present tense, that list refuses this variable a home.
- **`cmd/web/clientip.go` and `cmd/web/main.go` gain this ADR's citation**, one reason clause at
  each site the deleted blocks stood beside.
- **No production behaviour changes.** The code already has the shape this ADR states, and this ADR
  changes no logic.
- **Nothing enforces this, and nothing can.** `web` cannot detect that a proxy sits in front of it.
  An unset variable on a fronted deployment and an unset variable on a direct-facing deployment are
  identical at run time. The guide carries the rule, and a shared lockout is the only signal.
- **`#738`'s ground is unrecoverable.** The issue is deleted. This ADR restates the rule from the
  code as it stands and from the two deleted blocks quoted above. It does not reconstruct why #738
  chose the deny default, and no session should cite #738 for it.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** A trusted proxy is a deployment term, not a
  term in the measurement domain.
- **`.env.example` gains nothing.** It carries the required credential and the external-database
  block. It does not catalogue the optional `web` knobs, and `VERGE_SECURE_COOKIES` is absent from
  it for the same reason. `running.md` is the catalogue.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Trust the immediate peer's `X-Forwarded-For` by default** | This is what most reverse-proxied deployments assume, and it is the reason this ADR exists. On a direct-facing instance every client is the immediate peer, so every client picks its own rate-limit key by sending a header. The per-IP half of the login limiter then binds nothing at all, and [#322](https://github.com/winniel123/verge-asm/issues/322)'s online-guessing bound is gone. A control that is silently absent is worse than one that is loudly wrong |
| **Trust private ranges by default — loopback, RFC 1918, and the container network** | It guesses the topology instead of reading it. `web` publishes `8080` and an operator may reach it directly from a LAN, which is an RFC 1918 source that is not a proxy. The guess is also unfalsifiable at run time, so a wrong guess is invisible until someone forges a header. It buys convenience for one deployment shape and pays for it with an attacker-chosen key in another |
| **Take the leftmost `X-Forwarded-For` entry, the conventional "original client"** | The leftmost entry is the one furthest from `web` and the one nothing vouched for. An attacker prepends any value they want. §3's right-to-left walk exists because trust decays from the peer outward, and the leftmost reading inverts that |
| **Warn at boot when `VERGE_TRUSTED_PROXIES` is empty** | `web` cannot tell a direct-facing deployment from a fronted one, so the warning fires on every correct direct-facing boot. A log line that is wrong most of the time trains an operator to ignore it. The deployment that needs the warning is then the one deployment that gets no signal from it |
| **Detect the proxy at run time from the presence of `X-Forwarded-For`** | The header is caller-supplied. Detecting a proxy from it means an attacker enables proxy trust by sending one header. That is the default this ADR refuses, reached by a longer route |
| **Skip a malformed entry in the list and keep the rest** | A typo would then shrink the trusted set in silence, and a shrunk set is §Context's whole-instance lockout lever. The operator gets no signal, because the boot succeeds. The code already fails the deployment at `cmd/web/main.go:127` and this ADR keeps that |
| **Let the derived client IP key identity, an allow list or an audit column too** | `v1-spec.md` §4.3 refuses header-trust for identity, and the value's whole safety argument is that a forged one costs only a rate-limit key. Widening the reach without widening the check is the bypass class §4.3 names |
| **State the rule in `v1-spec.md` §4.3 instead** | §4.3 rules identity and access. Putting a rate-limiter keying rule inside it invites the reading that the two are one control, which is the confusion §4.3 exists to prevent. §4.3 stays the identity clause and this ADR cites it |
| **State it only in [`running.md`](../guides/running.md)** | The guide tells an operator what to set. It does not carry why the default denies, why the set must be exact in both directions, or why the value may not reach identity. A later session proposing a friendlier default would find an instruction and no position to argue against |
