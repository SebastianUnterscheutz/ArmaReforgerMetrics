# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o arma-metrics .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /build/arma-metrics .

# Create logs directory
RUN mkdir -p /app/logs

# Expose port 8880
EXPOSE 8880

# Run the application
CMD ["./arma-metrics"]