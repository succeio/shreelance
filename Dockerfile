# Multi-stage build for Go app
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Build the final minimal image
FROM alpine:3.19

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Copy UI static files
COPY --from=builder /app/ui ./ui

# Expose port
EXPOSE 8080

# Command to run
CMD ["./server"]
