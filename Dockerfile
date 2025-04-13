# Build stage
FROM golang:1.24-alpine AS builder

# Install git, build tools, pkgconfig, and opus headers
RUN apk add --no-cache git gcc musl-dev pkgconfig opus-dev

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
RUN apk add --no-cache ffmpeg python3 py3-pip opus opus-dev ca-certificates curl \
    && python3 -m venv /venv \
    && . /venv/bin/activate \
    && pip install --no-cache-dir yt-dlp \
    && ln -s /venv/bin/yt-dlp /usr/local/bin/yt-dlp \
    && mkdir -p /app/temp

# Install cloudflared (ARM64)
RUN curl -LO https://github.com/cloudflare/cloudflared/releases/download/2025.4.0/cloudflared-linux-arm64 \
    && chmod +x cloudflared-linux-arm64 \
    && mv cloudflared-linux-arm64 /usr/local/bin/cloudflared

# Copy the binary from the builder stage
COPY --from=builder /app/ai/ai .

# Copy environment variables file
COPY .env.example .env

# Start cloudflared WARP and then launch the bot
CMD sh -c "\
    cloudflared tunnel --url http://localhost:8080 --no-autoupdate & \
    sleep 5 && \
    ./ai"
