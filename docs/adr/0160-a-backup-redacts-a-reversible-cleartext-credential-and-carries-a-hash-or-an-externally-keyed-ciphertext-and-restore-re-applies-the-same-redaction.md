# ADR-0160: a backup redacts a reversible cleartext credential and carries a hash or an externally-keyed ciphertext, and restore re-applies the same redaction

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1367 ADR gaps: cmd/web backup.go](https://github.com/winniel123/verge-asm/issues/1367), gap 1
- **PR that deleted the comment:** [#1366](https://github.com/winniel123/verge-asm/pull/1366)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Corrects, at the site that states it:** [`docs/guides/backup-and-restore.md`](../guides/backup-and-restore.md), which said the SSO and channel secrets ride with their rows. They do not. [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) requires the edit at the guide, not only here
- **Extends, and bounds one phrase in:** [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md) §1. That section rules that the archive holds no secret and grounds the rule on the two key volumes the export never opens. It names no column. Its *"no secret"* phrase, read alone, over-reads the archive, so it takes a bounding note at its own site per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on, and bounds one clause in:** [ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md), which rules where a secret is held. This ADR rules what an export does with a secret the database already holds. ADR-0053's #121 bullet — *"a database backup carries the whole estate and no credential … it is the reason a backup does not need to be treated as a keyring"* — is true in ADR-0053's own sense of *credential* and over-reads when read alone, so it takes a bounding note at its own site per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)

## Context

Two columns of the business schema hold a credential in reversible cleartext.

| Column | What it is | Where the operator sets it |
| --- | --- | --- |
| `channel.secret` | the webhook `HMAC-SHA256` signing key | its own write path, which never reads the value back ([`docs/guides/notification-channels.md`](../guides/notification-channels.md)) |
| `sso_provider.client_secret` | the OAuth confidential-client secret | its own write path, write-only at the interface ([`docs/guides/sso.md`](../guides/sso.md)) |

Both are typed by the operator and never read back to a screen. `ADR-0053` rules that shape. Neither
is a session-minting key, so `ADR-0053` §*The session signing key* does not reach either one.

Three more write-only values ride in the same tables.

| Column | What it is | Verified at |
| --- | --- | --- |
| `account.password_hash` | a bcrypt hash ([`internal/auth/password.go:9`](../../internal/auth/password.go)) | [`internal/db/models.go:17`](../../internal/db/models.go) |
| `personal_token.token_hash` | a token hash | [`internal/db/models.go:173`](../../internal/db/models.go) |
| `account.totp_secret` | base64 XChaCha20-Poly1305 ciphertext | [`internal/auth/totpsecret.go:26`](../../internal/auth/totpsecret.go) |

The TOTP key is not in Postgres. `DeriveTOTPKey` ([`internal/auth/totpsecret.go:16`](../../internal/auth/totpsecret.go))
derives it by HKDF from the session signing key, and that key lives on the `web-state` volume.

**The code already draws the line and nothing on disk states it.**
`backupRedactedColumns` ([`cmd/web/backup.go:77`](../../cmd/web/backup.go)) names the first two
columns and no others. `redactBackupRow` emits each named column as JSON `null`.
`dumpBackupTable` calls it on the export path. `applyRestore`
([`cmd/web/restore.go:293`](../../cmd/web/restore.go)) calls the same function on the restore path.

**The only citation the deleted comment carried was `#739`, and that issue is deleted.**
`gh api repos/winniel123/verge-asm/issues/739` returns HTTP 410.
[`comment-policy.md`](../spec/comment-policy.md) §8.3 rules that a deleted issue suppresses nothing.

**The two ADRs beside it do not state the rule.** ADR-0124 §1 rules that the archive holds no secret
and grounds that on the two key volumes. It never names a column and never rules that a hash may
ride. ADR-0053 rules custody, which is a different act from export. `comment-policy.md` §4.7 records
that a survivor citing ADR-0053 for a rule ADR-0053 does not state is already an open defect
([#1354](https://github.com/winniel123/verge-asm/issues/1354)).

**The operator guide states the opposite of the code.**
[`docs/guides/backup-and-restore.md`](../guides/backup-and-restore.md) said, twice, that the SSO and
channel secrets ride with their rows and that the archive carries the same leak posture as `pgdata`.
The first half is false for both columns. The second half is therefore too strong.

## Decision

> **The backup archive redacts a credential the database holds in reversible cleartext, and it
> carries a write-only value that is a one-way hash or a ciphertext whose key is not in Postgres.
> `redactBackupRow` is the one site of the rule, and the export path and the restore path both call
> it, so a redacted column crosses no archive in either direction. A restore lands the column NULL
> and the operator re-enters the value.**

Five limbs.

### 1. The test is reversibility from the file, not write-only status at the interface

All five columns above are write-only at the console. That property decides nothing here, because it
is a property of the screen and the archive is not a screen.

The question the export asks is narrower. **Does a reader who holds the file recover the secret?**
A cleartext column answers yes. A one-way hash answers no. A ciphertext answers no while its key is
outside the file and outside the database the file was dumped from.

A column that answers yes is redacted. A column that answers no rides verbatim.

### 2. Two columns meet the test today, and the list is closed by name

`channel.secret` and `sso_provider.client_secret` are redacted. No other column is.

The list is a literal, so a third such column is added by hand. A new cleartext credential column in
the business schema belongs in `backupRedactedColumns` in the same change that adds it.

The three riding columns stay. Each fails the test of limb 1 on its own ground, and the grounds are
not interchangeable. Two are one-way hashes. The third is ciphertext under a key held on a volume.

### 3. The redaction is one function, and both directions call it

The export redacts because a leaked file must not yield the webhook signing key or the OAuth client
secret. The restore redacts because an archive taken before the redaction landed still holds both
columns in cleartext, and replaying such a file would write a stale secret back into a live
database.

One function serves both. A second implementation would be a second thing to keep correct.

### 4. The cost is a named re-entry, and the operator is told before the restore, not after

A restore truncates the tables the manifest names and replays the archive
([`cmd/web/restore.go:271`](../../cmd/web/restore.go)). The two redacted columns therefore land NULL,
whatever the database held a moment earlier. The webhook signing key and the OAuth client secret are
gone, and delivery signing and the SSO token exchange stay broken until the operator types each one
again.

**That is a real cost and this ADR accepts it.** It is bounded, it is one act per channel and per
provider, and both values are re-enterable by design. The condition on accepting it is that the
operator reads it in the guide before they restore. Limb 4 is not discharged by the code alone.

### 5. What this rule does not claim

- **It does not claim the archive carries no credential material.** `account.password_hash` and
  `personal_token.token_hash` ride. A hash resists reversal. It does not resist an offline guess
  against a weak password. The archive still deserves care.
- **It does not rule that the archived `account.totp_secret` is usable after a restore.** It is not.
  A restore rotates the session signing key ([`cmd/web/restore.go:226`](../../cmd/web/restore.go)),
  and `rotateSessionKey` re-derives the TOTP key from the new key
  ([`cmd/web/restore.go:395`](../../cmd/web/restore.go)). The restored ciphertext was sealed under
  the old key. This ADR records the fact because it is the ground of limb 1 for that column. It
  rules nothing about the second-factor path, which is out of this ticket's scope. The lockout that
  follows is filed as [#1419](https://github.com/winniel123/verge-asm/issues/1419).
- **It does not reach the host-level `pg_dump` path.** That dump reads the database and carries both
  cleartext columns. `docs/guides/sso.md` says so already, and this rule binds the in-app export
  alone.
- **It does not reach an excluded table.** Which tables the archive carries is a separate rule
  ([ADR-0161](./0161-the-backup-allowlist-and-the-exclusion-list-partition-the-business-schema-so-a-new-table-is-classified-by-a-human-or-the-test-fails.md)).
  This rule governs the columns of a table the archive already carries.

## Consequences

- **[`docs/guides/backup-and-restore.md`](../guides/backup-and-restore.md) is corrected at both sites
  that state the contrary claim.** ADR-0058 requires that. A reader of the old sentences would treat
  the archive as a keyring and would not plan the re-entry limb 4 costs them.
- **The restore section of that guide gains the re-entry step.** It is the one operator act limb 4
  creates.
- **`cmd/web/backup.go` and `cmd/web/restore.go` gain this ADR's citation** on the two comments that
  already state the rule. No behaviour changes.
- **No production behaviour changes.** The code has this shape today. What changes is that the shape
  has a record and the guide stops contradicting it.
- **A new cleartext credential column has a document to be held to at review.** Nothing fires on one
  automatically. `TestBackupTablesPartitionSchema` classifies tables, not columns.
- **[ADR-0053](./0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)
  and [ADR-0124](./0124-a-backup-carries-data-and-no-secret-and-updating-is-guided-not-self-applied.md)
  §1 each gain one bounding note**, at the clause that states *no credential* and *no secret*. Neither
  rule is withdrawn. Both sentences, read alone and in the present tense, tell a session the archive
  holds nothing that deserves care, and it holds three write-only values. ADR-0058 requires the note
  at each site rather than only here.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** Redaction is a property of one export path and
  not a domain term.
- **The TOTP finding is recorded and not fixed here.** It is a claim about the world that this ADR
  verified while establishing limb 1. Repairing it would change the second-factor path, which is a
  production change with its own review. It is filed as
  [#1419](https://github.com/winniel123/verge-asm/issues/1419), which owns the fix.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Carry the two columns, so a restore is turnkey** | This is the shape ADR-0124 already refused for the two key volumes, one hop further in. A leaked archive would then hand a reader the webhook signing key and the OAuth client secret, and both authorise an act against a system outside the estate. The cost of refusing it is one typed value per channel and per provider |
| **Redact the hashes and the TOTP ciphertext too** | A restore that drops `password_hash` locks every operator out of the instance it just restored, and the archive would carry no login at all. The hashes are what make a restored estate usable, and limb 1's test says they are not recoverable secrets |
| **Encrypt the whole archive under an operator passphrase, and redact nothing** | Moves the credential problem to a passphrase the operator must keep as long as the file, and a lost passphrase is a lost backup. It also buys nothing that redaction does not, because the two columns are re-enterable and nothing else in the archive needs the protection |
| **Redact on export only, and trust that every archive was taken after the redaction landed** | An archive is a file an operator keeps. A file taken before the redaction landed still holds both columns in cleartext, and a restore would write a stale secret over a live one. Limb 3 exists because the export cannot know when the file it is handed was written |
| **Correct the guide and file no ADR** | The guide is the wrong home for the rule. It is written for an operator taking a backup, and the rule binds a session adding a column to the schema. It also fails [`comment-policy.md`](../spec/comment-policy.md) §8.2, which asks for the record when the rule binds code outside the file that stated it |
| **Cite ADR-0124 §1 on the survivor** | §1 rules that the archive holds no secret and grounds it on the two key volumes the export never opens. It names no column and rules nothing about a hash. Citing it repeats the wrong-citation shape [#1354](https://github.com/winniel123/verge-asm/issues/1354) tracks |
| **Cite ADR-0053 on the survivor** | ADR-0053 rules custody — where a secret is held and that `web` renders "set" or "not set". Export redaction is a different act. `comment-policy.md` §4.7 names ADR-0053 as one of three ADRs a survivor should not cite without a second read |
| **Amend ADR-0124 §1 and file nothing** | Under ADR-0058's split an amendment carries a claim about the world. This is a rule with a rejected alternative, a test a future column is measured against, and an operator-visible cost. ADR-0124 keeps its §1 unaltered and gains nothing here |
