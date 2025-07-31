# Build stage
FROM golang:1.24 AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 creates a static binary
# -ldflags="-w -s" removes debug info and symbol table
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main cmd/main.go

# Final stage - use scratch for minimal size
FROM scratch

# # Copy CA certificates for HTTPS requests
# COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder stage
COPY --from=builder /app/main /main

# Expose port (adjust as needed)
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/main"]