# syntax=docker/dockerfile:1

# web, worker and prober share one build stage and one Go module
# (ADR-0001): a schema mismatch across the wire contract cannot arise from
# building the three binaries differently.
# tag golang:1.26.8-bookworm pinned by manifest-list digest (#333, #745)
# Patch pinned to 1.26.8 to agree with the go.mod `go 1.26.8` line (which enforces
# a minimum toolchain of go1.26.8) and .go-version (#745, #1247). The 1.26 minor
# is what golang.org/x/crypto v0.56.0 requires, and v0.56.0 is the only release that
# clears GO-2026-6354 and GO-2026-6355, the reachable x/crypto/ssh advisories.
# The tag beside the digest is decoration, because Docker resolves by digest and never
# validates it. scripts/check-go-pins.sh compares this line, and reads the digest's own
# org.opencontainers.image.version annotation for the gate (#1247).
FROM --platform=$BUILDPLATFORM golang:1.27.1-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# CGO_ENABLED=0 is a measurement decision, not a size one: the pushed
# prober must be statically linked, and it fixes Go's resolver to the
# declared answer path rather than the system one. GOAMD64 is pinned
# because Go's floating-point contraction is architecture-dependent
# (packaging-and-configuration.md §1).
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOAMD64=v1 GOARM64=v8.0

# `dev` here would stamp every build and make the runtime VERGE_VERSION unreachable
# forever (release-pipeline.md §3).
ARG VERGE_VERSION=""

# The prober's origin is the image that carries it, so it reports no version of its own (ADR-0139).
RUN go build -ldflags "-X github.com/winniel123/verge-asm/internal/buildinfo.version=${VERGE_VERSION}" -o /out/web ./cmd/web \
    && go build -ldflags "-X github.com/winniel123/verge-asm/internal/buildinfo.version=${VERGE_VERSION}" -o /out/worker ./cmd/worker \
    && go build -o /out/prober ./cmd/prober

# The instance carries a prober for EVERY matrix architecture so it can push the
# matching binary to a prober host of a different arch — an arm64 instance to an amd64
# host and vice versa (ADR-0103, packaging-and-configuration.md §1.5, #683). The worker
# selects prober-linux-<goarch> by the remote's `uname -m` and refuses a mismatch. Each
# is CGO_ENABLED=0 static (already set) with the same pinned float flags, so a pushed
# binary measures identically wherever it lands.
RUN mkdir -p /out/probers \
    && for a in amd64 arm64; do \
         GOOS=linux GOARCH="$a" GOAMD64=v1 GOARM64=v8.0 \
           go build -o "/out/probers/prober-linux-$a" ./cmd/prober; \
       done

# distroless has no shell to chown a bind/named-mount target at runtime, so
# the state directory is created and owned by the nonroot uid here; Docker
# copies its ownership into the named volume the first time it is mounted.
RUN mkdir -p /state && chown 65532:65532 /state

# The shared transcript-key volume mount point, owned by the nonroot uid the same
# way. Both web (reader) and worker (writer) mount this one volume so the single
# instance key that seals the Transcript corpus at rest is available to each
# (raw-job-output spec §5.3). Same runtime-chown limitation as /state.
RUN mkdir -p /transcript-key && chown 65532:65532 /transcript-key

# tag gcr.io/distroless/static-debian12:nonroot pinned by manifest-list digest (#333)
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS web
# Without this label GHCR never links the package, and GITHUB_TOKEN then loses push
# permission. It must be on the image before the first publish (§2.3, #1248).
LABEL org.opencontainers.image.source="https://github.com/winniel123/verge-asm"
COPY --from=builder /out/web /app/web
COPY --from=builder --chown=65532:65532 /state /app/state
COPY --from=builder --chown=65532:65532 /transcript-key /app/transcript-key
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/web", "-healthcheck"]
ENTRYPOINT ["/app/web"]

# tag gcr.io/distroless/static-debian12:nonroot pinned by manifest-list digest (#333)
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS worker
# Same reason as the web stage above (§2.3, #1248).
LABEL org.opencontainers.image.source="https://github.com/winniel123/verge-asm"
COPY --from=builder /out/worker /app/worker
COPY --from=builder /out/prober /app/prober
# The per-architecture probers the off-host router pushes (VERGE_PROBER_DIR=/app/probers).
COPY --from=builder /out/probers /app/probers
COPY --from=builder --chown=65532:65532 /state /app/state
COPY --from=builder --chown=65532:65532 /transcript-key /app/transcript-key
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/worker", "-healthcheck"]
ENTRYPOINT ["/app/worker"]
