---
title: Provisioning a prober
section: Scanning
order: 2
description: A worked Docker example for standing up the dedicated prober host that lets verge measure your exposure from the outside.
---

# Provisioning a prober — a worked Docker example

Standing up a **dedicated prober host**: the internet `Vantage` that lets verge see
your estate from the outside. This is [using.md](using.md) step 3 in full, with a
copy-paste Docker recipe instead of hand-edited `sshd` config.

**Why a second host at all.** `Exposure` needs an outside observer, unconditionally.
A single all-in-one install builds a complete, honest *internal* reachability
inventory, but it cannot see its own estate from the internet — probing your own
public address from inside is a hairpinning trap that never traverses the inbound
policy. So `Exposure` needs a **prober**: a second Linux host the instance reaches
over SSH and pushes its measurement binary to. Until one exists, exposure claims are
withheld and the system degrades to internal-only — it never reports `firewalled` or
`exposed` for something it did not look at. See
[packaging-and-configuration.md §4.2–4.3](../spec/packaging-and-configuration.md).

**What the recipe is — and is not.** It is a hardened SSH *target*, not a resident
prober. The instance ships the exact binary it wants at each invocation, which is what
makes version skew between instance and vantage impossible; a container that baked in
its own prober would reintroduce that skew. The recipe therefore runs *only* a minimal
SSH server. The full rationale is recorded alongside the artifact in
[`deploy/prober/README.md`](../../deploy/prober/README.md).

The recipe lives at [`deploy/prober/`](../../deploy/prober/). Run it on the **second
machine** — not next to the main stack.

---

## Prerequisites

- A second Linux host (`x86_64` or `aarch64`) with Docker and the Compose plugin. Its
  architecture need **not** match your instance's — an `arm64` instance pushes to an
  `amd64` host fine, because the instance carries a prober for both.
- A route from the instance to this host on the SSH port you will map.
- Your verge instance already up (see [running.md](running.md)) and logged in as admin.

The host does **not** need the Go toolchain or any verge binary. The instance brings
the prober with it at invocation.

---

## Step 1 — Render the public key in verge

In the UI, go to **Probers** and start provisioning. Supply the three non-secret
values (§4.5):

| Value | What to enter |
| --- | --- |
| host | a name or address the instance can reach this second machine on |
| port | the host port you will map below — default **22** |
| username | **`prober`** — the account the recipe ships (non-root) |

The instance generates the SSH keypair and renders the **public** half for you to
install. **Only the public key ever leaves the instance** — the private half stays on
the `worker-state` volume and never moves
([ADR-0053](../adr/0053-a-secret-is-held-only-where-its-act-is-performed-and-the-shared-store-holds-none.md)).
Copy the rendered public key; it is one line beginning `ssh-ed25519 AAAA…`.

> verge never asks you for a private key, and neither does the recipe — the recipe
> refuses one outright. If anything prompts you to paste a *private* key, stop.

---

## Step 2 — Bring the target up on the second host

On the second machine:

```sh
git clone https://github.com/winniel123/verge-asm
cd verge-asm/deploy/prober

cp .env.example .env
```

Edit `.env` and paste the key from step 1:

```sh
# .env
PROBER_PUBKEY=ssh-ed25519 AAAA…the key verge rendered…
# PROBER_SSH_PORT=22          # host port the instance connects to; default 22
```

Then build and start it:

```sh
docker compose up -d --build
docker compose logs
```

The first boot generates a persistent host key and prints its fingerprint:

```
prober: first boot — generating a persistent ed25519 host key
prober: authorized_keys installed with options: restrict
prober: host key fingerprint (pin this if verge asks you to compare):
256 SHA256:5OtpowuBHKG6UWExE+Hg3ltBhVstowINsvC7s/eGpbo verge-asm prober host key (ED25519)
Server listening on 0.0.0.0 port 2222.
```

That fingerprint is what the instance will pin in step 3 — note it if you want to
compare. sshd listens unprivileged on `2222` inside the container; compose maps your
chosen host port (default `22`) onto it, so from the instance's side this is an
ordinary SSH host on port 22.

### The host key must persist — this is why

The recipe stores the host key on a named volume (`prober-hostkeys`) on purpose. The
instance **pins** this key at provisioning, and a later change is a hard failure,
never a prompt (§3): an unpinned push would let whoever answers on that address run
verge's binary and return fabricated inventory. Keep the volume across restarts and
the key never changes. If you ever `docker compose down -v` (which deletes the
volume), the next boot mints a new key and the vantage will stop verifying until you
re-pin it in the UI.

---

## Step 3 — Provision, back in verge

Finish provisioning in the **Probers** form. The instance now:

1. **Reads and pins the host key** — compare it against the fingerprint from step 2 if
   the UI offers it.
2. **Checks `uname -s` / `uname -m`** — a host that is not Linux on `x86_64` /
   `aarch64` is refused here, with the reason, rather than failing silently hours later
   (§1.6).
3. **Observes your egress** — on the first connection the prober reads the instance's
   address from `SSH_CLIENT`. The instance then renders *your egress is `203.0.113.5`*
   and offers it for declaration (§4.4). This is a `Coverage` statement — *we cannot
   yet construct exposure* — not an alert.

You did not hand-edit any `sshd` config or `authorized_keys` to get here — the recipe
wrote both from the key you pasted.

---

## Step 4 — Harden the key to your egress

Until you declare your egress, **both** vantages verify `internet`, there is no
internal leg, and `Exposure` stays non-constructible — loud and honest, never a
fabricated `exposed`. Declaring the rendered egress gives the internal leg and unlocks
exposure.

The recipe already installs the key under `restrict`, which locks it to the one thing
the instance does — open a session and exec the pushed binary — disabling port,
agent, X11 forwarding and pty. Once verge has shown your egress address, tighten it
further by pinning the key to that source. Set it in `.env` and re-up:

```sh
# .env
PROBER_FROM=203.0.113.5     # the egress address verge rendered
```

```sh
docker compose up -d
```

The account's `authorized_keys` line is now `restrict,from="203.0.113.5" ssh-ed25519 …`
— the key is refused from any other source address (§3). This is operator-side
hardening on *your* `authorized_keys`; verge does not enforce it, it recommends it.

---

## Step 5 — Confirm exposure is now constructible

Back in verge, declare the rendered egress (the offer from step 3). With both an
internal and an internet vantage, the **Exposure** page becomes constructible. Kick a
batch if you don't want to wait for the cadence:

```sh
# on the instance host
docker compose run --rm worker -trigger dns
```

Exposure findings — what of yours is reachable from the internet — now appear, and
`sensitive-port-reached-from-internet` and its siblings can fire.

---

## Troubleshooting

| Symptom | Cause and fix |
| --- | --- |
| Vantage never verifies after a restart | The host-key volume was deleted (`down -v`), so the pin tripped. Re-pin in the UI, or restore the volume. |
| Provisioning refused with an architecture reason | The host is not Linux on `x86_64`/`aarch64` (§1.6). Use a supported host; nothing else is in the matrix. |
| `no public key supplied` in the logs | `PROBER_PUBKEY` is empty and no key is bind-mounted. Paste the rendered key into `.env`. |
| `refusing a PRIVATE key` in the logs | You pasted the private half. Use only the public key verge rendered — it starts `ssh-ed25519`/`ssh-rsa`. |
| `not a well-formed public key` in the logs | The pasted value is mangled (line-wrapped, quotes, or truncated). Paste it as one unbroken line. |
| First boot fails writing the host key | You replaced the named `prober-hostkeys` volume with a bind-mounted host dir; it comes up root-owned and the non-root container cannot write it. Use the named volume as shipped. |
| Connection refused from the instance | The host port isn't reachable, or `PROBER_SSH_PORT` doesn't match the port you entered in verge. Check the map and any firewall. |
| Want a different username | Rebuild with `docker compose build --build-arg PROBER_USER=<name>` and enter that name in verge. |

For failures beyond the prober host itself — a withheld `Exposure`, undelivered
notifications, a scan you cannot confirm ran — see the instance-wide
**[troubleshooting.md](troubleshooting.md)**.

---

## What lives where

- The recipe and its own record: [`deploy/prober/`](../../deploy/prober/) and its
  [`README.md`](../../deploy/prober/README.md).
- The model behind it: [packaging-and-configuration.md](../spec/packaging-and-configuration.md)
  §1.4–1.6, §2, §3, §4, and
  [ADR-0103](../adr/0103-a-vantage-is-one-position-and-the-prober-is-optional-provisioning-detail.md).
