FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make gcc libc-dev

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o theo ./cmd/theo

# Create a minimal runtime image
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates git

# Copy the binary from the builder stage
COPY --from=builder /app/theo /app/theo

# Set the entrypoint
ENTRYPOINT ["/app/theo"]