# =============================================================================
# Complaint Service – Multi-stage Docker Build (Go)
# =============================================================================
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY *.go ./
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o complaint-service .

# ── Runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/complaint-service .
EXPOSE 8083
CMD ["./complaint-service"]
