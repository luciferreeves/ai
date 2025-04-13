# Build stage
FROM golang:1.24-alpine AS builder

# Install git and build tools
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -o ai/ai ai/main.go

# Runtime stage
FROM alpine:3.21

# Set working directory
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ffmpeg python3 py3-pip opus libopus-dev ca-certificates && \
    pip3 install --no-cache-dir yt-dlp && \
    mkdir -p /app/temp

# Copy the binary from the builder stage
COPY --from=builder /app/ai/ai .

# Copy environment variables file
# Note: You can also use CapRover environment variables instead
COPY .env.example .env

# Expose port 80 for CapRover health checks
# The application doesn't need to use this port, it's just for health checks
EXPOSE 80

# Create a simple health check endpoint
RUN echo '#!/bin/sh\nwhile true; do echo -e "HTTP/1.1 200 OK\n\nOK" | nc -l -p 80; done' > /app/healthcheck.sh && \
    chmod +x /app/healthcheck.sh

# Start both the health check service and the application
CMD /app/healthcheck.sh & ./ai