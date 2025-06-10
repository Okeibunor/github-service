FROM golang:1.21.0-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy the binary from builder
COPY --from=builder /app/github-service .
COPY --from=builder /app/config.yaml .

# Set environment variables
ENV CONFIG_FILE=/app/config.yaml

# Run the service
CMD ["./github-service"] 