# syntax=docker/dockerfile:1

# web, worker and prober share one build stage and one Go module
# (ADR-0001): a schema mismatch across the wire contract cannot arise from
# building the three binaries differently.
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder
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

# distroless has no shell to chown a bind/named-mount target at runtime, so
# the state directory is created and owned by the nonroot uid here; Docker
# copies its ownership into the named volume the first time it is mounted.
RUN mkdir -p /state && chown 65532:65532 /state

FROM gcr.io/distroless/static-debian12:nonroot AS web
COPY --from=builder /out/web /app/web
COPY --from=builder --chown=65532:65532 /state /app/state
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/web", "-healthcheck"]
ENTRYPOINT ["/app/web"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
COPY --from=builder /out/worker /app/worker
COPY --from=builder /out/prober /app/prober
COPY --from=builder --chown=65532:65532 /state /app/state
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/worker", "-healthcheck"]
ENTRYPOINT ["/app/worker"]
