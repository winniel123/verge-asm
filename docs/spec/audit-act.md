# The `Act` corpus — an operator-act record for verge-asm

- **Status:** Accepted — handoff spec for map [#1786](https://github.com/winniel123/verge-asm/issues/1786), terminal ticket [#1793](https://github.com/winniel123/verge-asm/issues/1793)
- **Ruling:** **None yet.** This document names the decisions it believes owe an ADR (§10) and opens no issue. Only a human opens the issue that becomes an ADR, and its number becomes the ADR's number (`docs/spec/adr-governance.md`).
- **Build status:** **Not built.** No migration, no table, no handler and no copy has landed. `/to-tickets` cuts the implementation map from this document.
- **Decisions folded:** [#1787](https://github.com/winniel123/verge-asm/issues/1787) withdrawal set · [#1788](https://github.com/winniel123/verge-asm/issues/1788) transaction placement · [#1789](https://github.com/winniel123/verge-asm/issues/1789) act surface · [#1790](https://github.com/winniel123/verge-asm/issues/1790) stored union · [#1791](https://github.com/winniel123/verge-asm/issues/1791) producer gate · [#1792](https://github.com/winniel123/verge-asm/issues/1792) copy · [#1794](https://github.com/winniel123/verge-asm/issues/1794) limb 4 · [#1795](https://github.com/winniel123/verge-asm/issues/1795) the `Actor` union · [#1796](https://github.com/winniel123/verge-asm/issues/1796) the drawn example · [#1797](https://github.com/winniel123/verge-asm/issues/1797) ADR-0093 limb 2 · [#1805](https://github.com/winniel123/verge-asm/issues/1805) migrations · [#1820](https://github.com/winniel123/verge-asm/issues/1820) the Actor cell · [#1822](https://github.com/winniel123/verge-asm/issues/1822) the attribution columns

This document folds map #1786's Settled block and thirteen closed tickets into one buildable spec. It
makes **no new decision** about the model. Where a decision was thin, deferred or held at odds, this
document says so and does not smooth it over (§14).

## What this builds

An **`Act`** is one recorded act by one principal on this instance. It is the **fifth** Operational
corpus, beside `Dispatch`, `Message`, `Delivery` and `Transcript`.

Today the admin `audit` tab renders an empty state and says so in shipped copy:
*"This build keeps no separate queryable log of admin acts"* (`design-system/templates/settings.tmpl:867`).
`fillAuditSection` returns nil (`cmd/web/settings.go:918`). `docs/guides/accounts.md:152-154` states
it more baldly still — *"the **Audit** tab is honestly empty"*.

This effort ends that. It adds one append-only corpus, one recorder, one CI gate, one admin reader,
and the enumerated withdrawal of every sentence that specifies the refusal.

### This withdraws a refusal. It does not fill a gap

The audit log was **refused**, not overlooked. [#127](https://github.com/winniel123/verge-asm/issues/127)
ruled an operator-act record out of v1. ADR-0073 §1 made that refusal *"total rather than nearly
total"*. `CONTEXT.md`'s `Annotation` entry restates it *"without exception"*. So
[ADR-0058](../adr/0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) governs
this whole effort — **a superseded mechanism is withdrawn at the site that specifies it** — and §8 is
not an appendix. An implementation that ships the corpus and leaves §8 unbuilt has not finished.

### The ground: the demand side, not #127 §9's reopening condition

#127 §9 reopens *"when the spec admits a second party who can mutate"*. That condition is
**supporting**, not load-bearing. Two facts on today's tree are:

1. **Four ADRs already prescribe an audit write in their Decision blocks.** ADR-0113 (SSO link,
   self-unlink, admin-remove), ADR-0053 (a secret being set), ADR-0123 (the API enable), and
   `docs/spec/packaging-and-configuration.md` §5.1's test — *"if a change to it should appear in the
   audit trail, it may not live in the environment."* That is a live contradiction in the repo, and
   §8 · C enumerates eighteen sentences of it.
2. **The refusal was already overrun in the store.** Fifteen columns across twelve tables carry a
   `created_by`/`updated_by` FK to `account` — the option #127 §3 weighed and rejected **by name**.
   Six of the fifteen render today. The contradiction is the **shipping**, never the rendering
   ([#1822](https://github.com/winniel123/verge-asm/issues/1822)).

### Five conscious reversals

Each is priced and recorded, not accidental.

| Mechanism | Before | What this builds |
| --- | --- | --- |
| **#127** — an operator-act record is out of v1; *"named accounts create identity; they do not create a log"* | No operator act is written down with an actor on it | The `Act` corpus, under a four-limb predicate (§1) |
| **ADR-0073 §1's totality** — *"#127's ruling is therefore total rather than nearly total"* | The refusal reached its last site | The refusal survives **over the Declared layer only**. `Annotation` still carries no author; the **act** of annotating carries its actor (§8 · A.2) |
| **ADR-0126** — a read-audit is deferred *"because the audit facility is a repo-wide stub"* | Reads unaudited | One read is auditable — a `Transcript` disclosure, under limb 4 (§1.4) |
| **ADR-0093 limb 2** — *"nothing else in the Observed or Operational corpus may already date the act"* | An `Annotation` lives with no dated residue | Limb 2 restated on the **nature** of the residue. `Annotation` keeps its instant (§8 · D) |
| **Restrict-and-refuse on the attribution columns** — 14 FKs pin every account that ever declared anything | A working admin usually cannot be removed | All 14 become nullable `ON DELETE SET NULL`, paired with a `JOIN` sweep (§9) |

### Two marks

On `docs/spec/measurement-offers.md`'s convention.

| Mark | Means |
| --- | --- |
| `[derived]` | Follows from a locked decision named in the row |
| `[thin]` | Chosen with a stated price, or held at odds; revisable on named grounds |

---

## 1. The predicate — four limbs, each with its named reader

**An act is auditable when a principal**

1. **changed the estate's declaration**, or
2. **changed who may act on this instance**, or
3. **directed the instance to act on the network**, or
4. **caused the instance to disclose a value from a corpus the model seals.**

This is a **predicate**, never a list of routes. A route list goes stale on the next route, which is
the failure the predicate exists to avoid.

**The four limbs share one shape, and it was never *changed something*: a principal caused an effect
the instance cannot take back.** Limbs 1 and 2 change stored state. Limb 3 has no local mutation at
all — its effect is a real HTTP request with a third party on the other end. Limb 4 is limb 3 one
register over: the instance acting outward by disclosing instead of over the network. Once the bytes
are on the operator's screen they are outside the instance, and no later act retrieves them.

### 1.1 Limb 1 — the estate's declaration

**Reader:** the admin asking *who changed the seed list* — #11's own first example, which #127 §7
listed as unanswerable in v1.

32 classes. Every Declared term's declaration, withdrawal and edit, plus every operator dial.

### 1.2 Limb 2 — who may act on this instance

**Reader:** the admin asking *who granted this access*.

20 classes. Limb 2 catches what limb 1 cannot: an SSO link, a TOTP enrolment, a token mint and an
account removal are not Declared terms, and **three of the four demanding ADRs are about exactly
those** (ADR-0113, ADR-0053, ADR-0123).

**The whole login family is exempt, and this is a ruling rather than an omission.** A login, a failed
login, a TOTP step, a logout, a session revoke and a session expiry move *which credential is
presented*, never *who may act*. The account's next login succeeds either way, and the `session` table
already carries that fact. The cost of the other ruling decides it: `POST /login` and `POST /forgot`
are **unauthenticated** and take an attacker-supplied username, so auditing them makes an unbounded,
never-deleted corpus writable from the sign-in screen. That is a session-event log — a different
product. Eleven acts turn on this ruling. §2.3 lists them.

### 1.3 Limb 3 — directing the instance to act on the network

**Reader:** the admin asking *who launched a scan against production* — #11's own second example.

7 classes. Its ground is #127 §2: the probing-gate case is *"stronger than the ordinary
accountability case"*, because the act has a consequence **outside the estate**, with a real third
party on the other end. Limbs 1 and 2 both drop it.

**Limb 3 has no local mutation by definition.** §7.4 turns on this.

### 1.4 Limb 4 — a disclosure from a corpus the model seals

**Reader:** the admin asking *which of us three opened that transcript*.

1 class, reaching exactly one act today: a `Transcript` disclosure at `GET /run/{id}/raw` and
`GET /runs/{id}/raw` (`cmd/web/handlers.go:388-389`).

**The wording is load-bearing and was measured.** The loose first wording — *"a value it holds as a
secret"* — was **rejected**: it admits `GET /account/totp/enroll`, which renders the plaintext TOTP
seed to the enrolling account (`cmd/web/auth.go:1010`) while `account.totp_secret` is ciphertext under
ADR-0172. That is a mint-to-owner step with no accountability question — the instance minted the
secret **for the principal it handed it to**. So the limb borrows **ADR-0126's own corpus/column
line**: it is *"the first **corpus** Postgres holds a secret for"*. The mint-to-owner case falls out
with no exception, because a column on the identity table is not a corpus the model seals.

**On the mechanism objection.** *"A corpus the model seals"* can be read as naming mechanism, which a
predicate may not do. It does not. Sealing this corpus is a **modelled claim** made deliberately in
ADR-0126 at the price of reversing ADR-0053. The limb names that claim, never the cipher. The AEAD,
the nonce and the volume key may all change without touching the predicate.

**Trigger: the disclosure, never the request.** `rawOutputPage` renders a complete page on
`pgx.ErrNoRows` (`cmd/web/rawoutput.go:105-108`) and discloses nothing. The disclosure is the three
`transcript.Open` calls in `fillRawOutputView` (`:140,159,163`). **A miss writes no `Act`.**

**Guarantee: #1788's, with no exception, and the failure mode is named here rather than inherited
silently — a disclosure with no `Act`.** `[thin]` Recorder-before plus fail-closed was weighed and
refused on two grounds. Append-only generates no `DELETE`, so a row falsely asserting that an admin
read a secret can never be retracted, and that is an unretractable false accusation. And fail-closed
would let a database fault deny the operator the debugging evidence ADR-0126 exists to preserve, which
inverts the corpus's purpose in order to protect a record of it. **Held at about 60/40.** The counter
is recorded rather than smoothed: among cooperating operators an over-record is explainable and an
under-record is a silent gap. Consistency does real work in the answer — 61 classes under one rule is
legible, and 60-plus-one is not unless the exception is very well grounded. **Reopenable on that
ground and on no other.**

**§5.2's disclaimer is restated at limb 4 specifically**, because this is the limb most likely to be
misread as forensics. See §11.

**Reopening condition for the wording:** a secret-bearing value **not** held in a corpus the model
seals, disclosed to a principal who does **not** own it, falls outside limb 4.

### 1.5 Two firsts limb 4 introduces

- **The first auditable act on a `GET`** — two routes, one handler. A `POST`-only harvest misses it,
  and misses the SSO link callback too (§7.2).
- **The first act whose row count is driven by a held key rather than by a person deciding to change
  something.** A refresh, a back button or a prefetch re-fires the handler. `[thin]` This does not
  overturn §5.1 — its ground is that an `Act` is a row count — but the SPEC states it rather than
  letting an operator find it.

### 1.6 What the predicate does not reach

**A migration is not a principal, so an upgrade writes no `Act`.** A host operator running
`docker compose up` proves nothing to the instance — no session, no grant, no account. The exemption
stands on **the absent principal** alone. Do **not** re-cite legibility for it: `logVersionStamp`
writes a build stamp to stderr and `goose_db_version` renders on no console surface.

Three measurements support it, and each closes a route a later session would try:

- **The exemplar was unreachable.** `25200_seal_channel_and_sso_secrets.sql` can never write an `Act`
  on any install, because the `act` table arrives in a migration numbered above `25300`. On an
  existing install `25200` is already applied. On a fresh one it runs before the table exists.
- **Neither producer seam is sound.** SQL inside the migration buys free atomicity (goose wraps each
  migration in a transaction) and pays by hand-writing `subject JSONB` outside §4's exhaustive
  encoder, where §7's gate can never see it. Go in `migrateUp` keeps the encoder, but `migrateUp`
  knows only which version numbers it applied — its only expressible subject is a `goose_db_version`
  row rendered a second time.
- **A restore never runs `goose.Up`.** `cmd/web/restore.go:204-212` refuses any archive on a different
  schema version.

**There is no migration-side gate, and this SPEC says so** rather than letting a reader infer that
§7's AST test covers `db/migrations/`. It parses `cmd/web`.

**Reopening condition, exactly one:** a migration that runs on an authenticated principal's
instruction. If a `POST /settings/migrate` route ever ships, the admin **is** the principal and the
act is limb 1 or 2 with actor `Account`. **Destructiveness does not reopen it.**

---

## 2. The act surface — 61 classes, 23 exemptions

**Counts, and how they reconcile.** #1789 measured 63 auditable acts across 59 classes.
#1791's harvest corrected the arithmetic to **64 auditable route patterns behind 61 distinct handlers,
60 classes** — #1789 folded the two raw routes into one act and then subtracted the raw pair a second
time. **Every exemption and every limb assignment in #1789 stands. Only the headline numbers move.**
#1790 then added one class (the two-dial split, §2.2) and one conditional class
(`migration.applied`), and #1805 removed the conditional one. **The stored union has 61 variants.**

By limb, in **classes**: limb 1 · 32, limb 2 · 20, limb 3 · 7, limb 1+2 · 1, limb 4 · 1.

### 2.1 The 61 classes

The `Actor` is `Account` unless the row says otherwise. `Action` is the rendered label. `class` is the
stored `action` token. `Subject` shows the rendered cell with sample values.

| limb | route | class | Actor | Action | Subject |
|---|---|---|---|---|---|
| 2 | `POST /setup` | `setup.completed` | `GrantHolder(SetupToken)` | Instance set up | `root · admin` |
| 2 | `POST /reset` | `password.reset` | `GrantHolder(PasswordReset)` | Password reset | `alice` |
| 2 | `POST /invite` | `invite.accepted` | `GrantHolder(Invite{id})` | Invite accepted | `bob · viewer` |
| 3 | `POST /onboarding/finish` | `onboarding.finished` | | Scan dispatched | `hot` |
| 1 | `POST /seeds` | `seed.declared` | | Seed declared | `10.0.0.0/8` |
| 1 | `POST /seeds/delete` | `seed.withdrawn` | | Seed withdrawn | `10.0.0.0/8` |
| 1 | `POST /seeds/custody` | `seed.custody.moved` | | Custody moved | `example.com · custody extended` |
| 1 | `POST /seeds/zone` | `zone.declared` | | Zone declared | `example.com` |
| 1 | `POST /seeds/zone/interval` | `zone.cadence.set` | | Dial moved | `zone scan cadence · 24h` |
| 1 | `POST /seeds/dns/interval` | `dns.cadence.set` | | Dial moved | `dns scan cadence · 6h` |
| 1 | `POST /exclusions` | `exclusion.declared` | | Exclusion declared | `address 10.1.2.0/24` |
| 1 | `POST /exclusions/delete` | `exclusion.lifted` | | Exclusion lifted | `address 10.1.2.0/24` |
| 1 | `POST /settings/cold` | `cold.moved` | | Cold scan moved | `10.0.0.0/8 · cold opt-in` |
| 1 | `POST /settings/probers` | `vantage.declared` | | Vantage declared | `probe@edge-01.example.com:22` |
| 1 | `POST /settings/vantages/resolver` | `vantage.resolver.set` | | Resolver set | `local` |
| 1 | `POST /reports/schedule/new` | `schedule.declared` | | Schedule declared | `weekly exposure` |
| 1 | `POST /reports/schedule/{id}/edit` | `schedule.edited` | | Schedule edited | `weekly exposure` |
| 1 | `POST /reports/schedule/delete` | `schedule.withdrawn` | | Schedule withdrawn | `weekly exposure` |
| 1 | `POST /annotations` | `annotation.declared` | | Annotation declared | `10.0.4.9:443 · expired certificate` |
| 1 | `POST /annotations/withdraw` | `annotation.withdrawn` | | Annotation withdrawn | `10.0.4.9:443 · expired certificate` |
| 3 | `POST /proposals`, `POST /proposals/search` | `proposal.queried` | | Registries queried | `Example Holdings Ltd` |
| 1 | `POST /proposals/confirm` | `proposal.confirmed` | | Proposal confirmed | `198.51.100.0/24` |
| 1 | `POST /proposals/decline` | `proposal.declined` | | Proposal declined | `address 203.0.113.0/24` |
| 1 | `POST /proposals/undo-decline` | `proposal.decline.undone` | | Decline lifted | `address 203.0.113.0/24` |
| 3 | `POST /scans/trigger` | `scan.triggered` | | Scan triggered | `hot` |
| 3 | `POST /scans/stop` | `scan.stopped` | | Scan stopped | `dispatch 418 · hot` |
| 3 | `POST /scans/terminate` | `scan.terminated` | | Scan terminated | `dispatch 418 · hot` |
| 1 | `POST /verge-core/frequency` | `frequency.moved` | | Frequency moved | `port 8443 · add` |
| 1 | `POST /sources/toggle`, `POST /settings/sources` | `source.moved` | | Source moved | `crtsh · enabled` |
| 2 | `POST /profile/password` | `password.changed` | | Password changed | `alice` |
| 2 | `POST /profile/tokens` | `token.minted` | | Token minted | `ci reader (vga_7f21)` |
| 2 | `POST /profile/tokens/revoke` | `token.revoked` | | Token revoked | `ci reader (vga_7f21)` |
| 2 | `POST /profile/sso/unlink` | `sso.unlinked` | | SSO unlinked | `okta` |
| 2 | `POST /accounts` | `account.created` | | Account created | `bob · viewer` |
| 2 | `POST /account/totp/confirm` | `totp.enrolled` | | TOTP enrolled | `alice` |
| 2 | `POST /settings/accounts` | `invite.minted` | | Invite minted | `invite · viewer` |
| 2 | `POST /settings/accounts/role` | `account.role.moved` | | Role moved | `bob · admin` |
| 2 | `POST /settings/accounts/reenroll` | `totp.stripped` | | TOTP stripped | `bob` |
| 2 | `POST /settings/accounts/remove` | `account.removed` | | Account removed | `bob` |
| 1 | `POST /settings/channels` | `channel.declared` | | Channel declared | `hooks.example.com/services/T0/B0` |
| 1 | `POST /settings/channels/update` | `channel.updated` | | Channel updated | `hooks.example.com/services/T0/B0` |
| 1 | `POST /settings/channels/delete` | `channel.withdrawn` | | Channel withdrawn | `hooks.example.com/services/T0/B0` |
| 3 | `POST /settings/channels/test` | `channel.tested` | | Channel tested | `hooks.example.com/services/T0/B0` |
| 1 | `POST /settings/retention` | `transcript.currency.set` | | Dial moved | `transcript currency · 14 days` |
| 1 | `POST /coverage/retention` | `observation.currency.set` | | Dial moved | `observation currency · 30 days` |
| 1 | `POST /coverage/retention` | `dispatch.cadence.set` | | Dial moved | `dispatch cadence · 4` |
| 1 | `POST /settings/address-cap` | `address.cap.set` | | Dial moved | `address-scope cap · 1024` |
| 1 | `POST /settings/updates/check` | `update.check.moved` | | Update check moved | `update check · off` |
| 1+2 | `POST /settings/restore` | `restore.applied` | | Restore applied | `verge-2026-09-08.tar.zst · taken 2026-09-08T14:02Z` |
| 2 | `POST /settings/api` | `api.access.moved` | | API access moved | `API access · on` |
| 2 | `POST /settings/sso` | `sso.provider.declared` | | SSO provider declared | `okta` |
| 2 | `POST /settings/sso/update` | `sso.provider.updated` | | SSO provider updated | `okta` |
| 2 | `POST /settings/sso/secret` | `sso.provider.secret.set` | | SSO secret set | `okta` |
| 2 | `POST /settings/sso/delete` | `sso.provider.withdrawn` | | SSO provider withdrawn | `okta` |
| 2 | `POST /settings/sso/identity/remove` | `sso.binding.removed` | | SSO binding removed | `okta · bob` |
| 1 | `POST /settings/integrations/install` | `integration.installed` | | Integration installed | `pagerduty` |
| 1 | `POST /settings/integrations/remove`, `/disconnect` | `integration.removed` | | Integration removed | `pagerduty` |
| 1 | `POST /settings/integrations/channel` | `integration.channel.bound` | | Channel bound | `pagerduty · events.pagerduty.com` |
| 3 | `POST /settings/integrations/test` | `integration.tested` | | Integration tested | `pagerduty · events.pagerduty.com` |
| 2 | `GET /profile/sso/{slug}/link/callback` | `sso.binding.created` | | SSO linked | `okta · alice` |
| 4 | `GET /run/{id}/raw`, `GET /runs/{id}/raw` | `transcript.disclosed` | | Raw output disclosed | `job 9182 · run 412 · edge-01` |

**Four notes on this table.**

- **Two acts are not POST routes.** `sso.binding.created` is a `GET` (`cmd/web/sso.go:295`) and it is
  one of ADR-0113's three named audited acts. `transcript.disclosed` is a `GET` pair. A sweep of the
  81 POST registrations misses both. §7.2 turns on this.
- **`restore.applied` is written after `applyRestore` returns.** `applyRestore` truncates every backup
  table, the `act` corpus included (`cmd/web/restore.go:265+`), so a row written before the apply is
  erased by the apply. Its actor is `Account` — the restoring admin, whose username snapshot survives
  the `TRUNCATE` because §3 captures a value and not a join.
- **Five classes share the Action label `Dial moved`, and that is correct.** The Subject cell carries
  the dial name, so `Dial moved · transcript currency · 14 days` reads once and not twice. The
  **stored** `action` token stays distinct for all five.
- **Limb 4's label is `Raw output disclosed`, never *read*.** §1.4 worded the limb as a disclosure and
  refused the read register by name, so the copy follows or the interface contradicts the SPEC on the
  one act most likely to be misread as forensics.

### 2.2 `POST /coverage/retention` writes two rows

It moves the observation currency and the dispatch cadence in one submit. **Two dials, two rows.**
Folding both into one Subject cell is the list-valued subject §4.1 bars, because Subject is a rendered
column. Price accepted: one submit writes two rows, and a submit that moves only one dial writes one.

### 2.3 The 23 exemptions

Each carries its reason. A later session that re-derives the surface will find these tested, not
missed.

**Login family — 11 acts, ruled in §1.2.**

`POST /login` · `POST /login/totp` · `POST /logout` · `POST /signout` · `POST /forgot` ·
`POST /profile/session/revoke` · `POST /profile/sessions/revoke` ·
`POST /profile/sessions/revoke-others` · `POST /settings/sessions/revoke` ·
`POST /settings/sessions/revoke-account` · `GET /login/sso/{slug}/callback`

Two riders. **The two admin session revokes are the hardest of the family** — cross-principal,
admin-only, one of them typed-name-confirmed (`cmd/web/settings.go:978,997`). They stay exempt for
consistency, because the target's grant is untouched. A SPEC that wants them in wants a session-event
log and should say so rather than smuggling one in through limb 2. And **the SSO login callback
creates no binding** — an unlinked identity is refused (`cmd/web/sso.go:249`). It establishes a
session only.

**Staged acts — the second step is the act.**

| exempt | reason |
|---|---|
| `POST /account/totp/enable` | Stages an encrypted secret with `totp_enabled` still false (`cmd/web/auth.go:980-1009`). The confirm is the act |
| `POST /settings/restore/preflight` | Stages the archive in process memory only (`cmd/web/restore.go:222-229`). A reload abandons it. The apply is the act |
| `GET /profile/sso/{slug}/link` | Writes a tx cookie and redirects. No store write. The callback is the act |

**Read-only and per-account state.**

| exempt | reason |
|---|---|
| `POST /onboarding` | Wizard step navigation; writes nothing (`cmd/web/onboarding.go:127`) |
| `POST /seeds/preview` | Renders a withdrawal receipt into a flash and writes nothing (`cmd/web/seeds.go:285`) |
| `POST /exclusions/preview` | Read-only preview (`cmd/web/exclusions.go:76`) |
| `POST /messages/read`, `/read-all`, `/unread` | Per-account read state; already legible to its own reader |

**Two acts exempt on a measurement, not on convenience.**

- **`POST /settings/backup`.** It changes no declaration, changes no access, and directs no network
  act. It writes only `SetLastBackup`. The tempting counter — *a backup is a `Transcript` read at
  maximum scale, so limb 4 reaches it* — is **false against the tree**: `transcript` is on
  `backupExcluded` (`cmd/web/backup.go:80`), and `account.totp_secret`, `channel.secret` and
  `sso_provider.client_secret` are redacted to `null` on the way out (`:88-97`). ADR-0124's *a backup
  carries data and no credential* holds.
- **`POST /reports/schedule/run` — exempt by one line, with a tripwire.** Today it inserts one
  `report_delivery` at state `generated` and pushes nothing to a channel
  (`cmd/web/reports_schedule.go:383-428`). **The day "run now" actually delivers it becomes limb 3**,
  the same class as `POST /settings/channels/test`. Name it at review.

**Session expiry is not an act at all.** No handler runs. `requireLogin` refuses and redirects. There
is no principal and no site to hang a recorder on.

### 2.4 The bearer path expresses no act at all

`cmd/web/api_v1.go:27-36` mounts exactly six patterns and every one begins `GET `: inventory,
subjects, drift, signals, coverage, and `apiNotFound`. There is no POST, no PUT, no PATCH, no DELETE
and **no method-less pattern** — the comment at `:27` records why (*"A method-less pattern neither
dominates nor is dominated by GET /, which net/http refuses"*), so the absence is load-bearing.
`mountAPIv1` is the only other registrar in the tree.

Two consequences:

1. The bearer path can express **no** limb-1, limb-2 or limb-3 act, so **a leaked token can never
   forge a row.** This is ADR-0123's inexpressibility property, reused.
2. It also cannot reach the one auditable read — `mountAPIv1` serves no `Transcript`. So the
   inexpressibility is **total**, and no `Actor` variant is needed for the bearer at all.

---

## 3. The `Actor` — a closed union of two members

```
Actor =
  | Account      { account_id, username_snapshot }
  | GrantHolder  ( Grant )

Grant =
  | SetupToken
  | PasswordReset
  | Invite { invite_id }
```

**Never a nullable `account_id`**, because null is a shape and not a meaning. ADR-0126 set this idiom
for `Transcript` — *"a closed union … never a record with optional fields"*.

### 3.1 The discriminator, and its grain

The union discriminates on **how the principal proved themselves to the instance** — a session or a
grant — never on **who the row is about**. Reading it the second way makes `Account` assert two
different claims in two different rows, which is the ambiguity a `NULL` was rejected for.

**The grain is the kind of proof, never its method or strength within a kind.** A password session, an
SSO session and a TOTP-stepped session are all one `Account`. A finer grain smuggles the exempted
login-family distinction back onto every row. An SSO link is already its own auditable act, so the
fact lives in the corpus as a row and not as a variant.

### 3.2 What a row of each kind asserts

- **`Account`** — *the holder of a signed-in session for this account took this act.* It makes no
  claim about how that session was obtained.
- **`GrantHolder(SetupToken)`** — *whoever held the first-boot setup token took this act.* No account
  existed. The instance cannot say who read the token out of its own logs.
- **`GrantHolder(PasswordReset)`** — *a holder of a valid single-use reset grant took this act.* It
  does **not** assert that the named account's owner did.
- **`GrantHolder(Invite{id})`** — *a holder of the single-use invite grant with this id took this
  act.* The account the act created did not take it.

### 3.3 Four rulings the union carries

**`/reset` is `GrantHolder(PasswordReset)`, not `Account`.** Measured: the reset link rides the
instance's own web log — `db/migrations/21400_password_reset.sql:8-10` says the plaintext is *"written
to the web logs, exactly as the first-boot setup token is"*, gated by `VERGE_LOG_RESET_LINKS`
(`cmd/web/auth.go:1102-1104`). There is no mail on a self-hosted install, so the log reader and the
account holder are often different people. A row rendering `alice` would assert that alice reset her
own password when the host operator may have taken over her account. **A wrong attribution is worse
than a vague one.**

**`TokenBearer` is barred as a name.** "Bearer" has one meaning in this tree — ADR-0123's API token
path, the one principal that mints no `Actor` at all. "Grant" is already the shipped noun for both
objects (`21400_password_reset.sql:1-2`, `21600_invite.sql:1-2`), and it carries its own disclaimer: a
row asserts that *a holder* acted. The collision with ADR-0003/0018's consent grant is real and
accepted — that grant sits in the third-party source domain and this one in the auth domain, where the
migrations already use the word.

**The correlation rule decides which grant carries an id:** *a grant variant carries the grant's id
exactly when a second `Act` names the same grant.* `Invite` carries one. `POST /settings/accounts` is
audited, so the id joins the mint to the acceptance. And `21600_invite.sql:19-21` uses
`ON DELETE SET NULL` on both FKs *"so an invite outlives either account's deletion as a record."*
`PasswordReset` carries none, because `POST /forgot` is exempt and a consumed reset row is deleted by
**every** `POST /forgot` (`db/queries/password_reset.sql:14-16`, fired at `cmd/web/auth.go:1088`).
`SetupToken` carries none, because **no row exists** — it is an in-memory string compared by
`auth.TokensEqual` (`cmd/web/auth.go:216`). That is a meaning, not a `NULL`. The rule is stated rather
than the three cases, so a fourth grant answers it on arrival.

**`GrantHolder` carries no account and `Account` carries no session id.** The Subject column already
holds the account for all three grant acts. Omitting the field makes the ruling structural: there is
no account inside the `Actor` to render by mistake. A session id would dangle the way a reset id does.

### 3.4 `System` was dropped, and the empty variant is the reason

`Actor` had a third member, `System`. **No migration ever writes an `Act` (§1.6), so the variant is
uninhabited.** A closed union exists so the encoder is exhaustive. A variant nothing constructs ships
unexercised by the round-trip test and makes an unreachable state compile. That is ADR-0126's own
objection, and the objection §4 uses to reject the product shape one level up. `actor_kind`'s `CHECK`
carries no `'system'`.

### 3.5 The rendered Actor cell — an authored mark on `Account`

An account renders **`@alice`**. The three grants render **`setup token`**, **`password-reset link`**
and **`invite 12`**, each as an `st-tag`.

| Actor | renders | component |
|---|---|---|
| `Account{7, "alice"}` | `@alice` | mono text, in the `<td class="mono">` it already has |
| `GrantHolder(SetupToken)` | `setup token` | `st-tag` |
| `GrantHolder(PasswordReset)` | `password-reset link` | `st-tag` |
| `GrantHolder(Invite{12})` | `invite 12` | `st-tag` |

**The problem this solves.** `account.username` is `TEXT NOT NULL UNIQUE` (`db/migrations/00002_accounts.sql:9`)
with no format check and no reserved list. `validateCredentials` (`cmd/web/auth.go:1876-1889`) caps
the length at 64 and requires non-empty, and that is all. So an account named `setup token` renders
identically to a `GrantHolder`. The collision is **adversarial**: `inviteAccept`
(`cmd/web/auth.go:1206-1210`) lets an invitee choose their own username while unauthenticated, before
they are an account at all. And it is unretractable — an `Act` is never deleted, so a false
attribution is permanent. (`session.ip` does not carry as a precedent: a `session` row expires and
CASCADEs, so an ambiguous rendering there is self-clearing.)

**Disjointness is structural, never enumerated.** An account named `setup token` renders
`@setup token`. An account named `@alice` renders `@@alice`. Neither equals a grant label, because we
author every grant label and none begins with `@`. The property survives a grant kind added later and
an install that already holds the bad account.

**The reserved-username list is refused on totality, never on cost, and the SPEC says so** — a later
session that re-prices it will find it cheap. The three account-creating sites (`setupSubmit`
`auth.go:224`, `createAccount` `:960`, `inviteAccept` `:1210`) all funnel through
`validateCredentials`, so the list costs **one `case`**. It loses because it cannot reach an account an
install already has. (Two of the ticket's premises were also wrong: `cmd/web/sso.go` provisions no
account — it links at `:338,351` — and `db/queries/accounts.sql` ships no `UPDATE account SET
username`, so a create-time rule would have no rename backdoor.)

**Five facts the implementation must hold.**

1. **The mark lives in a renderer, never in the store.** `username_snapshot` keeps `alice`. Storing
   `@alice` corrupts the datum and double-marks it on the next reader.
2. **One seam.** The mark sits in one named function, so a second consumer inherits it by default
   rather than by discipline.
3. **The mark reaches Actor and the three Subject payloads that render a bare username** —
   `AccountRef`, `AccountAtRole`, `BindingRef` — and stops at the corpus. Actor and Subject both take
   the mark, or one row renders one person two ways. It goes no further: on the Team tab the column
   header is `Member` and the row carries an avatar, so the mark buys nothing there.
4. **A table test holds the disjointness, and §7's gate cannot.** It asserts two halves: every
   `Account` render carries the prefix, **and** no authored grant label begins with `@`. The second
   half is what makes disjointness the property rather than the prefix — without it a future label
   authored as `@something` compiles, round-trips, renders cleanly, and reopens the collision from the
   other side.
5. **Reopening condition, exactly one: a renderer that emits a bare `username_snapshot`.** A CSV
   export, a notification or an API field that bypasses the seam reopens the reserved list. Nothing
   else does — not cost, and not a new grant kind.

**Two costs, stated rather than hidden.** `[thin]` `@` is an **infix** in this system today
(`{{.RuleID}}@{{.RuleVersion}}` at `design-system/templates/coverage.tmpl:183`,
`{{$p.Username}}@{{$p.Endpoint}}` at `settings.tmpl:513`), so prefix position is a new use of a
character that currently means *joins two things*. Judged acceptable, because the second operand is
absent. And the same person reads `@alice` on Audit and `alice` on Team. That is the price of holding
the mark to the corpus that needs it.

**Component choice, measured.** `st-badge` carries **state** in this system — `enrolled`, `enabled`,
`set` (`settings.tmpl:586,587,681`), each with a status dot. `st-tag` carries a **kind** —
`{{.Role}}`, `{{.Kind}}`, `{{.Class}}`. A `GrantHolder` is a kind. The Team tab already ships this
exact pairing one tab over: identity as bare mono beside kind as `st-tag` (`settings.tmpl:679-680`).

---

## 4. The stored value — a closed union of 61 variants

**One variant per act class, each naming its own typed subject.** Never a `subject_type TEXT` plus a
nullable column per kind — the shape ADR-0126 rejected by name.

```go
type Act interface {
	Class() string   // the stored action token
	Subject() string // the rendered Subject cell
	isAct()
}

// 61 marker types over 26 shared payload structs.
type SeedDeclared struct{ SeedScope }
type SeedWithdrawn struct{ SeedScope }
type SSOProviderSecretSet struct{ ProviderRef }

func (SeedDeclared) Class() string { return "seed.declared" }
```

**The alternative was weighed and rejected.** An `Action` enum of 60 beside a `Subject` union of 26
payload types halves the code and makes an illegal pair expressible — `seed.withdrawn` carrying an
`SSOProvider` subject would compile. That is the same defect `subject_type TEXT` plus nullable columns
has, one level up.

### 4.1 The row

`action TEXT` plus `subject JSONB` is **`transcript`'s `variant TEXT` plus `outcome JSONB` one corpus
over** (`db/migrations/23700_transcript.sql:36-42`), so the precedent is exact and already shipped.

```sql
CREATE TABLE act (
    id          BIGSERIAL PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_kind  TEXT NOT NULL CHECK (actor_kind IN ('account','grant_holder')),
    actor       JSONB NOT NULL,
    action      TEXT NOT NULL,
    subject     JSONB NOT NULL
);
```

Five rulings ride that shape.

- **No FK to `account`.** §5.4 gives the ground.
- **No timestamp on any variant.** `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` is the idiom all
  four sibling corpora share (`18802_measurement_dispatch.sql:16`, `20500_message.sql:47`,
  `20700_delivery.sql:39`, `23700_transcript.sql:60`) and 45 tables in `db/migrations/` use. **The
  caller cannot forge the time.** An `Act` therefore timestamps **the recording**, not the act.
- **No variant carries a list-valued subject.** One `Act` per subject, never one per request (§6.2).
- **The payload is typed and rendered at read time**, never a frozen sentence. `[thin]` Price stated
  rather than hidden: a later copy change re-renders history, because the row stores the fact and not
  the sentence. That is the right trade against an append-only corpus, where a copy change would
  otherwise need an `UPDATE` the corpus does not generate.
- **`action` carries no `CHECK`, and this diverges from `transcript` deliberately.** A 61-token
  constraint needs a migration per act class and fails at runtime rather than in CI. The exhaustive
  encoder and §7's gate hold the set instead. `transcript`'s 3-token `CHECK` is affordable at three.
  This one is not affordable at 61.

**Migration numbering.** The `act` table's migration must be numbered above `origin/main`'s current
max in `db/migrations/` at the moment the implementation ticket runs (`25300` as this SPEC is written).
A duplicate goose version panics the `web` binary and fails the `compose` CI job at *"wait for a
healthy stack"*.

### 4.2 Three hazards, tested

1. **A withdrawn subject.** `seed.withdrawn` renders `10.0.0.0/8` after the `Seed` is gone.
   `account.removed` renders `bob` after the account is gone, and the Actor renders `alice` after account
   7 is gone. **Nothing in the render path touches a store.** Where the handler holds only an id
   (`removeSSOBinding` has an identity id, `stopScan` a dispatch id), the recorder resolves the value
   **before** the delete and stores the resolved string.
2. **A secret being set.** `SSOProviderSecretSet` has exactly one field and it is the provider slug.
   There is no field the value could go in. The round-trip test also rejects any payload field that is
   not a `string` or an `int64`, so a `[]byte`, a `map` or an `any` cannot appear — those are the holes
   a secret fits through. **The honest limit:** this bars a *field* for the value. It cannot stop a
   caller putting a secret into a field named `slug`. The SPEC claims the weaker thing.
3. **A restore.** The discontinuity row is written after `applyRestore` returns and its subject is the
   whole corpus at that instant. The admin's username survives the `TRUNCATE` because §5.4 captures a
   value and not a join.

### 4.3 An `Act` may outlive its subject, by design

Limb 4's subject self-destructs: `Transcript` ships bounded at 14 days and is excluded from backups,
while the corpus is unbounded and never deleted. This is the same oddity §5.4 accepts for a deleted
account, and it is answered the same way — **carry the fact, never a pointer to it.** The subject
renders `job 9182 · run 412 · edge-01`, which still reads after the transcript expires. **A bare job
id degrades to a dangling number** the day the dial retires the row.

### 4.4 The rendered table

The **four** shipped columns are kept: **When · Actor · Action · Subject**
(`design-system/templates/settings.tmpl:869-876`). All 61 variants fill them with **no empty cell**, so
no fifth column is owed.

**Two facts the implementation must carry.**

- **The Subject cell needs the shipped ellipsis treatment.** `.st-table td` is `white-space: nowrap`
  and `.st-card.flush` is `overflow: hidden`, so a long channel URL or annotation subject key is
  **silently clipped**, not scrolled. The shipped precedent is on the same screen:
  `settings.tmpl:584` renders the SSO issuer as
  `overflow:hidden;text-overflow:ellipsis;max-width:220px`.
- **`When` renders as a relative stamp** (`4m`), the design system's rule
  (`design-system/README.md:47`), with ISO 8601 on hover. §8 · D.2 rules the ADR-0073 §4 collision this
  creates.

### 4.5 The drawn example, reconciled

`design-system/examples/console/Settings.jsx` draws the console's IA under ADR-0110. Three apparent
conflicts were ruled, and the amended example landed as
[#1807](https://github.com/winniel123/verge-asm/pull/1807).

- **A fifth `Source IP` column — out.** See §11.
- **A `DateRangePicker` — in**, as `?tab=audit&period=<token>`. See §6.2.
- **Two `system` rows folding `Dispatch` and `Delivery` events — out, and never a conflict.**
  ADR-0110's Decision ports composition and *"swap[s] only inline sample data for real data of the
  same shape."* Those two rows are entries in a `ROWS` array — inline sample data, exactly what the
  Decision licenses the port to replace. They also fail the predicate on its own terms: a `Batch`
  completing has no principal. **So #127's *"folding them is the error to avoid"* is confirmed and was
  never contested.**

**ADR-0116 does not bind here and must not be cited for this.** Its own supersession header marks
Decision clauses 1-3 — *"no dropped affordances, no added ones"* among them — **unsettled** (#1288,
#1300), with *"do not read those clauses as settled in either direction."* Only ADR-0110 binds.

---

## 5. Retention, append-only, and the restore

### 5.1 Unbounded, no dial, never deleted

ADR-0126's bounded-by-default precedent does **not** carry, and ADR-0126 says why itself: it ships
bounded because *"verbatim bytes are the volume hazard row counts are not."* **An `Act` is a row
count.** ADR-0041's instrument is the reader, and this reader is an accountability question with no
expiry. And **a dial an admin can move to zero is the audit log deleting itself on the admin's own
instruction.**

### 5.2 Append-only is a rule gated in CI, and it claims nothing more

**No `UPDATE` and no `DELETE` is generated against the corpus.** Not a trigger, not a restricted grant
— both advertise tamper-resistance this deployment cannot deliver, since the admin holds the database
credential.

The SPEC writes the disclaimer in #127 §7's own words. What this buys is **accountability among
cooperating operators, never forensics**, and *"nobody should reopen this on the compromise case
believing a table would have helped."*

**This is about whether the answer may be trusted against the principal holding the database
credential, never about which questions the corpus answers.** *Which of us three opened that
transcript* is an accountability question among cooperating operators. It becomes forensics only when
asked about someone who can edit the table, and the disclaimer already refuses that for all 61
classes.

### 5.3 Backup carries the corpus, and a restore records its own discontinuity

An `Act` holds no secret — ADR-0053's split is that *a secret being set* is auditable and *its value*
is not — so it is ordinary backup data, unlike `Transcript`. A restore replaces the corpus wholesale,
which is append-only defeated by the one operation that ignores the rule. Excluding the corpus from
backups is worse: a restore would silently erase the whole history. **So the restore writes an `Act`
recording the break**, after the truncation.

### 5.4 An `Act` outlives the account by carrying the name, not a foreign key

**No FK to `account`.** The row holds the account id **and the username as it stood at the time**.
`account.username` is `UNIQUE` but reusable after a delete, so neither alone is sufficient.

**The shipped precedent is rejected deliberately.** The attribution columns name no `ON DELETE`
action, so they restrict, and `removeAccount` (`cmd/web/settings.go:479-486`) catches the violation and
refuses the removal. Copy that, and once every admin act writes an `Act`, **no admin who has ever
acted can be removed at all.** An FK from an Operational record into the identity table also
re-couples what the Operational fence exists to keep apart.

§9 carries this reasoning one table over: the existing columns stop restricting too.

### 5.5 Dev mode writes no `Act`

ADR-0197 exactly — *a dev-mode worker produces no message, and the guard runs before the producer
reads or writes anything*. The guard runs **before** the recorder reads or writes, which is the
ordering that ADR exists to fix. Cite it. Do not re-derive it.

---

## 6. The reader — the admin `audit` tab

### 6.1 Admin only, and ADR-0173 is untouched

The `api` tab stays *"the single carve-out"*. `audit` is one of the thirteen identifiers that refuse a
viewer outright (`cmd/web/auth.go:105-107`).

**A viewer can take an auditable act** under limb 2 — an SSO self-link, a TOTP enrolment — **and still
gets no reader.** Those acts are already legible to that viewer on Profile, so a second carve-out would
buy a surface with no reader it does not already have. This is stated so a later session does not
re-litigate it.

### 6.2 The date range rides the query string

**`?tab=audit&period=<token>`.** This is not a free choice. ADR-0158 limb 4 — *"a screen that needs a
scope over the whole population needs a server-side predicate"* — bites, because §5.1's corpus is
unbounded and a client-side scope reaches only the rows already sent. The idiom is already ported
twice: `design-system/templates/drift.tmpl:101-111` and `reports.tmpl:109-111` both ship a
server-rendered `?period=` preset panel with an apply form.

**ADR-0173's rider does not bite.** *"A tab that grows a control loses the carve-out"* governs `api`.
`audit` holds no carve-out to lose. And `validTab` reads `Get("tab")` only (`cmd/web/auth.go:105`,
`settings.go:279`), so a second query key changes no gate and ADR-0173 §1's *one identifier admits one
section* property is untouched.

### 6.3 The empty state

`[derived]` Fact plus next action, the shape the design system requires. See §8 · E for the strings.

**The dominant cause is an upgrade, not a single-admin install.** #127 §6's *"there is only one of
you"* is **refused as the empty state**: it was an argument against shipping, and it is the wrong
diagnosis. `POST /setup` is itself an auditable act, so a fresh install holds a row before an admin can
open the tab. Three further causes are narrow — a restore truncates the corpus and then writes its own
row, an upgraded instance starts at zero, and §7.6's named failure mode drops a row.

---

## 7. The producer seam — an explicit recorder, gated by an AST conformance test

### 7.1 The seam is a recorder call, not middleware

There is no route table to hang middleware on: 81 literal POST registrations, 31 under `/settings`,
every one wrapped in `s.requireAdmin`, with the actor already arriving as the third argument
(`acct db.Account`). And **middleware cannot name the Subject**, which is a rendered column.

**One method name, two receivers.** The recorder is pool-bound by default. The restore path derives a
tx-bound recorder and calls the same `Record` on it. `Record(` is a free token tree-wide today, so the
one-search-token property is available and not merely hoped for.

**The recorder call sits in a `*server` method or a package-level func in `cmd/web`.** The walk follows
only `s.<method>` and package-level funcs in that package, so a call on another receiver or in another
package is unreachable to the gate.

### 7.2 The harvest widens past POST, and one fix is required

| harvest | routes seen | auditable non-POST acts seen |
|---|---|---|
| the `adr0130_contract_test.go` shape as written | 81 POST | **0 of 3** |
| widened to every method | 140 (81 POST, 59 non-POST) | 3 of 3 |

**Required: `apiBearer` must join `gateMethods`**, or every `/api/v1/*` route resolves to the handler
name `apiBearer`. This only bites once the harvest passes POST, which is why `adr0130` never had to
care.

Four registrations carry no `s.<handler>` name and drop out — `GET /seeds`, `GET /subjects`,
`GET /settings/vantages` (all `s.redirectTo` behind a gate) and `GET /api/v1/` (`apiNotFound`). None is
an act.

### 7.3 The `stop` set is part of the rule, not an optimisation

**One line inside `s.backToScope` took the passing set from 5 routes to 16 — eleven false passes for
one line.** Worse, `backToScope` is a refusal path, so that line would also break §7.5 while the gate
reported PASS.

**Adding `backHelpers` and `bodyAnswers` to the walk's `stop` set removes all eleven**, with all four
real call shapes surviving. `adr0130` already carries the same hazard in a comment at `:278`. Here it
is quantified. A SPEC that describes the stop set as an optimisation has mis-stated the rule.

### 7.4 A sink-keyed gate was built and rejected in both directions

The tempting rule is to drop routes and demand a `Record` wherever a mutating query is reached. The
mutator set is derivable exactly: `db/queries/*.sql` yields 274 named queries, 129 of them mutating.

- **29 routes mutate but are exempt.** `GET /`, `GET /signals`, `GET /asset/{key}` and four more reach
  `MintSignalInstances (INSERT)`. `GET /healthz` reaches `RecordHeartbeat`. `GET /login` reaches
  `TouchSession`. **This codebase mutates on read paths by design**, so the rule is wrong in kind, not
  merely noisy.
- **7 auditable routes reach no mutating query at all** — the two raw routes, the four limb-3 acts, and
  `POST /settings/restore`, which writes through `s.pool.Begin` and raw `tx.Exec` rather than sqlc.
  **Limb 3 has no local mutation by definition**, so no sink rule can ever reach it.

**The route-keyed gate stands, with a sink rule added for limb 4 only**, where it is exact:
`transcript.Open` is reached by exactly two routes tree-wide, both through `fillRawOutputView`, and by
nothing else.

### 7.5 What the gate proves, stated honestly

> The conformance test proves that every auditable act's handler holds a `Record` call somewhere on
> its live call graph, and that no exempt route does. It does **not** prove the call runs, runs once,
> runs after the mutation, runs only on success, or carries the right `Actor` or `Subject`. A handler
> that records the wrong subject passes. A handler that records inside a branch never taken passes.
> Ordering, cardinality and content are held by review and by unit tests, never by this gate.

Be honest about the strength: **this does not make the violation *inexpressible* the way ADR-0123's
bearer path does. It makes it fail CI** — for POST. For a non-POST act outside limb 4, it does not even
do that.

**Three coverage gaps the SPEC names.**

1. **A new auditable `GET` is not caught.** POST defaults auditable and **fails closed**. Non-POST
   defaults exempt behind a three-route allowlist and **fails open**. Only limb 4's `transcript.Open`
   sink fails closed. A future limb-2 act on a `GET` — another callback shaped like the SSO link —
   would ship unrecorded and silent.
2. **Non-route acts sit outside the mux walk.** The bootstrap and the restore are both routes, so both
   are covered. `goose.Up` is not — and §1.6 rules it writes no `Act`, so nothing is owed there.
3. **The five integrations routes register inside `if integrationsEnabled`.** The flag is
   `const … = true` (`cmd/web/integrations.go:33`), so they harvest today. The **handlers** compile
   unconditionally. Only the **registrations** are gated. If the flag flipped, the harvest would read
   their absence as compliance rather than as a gap.

**Ordering is not expressible and the SPEC must not claim it.** Only one of the five wired routes has
the mutation and the `Record` in the same function. `declareSeed` already splits them across two
functions, where a lexical rule is blind, and a single `defer` inverts lexical order. **The restore's
after-the-truncation rule needs its own unit test.**

**Dev mode needs no new work.** A `Record` call placed only inside an `if s.devMode` branch leaves its
route reading as missing, which is correct.

### 7.6 Transaction placement — the recorder runs after the act

**An `Act` does not commit in the same transaction as the act it records.** The recorder runs after the
mutation, on its own connection. The SPEC carries one named failure mode: **an act with no `Act`**, and
it states that this failure can pass unnoticed.

**Recorder-before is barred, not merely worse.** It produces an **`Act` with no act** — the log
asserting something that did not happen. Append-only generates no `DELETE`, so a phantom row **can
never be retracted**. Recorder-before builds a failure the corpus's own rule forbids you from
repairing.

**Atomicity was weighed twice and declined twice.** `[thin]` It cannot be **total** — limb 3's acts
have no local mutation to join — so the rule carries an exception either way, and one uniform shape
beat a split by limb. And §5.2 already fixes this corpus's honesty register. **The cheaper atomic
shape is named and declined at roughly 65/35:** one `inTx` helper on `server` over **one** narrow
interface naming the auditable mutations plus the `Act` insert — one adapter, roughly 60 methods wide,
not ~60 adapters. ADR-0149's objection was to the 178-method aggregate, so this is arguable rather than
barred. **A later session may reopen it on those grounds and no others.** The 60-method figure is an
estimate and was not measured.

**A shipped precedent for non-atomicity already exists**: `declineLookup` (`cmd/web/proposals.go:281-317`)
loops over N proposals, does two unrelated writes per iteration, and **bails mid-loop** via
`s.serverError`, leaving a partially applied batch already committed.

**Eight further rulings the recorder carries.**

1. **Uniform across all 61 classes.** One recorder, one call-site shape, one thing for §7 to find. A
   split by limb would put a limb classifier inside a conformance test.
2. **Only a successful act writes an `Act`.** The predicate is written in the past tense of doing. A
   refused act directed nothing, and an unbounded never-deleted corpus must not be writable by failure.
3. **One `Act` per subject, never one per request.** A request-level row needs a list-valued subject,
   which fights §4 and gives the Subject column a second thing to render. And `declineLookup` bails
   mid-loop, so per-subject rows are the only shape that can be true about a partial batch. **Price
   accepted: a 200-item decline writes 200 rows.**
4. **The restore's `Act` sits inside the restore transaction, after the replay** — one named
   exception. `cmd/web/restore.go:301` is the only handler in `cmd/web` that holds a transaction, so it
   is the only place atomicity costs nothing. Outside it, a crash between commit and recorder leaves a
   wholesale-replaced corpus with **no record of the discontinuity at all**.
5. **The recorder call uses a context detached from request cancellation**, with its own short
   timeout. On `r.Context()` the record dies when the operator navigates away, so *an act with no
   `Act`* would fire on ordinary use rather than on a database fault. **Without this the failure mode
   is mispriced by an order of magnitude.** `cmd/web/auth.go:1948` already reaches for
   `context.Background()` in the chrome render for the same reason.
6. **No retry.** One attempt on the detached context. Ruling 5 removes the dominant cause. What remains
   is a genuine database fault. A retry inside a request whose mutation has already committed charges
   the operator latency for a guarantee the SPEC declined to make, and a retry that also fails leaves
   the identical hole.
7. **A zero-jobs dispatch writes an `Act`.** The boundary is **whether `Trigger` was reached**, not
   how many jobs came out. An unknown scan, a disabled scan and an already-in-flight scan never call
   `Trigger` and write nothing. `n == 0` did call it, and a `Dispatch` exists, so the instance acted.
8. **`now()` is transaction-start**, so under ruling 4 the restore's `Act` is stamped at the moment the
   restore began, not when the replay finished. For a long restore that gap is minutes. That is correct
   rather than defective: **the row marks where the discontinuity starts.**

### 7.7 The notice cannot be made durable, and the SPEC says so

Any durable notice must go through the database that just failed. The honest ceiling is a process-local
toast plus a log line. The shape is closed on three sides, measured:

- A flash is barred by **ADR-0157 §4 in its own words** — *"a courtesy, never a record a restart must
  survive… Put nothing on it that the operator must be able to read back."*
- ADR-0157 §4's two prescribed alternatives both fail here. *The page the redirect lands on* renders
  from the database, and the failure is that the row is missing. *A `Message`* collides with ADR-0064's
  grammar — a message's subject is what the fold says moved, and an unrecorded act is not an estate
  fact.
- **`cmd/web` has no instance-fault surface at all.** `degraded` in the tree belongs to vantages and
  sources, never to the instance.

---

## 8. The withdrawal set — ADR-0058, site by site

**42 rows across 15 files and 2 closed issues — 41 sentences withdrawn plus #11's own sentence
restored** ([#1787](https://github.com/winniel123/verge-asm/issues/1787)), **plus 6 further breaking
sites from ADR-0093's limb 2** ([#1797](https://github.com/winniel123/verge-asm/issues/1797)). Four
rows are flagged borderline.

**The test, in the polarity this effort needs.** ADR-0058's test reads: *would this sentence, read
alone and in the present tense, cause a competent session to build the refusal — to leave the actor
off, to ship the empty tab, to defer the record?*

**Two sentence shapes fail that test and are not in the set**, and keeping them out is most of the
work:

- **A per-object refusal.** *"An `Annotation` carries no author."* *"The toggle row carries NO actor."*
  These stay **true**: no Declared term acquires an actor.
- **A refusal of consumption.** ADR-0007's *"nothing needs to know a release happened or consult an
  audit trail"* is **not withdrawn** and is now **load-bearing**: it is the reason a derivation may not
  read an `Act`. **Cite it. Do not touch it.**

What is withdrawn is the **generalising** sentence: *no operator act is written down anywhere*, *the
ruling is total*, *the audit facility is a repo-wide stub*, *there is no audit log*, *the operational
record is four corpora*.

Where a single sentence carries one clause of each kind, the row splits it.

### A.1 · #127's resolution — a posted note, as §8 itself did to #11

| Site | Verbatim (abridged) | Replacement |
| --- | --- | --- |
| §1 | *"…and that widening is **not made here**, because nothing joins the group."* | The widening **is now made**. Strike only that clause; the sentence already contains its own replacement text |
| §1 | *"Nowhere in the model is an operator act written down with an actor on it."* | "An operator act that meets the four-limb predicate is written down as an `Act`, with its `Actor`. No **Declared term** acquires one." |
| §1 | *"Named accounts create **identity**; they do not create a **log**."* | Withdrawn. Named accounts create identity; the `Act` corpus creates the log, joined by a stored username-at-the-time rather than an FK |
| §3 | *"So the ruling is all-or-nothing, and it is nothing."* | "…and it is **all**." §3's actual argument — a field true at creation and silently stale after the first edit — is **confirmed and survives**: it is why the `Act` records the act rather than the object |
| §5 | *"**An act log** … **Deferred**, on the consumer."* | "**Admitted.** Append-only, one row per act, no reconstruction of prior state, unbounded, no dial." The *Declared history* half of §5 is **confirmed** — see §12 |
| §7 | *"**#11's own two examples are unanswerable.**"* | Both are answerable — limb 1 and limb 3. Limb 3 exists **because** of this sentence |
| §7 | *"**v1 answers *why is this here* and never *who put it here*.**"* | `Citation` still terminates at a `Seed`, untouched. The `Act` answers *who put it here* from a corpus no derivation may read, so the answer is never reachable **through** the `Citation` |
| §8 | *"**The repository is clean.**"* | False on today's tree, and it stopped holding **within about two hours of its own sweep** (§8 ran 2026-08-15 04:25Z; `CONTEXT.md`'s `Span` site landed at 06:11Z the same morning). Every one of the 16 sites §1787's floor did not name was authored after it |
| §8 | *"**One site specifies it, and it is #11's own resolution.**"* | Amended: that note is itself withdrawn — see A.7 |
| §9 | *"**Reopens when the spec admits a second party who can mutate.**"* | The ruling is superseded **on the demand side**, not by this condition firing. §9 is not the trigger and must stop being read as the only one |
| §10 | *"**No `CONTEXT.md` change** … widening it now … would specify the thing this ticket declines"* | The restraint was correct and its condition has fired. §10 declined the widening *"for as long as nothing shipped"* |

**#127 §4 is confirmed, not withdrawn, and this is the sharpest non-listing in the set.** *"If it is
worth writing it is worth rendering; if it is not worth rendering it is not worth writing"* is a rule
this effort **satisfies**: the corpus and the tab ship together. **It is the reason the implementation
map may not ship a corpus with the tab deferred.**

### A.2 · ADR-0073

| Site | Verbatim (abridged) | Replacement |
| --- | --- | --- |
| `0073…md:44-45` | *"**No account, no name, no initials, no avatar, no "declared by" cell — not stored and not rendered.**"* | *flagged.* Outcome stands; narrow the scope to the object: "…on the `Annotation` **object** — not stored on it and not rendered beside it. The **act** of annotating is recorded as an `Act` with its `Actor`, in a corpus no derivation may read." |
| `:45-46` | *"…and **#127's ruling is therefore total rather than nearly total**."* | First clause confirmed. Strike the second — #127's ruling is reversed at the corpus and survives only over the Declared layer |
| `:75-76` | *"**So this is not an exception to #127 and not a second ruling.** … it reopens on #127's condition and on no other."* | **The load-bearing withdrawal in this ADR.** §1's outcome must be **re-grounded**, because its stated ground is gone: it stands on the field sitting inside a Declared term, the layer the probing gate reads, with the `Act` corpus as where operator identity lives instead. It no longer reopens on #127's condition, and it does **not** reopen because #127 has |
| `:241-244` | *"**#127's ruling is now total.** … holds without exception"* | Fully withdrawn: "#127's ruling is reversed at the corpus (#1786) and survives over the Declared layer. This ADR's §1 outcome is unaffected and now rests on its own ground." |

*Flag on `:44-45`.* Read as a rule about the `Annotation` object it is confirmed and needs nothing.
Read alone with no object in view — which is how ADR-0058 says to read it — *"not stored"* is
unqualified and a competent session takes it to mean *nowhere*. A session that judges the object-scope
obvious may reasonably drop this row.

### A.3 · `CONTEXT.md`

Four sites. The replacement text is in §9.

| Site | Verbatim (abridged) | Call |
| --- | --- | --- |
| `CONTEXT.md:33-35` | *"It records what the system *did*, never what is true of the estate. The comparison path may read nothing in it at all."* | Sentences 1 and 3 **confirmed** — sentence 3 is the group's load-bearing clause and is what makes an `Act` safe here. Sentence 2 widens |
| `CONTEXT.md:1796` | *"so the operational record is **four** corpora"* | **five**. The adjacent ordinals stay right: `Transcript`'s *"fourth Operational corpus"* names its own position and does **not** move, nor does `db/migrations/23700_transcript.sql:4` or `docs/spec/raw-job-output.md:55` |
| `CONTEXT.md:530-532` (`Annotation`) | *"So #127's ruling that no operator act is recorded with an actor on it holds here **without exception**."* | First sentence (*"the instant it was declared and no author"*) **confirmed**. Second replaced |
| `CONTEXT.md:1581` (`Span` closure) | *"It records **no actor**, which would be the operator-act record **the model refuses**…"* | Outcome **confirmed and now more important**. Re-ground on the fence: an actor there would put operator identity in the **Observed** corpus, one join from the comparison path |

**`CONTEXT.md`'s `Annotation` `_Avoid_` list (`author, declared by`) is confirmed, not withdrawn.** It
is that entry's own vocabulary bar, and `Annotation` still carries no author. What the `Act` needs is
its **own** `_Avoid_` list.

### A.4 · Shipped copy and code

Final strings are in §8 · E. The sites:

| Site | Verbatim (abridged) | Call |
| --- | --- | --- |
| `design-system/templates/settings.tmpl:867` | the audit tab lede | *"Who did what, when."* survives verbatim and becomes the whole lede |
| `settings.tmpl:882-883` | *"No audit log"* + the two substitute links | Becomes a genuine #47 empty state — *no acts recorded yet* — never a statement that the facility is absent |
| `settings.tmpl:898` | the Sources callout — *"it keeps no log line of its own; it is dated by the batch…"* | A toggle is limb 1 and writes an `Act`. **#1787's bare strike is not sufficient** — see §8 · E.4 |
| `design-system/examples/console/Sources.jsx:64` | the same sentence in the React example | Same replacement. ADR-0110 makes the examples the console's IA spec |
| `cmd/web/settings.go:918` | `// No queryable log exists, so this ships an empty state, never fabricated data (ADR-0110).` | **Deleted** when `fillAuditSection` reads the corpus |
| `cmd/web/annotations.go:57` | `// An operator dial carries no author, so neither act records who declared it (ADR-0073).` | Half stays true, half becomes false. Replacement, within the comment gates: `// The row carries no author; the act does (ADR-0073, #1786).` |
| `db/migrations/18500_source_state.sql:18-19` | *"ADR-0073 rules that no operator act is written down with an actor on it"* | *"carries NO actor and NO instant of its own"* **confirmed**. The gloss becomes: ADR-0073 rules that no **Declared term** carries an actor |

**`db/migrations/21200_integration_state.sql:17` is confirmed and needs nothing** — it states the
per-object refusal without the generalisation. `20400_annotation.sql` is amended by §8 · D.3 rather
than by this row.

**No `sqlc` regeneration is forced by the withdrawal set.** `sqlc` lifts comments from `db/queries/`,
not `db/migrations/`, and `internal/db` is clean of these sentences. The one ADR-0073 citation that
*is* lifted (`db/queries/annotations.sql:7`) cites §3/§4 and is untouched. **§9's `JOIN` sweep does
force a regeneration** — that is a different change.

### A.5 · Specs and guides

| Site | Verbatim (abridged) | Call |
| --- | --- | --- |
| `docs/guides/accounts.md:152-154` | *"There is no audit log of these acts … the **Audit** tab is honestly empty."* | **The baldest statement of the refusal in the tree.** Replaced wholesale when the tab ships. It is a paragraph rewrite against a shipped tab, not a string swap |
| `docs/guides/sources.md:140-143` | *"…which is where the audit trail lives … not by a log line on the toggle."* | *"no per-toggle history and carries no actor or timestamp of its own"* **confirmed** (the row). *"which is where the audit trail lives"* withdrawn — the batch dates the **estate-side** fact. The final clause withdrawn outright |
| `docs/spec/raw-job-output.md:385-387` | *"**Reads are unaudited.** … The audit facility is a repo-wide stub … **No audit-of-reads in v1.**"* | §1.4 removes the stated ground, so the deferral cannot be inherited. Replacement: a `Transcript` disclosure is the **one** auditable read, and every other read stays unaudited on a **named** ground rather than on the stub |
| `docs/spec/raw-job-output.md:522-523` | *"**Open, accepted.** Reads are unaudited (§5.4)…"* | Moves from *open, accepted* to *closed by #1786* |
| `docs/adr/0126…md:119` | *"Audit every read of a `Transcript` \| The audit facility is a repo-wide stub…"* | Withdrawn: **the alternative is now adopted**, and ADR-0126's own ground — the one corpus Postgres holds a secret for — is what selects it |
| `docs/adr/0126…md:124` | *"**Reads are unaudited in v1.** … Accepted residual risk."* | Withdrawn with `:119` |

### A.6 · ADRs that inherited the refusal as a ground

| Site | Verbatim (abridged) | Call |
| --- | --- | --- |
| `0093…md:147-148` | *"**No actor anywhere.** #127 and ADR-0073 §1 are untouched…"* | *"adds no field to any Declared term"* **confirmed**. Heading becomes "**No actor on a Declared term.**" |
| `0093…md:149-151` | *"**No operator-act record.** … it stays out on #127's own reopening condition."* | Withdrawn. The record is in scope and the actor column is **not** left out; what ADR-0093 was right to refuse is a **dated Declared history**, which #127 §5 refuses on principle and this effort does not reopen |
| `0093…md:220-222` | *"**Refused, and not by this ADR.** … #127 ruled it out of scope"* | Withdrawn on the same terms. See also §8 · D |
| `0087…md:266-268` | *"**An actor.** #127 ruled the operator-act record out of scope … Refused"* | **Outcome confirmed and now matters more**: a closure carries no actor. Re-ground on the fence — the `Span` corpus is Observed and derivation-readable, so an actor there is exactly the join the Operational fence exists to prevent. **Without this repair a session reads "#127 is reversed" and builds the `who`** |
| `0074…md:76` (table row) | *"An operator-act record \| **Untouched.** #127 stands…"* | *flagged.* *"this message names no actor and sits in no Declared term"* **confirmed**. *"#127 stands"* withdrawn. Borderline: it is an impact-table cell, not a rule |

### A.7 · #11, and the note #127 §8 posted on it

**#11's sentence becomes true again, and the note that struck it is now the site that specifies the
refusal.** The amendment is owed on **#11**, not on #127.

| Site | Call |
| --- | --- |
| #11, *Users and roles* — *"Named accounts exist for the **audit trail** — who changed the seed list, who launched a scan against production."* | **Restored.** Both examples are answerable — limb 1 and limb 3 |
| #127 §8's note on #11 (2026-08-15) | Withdrawn by a second note on #11. The 2026-08-15 finding was **correct then and is superseded now** |
| #11, *What this does not cover*, bullet 3 — *"Whether the audit trail is surfaced in the UI … Left as fog."* | #127 §8 discharged it *"with the correction that the data was never created."* That correction is withdrawn; the data is created and the surface is the `audit` tab |

### A.8 · Prototypes — 2 sites, flagged

`prototypes/signals-annotated/index.html:9-11` and `:869-873` both state the refusal as a rule.
**A genuine conflict of precedent, and this SPEC does not resolve it.** ADR-0075 says a prototype is
*a dated record of a reading, never of a rule*, which exempts both lines. But ADR-0093's Context read
two prototypes as *"drawn states a session would build from"* and treated them as evidence.
**Recommendation: a dated note at the top of the prototype, on ADR-0075's terms, and no edit to the
drawn markup.**

### B · Sites tested and left alone

Listed because each was a candidate, and because a later session that greps the same words will find
them.

| Site | Why it stays |
| --- | --- |
| `docs/adr/0007…md:264` — *"nothing needs to know a release happened or consult an audit trail"* | **Refuses consumption.** Now load-bearing: the reason a derivation may not read an `Act` |
| #127 §4 | This effort **satisfies** it. Cite it; do not touch it |
| #127 §5, second product — *A Declared history* | Nothing here reopens it. See §12 |
| #127 §7, fourth bullet — *accountability, never forensics* | Adopted verbatim (§5.2) |
| #127 §6 | Shipped exactly as described: own surface under `Settings`, no nav slot, #47's empty rule |
| `docs/adr/0073…md:186-189` — *"the option that lost: store the author, render nothing"* | This effort renders. The refusal of the store-only shape is untouched |
| `db/migrations/20400_annotation.sql:10-11`, `21200_integration_state.sql:17`, `docs/guides/integrations.md:54` | Per-object refusals. True |
| `docs/adr/0074…md:43-44` | Scoping about ADR-0074's own reach. `[thin]` **Low confidence — a strict reading would list it beside `:76`** |
| `docs/adr/0039…md:194` | The operator/system split survives; ADR-0039's point is untouched |
| `docs/adr/0092…md:86`, `CONTEXT.md:516`, `signals.tmpl:241`, `db/migrations/25000…sql:11` — *"with no operator act"* | **A different sense entirely** — *nobody did anything*, never *nothing was recorded*. Not sites |
| `docs/wayfinder/map-*.md` (8 files) | Dated archives, all last written 2026-08-14/15 |
| `docs/adr/index.json` | **Derived.** Regenerated by the `doclint` workflow. It moves when the ADRs move |
| `docs/adr/0173…md:65`, `cmd/web/settings.go:221`, `settings_test.go:84`, `settings_fixtures.go:381,504-505` | Plumbing, not prose. It changes when the corpus lands |

### C · The demand side — 18 sentences that become true, owing no withdrawal

**Not one of these is in the withdrawal set.** They are the contradiction, not the refusal, and they
are the ground §What-this-builds rests on.

`docs/guides/accounts.md:10` *"Every act in verge-asm has an author."* · `accounts.md:28` ·
`docs/guides/first-run.md:24` · `docs/guides/running.md:61` · `docs/guides/using.md:187` *"a toggle is
a dated, audit-trailed act"* · `docs/guides/api.md:37,:41` · `docs/spec/v1-spec.md:419-422` ·
`docs/spec/packaging-and-configuration.md:260-262`, `:297-298`, `:303-305`, `:313-314`, `:332-333`,
`:644` · `docs/adr/0053…md:97-99`, `:220-221` · `docs/adr/0113…md:76-78` · `docs/adr/0132…md:27-29` ·
`docs/research/safe-active-probing.md:1248` · `docs/adr/0159…md:71,:95,:145,:193`

Two riders. **The three `packaging-and-configuration.md` carve-outs strengthen** — each is granted
because *this object has no author, so it owes no audit trail*, and the test becomes a real test rather
than a hypothetical one. **No repair is owed.** And **ADR-0159 `:145` pre-authorises the shape of the
client-IP question**, so §11 cites `:145` rather than restating it.

### D · ADR-0093's limb 2 — restated, not withdrawn

**The `Act` corpus defeats ADR-0093's limb 2 read literally, and it would strip `Annotation`'s instant
the day the corpus ships.** Limb 2 today: *"Nothing else in the Observed or Operational corpus may
already date the act, **and** a consumer that exists in v1 must need the date."* An `Act` is an
Operational record, and under limb 1 the declaration of an `Annotation` writes one, carrying the act
and its instant.

**D.1 · The restated limb, as the SPEC carries it.**

> **Limb 2 — nothing the act moved dates it, and a named v1 reader needs it dated**
>
> Nothing the act **moved** may already carry the instant — a measurement the estate produced, or a
> record of the system acting on what the act changed — **and** a consumer that exists in v1 must need
> the date. Both halves are required: a wish is not a reader, and #127 §4 is the rule.
>
> **A record of the act itself is not residue.** The `Act` corpus transcribes every Declared act, with
> an instant. It dates no consequence, because it is not one. Were a transcript to count, this limb
> would be false for every Declared term at once, and the test could never fire again. The residue that
> defeats an instant is **what the act moved**.

One line is added above ADR-0093's enumeration: *"Every row acquires an `Act` the day that corpus
ships. No verdict moves, because a transcript is not residue."*

**`Annotation` keeps its instant, and no verdict in the eleven-row table moves.** Exactly one row was
ever in play: residue only ever **defeats**, so no Declared term gains an instant under any reading.

**Two candidates were named and refused, and the SPEC records why so they are not re-tried.**

- **The derivation fence** (*limb 2 excludes the `Act` because no derivation reads it*) — **refused on
  a measurement.** Limb 2 reads *"Observed **or Operational**"*, and **two of its own rows rest on
  Operational residue**: a `Message` dates a `Seed` narrowing, a `Delivery` dates a `Channel`. The
  fence never discriminated.
- **A reader-relative reading** (*what dates the act for its named reader*) — **demoted to a supporting
  fact.** It has a real measurement behind it: `GET /signals` is `requireLogin`, so a viewer reads the
  annotation date, while `audit` refuses a viewer outright. But it hangs a Declared field on a
  permissions gate — open the audit tab to a viewer one day and `Annotation` loses its instant. **A
  settings change must not strip a modelled field.**

**Stripping the instant loses twice.** ADR-0073 §2 stands unreversed, and an undated standing mute on
an object with no expiry cannot be reviewed at all. And it would make limb 2 unconditionally false, so
ADR-0093's reopening condition could never fire again.

**D.2 · ADR-0073 §4 is narrowed at its site, and the residual is stated rather than repaired.**
§4 requires `Annotation`'s instant to render *"mono, absolute, uncoloured"*, because *"accepted 412
days ago"* is *"an expiry the operator implements by eye"*. §4.4 renders `When` as a relative stamp, so
an `annotation.declared` row renders `412d` beside it. **The column stays uniform.** §4 rules the
`Signals` annotation list. It does not rule every surface on which an annotation's date may appear.

`[thin]` **The ground is not that the hazard is absent, and the SPEC must not claim that it is.** §4's
own word is *"**particularly** with a colour that deepens"*, so the colour is aggravation and the bare
age is the ground. A bare age does arrive here. Two of §4's four requirements are nonetheless absent —
no deepening colour, and an audit log sorts by recording time rather than by staleness — and the datum
differs, because `When` dates **the recording** and not the declaration. **A 1-in-61 carve-out to
pre-empt a reading is worse than the reading.** This follows ADR-0073's own precedent, which conceded
the keystroke objection in the same terms.

**D.3 · The six breaking sites, two amended for form, four confirmed.**

| Site | What breaks |
| --- | --- |
| `docs/adr/0093…md:96` | limb 2's heading |
| `docs/adr/0093…md:97-101` | *"Nothing else in the Observed or Operational corpus may already date the act"* |
| `docs/adr/0093…md:123-126` | *"an `Annotation` can live its entire life with no dated residue anywhere in the model"* |
| `docs/adr/0093…md:273-274` | the reopening condition, which restates the limb |
| `docs/adr/0073…md:99-105` | **ADR-0093's own amendment block inside ADR-0073 §2 restates the limb verbatim.** #1787's sweep did not reach it |
| `CONTEXT.md:44-45` | the Declared-layer preamble |

**Amended for form** (both survive literally, because an `Act` is none of the corpora they list, and
both still lead a reader wrong): `CONTEXT.md:535` — *"no `Message`, `Batch`, `Gap` or `revealed`
anywhere in the model dates the act"* — and `db/migrations/20400_annotation.sql:16-18` — *"no `Message`
anywhere in the model dates the act"*. **This amends §8 · A.4's row for that file.** No `sqlc`
regeneration is forced.

**Confirmed and needing nothing:** `docs/guides/using.md:193`, `docs/adr/0073…md:251`,
`db/migrations/18500_source_state.sql:18-19`, `21200_integration_state.sql:17`.

**An enumeration of corpora goes stale the next time a corpus lands. That is what just happened, and it
is why the restated limb names no corpus.**

**D.4 · One repair rides along.** Limb 2's corpus enumeration says *"Observed or Operational"*. The
custody-withdrawal row's residue is a `Gap`, a `Span` holding no value, and **every `Span` is Derived**
(`CONTEXT.md:28`). The incumbent wording never covered its own table. *"What the act moved"* covers a
residue at any layer, and the repair is a side effect rather than a claim.

**D.5 · A free strengthening is declined, and the cost is stated.** `[thin]` A literal reading would
have given every row unconditional residue, including `Channel` — ADR-0093's *"thinnest cell"*, whose
limb-2 residue is conditional because a `Channel` created and never delivered to has none. Under the
restatement that cell stays exactly as thin as ADR-0093 left it, with the same mitigation: every such
cell also fails on a limb that is not thin. **No new ticket is owed.**

**D.6 · The withdrawal record is not a Declared history.** `POST /annotations/withdraw` is auditable
and withdrawal **removes the row**, so the `Act` corpus becomes the only surviving record that a given
annotation ever existed. This does not reopen #127 §5, which refuses **reconstructible prior values**.
The two variants carry `<subject key> · <signal name>` and **no prose**, so nothing reconstructs a
withdrawn annotation's reason. **The limit, stated: the day any variant carries the prose, this becomes
a Declared history for the one Declared term that holds prose. The prose is the reopening condition,
and nothing else.**

### E · The five replacement strings

Written against the shipped `st-` block. Copy is `verge-asm-design`'s to confirm when the
implementation map lands.

**E.1 · The remove-member dialog** — `design-system/templates/settings.tmpl:743`, and its twins
`design-system/examples/console/Settings.jsx:456` and `design-system/examples/DocsPage.jsx:113`.

Replaces *"Their annotations and audit history stay attributed. Personal API tokens are revoked."*

> `.detail` — Their acts stay in the audit log under the name they held. Personal API tokens are revoked.

`.desc` is untouched. **The annotation clause is dropped, not repaired**: ADR-0073 §1 bars a "declared
by" beside that one object, so the honest sentence names the acts and says nothing about annotations.
*"Under the name they held"* is §5.4's own form — the `Act` captures the username as a value, so
removal can neither orphan the history nor be blocked by it.

**The secondary `.detail` is struck, not written.** #1792 drafted *"An account that has declared
scopes, channels or other estate objects cannot be removed. Reassign that work or keep the account."*
**§9 supersedes that ruling**: the refusal stops existing, so describing it would be false, and
*"reassign"* names **no affordance anywhere in the tree** — no query, no handler and no template
reassigns authorship.

**E.2 · The audit tab lede** — `settings.tmpl:867`. Replaces the whole `st-lede`.

> Who did what, when.

That is the entire lede. **The second clause is not repaired here.** *"Source enablement keeps no log
line of its own"* becomes false, and *"dated by the batch whose recorded source set it moved"* stays
true about a **different instant** — it dates the estate, not the act. A lede that explained both
instants would restate §8 · D in shipped copy.

`[thin]` A measured alternative was drawn and **declined**: *"Who did what, when. Each row is one act
by one principal. Nothing here is deleted."* Both added sentences are true and already settled. It
loses on #1787's ruling, and because a retention claim in the lede pulls the reader toward §12.

**E.3 · The empty state** — `settings.tmpl:882-883`.

> `.msg` — No acts recorded yet
>
> `.det` — This instance has taken no auditable act since the record began. Your next seed, scan or team change lands here.

The two substitute links go. **"Since the record began" is load-bearing**: it is honest on an upgraded
instance without naming a migration.

**E.4 · The Sources-tab callout** — `settings.tmpl:898`, and `Sources.jsx:64`.

> Every account can read this catalogue. Enabling or disabling is an admin act — the toggle lands in the audit log; the estate change it causes is dated by the batch whose recorded source set it moved.

**#1787's bare strike is not sufficient, and this is the amendment.** Striking only *"it keeps no log
line of its own;"* leaves *"Enabling or disabling is an admin act — it is dated by the batch…"*, whose
subject is **an admin act**. An `Act` carries its own `When`, so the bare strike leaves the same error
one clause later. The replacement gives the batch clause a subject that really is the estate.

**E.5 · The removal refusal** — `cmd/web/settings.go:483`.

Today: *"…has declared scopes, channels, or other **attributed acts** and cannot be removed — reassign
or keep the account."*

**§9 removes the cause, so the branch becomes dead.** `isForeignKeyViolation` **stays** as a generic
guard, and its string loses *reassign*, because no reassign path ships and §9 does not build one.

**E.6 · The guide** — `docs/guides/accounts.md`. `:152-154` is replaced wholesale against the shipped
tab (§8 · A.5). `:145-150` **drops its third refusal and keeps two** — the bullet documenting
restrict-and-refuse is gone under §9, not merely re-worded.

---

## 9. The attribution columns stop pinning the account

**Fifteen columns across twelve tables carry a `created_by`/`updated_by` FK to `account`. Fourteen
restrict. One already ships `ON DELETE SET NULL`.** All fourteen become **nullable
`ON DELETE SET NULL`**. **None is deleted, and nothing new is rendered.**

**The ground is already written in the tree**, at `db/migrations/24700_seed_withdrawal.sql:32-38`:
*"The attribution is worth keeping while the account exists and is not worth making a member
undeletable."* That sentence is the rule. `seed_withdrawal` is only where it first bit, and
`db/migrations/seed_withdrawal_test.go` already holds it there. §5.4 reached the same conclusion one
table over and stopped short of this one: **the `Act` corpus is what makes the removal safe**, because
it captures the username as a value.

Five of the fourteen are already nullable, so the change is one clause with no data migration. Nine
widen from `NOT NULL`.

### 9.1 The pair that may never be split

**Every attribution JOIN is an INNER JOIN.** So `SET NULL` alone does not orphan the object — **it makes
the object vanish from the result set.** Two consequences are behavioural, not cosmetic:

- **`verge_core`.** `ListVergeCoreFrequencyEditsWithAuthor` feeds `shipped.WithFrequencyEdits(edits)`
  (`cmd/web/settings.go:827-838`). A dropped row **silently reverts that port's frequency action** —
  *"a port you can hide is a signal you can silence"* (`docs/spec/v1-spec.md` §3.5).
- **`subjects`.** `FindCoveringAddressSeed` is `:one`. A dropped row returns `ErrNoRows`, so the subject
  **loses its `Declared · Seed` hop** and its citation chain reads unterminated
  (`cmd/web/subjects.go:296-311,434-449`).

**The FK widening and the `JOIN` sweep land in one change.** The sweep has three limbs.

1. **Delete six dead JOINs** — `db/queries/exclusions.sql:15`, `vantages.sql:24`, `zone.sql:21`,
   `verge_core.sql:12`, `proposals.sql:16` **and `seeds.sql:15`**. Each JOINs `account` only to select a
   username that Go then discards. Deleting beats widening.
   **`seed` keeps its JOIN on the subject-detail path and loses it on the scope path — say this
   explicitly, or a later session tidies it back.**
2. **`LEFT JOIN` five live ones** — `subjects.sql:314,323,333`, `channels.sql:12`, `sso.sql:14`.
3. **Three dial sites need no SQL.** They resolve the name by scanning `ListAccounts` in Go
   (`cmd/web/settings.go:807,1345,1394`) and already blank on no match.

**Two consequences for the implementation map.** `ListVergeCoreFrequencyEditsWithAuthor` is **misnamed
afterwards**, and every `db/queries/` edit **forces a `sqlc` regeneration in the same PR** — the `sqlc`
check runs `sqlc generate` then `git diff --exit-code -- internal/db`.

### 9.2 What renders

Six of the fifteen render today and **all six stay exactly as they are**. They answer *who authored
this object's current state*, which no `Act` answers. Those are different facts, so the corpus makes
none of them redundant.

| | Column | Render site |
| --- | --- | --- |
| **Rendered · 6** | `seed.created_by` | `subjectdetail.tmpl:117` via `cmd/web/subjects.go:303,441` — *prose* |
| | `sso_provider.created_by` | `settings.tmpl:588` "Declared by" — *table cell* |
| | `channel.created_by` | `settings.tmpl:1254` "Declared by" — *table cell* |
| | `instance_config.api_updated_by` | `settings.tmpl:1223` "Enabled by" — *prose* |
| | `instance_config.seed_address_cap_updated_by` | `settings.tmpl:420` "Last changed … by" — *prose* |
| | `retention_settings.updated_by` | `settings.tmpl:1508` "Last changed … by" — *prose* |
| **Joined, then discarded · 5** | `exclusion`, `vantage`, `zone_file`, `verge_core_frequency_edit`, `proposer_lookup` | none |
| **Never selected · 4** | `cold_scan_scope`, `report_schedule`, `instance_config.update_check_updated_by`, `seed_withdrawal` | none |

**An authorless object renders `removed account`** as an `st-tag` in the two table cells, and
**`a removed account`** as plain text in the four prose sites. **A blank is refused**: `channelView.At`
is declared at `cmd/web/settings.go:86` and never assigned, so `settings.tmpl:1254` already ships a
username beside a blank date, and it reads as a bug. *(A finding for the implementation map: that blank
is a pre-existing defect in the same cell the sweep touches.)*

**The by-clause is not dropped.** Every one of the six templates already guards on the instant, so a
NULL author beside a non-NULL instant can mean only one thing. The string distinguishes *the author is
gone* from *nobody ever moved this*, which is the honest use of a fact the `Act` corpus cannot recover.

**These labels sit on Declared-object cells, never on the `Act`'s Actor cell, and neither begins with
`@`, so §3.5's structural disjointness holds unchanged.**

### 9.3 One test holds the pair, and §7's gate cannot

**A test that parses `db/queries/` and fails on an inner `JOIN account` against an attribution
column**, in `db/migrations/seed_withdrawal_test.go`'s shipped style. §7's gate parses `cmd/web` and
detects a `Record` call. It can say nothing about a query's JOIN kind. **The guarded set is five
JOINs**, because §9.1 deletes six.

### 9.4 #127 §3 and #127 §4, both satisfied

#127 §3's objection — *"true at creation and silently stale after the first edit"* — reaches only the
editable objects. **The cells read "Declared by", which claims creation and never current state**, so
the objection misses them as worded. The three dials carry `updated_by`, which names the last mover.
**No split on the FK: §9 is uniform across all fourteen.**

#127 §4 is satisfied and **needs no column deleted**. Its rule was written against a tree with no
render surface for an operator act. The Audit tab is that surface. The nine unrendered columns are
**not** ruled worth deleting, because deleting a column is destructive and the `Act` corpus does not
cover the acts they already record.

### 9.5 The upgrade residual, named and refused

`[thin]` The `Act` corpus records forward only. On an upgraded install, removing an account takes the
authorship of every object it declared **before the corpus existed**, and no `Act` covers those.
**A backfill is impossible**: the acts were never observed, and inventing rows for them is the phantom
row §7.6 bars outright. **No reopening condition.**

---

## 10. `CONTEXT.md` — the changes the corpus owes

**This SPEC does not edit `CONTEXT.md`, and the ground is #127 §10's own.** §10 declined the widening
*"for as long as nothing shipped"*, because widening the charter to accommodate a record that does not
ship *"would specify the thing this ticket declines, which is ADR-0058's failure in the growing
direction."* **Nothing has shipped.** A glossary entry for a corpus with no table is that failure
exactly, and map #1786 rules implementation out of scope.

**So the replacement text is specified here and lands with the table**, in the implementation map's
first ticket. `docs/agents/domain.md`'s rule then binds that ticket: **a `CONTEXT.md` edit must not
fork on a branch**, and the PR carrying it merges promptly rather than waiting behind a review queue.

### 10.1 The Operational charter widens — `CONTEXT.md:33-35`

Sentence 2 only. Sentences 1 and 3 are untouched, and **sentence 3 is what makes an `Act` safe here.**

> A fourth group, **Operational**, sits outside the table on purpose. It records **what happened at
> this install that is not a fact about the estate**. The comparison path may read nothing in it at
> all.

That phrasing is #127 §1's own.

### 10.2 The corpus count — `CONTEXT.md:1796`

*"so the operational record is **four** corpora"* → **five**, with `Act` named beside `Transcript`.

**Three adjacent ordinals stay right and must not be moved:** `Transcript`'s own *"fourth Operational
corpus, beside `Dispatch`, `Message` and `Delivery`"*, `db/migrations/23700_transcript.sql:4`, and
`docs/spec/raw-job-output.md:55`. Each names `Transcript`'s position, not the count.

### 10.3 The `Annotation` entry — `CONTEXT.md:530-532`

First sentence confirmed. Replace the last:

> So no **Declared term** records an actor, this one included. The act of annotating is recorded as an
> `Act` in the Operational group, which no derivation may read.

`CONTEXT.md:535`'s corpus enumeration is amended for form under §8 · D.3. The entry's `_Avoid_` list
(`author, declared by`) **stays** — it is that entry's own bar, and `Annotation` still carries no
author.

### 10.4 The `Span` closure — `CONTEXT.md:1581`

Outcome confirmed and re-grounded:

> It records **no actor**, which would put operator identity in the **Observed** corpus, one join from
> the comparison path. The record exists; it lives on the Operational side of the fence and nothing
> derived may read it.

### 10.5 The Declared-layer preamble — `CONTEXT.md:44-45`

Carries ADR-0093's limb 2 and takes §8 · D.1's restatement.

### 10.6 The new `Act` entry

Placed in the Operational group beside `Transcript`. It must carry six things. The four-limb predicate. The
`Actor` union. The fifth-corpus position, and *one recorded act by one principal on this instance*.
The Declared-act collision named rather than hidden — **a Declared act is the doing, an `Act` is the
Operational record of the doing**, which is `Dispatch`'s relationship to a `Scan` firing. Unbounded,
append-only, no dial. And its own bar:

> `_Avoid_`: audit entry, log line, event, activity, history, ledger.

`log` is already barred by `Transcript`, and `event`/`activity`/`history` by `Message`. `AUDIT-LEDGER`
is a recorded dead token (`docs/spec/comment-policy.md:1166`). **The interface label stays "Audit
log"**, on the `Asset` precedent — *acceptable as a collective noun in the interface, never as a
modelled thing*.

---

## 11. Decisions that owe an ADR — named, not opened

An ADR records one decision that passes three tests: **hard to reverse**, **a reader without context
would ask why**, **chosen over a named alternative**. This SPEC applies them and **opens no issue**.
Only a human opens the issue that becomes an ADR, and its number becomes the ADR's number
(`docs/spec/adr-governance.md`).

Seven candidates pass. They are ordered by what a later one depends on.

| # | Decision | Hard to reverse | Reader asks why | Named alternative |
|---|---|---|---|---|
| 1 | **An operator act is recorded as a fifth Operational corpus, under a four-limb predicate** — reversing #127 and narrowing ADR-0073 §1's totality to the Declared layer | A shipped, never-deleted corpus and 42 withdrawn sentences | #127 refused this on the record and ADR-0073 called the refusal total | #127 §5's deferral, and #127 §9's reopening condition as the trigger |
| 2 | **The recorder is not atomic with the act it records**, with one named failure mode — an act with no `Act` | The call-site shape of 61 handlers | Every audit log a reader has met claims atomicity | The one-`inTx`-helper shape, priced and declined at 65/35 (§7.6) |
| 3 | **One `Act` per subject, never per request** | Reversing it after the union ships rewrites the union | A 200-item decline writing 200 rows looks like a defect | A request-level row with a list-valued subject |
| 4 | **What counts as residue** — ADR-0093 limb 2 restated on the nature of the residue, so `Annotation` keeps its instant | ADR-0093's eleven-row table depends on it | Limb 2 read literally strips a modelled field the day the corpus ships | The derivation fence, and the reader-relative reading (§8 · D) |
| 5 | **A migration is not a principal, so an upgrade writes no `Act`** — with the empty variant as its consequence, never its subject | A shipped corpus that never records upgrades | An audit log with no system actor at all is unusual | #1789's destructiveness split, refused on the axis |
| 6 | **An Operational corpus makes its own rendering unambiguous rather than constraining a Declared-neighbouring term** | The refusal compounds — every day without a username rule adds accounts no later rule can reach | A reader finds `@alice` on Audit and `alice` on Team | A reserved-username list, costed at one `case` in `validateCredentials` |
| 7 | **An `Act` corpus buys the right to stop pinning an account to the objects it declared** | A widened column does not re-tighten while a NULL exists | It reverses a shipped refusal that a guide documents as a feature | Keep restrict-and-refuse |

**Two riders on this table.**

- **Candidate 4 carries the ADR-0073 §4 narrowing as a Consequences bullet and a site marker, not as a
  second ADR.** It is not a second decision about the model. It is what the first decision costs once
  the corpus renders a date, and it does not arise at all if the instant is stripped. This is ADR-0093's
  own pattern in the same file. **Candidate 6 carries the `@` sigil the same way** — the ADR is on the
  refusal, never on the glyph.
- **Candidate 1 may be split by the human who opens it.** The corpus and the predicate are one decision
  as written here. A reviewer who wants the predicate's four limbs on their own record should say so
  before the ADR is authored, because splitting afterwards costs a supersession.

**ADR numbering hazard.** New ADRs race on their number and no check catches it. Before writing one,
read the number every open PR already claims:

```sh
gh pr list --state open --json number --jq '.[].number' \
  | xargs -I{} gh pr diff {} --name-only \
  | grep '^docs/adr/'
```

Number above `origin/main`'s current max **and** above every number that command prints.

---

## 12. What this does not buy

**Accountability among cooperating operators, never forensics.** #127 §7's own words, adopted verbatim,
and restated at limb 4 specifically (§1.4). *"Nobody should reopen this on the compromise case
believing a table would have helped."* The admin holds the database credential.

**No Declared history.** Reconstructible prior values — *what did the seed list look like on Tuesday* —
were refused **on principle**, not on cost: a second history mechanism over the one layer defined as
not drifting, and *"one `WHERE seed_version = …` in a drift query is `ScanRun` back through the front
door, on the input side, where it is worse."* **Nothing in this effort reopens it, and #127 §5's
warning holds: most of what people mean by "audit trail" is this half, and it never ships.**

**No client IP on an `Act`.** The ground is **§5.1, not ADR-0159.** ADR-0159's Decision rules
`clientIP`, the forwarding-header path, and its §4 names an audit column as a future consumer that
*"needs a ruling against this ADR before it ships"* — it gates, it never forbade. **Do not re-cite
ADR-0159 for this.** `sessionIP` (`cmd/web/auth.go:1858`) is a second shipped derivation that reads
`RemoteAddr` only and feeds `session.ip`, rendered on the admin Sessions tab — the same screen, the
same datum. The column loses because **an `Act` is never deleted**, so the address would be retained
permanently with no mechanism to remove it, while a `session` row expires and CASCADEs. The precedent
is the same datum under the **opposite** retention rule. On a fronted deployment `sessionIP` reads the
proxy for every row, so the column is constant and useless on exactly the deployments most likely to
want it. **Reopening condition: §5.1 acquiring a retention bound, and nothing else.**

**No upgrade-notice surface.** The gap is real: `25200_seal_channel_and_sso_secrets.sql` cleared every
`channel.secret` and every `sso_provider.client_secret`, every SSO provider stopped working, and the
operator was told **nowhere**. It is **not** an `Act`: §5.2 scopes this corpus to *accountability among
cooperating operators*, and a migration is not a cooperating operator, so there is nobody to hold
accountable. The need is a notice mechanism with its own retention rule, and it sits past this
destination.

**No repair of ADR-0073 §1's unattributed-dial census.** §1 reads *"Every other dial in the model is
unattributed: notification routing, the coverage alert threshold, flap suppression, a `Channel`, a
`Scan`'s cadence"*, and `CONTEXT.md:530` restates it. **`channel.created_by` ships and
`settings.tmpl:1254` renders it in a "Declared by" column**, and `retention_settings.updated_by` renders one
too. **The Decision is untouched** — §1 rests on what an author field would make the object be, plus
the consumer test, never on a census of the store. Only a supporting sentence is false, and **it was
false before this effort opened**, so it is a pre-existing defect and **not** a withdrawal-set row.
Reversing a supporting claim inside an accepted ADR whose Decision survives is an ADR-0058 withdrawal
this effort did not scope.

---

## 13. Build order

For `/to-tickets`. Each ticket is one session and one PR.

1. **The corpus.** The `act` table migration (numbered above `origin/main`'s max), the closed `Actor`
   union, the 61-variant value union with its exhaustive encoder and round-trip test, and the three
   hazard tests of §4.2. No handler yet.
2. **The recorder.** One method name, two receivers. A detached context. After the mutation. One `Act`
   per subject. Wired to a first slice of handlers.
3. **The AST conformance gate.** The widened harvest, `apiBearer` in `gateMethods`, the `stop` set, the
   limb-4 sink rule, and the exemption map. **This must land before the bulk wiring**, or the bulk
   wiring has no gate to fail against.
4. **The remaining handlers**, in limb order. Limb 3 and limb 4 last, because neither has a local
   mutation and both need their own review of ordering.
5. **The restore's own row**, with the after-the-truncation unit test §7.5 requires.
6. **The reader** — `fillAuditSection`, the four columns with §4.4's ellipsis treatment, the `@` seam of
   §3.5 with its table test, `?tab=audit&period=<token>`, and the empty state.
7. **The copy** — the five strings of §8 · E, plus `docs/guides/accounts.md` and `docs/guides/sources.md`.
   `verge-asm-design` first.
8. **The attribution-column pair (§9).** The 14 FK widenings **and** the `JOIN` sweep in one PR, with
   the `sqlc` regeneration, the `db/queries/` JOIN test, and the `removed account` strings. **Never
   split this ticket.**
9. **The withdrawal set (§8).** ADR sites, `CONTEXT.md`, the two closed issues, the guides, the specs,
   the prototype note. `CONTEXT.md` rides with ticket 1, not with this one (§10).

**Unblock [#1720](https://github.com/winniel123/verge-asm/issues/1720) when this SPEC lands** — it is
the SSO audit-write ticket, blocked on this chart since 2026-09-09.

---

## 14. Where this is thin, stated rather than smoothed

1. **Non-atomicity.** `[thin]` One named failure mode: **an act with no `Act`**, and the operator may
   never learn it (§7.6, §7.7). The cheaper atomic shape is named and declined at 65/35, reopenable on
   those grounds and no others. The 60-method figure is an estimate and was not measured.
2. **Limb 4's guarantee.** `[thin]` Held at about 60/40. Recorder-before plus fail-closed was refused.
   The counter — an over-record is explainable and an under-record is a silent gap — is recorded rather
   than smoothed. Reopenable on consistency alone (§1.4).
3. **The gate fails open on a new auditable `GET`.** `[thin]` POST defaults auditable. Non-POST defaults
   exempt behind a three-route allowlist. Only limb 4's sink fails closed. **A future limb-2 act on a
   `GET` would ship unrecorded and silent** (§7.5).
4. **The gate proves reachability and nothing else.** `[thin]` Not that the call runs, runs once, runs
   after the mutation, runs only on success, or carries the right `Actor` or `Subject`. **Ordering is
   not expressible at all.**
5. **The union bars a field for a secret. It cannot stop a caller putting one in `slug`** (§4.2).
6. **`action` carries no `CHECK`.** `[thin]` The exhaustive encoder and the gate hold the set. A hand-written
   insert outside the encoder is not caught by the database.
7. **ADR-0073 §4's residual is stated and not repaired.** `[thin]` An `annotation.declared` row renders
   `412d`, the bare age §4 bars for that datum. A 1-in-61 carve-out was judged worse than the reading
   (§8 · D.2).
8. **The `@` mark is held by one seam and a table test, never by the store.** `[thin]` A renderer that
   emits a bare `username_snapshot` reopens the reserved-username list (§3.5).
9. **The upgrade residual on §9.** `[thin]` Pre-corpus authorship is lost on removal and no backfill is
   possible.
10. **Four withdrawal rows are flagged borderline** — ADR-0073 `:44-45`, ADR-0074 `:76`, and the two
    prototype lines, where ADR-0075 and ADR-0093's Context cut against each other (§8 · A.8).
11. **The integrations gate.** `[thin]` Five routes register inside `if integrationsEnabled`. If the
    const flipped, the harvest would read their absence as compliance rather than as a gap.
12. **The corpus's own volume is a row count and nothing bounds it.** `[thin]` §5.1 accepts this
    deliberately. Limb 4 is the first class whose row count is driven by a held key rather than by a
    person deciding to change something.

---

## 15. Traceability

| SPEC section | Source | Ticket |
| --- | --- | --- |
| §1 the predicate | Settled #2 | [#1789](https://github.com/winniel123/verge-asm/issues/1789), [#1794](https://github.com/winniel123/verge-asm/issues/1794) |
| §1.4 limb 4 | Settled #9 | [#1794](https://github.com/winniel123/verge-asm/issues/1794) |
| §1.6 migrations | — | [#1805](https://github.com/winniel123/verge-asm/issues/1805) |
| §2 the act surface | Settled #2 | [#1789](https://github.com/winniel123/verge-asm/issues/1789), [#1791](https://github.com/winniel123/verge-asm/issues/1791) |
| §3 the `Actor` | Settled #5 | [#1795](https://github.com/winniel123/verge-asm/issues/1795), [#1805](https://github.com/winniel123/verge-asm/issues/1805) |
| §3.5 the Actor cell | — | [#1820](https://github.com/winniel123/verge-asm/issues/1820) |
| §4 the stored value | Settled #12 | [#1790](https://github.com/winniel123/verge-asm/issues/1790) |
| §4.5 the drawn example | Settled #12 | [#1796](https://github.com/winniel123/verge-asm/issues/1796), [#1807](https://github.com/winniel123/verge-asm/pull/1807) |
| §5 retention and restore | Settled #6, #7, #13, #14 | [#1787](https://github.com/winniel123/verge-asm/issues/1787) |
| §6 the reader | Settled #10 | [#1796](https://github.com/winniel123/verge-asm/issues/1796) |
| §7 the producer seam | Settled #11 | [#1791](https://github.com/winniel123/verge-asm/issues/1791) |
| §7.6 transaction placement | Settled #8 | [#1788](https://github.com/winniel123/verge-asm/issues/1788) |
| §8 the withdrawal set | Settled #1 | [#1787](https://github.com/winniel123/verge-asm/issues/1787), [#1797](https://github.com/winniel123/verge-asm/issues/1797) |
| §8 · E the copy | Settled #3 | [#1792](https://github.com/winniel123/verge-asm/issues/1792), [#1822](https://github.com/winniel123/verge-asm/issues/1822) |
| §9 the attribution columns | Settled #1, #8 | [#1822](https://github.com/winniel123/verge-asm/issues/1822) |
| §10 `CONTEXT.md` | Settled #3 | [#1787](https://github.com/winniel123/verge-asm/issues/1787) |
| §12 out of scope | map #1786 | [#1796](https://github.com/winniel123/verge-asm/issues/1796), [#1805](https://github.com/winniel123/verge-asm/issues/1805), [#1822](https://github.com/winniel123/verge-asm/issues/1822) |
