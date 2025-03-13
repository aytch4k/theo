# Build stage
FROM golang:1.18-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc libc-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o theo ./cmd/theo

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/theo /app/theo

# Create data directory
RUN mkdir -p /app/data

# Set the entrypoint
ENTRYPOINT ["/app/theo"]