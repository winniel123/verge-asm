#!/bin/sh
# Prepare the SSH target, then hand off to sshd. Runs as the unprivileged login
# account (never root): it only writes files under that account's own home.
set -eu

HOME_DIR="${PROBER_HOME:-$HOME}"
# Fixed, username-independent path so the compose volume mount does not move with
# PROBER_USER (see docker-compose.yml).
HOSTKEY_DIR="/hostkeys"
HOSTKEY="$HOSTKEY_DIR/ssh_host_ed25519_key"
SSH_DIR="$HOME_DIR/.ssh"
AUTH_KEYS="$SSH_DIR/authorized_keys"

# --- Host key: generate once on the named volume, reuse forever -------------
# The first boot writes a persistent host key; every later boot reuses it, so
# the instance's pinned host key never changes and the vantage keeps verifying
# across restarts (§3). Losing this volume is the one event that trips the pin.
mkdir -p "$HOSTKEY_DIR"
chmod 700 "$HOSTKEY_DIR"
if [ ! -f "$HOSTKEY" ]; then
    echo "prober: first boot — generating a persistent ed25519 host key" >&2
    ssh-keygen -t ed25519 -f "$HOSTKEY" -N '' -C "verge-asm prober host key" >/dev/null
fi
chmod 600 "$HOSTKEY"

# --- authorized_keys: from the operator-supplied PUBLIC key -----------------
# The instance generates the keypair; only the public half ever leaves it
# (ADR-0053, §3). This recipe accepts that public half via PROBER_PUBKEY or a
# bind-mounted file and never prompts for, accepts or stores a private key.
if [ -n "${PROBER_PUBKEY:-}" ]; then
    KEY="$PROBER_PUBKEY"
elif [ -f /keys/authorized_keys ]; then
    KEY="$(cat /keys/authorized_keys)"
else
    echo "prober: no public key supplied." >&2
    echo "        Set PROBER_PUBKEY to the key verge rendered, or bind-mount it" >&2
    echo "        read-only at /keys/authorized_keys." >&2
    exit 1
fi

# Refuse a private key outright — the recipe must never hold one (ADR-0053, D4).
case "$KEY" in
    *PRIVATE*KEY*)
        echo "prober: refusing a PRIVATE key. Supply only the PUBLIC half verge" >&2
        echo "        renders (starts 'ssh-ed25519 ...' / 'ssh-rsa ...')." >&2
        exit 1
        ;;
esac

# Reject a multi-line KEY BEFORE it is embedded on the authorized_keys line at the
# end. `ssh-keygen -l` happily lists every fingerprint in a multi-key value and
# exits 0, so a value like "ssh-ed25519 AAAA...\nssh-ed25519 BBBB..." would pass
# the parse check below and then write a SECOND authorized_keys entry with no
# `restrict` and no `from=` prefix — a fully unrestricted key that defeats the
# key-option scoping (§3, §4.2). This gate mirrors the PROBER_FROM charset gate:
# fail closed on any embedded newline so only a single line is ever written.
# POSIX-sh newline literal (this script is #!/bin/sh, so $'\n' is unavailable and
# $(printf '\n') is stripped to empty by command substitution): printf a newline
# followed by a sentinel, then strip the sentinel back off.
NL=$(printf '\nX'); NL=${NL%X}
case "$KEY" in
    *"$NL"*)
        echo "prober: PROBER_PUBKEY must be a single line. A multi-line value would" >&2
        echo "        inject a second, unrestricted authorized_keys entry. Paste only" >&2
        echo "        the one line verge rendered, e.g. 'ssh-ed25519 AAAA... comment'." >&2
        exit 1
        ;;
esac

# Fail fast on a mangled paste: the value must parse as a public key, so a broken
# authorized_keys is caught here rather than as a silent auth failure at first push.
mkdir -p "$SSH_DIR"
chmod 700 "$SSH_DIR"
printf '%s\n' "$KEY" > "$SSH_DIR/.keycheck"
if ! ssh-keygen -l -f "$SSH_DIR/.keycheck" >/dev/null 2>&1; then
    rm -f "$SSH_DIR/.keycheck"
    echo "prober: PROBER_PUBKEY is not a well-formed public key. Paste the single" >&2
    echo "        line verge rendered, e.g. 'ssh-ed25519 AAAA... comment'." >&2
    exit 1
fi
rm -f "$SSH_DIR/.keycheck"

# `restrict` locks the key to the one thing the instance does — open a session
# and exec the pushed binary — disabling forwarding, agent, pty and X11 (§3).
# Once verge has rendered the instance's egress address, set PROBER_FROM to it
# and the key is additionally pinned to that source with from= (§3, §4.2).
OPTS="restrict"
if [ -n "${PROBER_FROM:-}" ]; then
    # PROBER_FROM is embedded verbatim inside from="...". Validate it BEFORE
    # embedding: reject anything outside a sane from-pattern charset (letters,
    # digits, dots, colons for IPv6, slashes for CIDR, commas, and the ? * !
    # pattern operators). This fails closed on a double-quote (which would close
    # the from="..." early and let an attacker append options like
    # command="/bin/sh"), on whitespace or a newline (which would start a whole
    # new authorized_keys line/key), and on a backslash — defeating `restrict`.
    case "$PROBER_FROM" in
        *[!A-Za-z0-9.:/,*?!-]*)
            echo "prober: PROBER_FROM contains a disallowed character. Use only a" >&2
            echo "        from= pattern (hosts, IPs, CIDRs, commas, ? * ! wildcards)," >&2
            echo "        e.g. '203.0.113.5' or '10.0.0.0/8,192.168.0.0/16'." >&2
            exit 1
            ;;
    esac
    OPTS="restrict,from=\"$PROBER_FROM\""
fi

printf '%s %s\n' "$OPTS" "$KEY" > "$AUTH_KEYS"
chmod 600 "$AUTH_KEYS"

echo "prober: authorized_keys installed with options: $OPTS" >&2
echo "prober: host key fingerprint (pin this if verge asks you to compare):" >&2
ssh-keygen -lf "$HOSTKEY" >&2 || true

# -e logs to stderr so `docker logs` shows auth and listener lines.
exec /usr/sbin/sshd -D -e -f /etc/ssh/sshd_config.active
