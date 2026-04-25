# syntax=docker/dockerfile:1.7
#
# Multi-stage, multi-arch build. BUILDPLATFORM / TARGETPLATFORM are set
# by `docker buildx` when building for linux/amd64 + linux/arm64; a
# plain `docker build` still works because they default to the local
# platform.

# ---------- builder ----------
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}

# -trimpath strips local file-system paths from the binary; -s -w
# drops the symbol/debug tables for a smaller image.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Migrations are copied separately at runtime; build the migration CLI
# too so the same image can run `./migrate -up` in init containers.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migration

# ---------- runtime ----------
# Distroless static: no shell, no package manager, nonroot user baked in.
# ~2 MB base vs ~8 MB for alpine. No apt/apk surface to patch either.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/api      /app/api
COPY --from=builder /out/migrate  /app/migrate
COPY --from=builder /app/cmd/migration/migrations /app/cmd/migration/migrations

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]
