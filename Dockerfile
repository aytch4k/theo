FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o theo ./cmd/theo

# Create a minimal runtime image
FROM alpine:3.18

# Install git (required for git operations)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/theo /app/theo

# Create data directory
RUN mkdir -p /app/data

# Set the entrypoint
ENTRYPOINT ["/app/theo"]