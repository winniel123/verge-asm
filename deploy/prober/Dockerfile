# syntax=docker/dockerfile:1

# A minimal, hardened SSH *target* for a dedicated verge-asm internet vantage.
#
# This is NOT a resident prober. The verge instance ships the exact prober
# binary it wants over SSH and exec's it per invocation (ADR-0103, §1.5, #14),
# which is what makes version skew between instance and vantage structurally
# impossible. Baking a prober in here would reintroduce exactly that skew and
# would need its own ADR. The only job of this image is to be a reachable,
# hardened SSH endpoint that runs an ordinary unprivileged login. The pushed
# prober is CGO_ENABLED=0 static (§1.4), so it needs no runtime from this image
# — musl vs glibc does not matter to it.
# tag alpine:3.20 pinned by manifest-list digest (#333)
FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc

# The single non-root login account. Its name is what the operator types into
# the verge Probers form as the username (§4.5); to use a different one, rebuild
# with --build-arg PROBER_USER=<name> and type that same name into verge. uid/gid
# 65532 mirrors the main stack's distroless nonroot (§2). Nothing here needs root
# at runtime, so the container runs strictly as this account.
ARG PROBER_USER=prober
ARG PROBER_UID=65532
ARG PROBER_GID=65532

RUN apk add --no-cache openssh-server \
    && addgroup -g "${PROBER_GID}" "${PROBER_USER}" \
    && adduser -D -u "${PROBER_UID}" -G "${PROBER_USER}" -s /bin/ash "${PROBER_USER}" \
    # The .ssh dir is under the account's home (sshd resolves authorized_keys per
    # user); the host-key dir is a fixed, username-independent path so the compose
    # volume mount does not depend on PROBER_USER. Both are owned by the account so
    # the unprivileged entrypoint can write them; the chown on /hostkeys is what
    # Docker copies onto a fresh named volume the first time it is mounted (the same
    # trick the main image uses for /state). StrictModes (sshd default) needs these
    # owned by the login user and not group/world-writable.
    && mkdir -p "/home/${PROBER_USER}/.ssh" /hostkeys \
    && chown -R "${PROBER_UID}:${PROBER_GID}" "/home/${PROBER_USER}" /hostkeys \
    && chmod 755 "/home/${PROBER_USER}" \
    && chmod 700 "/home/${PROBER_USER}/.ssh" /hostkeys

# Render sshd_config with the account baked in, so AllowUsers and the account
# stay in sync from the one ARG and the operator never hand-edits sshd config.
COPY sshd_config /etc/ssh/sshd_config.prober
RUN sed -i "s|__PROBER_USER__|${PROBER_USER}|g; s|__PROBER_HOME__|/home/${PROBER_USER}|g" \
        /etc/ssh/sshd_config.prober \
    && ln -sf /etc/ssh/sshd_config.prober /etc/ssh/sshd_config.active

COPY entrypoint.sh /usr/local/bin/prober-entrypoint
RUN chmod 0555 /usr/local/bin/prober-entrypoint

USER ${PROBER_UID}:${PROBER_GID}
ENV PROBER_HOME=/home/${PROBER_USER}
EXPOSE 2222
ENTRYPOINT ["/usr/local/bin/prober-entrypoint"]
