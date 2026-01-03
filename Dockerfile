# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd

# Final stage
FROM alpine:latest

WORKDIR /root/

# Install curl for health check
RUN apk --no-cache add curl

# Copy the binary from the builder stage
COPY --from=builder /app/api .

# Expose the application port
EXPOSE 8100

# Run the application
CMD ["./api"]
