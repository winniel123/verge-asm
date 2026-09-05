# ADR-0196: No outbound HTTP client this product builds follows a redirect, and an unfollowed 3xx admits nothing

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1315 ADR gaps: internal/queue (2/7)](https://github.com/winniel123/verge-asm/issues/1315), gap 1
- **PR that deleted the comment:** [#1314](https://github.com/winniel123/verge-asm/pull/1314)
- **Bounded by:** [ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md). That ADR rules **how a client arrives** — a runtime parameter the caller supplies, never a build tag. Its §6 sets the transport's configuration outside its own scope and says redirect policy is "ruled elsewhere". This ADR is that elsewhere. The two are complementary and neither contains the other
- **Sibling of, and not ruled by:** [ADR-0148](./0148-a-measurement-leaf-sends-an-authored-fixed-request-and-never-mutates-remote-state-or-follows-a-link.md). That ADR rules that a measurement leaf sends one authored request and follows no link. Its ground is **measurement fidelity**: following a 3xx moves the `status` and the `title` the leaf decides, so it stops measuring the asset being tracked. It reaches `internal/measure` alone
- **Sibling of, and not ruled by:** [ADR-0119](./0119-report-delivery-to-a-channel-is-a-link-only-ready-message-in-its-own-corpus.md). That ADR rules that report delivery rides the shared signed-POST transport, and its Consequences state that the SSRF guard, the no-bearer rule and the redirect refusal are enforced once, in `delivery.SendSigned`. Its ground is **the operator's declared attack surface**. It reaches the two delivery callers alone
- **Does not rest on** [ADR-0027](./0027-a-source-may-admit-without-observing.md). The deleted comment cited `ADR-0027 §7`. That ADR has no §7 and rules nothing about redirects. §Context measures the citation

## Context

`internal/queue/crtsh.go:78` carried this, until [#1314](https://github.com/winniel123/verge-asm/pull/1314) deleted it:

```go
// Redirects are not followed: a 3xx returns its own response unfollowed,
// so a compromised or MITM'd source cannot bounce the fetch to an
// arbitrary internal host (blind SSRF — IMDS at 169.254.169.254 or an
// RFC-1918 address). The caller treats any non-200 as transient failure
// and admits nothing, so an unfollowed 3xx is handled like any other
// non-200 (ADR-0027 §7). Same idiom as httpexchange.NetExchanger.
```

The compressed survivor sits at `internal/queue/crtsh.go:51`, beside the `CheckRedirect` field it
explains. It is uncited, because the citation the deleted block carried is dead.

### The citation was dead, and that is part of why the rule is unwritten

`ADR-0027 §7` does not exist. [ADR-0027](./0027-a-source-may-admit-without-observing.md) has no
numbered sections at all. Its headings are `## Context`, `## Decision`, `## Rationale`, five `###`
headings under Rationale, `## Consequences` and `## Alternatives rejected`. A grep of that file for
`redirect`, `3xx`, `non-200`, `SSRF` and `IMDS` returns zero hits. The ADR rules that a source may
admit a subject without observing it, and that a decoder translates shape and never fact. It rules
nothing about a transport.

A reader who followed the citation found an ADR about admission and could reasonably conclude the
redirect rule was ruled there. It was not ruled anywhere. This is the failure the citation was
supposed to prevent and instead produced.

### ADR-0140 §6 defers to a rule that did not exist

[ADR-0140](./0140-a-network-seam-is-a-runtime-parameter-the-caller-supplies-never-a-build-tag-and-never-a-hardcoded-client.md)
§6 reads:

> **The transport's configuration.** Timeouts, redirect policy and the `custody` dial guard are ruled
> elsewhere and are untouched here.

For the `custody` dial guard that is true — [ADR-0079](./0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)
and [ADR-0121](./0121-the-operator-declared-recursive-resolver-is-trusted-and-exempt-from-the-discovered-authority-egress-guard.md)
rule it. For redirect policy it was true of two surfaces and false of the rest. ADR-0140 §4 then
names `NewHTTPCTFetcher`'s redirect refusal by name, as an existing fact, without a rule to point at.

### Every outbound `http.Client` this product builds, and what is written about each

Seven production sites construct an `http.Client`. Three refuse a redirect and four do not.

| Site | What it reaches | Refuses a 3xx | Written where | Pinned by a test |
| --- | --- | --- | --- | --- |
| `internal/measure/httpexchange/exchange.go:157` | a measured `Endpoint` | Yes | [`v1-spec.md`](../spec/v1-spec.md) §3.3 line 186, [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md) line 372, ADR-0148 | `TestRedirectIsRecordedButNotFollowed` (`internal/measure/httpexchange/leaf_test.go:118`) |
| `internal/delivery/runner.go:52` | an operator-declared channel | Yes | [`notification-channels.md`](../spec/notification-channels.md) §4, ADR-0119 | `TestHTTPDoerRefusesRedirects` (`internal/delivery/delivery_test.go:219`) |
| `internal/queue/crtsh.go:48` | crt.sh and Cert Spotter | Yes | **Nowhere** — this ADR | `TestHTTPCTFetcherDoesNotFollowRedirect` (`internal/queue/crtsh_test.go:75`) |
| `internal/release/fetcher.go:30` | `api.github.com`'s release feed | **No** | Nowhere | None |
| `cmd/web/handlers.go:284` | ARIN RDAP, CAIDA, the RIR delegated stats | **No** | Nowhere | None |
| `cmd/web/handlers.go:285` | an OIDC issuer's discovery and token endpoints | **No** | Nowhere | None |
| `cmd/web/main.go:226` | this process's own `/healthz` over loopback | **No** | Nowhere | None |

The three refusing sites each spell the same one line, `CheckRedirect` returning
`http.ErrUseLastResponse`, and each arrived independently. Nothing made the fourth, fifth, sixth or
seventh adopt it.

**The three written statements do not reach the other four sites.** ADR-0148's ground is measurement
fidelity, and its scope is `internal/measure`'s leaves. ADR-0119's ground is the operator's declared
attack surface, and its scope is `SendSigned`'s two callers. A CT source fetch is neither a
measurement leaf nor a delivery. A release-feed poll, an RDAP lookup and an OIDC token exchange are
none of the three.

### The hazard is the same at every site, and it is not hypothetical

A redirect hands a third party the power to choose our next request's destination **after** we have
already decided the first destination is safe to reach. The decision and the request are separated.
On a host with a link-local metadata service that is a blind SSRF into `169.254.169.254`.
[ADR-0079](./0079-authority-presupposes-denotation-a-non-globally-reachable-address-is-probed-only-inside-a-declared-realm.md)
line 349 records what is at stake there: the product "retrieves cloud instance metadata into
`http-identity` on an instance the map calls a high-value target". That is the one place the product
reaches that address on purpose. On the OIDC client the next request carries the client secret in a
form body.

`TestHTTPCTFetcherDoesNotFollowRedirect` states the hazard as an executable case: a `httptest`
redirector answers `302` to `<target>/latest/meta-data/`, and the test fails if the next hop is
reached or if `pwned.example.com` appears in the body.

### No caller needs a redirect

Every one of the seven callers already treats a non-200 as failure. `internal/queue/crtsh.go:162`
fails the job on `ferr != nil || status != http.StatusOK` and routes to `retryOrDeadLetterCT`, so no
name is admitted. `internal/release/fetcher.go:52` returns an error on any non-200.
`internal/proposer/caida.go:62,83` and `internal/proposer/arin.go:98,102,132` do the same.
`cmd/web/main.go:233` fails the health check on a non-200. The OIDC client is handed to `go-oidc`
and `golang.org/x/oauth2`, both of which treat a non-200 discovery or token response as an error.

No production endpoint the product reaches is a redirector. `NewARIN` is constructed with
`https://rdap.arin.net/registry`, the authoritative ARIN endpoint, and not with an RDAP bootstrap
service that answers by redirect.

## Decision

> **No outbound HTTP client this product builds follows a redirect. Every `http.Client` a verge-asm
> package constructs sets `CheckRedirect` to `http.ErrUseLastResponse`, without exception and without
> regard to what the client is for. The unfollowed 3xx surfaces to the caller as its own response.
> The caller treats it as an ordinary non-200: the job or the request fails, and nothing is admitted
> into the estate.**

### 1. The rule is stated over every outbound client, and there is no source-fetcher exception

The scope call, stated plainly: **the rule binds every outbound HTTP client this product builds, and
not only the third-party source fetchers.**

The hazard is a property of the redirect, not of the caller. A redirect is a third party choosing our
next request's destination after we decided the first was safe. That is the SSRF and IMDS hazard in
its exact shape, and it does not become safe because the caller happens to fetch a CT log today and
a release feed tomorrow.

A narrow rule — *a source fetcher refuses a redirect* — has to be re-argued at every new client, by a
session that must first decide whether its client is a source fetcher. That question has no
principled answer. Is a release-feed poll a source fetch? Is an RDAP lookup? Both read a third
party's data over HTTP, and neither admits a subject. The classification does no work except to
create somewhere for the next client to fall through, and a client that falls through is a client
with the hazard.

The broad rule costs nothing to hold. §Context measures it: no shipped caller needs a redirect, and
the change at each non-conforming site is one field on a struct literal.

### 2. An unfollowed 3xx is an ordinary non-200, and a non-200 admits nothing

`http.ErrUseLastResponse` is not an error path. The client returns the 3xx response itself, with its
status, its headers and its body. The caller therefore needs no redirect-specific branch, and none of
the seven callers has one.

The second half is the one that matters for the estate. A non-200 is a **failure to observe**, never
an observation. A CT fetch that answers 302 admits no name, exactly as a fetch that answers 404
admits none. This preserves the rule
[ADR-0027](./0027-a-source-may-admit-without-observing.md) does rule — a source admits or it does
not — by keeping a transport outcome out of the admission path entirely.

The `Location` header is not discarded. Where a surface records it, it is recorded as a fact about
the response and never acted on. ADR-0148 §Context states this for `http-exchange`, where the
`Location` is the finding.

### 3. The rule binds the construction site, not the caller

The obligation lands on the code that builds the `http.Client`, because that is the only place the
policy is expressible. A caller handed a `Doer` cannot set `CheckRedirect` and must not try.

This is the bound ADR-0140 §4 already draws for the same family of concerns: a production adapter is
permitted to construct its own transport, and **an adapter carrying its own logic owes its own test
against a loopback server.** A redirect refusal is such logic. The three conforming sites each carry
that test, at `internal/queue/crtsh_test.go:75`, `internal/delivery/delivery_test.go:219` and
`internal/measure/httpexchange/leaf_test.go:118`. A new client owes one.

### 4. The three written statements stand beside this rule and are not withdrawn

[`v1-spec.md`](../spec/v1-spec.md) §3.3, [`packaging-and-configuration.md`](../spec/packaging-and-configuration.md)
and ADR-0148 rule the measurement leaf on a **measurement-fidelity** ground. Following a 3xx moves
the `status` and the `title` `http-exchange` decides, so redirect-following is a declared parameter
of that leaf, valued at *not followed*, and only a code change can move it. That ground is not this
ADR's ground, and this ADR does not supply it. The leaf would still refuse a redirect if no security
hazard existed.

[`notification-channels.md`](../spec/notification-channels.md) §4 and ADR-0119 rule the delivery
client on an **operator's-declared-attack-surface** ground: a followed 3xx would move the operator's
attack surface to a host they never declared. That ground is adjacent to this one and is not the
same. It is about a destination the operator chose. This ADR is about a destination nobody chose.

**None of the three is withdrawn, amended or superseded.** Each states more about its own surface
than this ADR does, on a ground this ADR does not carry.
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) is not
engaged, because no mechanism any of them specifies stops existing. A reader arriving at any of the
three, alone and in the present tense, is told something that is true.

### 5. What this rule does not reach

- **A redirect the product *sends*.** `http.Redirect` in `cmd/web` is a response, not a request.
  [ADR-0130](./0130-scroll-restore-is-hardened-by-a-same-url-prg-plus-full-url-key-contract.md) and
  [ADR-0157](./0157-a-receipt-on-a-self-refreshing-surface-rides-the-server-side-flash-never-the-toast-query.md)
  rule the PRG contract. This ADR rules what a client of ours follows, and a redirect we issue is
  followed by the operator's browser, not by us.
- **The OIDC front-channel.** [ADR-0112](./0112-single-sign-on-is-admitted-as-verified-oidc-never-header-trust.md)
  §2 has the app redirect the operator's browser to an identity provider. That is a front-channel
  redirect the browser follows. This ADR reaches only the OIDC **back-channel** client at
  `cmd/web/handlers.go:285`, which performs discovery and the token exchange.
- **The `custody` dial guard.** `delivery.NewHTTPDoer`'s `Dialer.Control` refuses a
  non-globally-reachable destination after resolution. That is a separate defence, ruled by
  ADR-0079 and ADR-0121, and it is not required by this ADR at any site. A redirect refusal and a
  dial guard fail differently and neither substitutes for the other.
- **A test client.** `cmd/web/auth_test.go:44` sets `CheckRedirect` to inspect a 303 the console
  issues. A test asserting on our own responses is not an outbound client.
- **Whether a call is made at all.** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md)
  §2 keeps the release check opt-out and air-gap-safe. This ADR rules how the client behaves once the
  call is made.

## Consequences

- **This ADR changes no Go code.** The three conforming clients are unchanged and already tested.
- **Four clients are known violations and are not fixed here.** Each is a one-field change at a
  struct literal, and each ships as its own ticket.
  - `internal/release/fetcher.go:30`. **ADR-0140's Consequences already name this file as a known
    violation** on a different ground — `NewHTTPFetcher` neither accepts a `Doer` nor carries a test.
    One ticket can discharge both, because taking a `Doer` moves the construction to a caller that
    must then set the policy.
  - `cmd/web/handlers.go:284`, the proposer registry. The ticket must confirm that no configured
    RDAP or delegated-stats endpoint answers by redirect before the change lands.
  - `cmd/web/handlers.go:285`, the OIDC back-channel. The client is handed to `go-oidc` and
    `golang.org/x/oauth2`. The ticket must confirm both libraries surface an unfollowed 3xx as an
    error rather than as a successful empty response.
  - `cmd/web/main.go:226`, the loopback health probe. It reaches only this process's own listener,
    which is why it is last and why it is still in. Deciding that this one client is safe to exempt
    is exactly the per-client re-argument §1 refuses, and the exemption would be worth less than the
    one line it saves.
- **A new outbound client owes a loopback test.** §3 fixes this at the construction site, on the bound
  ADR-0140 §4 already draws. Three such tests exist and are the model.
- **Nothing enforces this.** No check fires on an `http.Client` literal without a `CheckRedirect`
  field, so review carries the rule. A `go vet` or `gosec` rule is rejected in the table below.
- **No `CONTEXT.md` term moves.** Redirect policy is a transport property, not a domain term.
- **`internal/queue/crtsh.go`'s survivor gains a citation.** It was uncited, because the citation it
  carried was dead. The edit is applied at that site and cites this ADR's §1.
- **ADR-0140 §6 gains a pointer.** Its "ruled elsewhere" is now true of redirect policy. The edit is
  applied at ADR-0140's own §6. It is a cross-reference and not an
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) withdrawal,
  because §6 specifies no mechanism that stops existing.
- **The triage record's test claim was wrong and is corrected here.** [#1315](https://github.com/winniel123/verge-asm/issues/1315)
  states that only the delivery client carries a test pinning the behaviour. Three clients carry one.
  #1315's "Out of scope" therefore excludes two tests that already exist.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Rule the refusal for third-party source fetchers alone** | The class has no boundary in this code. A release-feed poll and an RDAP lookup both read a third party's data over HTTP and neither admits a subject, so a session classifying a new client gets no answer from the term. Every client that falls outside the class carries the full hazard, and four of the seven already have |
| **Rely on `ADR-0027 §7`, as the deleted comment did** | It does not exist. ADR-0027 has no numbered sections and zero occurrences of `redirect`, `3xx`, `SSRF` or `IMDS`. It rules that a source may admit without observing. A reader who followed the citation would conclude the rule was ruled and stop looking, which is what happened |
| **Widen ADR-0148 to cover every outbound client** | Its Decision is about what a **measurement leaf** sends, and its ground is that following a 3xx moves the `status` and `title` the leaf decides. That ground does not exist for a CT fetch, a release-feed poll or a token exchange — none of them decides a measured value. Widening it would put a security rule inside a document about measurement fidelity, and would make `http-exchange`'s declared-parameter machinery apply to clients that have no declared parameters |
| **Add a clause to ADR-0119** | ADR-0119 rules report delivery and its transport sharing. Its redirect sentence is a Consequence of `SendSigned` being one place, not a decision about clients in general. A rule about seven clients filed under an ADR about report delivery is unfindable from six of them |
| **Rely on the `custody` dial guard instead** | Only `delivery.NewHTTPDoer` carries it, and it is a different defence with a different failure mode. It refuses a non-globally-reachable **destination**; a redirect to an attacker-controlled globally-reachable host passes it and still hands a third party our next request. The two do not substitute |
| **A `gosec` or `go vet` rule that fails an `http.Client` literal with no `CheckRedirect`** | Not decidable from the literal. `internal/release/fetcher.go:30` and `cmd/web/main.go:226` are composite literals in one line, `internal/delivery/runner.go:52` sets a `Transport` built four statements earlier, and a client could equally be built by a helper and returned. A check that fires on the literal shape misses the helper and fires on a test client, which trains reviewers to suppress it. `gosec` runs `-severity high -confidence high` in CI, and this rule is neither |
| **Fix the four non-conforming clients on this ADR's own branch** | It touches `internal/release`, `internal/proposer`'s composition root and `cmd/web/main.go`, and the OIDC one needs a behavioural check against two third-party libraries. That is a production change with its own review, buried under a docs review |
| **Exempt the loopback health probe at `cmd/web/main.go:226`** | It reaches this process's own listener, so the exemption looks free. It is not: it makes the rule *no outbound client except where the destination is one we control*, and "a destination we control" is the judgement every one of the four violating sites would also claim. The exemption costs a re-argument at every future client and saves one line |
| **Exempt the OIDC back-channel, since an issuer is operator-declared** | The operator declares the **issuer**, not the hosts the issuer's 3xx names. The token request carries the client secret, so this is the one client where following a redirect leaks a credential rather than only a request. It is the least exemptible of the four |
| **State the rule in `CONTEXT.md` instead** | `CONTEXT.md` holds the domain vocabulary. A redirect is not a subject, a facet, a source or an act. It contains no occurrence of `redirect`, `SSRF` or `IMDS` today, and adding one would put a transport rule in the glossary a reader consults for the model |
