# Build stage - compile static file server
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY server/main.go .

# Build static binary for target platform
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o server main.go

# Final stage - scratch image (smallest possible)
FROM scratch

# Copy CA certificates for HTTPS (if needed)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the static binary
COPY --from=builder /build/server /server

# Copy static files
COPY static/ /app/

# Run as non-root user (UID 65534 = nobody)
USER 65534:65534

EXPOSE 8080

ENTRYPOINT ["/server"]
