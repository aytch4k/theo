FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o theo ./cmd/theo

# Create a minimal image
FROM alpine:latest

# Install git and other dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/theo /app/theo

# Create data directory
RUN mkdir -p /app/data

# Set environment variables
ENV THEO_DATA_DIR=/app/data

# Expose ports if needed
# EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["/app/theo"]