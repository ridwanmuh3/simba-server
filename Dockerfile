# =========================
# 1. Builder Stage
# =========================
FROM golang:1.25.5-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o app ./cmd/app/main.go


# =========================
# 2. Runtime Stage
# =========================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/app /app/app

EXPOSE 9000

RUN adduser -D appuser

USER appuser

ENTRYPOINT ["/app/app"]

HEALTHCHECK --interval=30s --timeout=5s \
  CMD wget -qO- http://localhost:9000/health || exit 1
