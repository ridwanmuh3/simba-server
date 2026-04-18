# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/app     ./cmd/app/main.go && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/seed    ./cmd/seed/seeder.go && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate/main.go


FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /out/app     /app/app
COPY --from=builder /out/seed    /app/seed
COPY --from=builder /out/migrate /app/migrate

RUN adduser -D -H -u 10001 appuser && \
    mkdir -p /app/logs /app/uploads && \
    chown -R appuser:appuser /app

USER 10001:10001

EXPOSE 3000

ENTRYPOINT ["/app/app"]

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=5 \
  CMD wget -qO- http://localhost:3000/health || exit 1
