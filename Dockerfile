# Build stage
FROM golang:1.23-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main .

# Runtime stage with chromedp headless shell
FROM chromedp/headless-shell:latest

WORKDIR /app

# Copy SSL certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the compiled Go binary
COPY --from=builder /build/main ./

ENTRYPOINT ["./main"]
