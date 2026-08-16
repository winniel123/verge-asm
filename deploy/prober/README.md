# A dedicated prober host, packaged

This directory is a **recipe** for standing up an internet `Vantage` — the outside
observer that makes `Exposure` constructible — on a second machine, with
`docker compose` instead of hand-edited `sshd` config and `authorized_keys`.

`Exposure` requires an internet vantage **unconditionally**: probing your own public
address from inside the estate is the hairpinning trap that never traverses the
inbound policy, so the instance needs a host *outside* to observe it
([packaging-and-configuration.md §4.2–4.3](../../docs/spec/packaging-and-configuration.md)).
The main stack is `docker compose up`; before this recipe, the prober host was "SSH
into a box and edit `authorized_keys` by hand." This closes that asymmetry.

For the full operator walkthrough — where the rendered public key comes from, the
host-key pin, declaring your egress — see **[docs/guides/prober.md](../../docs/guides/prober.md)**.
This README is the artifact's own record: what it is, why it is shaped this way, and
the quickstart.

---

## What this is, and what it is deliberately *not*

**This is an SSH _target_, not a resident prober.** It runs one thing: a minimal,
hardened OpenSSH server exposing a single non-root account. It ships **no** prober
binary and answers **no** measurement protocol of its own.

That is the whole point, and it is load-bearing rather than incidental:

> The verge instance **ships the exact prober binary it wants at invocation** and
> exec's it over SSH ([#14](https://github.com/winniel123/verge-asm/issues/14),
> [packaging spec §1.5](../../docs/spec/packaging-and-configuration.md)). The binary is
> `CGO_ENABLED=0` static precisely so it runs on this host's unknown libc (§1.4), and
> the instance's image carries a prober for **every** matrix architecture, so an
> `arm64` instance can push to this `amd64` host and vice versa. Because the binary
> arrives fresh each invocation, **version skew between instance and vantage is
> structurally impossible** — see
> [ADR-0103](../../docs/adr/0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md).

A container that baked in its own prober and answered over some non-SSH transport
would **reintroduce exactly that skew** — the resident binary would drift from the
instance's release — and would need its own ADR before it could ship. It is out of
scope here, permanently, unless that ADR is written. The default and only deliverable
is the SSH endpoint the instance pushes to.

**No verge image is published for this, and nothing is added to the `GOARCH`/CI
matrix.** Matrix membership is gated on the golden corpus running on that architecture
in CI (§1.2); an `sshd` image has no corpus and would never qualify. The prober host
"is the operator's rather than ours" (§1.5) — so this is a recipe you build locally
from a stock base image (`alpine` + `openssh-server`), not a supply-chain artifact
verge owns and patches.

---

## Posture

Parity with the main stack ([§2](../../docs/spec/packaging-and-configuration.md)),
enforced in [`docker-compose.yml`](docker-compose.yml):

| Control | Value |
| --- | --- |
| `user:` | `65532:65532` — non-root, mirroring the distroless `nonroot` uid |
| `cap_drop:` | `[ALL]` — the prober needs no capability; probing is TCP connect, not SYN |
| `security_opt:` | `no-new-privileges:true` |
| **not present** | `privileged`, `cap_add`, `network_mode: host`, `sysctls` |

sshd itself: **root login disabled, public-key auth only, no passwords, one account.**
Because the container is non-root and drops `CAP_NET_BIND_SERVICE`, sshd listens
unprivileged on **2222** inside; compose maps the host's real SSH port (default 22)
onto it. The `authorized_keys` entry carries the `restrict` option, and takes
`from=<egress>` once the instance has rendered its egress address (§3).

**The recipe only ever touches a _public_ key.** The instance generates the keypair
and only the public half leaves it
([ADR-0053](../../docs/adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)).
The entrypoint reads a public key from `PROBER_PUBKEY` or a bind-mounted file and
**refuses a private key outright**. It never prompts for, accepts, or stores one.

---

## Quickstart

Run this on the **second machine**, not next to the main stack.

```sh
cd deploy/prober
cp .env.example .env
# Paste the PUBLIC key verge rendered (Probers → provision) into PROBER_PUBKEY.
$EDITOR .env

docker compose up -d --build
docker compose logs        # first boot prints the host-key fingerprint
```

Then, in the verge UI under **Probers**, provision with this host's address, the
mapped port (default 22), and the username `prober` (the account this recipe ships).
The instance reads and pins the host key and checks `uname -s` / `uname -m`, then on
first connection observes your egress via `SSH_CLIENT` and offers it for declaration.

### Inputs

| Setting | Where | Purpose |
| --- | --- | --- |
| `PROBER_PUBKEY` | `.env` (env) | the public key verge rendered. Or bind-mount it at `/keys/authorized_keys` (see the commented volume). |
| `PROBER_SSH_PORT` | `.env` (env) | host port the instance connects to. Default `22`. This is the "port" you type into verge. |
| `PROBER_FROM` | `.env` (env) | optional. Once verge shows your egress address, set it to pin the key with `from=` (§3). |
| `PROBER_USER` | build arg | the login account name. Default `prober`. To change it, `docker compose build --build-arg PROBER_USER=<name>` and type that name into verge. |

### The one volume, and why it is required

`prober-hostkeys` persists sshd's host key across restarts. This is **not optional**:
without it, every restart regenerates the host key, which trips the instance's
host-key pin — "a change is a hard failure, never a prompt" (§3) — and the vantage
stops verifying, opening a `Gap` and making `Exposure` non-constructible. First boot
generates the key; keep the volume and it never changes. Losing it is the one event
that forces a re-pin in the UI.

---

## Files

| File | Role |
| --- | --- |
| `docker-compose.yml` | the stack: one `prober` service, posture parity, the host-key volume, port map |
| `Dockerfile` | thin build on `alpine` + `openssh-server`; bakes the one non-root account |
| `sshd_config` | the hardened config, rendered with the account name at build |
| `entrypoint.sh` | first-boot host key, writes `authorized_keys` from the public key, refuses a private key, exec's sshd |
| `.env.example` | copy to `.env` |
