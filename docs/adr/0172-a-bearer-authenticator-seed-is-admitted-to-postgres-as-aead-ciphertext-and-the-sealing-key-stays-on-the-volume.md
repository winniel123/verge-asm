# ADR-0172: a bearer authenticator seed is admitted to Postgres as AEAD ciphertext, and the sealing key stays on the volume

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1334 ADR gaps: cmd/web/auth.go](https://github.com/winniel123/verge-asm/issues/1334), gap 3
- **Sweep PR that deleted the comment:** [#1337](https://github.com/winniel123/verge-asm/pull/1337)
- **Rests on:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) (the custody rule, and the volume-secret pattern the sealing key still obeys) and [ADR-0126](./0126-verbatim-job-output-is-a-fourth-operational-corpus-retired-by-a-duration-dial-that-ships-bounded.md) (the shape of an admission — AEAD at rest, per-value nonce, key on a service volume — and the first one made)
- **Narrows:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md) line 66, whose *"No other secret, and no key, is in Postgres"* is false of this tree and is withdrawn at its own site under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Not bound by:** [ADR-0160](./0160-a-backup-redacts-a-reversible-cleartext-credential-and-carries-a-hash-or-an-externally-keyed-ciphertext-and-restore-re-applies-the-same-redaction.md). **ADR-0160 rules the BACKUP posture. It does not rule the ADMISSION.** It is cited below for three facts it establishes and none of them is restated here

## Context

`account.totp_secret` is a bearer authenticator seed. A reader who holds it computes every future
code for that account, so it is a credential in the strongest sense this product has: it defeats the
second factor without a password and without touching the account.

**It is in Postgres.** The column is `TEXT` on the founding accounts table
(`db/migrations/00002_accounts.sql:12`), and that migration's header already asserted the split this
ADR rules — *"the key lives in the web-only volume, never here, but the per-account TOTP secret is
account state and belongs with the account"* (`:4-6`).

**Since [#337](https://github.com/winniel123/verge-asm/issues/337) the column holds ciphertext.**
`beginTOTPEnroll` seals the seed before it writes (`cmd/web/auth.go:909-919`), `loginTOTP` opens it
to verify (`:307-312`), and `totpConfirm` opens it once more to confirm the enrolment (`:945-950`).
Those are the only three sites. `cmd/web/sso.go:249-253` routes an enrolled account to the same
pending-cookie challenge, so single sign-on adds a route to `loginTOTP` and no fourth reader.

**The false sentence.** `docs/adr/0053-…:66`, after its ADR-0126 narrowing, still reads:

> **Narrowed by ADR-0126: the `Transcript` corpus is held in Postgres *encrypted* under a volume key
> that never enters it. No other secret, and no key, is in Postgres.**

A TOTP seed in Postgres is a second such secret, so that last sentence is **false of the tree
today**, and it has been false since 2026-08-23, when `636d383` landed #337 — six days before
ADR-0126 was drafted. Two further sites restate the exclusivity and inherit the defect:
`docs/adr/0126-…:21` calls the `Transcript` corpus *"the first secret-bearing data Postgres holds"*,
and `CONTEXT.md:1872` says the same. `account.totp_secret` was in Postgres before it, in cleartext
from the founding migration and as ciphertext from 2026-08-23. ADR-0126's **title** survives intact:
the `Transcript` is still the one *corpus* Postgres holds a secret for, and a per-account seed is not
a corpus.

**The carve-out was decided in an issue and in no ADR.** `gh issue view 337` returns state
`CLOSED`, title *"Secrets: totp_secret stored cleartext in Postgres (vs ADR-0053)"*. That issue
moved the column from cleartext to ciphertext and wrote the reasoning into four comments in
`cmd/web/auth.go`. The #1337 sweep compressed them to one line — *"The sealing key never enters
Postgres, so a table leak discloses ciphertext (ADR-0053)"* (`cmd/web/auth.go:908`) — which cites
ADR-0053 for the half ADR-0053 does state and leaves the admission unruled. Nothing under
`docs/spec/`, `docs/guides/` or `CONTEXT.md` states it either.

**What ADR-0160 establishes, and what it does not.** ADR-0160 records that `account.totp_secret` is
base64 XChaCha20-Poly1305 ciphertext (`0160-…:30`), that the TOTP key is not in Postgres because
`DeriveTOTPKey` derives it from the session signing key on the `web-state` volume (`:32-33`), and
that a restore rotates that key, so the archived ciphertext is unusable afterwards — filed as
[#1419](https://github.com/winniel123/verge-asm/issues/1419) (`:113-119`). It establishes those
facts in service of one question, what an export carries, and says so itself: *"It rules nothing
about the second-factor path, which is out of this ticket's scope."* **ADR-0160 rules the backup
posture. It does not rule the admission.** This ADR does, and does not restate those three facts.

## Decision

> **A bearer authenticator seed is admitted to Postgres. `account.totp_secret` holds
> XChaCha20-Poly1305 ciphertext under a 32-byte sub-key HKDF-derived from the session signing key,
> which lives on the `web-state` volume and never enters the database. This is the second admission,
> and the general rule is: a secret may be admitted to Postgres only when it is sealed under a key
> Postgres never holds AND no other store gives better custody of the act that reads it. The sealing
> key is never admitted. No secret is admitted in cleartext.**

### 1. The construction, named exactly

`DeriveTOTPKey` (`internal/auth/totpsecret.go:16-24`) runs HKDF-SHA256 over the 32-byte session
signing key with a nil salt and the info string `totp-secret-aead` (`:14`, `:19`), reading out
`chacha20poly1305.KeySize` — 32 bytes (`:18`). The nil salt is deterministic on purpose, so a
restart re-derives the same sub-key and decrypts rows written before it (`:17`). The label exists so
no key serves two purposes.

`EncryptTOTPSecret` (`:26-41`) builds an XChaCha20-Poly1305 AEAD (`:31`), draws a fresh 24-byte
nonce from `crypto/rand` per value (`:35-37`), prepends it to the sealed output (`:39`) and base64s
the result into the `TEXT` column (`:40`). There is no additional authenticated data.

The sub-key is derived once at construction (`cmd/web/handlers.go:271-277`) from the key
`auth.LoadOrCreateKey` reads or creates under `VERGE_STATE_DIR`, default `/app/state`
(`cmd/web/main.go:89-90`, `internal/auth/key.go:10-16`), which compose mounts from the `web-state`
volume (`docker-compose.yml:36`, `:94`). **A sub-key of a volume key is a volume key**, and the
database sees neither.

### 2. The condition for a third admission, stated so it can be applied

Two tests, both required.

| Test | What it excludes |
| --- | --- |
| **Sealed under a key Postgres never holds** | Cleartext, and any scheme whose key material or key-encryption key rides in a row. A dump must disclose ciphertext and nothing that opens it |
| **No other store gives better custody of the act** | Anything a service can generate in place and keep on its own volume. ADR-0053's SSH key and session key both fail this test and stay where they are |

The seed passes the second test on a property of the act: it is **per-account row state**, minted
during an authenticated web request and read by `web` on a later request for the same account. The
alternative store is a per-account file on `web-state` — a shadow schema keyed by account id, with
its own lifecycle bug for every account delete and restore. The row is the better custody, and the
seal is what makes the row admissible.

Two refusals are permanent and are not weighed against convenience:

- **The sealing key is never admitted.** Not the session key, not a derived sub-key, not a
  key-encryption key. That is ADR-0053's rule and this ADR keeps it whole.
- **No secret is admitted in cleartext.** The state #337 found is not a lesser version of this
  ruling; it is outside it.

### 3. A seed is sealed because verification needs the value back; a recovery code is hashed

The two second-factor secrets are held differently, and the difference is not an inconsistency.

A recovery code is verified by comparison. `newRecoveryCodes` bcrypts each code
(`cmd/web/auth.go:1170-1187` via `auth.HashPassword`, `internal/auth/password.go:8-14`), the column
is `code_hash TEXT NOT NULL` (`db/migrations/21500_recovery_code.sql:19`), and `redeemRecoveryCode`
compares the presented code against the stored hashes (`cmd/web/auth.go:347-369`). Nothing needs the
code back.

A TOTP seed cannot be hashed, because RFC 6238 verification **recomputes** HMAC over the seed and
the current step (`auth.VerifyTOTPStep`, called at `cmd/web/auth.go:312`). So the rule is: **hash
where verification is a comparison, seal where the act needs the value back, and admit nothing that
needs neither.** Sealing is the weaker protection and is spent only where hashing cannot do the job.

### 4. The cleartext's reach is one handler and one response, and it is enumerated

The cleartext exists in `beginTOTPEnroll` between `auth.NewTOTPSecret` and the write
(`cmd/web/auth.go:892-919`), in the response that handler renders, and transiently in `loginTOTP`
and `totpConfirm` while the verifier consumes it.

It reaches a **template** by design: `totpEnrollData` puts it in `Secret` and in the `otpauth://`
URI (`cmd/web/auth.go:923-936`), and `design-system/templates/signin.tmpl:216-220` renders the QR
and the manual-entry string. That is the enrolment act itself, and the QR is encoded in-process, so
the seed reaches no third party.

It reaches **no log**: no `log.Printf` in `cmd/web/auth.go` carries the seed or the ciphertext. It
reaches **no error string**: every failure path calls `s.serverError` with a fixed label
(`cmd/web/auth.go:894`, `:911`, `:947`, `:309`), which logs the error and returns the constant body
`internal error` (`:1887-1890`), and no error in `internal/auth/totpsecret.go` interpolates a secret
or a key.

The dev build pins the seed to a fixture (`cmd/web/auth.go:902`) and **still seals it** before the
write (`:909`). There is no dev exemption from the admission rule.

### 5. A decrypt failure is a fault, never a wrong code

`DecryptTOTPSecret` fails on a bad base64, a short input or a failed `Open`
(`internal/auth/totpsecret.go:48-63`), and every caller returns HTTP 500 rather than treating the
error as a verification miss (`cmd/web/auth.go:307-311`, `:945-949`). No caller falls back to
reading the column as cleartext, so a legacy pre-#337 row is a hard fault and the account re-enrols;
it is never quietly accepted as a seed. `TestTOTPSecretEncryptedAtRest`
(`cmd/web/hardening_test.go:188-219`) pins both halves: the stored value neither equals nor contains
the seed (`:210-212`), and it decrypts back to it (`:213-219`).

## Consequences

- **An attacker holding a database dump alone gets ciphertext.** The dump carries the base64 blob
  and its per-value nonce and no key material, so it yields no code for any account and does not
  defeat the second factor. It does carry `password_hash` and `token_hash`, and a weak password is
  still guessable offline from its hash — the second factor is what stands after that, and this
  ruling is what keeps it standing.
- **A write-capable attacker is not stopped by this, and was not going to be.** The seal carries no
  additional authenticated data, so it is not bound to the account id: one row's ciphertext pasted
  into another row opens under the same sub-key. Anyone who can write that row can also clear
  `totp_enabled` or replace `password_hash` (`internal/db/accounts.sql.go:167`, `:194`), so the
  missing binding prices at little. Adding the account id as AAD is a one-line change should a
  cheaper write path appear.
- **A lost `web-state` volume is a lost second factor for every account, and #1419 is that failure
  today.** The sub-key derives from the session signing key, so whatever rotates that key rotates it.
  A restore does exactly that (`cmd/web/restore.go:226`, `:395-398`), and the restored ciphertext was
  sealed under the old key, so every enrolled account fails `loginTOTP` with a 500 and redeems no
  recovery code at that step. **That lockout is a live consequence of deriving from the session key
  rather than holding an independent `totp.key`**; it is filed as
  [#1419](https://github.com/winniel123/verge-asm/issues/1419) and this ADR does not fix it. The
  contrast is on disk: `internal/transcript/key.go:13-32` keeps its own key file, which no restore
  rotates.
- **`docs/adr/0053-…:66` is withdrawn at its own site** under ADR-0058, and the two restatements —
  `docs/adr/0126-…:21` and `CONTEXT.md:1872` — take the same correction. A session reading *"No
  other secret, and no key, is in Postgres"* in the present tense would refuse this column and would
  be reasoning from a sentence the tree stopped satisfying in August.
- **No production behaviour changes.** The code has this shape. What changes is that the shape has a
  document, and that ADR-0053 stops contradicting it.
- **The enrolment response carries the cleartext and sets no cache directive.** No handler in
  `cmd/` or `internal/` sets `Cache-Control`, so that response is heuristically cacheable. It is a
  small, real cost of rendering the seed at all, and it is the one open edge this ruling leaves.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the seed in cleartext** — the state before #337 | A read-only dump then hands an attacker every future code for every enrolled account, which is the exact leak class ADR-0053 built the session-key rule against: *"a read-only leak of the database — a backup, a replica, an export"*. It also makes the second factor worth less than the password, whose bcrypt hash survives the same dump |
| **Hash the seed, as the recovery codes are hashed** | RFC 6238 verification recomputes HMAC over the seed, so a one-way hash makes the factor unverifiable. This is not a stricter version of the ruling; it removes the feature |
| **Keep the seed out of Postgres entirely — a per-account file on `web-state`** | A second store keyed by account id, with no foreign key, no cascade, and no transaction with the row it belongs to. Every account delete and restore gains an orphan case, and the enrolment write stops being atomic with `totp_enabled`. Against the dump threat it buys nothing the seal does not already buy |
| **Give the seed its own key file, as `Transcript` has** | This is the better shape and the reason is #1419: an independent `totp.key` would survive the restore that rotates the session key, and the lockout would not exist. It is refused **here** because it is a production change to the second-factor path with its own migration question — what happens to rows sealed under the derived sub-key — and #1419 owns it. This ADR rules the admission, which holds under either key source |
| **Envelope encryption: a per-account data key sealed under a KEK** | ADR-0053 refused this shape for the SSH key and the refusal still holds: the KEK needs a home, and its home is the volume the sub-key already lives on. Per-record crypto-shredding buys nothing for a value that is deleted with its row (`internal/db/accounts.sql.go:167`) |
| **Amend ADR-0053 in place and file no ADR** | Under ADR-0058's split an amendment carries a claim about the world that changed. This is a rule with a stated condition for a third admission, two permanent refusals, a rejected alternative still open as #1419, and a test a future column is measured against. ADR-0053's own rule is untouched and is confirmed by the sealing key's custody |
