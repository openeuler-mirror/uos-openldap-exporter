# Multi-stage build for uos-openldap-exporter
FROM golang:1.25.4-alpine AS builder

# Install ca-certificates for secure connections
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o uos-openldap-exporter .

# Final stage
FROM scratch

# Copy certificates from builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/uos-openldap-exporter /usr/local/bin/uos-openldap-exporter

# Expose default port
EXPOSE 9330

# Run the binary
ENTRYPOINT ["uos-openldap-exporter"]