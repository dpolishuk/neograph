# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# No build dependencies needed with CGO_ENABLED=0

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates git

# Copy binary
COPY --from=builder /build/server .

# Create repos directory
RUN mkdir -p /app/repos

EXPOSE 3001

CMD ["./server"]
