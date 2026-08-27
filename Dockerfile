# syntax=docker/dockerfile:1

# web, worker and prober share one build stage and one Go module
# (ADR-0001): a schema mismatch across the wire contract cannot arise from
# building the three binaries differently.
# tag golang:1.25-bookworm pinned by manifest-list digest (#333)
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm@sha256:3b4a11519ad929d1e1d261a12cff056f0c85b735253d7d861346b9c6f8b36437 AS builder
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
RUN go build -o /out/web ./cmd/web \
    && go build -o /out/worker ./cmd/worker \
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

# tag gcr.io/distroless/static-debian12:nonroot pinned by manifest-list digest (#333)
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS web
COPY --from=builder /out/web /app/web
COPY --from=builder --chown=65532:65532 /state /app/state
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/web", "-healthcheck"]
ENTRYPOINT ["/app/web"]

# tag gcr.io/distroless/static-debian12:nonroot pinned by manifest-list digest (#333)
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS worker
COPY --from=builder /out/worker /app/worker
COPY --from=builder /out/prober /app/prober
# The per-architecture probers the off-host router pushes (VERGE_PROBER_DIR=/app/probers).
COPY --from=builder /out/probers /app/probers
COPY --from=builder --chown=65532:65532 /state /app/state
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/worker", "-healthcheck"]
ENTRYPOINT ["/app/worker"]
