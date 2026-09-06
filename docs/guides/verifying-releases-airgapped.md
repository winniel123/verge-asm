---
title: Verifying a release on an air-gapped host
section: Operating
order: 5
description: Verify a verge-asm release offline on an air-gapped or disconnected host from a carry-in kit — cosign save OCI layouts, cosign verify --local-image, cosign initialize and trusted_root.json, gh attestation download bundles, gh attestation verify with --custom-trusted-root and trusted_root.jsonl, the manifest blob under blobs/sha256, cosign verify-blob, sha256sum, keyless Sigstore with no network and no TUF mirror, and the --registry-referrers-mode legacy pin.
---

# Verifying a release on an air-gapped host

This page holds the carry-in kit and the offline commands. **It states no contract of its own.**

[verifying-releases.md](verifying-releases.md) states the contract. It carries the Fulcio identity
pin under [The trust anchor](verifying-releases.md#the-trust-anchor), and the bare-tag rule under
[The cheapest check needs no tooling](verifying-releases.md#the-cheapest-check-needs-no-tooling).
Read that guide first, then build the kit below.

---

## Why this is a page and not a section

The kit is 23 named files, 6 layout directories, 2 tool binaries, 2 trust roots and 12 verify
commands. That inventory roughly triples the connected-host guide, and it serves a different task
on a different day.

Both pages ship inside the `web` binary, so the split costs nothing. `docs/guides/embed.go` embeds
every `*.md` file in this directory. A second page also buys a second row in the in-app search.

---

## The kit

**The connected host produces every kit file. The air-gapped host only reads.**

| Group | Count | Note |
| --- | --- | --- |
| tools | 2 | `bin/cosign` v3.1.3 or later, `bin/gh` v2.96.0 or later |
| trust roots | 2 | `trusted_root.json` for cosign, `trusted_root.jsonl` for `gh` |
| OCI layouts | 6 directories | one `cosign save` per digest |
| attestation bundles | 6 files | one `gh attestation download` per digest |
| Release assets | 13 named files | plus the Trivy set |

The two trust roots are two different files. cosign reads `trusted_root.json`. `gh` reads
`trusted_root.jsonl`.

### Build the kit on the connected host

1. Copy the `cosign` and `gh` binaries into `bin/`. Each one must meet the floor in the table above.
2. Run `cosign initialize` to write `trusted_root.json`. **That file is not a Release asset.** The
   operator carries it in.
3. Run one `cosign save` per digest. The six digests give six OCI layout directories.
4. Run one `gh attestation download` per digest. The six digests give six bundle files.
5. Download the 13 named Release assets and the Trivy set. See
   [Verify the Release assets](verifying-releases.md#3-verify-the-release-assets) for the asset list.

The six digests are the two index digests and the four platform manifest digests. The
connected-host guide lists them under
[What one release publishes](verifying-releases.md#what-one-release-publishes).

---

## Six layouts, not one

`cosign save` names one digest, and it copies only that digest's material:

- It copies the signature and a cosign `.att` attestation of the **named digest only**.
- It drops every child signature.
- It has no `--recursive` flag.
- `--local-image` takes a directory path, not a digest.
- Two saves into one directory do not merge, because the second replaces `index.json`.

**Disk cost.** Each platform image is stored about twice.

---

## cosign alone does not cover every artifact

A native GitHub attestation reaches `blobs/` as orphan bytes that no cosign command can find. So
`cosign verify-attestation --local-image` cannot read the provenance. The command that does reach
it is:

```sh
gh attestation verify <layout>/blobs/sha256/<digest> \
  --repo winniel123/verge-asm \
  --bundle <bundle>.jsonl \
  --custom-trusted-root trusted_root.jsonl \
  --format json
```

**No upstream document states the next fact, so this page does.** A `gh` verification of an image
uses a **file**, and that file is the manifest blob inside the saved layout at
`<layout>/blobs/sha256/<digest>`. An OCI digest is the SHA-256 of the manifest bytes. So the layout
supplies the artifact, and the kit needs no extra copy of it.

---

## Two operator notes

- **`--trusted-root` is not optional offline.** Without it, cosign fails while it updates the TUF
  remote mirror.
- **The `gh` success case may print nothing.** In a non-interactive shell it printed nothing and
  returned exit code `0`. Use `--format json`, or test the exit code.

---

## The commands the air-gapped host runs

- six `cosign verify --local-image`, one per digest
- six `gh attestation verify`, one per digest
- one `cosign verify-blob` over `SHA256SUMS`
- one `sha256sum -c` in the download directory

The kit inventory counts 12 verify commands. Those twelve are the six cosign runs and the six `gh`
runs, one of each per digest. The `SHA256SUMS` pair adds the last two commands.

Take the identity and the flags for `cosign verify` and `cosign verify-blob` from
[Verify the image signatures with cosign](verifying-releases.md#2-verify-the-image-signatures-with-cosign)
and [Verify the Release assets](verifying-releases.md#3-verify-the-release-assets). Offline, each
`cosign verify` names a layout directory with `--local-image`, and each one names the trust root
with `--trusted-root`.

---

## Why keyless still works offline

Keyless verification makes no network call. The offline form is
`cosign verify --local-image --trusted-root <file>`, with the identity flags the main guide states.

**cosign v3 deleted `--offline`.** Every cosign v2 instruction is wrong for this path. The cosign
air-gap README section declares itself out of date, and it still prints `--offline=true`.

---

## The `--registry-referrers-mode legacy` pin

That pin is what makes this path work. A legacy `.sig` tag produces a layout that `--local-image`
accepts. A referrer signature produces a layout that fails with `no signatures associated with the
image saved in <dir>`.
