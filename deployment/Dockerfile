# syntax=docker/dockerfile:1.7

# Build stage: both binaries, static and stripped. BUILDPLATFORM and TARGETOS/TARGETARCH come from
# `docker buildx` for multi-arch builds; a plain `docker build` targets the local platform.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
# VERSION and COMMIT name the build in log records and traces, e.g. --build-arg VERSION=v1.0.0.
ARG VERSION=dev
ARG COMMIT=
ENV CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    pkg=github.com/snykk/go-rest-boilerplate/internal/platform/buildinfo && \
    flags="-s -w -X ${pkg}.version=${VERSION} -X ${pkg}.commit=${COMMIT}" && \
    go build -trimpath -ldflags="${flags}" -o /out/server ./cmd/server && \
    go build -trimpath -ldflags="${flags}" -o /out/migrate ./cmd/migrate

# Seed target, for local development only: `docker build --target seed` builds an image of the development seed
# data, which the compose `seed` profile uses. The default target below does not contain it.
FROM builder AS seed-builder
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM gcr.io/distroless/static-debian12:nonroot AS seed

WORKDIR /app
COPY --from=seed-builder /out/seed /app/

USER nonroot:nonroot
ENTRYPOINT ["/app/seed"]

# Runtime stage, the default target: distroless static, no shell, runs as nonroot. The migrations are embedded in
# /app/migrate.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/server /out/migrate /app/

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/server"]
