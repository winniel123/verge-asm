# Who owns a claim about certificate renewal timing

Research ticket [#67](https://github.com/winniel123/verge-asm/issues/67) — wayfinder research for
the verge-asm v1 spec.

**Question.** Is `certificate-expiring`'s **`N = 30 days`** attested by a source that owns the claim
— and if no owner can be found, does the number move or does it stay disclosed?

**Framing.** [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) ruled
that [#21](https://github.com/winniel123/verge-asm/issues/21)'s **attestation gate** binds every
project-authored table that asserts something about the **world**, and that `N` is one of exactly
three such tables in v1. It is a one-row table.
[#60](https://github.com/winniel123/verge-asm/issues/60) fixed the row's value at 30 and recorded
its reason as **stated but unverified against a source**. This note is that retrieval, and
[#37](https://github.com/winniel123/verge-asm/issues/37)'s precedent is why it had to be one: *a row
may not move on a re-reading of text already held. A verdict changes on retrieval.*

Three constraints bind before any evidence is gathered, and they are the ticket's own.

1. **The number ships either way.** Gate 2 is not a shipping gate. This note does not block
   [#12](https://github.com/winniel123/verge-asm/issues/12).
2. **`N` may not become a dial.** #60 killed a **per-install** `N` on three independent grounds and
   this note does not reopen it. §12 shows the ruling below respects all three.
3. **Frequency is not a position.** *Most clients renew at 30 days* is a frequency claim and §2.5 of
   [`sensitive-ports.md`](./sensitive-ports.md) excludes frequency from evidence. Keeping the
   frequency sentence apart from the position sentence is most of the work below, and it is where
   #60's rationale fails — §8.

---

## 1. Summary

| Decision | Answer |
|---|---|
| Does an owner exist for a claim about renewal timing | **Yes, and there are two of them, each owning a different half.** The **IETF** owns the *form* (RFC 9773); the **issuing CA** owns the *value*, because it is the party that fixed the validity period — §4, §5 |
| Is #60's stated rationale attested | **No, and it is worse than unattested: it is a non sequitur.** Its premise argues *against* its conclusion — §8 |
| Does `N` move | **Yes.** `N = 30 days` becomes **`N` = one third of the certificate's validity period, and one half where that period is 10 days or less** — §10 |
| Does ARI change the shape rather than the value | **The shape changes, and ARI is the symptom rather than the cause.** The cause is that certificate lifetimes are now **plural and shrinking on a published schedule**, which makes *any* fixed day count degenerate on part of the estate — §7 |
| Why the world's own answer is not a number | Because a fixed lead time is a **product** of a moving quantity. Ship the fraction and it never goes stale; ship the product and it goes stale silently — §11 |
| Does verge-asm read ARI in v1 | **No**, and not on cost: the ARI window is a **CA load-balancing** instrument, and the CA's own published backstop lands on the same point with no outbound request — §6.4 |
| Does §10.4's one-way default rule cover a renewal-lead-time default | **No, and the ticket was right to suspect it.** The permissive/restrictive axis is undefined for a **required parameter**. Resolved as *outside the domain* rather than as a pass — §6.3 |

**The headline is that the world already moved off 30, and #60's number was stale on the day it was
written.** Certbot removed its fixed 30-day threshold in **4.0.0**, and its documentation says so in
those words. lego removed the same default in **v5**. cert-manager never had one. Let's Encrypt
publishes 1/3-of-lifetime as its backstop and computes its ARI window at the same point. Four
independent implementations and the issuer agree on a **fraction**. One client, `acme.sh`, is the
lone holdout on a flat 30.

**Two measured non-statements frame it.** In RFC 8555 the word `renew` occurs **once** in 196,903
bytes, and it is about order mechanics. In RFC 9773 — a document whose entire subject is renewal
timing, where `renew` occurs **80** times — the string `30` occurs **zero** times. The protocol that
defines ACME says nothing about when to renew. The extension that defines when to renew declines to
name a number, on purpose.

---

## 2. What was retrieved

| Source | What it is | How |
|---|---|---|
| [RFC 8555](https://www.rfc-editor.org/rfc/rfc8555.txt) | ACME. IETF Standards Track, March 2019 | Full text, 196,903 B, searched locally |
| [RFC 9773](https://www.rfc-editor.org/rfc/rfc9773.txt) | ACME Renewal Information (ARI). IETF Standards Track, **June 2025** | Full text, 25,666 B, searched locally |
| [Let's Encrypt Integration Guide](https://letsencrypt.org/docs/integration-guide/) | The issuer's own documentation | Fetched, 40,811 B, quote extracted from the bytes |
| [`letsencrypt/boulder`, `core/objects.go`](https://github.com/letsencrypt/boulder/blob/main/core/objects.go) | ISRG's own ACME server — the code that computes the ARI window | Fetched raw, 17,670 B |
| [Certbot user guide](https://eff-certbot.readthedocs.io/en/stable/using.html) | EFF's documentation for the most-deployed client | Fetched, markup stripped, quote extracted |
| [CA/Browser Forum Baseline Requirements](https://github.com/cabforum/servercert/blob/main/docs/BR.md) | v2.2.9, dated 6 Aug 2026 | Fetched raw from the Forum's own repo, 353,070 B |
| Let's Encrypt blog and docs; `acme.sh`; lego; cert-manager | Deployment facts and client defaults | Per-source, cited inline |

Every quotation below was taken from **retrieved bytes**, and the RFC counts come from `grep` over
the retrieved file — not from a rendering, a search snippet or a summarising layer. That is the
discipline [#46](https://github.com/winniel123/verge-asm/issues/46)'s truncated-conditional finding
imposed, and §15 records where it caught something.

Under [`sensitive-ports.md`](./sensitive-ports.md) §10.5, **the IETF owns what its RFCs specify.**
§5.1 argues the issuer's ownership separately, because it is the load-bearing move.

---

## 3. RFC 8555 says nothing about renewal timing

**[measured]** `renew` occurs **once** in RFC 8555, at §7.4.2, and this is the whole of it:

> A certificate resource represents a single, immutable certificate. If the client wishes to obtain
> a renewed certificate, the client initiates a new order process to request one.

That is a statement about *how* to renew. It contains no *when*.

**[measured]** `30 day` occurs **zero** times. `lifetime` occurs twice — once about authorization
state, once in Security Considerations about account-key compromise. `expir` occurs 25 times and
every one is about an **ACME resource** (an order's `expires` field, an authorization expiring)
except §7.1.2's `contact` field, which mentions certificate expiration only to say what a server
might email about:

> For example, the server may wish to notify the client about server-initiated revocation or
> certificate expiration.

So the protocol every ACME client implements has **no position on renewal lead time**. Alone, that
is a citable non-statement in §2.7's `rpcbind`/RFC 1833 style and would have produced §2.7's
outcome. **The retrieval did not stop there, and the rest of it is not a non-statement.**

---

## 4. RFC 9773 — the protocol's owner has a position, and it is about the *form*

RFC 9773 was published **June 2025**, IETF Standards Track, authored by A. Gable of ISRG. Its §1
opens by enumerating exactly the thing #60 asserted about:

> Most ACME [RFC8555] clients today choose when to attempt to renew a certificate in one of three
> ways:
>
> 1. they may be configured to renew at a specific interval (e.g., via cron),
>
> 2. they may parse the issued certificate to determine its expiration date and renew a specific
>    amount of time before then, or
>
> 3. they may parse the issued certificate and renew when some percentage of its validity period has
>    passed.

And then, immediately:

> **The first two** create significant barriers against the issuing Certification Authority (CA)
> changing certificate lifetimes. All three ways may lead to load clustering for the issuing CA due
> to its inability to schedule renewal requests.

**Read that carefully, because the sentence is doing precise work and it is easy to over-read.**

- Form 2 is *a fixed amount of time before expiry* — `N = 30 days` exactly. It is named, by the
  protocol's owner, as creating **"significant barriers against the issuing CA changing certificate
  lifetimes."**
- Form 3 is *a percentage of the validity period*. It is **not in the criticised pair.** The
  criticism is scoped, in the document's own words, to *"the first two"*.
- The second sentence widens to *"all three"*, and what it names there is **load clustering for the
  issuing CA** — the CA's operational problem, not the subscriber's correctness problem, and not
  something a monitoring product's horizon bears on at all.

So the protocol's owner attests, on the record and on the Standards Track, that **a fixed day count
is the wrong form and a fraction of validity is not.** That is an attestation of shape. It names no
value, and §1 of the abstract says why naming one is not its business:

> This document specifies how an Automated Certificate Management Environment (ACME) server may
> provide suggestions to ACME clients as to when they should attempt to renew their certificates.
> This allows servers to mitigate load spikes and **ensures that clients do not make false
> assumptions about appropriate certificate renewal periods.**

**[measured]** The string `30` occurs **zero** times in RFC 9773. `day` occurs three times, all in
§4.3.2 capping the `Retry-After` polling interval. The document that owns renewal timing names no
lead time whatever, and that is deliberate rather than an omission.

---

## 5. The issuing CA owns the *value*, and its value is a fraction

### 5.1 Why the issuer is an owner under §10.5

§10.5 defines an owner as *the party that designed the protocol, or that authors the reference
implementation, speaking about the thing it designed or wrote.* Let's Encrypt / ISRG qualifies three
times over, and the third is the one that matters.

- **It designed the renewal-timing protocol.** RFC 9773's author is A. Gable, ISRG.
- **It authors the server implementation ARI was built in.** `boulder` is ISRG's own ACME server.
  RFC 9773's acknowledgments thank a contributor "for contributing an **independent** server
  implementation", which places boulder as the original.
- **It sets the validity period of the certificate in question.** From the
  [FAQ](https://letsencrypt.org/docs/faq/): *"There is no way to adjust these lifetimes, there are no
  exceptions."* **The party that fixes an interval is the party entitled to say where within that
  interval replacement is due.** That is ownership of the artefact in the most literal sense §10.5
  keys on, and it is why this is not §2.3's *corroborator standing where an owner should* — the
  failure [#66](https://github.com/winniel123/verge-asm/issues/66) caught.

The scope limit is real and is disclosed in §14: Let's Encrypt owns **Let's Encrypt certificates**,
in the same way §11.8 held that Cisco would own its own SNMP agent and not SNMP.

### 5.2 What it says — prose

From the [Integration Guide](https://letsencrypt.org/docs/integration-guide/), verbatim from the
retrieved bytes (the HTML entities are the page's own curly apostrophes):

> We recommend checking ACME Renewal Information for each certificate at least twice a day. The ARI
> endpoint will recommend when to renew.
>
> **As a backstop to ARI, we recommend renewing certificates automatically when they have a third of
> their total lifetime left. For certificates with a validity period under 10 days, we recommend
> renewing halfway through their total lifetime. For Let's Encrypt's current 90-day certificates,
> that means renewing 30 days before expiration.**

**This is the single most important sentence in the note, and the order of its clauses is the
finding.** The **rule** is *a third of their total lifetime left*. The **thirty days** is an
arithmetic illustration of that rule at one particular lifetime, offered as a convenience. `30` is
not the quantity. It is 90 × ⅓ evaluated once, at a lifetime that has a published expiry date
(§7.2).

**#60 shipped the illustration and discarded the rule.**

### 5.3 What it says — code

`boulder`'s `RenewalInfoSimple` in `core/objects.go` is the function that computes the window Let's
Encrypt actually serves over ARI. Verbatim:

```go
// RenewalInfoSimple constructs a `RenewalInfo` object and suggested window
// using a very simple renewal calculation: calculate a point 2/3rds of the way
// through the validity period (or halfway through, for short-lived certs), then
// give a 2%-of-validity wide window around that. Both the `issued` and
// `expires` timestamps are expected to be UTC.
func RenewalInfoSimple(issued time.Time, expires time.Time) RenewalInfo {
	validity := expires.Add(time.Second).Sub(issued)
	renewalOffset := validity / time.Duration(3)
	if validity < 10*24*time.Hour {
		renewalOffset = validity / time.Duration(2)
	}
	idealRenewal := expires.Add(-renewalOffset)
	margin := validity / time.Duration(100)
```

Two things follow, and the second is what disposes of §6.4.

**The prose and the code agree exactly** — `validity / 3`, halving below `10*24*time.Hour`. A
first-party prose position corroborated by the first-party implementation of the same rule is the
strongest footing available under §2.2, stronger than either alone.

**The dynamic window and the static backstop land on the same point.** The ARI `suggestedWindow` is
centred on ⅓-remaining with a margin of 1% of validity either side. So the value a client would
*fetch* from the CA and the value the CA tells it to *compute* when it cannot fetch are the same
number. That is why not reading ARI costs nothing (§6.4).

---

## 6. The reference clients — and the §10.4 question the ticket asked

### 6.1 They have already moved, and one of them says so explicitly

| Client | Default renewal point | ARI |
|---|---|---|
| **Certbot** (EFF) | **⅓ of lifetime remaining; ½ if lifetime ≤ 10 days** — since 4.0.0 | Since 4.1.0 (2025-06-10), on by default |
| **lego** (go-acme) | **⅓ remaining; ½ for short-lived** — the `--days 30` default was removed in v5 | On by default, cites RFC 9773 |
| **cert-manager** | **⅔ through the certificate's duration** = ⅓ remaining. Never had a day-count default | Experimental behind the `ACMEUseARI` gate, since 1.21 |
| **acme.sh** | **`DEFAULT_RENEW="${DEFAULT_RENEW:-30}"`** — a flat 30 days | Since 3.1.4 (2026-07-17), on by default |

Certbot's user guide states the transition in its own words, and this is the cleanest single
sentence in the corpus for the proposition that the world moved:

> This command attempts to renew any previously-obtained certificates which are ready for renewal.
> As of Certbot 4.0.0, a certificate is considered ready for renewal when less than 1/3rd of its
> lifetime remains. For certificates with a lifetime of 10 days or less, that threshold is 1/2 of
> the lifetime. **Prior to Certbot 4.0.0 the threshold was a fixed 30 days.**

cert-manager's documentation gives the reason a fraction is preferred, in a sentence that reads like
it was written for this ticket:

> `spec.renewBefore` specifies an absolute duration, while `spec.renewBeforePercentage` computes the
> effective 'renewBefore' using the actual duration of the issued certificate. **Using
> `spec.renewBeforePercentage` is recommended to prevent renewal loops in case the actual duration
> is less than expected.**

*A renewal loop when the actual duration is less than expected* is precisely the degenerate case
§7.3 describes, named by an implementation that hit it.

### 6.2 The convergence is unanimous except for one holdout, and that is not evidence

Four implementations agreeing is a **frequency observation**, and §2.5 excludes it. It is recorded
here as context and it carries no weight in §10's ruling. The ruling rests on §4 (the protocol
owner, on form) and §5 (the issuer, on value). If every client on earth had stayed at 30, the ruling
would be the same, because a client is not an owner of the certificate's validity period.

### 6.3 Does §10.4's one-way rule cover a renewal-lead-time default? No — and that is the answer

The ticket asked this precisely, suspecting it might be a case the rule does not cleanly cover. It
is, and the reason generalises.

§10.4 rules that *a shipped default is an attestation **only where it restricts** — a loopback bind,
a feature off by default, a daemon that refuses to start*, because a restriction is a **costly act**
the maintainer paid for, while a permissive default is the **absence** of an act and so is silent.

That dichotomy presupposes that **doing nothing is an available option**, and that the permissive
default is what doing nothing looks like. `listen_addresses` has that shape: bind widely is what you
get by not deciding.

**A renewal trigger has no such option.** A client that renews at no time does not function. Every
possible value is an act, none of them is the absence of an act, and so neither *restricts* nor
*permits* applies. The permissive/restrictive axis is **undefined** for the parameter, not merely
hard to read.

> **§10.4's one-way rule is scoped to defaults over which "no act" is a real option. Where a default
> is a *required parameter* — the software cannot run without a value — the rule is *outside the
> domain*, in [ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md)'s third
> register, and it neither admits nor excludes.**

That is the same three-way treatment ADR-0032 §4 gave gate 3, applied to gate 2's third form, and it
is stated as a rule so the next required-parameter default does not have to relitigate it.

**What a required-parameter default *is*, then.** It remains a maintainer position expressed in
code — but a position about **the maintainer's own behaviour**, not about the world. Certbot's ⅓
says *this is when Certbot renews*. It does not say *this is when a certificate must be replaced*,
because Certbot did not issue the certificate and does not set its lifetime. So a required-parameter
default lands in **§2.3's corroborator tier**: it corroborates and it may never carry a row alone.
§6.2 is that rule applied.

**The row does not need it.** Let's Encrypt's Integration Guide is §2.2's *second* form — the
vendor's own documentation, in prose, about the thing it issues — so the row is carried by a
first-party position and the client defaults sit where corroborators belong. That is the opposite of
`161/udp`'s failure and it is worth stating as the contrast.

### 6.4 verge-asm does not read ARI in v1

A tempting reading of §4 and §5.3 is that `certificate-expiring` should stop carrying a parameter
and read the CA's window per certificate. The retrieval establishes that this is **available** and
that it is nonetheless **wrong**.

**It is genuinely available, and this is recorded so a later session need not rediscover it.**
RFC 9773 §4.1 constructs the request path from the certificate's own **AKI keyIdentifier** and
**Serial Number** — both fields of a certificate verge-asm already observes — and the request is
*"an unauthenticated GET"*. §4.3.1 contemplates exactly our position:

> Servers should estimate their expected load based on the number of clients, keeping in mind that
> **third parties may also monitor renewalInfo endpoints.**

**It is the wrong value to read, on four grounds.**

**The window is a load-balancing instrument.** RFC 9773 §1 says so in its own words — *"enables
dynamic time-based load balancing"*, *"reduce the size of an upcoming mass-renewal spike"*.
Rendering a CA's load-shedding schedule to an operator as a replacement horizon is §2.7's laundering
with a Standards-Track source underneath it, and it is tempting for the same reason TA14-017A was
tempting for `111/tcp`.

**It buys nothing over the backstop.** §5.3 measured that boulder centres the window on the same ⅓
point the Integration Guide tells clients to compute. The dynamic value and the static rule
coincide, so the outbound request purchases a margin of ±1% of validity.

**It has no answer for most of the estate.** ARI exists only where an ACME CA implements it. A
commercial certificate, an internal CA, a self-signed certificate — the population
`certificate-self-signed` is named for — has no `renewalInfo` resource at all.

**It is an outbound request to a third party.** A per-certificate GET to every observed
certificate's issuer is a new source under [ADR-0003](../adr/0003-third-party-source-consent-bar.md),
a new exchange with a cadence under
[ADR-0028](../adr/0028-a-facets-cadence-is-the-cadence-of-its-exchange.md), and it tells every CA in
the WebPKI which certificates this operator watches. §4.3.1 establishes they expect third parties,
not that it is free.

Recorded as **out of scope for v1** rather than refused forever: *the CA recommended replacement on
this date and it has not happened* is a well-founded signal that reads a published value rather than
a parameter we chose — which would make it an **observation** and dissolve gate 2 for it entirely.

---

## 7. Why a fixed day count is now degenerate — the cause is lifetimes, not ARI

The ticket asked whether ARI changes the shape rather than the value. **The shape does change, and
ARI is the symptom.** The cause is that certificate lifetimes have become **plural and shrinking on
published schedules**, and a fixed lead time is a fixed *fraction* of a lifetime that is moving
underneath it.

### 7.1 The ceiling, from the body that sets it

CA/Browser Forum Baseline Requirements **v2.2.9, dated 6 Aug 2026**, §6.3.2, verbatim from the
Forum's own repository:

> Subscriber Certificates issued before 2026-03-15 SHOULD NOT have a Validity Period greater than
> 397 days and MUST NOT have a Validity Period greater than 398 days.
>
> Subscriber Certificates issued on or after 2026-03-15 and before 2027-03-15 SHOULD NOT have a
> Validity Period greater than 199 days and MUST NOT have a Validity Period greater than 200 days.
>
> Subscriber Certificates issued on or after 2027-03-15 and before 2029-03-15 SHOULD NOT have a
> Validity Period greater than 99 days and MUST NOT have a Validity Period greater than 100 days.
>
> Subscriber Certificates issued on or after 2029-03-15 SHOULD NOT have a Validity Period greater
> than 46 days and MUST NOT have a Validity Period greater than 47 days.

**The 200-day tier has been in force since 2026-03-15** — five months before this retrieval. The
next step is 2027-03-15.

### 7.2 The issuer's own schedule, which moves faster than the ceiling

Let's Encrypt, [*From 90 to 45*](https://letsencrypt.org/2025/12/02/from-90-to-45/):

> - **May 13, 2026:** Let's Encrypt will switch our `tlsserver` ACME profile to issue 45-day
>   certificates.
> - **February 10, 2027:** Let's Encrypt will switch our default `classic` ACME profile to issuing
>   64-day certificates …
> - **February 16, 2028:** We will further update the `classic` profile to issue 45-day certificates …

and, in the same post, the sentence that is #60's number by another name:

> If your client doesn't support ARI yet, ensure it runs on a schedule that is compatible with
> 45-day certificates. **For example, renewing at a hardcoded interval of 60 days will no longer be
> sufficient.** Acceptable behavior includes renewing certificates at approximately two thirds of
> the way through the current certificate's lifetime.

**The default 90-day lifetime that makes `30` equal `⅓` has a published expiry date of
2027-02-10** — under six months from this retrieval.

### 7.3 The degenerate case, which is already generally available

Let's Encrypt, [6-day and IP certificates GA, 15 Jan 2026](https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability/):

> **Short-lived and IP address certificates are now generally available from Let's Encrypt. These
> certificates are valid for 160 hours, just over six days.**

> **is `certificate-expiring` ever false on a six-day certificate under `N = 30`?**

**No. It fires from the instant of issuance to the instant of expiry, forever, on every such
certificate.** A predicate that cannot be false over a population has stopped partitioning it, and
under [ADR-0024](../adr/0024-a-rules-domain-is-the-extension-of-its-name.md) that is not
`not-evaluable` and not *outside the domain* — the subject is squarely inside the domain and the
rule simply says nothing. [#53](https://github.com/winniel123/verge-asm/issues/53) made the census
the thing the operator reads, and a census reading *412 of 412, every night, forever* carries no
information at all.

**This is what makes the shape change forced rather than preferred.** There is no fixed number that
survives the spread that already ships: any `N` ≥ 6 days is degenerate on the short-lived profile,
and any `N` < 6 days is useless on a 90-day certificate. **No fixed day count works, so the answer
cannot be a different fixed day count** — which is exactly the possibility the ticket's second
constraint left open, resolved by measurement rather than by preference.

---

## 8. #60's rationale, tested — and it fails twice

#60's recorded reason, carried into ADR-0004's amendment, is one sentence with two halves:

> "30 days is where the ACME clients in the modal estate already trigger renewal, so it is the last
> point at which the operator still has the action the signal is telling them to take."

### 8.1 The first half is a frequency claim, and it is also false

*Where the clients in the modal estate already trigger* is a statement about **how many** clients do
**what**. It is the same shape as the Shodan/Censys studies §2.5 refused and as Redis's prevalence
rhetoric §2.5 declined to cite. Even a perfect frequency measurement would not admit it.

It is **additionally false as of the retrieval**, which is the part #37's precedent needed: Certbot
4.0.0 and lego v5 have removed the fixed 30-day threshold, cert-manager never had one, and the
issuer publishes a fraction. Only `acme.sh` still ships 30. So the sentence is inadmissible *and*
untrue, and the second fact is what a retrieval could establish where a re-reading could not.

### 8.2 The second half does not follow from the first — it follows *against* it

This is the sharper failure and it needs no source, only the two clauses read together.

Suppose the premise were true and the modal ACME client triggered renewal 30 days before expiry.
Then a certificate observed at 30 days remaining is one whose **automation is firing right now, on
schedule**. That is not the operator's last chance to act. It is the moment at which the operator
has *no* action, because the machine has it. The operator's last chance is wherever the automation
has demonstrably **failed**, which is strictly later and about which the premise says nothing.

The inference runs backwards. *Clients renew at 30 days* is a reason to expect 30 days to be the
point of **maximum churn**, not of maximum danger — and
[#16](https://github.com/winniel123/verge-asm/issues/16) said exactly this from the other side when
it complained that *"a 30-day warning is right for manual renewals and noise for a team on ACME"*.
#60 answered that complaint correctly (it is a flap. Damping belongs in notification) and then, in a
different section, **adopted the flap's own mechanism as the number's justification**.

**So `N = 30` was not merely unattested. Its stated ground, if true, argues for a different number.**

### 8.3 What the sentence should have been — and §9 is why it could not be found

The claim #60 wrote is about **the operator's remediation capacity**: how long a human needs to
notice, find the owner, and deploy a replacement. Ask §10.5's question of it — *who designed, or
authors the reference implementation of, the operator's remediation capacity?*

**Nobody, because it is not an artefact.** No amount of further retrieval would have found that
sentence. §9 is why that is a defect in the claim rather than a gap in the corpus.

---

## 9. The step ADR-0032 named and did not perform: derive the claim first

ADR-0032 §3 ruled that gate 1's three claims do **not** generalise, and stated what a second table
owes instead:

> **So a second curated table does not inherit #21's three claims. It owes its own closed set,
> derived the same way — from what the rule reads — and the derivation is part of the table's cost.**

**That derivation was never performed for `N`.** ADR-0032 filled the slot by quoting #60's prose
gloss — *"the world — this is the last point the operator can still act"* — and then looked for an
owner for it. §8.3 is what happens next: no owner exists, because the claim was not derived from
what the rule reads.

**Performed now, in #37's style.** `certificate-expiring` reads `not_before`, `not_after` and the
clock. A horizon is a **cut in the interval `[not_before, not_after]`**. One interval, one instant:
the only thing a cut in an interval can assert is a **position within that interval**. The set
closes at one member by construction, not by enumeration:

> **Claim (the only one available to this table): *the certificate is inside the portion of its own
> validity period in which its issuer says replacement is due.***

Test the two candidates against it.

| Candidate claim | Derivable from what the rule reads? | Owner |
|---|---|---|
| *The certificate is inside the issuer's replacement window* | **Yes** — a position in `[not_before, not_after]`, both of which the rule reads | **The issuer**, which fixed the interval — §5.1 |
| *The operator can still act* (#60, ADR-0032) | **No** — imports the operator's capacity, which the rule reads nothing about | **None**, by construction |

**The owner became findable the moment the claim was derived rather than inherited.** That is the
general finding, and it is [ADR-0034](../adr/0034-derive-the-claim-before-looking-for-the-owner.md).

It also explains a shape this effort has now met three times. §2.7's `111/tcp` and §11's `161/udp`
both ended *we could not find anyone entitled to say so*, and in both the absence was genuine. Here
the absence was **manufactured by the claim's wording**, and the two look identical from inside a
retrieval. Distinguishing them is what ADR-0034 is for.

---

## 10. Ruling

> **`N` moves. `certificate-expiring`'s horizon is no longer a fixed number of days. It is
> `N = ⅓ × (not_after − not_before)`, and `N = ½ × (not_after − not_before)` where that validity
> period is 10 days or less.**
>
> **The row is attested.** The **IETF** attests the form — RFC 9773 §1 names a fixed lead time as
> creating *"significant barriers against the issuing CA changing certificate lifetimes"* and does
> not so name a percentage of validity. The **issuing CA** attests the value — Let's Encrypt's
> Integration Guide states ⅓-of-lifetime-remaining and the ½ rule under 10 days in prose, and
> `boulder` implements exactly that. Each owner attests the half it owns.

**Why this is a move on a retrieval and not on a re-reading.** #37's precedent bars moving a row on
text already held. Three things here were not held and could only be retrieved:

1. RFC 9773 exists and is a **finished Standards-Track RFC** rather than the draft the ticket expected.
2. Certbot **removed** its fixed 30-day threshold in 4.0.0 and documents having done so.
3. Let's Encrypt's six-day profile went **generally available on 2026-01-15**, which is what makes any fixed `N` degenerate (§7.3).

**Why the claim's rewording is licensed and is not a re-reading either.** ADR-0032 §3 *required*
this table to derive its own claim set and the derivation had never been done (§9). Doing an
outstanding obligation is not re-reading a discharged one.

**#60's rejected alternative is revived, and only its first ground is falsified.** #60 listed and
rejected:

> **`N` derived from the certificate's own validity period** (*expiring within one third of its
> lifetime*) — attractive because it handles 90-day ACME and 398-day commercial certificates
> uniformly. It loses as **a formula we invented sitting inside the comparison path with no
> attestation**, and it makes the census incomparable **between endpoints**, which is worse than the
> cross-install problem it was reaching past.

**Ground one is falsified by retrieval.** It is not a formula we invented — it is the formula the
issuer publishes, the reference server computes, and three of the four major clients ship. #60 could
not have known: it did not look.

**Ground two survives as stated and is overridden on argument, which is marked as such.** A
per-certificate horizon does make two endpoints' horizons differ. But §7.3 shows a **fixed** `N`
already makes the census incoherent across endpoints, and worse: under `N = 30` the fired population
is *every six-day certificate always, every 45-day certificate for two thirds of its life, every
398-day certificate for 7.5% of its life* — one column counting five different things. Under the
fraction the population is one thing everywhere: **certificates in the last third of their validity.**
The census becomes more comparable between endpoints, not less. This is reasoning rather than
retrieval and §14 flags it.

**What is withdrawn.** ADR-0004's #60 amendment sentence — *"30 days is where the ACME clients in the
modal estate already trigger renewal, so it is the last point at which the operator still has the
action the signal is telling them to take"* — is **withdrawn in both halves**, per this repo's
name-and-withdraw convention: the first is frequency and false, the second does not follow from it.
Its replacement is §10's attestation, which is a claim about the certificate and not about the
operator.

---

## 11. The rule this establishes about constants

> **Where a project-authored constant is the product of a fraction and a moving world quantity, ship
> the fraction. The product is a measurement of the world at a date, and it goes stale with nothing
> changing in the repository and no document anywhere being retracted.**

This is [ADR-0032](../adr/0032-an-evidence-standard-attaches-to-a-table-not-to-a-rule.md) §8's
**silent de-attestation** one level down, and it is strictly worse in one respect and strictly
better in another.

**Worse:** de-attestation needs a maintainer to flip a default. **Silent staleness needs nobody to
do anything at all** — the world parameter moves on a schedule its owner published, and the repo's
number is wrong on arithmetic while every document it cites still says what it said.

**Better:** it is **preventable by construction**. A row that ships the fraction cannot go stale,
because the moving quantity is read at evaluation time from the subject itself. There is no watch to
maintain and no watch list to join.

`N = 30` was stale **before it was written**: Certbot 4.0.0 had already removed the threshold when
#60 recorded the rationale, and nothing in this repository could have shown that. It is the effort's
first measured instance of a constant that was wrong on the day it landed, and it is the argument
for the rule.

---

## 12. This is not a dial — the ticket's second constraint, checked against #60's three grounds

#60 killed an operator-configurable `N` on three grounds. **All three concern *per-install*
variation, and a per-certificate fraction has none of those properties.**

| #60's ground | Does the fraction reopen it? |
|---|---|
| A settings field moves a version without a release, `Break`ing every timeline, and the operator cannot see it coming | **No.** The fraction is project-authored, fixed at the release, and identical in every install. No operator touches it |
| [ADR-0015](../adr/0015-the-value-space-is-the-commitment.md)'s model-layer damping with the dial handed outward | **No.** Nothing is handed outward. The rule narrows and widens with the **certificate's own** validity period, which is a measurement, not a preference |
| [ADR-0021](../adr/0021-a-version-leaf-is-a-decision-not-a-binary.md)'s gate is voided because CI gates one function while every install evaluates another | **No, and this is the decisive one.** CI gates **exactly** the function every install runs. The declared parameter set becomes `{⅓, ½, 10 days}` — still declared data in the repository, still mechanically checkable, still one function behind one leaf |

**#22's line is untouched.** The configurable/fixed cut is *inside versus outside the comparison
path*, and this parameter stays inside and stays ours. Nothing here makes an operator dial legal that
was not legal before.

**And the flap is unchanged.** On a healthy ACME estate the rule fires and clears every cycle under
the fraction exactly as it did under 30 — the horizon moved, the shape of the flap did not.
[ADR-0007](../adr/0007-drift-is-a-timeline-of-spans.md) put all damping in notification precisely so
the model keeps the flap count as a fact, and #60 routed #16's ACME complaint there. That routing
stands and this ruling does not disturb it.

---

## 13. Every figure and obligation that moves

| Where | Was | Is |
|---|---|---|
| `certificate-expiring`'s declared parameter | `N = 30 days` | **`⅓` of validity, `½` below a 10-day validity** |
| ADR-0004's #60 amendment, *"shipped at 30 days"* | 30 days | **Superseded.** The parenthetical is left standing per the name-and-withdraw convention and the amendment is annotated |
| ADR-0004's #60 rationale sentence | Stated but unverified | **Withdrawn in both halves** (§10) |
| ADR-0032 §5's gate-2 table, `N` row | *"Applies. Currently unattested, #67"* | **Applies. Attested** — IETF on form, issuer on value |
| ADR-0032's walk, `certificate-expiring` row | *"#21 gate 2 — unattested, #67"* | **Attested; and the claim is not the one recorded** (§9) |
| The rule's inputs | `not_after`, the clock | **`not_before`, `not_after`, the clock.** `not_before` is already read by `certificate-not-yet-valid`, so **no new measurement** and no facet change |
| The corpus obligation for clock rules | A row carries its evaluation instant | **And its `not_before`.** #60's obligation is widened, not replaced |
| The rule's `Predicate domain` | `certificate` is `Presented` | **Unchanged.** ADR-0024's domain is untouched |
| What it costs | — | **One `Break` on one rule for one cadence**, exactly as ADR-0008 and #60 priced a revision. **Vacuous before the first install**, and §11.6's argument applies: the price today is zero and after v1 it is a comparability cycle |
| [#12](https://github.com/winniel123/verge-asm/issues/12) | Carries `N = 30` | **Carries the fraction.** Not blocked — #67 never blocked #12 and does not now |
| ADR-0032 §8's watch list | 5432, 5984, 9042 · `N` filed as *chased* | **`N` leaves both piles.** A row that reads the moving quantity at evaluation time is neither watched nor chased (§11) |

---

## 14. Thin ground, and what was not established

**Flagged per the effort's standing rule.**

**The claim derivation in §9 is the load-bearing move and it is reasoning, not retrieval.** Every
source quoted here is real and verified, but they attest *⅓ of validity* only once the table's claim
is *the issuer's replacement window*. A reader who holds that the claim really is #60's *"the last
point the operator can still act"* should read this note as finding **no owner** and should leave
`N` at 30, disclosed. The whole ruling turns on ADR-0034's rule, and that dependency is stated
rather than buried.

**The issuer's attestation is scoped to its own certificates.** Let's Encrypt owns Let's Encrypt
certificates, in exactly the sense §11.8 held that Cisco would own its own SNMP agent. Applying ⅓ to
a commercial certificate, an internal CA's certificate or a self-signed one is an **extension beyond
the attestation**, and it is the row's disclosed weakness. It is mitigated and not cured by the fact
that Certbot, lego and cert-manager apply the same fraction issuer-agnostically — and §6.2 says why
that mitigation carries no evidentiary weight.

**The ⅓ and the 10-day threshold are one CA's numbers.** No second CA was retrieved stating a
different fraction, and none was retrieved stating the same one. The claim here is *the issuer of a
certificate owns its replacement window*, which predicts that a different CA could legitimately
publish a different fraction for its own certificates — and v1 applies one fraction to all of them.
That is a real limitation of a project-authored single-row table and it does not go away.

**#60's second ground (§10) is overridden on argument.** The census-comparability objection is
answered by reasoning about what a fixed `N` does across heterogeneous lifetimes, not by a source.

**What would change the verdict, in one line.** An owner sentence — from the IETF, or from a CA
about certificates it issues — placing the replacement point somewhere other than ⅓ of validity, or
a demonstration that the rule's claim really is about the operator rather than about the
certificate. The first would move the fraction. The second would restore the *no owner* finding and
return `N` to a disclosed constant.

---

## 15. Retrieval hazards and errors caught

Recorded per §9.5's practice. Four of these would have produced a wrong answer.

- **ARI is not a draft.** The ticket names it *"the draft/RFC"*. It is **RFC 9773**, Standards
  Track, **June 2025** — over a year old at retrieval. Treating it as in-progress would have
  under-weighted the strongest source in the note.
- **`lego`'s `master` branch is stale and returns HTTP 200.** lego's default branch is `main` and the
  shipped release is v5. `master` is a v4-era branch still serving `cmd/cmd_renew.go` with
  `Value: 30` and a `// TODO(ldez): in v5, remove this flag` comment beside it. **A retrieval that
  read `master` would have found a 30-day default that has not shipped for a major version, with no
  error to warn it.** This is §9.5's Red Hat hazard in a new form: not a blocked fetch, but a
  *successful* fetch of superseded bytes.
- **Certbot's source moved to a `src`-layout in 4.1.0** and `RENEWER_DEFAULTS` no longer exists.
  `renew_before_expiry` now has **no default at all**. A search for the old symbol returns nothing
  and could be misread as "not found, therefore unknown" rather than "removed, deliberately".
- **The 10-day halving threshold is *not* the CA/Browser Forum's short-lived definition, and they
  have diverged.** BR §1.6.1 defines a Short-lived Subscriber Certificate as ≤ 10 days before
  2026-03-15 and **≤ 7 days on or after**. Let's Encrypt, boulder, Certbot and lego all use **10
  days** for the renewal-halving rule. A session that conflates them would "correct" the renewal
  threshold to 7 and would be wrong: they are independent numbers that happened to coincide.
- **The `30`-occurs-zero-times count in RFC 9773 is a raw substring count**, not word-bounded, and is
  reported that way deliberately — had `30` appeared in an RFC number or a base64 blob the count
  would have needed disambiguating, which is §11.9's `161`-matches-`10161` trap. It reads zero, so
  none arises.
- **RFC 9773 §1 is the near-miss and it is the trap this ticket was built around.** Its form 2 —
  *"renew a specific amount of time before then"* — is the closest thing in the corpus to #60's
  sentence, and a fast read could quote it as attesting a lead time. It names no amount, and the
  paragraph immediately after it names forms 1 and 2 as **the problem the document exists to
  solve**. Quoting it in support of a fixed `N` would be citing a specification's description of the
  behaviour it was written to replace. The scoping word is *"The first two"*, and it was checked
  against the bytes rather than remembered.
- **Let's Encrypt's expiry-notification schedule is no longer retrievable from a live first-party
  page.** The service ended **4 June 2025**. `letsencrypt.org/docs/expiration-emails/` now contains
  three lines and does not state the historical schedule. It was **20 days and 7 days**, recovered
  only from an Internet Archive capture of that URL (2024-09-09) — one step weaker than direct
  retrieval, flagged per §9.5, and **not load-bearing**: it is a notification schedule, not a
  position on replacement timing, and it is cited nowhere in §10. Let's Encrypt's own stated reason
  for ending it points the same way as the rest of this note: *"more and more of our subscribers have
  been able to put reliable automation into place for certificate renewal."*
- **`letsencrypt.org/2024/04/25/guide-to-integrating-ari-into-existing-acme-clients/` does not state
  the window formula** — it shows example JSON only. The fraction comes from the Integration Guide's
  prose and from `boulder`, not from that post. A session citing the blog post for the fraction would
  be citing a page that does not contain it.
- **Not exhausted, and recorded as such.** No CA other than Let's Encrypt was retrieved for a
  replacement-window position. §14 states what that costs.

---

## Sources

**Standards**
- [RFC 8555, *Automatic Certificate Management Environment (ACME)*](https://www.rfc-editor.org/rfc/rfc8555.txt) — Barnes, Hoffman-Andrews, McCarney, Kasten. IETF Standards Track, March 2019
- [RFC 9773, *ACME Renewal Information (ARI) Extension*](https://www.rfc-editor.org/rfc/rfc9773.txt) — A. Gable, ISRG. IETF Standards Track, June 2025
- [CA/Browser Forum, TLS Baseline Requirements v2.2.9](https://github.com/cabforum/servercert/blob/main/docs/BR.md) (6 Aug 2026) — §1.6.1, §4.6, §6.3.2, retrieved from the Forum's own repository
- [Ballot SC-081v3, *Introduce Schedule of Reducing Validity and Data Reuse Periods*](https://cabforum.org/2025/04/11/ballot-sc081v3-introduce-schedule-of-reducing-validity-and-data-reuse-periods/)

**The issuer**
- [Let's Encrypt, Integration Guide](https://letsencrypt.org/docs/integration-guide/) — the ⅓ backstop
- [`letsencrypt/boulder`, `core/objects.go`](https://github.com/letsencrypt/boulder/blob/main/core/objects.go) — `RenewalInfoSimple`
- [Let's Encrypt, *Decreasing Certificate Lifetimes to 45 Days*](https://letsencrypt.org/2025/12/02/from-90-to-45/) (2 Dec 2025)
- [Let's Encrypt, *6-day and IP Address Certificates are Generally Available*](https://letsencrypt.org/2026/01/15/6day-and-ip-general-availability/) (15 Jan 2026)
- [Let's Encrypt, *Improving Resiliency and Reliability with ARI*](https://letsencrypt.org/2023/03/23/improving-resliiency-and-reliability-with-ari/) (23 Mar 2023)
- [Let's Encrypt, *Ending Support for Expiration Notification Emails*](https://letsencrypt.org/2025/01/22/ending-expiration-emails/) · [historical schedule, Internet Archive capture 2024-09-09](https://web.archive.org/web/20240909130853/https://letsencrypt.org/docs/expiration-emails/)
- [Let's Encrypt, FAQ](https://letsencrypt.org/docs/faq/) · [Profiles](https://letsencrypt.org/docs/profiles/)

**Clients — corroboration only, per §6.2**
- [Certbot user guide](https://eff-certbot.readthedocs.io/en/stable/using.html) · [`renewal.py`](https://raw.githubusercontent.com/certbot/certbot/main/certbot/src/certbot/_internal/renewal.py)
- [lego flag reference](https://go-acme.github.io/lego/references/ref-flags/index.html) · [`cmd/cmd_run_renew.go`](https://raw.githubusercontent.com/go-acme/lego/main/cmd/cmd_run_renew.go)
- [cert-manager, Certificate resource](https://cert-manager.io/docs/usage/certificate/) · [1.21 release notes](https://cert-manager.io/docs/releases/release-notes/release-notes-1.21)
- [`acme.sh`](https://raw.githubusercontent.com/acmesh-official/acme.sh/master/acme.sh) — `DEFAULT_RENEW`
