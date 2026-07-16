# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
# Install git (required for go mod) and ca-certificates
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build the binary
RUN go build -o /app/server ./cmd/server

# Final stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
EXPOSE 8080
# Use a non-root user for security (optional)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
ENTRYPOINT ["./server"]