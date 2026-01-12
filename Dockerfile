FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.io,direct
RUN go mod download

COPY . .

RUN go build -o /collector ./cmd/collector

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /collector .
# Copy migrations for potential runtime use or inspection
COPY --from=builder /app/migrations ./migrations
# Copy Web UI assets
COPY --from=builder /app/internal/web/templates ./internal/web/templates
COPY --from=builder /app/static ./static

EXPOSE 3100

CMD ["./collector"]
