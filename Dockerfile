# Multi-stage Docker build for HalluciDetect
# Stage 1: Build binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency definition
COPY go.mod ./
# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/hallucidetect ./cmd/hallucidetect

# Stage 2: Minimal runtime image
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates for HTTPS/TLS requests (if using LLM judge)
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/hallucidetect /app/hallucidetect
COPY --from=builder /app/testdata /app/testdata

EXPOSE 8080

ENTRYPOINT ["/app/hallucidetect"]
CMD ["serve", "--port", "8080", "--host", "0.0.0.0"]
